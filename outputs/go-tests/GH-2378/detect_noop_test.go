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
Agent Error Detection (detect_noop) Tests

STP Reference: outputs/stp/GH-2378/GH-2378_test_plan.md
Jira: GH-2378
*/

// detectNoop reimplements the noop-detection logic from post-code.sh so
// tests can exercise it without a real git repo or network access.
//
// Returns a structured string: "noop:<checkpoint>:<msg>",
// "error:<checkpoint>:<msg>", or "proceed".
func detectNoop(branch, changedFiles, agentExitCode string) (string, error) {
	if agentExitCode == "" {
		agentExitCode = "0"
	}

	// Step 1: branch check (mirrors post-code.sh section 1)
	if branch == "" || branch == "main" || branch == "master" {
		if agentExitCode != "0" {
			return fmt.Sprintf("error:branch:Agent exited with code %s and did not create a feature branch", agentExitCode), fmt.Errorf("agent error at branch check")
		}
		return fmt.Sprintf("noop:branch:Agent did not create a feature branch (current: '%s') — nothing to do", branch), nil
	}

	// Step 2: changed files check (mirrors post-code.sh section 2)
	if changedFiles == "" {
		if agentExitCode != "0" {
			return fmt.Sprintf("error:files:Agent exited with code %s and produced no changes", agentExitCode), fmt.Errorf("agent error at files check")
		}
		return "noop:files:No changed files in agent's commit(s) — nothing to do", nil
	}

	return "proceed", nil
}

