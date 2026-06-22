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

---

## 1. Summary

This PR mirrors upstream fullsend-ai/fullsend#2303 and introduces a two-pass review strategy to improve review quality and coverage for large PRs. The change is wide-scoped (17,037 additions / 2,300 deletions across 90+ files) and includes enhancements to the post-review CLI, forge interface, reconcile-status command, CLI infrastructure (vendor, mint, admin, run, discover-slugs), GCF provisioner, harness discovery/lint, scaffold, and binary vendoring.

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

### 2.2 Key Functions (LSP Call Graph Analysis)

The following functions form the critical path of the review posting pipeline:

```
newPostReviewCmd()
  ├── parseReviewResult()     — Parse JSON/plaintext review input
  ├── checkStaleHead()        — Compare reviewed SHA vs current PR HEAD
  │   └── forge.Client.GetPullRequestHeadSHA()
  ├── postStaleHeadNotice()   — Post failure when HEAD moved (returns staleHeadError)
  │   └── sticky.Post()
  ├── postFailureNotice()     — Post failure notice for agent errors
  │   └── sticky.Post()
  ├── sticky.Post()           — Upsert sticky review comment
  └── submitFormalReview()    — Core review submission
      ├── forge.Client.GetAuthenticatedUser()
      ├── forge.Client.ListPullRequestReviews()
      ├── dismissStaleRequestChanges()
      │   └── forge.Client.DismissPullRequestReview()
      ├── minimizeStaleReviews()
      │   └── forge.Client.MinimizeComment()
      ├── forge.Client.ListPullRequestFileDiffs()
      ├── findingsToReviewComments()   — Convert findings to inline comments
      │   ├── lineInHunks()
      │   ├── parseDiffLineRanges()
      │   └── formatFindingComment()
      └── forge.Client.CreatePullRequestReview()
```

### 2.3 Data Types

| Type | Location | Purpose |
|:-----|:---------|:--------|
| `ReviewResult` | `internal/cli/postreview.go:150` | Parsed review input (body, action, head_sha, reason, findings) |
| `ReviewFinding` | `internal/cli/postreview.go:159` | Structured finding (severity, category, file, line, description, remediation) |
| `staleHeadError` | `internal/cli/postreview.go:214` | Error type carrying `StaleHeadExitCode` (10) |
| `forge.ReviewComment` | `internal/forge/forge.go:125` | Inline review comment (path, line, body); `Line==0` = file-level |
| `forge.PullRequestFileDiff` | `internal/forge/forge.go:134` | File path + unified diff patch |
| `forge.PullRequestReview` | `internal/forge/forge.go:107` | Review metadata (ID, NodeID, User, State, Body) |

---

## 3. Test Scenarios

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
| TC-031 | MinimizeComment API error | Soft-fail; no panic, review still submitted | Medium |
| TC-032 | GetAuthenticatedUser error | Skips cleanup; review still submitted | Medium |
| TC-033 | ListPullRequestReviews error | Skips cleanup; review still submitted | Medium |

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
| TC-074 | Vendor root discovery | Correct path resolved | Medium |
| TC-075 | Download with checksum verification | Hash matches expected SHA256 | Medium |
| TC-076 | Cross-compilation support | Correct platform binary selected | Low |

### 3.12 CLI — Vendor, Mint, Admin, Run

| ID | Scenario | Expected Result | Priority |
|:---|:---------|:----------------|:---------|
| TC-077 | Vendor command basic flow | Successfully vendors dependencies | Medium |
| TC-078 | Mint setup command | Creates mint configuration | Medium |
| TC-079 | Admin command changes | Backward-compatible behavior | Medium |
| TC-080 | Run command with new flags | Correctly processes arguments | Medium |
| TC-081 | Discover slugs command | Returns expected slug list | Medium |

### 3.13 Harness Enhancements

| ID | Scenario | Expected Result | Priority |
|:---|:---------|:----------------|:---------|
| TC-082 | Remote discovery | Discovers remote harness configurations | Medium |
| TC-083 | Harness linting | Detects invalid harness YAML | Medium |
| TC-084 | Scaffold integration | End-to-end scaffold produces valid harness | Medium |

### 3.14 GCF Provisioner

| ID | Scenario | Expected Result | Priority |
|:---|:---------|:----------------|:---------|
| TC-085 | Provisioner with refactored interface | Correct function deployment | Medium |
| TC-086 | FakeClient for testing | Implements full interface for test isolation | Low |

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

| Tier | Count | Description |
|:-----|:------|:------------|
| Unit Tests | 72 | Function-level tests with fake forge client |
| Integration Tests | 8 | Multi-component tests (harness scaffold, admin E2E) |
| E2E Tests | 6 | End-to-end admin/CLI tests |
| **Total** | **86** | |

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

## 6. Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
|:-----|:-----------|:-------|:-----------|
| Large PR scope masks subtle regressions | Medium | High | Focus testing on LSP-traced call chains; prioritize review pipeline tests |
| GitHub API rate limiting during inline comment posting | Low | Medium | Graceful fallback when `ListPullRequestFileDiffs` fails |
| Stale-head race condition (HEAD changes between check and review submit) | Low | High | `commitSHA` parameter pins review to checked commit |
| Forge interface breakage (missing method implementations) | Low | High | Compile-time interface check (`var _ forge.Client = (*LiveClient)(nil)`) |
| Exit code 10 not propagated through shell scripts | Low | Medium | Verify post-review.sh handles `StaleHeadExitCode` |

---

## 7. Recommendations

1. **Priority Testing**: Focus on TC-008 through TC-013 (stale-head detection) and TC-034 through TC-041 (inline comment mapping) — these are the highest-risk scenarios unique to the two-pass review strategy.
2. **Integration Validation**: Run the full E2E admin test suite (`e2e/admin/`) to validate backward compatibility of CLI changes.
3. **Forge Interface**: Verify that `forge.FakeClient` implements all new methods (`ListPullRequestFileDiffs`, `DismissPullRequestReview`) — existing compile-time checks should catch this.
4. **Manual Verification**: Test the post-review flow end-to-end on a real PR to validate inline comments render correctly on GitHub's UI, especially file-level fallback comments.

---

*Generated by QualityFlow STP Builder — 2026-06-22*
