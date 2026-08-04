---
name: feedback-prove-tests-fail-without-fix
description: For a bug fix, temporarily revert the fix and capture the test failure output — keep the revert COMPILABLE
metadata:
  type: feedback
---

When shipping a bug fix with new tests, **prove the tests actually fail without the
fix**: back up the file, replace the fix with a no-op, run the suite, capture the
failure output, restore, re-run green. Report the captured failure output.

⚠️ **Keep the revert compilable.** Deleting a fix usually strands an import or a
variable, so the "revert" fails to build and proves nothing about the tests. Neutralise
the fix while keeping every symbol referenced — e.g. keep the `defer func(){…}()` block
and swap the real call for a log line that still uses the same logger import. (In
gardener this cost three attempts before it compiled.)

**Why:** a test that passes both with and without the fix is decoration. This is
especially load-bearing when the bug is *invisible* — e.g. a double `Delete` on a
sarulabs/di container returns nil and changes no observable state, so only a
call-counting assertion can see it.

**How to apply:** any fix + regression-test pair. Run TWO experiments when the suite
has two distinct assertions: one that breaks the fix (proves the primary assertion is
live) and one that breaks the *invariant the fix protects* (proves the secondary
counter/idempotency assertion is live, not vacuous). Report both.

See also [[feedback-verification-exit-codes]].
