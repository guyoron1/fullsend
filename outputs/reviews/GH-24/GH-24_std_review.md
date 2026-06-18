# STD Review Report: GH-24

**Reviewed:**
- STD YAML: `outputs/std/GH-24/GH-24_test_description.yaml`
- STP Source: `outputs/stp/GH-24/GH-24_test_plan.md`
- Go Stubs: `outputs/std/GH-24/go-tests/` (8 files, 34 test stubs)
- Python Stubs: N/A (not configured for project)

**Date:** 2026-06-18
**Reviewer:** QualityFlow Automated Review (v1.1.0)
**Review Rules Schema:** 1.1.0 (dynamically extracted, no static override)

---

## Verdict: APPROVED_WITH_FINDINGS

## Summary

| Metric | Value |
|:-------|:------|
| Dimensions reviewed | 7/7 |
| Critical findings | 0 |
| Major findings | 5 |
| Minor findings | 4 |
| Actionable findings | 9 |
| Confidence | MEDIUM |
| Weighted score | 76/100 |

## Dimension Scores

| Dimension | Weight | Pass Rate | Weighted |
|:----------|:-------|:----------|:---------|
| 1. STP-STD Traceability | 30% | 97% | 29.1 |
| 2. STD YAML Structure | 20% | 60% | 12.0 |
| 3. Pattern Matching Correctness | 10% | 20% | 2.0 |
| 4. Test Step Quality | 15% | 92% | 13.8 |
| 4.5. STD Content Policy | 10% | 75% | 7.5 |
| 5. PSE Docstring Quality | 10% | 88% | 8.8 |
| 6. Code Generation Readiness | 5% | 65% | 3.25 |
| **Total** | **100%** | | **76.45** |

## Traceability Summary

| Metric | Value |
|:-------|:------|
| STP scenarios | 34 |
| STD scenarios | 34 |
| Forward coverage (STP→STD) | 34/34 (100%) |
| Reverse coverage (STD→STP) | 34/34 (100%) |
| Orphan STD scenarios | 0 |
| Missing STD scenarios | 0 |

---

## Findings by Dimension

### Dimension 1: STP-STD Traceability (97/100)

#### 1a. Forward Traceability (STP → STD)

All 8 requirement groups in STP Section III are fully covered in the STD. Each STP scenario has a corresponding STD scenario with matching test objective:

| STP Requirement Group | STP Scenarios | STD Scenarios | Coverage |
|:----------------------|:--------------|:--------------|:---------|
| isRetryable identifies 5xx as retryable | 7 | TS-GH-24-001 – 007 | 100% |
| do() retries on 5xx with backoff | 5 | TS-GH-24-008 – 012 | 100% |
| No double-retry | 3 | TS-GH-24-013 – 015 | 100% |
| retryOnRepoRace handles only 404/409 | 5 | TS-GH-24-016 – 020 | 100% |
| isTransientStatus returns true only for 404/409 | 3 | TS-GH-24-021 – 023 | 100% |
| Error messages | 3 | TS-GH-24-024 – 026 | 100% |
| File operations with retryOnRepoRace | 4 | TS-GH-24-027 – 030 | 100% |
| Rate limit behavior preserved | 4 | TS-GH-24-031 – 034 | 100% |

#### 1b. Reverse Traceability (STD → STP)

All 34 STD scenarios have `requirement_id: "GH-24"` which maps to STP Section III. No orphan scenarios found.

**Positive note:** The STP review (D2-COV-001) flagged that 7 of 8 requirement groups in the STP had blank Requirement ID fields. The STD correctly populated all 34 scenarios with `requirement_id: "GH-24"`, resolving the traceability gap at the STD level.

#### 1c. Count Consistency

| Metadata Field | Declared | Actual | Status |
|:---------------|:---------|:-------|:-------|
| total_scenarios | 34 | 34 | ✅ PASS |
| unit_test_count | 30 | 30 | ✅ PASS |
| functional_count | 4 | 4 | ✅ PASS |
| e2e_count | 0 | 0 | ✅ PASS |
| p0_count | 15 | 15 | ✅ PASS |
| p1_count | 19 | 19 | ✅ PASS |

All metadata counts verified against actual scenario array. Zero discrepancies.

#### 1d. STP Reference

