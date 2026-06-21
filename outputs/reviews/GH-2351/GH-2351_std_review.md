# STD Review Report: GH-2351

**Reviewed:**
- STD YAML: `outputs/std/GH-2351/GH-2351_test_description.yaml`
- STP Source: `outputs/stp/GH-2351/GH-2351_test_plan.md`
- Go Stubs: `outputs/std/GH-2351/go-tests/` (3 files, 18 test functions)
- Python Stubs: N/A (not generated — no End-to-End scenarios)

**Date:** 2026-06-21
**Reviewer:** QualityFlow Automated Review (v1.1.0)
**Review Rules Schema:** N/A (no project-specific review_rules.yaml available)

---

## Verdict: APPROVED_WITH_FINDINGS

## Summary

| Metric | Value |
|:-------|:------|
| Dimensions reviewed | 7/7 |
| Critical findings | 0 |
| Major findings | 3 |
| Minor findings | 5 |
| Actionable findings | 7 |
| Weighted score | 85 |
| Confidence | MEDIUM |

## Traceability Summary

| Metric | Value |
|:-------|:------|
| STP scenarios | 18 |
| STD scenarios | 18 |
| Forward coverage (STP→STD) | 18/18 (100%) |
| Reverse coverage (STD→STP) | 18/18 (100%) |
| Orphan STD scenarios | 0 |
| Missing STD scenarios | 0 |

---

## Findings by Dimension

### Dimension 1: STP-STD Traceability — Score: 95/100

#### 1a. Forward Traceability (STP → STD) ✅

All 18 STP scenarios from Section III are covered by corresponding STD scenarios:

| STP Requirement Group | STP Scenarios | STD Scenarios | Status |
|:----------------------|:-------------|:-------------|:-------|
| ListRepositoryFiles returns all paths (P0) | 3 | TS-001, TS-002, TS-003 | ✅ TRACED |
| ComparePathPresence identifies missing paths (P0) | 5 | TS-004 – TS-008 | ✅ TRACED |
| Batch API pattern guards (P0) | 2 | TS-009, TS-010 | ✅ TRACED |
| FakeClient implements ListRepositoryFiles (P1) | 3 | TS-011, TS-012, TS-013 | ✅ TRACED |
| FakeClient thread safety (P2) | 1 | TS-014 | ✅ TRACED |
| LiveClient API pipeline (P1) | 4 | TS-015, TS-016, TS-017, TS-018 | ✅ TRACED |

#### 1b. Reverse Traceability (STD → STP) ✅

All 18 STD scenarios have `requirement_id: "GH-2351"` which matches the STP's tracked issue. Every scenario's `test_objective.title` has strong keyword overlap (≥0.70) with a corresponding STP scenario description.

#### 1c. Count Consistency ✅

| Metadata Field | Declared | Actual | Status |
|:---------------|:---------|:-------|:-------|
| `total_scenarios` | 18 | 18 | ✅ MATCH |
| `functional_count` | 4 | 4 | ✅ MATCH |
| `unit_test_count` | 14 | 14 | ✅ MATCH |
| `p0_count` | 10 | 10 | ✅ MATCH |
| `p1_count` | 7 | 7 | ✅ MATCH |
| `p2_count` | 1 | 1 | ✅ MATCH |

#### 1d. STP Reference ✅

`stp_reference.file` correctly points to `outputs/stp/GH-2351/GH-2351_test_plan.md` which exists and was verified.

#### 1e. Priority-Testability Consistency ✅

All 10 P0 scenarios are fully testable using `FakeClient` with no infrastructure or external dependencies. No P0 scenario is deferred or marked untestable.

#### Finding

