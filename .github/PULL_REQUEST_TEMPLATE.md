<!--
  Keep PRs focused and atomic. Prefer multiple small PRs to one large one.
-->

### Summary

<!-- One or two sentences describing what this PR does and why. -->

### Roadmap phase

<!-- Reference the relevant ROADMAP.md phase, e.g. "Phase 1 - VAST Delivery Core". -->

### Changes

-
-
-

### Test plan

- [ ]
- [ ]
- [ ]

### Checklist

- [ ] `go build ./... && go vet ./... && go test ./...` passes in `server/`
- [ ] `npm run lint && npm run typecheck && npm run build` passes in `dashboard/`
- [ ] `docker compose up --build -d` brings the stack up healthy (if compose / Dockerfile / migrations changed)
- [ ] Updated docs / ROADMAP.md if user-facing behavior or contracts changed
- [ ] No secrets, credentials, or real email addresses committed
