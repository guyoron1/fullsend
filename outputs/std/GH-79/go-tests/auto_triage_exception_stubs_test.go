package dispatch_auth

import (
	"testing"
)

/*
Auto-Triage Exception Tests (ADR 0051 Exception)

STP Reference: outputs/stp/GH-79/GH-79_test_plan.md
Jira: GH-79
*/

func TestAutoTriageException(t *testing.T) {
	/*
	Preconditions:
	    - Dispatch routing environment configured for issues events
	    - Auto-triage path does not call is_authorized (ADR 0051 exception)
	*/

	t.Run("any user opening issue triggers triage", func(t *testing.T) {
		t.Skip("Phase 1: Design only - awaiting implementation")
		/*
		Preconditions:
		    - EVENT=issues
		    - ACTION=opened
		    - COMMENT_AUTHOR_ASSOC=NONE (external user)

		Steps:
		    1. Execute dispatch routing for issues.opened event

		Expected:
		    - STAGE=triage regardless of user association
		*/
	})

	t.Run("issue edit by external user triggers triage", func(t *testing.T) {
		t.Skip("Phase 1: Design only - awaiting implementation")
		/*
		Preconditions:
		    - EVENT=issues
		    - ACTION=edited
		    - COMMENT_AUTHOR_ASSOC=NONE

		Steps:
		    1. Execute dispatch routing for issues.edited event

		Expected:
		    - STAGE=triage on issues.edited with NONE association
		*/
	})

	t.Run("NONE association user triggers auto-triage", func(t *testing.T) {
		t.Skip("Phase 1: Design only - awaiting implementation")
		/*
		Preconditions:
		    - EVENT=issues
		    - ACTION=opened
		    - COMMENT_AUTHOR_ASSOC=NONE

		Steps:
		    1. Execute dispatch routing for issue creation by NONE user

		Expected:
		    - STAGE=triage for NONE user on issues.opened
		    - ADR 0051 exception confirmed — NONE users blocked from slash commands but trigger auto-triage
		*/
	})
}
