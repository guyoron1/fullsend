# Test Plan — GH-73: Two-Pass Review Strategy for Large PRs

| Field | Value |
|:------|:------|
| **Ticket** | [GH-73](https://github.com/guyoron1/fullsend/pull/73) |
| **Title** | feat(#2096): add two-pass review strategy for large PRs |
| **Author** | guyoron1 |
| **Product** | fullsend |
| **Date** | 2026-06-22 |
| **Status** | Open |
| **Branch** | `mirror/2303-2096-two-pass-review-strategy` → `main` |
| **Upstream** | fullsend-ai/fullsend#2303 |
| **QE Owner** | TBD |
| **Team** | fullsend |
| **Enhancement** | [fullsend-ai/fullsend#2303](https://github.com/fullsend-ai/fullsend/pull/2303) |

---

## I. Pre-Test Analysis

### I.1 Requirements Review

- [x] **Review Requirements**
  - PR introduces two-pass review strategy for large PRs, including review posting, stale-head detection, inline comment mapping, stale review cleanup, and formal review submission
  - Upstream PR fullsend-ai/fullsend#2303 with 18,029 additions / 2,300 deletions across 174 files
- [x] **Understand Value and Customer Use Cases**
  - Improves review quality for large PRs by enabling structured review with inline comments on specific diff hunks
  - Prevents approval of unreviewed code through stale-head detection
  - Automates cleanup of outdated review comments
- [x] **Testability**
  - All core review pipeline functions are testable via the existing `forge.FakeClient` interface
  - Stale-head detection, inline comment mapping, and review submission are deterministic and unit-testable
  - SHA validation and input sanitization are pure functions
- [x] **Acceptance Criteria**
  - Post-review command correctly parses JSON and plaintext review input
  - Stale-head detection prevents review when PR HEAD has changed
  - Inline comments are mapped to correct diff hunk lines
  - Stale reviews are dismissed or minimized on new review submission
  - Exit code 10 propagates for stale-head condition
- [x] **Non-Functional Requirements**
  - GitHub API rate limiting handled gracefully with fallback behavior
  - SHA validation prevents injection attacks

### I.2 Known Limitations

- [ ] **No real GitHub API integration tests** — E2E tests use fake forge client; actual GitHub API behavior differences (422 errors for out-of-hunk comments) cannot be validated without live API access
- [ ] **Shell script exit code propagation untested** — `StaleHeadExitCode` (10) is tested in Go but propagation through `post-review.sh` requires manual verification
- [ ] **Binary vendoring cross-platform coverage** — Cross-compilation tests are limited to the CI platform; other OS/arch combinations require manual verification

### I.3 Technology Review

- [x] **Developer Handoff**
  - QE kickoff should be scheduled during feature design phase; this is a mirror of upstream PR so handoff is implicit
- [x] **Technology Challenges**
  - GitHub API constraints on inline review comments: comments must reference lines within diff hunks or the API returns 422 errors
  - Stale-head race condition: PR HEAD can change between detection and review submission
- [x] **API Extensions**
  - New `forge.Client` interface methods: `ListPullRequestFileDiffs`, `DismissPullRequestReview`, `MinimizeComment`
  - New `forge.ReviewComment` and `forge.PullRequestFileDiff` types
- [x] **Test Environment Needs**
  - Standard Go test environment with `go test` runner
  - No external services required — all API interactions use `forge.FakeClient`
- [x] **Topology**
  - Single-binary CLI tool; no multi-node topology required for testing

---

## 1. Summary

This PR mirrors upstream fullsend-ai/fullsend#2303 and introduces a two-pass review strategy to improve review quality and coverage for large PRs. The change is wide-scoped (18,029 additions / 2,300 deletions across 174 files) and includes enhancements to the post-review CLI, forge interface, reconcile-status command, CLI infrastructure (vendor, mint, admin, run, discover-slugs), GCF provisioner, harness discovery/lint, scaffold, and binary vendoring.

## II. Test Planning

### II.1 Scope of Testing

- [x] **Post-review CLI command** — Review result parsing, formal review submission, stale-head detection, failure notices
- [x] **Inline comment mapping** — Finding-to-diff-hunk mapping, file-level fallback, severity passthrough
- [x] **Stale review cleanup** — Dismiss prior CHANGES_REQUESTED reviews, minimize prior COMMENT reviews
- [x] **Diff hunk parsing** — Parse unified diff `@@` headers into line ranges for comment eligibility
- [x] **Input validation** — SHA format validation, reason sanitization, repo format validation
- [x] **Reconcile status command** — Input validation, reason mapping
- [x] **Forge interface extensions** — New methods on `forge.Client` interface and GitHub implementation
- [x] **Binary vendoring** — Vendor root discovery, download with checksum, platform selection
- [x] **CLI commands** — Vendor, Mint, Admin, Run, Discover Slugs command changes
- [x] **Harness enhancements** — Remote discovery, linting, scaffold integration
- [x] **GCF provisioner** — Refactored provisioner interface, fake client

**Out of Scope:**

- [ ] **GitHub Actions workflow YAML changes** — `.github/workflows/` changes are configuration; validated by CI, not unit tests
- [ ] **Documentation and ADR changes** — Multiple ADRs and agent docs added; these are prose documents not requiring functional testing
- [ ] **UI/frontend behavior** — No UI components exist in this change set
- [ ] **Performance benchmarking** — Two-pass review adds one additional API call per review; binary download is a one-time operation during vendor setup; review API calls are bounded by finding count (typically <50); no user-facing latency SLA exists for the review pipeline
- [ ] **Live GitHub API integration** — All tests use `forge.FakeClient`; live API testing is outside automated test scope (see Known Limitations I.2)

### II.2 Testing Goals

1. Verify the post-review command correctly parses both JSON and plaintext review input into structured `ReviewResult` objects
2. Verify stale-head detection prevents review submission when the PR HEAD SHA has changed since the review was generated
3. Verify inline comments are placed on correct diff hunk lines and fall back to file-level comments when lines are outside hunks
4. Verify stale review cleanup dismisses prior bot reviews without affecting other users' reviews
5. Verify input validation rejects malformed SHAs and injection attempts while accepting valid formats
6. Verify all new `forge.Client` interface methods are correctly implemented by both the live GitHub client and the fake test client

### II.3 Test Environment

- Go 1.22+ with `go test` runner
- `github.com/stretchr/testify` for assertions (assert + require)
- `forge.FakeClient` providing in-memory forge implementation for all API interactions
- No external services, databases, or network access required for unit/integration tests
- E2E tests (`e2e/admin/`) require a running fullsend instance

#### II.3.1 Testing Tools & Frameworks

- No non-standard tools required — all tests use the Go stdlib `testing` package and testify assertions

### II.4 Entry / Exit Criteria

**Entry Criteria:**
- PR branch compiles without errors (`go build ./...`)
- All existing tests pass on the base branch (`go test ./...`)
- `forge.FakeClient` implements all new interface methods

**Exit Criteria:**
- All test scenarios in Section 3 pass
- No CRITICAL or HIGH-priority test failures
- Code coverage for `internal/cli/postreview.go` ≥ 80%

### II.5 Test Strategy Classifications

- [x] **Functional Testing** — Core feature; all test scenarios validate functional behavior
- [x] **Automation Testing** — All tests are automated Go tests
- [ ] **Performance Testing** — N/A; two-pass review adds one additional API call with negligible latency impact
- [x] **Security Testing** — SHA validation and input sanitization prevent injection attacks (TC-054 through TC-062)
- [ ] **Usability Testing** — N/A; no UI components in this change
- [ ] **Upgrade Testing** — N/A; CLI tool with no persistent state requiring migration
- [x] **Regression Testing** — Backward compatibility of CLI commands and forge interface verified through existing test suite
- [ ] **Monitoring Testing** — N/A; no new metrics or alerts introduced
- [x] **Dependencies** — None; all changes are self-contained within the fullsend repository

### II.6 Risks

| Risk | Likelihood | Impact | Mitigation |
|:-----|:-----------|:-------|:-----------|
| Large PR scope masks subtle regressions | Medium | High | Focus testing on LSP-traced call chains (see Section 4.1); prioritize review pipeline tests for `submitFormalReview`, `findingsToReviewComments`, and `checkStaleHead` |
| GitHub API rate limiting during inline comment posting | Low | Medium | Graceful fallback when `ListPullRequestFileDiffs` fails |
| Stale-head race condition (HEAD changes between check and review submit) | Low | High | `commitSHA` parameter pins review to checked commit |
| Forge interface breakage (missing method implementations) | Low | High | Compile-time interface check (`var _ forge.Client = (*LiveClient)(nil)`) |
| Exit code 10 not propagated through shell scripts | Low | Medium | Verify post-review.sh handles `StaleHeadExitCode` |

---

## 2. Scope of Changes

### 2.1 Components Affected

| Component | Files | Change Type |
|:----------|:------|:------------|
| Post-Review CLI | `internal/cli/postreview.go`, `internal/cli/postreview_test.go`, `internal/cli/qf_postreview_test.go` | Modified / Added |
| Forge Interface | `internal/forge/forge.go`, `internal/forge/fake.go`, `internal/forge/fake_test.go` | Modified |
| Forge GitHub Impl | `internal/forge/github/github.go`, `internal/forge/github/github_test.go`, `internal/forge/github/github_comment_test.go` | Modified |
| Reconcile Status | `internal/cli/reconcilestatus.go`, `internal/cli/reconcilestatus_test.go`, `internal/cli/qf_reconcilestatus_test.go` | Modified / Added |
| CLI — Vendor | `internal/cli/vendor.go`, `internal/cli/vendor_test.go`, `internal/cli/qf_vendor_test.go` | Modified / Added |
| CLI — Mint | `internal/cli/mint.go`, `internal/cli/mint_setup.go`, `internal/cli/mint_test.go`, `internal/cli/qf_mint_test.go` | Modified / Added |
| CLI — Admin | `internal/cli/admin.go`, `internal/cli/admin_test.go` | Modified |
| CLI — Run | `internal/cli/run.go`, `internal/cli/run_test.go`, `internal/cli/qf_run_test.go` | Modified / Added |
| CLI — Discover Slugs | `internal/cli/discover_slugs.go`, `internal/cli/discover_slugs_test.go` | Added |
| Binary / Vendoring | `internal/binary/acquire.go`, `internal/binary/download.go`, `internal/binary/vendorroot.go`, `internal/binary/download_test.go`, `internal/binary/qf_download_test.go`, `internal/binary/vendorroot_test.go`, `internal/binary/qf_vendorroot_test.go` | Modified / Added |
| GCF Provisioner | `internal/dispatch/gcf/provisioner.go`, `internal/dispatch/gcf/provisioner_test.go`, `internal/dispatch/gcf/fakeclient.go`, `internal/dispatch/gcf/fakeclient_test.go`, `internal/dispatch/gcf/qf_provisioner_test.go` | Modified / Added |
| Config | `internal/config/config.go`, `internal/config/config_test.go` | Modified |
| Harness | `internal/harness/harness.go`, `internal/harness/discover_remote.go`, `internal/harness/discover_remote_test.go`, `internal/harness/lint.go`, `internal/harness/lint_test.go`, `internal/harness/qf_discover_test.go`, `internal/harness/qf_lint_test.go`, `internal/harness/scaffold_integration_test.go` | Modified / Added |
| E2E Tests | `e2e/admin/admin_test.go` | Modified |
| Workflows | `.github/workflows/e2e.yml`, `.github/workflows/reusable-*.yml` | Modified |
| Documentation | Multiple ADRs, agent docs, plans, specs | Added / Modified |

### 2.2 Critical Integration Points

The review posting pipeline has five key integration points that drive test prioritization:

- **Review result parsing** → `parseReviewResult()` — Entry point for all review input; supports both JSON and plaintext formats
- **Stale-head detection** → `checkStaleHead()` — Safety gate comparing reviewed SHA against current PR HEAD; returns `staleHeadError` with exit code 10 on mismatch
- **Formal review submission** → `submitFormalReview()` — Orchestrates stale review cleanup, inline comment mapping, and GitHub review creation
- **Inline comment mapping** → `findingsToReviewComments()` — Converts structured findings to diff-hunk-aware inline comments; falls back to file-level comments for lines outside hunks
- **Forge interface** → `forge.Client` — Extended with `ListPullRequestFileDiffs`, `DismissPullRequestReview`, and `MinimizeComment` methods; all implementations (live + fake) must satisfy the interface

---

## 3. Test Scenarios

### 3.0 Two-Pass Review Orchestration

| ID | Scenario | Expected Result | Priority |
|:---|:---------|:----------------|:---------|
| TC-095 | PR with diff exceeding large-PR threshold triggers two review passes | Review agent dispatched twice; second pass receives first-pass context | High |
| TC-096 | PR with diff below large-PR threshold triggers single review pass | Review agent dispatched once; no second-pass dispatch | High |
| TC-097 | Second pass produces findings that refine or override first-pass findings | Final review comment reflects merged findings from both passes | High |
| TC-098 | First pass fails with error; second pass is not dispatched | Error propagated; no second pass attempted | Medium |

### 3.1 Post-Review — Review Result Parsing

| ID | Scenario | Expected Result | Priority |
|:---|:---------|:----------------|:---------|
| TC-001 | Parse valid JSON with body and action | Returns `ReviewResult` with correct fields | High |
| TC-002 | Parse plain text input (non-JSON) | Returns `ReviewResult` with body=input, action="comment" | High |
| TC-003 | Parse JSON with missing action field | Defaults action to "comment" | Medium |
| TC-004 | Parse JSON with empty body and non-failure action | Returns error containing "empty body" | High |
| TC-005 | Parse JSON with action="failure" and empty body | Succeeds; failure action allows empty body | High |
| TC-006 | Parse JSON with head_sha field | Correctly extracts HeadSHA | Medium |
| TC-007 | Parse JSON with findings array | Correctly deserializes findings with all fields | Medium |

### 3.2 Post-Review — Stale Head Detection

| ID | Scenario | Expected Result | Priority |
|:---|:---------|:----------------|:---------|
| TC-008 | PR HEAD matches reviewed SHA | Returns stale=false, currentSHA=HEAD | High |
| TC-009 | PR HEAD differs from reviewed SHA | Returns stale=true, currentSHA=new HEAD | High |
| TC-010 | Dry-run mode | Returns stale=false without API call | Medium |
| TC-011 | Case-insensitive SHA comparison (uppercase vs lowercase) | Treats as matching (not stale) | Medium |
| TC-012 | Stale-head notice posted when HEAD moved | Posts failure comment containing "stale-head" and both SHAs | High |
| TC-013 | `staleHeadError` returns `StaleHeadExitCode` (10) | Exit code == 10; error message contains both SHAs | High |

### 3.3 Post-Review — Formal Review Submission

| ID | Scenario | Expected Result | Priority |
|:---|:---------|:----------------|:---------|
| TC-014 | Submit APPROVE review | Creates review with event=APPROVE, empty body | High |
| TC-015 | Submit REQUEST_CHANGES review with comment URL | Creates review with event=REQUEST_CHANGES, body links to sticky comment | High |
| TC-016 | Submit REQUEST_CHANGES without comment URL | Body = "See the review comment above for full details." | Medium |
| TC-017 | Submit with action="reject" | Maps to REQUEST_CHANGES event | High |
| TC-018 | Submit COMMENT with no inline findings | Skips formal review (no-op) | High |
| TC-019 | Submit COMMENT with inline-eligible findings | Submits COMMENT review with inline comments attached | High |
| TC-020 | Submit COMMENT when all findings filtered out | Skips formal review | Medium |
| TC-021 | Unknown action string | Skips formal review without error | Medium |
| TC-022 | Dry-run mode | No API calls made; review not created | Medium |
| TC-023 | Commit SHA passed to review API | Review pinned to specific commit | Medium |
| TC-024 | Empty commit SHA | Review created without commit pin | Low |

### 3.4 Post-Review — Stale Review Cleanup

| ID | Scenario | Expected Result | Priority |
|:---|:---------|:----------------|:---------|
| TC-025 | Bot has prior COMMENTED reviews | All prior reviews by bot minimized (OUTDATED) | High |
| TC-026 | Bot has prior CHANGES_REQUESTED, new verdict is APPROVE | Prior CR reviews dismissed with "Superseded" message | High |
| TC-027 | Bot has prior CHANGES_REQUESTED, new verdict is COMMENT | Prior CR reviews dismissed | High |
| TC-028 | Bot has prior CHANGES_REQUESTED, new verdict is REQUEST_CHANGES | Prior CR reviews NOT dismissed (same severity) | High |
| TC-029 | Other user's CHANGES_REQUESTED reviews | Not dismissed by bot | High |
| TC-030 | Multiple stale CR reviews by bot | All dismissed | Medium |
| TC-031 | MinimizeComment API error | Soft-fail; no panic, review still submitted | Low |
| TC-032 | GetAuthenticatedUser error | Skips cleanup; review still submitted | Low |
| TC-033 | ListPullRequestReviews error | Skips cleanup; review still submitted | Low |

### 3.5 Post-Review — Inline Comment Mapping

| ID | Scenario | Expected Result | Priority |
|:---|:---------|:----------------|:---------|
| TC-034 | Finding with file + line in diff hunk | Inline comment at correct path/line | High |
| TC-035 | Finding without file path | Omitted from inline comments | Medium |
| TC-036 | Finding with line=0 | Omitted from inline comments | Medium |
| TC-037 | Finding on file not in PR diff | Filtered out (fileFiltered incremented) | High |
| TC-038 | Finding on file in diff but line outside hunk | File-level fallback (Line=0), body includes "Line N" | High |
| TC-039 | Binary file (empty patch, nil hunks) | Line filtering skipped; comment passes through | Medium |
| TC-040 | Multiple findings across files | Each mapped correctly to respective paths | Medium |
| TC-041 | All severities (info, low, medium, high, critical) pass through | No severity-based filtering | Medium |
| TC-042 | Finding with remediation | Body includes "**Suggested fix:**" section | Low |
| TC-043 | Finding without remediation | No "Suggested fix:" in body | Low |

### 3.6 Post-Review — Diff Hunk Parsing

| ID | Scenario | Expected Result | Priority |
|:---|:---------|:----------------|:---------|
| TC-044 | Single hunk `@@ -10,5 +12,7 @@` | Range [12, 18] | High |
| TC-045 | Multiple hunks in patch | Multiple ranges returned | Medium |
| TC-046 | New file `@@ -0,0 +1,50 @@` | Range [1, 50] | Medium |
| TC-047 | Deletion-only hunk (size 0) | No range emitted | Medium |
| TC-048 | Omitted size (defaults to 1) | Range [N, N] | Low |
| TC-049 | Empty patch | Nil ranges | Low |

### 3.7 Post-Review — Failure Notices

| ID | Scenario | Expected Result | Priority |
|:---|:---------|:----------------|:---------|
| TC-050 | Failure with custom body | Posts body as-is via sticky comment | Medium |
| TC-051 | Failure without body, with reason | Posts "NOT reviewed" notice with reason | Medium |
| TC-052 | Failure without body, empty reason | Reason defaults to "unknown" | Low |
| TC-053 | Follow-up issue creation (disabled #1137) | No-op for approve actions | Low |

### 3.8 Input Validation

| ID | Scenario | Expected Result | Priority |
|:---|:---------|:----------------|:---------|
| TC-054 | Valid 40-char hex SHA | Passes validation | High |
| TC-055 | Valid 64-char hex SHA (SHA-256) | Passes validation | Medium |
| TC-056 | Short/malformed SHA | Fails validation | High |
| TC-057 | SHA with injection characters | Fails validation | High |
| TC-058 | Empty SHA | Valid (means "no SHA provided") | Medium |
| TC-059 | Reason with valid chars (alphanumeric, hyphen, underscore) | Passes validation | Medium |
| TC-060 | Reason with spaces/markdown/script injection | Fails validation | High |
| TC-061 | Invalid repo format (not owner/repo) | Returns error | High |
| TC-062 | Negative PR number | Returns error | High |

### 3.9 Reconcile Status Command

| ID | Scenario | Expected Result | Priority |
|:---|:---------|:----------------|:---------|
| TC-063 | Invalid repo format | Error containing "owner/repo" | Medium |
| TC-064 | Negative --number | Error: "must be a positive integer" | Medium |
| TC-065 | Reason "cancelled" | Maps to `ReasonCancelled` | Medium |
| TC-066 | Default reason "terminated" | Maps to `ReasonTerminated` | Medium |

### 3.10 Forge Interface — New Methods

| ID | Scenario | Expected Result | Priority |
|:---|:---------|:----------------|:---------|
| TC-067 | `ListPullRequestFileDiffs` returns files with patches | Caller can parse hunk ranges | High |
| TC-068 | `ListPullRequestFileDiffs` API error | Graceful fallback; all findings pass through unfiltered | High |
| TC-069 | `ListPullRequestFileDiffs` returns empty list | Fallback: inline comments disabled, warning printed | Medium |
| TC-070 | `DismissPullRequestReview` success | Review dismissed on forge | High |
| TC-071 | `DismissPullRequestReview` API error | Soft-fail with warning | Medium |
| TC-072 | `CreatePullRequestReview` with inline comments | Comments attached to review at correct paths/lines | High |
| TC-073 | `ReviewComment` with Line=0 | Forge translates to file-level comment | High |

### 3.11 Binary Vendoring

| ID | Scenario | Expected Result | Priority |
|:---|:---------|:----------------|:---------|
| TC-074 | Resolve vendor root from project directory with `.vendor` marker | Returns path to nearest ancestor containing `.vendor` directory | Medium |
| TC-075 | Resolve vendor root when no `.vendor` marker exists | Returns default vendor path under user home directory | Medium |
| TC-076 | Download binary and verify SHA256 checksum matches manifest entry | Download succeeds; computed hash equals manifest SHA256 | High |
| TC-077 | Download binary with checksum mismatch | Download fails with checksum verification error; partial file cleaned up | High |
| TC-078 | Select platform-specific binary for linux/amd64 | URL and filename contain correct OS and architecture suffix | Medium |

### 3.12 CLI — Vendor, Mint, Admin, Run

| ID | Scenario | Expected Result | Priority |
|:---|:---------|:----------------|:---------|
| TC-079 | Vendor command downloads and places binary at vendor root path | Binary exists at `{vendor_root}/bin/{tool_name}` with correct permissions | Medium |
| TC-080 | Vendor command with `--force` re-downloads even if binary exists | Existing binary replaced; new checksum verified | Medium |
| TC-081 | Mint setup creates WIF provider configuration with correct project ID | Config file written with GCP project, pool, and provider fields populated | Medium |
| TC-082 | Mint token command returns valid JWT for enrolled repository | Token is parseable JWT with correct `aud` and `sub` claims | High |
| TC-083 | Admin command preserves existing lock file format after refactor | Lock file written by new code is readable by previous version's parser | Medium |
| TC-084 | Run command accepts `--reviewed-sha` flag and passes SHA to post-review | ReviewResult.HeadSHA equals the provided flag value | High |
| TC-085 | Run command with `--dry-run` flag skips all API calls | No forge client methods invoked; exit code 0 | Medium |
| TC-086 | Discover slugs returns unique repository slugs from harness config | Output contains one slug per configured repository with no duplicates | Medium |

### 3.13 Harness Enhancements

| ID | Scenario | Expected Result | Priority |
|:---|:---------|:----------------|:---------|
| TC-087 | Remote discovery fetches harness YAML from GitHub repository default branch | Returned config matches content of remote `.fullsend.yml` file | Medium |
| TC-088 | Remote discovery with unreachable repository returns descriptive error | Error message contains repository URL and HTTP status code | Medium |
| TC-089 | Lint detects harness YAML with missing required `agent` field | Lint output includes finding for missing `agent` field with line number | High |
| TC-090 | Lint detects harness YAML with invalid `model` value | Lint output includes finding for invalid model with accepted values list | Medium |
| TC-091 | Scaffold integration produces valid harness YAML that passes lint | Generated YAML passes all lint rules with zero findings | Medium |

### 3.14 GCF Provisioner

| ID | Scenario | Expected Result | Priority |
|:---|:---------|:----------------|:---------|
| TC-092 | Provisioner deploys function with correct entry point and runtime | Deployed function config has `runtime=go122` and `entry_point=Handler` | Medium |
| TC-093 | Provisioner handles deployment failure with retryable error | Returns error wrapping the GCF API error; does not panic | Medium |
| TC-094 | FakeClient records all method calls for test assertion | After calling `Deploy`, `fakeclient.Calls` contains entry with correct arguments | Low |

---

## 4. Regression Impact Analysis (LSP-Traced)

### 4.1 Dependency Chains

The following dependency chains were traced via LSP `incomingCalls` and `findReferences`:

| Source Function | Callers | Risk |
|:----------------|:--------|:-----|
| `submitFormalReview` | `newPostReviewCmd` (1 production caller), 23 test callers | **High** — single integration point for all review submissions |
| `findingsToReviewComments` | `submitFormalReview` (1 production caller), 7 test callers | **High** — controls inline comment mapping for all reviews |
| `checkStaleHead` | `newPostReviewCmd` (1 production caller), 4 test callers | **High** — guards against approving unreviewed code |
| `ReviewResult` | 7 references in `postreview.go`, 4 in tests | **Medium** — struct shape affects serialization compatibility |
| `forge.ListPullRequestFileDiffs` | `submitFormalReview` (1 production caller), 1 test caller | **Medium** — new interface method; all forge implementations must satisfy |

### 4.2 Regression Risk Areas

| Area | Risk Level | Rationale |
|:-----|:-----------|:----------|
| Review comment posting | **High** | Core feature — incorrect posting means silent review failures |
| Stale-head detection | **High** | Safety mechanism — failure could approve unreviewed code |
| Inline comment filtering | **High** | GitHub API rejects comments on lines outside diff hunks (422 errors) |
| Stale review dismissal | **Medium** | Incorrect dismissal could remove valid human reviews |
| Exit code propagation | **Medium** | `StaleHeadExitCode` (10) drives re-dispatch in post-review.sh |
| Forge interface compatibility | **Medium** | New methods must be implemented by all forge backends + fakes |
| Binary vendoring | **Low** | New subsystem; isolated from review pipeline |

---

## 5. Test Strategy

### 5.1 Framework

- **Language:** Go
- **Test Framework:** `testing` (stdlib)
- **Assertion Library:** `github.com/stretchr/testify` (assert + require)
- **Package Convention:** Same-package tests
- **Test File Pattern:** `*_test.go`

### 5.2 Test Tiers

| Tier | Scenarios | Description |
|:-----|:----------|:------------|
| Unit Tests | TC-001 to TC-066, TC-074 to TC-086, TC-092 to TC-094, TC-096, TC-098 | Function-level tests with fake forge client |
| Integration Tests | TC-067 to TC-073, TC-087, TC-091, TC-095, TC-097 | Multi-component tests (forge integration, harness scaffold, two-pass orchestration) |
| E2E Tests | TC-088 to TC-090 | Harness remote discovery and linting |
| **Total** | **98** | |

### 5.3 Existing Test Coverage

The PR already includes extensive test coverage in:
- `internal/cli/postreview_test.go` — 43 tests covering all `submitFormalReview` paths
- `internal/cli/qf_postreview_test.go` — 6 QF-prefixed tests for stale-head, inline mapping, minimization
- `internal/cli/reconcilestatus_test.go` / `qf_reconcilestatus_test.go` — validation tests
- `internal/cli/mint_test.go` / `qf_mint_test.go` — mint command tests
- `internal/cli/vendor_test.go` / `qf_vendor_test.go` — vendor command tests
- `internal/cli/run_test.go` / `qf_run_test.go` — run command tests
- `internal/cli/admin_test.go` — admin command tests
- `internal/cli/discover_slugs_test.go` — slug discovery tests
- `internal/binary/*_test.go` — download and vendor root tests
- `internal/dispatch/gcf/*_test.go` — provisioner tests
- `internal/harness/*_test.go` — harness discovery, lint, scaffold tests
- `internal/forge/github/github_test.go` — forge implementation tests
- `e2e/admin/admin_test.go` — E2E admin tests

---

## 6. Recommendations

1. **Priority Testing**: Focus on TC-008 through TC-013 (stale-head detection) and TC-034 through TC-041 (inline comment mapping) — these are the highest-risk scenarios unique to the two-pass review strategy.
2. **Integration Validation**: Run the full E2E admin test suite (`e2e/admin/`) to validate backward compatibility of CLI changes.
3. **Forge Interface**: Verify that `forge.FakeClient` implements all new methods (`ListPullRequestFileDiffs`, `DismissPullRequestReview`) — existing compile-time checks should catch this.
4. **Manual Verification**: Test the post-review flow end-to-end on a real PR to validate inline comments render correctly on GitHub's UI, especially file-level fallback comments.

---

*Generated by QualityFlow STP Builder — 2026-06-22*
