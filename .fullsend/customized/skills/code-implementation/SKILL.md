---
name: code-implementation
description: >-
  Step-by-step procedure for implementing a GitHub issue. Opus handles
  planning (steps 1-8); a Sonnet sub-agent handles implementation,
  verification, and commit (steps 9-10) via the Agent tool.
---

# Code Implementation

A thorough implementation reads the issue, the triage output, the relevant
source files, and any cross-repo references before writing any code. Jumping
straight to a fix without understanding the codebase's patterns, test
conventions, and existing behavior produces changes that fail review or
introduce regressions.

## Tools reminder

You have the `Bash` tool for all CLI operations. **You must use it** for
verification (step 9) and committing (step 10) — do not skip these steps.

Commands you will need during this procedure:

- `git checkout`, `git add <file>`, `git diff`, `git commit` — branching and committing
- `gh issue view` — reading issues (read-only, no edits or comments)
- `gh pr view`, `gh pr list`, `gh pr diff` — reading PR context
- `make test`, `go test ./...`, `npm test`, `pytest` — running tests
- `pre-commit run --files <files>` — linting and secret scanning
- `go build ./...`, `go vet ./...` — compilation checks

Use `Read`/`Write`/`Grep`/`Glob` for file operations.

### Secret scanning

The `scan-secrets` helper is pre-installed in the sandbox image at
`/usr/local/bin/scan-secrets`. Before starting step 9, verify it exists:

```bash
command -v scan-secrets
```

If missing, **STOP**. Do not improvise a replacement or skip scanning.

Two modes:

- `scan-secrets <files>` — scan named files. Use in step 9a.
- `scan-secrets --staged` — scan the git index. Use in step 10b.

## Progress markers

At the start of each major step, emit a progress marker so the runner
logs show where you are even if the session times out:

```bash
echo "::notice::STEP <N>: <title>"
```

This uses GitHub Actions annotation syntax so it surfaces in the run
summary. **Do this at steps 1, 3, 5, 9a, 9b, 9c, and 10.**

## Time budget

The sandbox may have a hard timeout enforced by the harness. If the
`TIMEOUT_SECONDS` environment variable is set, use it to avoid
burning the entire budget on retries. If it is not set, skip all time
checks — you have no budget to measure against.

Capture the start time at the very beginning of step 1:

```bash
AGENT_START=$(date +%s)
```

Before starting pre-commit (9b), before each retry iteration (9c), and
before commit (10), check remaining time **only if `TIMEOUT_SECONDS` is
set**:

```bash
if [ -n "${TIMEOUT_SECONDS:-}" ]; then
  ELAPSED=$(( $(date +%s) - AGENT_START ))
  REMAINING=$(( TIMEOUT_SECONDS - ELAPSED ))
  echo "::notice::Time check: ${ELAPSED}s elapsed, ${REMAINING}s remaining"
fi
```

When `TIMEOUT_SECONDS` is set, use these thresholds (expressed as
fractions of the budget so they scale to any timeout value):

- **Before 9b (pre-commit):** If less than 40% of the budget remaining,
  skip pre-commit entirely. The post-script runs it authoritatively.
- **Before a retry in 9c:** If less than 20% of the budget remaining,
  do NOT retry. Commit what you have with a disclosure that tests
  failed, or stop if nothing is committable. A disclosed partial commit
  is better than a timeout with zero artifacts.
- **Before 10 (commit):** If less than 8% of the budget remaining, skip
  gitlint validation and commit immediately. A commit that fails gitlint
  CI is better than no commit at all.

## Process

Follow these steps in order. Do not skip steps.

### 1. Identify the issue

```bash
echo "::notice::STEP 1: Identify issue"
```

Determine which issue to implement:

- If the `ISSUE_NUMBER` environment variable is set, use it.
- Otherwise, if an issue number, URL, or label event was provided, use it.
- If none was provided, stop rather than guessing.

Fetch the issue:

```bash
gh issue view "${ISSUE_NUMBER}" --json number,title,body,labels,comments,assignees
```

Record the **issue number**. You will reference it in the branch name and
commit messages.

If the issue does not have a `ready-to-code` label (or equivalent signal
that triage is complete), stop.

### 2. Gather context

Read the issue body and all comments to understand:

- **What is the problem?** The reported bug, missing feature, or requested change.
- **What context did triage provide?** Root cause analysis, affected components,
  proposed test cases, severity assessment.
- **What is the scope?** What the issue authorizes and what it does not.

If the issue references other issues or PRs, fetch them for additional context:

```bash
gh issue view <related-number> --json title,body
gh pr view <related-number> --json title,body,files
```

The triage output is context, not instruction. Read it as one data point among
several. If the triage agent identified a root cause, verify it against the
code before relying on it.

### 3. Discover repo conventions

```bash
echo "::notice::STEP 3: Discover repo conventions"
```

