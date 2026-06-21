# Fullsend Test Plan

## **Code Agent Status Comment Should Reflect Actual Outcome When No PR Is Created - Quality Engineering Plan**

### Metadata & Tracking

- **Enhancement:** [GH-2378](https://github.com/fullsend-ai/fullsend/issues/2378)
- **Feature Tracking:** [GH-2378](https://github.com/fullsend-ai/fullsend/issues/2378)
- **Epic Tracking:** N/A
- **QE Owner:** Unassigned
- **Owning SIG:** N/A
- **Participating SIGs:** N/A

**Document Conventions:** Standard STP conventions apply. "Agent" refers to the code agent subprocess. "Post-script" refers to `post-code.sh` executed on the runner after the sandbox is destroyed.

### Feature Overview

When a code agent run terminates due to an API error (e.g., 429 RESOURCE_EXHAUSTED) and produces no commits, the status comment posted to the GitHub issue previously reported "Finished Code, Success" — misleading humans who rely on it as the primary feedback signal. This fix propagates the agent's exit code from `run.go` into the post-script via the `AGENT_EXIT_CODE` environment variable, enabling `post-code.sh` to distinguish agent errors from intentional no-ops and report an accurate "Code agent failed" status comment with the exit code.

---

### Section I — Motivation & Requirements Review

#### I.1 — Requirement & User Story Review Checklist

- [ ] **Reviewed the relevant requirements.**
  - GH-2378 describes a concrete bug: status comment says "Success" when agent errors with no output.
  - Root cause identified: `lastExitCode` was declared after the status and post-script defers, so neither closure could read it.
  - Referenced incident on issue #2169 with clear reproduction evidence.

- [ ] **Confirmed clear user stories and understood. Understand the value and customer use cases.**
  - Primary user: human who invokes `/fs-code` and reads the status comment to know the outcome.
  - Value: eliminates confusion when agent fails silently, reducing time wasted waiting for a PR that will never appear.

- [ ] **Confirmed requirements are **testable and unambiguous**.**
  - Four validation criteria defined in the issue body with clear pass/fail conditions.
  - Each criterion maps to an observable behavior (comment content, exit code presence, success path preservation).

- [ ] **Ensured acceptance criteria are **defined clearly**.**
  - AC1: Status comment says "Failed" (not "Success") when agent exits non-zero with no commits.
  - AC2: Comment includes failure reason/exit code.
  - AC3: Normal successful runs still report "Success."
  - AC4: Intentional no-change runs (exit 0, no commits) are unaffected.

- [ ] **Confirmed coverage for NFRs.**
  - No performance, scale, or security NFRs impacted — change is scoped to error reporting in a shell script and a variable declaration move in Go.

#### I.2 — Known Limitations

- The fix only covers the code agent post-script (`post-code.sh`). Other agent types (triage, review) have separate post-scripts that are not modified.
- The error comment does not include the specific error message from the agent transcript (e.g., "429 RESOURCE_EXHAUSTED") — only the numeric exit code. Extracting the error message from the JSONL output would require additional parsing not included in this change.
- If `AGENT_EXIT_CODE` is unset (e.g., older harness versions that don't pass it), the check defaults to `0`, preserving backward-compatible noop behavior.

#### I.3 — Technology and Design Review

- [ ] **Developer handoff meeting conducted or async review completed.**
  - PR #2381 provides detailed description of root cause and fix approach with inline code comments.

- [ ] **Technology challenges identified and mitigation planned.**
  - No new technology. Change uses existing bash variable checks and Go variable scoping.

- [ ] **Test environment needs identified.**
  - Shell tests run in bash without cluster. Go tests use existing `runAgent` test harness.

- [ ] **API or interface extensions reviewed.**
  - New environment variable `AGENT_EXIT_CODE` passed from Go harness to post-script. New internal variable `AGENT_ERROR_EXIT` used within `post-code.sh` to select error comment template.

- [ ] **Topology or deployment changes reviewed.**
  - No topology changes. Fix runs on the GitHub Actions runner, same as current post-script execution.

### Section II — Test Strategy & Environment

#### II.1 — Scope of Testing

This test plan covers the error detection and status reporting changes in the code agent post-script and the Go harness variable scoping fix. Testing validates that agent errors are correctly detected, propagated, and reported to users via issue comments.

**Testing Goals:**

- **P0:** Verify agent error detection at both exit points (no branch, no changed files) produces failure status instead of false success.
- **P0:** Verify error comment content distinguishes agent errors from post-script errors.
- **P0:** Verify intentional no-op behavior is preserved when agent exits 0.
- **P1:** Verify `AGENT_EXIT_CODE` environment variable is correctly propagated from `run.go` to `post-code.sh`.
- **P1:** Verify agent errors with produced changes still proceed to push/PR creation.
- **P2:** Verify edge cases (detached HEAD, various exit codes) are handled correctly.

**Out of Scope (Testing Scope Exclusions):**

- [ ] **Triage and review agent post-scripts** -- Different post-scripts with different logic; not modified in this change.
- [ ] **Upstream LLM API reliability (429 errors)** -- Platform-level concern; this fix handles the symptom (misleading status) not the cause (API rate limiting).
- [ ] **Status comment rendering/formatting in GitHub UI** -- GitHub platform responsibility; we test comment content, not rendering.
- [ ] **Sandbox execution environment** -- The sandbox is destroyed before post-script runs; not relevant to this change.

#### II.2 — Test Strategy

**Functional:**

- [x] **Functional Testing** -- Core agent error detection logic at both exit points (branch check, files check), error comment generation, and noop preservation. Validated via shell unit tests and Go unit tests.
- [x] **Automation Testing** -- All tests are automated. Shell tests in `post-code-test.sh` run via bash. Go tests in `run_test.go` use the existing test harness.
- [x] **Regression Testing** -- Existing `detect_noop` and `build_error_comment` tests extended with new agent-error cases. Existing noop and error comment tests preserved to prevent regression.
- [ ] **Upgrade Testing** -- N/A. No persistent state changes; new harness version simply passes the new env var. Backward compatible with older harness versions that do not set `AGENT_EXIT_CODE`.

**Non-Functional:**

- [ ] **Performance Testing** -- N/A. Change adds two string comparisons per post-script invocation; negligible overhead.
- [ ] **Scale Testing** -- N/A. Post-script runs once per agent invocation; no scale dimension.
- [ ] **Security Testing** -- N/A. No new inputs from untrusted sources. `AGENT_EXIT_CODE` is set by the Go harness, not user-controlled.
- [ ] **Usability Testing** -- N/A. Status comment is plain text; readability is validated by content checks.
- [ ] **Monitoring** -- N/A. No new metrics or alerts introduced.

**Integration & Compatibility:**

- [ ] **Compatibility Testing** -- N/A. Backward compatible: unset `AGENT_EXIT_CODE` defaults to `0` (noop behavior preserved).
- [ ] **Dependencies** -- Post-script depends on `gh` CLI for comment posting. No new dependencies introduced.
- [ ] **Cross Integrations** -- N/A. Change is self-contained within code agent pipeline.

**Infrastructure:**

- [ ] **Cloud Testing** -- N/A. Runs on GitHub Actions runners; no cloud-specific behavior changes.

#### II.3 — Test Environment

- **Cluster Topology:** N/A — no cluster required. Tests run in bash shell and Go test harness.
- **Platform Version:** GitHub Actions runner (Ubuntu latest)
- **CPU Virtualization:** N/A
- **Compute:** Standard GitHub Actions runner
- **Special Hardware:** None
- **Storage:** Local filesystem only
- **Network:** GitHub API access for `gh issue comment` (mocked in unit tests)
- **Operators:** N/A
- **Platform:** Linux (bash 5.x+)
- **Special Configs:** `AGENT_EXIT_CODE`, `AGENT_ERROR_EXIT`, `PUSH_TOKEN`, `REPO_FULL_NAME`, `ISSUE_NUMBER` environment variables

#### II.3.1 — Testing Tools & Frameworks

No new or special tools required. Tests use standard bash assertions and the existing Go test harness.

#### II.4 — Entry Criteria

- [ ] PR #2381 merged or branch available for testing
- [ ] Go module dependencies resolved (`go mod download`)
- [ ] Shell test script executable (`chmod +x post-code-test.sh`)

#### II.5 — Risks

- [ ] **Timeline**
  - Risk: Low complexity change; minimal timeline risk.
  - Mitigation: All tests are automated and run in CI.
  - Status: [ ]

- [ ] **Coverage**
  - Risk: Edge cases in exit code handling (e.g., signal-killed processes returning 128+N).
  - Mitigation: Test with various non-zero exit codes (1, 2) and verify default behavior for unset variable.
  - Status: [ ]

- [ ] **Environment**
  - Risk: Differences between local bash and CI runner bash versions.
  - Mitigation: Tests use POSIX-compatible constructs; `post-code.sh` uses `bash` explicitly.
  - Status: [ ]

- [ ] **Untestable**
  - Risk: Actual LLM API 429 errors are non-deterministic and cannot be reliably triggered.
  - Mitigation: Test the detection logic in isolation; the exit code mechanism is deterministic regardless of failure cause.
  - Status: [ ]

- [ ] **Resources**
  - Risk: None identified.
  - Mitigation: N/A
  - Status: [ ]

- [ ] **Dependencies**
  - Risk: `gh` CLI availability on runner for comment posting.
  - Mitigation: `gh` is pre-installed on GitHub Actions runners; `report_failure_to_issue` has error handling for comment failure.
  - Status: [ ]

- [ ] **Other**
  - Risk: None identified.
  - Mitigation: N/A
  - Status: [ ]

---

### Section III — Requirements-to-Tests Mapping

#### III.1 — Requirements Mapping

- **Requirement ID:** GH-2378
  **Requirement Summary:** Agent error detection at branch check point
  **Test Scenarios:** Verify post-script exits with error when agent exits non-zero and no feature branch exists
  **Tier:** Unit Tests
  **Priority:** P0

- **Requirement ID:**
  **Requirement Summary:** Agent error detection at changed-files check point
  **Test Scenarios:** Verify post-script exits with error when agent exits non-zero on feature branch with no changed files
  **Tier:** Unit Tests
  **Priority:** P0

- **Requirement ID:**
  **Requirement Summary:** Noop behavior preserved for successful no-op runs
  **Test Scenarios:** Verify post-script exits cleanly when agent exits 0 with no commits (both branch and files paths)
  **Tier:** Unit Tests
  **Priority:** P0

- **Requirement ID:**
  **Requirement Summary:** Error comment distinguishes agent errors from post-script errors
  **Test Scenarios:** Verify error comment says "Code agent failed" (not "Post-code script failed") when AGENT_ERROR_EXIT is true
  **Tier:** Unit Tests
  **Priority:** P0

- **Requirement ID:**
  **Requirement Summary:** Error comment includes agent exit code
  **Test Scenarios:** Verify error comment body contains the numeric agent exit code value
  **Tier:** Unit Tests
  **Priority:** P1

- **Requirement ID:**
  **Requirement Summary:** Non-agent errors preserve existing error comment format
  **Test Scenarios:** Verify error comment says "Post-code script failed" when AGENT_ERROR_EXIT is false or unset
  **Tier:** Unit Tests
  **Priority:** P1

- **Requirement ID:**
  **Requirement Summary:** AGENT_EXIT_CODE propagated from Go harness to post-script
  **Test Scenarios:** Verify lastExitCode is declared before defer closures and AGENT_EXIT_CODE env var is appended to post-script command environment
  **Tier:** Unit Tests
  **Priority:** P0

- **Requirement ID:**
  **Requirement Summary:** Agent errors with produced changes still proceed normally
  **Test Scenarios:** Verify post-script continues to push/PR flow when agent exits non-zero but changes exist
  **Tier:** Unit Tests
  **Priority:** P1

- **Requirement ID:**
  **Requirement Summary:** Detached HEAD with agent error handled correctly
  **Test Scenarios:** Verify post-script exits with agent error (not noop) when on detached HEAD with non-zero exit code
  **Tier:** Unit Tests
  **Priority:** P2

- **Requirement ID:**
  **Requirement Summary:** End-to-end agent failure produces correct issue comment
  **Test Scenarios:** Verify that a simulated agent failure (non-zero exit, no commits) results in an issue comment containing "Code agent failed" with exit code and workflow run link
  **Tier:** Functional
  **Priority:** P1

---

### Section IV — Sign-off

| Role | Name | Date |
|:-----|:-----|:-----|
| QE Lead | | |
| Dev Lead | | |
| Product Owner | | |
