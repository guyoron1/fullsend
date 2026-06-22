package dispatch_auth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Slash Command Authorization Tests

STP Reference: outputs/stp/GH-79/GH-79_test_plan.md
STD Reference: outputs/std/GH-79/GH-79_test_description.yaml
Jira: GH-79

Verifies that slash commands (/fs-triage, /fs-code, /fs-review) enforce
authorization via is_authorized and that unauthorized associations (NONE,
CONTRIBUTOR, FIRST_TIME_CONTRIBUTOR) are rejected.
*/

func TestSlashCommandAuthorization(t *testing.T) {
	workflows := bothWorkflows(t)

	t.Run("unauthorized user cannot trigger fs-triage", func(t *testing.T) {
		// [test_id:TS-GH-79-001] P0 MVP
		// Verify NONE association is blocked from /fs-triage dispatch.
		for _, wf := range workflows {
			t.Run(wf.Name, func(t *testing.T) {
				route := extractRouteBlock(wf.Content)
				require.NotEmpty(t, route, "route block should exist in %s", wf.Name)

				section := extractCommandSection(route, "/fs-triage")
				require.NotEmpty(t, section, "/fs-triage section must exist")

				// /fs-triage must gate on is_authorized
				assert.Contains(t, section, "is_authorized",
					"/fs-triage dispatch must call is_authorized")

				// The is_authorized function rejects NONE via catch-all
				assert.Contains(t, wf.Content, "*) return 1",
					"is_authorized must have catch-all returning 1 (reject)")

				// NONE is not in the authorized set
				isAuth := extractIsAuthorizedFunction(wf.Content)
				require.NotEmpty(t, isAuth)
				assert.NotContains(t, isAuth, "NONE",
					"NONE must not appear in is_authorized acceptance list")
			})
		}
	})

	t.Run("unauthorized user cannot trigger fs-code", func(t *testing.T) {
		// [test_id:TS-GH-79-002] P0 MVP
		// Verify NONE association is blocked from /fs-code dispatch.
		for _, wf := range workflows {
			t.Run(wf.Name, func(t *testing.T) {
				route := extractRouteBlock(wf.Content)
				require.NotEmpty(t, route)

				section := extractCommandSection(route, "/fs-code")
				require.NotEmpty(t, section, "/fs-code section must exist")

				assert.Contains(t, section, "is_authorized",
					"/fs-code dispatch must call is_authorized")
				assert.Contains(t, section, `STAGE="code"`,
					"/fs-code must set STAGE to code when authorized")
			})
		}
	})

	t.Run("unauthorized user cannot trigger fs-review", func(t *testing.T) {
		// [test_id:TS-GH-79-003] P0 MVP
		// Verify NONE association is blocked from /fs-review dispatch.
		for _, wf := range workflows {
			t.Run(wf.Name, func(t *testing.T) {
				route := extractRouteBlock(wf.Content)
				require.NotEmpty(t, route)

				section := extractCommandSection(route, "/fs-review")
				require.NotEmpty(t, section, "/fs-review section must exist")

				assert.Contains(t, section, "is_authorized",
					"/fs-review dispatch must call is_authorized")
				assert.Contains(t, section, `STAGE="review"`,
					"/fs-review must set STAGE to review when authorized")
			})
		}
	})

	t.Run("COLLABORATOR can trigger all slash commands", func(t *testing.T) {
		// [test_id:TS-GH-79-004] P0 MVP
		// Verify COLLABORATOR is in the authorized association set.
		for _, wf := range workflows {
			t.Run(wf.Name, func(t *testing.T) {
				isAuth := extractIsAuthorizedFunction(wf.Content)
				require.NotEmpty(t, isAuth)

				assert.Contains(t, isAuth, "COLLABORATOR",
					"COLLABORATOR must be in the is_authorized acceptance set")
				assert.Contains(t, isAuth, "return 0",
					"Authorized associations must return 0 (success)")
			})
		}
	})

	t.Run("NONE association rejected for all commands", func(t *testing.T) {
		// [test_id:TS-GH-79-005] P0 MVP
		// Verify NONE is rejected by is_authorized for every slash command.
		commands := []string{"/fs-triage", "/fs-code", "/fs-review", "/fs-fix", "/fs-retro", "/fs-prioritize"}

		for _, wf := range workflows {
			t.Run(wf.Name, func(t *testing.T) {
				route := extractRouteBlock(wf.Content)
				require.NotEmpty(t, route)

				for _, cmd := range commands {
					t.Run(cmd, func(t *testing.T) {
						section := extractCommandSection(route, cmd)
						if section == "" {
							// /fs-retro may be combined with /fullsend
							if cmd == "/fs-retro" {
								assert.Contains(t, route, "/fs-retro",
									"%s must exist in routing", cmd)
								return
							}
							t.Fatalf("%s section not found in routing", cmd)
						}

						// Every slash command must gate on is_authorized
						assert.Contains(t, section, "is_authorized",
							"%s must call is_authorized", cmd)
					})
				}

				// The catch-all rejects anything not OWNER|MEMBER|COLLABORATOR
				assert.Contains(t, wf.Content, "*) return 1",
					"is_authorized catch-all must return 1")
			})
		}
	})

	t.Run("FIRST_TIME_CONTRIBUTOR association rejected", func(t *testing.T) {
		// [test_id:TS-GH-79-006] P0 MVP
		// Verify FIRST_TIME_CONTRIBUTOR is not in the authorized set.
		for _, wf := range workflows {
			t.Run(wf.Name, func(t *testing.T) {
				isAuth := extractIsAuthorizedFunction(wf.Content)
				require.NotEmpty(t, isAuth)

				// FIRST_TIME_CONTRIBUTOR must not be listed
				assert.NotContains(t, isAuth, "FIRST_TIME_CONTRIBUTOR",
					"FIRST_TIME_CONTRIBUTOR must not be in authorized set")

				// The authorized set is exactly OWNER|MEMBER|COLLABORATOR
				assert.Contains(t, wf.Content, "OWNER|MEMBER|COLLABORATOR) return 0",
					"authorized set must be exactly OWNER|MEMBER|COLLABORATOR")
			})
		}
	})
}

func TestSlashCommandAuthorizationAllCommandsGated(t *testing.T) {
	// Regression: verify every slash command path includes bot check + auth check.
	workflows := bothWorkflows(t)

	commands := []struct {
		cmd   string
		stage string
	}{
		{"/fs-triage", "triage"},
		{"/fs-code", "code"},
		{"/fs-review", "review"},
		{"/fs-fix", "fix"},
		{"/fs-prioritize", "prioritize"},
	}

	for _, wf := range workflows {
		t.Run(wf.Name, func(t *testing.T) {
			route := extractRouteBlock(wf.Content)
			require.NotEmpty(t, route)

			for _, tc := range commands {
				t.Run(tc.cmd, func(t *testing.T) {
					section := extractCommandSection(route, tc.cmd)
					require.NotEmpty(t, section, "%s section must exist", tc.cmd)

					// Each command must check Bot user type
					assert.Contains(t, section, `"Bot"`,
						"%s must check for Bot user type", tc.cmd)

					// Each command must call is_authorized
					assert.Contains(t, section, "is_authorized",
						"%s must call is_authorized", tc.cmd)

					// Each command must set STAGE when authorized
					assert.Contains(t, section, strings.ToLower(tc.stage),
						"%s must set stage to %s", tc.cmd, tc.stage)
				})
			}
		})
	}
}
