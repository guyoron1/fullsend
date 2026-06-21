# FullSend Test Plan

## **Enrollment: Bounded Timeout for Repo-Maintenance Workflow Activation - Quality Engineering Plan**

### **Metadata & Tracking**

- **Enhancement(s):** [GH-2354](https://github.com/fullsend-ai/fullsend/issues/2354)
- **Feature Tracking:** N/A (standalone issue)
- **Epic Tracking:** N/A (standalone issue)
- **QE Owner(s):** TBD
- **Owning SIG:** N/A
- **Participating SIGs:** None

**Document Conventions (if applicable):** N/A

### **Feature Overview**

The enrollment install flow dispatches a repo-maintenance workflow via the GitHub API and polls for its completion. When GitHub is slow to register or execute workflows, the chained polling and retry loops in `awaitWorkflowRun` can block the CLI for extended periods. This feature addresses the need for bounded, predictable timeouts with exponential backoff and actionable user feedback during the enrollment polling phases, affecting both install and uninstall operations in `internal/layers/enrollment.go`.

---

### **I. Motivation and Requirements Review (QE Review Guidelines)**

This section documents the mandatory QE review process. The goal is to understand the feature's value,
technology, and testability before formal test planning.

#### **1. Requirement & User Story Review Checklist**

- [ ] **Review Requirements**
  - Reviewed the relevant requirements.
  - GH-2354 describes the problem: serial polling loops (`awaitWorkflowRegistration` + `dispatchRepoMaintenanceWithRetry` + `awaitWorkflowRun`) can block 10+ minutes when GitHub is slow.
  - Triage summary identifies root cause as sequential blocking polls with fixed retry counts and no early termination.
- [ ] **Understand Value and Customer Use Cases**
  - Confirmed clear user stories and understood.
  - Understand the difference between community and product requirements.
  - **What is the value of the feature for customers**.
  - Ensured requirements contain relevant **customer use cases**.
  - Every new repo onboarding encounters the enrollment flow; 10+ minute silent waits degrade UX for all users adopting FullSend.
- [ ] **Testability**
  - Confirmed requirements are **testable and unambiguous**.
  - Timeout bounds, backoff intervals, and progress messages are directly observable via `forge.FakeClient` and `ui.Printer` buffer output in unit/functional tests.
- [ ] **Acceptance Criteria**
  - Ensured acceptance criteria are **defined clearly** (clear user stories; product requirements clearly defined in Jira).
  - Issue states: install should fail fast with actionable guidance or complete within a bounded, predictable time without long silent waits.
- [ ] **Non-Functional Requirements (NFRs)**
  - Confirmed coverage for NFRs, including Performance, Security, Usability, Downtime, Connectivity, Monitoring (alerts/metrics), Scalability, Portability (e.g., cloud support), and Docs.
  - Primary NFR is CLI responsiveness and user experience during enrollment wait. No security, scalability, or monitoring NFRs identified.

#### **2. Known Limitations**

- The bounded timeout (`enrollmentWaitTimeout = 3 min`) and exponential backoff (`enrollmentPollInitial = 2s`, `enrollmentPollMax = 15s`) were introduced in PR #1954. This STP provides regression test coverage to ensure these safeguards are not inadvertently weakened or removed in future changes, and validates that the current behavior meets the requirements described in GH-2354.
- Actual GitHub workflow registration latency is outside FullSend's control; tests can only validate timeout behavior, not real registration speed.
- No `--no-wait` flag exists yet to dispatch and return immediately without polling.

#### **3. Technology and Design Review**

- [ ] **Developer Handoff/QE Kickoff**
  - A meeting where Dev/Arch walked QE through the design, architecture, and implementation details. **Critical for identifying untestable aspects early.**
  - PR #1954 review raised this issue. The enrollment layer (`internal/layers/enrollment.go`) uses `forge.Client` interface for all GitHub API interactions, enabling full mock-based testing via `forge.FakeClient`.
- [ ] **Technology Challenges**
  - Identified potential testing challenges related to the underlying technology.
  - Testing time-dependent behavior (polling intervals, timeouts) requires careful test design to avoid flaky time-sensitive assertions.
- [ ] **Test Environment Needs**
  - Determined necessary **test environment setups and tools**.
  - All tests run with `go test` using `forge.FakeClient` mock; no cluster or GitHub API access required.
- [ ] **API Extensions**
  - Reviewed new or modified APIs and their impact on testing.
  - `forge.Client` interface methods used: `DispatchWorkflow`, `ListWorkflowRuns`, `GetWorkflowRunLogs`, `ListRepoPullRequests`. No new API methods introduced.
- [ ] **Topology Considerations**
  - Evaluated multi-cluster, network topology, and architectural impacts.
  - N/A. Enrollment layer is a CLI component with no cluster or network topology dependencies.

### **II. Software Test Plan (STP)**

This STP serves as the **overall roadmap for testing**, detailing the scope, approach, resources, and schedule.

#### **1. Scope of Testing**

Testing will validate that the enrollment install and uninstall flows complete or fail within bounded, predictable timeouts, use exponential backoff for polling, provide progress feedback, handle user interruption gracefully, and produce actionable error messages on timeout or dispatch failure.

**Testing Goals**

**Functional Goals**

- **P0:** Verify enrollment install completes within timeout bound or fails with actionable error
- **P0:** Verify happy-path enrollment completes without regression when workflow registers quickly
- **P1:** Verify exponential backoff polling behavior (interval doubling, cap at maximum)
- **P1:** Verify progress messages are emitted with elapsed time during polling phases
- **P1:** Verify user interruption (Ctrl+C) stops enrollment cleanly without error

**Quality Goals**

- **P1:** Verify timeout error messages include manual recovery guidance
- **P1:** Verify dispatch failure returns descriptive error without blocking install

**Integration Goals**

- **P2:** Verify unenrollment uses same bounded timeout and backoff as enrollment

**Out of Scope (Testing Scope Exclusions)**

- [ ] GitHub Actions workflow registration latency -- *Rationale:* Platform-level concern managed by GitHub, not FullSend -- *PM/Lead Agreement:* TBD
- [ ] GitHub API rate limiting during polling -- *Rationale:* Infrastructure-level concern; FullSend relies on standard GitHub API behavior -- *PM/Lead Agreement:* TBD
- [ ] `--no-wait` flag implementation -- *Rationale:* Suggested improvement not yet implemented; out of scope for current testing -- *PM/Lead Agreement:* TBD
- [ ] Admin CLI dispatch timeout behavior -- *Rationale:* `admin.go` uses `DispatchWorkflow` but enrollment timeout constants are scoped to the enrollment layer; admin dispatch has its own timeout semantics -- *PM/Lead Agreement:* TBD

#### **2. Test Strategy**

**Functional**

- [ ] **Functional Testing** -- Validates that the feature works according to specified requirements and user stories
  - *Details:* Applicable. Core testing of timeout bounds, backoff behavior, progress output, user interruption handling, and error reporting using `forge.FakeClient` mocks.
- [ ] **Automation Testing** -- Confirms test automation plan is in place for CI and regression coverage (all tests are expected to be automated)
  - *Details:* Applicable. All tests are Go unit/functional tests runnable via `go test ./internal/layers/...` in CI.
- [ ] **Regression Testing** -- Verifies that new changes do not break existing functionality
  - *Details:* Applicable. Existing enrollment tests (`enrollment_test.go`) cover happy path, dispatch error, context cancellation, and workflow warning. New tests extend this coverage.

**Non-Functional**

- [ ] **Performance Testing** -- Validates feature performance meets requirements (latency, throughput, resource usage)
  - *Details:* Not applicable. Timeout values are configuration constants, not runtime performance targets.
- [ ] **Scale Testing** -- Validates feature behavior under increased load and at production-like scale
  - *Details:* Not applicable. Enrollment operates on a single workflow dispatch per install/uninstall invocation.
- [ ] **Security Testing** -- Verifies security requirements, RBAC, authentication, authorization, and vulnerability scanning
  - *Details:* Not applicable. Enrollment uses existing forge.Client authentication; no new security surface.
- [ ] **Usability Testing** -- Validates user experience and accessibility requirements
  - *Details:* Partially applicable. Progress messages and actionable error guidance are UX improvements validated through functional tests.
- [ ] **Monitoring** -- Does the feature require metrics and/or alerts?
  - *Details:* Not applicable. No new metrics or alerts required for enrollment timeout behavior.

**Integration & Compatibility**

- [ ] **Compatibility Testing** -- Ensures feature works across supported platforms, versions, and configurations
  - *Details:* Not applicable. Enrollment layer is Go code with no platform-specific behavior.
- [ ] **Upgrade Testing** -- Validates upgrade paths from previous versions, data migration, and configuration preservation
  - *Details:* Not applicable. Timeout constants are internal; no user configuration to migrate.
- [ ] **Dependencies** -- Blocked by deliverables from other components/products
  - *Details:* No blocking dependencies. `forge.Client` interface is stable and mockable.
- [ ] **Cross Integrations** -- Does the feature affect other features or require testing by other teams?
  - *Details:* `awaitWorkflowRun` is shared between Install and Uninstall. `DispatchWorkflow` is also called from `internal/cli/admin.go`. Changes to timeout constants affect both code paths.

**Infrastructure**

- [ ] **Cloud Testing** -- Does the feature require multi-cloud platform testing?
  - *Details:* Not applicable. Enrollment is a CLI feature independent of cloud platform.

#### **3. Test Environment**

- **Cluster Topology:** N/A (CLI unit/functional tests, no cluster required)
- **Platform & Product Version(s):** Go 1.23+, FullSend 0.x
- **CPU Virtualization:** N/A
- **Compute Resources:** Standard CI runner
- **Special Hardware:** None required
- **Storage:** N/A
- **Network:** N/A (all forge API calls are mocked)
- **Required Operators:** None
- **Platform:** GitHub Actions (CI execution)
- **Special Configurations:** None

#### **3.1. Testing Tools & Frameworks**

No additional tools required beyond the project's standard test infrastructure.

#### **4. Entry Criteria**

The following conditions must be met before testing can begin:

- [ ] Requirements and design documents are **approved and merged**
- [ ] Test environment can be **set up and configured** (see Section II.3 - Test Environment)
- [ ] `forge.FakeClient` supports configurable workflow run responses (already implemented)
- [ ] `enrollment.go` timeout and backoff constants are accessible for test assertions

#### **5. Risks**

- [ ] **Timeline/Schedule**
  - Risk: Timeout behavior changes may be deprioritized if the current 3-minute bound is deemed acceptable
  - Mitigation: Tests validate current behavior to prevent regression; future improvements build on existing test coverage
- [ ] **Test Coverage**
  - Risk: Time-dependent tests may not fully exercise real-world slow registration scenarios
  - Mitigation: Use `forge.FakeClient` with configurable delays to simulate slow responses without real-time waits
- [ ] **Untestable Aspects**
  - Risk: Actual GitHub workflow registration latency cannot be controlled in tests
  - Mitigation: Tests validate timeout and backoff behavior independent of real GitHub API latency
- [ ] **Dependencies**
  - Risk: Changes to `forge.Client` interface could break test mocks
  - Mitigation: `forge.FakeClient` is maintained alongside the interface; compile-time checks ensure compatibility

---

### **III. Test Scenarios & Traceability**

This section links requirements to test coverage, enabling reviewers to verify all requirements are tested.

#### **1. Requirements-to-Tests Mapping**

- **[GH-2354]** -- Enrollment install completes or fails within a bounded, predictable timeout
  - *Test Scenario:* Verify enrollment completes within timeout bound
  - *Test Type:* [Functional]
  - *Priority:* P0

- **[GH-2354]** -- Enrollment install completes or fails within a bounded, predictable timeout
  - *Test Scenario:* Verify timeout returns actionable error message
  - *Test Type:* [Functional]
  - *Priority:* P0

- **[GH-2354]** -- Enrollment install completes or fails within a bounded, predictable timeout
  - *Test Scenario:* Verify timeout behavior with slow workflow registration
  - *Test Type:* [Functional]
  - *Priority:* P0

- **[GH-2354]** -- Enrollment polling uses exponential backoff to avoid excessive API calls
  - *Test Scenario:* Verify wait time between status updates increases progressively
  - *Test Type:* [Functional]
  - *Priority:* P1

- **[GH-2354]** -- Enrollment polling uses exponential backoff to avoid excessive API calls
  - *Test Scenario:* Verify retry wait time does not exceed maximum bound
  - *Test Type:* [Functional]
  - *Priority:* P1

- **[GH-2354]** -- Enrollment polling uses exponential backoff to avoid excessive API calls
  - *Test Scenario:* Verify first retry occurs within expected timeframe
  - *Test Type:* [Functional]
  - *Priority:* P1

- **[GH-2354]** -- Enrollment provides progress feedback during each polling phase
  - *Test Scenario:* Verify progress messages emitted during polling
  - *Test Type:* [Functional]
  - *Priority:* P1

- **[GH-2354]** -- Enrollment provides progress feedback during each polling phase
  - *Test Scenario:* Verify elapsed time reported in status updates
  - *Test Type:* [Functional]
  - *Priority:* P1

- **[GH-2354]** -- Enrollment install succeeds within expected time when workflow registers quickly
  - *Test Scenario:* Verify fast enrollment completes without delay
  - *Test Type:* [Functional]
  - *Priority:* P0

- **[GH-2354]** -- Enrollment install succeeds within expected time when workflow registers quickly
  - *Test Scenario:* Verify enrollment reports success and workflow URL
  - *Test Type:* [Functional]
  - *Priority:* P0

- **[GH-2354]** -- Enrollment install succeeds within expected time when workflow registers quickly
  - *Test Scenario:* Verify enrollment reports reconciliation PRs
  - *Test Type:* [Functional]
  - *Priority:* P0

- **[GH-2354]** -- Enrollment timeout produces actionable guidance for manual recovery
  - *Test Scenario:* Verify error includes manual check guidance
  - *Test Type:* [Functional]
  - *Priority:* P1

- **[GH-2354]** -- Enrollment timeout produces actionable guidance for manual recovery
  - *Test Scenario:* Verify error includes elapsed time duration
  - *Test Type:* [Functional]
  - *Priority:* P1

- **[GH-2354]** -- Enrollment handles user interruption gracefully during polling
  - *Test Scenario:* Verify user interruption stops enrollment polling
  - *Test Type:* [Functional]
  - *Priority:* P1

- **[GH-2354]** -- Enrollment handles user interruption gracefully during polling
  - *Test Scenario:* Verify interruption treated as non-fatal
  - *Test Type:* [Functional]
  - *Priority:* P1

- **[GH-2354]** -- Enrollment handles user interruption gracefully during polling
  - *Test Scenario:* Verify CLI exits cleanly after interruption with no hanging processes
  - *Test Type:* [Functional]
  - *Priority:* P1

- **[GH-2354]** -- Enrollment unenrollment workflow uses same bounded timeout and backoff
  - *Test Scenario:* Verify unenrollment uses bounded timeout
  - *Test Type:* [Functional]
  - *Priority:* P2

- **[GH-2354]** -- Enrollment unenrollment workflow uses same bounded timeout and backoff
  - *Test Scenario:* Verify unenrollment backoff matches enrollment
  - *Test Type:* [Functional]
  - *Priority:* P2

- **[GH-2354]** -- Enrollment workflow dispatch failure is reported clearly
  - *Test Scenario:* Verify dispatch failure returns descriptive error
  - *Test Type:* [Functional]
  - *Priority:* P1

- **[GH-2354]** -- Enrollment workflow dispatch failure is reported clearly
  - *Test Scenario:* Verify dispatch error does not block install
  - *Test Type:* [Functional]
  - *Priority:* P1

- **[GH-2354]** -- Enrollment workflow dispatch failure is reported clearly
  - *Test Scenario:* Verify dispatch error during concurrent operations
  - *Test Type:* [Functional]
  - *Priority:* P1

---

### **IV. Sign-off and Approval**

This Software Test Plan requires approval from the following stakeholders:

* **Reviewers:**
  - [TBD / @tbd]
* **Approvers:**
  - [TBD / @tbd]
