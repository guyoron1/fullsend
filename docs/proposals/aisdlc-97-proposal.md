# AISDLC-97: Risk-Tiered Evidence and Merge Contract

Guy Oron · 2026-09-27 · Draft for review · Jira: AISDLC-97, part of AISDLC-29

## What this decides

AISDLC-29 asks for "the specific, dynamic testing thresholds an agent must meet to qualify for an auto-merge, based
on the scope of the change". This is that contract. For every pull request it sets:

- a risk tier, from the scope of the change;
- the evidence that tier requires;
- one outcome: **merge**, **fix first** (close a named gap and check again) or **hand to a person**.

It has to avoid both failures AISDLC-29 names:

- **Merging a regression.** The tiers make riskier changes prove more, and missing evidence never merges.
- **Blocking agents forever on flaky or missing tests.** A gap is routed into AISDLC-98's golden path: fix it now,
  defer it as debt, or escalate. It is not an open-ended block.

Five principles:

- **A script sets the tier, not a model.** It uses paths, size, author and linked issue, and the riskiest file
  decides.
- **Missing or unclear evidence never leads to a merge.**
- **GitHub keeps enforcing checks, reviews and approvals.** The agent only decides whether a PR is in scope for
  auto-merge. If it is, the agent turns on GitHub's merge-when-ready and GitHub merges.
- **A PR written by an agent always needs a human approval,** as Red Hat's guidelines for AI code assistants require.
- **Every decision is logged with its evidence,** and the thresholds are checked against what actually needed a fix
  later.

## 1. Where 97 sits in AISDLC-29

97 turns the other epics' outputs into a merge decision. It doesn't define how impact is computed, which tests run,
how debt is fixed or which tools are used. It defines how much evidence each scope of change needs, and what happens
when that evidence isn't there.

| Epic | Defines | 97 uses it for | 97 gives back |
|---|---|---|---|
| AISDLC-96: change impact and coverage adequacy | the blast radius and its confidence; what counts as adequate coverage of the affected behavior | setting the tier; the "tests cover the affected behavior" requirement | which tier needs which confidence and which level of adequacy |
| AISDLC-100: test selection and CI capacity | which tests run for a change, the fallback to a broader suite, CI handling of flaky, slow and unavailable tests, cached results | meeting each tier's test requirement at the lowest cost | the evidence bar each tier must reach |
| AISDLC-98: debt taxonomy and golden path | the debt categories, the golden path (fix now, defer, escalate), and how agent-written tests are validated | the "fix first" outcome, and deferring debt instead of blocking | when a merge decision sends a gap to the golden path, and a decision record it can audit |
| AISDLC-99: tooling | which tool produces each piece of evidence | everything in section 3 | the list of evidence the tools have to produce |

Where the epics overlap, this is the proposed split:

| Topic | Who owns what |
|---|---|
| Coverage adequacy | 96 defines what "adequate" means. 97 only sets which tier requires it. The 80% changed-line figure used below is a placeholder until 96 defines it. |
| Test requirements by risk | 97 sets the bar for each tier. 100 picks the tests, and the fallback, that meet it. |
| Flaky, slow and unavailable tests | 98 names them, 100 handles them in CI, and 97 decides what the merge does in the meantime. |
| Keeping merge evidence | 97 defines the record each decision writes. 98 uses it for audit, and adds the validation evidence for agent-written tests. |
| What an agent may fix on its own | 98. 97 only says when a fix is needed. |
| The benchmark for judging the gate | 97, as Ella noted on AISDLC-99: past PRs later followed by a customer case or a quick bug fix (Hofni's proposal). Section 6 uses it to set and check the thresholds. |

What 97 can settle now, and what it waits on:

| Can be settled now | Waits on |
|---|---|
| the tiers, each tier's evidence bar, the outcomes, the decision record, the rollout | 96 for blast radius and adequacy (until then, size stands in and T1+ can't merge automatically); 100 for cheaper evidence (until then, full suites run); 98 for the fix-first path and deferral (until then, gaps go to a person); 99 for the tools, including reversibility and weakened-test checks, which no tool does today |

## 2. Risk tiers, by scope of change

| Tier | What falls here | Outcome once its evidence is complete |
|---|---|---|
| T0 Pre-authorized | docs only; a patch or pin dependency bump that passes the supply-chain checks; new tests only | merge |
| T1 Low | up to 10 files and 100 lines; only adds code or sits behind a default-off flag; has a linked issue | merge |
| T2 Standard | up to 25 files and 800 lines, nothing sensitive | a team member approves |
| T3 Sensitive | auth or secrets paths, CI structure, migrations, minor or major dependency bumps, code other repositories use, weakened tests, anything larger | the code owner approves |
| T4 Restricted | CODEOWNERS, branch rules, the policy file itself, agent prompts, credentials, release configuration | people only; the agent doesn't touch it |

- Size stands in for blast radius until 96's impact model exists. Once it does, the tier is the higher of the two,
  and low or unknown impact confidence raises the tier by one.
- The size limits are starting values from Cloudflare's published review tiers and ADR 0089. They get checked
  against the benchmark (section 6).
- The existing ADR 0089 risk score can only raise a tier.
- **Always handed to a person, whatever the tier:** the PR edits the rules it's judged by; it's a draft; the code
  changed after review; the agent would merge its own work; a T1+ PR has no linked issue; tests were weakened.

## 3. Evidence each tier needs

Every item must be for the exact commit being merged. Rows marked GitHub are enforced by branch protection.

