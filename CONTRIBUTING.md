# Contributing to OpenAdSource

Thanks for considering a contribution. OpenAdSource is a self-hosted
video ad server (VAST 4.2) with a small, opinionated codebase — bug
reports, pull requests, documentation fixes, and new player-integration
recipes are all welcome.

This guide is the short version. The deeper background lives in
[`docs/architecture.md`](./docs/architecture.md) and
[`docs/self-hosting.md`](./docs/self-hosting.md).

## Repository layout

The repo is a monorepo with four top-level subtrees:

- **`server/`** — Go 1.25. Two long-running binaries
  (`cmd/adserver`, `cmd/worker`) plus the one-shot `cmd/seed`. Library
  code lives under `internal/`: `delivery` (the `/vast` handler),
  `registry` (atomic snapshot + Redis-driven invalidation),
  `selection` (bitset-based picker), `capping` (Redis Lua for
  freq + budget), `tracking` (HMAC signer + `/track` handler),
  `storage` (S3 presigner), `targeting` (IP + GeoIP + UA),
  `vast` (XML builder), `db`, `config`, `httpmw`, `metrics`.
- **`dashboard/`** — Next.js 16 (App Router) + React 19 + Tailwind v4
  + Drizzle ORM. Owns auth (`jose` JWT + bcrypt), campaign / ad
  CRUD, MinIO upload presigning, and the `/reports` funnel page.
- **`examples/test-player/`** — Static HTML test rig built on
  video.js + videojs-vast-vpaid. The canonical end-to-end check for
  the `/vast` → `/track` round-trip.
- **`docs/`** — Operator-facing docs: `architecture.md`,
  `self-hosting.md`, `vast-integration.md`, `api.md`.

Migrations live in `server/migrations/` (Go side, applied by the
one-shot `migrate` compose service) and `dashboard/drizzle/` (kept in
sync — the dashboard reads the same schema).

## Local development

### Prerequisites

- Docker Engine + Compose v2
- Go 1.25 (only if iterating on the Go side outside the containers)
- Node 22 + pnpm (only if iterating on the dashboard outside the
  containers)

### First-time setup

```bash
git clone https://github.com/eliau2005/openadsource
cd openadsource
cp .env.example .env
# Edit .env — at minimum, set JWT_SECRET and TRACKING_SECRET to
# high-entropy values. The placeholders work for local dev but the
# dashboard logs a warning until they're replaced.
```

### Running the full stack

```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build
```

The dev overlay publishes the otherwise-internal ports so local tools
can reach them: Postgres on `5432`, Redis on `6379`, the MinIO console
on `9001`, and the worker's `/metrics` endpoint on `9100`. The adserver
serves on `8088`, the dashboard on `3000`, and the bundled test player
on `8090`.

Once the `migrate` one-shot exits, seed a demo campaign:

```bash
docker compose --profile seed up seed
```

Then open <http://localhost:8090> — the test player should render the
demo ad followed by the fallback content. Create the first dashboard
user at <http://localhost:3000/setup>.

### Iteration loop

- **Go changes** — `docker compose up --build -d adserver worker`
  rebuilds and recreates just those two services. Tests run faster
  on the host: `cd server && go test ./...`.
- **Dashboard changes with hot-reload** — Compose doesn't bundle a
  `next dev` overlay; for fast iteration, run the dashboard locally
  against the compose Postgres:

  ```bash
  cd dashboard && pnpm install
  # Point DATABASE_URL at localhost:5432 (the dev overlay publishes it).
  pnpm dev
  ```

  Visit <http://localhost:3000> while the compose stack continues to
  serve the adserver, MinIO, Redis, and Postgres in the background.
- **Migrations** — add SQL files under `server/migrations/` and mirror
  the schema change in `dashboard/drizzle/`. `docker compose up
  migrate` reapplies idempotently.

## Conventional Commits

Commit messages must follow the
[Conventional Commits](https://www.conventionalcommits.org/) prefix
set. The repo's existing history is the reference:

| Prefix      | Use for                                                |
|-------------|--------------------------------------------------------|
| `feat:`     | A new user-visible feature                             |
| `fix:`      | A bug fix                                              |
| `perf:`     | A performance improvement with no behavior change      |
| `refactor:` | A code change that neither fixes a bug nor adds a feature |
| `docs:`     | Documentation-only changes (README, `docs/`, comments) |
| `test:`     | Adding or fixing tests                                 |
| `chore:`    | Tooling, dependency bumps, release plumbing            |
| `ci:`       | Changes to `.github/workflows/`                        |

Scopes are encouraged when the change is local to one subtree, e.g.
`feat(server):`, `feat(dashboard):`, `chore(release):`,
`docs(api):`.

The subject is the *why* (one line, imperative mood). Use the body for
context, references to incidents, or links to the issue the change
closes.

## Pull request checklist

Copy this into the PR description and tick each box before requesting
review:

- [ ] `cd server && go build ./... && go vet ./... && go test ./...` is clean
- [ ] `cd server && golangci-lint run` is clean (matches CI's
      `golangci/golangci-lint-action@v6`)
- [ ] `cd dashboard && pnpm lint && pnpm typecheck && pnpm build` is
      clean (if the dashboard was touched)
- [ ] Smoke-tested at <http://localhost:8090> against the dev overlay
- [ ] Commits follow the Conventional Commits prefix set above
- [ ] `CHANGELOG.md` updated under `[Unreleased]` for any user-visible
      change (skip for pure refactors / test-only / CI-only changes)
- [ ] PR links to the issue it closes, if applicable

CI runs every gate listed above on every push. Running them locally
before pushing avoids a round-trip.

Security-sensitive fixes should follow the disclosure path in
[`SECURITY.md`](./SECURITY.md) instead of an open PR.
