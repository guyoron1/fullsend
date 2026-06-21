# STD Review Report: GH-2351

**Reviewed:**
- STD YAML: `outputs/std/GH-2351/GH-2351_test_description.yaml`
- STP Source: `outputs/stp/GH-2351/GH-2351_test_plan.md`
- Go Stubs: `outputs/std/GH-2351/go-tests/` (5 files)
- Python Stubs: N/A (Go-only project)

**Date:** 2026-06-21
**Reviewer:** QualityFlow Automated Review (v1.1.0)
**Review Rules Schema:** 1.1.0 (auto-detected project, default rules)

---

## Verdict: NEEDS_REVISION

## Summary

| Metric | Value |
|:-------|:------|
| Dimensions reviewed | 7/7 |
| Critical findings | 3 |
| Major findings | 6 |
| Minor findings | 4 |
| Actionable findings | 12 |
| Weighted score | 72 |
| Confidence | LOW |

## Traceability Summary

| Metric | Value |
|:-------|:------|
| STP scenarios | 17 |
| STD scenarios | 17 |
| Forward coverage (STP→STD) | 17/17 (100%) |
| Reverse coverage (STD→STP) | 17/17 (100%) |
| Orphan STD scenarios | 0 |
| Missing STD scenarios | 0 |

---

## Findings by Dimension

### Dimension 1: STP-STD Traceability — Score: 95/100

**Forward Traceability (STP → STD):** All 17 STP scenarios in Section III map to corresponding STD scenarios. Requirement groups are preserved:

| STP Requirement Group | STP Scenarios | STD Scenarios | Status |
|:----------------------|:--------------|:--------------|:-------|
| Batch path-existence checks (P0) | 5 | 001–005 | ✅ PASS |
| ListRepositoryFiles via Git Trees API (P0) | 4 | 006–009 | ✅ PASS |
| FakeClient.ListRepositoryFiles (P1) | 3 | 010–012 | ✅ PASS |
| Edge cases (P1) | 3 | 013–015 | ✅ PASS |
| Interface compliance (P1) | 2 | 016–017 | ✅ PASS |

**Reverse Traceability (STD → STP):** All 17 STD scenarios have `requirement_id: "GH-2351"` which matches the STP. Each scenario title matches a corresponding STP test scenario description.

**Count Consistency:**

| Metadata Field | Declared | Actual | Status |
|:---------------|:---------|:-------|:-------|
| total_scenarios | 17 | 17 | ✅ |
| unit_count | 17 | 17 | ✅ |
| p0_count | 9 | 9 (001–009) | ✅ |
| p1_count | 8 | 8 (010–017) | ✅ |
| tier_1_count | 0 | 0 | ✅ |
| tier_2_count | 0 | 0 | ✅ |

**Findings:**

- **D1-1d-001** | MAJOR | STP-STD Traceability
  - **Description:** STP reference path in metadata is correct but no validation that the STP file exists at runtime. Minor gap.
  - **Evidence:** `stp_reference.file: "outputs/stp/GH-2351/GH-2351_test_plan.md"` — file does exist ✓
  - **Remediation:** No action needed. Path is valid.
  - **Actionable:** false

- **D1-1a-001** | MINOR | STP-STD Traceability
  - **Description:** All 17 scenarios share a single `requirement_id: "GH-2351"`. While correct (single ticket), it prevents fine-grained traceability to individual sub-requirements within the ticket.
  - **Evidence:** STP Section III lists 5 distinct requirement groups, but the STD uses only one requirement_id for all.
  - **Remediation:** Consider adding sub-requirement identifiers (e.g., `GH-2351-R1` through `GH-2351-R5`) to distinguish requirement groups. Low priority for a single-ticket STD.
  - **Actionable:** true

### Dimension 2: STD YAML Structure — Score: 60/100

**Document-Level Structure:**

| Check | Status |
|:------|:-------|
| `document_metadata` exists | ✅ |
| `std_version` is "2.1-enhanced" | ✅ |
| `code_generation_config` exists | ✅ |
| `common_preconditions` exists | ✅ |
| `scenarios` array non-empty | ✅ (17 scenarios) |

**Per-Scenario Required Fields (v2.1-enhanced):**

