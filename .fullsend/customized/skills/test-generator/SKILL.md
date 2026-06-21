---
name: test-generator
description: Generate working test code from STD YAML — language and framework driven by project config
model: claude-opus-4-6
---

# Test Generator Skill

## Purpose

Generates **working test code** from STD YAML specifications. Reads
project config to determine which languages and frameworks to target.
Not limited to Go and Python — any language declared in config.

**Output:** Working test files that compile/pass collection for each
configured language/framework.

**Note:** For test stubs (design review), use stub-generator skills instead.

---

## Input Required

- `jira_id`: Jira ticket ID (e.g., "MYPROJ-12345")

**Prerequisites:**
- STD YAML at `outputs/std/{JIRA_ID}/{JIRA_ID}_test_description.yaml`
- At least one of:
  - Language config file in `{project_context.config_dir}/` (tier mode)
  - `code_generation_config` in STD YAML (auto mode — `config_dir` may be null)

---

## Output

```
outputs/go-tests/{JIRA_ID}/           (if Go enabled)
├── {feature}_test.go
└── summary.yaml

outputs/python-tests/{JIRA_ID}/       (if Python enabled)
├── test_{feature}.py
├── conftest.py
└── summary.yaml

outputs/tests/{JIRA_ID}/{language}/   (any other language)
└── ...
```

---

## CRITICAL REQUIREMENT

**Generate ONE test case per STD scenario with `coverage_status: NEW` or
`PARTIAL_COVERAGE`. Skip `EXISTING_COVERAGE` — emit a reference comment instead.**

- 19 STD scenarios (12 NEW + 1 PARTIAL + 6 EXISTING) → 13 test functions + 6 reference comments
- Pattern-based file grouping is allowed, but EVERY non-EXISTING scenario gets a test
- For `EXISTING_COVERAGE` scenarios, emit a comment referencing the existing test:
  ```go
  // Covered by existing test: TestComparePathPresence_AllPresent
  // in internal/scaffold/pathpresence_test.go
  ```

When `coverage_status` is absent on a scenario, treat it as `NEW` (backward compatible).

---

## Workflow

### Step 1: Discover Language Targets

**Tier mode** (`config_dir` is not null):

Scan `{project_context.config_dir}/` for YAML files with
`enabled: true` and a `language:` field:

```bash
for f in {project_context.config_dir}/*.yaml; do
  # Skip non-language files (project.yaml, repositories.yaml, etc.)
  # A language config has: enabled, language, framework fields
done
```

Known file names: `go.yaml`, `python.yaml`, `tier1.yaml`, `tier2.yaml`

Each language config provides:
- `language` — "go", "python", etc.
- `framework` — "testing", "ginkgo-v2", "pytest", etc.
- `imports` — organized by category (standard, framework, project)
- `build_command` — validation command
- `test_patterns` — naming conventions

**Auto mode** (`config_dir` is null):

Read `code_generation_config` directly from the STD YAML. The STD YAML IS the config
in auto mode — it contains the detected language, framework, imports, and package name
from the test-strategy-resolver.

```yaml
# From STD YAML code_generation_config:
language: "go"
framework: "testing"
assertion_library: "testify"
package_name: "cli"
imports:
  standard: ["context", "testing"]
  framework: ["github.com/stretchr/testify/assert"]
  project: ["github.com/org/repo/internal/cli"]
```

No config directory scanning needed — generate for the single detected language.

### Step 2: Read STD YAML

Load `outputs/std/{JIRA_ID}/{JIRA_ID}_test_description.yaml`

Extract:
- Total scenario count
- Scenarios grouped by tier/type
- Test objectives, steps, assertions

### Step 3: Load Pattern Rules

**Tier mode:** For each enabled language, read patterns from:
- `{project_context.config_dir}/patterns/{language}_patterns.yaml`
- Fresh LSP patterns if available

**Auto mode:** Skip pattern loading (no pattern library exists for auto-detected
projects). Instead, read existing test files in `SOURCE_REPO_PATH` to learn
conventions (indentation, assertion style, helper patterns) directly from the repo.

### Step 4: Generate Tests Per Language

For each enabled language config, generate test files using the
appropriate framework section below.

---

## Framework: Go `testing` (standard library + testify)

When `framework: "testing"` in the language config:

