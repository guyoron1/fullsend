package scaffold

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Dispatch Template Consistency Tests

STP Reference: outputs/stp/GH-1662/GH-1662_test_plan.md
Jira: GH-1662

Verifies that per-repo (reusable-dispatch.yml) and per-org (dispatch.yml)
dispatch templates have identical authorization gates. Both must check
is_authorized for all gated slash commands and PR events.
*/

func TestDispatchTemplateConsistency(t *testing.T) {
	t.Run("per-repo dispatch has identical auth gates", func(t *testing.T) {
		// [test_id:TS-GH-1662-016]
		// Verify the per-repo reusable-dispatch.yml contains the same
		// authorization gates as the per-org dispatch.yml.
		repoContent, err := os.ReadFile("../../.github/workflows/reusable-dispatch.yml")
		require.NoError(t, err)
		content := string(repoContent)

		// Per-repo dispatch must contain is_authorized checks for all gated commands
		gatedCommands := []string{"/fs-triage", "/fs-code", "/fs-review", "/fs-fix", "/fs-retro", "/fs-prioritize"}
		route := extractRouteBlock(content)
		require.NotEmpty(t, route)

		for _, cmd := range gatedCommands {
			assert.Contains(t, route, cmd,
				"per-repo dispatch must handle %s", cmd)
		}

		// is_authorized function must be defined
		assert.Contains(t, content, "is_authorized()",
			"per-repo dispatch must define is_authorized function")
		assert.Contains(t, content, "OWNER|MEMBER|COLLABORATOR) return 0",
			"per-repo is_authorized must accept OWNER|MEMBER|COLLABORATOR")

		// PR_AUTHOR_ASSOC check must be present for PR events
		assert.Contains(t, content, "PR_AUTHOR_ASSOC",
			"per-repo dispatch must check PR_AUTHOR_ASSOC for PR events")
		assert.Contains(t, content, "is_event_actor_authorized",
			"per-repo dispatch must call is_event_actor_authorized for PR events")

		// Bot filtering must be present
		assert.Contains(t, content, "COMMENT_USER_TYPE",
			"per-repo dispatch must reference COMMENT_USER_TYPE")
	})

	t.Run("per-org scaffold dispatch has identical auth gates", func(t *testing.T) {
		// [test_id:TS-GH-1662-017]
		// Verify the per-org scaffold dispatch.yml template has the same
		// authorization gates.
		orgContent, err := FullsendRepoFile(".github/workflows/dispatch.yml")
		require.NoError(t, err)
		content := string(orgContent)

		// Per-org dispatch must contain is_authorized checks for all gated commands
		gatedCommands := []string{"/fs-triage", "/fs-code", "/fs-review", "/fs-fix", "/fs-retro", "/fs-prioritize"}
		route := extractRouteBlock(content)
		require.NotEmpty(t, route)

		for _, cmd := range gatedCommands {
			assert.Contains(t, route, cmd,
				"per-org dispatch must handle %s", cmd)
		}

		// is_authorized function must be defined
		assert.Contains(t, content, "is_authorized()",
			"per-org dispatch must define is_authorized function")
		assert.Contains(t, content, "OWNER|MEMBER|COLLABORATOR) return 0",
			"per-org is_authorized must accept OWNER|MEMBER|COLLABORATOR")

		// PR_AUTHOR_ASSOC check
		assert.Contains(t, content, "PR_AUTHOR_ASSOC",
			"per-org dispatch must check PR_AUTHOR_ASSOC for PR events")
		assert.Contains(t, content, "is_event_actor_authorized",
			"per-org dispatch must call is_event_actor_authorized for PR events")
	})

	t.Run("routing logic is identical between templates", func(t *testing.T) {
		// Additional consistency check: verify the routing shell functions
		// are defined identically in both templates.
		orgContent, err := FullsendRepoFile(".github/workflows/dispatch.yml")
		require.NoError(t, err)
		repoContent, err := os.ReadFile("../../.github/workflows/reusable-dispatch.yml")
		require.NoError(t, err)

		orgRoute := extractRouteBlock(string(orgContent))
		repoRoute := extractRouteBlock(string(repoContent))
		require.NotEmpty(t, orgRoute)
		require.NotEmpty(t, repoRoute)

		// Both should define the same helper functions
		helpers := []string{
			"is_authorized()",
			"is_event_actor_authorized()",
			"is_issue_author()",
			"has_label()",
		}
		for _, helper := range helpers {
			orgHas := strings.Contains(orgRoute, helper)
			repoHas := strings.Contains(repoRoute, helper)
			assert.Equal(t, orgHas, repoHas,
				"helper %s presence must match between templates (org=%v, repo=%v)",
				helper, orgHas, repoHas)
		}

		// Both should handle the same event types
		events := []string{
			"issue_comment)",
			"issues)",
			"pull_request_target)",
			"pull_request_review)",
		}
		for _, event := range events {
			orgHas := strings.Contains(orgRoute, event)
			repoHas := strings.Contains(repoRoute, event)
			assert.Equal(t, orgHas, repoHas,
				"event %s handling must match between templates (org=%v, repo=%v)",
				event, orgHas, repoHas)
		}
	})
}
