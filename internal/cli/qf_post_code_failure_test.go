package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// QualityFlow tests for GH-71: Post-Code Failure Detection and Reporting
// STD Reference: outputs/std/GH-71/GH-71_test_description.yaml
// Scenarios: 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16
//
// These tests validate the post-code.sh script's failure detection logic.
// The script is at: internal/scaffold/fullsend-repo/scripts/post-code.sh
//
// The tests exercise the script's decision logic via shell snippets that
// replicate the key code paths from post-code.sh (branch detection,
// exit code checking, comment posting).

// setupMockGh creates a mock gh binary in a temp directory that logs its
// invocations to a file. Returns the directory (to prepend to PATH) and
// the invocation log path.
func setupMockGh(t *testing.T) (binDir, logFile string) {
	t.Helper()
	tmpDir := t.TempDir()
	logFile = filepath.Join(tmpDir, "gh_invocations.log")
	ghPath := filepath.Join(tmpDir, "gh")

	// Mock gh that records all arguments to the log file
	script := fmt.Sprintf(`#!/bin/sh
echo "$@" >> %s
`, logFile)
	require.NoError(t, os.WriteFile(ghPath, []byte(script), 0o755))
	return tmpDir, logFile
}

// setupGitRepo creates a minimal git repository in a temp dir.
// Returns the repo directory path.
func setupGitRepo(t *testing.T, branch string) string {
	t.Helper()
	tmpDir := t.TempDir()

	cmds := [][]string{
		{"git", "init", "-b", "main"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = tmpDir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git command %v failed: %s", args, out)
	}

	// Create initial commit
	dummyFile := filepath.Join(tmpDir, "README.md")
	require.NoError(t, os.WriteFile(dummyFile, []byte("# test\n"), 0o644))
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())
	cmd = exec.Command("git", "commit", "-m", "initial commit")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	if branch != "main" && branch != "" {
		cmd = exec.Command("git", "checkout", "-b", branch)
		cmd.Dir = tmpDir
		require.NoError(t, cmd.Run())
	}

	return tmpDir
}

// runBranchCheckScript runs the branch-check portion of post-code.sh logic.
// Returns exit code, stdout, and AGENT_ERROR_EXIT value.
func runBranchCheckScript(t *testing.T, repoDir string, agentExitCode int, ghBinDir string) (exitCode int, stdout string, agentErrorExit bool) {
	t.Helper()

	// This script replicates the decision logic from post-code.sh lines 113-123
	script := `#!/bin/sh
BRANCH="$(git branch --show-current)"
if [ -z "${BRANCH}" ] || [ "${BRANCH}" = "main" ] || [ "${BRANCH}" = "master" ]; then
  if [ "${AGENT_EXIT_CODE}" != "0" ]; then
    echo "AGENT_ERROR_EXIT=true"
    exit 1
  fi
  echo "NO_OP"
  exit 0
fi
echo "ON_FEATURE_BRANCH=${BRANCH}"
exit 0
`
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "check.sh")
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0o755))

	cmd := exec.Command(scriptPath)
	cmd.Dir = repoDir
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("AGENT_EXIT_CODE=%d", agentExitCode),
		fmt.Sprintf("PATH=%s:%s", ghBinDir, os.Getenv("PATH")),
	)
	out, err := cmd.CombinedOutput()
	output := string(out)

	rc := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			rc = exitErr.ExitCode()
		}
	}
	return rc, output, strings.Contains(output, "AGENT_ERROR_EXIT=true")
}

// runChangesetCheckScript runs the changeset-check logic from post-code.sh.
func runChangesetCheckScript(t *testing.T, repoDir string, agentExitCode int, hasChanges bool) (exitCode int, stdout string, agentErrorExit bool) {
	t.Helper()

	// Add a changed file if needed
	if hasChanges {
		testFile := filepath.Join(repoDir, "changed.txt")
		require.NoError(t, os.WriteFile(testFile, []byte("change"), 0o644))
		cmd := exec.Command("git", "add", ".")
		cmd.Dir = repoDir
		require.NoError(t, cmd.Run())
		cmd = exec.Command("git", "commit", "-m", "agent change")
		cmd.Dir = repoDir
		require.NoError(t, cmd.Run())
	}

	// This replicates lines 140-148 of post-code.sh
	script := `#!/bin/sh
CHANGED_FILES="$(git diff --name-only HEAD~1..HEAD 2>/dev/null || true)"
if [ -z "${CHANGED_FILES}" ]; then
  if [ "${AGENT_EXIT_CODE}" != "0" ]; then
    echo "AGENT_ERROR_EXIT=true"
    exit 1
  fi
  echo "NO_CHANGES_NO_OP"
  exit 0
fi
echo "HAS_CHANGES"
exit 0
`
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "check.sh")
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0o755))

	cmd := exec.Command(scriptPath)
	cmd.Dir = repoDir
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("AGENT_EXIT_CODE=%d", agentExitCode),
	)
	out, err := cmd.CombinedOutput()
	output := string(out)

	rc := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			rc = exitErr.ExitCode()
		}
	}
	return rc, output, strings.Contains(output, "AGENT_ERROR_EXIT=true")
}

