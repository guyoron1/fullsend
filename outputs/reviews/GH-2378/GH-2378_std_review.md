# STD Review Report: GH-2378

**Reviewed:**
- STD YAML: `outputs/std/GH-2378/GH-2378_test_description.yaml`
- STP Source: `outputs/stp/GH-2378/GH-2378_test_plan.md`
- Go Stubs: `outputs/std/GH-2378/go-tests/` (4 files, 10 pending tests)
- Python Stubs: N/A (not generated)

**Date:** 2026-06-21
**Reviewer:** QualityFlow Automated Review (v1.1.0)
**Review Rules Schema:** N/A (dynamically extracted, default_ratio ~0.55)

---

## Verdict: NEEDS_REVISION

## Summary

| Metric | Value |
|:-------|:------|
| Dimensions reviewed | 7/7 |
| Critical findings | 3 |
| Major findings | 4 |
| Minor findings | 3 |
| Actionable findings | 10 |
| Weighted score | 62 |
| Confidence | MEDIUM |

## Traceability Summary

| Metric | Value |
|:-------|:------|
| STP scenarios | 10 |
| STD scenarios | 10 |
| Forward coverage (STP→STD) | 10/10 (100%) |
| Reverse coverage (STD→STP) | 10/10 (100%) |
| Orphan STD scenarios | 0 |
| Missing STD scenarios | 0 |

---

## Findings by Dimension

### Dimension 1: STP-STD Traceability — Score: 90/100

**Forward Traceability (STP → STD):**

| STP Row | STP Summary | STD Scenario | Match |
|:--------|:------------|:-------------|:------|
| 1 | Agent error detection at branch check point | 001 (TS-GH-2378-001) | ✅ Full |
| 2 | Agent error detection at changed-files check point | 002 (TS-GH-2378-002) | ✅ Full |
| 3 | Noop behavior preserved for successful no-op runs | 003 (TS-GH-2378-003) | ✅ Full |
| 4 | Error comment distinguishes agent errors from post-script errors | 004 (TS-GH-2378-004) | ✅ Full |
| 5 | Error comment includes agent exit code | 005 (TS-GH-2378-005) | ✅ Full |
| 6 | Non-agent errors preserve existing error comment format | 006 (TS-GH-2378-006) | ✅ Full |
| 7 | AGENT_EXIT_CODE propagated from Go harness to post-script | 007 (TS-GH-2378-007) | ✅ Full |
| 8 | Agent errors with produced changes still proceed normally | 008 (TS-GH-2378-008) | ✅ Full |
| 9 | Detached HEAD with agent error handled correctly | 009 (TS-GH-2378-009) | ✅ Full |
| 10 | End-to-end agent failure produces correct issue comment | 010 (TS-GH-2378-010) | ✅ Full |

**Reverse Traceability (STD → STP):**

All 10 STD scenarios reference `requirement_id: "GH-2378"` which is present in STP Section III. ✅

**Count Consistency:**
- `total_scenarios: 10` → actual: 10 ✅
- `unit_test_count: 9` → actual "Unit Tests" tier: 9 ✅
- `functional_count: 1` → actual "Functional" tier: 1 ✅
- `p0_count: 5` → actual: 5 ✅
- `p1_count: 4` → actual: 4 ✅
- `p2_count: 1` → actual: 1 ✅

**STP Reference:** `document_metadata.stp_reference.file` = `outputs/stp/GH-2378/GH-2378_test_plan.md` — file exists ✅

**Findings:**

- **D1-1a-001**
  - **Severity:** MINOR
  - **Dimension:** STP-STD Traceability
  - **Description:** STP Section III has empty `Requirement ID` fields for 9 of 10 requirement rows. Only the first row explicitly carries `GH-2378`. While all scenarios map to the same Jira ticket, the STP's missing IDs make automated traceability verification fragile.
  - **Evidence:** STP Section III rows 2–10 show blank `Requirement ID:` fields.
  - **Remediation:** Populate all STP Section III rows with `GH-2378` as the Requirement ID.
  - **Actionable:** true (STP fix, not STD)

---

### Dimension 2: STD YAML Structure — Score: 40/100

**Document-Level Structure:**
- [x] `document_metadata` exists with all required fields
- [x] `document_metadata.std_version` is "2.1-enhanced"
- [x] `code_generation_config` exists
- [x] `code_generation_config.std_version` is "2.1-enhanced"
- [x] `common_preconditions` exists
- [x] `scenarios` array exists and is non-empty (10 scenarios)
- [ ] `code_generation_config.package_name` is "tests" — acceptable

