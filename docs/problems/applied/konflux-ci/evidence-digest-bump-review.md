# Evidence: Review agent fully covered human review on digest bump PRs

Evidence from [konflux-ci/crossplane-control-plane](https://github.com/konflux-ci/crossplane-control-plane) demonstrating that the review agent provides equivalent or better coverage than human review for Renovate-authored single-line digest bump PRs.

This evidence informs the autonomy readiness discussion in [autonomy-spectrum.md](../../autonomy-spectrum.md). That document defines a binary per-repo autonomy model; this evidence argues for per-change-class graduation criteria, which would extend or supplement the current model.

## Summary

Across PRs #152–155 in crossplane-control-plane, the review agent approved each Renovate-authored single-line digest bump within 6–8 minutes with correct "Looks good to me" determinations. The human reviewer (org MEMBER) batch-approved all 4 with empty review bodies 6.6 days later, contributing zero additional findings.

On an earlier PR (#133) in the same repo with the same change class but a broken double-digest reference from a Renovate regex misconfiguration, the review agent correctly identified the critical malformed-image-reference issue. This single example suggests the agent can discriminate between safe and broken digest bumps, though one data point is not sufficient to establish the capability broadly (see Limitations).

## Data

### PRs #152–155: Safe digest bumps

| PR | Change | Agent review time | Agent determination | Human review | Human findings |
|---|---|---|---|---|---|
| #152 | Single-line digest bump (quay.io/konflux-ci/crossplane-components) | 6–8 min | Approved | ~6.6 days later, empty body | 0 |
| #153 | Single-line digest bump (quay.io/konflux-ci/crossplane-components) | 6–8 min | Approved | ~6.6 days later, empty body | 0 |
| #154 | Single-line digest bump (quay.io/konflux-ci/crossplane-components) | 6–8 min | Approved | ~6.6 days later, empty body | 0 |
| #155 | Single-line digest bump (quay.io/konflux-ci/crossplane-components) | 6–8 min | Approved | ~6.6 days later, empty body | 0 |

### PR #133: Broken digest bump (discrimination test)

PR #133 was a digest bump of the same class, but with a Renovate regex misconfiguration that produced a malformed double-digest image reference. The review agent correctly identified the critical issue and did not approve. This demonstrates that the agent's approvals on #152–155 reflect genuine evaluation, not indiscriminate approval.

## Analysis

**Human-vs-agent review delta:** Zero across all 4 safe digest bump PRs. The human reviewer added no findings, comments, or context that the review agent did not already provide.

**Latency cost:** The human review gate added 6.6 days of latency with no corresponding quality contribution. For a single-line digest bump, this delay provides no security or correctness benefit.

**Discrimination capability:** The agent's correct rejection of PR #133 shows it detected a syntactically obvious anomaly (malformed double-digest reference) within this change class. This is a necessary but not sufficient signal — it indicates the agent is not blindly approving, but a single data point involving a Renovate misconfiguration does not establish the discrimination capability needed for more subtle issues such as malicious digest substitution.

## Implications

This evidence is relevant to two questions in the autonomy model:

1. **Can digest bumps graduate to autonomous merge?** The data shows zero human review delta across 4 consecutive PRs, combined with demonstrated agent discrimination on a broken variant. This is a strong signal for autonomy readiness on Renovate-authored digest bumps, at least for single-image-reference changes in repos with CI coverage.

2. **Can bot dependency PRs skip the human review stage?** The 6.6-day latency from the human gate added no quality contribution. For low-risk, high-volume change classes like digest bumps, the human review stage is pure overhead when the review agent has demonstrated discrimination capability.

## Limitations

- This evidence covers a single repo (crossplane-control-plane) and a single change class (single-line digest bumps from Renovate).
- The sample size is 4 safe PRs and 1 broken PR. Larger samples across more repos would strengthen the case.
- The broken PR (#133) was a Renovate misconfiguration, not a supply chain attack. Whether the agent would detect a malicious digest substitution is a separate question — see [security-threat-model.md](../../security-threat-model.md).

## Source

Generated from [fullsend-ai/fullsend#3351](https://github.com/fullsend-ai/fullsend/issues/3351), mirrored as [#164](https://github.com/fullsend-ai/fullsend-design/issues/164). Original retro agent observation from [konflux-ci/crossplane-control-plane#152](https://github.com/konflux-ci/crossplane-control-plane/pull/152).
