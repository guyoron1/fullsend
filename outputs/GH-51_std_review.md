# STD Review Report: GH-51

**Reviewed:**
- STD YAML: outputs/std/GH-51/GH-51_test_description.yaml
- STP Source: outputs/stp/GH-51/GH-51_test_plan.md
- Go Stubs: outputs/std/GH-51/go-tests/ (6 files, 19 tests)
- Python Stubs: N/A (no python-tests directory)

**Date:** 2026-06-20
**Reviewer:** QualityFlow Automated Review (v1.1.0)
**Review Rules Schema:** 1.1.0

---

## Verdict: NEEDS_REVISION

## Summary

| Metric | Value |
|:-------|:------|
| Dimensions reviewed | 7/7 |
| Critical findings | 4 |
| Major findings | 14 |
| Minor findings | 6 |
| Actionable findings | 22 |
| Confidence | MEDIUM |
| Weighted score | 52 |

## Traceability Summary

| Metric | Value |
|:-------|:------|
| STP scenarios | 19 |
| STD scenarios | 19 |
| Forward coverage (STP to STD) | 19/19 (100%) |
| Reverse coverage (STD to STP) | 19/19 (100%) |
| Orphan STD scenarios | 0 |
| Missing STD scenarios | 0 |

---

## Findings by Dimension

### Dimension 1: STP-STD Traceability (Weight: 30%) -- Score: 68/100

#### 1a. Forward Traceability (STP to STD)

All 19 STP scenarios have corresponding STD scenarios. Requirement ID GH-51 maps correctly across all entries. Scenario text keyword overlap is above 0.50 threshold for all 19 scenarios.

No CRITICAL or MAJOR findings for forward coverage.

#### 1b. Reverse Traceability (STD to STP)

All 19 STD scenarios reference requirement_id "GH-51" which exists in the STP Section III. No orphan scenarios found.

#### 1c. Count Consistency

**Finding D1-1c-001:**
- finding_id: "D1-1c-001"
- severity: CRITICAL
- dimension: STP-STD Traceability
- description: Metadata uses non-standard tier labels "Functional" and "Unit Tests" instead of "Tier 1" and "Tier 2". The fields `functional_count` and `unit_count` are non-standard; the v2.1-enhanced schema expects `tier_1_count` and `tier_2_count`. Actual count by tier label: 10 scenarios with tier "Functional", 9 scenarios with tier "Unit Tests". The metadata counts (functional_count=10, unit_count=9) match the actual scenario counts for their respective labels, but the labels themselves are wrong.
- evidence: "document_metadata.functional_count: 10, document_metadata.unit_count: 9" -- no tier_1_count or tier_2_count fields exist
- remediation: Rename all scenario tier values from "Functional" to "Tier 1" and from "Unit Tests" to "Tier 2". Replace metadata fields functional_count/unit_count with tier_1_count/tier_2_count.
- actionable: true

#### 1d. STP Reference

STP reference file path "outputs/stp/GH-51/GH-51_test_plan.md" is valid and matches the actual file location.

No findings.

#### 1e. Tier Mismatch Between STP and STD

**Finding D1-1e-001:**
- finding_id: "D1-1e-001"
- severity: CRITICAL
- dimension: STP-STD Traceability
- description: All 19 STD scenarios use non-standard tier labels ("Functional", "Unit Tests") that do not match the STP's tier labels (which also uses "Functional" and "Unit Tests" instead of the canonical "Tier 1"/"Tier 2"). While STP and STD are internally consistent, both deviate from the v2.1-enhanced specification which requires "Tier 1" and "Tier 2". This means code generation will produce incorrect tier decorators.
- evidence: "STD scenario 001 tier: 'Functional', STP tier: 'Functional'. Expected: 'Tier 1'. STD scenario 002 tier: 'Unit Tests', STP tier: 'Unit Tests'. Expected: 'Tier 2'."
- remediation: In both STP and STD, replace "Functional" with "Tier 1" and "Unit Tests" with "Tier 2" everywhere.
- actionable: true

#### 1f. Near-Duplicate Scenarios