| Evidence | T0 | T1 | T2 | T3 | T4 |
|---|---|---|---|---|---|
| Required checks pass (GitHub) | yes | yes | yes | yes | yes |
| Tests cover the affected behavior (96's criteria, run through 100's selection) | existing suite (none for docs) | tests that touch the changed code | the changed lines actually ran (placeholder: 80%) | as T2, plus integration tests and tests of code that uses it | as T3 |
| Review with no blocking findings (GitHub) | yes | yes | yes | yes | advisory |
| Linked issue, and the change matches it | – | yes | yes | yes | yes |
| Human approval (GitHub) | once proven* | once proven* | a team member | the code owner | the code owner |
| Can be reverted or switched off | – | only adds code, or a flag | not irreversible | rollback note if not | yes |
| Track record: the tier's merges rarely needed a later fix | for automatic merges | for automatic merges | – | – | – |
| Model check: no concern flagged | for automatic merges | for automatic merges | advisory | advisory | – |

\* Until the tier has proven itself, a person approves. A PR written by an agent always needs a human approval.

The model check asks a fixed set of yes/no questions: does the change go beyond its issue, weaken tests, remove a
safeguard, or contradict its description? It returns a calibrated probability for each. It can only block, never
approve. Which model fills this role is a 99 question.

## 4. When the evidence has a problem

The problem names follow 98's taxonomy. 97 only decides what the merge does. 98 decides how the gap gets fixed or
deferred.

| Problem | What the merge decision does |
|---|---|
| missing or unavailable | fix first by running what produces it; if nothing can, defer as debt or hand to a person |
| irrelevant or partial (tests ran, but don't cover the changed behavior enough) | fix first (add tests) or defer as debt; never merge on it |
| stale (for an older commit, base or policy) | re-run and wait; stale evidence never counts |
| flaky or non-reproducible | one more run, and a second failure is a real failure. At T0–T1, only known, quarantined flaky tests that the change doesn't touch are accepted |
| slow | wait up to 24 hours, then fix first or hand to a person |
| contradictory (two sources disagree, e.g. CI green but the changed lines never ran) | hand to a person, showing both readings |
| unknown (the tool couldn't measure) | never passes |

The agent gets at most 2 fix attempts per commit and 4 per PR. After that, the PR goes to a person.

## 5. Outcomes

- **Merge:** the agent turns on merge-when-ready for that commit, and GitHub merges once its own checks pass. The
  verdict is published as a required check on the commit, so a new push needs a new verdict.
- **Fix first:** the gap goes into 98's golden path, either fixed now by the tool or agent that can close it or
  deferred as recorded debt. Then everything is checked again.
- **Hand to a person:** the agent posts what's missing and assigns the right person. People can always override, and
  overrides are logged.

Every decision writes an append-only record: the tier, each piece of evidence and where it came from, and the
outcome. Records are kept for at least 12 months. This is the evidence 98 audits and the benchmark is checked
against.

## 6. How the thresholds are set and checked

- **Starting values:** the tier sizes and the 80% placeholder above.
- **The benchmark** (Hofni's proposal, placed in 97 by Ella):
  - past PRs that were later followed by a customer case or a quick bug fix, labeled by how critical the problem
    was;
  - for each tier, it answers whether the gate would have merged a PR that later needed a fix;
  - a first pilot on the 246 recent fullsend and infra-deployments PRs already analyzed, which include 5 labeled
    defects, before building a larger set.
- **In production:** a monthly review of which automatic merges later needed a fix, per tier. A tier goes back to
  human approval after two such merges in 30 days.

## 7. Rollout and the work that has to come first

| Step | What happens | Needs first |
|---|---|---|
| 1. Observe | classify every PR and publish its tier and what's missing, for at least 30 days | a classification script (paths, size, author, agent-written or not, linked issue); buildable now |
| 2. Explicit | docs PRs and Renovate bumps in one repository; the approval fullsend already requires triggers each merge | two branch-protection changes on fullsend (the verdict as a required check, approvals reset on a new push); supply-chain checks for dependency bumps |
| 3. Automatic | only for PRs written by people or by bots like Renovate; no approval needed | no wrong merges in observe; the benchmark and an outcome baseline (fullsend#6892); a model for the checks, approved under Red Hat's AI policy (99); the repository stops requiring an approval on every PR |
| 4. T1 and beyond | T1 next; T2 and above stay human-approved until the data says otherwise | 96's impact model and adequacy criteria; reversibility and weakened-test checks (99) |

On the 246-PR sample, 48 fit T0 or T1. 35 of those were written by an agent, which leaves 13 for automatic
merging: 9 docs PRs by maintainers and 4 Renovate bumps.

## 8. Decisions needed

| Decision | Proposed owner |
|---|---|
| The split of overlapping topics in section 1 | Ella Shulman, with the owners of 96, 98 and 100 |
| Starting thresholds: tier sizes, the coverage placeholder, the 24-hour wait, the fix budget | Ella Shulman, Hofni Gartner |
| The benchmark: what counts as a later fix or customer case, how far back to look, how criticality is labeled | Hofni Gartner, Guy Oron, with ascerra (fullsend#6892) |
| The policy file: its format, kept in the repository | ifireball |
| Keep the ADR 0089 score as raise-only, or fold it into 96 | Marta (maruiz93) |
| The branch-protection changes on fullsend | ifireball, ascerra |
| Confirm the reading of Red Hat's AI guidelines: agent-written PRs are always human-approved; Renovate bumps are outside the guidelines | the policy owner |

The [full reference version](https://github.com/guyoron1/fullsend/blob/aisdlc-97-mve-contract/docs/proposals/aisdlc-97-mve-mergeability-contract.md), with the evidence behind each rule, the
dependency rules, the record format and the repository profiles, is kept separately for anyone implementing this.
