# Test Plan

## **Batch Path-Existence Checks via Git Trees API - Quality Engineering Plan**

### **Metadata & Tracking**

- **Enhancement(s):** [GH-2351](https://github.com/fullsend-ai/fullsend/issues/2351)
- **Feature Tracking:** [GH-2351](https://github.com/fullsend-ai/fullsend/issues/2351)
- **Epic Tracking:** GH-2351 (standalone)
- **QE Owner(s):** TBD
- **Owning SIG:** N/A
- **Participating SIGs:** N/A

**Document Conventions (if applicable):** N/A

### **Feature Overview**

This change replaces the O(N) sequential `GetFileContent` calls in `scaffold.ComparePathPresence` with a single batch `ListRepositoryFiles` call using the GitHub Git Trees API. The new `forge.Client.ListRepositoryFiles` method retrieves all file paths in a repository's default branch via `refs -> commit -> tree?recursive=1`, reducing 100+ sequential API calls to 3 fixed calls regardless of path count. This improves analyze latency and reduces rate-limit pressure for organizations with large vendored installs.

---

### **I. Motivation and Requirements Review (QE Review Guidelines)**

This section documents the mandatory QE review process. The goal is to understand the feature's value,
technology, and testability before formal test planning.

#### **1. Requirement & User Story Review Checklist**

- [ ] **Review Requirements**
  - Reviewed the relevant requirements.
  - Issue GH-2351 describes the performance problem: `ComparePathPresence` checks ~50 vendored paths with individual `GetFileContent` calls, producing 100+ sequential API calls per analyze run.
  - PR #1954 introduced the naive implementation; this change provides the batch replacement.
- [ ] **Understand Value and Customer Use Cases**
  - Confirmed clear user stories and understood.
  - Understand the difference between community and product requirements.
  - **What is the value of the feature for customers**.
  - Ensured requirements contain relevant **customer use cases**.
  - Users running `vendor analyze` on repos with vendored binaries experience unnecessary latency and rate-limit pressure. This fix benefits orgs with large vendored installs.
- [ ] **Testability**
  - Confirmed requirements are **testable and unambiguous**.
  - All changes are in pure Go code with `forge.FakeClient` test doubles. The batch behavior is verifiable by injecting errors on `GetFileContent` to ensure it is never called.
- [ ] **Acceptance Criteria**
  - Ensured acceptance criteria are **defined clearly** (clear user stories; product requirements clearly defined in Jira).
  - Acceptance criteria: `ComparePathPresence` must use `ListRepositoryFiles` (batch) instead of per-path `GetFileContent`. API call count must be O(1) regardless of path count.
- [ ] **Non-Functional Requirements (NFRs)**
  - Confirmed coverage for NFRs, including Performance, Security, Usability, Downtime, Connectivity, Monitoring (alerts/metrics), Scalability, Portability (e.g., cloud support), and Docs.
  - Primary NFR is performance: reducing API calls from O(N) to O(1). Thread safety of `FakeClient` is verified via mutex and concurrent access tests.

#### **2. Known Limitations**

- The Git Trees API returns a `truncated: true` flag for very large repositories (>100k files). `ListRepositoryFiles` treats this as an error rather than returning partial results — callers must handle this case.
- `ComparePathPresence` is not yet called from production code. Integration with `VendorBinaryLayer.Analyze` depends on PR #1954 merging and adopting the batch implementation.
- The current implementation fetches the entire repository tree. For repos where only a small subtree is relevant, this may transfer more data than necessary.

#### **3. Technology and Design Review**

- [ ] **Developer Handoff/QE Kickoff**
  - A meeting where Dev/Arch walked QE through the design, architecture, and implementation details. **Critical for identifying untestable aspects early.**
  - The implementation reuses the same refs/commits/trees Git API pattern already used by `CommitFiles` in `github.LiveClient`. The new method adds a `?recursive=1` parameter to retrieve all paths at once.
- [ ] **Technology Challenges**
  - Identified potential testing challenges related to the underlying technology.
  - The `LiveClient` implementation requires a real GitHub API or `httptest` server to test. Unit tests use `forge.FakeClient` which derives paths from map keys.
- [ ] **Test Environment Needs**
  - Determined necessary **test environment setups and tools**.
  - Standard Go test environment with `go test`. No special infrastructure required — all tests use in-memory mocks.
- [ ] **API Extensions**
  - Reviewed new or modified APIs and their impact on testing.
  - `forge.Client` interface extended with `ListRepositoryFiles(ctx, owner, repo) ([]string, error)`. Both `LiveClient` and `FakeClient` implement the new method. All existing interface consumers must be updated if they implement `Client` directly.
- [ ] **Topology Considerations**
  - Evaluated multi-cluster, network topology, and architectural impacts.
  - No topology impact. The change is purely client-side API call optimization.

### **II. Software Test Plan (STP)**

This STP serves as the **overall roadmap for testing**, detailing the scope, approach, resources, and schedule.

#### **1. Scope of Testing**

Testing covers the new `ListRepositoryFiles` method on the `forge.Client` interface (both `LiveClient` and `FakeClient` implementations), the rewritten `scaffold.ComparePathPresence` function, and the interface compliance of both client implementations.

**Testing Goals**

- **P0:** Verify `ComparePathPresence` correctly identifies missing paths using batch API and never calls `GetFileContent`
- **P0:** Verify `ListRepositoryFiles` returns all blob paths and handles truncated trees as errors
- **P1:** Verify `FakeClient.ListRepositoryFiles` correctly derives paths from `FileContents` map keys
- **P1:** Verify error propagation through the call chain with proper context wrapping
- **P2:** Verify edge cases (empty input, all-missing, concurrent access)

**Out of Scope (Testing Scope Exclusions)**

- [ ] GitHub API rate limiting and retry behavior
  - Covered by existing `retryOnTransient` infrastructure tests, not new to this change
- [ ] Git Trees API pagination/limits
  - Platform-level GitHub API behavior, not product-testable
- [ ] Integration with `VendorBinaryLayer.Analyze`
  - Production caller integration depends on PR #1954 merge; out of scope for this STP
- [ ] `GetFileContent` callers in `layers/` package
  - 24 existing references across 11 files are unchanged; tested by their own test suites

#### **2. Test Strategy**

**Functional**

- [ ] **Functional Testing** -- Validates that the feature works according to specified requirements and user stories
  - *Details:* Unit tests verify `ComparePathPresence` correctness (all-present, some-missing, all-missing, empty-input) and `ListRepositoryFiles` implementations.
- [ ] **Automation Testing** -- Confirms test automation plan is in place for CI and regression coverage (all tests are expected to be automated)
  - *Details:* All tests are standard Go unit tests run via `go test`. 6 tests for `ComparePathPresence`, additional tests for `FakeClient` and `LiveClient`.
- [ ] **Regression Testing** -- Verifies that new changes do not break existing functionality
  - *Details:* The `TestComparePathPresence_UsesOneAPICall` test acts as a regression guard — it injects an error on `GetFileContent` to ensure the batch pattern is never replaced with the O(N) pattern.

**Non-Functional**

- [ ] **Performance Testing** -- Validates feature performance meets requirements (latency, throughput, resource usage)
  - *Details:* The primary purpose of this change is performance improvement (O(N) to O(1) API calls). Performance is validated architecturally via the API-call-count guard test rather than benchmarks.
- [ ] **Scale Testing** -- Validates feature behavior under increased load and at production-like scale
  - *Details:* Not applicable. Scale benefit is inherent in the O(1) API call design.
- [ ] **Security Testing** -- Verifies security requirements, RBAC, authentication, authorization, and vulnerability scanning
  - *Details:* Not applicable. No new authentication or authorization changes.
- [ ] **Usability Testing** -- Validates user experience and accessibility requirements
  - *Details:* Not applicable. Internal API change with no user-facing interface.
- [ ] **Monitoring** -- Does the feature require metrics and/or alerts?
  - *Details:* Not applicable. No new metrics or alerts.

**Integration & Compatibility**

- [ ] **Compatibility Testing** -- Ensures feature works across supported platforms, versions, and configurations
  - *Details:* `ListRepositoryFiles` uses the standard GitHub Git Trees API (v3), which is stable and widely supported.
- [ ] **Upgrade Testing** -- Validates upgrade paths from previous versions
  - *Details:* Not applicable. The `forge.Client` interface change is internal; no external API contracts change.
- [ ] **Dependencies** -- Blocked by deliverables from other components/products
  - *Details:* Production integration blocked by PR #1954 merge. The batch implementation is ready to replace the naive `ComparePathPresence` once #1954 lands.
- [ ] **Cross Integrations** -- Does the feature affect other features or require testing by other teams?
  - *Details:* The `forge.Client` interface extension affects all implementations. `FakeClient` (test double) is updated. Any third-party `Client` implementations would need to add `ListRepositoryFiles`.

**Infrastructure**

- [ ] **Cloud Testing** -- Does the feature require multi-cloud platform testing?
  - *Details:* Not applicable. GitHub API is the only forge backend.

#### **3. Test Environment**

- **Cluster Topology:** N/A (unit tests only, no cluster required)
- **Platform & Product Version(s):** Go 1.26.0 (per go.mod)
- **CPU Virtualization:** N/A
- **Compute Resources:** Standard CI runner
- **Special Hardware:** None
- **Storage:** N/A
- **Network:** N/A (all tests use in-memory mocks)
- **Required Operators:** None
- **Platform:** Linux (CI), any OS for local development
- **Special Configurations:** None

#### **3.1. Testing Tools & Frameworks**

No new or special tools required. Standard Go testing infrastructure:

- **Test Framework:** `testing` (stdlib) + `testify` (assert/require)
- **CI/CD:** Standard `go test` pipeline
- **Other Tools:** None

#### **4. Entry Criteria**

The following conditions must be met before testing can begin:

- [ ] Requirements and design documents are **approved and merged**
- [ ] Test environment can be **set up and configured** (see Section II.3 - Test Environment)
- [ ] `forge.Client` interface changes are finalized and compile-time checks pass
- [ ] `FakeClient` implements `ListRepositoryFiles` for test double usage

#### **5. Risks**

- [ ] **Timeline/Schedule**
  - Risk: Production integration depends on PR #1954 merge timing
  - Mitigation: Batch implementation is self-contained and tested independently
- [ ] **Test Coverage**
  - Risk: `LiveClient.ListRepositoryFiles` cannot be tested without a real GitHub API or httptest mock
  - Mitigation: `FakeClient` provides comprehensive test coverage; LiveClient uses same patterns as existing tested methods
- [ ] **Test Environment**
  - Risk: None identified for unit tests
  - Mitigation: N/A
- [ ] **Untestable Aspects**
  - Risk: GitHub Git Trees API truncation behavior for very large repos (>100k files) cannot be triggered in unit tests
  - Mitigation: Error path for `truncated: true` is explicitly tested with mock response
- [ ] **Resource Constraints**
  - Risk: None identified
  - Mitigation: N/A
- [ ] **Dependencies**
  - Risk: `forge.Client` interface change is a breaking change for any external implementations
  - Mitigation: No known external implementations; `FakeClient` and `LiveClient` are the only implementations
- [ ] **Other**
  - Risk: None identified
  - Mitigation: N/A

---

### **III. Test Scenarios & Traceability**

This section links requirements to test coverage, enabling reviewers to verify all requirements are tested.

#### **1. Requirements-to-Tests Mapping**

- **Requirement ID:** GH-2351
- **Requirement:** Batch path-existence checks reduce API calls from O(N) to O(1)
- **Evidence:** `ComparePathPresence` -> `ListRepositoryFiles` replaces N x `GetFileContent`
- **Test Scenarios:**
  - Verify ComparePathPresence returns correct missing paths (positive)
  - Verify all paths reported present when all exist (positive)
  - Verify sorted missing paths when some absent (positive)
  - Verify GetFileContent is never called by ComparePathPresence (positive)
  - Verify error propagation from ListRepositoryFiles failure (negative)
- **Tier:** Unit Tests
- **Priority:** P0

---

- **Requirement ID:** GH-2351
- **Requirement:** ListRepositoryFiles retrieves all file paths via Git Trees API
- **Evidence:** `LiveClient.ListRepositoryFiles` uses refs -> commit -> tree?recursive=1 (3 API calls)
- **Test Scenarios:**
  - Verify ListRepositoryFiles returns all blob paths (positive)
  - Verify tree entries (directories) are excluded from results (positive)
  - Verify error when repository tree is truncated (negative)
  - Verify error propagation for invalid repo (negative)
- **Tier:** Unit Tests
- **Priority:** P0

---

- **Requirement ID:** GH-2351
- **Requirement:** FakeClient.ListRepositoryFiles derives paths from FileContents map
- **Evidence:** `FakeClient` strips "owner/repo/" prefix from FileContents keys
- **Test Scenarios:**
  - Verify FakeClient returns correct relative paths (positive)
  - Verify FakeClient returns empty list for empty map (positive)
  - Verify FakeClient respects error injection (negative)
- **Tier:** Unit Tests
- **Priority:** P1

---

- **Requirement ID:** GH-2351
- **Requirement:** ComparePathPresence handles edge cases correctly
- **Evidence:** Early return for empty input, sorted output, thread-safe FakeClient
- **Test Scenarios:**
  - Verify empty expected list short-circuits without API calls (positive)
  - Verify all-missing paths returned sorted (positive)
  - Verify concurrent ListRepositoryFiles calls are thread-safe (positive)
- **Tier:** Unit Tests
- **Priority:** P1

---

- **Requirement ID:** GH-2351
- **Requirement:** forge.Client interface extended with ListRepositoryFiles
- **Evidence:** New method on `Client` interface; compile-time checks for `FakeClient` and `LiveClient`
- **Test Scenarios:**
  - Verify FakeClient satisfies Client interface (positive)
  - Verify LiveClient satisfies Client interface (positive)
- **Tier:** Unit Tests
- **Priority:** P1

---

### **IV. Sign-off and Approval**

This Software Test Plan requires approval from the following stakeholders:

* **Reviewers:**
  - [TBD / @tbd]
* **Approvers:**
  - [TBD / @tbd]
