# CLAUDE.md

Guidance for Claude Code when working in this repository.

## Overview

`testproj` (module `github.com/vukyn/testproj`) is a clean-architecture Go
service generated from the pet-platform `platform-service` preset. It uses
Fiber v2, Bun ORM over SQLite, `sarulabs/di/v2` for dependency injection, and
the shared `github.com/vukyn/kuery` library (logging, ctx helpers, HTTP
responses, graceful shutdown, panic recovery, Bun hooks, crypto/ULID).

## Commands

```bash
make run                    # go run cmd/main.go
make build                  # build binary to bin/
make migrate-up DB=sqlite   # run db/migrate.go sqlite up
make migrate-down DB=sqlite # rollback last migration
make migrate-reset DB=sqlite# rollback all migrations
make web                    # Vite dev server in ui/
make build-web              # build ui/ -> internal/web/dist (embedded by Go)

go build ./...              # verify
go vet ./...
go test ./...              # web embed has a test; add your own per domain
```

Config is loaded from `.env` at the repo root via godotenv + envconfig.

## Architecture

Clean architecture, domain-driven layout. Entry: `cmd/main.go` ->
`internal/app` (`Init` builds the DI container, initializes the logger, forces
the DB singleton) -> `internal/server` (Fiber app + route registration).

### Layer flow per domain (`internal/domains/<domain>/`)

```
handlers/http  ->  usecase  ->  repository  ->  entity (Bun model / DB)
models/            request + response DTOs with .Validate()
exceptions/        domain error types {Message, Code}
```

Rules (non-negotiable, mirror the platform):

- `entity/` holds Bun ORM models only — no business logic. Audit fields
  (`CreatedAt/By`, `UpdatedAt/By`, `DeletedAt/By` with `soft_delete,nullzero`);
  timestamps set in the `BeforeAppendModel` hook.
- `repository/` exposes an `IRepository` interface in `irepository.go` plus an
  impl over `*bun.DB`. Repos wrap `sql.ErrNoRows` into domain exceptions and
  return errors without logging.
- `usecase/` depends on the repository INTERFACE, never the concrete impl. IDs
  for new rows use `kuery/cryp.ULID()`.
- `handlers/http/` are thin: resolve the request-scoped container with
  `pkgCtx.GetDiContainerRequestFromFiberCtx(c)`, build a `context.Context` with
  `pkgCtx.NewContextFromFiberCtx(c)`, call the usecase, and funnel responses
  through `pkgHttp.OK` / `pkgHttp.Err`.
  ⚠️ **A handler must NOT call `ctn.Delete()`** — it only borrows the container;
  `DiContainerMiddleware` created it and releases it (see Dependency injection).
  This is the reverse of the older platform rule, which had every handler carry a
  `defer ctn.Delete()`. That rule leaked by construction: the middleware is
  global, so a request that never reaches a handler (401, 403, 429, `/assets`,
  `/favicon.svg`, a 404, every SPA render) had nobody to run the defer, and
  sarulabs/di retains such a sub-container for the life of the process. A leftover
  defer is not a visible failure — a second `Delete` returns nil — it just runs
  every registered `Close` twice.
- Only handlers/middleware log.

### Dependency injection (`internal/di/`)

`di.NewBuilder()` registers definitions in dependency order:
`config -> db -> middleware -> repositories -> usecases`. DI names are the
constants in `internal/constants/di.go` (`config`, `db`, `middleware`,
`item.repository`, `item.usecase`). Singletons are `di.App`-scoped; repos and
usecases are `di.Request`-scoped.

⚠️ **`DiContainerMiddleware` owns the request container's whole lifetime.** It
creates the `di.Request` sub-container, stores it in Fiber locals, and releases it
with its own `defer` — so the release runs on every path: a handled 200, a 401 from
an auth middleware, a 403 from a role check, a 429 from a rate limiter, `/assets`,
`/favicon.svg`, a 404, every SPA catch-all render, a handler that returns an error,
and a panic recovered by `pkgRecover.NewFiberRecover()` (the recover middleware is
mounted INSIDE this one, and a `defer` runs during unwinding anyway).

