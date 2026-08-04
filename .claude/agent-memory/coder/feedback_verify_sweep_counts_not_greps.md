---
name: feedback-verify-sweep-counts-not-greps
description: The rtk shell hook rewrites grep/find and reformats their output, so `grep -lZ … | xargs -0` silently passes one giant filename — drive multi-file sweeps with shell globs and verify with `git diff --numstat`
metadata:
  type: feedback
---

Multi-file mechanical edits in this workspace must not be driven by a
`grep -rl … | xargs` pipeline. The rtk hook (see the user's global `RTK.md`) rewrites bare
`grep`/`find` into `rtk grep`/`rtk find`, which **reformats the output** — it prints
summaries like "88 matches in 10 files" and drops the `-Z`/`--null` separators. The
result is that `xargs -0` receives every path joined into a single argument and the whole
sweep silently no-ops with `Can't open …: No such file or directory`.

Use a plain shell glob loop instead, e.g.
`for f in internal/domains/*/handlers/http/handler.go; do perl -ni -e '...' "$f"; done`,
and verify the result with `git diff --numstat` (expected insertions/deletions), not with
another grep.

**Why:** the medioa2 DI-container fix needed exactly 86 identical lines removed across 17
files. Two `grep -rlZ | xargs -0 perl -ni` attempts appeared to run but changed nothing;
`git diff --numstat` was the only signal that told the truth. `numstat` also gives the
exact deletion count a report needs to claim exhaustiveness.

**How to apply:** any time I am about to delete/replace an identical line in more than a
couple of files: glob loop to edit, `git diff --numstat` to prove the count, and — where
the invariant matters — a test that scans the tree for the forbidden pattern so it stays
gone. Also relevant when reading command output: rtk-reformatted grep output is a summary,
not raw lines, so don't parse it. See
[[feedback-uncommitted-sweep-no-git-checkout]].
