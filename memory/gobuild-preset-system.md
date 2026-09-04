---
name: gobuild-preset-system
description: gobuild template engine = embed.FS + text/template + fs.WalkDir; self-contained presets base/fiber/platform-service; add preset = add folder
metadata: 
  node_type: memory
  type: project
---

gobuild refactored (PR#3) from Go-string-const templates to **`//go:embed all:templates` + `text/template` + `fs.WalkDir`**. Core in `gobuild/render.go`: `renderPreset(preset, data, destDir)` walks `templates/<preset>/`, renders `.tmpl` files (`missingkey=error`), auto-creates nested dirs. `templateData{ProjectName, GoVersion, Preset, ModulePath}`.

**Presets (self-contained — each folder = full file set, no layering):**
- `base` — minimal hello-world (bare main.go, empty go.mod). module = bare `{{.ProjectName}}`.
- `fiber` — Fiber **v3** app, `/health`, graceful shutdown, nested `internal/handler/`.
- `platform-service` (PR#4) — full clean-arch skeleton mirroring **isme**: Fiber **v2** + Bun/SQLite + sarulabs/di + kuery v1.19.0, example `item` CRUD domain (all 6 layers), DI builder, config, cmd/main.go, db/migrate.go + history, Makefile, README, CLAUDE.md. module = `github.com/vukyn/{{.ProjectName}}` (override via `--module`/`-m` flag). **PR#7 (2026-06-20): now emits the platform UI-embed STANDARD + a complete buildable Vite UI** — `internal/web` go:embed (`//go:embed all:dist` + committed `.gitkeep`), server serves embedded FS (`html.NewFileSystem`, `/assets`, favicon `SendFile`, APIBaseURL-injecting SPA catch-all), `Vite.BaseURL` config + `VITE_API_BASE_URL`, build-web→internal/web/dist, gitignore-all-dist-except-.gitkeep; PLUS a minimal Vite 7 + React 19 + Chakra v3 `ui/` (package.json/index.html-with-{{.APIBaseURL}}-token/tsconfig/eslint/main.tsx+ChakraProvider/Home page/favicon.svg/route-lazy-split+manualChunks). Verified end-to-end: scaffold→make build-web→go build→server 200. This IS the [[rainy-embed-ui]] standard — new scaffolds get it out of the box. (`--ui` is no longer "deferred"; the preset ships a real Vite project.)

**Conventions:** `.tmpl` suffix dropped on write; dotfiles stored WITHOUT leading dot + mapped (`env`→`.env`, `gitignore`→`.gitignore`) because go:embed rejects leading-dot names; `.raw.tmpl` for files needing literal `{{`. Add a file = drop `.tmpl` in folder; **add a preset = add a folder** (only Go change = new `templateData` field + `--preset` usage string). Flag: `--http-template`/`--preset` (default base).

**Tests** (first in repo): golden snapshots in `testdata/golden/<preset>/`, `go test . -update` regenerates. GOTCHA: generated golden `.gitignore` self-ignores sibling `.env`/`todo` → committer needs `git add -f testdata/golden/<preset>`.

**Versioning (PR#5, v1.4.0):** version = git tag, resolved at runtime via `runtime/debug.ReadBuildInfo()` in `version/version.go` (NO hand-maintained string — that drifted: code said 1.4.0 but no tag → `go install @latest` gave stale 1.3.0). Logic: real `Main.Version` tag → use it; pseudo-version (local `go install` of untagged checkout) or `(devel)` → fallback `dev (<rev>[, dirty])`. Release = `make tag VERSION=x.y.z` (guarded; errors if VERSION unset; also `.PHONY` needed because `version/` dir shadows the target). `make version` = `git describe`. **gobuild is a `go install` binary, NOT a consumed module → do NOT prune old tags** (the kuery "keep 5 tags" rule does NOT apply; pruning breaks `go install @vX.Y.Z`). After bump: tag + push, that's the only source of truth.

**Onboarder integration SHIPPED:** `/pet-onboard --new <name> [preset]` = scaffold mode (default preset `platform-service`) → builds gobuild from source (global binary may be stale — always `go build -C gobuild -o bin/gobuild .`), gens into `<root>/<name>/`, then onboarder audits IN PLACE (acquire:skip; fresh scaffold has no remote/commits = EXPECTED, mark `✅ remote pending` not ❌). Edits live in `.claude/commands/pet-onboard.md` + `.claude/agents/onboarder.md` (platform root NOT a git repo → no commit needed). Future: `--ui` (Vite/React/Chakra), `--db mongo`. See [[kuery-shared-lib-rule]].