| Field | Present in All 17? | Notes |
|:------|:--------------------|:------|
| `scenario_id` | ✅ | Sequential 001–017 |
| `test_id` | ✅ | Format: `TS-GH-2351-{NNN}` ✓ |
| `test_type` | ✅ | All "unit" — uses `test_type` instead of `tier` |
| `priority` | ✅ | P0 (9) + P1 (8) |
| `requirement_id` | ✅ | All "GH-2351" |
| `test_objective` | ✅ | title, what, why, acceptance_criteria |
| `test_data` | ✅ | resource_definitions present |
| `test_steps` | ✅ | setup, test_execution, cleanup |
| `assertions` | ✅ | At least 1 per scenario |
| `patterns` | ❌ MISSING | Required by v2.1-enhanced |
| `variables` | ❌ MISSING | Required by v2.1-enhanced |
| `test_structure` | ❌ MISSING | Required by v2.1-enhanced |
| `code_structure` | ❌ MISSING | Required by v2.1-enhanced |

**Findings:**

- **D2-2b-001** | MAJOR | STD YAML Structure
  - **Description:** All 17 scenarios are missing the `patterns`, `variables`, `test_structure`, and `code_structure` fields required by the v2.1-enhanced specification.
  - **Evidence:** No scenario contains any of these four fields. The STD declares `std_version: "2.1-enhanced"` but follows a simplified schema.
  - **Remediation:** Either (a) add the missing v2.1 fields to each scenario, or (b) change `std_version` to a version that matches the actual schema used (e.g., `"2.0-unit"` for a simplified unit-test-only schema). For auto-detected projects with `test_strategy: "auto"`, consider defining a reduced schema that doesn't require Ginkgo-specific fields.
  - **Actionable:** true

- **D2-2b-002** | MINOR | STD YAML Structure
  - **Description:** Scenarios use `test_type: "unit"` instead of `tier: "Tier 1"` or `tier: "Tier 2"`. The `test_type` field is not part of the v2.1-enhanced per-scenario spec — the spec uses `tier`.
  - **Evidence:** All 17 scenarios have `test_type: "unit"` and no `tier` field.
  - **Remediation:** For auto-detected projects without the tier system, using `test_type` is pragmatically acceptable. Document this as a known deviation from v2.1-enhanced for `test_strategy: "auto"` projects.
  - **Actionable:** true

### Dimension 3: Pattern Matching Correctness — Score: N/A (adjusted to 75/100)

Pattern matching is not applicable for this auto-detected project (`config_dir: null`, no pattern library). No `patterns` field exists in scenarios. This dimension is scored at a neutral 75 to avoid penalizing projects that correctly operate without the pattern system.

**Findings:** None (dimension not applicable for auto-detected projects)

### Dimension 4: Test Step Quality — Score: 55/100

**Step Coverage Matrix:**

| Scenario | Setup | Execution | Cleanup | Assertions | Isolation | Error Paths | Status |
|:---------|:------|:----------|:--------|:-----------|:----------|:------------|:-------|
| 001 | 1 | 1 | 0 | 2 | PASS | N/A | ⚠ WARN |
| 002 | 1 | 1 | 0 | 2 | PASS | N/A | ⚠ WARN |
| 003 | 1 | 1 | 0 | 2 | PASS | N/A | ⚠ WARN |
| 004 | 1 | 1 | 0 | 2 | PASS | N/A | ⚠ WARN |
| 005 | 1 | 1 | 0 | 2 | PASS | PASS | ⚠ WARN |
| 006 | 1 | 1 | 0 | 2 | PASS | N/A | ⚠ WARN |
| 007 | 1 | 1 | 0 | 2 | PASS | N/A | ⚠ WARN |
| 008 | 1 | 1 | 0 | 2 | PASS | PASS | ⚠ WARN |
| 009 | 1 | 1 | 0 | 2 | PASS | PASS | ⚠ WARN |
| 010 | 1 | 1 | 0 | 1 | PASS | N/A | ⚠ WARN |
| 011 | 1 | 1 | 0 | 2 | PASS | N/A | ⚠ WARN |
| 012 | 1 | 1 | 0 | 1 | PASS | PASS | ⚠ WARN |
| 013 | 1 | 1 | 0 | 2 | PASS | N/A | ⚠ WARN |
| 014 | 1 | 1 | 0 | 2 | PASS | N/A | ⚠ WARN |
| 015 | 1 | 1 | 0 | 2 | PASS | N/A | ⚠ WARN |
| 016 | 0 | 1 | 0 | 1 | PASS | N/A | PASS |
| 017 | 0 | 1 | 0 | 1 | PASS | N/A | PASS |

