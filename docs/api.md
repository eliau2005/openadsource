# HTTP API

OpenAdSource exposes four HTTP endpoints on the adserver, plus a
dedicated metrics listener on the worker. This is the formal contract.

| Endpoint            | Method | Response                                          |
|---------------------|--------|---------------------------------------------------|
| `/vast`             | GET    | `200 application/xml` — VAST 4.2 (inline or empty)|
| `/track`            | GET    | `200` or `204 image/gif` — 1×1 transparent GIF89a |
| `/healthz`          | GET    | `200 text/plain` — `"ok"`                         |
| `/metrics`          | GET    | `200 text/plain` — Prometheus exposition format   |
| `/metrics` (worker) | GET    | `200 text/plain` — Prometheus exposition format   |

All endpoints return `200` (or `204` for tracking dedupe) on the happy
path. Failures emit a syntactically valid empty response in the format
the player expects — never a 4xx / 5xx that a player might mis-handle.
The only exception is `/healthz` returning non-200 when the snapshot
hasn't loaded yet (used by the docker-compose healthcheck).

---

## `GET /vast`

Returns a VAST 4.2 XML response — either an `<InLine>` ad or an empty
`<VAST/>`.

### Query parameters

| Name       | Type / format                            | Default                  | Notes |
|------------|------------------------------------------|--------------------------|-------|
| `pos`      | string `pre` / `mid` / `post`            | `pre`                    | Position-targeted pool |
| `offset`   | int32                                    | `0`                      | Rotation index — same `(pos, country, device, offset)` returns the same ad deterministically |
| `country`  | ISO 3166-1 alpha-2 (e.g. `FR`, `US`)     | derived from GeoIP       | Overrides the geo resolver |
| `device`   | `desktop` / `mobile` / `ctv` / `tablet`  | derived from `User-Agent`| Overrides the UA classifier |
| `ad_id`    | UUID                                     | -                        | Forces a specific ad; still subject to freq + budget caps |

Unknown parameters are ignored.

### Request headers consulted

| Header             | Used for                                                      |
|--------------------|---------------------------------------------------------------|
| `X-Forwarded-For`  | Source IP — trusted only if the immediate hop is in `TRUSTED_PROXIES` |
| `X-Real-IP`        | Alternative source IP, same trust rule                        |
| `User-Agent`       | Device classification                                         |
| `Cookie: oas_uid`  | User ID for frequency-cap counting                            |

### Response headers

```
Content-Type: application/xml; charset=utf-8
Cache-Control: no-store, max-age=0
Set-Cookie: oas_uid=<uuid>; HttpOnly; SameSite=Lax; Path=/; Max-Age=...  (only when minted)
```

`Set-Cookie` is sent only on the first request from a new browser. The
`Secure` flag is added when `PUBLIC_BASE_URL` starts with `https://`.

### Status codes

Always `200`. Failures emit `<VAST version="4.2"/>` (no `<Ad>` child)
and the player treats that as a natural no-fill.

### Curl example

```bash
curl -i 'http://localhost:8088/vast?pos=pre&country=FR&device=mobile'
```

Pretty-print the result:

```bash
curl -s 'http://localhost:8088/vast?pos=pre' | xmllint --format -
```

### Internal behavior summary

1. Atomic load of the active-ads snapshot.
2. Bitset filter: `(pos ∩ country ∩ device ∩ global)`.
3. Pick first set bit at index `offset` (deterministic rotation).
4. Frequency cap check (`Redis EVALSHA freq_consume.lua`) — atomic
   INCR-with-cap against `freq:<uid>:<ad_id>` keyed off the cap rule's
   time window.
5. Budget reservation (`Redis EVALSHA budget_reserve.lua`) — atomic
   INCR-with-cap against `campaign:<id>:imps_total`.
6. Resolve `MediaFile` URL: BYO public URL or S3 presign (15 min TTL).
7. Build VAST: one `<Linear>`, one `<MediaFile>`, signed tracking URLs
   for impression / click / 4 quartiles.

Total Redis round trips on the hot path: at most 2 (freq, budget). Most
requests with no caps configured make 0.

---

## `GET /track`

Records a tracking event. Always returns a 1×1 transparent GIF89a; the
response status is the only programmatic signal.

### Query parameters (all required when signing is enabled)

| Name      | Type / format          | Notes                                         |
|-----------|------------------------|-----------------------------------------------|
| `event`   | one of                 | `impression`, `click`, `start`, `firstQuartile`, `midpoint`, `thirdQuartile`, `complete` |
| `ad_id`   | UUID                   | The ad whose event is being reported          |
| `imp_id`  | UUID                   | Stamped by the adserver into the VAST URLs    |
| `exp`     | unix-seconds integer   | When the URL expires (signed)                 |
| `sig`     | 32-char lowercase hex  | `HMAC-SHA256(TRACKING_SECRET, "ad_id\|imp_id\|event\|exp")` truncated |

When `TRACKING_SECRET` is empty (dev only), `sig` + `exp` checks are
skipped.

### Response

```
Content-Type: image/gif
Cache-Control: no-store, max-age=0
Pragma: no-cache
```

The body is always the 43-byte canonical 1×1 transparent GIF89a.

### Status codes

