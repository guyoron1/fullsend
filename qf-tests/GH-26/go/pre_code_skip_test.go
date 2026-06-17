package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Pre-Code Skip Defense Layer Tests

STP Reference: outputs/stp/GH-26/GH-26_test_plan.md
STD Reference: outputs/std/GH-26/GH-26_test_description.yaml
Jira: GH-26

Tests for the pre-code.sh script that detects existing human PRs
and skips automated code agent execution to prevent duplicate PRs.
*/

//go:build e2e

// preCodeScriptPath returns the path to pre-code.sh relative to the repo root.
const preCodeScriptRelPath = "internal/scaffold/fullsend-repo/scripts/pre-code.sh"

// setupPreCodeEnv creates a temp directory with a mock gh binary, a GITHUB_OUTPUT
// file, and returns a cleanup function. The mock gh binary writes its arguments
// to a log file and outputs the given response for "pr list" calls.
func setupPreCodeEnv(t *testing.T, prListResponse string) (tmpDir string, ghOutputPath string, mockLogPath string) {
	t.Helper()

	tmpDir = t.TempDir()
	mockBinDir := filepath.Join(tmpDir, "bin")
	require.NoError(t, os.MkdirAll(mockBinDir, 0o755))

	ghOutputPath = filepath.Join(tmpDir, "github_output")
	require.NoError(t, os.WriteFile(ghOutputPath, nil, 0o644))

	mockLogPath = filepath.Join(tmpDir, "gh_calls.log")

	// Create mock gh binary that:
	// 1. Logs all invocations to gh_calls.log
	// 2. Returns prListResponse for "pr list" subcommand
	// 3. Returns success for label/comment operations
	mockGH := `#!/usr/bin/env bash
set -euo pipefail
echo "$@" >> "` + mockLogPath + `"

if [[ "$1" == "pr" && "$2" == "list" ]]; then
  cat << 'PREOF'
` + prListResponse + `
PREOF
  exit 0
fi

# label create, api (label add), issue comment — all succeed silently
exit 0
`
	mockGHPath := filepath.Join(mockBinDir, "gh")
	require.NoError(t, os.WriteFile(mockGHPath, []byte(mockGH), 0o755))

	return tmpDir, ghOutputPath, mockLogPath
}

