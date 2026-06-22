package dispatch_auth

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/scaffold"
)

/*
Shared test helpers for GH-79 dispatch authorization tests.

These helpers load and parse the reusable-dispatch.yml workflow YAML
so individual test files can assert on routing logic structure.
*/

// loadDispatchWorkflow returns the per-org and per-repo dispatch workflow
// content. Both must contain identical routing logic.
func loadDispatchWorkflow(t *testing.T) (perOrg, perRepo string) {
	t.Helper()

	orgContent, err := scaffold.FullsendRepoFile(".github/workflows/dispatch.yml")
	require.NoError(t, err, "reading per-org dispatch.yml from scaffold")
	require.NotEmpty(t, orgContent, "per-org dispatch.yml should not be empty")

	repoContent, err := os.ReadFile("../../../.github/workflows/reusable-dispatch.yml")
	require.NoError(t, err, "reading per-repo reusable-dispatch.yml")
	require.NotEmpty(t, repoContent, "per-repo reusable-dispatch.yml should not be empty")

	return string(orgContent), string(repoContent)
}

// dispatchWorkflows is a helper type for iterating both workflow files.
type dispatchWorkflow struct {
	Name    string
	Content string
}

// bothWorkflows returns the per-org and per-repo workflows for table-driven tests.
func bothWorkflows(t *testing.T) []dispatchWorkflow {
	t.Helper()
	perOrg, perRepo := loadDispatchWorkflow(t)
	return []dispatchWorkflow{
		{Name: "per-org", Content: perOrg},
		{Name: "per-repo", Content: perRepo},
	}
}

// extractRouteBlock extracts the shell script from the "Determine stage" step.
func extractRouteBlock(workflow string) string {
	idx := strings.Index(workflow, "Determine stage")
	if idx == -1 {
		return ""
	}
	rest := workflow[idx:]

	runIdx := strings.Index(rest, "run: |")
	if runIdx == -1 {
		return ""
	}
	script := rest[runIdx:]

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
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- name:") {
			break
		}
		block = append(block, line)
	}
	return strings.Join(block, "\n")
}

// extractIssueCommentBlock extracts the issue_comment) case from the routing script.
func extractIssueCommentBlock(workflow string) string {
	route := extractRouteBlock(workflow)
	if route == "" {
		return ""
	}
	idx := strings.Index(route, "issue_comment)")
	if idx == -1 {
		return ""
	}
	section := route[idx:]
	// End at next top-level case (issues), pull_request_target), etc.)
	for _, marker := range []string{"\n            issues)", "\n            pull_request_target)"} {
		end := strings.Index(section, marker)
		if end != -1 {
			section = section[:end]
		}
	}
	return section
}

// extractIssuesBlock extracts the issues) case from the routing script.
func extractIssuesBlock(workflow string) string {
	route := extractRouteBlock(workflow)
	if route == "" {
		return ""
	}
	// Match "issues)" that is NOT "issue_comment)" — find standalone "issues)" case
	lines := strings.Split(route, "\n")
	startIdx := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "issues)" && !strings.Contains(lines[maxInt(0, i-1)], "issue_comment") {
			startIdx = i
			break
		}
	}
	if startIdx == -1 {
		return ""
	}
	// Collect until next top-level case
	var block []string
	for i := startIdx; i < len(lines); i++ {
		if i > startIdx {
			trimmed := strings.TrimSpace(lines[i])
			if trimmed == "pull_request_target)" || trimmed == "pull_request_review)" || trimmed == "esac" {
				break
			}
		}
		block = append(block, lines[i])
	}
	return strings.Join(block, "\n")
}

// extractPRTargetBlock extracts the pull_request_target) case from the routing script.
func extractPRTargetBlock(workflow string) string {
	route := extractRouteBlock(workflow)
	if route == "" {
		return ""
	}
	idx := strings.Index(route, "pull_request_target)")
	if idx == -1 {
		return ""
	}
	section := route[idx:]
	// End at next top-level case or esac
	for _, marker := range []string{"\n            pull_request_review)", "\n          esac"} {
		end := strings.Index(section, marker)
		if end != -1 {
			section = section[:end]
		}
	}
	return section
}

// extractCommandSection extracts the section for a specific slash command from the route block.
func extractCommandSection(route, command string) string {
	idx := strings.Index(route, command)
	if idx == -1 {
		return ""
	}
	section := route[idx:]
	// Find the next ;; that ends this case
	endIdx := strings.Index(section, ";;")
	if endIdx != -1 {
		section = section[:endIdx+2]
	}
	return section
}

// extractIsAuthorizedFunction extracts the is_authorized() function definition.
// It finds the function and captures up through "esac" + closing "}" to handle
// nested ${VAR} braces in the case statement.
func extractIsAuthorizedFunction(workflow string) string {
	route := extractRouteBlock(workflow)
	// Find the first is_authorized() that is a function definition (not a call)
	idx := strings.Index(route, "is_authorized() {")
	if idx == -1 {
		return ""
	}
	section := route[idx:]
	// Find "esac" which ends the case statement, then the closing "}"
	esacIdx := strings.Index(section, "esac")
	if esacIdx == -1 {
		return ""
	}
	endIdx := strings.Index(section[esacIdx:], "}")
	if endIdx == -1 {
		return ""
	}
	return section[:esacIdx+endIdx+1]
}

// extractIsEventActorAuthorizedFunction extracts the is_event_actor_authorized() definition.
func extractIsEventActorAuthorizedFunction(workflow string) string {
	route := extractRouteBlock(workflow)
	idx := strings.Index(route, "is_event_actor_authorized() {")
	if idx == -1 {
		return ""
	}
	section := route[idx:]
	esacIdx := strings.Index(section, "esac")
	if esacIdx == -1 {
		return ""
	}
	endIdx := strings.Index(section[esacIdx:], "}")
	if endIdx == -1 {
		return ""
	}
	return section[:esacIdx+endIdx+1]
}

// maxInt returns the larger of two ints (avoids shadowing built-in max).
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
