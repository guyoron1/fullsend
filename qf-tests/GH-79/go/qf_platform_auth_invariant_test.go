package dispatch_auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Platform Auth Invariant Tests

STP Reference: outputs/stp/GH-79/GH-79_test_plan.md
STD Reference: outputs/std/GH-79/GH-79_test_description.yaml
Jira: GH-79

Verifies that authorization is enforced at the platform level and cannot
be bypassed or disabled by per-repo configuration.
*/

func TestPlatformAuthInvariant(t *testing.T) {
	workflows := bothWorkflows(t)

	t.Run("per-repo configuration cannot bypass authorization checks", func(t *testing.T) {
		// [test_id:TS-GH-79-038] P2
		// Verify authorization is hardcoded in the routing logic, not
		// configurable via .fullsend/config.yaml or any other per-repo setting.
		for _, wf := range workflows {
			t.Run(wf.Name, func(t *testing.T) {
				route := extractRouteBlock(wf.Content)
				require.NotEmpty(t, route)

				// is_authorized is defined inline in the routing script, not read from config
				isAuth := extractIsAuthorizedFunction(wf.Content)
				require.NotEmpty(t, isAuth, "is_authorized must be defined inline")

				// The function uses a hardcoded case statement, not a config read
				assert.Contains(t, isAuth, "case",
					"is_authorized must use hardcoded case statement")
				assert.Contains(t, isAuth, "OWNER|MEMBER|COLLABORATOR",
					"authorized associations must be hardcoded")

				// The routing script must not read config for authorization decisions
				assert.NotContains(t, route, "config.yaml",
					"routing script must not read config.yaml for authorization")

				// The role-check step is separate and does NOT affect authorization
				// It only controls which stages are enabled, not WHO can trigger them
				assert.Contains(t, wf.Content, "Check role is enabled",
					"role-check step must exist separately from authorization")
			})
		}
	})
}