**Finding D1-1f-001:**
- finding_id: "D1-1f-001"
- severity: MAJOR
- dimension: STP-STD Traceability
- description: Scenarios 003 and 010 are near-duplicates. Both test that CLAUDE.md is not injected when the runtime is not Claude. Scenario 003 (P0, "Functional") and scenario 010 (P1, "Functional") have the same test_steps, same assertion logic (os.IsNotExist check), and nearly identical test_objective. The STP maps them to two separate requirement groups ("CLAUDE.md pointer is injected..." group 1 and "CLAUDE.md injection is skipped for non-Claude runtimes" group 4), but the test content is indistinguishable.
- evidence: "Scenario 003 test_objective.title: 'Verify no injection when runtime is not Claude'. Scenario 010 test_objective.title: 'Verify no injection for non-Claude agent runtime'. Both check os.IsNotExist on CLAUDE.md path."
- remediation: Merge scenarios 003 and 010 into a single scenario, or differentiate them by testing distinct aspects (e.g., 003 tests at the doInjectClaudeMDPointer level, 010 tests at the runAgent integration level with specific non-Claude runtime values).
- actionable: true

---

### Dimension 2: STD YAML Structure (Weight: 20%) -- Score: 45/100

#### 2a. Document-Level Structure

**Finding D2-2a-001:**
- finding_id: "D2-2a-001"
- severity: CRITICAL
- dimension: STD YAML Structure
- description: code_generation_config.framework is "testing" with testify assertion library. The project configuration specifies Go/Ginkgo v2 for Tier 1. The stubs correctly use Ginkgo v2 (onsi/ginkgo/v2), but the YAML metadata contradicts this. Code generation from this YAML would produce standard Go test functions instead of Ginkgo specs.
- evidence: "code_generation_config.framework: 'testing', code_generation_config.assertion_library: 'testify'. Expected: framework: 'ginkgo-v2', assertion_library: 'gomega'."
- remediation: Change code_generation_config.framework to "ginkgo-v2" and assertion_library to "gomega". Update imports to include onsi/ginkgo/v2 and onsi/gomega instead of stretchr/testify.
- actionable: true

**Finding D2-2a-002:**
- finding_id: "D2-2a-002"
- severity: MAJOR
- dimension: STD YAML Structure
- description: code_generation_config.imports references testify (stretchr/testify/assert, stretchr/testify/require) instead of Gomega. The Go stubs use Ginkgo v2, so the imports in the YAML are inconsistent with the actual stub code.
- evidence: "imports.test_framework includes 'github.com/stretchr/testify/assert' and 'github.com/stretchr/testify/require'. Stubs import 'github.com/onsi/ginkgo/v2'."
- remediation: Replace testify imports with gomega imports in code_generation_config.imports.test_framework.
- actionable: true

#### 2b. Per-Scenario Required Fields

**Finding D2-2b-001:**
- finding_id: "D2-2b-001"
- severity: CRITICAL
- dimension: STD YAML Structure
- description: No scenario has a `patterns` field. The v2.1-enhanced schema requires a `patterns` field on every scenario with primary pattern and helpers_required. All 19 scenarios are missing this required field.
- evidence: "Checked all 19 scenarios (001-019). None contain a 'patterns' field."
- remediation: Add a patterns field to each scenario with at least primary_pattern and helpers_required. For this project (no pattern library), use descriptive pattern names like "file-injection", "casing-detection", "guard-condition", "error-handling", "flag-propagation".
- actionable: true

**Finding D2-2b-002:**
- finding_id: "D2-2b-002"
- severity: MAJOR
- dimension: STD YAML Structure
- description: No scenario has a `code_structure` field. The v2.1-enhanced schema requires code_structure with Ginkgo structure hints (Describe/Context/It blocks). All 19 scenarios are missing this required field.
- evidence: "Checked all 19 scenarios (001-019). None contain a 'code_structure' field."
- remediation: Add code_structure field to each scenario defining the Ginkgo v2 Describe/Context/It block structure.
- actionable: true

