# Software Test Plan — GH-2247

## Document Information

| Field | Value |
|:------|:------|
| **Ticket** | [GH-2247](https://github.com/fullsend-ai/fullsend/issues/2247) |
| **Title** | reconcile-repos.sh produces shim blob without sentinel, creating bogus update PR |
| **Type** | Bug Fix |
| **Priority** | High |
| **Component** | dispatch (reconcile-repos.sh) |
| **Author** | QualityFlow Agent |
| **Date** | 2026-06-21 |
| **Status** | Draft |

---

## 1. Summary

### 1.1 Problem Statement

The `reconcile-repos.sh` script's shim drift detection compared re-encoded base64 strings instead of decoded text content. Because Bash command substitution strips trailing newlines, the re-encoded base64 could differ from the original even when the decoded text was identical. This caused false-positive "stale shim" detection, leading to bogus update PRs (e.g., PR #2101) that removed the `---` and sentinel (`# --- fullsend managed below - do not edit ---`) lines from enrolled repositories' shim workflows. If merged, such PRs would trigger infinite reconciliation churn.

### 1.2 Fix Description

The fix replaces the base64-to-base64 comparison with decoded text comparison:

1. **Decode both sides** — the expected template output and the remote file content from GitHub
2. **Strip carriage returns** (`tr -d '\r'`) to normalize line endings
3. **Extract managed content** via `extract_managed_content` (everything from sentinel onward)
4. **Compare resulting strings directly** instead of re-encoding to base64
5. **Fallback for pre-sentinel shims** — when no sentinel is found, compare full decoded content

### 1.3 Files Changed

| File | Change |
|:-----|:-------|
| `internal/scaffold/fullsend-repo/scripts/reconcile-repos.sh` | Replace `managed_content_b64()` comparison with decoded text comparison (lines 400-421) |
| `internal/scaffold/fullsend-repo/scripts/reconcile-repos-test.sh` | Add regression test (Test 5) for trailing newline false-positive |

### 1.4 Linked Issues

| Reference | Description |
|:----------|:------------|
| PR #2101 | The bogus PR that triggered this investigation — proposed removing sentinel lines from fullsend repo's own shim workflow |

---

## 2. Scope

### 2.1 In Scope

- Shim drift detection logic in `reconcile-repos.sh`
- Content comparison between expected template and remote deployed shim
- Sentinel line preservation and extraction
- User header preservation (license comments above sentinel)
- Content injection guard (non-comment YAML rejection)
- Pre-sentinel shim migration path
- Base64 encoding/decoding round-trip behavior

### 2.2 Out of Scope

- GitHub API behavior (mocked in tests)
- Branch creation / PR creation mechanics (tested separately)
- `yq` / `jq` CLI behavior (external tools)
- Shim template content correctness (separate concern)
- Per-repo guard variable logic
- Unenrollment flow

---

## 3. Requirements-to-Tests Mapping

### 3.1 Requirement: Shim drift detection uses decoded text comparison

| Field | Value |
|:------|:------|
| **Requirement ID** | GH-2247 |
| **Summary** | Shim drift detection compares decoded text instead of re-encoded base64 to avoid false-positive stale detection |
| **Source** | Bug report + regression analysis |
| **Evidence** | Lines 400-421 of reconcile-repos.sh — comparison changed from `managed_content_b64()` output to decoded `extract_managed_content` output |

| # | Test Scenario | Type | Tier | Priority |
|:--|:-------------|:-----|:-----|:---------|
| 1 | Verify identical content with different trailing newlines is not flagged stale | Positive | Unit Tests | P0 |
| 2 | Verify stale managed content is correctly detected | Positive | Unit Tests | P0 |
| 3 | Verify up-to-date shim with user header is not flagged stale | Positive | Unit Tests | P0 |
| 4 | Verify comparison handles carriage return normalization | Positive | Unit Tests | P1 |
| 5 | Verify empty remote content triggers enrollment (not update) | Edge Case | Unit Tests | P1 |

### 3.2 Requirement: Sentinel line preservation

| Field | Value |
|:------|:------|
| **Requirement ID** | GH-2247 |
| **Summary** | The sentinel line (`# --- fullsend managed below - do not edit ---`) must never be stripped from the shim blob written to enrolled repos |
| **Source** | Bug report — PR #2101 removed sentinel lines |
| **Evidence** | `shim_with_header_b64()` returns template which starts with sentinel; `extract_managed_content` uses sentinel as boundary |

| # | Test Scenario | Type | Tier | Priority |
|:--|:-------------|:-----|:-----|:---------|
| 6 | Verify sentinel present in blob for stale shim update | Positive | Unit Tests | P0 |
| 7 | Verify sentinel present in blob for pre-sentinel migration | Positive | Unit Tests | P0 |
| 8 | Verify sentinel present in blob for new enrollment | Positive | Unit Tests | P1 |

### 3.3 Requirement: User header preservation across shim updates

| Field | Value |
|:------|:------|
| **Requirement ID** | GH-2247 |
| **Summary** | User-owned content above the sentinel (e.g., license headers, SPDX identifiers) is preserved when the managed portion is updated |
| **Source** | Code analysis — `shim_with_header_b64()` extracts and re-prepends user header |
| **Evidence** | Lines 107-143 of reconcile-repos.sh; Test 1 in test script validates Copyright + SPDX preservation |

| # | Test Scenario | Type | Tier | Priority |
|:--|:-------------|:-----|:-----|:---------|
| 9 | Verify license header preserved in updated shim blob | Positive | Unit Tests | P0 |
| 10 | Verify multi-line comment header preserved | Positive | Unit Tests | P1 |
| 11 | Verify blank-only header is discarded | Edge Case | Unit Tests | P2 |

### 3.4 Requirement: Content injection guard rejects non-comment YAML

| Field | Value |
|:------|:------|
| **Requirement ID** | GH-2247 |
| **Summary** | Non-comment YAML content above the sentinel is rejected to prevent workflow injection |
| **Source** | Security analysis — injection guard at line 132 |
| **Evidence** | `shim_with_header_b64()` rejects headers containing lines that don't match `^[[:space:]]*(#|$)` |

| # | Test Scenario | Type | Tier | Priority |
|:--|:-------------|:-----|:-----|:---------|
| 12 | Verify non-comment YAML above sentinel is rejected | Negative | Unit Tests | P0 |
| 13 | Verify warning emitted when injection guard rejects content | Positive | Unit Tests | P1 |
| 14 | Verify injected workflow keys not present in output blob | Negative | Unit Tests | P0 |
| 15 | Verify `---` YAML document separator alone is treated as non-comment | Edge Case | Unit Tests | P1 |

### 3.5 Requirement: Pre-sentinel shim migration produces clean template

| Field | Value |
|:------|:------|
| **Requirement ID** | GH-2247 |
| **Summary** | Repos with pre-sentinel shims (no sentinel line) are correctly detected as stale and migrated to the sentinel-based template without content duplication |
| **Source** | Code analysis — `shim_with_header_b64()` returns raw template when no sentinel found in remote |
| **Evidence** | Lines 118-125 of reconcile-repos.sh; Test 3 in test script validates no duplication |

| # | Test Scenario | Type | Tier | Priority |
|:--|:-------------|:-----|:-----|:---------|
| 16 | Verify pre-sentinel shim flagged as stale | Positive | Unit Tests | P0 |
| 17 | Verify pre-sentinel migration does not duplicate old content | Negative | Unit Tests | P0 |
| 18 | Verify migrated blob contains sentinel and fresh template | Positive | Unit Tests | P0 |

### 3.6 Requirement: Reconciliation does not produce bogus update PRs

| Field | Value |
|:------|:------|
| **Requirement ID** | GH-2247 |
| **Summary** | End-to-end reconciliation run does not open update PRs for repos whose shim content is logically up-to-date |
| **Source** | Bug report — PR #2101 was a bogus update PR |
| **Evidence** | Full reconciliation flow: `shim_content_b64()` → fetch remote → compare → skip or update |

| # | Test Scenario | Type | Tier | Priority |
|:--|:-------------|:-----|:-----|:---------|
| 19 | Verify full reconciliation skips up-to-date repos | Positive | Functional | P0 |
| 20 | Verify full reconciliation updates genuinely stale repos | Positive | Functional | P0 |
| 21 | Verify no blob created when shim is current | Negative | Functional | P1 |
| 22 | Verify reconciliation handles mixed repo states correctly | Positive | Functional | P1 |

---

## 4. Test Classification Summary

| Tier | Count | Description |
|:-----|:------|:------------|
| Unit Tests | 18 | Isolated function tests with mocked `gh`/`yq`/`base64` — validates `shim_with_header_b64()`, `extract_managed_content`, `managed_content_b64()`, drift comparison, injection guard |
| Functional | 4 | Full reconciliation script execution with mock infrastructure — validates end-to-end script behavior across multiple repo states |
| **Total** | **22** | |

### Tier Rationale

- **Unit Tests (18):** The fix is scoped to bash functions within a single script. All tests can run without a cluster or real GitHub API by mocking `gh`, `yq`, and `base64` commands. The existing test harness (`reconcile-repos-test.sh`) already demonstrates this pattern with mock binaries in `$PATH`.

- **Functional (4):** These test the full reconciliation script execution with multiple repos in different states (up-to-date, stale, pre-sentinel, new). They exercise the interaction between all functions but still use mocked GitHub API calls. No real cluster or GitHub interaction is required.

- **No End-to-End tests:** The fix is entirely within a bash script's string comparison logic. End-to-end testing against real GitHub repos would test GitHub API behavior, not the fix. The mocked test harness provides equivalent coverage.

---

## 5. Existing Test Coverage

### 5.1 Tests in `reconcile-repos-test.sh`

The fix commit includes a comprehensive test file with 5 test cases:

| Test | Maps to Scenarios | Status |
|:-----|:-----------------|:-------|
| Test 1: Stale shim branch update is atomic | #6, #9, #19, #20 | Implemented |
| Test 2: Up-to-date shim with user header not flagged stale | #3, #21 | Implemented |
| Test 3: Pre-sentinel shim migration without duplication | #16, #17, #18 | Implemented |
| Test 4: Non-comment YAML above sentinel rejected | #12, #13, #14 | Implemented |
| Test 5: Identical content with different trailing newlines not flagged stale | #1, #4 | Implemented (regression test for GH-2247) |

### 5.2 Coverage Gaps

| Scenario | Gap | Recommendation |
|:---------|:----|:---------------|
| #2: Stale managed content correctly detected | Partially covered by Test 1 (uses stale content) but not isolated | Add explicit assertion for stale detection path |
| #5: Empty remote triggers enrollment | Not tested — Test 1 mock returns content for test-repo | Add mock repo returning empty content |
| #8: Sentinel in new enrollment blob | Not explicitly verified — Test 1 covers update path | Add sentinel assertion for new enrollment blob |
| #10: Multi-line comment header | Not tested — only 2-line header tested | Add test with 5+ line header |
| #11: Blank-only header discarded | Not tested | Add test with whitespace-only lines above sentinel |
| #15: `---` as non-comment | Implicitly tested (template starts with `---`) but not isolated | Add explicit test with only `---` above sentinel |
| #22: Mixed repo states | Partially covered by Test 1 (multiple repos) | Already has good coverage |

---

## 6. Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|:-----|:-----------|:-------|:-----------|
| False-positive drift detection recurrence | Low (after fix) | High — infinite PR churn | Test 5 is explicit regression test |
| Sentinel stripping in edge cases | Low | High — breaks reconciliation boundary | Tests 1-4 cover major paths |
| User header data loss | Low | Medium — cosmetic but annoying | Test 1 validates header preservation |
| Content injection via crafted header | Low | Critical — workflow tampering | Test 4 validates injection guard |
| Base64 encoding differences across platforms | Medium | Medium — could cause false drift on different OS | Fix eliminates base64 comparison entirely |

---

## 7. Test Environment

| Component | Requirement |
|:----------|:------------|
| **Shell** | Bash 4.4+ (set -euo pipefail) |
| **Dependencies** | `jq`, `base64`, mock `gh` and `yq` binaries |
| **Execution** | `bash internal/scaffold/fullsend-repo/scripts/reconcile-repos-test.sh` |
| **CI Integration** | Runs in GitHub Actions on ubuntu-latest runners |
| **No cluster required** | All GitHub API calls are mocked |

---

## 8. Acceptance Criteria Verification

| Acceptance Criteria | Test Coverage |
|:--------------------|:-------------|
| `reconcile-repos.sh` never produces a PR that removes the sentinel | Scenarios #6, #7, #8 — sentinel verified in all output blobs |
| PR #2101 scenario (logically identical content) does not trigger update | Scenario #1 (Test 5) — explicit regression test |
| User-owned headers above sentinel are preserved | Scenarios #9, #10 — header content verified in output blob |
| Non-comment YAML injection is blocked | Scenarios #12, #13, #14 — injection guard validated |
| Pre-sentinel repos are correctly migrated | Scenarios #16, #17, #18 — migration produces clean template |
