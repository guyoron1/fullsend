//go:build e2e

package tests

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Post-Retro Non-Fatal Error Handling Tests — GH-2305

STP Reference: outputs/stp/GH-2305/GH-2305_test_plan.md
STD Reference: outputs/std/GH-2305/GH-2305_test_description.yaml
Jira: GH-2305

These tests validate that post-retro.sh treats HTTP 401/403 errors on
comment posting as non-fatal (exit 0 with ::warning::) while keeping
other HTTP errors (500, 422) fatal.
*/

// postRetroScriptPath returns the path to post-retro.sh relative to the
// repository root. Tests must set the working directory or provide an
// absolute path.
func postRetroScriptPath(t *testing.T) string {
	t.Helper()
	// Walk up from the test binary location to find the repo root.
	// In CI this is set via REPO_ROOT; locally we probe for go.mod.
	root := os.Getenv("REPO_ROOT")
	if root == "" {
		// Fallback: find repo root by walking up from cwd.
		dir, err := os.Getwd()
		require.NoError(t, err)
		for {
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				root = dir
				break
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				t.Fatal("could not locate repo root (go.mod not found)")
			}
			dir = parent
		}
	}
	scriptPath := filepath.Join(root, "internal", "scaffold", "fullsend-repo", "scripts", "post-retro.sh")
	require.FileExists(t, scriptPath, "post-retro.sh must exist")
	return scriptPath
}

// agentResult builds a valid agent-result.json payload with the given proposals.
func agentResult(t *testing.T, summary string, proposals []map[string]string) []byte {
	t.Helper()
	result := map[string]interface{}{
		"summary":   summary,
		"proposals": proposals,
	}
	data, err := json.MarshalIndent(result, "", "  ")
	require.NoError(t, err)
	return data
}

// singleProposal returns one valid proposal for test fixtures.
func singleProposal(targetRepo string) map[string]string {
	return map[string]string{
		"target_repo":         targetRepo,
		"title":               "Refactor error handling in retro agent",
		"what_happened":       "The retro agent encountered an error.",
		"what_could_go_better": "Error handling could be more granular.",
		"proposed_change":     "Add non-fatal error path for auth failures.",
		"validation_criteria": "Script exits 0 on 401/403.",
	}
}

// setupTestDir creates a temporary directory with:
//   - iteration-0/output/agent-result.json (with given proposals)
//   - a mock gh binary at tmpDir/bin/gh
//
// Returns tmpDir path. Caller should defer os.RemoveAll(tmpDir).
func setupTestDir(t *testing.T, mockGhScript string, proposals []map[string]string) string {
	t.Helper()
	tmpDir := t.TempDir()

	// Create iteration output directory and agent-result.json.
	outputDir := filepath.Join(tmpDir, "iteration-0", "output")
	require.NoError(t, os.MkdirAll(outputDir, 0o755))

	summary := "Retro analysis complete. 1 proposal filed."
	if len(proposals) == 0 {
		summary = "Retro analysis complete. No proposals."
	}
	resultData := agentResult(t, summary, proposals)
	require.NoError(t, os.WriteFile(filepath.Join(outputDir, "agent-result.json"), resultData, 0o644))

	// Create mock gh binary.
	binDir := filepath.Join(tmpDir, "bin")
	require.NoError(t, os.MkdirAll(binDir, 0o755))
	ghPath := filepath.Join(binDir, "gh")
	require.NoError(t, os.WriteFile(ghPath, []byte(mockGhScript), 0o755))

	return tmpDir
}

// runPostRetro executes post-retro.sh in the given working directory with
// the mock gh on PATH. Returns stdout, stderr, and the exit code.
func runPostRetro(t *testing.T, scriptPath, workDir, binDir string, extraEnv ...string) (stdout, stderr string, exitCode int) {
	t.Helper()

	cmd := exec.Command("bash", scriptPath)
	cmd.Dir = workDir

	// Build environment: inherit minimal env, override PATH, set required vars.
	env := []string{
		fmt.Sprintf("PATH=%s:/usr/bin:/bin:/usr/sbin:/sbin", binDir),
		"GH_TOKEN=fake-token-for-testing",
		"ORIGINATING_URL=https://github.com/test-org/test-repo/pull/42",
		"HOME=" + os.Getenv("HOME"),
	}
	env = append(env, extraEnv...)
	cmd.Env = env

	var stdoutBuf, stderrBuf strings.Builder
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	exitCode = 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("unexpected error running post-retro.sh: %v", err)
		}
	}

	return stdoutBuf.String(), stderrBuf.String(), exitCode
}

