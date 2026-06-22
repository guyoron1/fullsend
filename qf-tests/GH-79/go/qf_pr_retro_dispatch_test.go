package dispatch_auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
PR Retro Dispatch Tests

STP Reference: outputs/stp/GH-79/GH-79_test_plan.md
STD Reference: outputs/std/GH-79/GH-79_test_description.yaml
Jira: GH-79

Verifies that PR closure unconditionally triggers STAGE=retro without
authorization, since the merge act itself requires write access.
*/

func TestPRRetroDispatch(t *testing.T) {
	workflows := bothWorkflows(t)

	t.Run("PR closure triggers retro unconditionally", func(t *testing.T) {
		// [test_id:TS-GH-79-039] P2
		// Verify pull_request_target.closed sets STAGE=retro without auth check.
		for _, wf := range workflows {
			t.Run(wf.Name, func(t *testing.T) {
				prBlock := extractPRTargetBlock(wf.Content)
				require.NotEmpty(t, prBlock, "pull_request_target block must exist")

				// closed action must be handled
				assert.Contains(t, prBlock, "closed",
					"pull_request_target.closed must be handled")

				// closed must set STAGE=retro
				assert.Contains(t, prBlock, `STAGE="retro"`,
					"PR close must set STAGE=retro")

				// The closed path must NOT call is_event_actor_authorized
				// Find the closed section specifically
				closedIdx := indexOf(prBlock, "closed)")
				require.NotEqual(t, -1, closedIdx, "closed case must exist")

				closedSection := prBlock[closedIdx:]
				// The closed section should set STAGE=retro directly
				assert.Contains(t, closedSection, `STAGE="retro"`,
					"closed section must set STAGE=retro directly")
				assert.NotContains(t, closedSection, "is_event_actor_authorized",
					"closed section must NOT check authorization")
			})
		}
	})

	t.Run("external user PR merge triggers retro", func(t *testing.T) {
		// [test_id:TS-GH-79-040] P2
		// Verify retro fires for all PR closures regardless of author association.
		for _, wf := range workflows {
			t.Run(wf.Name, func(t *testing.T) {
				prBlock := extractPRTargetBlock(wf.Content)
				require.NotEmpty(t, prBlock)

				// The closed case is unconditional — no association check
				closedIdx := indexOf(prBlock, "closed)")
				require.NotEqual(t, -1, closedIdx)

				closedSection := prBlock[closedIdx:]
				// Must not reference PR_AUTHOR_ASSOC in the closed path
				assert.NotContains(t, closedSection, "PR_AUTHOR_ASSOC",
					"closed/retro path must not check PR author association")
			})
		}
	})
}
