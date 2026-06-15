# FullSend Test Plan

## **fix(gcp): Remove Quota Project from Project Number Lookup - Quality Engineering Plan**

### **Metadata & Tracking**

- **Enhancement(s):** [GH-11](https://github.com/guyoron1/fullsend/pull/11)
- **Feature Tracking:** [GH-11](https://github.com/guyoron1/fullsend/pull/11)
- **Epic Tracking:** N/A
- **QE Owner(s):** TBD
- **Owning SIG:** N/A
- **Participating SIGs:** None

**Document Conventions (if applicable):** N/A

### **Feature Overview**

This bug fix modifies the `GetProjectNumber` method in the GCF dispatch provisioner to omit the `x-goog-user-project` quota header when calling the GCP Cloud Resource Manager API. The change removes the requirement for the `cloudresourcemanager` API to be enabled on the target GCP project, simplifying GCP project permissions for FullSend deployments. The fix creates a copy of the `gcp.Client` with the quota project cleared before making the API call, ensuring the original client is unmodified.

---

### **I. Motivation and Requirements Review (QE Review Guidelines)**

This section documents the mandatory QE review process. The goal is to understand the feature's value,
technology, and testability before formal test planning.

#### **1. Requirement & User Story Review Checklist**

- [ ] **Review Requirements**
  - Reviewed the relevant requirements.
  - PR description states the fix removes the need to enable `cloudresourcemanager` permissions on the GCP project. Mirrored from upstream [PR #2231](https://github.com/fullsend-ai/fullsend/pull/2231).
- [ ] **Understand Value and Customer Use Cases**
  - Confirmed clear user stories and understood.
  - Understand the difference between community and product requirements.
  - **What is the value of the feature for customers**.
  - Ensured requirements contain relevant **customer use cases**.
  - Customers deploying FullSend with self-managed GCP infrastructure no longer need to enable the Cloud Resource Manager API on their target project, reducing permission complexity and setup friction.
- [ ] **Testability**
  - Confirmed requirements are **testable and unambiguous**.
  - The change is testable via unit tests using httptest mock servers. Existing test coverage in `gcp_test.go` covers the `GetProjectNumber` method with success, empty response, and error scenarios. **Note:** LSP analysis detected that `gcp.Client` has no `QuotaProject` field — this is a compilation issue that must be resolved before testing.
- [ ] **Acceptance Criteria**
  - Ensured acceptance criteria are **defined clearly** (clear user stories; product requirements clearly defined in Jira).
  - Implicit acceptance: `GetProjectNumber` must succeed without requiring `cloudresourcemanager` API enabled on the target project. The original client must not be mutated.
- [ ] **Non-Functional Requirements (NFRs)**
  - Confirmed coverage for NFRs, including Performance, Security, Usability, Downtime, Connectivity, Monitoring (alerts/metrics), Scalability, Portability (e.g., cloud support), and Docs.
  - No significant NFR changes. The fix is a targeted behavioral change to a single API call with no performance, security, or scalability implications.

#### **2. Known Limitations**

- **Compilation issue:** LSP analysis confirms `gcp.Client` (defined at `internal/gcp/client.go:20`) has only `httpClient` and `tokenFunc` fields. The PR references `noQuotaClient.QuotaProject` which does not exist on the struct. This fix will not compile as written and requires the `QuotaProject` field to be added to `gcp.Client` or an alternative approach to omit the quota project header.
- The PR is mirrored from upstream PR #2231, suggesting the upstream repo may have a `QuotaProject` field that has not yet been synced to this fork.

#### **3. Technology and Design Review**

- [ ] **Developer Handoff/QE Kickoff**
  - A meeting where Dev/Arch walked QE through the design, architecture, and implementation details. **Critical for identifying untestable aspects early.**
  - The change is straightforward: create a value copy of `*c.Client`, clear the quota project field, and use the copy for the CRM API call. The original client remains unmodified.
- [ ] **Technology Challenges**
  - Identified potential testing challenges related to the underlying technology.
  - The `gcp.Client` struct uses unexported fields (`httpClient`, `tokenFunc`), so a value copy shares the underlying `*http.Client` pointer. This is acceptable since only the `QuotaProject` field (once added) needs to differ.
- [ ] **Test Environment Needs**
  - Determined necessary **test environment setups and tools**.
  - Unit tests require only httptest mock servers. No GCP credentials or live API access needed for test execution.
- [ ] **API Extensions**
  - Reviewed new or modified APIs and their impact on testing.
  - No API changes. The `GetProjectNumber` method signature is unchanged. The `GCFClient` interface method `GetProjectNumber(ctx context.Context, projectID string) (string, error)` remains the same.
- [ ] **Topology Considerations**
  - Evaluated multi-cluster, network topology, and architectural impacts.
  - N/A. This is a single API call modification with no topology implications.

---

### **II. Software Test Plan (STP)**

This STP serves as the **overall roadmap for testing**, detailing the scope, approach, resources, and schedule.

#### **1. Scope of Testing**

Testing focuses on verifying that the `GetProjectNumber` method correctly omits the quota project header when calling the GCP Cloud Resource Manager API, and that the change does not regress existing provisioning workflows. The scope includes the modified function, its caller (`provisionSelfManaged`), and client copy isolation.

**Testing Goals**

**Functional Goals:**
- **P0:** Verify `GetProjectNumber` successfully retrieves project numbers without requiring `cloudresourcemanager` API enabled on the target project
- **P0:** Verify the original `gcp.Client` is not mutated after `GetProjectNumber` call
- **P1:** Verify error handling paths remain functional (forbidden, empty response)

**Quality Goals:**
- **P1:** Verify client value copy does not introduce shared mutable state issues

**Integration Goals:**
- **P1:** Verify `provisionSelfManaged` workflow continues to function with the modified `GetProjectNumber`

**Out of Scope (Testing Scope Exclusions)**

- [ ] GCP Cloud Resource Manager API availability and behavior -- *Rationale:* Platform-level GCP API testing is outside FullSend product scope -- *PM/Lead Agreement:* TBD
- [ ] OAuth2/ADC token acquisition -- *Rationale:* Token management is tested independently in `gcp.Client` and is not modified by this PR -- *PM/Lead Agreement:* TBD
- [ ] WIF pool/provider provisioning -- *Rationale:* Downstream provisioning steps are not modified; only the project number lookup changes -- *PM/Lead Agreement:* TBD

#### **2. Test Strategy**

**Functional**

- [x] **Functional Testing** — Validates that the feature works according to specified requirements and user stories
  - *Details:* Verify `GetProjectNumber` calls CRM API without quota project header. Verify correct project number returned on success. Verify error paths for forbidden and empty responses.
- [x] **Automation Testing** — Confirms test automation plan is in place for CI and regression coverage (all tests are expected to be automated)
  - *Details:* All tests are Go unit tests using httptest mock servers, fully automated in CI via `go test ./internal/dispatch/gcf/...`.
- [x] **Regression Testing** — Verifies that new changes do not break existing functionality
  - *Details:* Existing `TestLiveGCFClient_GetProjectNumber` test cases (success, empty, error) provide regression coverage. Additional test for client isolation needed.

**Non-Functional**

- [ ] **Performance Testing** — Validates feature performance meets requirements (latency, throughput, resource usage)
  - *Details:* N/A. Single API call modification with no performance impact.
- [ ] **Scale Testing** — Validates feature behavior under increased load and at production-like scale
  - *Details:* N/A. No scale implications for a single API call change.
- [ ] **Security Testing** — Verifies security requirements, RBAC, authentication, authorization, and vulnerability scanning
  - *Details:* N/A. No security model changes. Authentication mechanism unchanged.
- [ ] **Usability Testing** — Validates user experience and accessibility requirements
  - *Details:* N/A. No user-facing changes.
- [ ] **Monitoring** — Does the feature require metrics and/or alerts?
  - *Details:* N/A. No new metrics or alerts needed.

**Integration & Compatibility**

- [ ] **Compatibility Testing** — Ensures feature works across supported platforms, versions, and configurations
  - *Details:* N/A. Go API change only, no platform-specific behavior.
- [ ] **Upgrade Testing** — Validates upgrade paths from previous versions
  - *Details:* N/A. No data migration or configuration changes.
- [ ] **Dependencies** — Blocked by deliverables from other components/products
  - *Details:* Depends on `gcp.Client` struct having a `QuotaProject` field (currently missing — see Known Limitations).
- [ ] **Cross Integrations** — Does the feature affect other features or require testing by other teams?
  - *Details:* The `provisionSelfManaged` function in `provisioner.go` is the sole caller. No cross-team impact.

**Infrastructure**

- [ ] **Cloud Testing** — Does the feature require multi-cloud platform testing?
  - *Details:* N/A. GCP-only feature, tested via mocked HTTP servers.

#### **3. Test Environment**

- **Cluster Topology:** N/A (unit tests only, no cluster required)
- **Platform & Product Version(s):** FullSend 0.x, GitHub Actions
- **CPU Virtualization:** N/A
- **Compute Resources:** Standard CI runner
- **Special Hardware:** N/A
- **Storage:** N/A
- **Network:** N/A (httptest mock servers)
- **Required Operators:** N/A
- **Platform:** GitHub Actions
- **Special Configurations:** N/A

#### **3.1. Testing Tools & Frameworks**

- **Test Framework:** Standard Go testing (no new tools)
- **CI/CD:** Standard (no new tools)
- **Other Tools:** None

#### **4. Entry Criteria**

The following conditions must be met before testing can begin:

- [ ] Requirements and design documents are **approved and merged**
- [ ] Test environment can be **set up and configured** (see Section II.3 - Test Environment)
- [ ] `gcp.Client` struct includes `QuotaProject` field (compilation prerequisite)
- [ ] PR compiles successfully (`go build ./...`)

#### **5. Risks**

- [ ] **Timeline/Schedule**
  - Risk: Compilation failure blocks all testing until `QuotaProject` field is added to `gcp.Client`
  - Mitigation: Sync `gcp.Client` changes from upstream PR #2231 or add the field to this PR
- [ ] **Test Coverage**
  - Risk: Existing tests do not verify the quota project header is omitted from requests
  - Mitigation: Add test assertion that verifies the request headers sent to the mock server
- [ ] **Test Environment**
  - Risk: N/A — unit tests require no special environment
  - Mitigation: N/A
- [ ] **Untestable Aspects**
  - Risk: Live GCP API behavior with/without quota project header cannot be verified in CI
  - Mitigation: Manual verification in a GCP project without `cloudresourcemanager` API enabled
- [ ] **Resource Constraints**
  - Risk: N/A
  - Mitigation: N/A
- [ ] **Dependencies**
  - Risk: Upstream PR #2231 may contain additional changes to `gcp.Client` not yet synced
  - Mitigation: Review upstream PR diff to identify all required changes
- [ ] **Other**
  - Risk: Value copy of `gcp.Client` shares `*http.Client` pointer — concurrent modifications to the HTTP client could cause races
  - Mitigation: The shared `*http.Client` is only used for reads (no concurrent field modification), so this is safe in practice

---

### **III. Test Scenarios & Traceability**

This section links requirements to test coverage, enabling reviewers to verify all requirements are tested.

#### **1. Requirements-to-Tests Mapping**

- **[GH-11]** -- GCP project number lookup succeeds without `cloudresourcemanager` API enabled on target project
  - *Test Scenario:* Verify project number lookup without quota project header [Functional]
  - *Priority:* P0
  - *Test Scenario:* Verify original client unmodified after lookup [Functional]
  - *Priority:* P0
  - *Test Scenario:* Verify error when CRM API returns forbidden [Functional]
  - *Priority:* P1

- **[GH-11]** -- Self-managed provisioning workflow completes with modified project number lookup
  - *Test Scenario:* Verify full provisioning workflow succeeds [Functional]
  - *Priority:* P1
  - *Test Scenario:* Verify provisioning fails gracefully on project number error [Functional]
  - *Priority:* P1

- **[GH-11]** -- Project number lookup returns correct value and handles errors
  - *Test Scenario:* Verify correct project number returned from API response [Functional]
  - *Priority:* P1
  - *Test Scenario:* Verify error for empty project number response [Functional]
  - *Priority:* P2
  - *Test Scenario:* Verify appropriate error with status code for forbidden response [Functional]
  - *Priority:* P1

- **[GH-11]** -- Client copy isolation ensures no mutation of original client state
  - *Test Scenario:* Verify client value copy does not share mutable state [Functional]
  - *Priority:* P1
  - *Test Scenario:* Verify concurrent lookups with isolated clients [Functional]
  - *Priority:* P2

---

### **IV. Sign-off and Approval**

This Software Test Plan requires approval from the following stakeholders:

* **Reviewers:**
  - [TBD / @reviewer]
* **Approvers:**
  - [TBD / @approver]
