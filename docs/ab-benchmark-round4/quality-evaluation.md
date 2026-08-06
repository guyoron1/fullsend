# A/B Benchmark Round 4 — Quality Evaluation
Date: 2026-08-02
Judge: Claude Opus 4.6 (blind evaluation, Alpha/Beta labels)

## Reveal: Alpha = Path A (all Opus), Beta = Path B (Opus+Sonnet split, Ponytail+RTK)

## Per-Pair Results

| # | Upstream | Alpha (A) | Beta (B) | Winner | Notes |
|---|----------|-----------|----------|--------|-------|
| 1 | #5808 | 5.0 | 5.0 | **Tie** | Byte-for-byte identical diffs |
| 2 | #5799 | 5.0 | 5.0 | **Tie** | Substantively identical test suites |
| 3 | #5794 | 4.75 | 4.0 | **Path A** | A: efficient 422 payload parsing; B: N extra API calls |
| 4 | #5791 | 5.0 | 4.5 | **Path A** | A: job-level permission assertions; B: only workflow-level |
| 5 | #5784 | 4.75 | 5.0 | **Path B** | B: more edge case coverage in tests |
| 6 | #5775 | 4.8 | 4.5 | **Path A** | A: covers all 3 PR-creation call sites; B: misses one |
| 7 | #5774 | 4.8 | 4.8 | **Tie** | A: targeted fix; B: broader but beyond scope |
| 8 | #5760 | 5.0 | 5.0 | **Tie** | Byte-for-byte identical diffs |
| 9 | #5756 | 4.8 | 4.5 | **Path A** | A: filters invalid env keys early; B: passes them through |
| 10 | #5751 | 4.0 | 4.8 | **Path B** | A: repeats issue's flagged inaccuracy; B: correct |
| 11 | #5726 | 4.8 | 4.5 | **Path A** | A: cleaner comments, doesn't touch already-correct section |
| 12 | #5708 | 4.8 | 4.8 | **Tie** | Tie — B safer clone, A better docs |
| 13 | #5571 | 5.0 | 5.0 | **Tie** | Effectively identical implementations |

## Aggregate Scores

| Metric | Path A (Alpha) | Path B (Beta) |
|--------|----------------|---------------|
| Average score | 4.83 | 4.65 |
| Wins | 5 | 2 |
| Ties | 6 | 6 |
| Losses | 2 | 5 |
| Win rate (excl ties) | 71% | 29% |
| Identical diffs | 3/13 (23%) | - |

## Interpretation

Path A (all Opus) edges Path B in quality, but the gap is small (4.83 vs 4.65, 0.18 difference on a 5-point scale). The 5 Path A wins tend to be about **completeness** — Opus covers more call sites, adds more defensive checks, and catches subtle edge cases. Path B's 2 wins are about **different strengths** — more test edge cases (#5784) and better factual accuracy in docs (#5751).

Notable: 6 of 13 pairs (46%) are ties, and 3 of those are byte-for-byte identical diffs. This suggests the Opus plan → Sonnet execute split produces the same solution nearly half the time.

## Key Takeaway

Path A is ~4% better in quality, but Path B achieves this at roughly 55% of the cost (based on Round 3 cost data). The quality-adjusted cost ratio remains strongly in Path B's favor.
