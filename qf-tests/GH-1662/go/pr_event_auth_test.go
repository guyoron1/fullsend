package scaffold

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
PR Event Authorization Tests

STP Reference: outputs/stp/GH-1662/GH-1662_test_plan.md
Jira: GH-1662

Verifies that pull_request_target event triggers (opened, synchronize,
ready_for_review) enforce actor authorization via PR_AUTHOR_ASSOC.
Member PRs trigger auto-review; external contributor PRs are skipped.
*/

// extractPRTargetBlock extracts the pull_request_target case branch from the
// routing script for precise PR event assertions.
func extractPRTargetBlock(workflow string) string {
	route := extractRouteBlock(workflow)
	if route == "" {
		return ""
	}

	prIdx := strings.Index(route, "pull_request_target)")
	if prIdx == -1 {
		return ""
	}

	section := route[prIdx:]
	// Find the end of this case (next top-level event or esac)
	endMarkers := []string{"pull_request_review)", "esac"}
	endIdx := len(section)
	for _, marker := range endMarkers {
		idx := strings.Index(section[1:], marker)
		if idx != -1 && idx+1 < endIdx {
			endIdx = idx + 1
		}
	}
	return section[:endIdx]
}

func TestPREventAuthorization(t *testing.T) {
	perOrg, perRepo := loadDispatchWorkflows(t)

	t.Run("member PR triggers auto-review", func(t *testing.T) {
		// [test_id:TS-GH-1662-006]
		// Verify that pull_request_target events (opened/synchronize/ready_for_review)
		// from authorized PR authors trigger auto-review by setting STAGE.
		for _, workflow := range []struct {
			name    string
			content string
		}{
			{"per-org", perOrg},
			{"per-repo", perRepo},
		} {
			t.Run(workflow.name, func(t *testing.T) {
				prBlock := extractPRTargetBlock(workflow.content)
				require.NotEmpty(t, prBlock, "pull_request_target block should exist in %s", workflow.name)

				// PR event path must check PR_AUTHOR_ASSOC via is_event_actor_authorized
				assert.Contains(t, prBlock, "is_event_actor_authorized",
					"PR event path must call is_event_actor_authorized")
				assert.Contains(t, prBlock, "PR_AUTHOR_ASSOC",
					"PR event path must reference PR_AUTHOR_ASSOC")

				// When authorized, STAGE should be set to "review"
				assert.Contains(t, prBlock, `STAGE="review"`,
					"authorized PR should set STAGE to review")

				// Covers opened, synchronize, and ready_for_review
				assert.Contains(t, prBlock, "opened|synchronize|ready_for_review",
					"PR event should handle opened, synchronize, and ready_for_review")
			})
		}
	})

	t.Run("external contributor PR skips auto-review", func(t *testing.T) {
		// [test_id:TS-GH-1662-007]
		// Verify that non-member PR authors (NONE, CONTRIBUTOR) do not trigger
		// auto-review. The is_event_actor_authorized function rejects them via
		// the catch-all case.
		for _, workflow := range []struct {
			name    string
			content string
		}{
			{"per-org", perOrg},
			{"per-repo", perRepo},
		} {
			t.Run(workflow.name, func(t *testing.T) {
				prBlock := extractPRTargetBlock(workflow.content)
				require.NotEmpty(t, prBlock)

				// The authorization check gates STAGE assignment — unauthorized
				// PRs simply skip (no STAGE set), resulting in dispatch skip.
				assert.Contains(t, prBlock, "is_event_actor_authorized",
					"PR path must have auth gate to reject unauthorized PR authors")

				// The is_event_actor_authorized function uses the same OWNER|MEMBER|COLLABORATOR
				// set, rejecting NONE and CONTRIBUTOR via catch-all
				assert.Contains(t, workflow.content, `case "${assoc}" in`,
					"is_event_actor_authorized must use parameter-based case statement")
			})
		}
	})

	t.Run("PR synchronize by non-member skips review", func(t *testing.T) {
		// [test_id:TS-GH-1662-008]
		// Verify that the synchronize event type also goes through the
		// is_event_actor_authorized gate — not just opened/ready_for_review.
		for _, workflow := range []struct {
			name    string
			content string
		}{
			{"per-org", perOrg},
			{"per-repo", perRepo},
		} {
			t.Run(workflow.name, func(t *testing.T) {
				prBlock := extractPRTargetBlock(workflow.content)
				require.NotEmpty(t, prBlock)

				// The "synchronize" event is handled in the same case branch as
				// opened and ready_for_review — they all pass through the same
				// is_event_actor_authorized gate.
				assert.Contains(t, prBlock, "synchronize",
					"synchronize must be handled in PR event routing")

				// Verify synchronize is in the same case pattern as opened
				assert.Contains(t, prBlock, "opened|synchronize|ready_for_review",
					"synchronize must be in the same case pattern as opened")

				// The authorization check is inside this combined case branch
				assert.Contains(t, prBlock, "is_event_actor_authorized",
					"the combined opened/synchronize/ready_for_review branch must check authorization")
			})
		}
	})
}
