package dispatch_auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
PR-Triggered Dispatch Authorization Tests

STP Reference: outputs/stp/GH-79/GH-79_test_plan.md
STD Reference: outputs/std/GH-79/GH-79_test_description.yaml
Jira: GH-79

Verifies that pull_request_target events (opened, synchronize,
ready_for_review) enforce authorization via is_event_actor_authorized
based on PR author association.
*/

func TestPRTriggeredDispatchAuthorization(t *testing.T) {
	workflows := bothWorkflows(t)

	t.Run("member PR author triggers auto-review", func(t *testing.T) {
		// [test_id:TS-GH-79-007] P0 MVP
		// Verify MEMBER PR author passes is_event_actor_authorized and
		// STAGE=review is set for opened/synchronize/ready_for_review events.
		for _, wf := range workflows {
			t.Run(wf.Name, func(t *testing.T) {
				prBlock := extractPRTargetBlock(wf.Content)
				require.NotEmpty(t, prBlock, "pull_request_target block must exist")

				// opened/synchronize/ready_for_review events must call is_event_actor_authorized
				assert.Contains(t, prBlock, "is_event_actor_authorized",
					"PR events must call is_event_actor_authorized")

				// MEMBER must be in the is_event_actor_authorized acceptance set
				actorAuth := extractIsEventActorAuthorizedFunction(wf.Content)
				require.NotEmpty(t, actorAuth)
				assert.Contains(t, actorAuth, "MEMBER",
					"MEMBER must be accepted by is_event_actor_authorized")

				// Authorized PR authors set STAGE=review
				assert.Contains(t, prBlock, `STAGE="review"`,
					"authorized PR events must set STAGE=review")
			})
		}
	})

	t.Run("external PR author blocked from auto-review", func(t *testing.T) {
		// [test_id:TS-GH-79-008] P0 MVP
		// Verify NONE PR author is rejected by is_event_actor_authorized.
		for _, wf := range workflows {
			t.Run(wf.Name, func(t *testing.T) {
				actorAuth := extractIsEventActorAuthorizedFunction(wf.Content)
				require.NotEmpty(t, actorAuth)

				// NONE is not in the authorized set
				assert.NotContains(t, actorAuth, "NONE",
					"NONE must not be in is_event_actor_authorized acceptance set")

				// Catch-all returns failure
				assert.Contains(t, actorAuth, "*) return 1",
					"is_event_actor_authorized must have catch-all returning 1")
			})
		}
	})

	t.Run("synchronize event checks PR author association", func(t *testing.T) {
		// [test_id:TS-GH-79-009] P0 MVP
		// Verify synchronize event is covered by the PR authorization gate.
		for _, wf := range workflows {
			t.Run(wf.Name, func(t *testing.T) {
				prBlock := extractPRTargetBlock(wf.Content)
				require.NotEmpty(t, prBlock)

				// synchronize must be handled in the PR target block
				assert.Contains(t, prBlock, "synchronize",
					"synchronize event must be handled in pull_request_target routing")

				// The synchronize case must call is_event_actor_authorized
				assert.Contains(t, prBlock, "is_event_actor_authorized",
					"synchronize event must check PR author authorization")
			})
		}
	})

	t.Run("ready_for_review event checks PR author association", func(t *testing.T) {
		// [test_id:TS-GH-79-010] P0 MVP
		// Verify ready_for_review event is covered by the PR authorization gate.
		for _, wf := range workflows {
			t.Run(wf.Name, func(t *testing.T) {
				prBlock := extractPRTargetBlock(wf.Content)
				require.NotEmpty(t, prBlock)

				assert.Contains(t, prBlock, "ready_for_review",
					"ready_for_review event must be handled in pull_request_target routing")

				assert.Contains(t, prBlock, "is_event_actor_authorized",
					"ready_for_review event must check PR author authorization")
			})
		}
	})
}
