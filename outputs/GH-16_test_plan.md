# Fullsend Test Plan

## **GCP: Remove Project from Number Call - Quality Engineering Plan**

### **Metadata & Tracking**

- **Enhancement(s):** [GH-16](https://github.com/guyoron1/fullsend/pull/16)
- **Feature Tracking:** GH-16
- **Epic Tracking:** N/A
- **QE Owner(s):** QualityFlow (automated)
- **Owning SIG:** sig-gcp
- **Participating SIGs:** N/A

**Document Conventions (if applicable):** Standard QualityFlow STP format. Test IDs follow `TS-GH-16-NNN` pattern.

### **Feature Overview**

This PR modifies the `GetProjectNumber` method in `internal/dispatch/gcf/gcp.go` to create a shallow copy of the `gcp.Client` struct with an empty `QuotaProject` field before making the Cloud Resource Manager (CRM) API call. This removes the requirement for users to enable `cloudresourcemanager` permissions on their GCP project, simplifying the onboarding experience.

The CRM API is a global Google API that does not require the `x-goog-user-project` header. By clearing the `QuotaProject` on the copy, the outgoing request omits this header, avoiding quota-project permission checks on the target project.

**Mirrored from upstream [PR #2231](https://github.com/fullsend-ai/fullsend/pull/2231).**

---

### **I. Motivation and Requirements Review (QE Review Guidelines)**

#### **1. Requirement & User Story Review Checklist**

- [x] **Review Requirements**
  - Reviewed the PR description and diff. The change is a targeted fix to the `GetProjectNumber` method.
  - The motivation is clear: avoid requiring users to enable the Cloud Resource Manager API on their GCP project.
- [x] **Understand Value and Customer Use Cases**
  - Users provisioning fullsend on GCP were required to enable `cloudresourcemanager` permissions. This fix removes that friction.
  - Value: Simplified GCP onboarding with fewer IAM/API prerequisites.
- [x] **Testability**
  - The change is highly testable via unit tests that verify the `x-goog-user-project` header is absent in CRM requests.
  - The shallow-copy pattern is verifiable by checking the original client's `QuotaProject` is unchanged after the call.
- [x] **Acceptance Criteria**
  - `GetProjectNumber` must NOT send `x-goog-user-project` header in the CRM API request.
  - The original `gcp.Client`'s `QuotaProject` field must remain unchanged after calling `GetProjectNumber`.
  - Subsequent API calls using the original client must still include `x-goog-user-project`.
  - Error handling (403, 500, network errors) must continue to work correctly.
- [x] **Non-Functional Requirements (NFRs)**
  - No performance impact (shallow struct copy is negligible).
  - No security impact (no credential or token changes).

#### **2. Known Limitations**

- The shallow copy (`noQuotaClient := *c.Client`) shares pointer fields (e.g., `http.Client`, token source) with the original. This is intentional for a single-use, read-only call but could cause issues if the copy were used for mutation-heavy operations.
- No integration test against a live GCP environment is included; the fix relies on unit tests with `httptest` mocks.

#### **3. Technology and Design Review**

- [x] **Developer Handoff/QE Kickoff**
  - The PR description and upstream PR reference provide sufficient context.
- [x] **Technology Challenges**
  - Shallow copy semantics in Go: pointer fields are shared. The test plan validates that this does not cause mutations to the original client.
- [x] **Test Environment Needs**
  - Unit tests only; no special environment needed. Uses `httptest.NewServer` for HTTP mocking.
- [x] **API Extensions**
  - No new APIs. Existing `GetProjectNumber` method signature is unchanged.
- [x] **Topology Considerations**
  - N/A. Single-process, single-goroutine change.

### **II. Software Test Plan (STP)**

#### **1. Scope of Testing**

The scope covers the `GetProjectNumber` method in `internal/dispatch/gcf/gcp.go` and its integration with the provisioning flow via `provisionSelfManaged` in `internal/dispatch/gcf/provisioner.go`.

**Testing Goals**

- Verify the `x-goog-user-project` header is omitted from CRM API requests.
- Verify the original `gcp.Client.QuotaProject` field is not mutated by the shallow copy.
- Verify subsequent API calls using the original client retain the `QuotaProject` value.
- Verify error propagation (403, 500, network failure) works correctly through the copied client.
- Verify the provisioning flow (`provisionSelfManaged`) integrates correctly with the modified `GetProjectNumber`.

**Out of Scope (Testing Scope Exclusions)**

- [x] Live GCP integration testing (no GCP credentials available in test environment)
- [x] Other `LiveGCFClient` methods (unchanged by this PR)
- [x] CLI layer (`internal/cli/admin.go`) testing (not directly affected)
- [x] CLAUDE.md deletion (documentation change, no functional impact)

#### **2. Test Strategy**

**Functional**

- [x] **Functional Testing** -- Validates that `GetProjectNumber` omits the quota project header and returns correct results.
  - *Details:* Unit tests using `httptest.NewServer` to capture outgoing request headers and verify the `x-goog-user-project` header is absent.
- [x] **Automation Testing** -- All tests are automated Go unit tests.
  - *Details:* Tests use `testing` package with `testify/assert` and `testify/require`.
- [x] **Regression Testing** -- Verifies that the shallow copy does not break existing functionality.
  - *Details:* Tests verify the original client is unchanged and subsequent calls still use the original `QuotaProject`.

**Non-Functional**

- [x] **Performance Testing** -- N/A. Shallow struct copy has negligible overhead.
  - *Details:* No performance testing required.
- [ ] **Scale Testing** -- N/A.
  - *Details:* Single API call, no scale considerations.
- [ ] **Security Testing** -- N/A. No credential or token changes.
  - *Details:* The change only affects header inclusion, not authentication flow.
- [ ] **Usability Testing** -- N/A.
  - *Details:* No user-facing changes.
- [ ] **Monitoring** -- N/A.
  - *Details:* No new metrics or alerts.

**Integration & Compatibility**

- [x] **Compatibility Testing** -- N/A. No API signature changes.
  - *Details:* `GetProjectNumber` method signature is unchanged; callers are unaffected.
- [ ] **Upgrade Testing** -- N/A.
  - *Details:* No state migration required.
- [x] **Dependencies** -- None.
  - *Details:* No external dependencies changed.
- [x] **Cross Integrations** -- Provisioning flow integration verified.
  - *Details:* `provisionSelfManaged` calls `GetProjectNumber`; functional tests validate the integration.

**Infrastructure**

- [ ] **Cloud Testing** -- N/A for unit tests.
  - *Details:* Would require live GCP project for integration testing (out of scope).

#### **3. Test Environment**

- **Cluster Topology:** N/A (unit tests only)
- **Platform:** Any platform with Go 1.24+ toolchain
- **CPU Virtualization:** N/A
- **Compute Resources:** Minimal (unit tests)
- **Special Hardware:** None
- **Storage:** None
- **Network:** Loopback only (`httptest.NewServer`)
- **Required Operators:** None
- **Special Configurations:** None

#### **3.1. Testing Tools & Frameworks**

- **Test Framework:** Go `testing` package + `testify` (assert/require)
- **CI/CD:** GitHub Actions
- **Other Tools:** `httptest` for HTTP server mocking

#### **4. Entry Criteria**

The following conditions must be met before testing can begin:

- [x] Requirements and design documents are **approved and merged**
- [x] Test environment can be **set up and configured** (Go toolchain available)
- [x] `go mod tidy` completes without errors
- [x] `make lint` passes

#### **5. Risks**

- [x] **Timeline/Schedule**
  - Risk: Low. Small, targeted change with clear test scope.
  - Mitigation: All tests are automated and run in CI.
- [x] **Test Coverage**
  - Risk: No live GCP integration test.
  - Mitigation: Unit tests with `httptest` mocks cover the header behavior. The upstream PR provides additional validation.
- [x] **Test Environment**
  - Risk: None. Unit tests require only Go toolchain.
  - Mitigation: N/A.
- [x] **Untestable Aspects**
  - Risk: Shallow copy sharing pointer fields (e.g., `http.Client` transport) cannot be fully validated without concurrency tests.
  - Mitigation: The copy is used for a single synchronous call and discarded immediately.
- [x] **Resource Constraints**
  - Risk: None.
  - Mitigation: N/A.
- [x] **Dependencies**
  - Risk: None.
  - Mitigation: N/A.

---

### **III. Test Scenarios & Traceability**

#### **1. Requirements-to-Tests Mapping**

| Test ID | Scenario | Priority | Tier | Requirement | Method |
|:--------|:---------|:---------|:-----|:------------|:-------|
| TS-GH-16-001 | `GetProjectNumber` omits `x-goog-user-project` header when `QuotaProject` is set on the client | P0 | Unit | AC: CRM request must not include quota project header | Capture request headers via `httptest.Server`; assert `x-goog-user-project` is empty |
| TS-GH-16-002 | `GetProjectNumber` returns error on HTTP 403 Permission Denied | P1 | Unit | AC: Error handling for permission denied responses | Mock server returns 403; assert error is returned |
| TS-GH-16-003 | `GetProjectNumber` handles empty `projectNumber` in response | P2 | Unit | AC: Graceful handling of edge cases | Mock server returns `{"projectNumber": ""}`; assert no panic |
| TS-GH-16-004 | Original `gcp.Client.QuotaProject` is unchanged after `GetProjectNumber` call | P0 | Unit | AC: Shallow copy must not mutate original client | Store original value; call `GetProjectNumber`; assert equality |
| TS-GH-16-005 | Subsequent API calls use original `QuotaProject` value | P0 | Unit | AC: Only CRM call omits header; other calls retain it | Make CRM call then regular call; assert header presence on second call |
| TS-GH-16-006 | Self-managed provisioning flow completes with modified `GetProjectNumber` | P1 | Functional | AC: Provisioning integrates correctly with fix | Mock all GCP endpoints; call `Provisioner.Provision`; assert success |
| TS-GH-16-007 | Provisioning aborts when `GetProjectNumber` fails | P1 | Functional | AC: Error propagation through provisioning chain | Mock CRM to return 500; assert `Provision` returns error; no SA creation attempted |
| TS-GH-16-008 | Error propagation from copied `gcp.Client.DoRequest` | P1 | Unit | AC: Errors from shallow copy propagate correctly | Close mock server; assert error on unreachable host |
| TS-GH-16-009 | HTTP 403 error message is descriptive for diagnosis | P2 | Unit | AC: Error messages help users diagnose permission issues | Mock 403 response; assert error contains "403" or "permission" |
| TS-GH-16-010 | OIDC dispatch layer installation works with modified client | P1 | Functional | AC: Full OIDC install chain works correctly | Mock GCP endpoints; call `Provisioner.Provision` with OIDC config; assert success |

#### **Call Graph (LSP Regression Analysis)**

```
internal/cli/admin.go:508
  └─> gcf.Provisioner (provisioner.go:97)
        └─> provisionSelfManaged (provisioner.go:282)
              └─> GetProjectNumber (gcp.go:886)  ← MODIFIED
                    └─> noQuotaClient.DoRequest (shallow copy, QuotaProject="")
```

**Impact radius:** The change is contained within `GetProjectNumber`. No other methods on `LiveGCFClient` are affected. The only production caller is `provisionSelfManaged`, which is invoked during the self-managed GCP provisioning flow from the CLI admin command.

---

### **IV. Sign-off and Approval**

This Software Test Plan requires approval from the following stakeholders:

* **Reviewers:**
  - QualityFlow (automated)
* **Approvers:**
  - @guyoron1
