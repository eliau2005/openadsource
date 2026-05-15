# Self-Hosting

This is the operator's cookbook: env vars, TLS, GeoIP setup, backup +
restore, scaling, and the upgrade path. The architecture overview lives
in `architecture.md`; the formal HTTP contract lives in `api.md`.

---

## Quickstart

```bash
git clone https://github.com/eliau2005/openadsource
cd openadsource
cp .env.example .env
# Edit .env — at minimum set JWT_SECRET, TRACKING_SECRET, and the Postgres
# / MinIO credentials.
docker compose up --build -d
```

Wait ~30 s for `migrate` to finish (it runs as a one-shot service and
exits), then:

- adserver: <http://localhost:8088/healthz>
- dashboard: <http://localhost:3000>
- adserver metrics: <http://localhost:8088/metrics>
- worker metrics: with the dev overlay, <http://localhost:9100/metrics>
- test player: <http://localhost:8090>
- MinIO console: <http://localhost:9001>

The first dashboard user is created via the **/setup** route — visit it
once and pick a password. After that the route is disabled.

To seed a demo campaign with one BYO-URL ad and one MinIO-hosted ad:

```bash
docker compose --profile seed up seed
```

---

## Environment variables

Every variable comes from `.env` (compose passes it through to the
containers). Defaults are wired in `server/internal/config/config.go`
and `dashboard/lib/env.ts`.

| Variable                     | Default                                     | Effect when unset                                 |
|------------------------------|---------------------------------------------|---------------------------------------------------|
| `POSTGRES_HOST/PORT/USER/PASSWORD/DB` | `postgres / 5432 / openadsource / placeholder / openadsource` | Postgres bootstrap + `DATABASE_URL` interpolation |
| `DATABASE_URL`               | composed from the above                     | Required for adserver, worker, dashboard          |
| `REDIS_URL`                  | `redis://redis:6379/0`                      | Counters + locks; worker refuses to start without it |
| `JWT_SECRET`                 | `placeholder_value`                         | Dashboard refuses to issue cookies (logs warning) |
| `TRACKING_SECRET`            | `placeholder_value`                         | `/track` skips signature verification (dev only)  |
| `PUBLIC_BASE_URL`            | `http://localhost:8088`                     | Stitched into every URL inside emitted VAST and also gates the `Secure` cookie flag — `https://...` → cookies are `Secure` |
| `S3_ENDPOINT / REGION / BUCKET / ACCESS_KEY_ID / SECRET_ACCESS_KEY` | MinIO defaults | When any required field is empty, `S3Configured()` is false; `internal_s3` ads return no-fill |
| `S3_PUBLIC_BASE_URL`         | `http://localhost:9000/openadsource`        | Browser-resolvable URL prefix the adserver hands back as the VAST `MediaFile` URL |
| `S3_PUBLIC_ENDPOINT`         | `http://localhost:9000`                     | Dashboard upload presigner — the URL the *browser* PUTs against |
| `GEOIP_DB_PATH`              | `/data/GeoLite2-Country.mmdb`               | Geo resolver returns `country=""`; country-targeted ads can't match |
| `TRUSTED_PROXIES`            | `10.0.0.0/8,172.16.0.0/12,192.168.0.0/16`   | XFF header trust list (CIDR); off-network hops are treated as the source IP |
| `REGISTRY_REFRESH_INTERVAL`  | `30s`                                       | Snapshot TTL; pub/sub invalidations always force a reload regardless |
| `TRACKING_TOKEN_TTL`         | `24h`                                       | Lifetime of the signed pixel URL `exp` field      |
| `WORKER_INTERVAL`            | `30s`                                       | Worker tick cadence                               |
| `RATE_LIMIT_VAST_RPS / _BURST`  | `100 / 200`                              | Per-IP rate on `/vast`; `0` disables              |
| `RATE_LIMIT_TRACK_RPS / _BURST` | `200 / 400`                              | Per-IP rate on `/track`; `0` disables             |
| `RATE_LIMIT_MAP_CAP`         | `100000`                                    | Max distinct source IPs tracked at once           |
| `METRICS_PORT`               | `9100`                                      | Worker `/metrics` listener; adserver always serves `/metrics` on its own `PORT` |
| `LOG_LEVEL`                  | `info`                                      | `debug / info / warn / error`                     |
| `ADSERVER_HOST_PORT / DASHBOARD_HOST_PORT / TEST_PLAYER_HOST_PORT` | `8088 / 3000 / 8090` | Host-side port mappings |

Generate real secrets before going public:

```bash
openssl rand -hex 32           # JWT_SECRET
openssl rand -hex 32           # TRACKING_SECRET
openssl rand -base64 24 | tr -d '/+='  # POSTGRES_PASSWORD (alpha-only safe)
```

Update `DATABASE_URL` after rotating the Postgres password — the URL is
parsed separately from the host/port/etc fields.

---

## Putting it behind TLS

The adserver intentionally has no TLS termination logic — every
production deployment fronts it with a reverse proxy that already
handles cert renewal. The canonical nginx config:

