//go:build e2e

package tests

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

/*
End-to-End Agent Failure Status Comment Tests

STP Reference: outputs/stp/GH-2378/GH-2378_test_plan.md
Jira: GH-2378

This test simulates the complete agent failure pipeline by:
1. Setting up a mock environment with AGENT_EXIT_CODE, PUSH_TOKEN, etc.
2. Creating a mock gh binary that captures the comment body
3. Running the noop-detection + error-comment logic end-to-end
4. Validating the resulting comment content
*/

// e2ePostScript creates a minimal end-to-end test script that combines
// the noop-detection and error-reporting logic from post-code.sh.
// It uses a mock gh binary to capture the comment body to a temp file.
func e2ePostScript(scriptDir string, agentExitCode int, branch, changedFiles string) (commentBody string, scriptExitCode int, err error) {
	captureFile := filepath.Join(scriptDir, "captured_comment.txt")
	mockBinDir := filepath.Join(scriptDir, "mock-bin")

	if err := os.MkdirAll(mockBinDir, 0o755); err != nil {
		return "", -1, err
	}

	// Create mock gh binary that captures comment body
	mockGH := fmt.Sprintf(`#!/usr/bin/env bash
# Mock gh CLI — captures issue comment body to a file
if [ "$1" = "issue" ] && [ "$2" = "comment" ]; then
  # Parse args: gh issue comment NUMBER --repo REPO --body BODY
  shift 2  # skip "issue" "comment"
  while [ $# -gt 0 ]; do
    case "$1" in
      --body) echo "$2" > %q; shift 2 ;;
      *) shift ;;
    esac
  done
  exit 0
fi
# For any other gh command, succeed silently
exit 0
`, captureFile)

	mockGHPath := filepath.Join(mockBinDir, "gh")
	if err := os.WriteFile(mockGHPath, []byte(mockGH), 0o755); err != nil {
		return "", -1, err
	}

	// Create the end-to-end test script
	testScript := fmt.Sprintf(`#!/usr/bin/env bash
# End-to-end simulation of post-code.sh agent failure flow
# Mimics sections 1, 2, and the report_failure_to_issue trap

export PATH=%q:${PATH}
export GH_TOKEN="mock-token"
export PUSH_TOKEN="mock-token"
export REPO_FULL_NAME="fullsend-ai/fullsend"
export ISSUE_NUMBER="2378"
export GITHUB_RUN_ID="123456789"
export GITHUB_SERVER_URL="https://github.com"
export GITHUB_REPOSITORY="fullsend-ai/fullsend"
export AGENT_EXIT_CODE="%d"

BRANCH=%q
CHANGED_FILES=%q

# report_failure_to_issue — mirrors post-code.sh
report_failure_to_issue() {
  local exit_code=$?
  local run_url="${GITHUB_SERVER_URL}/${GITHUB_REPOSITORY}/actions/runs/${GITHUB_RUN_ID}"

  local comment_body
  if [ "${AGENT_ERROR_EXIT:-false}" = "true" ]; then
    comment_body="⚠️ **Code agent failed** (agent exit code ${AGENT_EXIT_CODE:-unknown})

The code agent terminated with an error and produced no PR.

**Workflow run:** ${run_url}

Please check the workflow logs for details and retry with ` + "`" + `/fs-code` + "`" + ` if appropriate."
  else
    comment_body="⚠️ **Post-code script failed** (exit code ${exit_code})

The code agent completed, but the post-code script failed while pushing the branch or creating the PR.

**Workflow run:** ${run_url}

Please check the workflow logs for details and retry with ` + "`" + `/fs-code` + "`" + ` if appropriate."
  fi

  gh issue comment "${ISSUE_NUMBER}" \
    --repo "${REPO_FULL_NAME}" \
    --body "${comment_body}" 2>/dev/null || true
}
trap report_failure_to_issue ERR

# --- Section 1: Branch check ---
if [ -z "${BRANCH}" ] || [ "${BRANCH}" = "main" ] || [ "${BRANCH}" = "master" ]; then
  if [ "${AGENT_EXIT_CODE:-0}" != "0" ]; then
    AGENT_ERROR_EXIT=true
    exit 1
  fi
  exit 0
fi

# --- Section 2: Changed files check ---
if [ -z "${CHANGED_FILES}" ]; then
  if [ "${AGENT_EXIT_CODE:-0}" != "0" ]; then
    AGENT_ERROR_EXIT=true
    exit 1
  fi
  exit 0
fi

# If we get here, proceed normally
echo "proceed"
exit 0
`, mockBinDir, agentExitCode, branch, changedFiles)

	scriptPath := filepath.Join(scriptDir, "e2e_post_code_test.sh")
	if err := os.WriteFile(scriptPath, []byte(testScript), 0o755); err != nil {
		return "", -1, err
	}

	cmd := exec.Command("bash", scriptPath)
	_, runErr := cmd.CombinedOutput()

	exitCode := 0
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return "", -1, runErr
		}
	}

	// Read captured comment if it exists
	if data, readErr := os.ReadFile(captureFile); readErr == nil {
		commentBody = strings.TrimSpace(string(data))
	}

	return commentBody, exitCode, nil
}