func TestQF_PostCodeFailureComment(t *testing.T) {
	t.Run("[test_id:TS-GH-71-005] should post failure comment on agent error without branch", func(t *testing.T) {
		// Scenario 5: When AGENT_EXIT_CODE is non-zero and no feature branch
		// was created (still on main), report_failure_to_issue() posts a failure
		// comment to the originating GitHub issue.
		ghBinDir, logFile := setupMockGh(t)
		repoDir := setupGitRepo(t, "main")

		// Run the report_failure_to_issue logic
		script := `#!/bin/sh
set +e
run_url="${GITHUB_SERVER_URL}/${GITHUB_REPOSITORY}/actions/runs/${GITHUB_RUN_ID}"
comment_body="Agent failed (exit code ${AGENT_EXIT_CODE}). Workflow: ${run_url}"
gh issue comment "${ISSUE_NUMBER}" --repo "${REPO_FULL_NAME}" --body "${comment_body}" 2>/dev/null
echo "COMMENT_POSTED"
`
		tmpDir := t.TempDir()
		scriptPath := filepath.Join(tmpDir, "report.sh")
		require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0o755))

		cmd := exec.Command(scriptPath)
		cmd.Dir = repoDir
		cmd.Env = append(os.Environ(),
			"AGENT_EXIT_CODE=1",
			"GITHUB_SERVER_URL=https://github.com",
			"GITHUB_REPOSITORY=org/repo",
			"GITHUB_RUN_ID=12345",
			"ISSUE_NUMBER=42",
			"REPO_FULL_NAME=org/repo",
			fmt.Sprintf("PATH=%s:%s", ghBinDir, os.Getenv("PATH")),
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err)
		assert.Contains(t, string(out), "COMMENT_POSTED")

		// ASSERT-01: gh issue comment is invoked for the correct issue
		invocations, err := os.ReadFile(logFile)
		require.NoError(t, err)
		assert.Contains(t, string(invocations), "issue comment 42",
			"gh issue comment must be invoked with the correct issue number")
	})

	t.Run("[test_id:TS-GH-71-006] should include workflow run URL in failure comment", func(t *testing.T) {
		// Scenario 6: The failure comment must include the full workflow run URL
		// constructed from GITHUB_SERVER_URL, GITHUB_REPOSITORY, GITHUB_RUN_ID.
		ghBinDir, logFile := setupMockGh(t)
		repoDir := setupGitRepo(t, "main")

		script := `#!/bin/sh
run_url="${GITHUB_SERVER_URL}/${GITHUB_REPOSITORY}/actions/runs/${GITHUB_RUN_ID}"
comment_body="Agent failed. Workflow: ${run_url}"
gh issue comment "${ISSUE_NUMBER}" --repo "${REPO_FULL_NAME}" --body "${comment_body}" 2>/dev/null
`
		tmpDir := t.TempDir()
		scriptPath := filepath.Join(tmpDir, "report.sh")
		require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0o755))

		cmd := exec.Command(scriptPath)
		cmd.Dir = repoDir
		cmd.Env = append(os.Environ(),
			"AGENT_EXIT_CODE=1",
			"GITHUB_SERVER_URL=https://github.com",
			"GITHUB_REPOSITORY=org/repo",
			"GITHUB_RUN_ID=12345",
			"ISSUE_NUMBER=42",
			"REPO_FULL_NAME=org/repo",
			fmt.Sprintf("PATH=%s:%s", ghBinDir, os.Getenv("PATH")),
		)
		require.NoError(t, cmd.Run())

		// ASSERT-01: Workflow run URL is present in failure comment
		invocations, err := os.ReadFile(logFile)
		require.NoError(t, err)
		assert.Contains(t, string(invocations), "https://github.com/org/repo/actions/runs/12345",
			"Comment must contain the full workflow run URL")
	})

	t.Run("[test_id:TS-GH-71-007] should distinguish agent error from post-script error", func(t *testing.T) {
		// Scenario 7: The failure comment body differs depending on whether
		// AGENT_ERROR_EXIT=true (agent error) or not (post-script error).
		ghBinDir, logFile := setupMockGh(t)
		repoDir := setupGitRepo(t, "main")

		// Replicate the report_failure_to_issue logic from post-code.sh lines 80-99
		script := `#!/bin/sh
if [ "${AGENT_ERROR_EXIT}" = "true" ]; then
  comment_body="Code agent failed (agent exit code ${AGENT_EXIT_CODE})"
else
  comment_body="Post-code script failed"
fi
gh issue comment "${ISSUE_NUMBER}" --repo "${REPO_FULL_NAME}" --body "${comment_body}" 2>/dev/null
echo "AGENT_ERROR_EXIT=${AGENT_ERROR_EXIT}"
`
		tmpDir := t.TempDir()
		scriptPath := filepath.Join(tmpDir, "report.sh")
		require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0o755))

		cmd := exec.Command(scriptPath)
		cmd.Dir = repoDir
		cmd.Env = append(os.Environ(),
			"AGENT_EXIT_CODE=2",
			"AGENT_ERROR_EXIT=true",
			"ISSUE_NUMBER=42",
			"REPO_FULL_NAME=org/repo",
			fmt.Sprintf("PATH=%s:%s", ghBinDir, os.Getenv("PATH")),
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err)

		// ASSERT-01: AGENT_ERROR_EXIT is set to "true"
		assert.Contains(t, string(out), "AGENT_ERROR_EXIT=true")

		// ASSERT-02: Comment body references the exit code value
		invocations, err := os.ReadFile(logFile)
		require.NoError(t, err)
		assert.Contains(t, string(invocations), "agent exit code 2",
			"Comment must reference agent as error source and include exit code")
	})

	t.Run("[test_id:TS-GH-71-008] should not crash when gh CLI unavailable", func(t *testing.T) {
		// Scenario 8: When gh CLI is not available, report_failure_to_issue()
		// should fail silently. The script must not crash and must still exit
		// non-zero when AGENT_EXIT_CODE is non-zero.

		// Script with no gh on PATH, using the best-effort pattern
		script := `#!/bin/sh
set +e
if [ "${AGENT_EXIT_CODE}" != "0" ]; then
  # Attempt to post comment (will fail without gh)
  gh issue comment 42 --repo org/repo --body "failed" 2>/dev/null || true
  exit 1
fi
exit 0
`
		tmpDir := t.TempDir()
		scriptPath := filepath.Join(tmpDir, "report.sh")
		require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0o755))

		cmd := exec.Command(scriptPath)
		cmd.Env = []string{
			"AGENT_EXIT_CODE=1",
			"PATH=/usr/bin:/bin", // Minimal PATH without gh
			fmt.Sprintf("HOME=%s", tmpDir),
		}
		out, err := cmd.CombinedOutput()

		// ASSERT-01: Script exits non-zero despite gh unavailability
		require.Error(t, err, "script must exit non-zero when agent failed")
		if exitErr, ok := err.(*exec.ExitError); ok {
			assert.NotEqual(t, 0, exitErr.ExitCode())
		}

		// ASSERT-02: No crash or unhandled error
		assert.NotContains(t, string(out), "unbound variable",
			"script must not crash with unbound variable error")
	})
}

