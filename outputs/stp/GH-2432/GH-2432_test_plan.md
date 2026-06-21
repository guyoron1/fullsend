# FullSend Test Plan

| Field | Value |
|:------|:------|
| **Ticket** | GH-2432 |
| **Title** | bug(e2e): flaky 409 "Head branch is out of date" when merging enrollment PR |
| **Type** | Bug Fix |
| **Priority** | Medium |
| **Component** | E2E / Forge (GitHub Client) |
| **Product** | FullSend |
| **Platform** | GitHub Actions |
| **Version** | 0.x |
| **Date** | 2026-06-21 |
| **Author** | QualityFlow |

---

## Section I — Review Checklist

### I.1 Requirements Review

- [x] All acceptance criteria from GH-2432 are captured in Section II.1
- [x] Triage recommendations are mapped to test scenarios
- [x] Negative and edge-case scenarios are included
- [x] Each requirement has at least one corresponding test scenario

### I.2 Known Limitations

- GitHub issue data used as source of truth (Jira instance unavailable); acceptance criteria and triage data are equivalent
- Original race condition is non-deterministic — 409 may not reproduce in every E2E test run; unit tests with httptest mock guarantee coverage regardless of race timing
- Two complementary fix PRs (#2434 library-level, #2435 E2E-level) address the same issue; coordination between both approaches must be verified

### I.3 Technology Review

- [x] Go 1.23+ with `testing` + `testify` — standard project framework
- [x] `httptest.NewServer` for HTTP-level mocking of GitHub API responses
- [x] `FakeClient` for forge interface-level integration testing
- [x] GitHub Actions CI — standard project CI platform

---

## Section II — Test Plan

### II.1 Summary

This test plan covers the fix for a flaky E2E test failure in `TestAdminInstallUninstall` where the enrollment PR merge step intermittently fails with a GitHub API 409 "Head branch is out of date" error. The fix adds retry-with-branch-update logic to `MergeChangeProposal` in the forge GitHub client.

### II.2 Requirements

#### II.2.1 Requirement Analysis

| Req ID | Requirement | Source | Acceptance Criteria |
|:-------|:------------|:-------|:--------------------|
| REQ-001 | `MergeChangeProposal` must handle 409 "Head branch is out of date" errors by updating the PR branch and retrying the merge | GH-2432 issue body | Merge succeeds after branch update when base has advanced |
| REQ-002 | Retry logic must be bounded (max 3 attempts) to prevent infinite loops | GH-2432 triage recommendation | Function returns error after 3 failed attempts |
| REQ-003 | Non-409 errors must not be retried — they are returned immediately | GH-2432 PR #2434 description | 422 and other HTTP errors propagate without retry |
| REQ-004 | Context cancellation must be respected during retry delays | PR #2434 implementation | ctx.Done() aborts the retry loop |

#### II.2.2 Scope

**In scope:**
- `MergeChangeProposal` retry-on-409 behavior in `internal/forge/github/github.go`
- Unit tests for the retry logic in `internal/forge/github/github_merge_test.go`
- E2E enrollment PR merge path in `e2e/admin/admin_test.go`

This test plan covers two complementary fix PRs: PR #2435 (merged) adds `UpdatePullRequestBranch` to the `forge.Client` interface and E2E-level retry in `admin_test.go`; PR #2434 (open) adds library-level retry inside `MergeChangeProposal` with unit tests. Scenarios cover both approaches.

**Out of scope:**
- Other forge Client interface methods (no behavioral changes)
- The reconcile workflow that causes the base branch to advance (external trigger)
- GitHub API behavior itself (external dependency)

### II.3 Test Strategy

| Strategy | Applicable | Rationale |
|:---------|:-----------|:----------|
| Functional Testing | Y | 9 scenarios covering retry logic, error handling, and interface compliance |
| Automation Testing | Y | All tests automated; run on every PR and in merge queue |
| Regression Testing | Y | Regression impact analysis in Section II.7; existing tests verified |
| Performance Testing | N/A | No latency or throughput requirements; retry delay (≤9s worst-case) is acceptable |
| Security Testing | N/A | No RBAC, authentication, or authorization changes |
| Upgrade Testing | N/A | No persistent state; retry behavior is stateless |
| Usability Testing | N/A | No UI component |

### II.4 Test Scenarios

#### II.4.1 Unit Tests — `MergeChangeProposal` Retry Logic

| Test ID | Scenario | Priority | Pre-conditions | Steps | Expected Result | Tier |
|:--------|:---------|:---------|:---------------|:------|:----------------|:-----|
| TS-GH-2432-001 | Successful merge on first attempt (happy path) | P0 | GitHub API returns 200 OK on merge | 1. Call `MergeChangeProposal` with valid owner/repo/number | Merge succeeds, function returns nil. No update-branch call is made. | Unit |
| TS-GH-2432-002 | 409 triggers branch update and successful retry | P0 | GitHub API returns 409 on first merge, 200 on second | 1. Call `MergeChangeProposal`<br>2. First PUT to `/merge` returns 409<br>3. PUT to `/update-branch` returns 202<br>4. Second PUT to `/merge` returns 200 | Function returns nil. `update-branch` called once. Merge attempted twice. | Unit |
| TS-GH-2432-003 | Non-409 error is not retried | P1 | GitHub API returns 422 "not mergeable" | 1. Call `MergeChangeProposal`<br>2. PUT to `/merge` returns 422 | Function returns error containing "not mergeable". Merge attempted exactly once. No update-branch call. | Unit |
| TS-GH-2432-004 | Persistent 409 exhausts retries | P1 | GitHub API returns 409 on all merge attempts | 1. Call `MergeChangeProposal`<br>2. All 3 PUT to `/merge` return 409<br>3. PUT to `/update-branch` returns 202 each time | Function returns error. Merge attempted >1 times. Error message references the PR number. | Unit |

#### II.4.2 Functional Tests — Forge Client Interface Compliance

| Test ID | Scenario | Priority | Pre-conditions | Steps | Expected Result | Tier |
|:--------|:---------|:---------|:---------------|:------|:----------------|:-----|
| TS-GH-2432-005 | `MergeChangeProposal` remains compatible with `forge.Client` interface | P1 | Code compiles | 1. Verify `LiveClient` implements all `forge.Client` methods<br>2. Confirm method signature unchanged: `MergeChangeProposal(ctx, owner, repo string, number int) error` | Compilation succeeds. No interface violation. | Tier1 |
| TS-GH-2432-006 | `FakeClient.MergeChangeProposal` still works for other tests | P2 | FakeClient available | 1. Call `FakeClient.MergeChangeProposal`<br>2. Verify it delegates to the configured error function | Returns configured error or nil. Existing tests using FakeClient are unaffected. | Tier1 |

#### II.4.3 Integration / E2E Tests — Enrollment PR Merge

| Test ID | Scenario | Priority | Pre-conditions | Steps | Expected Result | Tier |
|:--------|:---------|:---------|:---------------|:------|:----------------|:-----|
| TS-GH-2432-007 | `TestAdminInstallUninstall` enrollment PR merge succeeds under race | P0 | E2E environment with halfsend org, reconcile workflow active | 1. Run `TestAdminInstallUninstall`<br>2. Test creates enrollment PR<br>3. Reconcile workflow may push to default branch<br>4. Test calls `MergeChangeProposal` | Enrollment PR merges successfully even if base branch advanced during test. Test passes without 409 flake. | Tier2 |

#### II.4.4 Edge Cases and Error Handling

| Test ID | Scenario | Priority | Pre-conditions | Steps | Expected Result | Tier |
|:--------|:---------|:---------|:---------------|:------|:----------------|:-----|
| TS-GH-2432-008 | Context cancelled during retry delay | P2 | Context with short timeout | 1. Call `MergeChangeProposal` with a context that cancels during the 3s delay<br>2. First merge returns 409 | Function returns `ctx.Err()`. Does not hang. | Unit |
| TS-GH-2432-009 | `update-branch` call fails but retry still proceeds | P2 | GitHub API returns error on update-branch | 1. Call `MergeChangeProposal`<br>2. Merge returns 409<br>3. update-branch returns error<br>4. Retry merge | Function continues to retry merge despite update-branch failure. Final result depends on merge outcome. | Unit |

### II.5 Test Environment

| Requirement | Value |
|:------------|:------|
| **Language** | Go 1.23+ |
| **Unit Test Execution** | `go test ./internal/forge/github/ -run TestMergeChangeProposal` |
| **E2E Test Execution** | `go test -tags e2e ./e2e/admin/ -run TestAdminInstallUninstall` |
| **Mock Strategy** | `httptest.NewServer` for unit tests (HTTP-level GitHub API mock); `FakeClient` for integration (forge interface-level mock) |

### II.6 Test Execution Matrix

| Test ID | Priority | Automated | Blocking | Run Frequency |
|:--------|:---------|:----------|:---------|:--------------|
| TS-GH-2432-001 | P0 | Yes | Yes | Every PR (unit) |
| TS-GH-2432-002 | P0 | Yes | Yes | Every PR (unit) |
| TS-GH-2432-003 | P1 | Yes | Yes | Every PR (unit) |
| TS-GH-2432-004 | P1 | Yes | Yes | Every PR (unit) |
| TS-GH-2432-005 | P1 | Yes (compile) | Yes | Every PR |
| TS-GH-2432-006 | P2 | Yes | Yes | Every PR (unit) |
| TS-GH-2432-007 | P0 | Yes | No | Merge queue (E2E) |
| TS-GH-2432-008 | P2 | Yes | Yes | Every PR (unit) |
| TS-GH-2432-009 | P2 | Yes | Yes | Every PR (unit) |

### II.7 Regression Impact Analysis

#### II.7.1 Changed Files

| File | Change Summary | Risk |
|:-----|:---------------|:-----|
| `internal/forge/github/github.go` | `MergeChangeProposal` rewritten with retry loop, branch-update call, and context-aware delay | Medium — core merge behavior changed |
| `internal/forge/github/github_merge_test.go` | New test file with 4 test cases covering retry scenarios | Low — new tests only |

#### II.7.2 Dependency Chain (LSP Analysis)

```
MergeChangeProposal (github.go:2059)
  ├── forge.Client interface (forge.go:313) — contract unchanged
  ├── LiveClient.put() (github.go:263) — existing helper, no change
  ├── LiveClient.do() (github.go:96) — existing helper, no change  
  ├── APIError (github.go:51) — used for 409 detection via errors.As
  │   └── Referenced in 55 locations across 5 files
  ├── checkStatus() (github.go:218) — NOT called (removed in refactor)
  └── Callers:
      ├── e2e/admin/admin_test.go:mergeEnrollmentPR (line ~263)
      ├── internal/forge/github/github_merge_test.go (4 test functions)
      └── internal/layers/enrollment.go — uses APIError pattern (line 195)
```

#### II.7.3 Regression Risk Areas

| Area | Risk | Mitigation |
|:-----|:-----|:-----------|
| Other callers of `MergeChangeProposal` | Low | Method signature unchanged; retry is transparent to callers |
| `APIError` detection pattern | Low | Same `errors.As` pattern used in `enrollment.go` and throughout codebase |
| `FakeClient` compatibility | Low | PR #2435 added `UpdatePullRequestBranch` to FakeClient; PR #2434 doesn't need it (retry is internal) |
| `LiveClient.put()` / `LiveClient.do()` helpers | None | No changes to these methods |

### II.8 Entry/Exit Criteria

**Entry criteria:**
- PR #2434 code changes are complete and compilable
- Go 1.23+ development environment available
- `httptest` mock server functional for unit tests

**Exit criteria:**
- All unit tests (TS-GH-2432-001 through -004, -008, -009) pass
- Interface compliance verified (TS-GH-2432-005, -006)
- E2E flake rate for enrollment PR merge drops to zero over 10+ merge queue runs

### II.9 Risks

| Risk ID | Description | Impact | Likelihood | Mitigation |
|:--------|:------------|:-------|:-----------|:-----------|
| RSK-001 | Two PRs (#2434, #2435) implement complementary retry logic — risk of duplicate retry if both merge | Medium | Low | PR #2434 adds library-level retry in `MergeChangeProposal`; PR #2435 adds E2E-level retry in `admin_test.go`. Approaches operate at different layers and are safe to coexist. Verify no double-retry amplification in E2E. |
| RSK-002 | Retry delay (3s per attempt, up to 3 attempts) adds up to 9s worst-case to E2E test duration | Low | Medium | Acceptable trade-off for flake elimination. Delay only triggered on 409, which is the failure case being fixed. |
| RSK-003 | Original race condition is non-deterministic — 409 may not reproduce in every E2E test run | Medium | High | Unit tests with `httptest` mock guarantee deterministic coverage of all retry paths regardless of race timing. E2E verification relies on absence of flakes over multiple runs. |

### II.10 Pass/Fail Criteria

- **Pass:** All unit tests (TS-GH-2432-001 through -004, -008, -009) pass. Interface compliance verified (TS-GH-2432-005, -006). E2E flake rate for enrollment PR merge drops to zero over 10+ merge queue runs.
- **Fail:** Any unit test fails, interface breaks, or 409 flake persists in E2E runs.

### II.11 References

| Resource | Link |
|:---------|:-----|
| Issue | [GH-2432](https://github.com/fullsend-ai/fullsend/issues/2432) |
| Fix PR (merged) | [#2435](https://github.com/fullsend-ai/fullsend/pull/2435) — adds `UpdatePullRequestBranch` to forge interface + E2E retry |
| Fix PR (open) | [#2434](https://github.com/fullsend-ai/fullsend/pull/2434) — retry logic inside `MergeChangeProposal` + unit tests |
| Failed run | [actions/runs/27770629379](https://github.com/fullsend-ai/fullsend/actions/runs/27770629379/job/82169303672) |
