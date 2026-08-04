---
name: di-container-leak-rollout
description: Request-scoped DI sub-container leak fix — rollout status per repo and which repos still carry the bug
metadata:
  type: project
---

The `DiContainerMiddleware` request-container leak (middleware created a
`sarulabs/di` sub-container that only a handler `defer ctn.Delete()` released, so
every short-circuited request — 401 / 403 / 429 / `/assets` / favicon / 404 / SPA
render — retained its container for the process's life) is being fixed
platform-wide, one repo at a time.

Rollout as of **2026-08-05**:

- **gardener** — DONE (PR #77, merged). Reference implementation for everyone else:
  `internal/middlewares/middleware.go` `DiContainerMiddleware` +
  `TestDiContainerMiddlewareReleasesRequestContainer`.
- **gobuild `platform-service` template** — DONE on branch
  `fix/di-container-release-in-template` (uncommitted at time of writing), so newly
  scaffolded services are no longer born with it.
- **isme, medioa2** — in progress by others.
- **tomatime** — ⚠️ STILL CARRIES THE BUG. It was scaffolded from the preset before
  the template fix, so regenerating won't help; the fix has to be applied to its
  checked-in code by hand.

**Why:** measured in gardener, live heap climbed monotonically to 58 MB over 60k
unauthenticated 401s versus 3–4 MB with the release in place. The leaking paths are
the cheap unauthenticated ones, so triggering it is free for an attacker.

**How to apply:** when touching any platform-service-shaped repo's middlewares or
handlers, check whether that repo has had the fix; if handlers still carry
`defer ctn.Delete()`, that repo is unfixed. The fix always has the same shape —
middleware `defer`s `request.DeleteWithSubContainers()` (never `Delete()`, which is a
conditional no-op when children exist) and handlers only borrow the container. See
[[gobuild-golden-fixture-workflow]] for the template side.
