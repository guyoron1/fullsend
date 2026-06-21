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

## 1. Summary

This test plan covers the fix for a flaky E2E test failure in `TestAdminInstallUninstall` where the enrollment PR merge step intermittently fails with a GitHub API 409 "Head branch is out of date" error. The fix adds retry-with-branch-update logic to `MergeChangeProposal` in the forge GitHub client.

## 2. Requirements

### 2.1 Requirement Analysis

| Req ID | Requirement | Source | Acceptance Criteria |
|:-------|:------------|:-------|:--------------------|
| REQ-001 | `MergeChangeProposal` must handle 409 "Head branch is out of date" errors by updating the PR branch and retrying the merge | GH-2432 issue body | Merge succeeds after branch update when base has advanced |
| REQ-002 | Retry logic must be bounded (max 3 attempts) to prevent infinite loops | GH-2432 triage recommendation | Function returns error after 3 failed attempts |
| REQ-003 | Non-409 errors must not be retried — they are returned immediately | GH-2432 PR #2434 description | 422 and other HTTP errors propagate without retry |
| REQ-004 | Context cancellation must be respected during retry delays | PR #2434 implementation | ctx.Done() aborts the retry loop |

### 2.2 Scope

**In scope:**
- `MergeChangeProposal` retry-on-409 behavior in `internal/forge/github/github.go`
- Unit tests for the retry logic in `internal/forge/github/github_merge_test.go`
- E2E enrollment PR merge path in `e2e/admin/admin_test.go`

**Out of scope:**
- Other forge Client interface methods (no behavioral changes)
- The reconcile workflow that causes the base branch to advance (external trigger)
- GitHub API behavior itself (external dependency)

## 3. Test Scenarios

### 3.1 Unit Tests — `MergeChangeProposal` Retry Logic

| Test ID | Scenario | Pre-conditions | Steps | Expected Result | Tier |
|:--------|:---------|:---------------|:------|:----------------|:-----|
| TS-GH-2432-001 | Successful merge on first attempt (happy path) | GitHub API returns 200 OK on merge | 1. Call `MergeChangeProposal` with valid owner/repo/number | Merge succeeds, function returns nil. No update-branch call is made. | Unit |
| TS-GH-2432-002 | 409 triggers branch update and successful retry | GitHub API returns 409 on first merge, 200 on second | 1. Call `MergeChangeProposal`<br>2. First PUT to `/merge` returns 409<br>3. PUT to `/update-branch` returns 202<br>4. Second PUT to `/merge` returns 200 | Function returns nil. `update-branch` called once. Merge attempted twice. | Unit |
| TS-GH-2432-003 | Non-409 error is not retried | GitHub API returns 422 "not mergeable" | 1. Call `MergeChangeProposal`<br>2. PUT to `/merge` returns 422 | Function returns error containing "not mergeable". Merge attempted exactly once. No update-branch call. | Unit |
| TS-GH-2432-004 | Persistent 409 exhausts retries | GitHub API returns 409 on all merge attempts | 1. Call `MergeChangeProposal`<br>2. All 3 PUT to `/merge` return 409<br>3. PUT to `/update-branch` returns 202 each time | Function returns error. Merge attempted >1 times. Error message references the PR number. | Unit |

### 3.2 Functional Tests — Forge Client Interface Compliance

| Test ID | Scenario | Pre-conditions | Steps | Expected Result | Tier |
|:--------|:---------|:---------------|:------|:----------------|:-----|
| TS-GH-2432-005 | `MergeChangeProposal` remains compatible with `forge.Client` interface | Code compiles | 1. Verify `LiveClient` implements all `forge.Client` methods<br>2. Confirm method signature unchanged: `MergeChangeProposal(ctx, owner, repo string, number int) error` | Compilation succeeds. No interface violation. | Tier1 |
| TS-GH-2432-006 | `FakeClient.MergeChangeProposal` still works for other tests | FakeClient available | 1. Call `FakeClient.MergeChangeProposal`<br>2. Verify it delegates to the configured error function | Returns configured error or nil. Existing tests using FakeClient are unaffected. | Tier1 |

### 3.3 Integration / E2E Tests — Enrollment PR Merge

| Test ID | Scenario | Pre-conditions | Steps | Expected Result | Tier |
|:--------|:---------|:---------------|:------|:----------------|:-----|
| TS-GH-2432-007 | `TestAdminInstallUninstall` enrollment PR merge succeeds under race | E2E environment with halfsend org, reconcile workflow active | 1. Run `TestAdminInstallUninstall`<br>2. Test creates enrollment PR<br>3. Reconcile workflow may push to default branch<br>4. Test calls `MergeChangeProposal` | Enrollment PR merges successfully even if base branch advanced during test. Test passes without 409 flake. | Tier2 |

