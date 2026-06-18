package tests

import (
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/fullsend-ai/fullsend/internal/sandbox"
)

/*
EnsureProvider First-Time Creation Tests

STP Reference: outputs/stp/GH-28/GH-28_test_plan.md
STD Reference: outputs/std/GH-28/GH-28_test_description.yaml
Jira: GH-28
*/

var _ = Describe("[GH-28] EnsureProvider first-time creation", func() {
	/*
		Markers:
			- tier1

		Preconditions:
			- Go toolchain 1.21+ available
			- Mock openshell binaries configured via GinkgoT().TempDir() and PATH override
	*/

	Context("when provider does not exist", Ordered, func() {
		var (
			tmpDir  string
			logFile string
		)

		BeforeAll(func() {
			tmpDir = GinkgoT().TempDir()
			logFile = filepath.Join(tmpDir, "call_log")

			// Mock openshell: create succeeds immediately, logs all calls.
			fakeOpenshell := filepath.Join(tmpDir, "openshell")
			script := "#!/bin/sh\n" +
				"echo \"$1 $2\" >> \"" + logFile + "\"\n" +
				"exit 0\n"
			Expect(os.WriteFile(fakeOpenshell, []byte(script), 0o755)).To(Succeed())
			GinkgoT().Setenv("PATH", tmpDir)
		})

		It("[test_id:TS-GH-28-013] should create provider successfully on first attempt", func() {
			err := sandbox.EnsureProvider("test-provider", "github", nil, nil)
			Expect(err).NotTo(HaveOccurred(),
				"EnsureProvider should succeed on first-time creation")
		})

		It("[test_id:TS-GH-28-014] should not call delete", func() {
			// Read the call log to verify delete was never invoked.
			data, readErr := os.ReadFile(logFile)
			Expect(readErr).NotTo(HaveOccurred(), "call log should exist")
			lines := strings.Split(strings.TrimSpace(string(data)), "\n")
			for _, line := range lines {
				Expect(line).NotTo(Equal("provider delete"),
					"delete should not be called during successful first creation")
			}
		})
	})

	Context("when credentials are empty", Ordered, func() {
		var tmpDir string

		BeforeAll(func() {
			tmpDir = GinkgoT().TempDir()

			// Mock openshell: create succeeds.
			fakeOpenshell := filepath.Join(tmpDir, "openshell")
			script := "#!/bin/sh\n" +
				"exit 0\n"
			Expect(os.WriteFile(fakeOpenshell, []byte(script), 0o755)).To(Succeed())
			GinkgoT().Setenv("PATH", tmpDir)
		})

		It("[test_id:TS-GH-28-015] should handle empty credentials gracefully", func() {
			// Pass empty string credential values — function should not panic.
			GinkgoT().Setenv("EMPTY_CRED", "")
			err := sandbox.EnsureProvider("test-provider", "github",
				map[string]string{"EMPTY_CRED": "${EMPTY_CRED}"}, nil)
			// The function should complete without panicking. Whether it
			// succeeds or returns an error is implementation-dependent, but
			// it must not crash.
			_ = err // success or clean error is acceptable
		})
	})
})