- `stp_reference.file`: `outputs/stp/GH-24/GH-24_test_plan.md` — file exists ✅
- `stp_reference.version`: `v1` ✅
- `stp_reference.sections_covered`: `Section III - Requirements-to-Tests Mapping` ✅

#### 1e. Priority-Testability Consistency

All 15 P0 scenarios describe fully testable behaviors using mock HTTP servers. No P0 scenario is documented as untestable or deferred. ✅

**Dimension 1 finding:**

- **D1-1a-001 (MINOR):** Tier naming uses "Unit Tests" and "Functional" instead of the standard "Tier 1"/"Tier 2" nomenclature from the v2.1-enhanced spec. This is consistent between STP and STD and matches the project's testing model, but diverges from the schema standard.
  - **evidence:** `tier: "Unit Tests"` in 30 scenarios, `tier: "Functional"` in 4 scenarios
  - **remediation:** Either update the STD to use "Tier 1"/"Tier 2" or document the project's tier naming convention in the project config. No functional impact since STP/STD are internally consistent.
  - **actionable:** true

---

### Dimension 2: STD YAML Structure (60/100)

#### 2a. Document-Level Structure

| Check | Status |
|:------|:-------|
| `document_metadata` exists | ✅ PASS |
| `document_metadata.std_version` is "2.1-enhanced" | ✅ PASS |
| `code_generation_config` exists | ✅ PASS |
| `code_generation_config.std_version` is "2.1-enhanced" | ✅ PASS |
| `code_generation_config.package_name` present | ✅ PASS (`github_test`) |
| `common_preconditions` exists | ✅ PASS |
| `scenarios` array exists and non-empty | ✅ PASS (34 scenarios) |

#### 2b. Per-Scenario Required Fields

| Field | Present in | Missing in | Status |
|:------|:-----------|:-----------|:-------|
| `scenario_id` | 34/34 | 0 | ✅ PASS |
| `test_id` | 34/34 | 0 | ✅ PASS (format: TS-GH-24-NNN) |
| `tier` | 34/34 | 0 | ✅ PASS |
| `priority` | 34/34 | 0 | ✅ PASS |
| `requirement_id` | 34/34 | 0 | ✅ PASS |
| `test_objective` | 34/34 | 0 | ✅ PASS (title + what + why + AC) |
| `test_steps` | 34/34 | 0 | ✅ PASS |
| `assertions` | 34/34 | 0 | ✅ PASS |
| `patterns` | 0/34 | 34 | ❌ FAIL |
| `test_structure` | 0/34 | 34 | ⚠️ N/A (framework-specific) |
| `code_structure` | 0/34 | 34 | ⚠️ N/A (framework-specific) |
| `test_data` | 0/34 | 34 | ❌ FAIL |
| `variables` | 2/34 | 32 | ❌ FAIL |

No duplicate `scenario_id` or `test_id` values found. All IDs are sequential (1-34) and follow the `TS-GH-24-{NUM:03d}` format.

#### 2c. v2.1-Specific Checks

**Framework context:** This project uses Go stdlib `testing` with testify assertions (not Ginkgo). Fields `test_structure` and `code_structure` describe Ginkgo-specific constructs (`Context -> BeforeAll -> It`) which are not applicable. The stubs correctly use `t.Run` subtests instead.

**Dimension 2 findings:**

- **D2-b-001 (MAJOR):** Missing `patterns` field in all 34 scenarios. No pattern metadata (primary pattern, helpers_required, decorators) is assigned to any scenario. While no project pattern library exists (`tier1_patterns.yaml` not found), the `patterns` field is part of the v2.1-enhanced schema and aids code generation and review.
  - **evidence:** Zero `patterns:` keys in the scenarios section of the STD YAML.
  - **remediation:** Add basic pattern metadata to each scenario. At minimum, assign a descriptive primary pattern ID (e.g., `"unit-function-call"`, `"unit-httptest-mock"`, `"integration-file-operation"`) to categorize each scenario's test approach.
  - **actionable:** true

- **D2-b-002 (MAJOR):** Missing `test_data` field in all 34 scenarios. Test data (resource definitions, API endpoints, mock configurations) is embedded in test_steps instead of being declared separately. Extracting test data into a dedicated field improves reusability and parameterization.
  - **evidence:** Zero `test_data:` keys in scenarios. Setup steps like "Create HTTP response with status 500" embed data inline.
  - **remediation:** Add `test_data` sections to scenarios, especially those using httptest mock servers (scenarios 8-20, 24-30, 34). Example: `test_data: { mock_server: { initial_status: 502, subsequent_status: 200 } }`.
  - **actionable:** true

