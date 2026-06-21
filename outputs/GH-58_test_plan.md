# FullSend Test Plan

## **[Restore Data Consistency Guard in EnsureOrgInMint] - Quality Engineering Plan**

### Metadata & Tracking

- **Enhancement(s)** -- [GH-58](https://github.com/guyoron1/fullsend/pull/58) (Mirror of upstream fullsend-ai/fullsend#2436)
- **Feature Tracking** -- GH-58: fix(#2433): restore data consistency guard in EnsureOrgInMint
- **Epic Tracking** -- GH-58
- **QE Owner(s)** -- TBD
- **Owning SIG** -- N/A
- **Participating SIGs** -- None

**Document Conventions:** N/A

### Feature Overview

This change restores a defense-in-depth cross-check in the `EnsureOrgInMint` function that prevents silent clobbering of the `ALLOWED_ORGS` environment variable on stale reads from the Cloud Run traffic-serving revision. When `ALLOWED_ORGS` is empty but `ROLE_APP_IDS` contains role-only entries, the mint has already been bootstrapped and the empty value indicates env var data loss rather than a first enrollment. The guard aborts the operation to prevent silently unenrolling all existing orgs, adapted for the role-only model where legacy org/role keys are filtered out via `RoleOnlyAppIDs`.

---

### Section I - Motivation and Requirements Review

#### Section I.1 - Requirement & User Story Review Checklist

- [ ] **Review Requirements**
  - Reviewed the relevant requirements.
  - PR restores a data consistency guard removed during role-only migration. The fix prevents `ALLOWED_ORGS` from being silently overwritten when the traffic-serving revision returns stale/empty data.

- [ ] **Understand Value and Customer Use Cases**
  - Confirmed clear user stories and understood. Understand the value and customer use cases.
  - Prevents accidental unenrollment of all orgs from a shared mint instance. Without this guard, a stale read during `fullsend mint enroll-org` could silently drop all existing org enrollments.

- [ ] **Testability**
  - Confirmed requirements are **testable and unambiguous**.
  - The guard logic is fully testable via the existing `fakeGCFClient` test infrastructure. Three distinct guard scenarios are covered: active role-only keys (error), empty ROLE_APP_IDS (proceed), and legacy-only keys (proceed).

- [ ] **Acceptance Criteria**
  - Ensured acceptance criteria are **defined clearly**.
  - When `ALLOWED_ORGS` is empty and `ROLE_APP_IDS` has role-only entries, `EnsureOrgInMint` must return an error containing "data inconsistency" and the count of configured roles. When `ROLE_APP_IDS` is empty or has only legacy keys, enrollment must proceed normally.

- [ ] **Non-Functional Requirements**
  - Confirmed coverage for NFRs.
  - No performance, scalability, or security NFRs changed. The guard adds a single JSON unmarshal and map iteration on a small env var value during enrollment -- negligible overhead.

#### Section I.2 - Known Limitations

- The consistency guard only detects empty `ALLOWED_ORGS`. Partial data loss (e.g., some orgs missing but not all) is not detected by this guard.
- The read-modify-write in `EnsureOrgInMint` is not locked; concurrent enrollment calls sharing the same mint can still race. This is documented and accepted as a known limitation.
- `GetServiceTrafficEnvVars` reads from the observed traffic-serving revision, which may lag behind a just-deployed revision during Cloud Run rollout.

#### Section I.3 - Technology and Design Review

- [ ] **Developer Handoff/QE Kickoff**
  - Review developer handoff for technical details on the data consistency guard implementation.
  - The fix modifies `EnsureOrgInMint` in `internal/dispatch/gcf/provisioner.go` and depends on `RoleOnlyAppIDs` from `internal/mintcore/handler.go`.

- [ ] **Technology Challenges**
  - Identify any technology challenges or new dependencies.
  - Uses `GetServiceTrafficEnvVars` to read from the Cloud Run traffic-serving revision instead of the function's env vars, avoiding stale data from diverged revisions.

- [ ] **Test Environment Needs**
  - Identify test environment requirements.
  - All unit tests use the `fakeGCFClient` mock -- no GCP credentials or live Cloud Functions required for functional testing.

- [ ] **API Extensions**
  - Identify any API changes or extensions.
  - No API changes. The guard is internal to the provisioner's `EnsureOrgInMint` method. Error messages are updated to include the GCP project ID for diagnostics.

- [ ] **Topology Considerations**
  - Identify topology and deployment considerations.
  - N/A -- the guard runs in the CLI process during `fullsend mint enroll-org` and `fullsend mint enroll-repo`. No topology changes.

### Section II - Test Planning

#### Section II.1 - Scope of Testing

This test plan covers the restored data consistency guard in `EnsureOrgInMint` and the related `GetServiceTrafficEnvVars` path used by `EnsureOrgInMint`, `RegisterPerRepoWIF`, and `RemoveOrgFromMint`. Testing validates that the guard correctly detects data inconsistency conditions, permits legitimate first enrollments, and filters legacy org/role keys.

**Testing Goals:**

**Functional Goals:**
- **P0:** Verify the data consistency guard blocks enrollment when `ALLOWED_ORGS` is empty but `ROLE_APP_IDS` has active role-only entries.
- **P0:** Verify first enrollment proceeds when both `ALLOWED_ORGS` and `ROLE_APP_IDS` are empty.
- **P0:** Verify enrollment proceeds when `ROLE_APP_IDS` has only legacy org/role keys.
- **P1:** Verify `EnsureOrgInMint` reads env vars from the traffic-serving revision (not the function's config).
- **P1:** Verify error messages include actionable diagnostic information (project ID, role count).

**Quality Goals:**
- **P1:** Verify no regression in existing enrollment, unenrollment, and per-repo WIF registration flows.

**Integration Goals:**
- **P2:** Verify the guard works correctly through the full `fullsend mint enroll-org` CLI flow.

**Out of Scope (Testing Scope Exclusions):**

- [ ] GCP Cloud Functions deployment and infrastructure -- *Rationale:* Infrastructure provisioning is tested by GCP platform team; the guard operates on env var data only. -- *PM/Lead Agreement:* TBD
- [ ] Cloud Run revision rollout timing -- *Rationale:* Traffic routing lag is a platform behavior outside FullSend's control. -- *PM/Lead Agreement:* TBD
- [ ] Concurrent enrollment race conditions -- *Rationale:* The read-modify-write race is a documented, accepted limitation. Locking would require external coordination. -- *PM/Lead Agreement:* TBD

#### Section II.2 - Test Strategy

**Functional:**

- [x] **Functional Testing**
  - Verify data consistency guard logic in `EnsureOrgInMint` across all three guard states: role-only keys present (block), empty ROLE_APP_IDS (allow), legacy-only keys (allow). Verify correct error message content.

- [x] **Automation Testing**
  - All test scenarios are automated as Go unit tests using `fakeGCFClient`. Tests run in CI via `go test ./internal/dispatch/gcf/...`.

- [x] **Regression Testing**
  - Verify existing enrollment, unenrollment, and per-repo WIF registration paths continue to work. Existing tests for `EnsureOrgInMint`, `RegisterPerRepoWIF`, and `RemoveOrgFromMint` exercise the `GetServiceTrafficEnvVars` path.

**Non-Functional:**

- [ ] **Performance Testing**
  - N/A. The guard adds a single JSON unmarshal of a small env var value. No measurable performance impact.

- [ ] **Scale Testing**
  - N/A. Guard operates on a single env var value; not sensitive to scale.

- [ ] **Security Testing**
  - N/A. No authentication or authorization changes. The guard is a read-only data integrity check.

- [ ] **Usability Testing**
  - N/A. No user-facing UI changes.

- [ ] **Monitoring**
  - N/A. No monitoring or alerting changes. The error message provides actionable guidance (`fullsend mint status --project=`).

**Integration & Compatibility:**

- [ ] **Compatibility Testing**
  - N/A. No API or CLI interface changes. Backward compatible with existing mint configurations.

- [ ] **Upgrade Testing**
  - N/A. The guard is additive -- upgrading the CLI binary activates it without migration.

- [ ] **Dependencies**
  - Depends on `mintcore.RoleOnlyAppIDs` for filtering legacy keys. This function is stable and has existing test coverage.

- [ ] **Cross Integrations**
  - Guard is invoked by `runMintEnrollOrg`, `runMintEnrollRepo`, `provisionWithExistingMint`, and `provisionSelfManaged`. All callers are covered by existing integration tests.

**Infrastructure:**

- [ ] **Cloud Testing**
  - N/A. All tests use `fakeGCFClient` mocks. No live GCP resources needed.

#### Section II.3 - Test Environment

- **Cluster Topology:** N/A -- unit tests run locally, no cluster required
- **Platform & Product Version(s):** FullSend 0.x on GitHub Actions
- **CPU Virtualization:** N/A
- **Compute Resources:** Standard CI runner (2 vCPU, 8 GB RAM)
- **Special Hardware:** None
- **Storage:** Standard ephemeral CI storage
- **Network:** Standard CI network; no external GCP connectivity required for unit tests
- **Required Operators:** None
- **Platform:** GitHub Actions (ubuntu-latest)
- **Special Configurations:** Go 1.23+, `go test` with race detector enabled

#### Section II.3.1 - Testing Tools & Frameworks

No new or special tools required. Tests use Go standard `testing` package and `testify/assert`.

#### Section II.4 - Entry Criteria

- [ ] Requirements and design documents are **approved and merged**
- [ ] Test environment can be **set up and configured** (see Section II.3 - Test Environment)
- [ ] `fakeGCFClient` mock supports `GetServiceTrafficEnvVars` with configurable `trafficEnvVars` field
- [ ] `mintcore.RoleOnlyAppIDs` function is available and tested

#### Section II.5 - Risks

- [ ] **Timeline/Schedule**
  - *Risk:* Low risk. The guard is a focused change with well-defined test scenarios.
  - *Impact:* Minimal -- estimated 1-2 hours for test implementation and review.

- [ ] **Test Coverage**
  - *Risk:* Guard only detects fully empty `ALLOWED_ORGS`. Partial data loss (some orgs missing) is not covered.
  - *Impact:* Medium -- partial data loss could still result in dropped enrollments, but this is an inherent limitation of the env-var-based approach.

- [ ] **Test Environment**
  - *Risk:* N/A. All tests use in-process mocks with no external dependencies.
  - *Impact:* None.

- [ ] **Untestable Aspects**
  - *Risk:* The actual Cloud Run `GetServiceTrafficEnvVars` API call reads from the traffic-serving revision, which may lag during rollout. This timing behavior cannot be unit-tested.
  - *Impact:* Low -- the guard provides defense-in-depth, not primary correctness.

- [ ] **Resource Constraints**
  - *Risk:* N/A. Tests run on standard CI resources.
  - *Impact:* None.

- [ ] **Dependencies**
  - *Risk:* `mintcore.RoleOnlyAppIDs` logic must correctly filter legacy org/role keys from role-only keys.
  - *Impact:* Low -- `RoleOnlyAppIDs` has existing unit test coverage in `handler_test.go`.

- [ ] **Other**
  - *Risk:* N/A.
  - *Impact:* None.

---

### Section III - Requirements-to-Tests Mapping

- **[GH-58]** -- Data consistency guard blocks enrollment when ALLOWED_ORGS is empty but ROLE_APP_IDS has active role-only entries
  - *Test Scenario:* Verify EnsureOrgInMint returns data inconsistency error when ALLOWED_ORGS is empty and ROLE_APP_IDS has role-only keys
  - *Tier:* Functional
  - *Priority:* P0

- **[GH-58]** -- Data consistency guard permits first enrollment when ROLE_APP_IDS is empty
  - *Test Scenario:* Verify EnsureOrgInMint proceeds when both ALLOWED_ORGS and ROLE_APP_IDS are empty
  - *Tier:* Functional
  - *Priority:* P0

- **[GH-58]** -- Data consistency guard permits enrollment when ROLE_APP_IDS has only legacy org/role keys
  - *Test Scenario:* Verify EnsureOrgInMint proceeds when ROLE_APP_IDS contains only legacy keys with "/" separator
  - *Tier:* Functional
  - *Priority:* P0

- **[GH-58]** -- Error message includes actionable diagnostic information
  - *Test Scenario:* Verify error contains role count and project ID for operator diagnosis
  - *Tier:* Functional
  - *Priority:* P1

- **[GH-58]** -- EnsureOrgInMint reads env vars from traffic-serving revision
  - *Test Scenario:* Verify GetServiceTrafficEnvVars is called instead of reading from function config env vars
  - *Tier:* Functional
  - *Priority:* P1

- **[GH-58]** -- Existing org enrollment is not disrupted by the guard
  - *Test Scenario:* Verify EnsureOrgInMint adds new org when ALLOWED_ORGS has existing entries
  - *Tier:* Functional
  - *Priority:* P0

- **[GH-58]** -- Already-enrolled org is handled idempotently
  - *Test Scenario:* Verify EnsureOrgInMint returns nil and skips update when org is already in ALLOWED_ORGS
  - *Tier:* Functional
  - *Priority:* P1

- **[GH-58]** -- Mint URL mismatch is detected
  - *Test Scenario:* Verify EnsureOrgInMint returns error when function URI does not match expected URL
  - *Tier:* Functional
  - *Priority:* P1

- **[GH-58]** -- RoleOnlyAppIDs correctly filters legacy keys
  - *Test Scenario:* Verify RoleOnlyAppIDs returns only keys without "/" separator and excludes org/role keys
  - *Tier:* Functional
  - *Priority:* P0

- **[GH-58]** -- RegisterPerRepoWIF reads from traffic-serving revision
  - *Test Scenario:* Verify RegisterPerRepoWIF uses GetServiceTrafficEnvVars for PER_REPO_WIF_REPOS
  - *Tier:* Functional
  - *Priority:* P1

- **[GH-58]** -- RemoveOrgFromMint reads from traffic-serving revision
  - *Test Scenario:* Verify RemoveOrgFromMint uses GetServiceTrafficEnvVars for ALLOWED_ORGS removal
  - *Tier:* Functional
  - *Priority:* P1

- **[GH-58]** -- Guard handles malformed ROLE_APP_IDS gracefully
  - *Test Scenario:* Verify EnsureOrgInMint proceeds without error when ROLE_APP_IDS contains invalid JSON
  - *Tier:* Functional
  - *Priority:* P2

- **[GH-58]** -- Full mint enrollment flow with data consistency guard active
  - *Test Scenario:* Verify fullsend mint enroll-org succeeds end-to-end with guard protecting against stale reads
  - *Tier:* Functional
  - *Priority:* P1

---

### Section IV - Sign-off

| Role | Name | Date |
|:-----|:-----|:-----|
| QE Lead | TBD | |
| Dev Lead | TBD | |
| PM | TBD | |
