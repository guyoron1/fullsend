# FullSend Test Plan

## **Fix Hardcoded /tmp/repo in Agent Run Tests - Quality Engineering Plan**

### **Metadata & Tracking**

- **Enhancement(s):** [GH-30](https://github.com/guyoron1/fullsend/issues/30)
- **Feature Tracking:** [GH-30](https://github.com/guyoron1/fullsend/issues/30)
- **Epic Tracking:** GH-30 (standalone bug fix)
- **QE Owner(s):** TBD
- **Owning SIG:** N/A
- **Participating SIGs:** None

**Document Conventions (if applicable):** N/A

### **Feature Overview**

This bug fix addresses sporadic test failures in `TestRunAgent_HarnessLoadPipeline` and related test functions in `internal/cli/run_test.go`. The tests hardcode `/tmp/repo` as the repository directory passed to `runAgent`, but when that path does not exist on the test machine, the tarball creation step in `sandbox.UploadDir` fails before execution reaches the expected `openshell` sandbox availability check. The fix replaces all hardcoded `/tmp/repo` references with `t.TempDir()` to ensure test hermeticity and isolation.

---

### **I. Motivation and Requirements Review (QE Review Guidelines)**

This section documents the mandatory QE review process. The goal is to understand the feature's value,
technology, and testability before formal test planning.

#### **1. Requirement & User Story Review Checklist**

- [ ] **Review Requirements**
  - Reviewed the relevant requirements.
  - Issue GH-30 clearly describes the sporadic failure, root cause (hardcoded `/tmp/repo`), and expected fix (`t.TempDir()`).
- [ ] **Understand Value and Customer Use Cases**
  - Confirmed clear user stories and understood.
  - Understand the difference between community and product requirements.
  - **What is the value of the feature for customers**.
  - Ensured requirements contain relevant **customer use cases**.
  - This fix improves CI reliability by eliminating a non-hermetic test dependency. Developers and CI systems benefit from deterministic test outcomes.
- [ ] **Testability**
  - Confirmed requirements are **testable and unambiguous**.
  - The fix is directly testable: running the affected tests with `-count=5` should produce zero flakes regardless of whether `/tmp/repo` exists on the host.
- [ ] **Acceptance Criteria**
  - Ensured acceptance criteria are **defined clearly** (clear user stories; product requirements clearly defined in Jira).
  - Acceptance: all affected tests reach the expected `openshell` or `harness file not found` error assertions consistently, never the tar error.
- [ ] **Non-Functional Requirements (NFRs)**
  - Confirmed coverage for NFRs, including Performance, Security, Usability, Downtime, Connectivity, Monitoring (alerts/metrics), Scalability, Portability (e.g., cloud support), and Docs.
  - No NFR impact. This is a test-only change with no production code modifications.

#### **2. Known Limitations**

- The fix ensures the repo directory exists but does not populate it with real project files. Tests still rely on the `openshell` check failing before any meaningful sandbox operation occurs.
- `t.TempDir()` directories are automatically cleaned up when the test completes; no persistent state is verified across test runs.

#### **3. Technology and Design Review**

- [ ] **Developer Handoff/QE Kickoff**
  - A meeting where Dev/Arch walked QE through the design, architecture, and implementation details. **Critical for identifying untestable aspects early.**
  - PR #33 is self-explanatory: a mechanical replacement of `/tmp/repo` with `t.TempDir()` across 10 test functions.
- [ ] **Technology Challenges**
  - Identified potential testing challenges related to the underlying technology.
  - No technology challenges. `t.TempDir()` is a standard Go testing library function available since Go 1.15.
- [ ] **Test Environment Needs**
  - Determined necessary **test environment setups and tools**.
  - Standard Go test environment (`go test`). No special setup required.
- [ ] **API Extensions**
  - Reviewed new or modified APIs and their impact on testing.
  - No API changes. Only test code is modified.
- [ ] **Topology Considerations**
  - Evaluated multi-cluster, network topology, and architectural impacts.
  - No topology considerations. Tests run locally without cluster interaction.

### **II. Software Test Plan (STP)**

This STP serves as the **overall roadmap for testing**, detailing the scope, approach, resources, and schedule.

#### **1. Scope of Testing**

Testing verifies that replacing hardcoded `/tmp/repo` with `t.TempDir()` in `internal/cli/run_test.go` eliminates sporadic test failures while preserving the original test intent. The scope covers all 10 affected test functions that call `runAgent` and the upstream `sandbox.UploadDir` function that creates the tarball.

**Testing Goals**

**Functional Goals:**

- **P0:** Verify all affected `runAgent` tests consistently reach their expected error assertions (openshell check or harness-not-found) without tar failures
- **P0:** Verify test isolation — each test uses its own temporary directory independent of host filesystem state
- **P1:** Verify `sandbox.UploadDir` correctly handles valid directory paths for tarball creation

**Quality Goals:**

- **P1:** Verify tests pass deterministically with `-count=5` to confirm flakiness is eliminated

**Out of Scope (Testing Scope Exclusions)**

- [ ] **Production `runAgent` behavior** -- *Rationale:* No production code is changed; only test setup code is modified -- *PM/Lead Agreement:* N/A
- [ ] **Sandbox creation and execution** -- *Rationale:* Tests never reach sandbox creation; they fail at the openshell availability check by design -- *PM/Lead Agreement:* N/A
- [ ] **Other test files in `internal/cli/`** -- *Rationale:* Only `run_test.go` contains the hardcoded path; other test files are unaffected -- *PM/Lead Agreement:* N/A

#### **2. Test Strategy**

**Functional**

- [ ] **Functional Testing** -- Validates that the feature works according to specified requirements and user stories
  - *Details:* Run the 10 affected test functions to confirm they reach expected error paths. Applicable.
- [ ] **Automation Testing** -- Confirms test automation plan is in place for CI and regression coverage (all tests are expected to be automated)
  - *Details:* All tests are already automated Go unit tests run via `go test`. No additional automation needed. Applicable.
- [ ] **Regression Testing** -- Verifies that new changes do not break existing functionality
  - *Details:* Run `go test ./internal/cli/ -count=1` to confirm no regressions in the full test suite. Applicable.

**Non-Functional**

- [ ] **Performance Testing** -- Validates feature performance meets requirements (latency, throughput, resource usage)
  - *Details:* N/A. Test-only change with no performance implications.
- [ ] **Scale Testing** -- Validates feature behavior under increased load and at production-like scale
  - *Details:* N/A. No production code changes.
- [ ] **Security Testing** -- Verifies security requirements, RBAC, authentication, authorization, and vulnerability scanning
  - *Details:* N/A. No security-relevant changes.
- [ ] **Usability Testing** -- Validates user experience and accessibility requirements
  - *Details:* N/A. No user-facing changes.
- [ ] **Monitoring** -- Does the feature require metrics and/or alerts?
  - *Details:* N/A. No monitoring changes.

**Integration & Compatibility**

- [ ] **Compatibility Testing** -- Ensures feature works across supported platforms, versions, and configurations
  - *Details:* `t.TempDir()` is available in Go 1.15+; FullSend requires Go 1.23+. No compatibility concern.
- [ ] **Upgrade Testing** -- Validates upgrade paths from previous versions, data migration, and configuration preservation
  - *Details:* N/A. No upgrade impact.
- [ ] **Dependencies** -- Blocked by deliverables from other components/products
  - *Details:* No external dependencies. Uses only Go standard library.
- [ ] **Cross Integrations** -- Does the feature affect other features or require testing by other teams?
  - *Details:* N/A. Change is isolated to test code.

**Infrastructure**

- [ ] **Cloud Testing** -- Does the feature require multi-cloud platform testing?
  - *Details:* N/A. Tests run locally without cloud resources.

#### **3. Test Environment**

- **Cluster Topology:** N/A (no cluster required)
- **Platform & Product Version(s):** Go 1.23+, any OS with POSIX temp directory support
- **CPU Virtualization:** N/A
- **Compute Resources:** Standard developer workstation or CI runner
- **Special Hardware:** N/A
- **Storage:** Standard filesystem with temp directory support
- **Network:** N/A (no network access required)
- **Required Operators:** N/A
- **Platform:** GitHub Actions (CI), Linux/macOS (local)
- **Special Configurations:** None

#### **3.1. Testing Tools & Frameworks**

- **Test Framework:** N/A (standard Go testing + testify)
- **CI/CD:** N/A (standard GitHub Actions)
- **Other Tools:** N/A

#### **4. Entry Criteria**

The following conditions must be met before testing can begin:

- [ ] Requirements and design documents are **approved and merged**
- [ ] Test environment can be **set up and configured** (see Section II.3 - Test Environment)
- [ ] PR #33 changes are available on the test branch

#### **5. Risks**

- [ ] **Timeline/Schedule**
  - Risk: N/A. Fix is a single-file, mechanical change already implemented in PR #33.
  - Mitigation: N/A.
- [ ] **Test Coverage**
  - Risk: Low. The fix touches 10 test functions; all must be validated.
  - Mitigation: Run all affected tests with `-count=5` to confirm deterministic behavior.
- [ ] **Test Environment**
  - Risk: N/A. Tests use `t.TempDir()` which works on all Go-supported platforms.
  - Mitigation: N/A.
- [ ] **Untestable Aspects**
  - Risk: The original flake depended on `/tmp/repo` not existing, which is hard to reproduce deterministically.
  - Mitigation: The fix removes the dependency entirely rather than trying to control the host state.
- [ ] **Resource Constraints**
  - Risk: N/A. No additional resources required.
  - Mitigation: N/A.
- [ ] **Dependencies**
  - Risk: N/A. No external dependencies.
  - Mitigation: N/A.
- [ ] **Other**
  - Risk: N/A.
  - Mitigation: N/A.

---

### **III. Test Scenarios & Traceability**

This section links requirements to test coverage, enabling reviewers to verify all requirements are tested.

#### **1. Requirements-to-Tests Mapping**

- **[GH-30]** -- Agent run tests use isolated temporary directories instead of hardcoded /tmp/repo
  - *Test Scenario:* Verify harness-load-pipeline test reaches openshell error without tar failure
  - *Tier:* Functional
  - *Priority:* P0

- **[GH-30]** -- Tests pass regardless of host filesystem state
  - *Test Scenario:* Verify all runAgent tests pass without pre-existing /tmp/repo directory
  - *Tier:* Functional
  - *Priority:* P0

- **[GH-30]** -- Tests pass when /tmp/repo happens to exist on host
  - *Test Scenario:* Verify all runAgent tests pass with pre-existing /tmp/repo directory
  - *Tier:* Functional
  - *Priority:* P1

- **[GH-30]** -- YML fallback harness resolution uses isolated directory
  - *Test Scenario:* Verify YML fallback test resolves harness and reaches expected error
  - *Tier:* Functional
  - *Priority:* P1

- **[GH-30]** -- Harness-not-found test uses isolated directory
  - *Test Scenario:* Verify harness-not-found returns descriptive error message
  - *Tier:* Functional
  - *Priority:* P0

- **[GH-30]** -- Test isolation across parallel execution
  - *Test Scenario:* Verify concurrent test runs do not share or collide on temp directories
  - *Tier:* Functional
  - *Priority:* P1

- **[GH-30]** -- UploadDir handles valid directory for tarball creation
  - *Test Scenario:* Verify UploadDir succeeds when given an existing directory path
  - *Tier:* Functional
  - *Priority:* P1

- **[GH-30]** -- UploadDir reports clear error for missing directory
  - *Test Scenario:* Verify UploadDir fails gracefully with descriptive error for non-existent path
  - *Tier:* Functional
  - *Priority:* P1

---

### **IV. Sign-off and Approval**

This Software Test Plan requires approval from the following stakeholders:

* **Reviewers:**
  - [TBD / @reviewer]
* **Approvers:**
  - [TBD / @approver]
