package scaffold

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Regression Tests for Gated Commands

STP Reference: outputs/stp/GH-1662/GH-1662_test_plan.md
Jira: GH-1662

Regression tests verifying that previously gated commands (/fs-fix, /fs-retro,
/fs-prioritize) remain correctly gated after dispatch routing changes.
These commands were gated before GH-1662 and must remain so.
*/

// extractCommandSection extracts the section of the routing script for a
// specific slash command, from the command name to the next ;; terminator.
func extractCommandSection(route, command string) string {
	idx := strings.Index(route, command)
	if idx == -1 {
		return ""
	}
	section := route[idx:]
	// Find the ;; that terminates this case branch
	endIdx := strings.Index(section, ";;")
	if endIdx == -1 {
		return section
	}
	return section[:endIdx]
}

func TestRegressionGatedCommands(t *testing.T) {
	perOrg, perRepo := loadDispatchWorkflows(t)

	t.Run("fs-fix still requires authorization after dispatch routing changes", func(t *testing.T) {
		// [test_id:TS-GH-1662-018]
		// Regression test: /fs-fix must retain its authorization gate.
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

				fixSection := extractCommandSection(route, "/fs-fix")
				require.NotEmpty(t, fixSection, "fs-fix section must exist in %s", workflow.name)

				assert.Contains(t, fixSection, "is_authorized",
					"fs-fix must retain is_authorized check")
				assert.Contains(t, fixSection, `"Bot"`,
					"fs-fix must retain Bot check")
				assert.Contains(t, fixSection, `STAGE="fix"`,
					"fs-fix must set STAGE to fix when authorized")
			})
		}
	})

	t.Run("fs-retro still requires authorization after dispatch routing changes", func(t *testing.T) {
		// [test_id:TS-GH-1662-019]
		// Regression test: /fs-retro must retain its authorization gate.
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

				// /fs-retro shares a case branch with /fullsend
				retroIdx := strings.Index(route, "/fs-retro")
				require.NotEqual(t, -1, retroIdx, "fs-retro must exist in routing")

				// Get the section from /fs-retro to its ;;
				retroSection := route[retroIdx:]
				endIdx := strings.Index(retroSection, ";;")
				if endIdx != -1 {
					retroSection = retroSection[:endIdx]
				}

				assert.Contains(t, retroSection, "is_authorized",
					"fs-retro must retain is_authorized check")
				assert.Contains(t, retroSection, `STAGE="retro"`,
					"fs-retro must set STAGE to retro when authorized")
			})
		}
	})

	t.Run("fs-prioritize still requires authorization after dispatch routing changes", func(t *testing.T) {
		// [test_id:TS-GH-1662-020]
		// Regression test: /fs-prioritize must retain its authorization gate.
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

				prioritizeSection := extractCommandSection(route, "/fs-prioritize")
				require.NotEmpty(t, prioritizeSection,
					"fs-prioritize section must exist in %s", workflow.name)

				assert.Contains(t, prioritizeSection, "is_authorized",
					"fs-prioritize must retain is_authorized check")
				assert.Contains(t, prioritizeSection, `"Bot"`,
					"fs-prioritize must retain Bot check")
				assert.Contains(t, prioritizeSection, `STAGE="prioritize"`,
					"fs-prioritize must set STAGE to prioritize when authorized")
			})
		}
	})
}