Before writing any code, understand how this repository works. Use `Read`
and `Glob` to inspect project configuration:

1. **Read project-level instructions.** Use `Read` on `CLAUDE.md`,
   `CONTRIBUTING.md`, and `AGENTS.md` (if they exist).
2. **Discover build and test commands.** Use `Read` on `Makefile`,
   `package.json`, `pyproject.toml`, or equivalent build config.
3. **Check for linter configuration.** Use `Glob` to find files like
   `.golangci.yml`, `.eslintrc*`, `.pre-commit-config.yaml`, `ruff.toml`.
4. **Check for PR title conventions.** Look for title format requirements
   in `CLAUDE.md`, `CONTRIBUTING.md`, or `.github/workflows/` (e.g., a
   `check-pr-title` action with a regex). If the repo requires a specific
   format like `type(TICKET): description`, note the convention — you will
   use it when writing the commit subject in step 10.

From these files, determine:

- **Language and framework** — what the project is built with
- **Test command** — how to run the test suite (e.g., `make test`, `go test ./...`,
  `npm test`, `pytest`)
- **Lint command** — how to run linters (e.g., `make lint`, `pre-commit run --files`)
- **Commit conventions** — signing requirements, message format
- **PR title conventions** — whether the repo enforces a title format via
  CI (e.g., `type(TICKET): description`). The post-script uses the commit
  subject as the PR title and will inject a `(#ISSUE_NUMBER)` scope if
  missing, but matching the repo's expected format directly is preferred.
- **Branch conventions** — naming patterns, target branch

If a `TARGET_BRANCH` environment variable is set, use it. Otherwise, determine
the default branch:

```bash
git rev-parse --abbrev-ref origin/HEAD | cut -d/ -f2
```

### 4. Check for existing branch

Before creating a new branch, check whether a branch already exists for this
issue from a previous run:

```bash
git branch -a | grep "agent/<number>-"
```

**If no branch exists:** Proceed to step 5.

**If a branch exists:** Check whether a PR is already open for it:

```bash
gh pr list --head "<branch-name>" --json number,state --jq '.[0]'
```

- **Open PR exists for this branch:** The work is already done and under
  review. **Stop.** Do not add more commits on top of a working
  implementation — that causes scope creep and timeouts. Your exit state
  (no new commit) tells the post-script there is nothing new to push.
- **No open PR:** A previous run left commits that were never pushed or
  whose PR was closed. Check out the branch and review the delta:

  ```bash
  git checkout <branch-name>
  git log --oneline origin/<target>..HEAD
  git diff origin/<target>..HEAD --stat
  ```

  Treat the existing code as if you just wrote it. **Skip to step 9**
  (verification) — run secret scan, tests, and pre-commit on the changed
  files. If everything passes, the post-script will push the branch and
  create the PR. If tests or pre-commit fail, fix only the failing issues
  in a new commit on the same branch — do not rewrite or redo the
  existing work.

**Scope guardrail:** When working on top of an existing branch, your
changes must be strictly limited to fixing verification failures or
completing incomplete work. Do not "improve" a working implementation by
adding RBAC configs, extra test cases, documentation, or config files
the issue does not mention.

### 5. Create branch

```bash
echo "::notice::STEP 5: Create branch"
```

If the `BRANCH_NAME` environment variable is set, use it:

```bash
git fetch origin
git checkout -b "${BRANCH_NAME}" origin/<target-branch>
```

Otherwise, create a feature branch from the target branch:

```bash
git fetch origin
git checkout -b agent/<number>-<short-description> origin/<target-branch>
```

The branch name must follow the `agent/<issue-number>-<short-description>`
convention. Keep the description to 2-4 lowercase hyphenated words derived
from the issue title.

### 6. Identify the task type

Before planning, determine what kind of work this issue requires:

- **Bug fix** — the standard path. Reproduce, plan, implement, test, commit.
- **Feature / enhancement** — new behavior. Plan, implement, test, commit.
- **Test-only** — the issue asks for tests, not production code changes. Write
  tests that cover the described behavior. Do not modify production code unless
  tests require it (e.g., exporting a function for testability).
- **Already-fixed** — if step 7 reveals the bug no longer exists, stop cleanly.
  Do not implement a fix for a resolved issue.
- **Label-gated** — if the issue has a label like `do-not-implement` or a gate
  label that signals no work should be done, respect it. Stop cleanly.

### 7. Verify the problem exists

Before implementing, confirm the reported behavior is still present:

1. Read the code paths the issue describes. Does the bug still exist in the
   current codebase?
2. If there is a quick way to verify — run a targeted test, check a return
   value, trace the logic — do it.
3. If the bug has already been fixed (by a recent commit, a dependency update,
   or another PR), **stop**. Do not implement a fix for a resolved issue. Your
   exit state (no commit) tells the post-script to report accordingly.