// ================================================================
// Mock gh scripts
// ================================================================

// mockGh403 returns 403 on comment posting (gh api .../comments),
// succeeds on issue creation (gh issue create).
const mockGh403 = `#!/usr/bin/env bash
# Log all calls for inspection.
echo "$@" >> "${GH_CALL_LOG:-/dev/null}"

# Comment posting via gh api — return 403.
if [[ "$1" == "api" ]] && echo "$@" | grep -q "comments"; then
  echo "HTTP 403 - Resource not accessible by integration" >&2
  exit 1
fi

# Issue creation via gh issue create — succeed.
if [[ "$1" == "issue" ]] && [[ "$2" == "create" ]]; then
  echo "https://github.com/test-org/test-repo/issues/100"
  exit 0
fi

# Default: succeed.
echo '{}'
exit 0
`

// mockGh401 returns 401 on comment posting.
const mockGh401 = `#!/usr/bin/env bash
echo "$@" >> "${GH_CALL_LOG:-/dev/null}"

if [[ "$1" == "api" ]] && echo "$@" | grep -q "comments"; then
  echo "HTTP 401 - Bad credentials" >&2
  exit 1
fi

if [[ "$1" == "issue" ]] && [[ "$2" == "create" ]]; then
  echo "https://github.com/test-org/test-repo/issues/100"
  exit 0
fi

echo '{}'
exit 0
`

// mockGh500 returns 500 on comment posting.
const mockGh500 = `#!/usr/bin/env bash
echo "$@" >> "${GH_CALL_LOG:-/dev/null}"

if [[ "$1" == "api" ]] && echo "$@" | grep -q "comments"; then
  echo "HTTP 500 - Internal Server Error" >&2
  exit 1
fi

if [[ "$1" == "issue" ]] && [[ "$2" == "create" ]]; then
  echo "https://github.com/test-org/test-repo/issues/100"
  exit 0
fi

echo '{}'
exit 0
`

// mockGh422 returns 422 on comment posting.
const mockGh422 = `#!/usr/bin/env bash
echo "$@" >> "${GH_CALL_LOG:-/dev/null}"

if [[ "$1" == "api" ]] && echo "$@" | grep -q "comments"; then
  echo "HTTP 422 - Validation Failed" >&2
  exit 1
fi

if [[ "$1" == "issue" ]] && [[ "$2" == "create" ]]; then
  echo "https://github.com/test-org/test-repo/issues/100"
  exit 0
fi

echo '{}'
exit 0
`

// mockGhSuccess succeeds for all API calls and logs them.
const mockGhSuccess = `#!/usr/bin/env bash
echo "$@" >> "${GH_CALL_LOG:-/dev/null}"

if [[ "$1" == "issue" ]] && [[ "$2" == "create" ]]; then
  echo "https://github.com/test-org/test-repo/issues/100"
  exit 0
fi

# gh api — return valid JSON.
echo '{"id": 1, "html_url": "https://github.com/test-org/test-repo/issues/100#issuecomment-1"}'
exit 0
`

// mockGhLogging logs all calls in order and succeeds for everything.
const mockGhLogging = `#!/usr/bin/env bash
# Log the full command for ordering verification.
echo "$@" >> "${GH_CALL_LOG}"

if [[ "$1" == "issue" ]] && [[ "$2" == "create" ]]; then
  echo "https://github.com/test-org/test-repo/issues/100"
  exit 0
fi

echo '{"id": 1, "html_url": "https://github.com/test-org/test-repo/issues/100#issuecomment-1"}'
exit 0
`

// ================================================================
// Tests — Non-fatal 401/403 error handling
// ================================================================

