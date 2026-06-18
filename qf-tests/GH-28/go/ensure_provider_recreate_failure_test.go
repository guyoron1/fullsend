package tests

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/fullsend-ai/fullsend/internal/sandbox"
)

/*
EnsureProvider Recreate Failure Tests

STP Reference: outputs/stp/GH-28/GH-28_test_plan.md
STD Reference: outputs/std/GH-28/GH-28_test_description.yaml
Jira: GH-28
*/

var _ = Describe("[GH-28] EnsureProvider recreate failure", func() {
	/*
		Markers:
			- tier1

		Preconditions:
			- Go toolchain 1.21+ available
			- Mock openshell binaries configured via GinkgoT().TempDir() and PATH override
	*/

	Context("when recreate fails after successful delete", Ordered, func() {
		var tmpDir string

		BeforeAll(func() {
			tmpDir = GinkgoT().TempDir()

			// Mock openshell: AlreadyExists on first create, delete succeeds,
			// second create fails with a generic error.
			fakeOpenshell := filepath.Join(tmpDir, "openshell")
			markerFile := filepath.Join(tmpDir, "state")
			script := "#!/bin/sh\n" +
				"if [ \"$1\" = \"provider\" ] && [ \"$2\" = \"create\" ]; then\n" +
				"  if [ ! -f \"" + markerFile + "\" ]; then\n" +
				"    touch \"" + markerFile + "\"\n" +
				"    echo 'Error: AlreadyExists: provider already exists' >&2\n" +
				"    exit 1\n" +
				"  else\n" +
				"    echo 'Error: failed to recreate provider' >&2\n" +
				"    exit 1\n" +
				"  fi\n" +
				"fi\n" +
				"if [ \"$1\" = \"provider\" ] && [ \"$2\" = \"delete\" ]; then\n" +
				"  exit 0\n" +
				"fi\n" +
				"exit 1\n"
			Expect(os.WriteFile(fakeOpenshell, []byte(script), 0o755)).To(Succeed())
			GinkgoT().Setenv("PATH", tmpDir)
		})

		It("[test_id:TS-GH-28-011] should return error indicating recreate failure", func() {
			err := sandbox.EnsureProvider("test-provider", "github", nil, nil)
			Expect(err).To(HaveOccurred(), "EnsureProvider should return error when recreate fails")
			Expect(err.Error()).To(ContainSubstring("recreate"),
				"error should indicate recreate (second create) failure")
		})
	})

	Context("when recreate fails", Ordered, func() {
		var (
			tmpDir      string
			secretValue string
		)

		BeforeAll(func() {
			tmpDir = GinkgoT().TempDir()
			secretValue = "recreate-redact-secret-zyx987"
			GinkgoT().Setenv("MY_SECRET", secretValue)

			// Mock openshell: AlreadyExists on first create, delete succeeds,
			// second create fails with secret in error output.
			fakeOpenshell := filepath.Join(tmpDir, "openshell")
			markerFile := filepath.Join(tmpDir, "state")
			script := "#!/bin/sh\n" +
				"if [ \"$1\" = \"provider\" ] && [ \"$2\" = \"create\" ]; then\n" +
				"  if [ ! -f \"" + markerFile + "\" ]; then\n" +
				"    touch \"" + markerFile + "\"\n" +
				"    echo 'Error: AlreadyExists' >&2\n" +
				"    exit 1\n" +
				"  else\n" +
				"    echo \"Error: recreate failed secret=" + secretValue + " token=" + secretValue + "\" >&2\n" +
				"    exit 1\n" +
				"  fi\n" +
				"fi\n" +
				"if [ \"$1\" = \"provider\" ] && [ \"$2\" = \"delete\" ]; then\n" +
				"  exit 0\n" +
				"fi\n" +
				"exit 1\n"
			Expect(os.WriteFile(fakeOpenshell, []byte(script), 0o755)).To(Succeed())
			GinkgoT().Setenv("PATH", tmpDir)
		})

		It("[test_id:TS-GH-28-012] should not include raw secret values in error", func() {
			err := sandbox.EnsureProvider("test-provider", "github",
				map[string]string{"MY_SECRET": "${MY_SECRET}"}, nil)
			Expect(err).To(HaveOccurred(), "EnsureProvider should return error on recreate failure")
			Expect(err.Error()).NotTo(ContainSubstring(secretValue),
				"secret value must not appear in recreate error message")
			Expect(err.Error()).To(ContainSubstring("***"),
				"error should contain redacted placeholder")
		})
	})
})
