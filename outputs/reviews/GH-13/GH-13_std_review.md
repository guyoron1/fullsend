# STD Review Report: GH-13

**Reviewed:**
- STD YAML: outputs/std/GH-13/GH-13_test_description.yaml
- STP Source: outputs/stp/GH-13/GH-13_test_plan.md
- Go Stubs: outputs/std/GH-13/go-tests/ (3 files, 11 test functions)
- Python Stubs: N/A

**Date:** 2026-06-15
**Reviewer:** QualityFlow Automated Review (v1.1.0)
**Review Rules Schema:** 1.1.0

---

## Verdict: NEEDS_REVISION

## Summary

| Metric | Value |
|:-------|:------|
| Dimensions reviewed | 7/7 |
| Critical findings | 1 |
| Major findings | 6 |
| Minor findings | 5 |
| Actionable findings | 10 |
| Confidence | LOW |
| Weighted score | 72 |

## Traceability Summary

| Metric | Value |
|:-------|:------|
| STP scenarios | 11 |
| STD scenarios | 11 |
| Forward coverage (STP to STD) | 11/11 (100%) |
| Reverse coverage (STD to STP) | 11/11 (100%) |
| Orphan STD scenarios | 0 |
| Missing STD scenarios | 0 |

---

## Findings by Dimension

### Dimension 1: STP-STD Traceability (Weight: 30%)

**Score: 88/100**

#### 1a. Forward Traceability (STP to STD)

All 11 STP scenarios in Section III map to corresponding STD scenarios. The mapping is verified below:

| STP Scenario | STD test_id | Keyword Overlap | Match |
|:-------------|:------------|:----------------|:------|
| Cross-references valid (P0) | TS-GH-13-001 | HIGH (cross-reference, valid, ADR) | Full |
| Tool allowlist hook claims (P0) | TS-GH-13-002 | HIGH (tool allowlist, hook, filtering) | Full |
| SSRF validator coverage (P0) | TS-GH-13-003 | HIGH (SSRF, validator, coverage, Bash, WebFetch) | Full |
| README index entry (P1) | TS-GH-13-004 | HIGH (README, index, entry, link) | Full |
| Document structure format (P1) | TS-GH-13-005 | HIGH (document, structure, format) | Full |
| Attack scenarios distinct (P1) | TS-GH-13-006 | HIGH (attack, scenarios, distinct, threat) | Full |
| Harness architecture references (P1) | TS-GH-13-007 | HIGH (harness, architecture, references) | Full |
| Harness initialization flow (P2) | TS-GH-13-008 | HIGH (harness, initialization, flow, injection) | Full |
| Sensitive information disclosure (P1) | TS-GH-13-009 | HIGH (sensitive, disclosure, endpoint, credential) | Full |
| Open questions completeness (P2) | TS-GH-13-010 | HIGH (open questions, actionable) | Full |
| Markdown links and formatting (P2) | TS-GH-13-011 | HIGH (markdown, links, formatting) | Full |

All requirement_ids are "GH-13" which matches the STP.

#### 1b. Reverse Traceability (STD to STP)

All 11 STD scenarios trace back to STP Section III entries. No orphan scenarios found.

#### 1c. Count Consistency

| Metadata Field | Claimed | Actual | Status |
|:---------------|:--------|:-------|:-------|
| total_scenarios | 11 | 11 | PASS |
| functional_count | 11 | 11 | PASS |
| e2e_count | 0 | 0 | PASS |
| p0_count | 3 | 3 | PASS |
| p1_count | 5 | 5 | PASS |
| p2_count | 3 | 3 | PASS |

Note: The metadata uses `functional_count` and `e2e_count` rather than `tier_1_count` and `tier_2_count`. This is acceptable given the tier naming convention used (see Dimension 2 finding D2-2b-001).

#### 1d. STP Reference

- `stp_reference.file`: "outputs/stp/GH-13/GH-13_test_plan.md" -- valid path, file exists. PASS.

#### 1e. Priority-Testability Consistency

All P0 scenarios (001, 002, 003) are fully testable via file system and codebase analysis. No contradictions found.

