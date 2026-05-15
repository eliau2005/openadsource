# OpenAdSource — Development Roadmap & Tasks

## Context

The repository currently contains only `ad-server-open-surce-MVP.md` — a specification for a self-hosted, open source video Ad Server supporting VAST 4.x, ad scheduling (pre/mid/post-roll), and frequency capping. There is no code yet.

This plan converts that spec into an executable roadmap: a phased sequence of concrete tasks that take the project from an empty repo to a v1.0 OSS release. The goal is a self-hostable system that runs end-to-end via `docker compose up`, with no required external SaaS dependencies, that another developer or small ad ops team could realistically deploy.

The plan extends the spec's four MVP phases with a **Phase 0** (project scaffolding before any feature work) and a **Phase 5** (hardening + v1.0 release), since the spec stops at feature completion and doesn't cover what's needed to actually ship an OSS project.

---

## Locked Tech Decisions

| Area | Choice | Rationale |
|---|---|---|
| Repo layout | Monorepo: `/server` (Go), `/dashboard` (Next.js), `/infra`, `/docs` | One clone, one release line |
| Delivery engine | Go 1.22+, `go-chi/chi` v5, `net/http` | Idiomatic, low overhead on VAST hot path |
| DB driver | `jackc/pgx` v5 + `sqlc` for typed queries | No ORM weight on the delivery hot path |
| Migrations | `golang-migrate/migrate` | Plain SQL, runs as a container init job |
| Cache/counters | Redis 7 via `redis/go-redis` v9 | Atomic INCR for frequency caps + rolling counters |
| Geo-IP | MaxMind GeoLite2 (`oschwald/geoip2-golang`) | In-memory mmdb lookup, no network call |
| Object storage | **Optional** — MinIO bundled in compose; AWS SDK v2 S3 client fully env-configurable | Storage is opt-in. Default flow is **BYO-URL**: users paste a public MP4/HLS URL (Bunny.net, Cloudflare Stream, any CDN) and pay zero egress. S3 is for users who want self-contained hosting; the client is endpoint/region/path-style agnostic so it works against S3, R2, Bunny Edge Storage, Wasabi, Backblaze B2, or self-hosted MinIO via env vars (`S3_ENDPOINT`, `S3_REGION`, `S3_BUCKET`, `S3_FORCE_PATH_STYLE`, `S3_PUBLIC_BASE_URL`). |
| Decision-engine hot path | In-memory campaign/targeting snapshot, TTL-refreshed; Redis for atomic counters only | No Postgres reads on `/vast`. Target: sub-50ms P99 at high RPS |
| Dashboard | Next.js 15 (App Router) + TypeScript + Tailwind + shadcn/ui | Matches spec; modern App Router patterns |
| Dashboard auth | Local email/password, bcrypt + JWT (HTTP-only cookie) | No external SaaS dependency for self-hosters |
| VAST generation | Templated XML via `encoding/xml` structs, targeting VAST 4.2 | Stable, broadly supported by players |
| License | Apache 2.0 | Permissive + patent grant — standard for OSS infra |
| Container orchestration | `docker compose` for local + reference prod deploy | Per the spec |

---

## Repository Structure (target)

