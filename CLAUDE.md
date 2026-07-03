# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Purpose

`gobuild` is a small CLI (module `github.com/vukyn/gobuild`) that scaffolds new Go projects from templates. Given a project name (positional arg or `--name`/`-n`), an optional `--go` version, an optional `--module`/`-m` Go module path, and a `--http-template`/`--preset` (default `base`), it creates a directory from the selected preset's template tree, then runs `go mod tidy` and `git init` inside it.

Presets:

- **base** (default) — plain "hello world" Go project: `main.go`, `go.mod`, `.env`, `Makefile`, `README.md`, `.gitignore`, `todo`.
- **fiber** — Fiber v3 HTTP server: adds `internal/handler/health.go` (`/health` → `{"status":"ok"}`), `main.go` with graceful shutdown (signal.NotifyContext SIGINT/SIGTERM + `app.Shutdown`), `APP_PORT` in `.env`, and a fiber dependency in `go.mod`.
- **platform-service** — full clean-architecture service mirroring the pet-platform template (isme/rainy shape): Fiber **v2**, Bun ORM over **SQLite** (`sqlitedialect` + `sqliteshim`, no CGO), `sarulabs/di/v2` DI container, and the shared `github.com/vukyn/kuery` helpers. Ships an example `item` domain (`/api/v1/items` CRUD: POST/GET-list/GET/PATCH/DELETE-soft-delete, no auth) across the standard layers (`entity`/`models`/`repository`/`usecase`/`handlers/http`/`exceptions`), a migration runner (`db/migrate.go` + `db/history/sqlite`), DI wiring (`config → db → middleware → repos → usecases`), and a `CLAUDE.md` for onboarder platform-fit. IDs use `kuery/cryp.ULID`. The kuery version pinned in the generated `go.mod` tracks what isme currently requires.

  Extension points (intentionally out of scope for the generated skeleton, documented in its `CLAUDE.md`): no `_test.go` files, no UI (`--ui` is a future enhancement — add a Vite/React `ui/` embedded into the Go binary), **SQLite-only** (a MongoDB variant would swap `di_db.go` + repo impls and drop the migration runner), and **no auth** (wire `kuery/auth` middleware in `internal/middlewares` + `internal/server` to protect routes).

- **iot** — minimal ESP32-S3 firmware skeleton (C++/PlatformIO, **non-Go**). Renders `platformio.ini` (single esp32-s3 env pinning the shared `kuino` lib), a thin `src/main.cpp` wired to `kuino::wifi`, `include/config.h.example`, `.gitignore`, `README.md`, `CLAUDE.md`. `go mod tidy` is skipped (no `go.mod`).

gobuild now emits **non-Go** presets too; `go mod tidy` runs only when a `go.mod` was rendered (`hasGoMod`), while `git init` always runs.

### Flags

- `--name`/`-n` — project name (or positional arg).
- `--go` — Go version (defaults to the local toolchain).
- `--http-template`/`--preset` — preset (`base|fiber|platform-service|iot`, default `base`).
- `--module`/`-m` — Go module path (defaults to `github.com/vukyn/<name>`). Threaded into `templateData.ModulePath`; the `platform-service` preset uses it for the `go.mod` module line and all internal imports. `base`/`fiber` ignore it.

Flags can appear before or after the positional name (`reorderArgs` in `main.go` normalizes ordering, since urfave/cli v2 otherwise stops flag parsing at the first positional arg). `valueFlags` lists every flag that consumes the following token so reordering skips flag values.

`base` and `fiber` are standalone "hello world" presets; **platform-service** is the one that follows the full platform service template (domains, DI, clean-architecture layers).

## Structure

```
main.go            # urfave/cli/v2 entrypoint, flag reordering, generateProject(): mkdir, render, tidy + git init
embed.go           # //go:embed all:templates → templatesFS
render.go          # templateData struct, dotfileNames map, outputName(), renderPreset()
templates/         # embedded template tree — one folder per preset (self-contained)
  README.md        # template conventions (.tmpl suffix, dotfile mapping, .raw, adding files/presets)
  base/            # default preset templates (*.tmpl)
  fiber/           # fiber preset templates, incl. internal/handler/health.go.tmpl
version/           # version.Current — the CLI's own version string
testdata/golden/   # golden-file snapshots per preset (gobuild_test.go)
```

Templates are real `text/template` files embedded via `//go:embed all:templates`. `renderPreset` walks `templates/<preset>`, strips the `.tmpl` suffix, applies dotfile mapping (`env`→`.env`, `gitignore`→`.gitignore`), and renders with `Option("missingkey=error")`. See `templates/README.md` for the full convention.

## Commands

```bash
make build      # go build -o bin/ ./$(PRJ)
make install    # go install ./$(PRJ)
make tag        # git tag -a v$(VERSION) + push (VERSION comes from .env)

go build ./...        # verify
go vet ./...
go test ./...         # golden-snapshot + unit tests (offline, hermetic)
go test . -update     # regenerate testdata/golden/<preset>/ after intentional template changes
```

The Makefile sources `.env` (currently empty placeholders for `PRJ`/`VERSION`).

## Conventions

- **Adding a generated file**: drop a new `<name>.tmpl` into the preset folder under `templates/<preset>/`. The walker picks it up automatically — no code changes. Dotfiles are stored without a leading dot and mapped via `dotfileNames` in `render.go`.
- **Adding a preset**: create `templates/<preset>/` with the full file set (presets are self-contained — no layering/overlay; minor static-file duplication is accepted), update the `--http-template` usage string in `main.go`, and add it to `presets` in `gobuild_test.go` then run `go test . -update`.
- New template fields go in `templateData` (`render.go`); reference them as `{{.Field}}`. Rendering uses `missingkey=error`, so a typo'd placeholder fails the build loudly.
- File permissions are deliberate for scaffolder output: `0755` dirs / `0644` files, annotated with `// #nosec` (G301/G306) — generated projects must be user-readable and contain no secrets. Keep those annotations when touching the write paths.
- **Golden fixtures + .gitignore**: `testdata/golden/<preset>/` includes fixtures named `.env`/`.gitignore`/`todo`. The per-preset golden `.gitignore` self-ignores its sibling `.env`/`todo`, so a plain `git add` skips four files; the first commit of new/changed goldens needs `git add -f testdata/golden`.
- Bump `version/version.go` when cutting a release; tag via `make tag`.