**Finding D2-2b-003:**
- finding_id: "D2-2b-003"
- severity: MAJOR
- dimension: STD YAML Structure
- description: Tier values use non-standard labels. All scenarios use "Functional" or "Unit Tests" instead of the required "Tier 1" or "Tier 2".
- evidence: "Scenario 001: tier: 'Functional'. Scenario 002: tier: 'Unit Tests'. Valid values per spec: 'Tier 1', 'Tier 2'."
- remediation: Replace tier "Functional" with "Tier 1" and tier "Unit Tests" with "Tier 2" across all 19 scenarios.
- actionable: true

#### 2c. v2.1-Specific Checks

**Finding D2-2c-001:**
- finding_id: "D2-2c-001"
- severity: MINOR
- dimension: STD YAML Structure
- description: Scenario 004 declares test_structure.type as "table-driven" but the scenario structure is identical to single-type scenarios (one setup, one test_execution, one cleanup). The table-driven aspect is only visible in the code_template content, not in the YAML structure. Scenario 012 also claims "table-driven" with similar structure.
- evidence: "Scenario 004: test_structure.type: 'table-driven'. Has single SETUP-01, TEST-01, CLEANUP-01 like all other scenarios."
- remediation: Either restructure table-driven scenarios to have multiple test_data entries representing the table rows, or change type to "single" and note the table-driven implementation approach in the code_template.
- actionable: true

---

### Dimension 3: Pattern Matching Correctness (Weight: 10%) -- Score: 0/100

**Finding D3-3a-001:**
- finding_id: "D3-3a-001"
- severity: MAJOR
- dimension: Pattern Matching Correctness
- description: No patterns field exists on any scenario, making pattern matching review impossible. The entire dimension scores 0. While no pattern library is available for this project, the STD schema still requires patterns metadata for code generation routing.
- evidence: "All 19 scenarios lack a 'patterns' field entirely."
- remediation: Add patterns field with at minimum primary_pattern to each scenario. Without a project pattern library, use descriptive identifiers.
- actionable: true

| Scenario | Primary Pattern | Helpers | Decorators | Status |
|:---------|:----------------|:--------|:-----------|:-------|
| 001-019 | MISSING | N/A | N/A | FAIL |

---

### Dimension 4: Test Step Quality (Weight: 15%) -- Score: 55/100

#### 4a. Step Completeness

All 19 scenarios have setup, test_execution, and cleanup steps. Cleanup steps consistently use "automatic via t.TempDir()" which is appropriate for filesystem-based tests.

No findings for step completeness.

#### 4b. Step Quality

**Finding D4-4b-001:**
- finding_id: "D4-4b-001"
- severity: MAJOR
- dimension: Test Step Quality
- description: Scenarios 015, 018, and 019 have vague, non-actionable test steps that do not describe concrete test actions. They assert only flag values rather than observable behavior.
- evidence: "Scenario 018 TEST-01: action='Verify CLAUDE.md injection proceeds when flag is true', code_template='assert.True(t, agentsMDAvailable)'. This only asserts a boolean variable, not any actual system behavior. Scenario 015 SETUP-01: action='Configure mock to trigger injection failure', command='Setup failing injection scenario', code_template is empty. Scenario 019 TEST-01 similarly just asserts assert.False(t, agentsMDAvailable)."
- remediation: Rewrite scenarios 015, 018, and 019 to test actual integration behavior. For 018: create a directory, simulate org AGENTS.md injection setting the flag, then verify doInjectClaudeMDPointer is called and creates CLAUDE.md. For 019: verify that with the flag false, no CLAUDE.md appears. For 015: inject a failing mock and verify the runAgent flow continues to completion.
- actionable: true

**Finding D4-4b-002:**
- finding_id: "D4-4b-002"
- severity: MAJOR
- dimension: Test Step Quality
- description: Scenario 003 test_execution step does not actually invoke any injection function. It only checks that CLAUDE.md does not exist, but never exercises the guard condition logic that should prevent injection. The test verifies a precondition (file does not exist) rather than testing the behavior (calling the injection path and confirming it was skipped).
- evidence: "Scenario 003 TEST-01 code_template: '_, statErr := os.Stat(filepath.Join(tmpDir, \"CLAUDE.md\"))\nassert.True(t, os.IsNotExist(statErr))'. No call to any injection function is made."
- remediation: Add a call to the runAgent flow or the guard condition check with a non-Claude runtime, then assert CLAUDE.md was not created.
- actionable: true

