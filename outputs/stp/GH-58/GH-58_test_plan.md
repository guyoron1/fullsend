# FullSend Test Plan

## **[Restore Data Consistency Guard in EnsureOrgInMint] - Quality Engineering Plan**

### Metadata & Tracking

- **Enhancement(s)** -- [fullsend#2436](https://github.com/fullsend-ai/fullsend/pull/2436) (local mirror: [GH-58](https://github.com/guyoron1/fullsend/pull/58))
- **Feature Tracking** -- GH-58: fix(#2433): restore data consistency guard in EnsureOrgInMint
- **Epic Tracking** -- GH-58
- **QE Owner(s)** -- TBD
- **Owning SIG** -- N/A
- **Participating SIGs** -- None

**Document Conventions:** N/A

### Feature Overview

This change restores a defense-in-depth cross-check in the org enrollment operation that prevents silent clobbering of the allowed-orgs configuration on stale reads from the Cloud Run traffic-serving revision. When the allowed-orgs list is empty but active role configurations exist (role-only entries in the app ID registry), the mint has already been bootstrapped and the empty value indicates configuration data loss rather than a first enrollment. The guard aborts the operation to prevent silently unenrolling all existing orgs, adapted for the role-only model where legacy org/role keys are filtered out via role-only key filtering.

> **Scope note:** This STP covers only the data consistency guard fix (upstream fullsend-ai/fullsend#2436). The local mirror branch (GH-58) bundles additional upstream changes across 159 files; those changes are not in scope for this test plan.

---

### Section I - Motivation and Requirements Review

#### Section I.1 - Requirement & User Story Review Checklist

- [ ] **Review Requirements**
  - Reviewed the relevant requirements.
  - PR restores a data consistency guard removed during role-only migration. The fix prevents the allowed-orgs configuration from being silently overwritten when the traffic-serving revision returns stale/empty data.

- [ ] **Understand Value and Customer Use Cases**
  - Confirmed clear user stories and understood. Understand the value and customer use cases.
  - Prevents accidental unenrollment of all orgs from a shared mint instance. Without this guard, a stale read during `fullsend mint enroll-org` could silently drop all existing org enrollments.

- [ ] **Testability**
  - Confirmed requirements are **testable and unambiguous**.
  - The guard logic is fully testable via existing mock infrastructure. Three distinct guard scenarios are covered: active role-only keys present (error), no app IDs configured (proceed), and legacy-only keys (proceed).

- [ ] **Acceptance Criteria**
  - Ensured acceptance criteria are **defined clearly**.
  - When the allowed-orgs list is empty and active role configurations exist, the org enrollment operation must return an error containing "data inconsistency" and the count of configured roles. When no app IDs are configured or only legacy keys exist, enrollment must proceed normally.

- [ ] **Non-Functional Requirements**
  - Confirmed coverage for NFRs.
  - No performance, scalability, or security NFRs changed. The guard adds a single JSON unmarshal and map iteration on a small env var value during enrollment -- negligible overhead.

#### Section I.2 - Known Limitations

- The consistency guard only detects empty `ALLOWED_ORGS`. Partial data loss (e.g., some orgs missing but not all) is not detected by this guard.
- The read-modify-write in the org enrollment operation is not locked; concurrent enrollment calls sharing the same mint can still race. This is documented and accepted as a known limitation.
- The traffic-serving revision read path reads from the observed traffic-serving revision, which may lag behind a just-deployed revision during Cloud Run rollout.

#### Section I.3 - Technology and Design Review

- [ ] **Developer Handoff/QE Kickoff**
  - Review developer handoff for technical details on the data consistency guard implementation.
  - The fix modifies the org enrollment operation in `internal/dispatch/gcf/provisioner.go` and depends on role-only key filtering logic in `internal/mintcore/handler.go`.

- [ ] **Technology Challenges**
  - Identify any technology challenges or new dependencies.
  - Reading mint configuration from the traffic-serving revision avoids stale data, but introduces a consistency window during Cloud Run rollout where the observed values may lag behind the latest deployment.

- [ ] **Test Environment Needs**
  - Identify test environment requirements.
  - All unit tests use mock infrastructure -- no GCP credentials or live Cloud Functions required for functional testing.

- [ ] **API Extensions**
  - Identify any API changes or extensions.
  - No API changes. The guard is internal to the provisioner's org enrollment method. Error messages are updated to include the GCP project ID for diagnostics.

- [ ] **Topology Considerations**
  - Identify topology and deployment considerations.
  - N/A -- the guard runs in the CLI process during `fullsend mint enroll-org` and `fullsend mint enroll-repo`. No topology changes.

### Section II - Test Planning

#### Section II.1 - Scope of Testing

This test plan covers the restored data consistency guard in the org enrollment operation and the related traffic-serving revision configuration read path used by org enrollment, per-repo WIF registration, and org removal. Testing validates that the guard correctly detects data inconsistency conditions, permits legitimate first enrollments, and filters legacy org/role keys.

**Testing Goals:**

**Functional Goals:**
- **P0:** Verify the data consistency guard blocks enrollment when allowed-orgs is empty but active role configurations exist.
- **P0:** Verify first enrollment proceeds when both allowed-orgs and app ID registry are empty.
- **P0:** Verify enrollment proceeds when app ID registry has only legacy org/role keys.
- **P1:** Verify org enrollment reads configuration from the traffic-serving revision (not the function's config).
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
  - Verify data consistency guard logic in the org enrollment operation across all three guard states: role-only keys present (block), no app IDs configured (allow), legacy-only keys (allow). Verify correct error message content.

- [x] **Automation Testing**
  - All test scenarios are automated as Go unit tests using mock infrastructure. Tests run in CI via `go test ./internal/dispatch/gcf/...`.

- [x] **Regression Testing**
  - Verify existing enrollment, unenrollment, and per-repo WIF registration paths continue to work. Existing tests for org enrollment, per-repo WIF registration, and org removal exercise the traffic-serving revision read path.

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
  - Depends on the role-only key filtering logic for separating legacy keys from active role configurations. This function is stable and has existing test coverage.

- [x] **Cross Integrations**
  - Guard is invoked by the enroll-org, enroll-repo, provision-with-existing-mint, and provision-self-managed CLI commands. All callers are covered by existing integration tests; no new cross-integration testing is needed for this guard change.

**Infrastructure:**

- [ ] **Cloud Testing**
  - N/A. All tests use mock infrastructure. No live GCP resources needed.

#### Section II.3 - Test Environment

- **Cluster Topology:** N/A -- unit tests run locally, no cluster required
- **Platform & Product Version(s):** FullSend 0.x on GitHub Actions (ubuntu-latest)
- **CPU Virtualization:** N/A
- **Compute Resources:** Standard CI runner
- **Special Hardware:** None
- **Storage:** N/A
- **Network:** No GCP credentials or external connectivity required for unit tests
- **Required Operators:** None
- **Platform:** GitHub Actions (ubuntu-latest)
- **Special Configurations:** Go 1.23+, `go test` with race detector enabled

#### Section II.3.1 - Testing Tools & Frameworks

No new or special tools required beyond the project's standard Go test toolchain.

#### Section II.4 - Entry Criteria

- [ ] Requirements and design documents are **approved and merged**
- [ ] Test environment can be **set up and configured** (see Section II.3 - Test Environment)
- [ ] Mock infrastructure supports configurable traffic-serving revision environment variables
- [ ] Role-only key filtering function is available and tested

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
  - *Risk:* The actual Cloud Run traffic-serving revision API call may lag during rollout. This timing behavior cannot be unit-tested.
  - *Impact:* Low -- the guard provides defense-in-depth, not primary correctness.

- [ ] **Resource Constraints**
  - *Risk:* N/A. Tests run on standard CI resources.
  - *Impact:* None.

- [ ] **Dependencies**
  - *Risk:* The role-only key filtering logic must correctly separate legacy org/role keys from role-only keys.
  - *Impact:* Low -- role-only key filtering has existing unit test coverage.

- [ ] **Other**
  - *Risk:* N/A.
  - *Impact:* None.

---

### Section III - Requirements-to-Tests Mapping

- **[GH-58]** -- Data consistency guard blocks enrollment when allowed-orgs is empty but active role configurations exist
  - *Test Scenario:* Verify org enrollment is blocked with a data inconsistency error when allowed orgs are empty but active role configurations exist
  - *Tier:* Functional
  - *Priority:* P0

- **[GH-58]** -- Data consistency guard permits first enrollment when no app IDs are configured
  - *Test Scenario:* Verify org enrollment proceeds when both allowed-orgs and app ID registry are empty (first enrollment)
  - *Tier:* Functional
  - *Priority:* P0

- **[GH-58]** -- Data consistency guard permits enrollment when app ID registry has only legacy org/role keys
  - *Test Scenario:* Verify org enrollment proceeds when app ID registry contains only legacy keys with "/" separator
  - *Tier:* Functional
  - *Priority:* P0

- **[GH-58]** -- Error message includes actionable diagnostic information
  - *Test Scenario:* Verify error contains role count and project ID for operator diagnosis
  - *Tier:* Functional
  - *Priority:* P1

- **[GH-58]** -- Org enrollment reads configuration from traffic-serving revision
  - *Test Scenario:* Verify mint configuration is read from the traffic-serving revision instead of function config env vars
  - *Tier:* Functional
  - *Priority:* P1

- **[GH-58]** -- Existing org enrollment is not disrupted by the guard
  - *Test Scenario:* Verify org enrollment adds new org when allowed-orgs already has existing entries
  - *Tier:* Functional
  - *Priority:* P0

- **[GH-58]** -- Already-enrolled org is handled idempotently
  - *Test Scenario:* Verify org enrollment returns success and skips update when org is already in allowed-orgs
  - *Tier:* Functional
  - *Priority:* P1

- **[GH-58]** -- Mint URL mismatch is detected
  - *Test Scenario:* Verify org enrollment returns error when function URI does not match expected URL
  - *Tier:* Functional
  - *Priority:* P1

- **[GH-58]** -- Role-only key filtering correctly separates legacy keys
  - *Test Scenario:* Verify role-only key filtering returns only keys without "/" separator and excludes legacy org/role keys
  - *Tier:* Functional
  - *Priority:* P0

- **[GH-58]** -- Per-repo WIF registration reads from traffic-serving revision
  - *Test Scenario:* Verify per-repo WIF registration reads the WIF repos list from the traffic-serving revision
  - *Tier:* Functional
  - *Priority:* P1

- **[GH-58]** -- Org removal reads from traffic-serving revision
  - *Test Scenario:* Verify org removal reads allowed-orgs from the traffic-serving revision for removal
  - *Tier:* Functional
  - *Priority:* P1

- **[GH-58]** -- Guard handles malformed app ID registry gracefully
  - *Test Scenario:* Verify org enrollment proceeds without error when the app ID registry contains invalid JSON
  - *Tier:* Functional
  - *Priority:* P2

- **[GH-58]** -- Enrollment fails gracefully when traffic-serving revision config cannot be read
  - *Test Scenario:* Verify org enrollment fails with a clear error when reading configuration from the traffic-serving revision returns an API error or timeout
  - *Tier:* Functional
  - *Priority:* P1

- **[GH-58]** -- Guard handles allowed-orgs with invalid format without crashing
  - *Test Scenario:* Verify org enrollment handles malformed or corrupt allowed-orgs data gracefully instead of panicking
  - *Tier:* Functional
  - *Priority:* P2

- **[GH-58]** -- Enrollment handles missing traffic-serving revision gracefully
  - *Test Scenario:* Verify org enrollment fails with a clear error when the Cloud Run service has no traffic-serving revision
  - *Tier:* Functional
  - *Priority:* P2

- **[GH-58]** -- Full mint enrollment flow with data consistency guard active
  - *Test Scenario:* Verify `fullsend mint enroll-org` succeeds end-to-end with guard protecting against stale reads
  - *Tier:* End-to-End
  - *Priority:* P1

---

### Section IV - Sign-off

| Role | Name | Date |
|:-----|:-----|:-----|
| QE Lead | TBD | |
| Dev Lead | TBD | |
| PM | TBD | |