**File structure:**
```go
//go:build {build_tags}

package {package_name}

import (
    "testing"
    // standard imports from config
    // framework imports from config
    // project imports from config
)

func TestFeatureName(t *testing.T) {
    // shared setup

    t.Run("scenario description", func(t *testing.T) {
        // test implementation
        // use assert.Equal(t, expected, actual)
        // use require.NoError(t, err) for fatal checks
    })
}
```

**Rules:**
- Function prefix from `test_patterns.function_prefix` (default: "Test")
- Subtest style from `test_patterns.subtest_style` (default: "t.Run")
- Assertion style from `test_patterns.assertion_style` (default: "testify")
- Build tags from `build_tags` array → `//go:build tag1 && tag2`
- Import paths from `imports.standard`, `imports.test_framework`, `imports.project`
- Package name from `default_package` or derived from test file location

**Validation:**
- Count `t.Run(` calls = count of STD scenarios
- All imports resolve (no unused imports)
- Build tag line present if `build_tags` configured

---

## Framework: Go `ginkgo-v2` (Ginkgo v2 + Gomega)

When `framework: "ginkgo-v2"` in the language config:

**File structure:**
```go
package {package_name}

import (
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
    // other imports from config
)

var _ = Describe("[JIRA-ID] Feature", func() {
    Context("scenario group", func() {
        It("[test_id:TS-XXX] should do X", func() {
            // test implementation
        })
    })
})
```

**Rules:**
- Dot imports for ginkgo and gomega
- `Describe/Context/It` hierarchy
- `[test_id:TS-XXX]` labels in `It()` descriptions
- `BeforeEach` for shared setup
- `Expect().To()` / `Expect().NotTo()` for assertions

**Validation:**
- Count `It(` blocks = count of STD Functional scenarios
- All `[test_id:TS-XXX]` present

---

## Framework: Python `pytest`

When `framework: "pytest"` in the language config:

**File structure:**
```python
"""Tests for {feature} — {JIRA_ID}."""
import pytest
# imports from config

class TestFeature:
    """Tests for feature X.

    Markers:
        - {markers}

    Preconditions:
        - {preconditions}
    """

    def test_scenario_name(self, fixture1, fixture2):
        """Scenario: {description} [TS-XXX]."""
        # test implementation
        assert result == expected
```

**Rules:**
- `def test_*()` naming convention
- Scenario ID in docstring for traceability
- `conftest.py` for shared fixtures (if multiple test files)
- Fixture naming: nouns, not verbs
- Context managers for resources
- No `time.sleep()` — use polling utilities

**Validation:**
- Count `def test_*` functions = count of STD End-to-End scenarios
- All scenario IDs in docstrings
- `pytest --collect-only` passes (if pytest available)

---

## Polarion Toggle

If `project_context.feature_toggles.polarion` is false, omit Polarion
test case ID markers from generated test code.

## Repo Rules Integration

When `project_context.repo_rules` is available (e.g., AGENTS.md rules),
apply those coding standards to all generated test code. Common rules:
- Implicit markers (don't add explicitly)
- Forbidden patterns (skip/skipif, etc.)
- Fixture guidelines
- Import conventions

---

## Step 5: Validate Complete Coverage

**CRITICAL VALIDATION — MANDATORY**

After all files generated:

1. Count STD scenarios per tier/type, excluding `EXISTING_COVERAGE`
2. Count generated test cases per language
3. Verify 1:1 mapping: every `NEW`/`PARTIAL_COVERAGE` scenario has a test
4. Verify every `EXISTING_COVERAGE` scenario has a reference comment
5. Report missing scenario IDs

---

## Step 6: Report Results

Generate summary per language:
- Language, framework
- Files generated, line counts
- Test count, scenario coverage
- LSP patterns used (true/false)
- Any errors or warnings

---

## Error Handling

**STD not found:** Error + suggest running `/std-builder` first.

**No language configs (tier mode):** Error + suggest creating language YAML in config.

**No code_generation_config (auto mode):** Error + STD YAML may not have been generated
with auto mode. Suggest re-running `/std-builder`.

**Pattern not recognized:** Warning + fall back to direct STD-to-test generation.

**Validation fails:** Save to `.invalid` extension, show errors, continue.

---

**End of Test Generator Skill**
