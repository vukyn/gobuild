---
name: rejection-test-masked-by-earlier-check
description: A validation-rejection test can pass because a DIFFERENT validator fires first — delete the check it names and it stays green; only mutation reveals it
metadata:
  type: feedback
---

When several validators run in sequence over related inputs, a test that asserts
"input X is rejected" proves only that *something* rejected it. Delete the check
the test is named after and it can stay green.

**Why:** in gobuild (PR #18, 2026-09-13) `TestGenerateProjectRejectsUnsafeInput`
asserted that a path-traversal / JSON-injection project name is refused. But the
module path *defaults* to `github.com/vukyn/<name>`, so `module.CheckPath` saw
the poisoned name first and errored on it. Removing the project-name regex
entirely left the test green in 4 of its 6 cases. The mutation matrix is the only
reason this was caught; reading the test, it looks airtight.

The same run turned up its sibling: the traversal case asserted "nothing was
written" by walking a directory the escape landed *above*. `../../escaped` from
`root/work` lands beside `root`, outside the window. Widening the layout to
`root/outer/work` and asserting over `root` fixed it.

**How to apply:** when a test exercises validator N, pin every *other* input to a
known-good value so nothing upstream can fire — in gobuild that meant passing an
explicit valid `--module` on the name cases. Then mutate: the test must name the
specific check it covers in the failure output. And for any test asserting "the
bad thing did not happen on disk", make the observed window strictly larger than
the blast radius the exploit demonstrated, not merely the directory you expected
it to use.

Related: [[feedback-prove-regression-tests]], [[mutate-the-producer-not-just-the-logic]].