**Per-Scenario Required Fields:**

| Field | Present in all 10? | Notes |
|:------|:-------------------|:------|
| `scenario_id` | ✅ Yes | Sequential 001–010 |
| `test_id` | ✅ Yes | Format `TS-GH-2378-{NNN}` matches config |
| `tier` | ✅ Yes | Uses "Unit Tests"/"Functional" — see finding |
| `priority` | ✅ Yes | P0/P1/P2 |
| `requirement_id` | ✅ Yes | All "GH-2378" |
| `patterns` | ❌ **MISSING** | Not present in any scenario |
| `variables` | ✅ Yes | closure_scope arrays present |
| `test_structure` | ✅ Yes | describe/context/it present |
| `code_structure` | ❌ **MISSING** | Not present in any scenario |
| `test_objective` | ✅ Yes | title/what/why/acceptance_criteria |
| `test_data` | ✅ Yes | environment_setup present |
| `test_steps` | ✅ Yes | setup/test_execution/cleanup |
| `assertions` | ✅ Yes | 1–4 assertions per scenario |

**Findings:**

- **D2-2b-001**
  - **Severity:** CRITICAL
  - **Dimension:** STD YAML Structure
  - **Description:** The `patterns` field is missing from all 10 scenarios. This is a required v2.1-enhanced field that specifies the primary test pattern and helper libraries. Without it, the code generator cannot select appropriate test templates or import helper libraries.
  - **Evidence:** None of the 10 scenarios in the `scenarios` array contain a `patterns` key.
  - **Remediation:** Add a `patterns` block to each scenario with at least `primary_pattern` and `helpers_required`. For bash-focused tests (001–006, 008–009), consider a "shell-function-unit" pattern. For Go harness test (007), use "unit-test-env-propagation". For the functional test (010), use "functional-integration".
  - **Actionable:** true

- **D2-2b-002**
  - **Severity:** CRITICAL
  - **Dimension:** STD YAML Structure
  - **Description:** The `code_structure` field is missing from all 10 scenarios. This field provides the Ginkgo/testing framework code skeleton hint that the code generator uses to emit properly structured test functions. Its absence will cause the generator to fall back to generic templates or fail.
  - **Evidence:** None of the 10 scenarios contain a `code_structure` key.
  - **Remediation:** Add `code_structure` to each scenario. For ginkgo-style tests: `"Context -> PendingIt"`. For stdlib testing: `"func TestXxx(t *testing.T) { t.Run(...) }"`. Must align with the chosen framework (see D6-6c-001).
  - **Actionable:** true

- **D2-2b-003**
  - **Severity:** MAJOR
  - **Dimension:** STD YAML Structure
  - **Description:** The `tier` field uses non-standard values "Unit Tests" and "Functional" instead of the expected "Tier 1" or "Tier 2" vocabulary. While these labels are descriptive, they do not match the standard tier taxonomy used by QualityFlow's code generation pipeline and test routing.
  - **Evidence:** Scenarios 001–009: `tier: "Unit Tests"`, Scenario 010: `tier: "Functional"`.
  - **Remediation:** Map "Unit Tests" → appropriate tier (likely "Tier 1" given project config has `tier1_tests: true`). Map "Functional" → "Tier 1" or "Tier 2" based on project testing strategy. Alternatively, if the project intentionally uses descriptive tier names, document this mapping in `project.yaml`.
  - **Actionable:** true

---

### Dimension 3: Pattern Matching Correctness — Score: 0/100

Pattern matching cannot be assessed because the `patterns` field is missing from all scenarios (see D2-2b-001). No pattern library exists at the expected path (`patterns/tier1_patterns.yaml`).

| Scenario | Primary Pattern | Helpers | Decorators | Status |
|:---------|:----------------|:--------|:-----------|:-------|
| 001–010 | MISSING | MISSING | MISSING | ❌ FAIL |

**Findings:**

(Covered by D2-2b-001 — no additional findings beyond the structural absence.)

---

### Dimension 4: Test Step Quality — Score: 70/100

