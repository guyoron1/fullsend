package dispatch_auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Auth Association Evaluation Tests

STP Reference: outputs/stp/GH-79/GH-79_test_plan.md
STD Reference: outputs/std/GH-79/GH-79_test_description.yaml
Jira: GH-79

Verifies the is_authorized and is_event_actor_authorized functions
correctly evaluate each GitHub author_association value.
*/

func TestAuthAssociationEvaluation(t *testing.T) {
	workflows := bothWorkflows(t)

	t.Run("org owners are recognized as authorized", func(t *testing.T) {
		// [test_id:TS-GH-79-024] P1
		// Verify OWNER passes is_authorized.
		for _, wf := range workflows {
			t.Run(wf.Name, func(t *testing.T) {
				isAuth := extractIsAuthorizedFunction(wf.Content)
				require.NotEmpty(t, isAuth)

				assert.Contains(t, isAuth, "OWNER",
					"OWNER must be in the is_authorized case pattern")
				assert.Contains(t, isAuth, "return 0",
					"Matching associations must return 0 (authorized)")
			})
		}
	})

	t.Run("org members are recognized as authorized", func(t *testing.T) {
		// [test_id:TS-GH-79-025] P1
		// Verify MEMBER passes is_authorized.
		for _, wf := range workflows {
			t.Run(wf.Name, func(t *testing.T) {
				isAuth := extractIsAuthorizedFunction(wf.Content)
				require.NotEmpty(t, isAuth)

				assert.Contains(t, isAuth, "MEMBER",
					"MEMBER must be in the is_authorized case pattern")
			})
		}
	})

	t.Run("repository collaborators are recognized as authorized", func(t *testing.T) {
		// [test_id:TS-GH-79-026] P1
		// Verify COLLABORATOR passes is_authorized.
		for _, wf := range workflows {
			t.Run(wf.Name, func(t *testing.T) {
				isAuth := extractIsAuthorizedFunction(wf.Content)
				require.NotEmpty(t, isAuth)

				assert.Contains(t, isAuth, "COLLABORATOR",
					"COLLABORATOR must be in the is_authorized case pattern")
			})
		}
	})

	t.Run("one-time contributors are rejected as unauthorized", func(t *testing.T) {
		// [test_id:TS-GH-79-027] P1
		// Verify CONTRIBUTOR is not in the authorized set.
		for _, wf := range workflows {
			t.Run(wf.Name, func(t *testing.T) {
				isAuth := extractIsAuthorizedFunction(wf.Content)
				require.NotEmpty(t, isAuth)

				// Only OWNER, MEMBER, COLLABORATOR return 0
				assert.Contains(t, wf.Content, "OWNER|MEMBER|COLLABORATOR) return 0",
					"authorized set must be exactly OWNER|MEMBER|COLLABORATOR")

				// CONTRIBUTOR is NOT in the set — it falls through to *) return 1
				assert.NotContains(t, isAuth, "CONTRIBUTOR) return 0",
					"CONTRIBUTOR must not return 0 in is_authorized")
			})
		}
	})

	t.Run("PR author with no association is rejected", func(t *testing.T) {
		// [test_id:TS-GH-79-028] P1
		// Verify is_event_actor_authorized rejects NONE for PR authors.
		for _, wf := range workflows {
			t.Run(wf.Name, func(t *testing.T) {
				actorAuth := extractIsEventActorAuthorizedFunction(wf.Content)
				require.NotEmpty(t, actorAuth)

				// Same pattern as is_authorized: only OWNER|MEMBER|COLLABORATOR accepted
				assert.Contains(t, actorAuth, "OWNER",
					"is_event_actor_authorized must accept OWNER")
				assert.Contains(t, actorAuth, "MEMBER",
					"is_event_actor_authorized must accept MEMBER")
				assert.Contains(t, actorAuth, "COLLABORATOR",
					"is_event_actor_authorized must accept COLLABORATOR")

				// Catch-all rejects everything else including NONE
				assert.Contains(t, actorAuth, "*) return 1",
					"is_event_actor_authorized must reject non-matching associations")
				assert.NotContains(t, actorAuth, "NONE",
					"NONE must not appear in is_event_actor_authorized acceptance list")
			})
		}
	})
}