func TestPostRetroNonFatalErrorHandling(t *testing.T) {
	scriptPath := postRetroScriptPath(t)

	// ----------------------------------------------------------------
	// TS-GH-2305-001: Verify 403 error on comment posting exits 0 with warning
	// ----------------------------------------------------------------
	t.Run("[test_id:TS-GH-2305-001] should exit 0 with warning when comment posting returns 403", func(t *testing.T) {
		proposals := []map[string]string{singleProposal("test-org/test-repo")}
		tmpDir := setupTestDir(t, mockGh403, proposals)
		binDir := filepath.Join(tmpDir, "bin")

		stdout, stderr, exitCode := runPostRetro(t, scriptPath, tmpDir, binDir)
		_ = stdout

		assert.Equal(t, 0, exitCode, "script should exit 0 on 403 comment-posting failure")
		assert.Contains(t, stderr, "::warning::", "stderr should contain a ::warning:: annotation")
		assert.Contains(t, stderr, "permissions", "warning should mention permissions")
	})

	// ----------------------------------------------------------------
	// TS-GH-2305-002: Verify 401 error on comment posting exits 0 with warning
	// ----------------------------------------------------------------
	t.Run("[test_id:TS-GH-2305-002] should exit 0 with warning when comment posting returns 401", func(t *testing.T) {
		proposals := []map[string]string{singleProposal("test-org/test-repo")}
		tmpDir := setupTestDir(t, mockGh401, proposals)
		binDir := filepath.Join(tmpDir, "bin")

		stdout, stderr, exitCode := runPostRetro(t, scriptPath, tmpDir, binDir)
		_ = stdout

		assert.Equal(t, 0, exitCode, "script should exit 0 on 401 comment-posting failure")
		assert.Contains(t, stderr, "::warning::", "stderr should contain a ::warning:: annotation")
		assert.Contains(t, stderr, "permissions", "warning should reference authentication/permission issue")
	})

	// ----------------------------------------------------------------
	// TS-GH-2305-003: Verify warning message contains repo and PR identifier
	// ----------------------------------------------------------------
	t.Run("[test_id:TS-GH-2305-003] should include repo and PR number in warning message", func(t *testing.T) {
		proposals := []map[string]string{singleProposal("test-org/test-repo")}
		tmpDir := setupTestDir(t, mockGh403, proposals)
		binDir := filepath.Join(tmpDir, "bin")

		_, stderr, exitCode := runPostRetro(t, scriptPath, tmpDir, binDir)

		require.Equal(t, 0, exitCode, "script should exit 0")

		// The warning annotation should include the originating repo and PR number
		// so operators can identify which comment was skipped.
		assert.Contains(t, stderr, "test-org/test-repo", "warning should contain repository identifier")
		assert.Contains(t, stderr, "42", "warning should contain PR number")
	})
}

// ================================================================
// Tests — Fatal error handling (must remain fatal)
// ================================================================

func TestPostRetroFatalErrorHandling(t *testing.T) {
	scriptPath := postRetroScriptPath(t)

	// ----------------------------------------------------------------
	// TS-GH-2305-004: Verify 500 error on comment posting remains fatal
	// ----------------------------------------------------------------
	t.Run("[test_id:TS-GH-2305-004] should exit non-zero when comment posting returns 500", func(t *testing.T) {
		proposals := []map[string]string{singleProposal("test-org/test-repo")}
		tmpDir := setupTestDir(t, mockGh500, proposals)
		binDir := filepath.Join(tmpDir, "bin")

		_, stderr, exitCode := runPostRetro(t, scriptPath, tmpDir, binDir)

		assert.NotEqual(t, 0, exitCode, "script should exit non-zero on 500 server error")
		assert.NotContains(t, stderr, "::warning::", "500 errors should not produce a ::warning:: annotation — they are fatal")
	})

	// ----------------------------------------------------------------
	// TS-GH-2305-005: Verify 422 error on comment posting remains fatal
	// ----------------------------------------------------------------
	t.Run("[test_id:TS-GH-2305-005] should exit non-zero when comment posting returns 422", func(t *testing.T) {
		proposals := []map[string]string{singleProposal("test-org/test-repo")}
		tmpDir := setupTestDir(t, mockGh422, proposals)
		binDir := filepath.Join(tmpDir, "bin")

		_, stderr, exitCode := runPostRetro(t, scriptPath, tmpDir, binDir)

		assert.NotEqual(t, 0, exitCode, "script should exit non-zero on 422 validation error")
		assert.NotContains(t, stderr, "::warning::", "422 errors should not produce a ::warning:: annotation — they are fatal")
	})
}

