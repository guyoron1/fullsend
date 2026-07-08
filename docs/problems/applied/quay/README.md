# Applied: quay

Organization-specific considerations for applying fullsend to the
[quay](https://github.com/quay/) GitHub organization.

Quay is an open-source container registry. It is a fullsend consumer
where autonomous agents triage issues, produce code, and review PRs.

> **Note:** This document currently focuses on a single investigation
> (CI-config autonomy readiness). Additional sections covering
> technology landscape, how general problem areas apply, and broader
> organizational context will be added as quay is explored further.

## Autonomy readiness evidence: CI-config-only changes

This section tracks evidence toward autonomy readiness for a specific
change class: **CI workflow configuration changes** (additions to path
exclusion lists, filter updates, and similar mechanical edits to
`.github/workflows/*.yaml` files).

### Hypothesis

For CI-config-only PRs in quay/quay, mandatory human review adds no
value beyond what the review agent already provides. If evidence
confirms this, CI workflow files could graduate to a tier where review
agent approval is sufficient without mandatory human review.

### Observed evidence

| # | PR | Change description | Human review time | Substantive comments | Review agent findings | Date |
|---|----|--------------------|-------------------|----------------------|----------------------|------|
| 1 | [quay/quay#6398](https://github.com/quay/quay/pull/6398) | 2-line addition of path exclusions to `sentinel.yaml` | ~2 min | 0 | None (ran 16 min after human approval) | 2026-07-07 |

### Validation criteria

Collect data from 5 total CI-config-only PRs in quay/quay (including any
already recorded above). For each, record:

1. Human review time (from PR creation to approval)
2. Whether the human left substantive comments (not just "LGTM")
3. Review agent findings (empty, aligned with human, or divergent)

If all 5 show human approval in under 5 minutes with no substantive
comments and review agent findings are empty or aligned, this change
class is autonomy-ready.

### Edge cases to evaluate separately

- **Mixed PRs** (CI config + application code) do not qualify — they
  require full review regardless of the CI config portion.
- **New workflow files** (vs. edits to existing ones) may need separate
  evaluation — adding a new workflow has higher blast radius than
  tweaking path filters on an existing one.

### Proposed graduation path

1. **Evidence collection** (current phase) — populate the table above
   as CI-config-only PRs are observed.
2. **Shadow mode** — once 3+ of the 5 target PRs confirm the pattern,
   begin logging whether the review agent would have approved before
   the human did, without changing actual merge requirements. Apply
   shadow analysis retroactively to the PRs already observed during
   evidence collection.
3. **Graduation** — if retroactive and prospective shadow analysis
   across all 5 observed PRs confirms the review agent would have
   approved before the human did, consider adding CI workflow files to
   [Tier 0 standing rules](../../intent-representation.md#tier-0-standing-rules-no-per-change-intent-needed)
   or a new autonomy tier where review agent approval is sufficient.

### Related issues

- [#138](https://github.com/fullsend-ai/fullsend/issues/138) —
  Autonomy readiness for documentation repos (mechanical consistency
  checks)
- [#148](https://github.com/fullsend-ai/fullsend/issues/148) —
  Fast-path for config-only PRs (reducing review agent cost)

### Open questions

- What is the decision process if the pattern does not hold (e.g., a
  human reviewer catches something the review agent missed)? Should the
  evidence table record negative data points explicitly?
- Should shadow mode require a minimum observation period (calendar
  time) in addition to the PR count threshold?
- How should review agent version changes during the evidence
  collection period affect the validity of earlier data points?

### Confidence level

**Low.** This is a single data point. Confidence will increase as more
CI-config-only PRs are observed and the validation criteria are met.
