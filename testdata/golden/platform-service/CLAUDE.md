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
  `pkgCtx.GetDiContainerRequestFromFiberCtx(c)` then `defer ctn.Delete()`, build
  a `context.Context` with `pkgCtx.NewContextFromFiberCtx(c)`, call the usecase,
  and funnel responses through `pkgHttp.OK` / `pkgHttp.Err`.
- Only handlers/middleware log.

### Dependency injection (`internal/di/`)

`di.NewBuilder()` registers definitions in dependency order:
`config -> db -> middleware -> repositories -> usecases`. DI names are the
constants in `internal/constants/di.go` (`config`, `db`, `middleware`,
`item.repository`, `item.usecase`). Singletons are `di.App`-scoped; repos and
usecases are `di.Request`-scoped. `DiContainerMiddleware` creates a
request-scoped sub-container per request and stores it in Fiber locals.

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