**Finding D4-4b-003:**
- finding_id: "D4-4b-003"
- severity: MINOR
- dimension: Test Step Quality
- description: Scenario 010 has the same test step issue as scenario 003 -- it only asserts file non-existence without invoking the guard condition code path. Combined with finding D1-1f-001, this makes both scenarios effectively untestable stubs that verify filesystem state rather than feature behavior.
- evidence: "Scenario 010 TEST-01 code_template checks os.IsNotExist without calling any function."
- remediation: Same as D4-4b-002. If scenarios 003 and 010 are kept separate, differentiate their test approaches.
- actionable: true

#### 4f. Assertion Quality

**Finding D4-4f-001:**
- finding_id: "D4-4f-001"
- severity: MINOR
- dimension: Test Step Quality
- description: All P0 scenarios have only P0 assertions. Scenarios 001 and 003 could reasonably have secondary P1 assertions (e.g., verifying log output, verifying no side effects).
- evidence: "Scenario 001: 2 assertions, both P0. Scenario 003: 1 assertion, P0."
- remediation: Consider adding P1 assertions for secondary verification (e.g., no unexpected files created, correct log messages emitted).
- actionable: true

| Scenario | Setup | Execution | Cleanup | Assertions | Status |
|:---------|:------|:----------|:--------|:-----------|:-------|
| 001 | 1 | 2 | 1 | 2 | PASS |
| 002 | 1 | 1 | 1 | 1 | PASS |
| 003 | 1 | 1 | 1 | 1 | WARN |
| 004 | 1 | 1 | 1 | 1 | PASS |
| 005 | 1 | 1 | 1 | 1 | PASS |
| 006 | 1 | 1 | 1 | 1 | PASS |
| 007 | 1 | 1 | 1 | 1 | PASS |
| 008 | 1 | 2 | 1 | 1 | PASS |
| 009 | 1 | 2 | 1 | 1 | PASS |
| 010 | 1 | 1 | 1 | 1 | WARN |
| 011 | 1 | 2 | 1 | 1 | PASS |
| 012 | 1 | 1 | 1 | 1 | PASS |
| 013 | 1 | 1 | 1 | 1 | PASS |
| 014 | 1 | 1 | 1 | 1 | PASS |
| 015 | 1 | 1 | 1 | 1 | FAIL |
| 016 | 1 | 1 | 1 | 1 | PASS |
| 017 | 1 | 1 | 1 | 1 | PASS |
| 018 | 1 | 1 | 1 | 1 | FAIL |
| 019 | 1 | 1 | 1 | 1 | FAIL |

---

### Dimension 4.5: STD Content Policy (Weight: 10%) -- Score: 50/100

#### 4.5a. Banned Content in STD YAML

**Finding D4.5-4.5a-001:**
- finding_id: "D4.5-4.5a-001"
- severity: MAJOR
- dimension: STD Content Policy
- description: document_metadata contains a `related_prs` field with PR URLs. PR URLs are implementation artifacts that belong in the STP (which references them in Section I), not in the STD. The STD describes what to test, not what code changed.
- evidence: "document_metadata.related_prs: [{repo: 'fullsend-ai/fullsend', pr_number: 51, url: 'https://github.com/guyoron1/fullsend/pull/51', title: 'Inject CLAUDE.md Pointer...', merged: true}]"
- remediation: Remove the related_prs field from document_metadata entirely. PR references should only appear in the STP.
- actionable: true

#### 4.5b. Implementation Details in Code Templates

