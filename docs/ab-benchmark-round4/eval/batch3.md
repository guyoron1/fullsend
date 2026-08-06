# Batch 3 Evaluation (Pairs 10-13)

## Pair 10: #5751 — Document compose/diff bidirectional invariant in contributing docs
Alpha: correctness=4 quality=4 completeness=4 conciseness=4 (avg=4.0)
Beta:  correctness=5 quality=5 completeness=5 conciseness=4 (avg=4.8)
Winner: Beta
Notes: Both add a `harness-composition.md` doc and link it from AGENTS.md. Alpha contains a factual error — it states the diff functions "were removed with the scaffold agent extraction" (repeating the ADR inaccuracy the issue explicitly calls out) and describes them as not currently existing, but the issue clearly says `DiffHarness` still exists in `internal/harness/diff.go`. Beta correctly handles this nuance. Beta also provides more detailed merge semantics tables (listing actual per-field behavior like "scalars: child overrides; slices: concatenated") and distinguishes between `mergeForgeConfigInto` (composition) and `mergeForgeConfig` (runtime) semantics with a note about their differences. Beta's checklist is more actionable with the "if diff functions exist / do not exist" branching. Overall Beta is more accurate and more useful to agents.

## Pair 11: #5726 — Update robots.txt to enable agent discovery of documentation and examples
Alpha: correctness=5 quality=5 completeness=4 conciseness=5 (avg=4.8)
Beta:  correctness=5 quality=4 completeness=4 conciseness=5 (avg=4.5)
Winner: Alpha
Notes: Both solutions are nearly identical — adding `Allow: /docs/` and `Allow: /llms.txt` before `Disallow: /` for each training crawler. The key difference: Alpha leaves `Applebot` and `Applebot-Extended` untouched (Applebot already had `Allow: /`), while Beta adds `Allow: /docs/` and `Allow: /llms.txt` to `Applebot-Extended` before its `Disallow: /`. This is fine but slightly inconsistent — if Applebot-Extended needs docs access, so might other bots already handled. Alpha also includes better header comments explaining the rationale for the changes (mentioning `llms.txt` provides a machine-readable index). Minor edge, Alpha wins for cleaner commentary and not touching the already-correct Applebot section.

## Pair 12: #5708 — Triage runner does not execute pre_script from harness configuration
Alpha: correctness=5 quality=5 completeness=5 conciseness=4 (avg=4.8)
Beta:  correctness=5 quality=5 completeness=5 conciseness=4 (avg=4.8)
Winner: Tie
Notes: Both solutions are extremely similar in scope and quality. Both: (1) add triage-specific pre-script tests (`TestRunAgent_TriagePreScriptExitsNonZero`, `TestRunAgent_TriagePreScriptExitsZero_ProceedsToSandbox`), (2) generalize `newSkipHarnessDir` into `newSkipHarnessDirForAgent`, (3) fix the compose.go forge-script inheritance bug with `clearInheritedForgeScripts`/`snapshotChildForgeScripts`, (4) add comprehensive compose_test.go tests, and (5) update ADR-0045. Beta has one additional touch — updating `docs/guides/user/bring-your-own-agent.md` with a footnote about the forge-script exception, and Beta's `clearInheritedForgeScripts` clones the ForgeConfig before mutating (avoiding shared pointer mutation from `mergeForgeBlocks`). Alpha mutates in-place which could be a subtle bug if the ForgeConfig pointer is shared. However, the issue asked specifically about triage runner not executing pre_script, and the forge-script inheritance fix goes beyond scope — both implementations go equally far beyond. Beta's clone-before-mutate is slightly more correct, but Alpha's comment documentation on `clearInheritedForgeScripts` is more thorough. Calling it a tie — Beta has a safety edge (clone), Alpha has a documentation edge.

## Pair 13: #5571 — Add post-script dedup guard for concurrent retro proposals
Alpha: correctness=5 quality=5 completeness=5 conciseness=5 (avg=5.0)
Beta:  correctness=5 quality=5 completeness=5 conciseness=5 (avg=5.0)
Winner: Tie
Notes: These diffs are effectively identical. Both implement the same `fullsend file-issue` CLI command with Jaccard title similarity, the same `SearchIssues` interface addition to `forge.Client`, the same GitHub and GitLab implementations, the same fake client, the same CLI docs updates, and the same comprehensive test suites (title similarity tests, dedup integration tests, edge cases for cross-repo, dry-run, search failure fallthrough, comment failure non-fatal). The only differences are issue number references (#800 vs #848 in comments) which are cosmetic (local fork issue numbers). The code, structure, and tests are byte-for-byte identical in substance.
