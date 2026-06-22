package dispatch_auth

import (
	"testing"
)

/*
Authorization Association Evaluation Tests

STP Reference: outputs/stp/GH-79/GH-79_test_plan.md
Jira: GH-79
*/

func TestAuthAssociationEvaluation(t *testing.T) {
	/*
	Preconditions:
	    - is_authorized and is_event_actor_authorized functions available
	    - Case-statement matching OWNER|MEMBER|COLLABORATOR implemented
	*/

	t.Run("org owners recognized as authorized", func(t *testing.T) {
		t.Skip("Phase 1: Design only - awaiting implementation")
		/*
		Preconditions:
		    - COMMENT_AUTHOR_ASSOC=OWNER

		Steps:
		    1. Call is_authorized()

		Expected:
		    - is_authorized returns 0 for OWNER
		*/
	})

	t.Run("org members recognized as authorized", func(t *testing.T) {
		t.Skip("Phase 1: Design only - awaiting implementation")
		/*
		Preconditions:
		    - COMMENT_AUTHOR_ASSOC=MEMBER

		Steps:
		    1. Call is_authorized()

		Expected:
		    - is_authorized returns 0 for MEMBER
		*/
	})

	t.Run("repository collaborators recognized as authorized", func(t *testing.T) {
		t.Skip("Phase 1: Design only - awaiting implementation")
		/*
		Preconditions:
		    - COMMENT_AUTHOR_ASSOC=COLLABORATOR

		Steps:
		    1. Call is_authorized()

		Expected:
		    - is_authorized returns 0 for COLLABORATOR
		*/
	})

	t.Run("one-time contributors rejected as unauthorized", func(t *testing.T) {
		t.Skip("Phase 1: Design only - awaiting implementation")
		/*
		[NEGATIVE]
		Preconditions:
		    - COMMENT_AUTHOR_ASSOC=CONTRIBUTOR

		Steps:
		    1. Call is_authorized()

		Expected:
		    - is_authorized returns non-zero for CONTRIBUTOR
		*/
	})

	t.Run("PR author with no association rejected", func(t *testing.T) {
		t.Skip("Phase 1: Design only - awaiting implementation")
		/*
		[NEGATIVE]
		Preconditions:
		    - PR_AUTHOR_ASSOC=NONE

		Steps:
		    1. Call is_event_actor_authorized(NONE)

		Expected:
		    - is_event_actor_authorized returns non-zero for NONE
		*/
	})
}
