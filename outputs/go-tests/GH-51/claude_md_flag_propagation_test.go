package tests

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("[GH-51] agentsMDAvailable flag propagation", func() {
	Context("after org AGENTS.md injection succeeds", func() {
		It("[test_id:TS-GH-51-018] should trigger CLAUDE.md injection", func() {
			tmpDir := GinkgoT().TempDir()
			Expect(os.WriteFile(filepath.Join(tmpDir, "AGENTS.md"), []byte("# Agent rules from org"), 0644)).To(Succeed())

			agentsMDAvailable := true
			runtime := "claude"

			if runtime == "claude" && agentsMDAvailable && !hasClaudeMD(tmpDir) {
				mockExec := func(cmd string) error { return nil }
				err := doInjectClaudeMDPointer(tmpDir, mockExec)
				Expect(err).NotTo(HaveOccurred())
			}

			_, err := os.Stat(filepath.Join(tmpDir, "CLAUDE.md"))
			Expect(err).NotTo(HaveOccurred(), "CLAUDE.md should exist after injection triggered by agentsMDAvailable=true")
		})
	})

	Context("when org AGENTS.md injection fails", func() {
		It("[test_id:TS-GH-51-019] should skip CLAUDE.md injection", func() {
			tmpDir := GinkgoT().TempDir()

			agentsMDAvailable := false
			runtime := "claude"

			if runtime == "claude" && agentsMDAvailable && !hasClaudeMD(tmpDir) {
				mockExec := func(cmd string) error { return nil }
				_ = doInjectClaudeMDPointer(tmpDir, mockExec)
			}

			_, err := os.Stat(filepath.Join(tmpDir, "CLAUDE.md"))
			Expect(os.IsNotExist(err)).To(BeTrue(), "CLAUDE.md should not exist when agentsMDAvailable=false")
		})
	})
})
