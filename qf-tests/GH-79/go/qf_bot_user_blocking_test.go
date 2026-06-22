package dispatch_auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Bot User Blocking Tests

STP Reference: outputs/stp/GH-79/GH-79_test_plan.md
STD Reference: outputs/std/GH-79/GH-79_test_description.yaml
Jira: GH-79

Verifies that Bot user types are blocked from slash commands before
authorization checks run, preventing infinite loops and resource waste.
*/

func TestBotUserBlocking(t *testing.T) {
	workflows := bothWorkflows(t)

	t.Run("Bot user blocked from slash commands", func(t *testing.T) {
		// [test_id:TS-GH-79-021] P1
		// Verify Bot user type check prevents dispatch despite any association.
		for _, wf := range workflows {
			t.Run(wf.Name, func(t *testing.T) {
				route := extractRouteBlock(wf.Content)
				require.NotEmpty(t, route)

				// Every slash command path must check COMMENT_USER_TYPE != Bot
				commands := []string{"/fs-triage", "/fs-code", "/fs-review", "/fs-fix", "/fs-prioritize"}
				for _, cmd := range commands {
					section := extractCommandSection(route, cmd)
					if section == "" {
						continue
					}
					assert.Contains(t, section, `"Bot"`,
						"%s must check for Bot user type", cmd)
				}
			})
		}
	})

	t.Run("Bot check precedes authorization check", func(t *testing.T) {
		// [test_id:TS-GH-79-022] P1
		// Verify Bot check runs before is_authorized in the dispatch path.
		for _, wf := range workflows {
			t.Run(wf.Name, func(t *testing.T) {
				route := extractRouteBlock(wf.Content)
				require.NotEmpty(t, route)

				section := extractCommandSection(route, "/fs-triage")
				require.NotEmpty(t, section)

				// In the conditional, Bot check must precede is_authorized
				// Pattern: COMMENT_USER_TYPE != "Bot" && is_authorized
				// This means Bot is checked FIRST (short-circuit evaluation)
				assert.Contains(t, section, `"Bot"`,
					"Bot check must exist in /fs-triage dispatch")
				assert.Contains(t, section, "is_authorized",
					"is_authorized must exist in /fs-triage dispatch")

				// Verify ordering: Bot check appears before is_authorized in the conditional
				botIdx := indexOf(section, `"Bot"`)
				authIdx := indexOf(section, "is_authorized")
				require.NotEqual(t, -1, botIdx, "Bot check not found")
				require.NotEqual(t, -1, authIdx, "is_authorized not found")

				assert.Less(t, botIdx, authIdx,
					"Bot check must precede is_authorized (short-circuit evaluation)")
			})
		}
	})

	t.Run("bot-suffix user login handled correctly", func(t *testing.T) {
		// [test_id:TS-GH-79-023] P1
		// Verify GitHub App bots with [bot] suffix in login are handled.
		for _, wf := range workflows {
			t.Run(wf.Name, func(t *testing.T) {
				route := extractRouteBlock(wf.Content)
				require.NotEmpty(t, route)

				// The dispatch routing must reference COMMENT_USER_TYPE
				// which GitHub sets to "Bot" for app installations
				assert.Contains(t, route, "COMMENT_USER_TYPE",
					"routing must use COMMENT_USER_TYPE for bot detection")

				// For PR fix path, [bot] suffix is also checked
				assert.Contains(t, wf.Content, `[bot]`,
					"workflow must handle [bot] suffix for GitHub App bots")
			})
		}
	})
}

// indexOf returns the position of needle in s, or -1 if not found.
func indexOf(s, needle string) int {
	for i := 0; i <= len(s)-len(needle); i++ {
		if s[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
