package dispatch_auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Bot Label Workflow Tests

STP Reference: outputs/stp/GH-79/GH-79_test_plan.md
STD Reference: outputs/std/GH-79/GH-79_test_description.yaml
Jira: GH-79

Verifies that label-based bot-to-bot handoff (triage → code → review)
works without authorization checks. Label application requires write
access, which serves as implicit authorization.
*/

func TestBotLabelWorkflows(t *testing.T) {
	workflows := bothWorkflows(t)

	t.Run("ready-to-code label triggers code dispatch", func(t *testing.T) {
		// [test_id:TS-GH-79-018] P1
		// Verify issues.labeled with ready-to-code sets STAGE=code.
		for _, wf := range workflows {
			t.Run(wf.Name, func(t *testing.T) {
				issuesBlock := extractIssuesBlock(wf.Content)
				require.NotEmpty(t, issuesBlock)

				assert.Contains(t, issuesBlock, "labeled",
					"issues.labeled must be handled")
				assert.Contains(t, issuesBlock, "ready-to-code",
					"ready-to-code label must be checked")
				assert.Contains(t, issuesBlock, `STAGE="code"`,
					"ready-to-code label must set STAGE=code")
			})
		}
	})

	t.Run("ready-for-review label triggers review dispatch", func(t *testing.T) {
		// [test_id:TS-GH-79-019] P1
		// Verify issues.labeled with ready-for-review sets STAGE=review.
		for _, wf := range workflows {
			t.Run(wf.Name, func(t *testing.T) {
				issuesBlock := extractIssuesBlock(wf.Content)
				require.NotEmpty(t, issuesBlock)

				assert.Contains(t, issuesBlock, "ready-for-review",
					"ready-for-review label must be checked")
				assert.Contains(t, issuesBlock, `STAGE="review"`,
					"ready-for-review label must set STAGE=review")
			})
		}
	})

	t.Run("label dispatch bypasses is_authorized check", func(t *testing.T) {
		// [test_id:TS-GH-79-020] P1
		// Verify the label dispatch path does not invoke is_authorized.
		for _, wf := range workflows {
			t.Run(wf.Name, func(t *testing.T) {
				issuesBlock := extractIssuesBlock(wf.Content)
				require.NotEmpty(t, issuesBlock)

				// The issues event block (handling opened/edited/labeled)
				// must NOT call is_authorized
				assert.NotContains(t, issuesBlock, "is_authorized",
					"label dispatch path must not call is_authorized — implicit via write access")
			})
		}
	})
}
