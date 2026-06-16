# FullSend Test Plan

## **Make EnsureProvider Idempotent via Delete-and-Recreate - Quality Engineering Plan**

### **Metadata & Tracking**

- **Enhancement(s):** [GH-15](https://github.com/guyoron1/fullsend/pull/15)
- **Feature Tracking:** [GH-15](https://github.com/guyoron1/fullsend/pull/15)
- **Epic Tracking:** GH-15 (upstream: [fullsend-ai/fullsend#2296](https://github.com/fullsend-ai/fullsend/pull/2296))
- **QE Owner(s):** TBD
- **Owning SIG:** N/A
- **Participating SIGs:** None

**Document Conventions (if applicable):** N/A

### **Feature Overview**

This change makes `EnsureProvider` idempotent by detecting `AlreadyExists` errors from the `openshell provider create` command and automatically deleting and recreating the provider with current credentials. Previously, a pre-existing provider from a prior agent run caused a hard failure, requiring manual cleanup between iterations. The change also extracts a `redactSecrets` helper function to centralize credential redaction in error messages, ensuring secrets are never leaked in any error output path.

---

### **I. Motivation and Requirements Review (QE Review Guidelines)**

This section documents the mandatory QE review process. The goal is to understand the feature's value,
technology, and testability before formal test planning.

#### **1. Requirement & User Story Review Checklist**

- [ ] **Review Requirements**
  - Reviewed the relevant requirements.
  - PR description clearly states the problem: `AlreadyExists` error blocked subsequent runs, requiring manual cleanup.
  - Fix scope is well-defined: the `EnsureProvider` function's idempotency handling.
- [ ] **Understand Value and Customer Use Cases**
  - Confirmed clear user stories and understood.
  - Understand the difference between community and product requirements.
  - **What is the value of the feature for customers**.
  - Ensured requirements contain relevant **customer use cases**.
  - Value: eliminates manual provider cleanup between agent runs, enabling fully autonomous iteration.
  - Use case: repeated `fullsend run` invocations on the same sandbox infrastructure without manual intervention.
- [ ] **Testability**
  - Confirmed requirements are **testable and unambiguous**.
  - All paths are testable via fake `openshell` binaries that simulate AlreadyExists, delete failure, and retry failure scenarios.
  - PR already includes 4 unit tests covering the primary paths.
- [ ] **Acceptance Criteria**
  - Ensured acceptance criteria are **defined clearly** (clear user stories; product requirements clearly defined in Jira).
  - AC1: `EnsureProvider` succeeds when provider already exists (delete + recreate).
  - AC2: Credentials are never exposed in error output.
  - AC3: Non-AlreadyExists errors propagate unchanged.
- [ ] **Non-Functional Requirements (NFRs)**
  - Confirmed coverage for NFRs, including Performance, Security, Usability, Downtime, Connectivity, Monitoring (alerts/metrics), Scalability, Portability (e.g., cloud support), and Docs.
  - Security: credential redaction in all error paths is a core NFR for this change.
  - Performance: the delete-and-recreate adds one extra CLI call only on the AlreadyExists path; no impact on happy path.

#### **2. Known Limitations**

- The idempotency strategy is delete-and-recreate, not update-in-place. There is a brief window between delete and recreate where no provider exists.
- If the `openshell provider delete` command fails, the original provider remains and the function returns an error; no rollback is attempted.
- Concurrent calls to `EnsureProvider` for the same provider name are not synchronized; race conditions are possible if called in parallel.

#### **3. Technology and Design Review**

- [ ] **Developer Handoff/QE Kickoff**
  - A meeting where Dev/Arch walked QE through the design, architecture, and implementation details. **Critical for identifying untestable aspects early.**
  - Change is straightforward: single function modification with clear error handling branches. PR diff is self-documenting.
- [ ] **Technology Challenges**
  - Identified potential testing challenges related to the underlying technology.
  - Testing requires fake `openshell` binaries to simulate CLI behavior. PR already demonstrates this pattern with shell scripts in `$TMPDIR`.
- [ ] **Test Environment Needs**
  - Determined necessary **test environment setups and tools**.
  - Unit tests require only Go toolchain. No cluster or real `openshell` binary needed.
  - Regression tests for the agent run integration would require a sandbox gateway.
- [ ] **API Extensions**
  - Reviewed new or modified APIs and their impact on testing.
  - No API changes. `EnsureProvider` function signature is unchanged. New `redactSecrets` is unexported.
- [ ] **Topology Considerations**
  - Evaluated multi-cluster, network topology, and architectural impacts.
  - N/A. Change is host-local CLI interaction only.

### **II. Software Test Plan (STP)**

This STP serves as the **overall roadmap for testing**, detailing the scope, approach, resources, and schedule.

#### **1. Scope of Testing**

Testing covers the idempotent provider creation logic, including AlreadyExists detection, delete-and-recreate flow, all error handling branches, and the centralized credential redaction helper. Integration testing covers the agent run workflow that iterates over provider definitions and calls EnsureProvider, included for regression confidence as the caller itself was not modified in this PR.

**Testing Goals**

**Functional Goals:**
- **P0:** Verify `EnsureProvider` succeeds when the provider already exists (delete + recreate path)
- **P0:** Verify credential values are never exposed in any error message
- **P1:** Verify non-AlreadyExists errors propagate correctly without triggering delete

**Quality Goals:**
- **P0:** Verify `redactSecrets` replaces all known credential values with `***`
- **P1:** Verify error messages include actionable context (provider name, failure phase)

**Integration Goals:**
- **P1:** Verify `runAgent` workflow completes successfully when providers already exist from a prior run

**Out of Scope (Testing Scope Exclusions)**

- [ ] **openshell CLI binary behavior** -- *Rationale:* Third-party tool; we test our integration with it, not its internals. -- *PM/Lead Agreement:* TBD
- [ ] **Gateway lifecycle management** -- *Rationale:* `EnsureGateway` is unchanged and has separate test coverage. -- *PM/Lead Agreement:* TBD
- [ ] **Sandbox creation, deletion, and exec** -- *Rationale:* Not modified in this PR; separate feature with existing coverage. -- *PM/Lead Agreement:* TBD
- [ ] **Provider types and credential formats** -- *Rationale:* `buildProviderArgs` is unchanged; credential format handling is not in scope. -- *PM/Lead Agreement:* TBD

#### **2. Test Strategy**

**Functional**

- [x] **Functional Testing** -- Validates that the feature works according to specified requirements and user stories
  - *Details:* Core testing of the AlreadyExists detection, delete-and-recreate flow, error propagation, and secret redaction across all code paths.
- [x] **Automation Testing** -- Confirms test automation plan is in place for CI and regression coverage (all tests are expected to be automated)
  - *Details:* All tests are automated Go unit tests using `testify`. PR includes 4 tests; STP identifies 14 total scenarios.
- [x] **Regression Testing** -- Verifies that new changes do not break existing functionality
  - *Details:* LSP analysis confirms `EnsureProvider` is called by `runAgent`. Existing `TestEnsureAvailable_OpenshellNotInPath` test verifies unchanged behavior.

**Non-Functional**

- [ ] **Performance Testing** -- Validates feature performance meets requirements (latency, throughput, resource usage)
  - *Details:* N/A. The additional CLI call on AlreadyExists path is negligible overhead.
- [ ] **Scale Testing** -- Validates feature behavior under increased load and at production-like scale
  - *Details:* N/A. Provider creation is a sequential setup step, not a scale-sensitive operation.
- [x] **Security Testing** -- Verifies security requirements, RBAC, authentication, authorization, and vulnerability scanning
  - *Details:* Credential redaction in all error paths is tested explicitly. `redactSecrets` ensures no secret values appear in error output.
- [ ] **Usability Testing** -- Validates user experience and accessibility requirements
  - *Details:* N/A. No user-facing interface changes.
- [ ] **Monitoring** -- Does the feature require metrics and/or alerts?
  - *Details:* N/A. No new metrics or alerts introduced.

**Integration & Compatibility**

- [ ] **Compatibility Testing** -- Ensures feature works across supported platforms, versions, and configurations
  - *Details:* N/A. Function signature unchanged; callers are unaffected.
- [ ] **Upgrade Testing** -- Validates upgrade paths from previous versions, data migration, and configuration preservation
  - *Details:* N/A. No persistent state or configuration migration involved.
- [ ] **Dependencies** -- Blocked by deliverables from other components/products
  - *Details:* Depends on `openshell` CLI being available. Tested via `EnsureAvailable` (unchanged).
- [ ] **Cross Integrations** -- Does the feature affect other features or require testing by other teams?
  - *Details:* The agent run workflow is the sole caller. No cross-team impact.

**Infrastructure**

- [ ] **Cloud Testing** -- Does the feature require multi-cloud platform testing?
  - *Details:* N/A. Host-local CLI interaction only.

#### **3. Test Environment**

- **Cluster Topology:** N/A (unit tests run locally, no cluster required)
- **Platform & Product Version(s):** Go 1.23+, GitHub Actions runners
- **CPU Virtualization:** N/A
- **Compute Resources:** Standard CI runner
- **Special Hardware:** None
- **Storage:** N/A
- **Network:** N/A
- **Required Operators:** None
- **Platform:** GitHub Actions
- **Special Configurations:** Fake `openshell` shell scripts in `$TMPDIR` for unit tests

#### **3.1. Testing Tools & Frameworks**

- **Test Framework:** Standard
- **CI/CD:** Standard
- **Other Tools:** None

#### **4. Entry Criteria**

The following conditions must be met before testing can begin:

- [ ] Requirements and design documents are **approved and merged**
- [ ] Test environment can be **set up and configured** (see Section II.3 - Test Environment)
- [ ] Go toolchain (1.23+) is available on CI runners
- [ ] PR branch is rebased on current main

#### **5. Risks**

- [ ] **Timeline/Schedule**
  - Risk: Low risk. Change is small and well-scoped.
  - Mitigation: PR already includes unit tests for all primary paths.
- [ ] **Test Coverage**
  - Risk: Functional-level integration tests for `runAgent` with pre-existing providers may require sandbox gateway access.
  - Mitigation: Unit tests with fake `openshell` binaries provide equivalent coverage for the idempotency logic.
- [ ] **Test Environment**
  - Risk: N/A. Unit tests run on standard CI without special infrastructure.
  - Mitigation: N/A.
- [ ] **Untestable Aspects**
  - Risk: Real `openshell` provider timing between delete and recreate cannot be tested without a live gateway.
  - Mitigation: The timing window is inherent to the delete-and-recreate approach and is documented as a known limitation.
- [ ] **Resource Constraints**
  - Risk: N/A. No special resources required.
  - Mitigation: N/A.
- [ ] **Dependencies**
  - Risk: Dependency on `openshell` CLI behavior for `AlreadyExists` error string matching.
  - Mitigation: Error string is stable and documented. Tests use the exact string format.
- [ ] **Other**
  - Risk: Concurrent `EnsureProvider` calls for the same provider name could race.
  - Mitigation: Current usage in `runAgent` is sequential (loop over provider definitions). Document as known limitation if parallel provider setup is added later.

---

### **III. Test Scenarios & Traceability**

This section links requirements to test coverage, enabling reviewers to verify all requirements are tested.

#### **1. Requirements-to-Tests Mapping**

- **[GH-15]** -- Provider creation is idempotent across repeated agent runs
  - *Test Scenario:* TS-GH-15-001: Verify provider recreated when AlreadyExists returned
  - *Tier:* Unit Tests
  - *Priority:* P0

- **[GH-15]** -- Provider creation is idempotent across repeated agent runs
  - *Test Scenario:* TS-GH-15-002: Verify error when delete fails during recreate
  - *Tier:* Unit Tests
  - *Priority:* P0

- **[GH-15]** -- Provider creation is idempotent across repeated agent runs
  - *Test Scenario:* TS-GH-15-003: Verify error when retry create fails after delete
  - *Tier:* Unit Tests
  - *Priority:* P0

- **[GH-15]** -- Provider recreation uses current credentials, not stale ones
  - *Test Scenario:* TS-GH-15-004: Verify recreated provider uses current credentials and environment (fresh credentials applied, environment variables re-expanded on recreate)
  - *Tier:* Unit Tests
  - *Priority:* P0

- **[GH-15]** -- Non-AlreadyExists provider creation errors propagate correctly
  - *Test Scenario:* TS-GH-15-006: Verify non-AlreadyExists error returned without delete
  - *Tier:* Unit Tests
  - *Priority:* P1

- **[GH-15]** -- Non-AlreadyExists provider creation errors propagate correctly
  - *Test Scenario:* TS-GH-15-007: Verify original error message preserved in output
  - *Tier:* Unit Tests
  - *Priority:* P1

- **[GH-15]** -- Provider delete failure during recreate is reported with redacted secrets
  - *Test Scenario:* TS-GH-15-008: Verify delete error does not leak credentials
  - *Tier:* Unit Tests
  - *Priority:* P1

- **[GH-15]** -- Provider create failure after delete is reported with redacted secrets
  - *Test Scenario:* TS-GH-15-009: Verify retry create error does not leak credentials
  - *Tier:* Unit Tests
  - *Priority:* P1

- **[GH-15]** -- Secret values are never exposed in any error output
  - *Test Scenario:* TS-GH-15-010: Verify redactSecrets replaces all credential values
  - *Tier:* Unit Tests
  - *Priority:* P0

- **[GH-15]** -- Secret values are never exposed in any error output
  - *Test Scenario:* TS-GH-15-011: Verify redaction with multiple secret values
  - *Tier:* Unit Tests
  - *Priority:* P0

- **[GH-15]** -- Secret values are never exposed in any error output
  - *Test Scenario:* TS-GH-15-012: Verify redaction with empty secrets list
  - *Tier:* Unit Tests
  - *Priority:* P2

- **[GH-15]** -- Non-AlreadyExists provider creation errors include redacted credentials
  - *Test Scenario:* TS-GH-15-015: Verify credentials are redacted in non-AlreadyExists error output
  - *Tier:* Unit Tests
  - *Priority:* P1

- **[GH-15]** -- Agent run workflow handles idempotent provider setup without interruption
  - *Test Scenario:* TS-GH-15-013: Verify agent run succeeds with pre-existing providers
  - *Tier:* Regression
  - *Priority:* P1

- **[GH-15]** -- Agent run workflow handles idempotent provider setup without interruption
  - *Test Scenario:* TS-GH-15-014: Verify agent run fails fast on non-idempotent error
  - *Tier:* Regression
  - *Priority:* P1

---

### **IV. Sign-off and Approval**

This Software Test Plan requires approval from the following stakeholders:

* **Reviewers:**
  - [TBD / @reviewer]
* **Approvers:**
  - [TBD / @approver]