- **D2-b-003 (MAJOR):** Missing `variables.closure_scope` in 32 of 34 scenarios. Only scenarios 1 and 8 declare variables. Scenarios using httptest servers (8-15, 16-20, 24-30, 34) should declare `callCount`, `server`, and `resp` variables at minimum.
  - **evidence:** Scenarios 2-7, 9-34 (except 8) lack `variables:` section. Scenario 8 correctly declares `callCount` (atomic.Int32) and `server` (httptest.Server).
  - **remediation:** Add `variables.closure_scope` to all scenarios that use mock servers, counters, or test subjects. Use scenario 8 as the template for httptest-based scenarios.
  - **actionable:** true

- **D2-b-004 (MINOR):** Missing `test_structure` and `code_structure` fields in all scenarios. These fields are designed for Ginkgo-based projects (`Context -> BeforeAll -> It` structure). Since this project uses Go stdlib `testing` with `t.Run` subtests, these fields are not applicable. The stubs correctly implement the stdlib testing pattern.
  - **evidence:** `code_generation_config.framework: "testing"` — stdlib testing, not Ginkgo.
  - **remediation:** No action required. If the v2.1-enhanced schema is updated to support stdlib `testing`, add the equivalent fields (e.g., `test_structure: { function: "TestXxx", subtests: "t.Run" }`). Alternatively, document this as an intentional schema adaptation.
  - **actionable:** false

---

### Dimension 3: Pattern Matching Correctness (20/100)

#### 3a-3c. Pattern, Helper, and Decorator Assignment

No pattern metadata exists in the STD YAML. This dimension cannot be fully evaluated.

| Scenario | Primary Pattern | Helpers | Decorators | Status |
|:---------|:----------------|:--------|:-----------|:-------|
| 1-7 | — | — | — | ❌ No pattern metadata |
| 8-12 | — | — | — | ❌ No pattern metadata |
| 13-15 | — | — | — | ❌ No pattern metadata |
| 16-20 | — | — | — | ❌ No pattern metadata |
| 21-23 | — | — | — | ❌ No pattern metadata |
| 24-26 | — | — | — | ❌ No pattern metadata |
| 27-30 | — | — | — | ❌ No pattern metadata |
| 31-34 | — | — | — | ❌ No pattern metadata |

#### 3d. Pattern Library Validation

No pattern library found at `config/projects/fullsend/patterns/tier1_patterns.yaml`. Skipped.

**Note:** The absence of patterns is already captured in D2-b-001. The 20/100 score reflects that while no incorrect patterns exist (no false positives), the absence of pattern metadata means the STD provides no categorical organization of test approaches.

---

### Dimension 4: Test Step Quality (92/100)

#### 4a. Step Completeness

| Category | Scenarios | Setup | Execution | Cleanup | Status |
|:---------|:----------|:------|:----------|:--------|:-------|
| isRetryable (1-7) | 7 | 6/7 have setup | 7/7 | 0/7 need cleanup | ✅ |
| do() retry (8-12) | 5 | 5/5 | 5/5 | 4/5 have cleanup | ✅ |
| No double-retry (13-15) | 3 | 3/3 | 3/3 | 3/3 | ✅ |
| retryOnRepoRace (16-20) | 5 | 5/5 | 5/5 | 5/5 | ✅ |
| isTransientStatus (21-23) | 3 | 0/3 (none needed) | 3/3 | 0/3 (none needed) | ✅ |
| Error messages (24-26) | 3 | 3/3 | 3/3 | 3/3 | ✅ |
| File operations (27-30) | 4 | 4/4 | 4/4 | 4/4 | ✅ |
| Rate limit (31-34) | 4 | 2/4 | 4/4 | 1/4 | ✅ |

**Cleanup justification:** Scenarios 1-7, 21-23, 31-33 have `cleanup: []` which is correct — these test pure functions (`isRetryable`, `isTransientStatus`) with in-memory HTTP response objects that require no explicit cleanup. All httptest.NewServer-based scenarios (8-15, 16-20, 24-30, 34) correctly include `server.Close()` cleanup.

#### 4b. Step Quality

Steps are specific and actionable throughout. Examples of good step quality:

- ✅ **Specific actions:** "Create HTTP response with status 502 and non-empty body" (not "Create a response")
- ✅ **Code-level commands:** `retryable, err := isRetryable(resp)` (concrete function call)
- ✅ **Measurable validation:** `retryable == true && err == nil` (not "should work")
- ✅ **Sequential IDs:** SETUP-01, TEST-01, TEST-02, CLEANUP-01 consistently used

No vague actions, uncertain verification language, or missing validations detected.

#### 4b.2. Abstraction Level

Test steps appropriately use function-level language (`isRetryable`, `do()`, `retryOnRepoRace`) which is correct for unit tests targeting internal API functions. The functions under test ARE the user interface at this level.

#### 4c. Logical Flow

All scenarios follow correct logical flow:
1. Setup creates mock servers/responses before use ✅
2. Test execution calls the function under test with setup resources ✅
3. Cleanup closes servers after test ✅
4. No circular dependencies ✅

#### 4d. Upgrade Test Structure

No upgrade scenarios present. N/A.

#### 4e. Test Dependency Structure

Scenarios are independent within each group. No inter-scenario dependencies exist — each test creates its own mock server and tests a single behavior. This is correct for unit tests. ✅

#### 4f. Assertion Quality

| Metric | Value |
|:-------|:------|
| Total assertions | 52 |
| P0 assertions | 26 |
| P1 assertions | 26 |
| Generic descriptions | 0 |
| Missing conditions | 0 |

All assertions have specific descriptions, measurable conditions, and assigned priorities. `[NEGATIVE]` indicators correctly used for negative test assertions. Good distribution between P0 and P1.

**Dimension 4 finding:**

- **D4-a-001 (MINOR):** Scenario 12 (context cancellation) has a cleanup step that calls both `server.Close()` and `cancel()`, but the test may need to handle the case where `cancel()` was already called during test execution. While this is minor and the Go context package handles double-cancel gracefully, the cleanup step could note this.
  - **evidence:** Scenario 12 cleanup: "Close mock server and call cancel()" — `cancel()` is also called in TEST-01.
  - **remediation:** Add a comment noting that `cancel()` is idempotent and safe to call twice, or restructure to call `cancel()` only in cleanup via defer.
  - **actionable:** true

---

### Dimension 4.5: STD Content Policy (75/100)

#### 4.5a. Banned Content

**STD YAML:**

- **D4.5-a-001 (MAJOR):** `document_metadata.related_prs` contains PR URLs and implementation metadata. Per STD content policy, PR URLs are implementation artifacts that belong in the STP (Section I), not the STD. The STD describes *what* to test, not *what code changed*.
  - **evidence:**
    ```yaml
    related_prs:
      - repo: "guyoron1/fullsend"
        pr_number: 24
        url: "https://github.com/guyoron1/fullsend/pull/24"
      - repo: "fullsend-ai/fullsend"
        pr_number: 2342
        url: "https://github.com/fullsend-ai/fullsend/pull/2342"
    ```
  - **remediation:** Remove the `related_prs` section from `document_metadata`. The STP reference (`stp_reference.file`) provides sufficient traceability to the source requirements. If PR context is needed for debugging, it should be in the STP, not the STD.
  - **actionable:** true

**Stub files:**
- No PR URLs in any stub file docstrings ✅
- No branch names or commit references ✅
- No developer names ✅
- All file-level comments reference STP file path ✅

#### 4.5b. No Implementation Details in Stubs

All 8 stub files contain only:
- Package declaration (`package github_test`) ✅
- Import of `"testing"` only ✅
- File-level PSE comments ✅
- `t.Run` subtests with PSE docstrings ✅
- `t.Skip("Phase 1: Design only - awaiting implementation")` pending markers ✅

No fixture implementations, helper functions, or project-internal imports found. Stubs are pure design artifacts. ✅

#### 4.5c. Test Environment Separation

No infrastructure provisioning, cluster setup, or feature gate code found in stubs. ✅

---

### Dimension 5: PSE Docstring Quality (88/100)

#### 5a. Go Stubs

**File-level quality:**

