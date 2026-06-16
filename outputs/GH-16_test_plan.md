# FullSend Test Plan

## **fix(gcp): remove the project from the number call - Quality Engineering Plan**

### **Metadata & Tracking**

- **Enhancement(s):** [Upstream PR #2231](https://github.com/fullsend-ai/fullsend/pull/2231) ([Fork PR #16](https://github.com/guyoron1/fullsend/pull/16))
- **Feature Tracking:** [Upstream PR #2231](https://github.com/fullsend-ai/fullsend/pull/2231)
- **Epic Tracking:** N/A
- **QE Owner(s):** TBD
- **Owning SIG:** N/A
- **Participating SIGs:** None

**Document Conventions (if applicable):** N/A

### **Feature Overview**

This fix modifies the `GetProjectNumber` method on `LiveGCFClient` in the GCF dispatch provisioner to avoid requiring the `cloudresourcemanager` API to be enabled on the target GCP project. The change creates a shallow copy of the embedded `gcp.Client` and clears the `QuotaProject` field so that the `x-goog-user-project` header is omitted when calling the Cloud Resource Manager API. This reduces the permission footprint required for FullSend's OIDC mint provisioning flow.

---

### **I. Motivation and Requirements Review (QE Review Guidelines)**

This section documents the mandatory QE review process. The goal is to understand the feature's value,
technology, and testability before formal test planning.

#### **1. Requirement & User Story Review Checklist**

- [ ] **Review Requirements**
  - Reviewed the relevant requirements.
  - PR removes quota project header from CRM API calls to eliminate a permission dependency. Mirrored from upstream PR #2231.
- [ ] **Understand Value and Customer Use Cases**
  - Confirmed clear user stories and understood.
  - Understand the difference between community and product requirements.
  - **What is the value of the feature for customers**.
  - Ensured requirements contain relevant **customer use cases**.
  - Users provisioning FullSend on GCP projects no longer need to enable the Cloud Resource Manager API, reducing setup friction and permission requirements.
- [ ] **Testability**
  - Confirmed requirements are **testable and unambiguous**.
  - The change is testable via unit tests with mock HTTP servers verifying that the `x-goog-user-project` header is absent in CRM requests. Existing test infrastructure (`newTestClient`, `httptest.Server`) supports this.
- [ ] **Acceptance Criteria**
  - Ensured acceptance criteria are **defined clearly** (clear user stories; product requirements clearly defined in Jira).
  - AC1: `GetProjectNumber` succeeds without CRM API enabled on target project. AC2: `x-goog-user-project` header is omitted for CRM calls. AC3: Other `LiveGCFClient` methods continue using the original client with `QuotaProject`. AC4: Existing tests pass.
- [ ] **Non-Functional Requirements (NFRs)**
  - Confirmed coverage for NFRs, including Performance, Security, Usability, Downtime, Connectivity, Monitoring (alerts/metrics), Scalability, Portability (e.g., cloud support), and Docs.
  - No performance, security, or scalability impact. The shallow struct copy is negligible overhead. No new monitoring or documentation requirements.

#### **2. Known Limitations**

- **Compilation issue detected:** LSP analysis reveals that `gcp.Client` does not currently have a `QuotaProject` field. The PR sets `noQuotaClient.QuotaProject = ""` on a type that lacks this field, which would cause a compilation error. This PR likely depends on an upstream change to `internal/gcp/client.go` that adds the `QuotaProject` field to the `Client` struct and wires it into `DoRequest` as the `x-goog-user-project` header. This dependency must be resolved before the PR can be merged.
- **Shallow copy limitation:** The fix uses a value copy (`noQuotaClient := *c.Client`) which copies pointer fields (`httpClient`, `tokenFunc`) by reference. This is safe for the current struct layout but could introduce shared-state bugs if `Client` gains mutable pointer fields in the future.

#### **3. Technology and Design Review**

- [ ] **Developer Handoff/QE Kickoff**
  - A meeting where Dev/Arch walked QE through the design, architecture, and implementation details. **Critical for identifying untestable aspects early.**
  - Single-function change with clear intent. The fix pattern (copy struct, clear field, use copy) is straightforward and well-understood in Go.
- [ ] **Technology Challenges**
  - Identified potential testing challenges related to the underlying technology.
  - Verifying the absence of an HTTP header requires inspection of outgoing requests in test servers. Existing `httptest.Server` infrastructure supports this.
  - Internal call chain exercised: `installOIDC` → `Provision` → `provisionSelfManaged` → `GetProjectNumber` → `gcp.Client.DoRequest`. The fix targets the `GetProjectNumber` entry point which creates a shallow copy of the embedded `gcp.Client` and clears the `QuotaProject` field.
- [ ] **Test Environment Needs**
  - Determined necessary **test environment setups and tools**.
  - No special environment needed. Standard Go test infrastructure with `httptest` servers is sufficient.
- [ ] **API Extensions**
  - Reviewed new or modified APIs and their impact on testing.
  - No API changes. The `GCFClient` interface is unchanged. The modification is internal to `GetProjectNumber` implementation.
- [ ] **Topology Considerations**
  - Evaluated multi-cluster, network topology, and architectural impacts.
  - N/A. No topology or deployment considerations for this bug fix.

### **II. Software Test Plan (STP)**

This STP serves as the **overall roadmap for testing**, detailing the scope, approach, resources, and schedule.

#### **1. Scope of Testing**

Testing covers the GCP project number lookup used during OIDC provisioning and verifies that the Cloud Resource Manager API call no longer requires the `cloudresourcemanager` API to be enabled on the target project. Focus areas: (1) quota project header omission for CRM requests, (2) no regression in the provisioning flow, (3) original client state integrity after the lookup.

**Testing Goals**

**Functional Goals:**
- **P0:** Verify the GCP project number lookup omits the `x-goog-user-project` header when calling the CRM API
- **P0:** Verify the original GCP client is not mutated by the project number lookup
- **P1:** Verify error handling paths work correctly with the copied client

**Quality Goals:**
- **P1:** Verify no regression in the provisioning flow that depends on `GetProjectNumber`

**Integration Goals:**
- **P1:** Verify the OIDC dispatch layer installation succeeds through the modified provisioning path

**Out of Scope (Testing Scope Exclusions)**

- [ ] **GCP Cloud Resource Manager API behavior** -- *Rationale:* External Google API; tested by Google. We test our client's request construction, not CRM's response handling. -- *PM/Lead Agreement:* TBD
- [ ] **gcp.Client authentication and token acquisition** -- *Rationale:* Unmodified code path; ADC token flow is out of scope for this fix. -- *PM/Lead Agreement:* TBD
- [ ] **Other LiveGCFClient methods (CreateServiceAccount, CreateWIFPool, etc.)** -- *Rationale:* Not modified by this PR. Regression risk is limited to shared client mutation, which is covered by TS-GH-16-004/005. -- *PM/Lead Agreement:* TBD

#### **2. Test Strategy**

**Functional**

- [x] **Functional Testing** — Validates that the feature works according to specified requirements and user stories
  - *Details:* Unit tests verify `GetProjectNumber` omits quota project header and handles errors correctly. Functional tests verify provisioning flow integration.
- [x] **Automation Testing** — Confirms test automation plan is in place for CI and regression coverage (all tests are expected to be automated)
  - *Details:* All tests are Go unit/integration tests runnable via `go test ./internal/dispatch/gcf/...`. Existing CI pipeline covers this package.
- [x] **Regression Testing** — Verifies that new changes do not break existing functionality
  - *Details:* Existing `TestLiveGCFClient_GetProjectNumber` and `TestProvisioner_Provision_*` test suites cover regression. New tests extend coverage for quota project isolation.

**Non-Functional**

- [ ] **Performance Testing** — Validates feature performance meets requirements (latency, throughput, resource usage)
  - *Details:* N/A. Shallow struct copy adds negligible overhead (nanoseconds).
- [ ] **Scale Testing** — Validates feature behavior under increased load and at production-like scale
  - *Details:* N/A. Single API call, no scale dimension.
- [ ] **Security Testing** — Verifies security requirements, RBAC, authentication, authorization, and vulnerability scanning
  - *Details:* N/A. The change reduces permissions required (removes a header), improving security posture. No new attack surface.
- [ ] **Usability Testing** — Validates user experience and accessibility requirements
  - *Details:* N/A. No user-facing change.
- [ ] **Monitoring** — Does the feature require metrics and/or alerts?
  - *Details:* N/A. No new metrics or alerts needed.

**Integration & Compatibility**

- [ ] **Compatibility Testing** — Ensures feature works across supported platforms, versions, and configurations
  - *Details:* N/A. GCP CRM v1 API is stable. No version-specific behavior.
- [ ] **Upgrade Testing** — Validates upgrade paths from previous versions, data migration, and configuration preservation
  - *Details:* N/A. No persistent state or configuration changes.
- [x] **Dependencies** — Blocked by deliverables from other components/products
  - *Details:* Depends on `QuotaProject` field being added to `gcp.Client` struct in `internal/gcp/client.go`. This upstream change must land first.
- [ ] **Cross Integrations** — Does the feature affect other features or require testing by other teams?
  - *Details:* The `DoRequest` method is shared by `internal/inference/vertex/gcp.go` (Vertex AI client). Those calls are unaffected since the copy is local to `GetProjectNumber`.

**Infrastructure**

- [ ] **Cloud Testing** — Does the feature require multi-cloud platform testing?
  - *Details:* N/A. GCP-only change, testable with mock HTTP servers.

#### **3. Test Environment**

- **Cluster Topology:** N/A (no cluster required)
- **Platform & Product Version(s):** FullSend 0.x, Go 1.23+
- **CPU Virtualization:** N/A
- **Compute Resources:** Standard CI runner — no GCP credentials required; tests use mock HTTP servers
- **Special Hardware:** N/A
- **Storage:** N/A
- **Network:** N/A (mock HTTP servers used)
- **Required Operators:** N/A
- **Platform:** GitHub Actions — mock-based tests, no cloud provider access needed
- **Special Configurations:** N/A

#### **3.1. Testing Tools & Frameworks**

- **Test Framework:** None beyond project standard
- **CI/CD:** None beyond project standard
- **Other Tools:** None

#### **4. Entry Criteria**

The following conditions must be met before testing can begin:

- [ ] Requirements and design documents are **approved and merged**
- [ ] Test environment can be **set up and configured** (see Section II.3 - Test Environment)
- [ ] `QuotaProject` field added to `gcp.Client` struct and wired into `DoRequest` header logic
- [ ] PR compiles successfully (`go build ./...` passes)

#### **5. Risks**

- [ ] **Timeline/Schedule**
  - Risk: Dependency on upstream `gcp.Client` change may delay testing
  - Mitigation: Coordinate with upstream to ensure `QuotaProject` field is added promptly
- [ ] **Test Coverage**
  - Risk: Existing unit tests do not verify HTTP header content (only response handling)
  - Mitigation: Add test that inspects request headers via `httptest.Server` handler
- [ ] **Test Environment**
  - Risk: N/A — no special environment needed
  - Mitigation: N/A
- [ ] **Untestable Aspects**
  - Risk: Cannot verify actual GCP CRM API behavior without live credentials
  - Mitigation: Mock-based tests verify request construction; live validation deferred to manual smoke test
- [ ] **Resource Constraints**
  - Risk: N/A — minimal test resources needed
  - Mitigation: N/A
- [ ] **Dependencies**
  - Risk: PR will not compile until `gcp.Client.QuotaProject` field exists
  - Mitigation: Track upstream change; block merge until dependency is met
- [ ] **Other**
  - Risk: Shallow copy may share mutable pointer fields if `Client` struct evolves
  - Mitigation: Document copy semantics; consider adding a `Clone()` method in the future

---

### **III. Test Scenarios & Traceability**

This section links requirements to test coverage, enabling reviewers to verify all requirements are tested.

#### **1. Requirements-to-Tests Mapping**

- **[GH-16]** -- GCP project number lookup succeeds without cloudresourcemanager API enabled on the target project
  - *Test Scenario:* Verify project number lookup omits quota project header (TS-GH-16-001)
  - *Tier:* Unit Tests
  - *Priority:* P0
  - *Test Scenario:* Verify lookup fails gracefully on permission denied (TS-GH-16-002)
  - *Tier:* Unit Tests
  - *Priority:* P1
  - *Test Scenario:* Verify lookup handles empty project number response (TS-GH-16-003)
  - *Tier:* Unit Tests
  - *Priority:* P2

- **[GH-16]** -- QuotaProject clearing does not mutate the original GCFClient's embedded Client
  - *Test Scenario:* Verify original client unchanged after GetProjectNumber (TS-GH-16-004)
  - *Tier:* Unit Tests
  - *Priority:* P0
  - *Test Scenario:* Verify subsequent API calls use original QuotaProject (TS-GH-16-005)
  - *Tier:* Unit Tests
  - *Priority:* P0

- **[GH-16]** -- Self-managed provisioning flow completes with modified GetProjectNumber
  - *Test Scenario:* Verify full provisioning succeeds end-to-end (TS-GH-16-006)
  - *Tier:* Functional
  - *Priority:* P1
  - *Test Scenario:* Verify provisioning aborts on project number error (TS-GH-16-007)
  - *Tier:* Functional
  - *Priority:* P1

- **[GH-16]** -- Error handling works correctly with the modified client
  - *Test Scenario:* Verify error propagation from copied client DoRequest (TS-GH-16-008)
  - *Tier:* Unit Tests
  - *Priority:* P1
  - *Test Scenario:* Verify HTTP 403 returns descriptive error message (TS-GH-16-009)
  - *Tier:* Unit Tests
  - *Priority:* P2

- **[GH-16]** -- OIDC dispatch layer installation succeeds through modified provisioning path
  - *Test Scenario:* Verify dispatch layer install with modified GCP client (TS-GH-16-010)
  - *Tier:* Functional
  - *Priority:* P1

---

### **IV. Sign-off and Approval**

This Software Test Plan requires approval from the following stakeholders:

* **Reviewers:**
  - [TBD / @tbd]
* **Approvers:**
  - [TBD / @tbd]
