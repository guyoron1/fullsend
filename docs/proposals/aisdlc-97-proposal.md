# AISDLC-97: Risk-Tiered MVE and Mergeability Contract

## What this decides

AISDLC-29 asks for "the specific, dynamic testing thresholds an agent must meet to qualify for an auto-merge, based
on the scope of the change". This contract gives every pull request a **risk tier** from the scope of the change,
the minimal viable **evidence** (MVE) that tier needs, and one **outcome**: **merge**, **remediate** (fix a named
gap, then check again) or **escalate** (hand to a person).

- **A script sets the tier, not a model:** paths, size, blast radius, author, linked issue and the files' history; the
  riskiest signal decides, and weaker signals can only raise the tier.
- **Missing or unclear evidence never merges,** so regressions don't slip through. A gap goes into 98's golden path
  (fix now, defer as debt, or escalate), so agents aren't blocked forever on flaky or missing tests. These are the
  two failures AISDLC-29 names.
- **GitHub keeps enforcing checks, reviews and approvals.** The agent only decides whether a PR is in scope, then
  turns on merge-when-ready.
- **Agent-written PRs keep a human approval, for now.** Red Hat's guidelines for AI code assistants require a human in
  the loop who verifies AI-generated code; when no person wrote the PR, we read that as a person approving it. So for
  agent PRs, the contract decides what evidence must be in place before a person is asked, and which person; the
  merge then happens on its own. PRs written by people, and bot bumps with no AI in them, can merge with no approval
  if the policy owner agrees (section 8). The decision record is the evidence for moving to a person on the loop.
- **Every decision is logged with its evidence,** and the thresholds are checked against which merges later needed
  a fix.

## 1. How 97 fits with the other epics

The other epics produce the evidence; 97 sets how much of it each tier needs and what happens when it's missing.

- **96** gives the blast radius and what "adequate coverage" means. Until then, a count of dependent packages stands
  in, and T1 and above can't merge automatically.
- **100** picks the tests that meet each tier's bar and handles flaky, slow and unavailable tests in CI. Until then,
  full suites run.
- **98** gives the debt categories (section 4) and the fix-now / defer / escalate path. Until then, gaps go to a
  person.
- **99** supplies a tool for each piece of evidence. Where none exists yet (reversibility, weakened tests), the
  evidence is "unknown" and never passes.

The benchmark (section 6) and the decision record (section 5) are 97's; 98 audits the record.

## 2. Risk tiers