| Stub File | Tests | PSE Present | STP Ref | test_id Tags | Status |
|:----------|:------|:------------|:--------|:-------------|:-------|
| is_retryable_stubs_test.go | 7 | 7/7 | ✅ | 7/7 | ✅ PASS |
| do_retry_stubs_test.go | 5 | 5/5 | ✅ | 5/5 | ✅ PASS |
| no_double_retry_stubs_test.go | 3 | 3/3 | ✅ | 3/3 | ✅ PASS |
| retry_on_repo_race_stubs_test.go | 5 | 5/5 | ✅ | 5/5 | ✅ PASS |
| is_transient_status_stubs_test.go | 3 | 3/3 | ✅ | 3/3 | ✅ PASS |
| error_messages_stubs_test.go | 3 | 3/3 | ✅ | 3/3 | ✅ PASS |
| file_operations_stubs_test.go | 4 | 4/4 | ✅ | 4/4 | ✅ PASS |
| rate_limit_preserved_stubs_test.go | 4 | 4/4 | ✅ | 4/4 | ✅ PASS |

**PSE Content Quality:**

- **Preconditions:** Specific and concrete. Good examples:
  - ✅ "Mock HTTP server that returns 502 on first call, 200 on second"
  - ✅ "HTTP response constructed with status 502 and non-empty body"
  - ✅ "Table-driven test cases for codes 500, 501, 502, 503, 504"

- **Steps:** Numbered, actionable, unambiguous. Good examples:
  - ✅ "1. Call isRetryable with the 500 response"
  - ✅ "1. Call do() with request to mock server / 2. Verify call count on mock server"
  - ✅ "1. Cancel context after first request received by mock server / 2. Verify do() returns context error"

- **Expected:** Measurable outcomes with verification conditions:
  - ✅ "isRetryable returns true / No error returned"
  - ✅ "do() returns successful response (err == nil, status 200) / Mock server received exactly 2 requests"
  - ✅ "Response body is fully drained (Read returns 0 bytes and io.EOF)"

- **[NEGATIVE] indicators:** Correctly used on all negative test PSE blocks (scenarios 7, 10, 12, 15, 18, 19, 20, 23, 30). ✅

**Markers section:** All files include `tier1` marker in file-level comment. ✅

**Dimension 5 finding:**

- **D5-a-001 (MINOR):** Scenarios 21-23 (`isTransientStatus` tests) omit the `Preconditions:` section in their PSE docstrings, going directly to `Steps:`. While these are simple function call tests with no preconditions, the PSE format should include `Preconditions:` with "None" or "N/A" for consistency.
  - **evidence:** `is_transient_status_stubs_test.go` subtests start with `Steps:` without a preceding `Preconditions:` section.
  - **remediation:** Add `Preconditions: None` or `Preconditions: Go toolchain 1.23+` to the 3 subtests for format consistency.
  - **actionable:** true

---

### Dimension 6: Code Generation Readiness (65/100)

#### 6a. Variable Declarations

Only 2 of 34 scenarios declare variables. Where present, declarations are valid:

| Scenario | Variable | Type | Valid Go Type | Status |
|:---------|:---------|:-----|:--------------|:-------|
| 1 | resp | *http.Response | ✅ | PASS |
| 1 | retryable | bool | ✅ | PASS |
| 8 | callCount | *atomic.Int32 | ✅ | PASS |
| 8 | server | *httptest.Server | ✅ | PASS |

The remaining 32 scenarios lack variable declarations, which reduces code generation automation confidence (see D2-b-003).

#### 6b. Import Completeness

`code_generation_config.imports` includes:

| Category | Imports | Used in Scenarios | Status |
|:---------|:--------|:------------------|:-------|
| standard | context, fmt, io, net/http, net/http/httptest, strings, sync/atomic, testing, time | All categories used | ✅ |
| test_framework | testify/assert, testify/require | Assertions across all scenarios | ✅ |
| project | internal/forge/github | Function under test | ✅ |

All imports are justified by scenario content. No unused imports detected. ✅

**Note:** Stub files only import `"testing"`, which is correct for the stub phase. Full imports are deferred to the implementation phase.

#### 6c. Code Structure Validity

The stubs demonstrate valid Go test structure:
- `func TestXxx(t *testing.T)` top-level test functions ✅
- `t.Run("[test_id:TS-GH-24-NNN] description", func(t *testing.T) { ... })` subtests ✅
- `t.Skip("Phase 1: Design only - awaiting implementation")` pending markers ✅
- Proper package declaration ✅
- Conceptually compilable (would compile with `go test` given proper imports) ✅

