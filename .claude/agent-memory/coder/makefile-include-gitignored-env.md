---
name: makefile-include-gitignored-env
description: `include ./.env` + a gitignored .env = every make target dies on a fresh clone; sgo still has it, gobuild fixed 2026-09-13
metadata:
  type: project
---

Several platform Makefiles open with

```make
include ./.env
export $(shell sed 's/=.*//' ./.env)
```

and `.env` is gitignored in those same repos. On any checkout that is not the
author's own — **every CI runner** — make never reaches a recipe:

```
Makefile:2: .env: No such file or directory
sed: ./.env: No such file or directory
make: *** No rule to make target `.env'.  Stop.      (exit 2)
```

Not just the target that needs `.env` — *all* of them, including `build` and
`version`, which never read it.

**Why:** GNU make treats a missing `include` as fatal and then tries to build the
missing file as a target. The fix is one character: `-include ./.env`, plus
guarding the export (`export $(shell [ -f ./.env ] && sed 's/=.*//' ./.env)`).
In gobuild only `tag` actually read a value out of `.env`, and it already errored
on its own when `VERSION` was unset — so nothing was lost.

**Status 2026-09-13:** fixed in **gobuild** (PR #20, needed before any workflow
could call `make`). **`sgo` still has the bare form with a gitignored `.env`** —
same breakage, unfixed, and it has no CI yet so nothing is failing loudly.
Re-derive rather than trusting this line:
`for d in */Makefile; do grep -l '^include ./.env' "$d"; done` then
`git -C <repo> check-ignore -q .env`.

**How to apply:** adding CI to any platform repo — check this first. A workflow
step that runs `make` will fail on a line the author has never seen fail.
