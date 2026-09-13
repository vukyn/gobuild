---
name: gobuild-golden-fixture-workflow
description: Verifying a gobuild template change — golden regeneration plus the end-to-end scaffold-and-build gate that go test alone does not cover
metadata:
  type: reference
---

Editing anything under `gobuild/templates/` has two verification surfaces, and
`go test ./...` only covers one of them.

1. **Goldens** — `testdata/golden/<preset>/` is a byte-for-byte snapshot;
   `TestRenderPresetGolden` fails until you run `go test . -update`. The
   `git add -f testdata/golden` caveat in `gobuild/CLAUDE.md` applies only to the
   fixture files named `.env` / `.gitignore` / `todo` (the per-preset golden
   `.gitignore` self-ignores its siblings). Already-tracked fixtures are unaffected
   by `.gitignore`, so an ordinary source-file change needs no `-f` — check with
   `git check-ignore -v <paths>` rather than reflexively forcing.
2. **The generated project must still compile**, and nothing in gobuild's own test
   suite compiles it — the goldens live under `testdata/`, which the Go tool ignores.
   So a template edit that leaves an import unused or missing passes every gobuild
   gate and breaks scaffolding for real. Always scaffold into a throwaway dir
   OUTSIDE the platform root and run `go build ./...` + `go vet ./...` there.
   `go mod tidy` runs during generation and needs network.

   Since 2026-09-13 there is a target for this: **`make scan-generated`**
   (`scripts/scan-generated.sh`) renders every preset into a temp dir and runs
   govulncheck + gosec on whatever produced a `go.mod`, `npm audit` on whatever
   produced a `ui/package.json`. The preset list is read off `templates/*/`, so
   adding a preset needs no edit to it. ~17s warm. `.github/workflows/check.yml`
   runs `make check` on PR/push and this weekly.
3. **`iot` sits outside both of the above.** No Go module and no `package.json`,
   so every Go/npm scanner has nothing to open — `scan-generated` prints it as an
   explicit `skip`. Its only real gate is **`pio run` on the scaffold** (~35s with
   the ESP32 toolchain cached). Deliberately not wired into the script: it would
   make every run depend on a PlatformIO install. Run it by hand whenever
   `templates/iot/` changes, above all the kuino pin — that pin was dead
   (`v0.1.0`, pruned) and every golden test passed anyway, because a golden proves
   the rendered text matches the template, never that a pin resolves.

**Why:** a broken generated project is worse than the bug being fixed, because it
breaks scaffolding entirely rather than degrading one service.

**How to apply:** any change to `templates/**` → edit, `go test . -update`,
`go build/vet/test` in gobuild, then the end-to-end scaffold-and-build. Reported
exit codes, not filtered output — `| tail` discards `go test`'s exit status.
Related: [[di-container-leak-rollout]].