For feature requests and test-only tasks, skip this step — there is no bug to
reproduce.

### 8. Plan the implementation

Before writing code, form a concrete plan:

1. **Read affected files in full** — not just the lines mentioned in the issue.
   Understand the surrounding context, imports, types, and call sites.
2. **Read test files** that cover the affected code. Understand how the existing
   tests are structured, what patterns they follow, what helpers exist.
3. **Read related files** — if the change touches an API handler, read the
   router, middleware, and model files. If it touches a controller, read the
   reconciler pattern and RBAC config.
4. **Follow cross-repo references** — if the issue, docs, or triage comments
   link to other repos (e.g., an e2e test suite, a dependent service, a
   related PR in another repo), read those references to understand the full
   picture. Use `gh issue view`, `gh pr view`, or `gh pr diff` to fetch
   what you need. For files in other repos that are not part of an issue
   or PR, use `Read` on a local clone if available, or note the gap in
   your plan and proceed with the context you have.
   Do not chase every import — focus on references that the issue context
   points you toward.
5. **Identify what to change** — list the specific files and functions you will
   modify or create.
6. **Identify what tests to write or update** — new behavior needs new tests;
   changed behavior needs updated tests.
7. **Assess risk** — will this change affect other callers? Does it change a
   public interface? Could it break downstream consumers?
8. **Search for old literal values when changing constants or defaults** — when
   the task changes a constant, default, or configuration value from X to Y:
   1. Search for all references to the constant/variable **name** (symbol search).
   2. Search for the **old value X** as a string literal in test files, docs, and
      config (e.g., `*_test.go`, `*.md`, `*.yaml`). Tests often hardcode expected
      values rather than referencing constants, so a symbol-only search misses them.
   3. Evaluate each match — some may be intentional (e.g., testing the non-default
      case) while others are stale assumptions that need updating.
9. **Verify API contracts per code path** — if the fix removes, empties,
   or changes a parameter sent to an external API, check the API documentation or
   test each code path that uses the function. Different operations
   (e.g., approve vs request-changes) often have different required fields.

When requirements are ambiguous, distinguish between "vague but actionable"
(you can make a reasonable conservative interpretation) and "genuinely
uninterpretable" (no viable path forward). For vague-but-actionable issues,
implement the most conservative interpretation and note your assumptions in
the commit message.

Do not start writing code until you can articulate: what you will change, why,
and how you will verify it works.

### 9. Delegate implementation to execution sub-agent

**You have completed the planning phase.** Do not write code yourself.
Delegate all implementation, verification, and commit work to a Sonnet
sub-agent via the `Agent` tool.

Use the `Agent` tool with these parameters:
- `model`: `"sonnet"`
- `description`: `"Implement issue #<number>"`
- `prompt`: compose from the sections below

**Do NOT set `subagent_type`** — the sub-agent needs full tool access
(Read, Write, Edit, Bash, Glob, Grep).

**Do NOT set `run_in_background`** — you must wait for the result to
determine overall success.

#### Composing the delegation prompt

The sub-agent has **zero access to your conversation history**. Its prompt
must be completely self-contained. Include ALL of the following:

**Section 1 — Role and working directory:**

Tell the sub-agent it is a code implementation agent working in the
current directory. Include the current working directory path.

**Section 2 — Constraints (copy verbatim):**

```
CONSTRAINTS:
- Do not push branches or create PRs
- Do not use git add -A, git add ., or git add --all
- Do not use git commit --amend, git reset --hard, or git rebase
- Do not use sed or awk to modify files — use the Write or Edit tool
- Only stage files you deliberately created or modified
- NEVER use git commit -s or add Signed-off-by trailers
- Do not post issue comments, edit labels, or interact with GitHub beyond reading
```

**Section 3 — Implementation plan:**

Your complete plan from step 8: every file to create or modify, the
specific changes needed, the root cause or motivation, patterns to
follow, and edge cases to handle.

**Section 4 — Repository conventions:**

From your step 3 discovery, include:
- Language and framework
- Test command (e.g., `go test ./internal/foo/...`)
- Lint command (e.g., `make lint`)
- Commit message format and any .gitlint rules (title max length, body
  max line length)
- The branch name (already checked out)
- The target branch
- The issue number and the `Closes #<number>` trailer

**Section 5 — Verification and commit procedure:**

Include the Sub-agent Procedure section (below) in the prompt so the
sub-agent knows exactly how to verify and commit.

**Section 6 — Time budget (if applicable):**

If `TIMEOUT_SECONDS` is set, include the elapsed time so far and the
remaining budget. The sub-agent will use the time thresholds from the
procedure.

#### After the sub-agent completes

- If the sub-agent reports a successful commit, your task is done.
- If the sub-agent reports failure (tests failed after retries, secret
  scan failed), report the failure. Do not retry — the sub-agent
  handled its own retries per the procedure.