```
/
├── README.md                  Project intro, screenshot, quickstart
├── LICENSE                    Apache 2.0
├── CONTRIBUTING.md            Dev setup, commit style, PR flow
├── CODE_OF_CONDUCT.md         Contributor Covenant
├── SECURITY.md                Disclosure policy + contact
├── docker-compose.yml         Default self-host stack
├── docker-compose.dev.yml     Override for hot-reload dev
├── .env.example               All required env vars, documented
├── .github/
│   ├── workflows/             ci.yml, release.yml, codeql.yml
│   └── ISSUE_TEMPLATE/
├── server/                    Go delivery engine
│   ├── cmd/
│   │   ├── adserver/main.go       HTTP server entrypoint
│   │   └── worker/main.go         Redis→Postgres flush worker
│   ├── internal/
│   │   ├── vast/                  VAST 4.2 XML builder + tests
│   │   ├── delivery/              /vast handler, decision engine
│   │   ├── tracking/              /track handler, event sink
│   │   ├── capping/               Frequency cap Redis logic
│   │   ├── targeting/             Geo-IP + device parsing
│   │   ├── selection/             Eligible-ad filter + ranking (memory-only hot path)
│   │   ├── registry/              In-memory campaign/targeting snapshot + TTL refresher
│   │   ├── storage/               Optional S3 client wrapper (env-driven, S3-compatible)
│   │   ├── db/                    pgx pool, sqlc generated code
│   │   ├── config/                env loading + validation
│   │   └── httplog/               request logging middleware
│   ├── migrations/                *.up.sql / *.down.sql
│   ├── queries/                   *.sql for sqlc
│   ├── sqlc.yaml
│   ├── go.mod
│   └── Dockerfile
├── dashboard/                 Next.js admin UI
│   ├── app/
│   │   ├── (auth)/login/page.tsx
│   │   ├── (app)/
│   │   │   ├── campaigns/         List + new + [id] edit
│   │   │   ├── advertisers/
│   │   │   ├── creatives/
│   │   │   └── reports/[campaignId]/page.tsx
│   │   ├── api/
│   │   │   ├── auth/[...]/route.ts
│   │   │   ├── upload/route.ts        Presigned S3 PUT
│   │   │   └── ...resource routes
│   │   └── layout.tsx
│   ├── lib/
│   │   ├── db.ts                  pg or Drizzle client
│   │   ├── auth.ts                JWT issue/verify
│   │   └── s3.ts                  Presigned URL helper
│   ├── components/                shadcn-generated + custom
│   ├── package.json
│   ├── tailwind.config.ts
│   └── Dockerfile
├── infra/
│   ├── nginx/                     Reference reverse proxy config
│   └── grafana/                   Optional dashboards JSON
├── docs/
│   ├── architecture.md
│   ├── self-hosting.md
│   ├── vast-integration.md        Player snippet examples
│   └── api.md                     /vast, /track contract
└── examples/
    └── test-player/               Plain HTML page wiring video.js to /vast
```

---

## Phase 0 — Project Foundation (no feature code yet)

**Goal:** A new contributor can clone, `docker compose up`, and reach an empty dashboard within 5 minutes.

- [ ] Initialize monorepo: top-level `README.md`, `LICENSE` (Apache 2.0), `.gitignore`, `.editorconfig`, `.gitattributes`.
- [ ] Add `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md` (Contributor Covenant 2.1), `SECURITY.md`.
- [ ] Scaffold `/server` Go module: `go mod init github.com/<org>/openadsource/server`, install chi, pgx, redis, aws-sdk-go-v2, geoip2-golang, zerolog, testify.
- [ ] Scaffold `/dashboard`: `npx create-next-app@latest dashboard --ts --tailwind --app --eslint`, install shadcn/ui, lucide-react, zod, react-hook-form, drizzle-orm (or pg).
- [ ] Write `docker-compose.yml` with services: `postgres:16`, `redis:7`, `minio`, `adserver`, `dashboard`, `worker`. Networks + named volumes for data persistence.
- [ ] Write `docker-compose.dev.yml` override mounting source for hot reload (`air` for Go, `next dev` for Next.js).
- [ ] Write `.env.example` documenting every variable: `POSTGRES_*`, `REDIS_URL`, `S3_*`, `JWT_SECRET`, `GEOIP_DB_PATH`, `PUBLIC_BASE_URL`, etc.
- [ ] `/server/Dockerfile` — multi-stage build (golang:1.22 → distroless/static).
- [ ] `/dashboard/Dockerfile` — multi-stage (node:20 → next standalone).
- [ ] GitHub Actions `.github/workflows/ci.yml`:
  - lint Go (`golangci-lint`), test (`go test ./...`)
  - lint TS (`eslint`), typecheck (`tsc --noEmit`), test (`vitest`)
  - build both Docker images
