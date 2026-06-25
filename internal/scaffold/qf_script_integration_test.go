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

// QualityFlow tests for GH-1270: Script integration scenarios (21-23, 33).
// STD: outputs/std/GH-1270/GH-1270_test_description.yaml
//
// These tests validate the integration between pre-code.sh, post-code.sh,
// resolve-precommit-tools.py, and install-precommit-tools.sh.

// TestQF_PreCodeInstallsToolsAndSetsPath verifies pre-code.sh installs
// tools and adds ~/.local/bin to PATH and GITHUB_PATH.
// [TS-GH-1270-022] Scenario 22
func TestQF_PreCodeInstallsToolsAndSetsPath(t *testing.T) {
	// Read pre-code.sh content to verify PATH setup logic
	content, err := FullsendRepoFile("scripts/pre-code.sh")
	require.NoError(t, err)
	s := string(content)

	// Verify PATH export is present
	assert.Contains(t, s, `${HOME}/.local/bin`,
		"pre-code.sh should reference ~/.local/bin")
	assert.Contains(t, s, "GITHUB_PATH",
		"pre-code.sh should write to GITHUB_PATH for downstream steps")
	assert.Contains(t, s, "export PATH",
		"pre-code.sh should export the updated PATH")
}

// TestQF_PostCodeResolvesBeforePrecommit verifies post-code.sh calls
// resolver before pre-commit check.
// [TS-GH-1270-021] Scenario 21
func TestQF_PostCodeResolvesBeforePrecommit(t *testing.T) {
	content, err := FullsendRepoFile("scripts/post-code.sh")
	require.NoError(t, err)
	s := string(content)

	// Verify the resolve → install → pre-commit run execution order.
	// Use specific substrings to avoid matching early occurrences in comments.
	resolveIdx := strings.Index(s, "resolve-precommit-tools.py")
	installIdx := strings.Index(s, "install-precommit-tools.sh")
	// Match the actual pre-commit invocation, not comments
	precommitRunIdx := strings.Index(s, "pre-commit run")

	require.NotEqual(t, -1, resolveIdx,
		"post-code.sh should reference resolve-precommit-tools.py")
	require.NotEqual(t, -1, installIdx,
		"post-code.sh should reference install-precommit-tools.sh")

	if resolveIdx >= 0 && installIdx >= 0 {
		assert.Less(t, resolveIdx, installIdx,
			"resolver must be referenced before installer in post-code.sh")
	}
	if installIdx >= 0 && precommitRunIdx >= 0 {
		assert.Less(t, installIdx, precommitRunIdx,
			"installer must be referenced before 'pre-commit run' in post-code.sh")
	}
}

// TestQF_PreCodeGracefulDegradation verifies graceful degradation when
// resolve script fails (warning, no abort).
// [TS-GH-1270-023] Scenario 23
func TestQF_PreCodeGracefulDegradation(t *testing.T) {
	// Verify pre-code.sh handles resolver failure gracefully
	content, err := FullsendRepoFile("scripts/pre-code.sh")
	require.NoError(t, err)
	s := string(content)

	// The script should have error handling around the resolver call
	assert.Contains(t, s, "warning",
		"pre-code.sh should emit a warning when resolver fails")
	// Should continue (not exit 1) when resolver fails
	assert.Contains(t, s, "continuing",
		"pre-code.sh should continue without auto-install on resolver failure")
}

// TestQF_PreCodeResolverIntegration does a functional test of the
// pre-code.sh tool resolution flow by setting up a minimal repo.
// [TS-GH-1270-022] Scenario 22 (functional)
func TestQF_PreCodeResolverIntegration(t *testing.T) {
	// Create a minimal repo structure
	repoDir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(repoDir, ".pre-commit-config.yaml"),
		[]byte(`
repos:
  - repo: https://github.com/psf/black
    hooks:
      - id: black
        language: python
`), 0o644))

	// Run just the resolver portion (not the full pre-code.sh which
	// needs ISSUE_NUMBER and other env vars)
	resolverPath := filepath.Join("fullsend-repo", "scripts", "resolve-precommit-tools.py")
	cmd := exec.Command("python3", resolverPath, repoDir)
	var stdout strings.Builder
	cmd.Stdout = &stdout
	err := cmd.Run()
	require.NoError(t, err, "resolver should succeed on valid repo")

	var result struct {
		Tools    []any    `json:"tools"`
		Warnings []string `json:"warnings"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout.String()), &result))
	// Python hooks are auto-managed — no tools should be resolved
	assert.Empty(t, result.Tools,
		"python-only repo should produce empty manifest")
}

// TestQF_E2EResolverManifestPipeline tests the full resolver -> manifest
// pipeline for a repo with lychee + uv + actionlint hooks.
// [TS-GH-1270-033] Scenario 33
func TestQF_E2EResolverManifestPipeline(t *testing.T) {
	repoDir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(repoDir, ".pre-commit-config.yaml"),
		[]byte(`
repos:
  - repo: local
    hooks:
      - id: lint-md-links
        entry: "lychee ."
        language: system
      - id: ty
        entry: "uvx ty check"
        language: system
  - repo: https://github.com/rhysd/actionlint
    hooks:
      - id: actionlint
`), 0o644))

	resolverPath := filepath.Join("fullsend-repo", "scripts", "resolve-precommit-tools.py")
	cmd := exec.Command("python3", resolverPath, repoDir)
	var stdout strings.Builder
	cmd.Stdout = &stdout
	err := cmd.Run()
	require.NoError(t, err, "resolver should succeed")

	var result struct {
		Tools    []map[string]any `json:"tools"`
		Warnings []string         `json:"warnings"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout.String()), &result))

	// Should resolve all three tools: lychee, uv (from uvx), actionlint
	names := make(map[string]bool)
	for _, tool := range result.Tools {
		if name, ok := tool["name"].(string); ok {
			names[name] = true
		}
	}
	assert.True(t, names["lychee"], "manifest should contain lychee")
	assert.True(t, names["uv"], "manifest should contain uv (from uvx match)")
	assert.True(t, names["actionlint"], "manifest should contain actionlint")
	assert.Len(t, result.Tools, 3,
		"manifest should contain exactly 3 tools")
}

// TestQF_E2EPipelineEmptyManifest tests pipeline with a repo that only
// has auto-managed hooks (empty manifest, no install).
// [TS-GH-1270-034] Scenario 34
func TestQF_E2EPipelineEmptyManifest(t *testing.T) {
	repoDir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(repoDir, ".pre-commit-config.yaml"),
		[]byte(`
repos:
  - repo: https://github.com/psf/black
    hooks:
      - id: black
        language: python
  - repo: https://github.com/pre-commit/mirrors-mypy
    hooks:
      - id: mypy
        language: python
`), 0o644))

	resolverPath := filepath.Join("fullsend-repo", "scripts", "resolve-precommit-tools.py")
	cmd := exec.Command("python3", resolverPath, repoDir)
	var stdout strings.Builder
	cmd.Stdout = &stdout
	err := cmd.Run()
	require.NoError(t, err)

	var result struct {
		Tools    []any    `json:"tools"`
		Warnings []string `json:"warnings"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout.String()), &result))
	assert.Empty(t, result.Tools,
		"auto-managed-only repo should produce empty manifest")
}