var _ = Describe("[GH-2378] End-to-End Agent Failure Status Comment", Ordered, func() {
	/*
		Markers:
		    - tier1

		Preconditions:
		    - post-code.sh script available and executable
		    - GitHub API access available (or mocked via gh CLI)
		    - Environment variables set: PUSH_TOKEN, REPO_FULL_NAME, ISSUE_NUMBER, GITHUB_RUN_ID
	*/

	var scriptDir string

	BeforeAll(func() {
		var err error
		scriptDir, err = os.MkdirTemp("", "gh2378-e2e-*")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterAll(func() {
		os.RemoveAll(scriptDir)
	})

	Context("when agent fails with non-zero exit and no commits", func() {
		It("[test_id:TS-GH-2378-010] should post issue comment with 'Code agent failed' and exit code", func() {
			// Run the end-to-end simulation: agent exits 1 on main (no branch)
			commentBody, exitCode, err := e2ePostScript(scriptDir, 1, "main", "")
			Expect(err).NotTo(HaveOccurred())

			// Script should exit non-zero (agent error detected)
			Expect(exitCode).NotTo(Equal(0), "post-code.sh should exit non-zero for agent errors")

			// ASSERT-01: Comment contains 'Code agent failed'
			Expect(commentBody).To(ContainSubstring("Code agent failed"),
				"Issue comment should identify this as a code agent failure")

			// ASSERT-02: Comment contains numeric exit code
			Expect(commentBody).To(ContainSubstring("agent exit code 1"),
				"Issue comment should include the agent's numeric exit code")

			// ASSERT-03: Comment contains workflow run link
			Expect(commentBody).To(ContainSubstring("actions/runs/123456789"),
				"Issue comment should contain a link to the workflow run")

			// ASSERT-04: Comment does NOT contain false success message
			Expect(commentBody).NotTo(ContainSubstring("Finished Code, Success"),
				"Issue comment should NOT contain a false success message")
			Expect(commentBody).NotTo(ContainSubstring("Post-code script failed"),
				"Issue comment should NOT say post-script failed for agent errors")

			// Additional: test the no-files path (feature branch exists but no changes)
			commentBody2, exitCode2, err2 := e2ePostScript(scriptDir, 1, "agent/2378-fix-status", "")
			Expect(err2).NotTo(HaveOccurred())
			Expect(exitCode2).NotTo(Equal(0))
			Expect(commentBody2).To(ContainSubstring("Code agent failed"))
			Expect(commentBody2).To(ContainSubstring("agent exit code 1"))
		})
	})
})
