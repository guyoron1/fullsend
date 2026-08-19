---
title: "0092. Implementation plan at the triaged gate"
status: Proposed
relates_to:
  - intent-representation
  - autonomy-spectrum
topics:
  - agents
  - planning
---

# 0092. Implementation plan at the triaged gate

Date: 2026-08-19

## Status

Proposed (implements [#113](https://github.com/fullsend-ai/fullsend/issues/113))

<!-- Once this ADR is Accepted, its content is frozen. Do not edit the Context,
     Decision, or Consequences sections. If circumstances change, write a new
     ADR that supersedes this one. Only status changes and links to superseding
     ADRs should be added after acceptance. -->

## Context

The pipeline already pauses for a human between triage and code. An issue that triage accepts but does not auto-promote gets the `triaged` label — "waits for human review (applies to features and uncategorized issues)" — and stays there until a human applies `ready-to-code` or runs `/fs-code`.

That gate exists, but the human standing at it has only the triage comment: a classification and a summary of *what* the issue is. Nothing says *how* the change would be made. So the promote decision is made blind to the approach, and the first artifact that reveals the approach is the finished PR.

[#113](https://github.com/fullsend-ai/fullsend/issues/113) has asked for the missing half since the original design review:

> The output would be a plan comment on the issue - perhaps updated in place rather than re-posted on every change. The implementation agent would read that as its input.

This ADR decides that one thing: **what a human sees at the `triaged` gate.**

Scope boundaries with three neighbours. [#797](https://github.com/fullsend-ai/fullsend/issues/797) (planning agents for feature brainstorming) was closed `not planned`, and the refinement track ([#2562](https://github.com/fullsend-ai/fullsend/issues/2562) — explore, refine, critique) shapes features *before* they become issues; neither is in scope. `needs-design` ([#6183](https://github.com/fullsend-ai/fullsend/issues/6183)) is the opposite case — it marks an issue as under-specified and routes it to a human to supply the requirement, confirmed by [#6374](https://github.com/fullsend-ai/fullsend/issues/6374) listing it as an actionable human task in `/nextwork`. This ADR applies only where the requirement is already clear and the approach is not.

## Options

### Option A: Plan comment on the issue at the `triaged` gate

A `plan` agent triggers on `triaged`, posts an implementation plan as a comment, and edits it in place on revision. The human promotes with `ready-to-code` as they do today.

- No new label, no new gate, no new state — the decision point already exists.
- Matches the shape asked for in #113.
- The plan is versioned only by the forge's comment edit history.

### Option B: Plan as a pull request in a proposals repo

The plan is a file on a branch; merging is the approval.

- Strongest provenance: CODEOWNERS, signed commits, revisions as diffs.
- Requires a proposals repo and a second enrollment surface, and replaces a label click with a PR review.
- Disproportionate for the common case; worth revisiting for high-risk changes.

### Option C: Extend the triage agent to emit a plan

No new agent; triage's output gains a plan section.

- Cheapest to build.
- Overloads a persona tuned for classification with a design task, and couples plan quality to triage's sandbox and prompt.
- A plan is only worth writing for issues that reach the gate; triage runs on all of them.

## Decision

Adopt **Option A**. Add a `plan` agent that posts an implementation plan comment on issues labelled `triaged`. Promotion stays exactly as it is today: a human applies `ready-to-code` or runs `/fs-code`.

No new labels. No new pipeline states. The only new verb is `/fs-plan [instruction]`, which regenerates the plan with the human's feedback and follows the `/fs-fix` pattern precisely — free text into `HUMAN_INSTRUCTION`, write-level ACL, and a matching `/fs-plan-stop`.

Everything else reuses a shipped primitive:

| Need | Mechanism |
|---|---|
| Trigger | CEL `trigger` on `harness/plan.yaml` over `NormalizedEvent` ([ADR 0061](0061-harness-cel-dispatch.md)) |
| Artifact | Issue comment, edited in place across revisions |
| Approval | The existing `triaged` → `ready-to-code` promotion |
| Revision | `/fs-plan [instruction]`, mirroring `/fs-fix` |
| Iteration cap | Count the plan bot's prior comments on the issue, mirroring how `reusable-fix.yml` counts prior fix commits |
| Handoff | Pre-script injects the approved plan as `plan.md`, mirroring the fix agent's `review-body.txt` |
| Output contract | `schemas/plan-result.schema.json` + `validation_loop` ([ADR 0022](0022-harness-level-output-schema-enforcement.md)) |
| Repo read access | `providers/github-ro.yaml` + `profiles/fullsend-github-ro.yaml`; no push token, no registries |

**The plan agent reads the repository.** It runs in a sandbox with the target repo checked out, like the code and fix agents, because a plan that names real files and functions can be checked against reality and a plan written from the issue text alone can only restate the issue. Its sandbox needs strictly less than the code agent's: `providers/github-ro.yaml` and `profiles/fullsend-github-ro.yaml` for read-only forge access, `providers/vertex-ai.yaml` for inference, and nothing else — no package registries, because it reads code rather than building it, and no push path, because the comment is posted by the post-script on the runner ([ADR 0017](0017-credential-isolation-for-sandboxed-agents.md)). Repository content is untrusted input, so the agent inherits the same sandbox posture as code and review; having no write credential at all makes its blast radius smaller than either.

**A plan must be grounded in the repository it read.** Every file path in a plan is either a real path discovered during inspection or a clearly marked new path in an existing directory that matches local conventions; a plan proposes no new framework, test runner, or layout where the repository already establishes one; and if repository inspection fails, the agent returns a blocking note saying so rather than a plausible guess. Without this rule repo access buys nothing — an ungrounded plan is an issue restatement with invented paths, and it is worse than no plan because it reads as authoritative. The prompt-level detail belongs in `agents/plan.md`; the requirement belongs here because it is the acceptance criterion for the artifact this ADR introduces.

**The plan is guidance, not a contract.** If implementation disproves the plan, the code agent implements what is correct and records the divergence in its PR body. A plan that cannot be departed from converts every planning error into a code error.

**Iteration state is derived, not stored.** The agent counts its own prior comments to compute the current iteration and fails closed to the cap. The forge holds the state, so nothing can desync.

**Deliverable:** `agents/plan.md`, `harness/plan.yaml`, `schemas/plan-result.schema.json`, and `docs/plan.md` in [`fullsend-ai/agents`](https://github.com/fullsend-ai/agents), plus a `fullsend-ai-planner` identity. No change to `fullsend run`, the dispatcher, the label set, or the sandbox model.

## Consequences

- **The promote decision stops being blind.** A human at the `triaged` gate sees the intended approach before authorizing code, and rejecting it costs a comment instead of a PR.
- **The code agent starts from an approved plan** rather than re-deriving intent from the issue body — the same input improvement the fix agent gets from `review-body.txt`.
- **Issues that reach `triaged` get slower and cost one more sandboxed agent run.** Repo access means a real sandbox with a checkout, not a cheap prompt call — closer to a triage run than a code run, but not free. The gate already made these issues wait for a human; this adds that cost before the wait begins.
- **Bugs and docs changes are unaffected**, because triage auto-promotes them straight to `ready-to-code` and never stops at `triaged`. Extending the plan step to them would require a new gate, which this ADR deliberately does not create.
- **Plans can go stale.** If the issue body is edited after the plan is posted, the plan may no longer match the request. Re-triage already re-runs on issue edits; the plan agent inherits that trigger and will overwrite its comment.
- **Plan revision can ping-pong** — [flapping-convergence.md](../problems/flapping-convergence.md)'s "review ping-pong" one stage earlier. The iteration cap bounds it; it does not prevent it.

### Open questions

- **Should a plan agent that cannot produce a defensible plan apply `needs-design`?** That would turn a triage guess about specification quality into an evidence-backed outcome, at the cost of spending a planning run to reach it.
- **Is `plan` its own agent or a mode of the code agent?** A single agent would share context between planning and implementation; the fix agent's precedent ([#197](https://github.com/fullsend-ai/fullsend/issues/197)) chose persona separation for exactly this trade-off.
