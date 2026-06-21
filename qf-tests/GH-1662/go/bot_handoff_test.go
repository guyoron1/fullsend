package scaffold

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Bot-to-Bot Agent Handoff Tests

STP Reference: outputs/stp/GH-1662/GH-1662_test_plan.md
Jira: GH-1662

Verifies that label-based bot-to-bot handoffs (e.g., triage agent adds a label
that triggers code agent) are unaffected by the new authorization gates.
Also verifies bot slash command handling.
*/

// extractLabelBlock extracts the issues/labeled case branch.
func extractLabelBlock(workflow string) string {
	route := extractRouteBlock(workflow)
	if route == "" {
		return ""
	}

	idx := strings.Index(route, `"labeled"`)
	if idx == -1 {
		return ""
	}

	// Get a window around the labeled section
	start := idx - 200
	if start < 0 {
		start = 0
	}
	end := idx + 400
	if end > len(route) {
		end = len(route)
	}
	return route[start:end]
}

func TestBotHandoff(t *testing.T) {
	perOrg, perRepo := loadDispatchWorkflows(t)

	t.Run("label-based handoff triggers downstream agent", func(t *testing.T) {
		// [test_id:TS-GH-1662-011]
		// Verify that label-based bot-to-bot handoffs (via issues.labeled events)
		// are unaffected by authorization gates. Label events should not go through
		// author_association checks.
		for _, workflow := range []struct {
			name    string
			content string
		}{
			{"per-org", perOrg},
			{"per-repo", perRepo},
		} {
			t.Run(workflow.name, func(t *testing.T) {
				labelBlock := extractLabelBlock(workflow.content)
				require.NotEmpty(t, labelBlock, "labeled event handling should exist in %s", workflow.name)

				// Label events should NOT have authorization gates
				assert.NotContains(t, labelBlock, "is_authorized",
					"label event path must NOT include is_authorized check")
				assert.NotContains(t, labelBlock, "is_event_actor_authorized",
					"label event path must NOT include is_event_actor_authorized check")

				// Verify label-triggered stages work
				route := extractRouteBlock(workflow.content)
				assert.Contains(t, route, "ready-to-code",
					"ready-to-code label should trigger code stage")
				assert.Contains(t, route, "ready-for-review",
					"ready-for-review label should trigger review stage")
			})
		}
	})

	t.Run("bot slash command is blocked by non-Bot check", func(t *testing.T) {
		// [test_id:TS-GH-1662-012]
		// Verify that slash commands from Bot user types are handled correctly.
		// The dispatch workflow checks COMMENT_USER_TYPE != "Bot" before processing
		// slash commands, ensuring bot accounts cannot trigger via comments.
		for _, workflow := range []struct {
			name    string
			content string
		}{
			{"per-org", perOrg},
			{"per-repo", perRepo},
		} {
			t.Run(workflow.name, func(t *testing.T) {
				route := extractRouteBlock(workflow.content)
				require.NotEmpty(t, route)

				// All slash command paths check COMMENT_USER_TYPE != "Bot"
				assert.Contains(t, route, `COMMENT_USER_TYPE`,
					"dispatch routing must reference COMMENT_USER_TYPE")
				assert.Contains(t, route, `!= "Bot"`,
					"dispatch routing must filter Bot user type")

				// Bot filtering is applied on slash command paths specifically
				// Each /fs-* command has the Bot check before is_authorized
				fsTriageIdx := strings.Index(route, "/fs-triage")
				require.NotEqual(t, -1, fsTriageIdx)
				// After /fs-triage, the Bot check should appear before STAGE is set
				triageSection := route[fsTriageIdx:]
				stageIdx := strings.Index(triageSection, `STAGE="triage"`)
				botIdx := strings.Index(triageSection, `"Bot"`)
				if stageIdx != -1 && botIdx != -1 {
					assert.Less(t, botIdx, stageIdx,
						"Bot check must appear before STAGE assignment in fs-triage path")
				}
			})
		}
	})
}
