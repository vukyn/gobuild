# Memory Index

## Project state
- [DI container leak rollout](project_di_container_leak_rollout.md) — per-repo fix status; tomatime still carries the bug

## Verification workflows
- [gobuild golden + scaffold gate](reference_gobuild_golden_workflow.md) — `go test . -update` plus an end-to-end scaffold-and-build outside the platform root
- [Verification = exit codes](feedback_verification_exit_codes.md) — never `| tail` a build/test; the filter has shown green for a red run
- [Prove tests fail without the fix](feedback_prove_tests_fail_without_fix.md) — revert experiment, and keep the revert compilable

## Multi-file mechanical sweeps
- [Verify sweeps with numstat, not grep](feedback_verify_sweep_counts_not_greps.md) — rtk hook breaks `grep -lZ | xargs -0`; glob loops + `git diff --numstat`
- [No git checkout during an uncommitted sweep](feedback_uncommitted_sweep_no_git_checkout.md) — it restores the lines the sweep removed; `cp` aside instead
