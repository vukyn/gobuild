# CLAUDE.md

Guidance for Claude Code when working in this repository.

## Overview

`testproj` (module `github.com/vukyn/testproj`) is a clean-architecture Go
service generated from the pet-platform `platform-service-v3` preset. It uses
Fiber **v3**, Bun ORM over **Postgres**, `sarulabs/di/v2` for dependency
injection, and the shared `github.com/vukyn/kuery` library (logging, ctx
helpers, HTTP responses, graceful shutdown, panic recovery, Bun hooks,
crypto/ULID, the DB factory and the migration runner).

## Commands

```bash
make run                      # go run cmd/main.go
make build                    # build binary to bin/
make db-up                    # start the local Postgres (docker-compose.yml)
make db-down                  # stop it
make migrate-up DB=postgres   # run db/migrate.go postgres up
make migrate-down DB=postgres # rollback last migration
make migrate-reset DB=postgres# rollback all migrations
make web                      # Vite dev server in ui/
make build-web                # build ui/ -> internal/web/dist (embedded by Go)

go build ./...              # verify
go vet ./...
go test ./...               # ships guards for known traps; add your own per domain
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

`internal/middlewares/container_ownership_test.go` pins this statically. When you add
runtime tests, pin it there too: assert the app container retains **zero** children
after a request on each short-circuit path, and that a registered `Close` runs
**exactly once** per request (so a re-added handler defer fails the suite instead of
silently double-closing).

### Database

Postgres via Bun (`pgdialect` + `pgdriver`, reached through `kuery/bun/db.Open` —
both are indirect dependencies, only `uptrace/bun` is a direct require).
`internal/di/di_db.go` builds the connection from `cfg.DB`; when `DB_DSN` is empty
the DSN is assembled from the discrete `DB_HOST`/`DB_PORT`/… fields.

⚠️ **`DB_SSLMODE` has no struct-tag default on purpose.** An empty value reaches
kuery's DSN builder, which fails closed to `require` — so a forgotten variable
yields TLS, not cleartext. A local docker-compose Postgres has no TLS, so `.env`
sets `DB_SSLMODE=disable` **explicitly**. Do not "tidy" that into a
`default:"disable"` struct tag: that re-arms cleartext behind kuery's back.
`DB_DSN`, when set, carries its own sslmode and wins outright.

No pool overrides are set: kuery's `applyPool` already supplies the Postgres
idle/lifetime defaults that stop a pooled connection being handed out after the
server closed it (the bare `EOF` failure).

Migrations live in `db/history/`: **one migration per file** (`NNN_<name>.go`,
declaring a `pkgMigrate.Migration` with `Up`/`Down func(bun.IDB) error`), all listed
in execution order in `db/history/migrations.go`, executed by `kuery/bun/migrate`
through `db/migrate.go`. ⚠️ A migration file that compiles but is missing from that
slice does nothing at all. Soft delete via `deleted_at`.

### Embedded UI (`internal/web/`)

The React UI in `ui/` is compiled by `make build-web` into `internal/web/dist`
and embedded into the binary at compile time via `//go:embed all:dist`
(`internal/web/web.go`), so the server is a single self-contained artifact.
`internal/server/server.go` serves it: the Fiber `html` engine renders
`index.html` (with `VITE_API_BASE_URL` injected), `/assets` is served from the
embedded `assets/` subtree via `fiber/v3/middleware/static`, root files like
`favicon.svg` get an explicit route before the SPA catch-all, and `/*` renders
the SPA. A committed `internal/web/dist/.gitkeep` keeps the embed pattern valid
before any build — `go build` works on a fresh checkout, the real bundle fills in
after `make build-web`. The UI uses route-level code splitting (`React.lazy` +
`Suspense` in `ui/src/App.tsx`) with vendor `manualChunks` in
`ui/vite.config.ts`.

