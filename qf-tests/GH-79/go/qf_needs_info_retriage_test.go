package dispatch_auth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Needs-Info Retriage Tests

STP Reference: outputs/stp/GH-79/GH-79_test_plan.md
STD Reference: outputs/std/GH-79/GH-79_test_description.yaml
Jira: GH-79

Verifies the needs-info re-triage logic: issue authors and non-NONE
users can trigger re-triage on needs-info issues, while random NONE
non-authors are blocked.
*/

func TestNeedsInfoRetriage(t *testing.T) {
	workflows := bothWorkflows(t)

	t.Run("issue author re-triggers triage on needs-info", func(t *testing.T) {
		// [test_id:TS-GH-79-029] P2
		// Verify issue author can re-trigger triage on needs-info labeled issues.
		for _, wf := range workflows {
			t.Run(wf.Name, func(t *testing.T) {
				route := extractRouteBlock(wf.Content)
				require.NotEmpty(t, route)

				// The catch-all (*) section in issue_comment handles needs-info
				commentBlock := extractIssueCommentBlock(wf.Content)
				require.NotEmpty(t, commentBlock)

				// Must check for needs-info label
				assert.Contains(t, commentBlock, "needs-info",
					"dispatch must check for needs-info label")

				// Must use is_issue_author function
				assert.Contains(t, commentBlock, "is_issue_author",
					"needs-info path must check is_issue_author")

				// Must set STAGE=triage when conditions met
				// The catch-all in issue_comment should set STAGE=triage for needs-info
				needsInfoSection := commentBlock[strings.Index(commentBlock, "needs-info"):]
				assert.Contains(t, needsInfoSection, `STAGE="triage"`,
					"needs-info re-triage must set STAGE=triage")
			})
		}
	})

	t.Run("CONTRIBUTOR comment triggers needs-info triage", func(t *testing.T) {
		// [test_id:TS-GH-79-030] P2
		// Verify non-NONE association triggers re-triage on needs-info issues.
		for _, wf := range workflows {
			t.Run(wf.Name, func(t *testing.T) {
				commentBlock := extractIssueCommentBlock(wf.Content)
				require.NotEmpty(t, commentBlock)

				// The logic checks: COMMENT_AUTHOR_ASSOC != "NONE" || is_issue_author
				// CONTRIBUTOR is != NONE, so they pass
				assert.Contains(t, commentBlock, `"NONE"`,
					"needs-info path must compare against NONE")
				assert.Contains(t, commentBlock, "is_issue_author",
					"needs-info path must check is_issue_author as fallback")
			})
		}
	})

	t.Run("NONE non-author blocked from needs-info triage", func(t *testing.T) {
		// [test_id:TS-GH-79-031] P2
		// Verify NONE non-author cannot trigger needs-info re-triage.
		for _, wf := range workflows {
			t.Run(wf.Name, func(t *testing.T) {
				commentBlock := extractIssueCommentBlock(wf.Content)
				require.NotEmpty(t, commentBlock)

				// The logic: if COMMENT_AUTHOR_ASSOC != "NONE" || is_issue_author
				// NONE + not issue author → both conditions fail → no triage
				assert.Contains(t, commentBlock, `COMMENT_AUTHOR_ASSOC`,
					"needs-info path must check COMMENT_AUTHOR_ASSOC")

				// Verify the logic requires either non-NONE OR issue author
				assert.Contains(t, commentBlock, "||",
					"needs-info path must use OR logic for NONE-vs-author check")
			})
		}
	})

	t.Run("feature-labeled issues skip needs-info triage", func(t *testing.T) {
		// [test_id:TS-GH-79-032] P2
		// Verify feature-labeled issues do not enter needs-info re-triage.
		for _, wf := range workflows {
			t.Run(wf.Name, func(t *testing.T) {
				commentBlock := extractIssueCommentBlock(wf.Content)
				require.NotEmpty(t, commentBlock)

				// The logic checks: has_label "needs-info" && ! has_label "feature"
				assert.Contains(t, commentBlock, "feature",
					"needs-info path must check for feature label exclusion")

				// The ! (not) before feature check ensures feature-labeled issues are skipped
				assert.Contains(t, commentBlock, `! has_label "feature"`,
					"feature label must exclude issues from needs-info triage")
			})
		}
	})
}
