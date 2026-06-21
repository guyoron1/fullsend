# FullSend Test Plan

| Field             | Value                                                                              |
|:------------------|:-----------------------------------------------------------------------------------|
| **Ticket**        | GH-53                                                                              |
| **Title**         | fix(#1230): run OutputPipeline on post-review before posting to forge              |
| **Type**          | Bug Fix                                                                            |
| **Priority**      | Normal                                                                             |
| **Product**       | FullSend                                                                           |
| **Platform**      | GitHub Actions                                                                     |
| **Version**       | 0.x                                                                                |
| **Date**          | 2026-06-21                                                                         |
| **Components**    | CLI Commands, Security Scanning, Code Generation (forge)                           |

---

## 1. Summary

The `post-review` command posted review content (body, finding descriptions,
finding remediations) directly to the GitHub API without running it through the
security `OutputPipeline`. This fix calls `security.OutputPipeline().Scan()` on
all user-visible text fields in the `ReviewResult` before any forge API call,
preventing leaked secrets and zero-width-obfuscated tokens from reaching GitHub.

A secondary change converts out-of-diff-hunk inline review findings from being
silently dropped to being posted as **file-level comments** (Line=0), preserving
visibility on the PR.

---

## 2. Requirements Mapping

| Req ID     | Requirement                                                                                       | Source         |
|:-----------|:--------------------------------------------------------------------------------------------------|:---------------|
| REQ-001    | Review body must be sanitized through OutputPipeline before posting to forge                       | GH-53 body     |
| REQ-002    | Finding description fields must be sanitized before posting as inline PR comments                  | GH-53 body     |
| REQ-003    | Finding remediation fields must be sanitized before posting                                        | GH-53 body     |
| REQ-004    | Zero-width Unicode obfuscation must be normalized before secret pattern matching                   | GH-53 body     |
| REQ-005    | Findings outside diff hunks should fall back to file-level comments instead of being dropped       | PR-53 diff     |
| REQ-006    | File-level fallback comments must include the original line number in the body                     | PR-53 diff     |

---

## 3. Test Scenarios

### 3.1 Security Sanitization — `sanitizeReviewResult`

| Test ID       | Scenario                                                              | Tier   | Requirement |
|:--------------|:----------------------------------------------------------------------|:-------|:------------|
| TS-GH-53-001  | Review body containing a GitHub PAT (`ghp_*`) is redacted             | Tier 1 | REQ-001     |
| TS-GH-53-002  | Review body without secrets passes through unchanged                  | Tier 1 | REQ-001     |
| TS-GH-53-003  | Empty review body remains empty after sanitization                    | Tier 1 | REQ-001     |
| TS-GH-53-004  | Finding `Description` containing a secret is redacted                 | Tier 1 | REQ-002     |
| TS-GH-53-005  | Finding `Remediation` containing a secret is redacted                 | Tier 1 | REQ-003     |
| TS-GH-53-006  | Secret obfuscated with zero-width Unicode chars is detected & redacted| Tier 1 | REQ-004     |
| TS-GH-53-007  | Multiple findings with mixed clean/secret text are individually sanitized | Tier 1 | REQ-002, REQ-003 |

### 3.2 File-Level Fallback — `findingsToReviewComments`

| Test ID       | Scenario                                                              | Tier   | Requirement |
|:--------------|:----------------------------------------------------------------------|:-------|:------------|
| TS-GH-53-008  | Finding in diff hunk is posted as normal inline comment with line number | Tier 1 | REQ-005   |
| TS-GH-53-009  | Finding outside diff hunk falls back to file-level comment (Line=0)   | Tier 1 | REQ-005     |
| TS-GH-53-010  | File-level fallback body contains `Line N` with original line number  | Tier 1 | REQ-006     |
| TS-GH-53-011  | Finding on file not in PR diff is omitted (fileFiltered counter incremented) | Tier 1 | REQ-005 |
| TS-GH-53-012  | All severity levels (info, low, medium, high, critical) pass through to inline comments when in hunk | Tier 1 | REQ-005 |
| TS-GH-53-013  | All severity levels fall back to file-level when outside hunk         | Tier 1 | REQ-005     |
| TS-GH-53-014  | Binary files (empty hunk list) skip line-level filtering              | Tier 1 | REQ-005     |

### 3.3 Integration — `submitFormalReview`

| Test ID       | Scenario                                                              | Tier   | Requirement |
|:--------------|:----------------------------------------------------------------------|:-------|:------------|
| TS-GH-53-015  | End-to-end: review with file-filtered, hunk-filtered, and in-hunk findings produces correct comment set | Tier 1 | REQ-005, REQ-006 |
| TS-GH-53-016  | Log output shows file-level fallback info message (not warning)       | Tier 1 | REQ-005     |
| TS-GH-53-017  | `sanitizeReviewResult` is called before `submitFormalReview` in the command flow | Tier 1 | REQ-001 |

### 3.4 Regression — OutputPipeline Consumers

| Test ID       | Scenario                                                              | Tier   | Requirement |
|:--------------|:----------------------------------------------------------------------|:-------|:------------|
| TS-GH-53-018  | `OutputPipeline` in `run.go` continues to function after security package changes | Tier 2 | REQ-001 |
| TS-GH-53-019  | `OutputPipeline` in `scan.go` continues to function after security package changes | Tier 2 | REQ-001 |

---

## 4. Regression Impact Analysis

### 4.1 Call Graph (LSP-traced)

```
newPostReviewCmd (postreview.go:33)
  └─ parseReviewResult (postreview.go:578)
  └─ sanitizeReviewResult (postreview.go:541)  ← NEW
  │    └─ security.OutputPipeline (scanner.go:97)
  │         └─ NewUnicodeNormalizer (unicode.go)
  │         └─ NewSecretRedactor (redactor.go)
  │    └─ Pipeline.Scan (scanner.go:58)
  └─ submitFormalReview (postreview.go:298)
       └─ findingsToReviewComments (postreview.go:393)  ← MODIFIED
       └─ forge.Client.CreatePullRequestReview (forge.go:308)
```

### 4.2 Other OutputPipeline Consumers (cross-file references)

| File                        | Line | Usage                              | Risk |
|:----------------------------|:-----|:-----------------------------------|:-----|
| `internal/cli/run.go`       | 1698 | Output scanning in `run` command   | Low  |
| `internal/cli/scan.go`      | 190  | Output scanning in `scan` command  | Low  |
| `internal/security/scanner_test.go` | 294 | Unit test for OutputPipeline | None |

The `OutputPipeline()` function is a stateless factory — it constructs a new
pipeline each call. Adding a new call site in `postreview.go` does not alter
behavior for existing consumers.

### 4.3 `findingsToReviewComments` Callers

The function is called only from `submitFormalReview` (postreview.go:340).
The behavior change (file-level fallback instead of drop) affects all review
submissions that have out-of-hunk findings. Existing tests have been updated
to reflect the new expected counts.

### 4.4 Forge Interface

`forge.ReviewComment` struct has fields `Path`, `Line`, `Body`. Setting
`Line=0` for file-level comments is consistent with the GitHub API behavior
for file-level PR review comments.

---

## 5. Components Affected

| Component          | Package Path            | Impact                                         |
|:-------------------|:------------------------|:-----------------------------------------------|
| CLI Commands       | `internal/cli/`         | New `sanitizeReviewResult` function; modified `findingsToReviewComments` |
| Security Scanning  | `internal/security/`    | `OutputPipeline()` now called from post-review path (no code changes in package) |
| Forge (GitHub)     | `internal/forge/`       | `ReviewComment.Line=0` used for file-level comments (existing API contract) |

---

## 6. Test Environment

| Requirement      | Value               |
|:-----------------|:--------------------|
| Platform         | GitHub Actions       |
| Go Version       | 1.23+                |
| CLI Tools        | fullsend, gh, go     |
| Test Framework   | Go standard testing + testify |

---

## 7. Existing Test Coverage

The PR includes comprehensive unit tests for all new and modified functions:

| Test Function                                              | Covers         |
|:-----------------------------------------------------------|:---------------|
| `TestSanitizeReviewResult_RedactsSecretsInBody`            | TS-GH-53-001   |
| `TestSanitizeReviewResult_NoSecretsPassesThrough`          | TS-GH-53-002   |
| `TestSanitizeReviewResult_EmptyBody`                       | TS-GH-53-003   |
| `TestSanitizeReviewResult_RedactsSecretsInFindings`        | TS-GH-53-004, 005 |
| `TestSanitizeReviewResult_ZeroWidthObfuscatedSecret`       | TS-GH-53-006   |
| `TestFindingsToReviewComments_FiltersByDiffHunks`          | TS-GH-53-008, 009, 010, 011 |
| `TestFindingsToReviewComments_AllSeveritiesPassThrough`    | TS-GH-53-012   |
| `TestFindingsToReviewComments_AllSeveritiesFallbackToFileLevel` | TS-GH-53-013 |
| `TestFindingsToReviewComments_EmptyPatchSkipsLineFiltering`| TS-GH-53-014   |
| `TestSubmitFormalReview_FiltersByPRFileDiffs`              | TS-GH-53-015, 016 |

---

## 8. Risks and Mitigations

| Risk                                                          | Likelihood | Impact | Mitigation                                    |
|:--------------------------------------------------------------|:-----------|:-------|:----------------------------------------------|
| False positive secret detection redacts legitimate review text | Low        | Medium | OutputPipeline uses well-scoped prefix patterns (ghp_, ghs_, etc.) |
| File-level comments (Line=0) rejected by GitHub API           | Low        | Medium | GitHub API supports file-level review comments; validated by existing forge tests |
| Performance impact of scanning large review bodies             | Low        | Low    | Pipeline is lightweight regex-based; runs once per review |
| Zero-width normalization alters non-ASCII review content       | Low        | Low    | UnicodeNormalizer only strips invisible/zero-width characters |
