# STD Review Report — GH-24

**Jira:** GH-24 — fix(forge): retry 5xx server errors at the HTTP client level
**Reviewer:** QualityFlow STD Reviewer (automated)
**Date:** 2026-06-17
**Verdict:** APPROVED_WITH_FINDINGS
**Weighted Score:** 93/100
**Confidence:** HIGH

---

## Executive Summary

The STD for GH-24 is well-structured, comprehensive, and demonstrates excellent traceability to the source STP. All 23 scenarios map 1:1 to the 23 test scenarios described in the STP's Section III requirements mapping. Test step quality is high with concrete code templates ready for generation. One major finding affects code generation readiness (package visibility), and two minor findings relate to naming accuracy and a small coverage gap.

---

## Dimension Scores

| # | Dimension | Weight | Score | Weighted |
|:--|:----------|:-------|:------|:---------|
| 1 | STP-STD Traceability | 30% | 100 | 30.0 |
| 2 | STD YAML Structure | 20% | 92 | 18.4 |
| 3 | Pattern Matching Correctness | 10% | 95 | 9.5 |
| 4 | Test Step Quality | 15% | 97 | 14.6 |
| 4.5 | STD Content Policy | 10% | 92 | 9.2 |
| 5 | PSE Docstring Quality | 10% | 95 | 9.5 |
| 6 | Code Generation Readiness | 5% | 40 | 2.0 |
| | **Total** | **100%** | | **93.2** |

---

## Dimension 1: STP-STD Traceability (100/100)

**Assessment:** Full traceability achieved.

All 7 requirement groups from STP Section III are covered by 23 STD scenarios with complete 1:1 mapping:

| STP Requirement Group | STD Scenarios | Coverage |
|:----------------------|:--------------|:---------|
| do() retries on 5xx (500-504) | TS-GH-24-001 through 004 | FULL |
| Rate limit retry (429, 403 secondary) unaffected | TS-GH-24-005 through 007 | FULL |
| retryOnRepoRace narrowed to 404/409 | TS-GH-24-008 through 010 | FULL |
| No double-retry across layers | TS-GH-24-011 through 013 | FULL |
| Retry exhaustion error messages | TS-GH-24-014 through 016 | FULL |
| Non-retryable errors pass through | TS-GH-24-017 through 020 | FULL |
| File operations single-layer retry | TS-GH-24-021 through 023 | FULL |

**Zero-trust verification:** Manually counted 23 `scenario_id` entries in YAML. Metadata claims `total_scenarios: 23` — **VERIFIED CORRECT**.

Priority counts verified:
- P0 claimed: 7, actual P0 scenarios (1-7): **7 VERIFIED**
- P1 claimed: 16, actual P1 scenarios (8-23): **16 VERIFIED**
- Functional claimed: 23, actual: **23 VERIFIED**
- E2E claimed: 0, actual: **0 VERIFIED**

All requirement_ids reference "GH-24" which is consistent with a single-ticket feature.

---

## Dimension 2: STD YAML Structure (92/100)

**Assessment:** Well-formed with minor issues.

### Verified Structure Elements
- `document_metadata`: Complete with all required fields
- `code_generation_config`: Present with framework, imports, patterns
- `common_preconditions`: Infrastructure and environment specified
- `scenarios`: Array of 23 well-formed entries

### Per-Scenario Schema Compliance
All 23 scenarios contain: `scenario_id`, `test_id`, `tier`, `priority`, `mvp`, `requirement_id`, `requirement_summary`, `variables`, `test_structure`, `test_objective` (with `what`, `why`, `acceptance_criteria`), `classification`, `specific_preconditions`, `test_data`, `test_steps` (with `setup`, `test_execution`, `cleanup`), `assertions`, `dependencies`.

### Test ID Format
All 23 test IDs follow the `TS-GH-24-NNN` format with sequential numbering 001-023. **VERIFIED CORRECT** against `_defaults.yaml` format `"TS-{JIRA_ID}-{NUM:03d}"`.

### Deductions
- (-5) Scenario 4 function name `TestIsRetryableReturnsFalseNon5xx` is factually inaccurate — 505 and 511 ARE 5xx status codes (see Finding F-002)
- (-3) No `python_stubs` section despite `_defaults.yaml` listing python stub output path (mitigated by project config: fullsend has no `python.yaml` with enabled tests for this component)