| Scenario | Setup | Execution | Cleanup | Assertions | Isolation | Error Paths | Status |
|:---------|:------|:----------|:--------|:-----------|:----------|:------------|:-------|
| 001 | 2 | 2 | 1 | 2 | ✅ PASS | ✅ Negative | PASS |
| 002 | 2 | 2 | 1 | 2 | ✅ PASS | ✅ Negative | PASS |
| 003 | 1 | 3 | 1 | 3 | ✅ PASS | ✅ Positive | PASS |
| 004 | 1 | 2 | 1 | 2 | ✅ PASS | ✅ Negative | PASS |
| 005 | 1 | 2 | 1 | 1 | ✅ PASS | ✅ Detail | PASS |
| 006 | 1 | 2 | 1 | 2 | ✅ PASS | ✅ Positive | PASS |
| 007 | 1 | 2 | 1 | 2 | ✅ PASS | ✅ Negative | PASS |
| 008 | 1 | 1 | 1 | 1 | ✅ PASS | ✅ Boundary | PASS |
| 009 | 1 | 1 | 1 | 1 | ✅ PASS | ✅ Edge case | PASS |
| 010 | 2 | 2 | 1 | 4 | ⚠️ WARN | ✅ E2E | WARN |

**Error Path Coverage:** Good. The STD has a healthy mix:
- **Negative/failure paths:** 001, 002, 004, 007, 009 (agent error detection)
- **Positive/preservation paths:** 003, 006 (noop and non-agent error preservation)
- **Boundary/edge cases:** 008 (changes exist despite error), 009 (detached HEAD)
- **Integration:** 010 (end-to-end pipeline)

**Findings:**

- **D4-4b-001**
  - **Severity:** MAJOR
  - **Dimension:** Test Step Quality
  - **Description:** Multiple test steps contain pseudo-commands instead of concrete, executable commands. Steps like "Simulate branch check returning false", "simulate detached HEAD", and "Set up mock to capture comment body" are design placeholders, not actionable test instructions. A test implementer cannot determine the concrete mechanism from these steps.
  - **Evidence:**
    - Scenario 001 SETUP-02: `command: "Simulate branch check returning false"`
    - Scenario 002 SETUP-02: `command: "Simulate branch check true, changed files count 0"`
    - Scenario 007 SETUP-01: `command: "Set up test case with expected exit code"`
    - Scenario 008 SETUP-01: `command: "export AGENT_EXIT_CODE=1; simulate changed files"`
    - Scenario 009 SETUP-01: `command: "export AGENT_EXIT_CODE=1; simulate detached HEAD"`
    - Scenario 010 SETUP-02: `command: "Set up mock to capture comment body"`
  - **Remediation:** Replace pseudo-commands with concrete mechanisms. For bash tests, use function mocking (e.g., `git() { echo "false"; }` for branch check simulation). For Go tests, use test fixtures or interfaces. For the mock scenario, specify the interception mechanism (e.g., `PATH-prepend mock gh binary` or `GH_MOCK_DIR`).
  - **Actionable:** true

- **D4-4b-002**
  - **Severity:** MINOR
  - **Dimension:** Test Step Quality
  - **Description:** Scenario 010 (E2E) step TEST-01 has `validation: "Script completes (may exit non-zero, that's expected)"` — uncertain verification language. Verification must be definitive about what constitutes pass/fail.
  - **Evidence:** Scenario 010, step TEST-01 validation field.
  - **Remediation:** Change to: "Script exits with expected non-zero code (e.g., 1). Capture exit code for later assertion."
  - **Actionable:** true

---

### Dimension 4.5: STD Content Policy — Score: 70/100

**Findings:**

- **D4.5-a-001**
  - **Severity:** MAJOR
  - **Dimension:** STD Content Policy
  - **Description:** The `document_metadata.related_prs` field contains a direct PR URL (`https://github.com/fullsend-ai/fullsend/pull/2381`). PR URLs are implementation artifacts that belong in the STP (which references them in Section I), not in the STD. The STD describes *what* to test, not *what code changed*. Including PR references creates a maintenance burden and ties the test design to a specific implementation.
  - **Evidence:**
    ```yaml
    related_prs:
      - repo: "fullsend-ai/fullsend"
        pr_number: 2381
        url: "https://github.com/fullsend-ai/fullsend/pull/2381"
        title: "Fix code agent status comment to reflect actual outcome"
        merged: true
    ```
  - **Remediation:** Remove the `related_prs` field entirely from `document_metadata`. The STP already references PR #2381 in Section I.3.
  - **Actionable:** true

**No Implementation Details in Stubs:** ✅ All 4 stub files contain only pending markers (`PendingIt` + `Skip`), framework imports, and PSE comments. No fixture implementations, helper function bodies, or concrete API calls found.

**Test Environment Separation:** ✅ No infrastructure creation, cluster setup, or feature gate enablement code in stubs.

---