- **D1-1b-001**
  - **Severity:** MINOR
  - **Dimension:** STP-STD Traceability
  - **Description:** STP Section III has several requirement groups with blank `Requirement ID` fields (only the first group explicitly lists "GH-2351"). The STD correctly assigns `requirement_id: "GH-2351"` to all scenarios since they all trace to the same Jira ticket, but the STP's blank fields create ambiguity in bidirectional tracing.
  - **Evidence:** STP Section III rows 2–6 have empty `Requirement ID` fields but describe distinct requirement groups.
  - **Remediation:** Populate each STP requirement group with a distinct sub-requirement identifier (e.g., "GH-2351-R1", "GH-2351-R2") or repeat "GH-2351" explicitly.
  - **Actionable:** false (STP issue, not STD)

---

### Dimension 2: STD YAML Structure — Score: 78/100

#### 2a. Document-Level Structure

| Check | Status |
|:------|:-------|
| `document_metadata` present | ✅ |
| `std_version: "2.1-enhanced"` | ✅ |
| `code_generation_config` present | ✅ |
| `code_generation_config.std_version` | ✅ |
| `common_preconditions` present | ✅ |
| `scenarios` array non-empty | ✅ (18 scenarios) |

#### 2b. Per-Scenario Required Fields

| Field | Present | Notes |
|:------|:--------|:------|
| `scenario_id` | ✅ 18/18 | Sequential "1" through "18" |
| `test_id` | ✅ 18/18 | Format TS-GH-2351-{NNN} ✅ |
| `tier` | ✅ 18/18 | Non-standard values (see finding) |
| `priority` | ✅ 18/18 | P0/P1/P2 ✅ |
| `requirement_id` | ✅ 18/18 | All "GH-2351" |
| `patterns` | ❌ 0/18 | **Missing** — see finding |
| `variables` | ✅ 18/18 | closure_scope present |
| `test_structure` | ✅ 18/18 | type + function_name + pattern |
| `code_structure` | ✅ 18/18 | Valid Go function templates |
| `test_objective` | ✅ 18/18 | title + what + why + acceptance_criteria |
| `test_data` | ⚠️ 7/18 | 11 scenarios have `test_data: {}` |
| `test_steps` | ✅ 18/18 | setup + test_execution present |
| `assertions` | ✅ 18/18 | At least 1 per scenario |

#### 2c. v2.1-Specific Checks

This project uses Go `testing` + `testify` (not Ginkgo), so Ginkgo-specific checks (Ordered decorator, `ExpectWithOffset`, `:=` vs `=` for closure variables) do not apply. The `classification` field exists with `test_type`, `scope`, and `automation_approach` — serving a similar role to the `patterns` field but with different schema.

No Python/Tier 2 scenarios are present, so Tier 2 checks do not apply.

#### Findings

- **D2-2b-001**
  - **Severity:** MAJOR
  - **Dimension:** STD YAML Structure
  - **Description:** The `patterns` field is missing from all 18 scenarios. Per v2.1-enhanced specification, each scenario must declare a primary pattern and helpers. The STD uses a `classification` field with `test_type`, `scope`, and `automation_approach` as an alternative, but this does not match the required schema.
  - **Evidence:** No scenario contains `patterns:` (only 1 occurrence of `test_patterns:` in `code_generation_config`, which is a different field).
  - **Remediation:** Add a `patterns` block to each scenario with `primary` and `helpers_required` keys. For this project's Go testing framework, map: `test_type: "Unit"` → `primary: "unit-test"`, `test_type: "Functional"` → `primary: "functional-test"`. Set `helpers_required: []` since testify is declared at the config level.
  - **Actionable:** true

- **D2-2b-002**
  - **Severity:** MINOR
  - **Dimension:** STD YAML Structure
  - **Description:** Tier values use non-standard naming: `"Unit Tests"` and `"Functional"` instead of the v2.1-enhanced standard `"Tier 1"` / `"Tier 2"`. While descriptive and internally consistent, this deviates from the spec.
  - **Evidence:** 14 scenarios have `tier: "Unit Tests"`, 4 scenarios have `tier: "Functional"`.
  - **Remediation:** Map `"Unit Tests"` → `"Tier 1"` and `"Functional"` → `"Tier 2"`, or document the project's tier naming convention in `code_generation_config`.
  - **Actionable:** true