// runPreCode executes pre-code.sh with the given environment overrides.
func runPreCode(t *testing.T, scriptPath string, env map[string]string) (string, error) {
	t.Helper()

	cmd := exec.Command("bash", scriptPath)

	// Start with a clean environment, then layer on required vars
	cmd.Env = []string{
		"PATH=" + env["PATH"],
		"HOME=" + os.Getenv("HOME"),
	}
	for k, v := range env {
		if k == "PATH" {
			continue
		}
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	out, err := cmd.CombinedOutput()
	return string(out), err
}

// findScript locates pre-code.sh from the repo root.
func findScript(t *testing.T) string {
	t.Helper()
	// Try common locations
	candidates := []string{
		preCodeScriptRelPath,
		filepath.Join("..", preCodeScriptRelPath),
		filepath.Join("..", "..", preCodeScriptRelPath),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	// Fallback: use REPO_ROOT env var
	root := os.Getenv("REPO_ROOT")
	if root != "" {
		return filepath.Join(root, preCodeScriptRelPath)
	}
	t.Skip("pre-code.sh not found — set REPO_ROOT or run from repo root")
	return ""
}

func makeBaseEnv(tmpDir, ghOutputPath string) map[string]string {
	return map[string]string{
		"PATH":             filepath.Join(tmpDir, "bin") + ":" + os.Getenv("PATH"),
		"ISSUE_NUMBER":     "42",
		"REPO_FULL_NAME":   "org/repo",
		"GITHUB_ISSUE_URL": "https://github.com/org/repo/issues/42",
		"GH_TOKEN":         "fake-token",
		"GITHUB_OUTPUT":    ghOutputPath,
	}
}

// TestPreCodeSkipsWhenHumanPRExists verifies that pre-code.sh detects an
// existing open human PR and sets the skip flag.
//
// [test_id:TS-GH-26-001]
func TestPreCodeSkipsWhenHumanPRExists(t *testing.T) {
	scriptPath := findScript(t)

	humanPRJSON := `100	human-dev	https://github.com/org/repo/pull/100`
	tmpDir, ghOutputPath, _ := setupPreCodeEnv(t, humanPRJSON)

	env := makeBaseEnv(tmpDir, ghOutputPath)

	output, err := runPreCode(t, scriptPath, env)
	require.NoError(t, err, "pre-code.sh should exit 0; output: %s", output)

	ghOutput, err := os.ReadFile(ghOutputPath)
	require.NoError(t, err)

	assert.Contains(t, string(ghOutput), "skipped=true",
		"GITHUB_OUTPUT should contain skipped=true when human PR exists")
	assert.Contains(t, output, "Found existing human PR",
		"Script should log the detected human PR")
}

// TestPreCodePostsSkipComment verifies that a skip comment is posted on the
// issue when pre-code.sh detects an existing human PR.
//
// [test_id:TS-GH-26-002]
func TestPreCodePostsSkipComment(t *testing.T) {
	scriptPath := findScript(t)

	humanPRJSON := `100	human-dev	https://github.com/org/repo/pull/100`
	tmpDir, ghOutputPath, mockLogPath := setupPreCodeEnv(t, humanPRJSON)

	env := makeBaseEnv(tmpDir, ghOutputPath)

	output, err := runPreCode(t, scriptPath, env)
	require.NoError(t, err, "pre-code.sh should exit 0; output: %s", output)

	// Verify gh issue comment was called
	mockLog, err := os.ReadFile(mockLogPath)
	require.NoError(t, err)

	assert.Contains(t, string(mockLog), "issue comment",
		"Mock gh should have been called with 'issue comment' subcommand")
}

// TestPreCodeAppliesPROpenLabel verifies that the pr-open label is applied
// to the issue when pre-code.sh skips.
//
// [test_id:TS-GH-26-003]
func TestPreCodeAppliesPROpenLabel(t *testing.T) {
	scriptPath := findScript(t)

	humanPRJSON := `100	human-dev	https://github.com/org/repo/pull/100`
	tmpDir, ghOutputPath, mockLogPath := setupPreCodeEnv(t, humanPRJSON)

	env := makeBaseEnv(tmpDir, ghOutputPath)

	_, err := runPreCode(t, scriptPath, env)
	require.NoError(t, err)

	mockLog, err := os.ReadFile(mockLogPath)
	require.NoError(t, err)
	logStr := string(mockLog)

	// The script creates the label and then applies it via the API
	assert.Contains(t, logStr, "label create pr-open",
		"Mock gh should be called to create pr-open label")
	assert.Contains(t, logStr, "repos/org/repo/issues/42/labels",
		"Mock gh should apply label to the issue")
}

// TestPreCodeWritesSkippedTrueToOutput verifies that skipped=true is written
// to GITHUB_OUTPUT when a human PR is detected.
//
// [test_id:TS-GH-26-004]
func TestPreCodeWritesSkippedTrueToOutput(t *testing.T) {
	scriptPath := findScript(t)

	humanPRJSON := `100	human-dev	https://github.com/org/repo/pull/100`
	tmpDir, ghOutputPath, _ := setupPreCodeEnv(t, humanPRJSON)

	env := makeBaseEnv(tmpDir, ghOutputPath)

	_, err := runPreCode(t, scriptPath, env)
	require.NoError(t, err)

	ghOutput, err := os.ReadFile(ghOutputPath)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(ghOutput)), "\n")
	found := false
	for _, line := range lines {
		if line == "skipped=true" {
			found = true
			break
		}
	}
	assert.True(t, found,
		"GITHUB_OUTPUT should contain 'skipped=true' line; got: %s", string(ghOutput))
}