### Dimension 5: PSE Docstring Quality — Score: 80/100

**Go Stubs:**

| Stub File | Tests | PSE Present | Quality |
|:----------|:------|:------------|:--------|
| `detect_noop_stubs_test.go` | 5 (001,002,003,008,009) | ✅ All | Good |
| `build_error_comment_stubs_test.go` | 3 (004,005,006) | ✅ All | Good |
| `agent_exit_code_propagation_stubs_test.go` | 1 (007) | ✅ All | Good |
| `e2e_agent_failure_stubs_test.go` | 1 (010) | ✅ All | Good |

**Per-stub analysis:**

- **detect_noop_stubs_test.go:**
  - All 5 Context blocks have Preconditions, Steps, and Expected sections ✅
  - Preconditions are specific (e.g., "AGENT_EXIT_CODE set to non-zero value (e.g., 1)") ✅
  - Steps are brief but clear (e.g., "1. Call detect_noop function") — acceptable for unit tests
  - Expected results are measurable (e.g., "detect_noop returns 'agent_error' (not 'noop')") ✅
  - Module-level comment references STP file ✅
  - test_ids present in all PendingIt descriptions ✅

- **build_error_comment_stubs_test.go:**
  - 3 Context blocks with complete PSE ✅
  - Steps include multi-step verification (call function, inspect output) ✅
  - Expected includes both positive and negative assertions (contains X, does NOT contain Y) ✅

- **agent_exit_code_propagation_stubs_test.go:**
  - 1 Context block with complete PSE ✅
  - Steps reference Go-specific verification (inspect cmd.Env) ✅

- **e2e_agent_failure_stubs_test.go:**
  - 1 Context block with complete PSE ✅
  - Expected has 4 clear assertions including negative check ✅

**Python Stubs:** N/A (not generated)

**Findings:**

- **D5-5a-001**
  - **Severity:** MINOR
  - **Dimension:** PSE Docstring Quality
  - **Description:** PSE Steps sections across all stubs are single-line descriptions (e.g., "1. Call detect_noop function") rather than detailed multi-step instructions. While acceptable for unit tests where the action is straightforward, the E2E test (010) would benefit from more detailed numbered steps describing the full pipeline execution.
  - **Evidence:** e2e_agent_failure_stubs_test.go Steps: "1. Run post-code.sh with agent failure conditions / 2. Capture the comment body posted to the issue" — lacks detail on how to set up the mock and capture.
  - **Remediation:** For scenario 010, expand Steps to include: (1) set environment variables, (2) create mock gh binary, (3) run post-code.sh, (4) read captured output from mock.
  - **Actionable:** true

---

### Dimension 6: Code Generation Readiness — Score: 25/100

**Findings:**

- **D6-6c-001**
  - **Severity:** CRITICAL
  - **Dimension:** Code Generation Readiness
  - **Description:** Critical framework mismatch between configuration and generated stubs. The `go.yaml` config specifies `framework: "testing"` (Go stdlib) and `code_generation_config` in the STD also says `framework: "testing"` with `assertion_library: "testify"` and `subtest_style: "t.Run"`. However, all 4 generated stub files use **Ginkgo v2** (`github.com/onsi/ginkgo/v2`) with `Describe`, `Context`, and `PendingIt` constructs. This means the code generator will produce stdlib `testing` + `testify` code that is structurally incompatible with the existing Ginkgo stubs. Tests generated from this STD will not compile against the stub structure.
  - **Evidence:**
    - `go.yaml`: `framework: "testing"`, `subtest_style: "t.Run"`
    - `code_generation_config`: `framework: "testing"`, `assertion_style: "testify"`
    - All stubs: `import . "github.com/onsi/ginkgo/v2"`, using `Describe()`, `Context()`, `PendingIt()`
  - **Remediation:** Either (a) update `go.yaml` and `code_generation_config` to `framework: "ginkgo-v2"` with ginkgo imports and assertion style, OR (b) regenerate stubs using stdlib `testing` + `testify` with `func TestXxx(t *testing.T)` and `t.Run()` subtests. Option (a) is recommended since the stubs are already written in Ginkgo style.
  - **Actionable:** true

