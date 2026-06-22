# Test Plan

## **Bound Enrollment Wait with Timeout and Backoff - Quality Engineering Plan**

### **Metadata & Tracking**

- **Enhancement(s):** [GH-76](https://github.com/guyoron1/fullsend/pull/76) (mirror of [fullsend-ai/fullsend#2359](https://github.com/fullsend-ai/fullsend/pull/2359))
- **Feature Tracking:** [GH-76](https://github.com/guyoron1/fullsend/pull/76)
- **Epic Tracking:** Issue #2354
- **QE Owner(s):** TBD
- **Owning SIG:** N/A
- **Participating SIGs:** None

**Document Conventions (if applicable):** N/A

### **Feature Overview**

This change replaces the hardcoded 36-iteration fixed-interval polling loop in the enrollment layer's `awaitWorkflowRun` with a time-bounded loop using exponential backoff. The total wait is capped at 3 minutes (matching the previous maximum), but polling starts at 2-second intervals and doubles up to 15 seconds, reducing API calls and giving faster feedback when the workflow completes quickly. Additionally, the status comment authentication is migrated from the deprecated `--status-token` (static token) to `--mint-url` (OIDC mint-based), and CI workflow files are updated to pass the new parameter.

---

### **I. Motivation and Requirements Review (QE Review Guidelines)**

This section documents the mandatory QE review process. The goal is to understand the feature's value,
technology, and testability before formal test planning.

#### **1. Requirement & User Story Review Checklist**

- [ ] **Review Requirements**
  - Reviewed the relevant requirements.
  - PR #76 and upstream PR #2359 describe the motivation: the previous fixed-interval polling loop (36 × 5s = 3min) was inefficient, making excessive API calls and providing slow initial feedback.
  - Issue #2354 tracks the original request to bound enrollment wait.

- [ ] **Understand Value and Customer Use Cases**
  - Confirmed clear user stories and understood.
  - Understand the difference between community and product requirements.
  - **What is the value of the feature for customers**.
  - Ensured requirements contain relevant **customer use cases**.
  - Operators running `fullsend install` benefit from faster enrollment feedback when workflows complete quickly, and reduced GitHub API rate limit consumption due to exponential backoff.

- [ ] **Testability**
  - Confirmed requirements are **testable and unambiguous**.
  - All changes are in pure Go functions with injectable dependencies (`forge.Client`, `ui.Printer`), making them fully testable with mocks. The `nextInterval` function is a pure function with deterministic output.

- [ ] **Acceptance Criteria**
  - Ensured acceptance criteria are **defined clearly** (clear user stories; product requirements clearly defined in Jira).
  - Acceptance criteria from upstream PR: (1) polling starts at 2s and doubles to 15s cap, (2) total wait bounded at 3 minutes, (3) progress messages show elapsed time, (4) timeout error includes actionable guidance.

- [ ] **Non-Functional Requirements (NFRs)**
  - Confirmed coverage for NFRs, including Performance, Security, Usability, Downtime, Connectivity, Monitoring (alerts/metrics), Scalability, Portability (e.g., cloud support), and Docs.
  - Performance: exponential backoff reduces API calls from ~36 to ~10-12 per enrollment wait. Security: migration from static tokens to OIDC mint improves token lifecycle management.

#### **2. Known Limitations**

- Exponential backoff may cause slower detection of workflow completion during the later phases of the wait (up to 15s delay between checks vs. the previous fixed 5s).
- The `--status-token` flag is deprecated but still functional for backward compatibility; it will be removed in a future release.
- The 3-minute total timeout is fixed and not configurable by the operator.

#### **3. Technology and Design Review**

- [ ] **Developer Handoff/QE Kickoff**
  - A meeting where Dev/Arch walked QE through the design, architecture, and implementation details. **Critical for identifying untestable aspects early.**
  - The change is self-contained in `internal/layers/enrollment.go` with a new `nextInterval` helper function. The `awaitWorkflowRun` method is called from both `Install` and `Uninstall` paths.

- [ ] **Technology Challenges**
  - Identified potential testing challenges related to the underlying technology.
  - Time-dependent behavior (backoff intervals, deadline-based loop) requires careful test design with controllable time sources or short timeouts.

- [ ] **Test Environment Needs**
  - Determined necessary **test environment setups and tools**.
  - Standard Go test environment with mocked `forge.Client` interface. No special infrastructure required.

- [ ] **API Extensions**
  - Reviewed new or modified APIs and their impact on testing.
  - CLI flag changes: `--mint-url` added to `reconcile-status` and `run` commands; `--status-token` deprecated. CI workflow parameter changed from `status-token` to `mint-url`.

- [ ] **Topology Considerations**
  - Evaluated multi-cluster, network topology, and architectural impacts.
  - No topology-specific impacts. Changes are CLI-level and apply uniformly across all deployment topologies.

### **II. Software Test Plan (STP)**

This STP serves as the **overall roadmap for testing**, detailing the scope, approach, resources, and schedule.

#### **1. Scope of Testing**

Testing covers the enrollment wait timeout/backoff behavior in `internal/layers/enrollment.go`, the `--mint-url` authentication migration in `internal/cli/reconcilestatus.go` and `internal/cli/run.go`, and the orphaned status comment reconciliation in `internal/statuscomment/statuscomment.go`. CI workflow parameter changes across 5 reusable workflow files are also in scope.

**Testing Goals**

**Functional Goals:**
- **P0:** Verify enrollment wait uses exponential backoff (2s→4s→8s→15s) and times out after 3 minutes with an actionable error message
- **P0:** Verify the `nextInterval` function correctly doubles intervals and caps at 15s
- **P1:** Verify context cancellation interrupts the enrollment wait promptly
- **P1:** Verify both Install and Uninstall enrollment paths use the bounded wait

**Quality Goals:**
- **P1:** Verify the `--mint-url` authentication flow works for reconcile-status and run commands
- **P1:** Verify orphaned status comment reconciliation handles terminated and cancelled reasons correctly

**Integration Goals:**
- **P1:** Verify CI workflows correctly pass `mint-url` parameter instead of deprecated `status-token`
- **P2:** Verify backward compatibility of deprecated `--status-token` flag with warning

**Out of Scope (Testing Scope Exclusions)**

- [ ] **GitHub Actions workflow dispatch and scheduling reliability** -- *Rationale:* Platform-level infrastructure tested by GitHub; fullsend tests its own dispatch calls via mocked forge.Client -- *PM/Lead Agreement:* TBD
- [ ] **OIDC token exchange with cloud identity providers** -- *Rationale:* Infrastructure-level concern; fullsend tests the mintclient call interface, not the underlying OIDC flow -- *PM/Lead Agreement:* TBD
- [ ] **End-to-end enrollment with real GitHub workflows** -- *Rationale:* Requires live GitHub org with configured repo-maintenance workflow; covered by existing e2e suite -- *PM/Lead Agreement:* TBD

#### **2. Test Strategy**

**Functional**

- [x] **Functional Testing** — Validates that the feature works according to specified requirements and user stories
  - *Details:* Unit tests for `awaitWorkflowRun`, `nextInterval`, `setupStatusNotifier`, `ReconcileOrphaned`, and CLI flag parsing. All use mocked dependencies.
- [x] **Automation Testing** — Confirms test automation plan is in place for CI and regression coverage (all tests are expected to be automated)
  - *Details:* All tests are Go unit tests running in CI via `go test`. New test files include `qf_enrollment_test.go`, `qf_reconcilestatus_test.go`, and `qf_statuscomment_test.go`.
- [x] **Regression Testing** — Verifies that new changes do not break existing functionality
  - *Details:* LSP analysis confirms `awaitWorkflowRun` is called from `Install` (line 98) and `Uninstall` (line 286). Existing `enrollment_test.go`, `run_test.go`, and `statuscomment_test.go` cover regression paths.

**Non-Functional**

- [ ] **Performance Testing** — Validates feature performance meets requirements (latency, throughput, resource usage)
  - *Details:* N/A — backoff behavior is validated functionally; no performance benchmarks required for polling intervals.
- [ ] **Scale Testing** — Validates feature behavior under increased load and at production-like scale
  - *Details:* N/A — enrollment is a single-repo operation, not a scale concern.
- [ ] **Security Testing** — Verifies security requirements, RBAC, authentication, authorization, and vulnerability scanning
  - *Details:* Migration from static token to OIDC mint improves security posture. Tests verify `--mint-url` authentication flow and that deprecated `--token` emits a warning.
- [ ] **Usability Testing** — Validates user experience and accessibility requirements
  - *Details:* N/A — CLI output changes (elapsed time format) are covered by functional tests.
- [ ] **Monitoring** — Does the feature require metrics and/or alerts?
  - *Details:* N/A — no new metrics or alerts introduced.

**Integration & Compatibility**

- [ ] **Compatibility Testing** — Ensures feature works across supported platforms, versions, and configurations
  - *Details:* CI workflow parameter change (`status-token` → `mint-url`) must be coordinated with all 5 reusable workflow files.
- [ ] **Upgrade Testing** — Validates upgrade paths from previous versions, data migration, and configuration preservation
  - *Details:* N/A — no persistent state changes; CLI flag deprecation provides backward compatibility.
- [ ] **Dependencies** — Blocked by deliverables from other components/products
  - *Details:* Depends on `mintclient` package for OIDC token minting. Already available in the codebase.
- [ ] **Cross Integrations** — Does the feature affect other features or require testing by other teams?
  - *Details:* Status comment system is used by all agent types (triage, coder, review, fix, retro). The auth migration affects all CI workflows.

**Infrastructure**

- [ ] **Cloud Testing** — Does the feature require multi-cloud platform testing?
  - *Details:* N/A — changes are platform-agnostic CLI/library code.

#### **3. Test Environment**

- **Cluster Topology:** N/A (unit tests only, no cluster required)
- **Platform & Product Version(s):** Go 1.22+, fullsend CLI
- **CPU Virtualization:** N/A
- **Compute Resources:** Standard CI runner
- **Special Hardware:** None
- **Storage:** N/A
- **Network:** N/A (mocked HTTP calls)
- **Required Operators:** None
- **Platform:** Linux (CI), macOS (local development)
- **Special Configurations:** None

#### **3.1. Testing Tools & Frameworks**

- **Test Framework:** Go standard `testing` package with `testify` assertions (standard tooling, not new)
- **CI/CD:** Standard CI pipeline (not new)
- **Other Tools:** None

#### **4. Entry Criteria**

The following conditions must be met before testing can begin:

- [ ] Requirements and design documents are **approved and merged**
- [ ] Test environment can be **set up and configured** (see Section II.3 - Test Environment)
- [ ] PR #76 changes are available on the test branch
- [ ] `mintclient` package is functional and accessible

#### **5. Risks**

- [ ] **Timeline/Schedule**
  - Risk: N/A — changes are self-contained and do not depend on external timelines.
  - Mitigation: N/A

- [ ] **Test Coverage**
  - Risk: Time-dependent behavior (backoff intervals, deadline loop) may be difficult to test deterministically without flaky timing issues.
  - Mitigation: Use short timeouts in tests (e.g., 100ms instead of 3min) and mock `time.After` behavior via context cancellation.

- [ ] **Test Environment**
  - Risk: N/A — standard Go test environment, no special infrastructure.
  - Mitigation: N/A

- [ ] **Untestable Aspects**
  - Risk: Real GitHub API rate limiting behavior under exponential backoff cannot be tested in unit tests.
  - Mitigation: Integration verified by existing e2e test suite; unit tests validate the backoff algorithm in isolation.

- [ ] **Resource Constraints**
  - Risk: N/A — no special resources required.
  - Mitigation: N/A

- [ ] **Dependencies**
  - Risk: `mintclient` API changes could break the new authentication flow.
  - Mitigation: `mintclient` is an internal package with stable interface; tests mock the mint call.

- [ ] **Other**
  - Risk: Deprecated `--status-token` flag removal timeline may cause confusion if not communicated.
  - Mitigation: Deprecation warning is emitted on use; removal planned for a future release with notice.

---

### **III. Test Scenarios & Traceability**

This section links requirements to test coverage, enabling reviewers to verify all requirements are tested.

#### **1. Requirements-to-Tests Mapping**

- **[GH-76]** -- Enrollment wait uses bounded timeout with exponential backoff
  - *Test Scenario:* Verify enrollment wait completes when workflow succeeds quickly
  - *Test Type:* Unit Tests
  - *Priority:* P0

- **[GH-76]** -- Enrollment wait uses bounded timeout with exponential backoff
  - *Test Scenario:* Verify backoff intervals follow 2s→4s→8s→15s progression
  - *Test Type:* Unit Tests
  - *Priority:* P0

- **[GH-76]** -- Enrollment wait uses bounded timeout with exponential backoff
  - *Test Scenario:* Verify wait times out after 3 minutes with actionable error
  - *Test Type:* Unit Tests
  - *Priority:* P0

- **[GH-76]** -- Enrollment wait uses bounded timeout with exponential backoff
  - *Test Scenario:* Verify backoff caps at 15s and does not exceed maximum
  - *Test Type:* Unit Tests
  - *Priority:* P0

- **[GH-76]** -- Enrollment wait times out gracefully with actionable error message
  - *Test Scenario:* Verify timeout error includes guidance to re-run install
  - *Test Type:* Unit Tests
  - *Priority:* P0

- **[GH-76]** -- Enrollment wait times out gracefully with actionable error message
  - *Test Scenario:* Verify timeout reports elapsed time accurately
  - *Test Type:* Unit Tests
  - *Priority:* P0

- **[GH-76]** -- Enrollment wait respects context cancellation during polling
  - *Test Scenario:* Verify context cancellation interrupts wait promptly
  - *Test Type:* Unit Tests
  - *Priority:* P1

- **[GH-76]** -- Enrollment wait respects context cancellation during polling
  - *Test Scenario:* Verify cancellation during backoff sleep exits cleanly
  - *Test Type:* Unit Tests
  - *Priority:* P1

- **[GH-76]** -- Enrollment progress messages report elapsed time
  - *Test Scenario:* Verify progress shows elapsed time format
  - *Test Type:* Unit Tests
  - *Priority:* P2

- **[GH-76]** -- Enrollment Install and Uninstall both use bounded await
  - *Test Scenario:* Verify Install path uses bounded workflow wait
  - *Test Type:* Unit Tests
  - *Priority:* P1

- **[GH-76]** -- Enrollment Install and Uninstall both use bounded await
  - *Test Scenario:* Verify Uninstall path uses bounded workflow wait
  - *Test Type:* Unit Tests
  - *Priority:* P1

- **[GH-76]** -- Enrollment Install and Uninstall both use bounded await
  - *Test Scenario:* Verify await failure is non-fatal for both paths
  - *Test Type:* Unit Tests
  - *Priority:* P1

- **[GH-76]** -- Exponential backoff doubles interval up to configured cap
  - *Test Scenario:* Verify nextInterval doubles current value
  - *Test Type:* Unit Tests
  - *Priority:* P1

- **[GH-76]** -- Exponential backoff doubles interval up to configured cap
  - *Test Scenario:* Verify nextInterval caps at enrollmentPollMax
  - *Test Type:* Unit Tests
  - *Priority:* P1

- **[GH-76]** -- Exponential backoff doubles interval up to configured cap
  - *Test Scenario:* Verify backoff with initial value at cap boundary
  - *Test Type:* Unit Tests
  - *Priority:* P1

- **[GH-76]** -- Status reconciliation uses mint-url for token acquisition
  - *Test Scenario:* Verify reconcile-status authenticates via mint-url
  - *Test Type:* Unit Tests
  - *Priority:* P1

- **[GH-76]** -- Status reconciliation uses mint-url for token acquisition
  - *Test Scenario:* Verify error when neither mint-url nor token provided
  - *Test Type:* Unit Tests
  - *Priority:* P1

- **[GH-76]** -- Status reconciliation uses mint-url for token acquisition
  - *Test Scenario:* Verify deprecated token flag emits warning
  - *Test Type:* Unit Tests
  - *Priority:* P1

- **[GH-76]** -- Run command status notifier migrated to mint-url
  - *Test Scenario:* Verify status notifier uses mint-url from flag
  - *Test Type:* Unit Tests
  - *Priority:* P1

- **[GH-76]** -- Run command status notifier migrated to mint-url
  - *Test Scenario:* Verify status notifier falls back to FULLSEND_MINT_URL env
  - *Test Type:* Unit Tests
  - *Priority:* P1

- **[GH-76]** -- Run command status notifier migrated to mint-url
  - *Test Scenario:* Verify error when no mint source available
  - *Test Type:* Unit Tests
  - *Priority:* P1

- **[GH-76]** -- Orphaned status comments reconciled across termination reasons
  - *Test Scenario:* Verify orphaned started comment updated to interrupted
  - *Test Type:* Unit Tests
  - *Priority:* P1

- **[GH-76]** -- Orphaned status comments reconciled across termination reasons
  - *Test Scenario:* Verify already-terminal comment is skipped
  - *Test Type:* Unit Tests
  - *Priority:* P1

- **[GH-76]** -- Orphaned status comments reconciled across termination reasons
  - *Test Scenario:* Verify cancelled reason produces cancelled label
  - *Test Type:* Unit Tests
  - *Priority:* P1

- **[GH-76]** -- Orphaned status comments reconciled across termination reasons
  - *Test Scenario:* Verify missing comment is not an error
  - *Test Type:* Unit Tests
  - *Priority:* P1

- **[GH-76]** -- CI workflows use mint-url instead of deprecated status-token
  - *Test Scenario:* Verify workflow parameter accepts mint-url
  - *Test Type:* Functional
  - *Priority:* P1

- **[GH-76]** -- CI workflows use mint-url instead of deprecated status-token
  - *Test Scenario:* Verify agent status posting works end-to-end with mint
  - *Test Type:* Functional
  - *Priority:* P1

---

### **IV. Sign-off and Approval**

This Software Test Plan requires approval from the following stakeholders:

* **Reviewers:**
  - [TBD / @reviewer]
  - [TBD / @reviewer]
* **Approvers:**
  - [TBD / @approver]
  - [TBD / @approver]
