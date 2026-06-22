package dispatch

import "testing"

/*
Needs-Info Re-triage Authorization Tests

STP Reference: outputs/stp/GH-79/GH-79_test_plan.md
Jira: GH-79
*/

func TestNeedsInfoRetriage(t *testing.T) {
	/*
	   Preconditions:
	       - Go toolchain 1.26.0+
	       - Dispatch package accessible
	*/

	t.Run("issue author with NONE association can re-trigger triage on needs-info issue", func(t *testing.T) {
		t.Skip("Phase 1: Design only - awaiting implementation")
		/*
		   Preconditions:
		       - Issue with needs-info label
		       - Comment from original issue author
		       - Author has NONE association

		   Steps:
		       1. Invoke dispatch for comment on needs-info issue from original author

		   Expected:
		       - Triage STAGE is dispatched for issue author with NONE on needs-info issue
		*/
	})

	t.Run("non-author with NONE association is blocked from re-triggering triage", func(t *testing.T) {
		t.Skip("Phase 1: Design only - awaiting implementation")
		/*
		   [NEGATIVE]
		   Preconditions:
		       - Issue with needs-info label
		       - Comment from user who is NOT the issue author
		       - Commenter has NONE association

		   Steps:
		       1. Invoke dispatch for comment on needs-info issue from non-author

		   Expected:
		       - No STAGE output set
		       - Non-author NONE commenter is blocked
		*/
	})

	t.Run("non-Bot user with non-NONE association can re-trigger triage", func(t *testing.T) {
		t.Skip("Phase 1: Design only - awaiting implementation")
		/*
		   Preconditions:
		       - Issue with needs-info label
		       - Comment from MEMBER user (non-Bot)

		   Steps:
		       1. Invoke dispatch for comment on needs-info issue from MEMBER

		   Expected:
		       - Triage STAGE is dispatched
		*/
	})
}