- **D2-2b-003**
  - **Severity:** MINOR
  - **Dimension:** STD YAML Structure
  - **Description:** 11 of 18 scenarios have empty `test_data: {}`. While acceptable for pure unit tests using FakeClient (where test data is inline in setup steps), the empty field adds noise.
  - **Evidence:** Scenarios 6–10, 12–14, 17, 18 have `test_data: {}`.
  - **Remediation:** Either populate `test_data.resource_definitions` with the FakeClient configuration described in each scenario's setup steps, or omit the field entirely (it is not required when inline setup is sufficient).
  - **Actionable:** true

---

### Dimension 3: Pattern Matching Correctness — Score: 50/100

#### Assessment

Pattern matching could not be fully evaluated because the `patterns` field is absent from all scenarios (see D2-2b-001). A baseline score of 50 is assigned.

However, the `classification` field provides equivalent test-type metadata:

| Scenario Range | `test_type` | `scope` | `automation_approach` | Consistent? |
|:---------------|:-----------|:--------|:---------------------|:------------|
| TS-001 – TS-014 | Unit | Single-component | Go test with FakeClient | ✅ |
| TS-015 – TS-018 | Functional | Single-component | Go test with HTTP mock | ✅ |

The `test_structure.pattern` field provides additional pattern metadata:

| Pattern | Scenarios | Appropriate? |
|:--------|:----------|:------------|
| `arrange-act-assert` | 1–8, 10–12 | ✅ |
| `error-injection-guard` | 9 | ✅ |
| `concurrent-goroutine` | 14 | ✅ |
| `http-mock-chain` | 15 | ✅ |
| `http-mock-filter` | 16 | ✅ |
| `http-mock-error` | 17 | ✅ |
| `http-mock-retry` | 18 | ✅ |

All pattern assignments in `test_structure.pattern` are semantically appropriate for their scenarios.

#### 3d. Pattern Library Validation — SKIPPED

Pattern library at `{config_dir}/patterns/tier1_patterns.yaml` was not available in this sandbox. Skipping library validation.

---

### Dimension 4: Test Step Quality — Score: 90/100

| Scenario | Setup | Execution | Cleanup | Assertions | Isolation | Error Paths | Status |
|:---------|:------|:----------|:--------|:-----------|:----------|:------------|:-------|
| TS-001 | 1 | 1 | 0 | 2 | ✅ | N/A | ✅ PASS |
| TS-002 | 1 | 1 | 0 | 2 | ✅ | ✅ negative | ✅ PASS |
| TS-003 | 1 | 1 | 0 | 1 | ✅ | ✅ negative | ✅ PASS |
| TS-004 | 1 | 1 | 0 | 2 | ✅ | N/A | ✅ PASS |
| TS-005 | 1 | 1 | 0 | 2 | ✅ | N/A | ✅ PASS |
| TS-006 | 1 | 1 | 0 | 2 | ✅ | N/A | ✅ PASS |
| TS-007 | 1 | 1 | 0 | 2 | ✅ | N/A | ✅ PASS |
| TS-008 | 1 | 1 | 0 | 1 | ✅ | ✅ negative | ✅ PASS |
| TS-009 | 1 | 1 | 0 | 1 | ✅ | ✅ guard | ✅ PASS |
| TS-010 | 1 | 1 | 0 | 2 | ✅ | N/A | ✅ PASS |
| TS-011 | 1 | 1 | 0 | 2 | ✅ | N/A | ✅ PASS |
| TS-012 | 1 | 1 | 0 | 2 | ✅ | N/A | ✅ PASS |
| TS-013 | 1 | 1 | 0 | 1 | ✅ | ✅ negative | ✅ PASS |
| TS-014 | 1 | 1 | 0 | 2 | ✅ | N/A | ✅ PASS |
| TS-015 | 1 | 1 | 1 | 2 | ✅ | N/A | ✅ PASS |
| TS-016 | 1 | 1 | 1 | 2 | ✅ | N/A | ✅ PASS |
| TS-017 | 1 | 1 | 1 | 1 | ✅ | ✅ negative | ✅ PASS |
| TS-018 | 1 | 1 | 1 | 2 | ✅ | N/A | ✅ PASS |

