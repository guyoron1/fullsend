package scaffold

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Slash Command Authorization Tests

STP Reference: outputs/stp/GH-1662/GH-1662_test_plan.md
Jira: GH-1662

Verifies that all slash commands (/fs-triage, /fs-code, /fs-review) enforce
authorization based on comment author association (OWNER, MEMBER, COLLABORATOR
are accepted; NONE, CONTRIBUTOR, FIRST_TIME_CONTRIBUTOR are rejected).
*/

// loadDispatchWorkflows returns the per-org and per-repo dispatch workflow
// content for use in authorization gate tests. Both must contain identical
// routing logic.
func loadDispatchWorkflows(t *testing.T) (perOrg, perRepo string) {
	t.Helper()

	orgContent, err := FullsendRepoFile(".github/workflows/dispatch.yml")
	require.NoError(t, err, "reading per-org dispatch.yml from scaffold")
	require.NotEmpty(t, orgContent, "per-org dispatch.yml should not be empty")

	repoContent, err := os.ReadFile("../../.github/workflows/reusable-dispatch.yml")
	require.NoError(t, err, "reading per-repo reusable-dispatch.yml")
	require.NotEmpty(t, repoContent, "per-repo reusable-dispatch.yml should not be empty")

	return string(orgContent), string(repoContent)
}

// extractRouteBlock extracts the shell script from the "Determine stage" step.
// This isolates the routing logic for precise assertion testing.
func extractRouteBlock(workflow string) string {
	// The route block starts after "Determine stage" and ends before the next
	// step (identified by "- name:"). We look for the run: block content.
	idx := strings.Index(workflow, "Determine stage")
	if idx == -1 {
		return ""
	}
	rest := workflow[idx:]

	// Find the "run: |" line that starts the script
	runIdx := strings.Index(rest, "run: |")
	if runIdx == -1 {
		return ""
	}
	script := rest[runIdx:]

	// Find the next step marker to bound the block
	lines := strings.Split(script, "\n")
	var block []string
	started := false
	for _, line := range lines {
		if !started {
			if strings.Contains(line, "run: |") {
				started = true
			}
			continue
		}
		// Stop at next step definition (unindented "- name:")
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- name:") {
			break
		}
		block = append(block, line)
	}
	return strings.Join(block, "\n")
}

func TestSlashCommandAuthorization(t *testing.T) {
	perOrg, perRepo := loadDispatchWorkflows(t)

	t.Run("authorized user triggers fs-triage successfully", func(t *testing.T) {
		// [test_id:TS-GH-1662-001]
		// Verify the /fs-triage path requires authorization via is_authorized
		// and that authorized associations (OWNER, MEMBER, COLLABORATOR) are accepted.
		for _, workflow := range []struct {
			name    string
			content string
		}{
			{"per-org", perOrg},
			{"per-repo", perRepo},
		} {
			t.Run(workflow.name, func(t *testing.T) {
				route := extractRouteBlock(workflow.content)
				require.NotEmpty(t, route, "route block should be found in %s", workflow.name)

				// The /fs-triage command path must call is_authorized
				assert.Contains(t, route, "/fs-triage")
				assert.Contains(t, route, "is_authorized",
					"dispatch routing must call is_authorized for slash commands")

				// The is_authorized function must accept OWNER, MEMBER, COLLABORATOR
				assert.Contains(t, workflow.content, "OWNER|MEMBER|COLLABORATOR",
					"is_authorized must accept OWNER, MEMBER, and COLLABORATOR")

				// Verify /fs-triage sets STAGE="triage" when authorized
				assert.Contains(t, route, `STAGE="triage"`,
					"fs-triage must set STAGE to triage when authorized")
			})
		}
	})

	t.Run("unauthorized user cannot trigger fs-triage", func(t *testing.T) {
		// [test_id:TS-GH-1662-002]
		// Verify the is_authorized function rejects non-member associations via
		// the catch-all (*) case that returns 1 (failure).
		for _, workflow := range []struct {
			name    string
			content string
		}{
			{"per-org", perOrg},
			{"per-repo", perRepo},
		} {
			t.Run(workflow.name, func(t *testing.T) {
				// The is_authorized function must have a catch-all that returns failure
				assert.Contains(t, workflow.content, "*) return 1",
					"is_authorized must reject non-matching associations via catch-all")

				// NONE and CONTRIBUTOR are NOT in the authorized set
				// The authorized set is exactly OWNER|MEMBER|COLLABORATOR
				assert.NotContains(t, workflow.content, "NONE|",
					"NONE must not appear in the authorized association set")
				assert.NotContains(t, workflow.content, "|NONE",
					"NONE must not appear in the authorized association set")
			})
		}
	})

	t.Run("unauthorized user cannot trigger fs-code", func(t *testing.T) {
		// [test_id:TS-GH-1662-003]
		// Verify /fs-code has an is_authorized gate to prevent unauthorized users
		// from triggering expensive code generation inference.
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

				// The /fs-code path exists
				assert.Contains(t, route, "/fs-code")
				// /fs-code path must include is_authorized check
				// Find the /fs-code section and verify it contains is_authorized
				codeIdx := strings.Index(route, "/fs-code")
				require.NotEqual(t, -1, codeIdx, "fs-code command must exist in routing")

				// Get section after /fs-code up to the next command
				codeSection := route[codeIdx:]
				nextCmd := strings.Index(codeSection[1:], "/fs-")
				if nextCmd != -1 {
					codeSection = codeSection[:nextCmd+1]
				}
				assert.Contains(t, codeSection, "is_authorized",
					"fs-code dispatch path must include is_authorized check")
				assert.Contains(t, codeSection, `STAGE="code"`,
					"fs-code must set STAGE to code when authorized")
			})
		}
	})

	t.Run("unauthorized user cannot trigger fs-review", func(t *testing.T) {
		// [test_id:TS-GH-1662-004]
		// Verify /fs-review has an is_authorized gate.
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

				assert.Contains(t, route, "/fs-review")
				reviewIdx := strings.Index(route, "/fs-review")
				require.NotEqual(t, -1, reviewIdx)

				reviewSection := route[reviewIdx:]
				nextCmd := strings.Index(reviewSection[1:], "/fs-")
				if nextCmd != -1 {
					reviewSection = reviewSection[:nextCmd+1]
				}
				assert.Contains(t, reviewSection, "is_authorized",
					"fs-review dispatch path must include is_authorized check")
				assert.Contains(t, reviewSection, `STAGE="review"`,
					"fs-review must set STAGE to review when authorized")
			})
		}
	})

	t.Run("CONTRIBUTOR association is rejected for slash commands", func(t *testing.T) {
		// [test_id:TS-GH-1662-005]
		// Verify CONTRIBUTOR is not in the authorized associations set.
		// The is_authorized and is_event_actor_authorized functions only accept
		// OWNER|MEMBER|COLLABORATOR — CONTRIBUTOR is caught by the *) fallthrough.
		for _, workflow := range []struct {
			name    string
			content string
		}{
			{"per-org", perOrg},
			{"per-repo", perRepo},
		} {
			t.Run(workflow.name, func(t *testing.T) {
				// The authorized set is exactly OWNER|MEMBER|COLLABORATOR.
				// CONTRIBUTOR must NOT be part of this set.
				assert.Contains(t, workflow.content, "OWNER|MEMBER|COLLABORATOR) return 0",
					"authorized set must be exactly OWNER|MEMBER|COLLABORATOR")
				assert.NotContains(t, workflow.content, "CONTRIBUTOR) return 0",
					"CONTRIBUTOR must not be in the authorized return-0 set")
			})
		}
	})
}