#### Findings

- **D1-1a-001** (severity: MAJOR, dimension: STP-STD Traceability)
  - **Description:** Tier naming convention uses "Functional" instead of the v2.1-enhanced expected values "Tier 1" or "Tier 2". While this is internally consistent between STP and STD, it deviates from the v2.1-enhanced schema specification.
  - **Evidence:** All 11 scenarios have `tier: "Functional"`. The STP also uses "[Functional]" tier labels. The v2.1-enhanced spec expects "Tier 1" or "Tier 2".
  - **Remediation:** Map "Functional" to "Tier 1" for all scenarios, since the STP consistently uses Functional testing tier. This is the standard mapping per QualityFlow conventions.
  - **Actionable:** true

---

### Dimension 2: STD YAML Structure (Weight: 20%)

**Score: 85/100**

#### 2a. Document-Level Structure

| Check | Status |
|:------|:-------|
| document_metadata section exists | PASS |
| std_version is "2.1-enhanced" | PASS |
| code_generation_config section exists | PASS |
| code_generation_config.std_version is "2.1-enhanced" | PASS |
| code_generation_config.package_name | PASS ("tests") |
| common_preconditions section exists | PASS |
| scenarios array exists and non-empty | PASS |

#### 2b. Per-Scenario Required Fields

All 11 scenarios were checked for required fields:

| Field | Present in All 11 | Notes |
|:------|:-------------------|:------|
| scenario_id | YES | Sequential 1-11 |
| test_id | YES | Format TS-GH-13-XXX |
| tier | YES | All "Functional" (see D1-1a-001) |
| priority | YES | P0/P1/P2 distributed |
| requirement_id | YES | All "GH-13" |
| patterns | YES | Primary + secondary for all |
| variables | YES | closure_scope present |
| test_structure | YES | type/describe/context/it |
| code_structure | YES | Go function templates |
| test_objective | YES | title/what/why/acceptance_criteria |
| test_data | YES | All have resource_definitions + api_endpoints |
| test_steps | YES | setup + test_execution + cleanup |
| assertions | YES | At least 2 per scenario |

Test ID format verification: All test IDs follow `TS-GH-13-{NUM:03d}` pattern. PASS.

No duplicate scenario_ids or test_ids found. PASS.

#### 2c. v2.1-Specific Checks

Framework is `testing` (not ginkgo-v2), so Ginkgo-specific checks (Ordered decorator, BeforeAll, ExpectWithOffset, `:=` vs `=`) do NOT apply.

No Tier 2 / Python scenarios exist, so Python-specific checks do not apply.

#### Findings

- **D2-2b-001** (severity: MAJOR, dimension: STD YAML Structure)
  - **Description:** Tier values use "Functional" instead of the v2.1-enhanced specification values of "Tier 1" or "Tier 2". The v2.1-enhanced schema defines tier as an enum of "Tier 1" and "Tier 2" only.
  - **Evidence:** All 11 scenarios have `tier: "Functional"`. Expected: `tier: "Tier 1"` for functional testing scenarios.
  - **Remediation:** Replace `tier: "Functional"` with `tier: "Tier 1"` in all 11 scenarios. Update metadata fields from `functional_count`/`e2e_count` to `tier_1_count`/`tier_2_count`.
  - **Actionable:** true

- **D2-2b-002** (severity: MINOR, dimension: STD YAML Structure)
  - **Description:** The `classification` field appears in all scenarios but is not part of the v2.1-enhanced required field set. This is not harmful but represents non-standard schema extension.
  - **Evidence:** Each scenario includes `classification: { test_type, scope, automation_approach }`.
  - **Remediation:** No action required. The field is informational and does not affect code generation.
  - **Actionable:** false

---

### Dimension 3: Pattern Matching Correctness (Weight: 10%)

**Score: 90/100**

No pattern library (`tier1_patterns.yaml`) is available, so pattern validation uses general heuristics only.