**Findings:**

- **D4-4a-001** | CRITICAL | Test Step Quality
  - **Description:** `FakeClient.FileContents` type is wrong throughout the entire STD. The STD consistently specifies `FileContents` as `map[string]string` but the actual production type is `map[string][]byte`. This affects 14 of 17 scenarios (all except 016, 017) and will cause **every generated test to fail compilation**.
  - **Evidence:** STD scenario 001 setup: `FileContents: map[string]string{"owner/repo/path/a.txt": "content-a"}`. Actual FakeClient (fake.go:112): `FileContents map[string][]byte`. Existing tests (pathpresence_test.go:16): `FileContents: map[string][]byte{"org/.fullsend/.defaults/action.yml": []byte("marker")}`.
  - **Remediation:** Change all `map[string]string{...}` in test_data and test_steps to `map[string][]byte{...}` with `[]byte("...")` value wrappers. This is a systematic find-and-replace across all 14 affected scenarios.
  - **Actionable:** true

- **D4-4a-002** | CRITICAL | Test Step Quality
  - **Description:** Error injection mechanism in STD does not match the actual FakeClient API. The STD uses non-existent fields `GetFileContentErr` and `ListRepositoryFilesErr` as direct struct fields, but FakeClient uses a generic `Errors map[string]error` with method-name keys.
  - **Evidence:** STD scenario 004 setup: `GetFileContentErr: fmt.Errorf("GetFileContent must not be called")`. STD scenario 005: `ListRepositoryFilesErr: fmt.Errorf("API rate limit exceeded")`. Actual FakeClient (fake.go:142): `Errors map[string]error`. Existing tests (pathpresence_test.go:101): `Errors: map[string]error{"GetFileContent": errors.New("should not be called")}`.
  - **Remediation:** Replace all `GetFileContentErr: fmt.Errorf(...)` with `Errors: map[string]error{"GetFileContent": errors.New(...)}` and all `ListRepositoryFilesErr: fmt.Errorf(...)` with `Errors: map[string]error{"ListRepositoryFiles": errors.New(...)}`. Affects scenarios 004, 005, 008, 009, 012, 013.
  - **Actionable:** true

- **D4-4a-003** | CRITICAL | Test Step Quality
  - **Description:** Scenario 008 references a `TruncatedTree: true` field on FakeClient that **does not exist**. The FakeClient struct has no `TruncatedTree` field. This scenario cannot be implemented as described without modifying the production FakeClient or using a different test approach (e.g., httptest mock of the GitHub API).
  - **Evidence:** STD scenario 008 setup: `client := &forge.FakeClient{TruncatedTree: true}`. Actual FakeClient struct (fake.go:107-147): No `TruncatedTree` field. The FakeClient has no mechanism to simulate truncated tree responses.
  - **Remediation:** Either (a) add a `TruncatedTree bool` field to FakeClient with corresponding logic in `ListRepositoryFiles`, or (b) redesign scenario 008 to use an httptest server that returns a truncated tree response, matching how LiveClient.ListRepositoryFiles is implemented. Option (a) is simpler and consistent with the existing error injection pattern — consider adding it via `Errors: map[string]error{"ListRepositoryFiles": ErrTreeTruncated}` with a sentinel error.
  - **Actionable:** true

