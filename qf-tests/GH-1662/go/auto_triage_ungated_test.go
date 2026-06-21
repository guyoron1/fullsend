package scaffold

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Auto-Triage Ungated Tests

STP Reference: outputs/stp/GH-1662/GH-1662_test_plan.md
Jira: GH-1662

Verifies that issues.opened and issues.edited events trigger auto-triage
WITHOUT any authorization check. This preserves the drive-by bug reporter
workflow where external users can open issues and get automatic triage.
*/

// extractIssuesBlock extracts the issues) case branch from the routing script.
func extractIssuesBlock(workflow string) string {
	route := extractRouteBlock(workflow)
	if route == "" {
		return ""
	}

	// Find the "issues)" case in the EVENT_NAME switch (not "issue_comment")
	// We need to match "issues)" but not "issue_comment)"
	lines := strings.Split(route, "\n")
	var block []string
	inBlock := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inBlock {
			// Match "issues)" but exclude "issue_comment)"
			if trimmed == "issues)" {
				inBlock = true
				block = append(block, line)
			}
			continue
		}
		// Stop at the next case branch
		if trimmed == ";;" && len(block) > 0 {
			block = append(block, line)
			// Check if next meaningful line starts a new case
			continue
		}
		if strings.HasSuffix(trimmed, ")") && !strings.HasPrefix(trimmed, "#") &&
			(strings.Contains(trimmed, "pull_request") || trimmed == "esac") {
			break
		}
		block = append(block, line)
	}
	return strings.Join(block, "\n")
}

func TestAutoTriageUngated(t *testing.T) {
	perOrg, perRepo := loadDispatchWorkflows(t)

	t.Run("external user issue triggers auto-triage", func(t *testing.T) {
		// [test_id:TS-GH-1662-009]
		// Verify that issues.opened event triggers auto-triage WITHOUT any
		// authorization check. The issues path should set STAGE unconditionally.
		for _, workflow := range []struct {
			name    string
			content string
		}{
			{"per-org", perOrg},
			{"per-repo", perRepo},
		} {
			t.Run(workflow.name, func(t *testing.T) {
				issuesBlock := extractIssuesBlock(workflow.content)
				require.NotEmpty(t, issuesBlock, "issues block should exist in %s", workflow.name)

				// issues.opened should set STAGE="triage" without any auth check
				assert.Contains(t, issuesBlock, `"opened"`,
					"issues block must handle the opened action")
				assert.Contains(t, issuesBlock, `STAGE="triage"`,
					"issues.opened must set STAGE to triage")

				// No authorization check in the issues block
				assert.NotContains(t, issuesBlock, "is_authorized",
					"issues.opened path must NOT include is_authorized check")
				assert.NotContains(t, issuesBlock, "is_event_actor_authorized",
					"issues.opened path must NOT include is_event_actor_authorized check")
				assert.NotContains(t, issuesBlock, "COMMENT_AUTHOR_ASSOC",
					"issues block must NOT check COMMENT_AUTHOR_ASSOC")
			})
		}
	})

	t.Run("edited issue re-triggers triage without auth", func(t *testing.T) {
		// [test_id:TS-GH-1662-010]
		// Verify that issues.edited also triggers auto-triage without authorization.
		for _, workflow := range []struct {
			name    string
			content string
		}{
			{"per-org", perOrg},
			{"per-repo", perRepo},
		} {
			t.Run(workflow.name, func(t *testing.T) {
				issuesBlock := extractIssuesBlock(workflow.content)
				require.NotEmpty(t, issuesBlock)

				// issues.edited should also set STAGE="triage"
				assert.Contains(t, issuesBlock, `"edited"`,
					"issues block must handle the edited action")

				// Both opened and edited are in the same condition, no auth check
				assert.Contains(t, issuesBlock, `"opened" || "${EVENT_ACTION}" == "edited"`,
					"opened and edited should be in the same conditional branch")
			})
		}
	})
}