#### 6d. Timeout Appropriateness

No explicit timeout constants referenced in test steps. For unit tests using httptest mock servers, the default Go test timeout (10 minutes) is more than sufficient. The context cancellation test (scenario 12) correctly uses `context.WithCancel` rather than a timeout, which is appropriate. ✅

---

## Recommendations

Ordered by severity:

1. **[MAJOR]** Remove `related_prs` from STD YAML document_metadata — **Remediation:** Delete the `related_prs` section. PR references belong in the STP, not the STD. The `stp_reference` field provides traceability. — **Actionable:** yes

2. **[MAJOR]** Add `patterns` metadata to all 34 scenarios — **Remediation:** Assign descriptive primary pattern IDs. Suggested patterns: `"unit-function-return"` (scenarios 1-7, 21-23, 31-33), `"unit-httptest-retry"` (8-12, 34), `"unit-httptest-no-double-retry"` (13-15), `"unit-httptest-scoped-retry"` (16-20), `"unit-error-message"` (24-26), `"functional-file-operation"` (27-30). — **Actionable:** yes

3. **[MAJOR]** Add `test_data` sections to httptest-based scenarios — **Remediation:** Extract mock server configuration into `test_data` fields. Example: `test_data: { mock_responses: [{ status: 502, count: 1 }, { status: 200, count: 1 }] }`. This improves parameterization and table-driven test generation. — **Actionable:** yes

4. **[MAJOR]** Add `variables.closure_scope` to remaining 32 scenarios — **Remediation:** Use scenario 8 as template. Httptest-based scenarios need `callCount`, `server`, `resp` variables. Simple function tests need `resp` and `retryable` variables. — **Actionable:** yes

5. **[MAJOR]** Add `variables.closure_scope` to the 32 scenarios missing them — **Remediation:** For isRetryable tests: declare `resp (*http.Response)` and `retryable (bool)`. For do() tests: declare `callCount (*atomic.Int32)`, `server (*httptest.Server)`, `resp (*http.Response)`. For file operation tests: declare `server`, `callCount`, and operation-specific variables. — **Actionable:** yes

6. **[MINOR]** Tier naming diverges from v2.1-enhanced standard — **Remediation:** Either map "Unit Tests" → "Tier 1" and "Functional" → a documented category, or formally document the project's tier naming convention in project config. — **Actionable:** yes

7. **[MINOR]** Missing `test_structure` and `code_structure` fields — **Remediation:** No action required for stdlib `testing` framework. Consider adding framework-appropriate equivalents if the v2.1-enhanced schema is extended to support non-Ginkgo frameworks. — **Actionable:** false

8. **[MINOR]** Missing `Preconditions:` section in isTransientStatus test PSE — **Remediation:** Add `Preconditions: None` to scenarios 21-23 PSE docstrings for format consistency. — **Actionable:** yes

9. **[MINOR]** Context cancellation test cleanup calls cancel() twice — **Remediation:** Add a comment noting cancel() idempotency, or restructure to use defer for cleanup. — **Actionable:** yes

---

## Confidence Notes

| Factor | Status |
|:-------|:-------|
| STD YAML parseable | YES |
| STP file available | YES |
| Go stubs present | YES (8 files, 34 tests) |
| Python stubs present | NO (not configured for project) |
| Pattern library available | NO (tier1_patterns.yaml not found) |
| All scenarios reviewed | YES |
| Project review rules loaded | YES (dynamically extracted from config) |

**Confidence rationale:** MEDIUM confidence. STD YAML is valid and fully parseable. STP is available enabling complete traceability review (Dimension 1). All 8 Go stub files are present and reviewed. Python stubs are correctly absent per project config (`python_tests` not explicitly configured). Pattern library is absent, limiting Dimension 3 evaluation to structural checks only. Review rules were dynamically extracted from config files with no static override — `default_ratio` is approximately 0.45 (MEDIUM range). The primary confidence limitation is the absence of a pattern library and the high default_ratio, which reduces project-specific review precision for pattern matching and helper library validation.

**Review precision note:** ~45% of review rules are using generic defaults. Project-specific review precision is reduced for pattern matching (Dimension 3) and code generation readiness (Dimension 6). To improve: add `tier1_patterns.yaml` to `config/projects/fullsend/patterns/` or create a `review_rules.yaml` with project-specific std_rules.