```nginx
upstream openadsource_adserver { server 127.0.0.1:8088; keepalive 32; }
upstream openadsource_dashboard { server 127.0.0.1:3000; keepalive 32; }

server {
    listen 443 ssl http2;
    server_name ads.example.com;

    ssl_certificate     /etc/letsencrypt/live/ads.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/ads.example.com/privkey.pem;

    # The adserver writes the user_id cookie + every tracking pixel URL
    # off PUBLIC_BASE_URL, so set that env var to https://ads.example.com.
    location /vast    { proxy_pass http://openadsource_adserver; }
    location /track   { proxy_pass http://openadsource_adserver; }
    location /healthz { proxy_pass http://openadsource_adserver; }

    # /metrics is server-to-server — restrict to your Prometheus subnet.
    location /metrics {
        allow 10.20.30.0/24;
        deny all;
        proxy_pass http://openadsource_adserver;
    }

    # XFF chain — required for the rate limiter + targeting IP resolver.
    proxy_set_header X-Forwarded-For   $proxy_add_x_forwarded_for;
    proxy_set_header X-Real-IP         $remote_addr;
    proxy_set_header X-Forwarded-Proto https;
    proxy_set_header Host              $host;
}

server {
    listen 443 ssl http2;
    server_name dashboard.example.com;
    ssl_certificate     /etc/letsencrypt/live/dashboard.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/dashboard.example.com/privkey.pem;

    location / { proxy_pass http://openadsource_dashboard; }
    proxy_set_header X-Forwarded-Proto https;
    proxy_set_header Host              $host;
}
```

After the proxy is in place:

1. Set `PUBLIC_BASE_URL=https://ads.example.com` in `.env`. The
   adserver detects the `https://` scheme and writes cookies with
   `Secure`. The dashboard's session cookie does the same.
2. Add the proxy's network range to `TRUSTED_PROXIES` so the rate
   limiter and the targeting IP resolver key off the real client IP
   instead of the proxy.
3. `docker compose up -d --force-recreate adserver dashboard worker`
   to pick the new env up.

---

## GeoIP setup

OpenAdSource reads the MaxMind **GeoLite2-Country** database for
country targeting. The free version is enough.

1. Sign up at <https://www.maxmind.com/en/geolite2/signup>.
2. Generate a license key and download the latest
   `GeoLite2-Country.mmdb` (or use their `geoipupdate` daemon for
   automated refreshes).
3. Place it where the container can read it. The compose file mounts
   `./data` to `/data` in the adserver, so:

   ```bash
   mkdir -p data && mv ~/Downloads/GeoLite2-Country.mmdb data/
   ```

4. `GEOIP_DB_PATH=/data/GeoLite2-Country.mmdb` is already the default;
   no env change needed.
5. Restart the adserver. `oas_snapshot_load_duration_seconds_count`
   should tick on the next refresh; targeting rules referencing
   `country` will now match.

When the file is missing, the geo resolver returns `country=""` and
country-targeted ads simply don't match — no error, no crash.

---

## Backup + restore

Three things have durable state: **Postgres** (the catalog +
`daily_stats`), **MinIO** (creative bytes), and **Redis** (in-flight
counters). Redis is treated as ephemeral — losing it costs at most one
worker tick of counters.

### Postgres

The container exposes `pg_dump`:

```bash
docker compose exec postgres \
  pg_dump -U openadsource -Fc openadsource \
  > backups/openadsource-$(date +%Y%m%d-%H%M).dump
```

Restore into an empty instance:

```bash
docker compose exec -T postgres \
  pg_restore -U openadsource -d openadsource --clean --if-exists \
  < backups/openadsource-20260515-1200.dump
```

For continuous archival use `pg_basebackup` + WAL shipping (out of
scope for this doc).

### MinIO

Bind the bucket to a host directory in `docker-compose.yml` (the
provided file already does — `./data/minio:/data`). Snapshot that
directory with `rsync` / `restic` / `borg` on whatever cadence makes
sense. The bucket layout is content-addressed (UUID-named keys), so the
files alone are enough to restore.

For cross-region backup, MinIO has a `mc mirror` command that copies
to any S3-compatible target — keep two buckets in sync if needed.

### Redis

Don't. Counters are designed to survive Redis loss:

- `track:<imp_id>:<event>` keys are dedupe markers with a 24 h TTL.
  Losing them means at most 24 h of duplicate fires are no longer
  deduped — a one-time over-count.
- `ad:<ad_id>:event:<ev>:<date>` keys are drained every
  `WORKER_INTERVAL`. The worst case is `WORKER_INTERVAL` of unreported
  events — the next tick continues.
- `campaign:<id>:imps_total` counters get reconciled on the next
  worker tick; the worker re-derives the pause state from Redis
  totals, so a wipe means a freshly drained Postgres `daily_stats`
  won't trigger a pause until Redis re-accumulates. Operators
  optionally `SET` the recovered total from `daily_stats` to skip the
  re-accumulation lag.

