package dispatch_auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Authorized User Dispatch Tests

STP Reference: outputs/stp/GH-79/GH-79_test_plan.md
STD Reference: outputs/std/GH-79/GH-79_test_description.yaml
Jira: GH-79

Verifies that OWNER, MEMBER, and COLLABORATOR associations can trigger
all slash commands, and that /fs-code is blocked when a PR already exists.
*/

func TestAuthorizedUserDispatch(t *testing.T) {
	workflows := bothWorkflows(t)

	t.Run("OWNER dispatches all slash commands", func(t *testing.T) {
		// [test_id:TS-GH-79-011] P0 MVP
		// Verify OWNER is in the is_authorized acceptance set.
		for _, wf := range workflows {
			t.Run(wf.Name, func(t *testing.T) {
				isAuth := extractIsAuthorizedFunction(wf.Content)
				require.NotEmpty(t, isAuth)

				assert.Contains(t, isAuth, "OWNER",
					"OWNER must be in the is_authorized acceptance set")
				assert.Contains(t, isAuth, "return 0",
					"Authorized associations must return 0")
			})
		}
	})

	t.Run("MEMBER dispatches all slash commands", func(t *testing.T) {
		// [test_id:TS-GH-79-012] P0 MVP
		// Verify MEMBER is in the is_authorized acceptance set.
		for _, wf := range workflows {
			t.Run(wf.Name, func(t *testing.T) {
				isAuth := extractIsAuthorizedFunction(wf.Content)
				require.NotEmpty(t, isAuth)

				assert.Contains(t, isAuth, "MEMBER",
					"MEMBER must be in the is_authorized acceptance set")
			})
		}
	})

	t.Run("COLLABORATOR dispatches all slash commands", func(t *testing.T) {
		// [test_id:TS-GH-79-013] P0 MVP
		// Verify COLLABORATOR is in the is_authorized acceptance set.
		for _, wf := range workflows {
			t.Run(wf.Name, func(t *testing.T) {
				isAuth := extractIsAuthorizedFunction(wf.Content)
				require.NotEmpty(t, isAuth)

				assert.Contains(t, isAuth, "COLLABORATOR",
					"COLLABORATOR must be in the is_authorized acceptance set")
			})
		}
	})

	t.Run("fs-code blocked when PR already exists", func(t *testing.T) {
		// [test_id:TS-GH-79-014] P0 MVP
		// Verify /fs-code checks ISSUE_HAS_PR before dispatching.
		for _, wf := range workflows {
			t.Run(wf.Name, func(t *testing.T) {
				route := extractRouteBlock(wf.Content)
				require.NotEmpty(t, route)

				section := extractCommandSection(route, "/fs-code")
				require.NotEmpty(t, section, "/fs-code section must exist")

				// /fs-code must check ISSUE_HAS_PR
				assert.Contains(t, section, "ISSUE_HAS_PR",
					"/fs-code must check for existing PR via ISSUE_HAS_PR")

				// When PR exists (ISSUE_HAS_PR == "true"), code is not dispatched
				// The logic is: if ISSUE_HAS_PR == "false" then allow
				assert.Contains(t, section, `"false"`,
					"/fs-code must only proceed when ISSUE_HAS_PR is false")
			})
		}
	})
}