#### 4a. Step Completeness

- All 18 scenarios have setup and test_execution steps ✅
- 14 unit test scenarios have `cleanup: []` — **acceptable** because FakeClient-based tests create no external resources requiring cleanup
- 4 functional (LiveClient) scenarios have cleanup steps to close mock HTTP servers ✅

#### 4b. Step Quality ✅

All steps are specific and actionable with concrete commands and validations:
- Actions reference specific functions/methods (e.g., "Call ListRepositoryFiles with valid owner and repo")
- Commands include Go code references (e.g., `fakeClient.ListRepositoryFiles(ctx, "myorg", "myrepo")`)
- Validations describe measurable outcomes (e.g., "Returns []string with all matching paths, no error")
- Step IDs are sequential (SETUP-01, TEST-01, CLEANUP-01)

No uncertain verification language detected.

#### 4c. Logical Flow ✅

All scenarios follow a clean arrange-act-assert flow. No circular dependencies. Resources used in test_execution are created in setup.

#### 4e. Test Dependency Structure ✅

All 18 scenarios are fully independent — no scenario depends on another's output. Each test creates its own FakeClient/mock server. This is excellent test isolation.

#### 4f. Assertion Quality ✅

All assertions are specific with measurable conditions and assigned priorities. Good distribution: P0 assertions for critical behaviors, P1 for supplementary checks.

#### 4g. Test Isolation ✅

Excellent. Every scenario creates its own FakeClient with dedicated state. No shared mutable state across scenarios. No external dependencies for unit tests.

#### 4h. Error Path and Edge Case Coverage ✅

| Requirement Area | Positive | Negative/Error | Boundary | Guard | Coverage |
|:----------------|:---------|:--------------|:---------|:------|:---------|
| ListRepositoryFiles | 1 (TS-001) | 2 (TS-002, TS-003) | — | — | ✅ Good |
| ComparePathPresence | 2 (TS-004, TS-005) | 1 (TS-008) | 2 (TS-006, TS-007) | 2 (TS-009, TS-010) | ✅ Excellent |
| FakeClient | 2 (TS-011, TS-012) | 1 (TS-013) | — | — | ✅ Good |
| Thread Safety | — | — | — | 1 (TS-014) | ✅ Appropriate |
| LiveClient | 2 (TS-015, TS-016) | 1 (TS-017) | — | 1 (TS-018) | ✅ Good |

Strong negative testing coverage across all requirement areas. The guard test pattern (TS-009) is particularly well-designed for regression prevention.

#### Finding

- **D4-4a-001**
  - **Severity:** MINOR
  - **Dimension:** Test Step Quality
  - **Description:** 14 unit test scenarios have empty `cleanup: []` arrays. While justified for FakeClient-based tests (no external resources), having explicit "no cleanup needed" comments would improve clarity.
  - **Evidence:** Scenarios 1–14 all have `cleanup: []`.
  - **Remediation:** No action required — empty cleanup is correct for these unit tests. Optionally, add a comment in the cleanup section: `# No cleanup needed — FakeClient has no external state`.
  - **Actionable:** false

---

### Dimension 4.5: STD Content Policy — Score: 85/100

#### 4.5a. Banned Content