**Finding D4.5-4.5b-001:**
- finding_id: "D4.5-4.5b-001"
- severity: MAJOR
- dimension: STD Content Policy
- description: Multiple scenarios contain concrete implementation code in code_template fields, including specific function calls (doInjectClaudeMDPointer, hasClaudeMD), mock implementations, and assertion code. While code_templates are acceptable in STD YAML as hints, these go beyond hints and contain complete runnable test implementations. This blurs the line between the STD design phase and the implementation phase.
- evidence: "Scenario 001 TEST-01 code_template contains full mock creation and function call: 'mockExec := func(ctx context.Context, cmd string) error { return nil }\nerr = doInjectClaudeMDPointer(tmpDir, mockExec)\nrequire.NoError(t, err)'"
- remediation: Reduce code_templates to pseudocode-level hints showing the test approach, not complete implementations. Leave implementation to the code generation phase.
- actionable: true

**Finding D4.5-4.5b-002:**
- finding_id: "D4.5-4.5b-002"
- severity: MINOR
- dimension: STD Content Policy
- description: code_generation_config.imports includes project-internal module "github.com/fullsend-ai/fullsend/internal/sandbox" which couples the STD to a specific internal package path. If the package is refactored, the STD becomes stale.
- evidence: "imports.project: ['github.com/fullsend-ai/fullsend/internal/sandbox']"
- remediation: Remove project-internal imports from STD YAML. Let the code generator resolve imports at generation time based on the actual codebase.
- actionable: true

---

### Dimension 5: PSE Docstring Quality (Weight: 10%) -- Score: 72/100

**Go Stubs:**

All 6 stub files have module-level comments with STP reference (outputs/stp/GH-51/GH-51_test_plan.md) and Jira reference (GH-51). No PR URLs in stubs. All 19 test blocks have PSE comment blocks with Preconditions/Steps/Expected sections. All use PendingIt() with Skip() per Ginkgo v2 conventions.

**Finding D5-5a-001:**
- finding_id: "D5-5a-001"
- severity: MAJOR
- dimension: PSE Docstring Quality
- description: TS-GH-51-003 is placed in the Context "when all guard conditions are met" in claude_md_injection_stubs_test.go, but it tests the opposite case (non-Claude runtime). Its preconditions state "Runtime is set to a non-Claude value" which contradicts the enclosing Context description "when all guard conditions are met".
- evidence: "File: claude_md_injection_stubs_test.go, Context: 'when all guard conditions are met', Test: TS-GH-51-003 'should not inject CLAUDE.md when runtime is not Claude'. PSE Preconditions: 'Runtime is set to a non-Claude value (e.g., codex)'"
- remediation: Move TS-GH-51-003 to claude_md_guard_conditions_stubs_test.go under the "when runtime is non-Claude agent" Context where TS-GH-51-010 already resides, or create a separate Context "when runtime is not Claude" in the injection stubs file.
- actionable: true

**Finding D5-5a-002:**
- finding_id: "D5-5a-002"
- severity: MAJOR
- dimension: PSE Docstring Quality
- description: TS-GH-51-015 is placed in the Context "when all guard conditions are met" in claude_md_injection_stubs_test.go, but it tests injection failure handling (a negative scenario). Its PSE preconditions state "doInjectClaudeMDPointer configured to fail" which contradicts the enclosing Context that describes the happy path.
- evidence: "File: claude_md_injection_stubs_test.go, Context: 'when all guard conditions are met', Test: TS-GH-51-015 'should continue agent run after injection failure'. PSE marker: [NEGATIVE]"
- remediation: Move TS-GH-51-015 to claude_md_error_handling_stubs_test.go under a new Context "when injection fails during agent run".
- actionable: true

**Finding D5-5c-001:**
- finding_id: "D5-5c-001"
- severity: MAJOR
- dimension: PSE Docstring Quality
- description: TS-GH-51-018 PSE Steps include "Verify agentsMDAvailable is true after org injection" which is a verification step that belongs in Expected, not Steps. Steps should describe actions, not verifications.
- evidence: "File: claude_md_flag_propagation_stubs_test.go, TS-GH-51-018 Steps: '1. Verify agentsMDAvailable is true after org injection 2. Check that CLAUDE.md injection code path is entered'"
- remediation: Restructure PSE. Steps should be: "1. Simulate org AGENTS.md injection success 2. Trigger runAgent flow with agentsMDAvailable=true and no existing CLAUDE.md". Expected should be: "agentsMDAvailable flag is true, CLAUDE.md injection code path is entered and creates the pointer file".
- actionable: true

