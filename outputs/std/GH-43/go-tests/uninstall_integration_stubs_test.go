package tests

import (
	. "github.com/onsi/ginkgo/v2"
)

/*
Uninstall Integration Tests

STP Reference: outputs/stp/GH-43/GH-43_test_plan.md
Jira: GH-43
*/

var _ = Describe("[GH-43] Uninstall with harness-first agent discovery", func() {
	/*
	Markers:
	    - tier1

	Preconditions:
	    - Go 1.26+ toolchain available
	    - forge.NewFakeClient() with DirContents, FileContentsRef, FileContents, and Installations maps
	    - Standard CI runner (GitHub Actions, Linux/macOS)
	*/

	Context("org-level uninstall with harness-discovered agents", Ordered, func() {
		/*
		Preconditions:
		    - FakeClient with harness files and matching Installations

		Steps:
		    1. Call runUninstall with fake client
		    2. Verify harness-discovered apps targeted for deletion

		Expected:
		    - Org-level uninstall uses harness slugs for app deletion
		    - Uninstall completes without error
		*/
		PendingIt("[test_id:TS-GH-43-007] should use harness-discovered slugs for app deletion", func() {
			Skip("Phase 1: Design only - awaiting implementation")
		})
	})

	Context("GitHub-specific uninstall with harness-discovered agents", Ordered, func() {
		/*
		Preconditions:
		    - FakeClient with harness files and matching GitHub Installations

		Steps:
		    1. Call runGitHubUninstall with fake client
		    2. Verify harness-discovered apps targeted for deletion

		Expected:
		    - GitHub-specific uninstall uses harness slugs for app deletion
		    - GitHub-specific uninstall completes without error
		*/
		PendingIt("[test_id:TS-GH-43-008] should use harness-discovered slugs for app deletion", func() {
			Skip("Phase 1: Design only - awaiting implementation")
		})
	})

	Context("with legacy config-only setup (no harness files)", Ordered, func() {
		/*
		Preconditions:
		    - No harness files in DirContents
		    - Config.yaml with agents block populated

		Steps:
		    1. Call discoverAgentSlugs with legacy config
		    2. Verify identical slug list to pre-refactoring behavior
		    3. Verify error handling for missing config repo

		Expected:
		    - Legacy config path produces identical results to pre-refactoring behavior
		    - Error handling for missing config repo preserved
		*/
		PendingIt("[test_id:TS-GH-43-009] should produce identical results to pre-refactoring behavior", func() {
			Skip("Phase 1: Design only - awaiting implementation")
		})
	})
})
