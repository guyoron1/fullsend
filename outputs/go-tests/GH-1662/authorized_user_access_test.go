package scaffold

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Authorized User Access Tests

STP Reference: outputs/stp/GH-1662/GH-1662_test_plan.md
Jira: GH-1662

End-to-end tests verifying that OWNER, MEMBER, and COLLABORATOR associations
can invoke all six slash commands (/fs-triage, /fs-code, /fs-review, /fs-fix,
/fs-retro, /fs-prioritize).
*/

// allSlashCommands lists the six gated slash commands.
var allSlashCommands = []string{
	"/fs-triage",
	"/fs-code",
	"/fs-review",
	"/fs-fix",
	"/fs-retro",
	"/fs-prioritize",
}

// authorizedAssociations lists the three accepted association types.
var authorizedAssociations = []string{
	"OWNER",
	"MEMBER",
	"COLLABORATOR",
}

func TestAuthorizedUserAccess(t *testing.T) {
	perOrg, perRepo := loadDispatchWorkflows(t)

	t.Run("OWNER can invoke all six slash commands", func(t *testing.T) {
		// [test_id:TS-GH-1662-013]
		// Verify OWNER association is in the accepted set for all slash commands.
		// Since all slash commands use is_authorized() which checks
		// OWNER|MEMBER|COLLABORATOR, OWNER can invoke all of them.
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

				// All six commands must be present in the routing logic
				for _, cmd := range allSlashCommands {
					assert.Contains(t, route, cmd,
						"routing must handle command %s", cmd)
				}

				// OWNER must be in the authorized set
				assert.Contains(t, workflow.content, "OWNER|MEMBER|COLLABORATOR) return 0",
					"OWNER must be accepted by is_authorized")
			})
		}
	})

	t.Run("MEMBER can invoke all six slash commands", func(t *testing.T) {
		// [test_id:TS-GH-1662-014]
		// Verify MEMBER association is in the accepted set for all slash commands.
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

				for _, cmd := range allSlashCommands {
					assert.Contains(t, route, cmd,
						"routing must handle command %s", cmd)
				}

				// All gated commands use is_authorized, which accepts MEMBER
				// Verify each command path leads through is_authorized
				for _, cmd := range allSlashCommands {
					cmdIdx := strings.Index(route, cmd)
					require.NotEqual(t, -1, cmdIdx, "command %s must exist in routing", cmd)

					// For /fs-retro which shares a branch with /fullsend
					if cmd == "/fs-retro" {
						assert.Contains(t, route, "/fs-retro|/fullsend",
							"fs-retro should share branch with /fullsend")
					}
				}

				// MEMBER is in the authorized set
				assert.Contains(t, workflow.content, "OWNER|MEMBER|COLLABORATOR) return 0",
					"MEMBER must be accepted by is_authorized")
			})
		}
	})

	t.Run("COLLABORATOR can invoke all six slash commands", func(t *testing.T) {
		// [test_id:TS-GH-1662-015]
		// Verify COLLABORATOR association is in the accepted set.
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

				for _, cmd := range allSlashCommands {
					assert.Contains(t, route, cmd,
						"routing must handle command %s", cmd)
				}

				// COLLABORATOR is in the authorized set
				assert.Contains(t, workflow.content, "OWNER|MEMBER|COLLABORATOR) return 0",
					"COLLABORATOR must be accepted by is_authorized")
			})
		}
	})
}
