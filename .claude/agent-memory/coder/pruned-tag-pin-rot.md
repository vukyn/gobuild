---
name: pruned-tag-pin-rot
description: kuery/kuino keep only 5 tags, so any pin older than that is DEAD — and proxy.golang.org does NOT keep pruned kuery versions, contrary to the platform CLAUDE.md
metadata:
  type: project
---

The platform's keep-5-newest-tags retention rule (kuery, kuino) silently kills
every pin older than five minor versions. As of 2026-09-13 two were already dead
and had been for a long time:

- `gobuild/templates/iot/platformio.ini.tmpl` pinned `kuino.git#v0.1.0` →
  `pio pkg install` fails with `fatal: Remote branch v0.1.0 not found in
  upstream origin`. Live tags: `v0.2.0 v0.2.1 v0.2.2 v0.3.0 v0.3.1`.
- `gobuild/templates/platform-service/go.mod.tmpl` pinned
  `github.com/vukyn/kuery v1.41.0` → every scaffold failed `go mod tidy`. Live
  tags: `v1.57.0` … `v1.61.0`.

⚠️ **The platform CLAUDE.md claims "old versions remain fetchable via
proxy.golang.org cache". That is FALSE for kuery.** Verified directly:
`proxy.golang.org/github.com/vukyn/kuery/@v/v1.41.0.info` returns **404 —
invalid version: unknown revision**. Deleting a kuery tag really does break
every consumer pinned to it, with no cache to fall back on.

**Why it stayed invisible:** gobuild's golden fixtures assert that rendered text
matches the template, and the rendered text was wrong in exactly the same way as
the template — so both dead pins passed every test for months. gobuild also
printed `Warning: Failed to run go mod tidy` and then `Project setup complete` at
exit 0, so even a human running it saw green. (Both fixed, PRs #18/#19.)

**How to apply:** treat any kuery/kuino pin more than ~5 minor versions behind as
dead until proven otherwise — `go list -m <mod>@<ver>` or `git ls-remote --tags`
takes seconds. Rendering or diffing is **not** verification for a pin; only
resolving it is. A push-triggered CI gate cannot catch this class at all (nothing
in the repo changes when a tag is deleted upstream) — it needs a **scheduled**
run, which is why gobuild's `check.yml` has a weekly cron.

Before bumping, diff the library's public headers across the jump rather than
assuming breakage: kuino v0.1.0 → v0.3.1 looked like two minor versions of risk
and was purely additive (`git diff <old>..HEAD -- src/kuino/*.h` was empty except
for a new module).

Related: [[kuery-shared-lib-rule]].
