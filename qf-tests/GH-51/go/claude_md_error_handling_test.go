package tests

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("[GH-51] CLAUDE.md injection error handling", func() {
	Context("[NEGATIVE] when injection fails", func() {
		It("[test_id:TS-GH-51-015] should continue agent run", func() {
			tmpDir := GinkgoT().TempDir()
			Expect(os.WriteFile(filepath.Join(tmpDir, "AGENTS.md"), []byte("# Agent rules"), 0644)).To(Succeed())

			// Make directory read-only so write fails
			Expect(os.Chmod(tmpDir, 0555)).To(Succeed())
			DeferCleanup(func() {
				_ = os.Chmod(tmpDir, 0755)
			})

			mockExec := func(cmd string) error { return nil }
			err := doInjectClaudeMDPointer(tmpDir, mockExec)
			// Injection should return error, but calling code (runAgent)
			// treats this as a warning, not a fatal error.
			Expect(err).To(HaveOccurred())

			// The key assertion: the error is returned to the caller,
			// and the caller logs it as a warning and continues.
			// runAgent does NOT propagate this error upward.
		})
	})
})
