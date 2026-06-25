package scaffold

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// QualityFlow tests for GH-1270: Resolver matching, deduplication,
// warnings, skip_install, malformed input, and shellcheck-py scenarios.
// STD: outputs/std/GH-1270/GH-1270_test_description.yaml
//
// These tests invoke resolve-precommit-tools.py from Go using os/exec,
// creating temp fixtures for each scenario.

// resolverResult captures the JSON output of resolve-precommit-tools.py.
type resolverResult struct {
	Tools    []map[string]any `json:"tools"`
	Warnings []string         `json:"warnings"`
}

// resolverScriptPath returns the path to the resolver Python script.
func resolverScriptPath() string {
	return filepath.Join("fullsend-repo", "scripts", "resolve-precommit-tools.py")
}

// runResolver executes resolve-precommit-tools.py against a temp repo dir
// and returns the parsed JSON result plus raw stderr.
func runResolver(t *testing.T, repoDir string, extraArgs ...string) (resolverResult, string) {
	t.Helper()
	args := append([]string{resolverScriptPath()}, repoDir)
	args = append(args, extraArgs...)
	cmd := exec.Command("python3", args...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	// The resolver exits 0 even on parse errors (by design).
	require.NoError(t, err, "resolver should exit 0; stderr: %s", stderr.String())
	var result resolverResult
	require.NoError(t, json.Unmarshal([]byte(stdout.String()), &result),
		"failed to parse resolver output: %s", stdout.String())
	return result, stderr.String()
}

// writePrecommitConfig writes a .pre-commit-config.yaml to the given dir.
func writePrecommitConfig(t *testing.T, dir, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, ".pre-commit-config.yaml"),
		[]byte(content), 0o644))
}

// --- Scenarios 1-3: Resolver matching ---

// TestQF_ResolverMatchEntry_UvRun verifies the resolver matches 'uv' match_entry
// for hooks with 'uv run mypy' entry.
// [TS-GH-1270-001] Scenario 1
func TestQF_ResolverMatchEntry_UvRun(t *testing.T) {
	dir := t.TempDir()
	writePrecommitConfig(t, dir, `
repos:
  - repo: local
    hooks:
      - id: mypy-check
        entry: "uv run mypy"
        language: system
`)
	result, _ := runResolver(t, dir)
	require.NotEmpty(t, result.Tools, "resolver should return at least one tool")
	found := false
	for _, tool := range result.Tools {
		if tool["name"] == "uv" {
			found = true
			break
		}
	}
	assert.True(t, found, "resolved tools should contain 'uv'")
}

// TestQF_ResolverMatchEntry_NoPartialSubstring verifies 'uv' match_entry
// does NOT match 'uvx-other-tool' (no false positive substring match).
// [TS-GH-1270-002] Scenario 2
func TestQF_ResolverMatchEntry_NoPartialSubstring(t *testing.T) {
	dir := t.TempDir()
	writePrecommitConfig(t, dir, `
repos:
  - repo: local
    hooks:
      - id: other-tool
        entry: "uvx-other-tool run check"
        language: system
`)
	result, _ := runResolver(t, dir)
	for _, tool := range result.Tools {
		assert.NotEqual(t, "uv", tool["name"],
			"'uv' should NOT match 'uvx-other-tool' — substring matching is unsafe")
	}
}

// TestQF_ResolverMatchEntry_UnknownEntry verifies resolver returns no match
// for an unknown entry command.
// [TS-GH-1270-003] Scenario 3
func TestQF_ResolverMatchEntry_UnknownEntry(t *testing.T) {
	dir := t.TempDir()
	writePrecommitConfig(t, dir, `
repos:
  - repo: local
    hooks:
      - id: unknown-tool
        entry: "some-unknown-tool run check"
        language: system
`)
	result, _ := runResolver(t, dir)
	assert.Empty(t, result.Tools,
		"unknown entry command should produce empty tools list")
}

// --- Scenarios 4-5: Deduplication ---

// TestQF_ResolverDedup_SameToolName verifies seen_names deduplication when
// both 'uvx' and 'uv' hooks resolve to the same tool.
// [TS-GH-1270-004] Scenario 4
func TestQF_ResolverDedup_SameToolName(t *testing.T) {
	dir := t.TempDir()
	writePrecommitConfig(t, dir, `
repos:
  - repo: local
    hooks:
      - id: ty
        entry: "uvx ty check"
        language: system
      - id: mypy-check
        entry: "uv run mypy"
        language: system
`)
	result, _ := runResolver(t, dir)
	uvCount := 0
	for _, tool := range result.Tools {
		if tool["name"] == "uv" {
			uvCount++
		}
	}
	assert.Equal(t, 1, uvCount,
		"two hooks resolving to 'uv' should produce exactly one manifest entry")
}