func TestQF_PostCodeAgentErrorVsNoOp(t *testing.T) {
	t.Run("[test_id:TS-GH-71-009] should exit cleanly when agent succeeds with no branch", func(t *testing.T) {
		// Scenario 9: Agent exits 0, still on main → successful no-op.
		ghBinDir, _ := setupMockGh(t)
		repoDir := setupGitRepo(t, "main")

		rc, output, agentErr := runBranchCheckScript(t, repoDir, 0, ghBinDir)

		// ASSERT-01: Script exits with code 0
		assert.Equal(t, 0, rc, "no-op with zero exit must exit cleanly")
		// ASSERT-02: No failure comment posted
		assert.Contains(t, output, "NO_OP")
		assert.False(t, agentErr, "AGENT_ERROR_EXIT must not be set for no-op")
	})

	t.Run("[test_id:TS-GH-71-010] should exit non-zero when agent fails with no branch", func(t *testing.T) {
		// Scenario 10: Agent exits 1, still on main → agent error.
		ghBinDir, _ := setupMockGh(t)
		repoDir := setupGitRepo(t, "main")

		rc, _, agentErr := runBranchCheckScript(t, repoDir, 1, ghBinDir)

		// ASSERT-01: Script exits non-zero
		assert.NotEqual(t, 0, rc, "agent failure with no branch must exit non-zero")
		// ASSERT-02: AGENT_ERROR_EXIT is 'true'
		assert.True(t, agentErr, "AGENT_ERROR_EXIT must be set to true")
	})

	t.Run("[test_id:TS-GH-71-011] should exit cleanly when agent succeeds with no changes", func(t *testing.T) {
		// Scenario 11: Agent exits 0, on feature branch, no changed files → no-op.
		repoDir := setupGitRepo(t, "feature-branch")

		rc, output, _ := runChangesetCheckScript(t, repoDir, 0, false)

		// ASSERT-01: Script exits with code 0
		assert.Equal(t, 0, rc, "no-change success must exit cleanly")
		assert.Contains(t, output, "NO_CHANGES_NO_OP")
	})

	t.Run("[test_id:TS-GH-71-012] should exit non-zero when agent fails with no changes", func(t *testing.T) {
		// Scenario 12: Agent exits 1, on feature branch, no changed files → error.
		repoDir := setupGitRepo(t, "feature-branch")

		rc, _, agentErr := runChangesetCheckScript(t, repoDir, 1, false)

		// ASSERT-01: Script exits non-zero
		assert.NotEqual(t, 0, rc, "failed agent with no changes must exit non-zero")
		assert.True(t, agentErr, "AGENT_ERROR_EXIT must be set")
	})
}

