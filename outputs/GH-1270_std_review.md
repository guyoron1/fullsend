# STD Review Report — GH-1270

**Issue:** GH-1270 — Expand precommit-tools.yaml registry coverage
**Reviewed:** 2026-06-25
**Reviewer:** QualityFlow STD Reviewer (auto-detected project mode)
**Verdict:** APPROVED_WITH_FINDINGS

---

## Review Summary

The STD for GH-1270 is a well-structured, comprehensive test design covering the
pre-commit-tools subsystem (resolver, installer, registry merge, scaffold registration,
and script integration). It contains 36 scenarios across 12 Go stub files with full
1:1 traceability to the STP. The overall quality is high, with clear test objectives,
proper negative test coverage, and well-organized stub groupings.

**Weighted Score: 88/100**

| Dimension | Weight | Score | Weighted |
|:----------|:-------|:------|:---------|
| 1. STP-STD Traceability | 30% | 95 | 28.5 |
| 2. STD YAML Structure | 20% | 82 | 16.4 |
| 3. Pattern Matching Correctness | 10% | 90 | 9.0 |
| 4. Test Step Quality | 15% | 88 | 13.2 |
| 4.5. STD Content Policy | 10% | 90 | 9.0 |
| 5. PSE Docstring Quality | 10% | 85 | 8.5 |
| 6. Code Generation Readiness | 5% | 80 | 4.0 |
| **Total** | **100%** | | **88.6** |

---

## Dimension 1: STP-STD Traceability (95/100)

### Verified

- **36/36 scenarios** in STD map to **36/36 rows** in STP Section III
- All test_ids follow correct format: `TS-GH-1270-{001..036}` — sequential, no gaps
- All scenarios reference `requirement_id: "GH-1270"` — valid for single-issue test plan
- STP Section III requirement summaries match STD test objective titles
- All three STP-identified gaps (uv match_entry, tekwizely/golang warnings, shellcheck-py) have dedicated scenarios
- Coverage status uniformly `"NEW"` across all 36 scenarios — consistent with new feature

### No Findings

Full bidirectional traceability confirmed. Every STP row has a corresponding STD
scenario, and every STD scenario traces back to a valid STP entry.

---

## Dimension 2: STD YAML Structure (82/100)

### Verified

- `document_metadata` section present with all required fields
- `code_generation_config` properly specifies Go/testify framework
- `common_preconditions` section well-structured with infrastructure, fixtures, environment
- All 36 scenarios have required fields: `scenario_id`, `test_id`, `test_type`, `priority`, `mvp`, `requirement_id`, `coverage_status`, `test_objective`, `test_steps`, `assertions`
- `test_objective` consistently includes `title`, `what`, `why`, `acceptance_criteria`
- Scenario IDs sequential 1-36 with no gaps

### Findings

#### F-001: Metadata count discrepancy for unit_count and functional_count [MAJOR]

**Severity:** Major
**Actionable:** true

The `document_metadata` declares:
- `unit_count: 24` — actual count from scenarios: **26**
- `functional_count: 10` — actual count from scenarios: **8**

Scenarios 13 (pip pin), 14 (npm pin), and 15 (unsupported arch) are typed as `"unit"`
in their `test_type` field but were apparently counted as functional in the metadata.
The total (36) and other breakdowns (e2e: 2, P0: 2, P1: 20, P2: 14) are correct.

**Remediation:** Update `document_metadata.unit_count` to `26` and
`document_metadata.functional_count` to `8`.

#### F-002: Scenarios 12 and 25 have significant overlap [MINOR]

**Severity:** Minor
**Actionable:** true

Both scenarios test checksum verification failure causing a hard stop:
- Scenario 12: "Verify binary install with mismatched checksum exits non-zero"
- Scenario 25: "Verify sha256sum failure causes exit 1 (hard stop, not skip)"

The distinction is thin — scenario 12 checks "exits non-zero" while scenario 25
specifically checks "exit code is exactly 1" and verifies "not exit 0 with a warning."
This is a legitimate edge case (exit 1 vs any non-zero), but the test steps and setup
are nearly identical.

**Remediation:** Clarify the distinct intent more prominently in scenario 25's test
steps. Consider whether scenario 12's assertions should be a subset of 25's, or
whether 12 focuses on the broader "binary not installed" verification while 25 focuses
on the specific exit code semantics.

#### F-003: Scenarios 9 and 10 are near-duplicates [MINOR]

**Severity:** Minor
**Actionable:** true

- Scenario 9: "Verify tool with skip_install:true is recognized but not installed"
- Scenario 10: "Verify skip_install tool does not appear in resolved manifest output"

