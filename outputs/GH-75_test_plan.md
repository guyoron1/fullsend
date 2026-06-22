# Test Plan

| Field            | Value                                                        |
|:-----------------|:-------------------------------------------------------------|
| **Ticket**       | GH-75                                                        |
| **Title**        | fix(#2432): retry merge on 409 after updating PR branch      |
| **Product**      | fullsend                                                     |
| **Author**       | QualityFlow                                                  |
| **Date**         | 2026-06-22                                                   |
| **Status**       | Draft                                                        |

---

## 1. Introduction

This test plan covers the change to `MergeChangeProposal` in the GitHub forge
client (`internal/forge/github/github.go`). The fix adds retry logic that
handles HTTP 409 Conflict responses — which occur when a PR's head branch is
out of date with its base — by calling the GitHub "update branch" API and
retrying the merge, up to 3 attempts with a 3-second delay between each.

### 1.1 Scope

| In Scope | Out of Scope |
|:---------|:-------------|
| `MergeChangeProposal` retry-on-409 logic | Other `forge.Client` methods |
| Branch update API call on conflict | Rate-limit retry logic (unchanged `do()` method) |
| Error propagation for non-409 failures | FakeClient implementation (no behavioral change) |
| Context cancellation during retry loop | E2E enrollment/unenrollment flows (existing coverage) |

### 1.2 References

| Document | Location |
|:---------|:---------|
| PR | https://github.com/guyoron1/fullsend/pull/75 |
| Upstream | fullsend-ai/fullsend#2434 |
| forge.Client interface | `internal/forge/forge.go:313` |
| Implementation | `internal/forge/github/github.go:2059-2092` |

---

## 2. Requirements Mapping

| Req ID | Requirement | Source |
|:-------|:------------|:-------|
| R1 | `MergeChangeProposal` SHALL squash-merge a PR on first attempt when no conflict exists | Interface contract (`forge.Client.MergeChangeProposal`) |
| R2 | On 409 Conflict, the method SHALL call the GitHub "update-branch" API to bring the PR branch up to date | PR description, implementation lines 2076-2079 |
| R3 | After updating the branch, the method SHALL wait 3 seconds then retry the merge | Implementation lines 2082-2086 |
| R4 | The method SHALL retry up to 3 total attempts before returning an error | Implementation `maxAttempts = 3`, line 2060 |
| R5 | Non-409 errors (e.g. 422 "not mergeable") SHALL be returned immediately without retry | Implementation lines 2072-2074 |
| R6 | Context cancellation SHALL abort the retry loop promptly | Implementation lines 2083-2085 |
| R7 | When retries are exhausted, the method SHALL return a descriptive error including the PR number and attempt count | Implementation line 2091 |

---

## 3. Test Scenarios

### 3.1 Happy Path — Successful First-Attempt Merge

| Field | Value |
|:------|:------|
| **Scenario ID** | TS-01 |
| **Requirement** | R1 |
| **Description** | Merge succeeds on first PUT to `/pulls/{n}/merge` with 200 OK |
| **Preconditions** | PR exists, head branch is up to date with base |
| **Test Steps** | 1. Create httptest server returning 200 + `{"sha":"abc123"}` for PUT merge<br>2. Call `MergeChangeProposal(ctx, owner, repo, number)`<br>3. Assert no error returned<br>4. Assert merge endpoint was called exactly once |
| **Expected Result** | Method returns `nil`; no update-branch call made |
| **Classification** | Unit Test |
| **Existing Coverage** | `TestMergeChangeProposal_Success` in `github_merge_test.go:15` |

### 3.2 Retry Path — 409 Triggers Branch Update and Successful Retry

| Field | Value |
|:------|:------|
| **Scenario ID** | TS-02 |
| **Requirement** | R2, R3 |
| **Description** | First merge attempt returns 409; update-branch is called; second merge succeeds |
| **Preconditions** | PR exists, head branch is behind base |
| **Test Steps** | 1. Create httptest server:<br>&nbsp;&nbsp;- First PUT `/merge` → 409 `{"message":"Head branch is out of date"}`<br>&nbsp;&nbsp;- PUT `/update-branch` → 202 Accepted<br>&nbsp;&nbsp;- Second PUT `/merge` → 200 OK<br>2. Call `MergeChangeProposal`<br>3. Assert no error<br>4. Assert merge called 2 times, update-branch called 1 time |
| **Expected Result** | Method returns `nil` after successful retry |
| **Classification** | Unit Test |
| **Existing Coverage** | `TestMergeChangeProposal_409UpdatesBranchAndRetries` in `github_merge_test.go:34` |

### 3.3 Non-409 Error — Immediate Failure Without Retry

| Field | Value |
|:------|:------|
| **Scenario ID** | TS-03 |
| **Requirement** | R5 |
| **Description** | Merge returns 422 ("not mergeable"); error is returned without retry |
| **Preconditions** | PR has merge conflicts or is in an unmergeable state |
| **Test Steps** | 1. Create httptest server returning 422 for PUT `/merge`<br>2. Call `MergeChangeProposal`<br>3. Assert error returned containing "not mergeable"<br>4. Assert merge endpoint called exactly once |
| **Expected Result** | Error returned immediately; no update-branch call |
| **Classification** | Unit Test |
| **Existing Coverage** | `TestMergeChangeProposal_NonConflictErrorNotRetried` in `github_merge_test.go:73` |

### 3.4 Retry Exhaustion — Persistent 409 After Max Attempts

| Field | Value |
|:------|:------|
| **Scenario ID** | TS-04 |
| **Requirement** | R4, R7 |
| **Description** | All 3 merge attempts return 409; method returns descriptive error |
| **Preconditions** | Base branch changes faster than the PR can catch up (race condition) |
| **Test Steps** | 1. Create httptest server always returning 409 for PUT `/merge`, 202 for PUT `/update-branch`<br>2. Call `MergeChangeProposal`<br>3. Assert error returned<br>4. Assert error message contains PR number and mentions retries<br>5. Assert merge was called 3 times |
| **Expected Result** | Error: `"merge pull request #N: branch remained out of date after 3 update-and-retry attempts"` |
| **Classification** | Unit Test |
| **Existing Coverage** | `TestMergeChangeProposal_409PersistsAfterRetries` in `github_merge_test.go:92` |

### 3.5 Context Cancellation During Retry Delay

| Field | Value |
|:------|:------|
| **Scenario ID** | TS-05 |
| **Requirement** | R6 |
| **Description** | Context is cancelled while waiting between retry attempts |
| **Preconditions** | Merge returns 409; context has a short deadline |
| **Test Steps** | 1. Create httptest server returning 409 for merge, 202 for update-branch<br>2. Create context with short timeout or cancel during delay<br>3. Call `MergeChangeProposal` with cancellable context<br>4. Assert `context.Canceled` or `context.DeadlineExceeded` error |
| **Expected Result** | Method returns context error promptly instead of completing all retries |
| **Classification** | Unit Test |
| **Existing Coverage** | `TestMergeChangeProposal_ContextCancellation` in `qf-tests/GH-2432/go/merge_retry_test.go:343` |

### 3.6 Update-Branch API Failure — Resilience

| Field | Value |
|:------|:------|
| **Scenario ID** | TS-06 |
| **Requirement** | R2, R3 |
| **Description** | Update-branch call fails (e.g. 403); merge retry still proceeds |
| **Preconditions** | Token may lack permissions to update the PR branch |
| **Test Steps** | 1. Create httptest server:<br>&nbsp;&nbsp;- PUT `/merge` → 409<br>&nbsp;&nbsp;- PUT `/update-branch` → 403 or 500<br>&nbsp;&nbsp;- Next PUT `/merge` → 200 (branch updated externally)<br>2. Call `MergeChangeProposal`<br>3. Assert no error (merge eventually succeeds) |
| **Expected Result** | Method is resilient to update-branch failure; continues retry loop |
| **Classification** | Unit Test |
| **Existing Coverage** | None — **new test recommended** |

### 3.7 E2E — Enrollment PR Merge

| Field | Value |
|:------|:------|
| **Scenario ID** | TS-07 |
| **Requirement** | R1, R2 (integration) |
| **Description** | Enrollment flow creates and merges a PR using `MergeChangeProposal` |
| **Preconditions** | E2E environment with test org and repo |
| **Test Steps** | 1. Run `fullsend admin install` to create enrollment PR<br>2. Wait for PR to appear<br>3. Call `MergeChangeProposal` on enrollment PR<br>4. Verify shim workflow is installed |
| **Expected Result** | Enrollment PR merges successfully; `.github/workflows/fullsend.yaml` exists |
| **Classification** | E2E Test |
| **Existing Coverage** | `e2e/admin/admin_test.go:263` |

---

## 4. Regression Impact Analysis

### 4.1 Call Graph

```
forge.Client (interface)
  └── MergeChangeProposal(ctx, owner, repo, number) error
        ├── github.LiveClient.MergeChangeProposal  ← CHANGED (retry logic added)
        │     ├── LiveClient.put()  (merge endpoint)
        │     ├── LiveClient.do()   (update-branch endpoint)
        │     └── APIError / checkStatus (error classification)
        ├── forge.FakeClient.MergeChangeProposal    ← UNCHANGED (delegates to error map)
        └── Callers:
              ├── e2e/admin/admin_test.go:263  (enrollment merge)
              └── e2e/admin/admin_test.go:653  (removal PR merge)
```

### 4.2 Regression Risk Assessment

| Area | Risk | Rationale |
|:-----|:-----|:----------|
| Enrollment/Unenrollment flows | **Low** | Same `MergeChangeProposal` signature; retry is additive behavior |
| Existing merge-on-first-try behavior | **None** | 200 response exits immediately on first attempt (no retry path taken) |
| Rate-limit handling | **None** | `do()` method unchanged; rate-limit retry is orthogonal |
| `retryOnTransient` shared helper | **None** | `MergeChangeProposal` uses its own retry loop, not `retryOnTransient` |
| `put()` helper | **Low** | `put()` is reused for merge; now also `do()` used directly for update-branch |
| Context propagation | **Low** | Context is correctly checked in the `select` block between retries |

### 4.3 Key Observations

1. **The retry loop is self-contained** — it does not reuse `retryOnTransient()`, avoiding any interaction with the transient-error retry logic for file operations.
2. **Update-branch errors are silently ignored** — the code checks `updateErr == nil` only for closing the body. A failed update-branch does not abort the retry loop, which is intentionally resilient.
3. **The 3-second delay is hardcoded** — not configurable, but appropriate for GitHub's async branch update propagation.

---

## 5. Test Coverage Summary

| Scenario ID | Description | Classification | Coverage Status |
|:------------|:------------|:---------------|:----------------|
| TS-01 | Successful first-attempt merge | Unit Test | Covered |
| TS-02 | 409 triggers update + retry success | Unit Test | Covered |
| TS-03 | Non-409 error returns immediately | Unit Test | Covered |
| TS-04 | Persistent 409 exhausts retries | Unit Test | Covered |
| TS-05 | Context cancellation aborts retry | Unit Test | Covered |
| TS-06 | Update-branch failure resilience | Unit Test | **Gap — recommend adding** |
| TS-07 | E2E enrollment PR merge | E2E Test | Covered |

### 5.1 Test Counts

| Type | Count |
|:-----|:------|
| Unit Tests | 6 (5 existing + 1 recommended) |
| E2E Tests | 1 (existing) |
| **Total** | **7** |

---

## 6. Recommendations

1. **Add TS-06 (update-branch failure resilience)**: The current test suite does
   not verify behavior when the update-branch API returns an error. The
   implementation silently continues, but this should be explicitly tested to
   prevent regressions if someone adds error handling to that call.

2. **Consider testing concurrent merge races**: In production, multiple agents
   could call `MergeChangeProposal` on different PRs targeting the same base
   branch, creating cascading 409s. While this is an integration-level concern,
   a unit test simulating alternating 409/success patterns would increase
   confidence.

---

*Generated by QualityFlow STP Builder — 2026-06-22*