| Scenario | Primary Pattern | Helpers | Decorators | Status |
|:---------|:----------------|:--------|:-----------|:-------|
| 1 (TS-GH-13-001) | doc-cross-reference-validation | filepath | [] | PASS |
| 2 (TS-GH-13-002) | codebase-claim-verification | os/exec | [] | PASS |
| 3 (TS-GH-13-003) | codebase-claim-verification | os/exec | [] | PASS |
| 4 (TS-GH-13-004) | doc-index-validation | strings | [] | PASS |
| 5 (TS-GH-13-005) | doc-structure-validation | strings | [] | PASS |
| 6 (TS-GH-13-006) | content-analysis-validation | strings | [] | PASS |
| 7 (TS-GH-13-007) | codebase-claim-verification | os/exec | [] | PASS |
| 8 (TS-GH-13-008) | codebase-claim-verification | os/exec | [] | PASS |
| 9 (TS-GH-13-009) | security-content-review | strings | [] | PASS |
| 10 (TS-GH-13-010) | content-analysis-validation | strings | [] | PASS |
| 11 (TS-GH-13-011) | doc-cross-reference-validation | filepath, regexp | [] | PASS |

All pattern assignments are reasonable for the test objectives. No mismatches detected.

#### Findings

- **D3-3c-001** (severity: MINOR, dimension: Pattern Matching Correctness)
  - **Description:** All 11 scenarios have empty `decorators: []`. While this is acceptable for a `testing` framework (no Ginkgo-style decorators), tier markers are typically expected.
  - **Evidence:** Every scenario has `decorators: []`.
  - **Remediation:** Consider adding a tier marker if the project uses decorator-based tier classification. Not required for the `testing` framework.
  - **Actionable:** false

---

### Dimension 4: Test Step Quality (Weight: 15%)

**Score: 82/100**

| Scenario | Setup | Execution | Cleanup | Assertions | Status |
|:---------|:------|:----------|:--------|:-----------|:-------|
| 1 (TS-GH-13-001) | 2 | 3 | 0 | 2 | PASS |
| 2 (TS-GH-13-002) | 2 | 3 | 0 | 2 | PASS |
| 3 (TS-GH-13-003) | 2 | 3 | 0 | 3 | PASS |
| 4 (TS-GH-13-004) | 1 | 3 | 0 | 2 | PASS |
| 5 (TS-GH-13-005) | 2 | 3 | 0 | 2 | PASS |
| 6 (TS-GH-13-006) | 1 | 3 | 0 | 2 | PASS |
| 7 (TS-GH-13-007) | 1 | 3 | 0 | 2 | PASS |
| 8 (TS-GH-13-008) | 1 | 3 | 0 | 2 | PASS |
| 9 (TS-GH-13-009) | 1 | 3 | 0 | 2 | PASS |
| 10 (TS-GH-13-010) | 1 | 3 | 0 | 2 | PASS |
| 11 (TS-GH-13-011) | 1 | 3 | 0 | 2 | PASS |

#### 4a. Step Completeness

All 11 scenarios have empty cleanup arrays (`cleanup: []`). For document-validation tests that only read files and do not create resources, this is acceptable. No resource creation occurs in any scenario -- these tests are purely read-only operations (reading markdown files, grepping source code). No cleanup needed.

#### 4b. Step Quality

Steps are generally specific and actionable. Each test_execution step has a validation field.

#### 4c. Logical Flow

All scenarios follow a logical read-then-analyze-then-verify flow. No circular dependencies or resource reference issues.

#### 4f. Assertion Quality

All scenarios have at least 2 assertions with specific descriptions and conditions. Priority distribution is appropriate (P0 for critical scenarios, P1/P2 for others).

#### Findings

- **D4-4b-001** (severity: MINOR, dimension: Test Step Quality)
  - **Description:** Several test_execution steps use high-level pseudo-commands rather than concrete Go code references. While acceptable at the design phase, this reduces code generation precision.
  - **Evidence:** Scenario 6 TEST-02: `command: "Compare scenario descriptions for overlap"`, Scenario 10 TEST-02: `command: "Check each question for specificity (references to scenarios/approaches)"`.
  - **Remediation:** Consider adding more specific algorithmic descriptions for content-analysis steps (e.g., specify what string matching or NLP heuristic would be used).
  - **Actionable:** true

