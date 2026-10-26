# SPIKE: Can we measure how likely a merged PR is to need a fix?

**ADRs:** [0037](../ADRs/0037-auto-merge-eligibility-from-consequence-tiers.md) (eligibility), [0079](../ADRs/0079-pr-fix-likelihood-is-advisory-and-validated-per-repository.md) (likelihood) · **Problem:** [graduated-approval-policy](../problems/graduated-approval-policy.md) · **Date:** 2026-10-26

Data and code are in `pr-risk-signals`, to be made public before the ADRs
merge. Its `EVIDENCE.md` lists every number below with its source and the
command that recomputes it.

## Answer

Partly. PR size within its repository, plus the kind of change, ranks which PRs
later need a fix, and the ranking holds on later PRs. Nothing else we tried
added to it. It is weak in some repositories, and no person has checked the
labels yet. That supports using likelihood to raise scrutiny, not to grant
merges.

## Data and outcome

We studied 8,734 PRs merged between October 2025 and September 2026 in nine
repositories: `fullsend`, `agents`, `hypershift`, `installer`,
`machine-config-operator`, `osac`, `rhdh`, an Ansible collection and `systemd`.

The outcome is a later PR fixing a medium- or high-severity defect that this PR
introduced, within 30 days. Candidate pairs came from later PRs and issues that
name the PR, revert it or change its lines. An AI code read judged each pair;
every defect got a second, independent read, and disagreements a third. A
classifier put each PR in one of nine work classes (bug fix, new feature, docs
and so on). Every rule was fitted on the earlier half of each repository's PRs
and checked on the later half.

## Results

### Fix rate by kind of change

All repositories, medium+ fixes, 90% interval.

| Kind of change | PRs | Rate |
|---|---|---|
| documentation | 879 | 0.5% (0.2–1.0%) |
| dependency update | 934 | 2.1% (1.5–3.1%) |
| bug fix | 2,669 | 3.9% (3.4–4.6%) |
| agent prompts or policy | 172 | 5.8% (3.5–9.5%) |
| CI, build or release | 843 | 10.6% (8.9–12.4%) |
| new feature | 1,459 | 12.7% (11.3–14.2%) |

### Three bands

Size third within the repository × a riskier class (new feature,
CI/build/release, agent prompts or policy), on later PRs:

| Band | PRs | Needed a medium+ fix |
|---|---|---|
| lower | 2,296 | 1.7% |
| middle | 1,455 | 6.6% |
| higher | 619 | 13.4% |

### Rules compared on later PRs

*Gate* is the share of fixes caught in the riskiest 30% of PRs. *Effort* is the
share caught in the riskiest PRs that together hold 20% of changed lines.

| Rule | AUC | Gate 30% | Effort 20% |
|---|---|---|---|
| three bands | 0.706 | 56% | 31% |
| bands, ordered by size inside | 0.744 | 65% | 27% |
| fitted model on size + kind | 0.763 | 67% | 1% |

The fitted model is best at gating and worst at spending a review budget,
because it puts the biggest PRs first.

Ordering by size inside the bands beats the plain bands in eight of nine
repositories. The bands rank poorly in `agents` (AUC 0.56), `rhdh` (0.60) and
`machine-config-operator` (0.61). Five repositories have under 20 medium+ fixes
in their later half, too few to confirm a rule on their own.

### What added nothing

None of these improved on size and kind of change:

- 12 merge-time signals beyond size: directories, churn, the files' fix
  history, reviews, commits, hours open, author history, tests changed, CI and
  dependency files, generated share and bot authorship. Each added nothing to
  size on held-out repositories, except touching CI (+0.02), which the bands
  already count as a riskier kind of change. A model with every signal scores
  AUC 0.717, below size and kind alone (0.763).
- Four file-history signals from the defect-prediction literature (change
  entropy, developers on the files in 90 days, days since the files last
  changed, top contributor share): size and kind plus these score 0.749.
- 52 review, approval and merge-process fields on `fullsend` and `agents`,
  including open review findings and fix runs: the best one adds 0.008. Open
  findings rise with size, mostly low-severity ones, and high-severity open
  findings add nothing either.
- The non-LLM change metrics of DRS-OSS score 0.729 on their own, below size
  and kind; they mostly restate size.

An independent pipeline, with a 60-day window and its own labels, also found no
established gain from structural signals beyond repository and kind of change.

### The ADR 0089 score

On the 283 PRs that carry it (19 medium+ fixes, a small sample), it ranks fixes
at AUC 0.62 (95% interval 0.52–0.72), against 0.78 (0.66–0.89) for the bands on
the same PRs. The review result records the composite score and the signal
values but not per-tier scores, so it can't be calibrated.

### Consequence tiers on a replay

The ADR 0037 tier rules applied to 1,796 `fullsend` and `agents` PRs (May to
September 2026):

| Tier | PRs | Needed a medium+ fix |
|---|---|---|
| T0 | 128 | 0 (0%) |
| T1 | 500 | 4.2% |
| T2 | 274 | 9.5% |
| T3 | 486 | 15.2% |
| protected paths | 408 | 9.3% |

## Limits

- The labels are AI code reads. Automatic defect labels are known to be noisy:
  in five open-source projects, 33.8% of issue reports were misclassified and
  39% of files marked defective never had a bug (Herzig, Just and Zeller, ICSE
  2013). Ours use a different method, two independent code reads per defect,
  but the blind human check of 40 pairs hasn't run yet.
- A fix that never touches the original's lines and never names it is missed.
  Forward samples in five repositories (212 PRs, each followed for 30 days)
  held 39 medium+ defects; this method found 35 of them, and all 15
  high-severity ones.
- No observed fix is not proof of safety. 0 fixes in 128 T0 PRs bounds the rate
  below about 2.3% at 95%, and says nothing about undetected defects.
- Size includes generated and vendored code. That is how the bands were
  measured, and it held better across repositories than production lines did.

## Open questions

- 30 or 60 days as the outcome window.
- Whether a model judgment of how central a changed feature is would raise the
  right PRs. Under ADR 0037 it can only raise a tier, and declared core paths
  come first.
- An LLM diff scorer (the DRS-OSS model) hasn't been piloted. It needs a GPU and
  published weights, and would have to pass ADR 0079 rule 2.