Both verify the same behavior (skip_install exclusion) from slightly different
perspectives. Scenario 9 calls resolve() directly; scenario 10 runs end-to-end and
parses JSON output. The distinction is valid (function-level vs integration-level) but
could be collapsed or more clearly differentiated.

**Remediation:** Consider whether both are needed, or merge into one scenario with
two assertion blocks (one for the function return, one for JSON output).

---

## Dimension 3: Pattern Matching Correctness (90/100)

### Verified

- Auto-detected project mode: no tier1_patterns.yaml available
- Test grouping follows logical functional areas: resolver matching, dedup, warnings, skip_install, installer, registry merge, script integration, scaffold, malformed input, shellcheck-py, e2e pipeline
- Stub filenames follow `qf_{functional_group}_stubs_test.go` pattern consistently
- Test functions use `TestXxx/t.Run("[test_id:...]")` pattern — correct for Go `testing` package
- Negative tests properly marked with `[NEGATIVE]` comment in stub docstrings

### Findings

No pattern-matching findings. In auto-detected mode, no project-specific patterns
are available for validation, but the structural patterns are correct for the
Go `testing` + testify framework.

---

## Dimension 4: Test Step Quality (88/100)

### Verified

- All scenarios have `setup`, `test_execution`, and `cleanup` sections
- Step IDs follow `SETUP-01`, `TEST-01`, `CLEANUP-01` convention
- Each step has `action`, `command`, and (for test_execution) `validation` fields
- Cleanup steps appropriately reference `t.TempDir()` for auto-cleanup or explicit server shutdown
- P0 scenarios (12, 25) have appropriately detailed steps with mock server setup
- E2E scenarios (33, 34) have multi-step test_execution sections

### Findings

#### F-004: Cleanup sections use inconsistent specificity [MINOR]

**Severity:** Minor
**Actionable:** true

Some cleanup steps are precise ("server.Close(), os.RemoveAll()") while others are
generic ("Auto cleanup" or "Handled by t.TempDir()"). For code generation readiness,
the cleanup steps that use mock HTTP servers should consistently specify
`server.Close()` as the cleanup action.

**Remediation:** Standardize cleanup steps: scenarios using `httptest.NewServer()`
(11, 12, 25, 26, 33) should all specify `server.Close()` in cleanup.
Scenarios using temp files via `t.TempDir()` can use `"Handled by t.TempDir()"`.

#### F-005: Some test_execution commands are pseudocode rather than executable [MINOR]

**Severity:** Minor
**Actionable:** false

Commands like `resolver.resolve(config, registry)` and `resolver.validate(entry)`
reference Python function calls but the stubs are Go tests. This is acceptable for
STD-level design (the Go stubs will exec the Python scripts via `os/exec`), but
creates a minor cognitive gap for the code generator.

**Remediation:** No action required for STD phase. The code generator should interpret
these as "invoke the resolver via subprocess" rather than direct function calls.

---

## Dimension 4.5: STD Content Policy (90/100)

### Verified

- No PII or secrets in test data
- No hardcoded URLs to real external services (mock servers used throughout)
- Checksum values in test fixtures are explicitly described as "known SHA256" or "wrong checksum" — not real checksums
- Registry entries reference real tool names (lychee, actionlint, uv) appropriately for the test domain
- `acceptance_criteria` are testable boolean conditions, not vague descriptions
- `failure_impact` fields provide meaningful risk context for every assertion

### Findings

No content policy violations found.

---

## Dimension 5: PSE Docstring Quality (85/100)

### Verified

- All 12 stub files have file-level docstring blocks with STP reference and Jira ID
- All 36 `t.Run()` subtests include `[test_id:TS-GH-1270-XXX]` in the test name
- Block comments inside each `t.Run()` include: Preconditions, Steps, Expected sections
- Negative tests marked with `[NEGATIVE]` tag at the start of the docstring
- `t.Skip("Phase 1: Design only - awaiting implementation")` consistently used as placeholder

### Findings

#### F-006: Stub imports are minimal — only `"testing"` imported [MINOR]

**Severity:** Minor
**Actionable:** true

All 12 stub files import only `"testing"`. The STD's `code_generation_config.imports`
section specifies `testify/assert`, `testify/require`, `os`, `os/exec`,
`path/filepath`, `strings`, `encoding/json`, `net/http`, `net/http/httptest`, `io`,
and `gopkg.in/yaml.v3` as needed imports. The stubs should include at minimum
`testify/assert` and `testify/require` since every implemented test will need them.