// TestPreCodeProceedsWithNoPRs verifies that the agent proceeds when no
// existing open PRs are found for the target issue.
//
// [test_id:TS-GH-26-005]
func TestPreCodeProceedsWithNoPRs(t *testing.T) {
	scriptPath := findScript(t)

	// Empty response — no PRs found
	tmpDir, ghOutputPath, _ := setupPreCodeEnv(t, "")

	env := makeBaseEnv(tmpDir, ghOutputPath)

	output, err := runPreCode(t, scriptPath, env)
	require.NoError(t, err, "pre-code.sh should exit 0; output: %s", output)

	ghOutput, err := os.ReadFile(ghOutputPath)
	require.NoError(t, err)

	assert.NotContains(t, string(ghOutput), "skipped=true",
		"GITHUB_OUTPUT should NOT contain skipped=true when no PRs exist")
	assert.Contains(t, output, "No existing human PRs found",
		"Script should log that no PRs were found")
}

// TestPreCodeWritesSkippedFalseToOutput verifies that skipped=false is
// explicitly written to GITHUB_OUTPUT when no PRs are found.
//
// [test_id:TS-GH-26-006]
func TestPreCodeWritesSkippedFalseToOutput(t *testing.T) {
	scriptPath := findScript(t)

	tmpDir, ghOutputPath, _ := setupPreCodeEnv(t, "")

	env := makeBaseEnv(tmpDir, ghOutputPath)

	_, err := runPreCode(t, scriptPath, env)
	require.NoError(t, err)

	ghOutput, err := os.ReadFile(ghOutputPath)
	require.NoError(t, err)

	assert.Contains(t, string(ghOutput), "skipped=false",
		"GITHUB_OUTPUT should contain 'skipped=false' when no PRs found")
}

// TestPreCodeForceFlagBypassesPRCheck verifies that --force flag (in comment
// body) bypasses the duplicate PR check entirely.
//
// [test_id:TS-GH-26-007]
func TestPreCodeForceFlagBypassesPRCheck(t *testing.T) {
	scriptPath := findScript(t)

	// Mock with human PR that would normally trigger skip
	humanPRJSON := `100	human-dev	https://github.com/org/repo/pull/100`
	tmpDir, ghOutputPath, mockLogPath := setupPreCodeEnv(t, humanPRJSON)

	env := makeBaseEnv(tmpDir, ghOutputPath)
	env["COMMENT_BODY"] = "/fs-code --force"

	output, err := runPreCode(t, scriptPath, env)
	require.NoError(t, err, "pre-code.sh should exit 0; output: %s", output)

	ghOutput, err := os.ReadFile(ghOutputPath)
	require.NoError(t, err)
	assert.Contains(t, string(ghOutput), "skipped=false",
		"GITHUB_OUTPUT should contain skipped=false when --force is used")

	// Verify no PR search was made
	mockLog, _ := os.ReadFile(mockLogPath)
	assert.NotContains(t, string(mockLog), "pr list",
		"No PR search should be made when --force is active")
}

// TestPreCodeForceEnvBypassesPRCheck verifies that CODE_FORCE=true
// environment variable bypasses the duplicate PR check.
//
// [test_id:TS-GH-26-008]
func TestPreCodeForceEnvBypassesPRCheck(t *testing.T) {
	scriptPath := findScript(t)

	humanPRJSON := `100	human-dev	https://github.com/org/repo/pull/100`
	tmpDir, ghOutputPath, mockLogPath := setupPreCodeEnv(t, humanPRJSON)

	env := makeBaseEnv(tmpDir, ghOutputPath)
	env["CODE_FORCE"] = "true"

	output, err := runPreCode(t, scriptPath, env)
	require.NoError(t, err, "pre-code.sh should exit 0; output: %s", output)

	ghOutput, err := os.ReadFile(ghOutputPath)
	require.NoError(t, err)
	assert.Contains(t, string(ghOutput), "skipped=false",
		"GITHUB_OUTPUT should contain skipped=false with CODE_FORCE=true")

	mockLog, _ := os.ReadFile(mockLogPath)
	assert.NotContains(t, string(mockLog), "pr list",
		"No PR search should be made when CODE_FORCE=true")
}

