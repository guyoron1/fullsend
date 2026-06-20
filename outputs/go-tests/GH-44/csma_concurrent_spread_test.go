package tests

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

/*
CSMA Concurrent Spread Tests

STP Reference: outputs/stp/GH-44/GH-44_test_plan.md
STD Reference: outputs/std/GH-44/GH-44_test_description.yaml
Jira: GH-44

Tests verify that concurrent runners (simulated via parallel subshells)
wake at staggered times when spread is enabled, and simultaneously when
spread is disabled (SPREAD_MAX_SEC=0).
*/

const numRunners = 5

// runParallelSpread launches numRunners parallel subshells that each call
// _github_csma_sleep_after_rate_limit and records their wake timestamp.
// Returns a slice of wake timestamps (unix seconds).
func runParallelSpread(mockDir, csmaLib string, spreadMaxSec int, resetTimestamp int64) []int64 {
	var (
		mu        sync.Mutex
		wg        sync.WaitGroup
		wakeTimes []int64
	)

	wg.Add(numRunners)
	for i := 0; i < numRunners; i++ {
		go func(idx int) {
			defer wg.Done()
			defer GinkgoRecover()

			script := fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail
export PATH="%s:$PATH"
export GITHUB_CSMA_SPREAD_MAX_SEC=%d
source "%s"
_github_csma_sleep_after_rate_limit 2>/dev/null || true
date +%%s
`, mockDir, spreadMaxSec, csmaLib)

			cmd := exec.Command("bash", "-c", script)
			cmd.Env = append(os.Environ(),
				fmt.Sprintf("PATH=%s:%s", mockDir, os.Getenv("PATH")),
				fmt.Sprintf("GITHUB_CSMA_SPREAD_MAX_SEC=%d", spreadMaxSec),
			)

			var outBuf strings.Builder
			cmd.Stdout = &outBuf
			_ = cmd.Run()

			tsStr := strings.TrimSpace(outBuf.String())
			// Take the last line if multiple lines of output
			lines := strings.Split(tsStr, "\n")
			lastLine := strings.TrimSpace(lines[len(lines)-1])

			ts, err := strconv.ParseInt(lastLine, 10, 64)
			if err == nil {
				mu.Lock()
				wakeTimes = append(wakeTimes, ts)
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()
	return wakeTimes
}

// createConcurrentMockGh creates a mock gh binary that returns 429 with a
// specified reset timestamp. Designed for concurrent use.
func createConcurrentMockGh(tmpDir string, resetTimestamp int64) string {
	mockPath := fmt.Sprintf("%s/gh", tmpDir)
	mockScript := fmt.Sprintf(`#!/usr/bin/env bash
if [[ "$1" == "api" ]]; then
  echo "x-ratelimit-remaining: 0" >&2
  echo "x-ratelimit-reset: %d" >&2
  echo '{"message":"API rate limit exceeded"}'
  exit 4
fi
exit 1
`, resetTimestamp)

	err := os.WriteFile(mockPath, []byte(mockScript), 0755)
	Expect(err).NotTo(HaveOccurred())
	return tmpDir
}

var _ = Describe("[GH-44] CSMA concurrent spread", func() {
	var (
		tmpDir       string
		originalPath string
		csmaLib      string
	)

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "csma-concurrent-*")
		Expect(err).NotTo(HaveOccurred())
		originalPath = os.Getenv("PATH")
		csmaLib = os.Getenv("CSMA_LIB_PATH")
		if csmaLib == "" {
			csmaLib = "github-api-csma.sh"
		}
	})

	AfterEach(func() {
		os.Setenv("PATH", originalPath)
		os.RemoveAll(tmpDir)
		os.Unsetenv("GITHUB_CSMA_SPREAD_MAX_SEC")
	})

	Context("parallel subshells with spread enabled", Ordered, func() {
		/*
		Scenario: TS-GH-44-012
		Priority: P1
		Spawns multiple parallel subshells that each call
		_github_csma_sleep_after_rate_limit with the same reset timestamp.
		Verifies that spread causes them to wake at different times.
		This directly validates the thundering-herd fix.
		*/
		It("[test_id:TS-GH-44-012] should wake at different times with spread enabled", func() {
			resetTime := time.Now().Add(1 * time.Second).Unix()
			mockDir := createConcurrentMockGh(tmpDir, resetTime)

			wakeTimes := runParallelSpread(mockDir, csmaLib, 10, resetTime)

			Expect(len(wakeTimes)).To(BeNumerically(">=", 2),
				"At least 2 subshells should have reported wake times")

			// Check that not all timestamps are identical (spread should cause variance)
			uniqueTimes := make(map[int64]bool)
			for _, t := range wakeTimes {
				uniqueTimes[t] = true
			}

			Expect(len(uniqueTimes)).To(BeNumerically(">=", 2),
				"At least 2 of %d subshells should wake at different times (spread should stagger wake times). Got times: %v",
				len(wakeTimes), wakeTimes)
		})
	})

	Context("parallel subshells with spread disabled", Ordered, func() {
		/*
		Scenario: TS-GH-44-013
		Priority: P2
		Control test for TS-GH-44-012. With SPREAD_MAX_SEC=0, all runners
		should wake at approximately the same time. This confirms that
		spread is the cause of staggered wake times in TS-GH-44-012.
		*/
		It("[test_id:TS-GH-44-013] should wake simultaneously when SPREAD_MAX_SEC=0", func() {
			resetTime := time.Now().Add(1 * time.Second).Unix()
			mockDir := createConcurrentMockGh(tmpDir, resetTime)

			wakeTimes := runParallelSpread(mockDir, csmaLib, 0, resetTime)

			Expect(len(wakeTimes)).To(BeNumerically(">=", 2),
				"At least 2 subshells should have reported wake times")

			// All timestamps should be within 1 second of each other
			var minTime, maxTime int64
			minTime = wakeTimes[0]
			maxTime = wakeTimes[0]
			for _, t := range wakeTimes[1:] {
				if t < minTime {
					minTime = t
				}
				if t > maxTime {
					maxTime = t
				}
			}

			Expect(maxTime - minTime).To(BeNumerically("<=", 1),
				"With SPREAD_MAX_SEC=0, all subshells should wake within 1 second of each other. Got range: %d-%d",
				minTime, maxTime)
		})
	})
})
