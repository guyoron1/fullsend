package scaffold

import (
	"testing"
)

/*
Auto-Triage Ungated Tests

STP Reference: outputs/stp/GH-1662/GH-1662_test_plan.md
Jira: GH-1662

Verifies that the issues.opened and issues.edited event paths remain ungated
(no authorization check), preserving the drive-by bug reporter workflow where
external users can open issues and receive automatic triage.
*/

func TestAutoTriageUngated(t *testing.T) {
	/*
	Preconditions:
	    - Dispatch workflow template rendered from scaffold
	*/

	t.Run("external user issue triggers auto-triage", func(t *testing.T) {
		t.Skip("Phase 1: Design only - awaiting implementation")
		/*
		Preconditions:
		    - Dispatch workflow content rendered from scaffold

		Steps:
		    1. Render dispatch workflow content
		    2. Verify issues.opened path has NO authorization gate
		    3. Verify STAGE is set unconditionally for issues.opened

		Expected:
		    - issues.opened event path does NOT include is_authorized check
		    - STAGE is set for triage on issues.opened regardless of author association
		*/
		// [test_id:TS-GH-1662-009]
	})

	t.Run("edited issue re-triggers triage without auth", func(t *testing.T) {
		t.Skip("Phase 1: Design only - awaiting implementation")
		/*
		Preconditions:
		    - Dispatch workflow content rendered from scaffold

		Steps:
		    1. Render dispatch workflow content
		    2. Verify issues.edited path has NO authorization gate

		Expected:
		    - issues.edited event path does NOT include is_authorized check
		    - STAGE is set for triage on issues.edited regardless of author association
		*/
		// [test_id:TS-GH-1662-010]
	})
}