// TestQF_ResolverDedup_UniqueManifestNames verifies JSON manifest has
// unique tool names only.
// [TS-GH-1270-005] Scenario 5
func TestQF_ResolverDedup_UniqueManifestNames(t *testing.T) {
	dir := t.TempDir()
	writePrecommitConfig(t, dir, `
repos:
  - repo: local
    hooks:
      - id: ty
        entry: "uvx ty check"
        language: system
      - id: mypy-check
        entry: "uv run mypy"
        language: system
`)
	result, _ := runResolver(t, dir)
	seen := make(map[string]bool)
	for _, tool := range result.Tools {
		name, _ := tool["name"].(string)
		assert.False(t, seen[name],
			"duplicate tool name in manifest: %s", name)
		seen[name] = true
	}
}

// --- Scenarios 6-8: Warning messages ---

// TestQF_ResolverWarning_SystemHookNotInRegistry verifies warning for
// language:system hook not in registry includes the command name.
// [TS-GH-1270-006] Scenario 6
func TestQF_ResolverWarning_SystemHookNotInRegistry(t *testing.T) {
	dir := t.TempDir()
	writePrecommitConfig(t, dir, `
repos:
  - repo: local
    hooks:
      - id: my-custom-lint
        entry: "my-custom-linter check"
        language: system
`)
	result, _ := runResolver(t, dir)
	require.NotEmpty(t, result.Warnings,
		"warning should be emitted for unregistered system hook")
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "my-custom-linter") && strings.Contains(w, "system") {
			found = true
			break
		}
	}
	assert.True(t, found,
		"warning should include command name 'my-custom-linter'; got warnings: %v", result.Warnings)
}

// TestQF_ResolverWarning_GolangHookMentionsGoToolchain verifies warning
// for language:golang hooks mentions Go toolchain requirement.
// [TS-GH-1270-007] Scenario 7
func TestQF_ResolverWarning_GolangHookMentionsGoToolchain(t *testing.T) {
	dir := t.TempDir()
	writePrecommitConfig(t, dir, `
repos:
  - repo: https://github.com/tekwizely/pre-commit-golang
    hooks:
      - id: go-vet
        language: golang
`)
	result, _ := runResolver(t, dir)
	require.NotEmpty(t, result.Warnings,
		"warning should be emitted for golang-language hook")
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "Go") || strings.Contains(w, "golang") {
			found = true
			break
		}
	}
	assert.True(t, found,
		"warning should mention Go toolchain; got: %v", result.Warnings)
}

// TestQF_ResolverWarning_PythonHookNoWarning verifies no warning is emitted
// for language:python hooks (auto-managed by pre-commit).
// [TS-GH-1270-008] Scenario 8
func TestQF_ResolverWarning_PythonHookNoWarning(t *testing.T) {
	dir := t.TempDir()
	writePrecommitConfig(t, dir, `
repos:
  - repo: https://github.com/psf/black
    hooks:
      - id: black
        language: python
`)
	result, _ := runResolver(t, dir)
	for _, w := range result.Warnings {
		assert.NotContains(t, strings.ToLower(w), "python",
			"no warning should be emitted for python-language hooks")
	}
}

// --- Scenarios 9-10: skip_install ---

// TestQF_ResolverSkipInstall_RecognizedButNotInstalled verifies tool with
// skip_install:true is recognized (no warning) and flagged for skip in manifest.
// [TS-GH-1270-009] Scenario 9
//
// The resolver includes the tool in the manifest with skip_install:true;
// the installer reads this flag and skips the actual download/install.
// This design ensures the hook is "recognized" (suppressing warnings)
// while the installer handles the skip.
func TestQF_ResolverSkipInstall_RecognizedButNotInstalled(t *testing.T) {
	dir := t.TempDir()
	// gitleaks is in the registry with skip_install: true
	writePrecommitConfig(t, dir, `
repos:
  - repo: https://github.com/zricethezav/gitleaks
    hooks:
      - id: gitleaks
`)
	result, _ := runResolver(t, dir)
	// The resolver includes gitleaks in the manifest (it matched), but with
	// skip_install: true. The INSTALLER is responsible for respecting this flag.
	// Verify the tool IS matched (no warning about missing from registry).
	for _, w := range result.Warnings {
		assert.NotContains(t, w, "gitleaks",
			"no warning should be emitted for skip_install tool (it IS in registry)")
	}
	// Verify the install block has skip_install flag
	found := false
	for _, tool := range result.Tools {
		if name, _ := tool["name"].(string); name == "gitleaks" {
			found = true
			skipInstall, _ := tool["skip_install"].(bool)
			assert.True(t, skipInstall,
				"gitleaks install entry should have skip_install: true")
		}
	}
	assert.True(t, found, "gitleaks should appear in resolved tools (with skip_install flag)")
}

