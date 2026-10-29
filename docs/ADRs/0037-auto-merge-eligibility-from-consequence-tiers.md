---
title: "37. Auto-merge eligibility comes from deterministic consequence tiers and per-tier evidence"
status: Accepted
relates_to:
  - graduated-approval-policy
  - autonomy-spectrum
  - review-autonomy-evidence
  - trustworthiness-evidence
topics:
  - auto-merge
  - risk
  - review
  - governance
---

# 37. Auto-merge eligibility comes from deterministic consequence tiers and per-tier evidence

Date: 2026-10-26

## Status

Review TBD


## Context

An earlier draft, ADR 0110
([#7151](https://github.com/fullsend-ai/fullsend/pull/7151), closed), proposed a
dedicated Auto-Merge agent that decides each merge. That path was dropped: the
decision stays deterministic, and GitHub's own branch protection and merge queue
do the merging. The
[graduated approval problem doc](../problems/graduated-approval-policy.md) asks
for risk routing with thresholds "derived from observed outcomes, not guessed"
that a compromised or poorly calibrated scorer can't steer. A spike that tried the
Gemara policy model for auto-merge reached the same starting point: our own
rules on GitHub with fullsend's own repositories first, with structured,
trustworthy evidence as the main gap
([#8790](https://github.com/fullsend-ai/fullsend/pull/8790),
[experiment](https://github.com/fullsend-ai/experiments/tree/main/0029-gemara-auto-merge)).

The only PR-level risk mechanism today is the
[ADR 0089](0089-pr-risk-assessment-scoring.md) score. When it is enabled, the
review post-script turns an approval into a comment if the score reaches a
threshold or is missing
([`scripts/post-review.src.sh` at `15e2d79`](https://github.com/fullsend-ai/agents/blob/15e2d79acdac44bb264d114d232e80b7ed579b33/scripts/post-review.src.sh#L627-L700)).
Being a weighted model judgment, it can average a severe signal away, and it
says nothing about the evidence a PR needs.

We replayed the rules below on 1,796 merged `fullsend` and `agents` pull
requests (May to September 2026). 7% landed at T0, 28% at T1, 15% at T2, 27% at
T3 and 23% on protected paths. None of the 128 T0 PRs needed a medium- or
high-severity fix within 30 days; T1 needed one 4.2% of the time, T2 9.5% and
T3 15.2% ([spike](../spikes/2026-10-26-pr-fix-likelihood.md)).

## Options

- **The ADR 0089 score as the tier.** Rejected. A good average can hide one
  serious signal, and the score ranks fixes worse than PR size and kind of
  change (see the spike).
- **A model decides eligibility.** Rejected. A model-assigned tier is open to
  prompt injection and drift. A model's approve or reject can still be recorded
  next to each decision, so an eval can measure its accuracy.
- **Protected path lists only.** Rejected. Paths miss reach: a small change to
  a shared library can break three repositories. Paths stay as one input that
  can only raise the tier.
- **Merge when CI is green.** Rejected. CI covers only what it tests.

## Decision

The Auto-Merge stage decides eligibility with a versioned, deterministic policy.

### Tier

Rules set a consequence tier from facts about the change, never from labels or
agent-reported scores: what behavior changes and whether it is reachable, how
far it reaches (callers, consumers, other repositories), whether it touches a
core path the repository declares or a rarely used feature, whether interfaces,
data, permissions or secrets change, and how hard it is to undo.

| Tier | Meaning |
|---|---|
| T0 | No intended change to shipped behavior, e.g. a mechanical docs fix |
| T1 | Limited behavior change with bounded exposure and easy recovery |
| T2 | Reachable behavior change with broader impact in the repository |
| T3 | Sensitive, cross-repository, incompatible or hard to recover |

A serious signal sets a floor. Other signals, and unknown facts, can only raise
the tier. A model judgment, such as an LLM's view of how central a feature is,
may raise a tier but never lower it; the first version doesn't use one. Changes
to protected paths (policy, prompts, credentials, workflows, evaluators) always
need a human, whatever their tier.

### Evidence

Every PR needs authorized intent, a completed review of its head revision, the
required CI runs, scope and protected-path checks, and security results. Higher
tiers add evidence for their failure modes:

- T1: proof of how new code is activated (callers, flags, registration);
- T2: behavioral tests that cover the affected paths, plus integration evidence;
- T3: code-owner review and recovery evidence.

Missing, stale, unknown or contradictory evidence makes a PR ineligible. A
waiver is a human policy decision and stays visible in the record.

### Grants

By default the stage records each PR's tier, evidence state and recommendation,
and merges nothing. Unattended merge needs a person to grant a cohort: a
repository, a work class, a tier and an agent configuration. A grant is
time-bounded and backed by measured outcomes. A severe outcome suspends it
automatically, and only a person restores or widens it. A change of model,
prompt or tool re-checks whether the grant still applies.

The ADR 0089 score stays advisory. It can raise scrutiny, as it does today, but
never sets or lowers a tier.

## Forge scope

Forge-neutral. The rules need a PR's diff, files, review and CI state, and the
repository's declared core paths. GitHub comes first. There,
each decision is published as a check run on the PR's head commit, so branch
protection and the merge queue can require it like any other CI check. GitLab
and Forgejo need their own state adapters; neither is built.

## Consequences

- A serious signal can no longer be outweighed by favorable ones, and every
  eligibility decision traces to rules, facts and evidence.
- Repositories must declare their core and protected paths; until they do,
  unknown reach raises the tier.
- Shadow mode produces the outcome data a grant needs before anything merges
  unattended.
- Collectors for reach, activation and coverage of affected paths must exist
  before a T1 or T2 cohort can be granted.
- The rules and path declarations are policy under human governance; an agent
  change to them always needs a person.