- **D6-6b-001**
  - **Severity:** MAJOR
  - **Dimension:** Code Generation Readiness
  - **Description:** The `code_generation_config.imports` section does not include `github.com/onsi/ginkgo/v2` or `github.com/onsi/gomega`, yet all stubs import and use Ginkgo v2. The code generator will not add the required framework imports, causing compilation failures.
  - **Evidence:** `code_generation_config.imports.test_framework` lists only `testify/assert` and `testify/require`. Stubs import `github.com/onsi/ginkgo/v2`.
  - **Remediation:** If using Ginkgo (recommended per D6-6c-001): add `github.com/onsi/ginkgo/v2` and optionally `github.com/onsi/gomega` to `imports.test_framework`. Remove `testify` imports if switching fully to Gomega, or keep both if using testify assertions within Ginkgo.
  - **Actionable:** true

**Variable Declarations:** ✅ Variable names are valid identifiers. Types are valid (`string`, `bool`, `int`, `[]string`). Lifecycle references are logical.

**Timeout Appropriateness:** No timeout references in test steps — acceptable for unit tests of bash functions and Go variable scoping.

---

## Recommendations

Ordered by severity:

1. **[CRITICAL] D2-2b-001 — Missing `patterns` field in all scenarios** — **Remediation:** Add `patterns` block to each scenario with `primary_pattern` and `helpers_required`. Use domain-appropriate patterns: "shell-function-unit" for bash tests, "unit-test-env-propagation" for Go harness test, "functional-integration" for E2E test. — **Actionable:** yes

2. **[CRITICAL] D2-2b-002 — Missing `code_structure` field in all scenarios** — **Remediation:** Add `code_structure` to each scenario with the framework-appropriate skeleton. Must align with framework decision in D6-6c-001. — **Actionable:** yes

3. **[CRITICAL] D6-6c-001 — Framework mismatch: config says "testing" but stubs use Ginkgo v2** — **Remediation:** Update `go.yaml` framework to "ginkgo-v2" and update `code_generation_config` to match. Update imports section accordingly. This is the root cause of D6-6b-001 as well. — **Actionable:** yes

4. **[MAJOR] D4.5-a-001 — PR URLs in `document_metadata.related_prs`** — **Remediation:** Remove the `related_prs` field from `document_metadata`. — **Actionable:** yes

5. **[MAJOR] D6-6b-001 — Missing Ginkgo imports in `code_generation_config`** — **Remediation:** Add `github.com/onsi/ginkgo/v2` to `imports.test_framework`. — **Actionable:** yes

6. **[MAJOR] D2-2b-003 — Non-standard tier values ("Unit Tests"/"Functional")** — **Remediation:** Replace with "Tier 1" or define project-specific tier mapping in `project.yaml`. — **Actionable:** yes

7. **[MAJOR] D4-4b-001 — Pseudo-commands in test steps** — **Remediation:** Replace "Simulate..." placeholders with concrete bash mocking or Go test fixture commands. — **Actionable:** yes

8. **[MINOR] D1-1a-001 — Empty Requirement IDs in STP Section III** — **Remediation:** Populate all STP rows with `GH-2378`. — **Actionable:** yes (STP fix)

9. **[MINOR] D4-4b-002 — Uncertain verification language in scenario 010** — **Remediation:** Make validation definitive. — **Actionable:** yes

10. **[MINOR] D5-5a-001 — Brief PSE Steps in E2E stub** — **Remediation:** Expand E2E test steps with detailed mock setup instructions. — **Actionable:** yes

---

## Confidence Notes

| Factor | Status |
|:-------|:-------|
| STD YAML parseable | YES |
| STP file available | YES |
| Go stubs present | YES (4 files) |
| Python stubs present | NO (not generated) |
| Pattern library available | NO |
| All scenarios reviewed | YES (10/10) |
| Project review rules loaded | PARTIAL (dynamic extraction, ~55% defaults) |

**Confidence rationale:** MEDIUM confidence. The STD YAML is fully parseable and the STP is available for traceability verification. All 10 scenarios were reviewed across all 7 dimensions. Go stubs are present and well-structured. However, no pattern library exists for pattern validation (Dimension 3d cannot be fully assessed), no Python stubs exist, and review rules are operating at ~55% defaults (no `review_rules.yaml` or `tier1_patterns.yaml` found). Review precision is reduced for pattern matching and framework-specific conventions.

**Review precision note:** 55% of review rules are using generic defaults. Project-specific review precision is reduced. To improve: add a `review_rules.yaml` to `config/projects/fullsend/` or enable `repo_files_fetch` to pull team standards from the repository. Keys using defaults: `stp_rules.abstraction.internal_to_user_mappings`, `stp_rules.abstraction.acceptable_locations`, `stp_rules.dependencies.*`, `stp_rules.strategy.*`, `stp_rules.metadata.*`, `stp_rules.scope.*`.
