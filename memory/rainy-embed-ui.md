---
name: rainy-embed-ui
description: "PLATFORM STANDARD: every UI-serving Go service embeds its built React UI via go:embed (internal/web) + route-code-split; NEVER commit the bundle; build-web before go build"
metadata:
  node_type: reference
  type: project
---

## PLATFORM STANDARD (canonical — applies to ALL UI-serving services, present & future)

Every Go service that serves a built React UI MUST embed it into the binary via `//go:embed` and lazy-route-split the bundle. NOT a per-service choice — this is the platform default. Adopted across ALL services 2026-06-20: rainy (#186 split/#189 embed), medioa2 (#61/#62), isme (#67/#68). A NEW service (or the gobuild `platform-service` preset scaffold) MUST start with this pattern; do NOT reintroduce runtime `html.New("internal/ui")` + `http.Dir` serving or commit the built bundle.

**The pattern (copy verbatim, adapt favicon name + module path):**
1. `internal/web/web.go` — `//go:embed all:dist` → `var embedded embed.FS`; `func FS() fs.FS { sub, err := fs.Sub(embedded, "dist"); if err != nil { panic(err) }; return sub }`. Package `web`. Import as `<module>/internal/web`.
2. `internal/web/web_test.go` — embed-root sanity test (`TestEmbedRoot`: `.gitkeep` present; when a real bundle is embedded, `index.html` + `assets/` resolve).
3. `internal/web/dist/.gitkeep` — committed; keeps `all:dist` valid on a fresh checkout (else `go build` fails "pattern all:dist: no matching files").
4. `internal/server/server.go`: `uiFS := web.FS()`; `html.NewFileSystem(http.FS(uiFS), ".html")`; `/assets` → `filesystem.New{Root: http.FS(fs.Sub(uiFS,"assets"))}` (log+os.Exit on the fs.Sub err); favicon → `filesystem.SendFile(c, http.FS(uiFS), "<favicon>.svg")`; SPA catch-all stays `c.Render("index", …)`.
5. `Makefile build-web`: `npm run build` → `rm -rf internal/web/dist` → `mv ui/dist internal/web/dist` → `touch internal/web/dist/.gitkeep`.
6. `.gitignore`: `internal/web/dist/*` + `!internal/web/dist/.gitkeep`. Real assets NEVER committed.
7. `ui/src/App.tsx`: every page `React.lazy` behind ONE `<Suspense>` (structural wrappers/guards stay eager). `ui/vite.config.ts manualChunks`: group `react-router`(+dom) via `if id.includes("node_modules/react-router") return "react-router"` BEFORE the array check (else react-router-dom → empty chunk + warning), plus `@chakra-ui`/`react-icons`/`axios`.

**Hard rules:**
- **build-web MUST run before `go build`** — go:embed reads files at COMPILE time. Deploy = `make build-web` → `go build` → ship the single self-contained binary (no co-shipping internal/ui, no CWD dependence).
- **NEVER commit the bundle** (only `.gitkeep`). Committing it caused medioa2/isme's recurring orphan-bundle-on-rebuild — both were `git rm`'d during migration.
- `internal/web` is internal → safe even for published modules (isme: no effect on `github.com/vukyn/isme/external` consumer surface).

**Per-service deltas:** favicon rainy=`rainy.svg`, medioa2+isme=`favicon.svg` (isme preserves a browser-globe-fallback comment). Entry-chunk drops: rainy 827→226kB, medioa2 1023→204kB, isme 647→203kB. medioa2's lazy-split exposed + fixed a `utils/axios↔apis/auth↔utils-barrel` import cycle (watch for barrel cycles when splitting). Stale-UI-needs-restart still applies (Fiber loads templates at startup). See [[demo-mock-source-of-truth]], [[isme-ui-serving]].

**Follow-up:** gobuild `platform-service` preset (scaffolds new services mirroring isme) should be updated to emit this go:embed + split pattern out-of-the-box — see [[gobuild-preset-system]]. Until then, apply manually to any new service.
