package tests

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("[GH-51] CLAUDE.md pointer injection", func() {
	Context("when Claude runtime with AGENTS.md and no CLAUDE.md", func() {
		var tmpDir string

		BeforeEach(func() {
			tmpDir = GinkgoT().TempDir()
			Expect(os.WriteFile(filepath.Join(tmpDir, "AGENTS.md"), []byte("# Agent rules"), 0644)).To(Succeed())
		})

		It("[test_id:TS-GH-51-001] should inject CLAUDE.md pointer file", func() {
			mockExec := func(cmd string) error { return nil }
			err := doInjectClaudeMDPointer(tmpDir, mockExec)
			Expect(err).NotTo(HaveOccurred())

			_, statErr := os.Stat(filepath.Join(tmpDir, "CLAUDE.md"))
			Expect(statErr).NotTo(HaveOccurred())
		})
	})
})
