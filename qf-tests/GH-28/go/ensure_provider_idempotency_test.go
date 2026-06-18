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
EnsureProvider Idempotency Tests

STP Reference: outputs/stp/GH-28/GH-28_test_plan.md
STD Reference: outputs/std/GH-28/GH-28_test_description.yaml
Jira: GH-28
*/

var _ = Describe("[GH-28] EnsureProvider idempotency", func() {
	/*
		Markers:
			- tier1

		Preconditions:
			- Go toolchain 1.21+ available
			- Ginkgo v2 and Gomega assertion library installed
			- Mock openshell binaries configured via GinkgoT().TempDir() and PATH override
	*/

	Context("when provider already exists", Ordered, func() {
		var (
			tmpDir     string
			logFile    string
			markerFile string
		)

		BeforeAll(func() {
			tmpDir = GinkgoT().TempDir()
			logFile = filepath.Join(tmpDir, "calls.log")
			markerFile = filepath.Join(tmpDir, "created")

			// Mock openshell: first create returns AlreadyExists,
			// delete succeeds, second create succeeds.
			fakeOpenshell := filepath.Join(tmpDir, "openshell")
			script := "#!/bin/sh\n" +
				`echo "$1 $2" >> "` + logFile + "\"\n" +
				"if [ \"$1\" = \"provider\" ] && [ \"$2\" = \"create\" ]; then\n" +
				"  if [ ! -f \"" + markerFile + "\" ]; then\n" +
				"    echo marker > \"" + markerFile + "\"\n" +
				"    echo 'Error: × status: AlreadyExists, message: \"provider already exists\"' >&2\n" +
				"    exit 1\n" +
				"  fi\n" +
				"  exit 0\n" +
				"fi\n" +
				"if [ \"$1\" = \"provider\" ] && [ \"$2\" = \"delete\" ]; then\n" +
				"  exit 0\n" +
				"fi\n" +
				"exit 1\n"
			Expect(os.WriteFile(fakeOpenshell, []byte(script), 0o755)).To(Succeed())
			GinkgoT().Setenv("PATH", tmpDir)
		})

		It("[test_id:TS-GH-28-001] should succeed without error", func() {
			err := sandbox.EnsureProvider("test-provider", "github", nil, nil)
			Expect(err).NotTo(HaveOccurred(), "EnsureProvider should succeed when provider already exists via delete+recreate")

			// Verify the call sequence: create, delete, create
			data, readErr := os.ReadFile(logFile)
			Expect(readErr).NotTo(HaveOccurred())
			lines := strings.Split(strings.TrimSpace(string(data)), "\n")
			Expect(lines).To(HaveLen(3), "expected 3 openshell calls: create, delete, create")
			Expect(lines[0]).To(Equal("provider create"))
			Expect(lines[1]).To(Equal("provider delete"))
			Expect(lines[2]).To(Equal("provider create"))
		})
	})

	Context("when provider already exists and is recreated", Ordered, func() {
		var (
			tmpDir  string
			argsLog string
		)

		BeforeAll(func() {
			tmpDir = GinkgoT().TempDir()
			argsLog = filepath.Join(tmpDir, "recreate_args")

			// Mock openshell: first create returns AlreadyExists,
			// delete succeeds, second create logs all args and succeeds.
			fakeOpenshell := filepath.Join(tmpDir, "openshell")
			markerFile := filepath.Join(tmpDir, "first_call")
			script := "#!/bin/sh\n" +
				"if [ \"$1\" = \"provider\" ] && [ \"$2\" = \"create\" ]; then\n" +
				"  if [ ! -f \"" + markerFile + "\" ]; then\n" +
				"    touch \"" + markerFile + "\"\n" +
				"    echo 'Error: AlreadyExists' >&2\n" +
				"    exit 1\n" +
				"  else\n" +
				"    echo \"$@\" > \"" + argsLog + "\"\n" +
				"    exit 0\n" +
				"  fi\n" +
				"fi\n" +
				"if [ \"$1\" = \"provider\" ] && [ \"$2\" = \"delete\" ]; then\n" +
				"  exit 0\n" +
				"fi\n" +
				"exit 1\n"
			Expect(os.WriteFile(fakeOpenshell, []byte(script), 0o755)).To(Succeed())
			GinkgoT().Setenv("PATH", tmpDir)
		})

		It("[test_id:TS-GH-28-002] should recreate with current credentials", func() {
			credValue := "fresh-credential-value-2026"
			GinkgoT().Setenv("MY_CRED", credValue)

			err := sandbox.EnsureProvider("test-provider", "github",
				map[string]string{"MY_CRED": "${MY_CRED}"}, nil)
			Expect(err).NotTo(HaveOccurred(), "EnsureProvider should succeed after delete+recreate")

			// Verify recreate was called with the credential key
			data, readErr := os.ReadFile(argsLog)
			Expect(readErr).NotTo(HaveOccurred(), "recreate args log should exist")
			argsStr := string(data)
			Expect(argsStr).To(ContainSubstring("--credential"), "recreate call should include credential flag")
			Expect(argsStr).To(ContainSubstring("MY_CRED"), "recreate call should include credential key name")
		})
	})

	Context("when called multiple consecutive times", Ordered, func() {
		var tmpDir string

		BeforeAll(func() {
			tmpDir = GinkgoT().TempDir()
		})

		It("[test_id:TS-GH-28-003] should succeed on every invocation", func() {
			// Each invocation gets a fresh mock that simulates
			// AlreadyExists on first create, then succeeds after delete+recreate.
			for i := 0; i < 3; i++ {
				iterDir := filepath.Join(tmpDir, "iter"+string(rune('0'+i)))
				Expect(os.MkdirAll(iterDir, 0o755)).To(Succeed())
				markerFile := filepath.Join(iterDir, "created")

				fakeOpenshell := filepath.Join(iterDir, "openshell")
				script := "#!/bin/sh\n" +
					"if [ \"$1\" = \"provider\" ] && [ \"$2\" = \"create\" ]; then\n" +
					"  if [ ! -f \"" + markerFile + "\" ]; then\n" +
					"    echo marker > \"" + markerFile + "\"\n" +
					"    echo 'Error: AlreadyExists' >&2\n" +
					"    exit 1\n" +
					"  fi\n" +
					"  exit 0\n" +
					"fi\n" +
					"if [ \"$1\" = \"provider\" ] && [ \"$2\" = \"delete\" ]; then\n" +
					"  exit 0\n" +
					"fi\n" +
					"exit 1\n"
				Expect(os.WriteFile(fakeOpenshell, []byte(script), 0o755)).To(Succeed())
				GinkgoT().Setenv("PATH", iterDir)

				err := sandbox.EnsureProvider("test-provider", "github", nil, nil)
				Expect(err).NotTo(HaveOccurred(), "EnsureProvider invocation %d should succeed", i+1)
			}
		})
	})
})
