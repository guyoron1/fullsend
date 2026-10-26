# SPIKE: Can we measure how likely a merged PR is to need a fix?

**ADRs:** [0037](../ADRs/0037-auto-merge-eligibility-from-consequence-tiers.md) (eligibility) · [0079](../ADRs/0079-pr-fix-likelihood-is-advisory-and-validated-per-repository.md) (likelihood) · **Problem:** [graduated-approval-policy](../problems/graduated-approval-policy.md) · **Data and code:** `pr-risk-signals` (to be public before the ADRs merge) · **Date:** 2026-10-26

## Answer

Partly. PR size within its repository plus the kind of change ranks which PRs
later need a fix, and holds on later PRs. Nothing else we tried added to it.
The ranking is weak in some repositories, and the labels are not yet checked by
people. That supports using likelihood to raise scrutiny, not to grant merges.

## Data and outcome

- 8,734 merged PRs from nine repositories: `fullsend`, `agents`, `hypershift`,
  `installer`, `machine-config-operator`, `osac`, `rhdh`, an Ansible collection
  and `systemd`, merged October 2025 to September 2026.
- **Outcome:** a later PR fixed a medium- or high-severity defect that this PR
  introduced, within 30 days. Candidate pairs came from later PRs and issues
  that name, revert or change the same lines. An AI code read judged each
  pair. Every defect got a second, independent read, and disagreements got a
  third.
- **Kind of change:** a classifier put each PR in one of nine work classes
  (bug fix, new feature, docs, and so on).
- **Method:** fit on the earlier half of each repository's PRs, check on the
  later half.

## Results

**Fix rate by kind of change** (all repos, medium+, 90% interval):

| Kind of change | PRs | Rate |
|---|---|---|
| documentation | 879 | 0.5% (0.2–1.0%) |
| dependency update | 934 | 2.1% (1.5–3.1%) |
| bug fix | 2,669 | 3.9% (3.4–4.6%) |
| agent prompts or policy | 172 | 5.8% (3.5–9.5%) |
| CI, build or release | 843 | 10.6% (8.9–12.4%) |
| new feature | 1,459 | 12.7% (11.3–14.2%) |

**Three bands** (size third within the repository × riskier class: new feature,
CI/build/release, agent prompts or policy), on later PRs:

| Band | PRs | Needed a medium+ fix |
|---|---|---|
| lower | 2,296 | 1.7% |
| middle | 1,455 | 6.6% |
| higher | 619 | 13.4% |

**Rules compared on later PRs.** *Gate* is the share of fixes caught in the
riskiest 30% of PRs. *Effort* is the share caught in the riskiest PRs that
together hold 20% of changed lines.

| Rule | AUC | Gate 30% | Effort 20% |
|---|---|---|---|
| three bands | 0.706 | 56% | 31% |
| bands, ordered by size inside | 0.744 | 65% | 27% |
| fitted model on size + kind | 0.763 | 67% | 1% |

The fitted model is best at gating and worst at spending a review budget,
because it puts the biggest PRs first.

**Per repository.** Ordering by size inside the bands beats the bands in eight
of nine repositories. The bands rank poorly in `agents` (AUC 0.56), `rhdh`
(0.60) and `machine-config-operator` (0.61). Five repositories have under 20
medium+ fixes in their later half, too few to confirm a rule on their own.

**What added nothing on top of size and kind** (later-PR AUC gain under 0.01):

- about 20 merge-time signals: files, directories, churn, review counts,
  commits, hours open, author history, tests changed, CI and dependency files;
- four file-history signals from the defect-prediction literature (change
  entropy, developers on the files in 90 days, days since the files last
  changed, top contributor share);
- 52 review, approval and merge-process fields on `fullsend` and `agents`,
  including open review findings and fix runs;
- the non-LLM metrics of DRS-OSS, which mostly restate size.

An independent pipeline, with a 60-day window and its own labels, also found
no established gain from structural signals beyond repository and kind of change.

**The ADR 0089 score.** On the 283 PRs that carry it (19 medium+ fixes, a
small sample), it ranks fixes at AUC 0.62, against 0.78 for the bands on the
same PRs. The review result
records the composite score and the signal values, but not per-tier scores, so
it cannot be calibrated.

**Consequence tiers on a replay.** We applied the ADR 0037 tier rules to
1,796 `fullsend` and `agents` PRs (May to September 2026):

| Tier | PRs | Needed a medium+ fix |
|---|---|---|
| T0 | 128 | 0 (0%) |
| T1 | 500 | 4.2% |
| T2 | 274 | 9.5% |
| T3 | 486 | 15.2% |
| protected paths | 408 | 9.3% |

## Limits

- **Labels are AI code reads.** The literature finds about one automatic defect
  label in eight wrong. A blind human check of 40 pairs is prepared but has
  not run.
- **Missed fixes.** A fix that never touches the original's lines and never
  names it is missed. Forward checks found this rare for medium+ defects, but
  not absent.
- **No observed fix is not proof of safety.** For example, 0 fixes in 128 T0
  PRs bounds the rate below about 2.3% at 95%, and says nothing about
  undetected defects.
- **Size includes generated and vendored code.** That is how the bands were
  measured, and they held better across repositories than production lines did.

## Open questions

- 30 or 60 days as the outcome window.
- Whether a model judgement of how central the changed feature is would raise
  the right PRs. Under ADR 0037 it can only raise a tier; a deterministic
  declaration of core paths comes first.
- An LLM diff scorer (the DRS-OSS model) has not been piloted. It needs a GPU
  and published weights, and it would have to pass ADR 0079 rule 2.
