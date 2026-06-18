package tests

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/fullsend-ai/fullsend/internal/sandbox"
)

/*
EnsureProvider Pipeline Integration Tests

STP Reference: outputs/stp/GH-28/GH-28_test_plan.md
STD Reference: outputs/std/GH-28/GH-28_test_description.yaml
Jira: GH-28
*/

var _ = Describe("[GH-28] EnsureProvider pipeline integration", func() {
	/*
		Markers:
			- tier1

		Preconditions:
			- Go toolchain 1.21+ available
			- Full mock environment for runAgent (mock openshell, mock gateway, environment variables)
	*/

	Context("when full run pipeline has pre-existing providers", Ordered, func() {
		var tmpDir string

		BeforeAll(func() {
			tmpDir = GinkgoT().TempDir()

			// Mock openshell: first create returns AlreadyExists (pre-existing provider),
			// delete succeeds, second create succeeds — simulating a repeated fullsend run.
			fakeOpenshell := filepath.Join(tmpDir, "openshell")
			markerFile := filepath.Join(tmpDir, "state")
			script := "#!/bin/sh\n" +
				"if [ \"$1\" = \"provider\" ] && [ \"$2\" = \"create\" ]; then\n" +
				"  if [ ! -f \"" + markerFile + "\" ]; then\n" +
				"    touch \"" + markerFile + "\"\n" +
				"    echo 'Error: × status: AlreadyExists, message: \"provider already exists\"' >&2\n" +
				"    exit 1\n" +
				"  else\n" +
				"    exit 0\n" +
				"  fi\n" +
				"fi\n" +
				"if [ \"$1\" = \"provider\" ] && [ \"$2\" = \"delete\" ]; then\n" +
				"  exit 0\n" +
				"fi\n" +
				"exit 0\n"
			Expect(os.WriteFile(fakeOpenshell, []byte(script), 0o755)).To(Succeed())
			GinkgoT().Setenv("PATH", tmpDir)
		})

		It("[test_id:TS-GH-28-016] should complete run successfully", func() {
			// Simulate the pipeline flow: EnsureProvider is called during runAgent
			// with a pre-existing provider. The idempotency fix should handle the
			// AlreadyExists error transparently via delete+recreate.
			credValue := "pipeline-fresh-cred-2026"
			GinkgoT().Setenv("PIPELINE_CRED", credValue)

			err := sandbox.EnsureProvider("pipeline-provider", "github",
				map[string]string{"PIPELINE_CRED": "${PIPELINE_CRED}"}, nil)
			Expect(err).NotTo(HaveOccurred(),
				"EnsureProvider should complete successfully with pre-existing provider in pipeline context")
		})
	})
})
