# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## The memory layer

@MEMORY.md

⚠️ **That import is the point of the file, not decoration.** `MEMORY.md` and
`memory/` are the distilled layer — one hard-won fact per file, with why it
matters — and they live **in the repository** because a machine's own Claude
memory directory is workspace-scoped and machine-local: this repo opened on
another machine, or outside the workspace the notes were written in, arrived with
none of them.

It is a **distillation, not the record.** This file and the repository's other
documents stay the authority; where a note disagrees with the file that owns the
subject, the repository wins and the note is what to fix. `MEMORY.md` carries the
rules the notes are written under — one line per note in the index, one fact per
file, say why rather than only what, and delete a wrong note rather than adding a
second one beside it.

## Purpose

`gobuild` is a small CLI (module `github.com/vukyn/gobuild`) that scaffolds new Go projects from templates. Given a project name (positional arg or `--name`/`-n`), an optional `--go` version, an optional `--module`/`-m` Go module path, and a `--http-template`/`--preset` (default `base`), it creates a directory from the selected preset's template tree, then runs `go mod tidy` and `git init` inside it.

Presets:

- **base** (default) — plain "hello world" Go project: `main.go`, `go.mod`, `.env`, `Makefile`, `README.md`, `.gitignore`, `todo`.
- **fiber** — Fiber v3 HTTP server: adds `internal/handler/health.go` (`/health` → `{"status":"ok"}`), `main.go` with graceful shutdown (signal.NotifyContext SIGINT/SIGTERM + `app.Shutdown`), `APP_PORT` in `.env`, and a fiber dependency in `go.mod`.
- **platform-service** — full clean-architecture service mirroring the pet-platform template (isme/rainy shape): Fiber **v2**, Bun ORM over **SQLite** (`sqlitedialect` + `sqliteshim`, no CGO), `sarulabs/di/v2` DI container, and the shared `github.com/vukyn/kuery` helpers. Ships an example `item` domain (`/api/v1/items` CRUD: POST/GET-list/GET/PATCH/DELETE-soft-delete, no auth) across the standard layers (`entity`/`models`/`repository`/`usecase`/`handlers/http`/`exceptions`), a migration runner (`db/migrate.go` + `db/history/sqlite`), DI wiring (`config → db → middleware → repos → usecases`), and a `CLAUDE.md` for onboarder platform-fit. IDs use `kuery/cryp.ULID`. The kuery version pinned in the generated `go.mod` tracks what isme currently requires — ⚠️ re-check it when touching the preset, since kuery prunes all but its 5 newest tags and a pruned pin is unresolvable even via proxy.golang.org.

  Extension points (intentionally out of scope for the generated skeleton, documented in its `CLAUDE.md`): thin test coverage (the preset ships only `container_ownership_test.go`, `web_test.go` and `internal/server/cors_test.go` — guards for traps, not domain tests), no UI (`--ui` is a future enhancement — add a Vite/React `ui/` embedded into the Go binary), **SQLite-only** (a MongoDB variant would swap `di_db.go` + repo impls and drop the migration runner), and **no auth** (wire `kuery/auth` middleware in `internal/middlewares` + `internal/server` to protect routes).

- **iot** — minimal ESP32-S3 firmware skeleton (C++/PlatformIO, **non-Go**). Renders `platformio.ini` (single esp32-s3 env pinning the shared `kuino` lib), a thin `src/main.cpp` wired to `kuino::wifi`, `include/config.h.example`, `.gitignore`, `README.md`, `CLAUDE.md`. `go mod tidy` is skipped (no `go.mod`).

gobuild now emits **non-Go** presets too; `go mod tidy` runs only when a `go.mod` was rendered (`hasGoMod`), while `git init` always runs.

### Flags

- `--name`/`-n` — project name (or positional arg).
- `--go` — Go version. The flag default is deliberately **empty** so `detectGoVersion()` reads the local toolchain (`go version`), falling back to `goVersionFallback` when the toolchain cannot be queried. ⚠️ Do not give this flag a static default: a literal `"1.24"` made the detect branch unreachable and every generated `go.mod` said `go 1.24` on a 1.27.1 toolchain, while this doc and the flag usage both claimed it followed the toolchain.
- `--http-template`/`--preset` — preset (`base|fiber|platform-service|iot`, default `base`).
- `--module`/`-m` — Go module path (defaults to `github.com/vukyn/<name>`). Threaded into `templateData.ModulePath`; the `platform-service` preset uses it for the `go.mod` module line and all internal imports. `base`/`fiber` ignore it.
- `--force`/`-f` — render into an existing **non-empty** directory. Without it, a non-empty destination is refused rather than silently overwritten. An existing `.env` is preserved even under `--force`.

### ⚠️ Input validation is a security boundary, not ergonomics

`text/template` escapes nothing, so every caller-supplied value lands verbatim in generated files. `validate.go` rejects (never sanitizes) before any rendering:

- **project name** — `^[a-zA-Z0-9][a-zA-Z0-9._-]{0,63}$`. This single pattern is also what keeps the destination inside the working directory: it admits no path separator, no `..`, no leading dot and no leading `/`, so there is deliberately **no second containment check** to keep in sync. A name of `svc", "overrides": {...}, "y": "` previously produced *valid* JSON in `ui/package.json` with an attacker `overrides` entry.
- **module path** — `golang.org/x/mod/module.CheckPath`, the same rules the go command applies. A newline in `--module` previously appended a `require github.com/attacker/backdoor v1.0.0` line to the generated `go.mod`, which the automatic `go mod tidy` then resolved.
- **Go version** — `^\d+\.\d+(\.\d+)?$`, same directive-injection surface.

Post-setup (`go mod tidy`, `git init`) failures are returned as errors. They used to print `Warning:` and then `Project setup complete` and exit 0.

Flags can appear before or after the positional name (`reorderArgs` in `main.go` normalizes ordering, since urfave/cli v2 otherwise stops flag parsing at the first positional arg). `valueFlags` lists every flag that consumes the following token so reordering skips flag values.

`base` and `fiber` are standalone "hello world" presets; **platform-service** is the one that follows the full platform service template (domains, DI, clean-architecture layers).

## Structure

```
main.go            # urfave/cli/v2 entrypoint (newApp), flag reordering, generateProject(): validate, mkdir, render, tidy + git init
validate.go        # validateProjectName/validateModulePath/validateGoVersion — the injection boundary
embed.go           # //go:embed all:templates → templatesFS
render.go          # templateData struct, dotfileNames map, outputName(), fileMode(), renderPreset()
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
- File permissions are deliberate for scaffolder output: `0755` dirs / `0644` files, annotated with `// #nosec` (G301/G306) — generated projects must be user-readable. **The one exception is `.env` (`0600`)**: it is the platform's secrets file and receives live credentials as soon as a scaffolded service is wired up, so `fileMode()` in `render.go` narrows it and `renderPreset` refuses to overwrite one that already exists. Keep the `#nosec` annotations accurate when touching the write paths — the old G306 justification said "no secrets", which stopped being true the moment `.env` was special-cased.
- **Golden fixtures + .gitignore**: `testdata/golden/<preset>/` includes fixtures named `.env`/`.gitignore`/`todo`. The per-preset golden `.gitignore` self-ignores its sibling `.env`/`todo`, so a plain `git add` skips four files; the first commit of new/changed goldens needs `git add -f testdata/golden`.
- Bump `version/version.go` when cutting a release; tag via `make tag`.

## ⚠️ Pinned dependency versions in templates rot silently

A preset's pins are not covered by any gate in this repo: the goldens only assert that the
rendered text matches, and the rendered text is wrong in exactly the same way as the
template. Two pins had already become **unresolvable** before anyone noticed:

- `templates/iot/platformio.ini.tmpl` pinned `kuino.git#v0.1.0`, pruned by kuino's
  keep-5-newest-tags retention. Every `iot` scaffold died at `pio pkg install` with
  `fatal: Remote branch v0.1.0 not found in upstream origin`.
- `templates/platform-service/go.mod.tmpl` pinned `github.com/vukyn/kuery v1.41.0`, pruned
  by kuery's identical rule. `proxy.golang.org` returns **404** for it, so the platform
  CLAUDE.md's "old versions remain fetchable via the proxy cache" does not hold — every
  `platform-service` scaffold failed `go mod tidy`.

Both were invisible because a failed `go mod tidy` used to print `Warning:` and then
`Project setup complete` at exit 0. That is now an error (see the flags section), which is
the only reason the kuery pin surfaced at all.

**When bumping a pin, rendering is not verification.** Scaffold into a throwaway directory
outside the platform root and actually build it: `go build ./...` + `go vet ./...` +
`go test ./...` for the Go presets, `pio run` for `iot` (~35s with the ESP32 toolchain
cached). The keep-5-newest rule applies to both kuino and kuery, so a pin more than five
minor versions behind should be assumed dead until proven otherwise.

## ⚠️ Preset `platform-service` propagates a root-file/catch-all trap

`templates/platform-service/internal/server/server.go.tmpl` routes exactly one root file
(`/favicon.svg`) and then `app.Get("/*", renderHomePage)`. A generated service is CORRECT
as generated — the preset ships only that one file and it is routed — but the shape breaks
the moment anyone adds a second root-level asset: the catch-all answers it with index.html
at **status 200**, so nothing 404s and logs look healthy.

That is exactly how gardener (scaffolded from this preset) shipped a manifest, icons and
`sw.js` that all returned HTML — Add-to-Home-Screen had no name or icon, iOS used a
screenshot of the page, and `registerSW()` died on the MIME type so the service worker
never installed and offline was dead in production. Fixed there in PR #106; rainy fixed
the same thing its own way earlier.

Audited 2026-08-10, **template NOT yet changed**. The plan, the exact handler to port, the
`.webmanifest` Content-Type trap, the one-segment limitation, and the golden-file
regeneration step are in `docs/pwa-root-file-audit.md`. ⚠️ Changing the template means
`go test -update` — the current golden
(`testdata/golden/platform-service/internal/server/server.go:104-108`) encodes the defect.