If you do want a Redis snapshot, run with `--save 900 1 --save 300 10`
and bind-mount `/data` — the RDB snapshots land in there.

---

## Scaling

The adserver and worker are designed to scale horizontally without
coordination.

### Adserver

Stateless. Run N replicas behind any L4 / L7 load balancer (no sticky
sessions needed). Each replica holds its own in-memory snapshot and
subscribes independently to the registry-invalidate pub/sub channel,
so dashboard edits land on every replica within milliseconds.

Each replica opens its own pgx pool — bump `pgxpool` defaults if your
fleet hits Postgres connection limits before it hits CPU. A small
PgBouncer in `transaction` mode is the usual answer.

### Worker

Lock-coordinated. `worker.NewLock(rc, 60_000)` uses Redis
`SET NX PX 60000` so exactly one replica drains per tick. N>1 replicas
are fine — they share the work by simply taking turns; if the leader
dies mid-tick, the 60 s TTL releases the lock and the next tick is
picked up by whoever wins the next race.

Don't run a worker as part of an autoscaled deployment that scales to
zero — at least one worker process must be running for counters to
ever flush.

### Bottlenecks (in order)

1. **Postgres** for the dashboard write path and `daily_stats` upserts.
   Vertical scale + read replicas for the dashboard. The adserver
   hot path doesn't read Postgres so it's unaffected.
2. **Redis** for the budget + freq Lua scripts. EVALSHA is cheap but
   each `/vast` request that has caps is one round-trip. Redis Cluster
   works (keys hash on `ad_id` / `campaign_id`); pgx and the freq /
   budget enforcers are cluster-compatible out of the box.
3. **Network egress from the resolver** when serving via S3. A CDN in
   front of MinIO is the standard answer.

### Caching layer

Don't put a cache in front of `/vast`. Each response is unique per
`imp_id`, country, device, and player — caching it is incorrect. A CDN
*can* sit in front, but configure it to bypass cache on `/vast` and
`/track` (the test player's example config does this).

---

## Upgrade path

```bash
git pull
docker compose up --build -d
```

The `migrate` one-shot service runs every time, applying any new SQL
migrations idempotently. The adserver + worker images rebuild from
source. The dashboard image rebuilds with `next build`.

Zero-downtime upgrades: do a rolling restart at the load balancer
layer — `docker compose up -d --no-deps --scale adserver=N+1` then
drop the old replicas. Snapshot reloads tolerate transient Postgres
unavailability so you can take the database down too if needed (within
a minute or so — beyond that snapshots will be stale).

Database migrations are forward-only. Roll forward, never roll back
the schema in production.

---

## Observability

Each binary exposes a Prometheus `/metrics` endpoint. Sample
`prometheus.yml` scrape config:

```yaml
scrape_configs:
  - job_name: openadsource-adserver
    static_configs:
      - targets: ['adserver:8080']

  - job_name: openadsource-worker
    static_configs:
      - targets: ['worker:9100']
```

Metric catalog: see `api.md`. The `oas_*` namespace covers HTTP
counters, VAST response composition, budget + freq rejections,
snapshot gauges, track-event counters, and worker tick / drain
counters. Standard `go_*` runtime metrics are exposed alongside.

A Grafana dashboard JSON is not bundled in v1.0 — build one from the
metric names in `api.md` and ship it back upstream as a contribution.

---

## Cookie + cross-origin notes

- `oas_uid` is the adserver's user-identification cookie, set on first
  `/vast` from each browser. `HttpOnly`, `SameSite=Lax`, `Path=/`. The
  `Secure` flag is set when `PUBLIC_BASE_URL` starts with `https://`.
- `oas_session` is the dashboard's session cookie. Same rules as
  `oas_uid` — `Secure` derived from the same scheme check.
- `Access-Control-Allow-Origin: *` is set on `/vast` and `/track`. The
  endpoints are deliberately public — players load them from any
  origin. The dashboard is not CORS-exposed.

---

## Security disclosure

The repository ships an empty `SECURITY.md` so the file path is
already wired into GitHub's "Security policy" link. Paste your real
disclosure address and policy into that file before publishing.

---

## Common operational tasks

| Task | Command |
|------|---------|
| Tail the adserver | `docker compose logs -f adserver` |
| Tail the worker | `docker compose logs -f worker` |
| Force a snapshot reload | `docker compose exec redis redis-cli PUBLISH oas:registry:invalidate dashboard-manual` |
| Inspect a Redis counter | `docker compose exec redis redis-cli GET "ad:<uuid>:event:impression:$(date -u +%F)"` |
| Inspect the rate-limit map size | scrape `/metrics` (the map is internal — `oas_http_requests_total{code="200",route="/vast"}` is the user-visible signal) |
| Reset the demo dataset | `docker compose down -v && docker compose up --build -d && docker compose --profile seed up seed` |
| Promote a `migrate` retry on a fresh Postgres | the service is one-shot; re-run with `docker compose up migrate` |