**Remediation:** Add framework imports (`testify/assert`, `testify/require`) to all
stubs. Add `os/exec` to stubs that will invoke Python scripts. Add `net/http/httptest`
to stubs that need mock HTTP servers (installer, checksum, e2e).

#### F-007: Parent test function docstrings lack structured preconditions [MINOR]

**Severity:** Minor
**Actionable:** true

The parent `TestXxx(t *testing.T)` functions have precondition blocks in free-form
comments. These should be consistent with the subtest format for code generation
parsing.

**Remediation:** Standardize parent-level preconditions to match the `Preconditions:`
format used in subtests.

---

## Dimension 6: Code Generation Readiness (80/100)

### Verified

- Package declaration `package scaffold` is consistent and correct for the target
- `qf_` filename prefix matches `code_generation_config.filename_prefix`
- File groupings are logical and would produce manageable test files
- `t.Skip()` placeholder pattern is correct for phased implementation
- Test function names are descriptive and follow Go conventions

### Findings

#### F-008: Mixed language test targets may confuse code generator [MAJOR]

**Severity:** Major
**Actionable:** true

The STD describes testing a Python script (`resolve-precommit-tools.py`) and a Bash
script (`install-precommit-tools.sh`) using Go test stubs. This is architecturally
correct (Go tests invoke scripts via subprocess), but the STD's `test_steps.command`
fields reference Python function calls (e.g., `resolver.resolve(config, registry)`)
rather than Go subprocess invocations.

The code generator needs clear signals that these are subprocess-based tests:
- Scenarios 1-10, 30-32, 35-36 (resolver): need `os/exec.Command("python3", ...)`
- Scenarios 11-15, 25-26 (installer): need `os/exec.Command("bash", ...)`
- Scenarios 21-24 (script integration): need `os/exec.Command("bash", ...)`
- Scenarios 27-29 (scaffold): direct Go function calls (correct as-is)
- Scenarios 33-34 (e2e): need both Python and Bash subprocess calls

**Remediation:** Add a `test_execution_mode` field to scenarios or add explicit notes
in the test steps indicating subprocess invocation. Example:
```yaml
test_execution:
  - step_id: "TEST-01"
    action: "Invoke resolver via subprocess"
    command: "exec.Command('python3', 'resolve-precommit-tools.py', '--config', fixture)"
    validation: "Parse stdout JSON, verify tool list"
```

#### F-009: code_generation_config.target_test_directories includes non-existent path [MINOR]

**Severity:** Minor
**Actionable:** true

`target_test_directories` includes `"internal/scaffold/scripts-tests"` which does not
appear to exist in the repository. The primary directory `"internal/scaffold"` is
correct. Having a non-existent directory in the config could confuse the code generator.

**Remediation:** Remove `"internal/scaffold/scripts-tests"` from
`target_test_directories` or create the directory if script-specific tests should be
isolated there. Use only `"internal/scaffold"` if all tests should be in the scaffold
package.

---

## Findings Summary

| ID | Severity | Dimension | Description | Actionable |
|:---|:---------|:----------|:------------|:-----------|
| F-001 | Major | YAML Structure | Metadata unit_count (24) and functional_count (10) disagree with actual values (26 and 8) | Yes |
| F-002 | Minor | YAML Structure | Scenarios 12 and 25 overlap significantly on checksum failure testing | Yes |
| F-003 | Minor | YAML Structure | Scenarios 9 and 10 near-duplicate on skip_install behavior | Yes |
| F-004 | Minor | Step Quality | Cleanup sections have inconsistent specificity across scenarios | Yes |
| F-005 | Minor | Step Quality | Test commands reference Python functions but stubs are Go subprocess tests | No |
| F-006 | Minor | PSE Quality | Stub imports minimal — missing testify, os/exec, httptest imports | Yes |
| F-007 | Minor | PSE Quality | Parent test function docstrings lack structured preconditions format | Yes |
| F-008 | Major | Codegen Readiness | Mixed-language test targets need explicit subprocess execution mode signals | Yes |
| F-009 | Minor | Codegen Readiness | target_test_directories includes non-existent path | Yes |

**Critical:** 0 | **Major:** 2 | **Minor:** 7 | **Actionable:** 8/9

---

## Verdict

**APPROVED_WITH_FINDINGS** — The STD is structurally sound with excellent traceability
and comprehensive coverage of the pre-commit-tools subsystem. The two major findings
(metadata count discrepancy and mixed-language execution mode) should be addressed
before code generation to avoid generator confusion, but neither blocks the STD's
approval as a test design document.