**Finding D5-5c-002:**
- finding_id: "D5-5c-002"
- severity: MINOR
- dimension: PSE Docstring Quality
- description: TS-GH-51-019 PSE has the same "Verify..." pattern in Steps section. Step 1 reads "Verify agentsMDAvailable is false after failed org injection" which is a verification, not an action.
- evidence: "File: claude_md_flag_propagation_stubs_test.go, TS-GH-51-019 Steps: '1. Verify agentsMDAvailable is false after failed org injection'"
- remediation: Rewrite Step 1 as "Simulate org AGENTS.md injection failure" and move verification to Expected.
- actionable: true

**Python Stubs:** N/A (no Python stubs exist).

---

### Dimension 6: Code Generation Readiness (Weight: 5%) -- Score: 35/100

#### 6a. Variable Declarations

Variable declarations are present and reasonable across all scenarios. Types are valid Go types. initialized_in and used_in references are consistent.

No findings.

#### 6b. Import Completeness

**Finding D6-6b-001:**
- finding_id: "D6-6b-001"
- severity: MAJOR
- dimension: Code Generation Readiness
- description: code_generation_config.imports lists testify (assert, require) but the actual stubs use Ginkgo v2 (onsi/ginkgo/v2). The imports are inconsistent with the test framework actually in use. A code generator using these imports would produce non-compilable test files.
- evidence: "imports.test_framework: ['github.com/stretchr/testify/assert', 'github.com/stretchr/testify/require']. Stubs use: 'github.com/onsi/ginkgo/v2'."
- remediation: Replace testify imports with Ginkgo v2 and Gomega imports: "github.com/onsi/ginkgo/v2" and "github.com/onsi/gomega".
- actionable: true

#### 6c. Code Structure Validity

**Finding D6-6c-001:**
- finding_id: "D6-6c-001"
- severity: MINOR
- dimension: Code Generation Readiness
- description: No code_structure field exists on any scenario, so code generation would have to infer the Ginkgo structure. The stubs demonstrate the correct structure (Describe -> Context -> PendingIt), but the YAML does not encode it for automated generation.
- evidence: "All 19 scenarios lack code_structure field."
- remediation: Add code_structure field matching the Ginkgo v2 pattern: "Context -> BeforeAll -> It" as specified in project configuration.
- actionable: true

#### 6d. Timeout Appropriateness

No timeout_constants defined in code_generation_config. Scenarios that involve filesystem operations and git commands (009) would benefit from timeout specifications. However, for unit-level filesystem tests using t.TempDir(), the absence of explicit timeouts is acceptable.

No findings (minor concern but not rising to finding level).

---

## Recommendations

Ordered by severity:

