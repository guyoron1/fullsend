---
title: "79. PR fix likelihood is advisory, raise-only and validated per repository"
status: Accepted
relates_to:
  - graduated-approval-policy
  - code-review
  - review-autonomy-evidence
topics:
  - risk
  - auto-merge
  - scoring
  - review
---

# 79. PR fix likelihood is advisory, raise-only and validated per repository

Date: 2026-10-26

## Status

Accepted

## Context

[ADR 0037](0037-auto-merge-eligibility-from-consequence-tiers.md) makes
eligibility depend on consequence: what a merged PR could break. It does not
say what a likelihood signal may do, meaning how likely a PR is to need a fix
after it merges. The [ADR 0089](0089-pr-risk-assessment-scoring.md) score is
one such signal, and others will be proposed.

Likelihood has been measured on 8,734 merged PRs in nine repositories, with a
medium- or high-severity fix within 30 days as the outcome
([spike](../spikes/2026-10-26-pr-fix-likelihood.md)). PR size within its
repository and the kind of change carry the signal: on later PRs, 1.7% of the
lowest band needed a fix and 13.4% of the highest. Nothing else added to
them: 12 other merge-time signals (only touching CI added a little, and the
bands already count CI changes as riskier), four file-history signals from
the defect-prediction literature, and 52 review, approval and merge-process
fields on `fullsend` and `agents`. The bands rank poorly in three of the nine repositories.
The best formula depends on use: a size-ordered model catches more fixes when
gating PRs, but far fewer within a fixed budget of reviewed lines. The
outcome labels are AI code reads that no person has checked yet.

## Decision

1. **Likelihood only raises scrutiny.** A likelihood signal may order review,
   send a PR to a human, downgrade an approval or raise its evidence
   requirement. It never lowers a tier, waives evidence or grants
   eligibility.
2. **A signal is validated on the repository that uses it before it counts as
   evidence.** Only a validated signal may support a cohort's grant or replace
   the default. Validation uses completed outcome windows and fits on earlier
   PRs, checking on later ones. It reports, per work class, ranking (AUC) and
   the share of fixes caught in the riskiest 30% of PRs and in the riskiest
   20% of changed lines. The signal must beat the default on that
   repository, and a blind human check of a sample of its outcome labels must
   agree at least 85% of the time.
3. **The default is three bands.** Size within the repository plus a riskier
   work class (new feature; CI, build or release; agent prompts or policy)
   gives `lower`, `middle` or `higher`. A repository without measured
   outcomes gets `unable_to_assess`, never `lower`. Each assessment records
   its band, reasons, head commit and version, and is stale when the head
   changes.
4. **The ADR 0089 score keeps its current raise-only role.** It does not count
   as evidence for a grant until it passes rule 2, which needs the review
   result to record its per-tier scores.

## Forge scope

Forge-neutral. Validation needs merged-PR metadata, file lists and a clone;
an assessment needs a PR's size, work class and head commit. GitHub is
implemented first; GitLab and Forgejo need a merge-request fetcher.

## Consequences

- Policy gains measured reasons to escalate, with a fix rate per band and an
  explicit `unable_to_assess`.
- No score can make a PR eligible, so a miscalibrated or gamed score costs
  review time, not safety.
- Outcome measurement and label checks become standing work for any
  repository that wants a likelihood signal to support a grant.
- The default depends on a work-class classifier; when it is unavailable the
  PR counts as riskier.