| Status | Meaning                                                       |
|--------|---------------------------------------------------------------|
| `200`  | Event was recorded (counter INCR'd, dedupe key set)           |
| `204`  | Event was rejected (invalid sig / missing param / duplicate (imp_id, event)) |

Bytes returned are identical for both — players never see a 4xx. The
difference between `200` and `204` is only visible to ops via metrics
or logs.

### Idempotency

Each `(imp_id, event)` tuple is recorded once per 24 h (the
`TRACKING_TOKEN_TTL` window). Repeated fires return `204` and do not
INCR the counter.

### Curl example

```bash
# Real URLs come out of /vast — this is for the shape only.
curl -i 'http://localhost:8088/track?event=impression&ad_id=9a8b…&imp_id=…&exp=1747400000&sig=abc…'
```

---

## `GET /healthz`

Liveness + readiness probe. Returns `200 ok` once the registry
snapshot has loaded successfully at least once; `503 not ready`
otherwise.

Docker compose uses this as the adserver healthcheck. Kubernetes users
typically wire it as both `livenessProbe` and `readinessProbe`.

```bash
curl http://localhost:8088/healthz
# ok
```

---

## `GET /metrics` (adserver, on `PORT`)

Prometheus exposition format. Mounted outside the per-IP rate limiter
so scrapes are never throttled.

Metric catalog:

| Name                                       | Type      | Labels         | Meaning |
|--------------------------------------------|-----------|----------------|---------|
| `oas_http_requests_total`                  | counter   | route, code    | Every adserver request, route = chi route pattern (bounded cardinality) |
| `oas_http_request_duration_seconds`        | histogram | route          | End-to-end request duration (0.5ms→5s buckets) |
| `oas_vast_responses_total`                 | counter   | type           | `type ∈ {inline, empty}` — composition of `/vast` responses |
| `oas_budget_rejections_total`              | counter   | -              | Candidate ads dropped by campaign budget |
| `oas_freq_rejections_total`                | counter   | -              | Candidate ads dropped by per-user freq cap |
| `oas_snapshot_ads`                         | gauge     | -              | Active ads in the current snapshot |
| `oas_snapshot_load_duration_seconds`       | histogram | -              | Snapshot reload latency (1ms→16s buckets) |
| `oas_track_events_total`                   | counter   | event, status  | `/track` ingestion outcomes — `status ∈ {ok, duplicate, invalid}` |

Plus the standard `go_*` runtime metrics.

Scrape config (Prometheus):

```yaml
scrape_configs:
  - job_name: openadsource-adserver
    static_configs:
      - targets: ['adserver:8080']
```

---

## `GET /metrics` (worker, on `METRICS_PORT`)

Dedicated listener on `:9100` (configurable via `METRICS_PORT`). Same
exposition format, distinct collector set:

| Name                                  | Type      | Labels  | Meaning |
|---------------------------------------|-----------|---------|---------|
| `oas_worker_tick_duration_seconds`    | histogram | -       | End-to-end duration of one drainer tick |
| `oas_worker_ticks_total`              | counter   | result  | `result ∈ {ok, locked, error}` — tick outcomes |
| `oas_worker_drained_total`            | counter   | event   | Per-event count drained from Redis into Postgres |
| `oas_worker_campaigns_paused_total`   | counter   | -       | Campaigns paused on budget exhaustion |

Plus the standard `go_*` runtime metrics.

Scrape config:

```yaml
  - job_name: openadsource-worker
    static_configs:
      - targets: ['worker:9100']
```

Cardinality is bounded across the catalog — there are no labels keyed
off user IDs, IPs, or raw URLs.

---

## Rate limiting

Per-IP token bucket using `golang.org/x/time/rate`. Defaults:

| Endpoint  | Sustained rate    | Burst   | Env override                                |
|-----------|-------------------|---------|---------------------------------------------|
| `/vast`   | 100 req/s         | 200     | `RATE_LIMIT_VAST_RPS`, `RATE_LIMIT_VAST_BURST` |
| `/track`  | 200 req/s         | 400     | `RATE_LIMIT_TRACK_RPS`, `RATE_LIMIT_TRACK_BURST` |
| `/healthz`, `/metrics` | unlimited     | -       | -                                           |

Set the rate to `0` to disable. Over-limit responses are:

- `/vast`: a normal empty VAST (`<VAST version="4.2"/>`). Players see a
  natural no-fill.
- `/track`: a normal 1×1 transparent GIF with `200 OK`. Players see no
  difference.

The limiter never emits `429`. This is intentional — limiter feedback
at the protocol layer would either be ignored (players retry blindly)
or amplified (frantic retry loops).

---

## CORS

`/vast` and `/track` are CORS-open:

```
Access-Control-Allow-Origin: *
Access-Control-Allow-Methods: GET, OPTIONS
Access-Control-Max-Age: 86400
```

`/healthz` and `/metrics` are server-to-server — no CORS headers.

---

## Versioning

OpenAdSource follows [Semantic Versioning](https://semver.org/) on the
HTTP contract. Breaking changes to `/vast` parameters, `/track`
signing, or metric names happen only on major-version bumps and are
documented in `CHANGELOG.md` ahead of the release.

The XML output is VAST 4.2. Downgrading to VAST 3 / VAST 2 is an
operator-side concern (a proxy in front of the adserver can rewrite
the `version=` attribute and strip 4.2-only elements if you have
inventory that requires it).