1. **[CRITICAL] D1-1c-001** -- Tier labels use "Functional"/"Unit Tests" instead of "Tier 1"/"Tier 2" throughout metadata. -- **Remediation:** Replace all tier labels with "Tier 1" and "Tier 2" per v2.1-enhanced spec. -- **Actionable:** yes
2. **[CRITICAL] D1-1e-001** -- Tier labels in scenarios use non-standard values that will produce wrong code generation output. -- **Remediation:** Change all scenario tier values to "Tier 1" or "Tier 2". -- **Actionable:** yes
3. **[CRITICAL] D2-2a-001** -- Framework mismatch: YAML says "testing"/testify but stubs use Ginkgo v2. -- **Remediation:** Change code_generation_config.framework to "ginkgo-v2" and assertion_library to "gomega". -- **Actionable:** yes
4. **[CRITICAL] D2-2b-001** -- All 19 scenarios missing required `patterns` field. -- **Remediation:** Add patterns field with primary_pattern and helpers_required to each scenario. -- **Actionable:** yes
5. **[MAJOR] D1-1f-001** -- Scenarios 003 and 010 are near-duplicates testing the same behavior. -- **Remediation:** Merge or differentiate them. -- **Actionable:** yes
6. **[MAJOR] D2-2a-002** -- Imports list testify instead of Ginkgo/Gomega. -- **Remediation:** Update imports section. -- **Actionable:** yes
7. **[MAJOR] D2-2b-002** -- All 19 scenarios missing required `code_structure` field. -- **Remediation:** Add code_structure field to each scenario. -- **Actionable:** yes
8. **[MAJOR] D2-2b-003** -- Tier values are non-standard ("Functional"/"Unit Tests"). -- **Remediation:** Use "Tier 1"/"Tier 2". -- **Actionable:** yes
9. **[MAJOR] D3-3a-001** -- No patterns field on any scenario, dimension scores 0. -- **Remediation:** Add patterns metadata. -- **Actionable:** yes
10. **[MAJOR] D4-4b-001** -- Scenarios 015, 018, 019 have vague steps asserting only flag values. -- **Remediation:** Rewrite with concrete test actions and observable outcomes. -- **Actionable:** yes
11. **[MAJOR] D4-4b-002** -- Scenario 003 does not invoke any injection function. -- **Remediation:** Add actual function call with guard condition check. -- **Actionable:** yes
12. **[MAJOR] D4.5-4.5a-001** -- related_prs in document_metadata is a content policy violation. -- **Remediation:** Remove related_prs field. -- **Actionable:** yes
13. **[MAJOR] D4.5-4.5b-001** -- Code templates contain full implementations, not design hints. -- **Remediation:** Reduce to pseudocode-level hints. -- **Actionable:** yes
14. **[MAJOR] D5-5a-001** -- TS-GH-51-003 in wrong Context (happy path Context but tests negative case). -- **Remediation:** Move to guard conditions stub file. -- **Actionable:** yes
15. **[MAJOR] D5-5a-002** -- TS-GH-51-015 in wrong Context (happy path but tests failure). -- **Remediation:** Move to error handling stub file. -- **Actionable:** yes
16. **[MAJOR] D5-5c-001** -- TS-GH-51-018 PSE has "Verify" steps that belong in Expected. -- **Remediation:** Restructure PSE sections. -- **Actionable:** yes
17. **[MAJOR] D6-6b-001** -- Imports inconsistent with actual test framework. -- **Remediation:** Replace testify imports with Ginkgo/Gomega. -- **Actionable:** yes
18. **[MINOR] D2-2c-001** -- "table-driven" type declaration mismatch with actual structure. -- **Remediation:** Restructure or reclassify. -- **Actionable:** yes
19. **[MINOR] D4-4b-003** -- Scenario 010 same step issue as 003. -- **Remediation:** Fix alongside 003. -- **Actionable:** yes
20. **[MINOR] D4-4f-001** -- All P0 scenarios have only P0 assertions. -- **Remediation:** Add secondary P1 assertions. -- **Actionable:** yes
21. **[MINOR] D4.5-4.5b-002** -- Internal package import in STD YAML. -- **Remediation:** Remove project-internal imports. -- **Actionable:** yes
22. **[MINOR] D5-5c-002** -- TS-GH-51-019 PSE Steps contain verification. -- **Remediation:** Move to Expected. -- **Actionable:** yes
23. **[MINOR] D6-6c-001** -- No code_structure field on any scenario. -- **Remediation:** Add code_structure field. -- **Actionable:** yes

---

## Confidence Notes

| Factor | Status |
|:-------|:-------|
| STD YAML parseable | YES |
| STP file available | YES |
| Go stubs present | YES (6 files, 19 tests) |
| Python stubs present | NO |
| Pattern library available | NO |
| All scenarios reviewed | YES |
| Project review rules loaded | YES (dynamic extraction, default_ratio=0.65) |

**Confidence rationale:** Confidence is MEDIUM. The STD YAML is parseable and the STP is available for full traceability review. Go stubs are present and complete (19/19 tests). However, no pattern library is available (Dimension 3 review is limited to presence checks), no Python stubs exist (expected since project uses Go/Ginkgo for Tier 1), and review rules have a default_ratio of 0.65, meaning 65% of rules are using generic defaults. Review precision is reduced for pattern-specific and project-specific checks. Consider adding a project-specific review_rules.yaml or enabling repo_files_fetch to improve review precision.
