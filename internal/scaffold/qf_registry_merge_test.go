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

// QualityFlow tests for GH-1270: Per-repo registry merge scenarios (16-20).
// STD: outputs/std/GH-1270/GH-1270_test_description.yaml
//
// These tests invoke merge_registries() via a Python helper script that
// imports the resolver module and calls merge_registries directly.

// mergePyScript is a small Python script that imports the resolver module
// and calls merge_registries with upstream + local YAML files.
const mergePyScript = `
import importlib.util, json, sys, os
spec = importlib.util.spec_from_file_location("resolver", sys.argv[1])
resolver = importlib.util.module_from_spec(spec)
spec.loader.exec_module(resolver)

import yaml
with open(sys.argv[2]) as f:
    upstream = yaml.safe_load(f)
with open(sys.argv[3]) as f:
    local = yaml.safe_load(f)
merged, warnings = resolver.merge_registries(upstream, local)
print(json.dumps({"merged": merged, "warnings": warnings}))
`

type mergeResult struct {
	Merged   map[string]any `json:"merged"`
	Warnings []string       `json:"warnings"`
}

func runMerge(t *testing.T, upstreamYAML, localYAML string) mergeResult {
	t.Helper()
	dir := t.TempDir()

	scriptPath := filepath.Join(dir, "merge_test.py")
	require.NoError(t, os.WriteFile(scriptPath, []byte(mergePyScript), 0o644))

	upstreamPath := filepath.Join(dir, "upstream.yaml")
	require.NoError(t, os.WriteFile(upstreamPath, []byte(upstreamYAML), 0o644))

	localPath := filepath.Join(dir, "local.yaml")
	require.NoError(t, os.WriteFile(localPath, []byte(localYAML), 0o644))

	resolverPath := filepath.Join("fullsend-repo", "scripts", "resolve-precommit-tools.py")

	cmd := exec.Command("python3", scriptPath, resolverPath, upstreamPath, localPath)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	require.NoError(t, err, "merge script failed; stderr: %s", stderr.String())

	var result mergeResult
	require.NoError(t, json.Unmarshal([]byte(stdout.String()), &result),
		"failed to parse merge output: %s", stdout.String())
	return result
}

func mergedToolCount(result mergeResult) int {
	tools, ok := result.Merged["tools"].([]any)
	if !ok {
		return 0
	}
	return len(tools)
}

// TestQF_MergeRegistries_AdditiveMerge verifies additive merge appends
// new entries to upstream registry.
// [TS-GH-1270-016] Scenario 16
func TestQF_MergeRegistries_AdditiveMerge(t *testing.T) {
	upstream := `
tools:
  - hook_id: lint-a
    repo: local
    install:
      name: tool-a
  - hook_id: lint-b
    repo: local
    install:
      name: tool-b
`
	local := `
tools:
  - hook_id: lint-c
    repo: local
    install:
      name: tool-c
`
	result := runMerge(t, upstream, local)
	assert.Equal(t, 3, mergedToolCount(result),
		"merged registry should contain upstream (A, B) + new per-repo (C)")
	assert.Empty(t, result.Warnings)
}

// TestQF_MergeRegistries_Override verifies matching (repo, hook_id) key
// overrides the upstream entry.
// [TS-GH-1270-017] Scenario 17
func TestQF_MergeRegistries_Override(t *testing.T) {
	upstream := `
tools:
  - hook_id: lint
    repo: https://github.com/example/lint
    install:
      name: linter
      version: "1.0"
`
	local := `
tools:
  - hook_id: lint
    repo: https://github.com/example/lint
    install:
      name: linter
      version: "2.0"
`
	result := runMerge(t, upstream, local)
	assert.Equal(t, 1, mergedToolCount(result),
		"override should not create duplicate")
	// Check version is from per-repo
	tools := result.Merged["tools"].([]any)
	entry := tools[0].(map[string]any)
	install := entry["install"].(map[string]any)
	assert.Equal(t, "2.0", install["version"],
		"per-repo version should replace upstream")
}

// TestQF_MergeRegistries_Exclude verifies exclude:true suppresses
// the matching upstream entry.
// [TS-GH-1270-018] Scenario 18
func TestQF_MergeRegistries_Exclude(t *testing.T) {
	upstream := `
tools:
  - hook_id: tool-a
    repo: local
    install:
      name: tool-a
  - hook_id: tool-b
    repo: local
    install:
      name: tool-b
  - hook_id: tool-c
    repo: local
    install:
      name: tool-c
`
	local := `
tools:
  - hook_id: tool-b
    repo: local
    exclude: true
`
	result := runMerge(t, upstream, local)
	assert.Equal(t, 2, mergedToolCount(result),
		"excluded entry should be removed")
	// Verify tool-b is not in result
	raw, _ := json.Marshal(result.Merged)
	assert.NotContains(t, string(raw), `"hook_id": "tool-b"`,
		"excluded tool-b should be absent")
	assert.Contains(t, string(raw), "tool-a")
	assert.Contains(t, string(raw), "tool-c")
}

// TestQF_MergeRegistries_InvalidEntry_MissingHookID verifies warning for
// a per-repo entry missing the required hook_id field.
// [TS-GH-1270-019] Scenario 19
func TestQF_MergeRegistries_InvalidEntry_MissingHookID(t *testing.T) {
	upstream := `
tools:
  - hook_id: lint
    repo: local
    install:
      name: linter
`
	local := `
tools:
  - repo: local
    install:
      name: orphan
`
	result := runMerge(t, upstream, local)
	assert.Equal(t, 1, mergedToolCount(result),
		"invalid entry should be skipped")
	require.NotEmpty(t, result.Warnings,
		"warning should be emitted for missing hook_id")
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "hook_id") {
			found = true
			break
		}
	}
	assert.True(t, found,
		"warning should mention 'hook_id'; got: %v", result.Warnings)
}

// TestQF_MergeRegistries_EmptyLocalFallback verifies empty per-repo registry
// falls back to upstream only.
// [TS-GH-1270-020] Scenario 20
func TestQF_MergeRegistries_EmptyLocalFallback(t *testing.T) {
	upstream := `
tools:
  - hook_id: lint
    repo: local
    install:
      name: linter
  - hook_id: fmt
    repo: local
    install:
      name: formatter
`
	local := `
tools: []
`
	result := runMerge(t, upstream, local)
	assert.Equal(t, 2, mergedToolCount(result),
		"empty per-repo should leave upstream unchanged")
	assert.Empty(t, result.Warnings,
		"no warnings for valid empty per-repo registry")
}
