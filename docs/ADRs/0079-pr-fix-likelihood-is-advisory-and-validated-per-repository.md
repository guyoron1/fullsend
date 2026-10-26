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

Review TBD

## Context

[ADR 0037](0037-auto-merge-eligibility-from-consequence-tiers.md) makes
eligibility depend on consequence: what a merged PR could break. This ADR
decides what a likelihood signal may do, meaning an estimate of how likely a PR
is to need a fix after it merges. The
[ADR 0089](0089-pr-risk-assessment-scoring.md) score is one such signal, and
others will be proposed.

We measured likelihood on 8,734 merged PRs in nine repositories, with a medium-
or high-severity fix within 30 days as the outcome
([spike](../spikes/2026-10-26-pr-fix-likelihood.md)). PR size within its
repository and the kind of change carry the signal: on later PRs, 1.7% of the
lowest band needed a fix and 13.4% of the highest. Nothing else added to them:
12 other merge-time signals, four file-history signals from the
defect-prediction literature, and 52 review, approval and merge-process fields
on `fullsend` and `agents`. Touching CI added a little, but the bands already
count CI changes as riskier.

Three caveats shape the decision. The bands rank poorly in three of the nine
repositories. The best formula depends on use: a size-ordered model catches
more fixes when gating PRs, but far fewer within a fixed budget of reviewed
lines. And the outcome labels are AI code reads that no person has checked yet.

## Decision

1. **Likelihood only raises scrutiny.** A signal may order review, send a PR to
   a human, downgrade an approval or raise its evidence requirement. It never
   lowers a tier, waives evidence or grants eligibility.
2. **A signal counts as evidence only after it is validated on the repository
   that uses it.** Only then may it support a cohort's grant or replace the
   default. Validation uses completed outcome windows, fits on earlier PRs and
   checks on later ones. It reports, per work class, the AUC and the share of
   fixes caught in the riskiest 30% of PRs and in the riskiest 20% of changed
   lines. The signal must beat the default on that repository, and a blind
   human check of a sample of its outcome labels must agree at least 85% of
   the time.
3. **The default is three bands.** Size within the repository plus a riskier
   work class (new feature; CI, build or release; agent prompts or policy)
   gives `lower`, `middle` or `higher`. A repository without measured outcomes
   gets `unable_to_assess`, never `lower`. Each assessment records its band,
   reasons, head commit and version, and goes stale when the head changes.
4. **The ADR 0089 score keeps its current raise-only role.** It can't support a
   grant until it passes rule 2, which needs the review result to record its
   per-tier scores.

## Forge scope

Forge-neutral. Validation needs merged-PR metadata, file lists and a clone; an
assessment needs a PR's size, work class and head commit. GitHub comes first;
GitLab and Forgejo need a merge-request fetcher.

## Consequences

- Policy gets measured reasons to escalate: a fix rate per band, and an
  explicit `unable_to_assess`.
- No score can make a PR eligible, so a miscalibrated or gamed score costs
  review time, not safety.
- Outcome measurement and label checks become standing work for any repository
  that wants a score to support a grant.
- The default depends on a work-class classifier; without it, a PR counts as
  riskier.