- **D4-4b-002** (severity: MINOR, dimension: Test Step Quality)
  - **Description:** Scenario 1 (TS-GH-13-001) and Scenario 11 (TS-GH-13-011) have significant overlap in test scope. Both verify that relative markdown links resolve to existing files.
  - **Evidence:** Scenario 1 test_objective: "Verify cross-references...are valid and link targets exist". Scenario 11 test_objective: "Verify all relative markdown links resolve and markdown formatting renders correctly". The link validation portion overlaps.
  - **Remediation:** Clarify the distinct scope of each: Scenario 1 focuses specifically on named cross-references (security-threat-model.md, agent-architecture.md, ADRs) while Scenario 11 is a comprehensive markdown link and formatting check. Consider documenting the scope boundary more explicitly.
  - **Actionable:** true

---

### Dimension 4.5: STD Content Policy (Weight: 10%)

**Score: 40/100**

#### 4.5a. Banned Content in STD YAML and Stub Files

##### STD YAML

- **D4.5-4.5a-001** (severity: CRITICAL, dimension: STD Content Policy)
  - **Description:** The `document_metadata` section contains a `related_prs` field with full PR URLs. PR URLs are implementation artifacts that belong in the STP (which references them in Section I), not in the STD. The STD describes what to test, not what code changed.
  - **Evidence:**
    ```yaml
    related_prs:
      - repo: "fullsend-ai/fullsend"
        pr_number: 2011
        url: "https://github.com/fullsend-ai/fullsend/pull/2011"
        title: "MCP Configuration Drift Problem Document"
        merged: false
      - repo: "guyoron1/fullsend"
        pr_number: 13
        url: "https://github.com/guyoron1/fullsend/pull/13"
        title: "MCP Configuration Drift Problem Document (fork)"
        merged: false
    ```
  - **Remediation:** Remove the entire `related_prs` section from `document_metadata`. PR references belong in the STP metadata, not the STD.
  - **Actionable:** true

##### Stub Files

No PR URLs, branch names, commit SHAs, or developer names found in any of the 3 stub files. PASS.

STP reference in module-level comments uses the correct STP file path format. PASS.

#### 4.5b. No Implementation Details in Stubs

All stub function bodies contain only `t.Skip(...)` with the correct pending marker format. No implementation code, fixture implementations, or concrete API calls found. PASS.

#### 4.5c. Test Environment Separation

No infrastructure creation, cluster setup, or feature gate enablement code found in stubs. PASS.

---

### Dimension 5: PSE Docstring Quality (Weight: 10%)

**Score: 80/100**

**Go Stubs:** 3 files reviewed, 11 test functions total.

#### File: cross_reference_validation_stubs_test.go (3 tests)

| Test Function | test_id | PSE Present | Quality |
|:-------------|:--------|:------------|:--------|
| TestCrossReferencesValid | TS-GH-13-001 | YES | Good |
| TestReadmeIndexEntry | TS-GH-13-004 | YES | Good |
| TestMarkdownLinksAndFormatting | TS-GH-13-011 | YES | Good |

#### File: codebase_claim_verification_stubs_test.go (4 tests)

| Test Function | test_id | PSE Present | Quality |
|:-------------|:--------|:------------|:--------|
| TestToolAllowlistHookClaims | TS-GH-13-002 | YES | Good |
| TestSSRFValidatorCoverageClaims | TS-GH-13-003 | YES | Good |
| TestHarnessArchitectureReferences | TS-GH-13-007 | YES | Good |
| TestHarnessInitializationFlowConsistency | TS-GH-13-008 | YES | Good |

#### File: content_analysis_validation_stubs_test.go (4 tests)

| Test Function | test_id | PSE Present | Quality |
|:-------------|:--------|:------------|:--------|
| TestDocumentStructureFormat | TS-GH-13-005 | YES | Good |
| TestAttackScenariosDistinct | TS-GH-13-006 | YES | Good |
| TestNoSensitiveDisclosure | TS-GH-13-009 | YES | Good |
| TestOpenQuestionsComplete | TS-GH-13-010 | YES | Good |