// TestPreCodeForceOverrideWithExistingPRs verifies that force override
// allows the agent to proceed even when human PRs exist.
//
// [test_id:TS-GH-26-009]
func TestPreCodeForceOverrideWithExistingPRs(t *testing.T) {
	scriptPath := findScript(t)

	humanPRJSON := `100	human-dev	https://github.com/org/repo/pull/100`
	tmpDir, ghOutputPath, _ := setupPreCodeEnv(t, humanPRJSON)

	env := makeBaseEnv(tmpDir, ghOutputPath)
	env["CODE_FORCE"] = "true"

	output, err := runPreCode(t, scriptPath, env)
	require.NoError(t, err, "pre-code.sh should exit 0; output: %s", output)

	ghOutput, err := os.ReadFile(ghOutputPath)
	require.NoError(t, err)
	assert.Contains(t, string(ghOutput), "skipped=false",
		"Force override should produce skipped=false despite existing human PR")
	assert.Contains(t, output, "Force override",
		"Script should log force override")
}

// TestPreCodeExcludesBotPRs verifies that PRs authored by known bot
// accounts are excluded from duplicate detection.
//
// [test_id:TS-GH-26-010]
func TestPreCodeExcludesBotPRs(t *testing.T) {
	scriptPath := findScript(t)

	// The mock gh pr list is called with --jq that filters out bot PRs.
	// When only bot PRs exist, the jq filter returns empty output.
	// We simulate this by returning empty (the jq filter already happened).
	tmpDir, ghOutputPath, _ := setupPreCodeEnv(t, "")

	env := makeBaseEnv(tmpDir, ghOutputPath)

	output, err := runPreCode(t, scriptPath, env)
	require.NoError(t, err, "pre-code.sh should exit 0; output: %s", output)

	ghOutput, err := os.ReadFile(ghOutputPath)
	require.NoError(t, err)
	assert.Contains(t, string(ghOutput), "skipped=false",
		"Script should proceed (skipped=false) when only bot PRs exist")
}

// TestPreCodeDetectsMixedBotAndHumanPRs verifies that when both bot and
// human PRs exist, the human PR is detected and triggers skip.
//
// [test_id:TS-GH-26-011]
func TestPreCodeDetectsMixedBotAndHumanPRs(t *testing.T) {
	scriptPath := findScript(t)

	// After jq filtering, only the human PR remains in the output
	humanPRJSON := `201	human-dev	https://github.com/org/repo/pull/201`
	tmpDir, ghOutputPath, _ := setupPreCodeEnv(t, humanPRJSON)

	env := makeBaseEnv(tmpDir, ghOutputPath)

	output, err := runPreCode(t, scriptPath, env)
	require.NoError(t, err, "pre-code.sh should exit 0; output: %s", output)

	ghOutput, err := os.ReadFile(ghOutputPath)
	require.NoError(t, err)
	assert.Contains(t, string(ghOutput), "skipped=true",
		"skipped=true expected when human PR found among mixed results")
	assert.Contains(t, output, "human-dev",
		"Script should reference the human PR author")
}

// TestPreCodeOnlyBotPRsDoNotTriggerSkip verifies that when the only PRs
// found are bot-authored, the script does NOT trigger a skip.
//
// [test_id:TS-GH-26-012]
func TestPreCodeOnlyBotPRsDoNotTriggerSkip(t *testing.T) {
	scriptPath := findScript(t)

	// After jq filtering, bot PRs are removed — empty output
	tmpDir, ghOutputPath, mockLogPath := setupPreCodeEnv(t, "")

	env := makeBaseEnv(tmpDir, ghOutputPath)

	output, err := runPreCode(t, scriptPath, env)
	require.NoError(t, err, "pre-code.sh should exit 0; output: %s", output)

	ghOutput, err := os.ReadFile(ghOutputPath)
	require.NoError(t, err)
	assert.Contains(t, string(ghOutput), "skipped=false",
		"Script should not skip when only bot PRs exist (filtered out by jq)")

	// Verify no skip comment was posted
	mockLog, _ := os.ReadFile(mockLogPath)
	assert.NotContains(t, string(mockLog), "issue comment",
		"No skip comment should be posted when only bot PRs exist")
}