- **D4.5-4.5a-001**
  - **Severity:** MAJOR
  - **Dimension:** STD Content Policy
  - **Description:** `document_metadata.related_prs` contains a PR/issue reference with URL. The STD is a design document describing *what* to test, not *what code changed*. PR URLs are implementation artifacts that belong in the STP (which references them in Section I), not in the STD.
  - **Evidence:**
    ```yaml
    related_prs:
      - repo: "fullsend-ai/fullsend"
        pr_number: 2351
        url: "https://github.com/fullsend-ai/fullsend/issues/2351"
        title: "Batch path-existence checks via Git Trees API"
        merged: false
    ```
  - **Remediation:** Remove the `related_prs` section from `document_metadata`. The STP already contains the issue reference in Section I.
  - **Actionable:** true

#### 4.5b. No Implementation Details in Stubs ✅

All stub files contain only:
- PSE docstrings (design content)
- `t.Skip("Phase 1: Design only - awaiting implementation")` bodies (appropriate pending marker)
- Standard library imports (`testing`)

No fixture implementations, no helper function code, no concrete API calls. Stubs are clean design artifacts.

#### 4.5c. Test Environment Separation ✅

No infrastructure setup, cluster configuration, or feature gate enablement found in stubs or STD YAML. Test environment requirements are properly documented in `common_preconditions`.

---

### Dimension 5: PSE Docstring Quality — Score: 92/100

**Go Stubs:** 3 files reviewed, 18 test functions total.

#### 5a. PSE Quality Assessment

**File: `list_repository_files_stubs_test.go`** (7 test functions)

| Test Function | Test ID | Preconditions | Steps | Expected | [NEGATIVE] | Quality |
|:-------------|:--------|:-------------|:------|:---------|:----------|:--------|
| `TestListRepositoryFiles_ReturnsAllBlobPaths` | TS-001 ✅ | Specific ✅ | Numbered ✅ | Measurable ✅ | N/A | ✅ Good |
| `TestListRepositoryFiles_ErrorOnTruncatedTree` | TS-002 ✅ | Specific ✅ | Numbered ✅ | Measurable ✅ | ✅ Present | ✅ Good |
| `TestListRepositoryFiles_ErrNotFoundForNonexistentRepo` | TS-003 ✅ | Specific ✅ | Numbered ✅ | Measurable ✅ | ✅ Present | ✅ Good |
| `TestFakeClient_ListRepositoryFiles_PrefixFiltering` | TS-011 ✅ | Specific ✅ | Numbered ✅ | Measurable ✅ | N/A | ✅ Good |
| `TestFakeClient_ListRepositoryFiles_NoMatch` | TS-012 ✅ | Specific ✅ | Numbered ✅ | Measurable ✅ | N/A | ✅ Good |
| `TestFakeClient_ListRepositoryFiles_InjectedError` | TS-013 ✅ | Specific ✅ | Numbered ✅ | Measurable ✅ | ✅ Present | ✅ Good |
| `TestFakeClient_ListRepositoryFiles_ThreadSafe` | TS-014 ✅ | Specific ✅ | Numbered ✅ | Measurable ✅ | N/A | ✅ Good |

**File: `compare_path_presence_stubs_test.go`** (7 test functions)

| Test Function | Test ID | Preconditions | Steps | Expected | [NEGATIVE] | Quality |
|:-------------|:--------|:-------------|:------|:---------|:----------|:--------|
| `TestComparePathPresence_AllPresent` | TS-004 ✅ | Specific ✅ | Numbered ✅ | Measurable ✅ | N/A | ✅ Good |
| `TestComparePathPresence_SomeMissing` | TS-005 ✅ | Specific ✅ | Numbered ✅ | Measurable ✅ | N/A | ✅ Good |
| `TestComparePathPresence_AllMissingEmptyRepo` | TS-006 ✅ | Specific ✅ | Numbered ✅ | Measurable ✅ | N/A | ✅ Good |
| `TestComparePathPresence_EmptyInputReturnsNil` | TS-007 ✅ | Specific ✅ | Numbered ✅ | Measurable ✅ | N/A | ✅ Good |
| `TestComparePathPresence_ErrorPropagation` | TS-008 ✅ | Specific ✅ | Numbered ✅ | Measurable ✅ | ✅ Present | ✅ Good |
| `TestComparePathPresence_UsesOneAPICall` | TS-009 ✅ | Specific ✅ | Numbered ✅ | Measurable ✅ | N/A | ✅ Good |
| `TestComparePathPresence_SingleCallForManyPaths` | TS-010 ✅ | Specific ✅ | Numbered ✅ | Measurable ✅ | N/A | ✅ Good |