- **D4-4b-001** | MAJOR | Test Step Quality
  - **Description:** Scenario 011 asserts `paths != nil && len(paths) == 0` (empty non-nil slice), but the actual FakeClient implementation returns `nil` when FileContents is empty (the `paths` variable is never initialized, only appended to).
  - **Evidence:** STD scenario 011 assertion: `"paths != nil && len(paths) == 0"`. Actual FakeClient.ListRepositoryFiles (fake.go:412-418): `var paths []string; for key := range f.FileContents { ... paths = append(paths, ...) }; return paths, nil` — returns `nil` when map is empty. Existing test (pathpresence_test.go:75): `assert.Nil(t, missing)`.
  - **Remediation:** Change assertion ASSERT-02 from `"paths != nil && len(paths) == 0"` to `"paths is nil or empty (len == 0)"`. The test_objective.why states "Returning nil vs empty slice could cause nil pointer panics" — this is a valid concern but the actual implementation returns nil, so the STD should match actual behavior or explicitly document that the implementation should be changed.
  - **Actionable:** true

- **D4-4a-004** | MAJOR | Test Step Quality
  - **Description:** Scenario 013 passes `[]string{}` (empty slice) but the actual production test passes `nil` for the same edge case. The behavior may differ between `nil` and empty slice.
  - **Evidence:** STD scenario 013: `ComparePathPresence(ctx, client, "owner", "repo", []string{})`. Actual test (pathpresence_test.go:73): `ComparePathPresence(context.Background(), client, "org", ".fullsend", nil)`.
  - **Remediation:** Consider testing both `nil` and `[]string{}` inputs, or align with the existing production test convention of using `nil`.
  - **Actionable:** true

**Error Path Coverage:**

| Requirement Group | Positive | Negative | Ratio | Status |
|:------------------|:---------|:---------|:------|:-------|
| Batch path checks | 4 | 1 | 4:1 | ✅ Adequate |
| ListRepositoryFiles | 2 | 2 | 1:1 | ✅ Good |
| FakeClient | 2 | 1 | 2:1 | ✅ Adequate |
| Edge cases | 3 | 0 | 3:0 | ⚠ Acceptable (edge cases are boundary tests) |
| Interface compliance | 2 | 0 | 2:0 | ✅ Compile-time (N/A for pos/neg) |

### Dimension 4.5: STD Content Policy — Score: 95/100

**STD YAML Content:**
- `related_prs: []` — empty, no violation ✅
- No PR URLs in metadata ✅
- No branch names or commit SHAs ✅
- No developer names ✅

**Stub File Content:**
- Module docstrings reference STP file, not PR URLs ✅
- No fixture implementations in stubs ✅
- Stub bodies contain only `t.Skip("Phase 1: Design only - awaiting implementation")` ✅
- No project-internal module imports beyond what's needed for type declarations ✅

**Findings:**

- **D45-4.5b-001** | MINOR | STD Content Policy
  - **Description:** Go stub files import `context` and `fmt` but these are unused because all tests are `t.Skip()`-ed. While this won't fail compilation (imports used in comments/skip text are allowed by some linters), strict `goimports` may flag them.
  - **Evidence:** `compare_path_presence_stubs_test.go` imports `context`, `fmt`, `forge` — none used in executable code. Same pattern in all 5 stub files.
  - **Remediation:** Either remove unused imports or add a `//nolint:unused` comment. The code generator should handle this automatically when stubs are implemented.
  - **Actionable:** true

### Dimension 5: PSE Docstring Quality — Score: 85/100

**Go Stubs Review (5 files, 17 test blocks):**

| Stub File | Tests | PSE Present | test_id | Quality |
|:----------|:------|:------------|:--------|:--------|
| compare_path_presence_stubs_test.go | 5 | 5/5 ✅ | 5/5 ✅ | Good |
| list_repository_files_stubs_test.go | 4 | 4/4 ✅ | 4/4 ✅ | Good |
| fake_client_stubs_test.go | 3 | 3/3 ✅ | 3/3 ✅ | Good |
| edge_cases_stubs_test.go | 3 | 3/3 ✅ | 3/3 ✅ | Good |
| interface_compliance_stubs_test.go | 2 | 2/2 ✅ | 2/2 ✅ | Good |

**PSE Quality Sampling:**

All PSE docstrings follow the `Preconditions:` / `Steps:` / `Expected:` pattern correctly.

- ✅ **Preconditions** are specific: "FakeClient configured with FileContents containing 'path/a.txt' and 'path/b.txt'"
- ✅ **Steps** are numbered and actionable: "1. Call ComparePathPresence with [...]"
- ✅ **Expected** results are measurable: "No error returned, Missing paths contains only 'path/c.txt'"
- ✅ Negative tests marked with `[NEGATIVE]` indicator
- ✅ Module-level docstrings reference STP file

