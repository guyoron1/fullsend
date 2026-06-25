package scaffold

import (
	"testing"
)

/*
Resolver skip_install Tests

STP Reference: outputs/stp/GH-1270/GH-1270_test_plan.md
Jira: GH-1270
*/

func TestResolverSkipInstall(t *testing.T) {
	/*
	Preconditions:
	    - Registry contains entries with skip_install:true (e.g., shellcheck-py)
	*/

	t.Run("[test_id:TS-GH-1270-009] should recognize but not install tool with skip_install true", func(t *testing.T) {
		t.Skip("Phase 1: Design only - awaiting implementation")
		/*
		Preconditions:
		    - Fixture with hook matching a skip_install:true registry entry
		    - No warning expected for the skipped tool

		Steps:
		    1. Call resolve() with fixture

		Expected:
		    - Tool with skip_install:true does not appear in install manifest
		    - No warning emitted for the skipped tool's hooks
		*/
	})

	t.Run("[test_id:TS-GH-1270-010] should omit skip_install tool from resolved manifest output", func(t *testing.T) {
		t.Skip("Phase 1: Design only - awaiting implementation")
		/*
		Preconditions:
		    - Resolver run end-to-end with skip_install tool present in registry

		Steps:
		    1. Parse JSON output and search for skip_install tool name

		Expected:
		    - JSON manifest does not contain any reference to skip_install tool
		*/
	})
}