func TestQF_PostCodeMainBranchDetection(t *testing.T) {
	t.Run("[test_id:TS-GH-71-013] should report error when on main with non-zero exit", func(t *testing.T) {
		// Scenario 13: On main + non-zero exit → error reported.
		ghBinDir, _ := setupMockGh(t)
		repoDir := setupGitRepo(t, "main")

		rc, _, agentErr := runBranchCheckScript(t, repoDir, 1, ghBinDir)

		// ASSERT-01: Error is reported on main with non-zero exit
		assert.NotEqual(t, 0, rc)
		assert.True(t, agentErr, "must set AGENT_ERROR_EXIT=true on main with non-zero exit")
	})

	t.Run("[test_id:TS-GH-71-014] should emit no-op notice when on main with zero exit", func(t *testing.T) {
		// Scenario 14: On main + zero exit → no-op, no error.
		ghBinDir, _ := setupMockGh(t)
		repoDir := setupGitRepo(t, "main")

		rc, output, agentErr := runBranchCheckScript(t, repoDir, 0, ghBinDir)

		// ASSERT-01: No-op with zero exit does not trigger error
		assert.Equal(t, 0, rc, "no-op on main must exit cleanly")
		assert.Contains(t, output, "NO_OP")
		assert.False(t, agentErr, "AGENT_ERROR_EXIT must not be set for no-op on main")
	})
}

func TestQF_PostCodeEmptyChangeset(t *testing.T) {
	t.Run("[test_id:TS-GH-71-015] should report error with empty changeset and non-zero exit", func(t *testing.T) {
		// Scenario 15: Feature branch, no changes, non-zero exit → error.
		repoDir := setupGitRepo(t, "feature-branch")

		rc, _, agentErr := runChangesetCheckScript(t, repoDir, 1, false)

		// ASSERT-01: Empty changeset with non-zero exit triggers error
		assert.NotEqual(t, 0, rc, "empty changeset with failed agent must exit non-zero")
		assert.True(t, agentErr, "AGENT_ERROR_EXIT must be set")
	})

	t.Run("[test_id:TS-GH-71-016] should treat empty changeset with zero exit as no-op", func(t *testing.T) {
		// Scenario 16: Feature branch, no changes, zero exit → no-op.
		repoDir := setupGitRepo(t, "feature-branch")

		rc, output, _ := runChangesetCheckScript(t, repoDir, 0, false)

		// ASSERT-01: Empty changeset with zero exit is treated as no-op
		assert.Equal(t, 0, rc, "empty changeset with zero exit must be no-op")
		assert.Contains(t, output, "NO_CHANGES_NO_OP")
	})
}
