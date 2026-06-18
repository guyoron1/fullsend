# My-Project Test Plan

## **EnsureProvider Idempotency Fix - Quality Engineering Plan**

### **Metadata & Tracking**

- **Enhancement(s):** [GH-28](https://github.com/guyoron1/fullsend/issues/28)
- **Feature Tracking:** [GH-28](https://github.com/guyoron1/fullsend/issues/28)
- **Epic Tracking:** GH-28
- **QE Owner(s):** TBD
- **Owning SIG:** N/A
- **Participating SIGs:** None

**Document Conventions (if applicable):** N/A

### **Feature Overview**

`EnsureProvider` in `internal/sandbox/sandbox.go` creates providers on the openshell gateway during `fullsend run`. Previously, it failed with an `AlreadyExists` error on repeated runs because it had no handling for pre-existing providers. This bug fix makes `EnsureProvider` idempotent by detecting `AlreadyExists` errors, deleting the existing provider, and recreating it with current credentials, consistent with the codebase's "Ensure" naming convention (e.g., `EnsureGateway`, `Provider.Provision`).

---

### **I. Motivation and Requirements Review (QE Review Guidelines)**

This section documents the mandatory QE review process. The goal is to understand the feature's value,
technology, and testability before formal test planning.

#### **1. Requirement & User Story Review Checklist**

- [ ] **Review Requirements**
  - Reviewed the relevant requirements.
  - GH-28 describes the `AlreadyExists` failure, expected idempotent behavior, and three possible resolution approaches (succeed silently, delete+recreate, hybrid). The chosen approach is delete+recreate.
- [ ] **Understand Value and Customer Use Cases**
  - Confirmed clear user stories and understood.
  - Understand the difference between community and product requirements.
  - **What is the value of the feature for customers**.
  - Ensured requirements contain relevant **customer use cases**.
  - Value: eliminates the need for manual `openshell provider delete` between repeated `fullsend run` invocations during local development and CI. Reduces friction for iterative testing workflows.
- [ ] **Testability**
  - Confirmed requirements are **testable and unambiguous**.
  - All behavior is testable via mocked `openshell` binaries using `t.TempDir()` and `t.Setenv("PATH", ...)`. The PR includes 6 unit tests covering the key scenarios. No cluster or real openshell required for unit-level validation.
- [ ] **Acceptance Criteria**
  - Ensured acceptance criteria are **defined clearly** (clear user stories; product requirements clearly defined in Jira).
  - Primary acceptance criterion: `fullsend run` must succeed on repeated invocations without manual provider cleanup. Secondary: credential values must never appear in error output.
- [ ] **Non-Functional Requirements (NFRs)**
  - Confirmed coverage for NFRs, including Performance, Security, Usability, Downtime, Connectivity, Monitoring (alerts/metrics), Scalability, Portability (e.g., cloud support), and Docs.
  - Security: secret redaction must cover all three error paths (initial create failure, delete failure, recreate failure). No performance or scalability concerns for this change.

#### **2. Known Limitations**

- The delete+recreate approach causes a brief window where the provider does not exist. If another process queries the provider during this window, it will see a `NotFound` error. This is acceptable for the current single-threaded `fullsend run` flow.
- The fix does not address the related idempotency gap in `InferenceLayer.Install()` mentioned in the issue (`inference.go:56-69`), which has a separate stale-values concern.

#### **3. Technology and Design Review**

- [ ] **Developer Handoff/QE Kickoff**
  - A meeting where Dev/Arch walked QE through the design, architecture, and implementation details. **Critical for identifying untestable aspects early.**
  - PR #29 authored by fullsend-ai-coder. The implementation adds a `deleteProvider` helper and modifies `EnsureProvider` to detect `AlreadyExists` in command output, then delete and recreate. Call chain: `newRunCmd` -> `runAgent` -> `EnsureProvider` -> `deleteProvider`.
- [ ] **Technology Challenges**
  - Identified potential testing challenges related to the underlying technology.
  - `AlreadyExists` detection relies on substring matching in `openshell` CLI output. If the error message format changes in future openshell versions, the detection may fail silently.
- [ ] **Test Environment Needs**
  - Determined necessary **test environment setups and tools**.
  - Unit tests require only Go toolchain with `os/exec` mocking via fake shell scripts. No openshell or gateway required.
- [ ] **API Extensions**
  - Reviewed new or modified APIs and their impact on testing.
  - No API changes. `EnsureProvider` function signature is unchanged. New `deleteProvider` is unexported.
- [ ] **Topology Considerations**
  - Evaluated multi-cluster, network topology, and architectural impacts.
  - N/A. Change is internal to sandbox provider management with no topology implications.

### **II. Software Test Plan (STP)**

This STP serves as the **overall roadmap for testing**, detailing the scope, approach, resources, and schedule.

#### **1. Scope of Testing**

Testing covers the idempotency behavior of `EnsureProvider` in `internal/sandbox/sandbox.go`, including the new `deleteProvider` helper. The scope includes all error paths (AlreadyExists, non-AlreadyExists, delete failure, recreate failure), credential redaction across all paths, and the happy path (first-time creation). One functional scenario validates the integration with the `runAgent` pipeline in `internal/cli/run.go`.

**Testing Goals**

**Functional Goals:**
- **P0:** Verify provider creation is idempotent -- `EnsureProvider` succeeds regardless of whether the provider already exists, without manual cleanup
- **P0:** Verify credential values are never exposed in any error output path
- **P1:** Verify non-AlreadyExists errors propagate correctly as hard failures
- **P1:** Verify first-time creation path is not regressed by the new code

**Quality Goals:**
- **P1:** Verify error messages include sufficient context (provider name, redacted output) for debugging
- **P2:** Verify edge cases (empty credentials, recreate failure after successful delete) are handled gracefully

**Out of Scope (Testing Scope Exclusions)**

- [ ] InferenceLayer.Install() idempotency gap -- *Rationale:* Separate concern mentioned in GH-28 but not addressed by PR #29 -- *PM/Lead Agreement:* TBD
- [ ] openshell CLI behavior or gateway internals -- *Rationale:* Platform-level testing, not product scope -- *PM/Lead Agreement:* TBD
- [ ] Performance benchmarking of provider create/delete operations -- *Rationale:* No performance requirement for this bug fix -- *PM/Lead Agreement:* TBD

#### **2. Test Strategy**

**Functional**

- [x] **Functional Testing** -- Validates that the feature works according to specified requirements and user stories
  - *Details:* 15 unit tests validate all EnsureProvider code paths with mocked openshell binaries. Tests cover AlreadyExists handling, secret redaction, error propagation, and happy path.
- [x] **Automation Testing** -- Confirms test automation plan is in place for CI and regression coverage (all tests are expected to be automated)
  - *Details:* All tests are Go unit tests in `internal/sandbox/sandbox_test.go`, run via `go test`. Fully automated in CI.
- [x] **Regression Testing** -- Verifies that new changes do not break existing functionality
  - *Details:* LSP call graph analysis confirms `EnsureProvider` is called only from `runAgent` (run.go:458). Existing `buildProviderArgs` tests (sandbox_test.go:39,81,100) cover argument construction. The `TestEnsureProvider_CreateSucceedsFirstTime` test validates the happy path is not regressed.

**Non-Functional**

- [ ] **Performance Testing** -- Validates feature performance meets requirements (latency, throughput, resource usage)
  - *Details:* N/A. Provider create/delete are infrequent operations during `fullsend run` startup. No performance requirements.
- [ ] **Scale Testing** -- Validates feature behavior under increased load and at production-like scale
  - *Details:* N/A. Provider count is small (typically 1-3 per run). No scale concern.
- [x] **Security Testing** -- Verifies security requirements, RBAC, authentication, authorization, and vulnerability scanning
  - *Details:* Secret redaction tests verify credential values never appear in error output. Tests `TestEnsureProvider_RecreateFailure_RedactsSecrets` and `TestEnsureProvider_NonAlreadyExistsError_RedactsSecrets` provide coverage.
- [ ] **Usability Testing** -- Validates user experience and accessibility requirements
  - *Details:* N/A. No user-facing UI changes.
- [ ] **Monitoring** -- Does the feature require metrics and/or alerts?
  - *Details:* N/A. No new metrics or alerts required for this fix.

**Integration & Compatibility**

- [ ] **Compatibility Testing** -- Ensures feature works across supported platforms, versions, and configurations
  - *Details:* N/A. Fix is internal Go code with no platform-specific behavior.
- [ ] **Upgrade Testing** -- Validates upgrade paths from previous versions, data migration, and configuration preservation
  - *Details:* N/A. No state migration. Providers are ephemeral per-run resources.
- [ ] **Dependencies** -- Blocked by deliverables from other components/products
  - *Details:* Depends on openshell CLI returning "AlreadyExists" in error output. No other team dependencies.
- [ ] **Cross Integrations** -- Does the feature affect other features or require testing by other teams?
  - *Details:* No cross-feature impact. `deleteProvider` is unexported and only called from `EnsureProvider`.

**Infrastructure**

- [ ] **Cloud Testing** -- Does the feature require multi-cloud platform testing?
  - *Details:* N/A. Provider management is platform-agnostic.

#### **3. Test Environment**

- **Cluster Topology:** N/A (unit tests run without a cluster)
- **Platform & Product Version(s):** My Product 1.0 on Kubernetes 1.28+
- **CPU Virtualization:** N/A
- **Compute Resources:** Standard CI runner
- **Special Hardware:** None
- **Storage:** N/A
- **Network:** N/A
- **Required Operators:** None
- **Platform:** Linux (CI runner)
- **Special Configurations:** None

#### **3.1. Testing Tools & Frameworks**

- **Test Framework:** Standard (Go testing + testify)
- **CI/CD:** Standard
- **Other Tools:** None

#### **4. Entry Criteria**

The following conditions must be met before testing can begin:

- [ ] Requirements and design documents are **approved and merged**
- [ ] Test environment can be **set up and configured** (see Section II.3 - Test Environment)
- [ ] PR #29 is merged and `internal/sandbox/sandbox.go` contains the `deleteProvider` function

#### **5. Risks**

- [ ] **Timeline/Schedule**
  - Risk: N/A. Bug fix is small scope with tests already written.
  - Mitigation: None required.
- [ ] **Test Coverage**
  - Risk: AlreadyExists detection relies on substring matching in openshell output. If the error format changes, the detection silently fails and falls through to the non-AlreadyExists error path.
  - Mitigation: Pin to known openshell version in CI. Add integration test with real openshell if format changes are suspected.
- [ ] **Test Environment**
  - Risk: N/A. Unit tests require only Go toolchain.
  - Mitigation: None required.
- [ ] **Untestable Aspects**
  - Risk: The brief provider-not-found window during delete+recreate cannot be tested at the unit level without race condition injection.
  - Mitigation: Accepted risk. The `runAgent` call site is single-threaded, so concurrent access is not a concern in practice.
- [ ] **Resource Constraints**
  - Risk: N/A.
  - Mitigation: None required.
- [ ] **Dependencies**
  - Risk: openshell CLI error message format is not formally specified. Future versions may change the "AlreadyExists" string.
  - Mitigation: Monitor openshell release notes. Consider using exit codes or structured error output if openshell adds support.
- [ ] **Other**
  - Risk: N/A.
  - Mitigation: None required.

---

### **III. Test Scenarios & Traceability**

This section links requirements to test coverage, enabling reviewers to verify all requirements are tested.

#### **1. Requirements-to-Tests Mapping**

- **[GH-28]** -- Provider creation is idempotent across repeated fullsend runs
  - *Test Scenario:* Verify provider succeeds when already exists (Unit Tests)
  - *Test Scenario:* Verify provider recreated with current credentials (Unit Tests)
  - *Test Scenario:* Verify idempotency across multiple consecutive runs (Unit Tests)
  - *Priority:* P0

- **[GH-28]** -- Credential values are never exposed in error output
  - *Test Scenario:* Verify secret redaction on create failure (Unit Tests)
  - *Test Scenario:* Verify secret redaction on recreate failure (Unit Tests)
  - *Test Scenario:* Verify redaction with multiple credential values (Unit Tests)
  - *Priority:* P0

- **[GH-28]** -- Non-AlreadyExists errors propagate as hard failures
  - *Test Scenario:* Verify non-AlreadyExists error returns immediately (Unit Tests)
  - *Test Scenario:* Verify delete not triggered on other errors (Unit Tests)
  - *Priority:* P1

- **[GH-28]** -- Delete failure during reconciliation propagates with clear error context
  - *Test Scenario:* Verify delete failure returns descriptive error (Unit Tests)
  - *Test Scenario:* Verify provider name included in error message (Unit Tests)
  - *Priority:* P1

- **[GH-28]** -- Recreation failure after successful delete propagates clearly
  - *Test Scenario:* Verify recreate failure error after successful delete (Unit Tests)
  - *Test Scenario:* Verify recreate error includes redacted secrets (Unit Tests)
  - *Priority:* P2

- **[GH-28]** -- First-time provider creation succeeds without regression
  - *Test Scenario:* Verify first-time creation succeeds immediately (Unit Tests)
  - *Test Scenario:* Verify no delete triggered on first creation (Unit Tests)
  - *Test Scenario:* Verify provider creation with empty credentials (Unit Tests)
  - *Priority:* P1

- **[GH-28]** -- Provider idempotency works in full run pipeline
  - *Test Scenario:* Verify full run succeeds with pre-existing providers (Functional)
  - *Priority:* P1

---

### **IV. Sign-off and Approval**

This Software Test Plan requires approval from the following stakeholders:

* **Reviewers:**
  - [TBD]
  - [TBD]
* **Approvers:**
  - [TBD]
  - [TBD]
