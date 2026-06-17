# FullSend Test Plan

## **Add Lint() Diagnostic Method to Harness (ADR-0045 Phase 3) - Quality Engineering Plan**

### **Metadata & Tracking**

- **Enhancement(s):** [GH-23](https://github.com/guyoron1/fullsend/issues/23)
- **Feature Tracking:** [GH-23](https://github.com/guyoron1/fullsend/issues/23)
- **Epic Tracking:** [#2326](https://github.com/guyoron1/fullsend/issues/2326) (ADR-0045 implementation)
- **QE Owner(s):** TBD
- **Owning SIG:** N/A
- **Participating SIGs:** None

**Document Conventions (if applicable):** N/A

### **Feature Overview**

GH-23 introduces a `Lint()` diagnostic method on the `Harness` struct in `internal/harness/lint.go`, implementing Phase 3 PR 1 of [ADR-0045](https://github.com/fullsend-ai/fullsend/blob/main/docs/ADRs/0045-forge-portable-harness-schema.md). Unlike `Validate()`, which returns hard errors for structurally invalid harnesses, `Lint()` returns non-fatal `[]Diagnostic` warnings that surface best-practice recommendations. The first lint rule warns when the `role` field is missing, preparing for Phase 4 which will make `role` required. The PR also introduces the `DiagnosticSeverity` type (with `SeverityWarning` and `SeverityError` constants), the `Diagnostic` struct (with `Severity`, `Field`, `Message` fields and a `String()` formatter), and the Phase 3 implementation plan document. No existing code is modified; `Validate()` behavior is unchanged.

---

### **I. Motivation and Requirements Review (QE Review Guidelines)**

This section documents the mandatory QE review process. The goal is to understand the feature's value,
technology, and testability before formal test planning.

#### **1. Requirement & User Story Review Checklist**

- [x] **Review Requirements**
  - Reviewed the relevant requirements.
  - GH-23 implements ADR-0045 Phase 3 PR 1: add a `Lint()` method to the `Harness` struct that returns `[]Diagnostic` for non-fatal warnings, separate from `Validate()` which returns hard errors.
  - Requirements derived from the PR body, Phase 3 plan document, and ADR-0045.
- [x] **Understand Value and Customer Use Cases**
  - Confirmed clear user stories and understood.
  - Understand the difference between community and product requirements.
  - **What is the value of the feature for customers**.
  - Ensured requirements contain relevant **customer use cases**.
  - The `Lint()` method provides non-fatal diagnostic feedback to harness authors, warning about fields that will become required in future versions. This helps users proactively fix harness configurations before breaking changes are enforced.
- [x] **Testability**
  - Confirmed requirements are **testable and unambiguous**.
  - The feature is fully testable: `Lint()` is a pure method on the `Harness` struct with deterministic input/output behavior. The developer has written 6 passing subtests achieving 100% code coverage on `lint.go`.
- [x] **Acceptance Criteria**
  - Ensured acceptance criteria are **defined clearly** (clear user stories; product requirements clearly defined in Jira).
  - Acceptance criteria derived from PR body and test plan: (1) `Lint()` returns `[]Diagnostic` with warning when `role` is empty, (2) `Lint()` returns nil when no issues found, (3) `Diagnostic.String()` formats correctly for all severity levels, (4) all existing tests pass, (5) 100% code coverage on `lint.go`.
- [x] **Non-Functional Requirements (NFRs)**
  - Confirmed coverage for NFRs, including Performance, Security, Usability, Downtime, Connectivity, Monitoring (alerts/metrics), Scalability, Portability (e.g., cloud support), and Docs.
  - No NFRs applicable. `Lint()` is a pure library method with no I/O, no network calls, and no security surface. Performance is bounded by the number of fields checked (currently 1).

#### **2. Known Limitations**

- `Lint()` has no callers yet; CLI integration is planned for Phase 3 PR 3. This STP covers only the library-level behavior.
- Only one lint rule (missing `role`) is implemented. Future rules (missing `slug`, forge section completeness) are out of scope for this PR and this STP.
- `Lint()` should only be called after a successful `Validate()` — its results are meaningless on a structurally invalid harness. This precondition is documented but not enforced programmatically.

#### **3. Technology and Design Review**

- [x] **Developer Handoff/QE Kickoff**
  - A meeting where Dev/Arch walked QE through the design, architecture, and implementation details. **Critical for identifying untestable aspects early.**
  - No formal handoff required. The change is a small, self-contained library addition with clear design documented in ADR-0045 and the Phase 3 plan.
- [x] **Technology Challenges**
  - Identified potential testing challenges related to the underlying technology.
  - No technology challenges. The feature uses standard Go constructs (methods, structs, iota constants) and requires no external dependencies.
- [x] **Test Environment Needs**
  - Determined necessary **test environment setups and tools**.
  - Standard Go 1.23+ toolchain with testify assertion library. No special infrastructure required.
- [x] **API Extensions**
  - Reviewed new or modified APIs and their impact on testing.
  - New public API additions: `DiagnosticSeverity` type, `SeverityWarning` and `SeverityError` constants, `Diagnostic` struct with `String()` method, `Harness.Lint()` method. All are additive; no existing APIs are modified.
- [x] **Topology Considerations**
  - Evaluated multi-cluster, network topology, and architectural impacts.
  - No topology considerations. This is a pure Go library method with no network or infrastructure dependencies.

---

### **II. Software Test Plan (STP)**

This STP serves as the **overall roadmap for testing**, detailing the scope, approach, resources, and schedule.

#### **1. Scope of Testing**

This STP covers testing of the `Lint()` diagnostic method added to the `Harness` struct in the harness package, the `DiagnosticSeverity` type and its string representation, and the `Diagnostic` struct and its `String()` formatter. The feature is part of the harness component (Agent Harness), which is a core in-scope FullSend component.

**Testing Goals**

- Verify that `Lint()` returns a warning diagnostic when the `role` field is empty
- Verify that `Lint()` returns nil (not an empty slice) when no issues are found
- Verify that `Diagnostic.String()` produces correct human-readable output for all severity levels
- Verify that existing `Validate()` behavior is unchanged (regression)

**Out of Scope (Testing Scope Exclusions)**

- [x] Integration testing of `Lint()` with CLI callers -- *Rationale:* No callers exist yet; CLI integration is Phase 3 PR 3. -- *PM/Lead Agreement:* Per Phase 3 implementation plan.
- [x] Future lint rules (missing slug, forge section completeness) -- *Rationale:* Not implemented in this PR. Each future rule will have its own STP. -- *PM/Lead Agreement:* Per Phase 3 implementation plan.
- [x] Performance or scale testing -- *Rationale:* `Lint()` is a pure method checking a single field; performance is trivially bounded. -- *PM/Lead Agreement:* N/A.

#### **2. Test Strategy**

**Functional**

- [x] **Functional Testing** -- Validates that the feature works according to specified requirements and user stories
  - *Details:* Verify `Lint()` returns `[]Diagnostic` with a warning when `role` is empty. Verify `Lint()` returns nil when `role` is set. Verify `Diagnostic.String()` formats correctly for `SeverityWarning`, `SeverityError`, and unknown severity values.
- [x] **Automation Testing** -- Confirms test automation plan is in place for CI and regression coverage (all tests are expected to be automated)
  - *Details:* All tests are automated using Go `testing` package with testify assertions. Tests run in CI via `make go-test`. Developer has achieved 100% code coverage on `lint.go`.
- [x] **Regression Testing** -- Verifies that new changes do not break existing functionality
  - *Details:* Verify existing `Validate()` method behavior is unchanged by the addition of `Lint()`. All existing harness tests must continue to pass. Run `make go-test` to confirm full test suite passes.

**Non-Functional**

- [ ] **Performance Testing** -- Validates feature performance meets requirements (latency, throughput, resource usage)
  - *Details:* Not applicable. `Lint()` is a pure library method with O(1) complexity (checks one field).
- [ ] **Scale Testing** -- Validates feature behavior under increased load and at production-like scale (e.g., large number of resources, nodes, or concurrent operations)
  - *Details:* Not applicable.
- [ ] **Security Testing** -- Verifies security requirements, RBAC, authentication, authorization, and vulnerability scanning
  - *Details:* Not applicable. `Lint()` performs no I/O, authentication, or authorization.
- [ ] **Usability Testing** -- Validates user experience and accessibility requirements
  - *Details:* Not applicable. No UI or CLI changes in this PR.
- [ ] **Monitoring** -- Does the feature require metrics and/or alerts?
  - *Details:* Not applicable. No metrics or alerts required for a library method.

**Integration & Compatibility**

- [ ] **Compatibility Testing** -- Ensures feature works across supported platforms, versions, and configurations
  - *Details:* Not applicable. Standard Go library code with no platform-specific behavior.
- [ ] **Upgrade Testing** -- Validates upgrade paths from previous versions, data migration, and configuration preservation
  - *Details:* Not applicable. `Lint()` does not create or modify persistent state.
- [ ] **Dependencies** -- Blocked by deliverables from other components/products. Identify what we need from other teams before we can test.
  - *Details:* No dependencies. `Lint()` uses only the existing `Harness` struct fields.
- [ ] **Cross Integrations** -- Does the feature affect other features or require testing by other teams? Identify the impact we cause.
  - *Details:* No cross-integration impact. `Lint()` is additive with no callers in this PR.

**Infrastructure**

- [ ] **Cloud Testing** -- Does the feature require multi-cloud platform testing? Consider cloud-specific features.
  - *Details:* Not applicable.

#### **3. Test Environment**

- **Cluster Topology:** N/A (library-level unit tests)
- **Platform & Product Version(s):** FullSend 0.x on GitHub Actions
- **CPU Virtualization:** N/A
- **Compute Resources:** Standard CI runner
- **Special Hardware:** N/A
- **Storage:** N/A
- **Network:** N/A
- **Required Operators:** N/A
- **Platform:** GitHub Actions
- **Special Configurations:** Go 1.23+ toolchain

#### **3.1. Testing Tools & Frameworks**

- **Test Framework:** Go `testing` package with [testify](https://github.com/stretchr/testify) assertions
- **CI/CD:** GitHub Actions via `make go-test`
- **Other Tools:** `go tool cover` for coverage analysis

#### **4. Entry Criteria**

The following conditions must be met before testing can begin:

- [x] Requirements and design documents are **approved and merged** (ADR-0045, Phase 3 plan)
- [x] Test environment can be **set up and configured** (Go 1.23+ toolchain available in CI)
- [x] `Validate()` existing tests pass (baseline regression check)

#### **5. Risks**

- [x] **Timeline/Schedule**
  - Risk: N/A -- small, self-contained library addition with developer-written tests already passing.
  - Mitigation: N/A
- [x] **Test Coverage**
  - Risk: Low -- developer has achieved 100% code coverage on `lint.go`. Risk is limited to logic completeness (e.g., future lint rules not yet implemented).
  - Mitigation: Each future lint rule will have its own STP and test scenarios.
- [x] **Test Environment**
  - Risk: N/A -- standard Go toolchain, no special infrastructure.
  - Mitigation: N/A
- [x] **Untestable Aspects**
  - Risk: The precondition that `Lint()` should only be called after `Validate()` is not enforced programmatically. Behavior of `Lint()` on an invalid harness is undefined.
  - Mitigation: Document this precondition clearly. Consider adding a programmatic guard in a future PR.
- [x] **Resource Constraints**
  - Risk: N/A
  - Mitigation: N/A
- [x] **Dependencies**
  - Risk: N/A -- no external dependencies.
  - Mitigation: N/A
- [x] **Other**
  - Risk: N/A
  - Mitigation: N/A

---

### **III. Test Scenarios & Traceability**

This section links requirements to test coverage, enabling reviewers to verify all requirements are tested.

#### **1. Requirements-to-Tests Mapping**

- **[GH-23]** -- Lint() returns warning diagnostic when role is missing
  *Test Scenario:* Call Lint() on a Harness with empty Role field; verify it returns []Diagnostic with exactly one entry having Severity SeverityWarning, Field "role", and Message containing "required in a future version"
  *Priority:* P0 [Functional]

- **[GH-23]** -- Lint() returns nil for valid harness with role set
  *Test Scenario:* Call Lint() on a Harness with Role "triage"; verify it returns nil (not empty slice)
  *Priority:* P0 [Functional]

- **[GH-23]** -- Lint() returns nil when both role and slug are set
  *Test Scenario:* Call Lint() on a Harness with Role "triage" and Slug "my-slug"; verify it returns nil
  *Priority:* P1 [Functional]

- **[GH-23]** -- Diagnostic.String() formats warning severity correctly
  *Test Scenario:* Create Diagnostic with Severity SeverityWarning, Field "role", Message "msg"; verify String() returns "warning: role: msg"
  *Priority:* P1 [Functional]

- **[GH-23]** -- Diagnostic.String() formats error severity correctly
  *Test Scenario:* Create Diagnostic with Severity SeverityError, Field "role", Message "msg"; verify String() returns "error: role: msg"
  *Priority:* P1 [Functional]

- **[GH-23]** -- Unknown severity falls back to numeric format
  *Test Scenario:* Create Diagnostic with Severity DiagnosticSeverity(99), Field "x", Message "msg"; verify String() returns "DiagnosticSeverity(99): x: msg"
  *Priority:* P2 [Functional]

- **[GH-23]** -- Existing Validate() behavior unchanged after Lint() addition
  *Test Scenario:* Verify that Harness.Validate() continues to return errors for structurally invalid harnesses (e.g., missing required fields) and returns nil for valid harnesses, confirming no behavioral change from the addition of Lint()
  *Priority:* P0 [Functional]

---

### **IV. Sign-off and Approval**

This Software Test Plan requires approval from the following stakeholders:

* **Reviewers:**
  - [Name / @github-username]
  - [Name / @github-username]
* **Approvers:**
  - [Name / @github-username]
  - [Name / @github-username]