| Tier | Scope of change | Outcome with complete evidence |
|---|---|---|
| T0 Pre-authorized | docs only; a patch or pin dependency bump that passes supply-chain checks; new tests only | merge (an agent's PR after a human approves) |
| T1 Low | up to 10 files and 100 lines; new code only (new files or functions nothing calls yet) or behind a default-off flag; few dependents; linked issue | merge (an agent's PR after a human approves) |
| T2 Standard | up to 25 files and 800 lines; nothing sensitive; no new dependency | a team member approves |
| T3 Sensitive | auth or secrets, CI structure, migrations, new dependencies or minor or major bumps, core code, code other repositories use, weakened tests, anything larger | the code owner approves |
| T4 Restricted | CODEOWNERS, branch rules, the policy file, agent prompts, credentials, release config | people only; the agent doesn't touch it |

- **Floors: the riskiest signal sets the tier, and nothing averages it down.** Each of these sets a minimum on its own:
  - Blast radius: a script counts the packages that depend on each changed package. A code change (not only
    comments) in a package with 10 or more dependents is at least T2. Core code is T3: paths the repository declares
    as core in the policy file, or a package most of the repository depends on. Once 96's impact model exists, it
    replaces the count, and low or unknown confidence adds one.
  - A line added inside an existing function changes behavior, so it isn't "new code only".
  - A new import of a risky standard package (running commands, unsafe memory, cryptography) is at least T2.
- **Raises: weaker signals add up, and can only push the tier up.** A changed file that was reverted in the last 90
  days raises it by one. So do two or more of these: a file that usually changes together with a changed file is
  missing from the PR; the file is a churn hotspot; it was untouched for 6 months; recent commits on it mention a
  workaround or hack; the ADR 0089 score is 3 or more. These are ADR 0089's history signals, used as raises instead of
  a weighted average, so one serious signal can't be diluted by several harmless ones. The same history goes to the
  reviewer and the model check as context.
- **Every threshold is a starting value:** the sizes (Cloudflare's published review tiers), 10 dependents, two
  signals, 90 days and 6 months. The benchmark (section 6) sets and checks them.
- **Always handed to a person:** the PR edits the rules it's judged by; it's a draft; the code changed after review;
  the agent would merge its own work; a T1+ PR has no linked issue; tests were weakened.
- **Also per tier:** T0, a docs PR with generated or binary files, or a "pin" that is really a minor or major bump;
  T1, the flag defaults on, or the issue is labeled security or breaking-change; T2, impact confidence is unknown;
  T3, an irreversible change with no rollback note.

## 3. Minimal viable evidence (MVE) per tier

All evidence must be for the exact commit being merged. Rows marked GitHub are enforced by branch protection.

| Evidence | T0 | T1 | T2 | T3 | T4 |
|---|---|---|---|---|---|
| Required checks pass (GitHub) | ✓ | ✓ | ✓ | ✓ | ✓ |
| Tests cover the change (96's criteria, 100's selection) | existing suite (none for docs) | tests that touch the changed code | the changed lines ran (80% placeholder) | as T2, plus integration tests and tests of code that uses it | as T3 |
| Review with no blocking findings (GitHub) | ✓ | ✓ | ✓ | ✓ | advisory |
| Linked issue that matches the change | – | ✓ | ✓ | ✓ | ✓ |
| Human approval (GitHub) | once proven* | once proven* | a team member | the code owner | the code owner |
| Can be reverted or switched off | – | only adds code, or a flag | not irreversible | rollback note if not | ✓ |
| Track record: few later fixes | for auto-merge | for auto-merge | – | – | – |
| Model check: no concern | for auto-merge | for auto-merge | advisory | advisory | – |

\* A person approves until the tier has proven itself, and always when an agent wrote the PR.

**Model check:** fixed yes/no questions (does the change go beyond its issue, weaken tests, remove a safeguard, undo a
recent deliberate change, or contradict its description?), each with a calibrated probability. It can block, never approve. Which model is a 99
question.

**Substitutes** count only if the policy file declares them, and each use is recorded. None replaces the track
record or a human approval of an agent-written PR.

- Changed lines never ran (T2): agent-written tests that passed 98's golden path and ran in CI on this commit; or,
  if the package has no measurable coverage, a debt record that raises the tier by one.
- Tests at T1: 100's selected subset, if its confidence is high.
- Checks on the exact commit: a merge-queue run on the merged result.
- Tests of code that uses it (T3), if they can't run: approval from that code's owner.

## 4. When the evidence has a problem

The problem names follow 98's taxonomy. 97 decides what the merge does; 98 decides how the gap is fixed or deferred.

| Problem | What the merge decision does |
|---|---|
| missing or unavailable | fix first by running what produces it; if nothing can, defer as debt or hand to a person |
| irrelevant or partial (tests ran but don't cover the change) | fix first (add tests) or defer as debt; never merge on it |
| stale (older commit, base or policy) | re-run and wait; stale evidence never counts |
| flaky or non-reproducible | one re-run; a second failure is real. T0–T1 accept only known, quarantined flaky tests the change doesn't touch |
| slow | wait up to 24 hours, then fix first or hand to a person |
| contradictory (e.g. CI green but the changed lines never ran) | hand to a person, showing both readings |
| unknown (the tool couldn't measure) | never passes |

The agent gets 2 fix attempts per commit and 4 per PR, then the PR goes to a person.

## 5. Outcomes and the record

- **Merge:** the agent turns on merge-when-ready for that commit. The verdict is a required check on the commit, so
  a new push needs a new verdict.
- **Remediate (fix first):** the gap goes into 98's golden path, fixed now or deferred as recorded debt, and then everything is
  checked again.
- **Escalate (hand to a person):** the agent posts what's missing and assigns the right person. People can always override,
  and overrides are logged.

Every decision appends a record (the tier, each piece of evidence and its source, the outcome), kept for at least
12 months. 98 audits it, and the benchmark is checked against it.

## 6. How the thresholds are set and checked

- **Starting values:** the thresholds in section 2 and the 80% placeholder above.
- **Benchmark** (Hofni's proposal, placed in 97 by Ella): past PRs later followed by a customer case or a quick bug
  fix, labeled by how critical the problem was. For each tier: would the gate have merged a PR that later needed a
  fix, and which signals would have flagged it? It needs several repositories and post-merge outcomes (reverts,
  fix-forwards, customer cases), not review findings. The 246 fullsend and infra-deployments PRs analyzed so far were
  collected to evaluate the ADR 0089 scorer, and their 5 defects come from review findings, so they can test the
  pipeline but not the thresholds.
- **In production:** a monthly review, per tier, of automatic merges that later needed a fix. Two in 30 days send
  the tier back to human approval.

## 7. Rollout and the work that comes first

| Step | What happens | Needs first |
|---|---|---|
| 1. Observe | classify every PR and publish its tier and what's missing, for at least 30 days | a classification script (paths, size, dependents, author, agent-written or not, linked issue, file history); buildable now |
| 2. Explicit | docs PRs and Renovate bumps in one repository; fullsend's existing required approval triggers each merge | on fullsend: the verdict as a required check, approvals reset on a new push; supply-chain checks for bumps |
| 3. Automatic | only PRs by people or bots like Renovate; no approval needed | no wrong merges in observe; the benchmark and an outcome baseline (fullsend#6892); a model for the checks, approved under Red Hat's AI policy (99); the repository stops requiring an approval on every PR |
| 4. T1 and up | T1 next; T2 and above stay human-approved until the data says otherwise | 96's impact model and adequacy criteria; reversibility and weakened-test checks (99) |

In the 246 PRs checked so far, 13 could merge with no approval under these rules: 9 docs PRs by maintainers and 4
Renovate bumps. Most other low-tier PRs were written by an agent.

## 8. Open questions

- Is our reading of Red Hat's AI guidelines right: agent-written PRs are always human-approved, and Renovate bumps fall
  outside them?
- Is the split with 96, 98 and 100 in section 1 the right one?
- Blast radius and history: the dependent count stands in until 96's model exists, and the history signals come from
  ADR 0089. Should ADR 0089's script compute them for both, or should they fold into 96's impact model?