All 11 test functions have PSE docstrings with Preconditions, Steps, and Expected sections. All test_ids are embedded in the t.Skip message in the correct format.

#### Findings

- **D5-5a-001** (severity: MAJOR, dimension: PSE Docstring Quality)
  - **Description:** Stub file marker comments use `tier1` but the STD YAML uses `tier: "Functional"`. The markers should be consistent with the tier naming used in the STD.
  - **Evidence:** All 3 stub files contain `Markers: - tier1` in the module-level comment block, but the YAML says `tier: "Functional"`.
  - **Remediation:** Align stub markers with STD tier naming. If tier is corrected to "Tier 1" per D2-2b-001, then `tier1` markers are correct. If "Functional" is retained, markers should say `functional`.
  - **Actionable:** true

- **D5-5c-001** (severity: MAJOR, dimension: PSE Docstring Quality)
  - **Description:** Some PSE Preconditions sections include items that are setup actions rather than pre-existing state conditions. For example, "Document's claims about ToolAllowlistPreToolHook extracted" describes an action (extracting claims), not a precondition (the document exists and contains claims about the hook).
  - **Evidence:** codebase_claim_verification_stubs_test.go, TestToolAllowlistHookClaims precondition: "Document's claims about ToolAllowlistPreToolHook extracted". This describes a completed action, not a state condition.
  - **Remediation:** Rephrase action-like preconditions as state conditions. Example: "Document's claims about ToolAllowlistPreToolHook extracted" should be "mcp-config-drift.md contains claims about ToolAllowlistPreToolHook filtering mechanism".
  - **Actionable:** true

- **D5-5c-002** (severity: MAJOR, dimension: PSE Docstring Quality)
  - **Description:** PSE Preconditions in the per-test docstrings partially duplicate the module-level preconditions. For example, "mcp-config-drift.md content loaded" appears as a per-test precondition in multiple tests, but "docs/problems/mcp-config-drift.md exists in the PR branch" is already stated at the module level. The per-test precondition describes a setup action ("content loaded") rather than a unique precondition.
  - **Evidence:** Scenarios 5, 6, 9, 10, 11 all have "mcp-config-drift.md content loaded" as a precondition when the module-level already covers "docs/problems/mcp-config-drift.md exists in the PR branch".
  - **Remediation:** Remove duplicated preconditions from per-test PSE blocks. Keep only test-specific preconditions that go beyond the module-level shared preconditions.
  - **Actionable:** true

**Python Stubs:** N/A (not generated for this project)

---

### Dimension 6: Code Generation Readiness (Weight: 5%)

**Score: 85/100**

#### 6a. Variable Declarations

All closure_scope variables use valid Go types (string, []string). Initialization references (TestSetup, TestExecution) are valid lifecycle stages. PASS.

#### 6b. Import Completeness

`code_generation_config.imports` includes:
- Standard: context, os, path/filepath, strings, testing
- Test framework: testify/assert, testify/require
- Project: internal/config, internal/forge

Scenarios reference helpers: filepath, strings, os/exec, regexp.

| Helper Used | In Imports | Status |
|:------------|:-----------|:-------|
| filepath | path/filepath (standard) | PASS |
| strings | strings (standard) | PASS |
| regexp | NOT in imports | WARN |
| os/exec | NOT in imports | WARN |

#### Findings

- **D6-6b-001** (severity: MAJOR, dimension: Code Generation Readiness)
  - **Description:** Two helper libraries referenced in scenario patterns are missing from `code_generation_config.imports`: `regexp` (used by scenario 11) and `os/exec` (used by scenarios 2, 3, 7, 8).
  - **Evidence:** Scenario 11 helpers_required includes `regexp` with functions `Compile, FindAllStringSubmatch`. Scenarios 2,3,7,8 helpers_required include `os/exec` with function `Command`. Neither `regexp` nor `os/exec` appears in `code_generation_config.imports.standard`.
  - **Remediation:** Add `"regexp"` and `"os/exec"` to `code_generation_config.imports.standard`.
  - **Actionable:** true

