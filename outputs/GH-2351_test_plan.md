# Fullsend Test Plan

## **Batch Path-Existence Checks via Git Trees API - Quality Engineering Plan**

### Metadata & Tracking

- **Enhancement:** [GH-2351](https://github.com/fullsend-ai/fullsend/issues/2351) — Batch path-existence checks via Git Trees API
- **Feature Tracking:** [GH-2351](https://github.com/fullsend-ai/fullsend/issues/2351)
- **Epic Tracking:** N/A
- **QE Owner:** QualityFlow (automated)
- **Owning SIG:** N/A
- **Participating SIGs:** N/A

**Document Conventions:** Standard STP format. Tier classifications follow the Unit Tests / Functional / End-to-End taxonomy. Priority levels: P0 (core functionality), P1 (important functionality), P2 (edge cases).

### Feature Overview

This feature adds a new `ListRepositoryFiles` method to the `forge.Client` interface that retrieves all file paths in a repository's default branch using a single recursive Git Trees API call. The new `ComparePathPresence` function in the `scaffold` package uses this method to batch-check which expected paths exist in a repo, replacing an O(N) sequential `GetFileContent` pattern with O(1) API calls (3 fixed calls regardless of path count). The change spans the interface definition, the GitHub `LiveClient` implementation, the `FakeClient` test double, and a comprehensive test suite. This is preparatory work for PR #1954 which will introduce the production caller in `vendormanifest.go`.

---

### Section I — Motivation and Requirements Review

#### I.1 — Requirement & User Story Review Checklist

- [ ] **Reviewed the relevant requirements.**
  - GH-2351 specifies adding `ListRepositoryFiles` to replace O(N) `GetFileContent` calls with a single Git Trees API call for batch path-existence checks.
  - Commit message provides clear scope: interface addition, GitHub implementation, fake client implementation, `ComparePathPresence` function, and tests.

- [ ] **Confirmed clear user stories and understood. Understand the value and customer use cases.**
  - The value is a performance improvement: reducing 100+ sequential API calls to 3 fixed calls regardless of path count.
  - User story: as a scaffold component, I need to check whether expected files exist in a repository without making one API call per file.

- [ ] **Confirmed requirements are **testable and unambiguous**.**
  - Requirements are testable: the function accepts expected paths, returns missing paths, and uses a single batch API call instead of per-path calls.
  - The test suite includes a guard test (`TestComparePathPresence_UsesOneAPICall`) that injects an error on `GetFileContent` to prove it is never called.

- [ ] **Ensured acceptance criteria are **defined clearly**.**
  - Acceptance criteria are implied by the commit scope: `ListRepositoryFiles` returns all file paths via Git Trees API; `ComparePathPresence` identifies missing paths using batch lookup; `FakeClient` implements the interface for testing; all tests pass.

- [ ] **Confirmed coverage for NFRs.**
  - Performance NFR: O(1) API calls vs O(N) — validated by design (3 fixed API calls: refs, commit, tree).
  - Thread safety NFR: `FakeClient.ListRepositoryFiles` uses mutex locking; thread safety test covers concurrent calls.
  - Error handling NFR: truncated tree returns explicit error; forge errors propagate correctly.

#### I.2 — Known Limitations

- **Truncated trees:** The Git Trees API may truncate results for very large repositories (100k+ files). The implementation returns an explicit error (`"repository tree too large (truncated)"`) rather than silently returning incomplete data. Repos hitting this limit would need an alternative approach.
- **No production caller yet:** `ComparePathPresence` has no production callers in this changeset. PR #1954 will introduce the production integration in `vendormanifest.go`. Until then, the function is tested but not exercised in production code paths.
- **Default branch only:** `ListRepositoryFiles` operates on the repository's default branch only. Branch-specific path checking is not supported by this implementation.

#### I.3 — Technology and Design Review

- [ ] **Developer handoff completed; design and implementation approach reviewed.**
  - Implementation reuses the existing refs → commit → tree pattern from `CommitFiles` in the GitHub `LiveClient`.
  - The `FakeClient` implementation derives paths from the existing `FileContents` map keys, maintaining consistency with other fake methods.

- [ ] **Technology challenges and risks identified.**
  - Git Trees API has a truncation limit for very large repositories. The implementation handles this with an explicit error.
  - The `retryOnTransient` wrapper is used for the branch ref lookup, consistent with existing patterns.

- [ ] **Test environment needs identified.**
  - Unit tests use `FakeClient` — no cluster or external service required.
  - Integration testing of `LiveClient.ListRepositoryFiles` would require a real GitHub API token and test repository.

- [ ] **API extensions and changes reviewed.**
  - New method `ListRepositoryFiles(ctx, owner, repo) ([]string, error)` added to `forge.Client` interface.
  - All existing `Client` implementations must implement this method (breaking interface change).

- [ ] **Topology and deployment requirements reviewed.**
  - No topology or deployment changes. This is a client-side library change with no infrastructure impact.

### Section II — Test Planning

#### II.1 — Scope of Testing

This test plan covers the `ListRepositoryFiles` method added to the `forge.Client` interface and its implementations (`LiveClient` for GitHub, `FakeClient` for testing), as well as the `ComparePathPresence` function in the `scaffold` package that uses this method for batch path-existence checking.

**Testing Goals:**

- **P0:** Verify `ComparePathPresence` correctly identifies missing and present paths using batch lookup
- **P0:** Verify `FakeClient.ListRepositoryFiles` correctly derives paths from `FileContents` map
- **P1:** Verify `LiveClient.ListRepositoryFiles` correctly calls Git Trees API (refs → commit → tree?recursive=1)
- **P1:** Verify error handling for API failures, truncated trees, and missing repositories
- **P2:** Verify thread safety of concurrent `ListRepositoryFiles` calls on `FakeClient`

**Out of Scope (Testing Scope Exclusions):**

- [ ] **GitHub API behavior and rate limiting** — Platform-level concern tested by GitHub; we test our client's handling of API responses.
- [ ] **Git Trees API correctness** — We assume the API returns correct data; we test our parsing and error handling.
- [ ] **Production integration with `vendormanifest.go`** — Deferred to PR #1954 which introduces the production caller.
- [ ] **Branch-specific file listing** — Not supported by this implementation; only default branch is in scope.

#### II.2 — Test Strategy

**Functional:**

- [x] **Functional Testing**
  - Verify core `ComparePathPresence` behavior: all present, some missing, all missing, empty input.
  - Verify `ListRepositoryFiles` implementations return correct paths.
  - Verify error propagation from forge client to caller.

- [x] **Automation Testing**
  - All tests are automated Go unit tests using `testify/assert` and `testify/require`.
  - Tests use `FakeClient` for deterministic, fast execution.

- [x] **Regression Testing**
  - Guard test (`TestComparePathPresence_UsesOneAPICall`) ensures the batch pattern is maintained.
  - Error injection on `GetFileContent` prevents regression to per-path calling pattern.

**Non-Functional:**

- [ ] **Performance Testing**
  - Not applicable at unit test level. Performance benefit (O(1) vs O(N) API calls) is architectural and validated by design.

- [ ] **Scale Testing**
  - Not applicable. The Git Trees API handles scale; truncation error handling is tested.

- [ ] **Security Testing**
  - Not applicable. No new authentication or authorization logic introduced.

- [ ] **Usability Testing**
  - Not applicable. Internal API, no user-facing interface changes.

- [ ] **Monitoring**
  - Not applicable. No new metrics or observability changes.

**Integration & Compatibility:**

- [ ] **Compatibility Testing**
  - Not applicable. No version compatibility concerns for this internal API addition.

- [ ] **Upgrade Testing**
  - Not applicable. Interface addition is backward-compatible at the binary level.

- [ ] **Dependencies**
  - No new dependencies introduced. Uses existing `forge` and `scaffold` packages.

- [ ] **Cross Integrations**
  - Integration with `vendormanifest.go` deferred to PR #1954.

**Infrastructure:**

- [ ] **Cloud Testing**
  - Not applicable. No cloud-specific infrastructure changes.

#### II.3 — Test Environment

- **Cluster Topology:** Not required — all tests run locally with mocked dependencies
- **Platform Version:** Go 1.x (as specified in go.mod)
- **CPU Virtualization:** Not applicable
- **Compute:** Standard CI runner
- **Special Hardware:** None required
- **Storage:** None required
- **Network:** None required for unit tests; GitHub API access needed for integration tests
- **Operators:** None
- **Platform:** Linux/macOS CI environment
- **Special Configs:** `GITHUB_TOKEN` environment variable for integration tests against live API

#### II.3.1 — Testing Tools & Frameworks

No new or special tools required. Standard Go testing with `testify`.

#### II.4 — Entry Criteria

- [ ] All code changes from GH-2351 merged to feature branch
- [ ] `go build ./...` succeeds without errors
- [ ] `go vet ./...` reports no issues
- [ ] CI pipeline is green on the PR branch

#### II.5 — Risks

- [ ] **Timeline**
  - Risk: PR #1954 (production caller) may introduce integration issues not caught by unit tests alone.
  - Mitigation: Guard test ensures batch pattern is enforced; integration tests will be added with PR #1954.
  - Status: [ ] Monitoring

- [ ] **Coverage**
  - Risk: `LiveClient.ListRepositoryFiles` is not tested with a real GitHub API in this changeset.
  - Mitigation: Implementation reuses proven refs → commit → tree pattern from `CommitFiles`; manual verification against live API recommended.
  - Status: [ ] Accepted

- [ ] **Environment**
  - Risk: Large repositories may hit Git Trees API truncation limit.
  - Mitigation: Explicit error returned for truncated trees; documented as known limitation.
  - Status: [ ] Mitigated

- [ ] **Untestable**
  - Risk: None identified. All new code is testable via `FakeClient`.
  - Mitigation: N/A
  - Status: [x] Clear

- [ ] **Resources**
  - Risk: None. No additional test infrastructure required.
  - Mitigation: N/A
  - Status: [x] Clear

- [ ] **Dependencies**
  - Risk: Breaking interface change requires all `forge.Client` implementations to add `ListRepositoryFiles`.
  - Mitigation: Only two implementations exist (`LiveClient`, `FakeClient`); both updated in this changeset.
  - Status: [x] Mitigated

- [ ] **Other**
  - Risk: None identified.
  - Mitigation: N/A
  - Status: [x] Clear

---

### Section III — Requirements-to-Tests Mapping

#### III.1 — Requirements Mapping

- **Requirement ID:** GH-2351
  **Requirement Summary:** Batch file listing returns all repository file paths via single Git Trees API call
  **Test Scenarios:**
  - Verify `ListRepositoryFiles` returns all blob paths from recursive tree (positive)
  - Verify `ListRepositoryFiles` returns error for truncated tree response (negative)
  - Verify `ListRepositoryFiles` returns `ErrNotFound` for nonexistent repository (negative)
  **Tier:** Unit Tests
  **Priority:** P0

- **Requirement ID:**
  **Requirement Summary:** `ComparePathPresence` correctly identifies missing paths using batch lookup
  **Test Scenarios:**
  - Verify all paths reported present when all exist in repo (positive)
  - Verify correct missing paths returned when some are absent (positive)
  - Verify all paths reported missing for empty repository (positive)
  - Verify empty input returns nil without API calls (edge case)
  - Verify error propagation when `ListRepositoryFiles` fails (negative)
  **Tier:** Unit Tests
  **Priority:** P0

- **Requirement ID:**
  **Requirement Summary:** `ComparePathPresence` uses batch API pattern instead of per-path calls
  **Test Scenarios:**
  - Verify `GetFileContent` is never called by `ComparePathPresence` (guard test — positive)
  - Verify single `ListRepositoryFiles` call replaces N `GetFileContent` calls (positive)
  **Tier:** Unit Tests
  **Priority:** P0

- **Requirement ID:**
  **Requirement Summary:** `FakeClient` implements `ListRepositoryFiles` using `FileContents` map keys
  **Test Scenarios:**
  - Verify `FakeClient` returns paths matching `owner/repo/` prefix from `FileContents` (positive)
  - Verify `FakeClient` returns empty slice for no matching files (positive)
  - Verify `FakeClient` returns injected error when configured (negative)
  **Tier:** Unit Tests
  **Priority:** P1

- **Requirement ID:**
  **Requirement Summary:** `FakeClient.ListRepositoryFiles` is thread-safe under concurrent access
  **Test Scenarios:**
  - Verify no data races with 20 concurrent goroutines calling `ListRepositoryFiles` (positive)
  **Tier:** Unit Tests
  **Priority:** P2

- **Requirement ID:**
  **Requirement Summary:** GitHub `LiveClient` implements `ListRepositoryFiles` via refs/commit/tree API chain
  **Test Scenarios:**
  - Verify `LiveClient` follows refs → commit SHA → tree SHA → recursive tree pipeline (positive)
  - Verify `LiveClient` filters tree entries to blobs only, excluding tree-type entries (positive)
  - Verify `LiveClient` returns error when default branch ref lookup fails (negative)
  - Verify `LiveClient` retries transient errors on branch ref lookup (positive)
  **Tier:** Functional
  **Priority:** P1

---

### Section IV — Sign-off

| Role | Name | Date | Signature |
|:-----|:-----|:-----|:----------|
| QE Lead | | | |
| Dev Lead | | | |
| PM | | | |
