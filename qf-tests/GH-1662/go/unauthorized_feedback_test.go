package scaffold

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Unauthorized Command Feedback Tests

STP Reference: outputs/stp/GH-1662/GH-1662_test_plan.md
Jira: GH-1662

Verifies behavior when unauthorized users attempt slash commands or PR events.
Currently, unauthorized attempts result in a silent skip (no STAGE set).
Tests validate the skip path exists and is well-defined.
*/

func TestUnauthorizedFeedback(t *testing.T) {
	perOrg, perRepo := loadDispatchWorkflows(t)

	t.Run("unauthorized command produces defined skip behavior", func(t *testing.T) {
		// [test_id:TS-GH-1662-021]
		// Verify that when an unauthorized user attempts a slash command, the
		// dispatch has a defined skip path. Currently this is a silent skip —
		// STAGE is not set, and the workflow outputs an empty stage.
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

				// The skip path: when STAGE remains empty, the workflow logs
				// "No stage matched — skipping dispatch" and exits cleanly.
				assert.Contains(t, route, "No stage matched",
					"dispatch must have a skip message when no stage is set")
				assert.Contains(t, route, `echo "stage=" >>`,
					"dispatch must output empty stage when skipping")

				// The skip path exits with code 0 (not an error)
				assert.Contains(t, route, "exit 0",
					"dispatch skip path must exit cleanly (exit 0)")

				// Verify the STAGE starts empty — unauthorized paths leave it empty
				assert.Contains(t, route, `STAGE=""`,
					"STAGE must be initialized to empty string")
			})
		}
	})

	t.Run("silent skip for unauthorized PR event trigger", func(t *testing.T) {
		// [test_id:TS-GH-1662-022]
		// Verify that unauthorized PR events silently skip without errors.
		// When is_event_actor_authorized returns false, the case branch simply
		// doesn't set STAGE, resulting in the clean skip path.
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

				// The PR authorization is an if-block with no else clause.
				// When is_event_actor_authorized fails, execution falls through
				// without setting STAGE, triggering the clean skip path.
				assert.Contains(t, prBlock, "if is_event_actor_authorized",
					"PR path must use conditional authorization check")

				// Verify there's no explicit error/warning for unauthorized PRs
				// (silent skip behavior — the skip message comes from the
				// common "No stage matched" handler, not the PR-specific path)
				prAuthSection := prBlock
				afterAuth := strings.Index(prAuthSection, "is_event_actor_authorized")
				if afterAuth != -1 {
					sectionAfterAuth := prAuthSection[afterAuth:]
					// No explicit error messages in the PR auth section
					assert.NotContains(t, sectionAfterAuth, "::error::",
						"unauthorized PR path should not produce error output")
				}
			})
		}
	})
}
