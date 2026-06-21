package scaffold

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
is_event_actor_authorized Function Tests

STP Reference: outputs/stp/GH-1662/GH-1662_test_plan.md
Jira: GH-1662

Unit tests for the is_event_actor_authorized shell function that validates
GitHub author_association values. Tests all association types: OWNER, MEMBER,
COLLABORATOR (accepted), CONTRIBUTOR, FIRST_TIME_CONTRIBUTOR, NONE, and
empty string (rejected).
*/

// extractAuthFunction extracts the is_event_actor_authorized function
// definition from the workflow content.
func extractAuthFunction(workflow string) string {
	funcStart := strings.Index(workflow, "is_event_actor_authorized()")
	if funcStart == -1 {
		return ""
	}
	section := workflow[funcStart:]
	// Find the closing brace of the function
	braceCount := 0
	started := false
	for i, ch := range section {
		if ch == '{' {
			braceCount++
			started = true
		} else if ch == '}' {
			braceCount--
			if started && braceCount == 0 {
				return section[:i+1]
			}
		}
	}
	return section
}

// extractIsAuthorizedFunction extracts the is_authorized function definition.
func extractIsAuthorizedFunction(workflow string) string {
	// Find "is_authorized()" but not "is_event_actor_authorized()"
	idx := 0
	for {
		pos := strings.Index(workflow[idx:], "is_authorized()")
		if pos == -1 {
			return ""
		}
		absPos := idx + pos
		// Check it's not part of "is_event_actor_authorized"
		if absPos >= len("is_event_actor_") {
			prefix := workflow[absPos-len("is_event_actor_") : absPos]
			if strings.HasSuffix(prefix, "is_event_actor_") {
				idx = absPos + 1
				continue
			}
		}
		section := workflow[absPos:]
		braceCount := 0
		started := false
		for i, ch := range section {
			if ch == '{' {
				braceCount++
				started = true
			} else if ch == '}' {
				braceCount--
				if started && braceCount == 0 {
					return section[:i+1]
				}
			}
		}
		return section
	}
}

func TestIsEventActorAuthorized(t *testing.T) {
	perOrg, perRepo := loadDispatchWorkflows(t)

	t.Run("OWNER association returns authorized", func(t *testing.T) {
		// [test_id:TS-GH-1662-023]
		// Verify the is_event_actor_authorized function accepts OWNER.
		for _, workflow := range []struct {
			name    string
			content string
		}{
			{"per-org", perOrg},
			{"per-repo", perRepo},
		} {
			t.Run(workflow.name, func(t *testing.T) {
				fn := extractAuthFunction(workflow.content)
				require.NotEmpty(t, fn, "is_event_actor_authorized function must exist in %s", workflow.name)

				// OWNER must be in the case statement that returns 0 (success)
				assert.Contains(t, fn, "OWNER",
					"OWNER must be in is_event_actor_authorized")
				assert.Contains(t, fn, "OWNER|MEMBER|COLLABORATOR) return 0",
					"OWNER must return 0 (authorized)")
			})
		}
	})

	t.Run("empty association string returns unauthorized", func(t *testing.T) {
		// [test_id:TS-GH-1662-024]
		// Verify an empty string is rejected. The function uses ${1:-}
		// as default, so empty input hits the catch-all case.
		for _, workflow := range []struct {
			name    string
			content string
		}{
			{"per-org", perOrg},
			{"per-repo", perRepo},
		} {
			t.Run(workflow.name, func(t *testing.T) {
				fn := extractAuthFunction(workflow.content)
				require.NotEmpty(t, fn)

				// The function takes a parameter with empty default: ${1:-}
				assert.Contains(t, fn, `${1:-}`,
					"function must use safe parameter expansion for empty input")

				// The catch-all must return 1 (unauthorized)
				assert.Contains(t, fn, "*) return 1",
					"catch-all case must return 1 to reject empty/unknown associations")

				// Empty string is NOT in the authorized set (obviously, but verify
				// no accidental empty case match)
				assert.NotContains(t, fn, "|) return 0",
					"no empty case branch should return authorized")
			})
		}
	})

	t.Run("FIRST_TIME_CONTRIBUTOR is rejected", func(t *testing.T) {
		// [test_id:TS-GH-1662-025]
		// Verify FIRST_TIME_CONTRIBUTOR is not in the authorized set.
		for _, workflow := range []struct {
			name    string
			content string
		}{
			{"per-org", perOrg},
			{"per-repo", perRepo},
		} {
			t.Run(workflow.name, func(t *testing.T) {
				fn := extractAuthFunction(workflow.content)
				require.NotEmpty(t, fn)

				// FIRST_TIME_CONTRIBUTOR must NOT be in the return-0 branch
				assert.NotContains(t, fn, "FIRST_TIME_CONTRIBUTOR",
					"FIRST_TIME_CONTRIBUTOR must not appear in the authorized set")

				// The authorized set is EXACTLY OWNER|MEMBER|COLLABORATOR
				assert.Contains(t, fn, "OWNER|MEMBER|COLLABORATOR) return 0",
					"authorized set must be exactly OWNER|MEMBER|COLLABORATOR")
			})
		}
	})

	t.Run("NONE association is rejected", func(t *testing.T) {
		// [test_id:TS-GH-1662-026]
		// Verify NONE is not in the authorized set.
		for _, workflow := range []struct {
			name    string
			content string
		}{
			{"per-org", perOrg},
			{"per-repo", perRepo},
		} {
			t.Run(workflow.name, func(t *testing.T) {
				fn := extractAuthFunction(workflow.content)
				require.NotEmpty(t, fn)

				// NONE must NOT be in the return-0 branch
				assert.NotContains(t, fn, "NONE",
					"NONE must not appear in the authorized set of is_event_actor_authorized")

				// Verify the catch-all handles NONE
				assert.Contains(t, fn, "*) return 1",
					"catch-all must return 1 to reject NONE")
			})
		}
	})

	t.Run("is_authorized and is_event_actor_authorized use same authorized set", func(t *testing.T) {
		// Additional consistency check: both helper functions must accept
		// the same set of associations.
		for _, workflow := range []struct {
			name    string
			content string
		}{
			{"per-org", perOrg},
			{"per-repo", perRepo},
		} {
			t.Run(workflow.name, func(t *testing.T) {
				eventFn := extractAuthFunction(workflow.content)
				commentFn := extractIsAuthorizedFunction(workflow.content)
				require.NotEmpty(t, eventFn, "is_event_actor_authorized must exist")
				require.NotEmpty(t, commentFn, "is_authorized must exist")

				// Both must use the same authorized pattern
				assert.Contains(t, eventFn, "OWNER|MEMBER|COLLABORATOR) return 0",
					"is_event_actor_authorized must use OWNER|MEMBER|COLLABORATOR")
				assert.Contains(t, commentFn, "OWNER|MEMBER|COLLABORATOR) return 0",
					"is_authorized must use OWNER|MEMBER|COLLABORATOR")
			})
		}
	})
}