// ================================================================
// Tests — Happy-path behavior preserved
// ================================================================

func TestPostRetroHappyPath(t *testing.T) {
	scriptPath := postRetroScriptPath(t)

	// ----------------------------------------------------------------
	// TS-GH-2305-006: Verify successful comment posting with one proposal
	// ----------------------------------------------------------------
	t.Run("[test_id:TS-GH-2305-006] should post comment and exit 0 with one proposal", func(t *testing.T) {
		proposals := []map[string]string{singleProposal("test-org/test-repo")}
		tmpDir := setupTestDir(t, mockGhSuccess, proposals)
		binDir := filepath.Join(tmpDir, "bin")

		callLog := filepath.Join(tmpDir, "calls.log")

		stdout, _, exitCode := runPostRetro(t, scriptPath, tmpDir, binDir,
			fmt.Sprintf("GH_CALL_LOG=%s", callLog))

		assert.Equal(t, 0, exitCode, "script should exit 0 on success")

		// Verify the comment endpoint was called.
		logData, err := os.ReadFile(callLog)
		require.NoError(t, err, "call log should exist")
		logStr := string(logData)

		assert.Contains(t, logStr, "comments", "mock gh should have been called with the comments endpoint")
		_ = stdout
	})

	// ----------------------------------------------------------------
	// TS-GH-2305-007: Verify successful comment posting with no proposals
	// ----------------------------------------------------------------
	t.Run("[test_id:TS-GH-2305-007] should post comment and exit 0 with no proposals", func(t *testing.T) {
		proposals := []map[string]string{} // no proposals
		tmpDir := setupTestDir(t, mockGhSuccess, proposals)
		binDir := filepath.Join(tmpDir, "bin")

		callLog := filepath.Join(tmpDir, "calls.log")

		_, _, exitCode := runPostRetro(t, scriptPath, tmpDir, binDir,
			fmt.Sprintf("GH_CALL_LOG=%s", callLog))

		assert.Equal(t, 0, exitCode, "script should exit 0 even with no proposals")

		// Verify comment was still posted (summary-only comment).
		logData, err := os.ReadFile(callLog)
		require.NoError(t, err, "call log should exist")
		logStr := string(logData)
		assert.Contains(t, logStr, "comments", "comment should be posted even with zero proposals")
	})

	// ----------------------------------------------------------------
	// TS-GH-2305-008: Verify proposal issues created before comment posting
	// ----------------------------------------------------------------
	t.Run("[test_id:TS-GH-2305-008] should create proposal issues before posting comment", func(t *testing.T) {
		proposals := []map[string]string{singleProposal("test-org/test-repo")}
		tmpDir := setupTestDir(t, mockGhLogging, proposals)
		binDir := filepath.Join(tmpDir, "bin")

		callLog := filepath.Join(tmpDir, "calls.log")

		_, _, exitCode := runPostRetro(t, scriptPath, tmpDir, binDir,
			fmt.Sprintf("GH_CALL_LOG=%s", callLog))

		require.Equal(t, 0, exitCode, "script should exit 0")

		logData, err := os.ReadFile(callLog)
		require.NoError(t, err, "call log should exist")

		lines := strings.Split(strings.TrimSpace(string(logData)), "\n")

		// Find the indices of issue creation and comment posting calls.
		issueCreateIdx := -1
		commentIdx := -1
		for i, line := range lines {
			if strings.Contains(line, "issue") && strings.Contains(line, "create") {
				if issueCreateIdx == -1 {
					issueCreateIdx = i
				}
			}
			if strings.Contains(line, "comments") {
				commentIdx = i
			}
		}

		require.NotEqual(t, -1, issueCreateIdx, "issue creation call should appear in log")
		require.NotEqual(t, -1, commentIdx, "comment posting call should appear in log")
		assert.Less(t, issueCreateIdx, commentIdx,
			"issue creation (line %d) should precede comment posting (line %d)", issueCreateIdx, commentIdx)
	})
}

