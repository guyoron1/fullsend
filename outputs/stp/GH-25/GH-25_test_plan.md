# FullSend Test Plan

| Field | Value |
|:------|:------|
| **Ticket** | GH-25 |
| **Title** | perf(#2351): batch path-existence checks via Git Trees API |
| **Author** | QualityFlow |
| **Date** | 2026-06-18 |
| **Version** | 0.x |
| **Product** | FullSend |
| **Platform** | GitHub Actions |
| **Status** | Draft |

---

## 1. Summary

This test plan covers the changes introduced in PR #25 (mirror of fullsend-ai/fullsend#2360), which adds a batched file-listing capability to the `forge.Client` interface using the GitHub Git Trees API. The primary goal is to replace the O(N) `GetFileContent` pattern used by `ComparePathPresence` with a single recursive tree fetch, reducing 100+ sequential API calls to 3 fixed calls regardless of path count.

### 1.1 Scope

**In Scope:**
- New `forge.Client.ListRepositoryFiles(ctx, owner, repo)` interface method
- `github.LiveClient.ListRepositoryFiles` implementation (Git Trees API: refs → commit → tree?recursive=1)
- `forge.FakeClient.ListRepositoryFiles` test-double implementation
- `scaffold.ComparePathPresence` refactored to use batched file listing
- `harness.DiscoverRemoteAgents` — new remote agent discovery function
- `harness.Lint` — new harness diagnostics function
- `config.OrgConfig` changes (new `MintURL` field, dispatch mode)
- `cli/run.go` and `cli/reconcilestatus.go` — updated status/dispatch logic
- `statuscomment` — expanded status comment management

**Out of Scope:**
- Upstream PR (fullsend-ai/fullsend#2360) — tested separately in upstream CI
- Workflow YAML changes (`.github/workflows/reusable-*.yml`) — infrastructure, not application logic
- Documentation-only files (`docs/`, `README.md`)
- Scaffold template files (`internal/scaffold/fullsend-repo/`) — static content
- External dependencies (GitHub API availability, network conditions)

### 1.2 Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|:-----|:-----------|:-------|:-----------|
| Truncated tree response for very large repos | Medium | High | `ListRepositoryFiles` returns error on `truncated: true` — must be tested |
| Empty repository (no commits/tree) | Low | Medium | Test that `ErrNotFound` is returned correctly |
| API rate limiting during tree fetch (3 calls) | Low | Medium | Existing retry/backoff in `LiveClient.do()` handles this |
| `FakeClient.ListRepositoryFiles` diverges from `LiveClient` behavior | Medium | Medium | Contract tests ensure consistent interface |
| `ComparePathPresence` regression — missing paths not detected | Low | High | Existing + new test cases cover all presence patterns |

---

## 2. Requirements Mapping

| ID | Requirement | Source | Priority |
|:---|:------------|:-------|:---------|
| REQ-01 | `ListRepositoryFiles` returns all file paths in default branch via Git Trees API | PR description | Critical |
| REQ-02 | `ListRepositoryFiles` uses exactly 3 API calls (repo → ref → tree) | PR description | Major |
| REQ-03 | `ListRepositoryFiles` returns `ErrNotFound` for nonexistent repos | `forge.go` interface contract | Major |
| REQ-04 | `ListRepositoryFiles` returns error when tree is truncated | `github.go:1020-1022` | Major |
| REQ-05 | `ComparePathPresence` uses `ListRepositoryFiles` instead of per-path `GetFileContent` | `pathpresence.go` | Critical |
| REQ-06 | `ComparePathPresence` returns sorted missing paths | `pathpresence.go:35` | Normal |
| REQ-07 | `FakeClient.ListRepositoryFiles` enumerates `FileContents` keys | `fake.go:403-419` | Major |
| REQ-08 | `DiscoverRemoteAgents` discovers agent roles from remote harness files | `discover_remote.go` | Major |
| REQ-09 | `Harness.Lint()` returns diagnostic warnings for missing role | `lint.go` | Normal |

---

## 3. Test Scenarios

### 3.1 `forge.Client.ListRepositoryFiles` — Interface Contract

| ID | Scenario | Tier | Requirement | Expected Result |
|:---|:---------|:-----|:------------|:----------------|
| TS-GH-25-001 | List files in repo with multiple files across nested directories | Tier1 | REQ-01 | Returns all blob paths, excludes tree (directory) entries |
| TS-GH-25-002 | List files in empty repo (no commits) | Tier1 | REQ-03 | Returns `forge.ErrNotFound` or empty slice |
| TS-GH-25-003 | List files in nonexistent repo | Tier1 | REQ-03 | Returns error wrapping `forge.ErrNotFound` |
| TS-GH-25-004 | Tree response is truncated (very large repo) | Tier1 | REQ-04 | Returns error containing "truncated" |
| TS-GH-25-005 | API call count is exactly 3 (repo → ref → tree) for normal repo | Tier1 | REQ-02 | Verified via httptest request counting |

### 3.2 `github.LiveClient.ListRepositoryFiles` — Implementation

| ID | Scenario | Tier | Requirement | Expected Result |
|:---|:---------|:-----|:------------|:----------------|
| TS-GH-25-006 | Happy path: mock GitHub API returns repo info, ref, and recursive tree | Tier1 | REQ-01 | Returns correct file paths |
| TS-GH-25-007 | Repo API returns 404 | Tier1 | REQ-03 | Returns `forge.ErrNotFound` |
| TS-GH-25-008 | Branch ref API returns 404 (async repo init) | Tier1 | REQ-01 | Retries via `retryOnTransient`, eventually succeeds or fails |
| TS-GH-25-009 | Tree API returns `truncated: true` | Tier1 | REQ-04 | Returns descriptive error |
| TS-GH-25-010 | Tree contains mix of blobs and tree entries | Tier1 | REQ-01 | Only blob paths returned |
| TS-GH-25-011 | Rate limit (429) during tree fetch | Tier1 | REQ-01 | Retry logic in `do()` handles it transparently |

### 3.3 `forge.FakeClient.ListRepositoryFiles` — Test Double

| ID | Scenario | Tier | Requirement | Expected Result |
|:---|:---------|:-----|:------------|:----------------|
| TS-GH-25-012 | FakeClient with populated FileContents returns matching paths | Tier1 | REQ-07 | Returns paths stripped of "owner/repo/" prefix |
| TS-GH-25-013 | FakeClient with empty FileContents returns empty slice | Tier1 | REQ-07 | Returns nil/empty |
| TS-GH-25-014 | FakeClient with injected error returns that error | Tier1 | REQ-07 | Returns injected error |
| TS-GH-25-015 | FakeClient FileContents with multiple repos returns only target repo paths | Tier1 | REQ-07 | Paths from other repos excluded |

### 3.4 `scaffold.ComparePathPresence` — Batched Path Checking

| ID | Scenario | Tier | Requirement | Expected Result |
|:---|:---------|:-----|:------------|:----------------|
| TS-GH-25-016 | All expected paths exist in repo | Tier1 | REQ-05 | Returns empty missing slice, no error |
| TS-GH-25-017 | Some expected paths missing | Tier1 | REQ-05, REQ-06 | Returns sorted list of missing paths |
| TS-GH-25-018 | All expected paths missing | Tier1 | REQ-05, REQ-06 | Returns all paths sorted |
| TS-GH-25-019 | Empty expected paths slice | Tier1 | REQ-05 | Returns nil immediately (no API call) |
| TS-GH-25-020 | Forge error during ListRepositoryFiles | Tier1 | REQ-05 | Returns wrapped error |
| TS-GH-25-021 | Verify GetFileContent is never called (batch behavior) | Tier1 | REQ-05 | GetFileContent error injection does not trigger |

### 3.5 `harness.DiscoverRemoteAgents` — Remote Agent Discovery

| ID | Scenario | Tier | Requirement | Expected Result |
|:---|:---------|:-----|:------------|:----------------|
| TS-GH-25-022 | Discover agents from remote harness directory with YAML files | Tier1 | REQ-08 | Returns sorted AgentInfo slice with role and slug |
| TS-GH-25-023 | Harness directory does not exist (ErrNotFound) | Tier1 | REQ-08 | Returns (nil, nil) |
| TS-GH-25-024 | Harness directory contains non-YAML files | Tier1 | REQ-08 | Non-YAML files skipped |
| TS-GH-25-025 | Parse error in one harness file, others valid | Tier1 | REQ-08 | Valid agents returned, error contains parse failure |
| TS-GH-25-026 | Harness file with empty role and slug | Tier1 | REQ-08 | File skipped, not in results |
| TS-GH-25-027 | Results sorted by Role then Filename | Tier1 | REQ-08 | Deterministic ordering verified |

### 3.6 `harness.Lint` — Harness Diagnostics

| ID | Scenario | Tier | Requirement | Expected Result |
|:---|:---------|:-----|:------------|:----------------|
| TS-GH-25-028 | Harness with empty role field | Tier1 | REQ-09 | Returns warning diagnostic for "role" |
| TS-GH-25-029 | Harness with role set | Tier1 | REQ-09 | Returns nil (no diagnostics) |
| TS-GH-25-030 | Diagnostic severity String() coverage | Tier1 | REQ-09 | "warning" and "error" strings correct |

---

## 4. Regression Analysis

### 4.1 LSP Call Graph Summary

Analysis performed using gopls LSP on the source repository.

**`ComparePathPresence` callers (6 test call sites):**
- `TestComparePathPresence_AllPresent` (pathpresence_test.go:14)
- `TestComparePathPresence_SomeMissing` (pathpresence_test.go:32)
- `TestComparePathPresence_AllMissing` (pathpresence_test.go:53)
- `TestComparePathPresence_EmptyExpected` (pathpresence_test.go:66)
- `TestComparePathPresence_ForgeError` (pathpresence_test.go:78)
- `TestComparePathPresence_UsesOneAPICall` (pathpresence_test.go:92)

No production callers found in the current PR branch — `ComparePathPresence` is a new function meant to replace scattered `GetFileContent` call patterns.

**`ListRepositoryFiles` references (4 sites across 3 files):**
- `forge.go:199` — interface definition
- `fake_test.go:475,551` — fake client test coverage
- `pathpresence.go:20` — production consumer

**`forge.Client` interface references (100+ sites across 33 files):**
The `Client` interface is the central abstraction used by all forge-dependent code. Adding `ListRepositoryFiles` extends the interface, requiring all implementations (`LiveClient`, `FakeClient`) to satisfy it. LSP confirmed both implementations exist.

### 4.2 Dependency Chains

```
forge.Client.ListRepositoryFiles  (new interface method)
  ├── github.LiveClient.ListRepositoryFiles  (Git Trees API implementation)
  │     ├── LiveClient.get()  → LiveClient.do()  (HTTP + retry)
  │     ├── GET /repos/{owner}/{repo}  (default branch)
  │     ├── GET /repos/{owner}/{repo}/git/ref/heads/{branch}  (commit SHA)
  │     └── GET /repos/{owner}/{repo}/git/trees/{sha}?recursive=1  (file list)
  ├── forge.FakeClient.ListRepositoryFiles  (test double)
  │     └── FakeClient.FileContents  (in-memory map)
  └── scaffold.ComparePathPresence  (consumer)
        └── set membership check  (local, no API)
```

### 4.3 Regression Risk Areas

| Area | Risk | Test Coverage |
|:-----|:-----|:-------------|
| `forge.Client` interface compatibility | All implementations must add `ListRepositoryFiles` | Compile-time `var _ Client = (*LiveClient)(nil)` check |
| `ComparePathPresence` behavior change | Was O(N) `GetFileContent`, now O(1) batch | 6 existing test cases + TS-GH-25-021 verifies no per-path calls |
| `retryOnTransient` reuse in `ListRepositoryFiles` | Shared retry logic used by commit/file ops | Existing retry tests cover `retryOnTransient` |
| `DiscoverRemoteAgents` depends on `ListDirectoryContents` + `GetFileContentAtRef` | Existing forge methods, no new API surface | New test file `discover_remote_test.go` (226 additions) |

---

## 5. Test Environment

| Component | Details |
|:----------|:--------|
| **Language** | Go 1.22+ |
| **Test Framework** | `testing` + `github.com/stretchr/testify` |
| **HTTP Mocking** | `net/http/httptest` for `LiveClient` tests |
| **Forge Mocking** | `forge.FakeClient` for unit tests |
| **CI Platform** | GitHub Actions |
| **Build Command** | `go test ./...` |

---

## 6. Test Execution Strategy

### 6.1 Tier 1 — Unit Tests (30 scenarios)

All scenarios listed above are Tier 1 unit tests. They use `forge.FakeClient` or `httptest` servers and run in-process with no external dependencies.

**Execution:** `go test ./internal/forge/... ./internal/scaffold/... ./internal/harness/...`

**Pass Criteria:** All tests pass, no race conditions (`-race` flag).

### 6.2 Integration Considerations

The `ListRepositoryFiles` implementation makes real GitHub API calls. Integration testing would require:
- A test repository with known file structure
- Valid GitHub token with `contents:read` scope
- Network access to `api.github.com`

These are covered by the upstream repo's CI and are out of scope for this STP.

---

## 7. Test Counts

| Tier | Count |
|:-----|:------|
| Tier 1 (Unit) | 30 |
| Tier 2 (Integration) | 0 |
| **Total** | **30** |

---

## 8. Approval

| Role | Name | Date | Status |
|:-----|:-----|:-----|:-------|
| Author | QualityFlow | 2026-06-18 | Complete |
| Reviewer | — | — | Pending |
