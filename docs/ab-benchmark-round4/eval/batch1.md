# Benchmark Round 3 - Batch 1 Evaluation

Evaluated: 2026-08-02
Judge: Claude Opus 4.6

---

## Pair 1: #5808 — internal/config: defaults.auto_merge is parsed but never consumed
Alpha: correctness=5 quality=5 completeness=5 conciseness=5 (avg=5.0)
Beta:  correctness=5 quality=5 completeness=5 conciseness=5 (avg=5.0)
Winner: Tie
Notes: The two diffs are byte-for-byte identical. Both cleanly remove AutoMerge from the struct, default initializer, tests, YAML fixtures, ADR docs, and TypeScript type definition. No differences to distinguish.

## Pair 2: #5799 — Verify ${VAR} substitution semantics for unset harness env vars
Alpha: correctness=5 quality=5 completeness=5 conciseness=5 (avg=5.0)
Beta:  correctness=5 quality=5 completeness=5 conciseness=5 (avg=5.0)
Winner: Tie
Notes: Substantively identical solutions. Both add TestEnvVarExpansionSemantics with the same 9 subtests covering unset/empty/set vars across runner_env, env.runner, env.sandbox, mixed literals, and host_file optional/required paths. Only cosmetic differences (issue reference number in comment, minor line-break formatting in assertion messages).

## Pair 3: #5794 — mint: undefined recovery behavior when a config.yaml repo no longer exists in the org
Alpha: correctness=5 quality=5 completeness=5 conciseness=4 (avg=4.75)
Beta:  correctness=4 quality=4 completeness=5 conciseness=3 (avg=4.0)
Winner: Alpha
Notes: Alpha parses GitHub's 422 error payload to extract invalid repo names directly, then retries once with valid repos only -- efficient (2 API calls total) and precise. Beta makes N individual GET /repos/{org}/{repo}/installation calls to validate each repo before retrying, which is O(N) extra API calls and less efficient. Alpha also includes defense-in-depth (RepoNamePattern validation on parsed names, json.RawMessage for flexible GitHub response formats). Beta is significantly larger due to handler-level recovery logic and adds a FindInstallation fallback for repos[0] deletion, which is useful but adds complexity. Alpha's approach is architecturally cleaner -- keeping recovery inside CreateInstallationToken rather than spreading it across handler and github layers.

## Pair 4: #5791 — Upgrade stage workflow test permissions assertions
Alpha: correctness=5 quality=5 completeness=5 conciseness=5 (avg=5.0)
Beta:  correctness=5 quality=4 completeness=4 conciseness=5 (avg=4.5)
Winner: Alpha
Notes: Both correctly upgrade from assert.Contains to YAML-parsed structural checks across all six stage tests. Alpha goes further by also asserting job-level permissions inheritance (assert.Nil on job permissions for reusable workflow calls), which the issue explicitly requested as item 5 ("negative assertions for permissions that should NOT be present"). Beta only checks workflow-level permissions for 5 of 6 tests, adding job-level checks only for retro's debounce job. Alpha's consistent job-level assertions are a meaningful completeness advantage for security-relevant tests.

## Pair 5: #5784 — analyze-transcript: add test coverage for the analyzer script
Alpha: correctness=5 quality=5 completeness=4 conciseness=5 (avg=4.75)
Beta:  correctness=5 quality=5 completeness=5 conciseness=5 (avg=5.0)
Winner: Beta
Notes: Both create comprehensive test suites with identical test fixtures and the same Makefile integration. Beta includes ~53 more lines of tests with meaningful additional coverage: empty file detection, scopeSpans OTLP variant, exact non-UTF8 parse count, additional _check_block_error paths (normal tool_result, exit code, EACCES keyword), line range early-stop, and more precise host-filter JSON assertions that verify excluded hosts are truly absent. Both are well-structured; Beta simply covers more edge cases from the issue's "key areas" list.

---

## Summary

| Pair | Alpha | Beta | Winner |
|------|-------|------|--------|
| 1 (#5808) | 5.0 | 5.0 | Tie |
| 2 (#5799) | 5.0 | 5.0 | Tie |
| 3 (#5794) | 4.75 | 4.0 | Alpha |
| 4 (#5791) | 5.0 | 4.5 | Alpha |
| 5 (#5784) | 4.75 | 5.0 | Beta |

**Overall: Alpha 2, Beta 1, Ties 2**

Alpha average across all pairs: 4.90
Beta average across all pairs: 4.50
