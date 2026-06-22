package dispatch_auth

import (
	"testing"
)

/*
Needs-Info Re-Triage Tests

STP Reference: outputs/stp/GH-79/GH-79_test_plan.md
Jira: GH-79
*/

func TestNeedsInfoRetriage(t *testing.T) {
	/*
	Preconditions:
	    - Dispatch routing environment configured for issue_comment events
	    - needs-info label handling logic available in dispatch
	*/

	t.Run("issue author re-triggers triage on needs-info", func(t *testing.T) {
		t.Skip("Phase 1: Design only - awaiting implementation")
		/*
		Preconditions:
		    - is_issue_author=true
		    - ISSUE_LABELS contains needs-info (not feature)
		    - COMMENT_AUTHOR_ASSOC=NONE

		Steps:
		    1. Execute dispatch routing for issue_comment event

		Expected:
		    - STAGE=triage when is_issue_author and needs-info label present
		*/
	})

	t.Run("CONTRIBUTOR comment triggers needs-info triage", func(t *testing.T) {
		t.Skip("Phase 1: Design only - awaiting implementation")
		/*
		Preconditions:
		    - COMMENT_AUTHOR_ASSOC=CONTRIBUTOR
		    - ISSUE_LABELS contains needs-info
		    - is_issue_author=false

		Steps:
		    1. Execute dispatch routing for issue_comment event

		Expected:
		    - STAGE=triage for CONTRIBUTOR (non-NONE) on needs-info issue
		*/
	})

	t.Run("NONE non-author blocked from needs-info triage", func(t *testing.T) {
		t.Skip("Phase 1: Design only - awaiting implementation")
		/*
		[NEGATIVE]
		Preconditions:
		    - COMMENT_AUTHOR_ASSOC=NONE
		    - is_issue_author=false
		    - ISSUE_LABELS contains needs-info

		Steps:
		    1. Execute dispatch routing for issue_comment event

		Expected:
		    - STAGE is not set to triage for NONE non-author on needs-info issue
		*/
	})

	t.Run("feature-labeled issues skip needs-info triage", func(t *testing.T) {
		t.Skip("Phase 1: Design only - awaiting implementation")
		/*
		[NEGATIVE]
		Preconditions:
		    - ISSUE_LABELS contains needs-info and feature
		    - COMMENT_AUTHOR_ASSOC=MEMBER

		Steps:
		    1. Execute dispatch routing for issue_comment event

		Expected:
		    - Needs-info triage path not taken when feature label present
		*/
	})
}
