package tests

import (
	"os"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

/*
CSMA Regression Tests

STP Reference: outputs/stp/GH-44/GH-44_test_plan.md
STD Reference: outputs/std/GH-44/GH-44_test_description.yaml
Jira: GH-44

Tests verify that the post-reset spread changes do not break existing
CSMA retry behavior: happy path, secondary rate-limit recovery,
exhausted retries error surfacing, and exit-0 rate-limit detection.
*/

// runTestScenario executes a named scenario from post-prioritize-test.sh.
// Returns stdout, stderr, and the exit code.
func runTestScenario(scenario string, extraEnv ...string) (stdout, stderr string, exitCode int) {
	testHarness := os.Getenv("CSMA_TEST_HARNESS")
	if testHarness == "" {
		testHarness = "post-prioritize-test.sh"
	}

	cmd := exec.Command("bash", testHarness, scenario)
	cmd.Env = append(os.Environ(), extraEnv...)

	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	exitCode = 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	return outBuf.String(), errBuf.String(), exitCode
}

var _ = Describe("[GH-44] CSMA regression tests", func() {
	var (
		tmpDir       string
		originalPath string
	)

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "csma-regression-*")
		Expect(err).NotTo(HaveOccurred())
		originalPath = os.Getenv("PATH")
		// Ensure spread override is not set for regression tests
		os.Unsetenv("GITHUB_CSMA_SPREAD_MAX_SEC")
	})

	AfterEach(func() {
		os.Setenv("PATH", originalPath)
		os.RemoveAll(tmpDir)
		os.Unsetenv("GITHUB_CSMA_SPREAD_MAX_SEC")
	})

	Context("happy-path test", Ordered, func() {
		/*
		Scenario: TS-GH-44-008
		Priority: P0 (MVP)
		Runs the existing happy-path test scenario to confirm the post-reset
		spread changes do not break the normal (non-rate-limited) CSMA workflow.
		*/
		It("[test_id:TS-GH-44-008] should pass existing post-prioritize happy-path test", func() {
			stdout, stderr, exitCode := runTestScenario("happy-path")

			Expect(exitCode).To(Equal(0),
				"Happy-path test should exit with code 0. stdout=%s stderr=%s", stdout, stderr)

			// Verify no unexpected error messages in stderr
			Expect(stderr).NotTo(ContainSubstring("FATAL"),
				"No FATAL errors should appear in happy-path test")
			Expect(stderr).NotTo(ContainSubstring("panic"),
				"No panics should appear in happy-path test")
		})
	})

	Context("rate-limit retry with secondary limit", Ordered, func() {
		/*
		Scenario: TS-GH-44-009
		Priority: P0 (MVP)
		Simulates a secondary rate limit (HTTP 403 with "secondary rate limit"
		message). Validates that the CSMA backoff with spread still recovers.
		*/
		It("[test_id:TS-GH-44-009] should recover from secondary rate limit", func() {
			stdout, stderr, exitCode := runTestScenario("rate-limit-retry")

			Expect(exitCode).To(Equal(0),
				"Rate-limit retry test should recover and exit with code 0. stdout=%s stderr=%s",
				stdout, stderr)
		})
	})

	Context("exhausted retries", Ordered, func() {
		/*
		Scenario: TS-GH-44-010
		Priority: P1
		Validates that when all retry attempts are exhausted, the CSMA library
		surfaces an appropriate error with a non-zero exit code. Spread delay
		must not mask or alter error reporting.
		*/
		It("[test_id:TS-GH-44-010] should surface error correctly when retries exhausted", func() {
			stdout, stderr, exitCode := runTestScenario("exhausted-retries")

			Expect(exitCode).NotTo(Equal(0),
				"Exhausted retries should result in non-zero exit code. stdout=%s", stdout)

			// Error message should be present in stderr for CI visibility
			Expect(stderr).To(SatisfyAny(
				ContainSubstring("retries exhausted"),
				ContainSubstring("rate limit"),
				ContainSubstring("exceeded"),
				ContainSubstring("failed"),
			), "Error message should be written to stderr when retries are exhausted")
		})
	})

	Context("exit-0 rate-limit detection", Ordered, func() {
		/*
		Scenario: TS-GH-44-011
		Priority: P1
		Validates that the CSMA library correctly detects rate-limit responses
		that return exit code 0 (gh CLI returns 0 but includes rate-limit headers)
		and retries with spread.
		*/
		It("[test_id:TS-GH-44-011] should detect and retry on exit-0 rate-limit", func() {
			stdout, stderr, exitCode := runTestScenario("exit0-rate-limit")

			Expect(exitCode).To(Equal(0),
				"Exit-0 rate-limit should be detected, retried, and eventually succeed. stdout=%s stderr=%s",
				stdout, stderr)
		})
	})
})