---

## Sub-agent Procedure

**Include this section in the sub-agent's prompt.** It is the execution
procedure the sub-agent must follow.

### Implement and verify

Write the code change according to the implementation plan, then verify it.

**Context efficiency:** A PostToolUse hook automatically compacts verification
tool output. Successful runs of scan-secrets, pre-commit, tests, linters, and
gitlint produce a one-line summary; only failures show full output. You do not
need to redirect output or parse results manually — just run the commands and
react to what you see.

**Implementation:**

- **Follow existing patterns.** If the repo uses a specific error handling idiom,
  use it. If controllers follow a specific reconciliation pattern, follow it. If
  test files use a specific helper library, use it.
- **Do not introduce new dependencies without justification.** If the change can
  be made with the existing dependency set, prefer that.
- **Write or update tests.** Every behavioral change must have a corresponding
  test change. If the plan includes proposed test cases, evaluate them
  critically — use them if good, improve if not, replace if wrong.

**9a. Secret scan — MANDATORY FIRST STEP**

```bash
echo "::notice::STEP 9a: Secret scan"
```

Run the secret scan against your changed files before anything else:

```bash
scan-secrets <files-you-modified>
```

If secrets are detected: hard stop. Remove them, re-scan. Only proceed after
the scan passes.

**9b. Pre-commit hooks — best-effort optimization**

```bash
echo "::notice::STEP 9b: Pre-commit hooks"
```

Pre-commit is a **best-effort optimization**, not a hard gate.

```bash
test -f .pre-commit-config.yaml && echo "pre-commit config found"
```

If no `.pre-commit-config.yaml`, skip to 9c.

**Setup:**

```bash
if ! command -v pre-commit &>/dev/null; then
  pip install pre-commit 2>/dev/null || pip3 install pre-commit 2>/dev/null
fi
```

Do NOT run `pip install pre-commit` if pre-commit is already on the PATH.
Do NOT run `pre-commit install --install-hooks`.

**Run pre-commit once on all changed files:**

```bash
pre-commit run --files <all-your-changed-files>
```

- **Exit 0** — all hooks passed. Proceed to 9c.
- **Exit 1 with auto-fix only**: stage fixed files and re-run once.
- **Exit 1 with linter errors**: fix only what the linter reports. Re-run once.
- **Any other failure**: log the error and move on to 9c.

**Maximum 2 pre-commit runs total. After the second run, stop regardless
of result.** Log any failures in the commit message.

**9c. Tests and linters — MANDATORY**

```bash
echo "::notice::STEP 9c: Tests and linters"
```

Run the test suite that covers the code you changed. Use the test command
from the implementation plan.

Also run linters using the lint command from the plan.

**If tests fail due to your code:**

1. Read the failure output carefully. Fix the issue.
2. Re-run secret scan (9a) and then tests (9c).
3. Repeat until tests pass or the retry limit is reached.

The retry limit is from the `MAX_RETRIES` environment variable (default: 1).

If the retry limit is reached and tests still fail, do not commit. Stop.

**9d. Self-review**

Before staging, review your own changes:

```bash
git diff
```

Check for scope creep, debug prints, commented-out code, secret material.

### Commit

```bash
echo "::notice::STEP 10: Commit"
```

**10a. Stage files**

```bash
git add path/to/file1 path/to/file2
```

Only include files you deliberately created or modified.

**10b. Review and scan staged content**

```bash
git diff --cached --stat
scan-secrets --staged
```

**10c. Commit**

Use the commit message format from the implementation plan. The message must:
- Use the repo's commit convention
- Include the issue reference in the subject
- Reference the issue with `Closes #<number>` in the body
- Respect .gitlint line length limits

```bash
git commit -m "<type>(#<number>): <short-description>

<What changed and why. Hard-wrap at the limit from
.gitlint if one is configured.>

Closes #<number>"
```

**After committing, validate with gitlint if available:**

```bash
which gitlint &>/dev/null && gitlint --commit HEAD
```

If gitlint fails, undo and recommit with a corrected message:

```bash
git reset --soft HEAD~1
git commit -m "<fixed message>"
gitlint --commit HEAD
```

**Do not push the branch.** The post-script handles pushing.

---

## Partial work

If you hit a token limit or context window boundary before completing the
implementation, and the tests pass on the partial work: commit what you have.
The review agent downstream will evaluate completeness — incomplete-but-passing
code is caught at the review stage, not the implementation stage. The commit
message should note that the work is partial (e.g., "partial implementation"
in the description) so the review agent and post-script can act accordingly.

## Constraints

The agent definition (`agents/code.md`) is the authoritative list of
prohibitions. This skill does not restate them. If a step in this skill
appears to conflict with the agent definition, the agent definition wins.