// ================================================================
// Tests — Non-fatal error across proposal states
// ================================================================

func TestPostRetroNonFatalAcrossProposalStates(t *testing.T) {
	scriptPath := postRetroScriptPath(t)

	// ----------------------------------------------------------------
	// TS-GH-2305-009: Verify 403 with no proposals still exits 0
	// ----------------------------------------------------------------
	t.Run("[test_id:TS-GH-2305-009] should exit 0 on 403 with no proposals", func(t *testing.T) {
		proposals := []map[string]string{} // no proposals
		tmpDir := setupTestDir(t, mockGh403, proposals)
		binDir := filepath.Join(tmpDir, "bin")

		_, stderr, exitCode := runPostRetro(t, scriptPath, tmpDir, binDir)

		assert.Equal(t, 0, exitCode, "script should exit 0 despite 403 and no proposals")
		assert.Contains(t, stderr, "::warning::", "warning annotation should still be emitted")
	})

	// ----------------------------------------------------------------
	// TS-GH-2305-010: Verify 403 with multiple proposals still exits 0
	// ----------------------------------------------------------------
	t.Run("[test_id:TS-GH-2305-010] should exit 0 on 403 with multiple proposals and create all proposals", func(t *testing.T) {
		proposals := []map[string]string{
			singleProposal("test-org/test-repo"),
			{
				"target_repo":          "test-org/test-repo",
				"title":                "Improve logging in retro agent",
				"what_happened":        "Logs were insufficient for debugging.",
				"what_could_go_better": "Add structured logging.",
				"proposed_change":      "Use slog for all log output.",
				"validation_criteria":  "All log lines are structured JSON.",
			},
			{
				"target_repo":          "test-org/test-repo",
				"title":                "Add retry logic for transient failures",
				"what_happened":        "Transient API failures caused run failure.",
				"what_could_go_better": "Retry on 5xx errors with backoff.",
				"proposed_change":      "Add retry wrapper around gh API calls.",
				"validation_criteria":  "Retries up to 3 times on 5xx.",
			},
		}
		tmpDir := setupTestDir(t, mockGh403, proposals)
		binDir := filepath.Join(tmpDir, "bin")

		callLog := filepath.Join(tmpDir, "calls.log")

		_, stderr, exitCode := runPostRetro(t, scriptPath, tmpDir, binDir,
			fmt.Sprintf("GH_CALL_LOG=%s", callLog))

		assert.Equal(t, 0, exitCode, "script should exit 0 despite 403 on comment posting")
		assert.Contains(t, stderr, "::warning::", "warning annotation should be emitted for comment failure")

		// Verify all 3 proposal issues were created.
		logData, err := os.ReadFile(callLog)
		require.NoError(t, err, "call log should exist")
		logStr := string(logData)

		issueCreateCount := strings.Count(logStr, "issue create")
		assert.Equal(t, 3, issueCreateCount,
			"all 3 proposal issues should have been created; got %d", issueCreateCount)
	})

	// ----------------------------------------------------------------
	// TS-GH-2305-011: Verify completion message on successful run
	// ----------------------------------------------------------------
	t.Run("[test_id:TS-GH-2305-011] should print completion message on successful run", func(t *testing.T) {
		proposals := []map[string]string{singleProposal("test-org/test-repo")}
		tmpDir := setupTestDir(t, mockGhSuccess, proposals)
		binDir := filepath.Join(tmpDir, "bin")

		stdout, _, exitCode := runPostRetro(t, scriptPath, tmpDir, binDir)

		require.Equal(t, 0, exitCode, "script should exit 0")
		assert.Contains(t, stdout, "Post-retro complete",
			"stdout should contain the 'Post-retro complete' completion message")
	})
}
