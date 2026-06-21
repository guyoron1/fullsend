# Test Plan

## **Code Agent Status Comment Should Reflect Actual Outcome When No PR Is Created - Quality Engineering Plan**

### **Metadata & Tracking**

- **Enhancement:** [GH-2378](https://github.com/fullsend-ai/fullsend/issues/2378)
- **Feature Tracking:** [GH-2378](https://github.com/fullsend-ai/fullsend/issues/2378)
- **Epic Tracking:** [GH-2378](https://github.com/fullsend-ai/fullsend/issues/2378)
- **QE Owner:** Unassigned
- **Owning SIG:** N/A
- **Participating SIGs:** N/A

**Document Conventions:** Priority levels follow P0 (critical) > P1 (important) > P2 (edge case). Test tiers: Functional (single-feature isolation), End-to-End (multi-feature workflow).

### **Feature Overview**

When a code agent run terminates due to an API error (e.g., 429 RESOURCE_EXHAUSTED) but produces no commits, the status comment posted to the originating GitHub issue incorrectly reports "Finished Code, Success." This fix propagates the agent's exit code from the Go runner (`internal/cli/run.go`) to the post-code shell script via the `AGENT_EXIT_CODE` environment variable. The post-script now checks this exit code at both no-op exit points (no feature branch and no changed files), and when non-zero, exits with failure and posts a distinct "Code agent failed" error comment instead of the misleading success message.

---

### **I. Motivation and Requirements Review (QE Review Guidelines)**

#### **I.1 - Requirement & User Story Review Checklist**

- [ ] **Reviewed the relevant requirements.** -- Confirmed the requirement is clear: the status comment must accurately reflect agent outcome when no PR is created.
  - GH-2378 describes the gap between agent session failure (`is_error: true`) and the misleading "Success" status comment.
  - PR #2375 is the related manual fix by a confused human; PR #2381 is the automated fix under test.
- [ ] **Confirmed clear user stories and understood. Understand the value and customer use cases.** -- The primary user is a developer who invokes `/fs-code` and relies on the status comment as their main signal of outcome.
  - When it says "Success" but no PR appears, the user wastes time investigating or retrying blindly.
- [ ] **Confirmed requirements are **testable and unambiguous**.** -- The validation criteria are concrete and testable: status comment must say "Failed" on agent error, "Success" on actual success, and intentional no-ops must remain unaffected.
  - Exit code propagation is deterministic and observable via environment variable.
- [ ] **Ensured acceptance criteria are **defined clearly**.** -- Four acceptance criteria specified in the issue body are verifiable.
  - (1) Status comment says "Failed" on error, (2) includes failure reason, (3) normal success unaffected, (4) intentional no-change unaffected.
- [ ] **Confirmed coverage for NFRs.** -- Non-functional requirements considered.
  - No performance, security, or scalability impact — changes are confined to exit-code propagation and conditional string formatting in shell scripts.

#### **I.2 - Known Limitations**

- The fix detects agent errors solely via non-zero exit code. If the agent exits 0 but logs `is_error: true` in its JSONL output, the status comment will still report success. Parsing JSONL output for error status is out of scope for this change.
- The error comment message is generic ("Code agent failed") and does not extract or display the specific error reason (e.g., "429 RESOURCE_EXHAUSTED") from agent logs. This is noted as a future enhancement opportunity.

#### **I.3 - Technology and Design Review**

- [ ] **Developer handoff completed.** -- PR #2381 provides clear description of root cause (variable scope ordering in Go defers) and the fix approach.
  - Root cause: `lastExitCode` was declared after the post-script defer closure, making it invisible to the closure.
- [ ] **Technology challenges reviewed.** -- No new technologies introduced.
  - The fix uses standard Go variable scoping and shell environment variables.
- [ ] **Test environment needs assessed.** -- Shell-based tests in `post-code-test.sh` are self-contained; Go unit tests exist for `runAgent`.
  - 13 existing test functions exercise `runAgent` (confirmed via LSP `incomingCalls`).
- [ ] **API extensions reviewed.** -- One new environment variable interface: `AGENT_EXIT_CODE` passed from Go runner to post-script.
  - This is an internal interface, not user-facing API.
- [ ] **Topology and deployment considerations reviewed.** -- No topology impact; the change runs within the existing sandbox execution model.

---

### **II. Software Test Plan (STP)**

#### **II.1 - Scope of Testing**

This test plan covers the accurate reporting of agent execution outcomes in the GitHub issue status comment. The scope includes: exit code propagation from the Go runner to the post-code shell script, conditional failure detection at both no-op exit points in the post-script, and distinct error comment formatting for agent errors versus post-script errors.

**Testing Goals:**

- **P0:** Verify that agent errors (non-zero exit code) with no commits produce a failure status comment, not a success message
- **P0:** Verify that normal successful runs and intentional no-change runs remain unaffected by the fix
- **P1:** Verify that the error comment content distinguishes agent errors from post-script errors
- **P1:** Verify that exit code propagation works correctly across the Go-to-shell boundary
- **P2:** Verify edge cases such as detached HEAD state and non-standard exit codes

**Out of Scope (Testing Scope Exclusions):**

- [ ] **JSONL output parsing for `is_error` status** -- Out of scope per issue description; the fix relies solely on exit code. Agreement: N/A
- [ ] **Status comment update mechanism (statuscomment.Notifier)** -- The notifier infrastructure is pre-existing and well-tested (11 references in run_test.go). Agreement: N/A
- [ ] **Vertex AI / LLM API error simulation** -- The specific API error that triggers agent failure is upstream; we test the exit-code-based detection, not the error source. Agreement: N/A
- [ ] **GitHub Actions workflow integration testing** -- The post-code.sh script is tested via shell unit tests, not via full workflow runs. Agreement: N/A

#### **II.2 - Test Strategy**

**Functional**

- [x] **Functional Testing** -- Core functionality: exit code propagation, failure detection at both no-op exit points, error comment content branching.
  - Applicable: Yes. Primary focus of this change.
- [x] **Automation Testing** -- All test scenarios are automated via shell tests (`post-code-test.sh`) and Go unit tests (`run_test.go`).
  - Applicable: Yes. No manual testing required.
- [x] **Regression Testing** -- Verify existing no-op and success paths are unaffected by the new exit code checks.
  - Applicable: Yes. Critical to ensure no false-failure reports.
- [ ] **Upgrade Testing** -- Not applicable; no versioned API or data migration involved.
  - Applicable: No.

**Non-Functional**

- [ ] **Performance Testing** -- Not applicable; exit code check is a trivial integer comparison.
  - Applicable: No.
- [ ] **Scale Testing** -- Not applicable; single-execution path, no scaling dimension.
  - Applicable: No.
- [ ] **Security Testing** -- Not applicable; no new authentication, authorization, or data handling.
  - Applicable: No.
- [ ] **Usability Testing** -- Not applicable; the fix improves usability of status comments but requires no UX testing.
  - Applicable: No.
- [ ] **Monitoring** -- Not applicable; no new metrics or alerts introduced.
  - Applicable: No.

**Integration & Compatibility**

- [ ] **Compatibility Testing** -- Not applicable; internal interface only.
  - Applicable: No.
- [ ] **Dependencies** -- No new external dependencies. `AGENT_EXIT_CODE` is an internal env var contract.
  - Applicable: No.
- [ ] **Cross Integrations** -- Not applicable; change is self-contained within runner and post-script.
  - Applicable: No.

**Infrastructure**

- [ ] **Cloud Testing** -- Not applicable; no cloud-provider-specific behavior.
  - Applicable: No.

#### **II.3 - Test Environment**

- **Cluster Topology:** N/A — no cluster required; tests run in shell and Go test harness
- **Platform Version:** Go 1.22+ (module requirement from go.mod)
- **CPU Virtualization:** N/A
- **Compute:** Standard CI runner
- **Special Hardware:** None
- **Storage:** Local filesystem only
- **Network:** N/A — no network interactions in test scope
- **Operators:** N/A
- **Platform:** Linux (bash 4+) for shell tests; any Go-supported platform for unit tests
- **Special Configs:** `AGENT_EXIT_CODE` environment variable must be injectable in test harness

#### **II.3.1 - Testing Tools & Frameworks**

No new or special tools required. Standard Go testing framework and bash shell tests are used.

#### **II.4 - Entry Criteria**

- [ ] PR #2381 is merged or branch `agent/2378-status-comment-agent-error` is available for testing
- [ ] Go module dependencies are resolved (`go mod download`)
- [ ] Shell test file `post-code-test.sh` is executable
- [ ] `AGENT_EXIT_CODE` environment variable is supported by the runner (run.go change deployed)

#### **II.5 - Risks**

- [ ] **Timeline**
  - *Risk:* Low — fix is small (3 files, ~123 additions) and well-scoped
  - *Mitigation:* Shell tests already included in PR #2381
  - *Status:* [ ] Resolved
- [ ] **Coverage**
  - *Risk:* Medium — agents that exit 0 but fail internally (JSONL `is_error: true` without non-zero exit) will still report success
  - *Mitigation:* Documented as known limitation; future enhancement to parse JSONL output
  - *Status:* [ ] Accepted
- [ ] **Environment**
  - *Risk:* Low — tests are self-contained shell and Go unit tests
  - *Mitigation:* No special environment dependencies
  - *Status:* [ ] Resolved
- [ ] **Untestable**
  - *Risk:* Low — all code paths are deterministic and testable via exit code injection
  - *Mitigation:* Shell test helper `detect_noop` mirrors production logic
  - *Status:* [ ] Resolved
- [ ] **Resources**
  - *Risk:* Low — no special resources or access required
  - *Mitigation:* Standard CI runner sufficient
  - *Status:* [ ] Resolved
- [ ] **Dependencies**
  - *Risk:* Low — the `lastExitCode` variable move in run.go requires careful scoping but has no external dependency
  - *Mitigation:* Existing 13 `runAgent` test functions provide regression safety net
  - *Status:* [ ] Resolved
- [ ] **Other**
  - *Risk:* The `AGENT_EXIT_CODE` env var is a new implicit contract between the Go runner and all post-scripts. If other post-scripts (e.g., post-triage.sh, post-review.sh) are added, they must be aware of this variable.
  - *Mitigation:* Variable defaults to "0" when unset (`${AGENT_EXIT_CODE:-0}`) so unaware scripts are unaffected
  - *Status:* [ ] Resolved

---

### **III. Test Scenarios & Traceability**

#### **III.1 - Requirements-to-Tests Mapping**

- **[GH-2378]** -- Agent exit code is propagated from Go runner to post-script via AGENT_EXIT_CODE environment variable
  - *Test Scenario:* Verify AGENT_EXIT_CODE env var is set on post-script command [Functional]
  - *Priority:* P0
  - *Evidence:* LSP analysis — `runAgent` (run.go:120) sets `AGENT_EXIT_CODE` via `fmt.Sprintf` at line 543; `lastExitCode` moved before defer at line 516

- **[GH-2378]** -- Status comment reports failure when agent errors on main/detached HEAD with no branch created
  - *Test Scenario:* Verify failure exit when agent exit code is non-zero and no feature branch exists [Functional]
  - *Priority:* P0
  - *Test Scenario:* Verify noop exit when agent exit code is zero and no feature branch exists [Functional]
  - *Priority:* P0
  - *Test Scenario:* Verify error on detached HEAD with non-zero exit code [Functional]
  - *Priority:* P1
  - *Evidence:* post-code.sh lines 116-120 — new `AGENT_EXIT_CODE` check in branch validation section

- **[GH-2378]** -- Status comment reports failure when agent errors on feature branch with no changed files
  - *Test Scenario:* Verify failure exit when agent exit code is non-zero and no files changed [Functional]
  - *Priority:* P0
  - *Test Scenario:* Verify noop exit when agent exit code is zero and no files changed [Functional]
  - *Priority:* P0
  - *Evidence:* post-code.sh lines 141-145 — new `AGENT_EXIT_CODE` check in changed-files section

- **[GH-2378]** -- Error comment distinguishes agent errors from post-script errors
  - *Test Scenario:* Verify agent error comment says "Code agent failed" with agent exit code [Functional]
  - *Priority:* P1
  - *Test Scenario:* Verify agent error comment does not say "Post-code script failed" [Functional]
  - *Priority:* P1
  - *Test Scenario:* Verify non-agent errors still produce "Post-code script failed" message [Functional]
  - *Priority:* P1
  - *Evidence:* `report_failure_to_issue()` in post-code.sh branches on `AGENT_ERROR_EXIT` flag; `build_error_comment()` in test file mirrors this logic

- **[GH-2378]** -- Agent error with changes produced still proceeds to PR creation
  - *Test Scenario:* Verify changes are processed normally even when agent exit code is non-zero [Functional]
  - *Priority:* P1
  - *Test Scenario:* Verify error exit code does not block PR creation when commits exist [Functional]
  - *Priority:* P1
  - *Evidence:* `detect_noop` test case "proceed-agent-failed-with-changes" — changes take precedence over error exit code

- **[GH-2378]** -- Existing successful run behavior is preserved (regression)
  - *Test Scenario:* Verify successful agent run with commits still reports success [Functional]
  - *Priority:* P0
  - *Test Scenario:* Verify successful agent run with no changes reports noop correctly [Functional]
  - *Priority:* P0
  - *Evidence:* LSP analysis — 13 existing `runAgent` test functions in run_test.go provide regression coverage; `detect_noop` tests confirm existing noop paths unchanged

- **[GH-2378]** -- End-to-end agent error flow from runner through status comment
  - *Test Scenario:* Verify complete error flow from agent exit through post-script failure to issue comment [End-to-End]
  - *Priority:* P1
  - *Evidence:* Integration path: `runAgent` (run.go:120) → defer closure (line 525) → `AGENT_EXIT_CODE` env → post-code.sh → `report_failure_to_issue()` → `gh issue comment`

---

### **IV. Sign-off and Approval**

| Role | Name | Date | Approval |
|:-----|:-----|:-----|:---------|
| QE Lead | | | [ ] |
| Dev Lead | | | [ ] |
| PM | | | [ ] |