Whoever creates a sub-container releases it. Handlers only *borrow* one, so they must
not call `Delete`; the only legitimate `Delete` sites are the ones that opened their
own sub-container with no request around it (e.g. a startup bootstrap task) plus the
app container's own teardown in `cmd/main.go`. The rule used to be the opposite (a
`defer ctn.Delete()` in every handler) and it leaked in production: this middleware
is mounted globally, ahead of the routes, so it had already built a sub-container by
the time anything decided the request would not reach a handler — and `sarulabs/di`
keeps every sub-container in its parent's `children` map until deleted, so each of
those was retained for the life of the **process**. A handler-owned lifetime cannot
cover a request that never reaches a handler. The leaking paths were also the
cheapest, unauthenticated ones, so the leak was free to trigger.

It calls **`DeleteWithSubContainers`**, not `Delete`: `Delete` is conditional — with
any child present it merely sets `deleteIfNoChild` and returns nil, leaving the
container in the parent's `children` map, i.e. the leak. A single owner needs an
unconditional release. Its documented hazard (tearing down a sub-container another
goroutine still uses) does not apply while nothing touches the container after its
handler returns — no `SendStream`/`SetBodyStreamWriter`, and no goroutine that
resolves from it. That is a **precondition of this design**: if you ever hand a
request-scoped dependency to a goroutine that outlives the request, this release is a
use-after-free and the ownership has to be rethought, not worked around.

When you add tests, pin this: assert the app container retains **zero** children
after a request on each short-circuit path, and that a registered `Close` runs
**exactly once** per request (so a re-added handler defer fails the suite instead of
silently double-closing).

### Database

SQLite at `db/app.db` (Bun `sqlitedialect` + `sqliteshim` driver, no CGO).
Migrations are plain Go funcs in `db/history/sqlite/sqlite.go`, run by
`db/migrate.go`. Soft delete via `deleted_at`.

### Embedded UI (`internal/web/`)

The React UI in `ui/` is compiled by `make build-web` into `internal/web/dist`
and embedded into the binary at compile time via `//go:embed all:dist`
(`internal/web/web.go`), so the server is a single self-contained artifact.
`internal/server/server.go` serves it: the Fiber `html` engine renders
`index.html` (with `VITE_API_BASE_URL` injected), `/assets` is served from the
embedded `assets/` subtree, root files like `favicon.svg` get an explicit route
before the SPA catch-all, and `/*` renders the SPA. A committed
`internal/web/dist/.gitkeep` keeps the embed pattern valid before any build —
`go build` works on a fresh checkout, the real bundle fills in after
`make build-web`. The UI uses route-level code splitting (`React.lazy` +
`Suspense` in `ui/src/App.tsx`) with vendor `manualChunks` in
`ui/vite.config.ts`.

## Conventions

- Interfaces prefixed `I` (`IRepository`, `IUseCase`); files `snake_case.go`.
- `any`, not `interface{}`. No abbreviated variable names.
- `ctx context.Context` is the first parameter of repository/usecase methods.
- Import groups: stdlib | third-party | internal, with domain-prefixed aliases
  (`itemEntity`, `pkgCtx`, `pkgHttp`).
- `pkg/`-style reusable code belongs in `github.com/vukyn/kuery`, not a local
  package.

## Extension points (out of scope for the generated skeleton)

- **Tests** — the skeleton ships no `_test.go`. Add table-driven tests per
  domain (usecase against repository fakes; handlers via Fiber `app.Test`).
- **UI** — a complete, minimal Vite + React 19 + TypeScript + Chakra UI v3
  project ships under `ui/` (`package.json`, `index.html`, `tsconfig*`,
  `vite.config.ts`, `eslint.config.js`, `src/main.tsx` with `ChakraProvider`,
  a single `Home` route via `React.lazy` + `Suspense`, vendor `manualChunks`,
  and `public/favicon.svg`). It is buildable as-is: `make build-web` runs
  `npm install && npm run build` and stages `ui/dist` into `internal/web/dist`,
  which is embedded into the binary (see Architecture). Build the UI before
  `go build` to ship a real bundle. Grow it by adding pages under
  `ui/src/pages/` and registering lazy routes in `ui/src/App.tsx`.
- **MongoDB** — this preset is SQLite-only. A Mongo-backed variant would swap
  `internal/di/di_db.go` and the repository impls for the Mongo driver and drop
  the `db/` migration runner.
- **Authentication** — no auth middleware is wired (the example routes are
  open). To protect routes, add the `kuery/auth` middleware in
  `internal/middlewares` and apply it in `internal/server` route registration,
  as the platform's downstream services do.
