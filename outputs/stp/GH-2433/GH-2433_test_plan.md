# FullSend Test Plan

## **Restore Data Consistency Guard in EnsureOrgInMint - Quality Engineering Plan**

### **Metadata & Tracking**

- **Enhancement(s):** [GH-2433](https://github.com/fullsend-ai/fullsend/issues/2433)
- **Feature Tracking:** [GH-2433](https://github.com/fullsend-ai/fullsend/issues/2433)
- **Epic Tracking:** GH-2433 (standalone bug fix; original bug: GH-1842)
- **QE Owner(s):** TBD
- **Owning SIG:** N/A
- **Participating SIGs:** None

**Document Conventions (if applicable):** N/A

### **Feature Overview**

PR #2331 removed a defense-in-depth cross-check in `EnsureOrgInMint` that prevented silent clobbering of `ALLOWED_ORGS` on stale reads, because the guard relied on org-scoped `ROLE_APP_IDS` keys which no longer exist in the role-only model. This fix restores the guard adapted for the role-only model: if `ALLOWED_ORGS` is empty but `mintcore.RoleOnlyAppIDs()` finds role-only entries in `ROLE_APP_IDS`, the mint has been bootstrapped and empty `ALLOWED_ORGS` indicates env var data loss. The guard returns an error instead of silently writing only the new org, which would unenroll all existing orgs.

---

### **I. Motivation and Requirements Review (QE Review Guidelines)**

This section documents the mandatory QE review process. The goal is to understand the feature's value,
technology, and testability before formal test planning.

#### **1. Requirement & User Story Review Checklist**

- [ ] **Review Requirements**
  - Reviewed the relevant requirements.
  - GH-2433 describes the regression clearly: PR #2331 removed the stale-read guard added by PR #1846 (which fixed GH-1842) without adding a replacement for the role-only model.
- [ ] **Understand Value and Customer Use Cases**
  - Confirmed clear user stories and understood.
  - Understand the difference between community and product requirements.
  - **What is the value of the feature for customers**.
  - Ensured requirements contain relevant **customer use cases**.
  - The guard prevents silent unenrollment of all existing orgs during mint enrollment — a data loss scenario with no error signal. Critical for multi-org deployments.
- [ ] **Testability**
  - Confirmed requirements are **testable and unambiguous**.
  - The guard logic is fully testable with mocked GCF clients. All paths (empty ALLOWED_ORGS + role-only entries, empty both, legacy keys only, populated ALLOWED_ORGS) are discrete and deterministic.
- [ ] **Acceptance Criteria**
  - Ensured acceptance criteria are **defined clearly** (clear user stories; product requirements clearly defined in Jira).
  - Acceptance criteria are embedded in the issue body: guard triggers when ALLOWED_ORGS is empty but ROLE_APP_IDS has role-only entries; first enrollment (both empty) proceeds normally; legacy keys are filtered out.
- [ ] **Non-Functional Requirements (NFRs)**
  - Confirmed coverage for NFRs, including Performance, Security, Usability, Downtime, Connectivity, Monitoring (alerts/metrics), Scalability, Portability (e.g., cloud support), and Docs.
  - No performance impact — guard is a single map iteration before existing logic. Error message includes actionable remediation command (`fullsend mint status`).

#### **2. Known Limitations**

- The guard relies on `mintcore.RoleOnlyAppIDs` filtering, which distinguishes role-only keys (e.g., `"coder"`) from legacy org-scoped keys (e.g., `"acme/coder"`) by the presence of `/`. If a future role name contains `/`, it would be incorrectly filtered as a legacy key.
- Malformed `ROLE_APP_IDS` JSON silently falls through (unmarshal error ignored), allowing enrollment to proceed. This is intentional — a malformed env var should not block enrollment — but means the guard cannot protect against corruption of ROLE_APP_IDS itself.
- The guard does not distinguish between "ALLOWED_ORGS was never set" and "ALLOWED_ORGS was set but is now empty due to data loss" when ROLE_APP_IDS is also empty. Both are treated as first enrollment.

#### **3. Technology and Design Review**

- [ ] **Developer Handoff/QE Kickoff**
  - A meeting where Dev/Arch walked QE through the design, architecture, and implementation details. **Critical for identifying untestable aspects early.**
  - Fix is a 20-line guard in `provisioner.go:419-431`. The implementation uses existing `mintcore.RoleOnlyAppIDs()` to detect bootstrapped mints. Code review by 6 of 9 independent review agents flagged the missing guard as the highest-consensus issue.
- [ ] **Technology Challenges**
  - Identified potential testing challenges related to the underlying technology.
  - No new technology challenges. The guard uses standard Go patterns (JSON unmarshal, map iteration) and integrates with existing test infrastructure (fake GCF client).
- [ ] **Test Environment Needs**
  - Determined necessary **test environment setups and tools**.
  - Unit tests run with `go test` using the existing `fakeGCFClient`. No cluster or cloud resources required.
- [ ] **API Extensions**
  - Reviewed new or modified APIs and their impact on testing.
  - No API changes. The guard is internal to `EnsureOrgInMint` and surfaces as a new error return path.
- [ ] **Topology Considerations**
  - Evaluated multi-cluster, network topology, and architectural impacts.
  - N/A — the guard operates on in-memory env var maps read from Cloud Run. No topology dependencies.

### **II. Software Test Plan (STP)**

This STP serves as the **overall roadmap for testing**, detailing the scope, approach, resources, and schedule.

#### **1. Scope of Testing**

Testing covers the restored data consistency guard in `EnsureOrgInMint` (`internal/dispatch/gcf/provisioner.go`). The guard prevents silent org unenrollment when `ALLOWED_ORGS` is empty on a bootstrapped mint by checking for role-only entries in `ROLE_APP_IDS`. Testing validates the guard triggers correctly on data inconsistency, passes through on legitimate first enrollment, handles legacy key formats, and propagates errors through the provisioning and CLI call chains.

**Testing Goals**

**Functional Goals:**
- **P0:** Verify the data consistency guard correctly detects and blocks enrollment when ALLOWED_ORGS is empty but ROLE_APP_IDS has role-only entries (prevents silent org unenrollment)
- **P0:** Verify first enrollment proceeds normally when both ALLOWED_ORGS and ROLE_APP_IDS are empty (no false positives)
- **P1:** Verify legacy org-scoped ROLE_APP_IDS keys are correctly filtered and do not trigger the guard

**Quality Goals:**
- **P1:** Verify error messages are actionable and include sufficient context (role count, project ID, remediation command)
- **P1:** Verify malformed ROLE_APP_IDS JSON does not cause panics or false positive guard triggers

**Integration Goals:**
- **P1:** Verify error propagation through provisionWithExistingMint and CLI commands
- **P2:** Verify CLI user experience when data inconsistency is detected

**Out of Scope (Testing Scope Exclusions)**

- [ ] Cloud Run revision traffic splitting and env var propagation -- *Rationale:* GCP platform-level behavior tested by GCP; guard operates on already-read env var maps -- *PM/Lead Agreement:* TBD
- [ ] WIF pool and provider configuration -- *Rationale:* Unrelated to the ALLOWED_ORGS guard; tested separately in existing WIF tests -- *PM/Lead Agreement:* TBD
- [ ] Actual Cloud Run deployment with stale revision reads -- *Rationale:* Would require live GCP infrastructure; the stale-read scenario is simulated via mock env vars -- *PM/Lead Agreement:* TBD

#### **2. Test Strategy**

**Functional**

- [x] **Functional Testing** — Validates that the feature works according to specified requirements and user stories
  - *Details:* Core guard logic tested across all code paths: data inconsistency detection, first enrollment pass-through, legacy key filtering, populated ALLOWED_ORGS bypass, malformed JSON handling.
- [x] **Automation Testing** — Confirms test automation plan is in place for CI and regression coverage (all tests are expected to be automated)
  - *Details:* All tests are Go unit tests in `provisioner_test.go`, run via `go test ./internal/dispatch/gcf/...`. PR #2436 includes 3 new test functions covering the guard.
- [x] **Regression Testing** — Verifies that new changes do not break existing functionality
  - *Details:* Existing `TestEnsureOrgInMint_DerivesAllowedRolesWhenEmpty` updated to set non-empty ALLOWED_ORGS, ensuring the guard does not interfere with ALLOWED_ROLES derivation. Full existing test suite validates no regressions.

**Non-Functional**

- [ ] **Performance Testing** — Validates feature performance meets requirements (latency, throughput, resource usage)
  - *Details:* N/A — guard adds a single map iteration (O(n) where n = number of roles, typically 2-5). No measurable performance impact.
- [ ] **Scale Testing** — Validates feature behavior under increased load and at production-like scale
  - *Details:* N/A — guard operates on small in-memory maps. Scale characteristics unchanged.
- [ ] **Security Testing** — Verifies security requirements, RBAC, authentication, authorization, and vulnerability scanning
  - *Details:* N/A — no new auth paths. The guard is a read-only check that does not modify state.
- [ ] **Usability Testing** — Validates user experience and accessibility requirements
  - *Details:* Error message includes actionable remediation: `run 'fullsend mint status --project=<id>' to investigate`. Validated in unit tests.
- [ ] **Monitoring** — Does the feature require metrics and/or alerts?
  - *Details:* N/A — the guard returns a standard Go error. No new metrics or alerts needed.

**Integration & Compatibility**

- [ ] **Compatibility Testing** — Ensures feature works across supported platforms, versions, and configurations
  - *Details:* N/A — pure Go logic with no platform dependencies.
- [ ] **Upgrade Testing** — Validates upgrade paths from previous versions, data migration, and configuration preservation
  - *Details:* N/A — the guard is additive and does not change data formats. Mints enrolled before this fix continue to work.
- [x] **Dependencies** — Blocked by deliverables from other components/products
  - *Details:* Depends on `mintcore.RoleOnlyAppIDs()` (existing function, no changes needed). No external dependencies.
- [x] **Cross Integrations** — Does the feature affect other features or require testing by other teams?
  - *Details:* The guard affects all callers of `EnsureOrgInMint`: `provisionWithExistingMint`, `Provision`, CLI `mint enroll`, and CLI `mint enroll-repo`. Existing tests for these callers validate error propagation.

**Infrastructure**

- [ ] **Cloud Testing** — Does the feature require multi-cloud platform testing?
  - *Details:* N/A — guard logic is cloud-agnostic. GCF client is fully mocked in tests.

#### **3. Test Environment**

- **Cluster Topology:** N/A (no cluster required; unit tests only)
- **Platform & Product Version(s):** FullSend 0.x on GitHub Actions
- **CPU Virtualization:** N/A
- **Compute Resources:** Standard CI runner (GitHub Actions)
- **Special Hardware:** None
- **Storage:** N/A
- **Network:** N/A (no network calls in tests; GCF client mocked)
- **Required Operators:** None
- **Platform:** GitHub Actions (Linux)
- **Special Configurations:** None

#### **3.1. Testing Tools & Frameworks**

- **Test Framework:** Standard (Go testing + testify)
- **CI/CD:** Standard (GitHub Actions)
- **Other Tools:** None

#### **4. Entry Criteria**

The following conditions must be met before testing can begin:

- [ ] Requirements and design documents are **approved and merged**
- [ ] Test environment can be **set up and configured** (see Section II.3 - Test Environment)
- [ ] PR #2436 is rebased on current main and passes CI
- [ ] `mintcore.RoleOnlyAppIDs` function exists and filters org-scoped keys correctly

#### **5. Risks**

- [ ] **Timeline/Schedule**
  - Risk: Low. Fix is a 20-line guard with clear acceptance criteria. PR #2436 already implements the fix with tests.
  - Mitigation: PR is ready for review; no additional development needed.
- [ ] **Test Coverage**
  - Risk: Guard cannot be tested against actual Cloud Run stale-read scenarios in CI.
  - Mitigation: Mock-based tests simulate the exact env var states that cause data inconsistency. The original bug (GH-1842) was also validated this way.
- [ ] **Test Environment**
  - Risk: None. All tests run locally or on standard CI runners with no external dependencies.
  - Mitigation: N/A
- [ ] **Untestable Aspects**
  - Risk: The actual Cloud Run revision divergence that causes stale reads cannot be reproduced in test.
  - Mitigation: The guard is designed to detect the *symptom* (empty ALLOWED_ORGS with non-empty ROLE_APP_IDS) rather than the *cause* (revision divergence), making it fully testable via env var mocks.
- [ ] **Resource Constraints**
  - Risk: None. No special infrastructure required.
  - Mitigation: N/A
- [ ] **Dependencies**
  - Risk: `mintcore.RoleOnlyAppIDs` behavior could change in future PRs, potentially breaking the guard.
  - Mitigation: Dedicated unit tests (`TestRoleOnlyAppIDs_IgnoresLegacyOrgScopedKeys`, `TestRoleOnlyAppIDs_ReturnsNilForEmpty`) pin the expected behavior.
- [ ] **Other**
  - Risk: Future role names containing `/` would be incorrectly classified as legacy keys.
  - Mitigation: Document the constraint; consider adding validation for role names if `/` is introduced.

---

### **III. Test Scenarios & Traceability**

This section links requirements to test coverage, enabling reviewers to verify all requirements are tested.

#### **1. Requirements-to-Tests Mapping**

- **[GH-2433]** — Data consistency guard prevents silent org unenrollment when ALLOWED_ORGS is empty on a bootstrapped mint
  - *Test Scenario:* Verify guard returns error when ALLOWED_ORGS empty with role-only entries (Unit Tests)
  - *Priority:* P0
  - *Test Scenario:* Verify error message includes role count and project ID (Unit Tests)
  - *Priority:* P1
  - *Test Scenario:* Verify no env var write occurs on data inconsistency (Unit Tests)
  - *Priority:* P0

- **[GH-2433]** — First enrollment proceeds normally when both ALLOWED_ORGS and ROLE_APP_IDS are empty
  - *Test Scenario:* Verify first enrollment succeeds with empty state (Unit Tests)
  - *Priority:* P0
  - *Test Scenario:* Verify ALLOWED_ORGS written on first enrollment (Unit Tests)
  - *Priority:* P1

- **[GH-2433]** — Legacy org-scoped ROLE_APP_IDS keys do not trigger the data consistency guard
  - *Test Scenario:* Verify enrollment proceeds with legacy org-scoped keys only (Unit Tests)
  - *Priority:* P1
  - *Test Scenario:* Verify mixed legacy and role-only keys trigger guard (Unit Tests)
  - *Priority:* P1

- **[GH-2433]** — Existing org enrollment flow is unaffected when ALLOWED_ORGS is populated
  - *Test Scenario:* Verify enrollment succeeds with pre-existing ALLOWED_ORGS (Unit Tests)
  - *Priority:* P1
  - *Test Scenario:* Verify duplicate org enrollment is idempotent (Unit Tests)
  - *Priority:* P2

- **[GH-2433]** — Provisioning with existing mint correctly propagates data consistency errors
  - *Test Scenario:* Verify provisioning aborts on data inconsistency (Functional)
  - *Priority:* P1
  - *Test Scenario:* Verify error wraps with org context for debugging (Functional)
  - *Priority:* P2

- **[GH-2433]** — CLI mint enroll command surfaces data consistency errors to user
  - *Test Scenario:* Verify CLI enroll shows actionable error on inconsistency (End-to-End)
  - *Priority:* P1
  - *Test Scenario:* Verify CLI suggests mint status command in error output (End-to-End)
  - *Priority:* P2

- **[GH-2433]** — CLI mint enroll-repo command surfaces data consistency errors to user
  - *Test Scenario:* Verify CLI enroll-repo shows error on inconsistency (End-to-End)
  - *Priority:* P2

- **[GH-2433]** — Malformed ROLE_APP_IDS JSON does not cause panic or false positive guard trigger
  - *Test Scenario:* Verify enrollment proceeds with malformed ROLE_APP_IDS JSON (Unit Tests)
  - *Priority:* P1
  - *Test Scenario:* Verify no panic on empty ROLE_APP_IDS string (Unit Tests)
  - *Priority:* P2

---

### **IV. Sign-off and Approval**

This Software Test Plan requires approval from the following stakeholders:

* **Reviewers:**
  - [TBD / @reviewer]
  - [TBD / @reviewer]
* **Approvers:**
  - [TBD / @approver]
  - [TBD / @approver]