⚠️ **Root-level static files vs the SPA catch-all.** `webRoutes` routes exactly
one root file (`/favicon.svg`) and then `app.Get("/*", renderHomePage)`. As
generated this is correct, but the shape breaks the moment a second root-level
asset is added (a `.webmanifest`, PWA icons, `sw.js`): the catch-all answers it
with `index.html` at **status 200**, so nothing 404s and the logs look healthy.
That is how a downstream platform service shipped a dead service worker and an
Add-to-Home-Screen with no name or icon. Give every new root-level file its own
route ahead of the catch-all, and check the `Content-Type`.

## Fiber v3 notes

This preset is Fiber **v3**, not the v2 the older `platform-service` preset
generates. The differences that matter, and why:

- **`fiber.Ctx` is an interface** — handlers are `func(c fiber.Ctx) error`, not
  `func(c *fiber.Ctx) error`.
- **Body/query decoding moved to `Bind()`** — `c.Bind().Body(&req)` and
  `c.Bind().Query(&req)` replace `BodyParser`/`QueryParser`. `Bind` defaults to
  *WithoutAutoHandling*, so it returns the raw parse error just as v2 did and
  `pkgHttp.Err` still owns the response envelope. ⚠️ Do not call
  `.WithAutoHandling()` — it sets a status behind the handler's back and breaks
  that contract.
- **kuery's v3 twins** — use `kuery/ctxv3`, `kuery/http/fiberv3` and
  `kuery/recoverv3`, never their v2 originals. ⚠️ `kuery/ctx` and `kuery/ctxv3`
  each declare their **own** `type ContextKey string`, so a value stored under one
  is invisible to the other even though the key strings are identical. A file that
  only calls `pkgCtx.GetUserID(ctx context.Context)` — `repository.go` is exactly
  that — compiles against either package and, on the wrong one, writes an empty
  `created_by`/`updated_by` on every row forever with no error. Only looking at the
  column after a real write catches it.
- **`fiberzerolog` does not exist for v3** (gofiber/contrib never published a v2
  module of it; its API is `*fiber.Ctx` throughout). Request logging goes through
  `fiber/v3/middleware/logger` with a `LoggerFunc` writing to kuery's zerolog. Note
  `logger.Data` has no status field — read it off `c.Response().StatusCode()`.
- **`middleware/filesystem` is gone** — `fiber/v3/middleware/static` replaces it
  (`static.Config.FS` takes an `fs.FS`, not an `http.FileSystem`), and single-file
  sends are `c.SendFile("name", fiber.SendFile{FS: uiFS})`.
- **CORS `AllowOrigins` is a `[]string`**, and v3 computes
  `allowAllOrigins := len(AllowOrigins) == 0 && AllowOriginsFunc == nil`. An
  **empty slice is allow-all** without ever containing `"*"` — see the CORS section
  below. A malformed origin makes v3 **panic** at boot instead of silently not
  matching.
- **`ProxyHeader` alone is a no-op** — see the proxy section below.
- **`go.mod` carries no `fiber/v2` at all**, even though kuery is one module
  holding both helper lines: this service imports only the v3 twins, and
  module-graph pruning means `go mod tidy` records just what the imported packages
  need. ⚠️ So a `github.com/gofiber/fiber/v2 // indirect` line turning up in
  `go.mod` is a **tripwire**: something started importing a kuery *v2* package —
  most likely a file that slipped from `ctxv3` back to `ctx`. Find the import and
  change it back; do not "fix" the `go.mod` line.

## Conventions

- Interfaces prefixed `I` (`IRepository`, `IUseCase`); files `snake_case.go`.
- `any`, not `interface{}`. No abbreviated variable names.
- `ctx context.Context` is the first parameter of repository/usecase methods.
- Import groups: stdlib | third-party | internal, with domain-prefixed aliases
  (`itemEntity`, `pkgCtx`, `pkgHttp`).
- `pkg/`-style reusable code belongs in `github.com/vukyn/kuery`, not a local
  package.
- Repository queries stay portable: `LOWER(...) LIKE ...`, not Postgres-only
  `ILIKE`. If you add an upsert, Postgres needs the `ON CONFLICT` target
  alias-qualified when the model carries a bun alias.

## Extension points (out of scope for the generated skeleton)

