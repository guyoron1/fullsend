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

Accepted

## Context

ADR 0110 (in review,
[#7151](https://github.com/fullsend-ai/fullsend/pull/7151)) decides who may
merge: a dedicated Auto-Merge stage whose trusted runtime holds the credential
and checks current forge state. It leaves the policy that decides which pull
requests are eligible, and the cohort and rollout rules, to follow-up work.
The [graduated approval problem doc](../problems/graduated-approval-policy.md)
asks for routing by risk, with thresholds "derived from observed outcomes, not
guessed", and resistant to a compromised or poorly calibrated scorer.

Today the only PR-level risk mechanism is the
[ADR 0089](0089-pr-risk-assessment-scoring.md) composite score. When enabled,
the review post-script downgrades an approval to a comment if the score reaches
a threshold or is missing
([`scripts/post-review.src.sh` at `15e2d79`](https://github.com/fullsend-ai/agents/blob/15e2d79acdac44bb264d114d232e80b7ed579b33/scripts/post-review.src.sh#L627-L700)).
It is a weighted model judgement, so a severe signal can be averaged away, and
it does not say what evidence a PR needs.

A replay of 1,796 merged `fullsend` and `agents` pull requests (May to
September 2026) against the rules below placed 7% at T0, 28% at T1, 15% at T2,
27% at T3 and 23% on protected paths. None of the 128 T0 PRs needed a medium-
or high-severity fix within 30 days; T1 needed one 4.2% of the time, T2 9.5%
and T3 15.2% ([spike](../spikes/2026-10-26-pr-fix-likelihood.md)).

## Options

- **The ADR 0089 score as the tier.** Rejected: a favourable average can hide
  one serious signal, and the score ranks fixes worse than PR size and kind of
  change do (see the spike).
- **A model decides eligibility.** Rejected: ADR 0110 already keeps the model
  advisory, and a model-assigned tier is open to prompt injection and drift.
- **Protected path lists only.** Rejected: paths miss reach. A small change to
  a shared library can break three repositories. Paths remain one input that
  can only raise the tier.
- **Merge whenever CI is green.** Rejected: CI covers only what it tests, and
  the evidence a change needs depends on what it can break.

## Decision

The Auto-Merge stage decides eligibility with a versioned, deterministic policy
in three parts.

**1. A consequence tier from trusted facts.** Rules compute the tier from what
the change does and what it touches, never from labels or agent-reported
scores:

- what behavior changes, and whether it is reachable;
- how far it reaches: callers, consumers and other repositories;
- how important the affected area is: a repository's core paths, which it
  declares, rank above a rarely used feature;
- whether interfaces, data, permissions or secrets change;
- how hard the change is to undo.

| Tier | Meaning |
|---|---|
| T0 | No intended change to shipped behavior, e.g. a mechanical docs fix |
| T1 | Limited behavior change with bounded exposure and easy recovery |
| T2 | Reachable behavior change with broader impact in the repository |
| T3 | Sensitive, cross-repository, incompatible or hard to recover |

A serious signal sets a floor and other signals can only raise the tier. An
unknown fact raises it. A model judgement, such as an LLM's view of how
central a feature is, may raise a tier but never lower it, and the first
version does not need one. Changes to protected paths (policy, prompts,
credentials, workflows, evaluators) always need a human, whatever their tier.

**2. Required evidence per tier.** Every tier needs:

- authorized intent;
- a completed review bound to the head revision;
- the required CI runs;
- scope and protected-path checks;
- security results.

Higher tiers add evidence for their failure modes:

- T1: proof of how new code is activated (callers, flags, registration);
- T2: behavioral tests that cover the affected paths, plus integration evidence;
- T3: code-owner review and recovery evidence.

Missing, stale, unknown or contradictory evidence makes a PR ineligible. A
waiver is a human-owned policy decision, and the record keeps showing it as
waived.

**3. Unattended merge only under a human grant, shadow by default.** The stage
records every PR's tier, evidence state and recommendation, and merges
nothing, until a person grants a cohort. A cohort is a repository, a work
class, a tier and an agent configuration. A grant is time-bounded and backed
by measured outcomes. A severe outcome suspends it automatically; only a
person loosens or restores it. A change of model, prompt or tool re-checks
whether the grant still applies.

The ADR 0089 score stays an advisory signal. It can raise scrutiny, as it does
today, but it does not set or lower a tier.

## Forge scope

Forge-neutral. The rules need a PR's diff, files, review and CI state and the
repository's declared core paths. GitHub is implemented first, consistent with
ADR 0110. GitLab and Forgejo need their own state adapters; neither is
implemented.

## Consequences

- A serious signal can no longer be outweighed by favourable ones, and every
  eligibility decision can be traced to rules, facts and evidence.
- Repositories must declare their core paths and protected paths; until they
  do, unknown reach raises the tier.
- Shadow mode produces the outcome data a grant needs before any repository
  merges unattended.
- Evidence collectors (reach, activation, coverage of affected paths) become
  required work before T1 or T2 cohorts can be granted.
- The rule set and core-path declarations are policy files under human
  governance; an agent change to them always needs a person.