- [ ] Health endpoints stubbed: `GET /healthz` (server), `/api/healthz` (dashboard) — used by compose healthchecks.
- [ ] Bootstrap migrations tooling: `migrations/0001_init.sql` empty placeholder; `migrate` container service runs on startup.

**Exit criteria:** `docker compose up --build` brings up all services, healthchecks pass, `curl localhost:8080/healthz` returns 200, dashboard renders a placeholder page.

---

## Phase 1 — VAST Delivery Core (spec Phase 1)

**Goal:** A real video player can request an ad and play it.

- [ ] Migrations: `0002_campaigns_ads.sql` creating `advertisers`, `campaigns`, `ads`, `cap_rules` per the spec schema. Add `created_at`/`updated_at`, indexes on `campaign_id`, `status`, `(position_type, status)`. **Extend the spec's `ads` schema** with storage-agnostic columns so creatives can be either externally hosted or internally stored:
  - `media_source` TEXT NOT NULL CHECK (`media_source` IN ('external_url','internal_s3')) DEFAULT 'external_url'
  - `media_url` TEXT NOT NULL — for `external_url` this is the fully-qualified public URL (CDN, Bunny, Cloudflare Stream, raw HTTPS MP4/HLS); for `internal_s3` this is the S3 object key
  - `media_mime` TEXT NOT NULL — e.g. `video/mp4`, `application/x-mpegURL`, `application/dash+xml`
  - `media_duration_ms` INTEGER NULL — populated by ffprobe (internal) or manual entry (external)
  - `media_width` / `media_height` INTEGER NULL
  - `media_bitrate_kbps` INTEGER NULL
  The existing `video_url` column from the spec is replaced by `media_url`; document the migration path in the file header.
- [ ] sqlc config + initial queries: `GetAdByID`, `GetActiveAdsByPosition`, `IncrementImpressionsServed`.
- [ ] `internal/vast/`: Go structs mirroring VAST 4.2 (`VAST`, `Ad`, `InLine`, `Creatives`, `Linear`, `MediaFiles`, `TrackingEvents`, `VideoClicks`).
- [ ] `internal/vast/builder.go`: `Build(ad Ad, requestCtx Context) (string, error)` returning serialized XML with correct namespace + version attributes.
- [ ] Golden-file tests in `internal/vast/builder_test.go`: snapshot known good XML; validate against the IAB VAST 4.2 XSD (vendored).
- [ ] `internal/delivery/handler.go`: `GET /vast?ad_id=...` → loads ad → emits XML with `Content-Type: application/xml`. Returns valid empty `<VAST>` (no-fill) on missing/expired ad — never 500 to a player.
- [ ] `internal/storage/resolver.go`: a single `ResolveMediaURL(ad) (string, mime string)` function that branches on `ad.media_source`:
  - `external_url` → pass `media_url` straight through (zero work, zero egress through us).
  - `internal_s3` → if `S3_PUBLIC_BASE_URL` is set, return `<base>/<key>`; otherwise return a presigned GET URL (TTL ~1h, cached in-process to avoid signing per request).
  The VAST builder is unaware of storage backend — it only consumes the resolved URL + mime, and emits `<MediaFile type="video/mp4">` for progressive MP4 or `<MediaFile type="application/x-mpegURL" delivery="streaming">` for HLS.
- [ ] `internal/storage/s3.go`: thin AWS SDK v2 wrapper. **Constructed only when S3 env vars are present** — the server must boot and serve `external_url` creatives even if no S3 is configured. Endpoint/region/path-style all driven by env vars per the tech decisions table.
- [ ] `cmd/adserver/main.go`: wire chi router, middleware (recover, request ID, structured logs, CORS), graceful shutdown on SIGTERM.
- [ ] Seed script `server/scripts/seed.go`: insert one advertiser, one campaign, one ad pointing at a sample MP4 in MinIO.
- [ ] `examples/test-player/index.html`: video.js or hls.js page configured to call `/vast?ad_id=<seeded-id>` so the developer can visually confirm ad playback.