// TestQF_ResolverSkipInstall_InstallerSkips verifies that the installer
// respects skip_install:true and does not actually install the tool.
// [TS-GH-1270-010] Scenario 10
//
// The resolver includes gitleaks in the manifest (to suppress warnings),
// but the installer reads skip_install and skips the download.
func TestQF_ResolverSkipInstall_InstallerSkips(t *testing.T) {
	// Create a manifest with only a skip_install tool
	manifest := map[string]any{
		"tools": []map[string]any{
			{
				"type":         "binary",
				"name":         "gitleaks",
				"skip_install": "true",
			},
		},
		"warnings": []string{},
	}

	dir := t.TempDir()
	data, err := json.Marshal(manifest)
	require.NoError(t, err)
	manifestPath := filepath.Join(dir, "manifest.json")
	require.NoError(t, os.WriteFile(manifestPath, data, 0o644))

	cmd := exec.Command("bash", installerScriptPath(), manifestPath)
	homeDir := t.TempDir()
	cmd.Env = []string{
		"HOME=" + homeDir,
		"PATH=" + os.Getenv("PATH"),
	}
	var stdout strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout
	err = cmd.Run()
	require.NoError(t, err, "installer should exit 0 for skip_install tool")
	output := stdout.String()

	assert.Contains(t, output, "skipped",
		"installer should skip the skip_install tool")
	assert.Contains(t, output, "gitleaks",
		"output should mention gitleaks by name")
}

// --- Scenarios 30-32: Malformed input handling ---

// TestQF_ResolverMalformedYAML verifies resolver returns empty tools for
// invalid YAML in .pre-commit-config.yaml.
// [TS-GH-1270-030] Scenario 30
func TestQF_ResolverMalformedYAML(t *testing.T) {
	dir := t.TempDir()
	writePrecommitConfig(t, dir, `
{{{this is not valid YAML!!!
  - broken: [unterminated
`)
	result, _ := runResolver(t, dir)
	assert.Empty(t, result.Tools,
		"malformed YAML should produce empty tools list")
	assert.NotEmpty(t, result.Warnings,
		"malformed YAML should produce a warning")
}

// TestQF_ResolverMissingReposField verifies resolver returns empty tools
// when .pre-commit-config.yaml lacks the 'repos' field.
// [TS-GH-1270-031] Scenario 31
func TestQF_ResolverMissingReposField(t *testing.T) {
	dir := t.TempDir()
	writePrecommitConfig(t, dir, `
ci:
  autofix_commit_msg: "fix"
`)
	result, _ := runResolver(t, dir)
	assert.Empty(t, result.Tools,
		"missing repos field should produce empty tools list")
}

// TestQF_ResolverNonListRepos verifies resolver handles non-list repos field.
// [TS-GH-1270-032] Scenario 32
func TestQF_ResolverNonListRepos(t *testing.T) {
	dir := t.TempDir()
	writePrecommitConfig(t, dir, `
repos: "not-a-list"
`)
	result, _ := runResolver(t, dir)
	assert.Empty(t, result.Tools,
		"non-list repos field should produce empty tools list")
}

// --- Scenarios 35-36: shellcheck-py variant ---

// TestQF_ResolverShellcheckPy_NoWarning verifies no warning is emitted for
// shellcheck-py/shellcheck-py hook (language: python, auto-managed).
// [TS-GH-1270-035] Scenario 35
func TestQF_ResolverShellcheckPy_NoWarning(t *testing.T) {
	dir := t.TempDir()
	writePrecommitConfig(t, dir, `
repos:
  - repo: https://github.com/shellcheck-py/shellcheck-py
    hooks:
      - id: shellcheck
        language: python
`)
	result, _ := runResolver(t, dir)
	for _, w := range result.Warnings {
		assert.NotContains(t, strings.ToLower(w), "shellcheck",
			"no warning should be emitted for shellcheck-py (language: python)")
	}
}

// TestQF_ResolverShellcheckPy_NotInManifest verifies shellcheck-py is not
// included in the resolved tools manifest.
// [TS-GH-1270-036] Scenario 36
func TestQF_ResolverShellcheckPy_NotInManifest(t *testing.T) {
	dir := t.TempDir()
	writePrecommitConfig(t, dir, `
repos:
  - repo: https://github.com/shellcheck-py/shellcheck-py
    hooks:
      - id: shellcheck
        language: python
`)
	result, _ := runResolver(t, dir)
	for _, tool := range result.Tools {
		name, _ := tool["name"].(string)
		assert.NotEqual(t, "shellcheck", name,
			"shellcheck-py should not appear in install manifest (auto-managed)")
	}
}

// --- Scenario 24: Missing .pre-commit-config.yaml ---

// TestQF_ResolverMissingConfig verifies graceful handling when
// .pre-commit-config.yaml is absent.
// [TS-GH-1270-024] Scenario 24
func TestQF_ResolverMissingConfig(t *testing.T) {
	dir := t.TempDir()
	// Don't create any config file
	result, _ := runResolver(t, dir)
	assert.Empty(t, result.Tools,
		"missing config should produce empty tools")
}
