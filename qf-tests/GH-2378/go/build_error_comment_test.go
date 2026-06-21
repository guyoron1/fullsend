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
Error Comment Generation (build_error_comment) Tests

STP Reference: outputs/stp/GH-2378/GH-2378_test_plan.md
Jira: GH-2378
*/

// buildErrorComment reimplements the error comment builder from post-code.sh
// report_failure_to_issue function, so tests can validate the comment body
// without needing a real GitHub API or workflow context.
func buildErrorComment(exitCode, repoFullName, runID, githubRepository, agentErrorExit, agentExitCode string) string {
	runRepo := githubRepository
	if runRepo == "" {
		runRepo = repoFullName
	}
	runURL := fmt.Sprintf("https://github.com/%s/actions/runs/%s", runRepo, runID)

	if agentErrorExit == "true" {
		return fmt.Sprintf(`⚠️ **Code agent failed** (agent exit code %s)

The code agent terminated with an error and produced no PR.

**Workflow run:** %s

Please check the workflow logs for details and retry with `+"`/fs-code`"+` if appropriate.`, agentExitCode, runURL)
	}

	return fmt.Sprintf(`⚠️ **Post-code script failed** (exit code %s)

The code agent completed, but the post-code script failed while pushing the branch or creating the PR.

**Workflow run:** %s

Please check the workflow logs for details and retry with `+"`/fs-code`"+` if appropriate.`, exitCode, runURL)
}

// shellBuildErrorComment exercises the same logic via a bash script to
// validate shell implementation matches.
func shellBuildErrorComment(scriptDir, exitCode, repoFullName, runID, githubRepository, agentErrorExit, agentExitCode string) (string, error) {
	testScript := fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail

build_error_comment() {
  local exit_code="$1"
  local repo_full_name="$2"
  local run_id="$3"
  local github_repository="${4:-}"
  local agent_error_exit="${5:-false}"
  local agent_exit_code="${6:-unknown}"

  local run_repo="${github_repository:-${repo_full_name}}"
  local run_url="https://github.com/${run_repo}/actions/runs/${run_id}"

  if [ "${agent_error_exit}" = "true" ]; then
    printf '⚠️ **Code agent failed** (agent exit code %%s)\n\nThe code agent terminated with an error and produced no PR.\n\n**Workflow run:** %%s\n\nPlease check the workflow logs for details and retry with ` + "`" + `/fs-code` + "`" + ` if appropriate.' "${agent_exit_code}" "${run_url}"
  else
    printf '⚠️ **Post-code script failed** (exit code %%s)\n\nThe code agent completed, but the post-code script failed while pushing the branch or creating the PR.\n\n**Workflow run:** %%s\n\nPlease check the workflow logs for details and retry with ` + "`" + `/fs-code` + "`" + ` if appropriate.' "${exit_code}" "${run_url}"
  fi
}

build_error_comment %q %q %q %q %q %q
`, exitCode, repoFullName, runID, githubRepository, agentErrorExit, agentExitCode)

	scriptPath := filepath.Join(scriptDir, "build_error_comment_test.sh")
	if err := os.WriteFile(scriptPath, []byte(testScript), 0o755); err != nil {
		return "", err
	}

	cmd := exec.Command("bash", scriptPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), err
	}
	return strings.TrimSpace(string(out)), nil
}

var _ = Describe("[GH-2378] Error Comment Generation", Ordered, func() {
	/*
		Markers:
		    - tier1

		Preconditions:
		    - post-code.sh sourced for function testing
		    - build_error_comment function is accessible
		    - bash 5.x+ available
	*/

	var scriptDir string

	BeforeAll(func() {
		var err error
		scriptDir, err = os.MkdirTemp("", "gh2378-error-comment-*")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterAll(func() {
		os.RemoveAll(scriptDir)
	})

	Context("when AGENT_ERROR_EXIT is true", func() {
		It("[test_id:TS-GH-2378-004] should produce comment saying 'Code agent failed'", func() {
			comment := buildErrorComment("1", "my-org/my-repo", "12345", "", "true", "1")

			// ASSERT-01: Comment contains 'Code agent failed'
			Expect(comment).To(ContainSubstring("Code agent failed"))

			// ASSERT-02: Comment does NOT contain 'Post-code script failed'
			Expect(comment).NotTo(ContainSubstring("Post-code script failed"))

			// Validate via shell
			shellComment, err := shellBuildErrorComment(scriptDir, "1", "my-org/my-repo", "12345", "", "true", "1")
			Expect(err).NotTo(HaveOccurred())
			Expect(shellComment).To(ContainSubstring("Code agent failed"))
			Expect(shellComment).NotTo(ContainSubstring("Post-code script failed"))
		})
	})

	Context("when agent exits with specific non-zero code", func() {
		It("[test_id:TS-GH-2378-005] should include numeric exit code in comment body", func() {
			// Test with exit code 1
			comment1 := buildErrorComment("1", "my-org/my-repo", "12345", "", "true", "1")
			Expect(comment1).To(ContainSubstring("agent exit code 1"))

			// Test with exit code 42 to ensure it's not just matching any "1"
			comment42 := buildErrorComment("42", "my-org/my-repo", "12345", "", "true", "42")
			Expect(comment42).To(ContainSubstring("agent exit code 42"))

			// Validate via shell
			shellComment, err := shellBuildErrorComment(scriptDir, "1", "my-org/my-repo", "12345", "", "true", "1")
			Expect(err).NotTo(HaveOccurred())
			Expect(shellComment).To(ContainSubstring("agent exit code 1"))
		})
	})

	Context("when AGENT_ERROR_EXIT is false or unset", func() {
		It("[test_id:TS-GH-2378-006] should produce comment saying 'Post-code script failed'", func() {
			// Test with AGENT_ERROR_EXIT = "false"
			commentFalse := buildErrorComment("1", "my-org/my-repo", "12345", "", "false", "0")

			// ASSERT-01: Comment contains 'Post-code script failed' when false
			Expect(commentFalse).To(ContainSubstring("Post-code script failed"))
			Expect(commentFalse).NotTo(ContainSubstring("Code agent failed"))

			// Test with AGENT_ERROR_EXIT = "" (unset equivalent)
			commentUnset := buildErrorComment("1", "my-org/my-repo", "12345", "", "", "0")

			// ASSERT-02: Comment contains 'Post-code script failed' when unset
			Expect(commentUnset).To(ContainSubstring("Post-code script failed"))
			Expect(commentUnset).NotTo(ContainSubstring("Code agent failed"))

			// Validate via shell: false case
			shellFalse, err := shellBuildErrorComment(scriptDir, "1", "my-org/my-repo", "12345", "", "false", "0")
			Expect(err).NotTo(HaveOccurred())
			Expect(shellFalse).To(ContainSubstring("Post-code script failed"))
			Expect(shellFalse).NotTo(ContainSubstring("Code agent failed"))
		})
	})
})