- **Tests** — the skeleton ships guards for known traps only
  (`internal/middlewares/container_ownership_test.go`, `internal/web/web_test.go`,
  `internal/server/cors_test.go`, `internal/server/proxy_test.go`), not domain
  tests. Add table-driven tests per domain (usecase against repository fakes;
  handlers via Fiber `app.Test`).
- **UI** — a complete, minimal Vite + React 19 + TypeScript + Chakra UI v3
  project ships under `ui/`. It is buildable as-is: `make build-web` runs
  `npm install && npm run build` and stages `ui/dist` into `internal/web/dist`,
  which is embedded into the binary (see Architecture). Build the UI before
  `go build` to ship a real bundle. Grow it by adding pages under
  `ui/src/pages/` and registering lazy routes in `ui/src/App.tsx`.
- **MongoDB** — this preset is Postgres-only. A Mongo-backed variant would swap
  `internal/di/di_db.go` and the repository impls for the Mongo driver and drop
  the `db/` migration runner.
- **Authentication** — no auth middleware is wired (the example routes are
  open). ⚠️ On this preset that is not a one-line swap of the v2 recipe:
  `kuery/ctxv3` has **drifted** from `kuery/ctx` — it is missing `SessionIDKey`,
  `SetSessionIDToFiberCtx`, `GetSessionIDFromFiberCtx` and `GetSessionID`, and it
  ships **no test files at all**. `kuery/authv3` and `kuery/rbacv3` exist but have
  likewise drifted from their v2 twins (no `RevocationChecker`). Diff the v3
  package against its v2 original before relying on it, and expect to contribute
  the missing helpers back to kuery rather than duplicating them here.
- **Rate limiting** — `fiber/v3/middleware/limiter`. It buckets by `c.IP()`, so
  the proxy wiring below is a prerequisite, not an optimisation: without it every
  caller shares one bucket.

## ⚠️ TODO before deploying: CORS_ALLOW_ORIGINS

`internal/server/server.go` configures `cors.New` from `cfg.CORS.AllowOrigins`
(`CORS_ALLOW_ORIGINS` in `.env`), a comma-separated allow-list that ships
pointing at the local development origins.

Set it to this service's real browser origin(s). Three things make this easy to
get wrong:

- **Blank does not mean "off".** Fiber v3 treats an **empty `AllowOrigins`
  slice** as allow-all (`len(cfg.AllowOrigins) == 0 && cfg.AllowOriginsFunc ==
  nil`) — and the slice never contains `"*"` while doing so, so a check for the
  wildcard string would not notice. `corsAllowOrigins` in
  `internal/server/server.go` exists precisely to intercept that and fall back to
  `defaultCORSAllowOrigins` instead — do not "simplify" it back to a bare
  `cors.New()`, and keep its test asserting the **length**.
- **A malformed origin panics at boot.** v3's `configDefault` panics with
  `[CORS] Invalid origin format in configuration` on a typo'd value rather than
  silently not matching. Deliberate and loud — fix the value.
- **The example routes are unauthenticated.** `/api/v1/items` ships as open
  CRUD, so a wildcard origin is a wildcard on write endpoints, not just reads.

## ⚠️ Behind a proxy: APP_PROXY_HEADER needs TrustProxy

On Fiber v3, `Config.ProxyHeader` **alone does nothing**. `c.IP()` reads the
header only when `IsProxyTrusted()` is true, which needs `Config.TrustProxy`
**plus** a `TrustProxyConfig` rule (`Loopback`, `Private`, `LinkLocal`,
`Proxies`, `UnixSocket` — all default `false`) matching the remote address.

`proxyTrust` in `internal/server/server.go` derives both from
`APP_PROXY_HEADER`: empty means "no proxy, trust the socket's remote address";
set means `TrustProxy: true` with `Loopback` + `Private` (the usual fly.io-style
shape). `internal/server/proxy_test.go` pins it.

This is the **one deliberate divergence** from the older `platform-service`
preset, which sets no `ProxyHeader` at all. Copying a Fiber **v2** service's
config across — where `ProxyHeader` alone was sufficient and `EnableIPValidation`
was the guard — is wrong here and fails silently: every caller collapses into the
proxy's own address, so every rate limiter sees one bucket and every audit row
records one value, with no error logged anywhere.
