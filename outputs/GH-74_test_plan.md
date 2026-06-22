# Test Plan

## **Restore Data Consistency Guard in EnsureOrgInMint - Quality Engineering Plan**

### **Metadata & Tracking**

- **Enhancement(s):** [GH-74](https://github.com/guyoron1/fullsend/issues/74)
- **Feature Tracking:** [GH-74](https://github.com/guyoron1/fullsend/issues/74) — Mirror of upstream fullsend-ai/fullsend#2436
- **Epic Tracking:** [GH-2433](https://github.com/fullsend-ai/fullsend/issues/2433) — Data consistency guard
- **QE Owner(s):** Unassigned
- **Owning SIG:** N/A
- **Participating SIGs:** N/A

**Document Conventions (if applicable):** N/A

### **Feature Overview**

This fix restores a data consistency guard in the `EnsureOrgInMint` function within the GCF provisioner that was accidentally removed. The guard detects a dangerous state where `ALLOWED_ORGS` is empty but `ROLE_APP_IDS` contains role-only entries, which indicates environment variable data loss rather than a genuine first enrollment. Without the guard, enrolling a new org in this state would silently unenroll all existing organizations. The guard aborts with an actionable error message directing the operator to investigate with `fullsend mint status`.

---

### **I. Motivation and Requirements Review (QE Review Guidelines)**

This section documents the mandatory QE review process. The goal is to understand the feature's value,
technology, and testability before formal test planning.

#### **1. Requirement & User Story Review Checklist**

- [ ] **Review Requirements**
  - Reviewed the relevant requirements.
  - The issue description is concise: restore an accidentally removed guard. The upstream issue (fullsend-ai/fullsend#2436) and root issue (#2433) provide full context.
- [ ] **Understand Value and Customer Use Cases**
  - Confirmed clear user stories and understood.
  - Understand the difference between community and product requirements.
  - **What is the value of the feature for customers**.
  - Ensured requirements contain relevant **customer use cases**.
  - The guard prevents silent data loss during mint enrollment. Without it, a stale read or env var data loss could cause all existing organizations to be unenrolled when a new org is added.
- [ ] **Testability**
  - Confirmed requirements are **testable and unambiguous**.
  - The guard logic is fully testable with mocked GCF clients. The `fakeGCFClient` test infrastructure allows precise control over `ALLOWED_ORGS` and `ROLE_APP_IDS` env var state.
- [ ] **Acceptance Criteria**
  - Ensured acceptance criteria are **defined clearly** (clear user stories; product requirements clearly defined in Jira).
  - Acceptance criteria inferred from implementation: (1) guard fires when ALLOWED_ORGS empty + ROLE_APP_IDS has role-only keys, (2) guard does not fire on first enrollment, (3) error message is actionable with project ID and suggested command.
- [ ] **Non-Functional Requirements (NFRs)**
  - Confirmed coverage for NFRs, including Performance, Security, Usability, Downtime, Connectivity, Monitoring (alerts/metrics), Scalability, Portability (e.g., cloud support), and Docs.
  - Concurrency safety: the guard must fire independently per goroutine. No shared mutable state between concurrent `EnsureOrgInMint` calls. Performance is not a concern as this is a low-frequency provisioning operation.

#### **2. Known Limitations**

- The `EnsureOrgInMint` function uses a read-modify-write pattern without locking. Concurrent calls from parallel per-repo installs sharing the same mint can race, causing one update to overwrite the other. This is documented in the function's warning comment and is accepted behavior — a lost update is corrected on the next run.
- The guard relies on `ROLE_APP_IDS` JSON parsing; malformed JSON is treated as empty (guard does not fire), which is a deliberate defensive choice to avoid blocking enrollment on corrupt data.

#### **3. Technology and Design Review**

- [ ] **Developer Handoff/QE Kickoff**
  - A meeting where Dev/Arch walked QE through the design, architecture, and implementation details. **Critical for identifying untestable aspects early.**
  - The change is a restoration of previously existing logic. The guard code at `provisioner.go:419-431` is self-contained and well-documented with inline comments explaining the rationale.
- [ ] **Technology Challenges**
  - Identified potential testing challenges related to the underlying technology.
  - No significant challenges. The `fakeGCFClient` provides full control over GCF API responses. The `mintcore.RoleOnlyAppIDs` helper handles legacy key filtering.
- [ ] **Test Environment Needs**
  - Determined necessary **test environment setups and tools**.
  - All tests run locally with mocked dependencies. No GCP project or cluster needed.
- [ ] **API Extensions**
  - Reviewed new or modified APIs and their impact on testing.
  - No new APIs. The change modifies internal guard logic within the existing `EnsureOrgInMint` method. The method signature is unchanged.
- [ ] **Topology Considerations**
  - Evaluated multi-cluster, network topology, and architectural impacts.
  - Not applicable. The guard operates on environment variable state within a single GCF function deployment.

### **II. Software Test Plan (STP)**

This STP serves as the **overall roadmap for testing**, detailing the scope, approach, resources, and schedule.

#### **1. Scope of Testing**

Testing validates that the restored data consistency guard in `EnsureOrgInMint` correctly detects and blocks enrollment when environment variable state indicates data loss. The scope covers the guard's detection logic, error messaging, edge case handling, error propagation through provisioning flows, and concurrent enrollment behavior.

**Testing Goals**

- **P0:** Verify the guard correctly detects data inconsistency (empty `ALLOWED_ORGS` with role-only `ROLE_APP_IDS` entries) and blocks enrollment
- **P0:** Verify first enrollment succeeds when both env vars are empty (no false positive)
- **P0:** Verify the guard does not fire when `ALLOWED_ORGS` is populated (normal operation unaffected)
- **P1:** Verify error messages contain actionable diagnostic information (role count, project ID, suggested command)
- **P1:** Verify guard handles edge cases in `ROLE_APP_IDS` (malformed JSON, empty, missing) without false positives
- **P1:** Verify guard errors propagate correctly through `provisionWithExistingMint` and `provisionSelfManaged`
- **P2:** Verify guard fires independently per goroutine under concurrent enrollment

**Out of Scope (Testing Scope Exclusions)**

- [ ] GCP Cloud Functions deployment and infrastructure testing
  - *Rationale:* Guard logic is tested with mocked GCF client; actual GCP deployment is infrastructure-level
- [ ] End-to-end mint enrollment workflow against live GCP
  - *Rationale:* The guard is a defensive check within the provisioner; live integration is covered by e2e tests separately
- [ ] `mintcore.RoleOnlyAppIDs` implementation testing
  - *Rationale:* This is a dependency with its own test suite in `internal/mintcore/`

#### **2. Test Strategy**

**Functional**

- [ ] **Functional Testing** — Validates that the feature works according to specified requirements and user stories
  - *Details:* All guard behaviors tested through unit tests with `fakeGCFClient`. Covers detection, bypass, edge cases, error propagation, and concurrency.
- [ ] **Automation Testing** — Confirms test automation plan is in place for CI and regression coverage (all tests are expected to be automated)
  - *Details:* All tests are Go unit tests in `internal/dispatch/gcf/` and `qf-tests/GH-2433/go/`. They run in CI via `go test`.
- [ ] **Regression Testing** — Verifies that new changes do not break existing functionality
  - *Details:* Existing `TestEnsureOrgInMint_*` tests in `provisioner_test.go` (25+ test functions) ensure the guard restoration does not break normal enrollment flows.

**Non-Functional**

- [ ] **Performance Testing** — Validates feature performance meets requirements (latency, throughput, resource usage)
  - *Details:* Not applicable. `EnsureOrgInMint` is a low-frequency provisioning operation (called once per org enrollment).
- [ ] **Scale Testing** — Validates feature behavior under increased load and at production-like scale
  - *Details:* Not applicable. The guard is a single conditional check on env var values.
- [ ] **Security Testing** — Verifies security requirements, RBAC, authentication, authorization, and vulnerability scanning
  - *Details:* The guard is a defense-in-depth measure that prevents silent data loss. It does not modify authentication or authorization flows.
- [ ] **Usability Testing** — Validates user experience and accessibility requirements
  - *Details:* The error message includes role count, project ID, and a suggested `fullsend mint status` command for operator investigation.
- [ ] **Monitoring** — Does the feature require metrics and/or alerts?
  - *Details:* Not applicable. The guard produces an error that surfaces through normal CLI error reporting.

**Integration & Compatibility**

- [ ] **Compatibility Testing** — Ensures feature works across supported platforms, versions, and configurations
  - *Details:* Not applicable. The guard logic is platform-independent Go code with no OS or architecture dependencies.
- [ ] **Upgrade Testing** — Validates upgrade paths from previous versions, data migration, and configuration preservation
  - *Details:* Not applicable. The guard is stateless; it reads env vars on each call. No migration needed.
- [ ] **Dependencies** — Blocked by deliverables from other components/products
  - *Details:* Depends on `mintcore.RoleOnlyAppIDs` for legacy key filtering. This function is stable and tested in `internal/mintcore/`.
- [ ] **Cross Integrations** — Does the feature affect other features or require testing by other teams?
  - *Details:* The guard affects `provisionWithExistingMint`, `provisionSelfManaged`, `runMintEnrollOrg`, and `runMintEnrollRepo`. All callers propagate the guard error correctly.

**Infrastructure**

- [ ] **Cloud Testing** — Does the feature require multi-cloud platform testing?
  - *Details:* Not applicable. All tests use mocked GCF client.

#### **3. Test Environment**

- **Cluster Topology:** Not required — all tests run locally with mocked dependencies
- **Platform & Product Version(s):** Go 1.26+, any OS supported by Go toolchain
- **CPU Virtualization:** Not applicable
- **Compute Resources:** Standard CI runner (no special requirements)
- **Special Hardware:** None
- **Storage:** None
- **Network:** None (no network calls in unit tests)
- **Required Operators:** None
- **Platform:** Linux, macOS, or Windows (Go cross-platform)
- **Special Configurations:** None

#### **3.1. Testing Tools & Frameworks**

- **Test Framework:** Go standard `testing` package
- **CI/CD:** Standard (no special tools)
- **Other Tools:** `testify/assert` and `testify/require` for assertions

#### **4. Entry Criteria**

The following conditions must be met before testing can begin:

- [ ] Requirements and design documents are **approved and merged**
- [ ] Test environment can be **set up and configured** (see Section II.3 - Test Environment)
- [ ] The `fakeGCFClient` test infrastructure supports `trafficEnvVars` and `functionInfoAfterCreate` fields
- [ ] The `mintcore.RoleOnlyAppIDs` function is available and tested

#### **5. Risks**

- [ ] **Timeline/Schedule**
  - Risk: Low — the fix is a restoration of existing logic with known behavior
  - Mitigation: Tests are already written in `qf-tests/GH-2433/go/` and `provisioner_test.go`
- [ ] **Test Coverage**
  - Risk: Guard edge cases may not cover all possible `ROLE_APP_IDS` formats encountered in production
  - Mitigation: Edge case tests cover malformed JSON, empty string, missing key, empty object, legacy keys, and mixed keys
- [ ] **Test Environment**
  - Risk: None — tests use mocked dependencies with no external infrastructure
  - Mitigation: N/A
- [ ] **Untestable Aspects**
  - Risk: The read-modify-write race condition in `EnsureOrgInMint` cannot be fully reproduced in unit tests
  - Mitigation: Documented as a known limitation; concurrent test validates per-goroutine isolation with independent fake clients
- [ ] **Resource Constraints**
  - Risk: None — tests are lightweight Go unit tests
  - Mitigation: N/A
- [ ] **Dependencies**
  - Risk: `mintcore.RoleOnlyAppIDs` behavior change could affect guard correctness
  - Mitigation: The function has its own test suite; guard tests use known input/output pairs
- [ ] **Other**
  - Risk: None identified
  - Mitigation: N/A

---

### **III. Test Scenarios & Traceability**

This section links requirements to test coverage, enabling reviewers to verify all requirements are tested.

#### **1. Requirements-to-Tests Mapping**

- **Requirement ID:** GH-74
- **Requirement:** Data consistency guard detects and blocks enrollment when ALLOWED_ORGS is empty but ROLE_APP_IDS has configured roles
- **Evidence:** EnsureOrgInMint guard at provisioner.go:419-431 checks for empty ALLOWED_ORGS with populated role-only ROLE_APP_IDS entries
- **Test Scenarios:**
  - TS-GH-74-001: Verify guard returns error on empty ALLOWED_ORGS with role-only entries (Unit Tests, P0)
  - TS-GH-74-002: Verify no env var update attempted when guard fires (Unit Tests, P0)
  - TS-GH-74-003: Verify guard permits enrollment when ALLOWED_ORGS populated (Unit Tests, P0)

---

- **Requirement:** First enrollment succeeds when both ALLOWED_ORGS and ROLE_APP_IDS are empty
- **Evidence:** Guard must not fire when no prior state exists — both env vars empty is a clean state
- **Test Scenarios:**
  - TS-GH-74-004: Verify first enrollment proceeds with empty state (Unit Tests, P0)
  - TS-GH-74-005: Verify UpdateServiceEnvVars called on first enrollment (Unit Tests, P0)

---

- **Requirement:** Guard is bypassed when ALLOWED_ORGS already has enrolled orgs
- **Evidence:** Non-empty ALLOWED_ORGS indicates healthy state; guard only applies to empty ALLOWED_ORGS
- **Test Scenarios:**
  - TS-GH-74-006: Verify guard bypassed with populated ALLOWED_ORGS (Unit Tests, P0)
  - TS-GH-74-007: Verify existing orgs preserved during new enrollment (Unit Tests, P1)

---

- **Requirement:** Guard correctly filters legacy org/role keys from role-only keys
- **Evidence:** mintcore.RoleOnlyAppIDs filters keys; legacy keys like "org/agent" should not trigger guard
- **Test Scenarios:**
  - TS-GH-74-008: Verify legacy-only keys do not trigger guard (Unit Tests, P1)
  - TS-GH-74-009: Verify mixed legacy and role-only keys trigger guard (Unit Tests, P1)

---

- **Requirement:** Guard error message contains actionable diagnostic information
- **Evidence:** Error format at provisioner.go:427-429 includes role count, project ID, and fullsend mint status command
- **Test Scenarios:**
  - TS-GH-74-010: Verify error contains role count and project ID (Unit Tests, P1)
  - TS-GH-74-011: Verify error contains suggested mint status command (Unit Tests, P1)

---

- **Requirement:** Guard handles ROLE_APP_IDS edge cases without triggering
- **Evidence:** json.Unmarshal silently fails on bad input; empty/missing maps have len 0; guard must not false-positive
- **Test Scenarios:**
  - TS-GH-74-012: Verify guard safe on malformed ROLE_APP_IDS JSON (Unit Tests, P1)
  - TS-GH-74-013: Verify guard safe on empty or missing ROLE_APP_IDS (Unit Tests, P1)
  - TS-GH-74-014: Verify guard safe on empty JSON object ROLE_APP_IDS (Unit Tests, P1)

---

- **Requirement:** Guard errors propagate through provisioning flows
- **Evidence:** LSP incomingCalls shows EnsureOrgInMint called from provisionWithExistingMint (line 622) and provisionSelfManaged (line 897)
- **Test Scenarios:**
  - TS-GH-74-015: Verify provisionWithExistingMint propagates guard error (Unit Tests, P1)
  - TS-GH-74-016: Verify provisionSelfManaged propagates guard error (Unit Tests, P1)

---

- **Requirement:** Guard fires independently per goroutine under concurrent enrollment
- **Evidence:** EnsureOrgInMint is called per-org in parallel enrollment scenarios; each call reads its own env var state
- **Test Scenarios:**
  - TS-GH-74-017: Verify concurrent enrollments isolate guard evaluation (Unit Tests, P2)
  - TS-GH-74-018: Verify stale-read goroutine fails while fresh succeeds (Unit Tests, P2)

---

- **Requirement:** No env var update is attempted when the guard fires
- **Evidence:** Guard returns error before reaching UpdateServiceEnvVars call at provisioner.go:462
- **Test Scenarios:**
  - TS-GH-74-019: Verify UpdateServiceEnvVars not called on guard trigger (Unit Tests, P1)

---

### **IV. Sign-off and Approval**

This Software Test Plan requires approval from the following stakeholders:

* **Reviewers:**
  - [Name / @github-username]
  - [Name / @github-username]
* **Approvers:**
  - [Name / @github-username]
  - [Name / @github-username]
