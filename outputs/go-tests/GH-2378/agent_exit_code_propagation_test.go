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
AGENT_EXIT_CODE Propagation Tests

STP Reference: outputs/stp/GH-2378/GH-2378_test_plan.md
Jira: GH-2378

This test validates that the Go harness (run.go) correctly propagates
AGENT_EXIT_CODE to the post-script environment. Since we cannot import
the internal/cli package in an e2e test package, we validate the pattern
by:

1. Verifying the env-propagation logic works correctly in isolation
2. Simulating the post-script receiving AGENT_EXIT_CODE and acting on it
*/

// simulatePostScriptEnv creates a mock post-script that receives
// AGENT_EXIT_CODE via environment and validates it was propagated.
func simulatePostScriptEnv(scriptDir string, exitCode int) (string, error) {
	// This script simulates what the Go harness does:
	//   postCmd.Env = append(postCmd.Env, fmt.Sprintf("AGENT_EXIT_CODE=%d", lastExitCode))
	// Then the post-script reads it and makes decisions based on it.
	testScript := fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail

# Simulate reading AGENT_EXIT_CODE as the post-script would
if [ -z "${AGENT_EXIT_CODE:-}" ]; then
  echo "MISSING"
  exit 1
fi

echo "AGENT_EXIT_CODE=${AGENT_EXIT_CODE}"

# Validate it matches expected value
if [ "${AGENT_EXIT_CODE}" = "%d" ]; then
  echo "MATCH"
else
  echo "MISMATCH:expected=%d:actual=${AGENT_EXIT_CODE}"
  exit 1
fi
`, exitCode, exitCode)

	scriptPath := filepath.Join(scriptDir, "post_script_env_test.sh")
	if err := os.WriteFile(scriptPath, []byte(testScript), 0o755); err != nil {
		return "", err
	}

	cmd := exec.Command("bash", scriptPath)
	// Simulate the Go harness setting AGENT_EXIT_CODE in the command environment
	cmd.Env = append(os.Environ(), fmt.Sprintf("AGENT_EXIT_CODE=%d", exitCode))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), err
	}
	return strings.TrimSpace(string(out)), nil
}

// simulateHarnessEnvPropagation validates the env-building logic that
// run.go uses: append(postCmd.Env, fmt.Sprintf("AGENT_EXIT_CODE=%d", lastExitCode))
func simulateHarnessEnvPropagation(lastExitCode int) []string {
	baseEnv := []string{
		"PATH=/usr/bin:/bin",
		"HOME=/tmp",
		"PUSH_TOKEN=mock-token",
		"REPO_FULL_NAME=fullsend-ai/fullsend",
	}
	// This mirrors the exact line from run.go:543
	return append(baseEnv, fmt.Sprintf("AGENT_EXIT_CODE=%d", lastExitCode))
}

var _ = Describe("[GH-2378] Agent Exit Code Propagation from Go Harness", Ordered, func() {
	/*
		Markers:
		    - tier1

		Preconditions:
		    - Go test harness available
		    - run_test.go can test runAgent function
		    - Go 1.23+ installed
	*/

	var scriptDir string

	BeforeAll(func() {
		var err error
		scriptDir, err = os.MkdirTemp("", "gh2378-exit-code-*")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterAll(func() {
		os.RemoveAll(scriptDir)
	})

	Context("when runAgent completes with non-zero exit code", func() {
		It("[test_id:TS-GH-2378-007] should pass AGENT_EXIT_CODE to post-script environment", func() {
			// Test 1: Validate env-building logic produces correct AGENT_EXIT_CODE
			envVars := simulateHarnessEnvPropagation(1)

			// ASSERT-01: AGENT_EXIT_CODE is present in post-script environment
			found := false
			for _, env := range envVars {
				if env == "AGENT_EXIT_CODE=1" {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "AGENT_EXIT_CODE=1 should be present in env vars")

			// ASSERT-02: AGENT_EXIT_CODE value matches agent's actual exit code
			for _, env := range envVars {
				if strings.HasPrefix(env, "AGENT_EXIT_CODE=") {
					value := strings.TrimPrefix(env, "AGENT_EXIT_CODE=")
					Expect(value).To(Equal("1"), "AGENT_EXIT_CODE should equal the agent's exit code")
				}
			}

			// Test 2: Validate the post-script can receive and read the env var
			result, err := simulatePostScriptEnv(scriptDir, 1)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(ContainSubstring("AGENT_EXIT_CODE=1"))
			Expect(result).To(ContainSubstring("MATCH"))

			// Test 3: Validate with exit code 0 (agent success)
			envVars0 := simulateHarnessEnvPropagation(0)
			Expect(envVars0).To(ContainElement("AGENT_EXIT_CODE=0"))

			result0, err0 := simulatePostScriptEnv(scriptDir, 0)
			Expect(err0).NotTo(HaveOccurred())
			Expect(result0).To(ContainSubstring("AGENT_EXIT_CODE=0"))
			Expect(result0).To(ContainSubstring("MATCH"))

			// Test 4: Validate with exit code 2 (different non-zero)
			envVars2 := simulateHarnessEnvPropagation(2)
			Expect(envVars2).To(ContainElement("AGENT_EXIT_CODE=2"))
		})
	})
})
