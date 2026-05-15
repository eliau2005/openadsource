# Changelog

All notable changes to OpenAdSource are documented here. The format is
based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and
the project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

(Nothing yet — the slot for the next release.)

---

## [1.0.0] - 2026-05-16

The first public release. The repository ships the full five-phase
build: a working VAST 4.2 adserver, an atomically-capped decision
engine, a reconciler worker, a Next.js dashboard, signed tracking
pixels, per-IP rate limiting, Prometheus `/metrics`, four operator
docs, and a multi-arch release pipeline.

### Added — Phase 0 (Bootstrap)

- Repository bootstrap: monorepo layout (`server/`, `dashboard/`,
  `examples/`, `docs/`), `.editorconfig`, `.gitattributes`, `.gitignore`
- Empty community files (`LICENSE`, `CONTRIBUTING.md`,
  `CODE_OF_CONDUCT.md`, `SECURITY.md`) — file paths exist, canonical
  text pasted by the operator
- Docker Compose stack (Postgres 16, Redis 7, MinIO, adserver, worker,
  dashboard, one-shot `migrate` + `seed`) with a `dev` overlay
- CI workflow (`golangci-lint`, `go test`, `next build`,
  `eslint`, `tsc --noEmit`)
- CodeQL workflow for Go + TypeScript

### Added — Phase 1 (Schema + creatives)

- Postgres migrations for `campaigns`, `ads`, `targeting`, `cap_rules`,
  `daily_stats`, dashboard `users`
- One-shot `seed` binary — uploads a sample MP4 to MinIO and creates a
  demo campaign with one BYO-URL ad and one `internal_s3` ad
- `internal/storage` resolver — public-URL passthrough for
  `external_url` ads, AWS SDK v2 presigning for `internal_s3` ads
- Bundled static test player (`examples/test-player`, video.js +
  videojs-vast-vpaid)

### Added — Phase 2 (Dashboard)

- Next.js 16 dashboard with the App Router, Tailwind v4, Drizzle ORM,
  shadcn/ui-style components
- `/setup` first-user flow, JWT session cookie via `jose`, bcryptjs
  password hashing
- Campaign + ad CRUD with presigned MinIO uploads
- `oas:registry:invalidate` pub/sub publish on every catalog mutation
  so the running adserver reloads its snapshot within milliseconds

### Added — Phase 3 (Decision engine)

- `internal/registry`: immutable `Snapshot` swapped under
  `atomic.Pointer`, reloaded on a 30 s TTL ticker and on the
  invalidation pub/sub channel; hand-rolled `[]uint64` bitsets for
  position / country / device / global filters
- `internal/selection`: bitset-based ad picker — **2264 ns/op, 0
  allocs/op** in `BenchmarkSelect`. `Scratch` buffer pooled via
  `sync.Pool`
- `internal/capping`: Redis Lua scripts (EVALSHA) for atomic INCR-with-
  cap on `freq:<uid>:<ad_id>` and `campaign:<id>:imps_total`
- `internal/targeting`: trusted-proxy-aware IP resolver, MaxMind
  GeoLite2-Country lookup, UA → device classifier
- `delivery.Handler` — composes the snapshot read, targeting filter,
  selection, freq + budget enforcers, storage resolver, and VAST
  builder into `GET /vast`. Always returns syntactically valid VAST
  (inline or empty) — no 5xx leaks to the player

### Added — Phase 4 (Tracking + worker)

- `internal/tracking`: HMAC-SHA256 signer over `(ad_id, imp_id, event,
  exp)`, 24 h `exp` default, idempotency dedupe on `(imp_id, event)`
  via Redis `SETNX`, 7-event vocabulary (impression, click, start,
  firstQuartile, midpoint, thirdQuartile, complete)
- `GET /track` always returns the 43-byte canonical 1×1 transparent
  GIF89a; status is the only side channel (`200` accepted, `204`
  rejected/duplicate)
- `oas_uid` cookie: HttpOnly + SameSite=Lax + Path=/ + scheme-aware
  Secure flag
- `cmd/worker`: tick loop guarded by a Redis distributed lock
  (`SET NX PX 60000` + Lua compare-delete release); drains Redis
  counters with `GETDEL`; upserts into Postgres `daily_stats`;
  reconciles per-campaign budgets and pauses completed campaigns;
  publishes registry-invalidate on pause
- Dashboard `/reports`: funnel + daily-bar visualisation from
  `daily_stats`

### Added — Phase 5 (Hardening)

- Per-IP token-bucket rate limiting on `/vast` (100 r/s, burst 200) and
  `/track` (200 r/s, burst 400) via `golang.org/x/time/rate`;
  background goroutine sweeps idle entries; hard cap on map size
  (default 100 k). Over-limit responses are empty VAST / 1×1 GIF — no
  `429` to the player
- Prometheus `/metrics` on the adserver (`PORT`) and worker
  (`METRICS_PORT`, default 9100). `oas_*` namespace, cardinality
  bounded — `oas_http_requests_total{route,code}`, request duration
  histogram, VAST response composition, budget + freq rejection
  counters, snapshot gauge + load histogram, `/track` event counter,
  worker tick / drain / pause counters
- `chi`-aware HTTP instrumentation middleware using `RoutePattern()`
  so the `route` label stays bounded
- `.github/dependabot.yml` watching `gomod` (server), `npm`
  (dashboard), `docker` (both), and `github-actions`; minor + patch
  grouped weekly
- Cookie audit: confirmed `oas_session` (dashboard) and `oas_uid`
  (adserver) both key `Secure` off `PUBLIC_BASE_URL` scheme
- Docs in `/docs`: `architecture.md`, `self-hosting.md`,
  `vast-integration.md`, `api.md`
- Top-level `README.md` with quickstart, feature checklist, phase
  status, architecture mini-diagram, doc links
- `.goreleaser.yaml` — multi-OS / multi-arch binaries for adserver +
  worker + seed (`linux/{amd64,arm64}` + `darwin/{amd64,arm64}`),
  stripped + trimmed, checksum file
- `.github/workflows/release.yml` — tag-driven (`v*`) GoReleaser run
  plus multi-arch Docker buildx for `openadsource-server` and
  `openadsource-dashboard` images pushed to GHCR

### Carry-overs to v1.1

- Populate `LICENSE` (Apache 2.0), `CONTRIBUTING.md`,
  `CODE_OF_CONDUCT.md` (Contributor Covenant 2.1), `SECURITY.md`
- Ship a Grafana dashboard JSON under `infra/grafana/`
- `testcontainers-go` integration test for the worker drain loop
- Dashboard form for `cap_rules` (operators currently set caps via SQL)
- Multiple cap rules per ad (snapshot exposes only the most-recent)
- Reports page polish (`recharts` / `tremor` upgrade)
- Sample `nginx` config split out under `infra/nginx/`

[Unreleased]: https://github.com/eliau2005/openadsource/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/eliau2005/openadsource/releases/tag/v1.0.0