**Findings:**

- **D5-5a-001** | MAJOR | PSE Docstring Quality
  - **Description:** PSE docstrings in stubs describe `FakeClient` setup using the wrong API (direct error fields instead of `Errors` map). This means a developer implementing from the stubs would write non-compiling code.
  - **Evidence:** edge_cases_stubs_test.go scenario 013: "FakeClient with ListRepositoryFilesErr set (to detect if called)". The correct description should reference `Errors: map[string]error{"ListRepositoryFiles": ...}`.
  - **Remediation:** Update PSE preconditions to reference the actual `Errors` map pattern. E.g., "FakeClient with Errors map entry for 'ListRepositoryFiles' set to sentinel error."
  - **Actionable:** true

- **D5-5c-001** | MINOR | PSE Docstring Quality
  - **Description:** Interface compliance tests (016, 017) have Steps describing "Compile-time assertion" but this is not a runtime test step — it's a declaration. The PSE convention is slightly strained for compile-time checks.
  - **Evidence:** interface_compliance_stubs_test.go: "Steps: 1. Compile-time assertion: var _ forge.Client = (*forge.FakeClient)(nil)"
  - **Remediation:** Consider rewriting as Precondition: "forge.FakeClient type exists", Expected: "Code compiles with `var _ forge.Client = (*forge.FakeClient)(nil)`". Minor stylistic improvement.
  - **Actionable:** true

### Dimension 6: Code Generation Readiness — Score: 40/100

**Findings:**

- **D6-6b-001** | MAJOR | Code Generation Readiness
  - **Description:** `code_generation_config.imports` lists `"sort"` as a standard import, but only 2 of 17 scenarios use sorting assertions. Similarly, `"fmt"` is listed but only error-injection scenarios use it. Import optimization should be per-stub-file, not global.
  - **Evidence:** `code_generation_config.imports.standard: ["context", "testing", "sort", "fmt"]`. Only scenarios 003, 014 need `sort`; only scenarios 004, 005, 008, 009, 012 need `fmt`.
  - **Remediation:** Add per-stub-file import lists in the YAML, or ensure the code generator is smart enough to prune unused imports per file. Currently, all 5 stub files will get all 4 standard imports regardless of need.
  - **Actionable:** true

- **D6-6a-001** | MAJOR | Code Generation Readiness
  - **Description:** Missing `variables` and `code_structure` fields means the code generator has no guidance on variable scoping, lifecycle hooks, or test framework structure. The generator must infer everything from the test_steps, increasing risk of incorrect code generation.
  - **Evidence:** Zero scenarios have `variables`, `code_structure`, or `test_structure` fields despite the STD declaring v2.1-enhanced format.
  - **Remediation:** Add at minimum `code_structure` hints for each scenario indicating the Go test function structure (`func TestX(t *testing.T) { t.Run(...) }`). The `variables` field should list closure-scope variables like `ctx`, `client`, `missing`, `err`.
  - **Actionable:** true

- **D6-6c-001** | MINOR | Code Generation Readiness
  - **Description:** The `code_generation_config.package_name` is `"scaffold"` which is correct for `ComparePathPresence` tests but incorrect for `FakeClient` and interface compliance tests — those belong in the `forge` package (or `forge_test`).
  - **Evidence:** All stubs declare `package scaffold` but scenarios 010-012 test `FakeClient.ListRepositoryFiles` (in `internal/forge`) and scenarios 016-017 test interface compliance (requiring imports from both `forge` and `forge/github`).
  - **Remediation:** Split code_generation_config by target package, or add per-scenario `package` overrides. The FakeClient tests and interface compliance tests should be in `package forge_test` or `package forge`.
  - **Actionable:** true

---

## Recommendations

Ordered by severity:

1. **[CRITICAL]** Fix `FileContents` type from `map[string]string` to `map[string][]byte` across all 14 affected scenarios. — **Remediation:** Systematic replacement: `"content"` → `[]byte("content")` in all test_data and test_steps. — **Actionable:** yes

