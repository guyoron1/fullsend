package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("[GH-51] CLAUDE.md git exclude behavior", func() {
	Context("after successful injection", func() {
		It("[test_id:TS-GH-51-008] should add CLAUDE.md to git exclude", func() {
			tmpDir := GinkgoT().TempDir()
			Expect(os.WriteFile(filepath.Join(tmpDir, "AGENTS.md"), []byte("# Agent rules"), 0644)).To(Succeed())

			var capturedCmds []string
			mockExec := func(cmd string) error {
				capturedCmds = append(capturedCmds, cmd)
				return nil
			}

			err := doInjectClaudeMDPointer(tmpDir, mockExec)
			Expect(err).NotTo(HaveOccurred())
			Expect(capturedCmds).NotTo(BeEmpty())

			found := false
			for _, cmd := range capturedCmds {
				if strings.Contains(cmd, "exclude") {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "expected exec command referencing git exclude, got: %v", capturedCmds)
		})

		It("[test_id:TS-GH-51-009] should hide injected file from git status", func() {
			tmpDir := GinkgoT().TempDir()

			// Initialize a git repo
			cmd := exec.Command("git", "init", tmpDir)
			Expect(cmd.Run()).To(Succeed())

			Expect(os.WriteFile(filepath.Join(tmpDir, "AGENTS.md"), []byte("# Agent rules"), 0644)).To(Succeed())

			// Use real exec that runs shell commands in the repo
			realExec := func(shellCmd string) error {
				c := exec.Command("bash", "-c", shellCmd)
				c.Dir = tmpDir
				return c.Run()
			}

			err := doInjectClaudeMDPointer(tmpDir, realExec)
			Expect(err).NotTo(HaveOccurred())

			// Verify CLAUDE.md not in git status
			statusCmd := exec.Command("git", "-C", tmpDir, "status")
			output, err := statusCmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred())
			Expect(string(output)).NotTo(ContainSubstring("CLAUDE.md"))
		})
	})
})
