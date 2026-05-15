# Security Policy

OpenAdSource ships as a self-hosted product. Vulnerabilities reported
here cover the source tree in this repository — operator-side
deployment configuration (TLS certificates, network ACLs, secret
rotation) is the responsibility of the entity running the stack and
is documented in [`docs/self-hosting.md`](./docs/self-hosting.md).

## Reporting a vulnerability

**Do not open a public GitHub issue, discussion thread, or pull
request for a suspected vulnerability.** Public disclosure before a
fix ships can expose every operator running the stack.

Email **eliau.elkouby@gmail.com** with:

- The affected version — a release tag (e.g. `v1.0.0`) or a commit
  SHA from the `main` branch.
- Reproduction steps — clear enough that someone running a fresh
  `docker compose up` can reproduce the issue.
- An impact assessment — what an attacker can do, and what level of
  access they need to do it.
- Any proof-of-concept code or captured network traffic, if you have
  them.

You will receive an acknowledgement within **5 business days**. If
you don't, please follow up — the address may have changed.

## Coordinated disclosure

OpenAdSource follows a standard **90-day coordinated disclosure**
window from the date of the initial report:

- The maintainers triage, develop a fix, and prepare a release.
- The reporter is kept in the loop through fix development and is
  credited in the release notes (unless they request anonymity).
- A patched release ships, advisories are filed (GitHub Security
  Advisory + CVE where applicable), and the issue becomes public.

The window can shorten by mutual agreement if a fix lands sooner, or
extend if the reporter agrees, the fix is structurally complex, or
third-party coordination (e.g. an upstream dependency) is required.

## Supported versions

| Version | Supported                                       |
|---------|-------------------------------------------------|
| 1.0.x   | Yes — current stable; all security fixes ship here |
| < 1.0   | No — pre-1.0 builds are not maintained; please upgrade |

Once 1.1 ships, 1.0.x will continue to receive security fixes for at
least 6 months. The supported-versions table is updated on every
minor release.

## Scope

The following classes of issue are considered security vulnerabilities
in this project:

- Authentication or authorization bypass in the dashboard — `jose`
  JWT verification, bcrypt password handling, or the `/setup`
  first-user flow.
- Tracking-pixel signature bypass — forging a valid `sig` on a
  `/track` URL, replaying a captured URL for a different event,
  inflating counters past the per-`(imp_id, event)` dedupe window.
- SQL injection on any Postgres path — the Go server's pgx queries or
  the dashboard's Drizzle queries.
- SSRF or path traversal in `internal/storage` (the S3 presigner and
  the BYO URL resolver) or in the dashboard's upload presigning.
- Unauthenticated disclosure of campaign, ad, or `daily_stats` data.
- Denial-of-service vectors that bypass the per-IP rate limiter on
  `/vast` or `/track` (e.g. amplifying behind a trusted proxy, or
  causing the limiter's visitor map to grow unbounded).
- Credential or secret leakage in logs, error payloads, or the
  Prometheus `/metrics` exposition.

The following are explicitly **out of scope** — please don't report
them as security issues:

- Default credentials in `.env.example` (MinIO root user, Postgres
  password, `JWT_SECRET=placeholder_value`, etc.). These are
  placeholders by design; the README and `docs/self-hosting.md` both
  call out that operators must replace them before any non-local
  deployment.
- Missing `Content-Security-Policy` or other defense-in-depth headers
  on a fresh local stack. Operators add these at the reverse proxy
  layer (see `docs/self-hosting.md`).
- Rate-limiter false positives, log volume, or other operational
  tunables — open a regular issue or PR instead.
- Issues that require physical access to the server, root on the
  host, or compromise of the operator's CI / GHCR tokens.