**File: `live_client_stubs_test.go`** (4 test functions)

| Test Function | Test ID | Preconditions | Steps | Expected | [NEGATIVE] | Quality |
|:-------------|:--------|:-------------|:------|:---------|:----------|:--------|
| `TestLiveClient_ListRepositoryFiles_APIPipeline` | TS-015 ✅ | Specific ✅ | Numbered ✅ | Measurable ✅ | N/A | ✅ Good |
| `TestLiveClient_ListRepositoryFiles_BlobsOnly` | TS-016 ✅ | Specific ✅ | Numbered ✅ | Measurable ✅ | N/A | ✅ Good |
| `TestLiveClient_ListRepositoryFiles_RefLookupError` | TS-017 ✅ | Specific ✅ | Numbered ✅ | Measurable ✅ | ✅ Present | ✅ Good |
| `TestLiveClient_ListRepositoryFiles_RetriesTransientErrors` | TS-018 ✅ | Specific ✅ | Numbered ✅ | Measurable ✅ | N/A | ✅ Good |

#### 5c. PSE Section Classification ✅

No misclassifications detected:
- Preconditions describe setup state only (no "Verify..." steps)
- Steps describe actions only (no verification steps)
- Expected sections describe observable outcomes with verification methods

#### Module-Level Documentation ✅

- All stub files reference the STP file in module-level comments
- No PR URLs in stub file comments
- Jira ticket reference (GH-2351) appropriately included

#### Finding

- **D5-5a-001**
  - **Severity:** MINOR
  - **Dimension:** PSE Docstring Quality
  - **Description:** File-level markers in `list_repository_files_stubs_test.go` declare only `Markers: - unit` but the file contains both unit test scenarios (TS-001–003, TS-011–014) spanning P0, P1, and P2 priorities. While the marker correctly identifies the test type, adding priority-level markers would improve filtering.
  - **Evidence:** File-level comment: `Markers: - unit`. Contains P0, P1, and P2 scenarios.
  - **Remediation:** No action required — marker indicates test type, not priority. Priority is documented per-test in the test_id docstring.
  - **Actionable:** false

---

### Dimension 6: Code Generation Readiness — Score: 88/100

#### 6a. Variable Declarations ✅

All `variables.closure_scope` entries across 18 scenarios use valid Go types:
- `*forge.FakeClient`, `[]string`, `error`, `sync.WaitGroup`, `*forge.LiveClient`
- `initialized_in` and `used_in` values are consistent with test lifecycle

#### 6b. Import Completeness ✅

| Import | Used By Scenarios | Status |
|:-------|:-----------------|:-------|
| `context` | All (ctx parameter) | ✅ |
| `testing` | All | ✅ |
| `fmt` | TS-002 (fmt.Errorf) | ✅ |
| `strings` | TS-002 (strings.Contains) | ✅ |
| `sync` | TS-014 (sync.WaitGroup) | ✅ |
| `errors` | TS-008, TS-013 (errors.Is) | ✅ |
| `testify/assert` | All assertions | ✅ |
| `testify/require` | Critical assertions | ✅ |
| `forge` | All (FakeClient/LiveClient) | ✅ |
| `scaffold` | TS-004–010 (ComparePathPresence) | ✅ |

All referenced types and functions have corresponding imports declared.

#### 6c. Code Structure Validity ✅

All 18 `code_structure` blocks contain valid Go test function signatures:
- Proper `func Test...(t *testing.T)` format
- Comment blocks describe arrange-act-assert structure
- No syntax errors in templates

#### 6d. Timeout Appropriateness ✅