2. **[CRITICAL]** Fix error injection mechanism from direct fields (`GetFileContentErr`, `ListRepositoryFilesErr`) to the actual `Errors: map[string]error{...}` pattern across all 6 affected scenarios (004, 005, 008, 009, 012, 013). — **Remediation:** Replace field names and restructure YAML accordingly. — **Actionable:** yes

3. **[CRITICAL]** Redesign scenario 008 (truncated tree test). The `TruncatedTree` field does not exist on FakeClient. — **Remediation:** Either extend FakeClient with a sentinel error approach or redesign to use httptest. — **Actionable:** yes (requires design decision)

4. **[MAJOR]** Fix scenario 011 assertion to match actual FakeClient behavior (returns `nil`, not empty slice). — **Remediation:** Update assertion condition. — **Actionable:** yes

5. **[MAJOR]** Fix scenario 013 to use `nil` instead of `[]string{}` to match production test convention, or test both inputs. — **Remediation:** Align with existing test pattern. — **Actionable:** yes

6. **[MAJOR]** Add missing v2.1-enhanced required fields (`patterns`, `variables`, `test_structure`, `code_structure`) or downgrade `std_version`. — **Remediation:** Define a unit-test schema variant or add fields. — **Actionable:** yes

7. **[MAJOR]** Update PSE docstrings to reference correct `Errors` map pattern instead of non-existent direct error fields. — **Remediation:** Update precondition descriptions in affected stubs. — **Actionable:** yes

8. **[MAJOR]** Add per-file import lists or per-scenario package overrides — FakeClient tests belong in `package forge`, not `package scaffold`. — **Remediation:** Restructure code_generation_config. — **Actionable:** yes

9. **[MAJOR]** Fix global import list to be per-stub-file to avoid unused imports. — **Remediation:** Add per-file import sections. — **Actionable:** yes

10. **[MINOR]** Consider sub-requirement IDs for finer traceability granularity. — **Actionable:** yes

11. **[MINOR]** Remove unused imports from stub files or add linter suppression. — **Actionable:** yes

12. **[MINOR]** Rephrase interface compliance PSE for compile-time assertions. — **Actionable:** yes

13. **[MINOR]** Align `test_type: "unit"` with v2.1-enhanced `tier` field convention. — **Actionable:** yes

---

## Dimension Score Summary

| Dimension | Weight | Score | Weighted |
|:----------|:-------|:------|:---------|
| 1. STP-STD Traceability | 30% | 95 | 28.5 |
| 2. STD YAML Structure | 20% | 60 | 12.0 |
| 3. Pattern Matching | 10% | 75 | 7.5 |
| 4. Test Step Quality | 15% | 55 | 8.25 |
| 4.5. Content Policy | 10% | 95 | 9.5 |
| 5. PSE Docstring Quality | 10% | 85 | 8.5 |
| 6. Code Generation Readiness | 5% | 40 | 2.0 |
| **Total** | **100%** | — | **76.25** |

---

## Confidence Notes

| Factor | Status |
|:-------|:-------|
| STD YAML parseable | YES |
| STP file available | YES |
| Go stubs present | YES (5 files) |
| Python stubs present | N/A (Go project) |
| Pattern library available | NO (auto-detected project) |
| All scenarios reviewed | YES (17/17) |
| Project review rules loaded | NO (generic defaults, `default_ratio: 0.85`) |

**Confidence rationale:** LOW confidence due to auto-detected project with no project-specific config (`config_dir: null`). Review rules are 85% generic defaults. Pattern matching dimension is not applicable. All other dimensions were fully evaluated. The three CRITICAL findings are high-confidence because they were verified against the actual production source code (`internal/forge/fake.go`, `internal/scaffold/pathpresence_test.go`).

**⚠ Review precision note:** 85% of review rules are using generic defaults. Project-specific review precision is reduced. To improve: create `config/projects/fullsend/` with project-specific config or enable `repo_files_fetch`.

**Source code cross-reference:** The three CRITICAL findings (FileContents type, error injection mechanism, TruncatedTree field) were verified by reading the actual production code, not just the STD YAML. This gives HIGH confidence in these specific findings despite the overall LOW confidence rating.