#### 6c. Code Structure Validity

All code_structure blocks contain syntactically reasonable Go function templates with the `testing` framework pattern (`func TestXxx(t *testing.T)`). PASS.

#### 6d. Timeout Appropriateness

All scenarios are read-only file system and grep operations. The default timeout of "30s" is appropriate. No long-running operations exist. PASS.

---

## Recommendations

1. **[CRITICAL]** Remove `related_prs` from `document_metadata` -- PR URLs are implementation artifacts that do not belong in the STD. -- **Remediation:** Delete the entire `related_prs` block (lines 14-24 of the YAML). -- **Actionable:** yes

2. **[MAJOR]** Replace `tier: "Functional"` with `tier: "Tier 1"` in all 11 scenarios and update metadata fields accordingly. -- **Remediation:** Change all `tier: "Functional"` to `tier: "Tier 1"`. Rename `functional_count` to `tier_1_count` and `e2e_count` to `tier_2_count` in metadata. -- **Actionable:** yes

3. **[MAJOR]** Add missing imports `regexp` and `os/exec` to `code_generation_config.imports.standard`. -- **Remediation:** Append `"regexp"` and `"os/exec"` to the standard imports list. -- **Actionable:** yes

4. **[MAJOR]** Fix PSE precondition phrasing from action-descriptions to state conditions in codebase_claim_verification_stubs_test.go. -- **Remediation:** Rephrase "extracted" preconditions as state assertions (e.g., "Document contains claims about X"). -- **Actionable:** yes

5. **[MAJOR]** Remove duplicated preconditions from per-test PSE blocks that already appear at module level. -- **Remediation:** Keep only test-specific preconditions; rely on module-level preconditions for shared state. -- **Actionable:** yes

6. **[MAJOR]** Align stub marker annotations with STD tier naming convention. -- **Remediation:** Once tier naming is resolved (Recommendation 2), ensure stub markers match. -- **Actionable:** yes

7. **[MINOR]** Clarify scope boundary between Scenario 1 (TS-GH-13-001) and Scenario 11 (TS-GH-13-011) to avoid test overlap. -- **Remediation:** Add explicit scope notes distinguishing named cross-reference validation from comprehensive link checking. -- **Actionable:** yes

8. **[MINOR]** Add more specific algorithmic descriptions for content-analysis test steps (scenarios 6, 10). -- **Remediation:** Replace high-level pseudo-commands with concrete algorithmic descriptions. -- **Actionable:** yes

9. **[MINOR]** The non-standard `classification` field in all scenarios is harmless but represents schema extension. -- **Remediation:** No action required. -- **Actionable:** no

10. **[MINOR]** Empty decorator arrays are acceptable for the `testing` framework but could include tier markers if the project adopts decorator-based classification. -- **Remediation:** No action required unless project conventions change. -- **Actionable:** no

11. **[MINOR]** Consider documenting why Scenario 1 and Scenario 11 both validate markdown links but at different specificity levels. -- **Remediation:** Add a note in scenario 11 that it is a superset validation that includes formatting checks beyond link resolution. -- **Actionable:** yes

---

## Confidence Notes

| Factor | Status |
|:-------|:-------|
| STD YAML parseable | YES |
| STP file available | YES |
| Go stubs present | YES (3 files, 11 tests) |
| Python stubs present | NO (not expected) |
| Pattern library available | NO |
| All scenarios reviewed | YES |
| Project review rules loaded | NO (dynamic extraction, default_ratio ~0.65) |

**Confidence rationale:** Confidence is LOW because (1) no pattern library (`tier1_patterns.yaml`) is available for pattern validation, (2) review rules are using generic defaults with a `default_ratio` of 0.65 (65% of rules from defaults), and (3) no `review_rules.yaml` static override exists. Review precision is reduced: 65% of rules using generic defaults. Consider adding project-specific `review_rules.yaml` or enabling `repo_files_fetch` to improve review precision.
