# Test Plan

## **Restore Data Consistency Guard in EnsureOrgInMint After ROLE_APP_IDS Migration - Quality Engineering Plan**

### **Metadata & Tracking**

- **Enhancement(s):** [GH-2433](https://github.com/fullsend-ai/fullsend/issues/2433)
- **Feature Tracking:** [GH-2433](https://github.com/fullsend-ai/fullsend/issues/2433)
- **Epic Tracking:** Related: [#1842](https://github.com/fullsend-ai/fullsend/issues/1842) (original bug), [PR #1846](https://github.com/fullsend-ai/fullsend/pull/1846) (original fix), [PR #2331](https://github.com/fullsend-ai/fullsend/pull/2331) (regression)
- **QE Owner(s):** TBD
- **Owning SIG:** N/A
- **Participating SIGs:** N/A

**Document Conventions (if applicable):** N/A

### **Feature Overview**

This bug fix restores a defense-in-depth data consistency guard in `EnsureOrgInMint` (`provisioner.go`) that was removed during the role-only `ROLE_APP_IDS` migration in PR #2331. The guard prevents silent unenrollment of all existing organizations when `ALLOWED_ORGS` is read as empty due to Cloud Run revision divergence or partial failure. The fix adapts the guard for the new role-only model by checking whether `ROLE_APP_IDS` has role-only entries (via `mintcore.RoleOnlyAppIDs`) when `ALLOWED_ORGS` is empty, aborting enrollment if a data inconsistency is detected.

---

### **I. Motivation and Requirements Review (QE Review Guidelines)**

This section documents the mandatory QE review process. The goal is to understand the feature's value,
technology, and testability before formal test planning.

#### **1. Requirement & User Story Review Checklist**

- [ ] **Review Requirements**
  - Reviewed the relevant requirements.
  - Issue GH-2433 provides clear provenance: original bug #1842, original fix PR #1846, regression PR #2331. The proposed fix with code sample is well-defined.
- [ ] **Understand Value and Customer Use Cases**
  - Confirmed clear user stories and understood.
  - Understand the difference between community and product requirements.
  - **What is the value of the feature for customers**.
  - Ensured requirements contain relevant **customer use cases**.
  - Prevents silent data loss where `EnsureOrgInMint` could clobber all enrolled organizations due to stale reads from Cloud Run revisions. This is a critical data integrity protection for multi-org mint deployments.
- [ ] **Testability**
  - Confirmed requirements are **testable and unambiguous**.
  - The guard logic is a pure conditional check on env var state, fully testable with mocked `GCFClient`. All code paths are exercisable in unit tests. Existing test infrastructure (`newFakeGCFClient()`) supports the testing pattern.
- [ ] **Acceptance Criteria**
  - Ensured acceptance criteria are **defined clearly** (clear user stories; product requirements clearly defined in Jira).
  - AC1: When `ALLOWED_ORGS` is empty and `ROLE_APP_IDS` has role-only entries, `EnsureOrgInMint` returns an error containing "data inconsistency". AC2: When both are empty (first enrollment), enrollment proceeds normally. AC3: Legacy org/role keys do not trigger the guard.
- [ ] **Non-Functional Requirements (NFRs)**
  - Confirmed coverage for NFRs, including Performance, Security, Usability, Downtime, Connectivity, Monitoring (alerts/metrics), Scalability, Portability (e.g., cloud support), and Docs.
  - No performance impact: guard adds one JSON unmarshal and a map filter before the existing enrollment logic. No new dependencies. Error message includes actionable diagnostic information (`fullsend mint status --project=`).

#### **2. Known Limitations**

- The guard only detects data inconsistency when `ALLOWED_ORGS` is completely empty. A partially stale read (e.g., missing some orgs but not all) is not detected by this guard and relies on the existing `mergeAllowedOrgs` union logic.
- The `EnsureOrgInMint` read-modify-write pattern remains vulnerable to concurrent enrollment races. The guard does not address concurrent write conflicts.
- If `ROLE_APP_IDS` JSON is malformed, the guard silently skips (does not trigger), allowing enrollment to proceed. This is by design to avoid blocking enrollment on unrelated data corruption.

#### **3. Technology and Design Review**

- [ ] **Developer Handoff/QE Kickoff**
  - A meeting where Dev/Arch walked QE through the design, architecture, and implementation details. **Critical for identifying untestable aspects early.**
  - Fix was flagged by 6 of 9 independent review agents as the highest-consensus issue in PR #2331. The triage agent provided a detailed analysis with recommended fix and test plan.
- [ ] **Technology Challenges**
  - Identified potential testing challenges related to the underlying technology.
  - No significant challenges. The fix uses existing `mintcore.RoleOnlyAppIDs()` function and standard JSON unmarshal. Test infrastructure already supports the required mocking patterns.
- [ ] **Test Environment Needs**
  - Determined necessary **test environment setups and tools**.
  - Unit tests only; no cluster or GCP infrastructure required. Tests use the existing `fakeGCFClient` test double.
- [ ] **API Extensions**
  - Reviewed new or modified APIs and their impact on testing.
  - No API changes. The guard modifies internal behavior of `EnsureOrgInMint` only. The function signature and external behavior (success path) are unchanged.
- [ ] **Topology Considerations**
  - Evaluated multi-cluster, network topology, and architectural impacts.
  - N/A. The guard operates on env var state from a single Cloud Run service. No topology-dependent behavior.

### **II. Software Test Plan (STP)**

This STP serves as the **overall roadmap for testing**, detailing the scope, approach, resources, and schedule.

#### **1. Scope of Testing**

Testing covers the restored data consistency guard in `EnsureOrgInMint` and its interaction with the role-only `ROLE_APP_IDS` model. The scope includes the guard logic itself, the `mintcore.RoleOnlyAppIDs` filtering function, error message content, and the integration with CLI callers (`mint enroll`).

**Testing Goals**

**Functional Goals:**
- **P0:** Verify the guard correctly detects data inconsistency (empty `ALLOWED_ORGS` with populated role-only `ROLE_APP_IDS` entries) and returns an actionable error
- **P0:** Verify first enrollment (both empty) proceeds without triggering the guard
- **P1:** Verify legacy org-scoped keys in `ROLE_APP_IDS` are correctly filtered and do not trigger false positives
- **P1:** Verify edge cases (malformed JSON, nil map, missing `ROLE_APP_IDS` key) are handled gracefully

**Quality Goals:**
- **P1:** Verify error messages contain sufficient diagnostic information (role count, project ID, suggested command)
- **P2:** Verify the guard does not impact normal enrollment performance

**Integration Goals:**
- **P1:** Verify both `provisionWithExistingMint` and `provisionSelfManaged` invoke the guard through `EnsureOrgInMint`
- **P2:** Verify CLI `mint enroll` surfaces the guard error with actionable output

**Out of Scope (Testing Scope Exclusions)**

- [ ] Cloud Run revision divergence simulation -- *Rationale:* Platform-level behavior; guard assumes stale reads can happen and tests the detection logic, not the cause -- *PM/Lead Agreement:* TBD
- [ ] Concurrent enrollment race condition resolution -- *Rationale:* Existing known limitation documented in code comments; guard does not change concurrency behavior -- *PM/Lead Agreement:* TBD
- [ ] GCP Secret Manager operations -- *Rationale:* Unrelated to the guard; PEM storage is unchanged -- *PM/Lead Agreement:* TBD

#### **2. Test Strategy**

**Functional**

- [x] **Functional Testing** -- Validates that the feature works according to specified requirements and user stories
  - *Details:* Unit tests covering all guard code paths: data inconsistency detection, first enrollment pass-through, legacy key filtering, edge cases (malformed JSON, nil maps). Tests use `fakeGCFClient` with configurable `trafficEnvVars`.
- [x] **Automation Testing** -- Confirms test automation plan is in place for CI and regression coverage (all tests are expected to be automated)
  - *Details:* All tests are Go unit tests in `provisioner_test.go`, executed by `go test`. Already integrated into CI pipeline.
- [x] **Regression Testing** -- Verifies that new changes do not break existing functionality
  - *Details:* Existing `TestEnsureOrgInMint_*` test suite (20+ tests) covers all pre-existing enrollment behaviors. New tests are additive and do not modify existing test cases.

**Non-Functional**

- [ ] **Performance Testing** -- Validates feature performance meets requirements (latency, throughput, resource usage)
  - *Details:* N/A. Guard adds one `json.Unmarshal` and one map iteration; negligible overhead.
- [ ] **Scale Testing** -- Validates feature behavior under increased load and at production-like scale
  - *Details:* N/A. Guard operates on a single JSON value, independent of org count or scale.
- [ ] **Security Testing** -- Verifies security requirements, RBAC, authentication, authorization, and vulnerability scanning
  - *Details:* N/A. Guard does not change authentication or authorization behavior.
- [ ] **Usability Testing** -- Validates user experience and accessibility requirements
  - *Details:* N/A. Error message content is verified by unit tests for actionability.
- [ ] **Monitoring** -- Does the feature require metrics and/or alerts?
  - *Details:* N/A. The guard returns an error; no new metrics or alerts are introduced.

**Integration & Compatibility**

- [ ] **Compatibility Testing** -- Ensures feature works across supported platforms, versions, and configurations
  - *Details:* N/A. Guard logic is platform-independent Go code.
- [ ] **Upgrade Testing** -- Validates upgrade paths from previous versions, data migration, and configuration preservation
  - *Details:* N/A. No data migration; guard activates on existing env var state.
- [ ] **Dependencies** -- Blocked by deliverables from other components/products
  - *Details:* Depends on `mintcore.RoleOnlyAppIDs()` which already exists and is tested.
- [ ] **Cross Integrations** -- Does the feature affect other features or require testing by other teams?
  - *Details:* Guard is called by `provisionWithExistingMint` and `provisionSelfManaged`. Both paths are covered by existing integration through `EnsureOrgInMint`.

**Infrastructure**

- [ ] **Cloud Testing** -- Does the feature require multi-cloud platform testing?
  - *Details:* N/A. All tests use mocked GCP clients.

#### **3. Test Environment**

- **Cluster Topology:** N/A (unit tests only)
- **Platform & Product Version(s):** Go 1.26+, fullsend CLI
- **CPU Virtualization:** N/A
- **Compute Resources:** Standard CI runner
- **Special Hardware:** N/A
- **Storage:** N/A
- **Network:** N/A
- **Required Operators:** N/A
- **Platform:** Linux (CI), macOS (local dev)
- **Special Configurations:** N/A

#### **3.1. Testing Tools & Frameworks**

- **Test Framework:** Standard Go `testing` package (not new)
- **CI/CD:** Standard CI pipeline (not new)
- **Other Tools:** N/A

#### **4. Entry Criteria**

The following conditions must be met before testing can begin:

- [ ] Requirements and design documents are **approved and merged**
- [ ] Test environment can be **set up and configured** (see Section II.3 - Test Environment)
- [ ] `mintcore.RoleOnlyAppIDs()` function is available and tested
- [ ] `fakeGCFClient` supports `trafficEnvVars` field for `GetServiceTrafficEnvVars` mocking

#### **5. Risks**

- [ ] **Timeline/Schedule**
  - Risk: Low risk. Fix is a small, well-scoped code change with clear test cases.
  - Mitigation: Test cases are already defined in the triage agent's analysis.
- [ ] **Test Coverage**
  - Risk: Guard only covers the "completely empty ALLOWED_ORGS" case; partially stale reads are not detected.
  - Mitigation: Document as known limitation. The existing `mergeAllowedOrgs` union logic provides partial protection.
- [ ] **Test Environment**
  - Risk: N/A. No special environment required.
  - Mitigation: N/A.
- [ ] **Untestable Aspects**
  - Risk: Actual Cloud Run revision divergence cannot be simulated in unit tests.
  - Mitigation: Tests mock the `GetServiceTrafficEnvVars` response to simulate the symptoms (empty `ALLOWED_ORGS`), not the root cause.
- [ ] **Resource Constraints**
  - Risk: N/A. Standard CI resources sufficient.
  - Mitigation: N/A.
- [ ] **Dependencies**
  - Risk: `mintcore.RoleOnlyAppIDs` behavior change could break guard logic.
  - Mitigation: `RoleOnlyAppIDs` has its own unit tests (`TestRoleOnlyAppIDs_*`) that validate the filtering contract.
- [ ] **Other**
  - Risk: N/A.
  - Mitigation: N/A.

---

### **III. Test Scenarios & Traceability**

This section links requirements to test coverage, enabling reviewers to verify all requirements are tested.

#### **1. Requirements-to-Tests Mapping**

- **[GH-2433]** -- Data consistency guard detects stale ALLOWED_ORGS when ROLE_APP_IDS has role-only entries
  - *Test Scenario:* Verify guard returns error when ALLOWED_ORGS is empty but ROLE_APP_IDS has role-only entries
  - *Tier:* Unit Tests
  - *Priority:* P0

- **[GH-2433]** -- First enrollment proceeds when both ALLOWED_ORGS and ROLE_APP_IDS are empty
  - *Test Scenario:* Verify enrollment succeeds when both ALLOWED_ORGS and ROLE_APP_IDS are empty (genuine first enrollment)
  - *Tier:* Unit Tests
  - *Priority:* P0

- **[GH-2433]** -- Guard does not block enrollment when ALLOWED_ORGS is populated
  - *Test Scenario:* Verify guard is bypassed when ALLOWED_ORGS has existing orgs
  - *Tier:* Unit Tests
  - *Priority:* P0

- **[GH-2433]** -- Legacy org-scoped keys in ROLE_APP_IDS do not trigger guard
  - *Test Scenario:* Verify guard ignores org/role keys (containing "/") and only evaluates role-only keys
  - *Tier:* Unit Tests
  - *Priority:* P1

- **[GH-2433]** -- Error message includes diagnostic information for operator triage
  - *Test Scenario:* Verify error contains role count, project ID, and suggested mint status command
  - *Tier:* Unit Tests
  - *Priority:* P1

- **[GH-2433]** -- Guard handles malformed or missing ROLE_APP_IDS gracefully
  - *Test Scenario:* Verify guard does not trigger on malformed JSON, nil map, or missing ROLE_APP_IDS key
  - *Tier:* Unit Tests
  - *Priority:* P1

- **[GH-2433]** -- Guard handles ROLE_APP_IDS with empty JSON object
  - *Test Scenario:* Verify enrollment proceeds when ROLE_APP_IDS is "{}" (empty object, no roles configured)
  - *Tier:* Unit Tests
  - *Priority:* P1

- **[GH-2433]** -- Provision flows invoke guard via EnsureOrgInMint for each org
  - *Test Scenario:* Verify provisionWithExistingMint and provisionSelfManaged call EnsureOrgInMint and propagate guard errors
  - *Tier:* Unit Tests
  - *Priority:* P1

- **[GH-2433]** -- CLI mint enroll surfaces guard error to operator
  - *Test Scenario:* Verify mint enroll command prints actionable error when data inconsistency guard fires
  - *Tier:* Functional
  - *Priority:* P2

- **[GH-2433]** -- Guard behavior under concurrent enrollment attempts
  - *Test Scenario:* Verify guard fires correctly when concurrent enrollments encounter stale ALLOWED_ORGS
  - *Tier:* Functional
  - *Priority:* P2

---

### **IV. Sign-off and Approval**

This Software Test Plan requires approval from the following stakeholders:

* **Reviewers:**
  - [TBD]
  - [TBD]
* **Approvers:**
  - [TBD]
  - [TBD]
