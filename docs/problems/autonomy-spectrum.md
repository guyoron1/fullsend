# Autonomy Spectrum

When should agents auto-merge, and when should they escalate to humans?

## The model: binary with CODEOWNERS

The autonomy model is **binary per-repo** with **CODEOWNERS as the escape hatch**:

- A repo is either "agent-autonomous" or it isn't
- Within an autonomous repo, specific file paths can require human approval via CODEOWNERS
- Repos graduate to autonomous mode once they meet readiness criteria

This avoids the complexity of fine-grained per-change-type rules while still protecting critical paths.

## CODEOWNERS as the control mechanism

CODEOWNERS is already a well-understood GitHub mechanism. In the agentic context, it means:

- **Agent-owned paths** — agents can review and merge without human approval
- **Human-owned paths** — changes here always require human approval, regardless of what the agent thinks

### Likely candidates for human-owned paths

- Security policies and RBAC configuration
- API surface (CRD schemas, REST endpoints, protobuf definitions)
- Deployment manifests and release configuration
- Agent configuration and system prompts (critical — agents must not modify their own guardrails)
- **CODEOWNERS files themselves** — always human-owned, never agent-modifiable. This is a hard rule, not a suggestion. If agents could modify CODEOWNERS, they could remove their own guardrails.
- Cross-repo interface contracts
- UX-facing components
- **Test files for human-owned production paths** — if production code at a path is human-owned, its corresponding tests should be too. Tests are part of the security boundary for the code they cover; an attacker who can weaken tests autonomously can blind the review agents to vulnerabilities in the production code. See [security-threat-model.md](security-threat-model.md#cross-cutting-attack-pattern-temporal-split-payload-test-poisoning).
- **Tekton pipeline and task definitions, Dockerfiles** — these are the build system and define what executes during builds. Agents may legitimately modify these as part of feature implementation, so blanket CODEOWNERS may be too restrictive. Whether these are human-owned or instead receive heightened review agent scrutiny without gating is a per-repo trade-off. See [security-threat-model.md](security-threat-model.md#the-xz-variant-test-data-as-covert-payload-storage).

### How CODEOWNERS interacts with agents

When a PR touches both agent-owned and human-owned paths:
- The human-owned paths block auto-merge
- The agent can still review the entire PR and provide feedback
- The human approves the guarded paths; the agent's review covers the rest

## Graduation criteria

What does a repo need before agents can be trusted to merge autonomously?

Possible criteria (all TBD — this needs experimentation):

- Minimum test coverage threshold (what number? per-package or overall?)
- CI pipeline includes integration/e2e tests, not just unit tests
- Linting and formatting enforced in CI
- CODEOWNERS file covers all security-sensitive paths
- History of successful agent-reviewed PRs (agents review but don't merge, humans validate the agent's judgment)
- No recent security incidents attributable to missed review

These criteria are all properties of the repo and the agents. But graduation also changes the role of the humans responsible for guarded paths — from active participants to approvers of agent output. Whether those humans can remain effective in that reduced role is an open question explored in [human-factors.md](human-factors.md).

## The probationary period

Before flipping a repo to full autonomy, run agents in "shadow mode":

1. Agents review PRs and produce recommendations
2. Humans still approve and merge
3. Compare agent decisions to human decisions over time
4. When confidence in alignment is high, graduate to autonomous mode

This builds trust incrementally and provides data on agent reliability.

## Alternative: per-decision escalation dimensions

The binary per-repo model is simple but coarse. An alternative (or supplement) is to evaluate each decision against a set of escalation dimensions at runtime. Instead of asking "is this repo autonomous?" the agent asks "is this particular action safe to proceed with?"

Example dimensions:

| Dimension | Low (agent proceeds) | High (escalate) |
|---|---|---|
| **Reversibility** | Undo in minutes, no data loss | Hours/days to roll back, or irreversible |
| **Blast radius** | One component, one agent | Multiple services, teams, or agents |
| **Visibility** | Internal only | Visible to users, customers, or third parties |

The rule is simple: if any dimension is high, escalate. No special cases for "strategic" vs "operational" — the dimensions apply uniformly.

### How this could supplement the binary model

Per-decision evaluation doesn't have to replace the per-repo binary model. It could layer on top of it:

- **Non-autonomous repos** stay non-autonomous — humans review everything regardless.
- **Autonomous repos** use dimensional checks as a runtime safety net. An agent operating in an autonomous repo would still escalate if it recognizes that a change is irreversible, cross-cutting, or user-visible — even if the files involved aren't in CODEOWNERS.

This addresses the gap where the binary model can miss risky changes that don't happen to touch a guarded path. CODEOWNERS catches known-sensitive files; dimensional checks catch emergent risk in the change itself.

### Trade-offs

- Requires agents to accurately self-assess dimensions in real time — a judgment call the binary model avoids entirely.
- The dimensions listed above are examples, not necessarily exhaustive. Different organizations might weight or define them differently.
- Could produce false escalations (agent is uncertain, so it escalates conservatively) or false confidence (agent misjudges blast radius). Shadow mode data would help calibrate.

## Evidence-driven autonomy classes

The binary per-repo model and per-decision escalation dimensions above describe autonomy in broad strokes. In practice, specific categories of agent action accumulate evidence that they can be trusted at a higher autonomy level before the repo as a whole graduates. These are **autonomy classes** — narrow, well-defined categories of agent behavior where empirical evidence supports granting the agent more authority.

Each autonomy class is defined by:

1. **Qualifying criteria** — what properties a change must have to fall into this class
2. **Evidence** — observed cases where the agent outperformed or matched human review
3. **Validation plan** — how to confirm the pattern holds before changing autonomy level
4. **Proposed autonomy change** — what the agent would do differently once validated (e.g., CHANGES_REQUESTED instead of COMMENT)
5. **Boundary conditions** — what the agent should *not* do even within this class

Autonomy classes are distinct from repo-level graduation. A repo that is not autonomous can still have specific autonomy classes where the review agent operates at a higher level — as long as the class boundaries are narrow enough that false positives are acceptable and domain-specific judgment is not required.

### Mechanical consistency in documentation repos

**Qualifying criteria:**

- Target repo is documentation-only (no application code)
- Finding category is mechanical consistency: typos, numbering gaps, cross-reference validation, scope coherence between document sections
- Finding does not require project-specific domain knowledge to evaluate

**Evidence:** See [applied docs](applied/) for the specific tracking data. In the baseline observation, a review agent caught mechanical consistency findings (numbering gaps, scope ambiguity between document sections) that multiple human reviewers missed — these merged without fix. The agent also caught a status inconsistency and a typo that the author fixed. However, the agent produced one domain-incorrect finding that a human reviewer dismissed because the project does not use the resource types in question.

**Boundary conditions:** The domain-incorrect finding demonstrates a clear autonomy boundary. Mechanical consistency checks (pattern-matchable, document-internal) are within the agent's reliable capability. Domain-specific architectural judgments (requiring knowledge of which technologies the project actually uses) are not.

**Proposed autonomy change:** Options for increasing review authority on validated mechanical consistency findings:

1. **CHANGES_REQUESTED for medium+ mechanical findings** — the review agent blocks merge until the author addresses mechanical consistency findings at medium severity or above. Domain-specific findings remain at COMMENT level. This is the most direct response to the evidence (medium-severity findings merging without fix under COMMENT).
2. **Auto-file tracking issues for unresolved findings** — instead of blocking merge, the agent files a tracking issue when medium+ mechanical findings go unresolved. Lower friction than CHANGES_REQUESTED but doesn't prevent the problem (findings still merge without fix).
3. **COMMENT with explicit acknowledgment prompt** — keep COMMENT but require the author to explicitly acknowledge each medium+ finding (dismiss or address). This is intermediate friction — less blocking than CHANGES_REQUESTED but more than the current silent merge.

Option 1 is the strongest match for the evidence (COMMENT was insufficient to prevent merge of medium-severity issues), but requires the validation plan to succeed first. Options 2 and 3 are lower-risk alternatives that could be adopted with less evidence.

**Validation plan:** Track the next 5 ADR PRs merged in the qualifying repo. For each, compare agent mechanical consistency findings against human reviewer comments. Success: the agent catches mechanical consistency issues that no human flags on at least 3 of 5 PRs. A case where a human catches a mechanical consistency issue the agent missed weakens the signal. See [applied/konflux-ci](applied/konflux-ci/) for the specific tracking data.

**Cross-references:**

- Distinct from the bot-dependency-bump autonomy class (where review value comes from security verification, not consistency checking)
- The scope-ambiguity finding is evidence for auto-filing tracking issues for unresolved medium+ findings
- The ADR number collision caught by a human reviewer (but missed by the agent) was a cross-PR context issue — a separate capability gap, not a mechanical consistency failure

### Other potential autonomy classes

Additional autonomy classes may emerge from operational evidence. Examples under observation:

- **CI-config-only changes** — trivial workflow file modifications where human reviewers consistently approve in under 5 minutes with no substantive comments
- **Bot dependency bumps** — automated version bumps where review value is security verification rather than code quality

Each class requires its own evidence trail and validation plan before any autonomy change.

## Open questions

- Who decides when a repo is ready for autonomy? (See [governance.md](governance.md))
- Can autonomy be revoked? Under what circumstances? Automatically if a bad merge is detected?
- How do we handle repos with poor test coverage today? Do agents help improve coverage as a prerequisite to their own autonomy?
- Is per-repo binary too coarse? Should there be sub-repo zones of autonomy beyond what CODEOWNERS provides?
- What about cross-repo changes? If a change spans an autonomous repo and a non-autonomous one, which rules apply?
- How do we handle the CODEOWNERS bootstrap — who decides the initial set of guarded paths?
- Should graduation criteria include human factors — not just "can agents be trusted here?" but "can the humans responsible for guarded paths remain effective once they stop implementing?" (See [human-factors.md](human-factors.md))
- Automation research suggests that [intermediate levels of automation preserve better human oversight than full automation](human-factors.md#the-out-of-the-loop-performance-problem). Does this argue for a third autonomy level between "human-approved" and "fully autonomous" — one where agents do most of the work but humans retain some implementation role?
- If autonomy can be revoked, should human engagement metrics (approval times, review depth, domain expert confidence) be among the triggers, alongside code quality metrics?
