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
EnsureProvider Error Handling Tests

STP Reference: outputs/stp/GH-28/GH-28_test_plan.md
STD Reference: outputs/std/GH-28/GH-28_test_description.yaml
Jira: GH-28
*/

var _ = Describe("[GH-28] EnsureProvider error handling", func() {
	/*
		Markers:
			- tier1

		Preconditions:
			- Go toolchain 1.21+ available
			- Mock openshell binaries configured via GinkgoT().TempDir() and PATH override
	*/

	Context("when create fails with non-AlreadyExists error", Ordered, func() {
		var tmpDir string

		BeforeAll(func() {
			tmpDir = GinkgoT().TempDir()

			// Mock openshell: create fails with a generic error (not AlreadyExists).
			fakeOpenshell := filepath.Join(tmpDir, "openshell")
			script := "#!/bin/sh\n" +
				"echo \"Error: connection refused\" >&2\n" +
				"exit 1\n"
			Expect(os.WriteFile(fakeOpenshell, []byte(script), 0o755)).To(Succeed())
			GinkgoT().Setenv("PATH", tmpDir)
		})

		It("[test_id:TS-GH-28-007] should return error immediately without attempting delete", func() {
			err := sandbox.EnsureProvider("test-provider", "github", nil, nil)
			Expect(err).To(HaveOccurred(), "EnsureProvider should return error on non-AlreadyExists failure")
			Expect(err.Error()).To(ContainSubstring("provider create"),
				"error should indicate create failure")
			Expect(err.Error()).To(ContainSubstring("connection refused"),
				"error should contain the original error context")
		})
	})

	Context("when create fails with permission denied error", Ordered, func() {
		var (
			tmpDir  string
			logFile string
		)

		BeforeAll(func() {
			tmpDir = GinkgoT().TempDir()
			logFile = filepath.Join(tmpDir, "call_log")

			// Mock openshell with call logging: returns permission denied on create.
			fakeOpenshell := filepath.Join(tmpDir, "openshell")
			script := "#!/bin/sh\n" +
				"echo \"$1 $2\" >> \"" + logFile + "\"\n" +
				"if [ \"$1\" = \"provider\" ] && [ \"$2\" = \"create\" ]; then\n" +
				"  echo \"Error: permission denied\" >&2\n" +
				"  exit 1\n" +
				"fi\n" +
				"if [ \"$1\" = \"provider\" ] && [ \"$2\" = \"delete\" ]; then\n" +
				"  exit 0\n" +
				"fi\n" +
				"exit 1\n"
			Expect(os.WriteFile(fakeOpenshell, []byte(script), 0o755)).To(Succeed())
			GinkgoT().Setenv("PATH", tmpDir)
		})

		It("[test_id:TS-GH-28-008] should not trigger delete", func() {
			err := sandbox.EnsureProvider("test-provider", "github", nil, nil)
			Expect(err).To(HaveOccurred(), "EnsureProvider should return error")

			// Verify delete was never called by checking the call log.
			data, readErr := os.ReadFile(logFile)
			Expect(readErr).NotTo(HaveOccurred(), "call log should exist")
			lines := strings.Split(strings.TrimSpace(string(data)), "\n")
			for _, line := range lines {
				Expect(line).NotTo(Equal("provider delete"),
					"delete should not be called for non-AlreadyExists error")
			}
		})
	})

	Context("when delete fails during reconciliation", Ordered, func() {
		var tmpDir string

		BeforeAll(func() {
			tmpDir = GinkgoT().TempDir()

			// Mock openshell: AlreadyExists on create, failure on delete.
			fakeOpenshell := filepath.Join(tmpDir, "openshell")
			script := "#!/bin/sh\n" +
				"if [ \"$1\" = \"provider\" ] && [ \"$2\" = \"create\" ]; then\n" +
				"  echo 'Error: × status: AlreadyExists, message: \"provider already exists\"' >&2\n" +
				"  exit 1\n" +
				"fi\n" +
				"if [ \"$1\" = \"provider\" ] && [ \"$2\" = \"delete\" ]; then\n" +
				"  echo \"Error: failed to delete provider\" >&2\n" +
				"  exit 1\n" +
				"fi\n" +
				"exit 1\n"
			Expect(os.WriteFile(fakeOpenshell, []byte(script), 0o755)).To(Succeed())
			GinkgoT().Setenv("PATH", tmpDir)
		})

		It("[test_id:TS-GH-28-009] should return descriptive error", func() {
			err := sandbox.EnsureProvider("test-provider", "github", nil, nil)
			Expect(err).To(HaveOccurred(), "EnsureProvider should return error when delete fails")
			Expect(err.Error()).To(ContainSubstring("delete"),
				"error should indicate delete failure")
		})
	})

	Context("when delete fails", Ordered, func() {
		var (
			tmpDir       string
			providerName string
		)

		BeforeAll(func() {
			tmpDir = GinkgoT().TempDir()
			providerName = "my-test-provider"

			// Mock openshell: AlreadyExists on create, failure on delete.
			fakeOpenshell := filepath.Join(tmpDir, "openshell")
			script := "#!/bin/sh\n" +
				"if [ \"$1\" = \"provider\" ] && [ \"$2\" = \"create\" ]; then\n" +
				"  echo 'Error: × status: AlreadyExists, message: \"provider already exists\"' >&2\n" +
				"  exit 1\n" +
				"fi\n" +
				"if [ \"$1\" = \"provider\" ] && [ \"$2\" = \"delete\" ]; then\n" +
				"  echo \"Error: failed to delete provider $3\" >&2\n" +
				"  exit 1\n" +
				"fi\n" +
				"exit 1\n"
			Expect(os.WriteFile(fakeOpenshell, []byte(script), 0o755)).To(Succeed())
			GinkgoT().Setenv("PATH", tmpDir)
		})

		It("[test_id:TS-GH-28-010] should include provider name in error message", func() {
			err := sandbox.EnsureProvider(providerName, "github", nil, nil)
			Expect(err).To(HaveOccurred(), "EnsureProvider should return error when delete fails")
			Expect(err.Error()).To(ContainSubstring(providerName),
				"error should include provider name for debugging context")
		})
	})
})
