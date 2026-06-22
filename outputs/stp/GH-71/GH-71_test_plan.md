# Test Plan

## **Report Failure When Agent Errors With No Commits - Quality Engineering Plan**

### **Metadata & Tracking**

- **Enhancement(s):** [GH-71](https://github.com/guyoron1/fullsend/pull/71)
- **Feature Tracking:** [GH-71](https://github.com/guyoron1/fullsend/pull/71) — fix(#2378): report failure when agent errors with no commits
- **Epic Tracking:** [#2378](https://github.com/fullsend-ai/fullsend/issues/2378) (upstream)
- **QE Owner(s):** QualityFlow (auto-generated)
- **Owning SIG:** N/A
- **Participating SIGs:** N/A

**Document Conventions (if applicable):** N/A

### **Feature Overview**

This fix addresses issue #2378 where agent runs that exit with a non-zero exit code but produce no commits silently succeed without reporting the failure back to the user. The change propagates the agent's exit code (`lastExitCode`) from the `runAgent()` function to the post-script via the `AGENT_EXIT_CODE` environment variable. The post-code script now detects agent errors (non-zero exit with no feature branch or no changed files), sets `AGENT_ERROR_EXIT=true`, and posts a descriptive failure comment to the originating GitHub issue via `gh issue comment`. Additionally, the status comment system uses `lastExitCode` to distinguish successful runs from agent errors when determining the completion status.

---

### **I. Motivation and Requirements Review (QE Review Guidelines)**

This section documents the mandatory QE review process. The goal is to understand the feature's value,
technology, and testability before formal test planning.

#### **1. Requirement & User Story Review Checklist**

- [ ] **Review Requirements**
  - Reviewed the relevant requirements.
  - GH-71 is a mirror of upstream fullsend-ai/fullsend#2381 fixing #2378. The requirement is clear: when an agent process errors out without producing any commits, the system must report a failure status to the user rather than silently succeeding.
- [ ] **Understand Value and Customer Use Cases**
  - Confirmed clear user stories and understood.
  - Understand the difference between community and product requirements.
  - **What is the value of the feature for customers**.
  - Ensured requirements contain relevant **customer use cases**.
  - Users triggering code agents via `/fs-code` expect feedback when the agent fails. Without this fix, a failed agent run appears as a no-op, leaving users unaware that something went wrong and unable to take corrective action.
- [ ] **Testability**
  - Confirmed requirements are **testable and unambiguous**.
  - The fix is testable at multiple levels: unit tests for `lastExitCode` propagation in `run.go`, functional tests for `post-code.sh` exit behavior under various `AGENT_EXIT_CODE` values, and the `report_failure_to_issue()` function's comment posting behavior.
- [ ] **Acceptance Criteria**
  - Ensured acceptance criteria are **defined clearly** (clear user stories; product requirements clearly defined in Jira).
  - AC1: When agent exits non-zero and produces no branch/commits, a failure comment is posted to the issue. AC2: When agent exits zero with no branch, it is treated as a no-op (no error). AC3: Status comments reflect the correct completion status based on exit code.
- [ ] **Non-Functional Requirements (NFRs)**
  - Confirmed coverage for NFRs, including Performance, Security, Usability, Downtime, Connectivity, Monitoring (alerts/metrics), Scalability, Portability (e.g., cloud support), and Docs.
  - No performance, security, or scalability concerns. The fix adds a single `gh issue comment` call on the error path. The `report_failure_to_issue()` function is best-effort and does not block the exit.

#### **2. Known Limitations**

- The `report_failure_to_issue()` function is best-effort: if the `gh` CLI is unavailable or the token has expired, the failure comment will not be posted, though the script still exits non-zero.
- The `AGENT_EXIT_CODE` variable only captures the last iteration's exit code in multi-iteration validation loops.
- Status comment failure reporting depends on the mint token service being available; if minting fails, the status comment is skipped with a warning.

#### **3. Technology and Design Review**

- [ ] **Developer Handoff/QE Kickoff**
  - A meeting where Dev/Arch walked QE through the design, architecture, and implementation details. **Critical for identifying untestable aspects early.**
  - Code reviewed via PR #71. The fix modifies `internal/cli/run.go` (Go, adding `lastExitCode` variable and `AGENT_EXIT_CODE` env var to post-script defer) and `internal/scaffold/fullsend-repo/scripts/post-code.sh` (Bash, adding ERR trap and failure comment logic).
- [ ] **Technology Challenges**
  - Identified potential testing challenges related to the underlying technology.
  - The post-code.sh script requires shell-level testing with mocked `gh` CLI. The `runAgent()` function requires mocking the sandbox runtime to control exit codes.
- [ ] **Test Environment Needs**
  - Determined necessary **test environment setups and tools**.
  - Standard Go test environment with `testify` assertions. Shell script tests require a mock `gh` binary or test harness.
- [ ] **API Extensions**
  - Reviewed new or modified APIs and their impact on testing.
  - No new APIs. The `AGENT_EXIT_CODE` environment variable is the new interface between `runAgent()` and post-scripts.
- [ ] **Topology Considerations**
  - Evaluated multi-cluster, network topology, and architectural impacts.
  - N/A — changes are CLI-layer only with no topology dependencies.

### **II. Software Test Plan (STP)**

This STP serves as the **overall roadmap for testing**, detailing the scope, approach, resources, and schedule.

#### **1. Scope of Testing**

Testing covers the propagation of agent exit codes through the fullsend CLI run pipeline, the post-code script's failure detection and reporting logic, and the status comment system's handling of non-zero exit codes. The scope includes both the Go code path (`internal/cli/run.go`) and the Bash post-script (`scripts/post-code.sh`).

**Testing Goals**

- **P0:** Verify agent exit code is correctly propagated to post-script via `AGENT_EXIT_CODE` environment variable
- **P0:** Verify failure comment is posted to the GitHub issue when agent errors with no commits
- **P1:** Verify post-code script correctly distinguishes agent errors from no-op runs (zero vs non-zero exit)
- **P1:** Verify status comment reflects correct failure/success/cancelled status based on exit code
- **P2:** Verify reconcile-status command correctly finalizes orphaned comments

**Out of Scope (Testing Scope Exclusions)**

- [ ] Sandbox creation, bootstrap, and agent runtime execution
  - *Rationale:* These are upstream of the fix and covered by existing tests
- [ ] GitHub API rate limiting and error handling
  - *Rationale:* Platform-level concern; `gh` CLI handles retries internally
- [ ] Mint token service functionality
  - *Rationale:* Separate subsystem with its own test coverage
- [ ] Pre-commit hooks, secret scanning, and branch push logic in post-code.sh
  - *Rationale:* Unchanged by this PR; covered by existing tests

#### **2. Test Strategy**

**Functional**

- [ ] **Functional Testing** — Validates that the feature works according to specified requirements and user stories
  - *Details:* Verify exit code propagation, failure comment posting, and status comment derivation across all code paths (success, failure, cancellation, no-op).
- [ ] **Automation Testing** — Confirms test automation plan is in place for CI and regression coverage (all tests are expected to be automated)
  - *Details:* All tests are Go unit tests (`*_test.go`) or shell script functional tests runnable in CI. No manual testing required.
- [ ] **Regression Testing** — Verifies that new changes do not break existing functionality
  - *Details:* Existing `run_test.go` and `reconcilestatus_test.go` tests continue to pass. LSP analysis confirms `runAgent()` callers are unchanged.

**Non-Functional**

- [ ] **Performance Testing** — Validates feature performance meets requirements (latency, throughput, resource usage)
  - *Details:* Not applicable. The fix adds a single env var assignment and one conditional `gh issue comment` call on the error path.
- [ ] **Scale Testing** — Validates feature behavior under increased load and at production-like scale
  - *Details:* Not applicable. No scale-sensitive changes.
- [ ] **Security Testing** — Verifies security requirements, RBAC, authentication, authorization, and vulnerability scanning
  - *Details:* Not applicable. No new credentials, tokens, or authentication flows introduced. `PUSH_TOKEN` handling is unchanged.
- [ ] **Usability Testing** — Validates user experience and accessibility requirements
  - *Details:* The failure comment message is human-readable and includes a link to the workflow run for debugging.
- [ ] **Monitoring** — Does the feature require metrics and/or alerts?
  - *Details:* Not applicable. The fix uses existing `::error::` and `::warning::` GitHub Actions annotations.

**Integration & Compatibility**

- [ ] **Compatibility Testing** — Ensures feature works across supported platforms, versions, and configurations
  - *Details:* Not applicable. CLI changes are platform-independent Go code and POSIX shell.
- [ ] **Upgrade Testing** — Validates upgrade paths from previous versions, data migration, and configuration preservation
  - *Details:* Not applicable. No persistent state or configuration changes.
- [ ] **Dependencies** — Blocked by deliverables from other components/products
  - *Details:* Depends on `gh` CLI being available on the runner. This is a pre-existing dependency.
- [ ] **Cross Integrations** — Does the feature affect other features or require testing by other teams?
  - *Details:* The `AGENT_EXIT_CODE` env var is a new interface consumed by all post-scripts. Other post-scripts (post-review.sh, post-fix.sh) should be verified for compatibility.

**Infrastructure**

- [ ] **Cloud Testing** — Does the feature require multi-cloud platform testing?
  - *Details:* Not applicable. Changes are CI-runner-level, not cloud-dependent.

#### **3. Test Environment**

- **Cluster Topology:** N/A — no cluster required (CLI and shell script tests)
- **Platform & Product Version(s):** Go 1.26+, GitHub Actions runner (ubuntu-latest)
- **CPU Virtualization:** N/A
- **Compute Resources:** Standard CI runner
- **Special Hardware:** None
- **Storage:** Standard filesystem
- **Network:** GitHub API access for `gh` CLI (mocked in unit tests)
- **Required Operators:** None
- **Platform:** Linux (CI runner)
- **Special Configurations:** `GH_TOKEN` or `PUSH_TOKEN` environment variable for `gh` CLI

#### **3.1. Testing Tools & Frameworks**

- **Test Framework:** Go `testing` package with `testify` assertions (standard — no new tools)
- **CI/CD:** GitHub Actions (standard — no new tools)
- **Other Tools:** None

#### **4. Entry Criteria**

The following conditions must be met before testing can begin:

- [ ] Requirements and design documents are **approved and merged**
- [ ] Test environment can be **set up and configured** (see Section II.3 - Test Environment)
- [ ] PR #71 changes are available on the test branch
- [ ] Go module dependencies are resolved (`go mod download`)

#### **5. Risks**

- [ ] **Timeline/Schedule**
  - Risk: Low risk — fix is well-scoped to two files with clear behavior changes
  - Mitigation: All tests are automated and run in CI
- [ ] **Test Coverage**
  - Risk: Shell script (`post-code.sh`) error paths may be difficult to test in isolation
  - Mitigation: Use mock `gh` binary and controlled environment variables to simulate failure conditions
- [ ] **Test Environment**
  - Risk: `gh` CLI behavior may differ between versions
  - Mitigation: Pin `gh` CLI version in CI; mock API responses in functional tests
- [ ] **Untestable Aspects**
  - Risk: The `report_failure_to_issue()` ERR trap interaction with `set -euo pipefail` may have edge cases
  - Mitigation: Test with various failure points (branch check, changed files check, push failure)
- [ ] **Resource Constraints**
  - Risk: None identified
  - Mitigation: N/A
- [ ] **Dependencies**
  - Risk: Upstream `statuscomment` package changes could affect `PostCompletion` behavior
  - Mitigation: LSP analysis confirms `PostCompletion` interface is stable; existing tests cover the contract
- [ ] **Other**
  - Risk: None identified
  - Mitigation: N/A

---

### **III. Test Scenarios & Traceability**

This section links requirements to test coverage, enabling reviewers to verify all requirements are tested.

#### **1. Requirements-to-Tests Mapping**

- **Requirement ID:** GH-71
- **Requirement Summary:** Agent exit code is propagated to post-script for failure detection
- **Test Scenarios:**
  - Verify AGENT_EXIT_CODE set when agent exits non-zero (positive)
  - Verify AGENT_EXIT_CODE is zero on successful agent run (positive)
  - Verify post-script receives exit code via environment (positive)
  - Verify lastExitCode updated after each iteration (positive)
- **Test Type:** Unit Tests
- **Priority:** P0

---

- **Requirement ID:** GH-71
- **Requirement Summary:** Post-code script reports failure to issue when agent errors with no commits
- **Test Scenarios:**
  - Verify failure comment posted on agent error without branch (positive)
  - Verify failure comment includes workflow run URL (positive)
  - Verify failure comment distinguishes agent error from post-script error (positive)
  - Verify no comment when gh CLI unavailable (negative)
- **Test Type:** Functional
- **Priority:** P0

---

- **Requirement ID:** GH-71
- **Requirement Summary:** Post-code script distinguishes agent error from no-op
- **Test Scenarios:**
  - Verify clean exit when agent succeeds with no branch (positive)
  - Verify error exit when agent fails with no branch (negative)
  - Verify clean exit when agent succeeds with no changes (positive)
  - Verify error exit when agent fails with no changes (negative)
- **Test Type:** Functional
- **Priority:** P1

---

- **Requirement ID:** GH-71
- **Requirement Summary:** Post-code script detects agent error on main/master with no feature branch
- **Test Scenarios:**
  - Verify error reported when on main with non-zero exit (negative)
  - Verify no-op notice when on main with zero exit (positive)
- **Test Type:** Functional
- **Priority:** P1

---

- **Requirement ID:** GH-71
- **Requirement Summary:** Post-code script detects agent error with no changed files
- **Test Scenarios:**
  - Verify error reported with empty changeset and non-zero exit (negative)
  - Verify no-op when empty changeset and zero exit (positive)
- **Test Type:** Functional
- **Priority:** P1

---

- **Requirement ID:** GH-71
- **Requirement Summary:** Status comment reflects failure when agent exits non-zero
- **Test Scenarios:**
  - Verify status comment posts failure on non-zero exit (positive)
  - Verify status comment posts success on zero exit (positive)
  - Verify status comment posts cancelled on context cancellation (positive)
- **Test Type:** Unit Tests
- **Priority:** P1

---

- **Requirement ID:** GH-71
- **Requirement Summary:** Reconcile-status command finalizes orphaned status comments
- **Test Scenarios:**
  - Verify orphaned comment finalized as terminated (positive)
  - Verify orphaned comment finalized as cancelled (positive)
  - Verify no-op when comment already finalized (positive)
- **Test Type:** Unit Tests
- **Priority:** P2

---

### **IV. Sign-off and Approval**

This Software Test Plan requires approval from the following stakeholders:

* **Reviewers:**
  - [Name / @github-username]
  - [Name / @github-username]
* **Approvers:**
  - [Name / @github-username]
  - [Name / @github-username]
