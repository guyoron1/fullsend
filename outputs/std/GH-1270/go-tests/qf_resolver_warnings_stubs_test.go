package scaffold

import (
	"testing"
)

/*
Resolver Warning Message Tests

STP Reference: outputs/stp/GH-1270/GH-1270_test_plan.md
Jira: GH-1270
*/

func TestResolverWarnings(t *testing.T) {
	/*
	Preconditions:
	    - Default .pre-commit-tools.yaml registry loaded
	    - Resolver stderr captured for warning verification
	*/

	t.Run("[test_id:TS-GH-1270-006] should emit warning with command name for unregistered system hook", func(t *testing.T) {
		t.Skip("Phase 1: Design only - awaiting implementation")
		/*
		Preconditions:
		    - Fixture with language:system hook not in registry

		Steps:
		    1. Call resolve() and capture stderr

		Expected:
		    - Warning message is emitted to stderr
		    - Warning includes the hook command name
		    - Warning does not cause resolver failure
		*/
	})

	t.Run("[test_id:TS-GH-1270-007] should emit warning mentioning Go toolchain for language:golang hook", func(t *testing.T) {
		t.Skip("Phase 1: Design only - awaiting implementation")
		/*
		Preconditions:
		    - Fixture with language:golang hook (e.g., tekwizely/pre-commit-golang)

		Steps:
		    1. Call resolve() and capture stderr

		Expected:
		    - Warning mentions Go toolchain or "go" requirement
		    - Warning is specific to language:golang hooks
		*/
	})

	t.Run("[test_id:TS-GH-1270-008] should not emit warning for language:python hooks", func(t *testing.T) {
		t.Skip("Phase 1: Design only - awaiting implementation")
		/*
		[NEGATIVE]
		Preconditions:
		    - Fixture with language:python hook (auto-managed by pre-commit)

		Steps:
		    1. Call resolve() and capture stderr

		Expected:
		    - No warning emitted for language:python hooks
		    - Resolver silently skips python-language hooks
		*/
	})
}
