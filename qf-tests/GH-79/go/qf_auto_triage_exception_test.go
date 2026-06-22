package dispatch_auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Auto-Triage Exception Tests

STP Reference: outputs/stp/GH-79/GH-79_test_plan.md
STD Reference: outputs/std/GH-79/GH-79_test_description.yaml
Jira: GH-79

Verifies that issues.opened and issues.edited events trigger auto-triage
WITHOUT authorization check (ADR 0051 exception for drive-by bug reporters).
*/

func TestAutoTriageException(t *testing.T) {
	workflows := bothWorkflows(t)

	t.Run("any user opening issue triggers triage", func(t *testing.T) {
		// [test_id:TS-GH-79-015] P1
		// Verify issues.opened sets STAGE=triage without is_authorized.
		for _, wf := range workflows {
			t.Run(wf.Name, func(t *testing.T) {
				issuesBlock := extractIssuesBlock(wf.Content)
				require.NotEmpty(t, issuesBlock, "issues block must exist in routing")

				// issues.opened must set STAGE=triage
				assert.Contains(t, issuesBlock, "opened",
					"issues.opened must be handled")
				assert.Contains(t, issuesBlock, `STAGE="triage"`,
					"issues.opened must set STAGE=triage")

				// issues block must NOT call is_authorized
				assert.NotContains(t, issuesBlock, "is_authorized",
					"issues.opened/edited must NOT call is_authorized — ADR 0051 exception")
			})
		}
	})

	t.Run("issue edit by external user triggers triage", func(t *testing.T) {
		// [test_id:TS-GH-79-016] P1
		// Verify issues.edited also triggers triage without authorization.
		for _, wf := range workflows {
			t.Run(wf.Name, func(t *testing.T) {
				issuesBlock := extractIssuesBlock(wf.Content)
				require.NotEmpty(t, issuesBlock)

				assert.Contains(t, issuesBlock, "edited",
					"issues.edited must be handled")
				assert.Contains(t, issuesBlock, `STAGE="triage"`,
					"issues.edited must set STAGE=triage")
			})
		}
	})

	t.Run("NONE association user triggers auto-triage on issue open", func(t *testing.T) {
		// [test_id:TS-GH-79-017] P1
		// Explicitly confirm NONE users can trigger auto-triage via issue events.
		for _, wf := range workflows {
			t.Run(wf.Name, func(t *testing.T) {
				issuesBlock := extractIssuesBlock(wf.Content)
				require.NotEmpty(t, issuesBlock)

				// The issues.opened path has no association check at all
				assert.NotContains(t, issuesBlock, "COMMENT_AUTHOR_ASSOC",
					"issues event path must not check author association")
				assert.NotContains(t, issuesBlock, "is_authorized",
					"issues event path must not call is_authorized")
			})
		}
	})
}