### 3.4 Edge Cases and Error Handling

| Test ID | Scenario | Pre-conditions | Steps | Expected Result | Tier |
|:--------|:---------|:---------------|:------|:----------------|:-----|
| TS-GH-2432-008 | Context cancelled during retry delay | Context with short timeout | 1. Call `MergeChangeProposal` with a context that cancels during the 3s delay<br>2. First merge returns 409 | Function returns `ctx.Err()`. Does not hang. | Unit |
| TS-GH-2432-009 | `update-branch` call fails but retry still proceeds | GitHub API returns error on update-branch | 1. Call `MergeChangeProposal`<br>2. Merge returns 409<br>3. update-branch returns error<br>4. Retry merge | Function continues to retry merge despite update-branch failure. Final result depends on merge outcome. | Unit |

## 4. Regression Impact Analysis

### 4.1 Changed Files

| File | Change Summary | Risk |
|:-----|:---------------|:-----|
| `internal/forge/github/github.go` | `MergeChangeProposal` rewritten with retry loop, branch-update call, and context-aware delay | Medium — core merge behavior changed |
| `internal/forge/github/github_merge_test.go` | New test file with 4 test cases covering retry scenarios | Low — new tests only |

### 4.2 Dependency Chain (LSP Analysis)

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

### 4.3 Regression Risk Areas

| Area | Risk | Mitigation |
|:-----|:-----|:-----------|
| Other callers of `MergeChangeProposal` | Low | Method signature unchanged; retry is transparent to callers |
| `APIError` detection pattern | Low | Same `errors.As` pattern used in `enrollment.go` and throughout codebase |
| `FakeClient` compatibility | Low | PR #2435 added `UpdatePullRequestBranch` to FakeClient; PR #2434 doesn't need it (retry is internal) |
| `LiveClient.put()` / `LiveClient.do()` helpers | None | No changes to these methods |

## 5. Test Environment

| Requirement | Value |
|:------------|:------|
| **Language** | Go 1.23+ |
| **Test Framework** | `testing` + `testify` (assert/require) |
| **Unit Test Execution** | `go test ./internal/forge/github/ -run TestMergeChangeProposal` |
| **E2E Test Execution** | `go test -tags e2e ./e2e/admin/ -run TestAdminInstallUninstall` |
| **CI Platform** | GitHub Actions |
| **Mock Strategy** | `httptest.NewServer` for unit tests; `FakeClient` for integration |

## 6. Test Execution Matrix

| Test ID | Automated | Blocking | Run Frequency |
|:--------|:----------|:---------|:--------------|
| TS-GH-2432-001 | Yes | Yes | Every PR (unit) |
| TS-GH-2432-002 | Yes | Yes | Every PR (unit) |
| TS-GH-2432-003 | Yes | Yes | Every PR (unit) |
| TS-GH-2432-004 | Yes | Yes | Every PR (unit) |
| TS-GH-2432-005 | Yes (compile) | Yes | Every PR |
| TS-GH-2432-006 | Yes | Yes | Every PR (unit) |
| TS-GH-2432-007 | Yes | No | Merge queue (E2E) |
| TS-GH-2432-008 | Yes | Yes | Every PR (unit) |
| TS-GH-2432-009 | Yes | Yes | Every PR (unit) |

## 7. Pass/Fail Criteria

- **Pass:** All unit tests (TS-GH-2432-001 through -004, -008, -009) pass. Interface compliance verified (TS-GH-2432-005, -006). E2E flake rate for enrollment PR merge drops to zero over 10+ merge queue runs.
- **Fail:** Any unit test fails, interface breaks, or 409 flake persists in E2E runs.

## 8. References

| Resource | Link |
|:---------|:-----|
| Issue | [GH-2432](https://github.com/fullsend-ai/fullsend/issues/2432) |
| Fix PR (merged) | [#2435](https://github.com/fullsend-ai/fullsend/pull/2435) — adds `UpdatePullRequestBranch` to forge interface + E2E retry |
| Fix PR (open) | [#2434](https://github.com/fullsend-ai/fullsend/pull/2434) — retry logic inside `MergeChangeProposal` + unit tests |
| Failed run | [actions/runs/27770629379](https://github.com/fullsend-ai/fullsend/actions/runs/27770629379/job/82169303672) |