---

## Dimension 3: Pattern Matching Correctness (95/100)

**Assessment:** Patterns are internally consistent and appropriate.

No `tier1_patterns.yaml` exists for pattern validation, so this dimension evaluates internal consistency of pattern usage:

| Pattern | Scenarios Using | Correctness |
|:--------|:---------------|:------------|
| Table-driven (`t.Run` loop) | 1, 4, 8, 9, 12 | Correct — used for parameterized status codes |
| Single test | 2, 3, 5-7, 10-11, 13-23 | Correct — used for unique flow validation |
| httptest mock server | 2, 3, 10-13, 17-23 | Correct — used for multi-call integration |
| Pure function test | 1, 4-9 | Correct — direct function call, no server |

**Pattern consistency:** Table-driven tests correctly use `tc.statusCode` and `tc.name` fields. Mock servers consistently use `callCount` tracking. `defer server.Close()` present in all server-based tests.

(-5) Minor: No explicit backoff/timing validation patterns despite STP mentioning "exponential: 1s, 2s, 4s" backoff. Acceptable since STP marks performance testing as Not Applicable.

---

## Dimension 4: Test Step Quality (97/100)

**Assessment:** Excellent step quality with concrete, deterministic steps.

All test steps include:
- `step_id`: Consistently formatted (SETUP-01, TEST-01)
- `action`: Clear natural language description of what to do
- `command`: Specific function/method call
- `validation`: Expected observable outcome
- `code_template`: Complete, compilable Go code snippet

**Strengths:**
- Code templates exactly match the assertions in the `assertions` section
- Mock server setup code is realistic and would produce correct behavior
- Call counting logic is correct across all scenarios
- Error message checks use `assert.Contains` appropriately for substring matching

