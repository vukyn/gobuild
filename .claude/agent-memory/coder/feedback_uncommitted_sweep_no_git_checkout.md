---
name: feedback-uncommitted-sweep-no-git-checkout
description: Never use `git checkout -- <file>` to undo a temporary edit while an uncommitted multi-file sweep is in progress — it restores the very lines the sweep removed
metadata:
  type: feedback
---

While a large uncommitted change is in the working tree, do **not** use
`git checkout -- <file>` to undo a temporary/experimental edit. It resets the file to
**HEAD**, which silently reinstates the lines the sweep deliberately removed. Undo a
temporary edit by re-applying the sweep's own transformation to that one file, or by
copying the file aside (`cp file /tmp/x`) before the experiment and copying it back.

**Why:** during the medioa2 DI-container-leak fix (86 identical `defer ctn.Delete()`
lines removed across 17 files, nothing committed), I re-added one defer to prove a new
static guard fired, then "reverted" with `git checkout --`. That restored the defer from
HEAD, quietly undoing one of the 86 removals. Only re-checking the diff count
(`git diff --numstat` → expected 86 deletions) caught it. The committer agent would have
shipped an incomplete sweep, and the bug it reintroduces is invisible at runtime.

**How to apply:** whenever the working tree is intentionally dirty and I run a
revert-experiment (prove-the-test-fails, prove-the-guard-fires), snapshot with `cp`
before and restore with `cp` after — and re-assert the sweep's invariant (a total diff
count, or the guard test itself) as the last step. See
[[feedback-verify-sweep-counts-not-greps]].
