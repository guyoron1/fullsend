# AISDLC-97: Risk-Tiered Minimal Viable Evidence and Mergeability Contract

Proposal v0.5 · 2026-09-27 · Guy Oron · Jira: [AISDLC-97] (parent [AISDLC-29])

> This is the full reference version, for implementers. The short proposal for review is
> [aisdlc-97-proposal.md](https://github.com/guyoron1/fullsend/blob/aisdlc-97-mve-contract/docs/proposals/aisdlc-97-proposal.md).

## 1. Summary

This proposal defines when an ADLC agent may merge a pull request on its own, when it has to fix something first,
and when it has to hand the PR to a person.

- **Every PR gets a risk tier, T0 to T4.** A script computes it from the diff and the forge; no model is involved.
  The tier is set by the riskiest thing in the PR, never by an average.
- **Each tier has a minimum set of evidence**, the Minimal Viable Evidence (MVE): which checks, tests, reviews and
  approvals must exist for that exact commit.
- **Every decision ends in one of three outcomes:** MERGE, REMEDIATE (the agent closes a named gap and tries again)
  or ESCALATE (a person takes over). Missing or unclear evidence can never lead to MERGE.
- **Branch protection does the enforcing.** Required checks, review verdicts and approvals stay with GitHub. The
  agent decides only whether a PR is inside the scope the repository declared safe for auto-merge. If it is, the
  agent turns on GitHub's merge-when-ready for that commit, and GitHub merges.
- **Changes written by an agent always get a human approving review,** as we read Red Hat's guidelines for AI code
  assistants (11.11 asks the policy owner to confirm). Merging with no approval is reserved for changes written by
  people or by deterministic bots such as Renovate.
- **Every decision is recorded** against the commit, the base and the policy version. Thresholds are tuned against
  what actually gets reverted or fixed after merge.

| Tier | Typical change | Outcome once its evidence is complete |
|---|---|---|
| T0 Pre-authorized | docs, a patch-level dependency bump, new tests only | MERGE |
| T1 Low | a small fix that only adds code or sits behind a default-off flag, with a linked issue | MERGE |
| T2 Standard | a feature slice, a refactor inside one package | a member approves and merges |
| T3 Sensitive | auth, CI structure, migrations, interfaces other repositories use, large changes | the code owner approves and merges |
| T4 Restricted | CODEOWNERS, rulesets, the policy file, agent prompts, credentials, release configuration | people only; the agent does not touch it |

On fullsend's `main` branch every PR needs one approval today, so this starts in *explicit* mode: that approval
triggers the merge, and the agent's verdict decides whether the PR is in scope. Section 10 lists what has to change
before anything merges unattended.

**Acceptance criteria:** tiers and their evidence are sections 4 and 5; missing, stale, flaky and contradictory
evidence is section 6; one outcome per tier is section 7. Retention and audit, the fourth in-scope item, is
section 8.

## 2. Scope and interfaces

In scope, as the ticket defines it: risk tiers and their deterministic inputs; the evidence each tier requires, its
permitted substitutes and its disqualifying conditions; one outcome per tier; evidence retention and audit.

Out of scope: building the enforcement, which belongs to ADR 0110 and the auto-merge stage ([fullsend#7151]), and
the blast-radius algorithm (AISDLC-96).

| Counterpart | Gives this contract | Gets from this contract |
|---|---|---|
| AISDLC-96, change impact | the affected behavior and consumers, with a confidence level | how low confidence raises the tier; how much relevant test evidence each tier needs |
| AISDLC-98, debt and golden path | the flaky-test quarantine list; the remediation workflow | which REMEDIATE outcomes route to the golden path; the mergeability record (8.1) as retained merge evidence |
| AISDLC-99, tooling | the tools that produce each kind of evidence, including the outcome baseline | the evidence classes (5.1) those tools must emit |
| AISDLC-100, test selection | the selected test subset and its fallback | the test categories each tier requires; when a cached result may count |
| ADR 0110, auto-merge boundary | trust zones, operating modes, receipts | the cohort definition and risk threshold it left open; the repository prerequisites (5.0) |

## 3. How a decision is made

1. **Check the repository.** If its branch protection doesn't meet the prerequisites in 5.0, ESCALATE.
2. **Classify the PR.** Compute the inputs, take the highest tier, apply the hard disqualifiers (section 4).
3. **Collect the tier's evidence** for the current head commit, base commit and policy version (section 5).
4. **Grade each item** as present, partial, missing, stale, unreliable, contradictory or unknown (section 6).
5. **Decide:** MERGE, REMEDIATE, ESCALATE, or WAIT for a bounded time (section 7).
6. **Record** the decision in an append-only log (section 8).

Four rules run through every step:

- **Deterministic first, model last.** A model's judgment can raise a tier or veto a merge. It can never lower a tier
  or approve a merge. This is what keeps the gate safe against prompt injection and a badly calibrated model.
- **Unknown never passes.** Anything that couldn't be measured counts as not present.
- **Policy is not risk.** Paths people always own (CODEOWNERS, CI, agent configuration, credentials) are a tier of
  their own, not a "severe finding" inside a score. In the sample, 21 of the 31 PRs with a "major or critical"
  finding had it only for this protected-path rule, 17 of them Renovate digest bumps.
- **Every refusal names what's missing.** A decision carries reason codes and a blocking list, so a person who gets
  an escalation sees exactly which gate is unmet.

## 4. Risk tiers

### 4.1 Inputs

Everything here comes from the diff, the GitHub API and a policy file kept in the repository. Line and file counts
leave out lockfiles, minified and generated files and source maps; database migrations always count.

| Input | Values |
|---|---|
| Path class of each file, from the repository's profile (Appendix B) | `restricted`, `sensitive`, `ci`, `dependency`, `test`, `docs`, `generated`, `source` |
| Change kind | docs only, test additions, test weakening, dependency pin or patch, dependency minor or major, config, source, migration or irreversible, binary |
| Size | files and lines changed |
| CI and config structure | version or digest bump only, or structural: a step, permission, secret, trigger, image or build target changed |
| Author | member (from a fork or not), allowlisted bot, first-time contributor, external fork, unlisted bot |
| AI authorship | `agent` (an agent opened the PR or wrote any commit on it), `ai-assisted` (a person who marked AI use with an `Assisted-by`, `Generated-by` or `Co-authored-by` trailer), `human` |
| Linked issue | present or absent, from the repository's intent source (closing keywords or a Jira key); labels such as `security`, `breaking-change`, `needs-design` |
| Declared PR dependencies | each one merged, unmerged, or closed without merging |
| Consumers | code, charts or overlays outside the repository that use a changed interface, or `unknown` |
| Reversibility | behind a default-off flag, behind a default-on flag, only adds code, changes behavior, irreversible |
| Test-suite delta | stronger, neutral, weakened |
| Impact and its confidence | from the AISDLC-96 model |
| Dependents | per changed package, how many packages in the repository import it, directly or not; whether a changed path is declared `core` in the policy file |
| Risky imports | new third-party modules; new imports of standard packages that run commands, use unsafe memory or do cryptography |
| File history | per changed file: reverts in the last 90 days; files that changed with it in 3 or more commits in the last 90 days but aren't in the PR; commits in the last 30 days; days since its last change; commits in the last 90 days whose message says workaround, hack or temporary |
| Advisory risk score | the ADR 0089 composite for this commit, 1–5, or `degraded` |

Two inputs have no tool to compute them yet: reversibility and test-suite delta. In the sample, 55 of 246 PRs deleted
lines in test files, and nothing in a file name tells a refactor from a loosened assertion. Until a producer
exists, both are `unknown`, and unknown never passes. Test weakening gets its own input because a test-only PR that
loosens an assertion is a guardrail change, not coverage (the split-payload attack in fullsend's threat model).

### 4.2 Tiers

A PR's tier is the highest tier of any of its files or inputs.

| Tier | The PR stays at or below this tier only if |
|---|---|
| **T0** Pre-authorized | every file is docs; or it is a dependency pin or patch bump that passes the dependency rules (4.4); or it only adds tests; or it only bumps the digest of a pinned CI action and an allowlisted bot opened it. In every case: at most 20 files, a member or allowlisted-bot author, nothing sensitive, restricted or binary, no structural CI change, no weakened tests |
| **T1** Low | at most 10 files and 100 lines; only source, docs, test or config files; the change is new code only (new files, or new functions nothing existing calls yet; a line added inside an existing function is a behavior change) or sits behind a default-off flag; no changed package has 10 or more dependents; a linked issue; a member or allowlisted-bot author; the affected packages have tests that run in CI |
| **T2** Standard | at most 25 files and 800 lines; nothing sensitive, CI or restricted; dependency changes are pin or patch only, with no new third-party module; not irreversible |
| **T3** Sensitive | anything larger; any sensitive path (auth, crypto, RBAC, tokens, secrets); a structural CI change; a minor or major dependency bump or a new third-party module; core code (a path the policy file declares `core`, or a package more than half the repository depends on); a migration or irreversible change; a consumer outside the repository (4.4); a first-time or external-fork author; weakened tests; a linked issue labeled `security`, `breaking-change` or `needs-design` |
| **T4** Restricted | any restricted file: CODEOWNERS, branch protection or rulesets, the auto-merge policy file, agent, harness, prompt, skill or hook definitions, credential and sandbox configuration, release, deployment, packaging or signing configuration, `.gitmodules` URL changes, binaries, and the tests that cover these paths |

**Floors.** The tier is the highest floor any input sets, and nothing averages it down. Beyond the table, a code
change (not only comments or docs) in a package with 10 or more dependents is at least T2, a new import of a risky
standard package is at least T2, and an advisory risk score of 4 or more is at least T3.

**The dependent count stands in for blast radius** until AISDLC-96's impact model exists, and the record says so.
Once it exists, the tier is the higher of the size tier and the impact tier: a change inside one package with no
exported symbol changed can be T1; exported functions or callers in other packages make it at least T2; callers in
other repositories or through shared templates make it at least T3. Impact confidence that comes back low or unknown
from a model that ran raises the tier by one, to at least T2. The sample put 17 of 246 PRs in T3 on size alone, two
of them only 2 files each. In fullsend on 2026-09-28, 14 of the 76 sample PRs in T1 or T2 changed a package with 10 or
more dependents (from `go list` on main); `internal/forge` alone has 40 of the repository's 70 packages depending on
it.

**Raises.** Weaker signals can only push the tier up. A changed file reverted in the last 90 days raises it by one.
Two or more of these raise it by one: a missing co-change partner, more than 10 commits to the file in 30 days, no
change to it in 180 days, workaround or hack commit messages in 90 days, an advisory risk score of 3. These are ADR
0089's git-history signals, used as raises instead of its weighted average, so one serious signal can't be diluted by
several harmless ones. Commits labeled fix don't count: in fullsend they are 1,349 of 2,806 commits in 90 days, so
they would flag nearly every file. The same history is passed to the reviewer and to E9 as context.

The size limits are starting values, taken from [Cloudflare's AI code review] tiers and the ADR 0089 rubric. They,
the 10-dependent threshold, the two-signal rule and the 30-, 90- and 180-day windows get recalibrated on outcomes (8.5).

### 4.3 Hard disqualifiers

Each of these escalates, with a reason code, whatever the tier and evidence:

- the PR changes the policy file, agent definitions, workflows, CODEOWNERS or rulesets that govern the decision
  (`policy_file_changed`);
- it is a draft, was retargeted after review, or targets a base that isn't the default branch, an allowlisted
  release branch or an open PR's branch (`draft`, `stale_base`, `unsupported_base`);
- the head or base commit isn't the one that was reviewed (`head_not_reviewed`, `stale_head`);
- the agent would merge a PR it wrote or changed (`self_merge`);
- intent is unclear: no linked issue at T1 or above, or the PR closes an issue it only partly delivers
  (`intent_unknown`, `partial_issue_closure`);
- tests were weakened and the PR isn't T3 or above with a human approval (`test_weakened`);
- it duplicates, or is superseded by, another open PR (`duplicate_work`, `superseded`);
- a declared dependency was closed without merging (`dependency_closed`).

Human holds, changes-requested reviews and unresolved threads aren't on this list, because branch protection
already blocks them (5.0).

### 4.4 Dependencies

Dependencies run three ways, and each has its own rule. The full conditions are in Appendix A.

- **Inbound** (packages this repository uses): a dependency bump is T0 only when it is a pure patch or pin bump with
  no new package, a consistent lockfile, no registry change, no new known vulnerability and an unchanged or
  allowlisted license.
  Renovate's own config already expresses part of this; the scanner and lockfile checks are what it can't.
- **Outbound** (code elsewhere that uses this change): any consumer outside the repository makes the PR T3. Its
  evidence includes the consumers' tests, and the producer merges before its consumers.
- **Between PRs:** a stacked PR or an unmerged declared dependency waits; a dependency closed without merging
  escalates.

## 5. Minimal viable evidence

### 5.0 Who enforces what, and what a repository needs first

ifireball's review of ADR 0110 ([fullsend#7151], 2026-09-23) draws the line: branch protection enforces human
holds, required checks and review verdicts, "NOT the agent". The agent decides whether the PR is inside the scope
the repository's policy declared auto-mergeable. The evidence splits the same way:

| Enforced by branch protection | Checked by the agent |
|---|---|
| E1 required checks; E3 review verdict (the review agent files blocking findings as a changes-requested review or an unresolved thread); E6 human and code-owner approval; human holds | E2 relevant tests, E4 classification, E5 intent and scope, E7 reversibility, E8 track record, E9 model veto, the disqualifiers (4.3), the dependency rules (4.4) |

The left column is the same for every PR on a branch, so a ruleset can express it. The right column changes with the
tier, so it needs the agent. The agent records the left column for audit but doesn't re-check it. The one arguable
defect in the sample, [fullsend#6994], is held exactly this way: its review bot filed a changes-requested review on
2026-09-04, and the PR was still unmerged on 2026-09-24.

That only works if branch protection is set up for it. A repository can receive an agent MERGE verdict only when its
target branch meets P1–P5. P6 is needed only for automatic mode. The agent reads the rules on every evaluation;
anything unmet is ESCALATE with `protection_insufficient`.

| | Prerequisite | Why | fullsend `main`, 2026-09-27 |
|---|---|---|---|
| P1 | the policy's required checks are required status checks, strict | E1 belongs to branch protection | met (`behavior`, `e2e`, strict) |
| P2 | the scope verdict is itself a required check | ties the verdict to one commit (7.3) | not met |
| P3 | `require_code_owner_review`, and CODEOWNERS covers every sensitive and restricted path | E6 at T3 and T4 belongs to branch protection | review required; coverage not checked |
| P4 | `required_review_thread_resolution`: unresolved threads block merge | open findings block | met |
| P5 | an approval doesn't survive a new push (`dismiss_stale_reviews_on_push` or `require_last_push_approval`) | an approval shouldn't carry over to a commit the reviewer never saw | not met (whether `require_extra_approval_for_unattributed_changes` covers it is unverified) |
| P6 | automatic mode only: `required_approving_review_count` set to 0 | with 1 or more, every PR needs a person | not met (1 approval), so explicit mode only |

Source: `GET /repos/fullsend-ai/fullsend/rules/branches/main`, read 2026-09-24 and again 2026-09-27 (unchanged). Classic branch protection, if any,
needs admin access to read and isn't included.

### 5.1 Evidence classes

| | Evidence | What it proves | Produced by |
|---|---|---|---|
| E1 | Required checks | every required check passed on this exact commit | branch protection |
| E2 | Relevant tests | tests that exercise the affected behavior passed on this commit; from T2, the changed lines actually ran | CI, with the AISDLC-96 impact set and AISDLC-100 selection |
| E3 | Review | a structured review of this commit with no open critical, major or human-required finding, written by the review stage into storage the PR can't edit (not parsed from a comment) | review agent |
| E4 | Classification record | the tier and every input behind it | classification script |
| E5 | Intent and scope | a linked issue, and the diff does what it asks and nothing more | review agent |
| E6 | Human approval | a person with the right authority approved (CODEOWNERS for owned paths); bot and app approvals don't count | branch protection |
| E7 | Reversibility | the change can be switched off or reverted without data loss, or has a written rollback note | classification script; author |
| E8 | Cohort track record | this repository and tier have a post-merge outcome baseline, with reverts and fix-forwards under the policy threshold | outcome measurement ([fullsend#6892]) |
| E9 | Model veto | a calibrated model answered a fixed set of concern questions and none crossed its threshold (5.5) | a decision model chosen in AISDLC-99 |

Each item is bound to the commit it was produced for and, where it depends on them, to the base commit and the
policy version.

### 5.2 Minimum evidence per tier

R = required, O = recorded when present but never blocking, – = not needed. Rows marked (SCM) are enforced by branch
protection. A tier's MVE is met when every R cell is `present` for the current commit, base and policy.

| Evidence | T0 | T1 | T2 | T3 | T4 |
|---|---|---|---|---|---|
| E1 required checks (SCM) | R | R | R | R | R |
| E2 relevant tests | docs: –; dependency and test-only: existing suite green | tests that reference the changed code passed | the changed lines ran in a passing test, or a debt record (5.3) | as T2, plus integration or end-to-end evidence, and the consumers' tests when other repositories use it | as T3 |
| E3 review (SCM) | R (docs findings can be fixed) | R | R | R | advisory; a person decides |
| E4 classification | R | R | R | R | R |
| E5 intent and scope | – | linked issue | criteria covered | R | R |
| E6 human approval (SCM) | – | – after graduation (10); R before | one member | the code owner | the code owner; nothing automated |
| E7 reversibility | – | only adds code, or a default-off flag | not irreversible | rollback note if irreversible | R |
| E8 track record | R for automatic mode | R for automatic mode | – | – | – |
| E9 model veto | R for automatic mode | R for automatic mode | O | O | – |

Whatever the tier, a PR written by an agent needs a human approving review of that commit before it merges (10).

### 5.3 Substitutes

A substitute is allowed only when the policy file declares it. Every use is recorded by name, so it can be counted.

| Required evidence | Substitute | Only when |
|---|---|---|
| E2 at T2 (changed lines ran) | agent-written tests validated through AISDLC-98's golden path | they ran in the repository's real CI on this commit, the suite got strictly stronger, and the review covers them |
| E2 at T2 | an explicit verification-debt record | the package has no measurable coverage; the record names the gap and the tier goes up by one |
| E2 at T1 | a passing run of AISDLC-100's selected subset | the selection's confidence is high; otherwise the broader suite runs |
| E1 on the exact commit | a merge-queue run on the merge result | the queue re-runs every required check and the record keeps both commits |
| E6 at T2 | a second, independent model's review | branch protection counts app approvals, the cohort policy allows it explicitly, and both reviews agree (a disagreement is `contradictory`) |
| E3 freshness | a cached review | commit, base and policy unchanged, and under 7 days old |
| E8 | none | a cohort without an outcome baseline can't use automatic mode |
| E9 in automatic mode | an LLM judge answering the same questions with a validated probability each | the cohort policy names it and its calibration was measured in observe mode; a free-text verdict doesn't count |
| E2's consumer tests at T3 | approval from the consumer's code owner | the consumers' tests can't be run against the change |

### 5.4 Per-tier disqualifiers

These void a tier's evidence and escalate:

| Tier | Escalate when |
|---|---|
| T0 | a docs change also carries a generated or binary file; a dependency bump touches anything but the manifest and lockfile, or is a minor or major bump disguised as a pin, or fails the dependency rules (Appendix A); a test-only change weakens any existing assertion |
| T1 | the linked issue is labeled `needs-design`, `security` or `breaking-change`; the flag defaults on; the change reaches a sensitive, CI or restricted path through a shared template or base overlay |
| T2 | the review has a human-required finding; impact confidence is unknown; changed lines never ran and no substitute is declared |
| T3 | the code owner hasn't approved; an irreversible change has no rollback note; the integration evidence is unreliable |
| T4 | always: T4 only ever escalates, and the agent never changes the guardrails it runs under |

### 5.5 E9, the model veto

E9 is the one place a model's judgment enters the decision, and it can only say no. The model reads a bounded,
sanitized context (changed paths, the diff within the policy's size limits, the linked issue and the PR description;
never the raw PR) and returns a calibrated probability for each yes/no question. Every question is phrased so that
"yes" is a concern.

| Question | "Yes" means | Tiers |
|---|---|---|
| `exceeds_issue_scope` | the diff changes behavior the linked issue didn't ask for | T1 and above |
| `weakens_tests` | the change loosens, removes, skips or mocks away an existing test or assertion | all |
| `removes_safeguard` | the change removes or disables a safety, sync, retry or permission control, or changes permission data | all |
| `description_mismatch` | the PR description claims something the diff doesn't do, or leaves out something it does | all |
| `dependency_bump_mismatch` | the dependency change does more than the bump it claims | T0 dependency lane |

- **Veto.** A probability at or above the threshold escalates, naming the question, or remediates when the gap is
  fixable (such as a description the agent can correct). The threshold starts at 0.5 with a 0.1 band below it; a
  value in the band is `partial`, which escalates in automatic mode and is only recorded at T2 and above. Both
  numbers get fitted per question and tier in observe mode against real outcomes; the band exists because repeat
  runs of the same input drift by a few hundredths.
- **Failures.** An error or timeout is `missing` (re-run once, then escalate in automatic mode). A context over the
  model's input limit, or unreadable output, is `unknown`. A new commit, base or policy makes it `stale`. At T2 and
  above, a missing or unknown E9 is recorded and doesn't block, because a person reviews anyway.
- **Not a pass.** A clean E9 changes nothing; the rest of the evidence still decides. The questions and thresholds
  live in the policy file, so changing a question resets its results and its calibration.
- **Producer.** Which model fills this slot is AISDLC-99's decision. It has to be an AI tool and model approved for
  this use, fed sanitized context (11.7).

## 6. Missing, stale, flaky and contradictory evidence

Every required item is in exactly one state, and the state decides what happens next.

| State | Meaning | What happens |
|---|---|---|
| present | produced for this commit, base and policy, within its time limit, and meets the tier's threshold | counts |
| partial | produced, but below the threshold (for example, 40% of changed lines ran at T2) | REMEDIATE if the agent can close the gap (write tests, run the broader suite); otherwise ESCALATE with the measured value |
| missing | never produced for this commit | REMEDIATE by running the producer; ESCALATE when nothing can produce it (the outcome baseline) |
| stale | produced for another commit, base or policy, or too old | never counts. The agent re-runs the producer and WAITs; this doesn't use a remediation attempt. A base update makes the review, classification, intent and model evidence stale |
| unreliable | a test counted for E2 passed only on retry, or flakes above the policy rate | at T0–T1, accepted with a mark only if the test is quarantined (AISDLC-98) and not affected by this change; otherwise one deterministic re-run: a pass is accepted with a mark and a debt record, a second failure is a failing test (REMEDIATE if fixable, else ESCALATE) |
| contradictory | two producers disagree: CI green while the changed lines never ran; a `risk/low` label on a sensitive path; two reviews with different verdicts | the more conservative reading wins and the PR escalates, showing both readings. Human signals outrank bot signals |
| unknown or degraded | the producer ran but couldn't measure (an API or model outage, a degraded score, unknown impact) | never passes. A degraded advisory score is recorded and ignored; a degraded required item counts as missing; unknown impact raises the tier |

**Order when signals conflict:** branch protection's gates come first and the agent doesn't re-check them. Within
the agent's decision: classification and disqualifiers, then measured test evidence, then the advisory score, then
the model veto.

**Limits.** At most two REMEDIATE attempts per commit and four per PR; the next unmet gate escalates with
`remediation_budget_exhausted`. WAIT covers pending CI (`ci_pending`), unknown mergeability (`mergeability_unknown`),
conflicts (`merge_conflict`), an existing merge-queue entry (`already_queued`), an unmerged dependency
(`dependency_unmerged`), a stacked PR (`stacked_on_open_pr`) and refreshing evidence (`evidence_refreshing`). It
ends after 24 hours: pending CI, unknown mergeability and conflicts become REMEDIATE (re-run or rebase) while
attempts remain, and everything else becomes ESCALATE.

## 7. Outcomes

### 7.1 The decision function

```
if the repository's prerequisites are unmet    → ESCALATE (protection_insufficient)
classify the head commit                       → tier, disqualifiers, inputs
if any disqualifier                            → ESCALATE (reason codes)
if the tier is T4                              → ESCALATE (people only, no remediation)
grade the tier's evidence, except the rows branch protection enforces
if anything is contradictory                   → ESCALATE (both readings shown)
if a model-veto question crosses its threshold → ESCALATE (question named), or REMEDIATE if fixable
if a declared dependency is unmerged           → WAIT (bounded), then re-evaluate
if anything is missing, partial or unreliable (and not accepted with a mark):
    fixable and attempts left                  → REMEDIATE (the named gap)
    otherwise                                  → ESCALATE (the blocking list)
if anything is stale                           → re-run it, WAIT (bounded), then re-evaluate
if every required item is present:
    T0, T1 → MERGE: publish the verdict on this commit and enable merge-when-ready; GitHub merges
             (automatic only after graduation and P6, and never for an agent-authored PR; otherwise explicit)
    T2, T3 → ESCALATE: outside the auto-merge scope; a person approves and merges
otherwise                                      → ESCALATE (unhandled_state), so nothing ends without an outcome
```

### 7.2 Who acts at each tier

| Tier | Outcome | Who acts | What the agent may do |
|---|---|---|---|
| T0 | MERGE | nobody in automatic mode; in explicit mode, the approver branch protection requires | enable merge-when-ready on the commit its verdict covers; GitHub merges through the configured path (the merge queue on fullsend) |
| T1 | MERGE | as T0 | as T0; automatic mode only once the T1 cohort has an outcome baseline and the repository meets P6 |
| T2 | ESCALATE | a member reviews, approves and merges | fix findings and get CI green first; the verdict check reports `neutral` so it doesn't block the person |
| T3 | ESCALATE to the code owner | the code owner approves (P3 makes GitHub require it) and merges | as T2, but no changes to sensitive, CI or interface files without the owner |
| T4 | ESCALATE, people only | the code owner reviews and merges | report findings and stop |

### 7.3 What each outcome does

- **MERGE:** the agent turns on merge-when-ready for the exact commit its verdict covers. GitHub merges once its own
  gates pass. The agent never clicks merge, never re-checks those gates, and nothing bypasses branch protection or a
  required queue.
- **REMEDIATE:** names one gap, sends it to the producer that can close it (re-run the review, run the broader
  suite, write tests through AISDLC-98's golden path, rebase), and re-evaluates the new commit.
- **ESCALATE:** posts the blocking list and reason codes on the PR, assigns the person the policy names for that
  gate, and stops. A later human action starts a fresh evaluation; the old result never comes back as authority.
- **Binding the verdict to one commit.** GitHub turns auto-merge off only when someone *without* write access pushes
  ([GitHub's auto-merge docs]). A push by a maintainer, or by an agent with write access, leaves it on, so a verdict
  for one commit could merge the next. The fix is P2: the verdict is a required check run on the head commit,
  `success` when in scope and `neutral` with the reasons when not (GitHub counts `neutral` as passing, so people can
  still merge out-of-scope PRs). A new commit has no verdict until the agent re-evaluates. Before posting `neutral`,
  the agent turns auto-merge off, so a crash between the two steps leaves the PR blocked, not merging.

## 8. Retention and audit

### 8.1 The mergeability record

Each decision writes one append-only record, bound to the forge, repository, PR number, head commit, base branch,
base commit and policy hash. It holds:

- the tier, its inputs, and anything that raised it;
- each evidence item's state, value, source and time;
- the substitutes used;
- the outcome, reason codes, blocking list and remediation attempt;
- the producer runs and any human action.

It holds no credentials and no PR content beyond file paths and counts. E1, E3 and E6 appear as the branch-protection
state observed at decision time; they are audit data, not inputs (5.0). An example is in Appendix C.

### 8.2 Storage

- Records live in host-authenticated storage that PR authors and the merge identity can't write. PR comments, labels
  and check summaries mirror the record for people; they are never an input to a decision.
- A MERGE record is written *before* the agent enables merge-when-ready. A second record follows once GitHub merges
  or the verdict is withdrawn, so a crash between the two shows up instead of disappearing.
- A re-evaluation writes a new record; it never rewrites an old one.

### 8.3 Retention

- Records and their evidence pointers: at least 12 months, and never less than the support life of the release the
  merged commit ships in.
- Policy versions: indefinitely, in git, so any record's policy hash resolves to the exact rules that applied.
- Raw artifacts (review transcripts, coverage profiles, CI logs): each tool's own retention, but the record keeps the
  run id and a content hash, so a missing artifact is detectable.

### 8.4 What the audit must answer

1. Why did this PR merge, under which policy version, on which evidence, and who or what produced each item?
2. Which merges in the last N days were automatic, by tier and cohort, and which used a substitute?
3. Which automatic merges were reverted or fixed forward within the look-back window (initially 90 days), by tier?
4. Which decisions used unreliable evidence, and did those merges revert more often?
5. How often did the advisory score disagree with the tier, and in which direction?
6. Which escalations did a person override, and what happened to those merges?
7. For each model-veto question, how did its probabilities look on merges later reverted or fixed forward, compared
   with clean merges, and does the threshold still hold?
8. Which merges had consumers outside the repository, and did the producer merge first?

### 8.5 Calibration and revocation

- **Calibration:** a monthly review of questions 3–7, per repository and tier. Tier sizes, the dependent threshold,
  the raise rule and its windows, the T2 coverage threshold and the model-veto thresholds change only through a
  policy change, which is itself T4. They are set against the benchmark's post-merge outcomes (reverts, fix-forwards,
  customer cases) across several repositories. The 246-PR sample in section 9 was collected to evaluate the ADR 0089
  scorer, and its defects are read from review findings, so it tests the pipeline, not the thresholds. Model-veto thresholds are
  first fitted in observe mode, before automatic mode relies on them.
- **Automatic revocation:** a cohort drops back from automatic to observe mode when two or more automatic merges are
  reverted or fixed forward within 30 days, or when its outcome baseline falls below threshold. Turning it back on
  is a human policy change.
- **Break glass:** a person can merge over an escalation at any time through GitHub. The record marks the override
  and overrides are reported (Cloudflare reports 0.6% of merge requests use theirs). An override never makes the
  policy more lenient.

## 9. Evidence base

The related work was read in full and judged on its numbers, not its reputation. The details behind this section are
available on request.

**ADR 0089's risk score** ([ADR 0089], implementation [agents#861]) works as a signal but not as a gate. The sample
is every PR in `fullsend-ai/fullsend` and `redhat-appstudio/infra-deployments` that carried both a risk comment and a
review since the scorer shipped on 2026-08-25: 246 PRs.

| Finding | Numbers |
|---|---|
| Scores are lopsided; the top of the scale never occurs | 53 / 177 / 15 / 1 / 0 across scores 1–5 |
| Correlation with how much review found is real but weak | Spearman rho 0.387, 95% CI 0.253–0.506 |
| By the review's own labels, score 1 looks clean | 0 of 53 at score 1 vs 31 of 193 at 2 or more had a major or critical finding |
| Read as actual defects, the difference disappears | of the 31: 21 protected-path rule (17 Renovate), 5 process or docs, 4 behavior defects, 1 arguable ([fullsend#6994]); defects by score 0/53, 2/177, 2/15, 0/1; score 1 vs 2 indistinguishable (Fisher p ≥ 0.59) |
| A bash-only tier-1 gate isn't the same as the composite | tier 1 rounds to 1 on 104 PRs, 6 of them with a major or critical label |
| The score is silently missing on many PRs | about 15% of recent coder PRs have none ([fullsend#7387]); model outages and token limits removed it from about 10 more in the sample |
| The same commit can score differently on re-review | tier 1 was re-emitted by the model; fixed by computing it in the script ([agents#1245], open) |
| Severity labels depend on the model | on 4 score-1 PRs, a sonnet-only run found the same defects as opus but rated two of them HIGH where opus said low or medium |

So this design uses the score only to raise a tier, takes protected paths and dependency bumps out of it, requires a
classification that a script computes and that can't silently not run, and calibrates on post-merge outcomes. The
2026-09-10 comment on [fullsend#4698] concluded that "score 1 alone is a safe gate"; the defect reading above doesn't
support that, so this proposal doesn't use it; the comment was corrected on 2026-09-28. The routing draft [agents#1246]
was closed on 2026-09-28 for the same reason.

**The tier rules, applied to the same 246 PRs** (2026-09-23), with reversibility, test delta and impact left unknown:

| Result | fullsend (209) | infra-deployments (37) |
|---|---|---|
| T0 / T1 / T2 / T3 / T4 | 30 / 20 / 56 / 70 / 33 | 0 / 0 / 10 / 25 / 2 |
| PRs the advisory score raised | 0 | 0 |
| T0 or T1 shape, no disqualifier | 48: 24 docs-only, 4 dependency bumps, 20 small changes with a linked issue | 0 |

- **The 48 are an upper bound.** Only the 24 docs-only PRs are eligible today. The 4 dependency bumps also need the
  dependency checks, and the 20 T1 changes need a reversibility producer.
- **By authorship** (read 2026-09-27), 35 of the 48 were agent-authored: `fullsend-ai-coder` opened all 20 T1
  changes and 15 of the 24 docs PRs. Those stay in explicit mode (section 10). That leaves 13 for the automatic lane:
  9 docs PRs by maintainers, all 9 marked AI-assisted, and the 4 Renovate bumps.
- **Where the defects landed:** four of the five labeled defects land in T3. The arguable one ([fullsend#6994])
  lands in T1 and is held by its changes-requested review. The 21 protected-path PRs land in T4 (12) and T3 (9).
- **The tiers aren't the old score:** only 16 of the 53 score-1 PRs are candidates, and 32 candidates carry score 2.
- **Fixes the check forced:** it exposed five weak rules, all fixed above:
  - a blanket refusal of forks (30 of 37 infra PRs come from members' forks);
  - a single intent source (infra uses Jira keys, not closing keywords);
  - one path list for every repository (a GitOps repository's manifests are its product);
  - raw size as blast radius;
  - a CI class so coarse it put 48 PRs in T3, only 4 of them pure action bumps.

**Other sources, and what was taken from each:**

- **[Cloudflare's AI code review]** (131,246 review runs on 48,095 merge requests in 30 days).
  - Taken: size tiers plus security paths with a max rule; a strict, published verdict rubric; "what not to flag"
    lists; lockfile and generated-file filtering; tracked break-glass.
  - Not taken: their tiers size the review effort, not the merge evidence.
- **ADR 0110 and the auto-merge contract** ([fullsend#7151], [agents#1132], [agents#1219]).
  - The authority boundary: the agent recommends, a trusted runtime merges, and no merge credential sits in the
    sandbox.
  - The rule that a missing or stale assessment stops automatic merging.
  - The invariants, reason codes and receipts reused in sections 6–8, from the v1 contract removed from the PR on
    2026-09-22 (readable at commit `f9c12e46`).
  - ifireball's review, which set the split in 5.0.
- **FullSend's problem documents** (autonomy spectrum, trustworthiness evidence, review autonomy, graduated approval,
  intent, repo readiness, governance).
  - Graduation criteria are marked "all TBD" there; E8 and section 10 propose them.
  - Status kept separate from score.
  - "Any high dimension escalates."
  - Test changes are pre-authorized only when strictly stronger.
  - Policy changes need a higher bar than code.
- **[fullsend#3016]** shows the practical blocker: Renovate PRs wait 8–36 hours for an approval even when labeled
  low risk. That is why T0 is the first cohort. [fullsend#6892] asks for the revert view E8 needs.
- **Model judgment:** E9 follows the shape of typed decision models: a calibrated probability per atomic question,
  composed in code, with thresholds checked against this domain's own outcomes, because typed output guarantees the
  format, not the truth. Choosing a producer belongs to AISDLC-99.

## 10. Rollout

1. **Observe.** Land the policy schema and the classification script. Classify every PR and publish the tier and
   blocking list as a check summary. Compare against what people decided for at least 30 days and 50 decisions per
   cohort.
2. **Explicit, T0, one repository.** A trusted person triggers each evaluation, and every gate applies. On fullsend
   `main`, that trigger is the approval branch protection already requires. P2 and P5 come first (5.0).
3. **Automatic, T0.** Only once the observe period shows no false MERGE verdicts (a MERGE a person then had to
   block), the outcome baseline (E8) exists, and the repository meets P6.
   - Automatic mode never covers a PR whose AI authorship is `agent` (4.1). Red Hat's guidelines for AI code
     assistants require a person to review and test AI-generated code before it is integrated.
   - A human author, AI-assisted or not, has done that, and a bot change with no AI in it, such as a Renovate bump,
     is outside the guidelines.
   - An agent-authored PR has no person in the loop, so it stays in explicit mode, where the trigger has to be a
     human approving review of that commit (11.11).
4. **T1:** repeat steps 2 and 3. T2 and above stay at "remediate, then a person approves" until the outcome data
   says otherwise.
5. **Expand** only after a dated review of post-merge outcomes and an explicit policy change.

## 11. Open decisions

These are follow-ups for their owners, not blockers for the contract.

1. **Thresholds.** Tier sizes (4.2), the T2 coverage threshold (initially 80% of changed lines), time limits and the
   remediation budget need agreement from the repositories that go first. Proposed owners: Ella Shulman and Hofni
   Gartner.
2. **The policy file.** ifireball's review of ADR 0110 settles where it lives: in the repository, where the agent
   reads it. Still open: its name and format, and how ADR 0069 presets are inherited. Owner: ifireball.
3. **The ADR 0089 score.** Does it stay as a raise-only signal once the classification exists, or fold into
   AISDLC-96's impact model? Owner: Marta (maruiz93).
4. **The outcome baseline for E8.** What counts as a fix-forward, and the look-back window. Owner: ascerra
   ([fullsend#6892]). Hofni's AISDLC-99 benchmark (PRs later followed by a bug-fix commit or a customer case) is the
   likely source and should use the same definition.
5. **A second model's review in place of the T2 approval.** Possible only if branch protection counts app approvals,
   so it's a protection decision before it's a policy one. The team may prefer to forbid it until independent review
   runtimes exist.
6. **Coverage producers for E2.** Which tool measures changed-line execution in each repository is an AISDLC-99
   question; this contract fixes only the evidence shape.
7. **The E9 producer.** AISDLC-99 evaluates candidates against 5.5, including whether each one's calibration holds on
   real outcomes. Red Hat's AI code-assistant guidelines apply: only a tool and model approved for the use,
   fed sanitized context with no credentials or customer data.
8. **Consumer maps.** Go modules, Argo and Konflux relationships and interface registries each need a producer.
   AISDLC-99 lists candidates; AISDLC-96 owns the impact model that uses them.
9. **Downstream test capacity.** Running consumers' tests against a producer change is a CI-capacity question for
   AISDLC-100.
10. **Branch protection changes** for repositories that adopt this (5.0): the verdict as a required check (P2), no
    approval carried across pushes (P5), and for automatic mode no blanket approval count (P6). Owners: each
    repository's admins; for fullsend, ifireball and ascerra, since it touches ADR 0110's enforcement.
11. **Red Hat's AI policy and automatic mode.** Section 10 reads Red Hat's guidelines for AI code assistants this way:
    - an agent-authored PR needs a human review before it merges;
    - a human author, AI-assisted or not, is already the person in the loop;
    - a bot change with no AI in it is outside the guidelines.

    The policy owner should confirm that reading, including whether dependency bumps are covered and whether code
    Red Hat ships differs from services it runs. Upstream projects' own AI-contribution policies also apply.
    The mergeability record and the E8 outcome data are the evidence for revisiting the rule. Owner: the policy
    owner; for fullsend, ascerra and ifireball.

---

## Appendix A. Dependency rules in full

**Inbound: packages this repository uses.** The T0 dependency lane merges unattended only when all of these hold:

| Condition | Why |
|---|---|
| a bump only: no package added, removed or renamed | a new package is the supply-chain and typosquatting case, so T3 |
| patch or pin/digest movement, read from the manifest diff, not the PR title | titles are author-controlled |
| the lockfile matches the manifest, integrity hashes are kept, the registry and source URLs are unchanged | a swapped source is the attack; the version is the disguise |
| no new known vulnerability, from the repository's scanner (Dependabot alerts, Trivy, Snyk) as a required check | a bump that fixes a CVE stays T0 |
| the license is unchanged or on the allowlist | irreversible once shipped |
| CI actions: pinned by digest, digest bump only, same action name | 4 of the 27 Renovate PRs in the sample were exactly this |

**Outbound: code elsewhere that uses this change.** A change to an exported function, a published module, a CRD,
protobuf or REST definition, a container image, a shared template or a base overlay can break code that lives
elsewhere.

- Each repository profile names where its consumers are recorded. A published interface with no consumer map counts
  as unknown impact, which raises the tier.
- Any change with a consumer outside the repository or component is T3. ADR 0089's change-coupling signal (files
  that usually change together but are missing from the PR) is the in-repository version and can only raise.
- At T3, E2 includes the consumers' own tests run against this change (AISDLC-100 picks the subset). If they can't
  be run, the consumer's code owner can approve instead. Missing downstream evidence escalates, naming the consumers.
  It never remediates, because the agent can't produce that evidence alone.
- The producer merges before its consumers, and the record carries the consumers and any cross-repository dependency
  so the audit can check the order. The infra-deployments defect in the sample (#13738, a base ApplicationSet losing
  auto-sync) is exactly this kind of change: a base reaches every overlay.

**Between PRs: the merge order.**

- The base must be the default branch or an allowlisted release branch. A stacked PR, whose base is another open
  PR's branch, WAITs until the parent merges and it is retargeted; its evidence goes stale and is re-evaluated. Any
  other base escalates (`unsupported_base`).
- `Depends on` references, in the same repository or another, are read deterministically. An unmerged dependency
  WAITs, a merged one clears, and one closed without merging escalates (`dependency_closed`).
- WAIT ends in ESCALATE after the policy timeout (section 6).
- Two open PRs touching the same files need no rule: merging one makes the other's evidence stale, and the merge
  queue serializes them.

## Appendix B. Repository profiles (from the 2026-09-23 check)

**fullsend-ai/fullsend** (Go CLI and docs site)

| Class | Paths |
|---|---|
| `restricted` | CODEOWNERS, AGENTS.md, CLAUDE.md, .gitmodules, agents/, .agents/, skills/, harness/, plugins/, policies/, profiles/, providers/, .claude/, .cursor/, .pi/, images/ |
| `sensitive` | any path segment named mint, auth, oidc, rbac, permissions, secrets, crypto, token or tokens, trust, credential |
| `ci` | .github/, hack/, scripts/, Makefile, Dockerfile, Containerfile, .pre-commit-config.yaml, renovate.json |
| `dependency` | go.mod, go.sum, package.json, package-lock.json |
| `test` | `*_test.go`, `*_test.py`, `*-test.sh`, `*.spec.*`, `*.test.*`, `*.feature`, e2e/, testdata/, fixtures/ |
| `docs` | docs/ and root `*.md`; images under docs/ count as docs |

- Noise: lockfiles, `*.min.*`, `*.map`.
- Intent source: GitHub closing keywords.
- Allowlisted bots: renovate-fullsend, fullsend-ai-coder.

**redhat-appstudio/infra-deployments** (GitOps)

| Class | Paths |
|---|---|
| `restricted` | OWNERS, AGENTS.md, .github/CODEOWNERS, skills/ |
| `ci` | .github/, .tekton/, hack/ |
| `sensitive` | any base or production overlay, argo-cd-apps/base, configs/, and any manifest whose path names secrets, RBAC, roles, service accounts or auth |
| `source` | all other manifests, because they are the product |

- Intent source: a Jira key in the PR title.
- Members' forks count as members.

## Appendix C. Example mergeability record

```json
{
  "schema_version": "0.1",
  "subject": {
    "repository_id": "R_kgDO...", "pr": 7385,
    "head_sha": "…", "base_ref": "main", "base_sha": "…",
    "policy_hash": "sha256:…", "classifier_version": "…"
  },
  "classification": {
    "tier": "T2",
    "inputs": { "files": 6, "lines": 212, "path_classes": ["source", "test"],
                "author_class": "allowlisted-bot", "ai_authorship": "agent",
                "reversibility": "additive-only", "test_delta": "strictly-stronger",
                "impact_confidence": "medium", "advisory_composite": 2 },
    "raised_by": [],
    "disqualifiers": []
  },
  "dependencies": { "declared": [], "consumers": [], "consumer_map": "go.mod-org-scan@…" },
  "evidence": {
    "E1": { "status": "present", "source": "github:checks", "as_of": "…" },
    "E2": { "status": "partial", "value": 0.4, "threshold": 0.8,
            "source": "coverage-profile@run-123", "as_of": "…" },
    "E3": { "status": "present", "value": { "critical": 0, "major": 0, "human_required": 0 },
            "source": "review@run-456", "as_of": "…" },
    "E4": { "status": "present", "source": "classify@v0.1", "as_of": "…" },
    "E5": { "status": "present", "source": "review@run-456", "issue": 7380 },
    "E7": { "status": "present", "value": "additive-only" },
    "E9": { "status": "present", "source": "e9-producer@questions-v1", "as_of": "…",
            "value": { "exceeds_issue_scope": 0.07, "weakens_tests": 0.02,
                       "removes_safeguard": 0.03, "description_mismatch": 0.11 },
            "threshold": 0.5, "band": 0.1 }
  },
  "substitutes_used": [],
  "decision": "REMEDIATE",
  "reason_codes": ["coverage_partial"],
  "blocking": ["E2"],
  "remediation_cycle": 1,
  "actors": { "producer_runs": ["review@run-456"], "human": null },
  "decided_at": "…"
}
```

## Appendix D. Sources

- Jira: [AISDLC-97], [AISDLC-29], and the sibling epics AISDLC-96, 98, 99 and 100
- [ADR 0089] and the [pr-risk-assessment skill]
- [fullsend#4698], including the 2026-09-10 measurement comment
- [agents#861], [agents#1245], [agents#1246]
- [Cloudflare's AI code review]
- [fullsend#7151]: ADR 0110, its review threads, and the removed `docs/normative/auto-merge/v1/README.md` at commit
  `f9c12e46`
- [agents#1132], [agents#1219]
- [fullsend#3016], [fullsend#6892], [fullsend#7387], [fullsend#6994]
- `docs/problems/` in fullsend-ai/fullsend: autonomy-spectrum.md, trustworthiness-evidence.md,
  review-autonomy-evidence.md, graduated-approval-policy.md, intent-representation.md, repo-readiness.md,
  governance.md, security-threat-model.md
- [GitHub's auto-merge docs]
- The measurement run data and the tier-check script, inputs and output, available on request

## Changelog

- **v0.5** (2026-09-27): restructured for reading: summary first, plainer wording, reference material moved to
  appendices; internal-only links removed. No rule changed.
- **v0.4** (2026-09-27): applies Red Hat's guidelines for AI code assistants:
  - an AI-authorship input;
  - agent-authored PRs are kept out of automatic mode;
  - the E9 producer must be approved and fed sanitized input;
  - the authorship of the 48 candidates.
- **v0.3** (2026-09-24): follows ifireball's review:
  - branch protection enforces, the agent decides scope;
  - the repository prerequisites;
  - the verdict is bound to one commit;
  - four gaps in the decision logic are closed.
- **v0.2** (2026-09-23): adds the model veto (E9), the dependency rules, six input corrections from the sample check,
  and the repository profiles.

[AISDLC-97]: https://redhat.atlassian.net/browse/AISDLC-97
[AISDLC-29]: https://redhat.atlassian.net/browse/AISDLC-29
[fullsend#7151]: https://github.com/fullsend-ai/fullsend/pull/7151
[fullsend#6994]: https://github.com/fullsend-ai/fullsend/pull/6994
[fullsend#6892]: https://github.com/fullsend-ai/fullsend/issues/6892
[fullsend#3016]: https://github.com/fullsend-ai/fullsend/issues/3016
[fullsend#7387]: https://github.com/fullsend-ai/fullsend/issues/7387
[fullsend#4698]: https://github.com/fullsend-ai/fullsend/issues/4698
[ADR 0089]: https://github.com/fullsend-ai/fullsend/blob/main/docs/ADRs/0089-pr-risk-assessment-scoring.md
[pr-risk-assessment skill]: https://github.com/fullsend-ai/agents/blob/main/skills/pr-risk-assessment/SKILL.md
[agents#861]: https://github.com/fullsend-ai/agents/pull/861
[agents#1245]: https://github.com/fullsend-ai/agents/pull/1245
[agents#1246]: https://github.com/fullsend-ai/agents/pull/1246
[agents#1132]: https://github.com/fullsend-ai/agents/issues/1132
[agents#1219]: https://github.com/fullsend-ai/agents/pull/1219
[Cloudflare's AI code review]: https://blog.cloudflare.com/ai-code-review/
[GitHub's auto-merge docs]: https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/incorporating-changes-from-a-pull-request/automatically-merging-a-pull-request
