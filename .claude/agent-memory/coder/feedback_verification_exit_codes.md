---
name: feedback-verification-exit-codes
description: Report verification as raw exit codes; never pipe go build/vet/test through tail or a filtered summary
metadata:
  type: feedback
---

Report verification results as **exit codes captured directly**, never as filtered or
tail-piped output. The pattern to use:

```
go test ./... > /tmp/t.log 2>&1; echo "exit=$?"; grep -cE '^ok' /tmp/t.log; grep -E '^(FAIL|--- FAIL)' /tmp/t.log
```

**Why:** this environment's output filter has printed a passing-looking summary for a
run that actually FAILED, and `cmd | tail` discards the command's exit status (the
pipeline reports `tail`'s status instead). Either alone can make a red run look green,
and "never claim success without running the checks" is only meaningful if the
observation is trustworthy.

**How to apply:** every backend verification round (`go build ./...`, `go vet ./...`,
`go test ./...`, `-race`) and every frontend one (`npm run lint`, `npx tsc -b`).
Redirect to a log file, echo `$?` immediately on the same line, then grep the log for
detail. Quote the exit codes verbatim in the report rather than paraphrasing
("all green"). Also useful: a test whose code logs to stdout can bury the `--- FAIL`
lines in hundreds of KB of JSON log noise — grep the log rather than reading it.

See also [[feedback-prove-tests-fail-without-fix]].
