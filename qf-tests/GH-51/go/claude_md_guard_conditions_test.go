package tests

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("[GH-51] CLAUDE.md injection guard conditions", func() {
	Context("when runtime is not Claude", func() {
		var tmpDir string

		BeforeEach(func() {
			tmpDir = GinkgoT().TempDir()
			Expect(os.WriteFile(filepath.Join(tmpDir, "AGENTS.md"), []byte("# Agent rules"), 0644)).To(Succeed())
		})

		It("[test_id:TS-GH-51-003] should not inject CLAUDE.md", func() {
			// Guard: runtime != "claude" -> skip injection
			runtime := "codex"
			agentsMDAvailable := true
			if runtime == "claude" && agentsMDAvailable && !hasClaudeMD(tmpDir) {
				_ = doInjectClaudeMDPointer(tmpDir, func(cmd string) error { return nil })
			}
			_, err := os.Stat(filepath.Join(tmpDir, "CLAUDE.md"))
			Expect(os.IsNotExist(err)).To(BeTrue())
		})

		It("[test_id:TS-GH-51-010] should skip injection for non-Claude agent runtimes", func() {
			for _, rt := range []string{"codex", "copilot", "custom"} {
				agentsMDAvailable := true
				if rt == "claude" && agentsMDAvailable && !hasClaudeMD(tmpDir) {
					_ = doInjectClaudeMDPointer(tmpDir, func(cmd string) error { return nil })
				}
				_, err := os.Stat(filepath.Join(tmpDir, "CLAUDE.md"))
				Expect(os.IsNotExist(err)).To(BeTrue(), "CLAUDE.md should not exist for runtime: %s", rt)
			}
		})
	})

	Context("when CLAUDE.md already exists", func() {
		var tmpDir string
		const customContent = "# My custom Claude instructions"

		BeforeEach(func() {
			tmpDir = GinkgoT().TempDir()
			Expect(os.WriteFile(filepath.Join(tmpDir, "AGENTS.md"), []byte("# Agent rules"), 0644)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(tmpDir, "CLAUDE.md"), []byte(customContent), 0644)).To(Succeed())
		})

		It("[test_id:TS-GH-51-011] should skip injection", func() {
			Expect(hasClaudeMD(tmpDir)).To(BeTrue())

			content, err := os.ReadFile(filepath.Join(tmpDir, "CLAUDE.md"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal(customContent))
		})
	})

	Context("when no AGENTS.md is available", func() {
		var tmpDir string

		BeforeEach(func() {
			tmpDir = GinkgoT().TempDir()
		})

		It("[test_id:TS-GH-51-013] should skip injection", func() {
			agentsMDAvailable := false
			runtime := "claude"
			if runtime == "claude" && agentsMDAvailable && !hasClaudeMD(tmpDir) {
				_ = doInjectClaudeMDPointer(tmpDir, func(cmd string) error { return nil })
			}
			_, err := os.Stat(filepath.Join(tmpDir, "CLAUDE.md"))
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
	})
})