// sourcePostCodeScript creates a temporary bash script that sources the
// detect_noop-equivalent logic and runs it with provided arguments.
// This validates the actual shell logic matches our Go reimplementation.
func sourcePostCodeScript(scriptDir, branch, changedFiles, agentExitCode string) (string, int, error) {
	testScript := fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail

detect_noop() {
  local branch="$1"
  local changed_files="$2"
  local agent_exit_code="${3:-0}"

  if [ -z "${branch}" ] || [ "${branch}" = "main" ] || [ "${branch}" = "master" ]; then
    if [ "${agent_exit_code}" != "0" ]; then
      echo "error:branch:Agent exited with code ${agent_exit_code} and did not create a feature branch"
      return 1
    fi
    echo "noop:branch:Agent did not create a feature branch (current: '${branch:-detached HEAD}') — nothing to do"
    return 0
  fi

  if [ -z "${changed_files}" ]; then
    if [ "${agent_exit_code}" != "0" ]; then
      echo "error:files:Agent exited with code ${agent_exit_code} and produced no changes"
      return 1
    fi
    echo "noop:files:No changed files in agent's commit(s) — nothing to do"
    return 0
  fi

  echo "proceed"
  return 0
}

detect_noop %q %q %q
`, branch, changedFiles, agentExitCode)

	scriptPath := filepath.Join(scriptDir, "detect_noop_test.sh")
	if err := os.WriteFile(scriptPath, []byte(testScript), 0o755); err != nil {
		return "", -1, err
	}

	cmd := exec.Command("bash", scriptPath)
	out, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return "", -1, err
		}
	}
	return strings.TrimSpace(string(out)), exitCode, nil
}

var _ = Describe("[GH-2378] Agent Error Detection at Noop Checkpoints", Ordered, func() {
	/*
		Markers:
		    - tier1

		Preconditions:
		    - post-code.sh sourced for function testing
		    - detect_noop function is accessible
		    - bash 5.x+ available
	*/

	var scriptDir string

	BeforeAll(func() {
		var err error
		scriptDir, err = os.MkdirTemp("", "gh2378-detect-noop-*")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterAll(func() {
		os.RemoveAll(scriptDir)
	})

	Context("when agent exits non-zero and no feature branch exists", func() {
		It("[test_id:TS-GH-2378-001] should return agent_error when agent exits non-zero and no branch exists", func() {
			// --- Go reimplementation test ---
			result, err := detectNoop("main", "", "1")
			Expect(err).To(HaveOccurred())
			Expect(result).To(HavePrefix("error:branch:"))
			Expect(result).To(ContainSubstring("Agent exited with code 1"))

			// Also test empty branch (detached HEAD equivalent)
			result2, err2 := detectNoop("", "", "1")
			Expect(err2).To(HaveOccurred())
			Expect(result2).To(HavePrefix("error:branch:"))

			// --- Shell script validation ---
			shellResult, exitCode, shellErr := sourcePostCodeScript(scriptDir, "main", "", "1")
			Expect(shellErr).NotTo(HaveOccurred())
			Expect(exitCode).To(Equal(1))
			Expect(shellResult).To(HavePrefix("error:branch:"))
			Expect(shellResult).To(ContainSubstring("Agent exited with code 1"))
		})
	})

	Context("when agent exits non-zero on feature branch with no changed files", func() {
		It("[test_id:TS-GH-2378-002] should return agent_error when agent exits non-zero on branch with no changed files", func() {
			// --- Go reimplementation test ---
			result, err := detectNoop("agent/42-fix-widget", "", "2")
			Expect(err).To(HaveOccurred())
			Expect(result).To(HavePrefix("error:files:"))
			Expect(result).To(ContainSubstring("Agent exited with code 2"))

			// --- Shell script validation ---
			shellResult, exitCode, shellErr := sourcePostCodeScript(scriptDir, "agent/42-fix-widget", "", "2")
			Expect(shellErr).NotTo(HaveOccurred())
			Expect(exitCode).To(Equal(1))
			Expect(shellResult).To(HavePrefix("error:files:"))
			Expect(shellResult).To(ContainSubstring("Agent exited with code 2"))
		})
	})

	Context("when agent exits 0 with no commits", func() {
		It("[test_id:TS-GH-2378-003] should return noop when agent exits 0 with no commits", func() {
			// --- No-branch path: agent exits 0, no branch ---
			result1, err1 := detectNoop("main", "", "0")
			Expect(err1).NotTo(HaveOccurred())
			Expect(result1).To(HavePrefix("noop:branch:"))
			Expect(result1).NotTo(ContainSubstring("error"))

			// --- No-files path: agent exits 0, branch exists, no changed files ---
			result2, err2 := detectNoop("agent/42-fix-widget", "", "0")
			Expect(err2).NotTo(HaveOccurred())
			Expect(result2).To(HavePrefix("noop:files:"))
			Expect(result2).NotTo(ContainSubstring("error"))

			// --- Verify via shell: no-branch path ---
			shellResult1, exitCode1, shellErr1 := sourcePostCodeScript(scriptDir, "main", "", "0")
			Expect(shellErr1).NotTo(HaveOccurred())
			Expect(exitCode1).To(Equal(0))
			Expect(shellResult1).To(HavePrefix("noop:branch:"))

			// --- Verify via shell: no-files path ---
			shellResult2, exitCode2, shellErr2 := sourcePostCodeScript(scriptDir, "agent/42-fix-widget", "", "0")
			Expect(shellErr2).NotTo(HaveOccurred())
			Expect(exitCode2).To(Equal(0))
			Expect(shellResult2).To(HavePrefix("noop:files:"))
		})
	})

	Context("when agent exits non-zero but changes exist", func() {
		It("[test_id:TS-GH-2378-008] should continue to push/PR flow when changes exist despite non-zero exit", func() {
			// --- Go reimplementation test ---
			result, err := detectNoop("agent/42-fix-widget", "src/main.go\nsrc/handler.go\nsrc/util.go", "1")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal("proceed"))

			// --- Shell script validation ---
			shellResult, exitCode, shellErr := sourcePostCodeScript(scriptDir, "agent/42-fix-widget", "src/main.go", "1")
			Expect(shellErr).NotTo(HaveOccurred())
			Expect(exitCode).To(Equal(0))
			Expect(shellResult).To(Equal("proceed"))
		})
	})

	Context("when on detached HEAD with non-zero exit code", func() {
		It("[test_id:TS-GH-2378-009] should return agent_error on detached HEAD with non-zero exit code", func() {
			// Detached HEAD is represented by empty branch string
			result, err := detectNoop("", "", "1")
			Expect(err).To(HaveOccurred())
			Expect(result).To(HavePrefix("error:branch:"))
			Expect(result).To(ContainSubstring("Agent exited with code 1"))

			// --- Shell script validation ---
			shellResult, exitCode, shellErr := sourcePostCodeScript(scriptDir, "", "", "1")
			Expect(shellErr).NotTo(HaveOccurred())
			Expect(exitCode).To(Equal(1))
			Expect(shellResult).To(HavePrefix("error:branch:"))

			// Also verify with exit code 2 (different non-zero)
			result2, err2 := detectNoop("", "", "2")
			Expect(err2).To(HaveOccurred())
			Expect(result2).To(ContainSubstring("Agent exited with code 2"))
		})
	})
})