No timeout references needed — unit tests with FakeClient execute synchronously. Functional tests with HTTP mocks also execute synchronously. No long-running operations.

#### Finding

- **D6-6b-001**
  - **Severity:** MAJOR
  - **Dimension:** Code Generation Readiness
  - **Description:** The `code_generation_config.package_name` is `"scaffold"` but scenarios TS-011–TS-014 test `forge.FakeClient` directly and scenarios TS-015–TS-018 test `forge.LiveClient`. The Go stubs correctly use `package scaffold` (suggesting black-box testing from the `scaffold` package), but the FakeClient and LiveClient tests would more naturally belong in `package forge_test` or `package forge`. This may cause compilation issues if `FakeClient`/`LiveClient` internal fields are accessed.
  - **Evidence:** `code_generation_config.package_name: "scaffold"` but scenarios 11–18 test `forge` package types. Go stubs all declare `package scaffold`.
  - **Remediation:** Either (a) split code generation into two packages: `scaffold` for ComparePathPresence tests and `forge` for FakeClient/LiveClient tests, or (b) verify that all FakeClient/LiveClient fields accessed in tests are exported and accessible from the `scaffold` package. If `FakeClient.FileContents`, `FakeClient.ListRepositoryFilesErr`, etc. are exported, `package scaffold` is acceptable.
  - **Actionable:** true

---

## Recommendations

1. **[MAJOR] D2-2b-001: Add `patterns` field to all scenarios** — **Remediation:** Add `patterns: { primary: "unit-test", helpers_required: [] }` (or `"functional-test"` for TS-015–018) to each scenario to comply with v2.1-enhanced schema. — **Actionable:** yes
2. **[MAJOR] D4.5-4.5a-001: Remove `related_prs` from STD metadata** — **Remediation:** Delete the `related_prs` section from `document_metadata`. The STP already references the issue. — **Actionable:** yes
3. **[MAJOR] D6-6b-001: Resolve package_name split for forge/scaffold tests** — **Remediation:** Verify that all `forge` types accessed in tests are exported (likely they are, given FakeClient's design). Document the cross-package testing approach in `code_generation_config`. — **Actionable:** yes
4. **[MINOR] D2-2b-002: Standardize tier naming** — **Remediation:** Map "Unit Tests" → "Tier 1", "Functional" → "Tier 2" per v2.1 spec. — **Actionable:** yes
5. **[MINOR] D2-2b-003: Populate or remove empty `test_data` fields** — **Remediation:** Either inline FakeClient configurations as `resource_definitions` or remove `test_data: {}`. — **Actionable:** yes
6. **[MINOR] D1-1b-001: STP requirement IDs are partially blank** — **Remediation:** Populate STP Section III requirement IDs (STP-side fix). — **Actionable:** no
7. **[MINOR] D4-4a-001: Empty cleanup arrays** — **Remediation:** No action needed — correct for FakeClient tests. — **Actionable:** no
8. **[MINOR] D5-5a-001: File-level markers could include priority** — **Remediation:** Informational only. — **Actionable:** no

---

## Confidence Notes

| Factor | Status |
|:-------|:-------|
| STD YAML parseable | YES |
| STP file available | YES |
| Go stubs present | YES (3 files, 18 functions) |
| Python stubs present | NO (not applicable — no E2E scenarios) |
| Pattern library available | NO |
| All scenarios reviewed | YES (18/18) |
| Project review rules loaded | NO |

**Confidence rationale:** MEDIUM — STD YAML is valid, STP is available for full traceability review, and Go stubs are present with complete scenario coverage. Confidence is reduced from HIGH because: (1) no pattern library was available for Dimension 3d validation, (2) no project-specific `review_rules.yaml` was loaded — all rules applied are general defaults, reducing domain-specific review precision. Review precision reduced: 100% of rules using generic defaults. Consider adding project-specific `review_rules.yaml` or enabling `repo_files_fetch`.