**Exit criteria:** Open `examples/test-player/index.html`, an ad plays from MinIO, browser network tab shows valid VAST XML.

---

## Phase 2 — Campaign Management UI (spec Phase 2)

**Goal:** A non-engineer can create a campaign, upload a creative, and see it served.

- [ ] **Dashboard auth**:
  - `users` table migration (id, email, password_hash, role, created_at).
  - `lib/auth.ts`: bcrypt hash/verify, JWT issue/verify, HTTP-only secure cookie.
  - `/(auth)/login/page.tsx` + `/api/auth/login/route.ts`, `/api/auth/logout/route.ts`.
  - `middleware.ts` protecting `/(app)/*` routes; redirect unauthenticated to `/login`.
  - First-run bootstrap: if zero users, dashboard renders a "create admin" form instead of login.
- [ ] **Resource CRUD** (Next.js Server Actions or Route Handlers, Drizzle/pg against the same DB the Go server uses):
  - Advertisers: list / create / edit / archive.
  - Campaigns: list / create / edit (name, advertiser, dates, total_budget_impressions, status).
  - Ads: list / create / edit (campaign, position_type, mid_roll_offset, landing_page_url, priority).
- [ ] **Creative ingestion — two paths, equal first-class support**:
  - **Path A — Bring Your Own URL (default, recommended in docs)**: form field accepts a public MP4/HLS URL. Backend issues a `HEAD` (and a small ranged `GET` if HEAD is denied) to validate: reachable, `Content-Type` matches an allow-list (`video/mp4`, `application/x-mpegURL`, `application/dash+xml`), `Content-Length` under a configurable cap, served over HTTPS. Stores `media_source='external_url'`, `media_url=<input>`, `media_mime=<probed>`. Duration/dimensions/bitrate are optional (fillable manually); if the URL is a same-origin MP4, attempt a remote `ffprobe -i <url>` in a worker job to populate them. **No bytes ever traverse our server.**
  - **Path B — Upload to internal storage (optional, only when S3 is configured)**: `POST /api/upload` returns a presigned PUT URL; client uploads directly to MinIO/S3/R2/Bunny/Wasabi; on success, POSTs metadata back. Server runs `ffprobe` on the resulting object. Stores `media_source='internal_s3'`, `media_url=<key>`. If no S3 env vars are configured, this path is hidden in the UI and the API returns 501.
  - Dashboard UI: a single "New Creative" form with a tab/segmented control switching between **"Use existing URL"** (default) and **"Upload file"** (disabled with a tooltip linking to docs when S3 isn't configured).
  - In-browser `<video>` preview before save works for both paths.
  - SSRF guard on Path A: reject `media_url` resolving to private/loopback/link-local IPs; reject non-HTTPS in production mode.
- [ ] **shadcn/ui components**: Data tables (TanStack Table), forms (react-hook-form + zod), toasts, dialogs. Pull components via `npx shadcn@latest add ...`.
- [ ] **Layout & nav**: Sidebar (Campaigns / Advertisers / Creatives / Reports / Settings), topbar with user menu.
- [ ] Empty/loading/error states for every list page.
- [ ] **Tests**: Vitest unit tests for `lib/auth.ts`; Playwright smoke test for login → create campaign → upload creative.

**Exit criteria:** Logged-in user creates a campaign + uploads a creative through the UI; the seeded ad from Phase 1 is replaced by a real upload and still serves correctly through `/vast`.

---

## Phase 3 — Decision Engine (spec Phase 3)

**Goal:** `/vast` accepts targeting parameters and returns the best eligible ad, not just a fixed ID — **without ever touching Postgres or disk on the hot path.**

### Hard performance contract (non-negotiable for this phase)

`/vast` handling must satisfy all of the following:

- **Zero Postgres queries per request.** The selection set is served from an in-process, in-memory snapshot. Postgres is only read at snapshot-refresh time, never on the request path.
- **Zero disk reads per request.** The GeoIP `.mmdb` is mmap'd and held in memory; the snapshot is a plain Go struct graph; there is no file I/O in the request lifecycle.
- **Redis is used only for atomic counters** (frequency cap check, budget INCR). These are single round-trip operations — no Lua scripts that scan keys, no `KEYS *`, no transactions touching more than one or two keys.
- **All persistent statements that *do* run (in the refresher and worker, never on the hot path) use prepared statements** via sqlc's generated code with a long-lived `pgxpool`.
- Target: **P99 < 50ms at 1k RPS on a 2-core VM** (verified in Phase 5 load test).

- [ ] **In-memory registry** (`internal/registry/`):
  - `Snapshot` struct holds all active, in-flight campaigns + their ads + targeting + cap rules, indexed for O(1)/O(log n) lookup:
    - `byPosition map[string][]*Ad` (pre/mid/post → eligible ads)
    - `byCountry map[string]bitset` (ISO country → bitset of eligible ad indices)
    - `byDevice map[string]bitset`
    - `byID map[uuid.UUID]*Ad` for direct-test lookups
    - Pre-computed time-window booleans refreshed at TTL tick (so we don't compare `time.Now()` against per-ad dates beyond what's already in the snapshot's eligible set).
  - `Loader` runs a single prepared-statement query (sqlc-generated) joining `campaigns + ads + campaign_targeting + cap_rules`, builds a fresh `Snapshot`, and atomically swaps it via `atomic.Pointer[Snapshot]`.
  - **Refresh strategy**: TTL-based, default `REGISTRY_REFRESH_INTERVAL=30s`. Optional Redis pub/sub channel `oas:registry:invalidate` published by the dashboard on any campaign/ad/targeting/cap mutation, which forces an immediate reload. This gives 30s worst-case staleness with sub-second propagation on the happy path.
  - **Boot behavior**: server fails health-ready until the first snapshot loads successfully; subsequent loader errors log and retain the previous snapshot (fail-open on staleness, never serve from nothing).
- [ ] **Endpoint contract change**: `GET /vast?pos=pre|mid|post&offset=<sec>&user_id=<id>&country=<auto>&device=<auto>` — `ad_id` becomes optional and is only honored for testing.
- [ ] **Targeting extraction** (`internal/targeting/`):
  - Resolve client IP (respect `X-Forwarded-For` with trusted proxy list).
  - `geo.go`: load GeoLite2-Country.mmdb at startup, lookup → ISO country code.
  - `device.go`: minimal UA parser (mobile / tablet / desktop / ctv) — start with a regex table, swap to `mileusna/useragent` if needed.
- [ ] **Targeting schema extension** — migration `0003_targeting.sql`:
  - `campaign_targeting` table: `campaign_id`, `countries text[]`, `devices text[]` (nullable = all).
- [ ] **Selection logic** (`internal/selection/`) — operates **exclusively against the in-memory `Snapshot`**, never against Postgres:
  - `Eligible(snapshot *Snapshot, req Request) []*Ad`: start from `snapshot.byPosition[req.Pos]`, intersect with `byCountry[req.Country]` and `byDevice[req.Device]` bitsets (single AND across two `*big.Int` / hand-rolled `[]uint64`). For mid-roll, narrow further to ads whose `mid_roll_offset` is within ±2s of the requested offset.
  - Ranking: stable sort by `priority DESC`, then **weighted-random pacing** — weights derived from `(remaining_budget / hours_remaining_in_campaign)` precomputed on the snapshot, so the selector picks via a single `rand.Float64()` against a precomputed cumulative weight table. No per-request math beyond a binary search.
  - **Budget & cap checks are the only Redis calls on the hot path**, executed *after* memory-only filtering has narrowed the candidate set to ≤ ~10 ads — usually 1. This keeps Redis RTTs bounded regardless of campaign count.
  - Allocation discipline: selection must not allocate per request beyond a small reusable `[]*Ad` candidate slice from a `sync.Pool`. Verified with `go test -benchmem` in `selection_bench_test.go` (target: ≤ 2 allocs/op).
- [ ] **Budget enforcement**:
  - Atomic check-and-increment in Redis: `INCR campaign:{id}:impressions_total` plus a compare against a per-campaign `budget_remaining` key seeded by the worker on each tick. Implemented as a tiny KEYS-free Lua script (`EVAL`) cached via `EVALSHA` so it's a single round trip with no key scan.
  - If `INCR` exceeds budget, the script `DECR`s back and returns -1 → selector drops the candidate and tries the next-ranked ad from its in-memory shortlist (no extra Postgres call).
  - The worker reconciles Redis totals → `daily_stats` and updates the snapshot's `budget_remaining` once per tick.
- [ ] **No-fill path**: when no ad qualifies, return a valid empty VAST document (never 204, never an error — players misbehave on both).
- [ ] **Tests**:
  - Table-driven Go tests covering each filter against fixture snapshots.
  - Property test ensuring the selector never returns an ad that fails any single filter.
  - Benchmark `BenchmarkSelect_1k_campaigns` asserting ≤ 50µs/op and ≤ 2 allocs/op against a synthetic snapshot of 1,000 campaigns × 5 ads, to catch perf regressions in CI.
  - Race test (`go test -race`) on the snapshot swap to confirm reads during refresh never see a torn pointer.
- [ ] **Dashboard targeting UI**: form fields on campaign edit page (multi-select country, device checkboxes). On save, the dashboard publishes to `oas:registry:invalidate` (or hits an internal `POST /internal/registry/reload` endpoint authenticated with a shared secret) so the snapshot picks up changes within ~1s instead of waiting for the next TTL tick.

**Exit criteria:** Two campaigns targeting different countries are configured. Curling `/vast` from different `X-Forwarded-For` IPs returns different ads. Setting a campaign budget to 1 returns no-fill on the second request.

---

## Phase 4 — Tracking & Frequency Capping (spec Phase 4)

**Goal:** Impressions, clicks, quartiles are reliably counted; users don't see the same ad more than the configured cap.

- [ ] **VAST tracking pixel embedding**: `vast/builder.go` injects `<Tracking event="...">` URLs pointing at `/track?event=<event>&ad_id=<id>&imp_id=<imp_id>&sig=<hmac>` for: `start`, `firstQuartile`, `midpoint`, `thirdQuartile`, `complete`. Also `<Impression>` and `<ClickTracking>`.
- [ ] **Signed impression IDs**: HMAC-SHA256 over `(ad_id|imp_id|expires)` with `TRACKING_SECRET`. Rejects forged/replayed events. `imp_id` is a UUID minted at `/vast` time.
- [ ] **`/track` handler** (`internal/tracking/`):
  - Validates signature + expiry.
  - Writes event to Redis: `INCR ad:{id}:event:{event}:{date}` plus an idempotency `SETNX track:{imp_id}:{event} 1 EX 86400` so duplicates from retries don't double-count.
  - Returns `204 No Content` with a 1×1 GIF body for image-pixel compatibility.
- [ ] **User identification & frequency capping** (`internal/capping/`):
  - User ID resolution order: explicit `user_id` query param → first-party cookie (`oas_uid`, set on `/vast`) → IP+UA hash fallback.
  - `cap_rules` table per the spec; campaign-level capping in MVP, ad-level optional.
  - Pre-selection check: Redis `INCR user:{uid}:ad:{ad_id}:count` with `EXPIRE = time_window_seconds`; if return value > `max_impressions`, exclude.
- [ ] **Worker** (`cmd/worker/main.go`):
  - Tick every 30s.
  - Drains Redis counters for the previous bucket into `daily_stats` rows (campaign_id, ad_id, date, impressions, clicks, q25, q50, q75, complete).
  - Reconciles `campaigns.total_budget_impressions` remaining; pauses campaigns that hit budget.
  - Single-writer guard via a Redis lock so multiple worker replicas are safe.
- [ ] **Reports dashboard** (`/reports/[campaignId]`):
  - Totals: impressions, clicks, CTR, completion rate.
  - Quartile funnel bar chart (`recharts` or `tremor`).
  - Daily timeseries (last 30 days, line chart).
  - Read directly from `daily_stats` for speed.
- [ ] **Tests**: integration test that fires a full `/vast` → `/track` flow against a real Postgres + Redis (testcontainers-go), asserts counts after worker flush.

**Exit criteria:** A test player playing an ad to completion produces 1 impression, 4 quartile events, and (on click) 1 click in the reports view. Configuring a cap of 2/day and refreshing 3 times returns no-fill on the third.

---

## Phase 5 — v1.0 Hardening & OSS Release

**Goal:** A reasonable engineer can self-host this in production without reading the source.

- [ ] **Security**:
  - Threat-model `/track` (replay, forgery, DoS) and `/vast` (SSRF via redirect URLs, XML billion-laughs is N/A since we only emit XML).
  - Rate limit `/vast` and `/track` per IP (`ulule/limiter` or Redis token bucket).
  - Enforce HTTPS-only cookies, set `Secure`, `SameSite=Lax`, `HttpOnly`.
  - Dependency scanning: Dependabot + `govulncheck` + `npm audit` in CI.
  - Add `SECURITY.md` with disclosure email and PGP key.
- [ ] **Observability**:
  - `/metrics` Prometheus endpoint on both server and worker. Counters: requests, no-fills, errors per route; histogram for request duration.
  - Optional Grafana dashboard JSON in `/infra/grafana/`.
  - Structured logs (zerolog) with trace IDs propagated to the dashboard via response header.
- [ ] **Performance**:
  - Load test `/vast` with `vegeta` or `k6`. Target: P99 < 50ms at 1k RPS on a 2-core box.
  - Add `pgxpool` tuning + Redis pipelining where applicable.
  - Profile with `pprof`; eliminate any per-request allocation hotspots in the VAST builder (reusable buffers).
- [ ] **Docs** (`/docs`):
  - `architecture.md` — diagram, request lifecycle, data flow.
  - `self-hosting.md` — production deployment with TLS (nginx + Let's Encrypt example), backup/restore, scaling notes.
  - `vast-integration.md` — copy-paste snippets for video.js, Shaka, hls.js, JW Player.
  - `api.md` — formal contract for `/vast` and `/track`.
  - Top-level `README.md` with hero screenshot, 60-second quickstart, feature checklist, license badge.
- [ ] **Releases**:
  - GoReleaser config for tagged binaries (linux/amd64, linux/arm64, darwin/arm64).
  - Multi-arch Docker images pushed to GHCR on tag: `ghcr.io/<org>/openadsource-server:vX.Y.Z`, `openadsource-dashboard:vX.Y.Z`.
  - SemVer + changelog (`CHANGELOG.md`, Keep a Changelog format).
  - `release.yml` workflow triggered by `v*` tags.
- [ ] **Project hygiene**:
  - Issue + PR templates.
  - "Good first issue" labels seeded with 5–10 small tasks.
  - A demo/landing page (optional: `dashboard/app/(marketing)/page.tsx` or a simple Astro site under `/site`).

**Exit criteria:** Tag `v1.0.0`, GitHub release publishes images + binaries, README quickstart works on a fresh VM in under 10 minutes.

---

## Critical Files (to be created / modified)

| Path | Purpose |
|---|---|
| `docker-compose.yml`, `docker-compose.dev.yml` | Phase 0 — full self-host stack |
| `.env.example` | Phase 0 — single source of truth for config |
| `server/cmd/adserver/main.go` | Phase 1 — HTTP server entrypoint |
| `server/cmd/worker/main.go` | Phase 4 — Redis→PG flush worker |
| `server/internal/vast/builder.go` | Phase 1 — VAST XML generation |
| `server/internal/delivery/handler.go` | Phase 1+3 — `/vast` endpoint |
| `server/internal/tracking/handler.go` | Phase 4 — `/track` endpoint |
| `server/internal/capping/redis.go` | Phase 4 — frequency cap logic |
| `server/internal/selection/select.go` | Phase 3 — ad picker (memory-only, bitset filters) |
| `server/internal/selection/selection_bench_test.go` | Phase 3 — perf regression guard |
| `server/internal/registry/snapshot.go` | Phase 3 — in-memory snapshot + atomic swap |
| `server/internal/registry/loader.go` | Phase 3 — prepared-statement loader + TTL/pubsub refresher |
| `server/internal/storage/resolver.go` | Phase 1 — branches external_url vs internal_s3 |
| `server/internal/storage/s3.go` | Phase 1 — optional, S3-compatible (env-driven) |
| `server/internal/targeting/geo.go` | Phase 3 — GeoLite2 lookup (mmap) |
| `server/migrations/00*.sql` | All phases — DB schema |
| `server/queries/*.sql` | sqlc-generated query layer |
| `dashboard/app/(app)/campaigns/**` | Phase 2 — campaign CRUD |
| `dashboard/app/(app)/reports/[campaignId]/page.tsx` | Phase 4 — reporting UI |
| `dashboard/lib/auth.ts`, `dashboard/middleware.ts` | Phase 2 — JWT auth |
| `dashboard/app/api/upload/route.ts` | Phase 2 — presigned upload |
| `examples/test-player/index.html` | Phase 1 — manual verification harness |
| `.github/workflows/{ci,release,codeql}.yml` | Phase 0 + 5 — automation |
| `docs/{architecture,self-hosting,vast-integration,api}.md` | Phase 5 — release docs |

---

## End-to-End Verification

For each phase, the bar is "demonstrable in `docker compose up`":

1. **Phase 0**: `docker compose up --build` — all containers healthy, `curl localhost:8080/healthz` returns 200, dashboard loads.
2. **Phase 1**: Run seed script, open `examples/test-player/index.html`, ad plays from MinIO. `curl localhost:8080/vast?ad_id=<id>` returns valid VAST 4.2 XML (validate with `xmllint --schema vast4.xsd`).
3. **Phase 2**: Log in, create advertiser → campaign → upload creative, verify the new ad serves via `/vast`. Playwright smoke test passes in CI.
4. **Phase 3**: Two campaigns with disjoint country targeting; `curl -H "X-Forwarded-For: 1.1.1.1" /vast` vs `8.8.8.8` returns the correct ad. Budget=1 returns no-fill on second request. Run with `POSTGRES_HOST` pointed at an unreachable address *after boot*: `/vast` keeps serving from the snapshot, proving zero-PG-on-hot-path. `BenchmarkSelect_1k_campaigns` passes its ≤ 50µs / ≤ 2 allocs budget. **BYO-URL verification**: a campaign whose creative is a public CDN URL (e.g. an MP4 on `archive.org`) serves a `<MediaFile>` pointing at that URL with no S3 service running in the compose stack.
5. **Phase 4**: Play ad to completion → check `/reports/[id]`: 1 impression, 1 q25, 1 q50, 1 q75, 1 complete. Set cap=2; third `/vast` request returns no-fill. Integration test under `server/internal/tracking/integration_test.go` passes against testcontainers-go.
6. **Phase 5**: `k6 run loadtest.js` shows P99 < 50ms @ 1k RPS. `govulncheck` + `npm audit` clean. Tagged release publishes images to GHCR. Fresh VM follows README quickstart and reaches a working dashboard inside 10 minutes.

---

## Out of Scope for v1.0 (noted for future direction, not planned in detail)

- **VPAID / OMID** support (deprecated in favor of VAST 4 SIMID/IMA; revisit when there's user demand).
- **Server-Side Ad Insertion (SSAI)** stitching.
- **Header bidding / OpenRTB** integration.
- **Multi-tenancy** (org/workspace model, per-tenant RBAC).
- **Programmatic ad creation API** (currently the dashboard is the source of truth).
- **A/B testing of creatives within a campaign**.
- **Audit log & soft-delete** across all tables.

These belong in a v2.0 RFC after the v1.0 release ships and real users surface priorities.
