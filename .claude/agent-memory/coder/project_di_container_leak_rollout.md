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
- **isme** — DONE on branch `fix/di-container-release-in-middleware` (uncommitted at
  time of writing): 55 handler defers removed, middleware extracted out of
  `internal/server/server.go` into `internal/middlewares/di_container_middleware.go`
  so it takes the app container as a parameter and is testable. Note the trap:
  `internal/di/di_middleware.go` builds an App-scoped sub-container that is
  **deliberately never deleted** (it holds the process-lifetime auth usecase) —
  an audit flags it, but deleting it would Close an object the middleware keeps using.
- **medioa2** — DONE on branch `fix/di-container-release-in-middleware` (uncommitted at
  time of writing): 86 handler defers removed across 17 files, middleware moved to
  `internal/middlewares/di_container_middleware.go` taking the app container as a
  parameter (same shape as isme). Two additions worth reusing elsewhere: a **static
  ownership guard test** (`TestNoHandlerReleasesRequestContainer` scans `internal/` for
  `ctn.Delete()`) because the runtime Close counter mounts stub routes and therefore can
  never see a defer left behind in a REAL handler; and the note that `Delete()` vs
  `DeleteWithSubContainers()` makes **no observable difference for the request container
  today** in these services — a request container has no children (only the app
  container is ever `SubContainer()`ed), so the conditional-`Delete` trap is latent, not
  active. `DeleteWithSubContainers` is still right: an owner's release must not depend on
  whether someone later nests a sub-container.
- **rainy, memz** — not done yet (handler defer counts at the time of the isme fix:
  rainy 101, memz 5).
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