(-3) Scenario 10 (`TestRetryOnRepoRaceDoesNotRetry5xx`) step TEST-01 references `retryOnRepoRace` as a standalone function, but the actual implementation context (whether it's a method or package-level function) is not clarified in the test_data or preconditions.

---

## Dimension 4.5: STD Content Policy (92/100)

**Assessment:** Clean content with one naming accuracy issue.

- No PII or real credentials
- All URLs use mock patterns (`org/repo`, `abc123`)
- Test data uses safe placeholder values
- No hard-coded secrets or tokens

(-8) Factual inaccuracy in naming: Scenario 4 and its function `TestIsRetryableReturnsFalseNon5xx` labels HTTP status codes 505 and 511 as "non-5xx". These are in fact 5xx status codes (500-599 range). The correct description is "non-retryable 5xx" or "5xx codes outside the 500-504 retry range". This propagates through the test_objective title, requirement_summary in the scenario, and the Go stub function name.

---

## Dimension 5: PSE Docstring Quality (95/100)

**Assessment:** Consistent, informative PSE blocks.

All 8 Go stub files follow a consistent PSE format:

```
/*
Preconditions:
    - <setup requirements>

Steps:
    1. <action>
    2. <action>

Expected:
    - <outcome>
*/
```

**Strengths:**
- Every test function has a PSE block
- Preconditions distinguish "pure function test" from "mock server required"
- Steps are numbered and action-oriented
- Expected outcomes are specific and measurable
- File-level comments include STP reference and Jira ID
- Inline `[test_id:TS-GH-24-NNN]` comments enable traceability

(-5) PSE blocks could benefit from including the requirement group context (e.g., "Group 1: 5xx retry"), though the file-level comment partially addresses this.

---

## Dimension 6: Code Generation Readiness (40/100)

**Assessment:** Major package visibility issue blocks compilation.

### Critical Issue: Package Name vs. Unexported Symbol Access

The `code_generation_config` specifies:
```yaml
package_name: "github_test"
```

All 8 Go stub files declare `package github_test` (external test package). However, the tests call **unexported** functions and methods:

| Symbol | Export Status | Files Using |
|:-------|:-------------|:------------|
| `isRetryable()` | unexported | isretryable_5xx, ratelimit_retry |
| `isTransientStatus()` | unexported | transient_status |
| `retryOnRepoRace()` | unexported | transient_status |
| `newTestClient()` | unexported | do_retry_5xx, double_retry, error_messages, file_operations, non_retryable |
| `client.do()` | unexported | do_retry_5xx, error_messages, non_retryable, transient_status |

In Go, external test packages (`package foo_test`) **cannot access unexported symbols** from the package under test. This means all generated tests would fail to compile.

**Remediation:** Change `package_name` from `"github_test"` to `"github"` (internal test package) in both:
1. `code_generation_config.package_name` in the STD YAML
2. All 8 Go stub files' `package` declarations

### Secondary Issues
- (-5) `newTestClient` is referenced but not defined in any stub — it's assumed to exist as a test helper in the target package. This should be documented in `common_preconditions` or a setup note.
- (-5) Import `"fmt"` is listed in `code_generation_config.imports.standard` but only used in 4 of 8 stub files. Unused imports cause Go compilation errors. Each stub should declare only the imports it uses (currently correct in stubs, but the config suggests adding all).

---

## Findings Summary

### F-001 [MAJOR] — Package name prevents compilation of generated tests
- **Dimension:** Code Generation Readiness
- **Severity:** Major
- **Location:** `code_generation_config.package_name` and all 8 Go stub files
- **Description:** `package_name: "github_test"` (external test package) is incompatible with calls to unexported functions (`isRetryable`, `isTransientStatus`, `retryOnRepoRace`, `newTestClient`, `do`). External test packages cannot access unexported symbols.
- **Remediation:** Change `package_name` to `"github"` in the STD YAML `code_generation_config` section, and update all 8 Go stub file package declarations from `package github_test` to `package github`.
- **Actionable:** true

### F-002 [MINOR] — Inaccurate naming: "non-5xx" for 505/511 status codes
- **Dimension:** STD Content Policy / YAML Structure
- **Severity:** Minor
- **Location:** Scenario 4 (`test_id: TS-GH-24-004`), function `TestIsRetryableReturnsFalseNon5xx`
- **Description:** HTTP status codes 505 and 511 are labeled as "non-5xx" in the scenario title, function name, and test objective. These codes ARE in the 5xx range (500-599). The correct term is "non-retryable 5xx" or "5xx codes outside 500-504 range".
- **Remediation:** Rename to `TestIsRetryableReturnsFalseOutOfRange5xx` or `TestIsRetryableReturnsFalseNonRetryable5xx`. Update `test_objective.title`, `requirement_summary`, and stub function name accordingly.
- **Actionable:** true

### F-003 [MINOR] — Missing 501 coverage in retryable 5xx test
- **Dimension:** STP-STD Traceability
- **Severity:** Minor
- **Location:** Scenario 1 (`test_id: TS-GH-24-001`)
- **Description:** The requirement states "5xx server errors (500-504)" but scenario 1 only tests 500, 502, 503, 504 — skipping 501 Not Implemented. If the implementation retries all codes in the 500-504 range, 501 should be tested. If 501 is intentionally excluded, this should be documented.
- **Remediation:** Either add 501 to the table-driven test cases in scenario 1, or add a note in `test_data` explaining why 501 is excluded (e.g., "501 Not Implemented is a client-error-like code not typically transient").
- **Actionable:** true

---

## Artifacts Reviewed

| Artifact | Status | Notes |
|:---------|:-------|:------|
| STD YAML | Reviewed | `outputs/std/GH-24/GH-24_test_description.yaml` (2358 lines) |
| STP | Reviewed | `outputs/stp/GH-24/GH-24_test_plan.md` (267 lines) |
| Go Stubs (8 files) | Reviewed | All files in `outputs/std/GH-24/go-tests/` |
| Python Stubs | N/A | No Python stubs generated (expected: project uses Go only for this component) |
| Pattern Library | N/A | No `tier1_patterns.yaml` found |

---

## Review Rules Context

- **Project:** fullsend (FullSend)
- **Config source:** `/sandbox/workspace/agent-input/config/projects/fullsend/`
- **Framework:** Go `testing` + `testify`
- **Review rules source:** Dynamic extraction from config files (no static `review_rules.yaml`)
- **repo_rules:** Not fetched (`repo_files_fetch: false`)
- **Default ratio:** ~0.45 (MEDIUM confidence for project-specific rules)
