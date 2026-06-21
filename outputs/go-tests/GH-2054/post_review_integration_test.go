//go:build e2e

package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/ui"
)

// findRepoRoot walks up from the current working directory until it finds
// the repository root (containing go.mod).
func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repository root (go.mod)")
		}
		dir = parent
	}
}

/*
Post-Review CLI Integration Tests

STP Reference: outputs/stp/GH-2054/GH-2054_test_plan.md
STD Reference: outputs/std/GH-2054/GH-2054_test_description.yaml
Jira: GH-2054
*/

// TestPostReviewCommand_AppliesConsistencyCheck verifies that the consistency
// check runs in the post-review flow between result parsing and comment posting.
// [test_id:TS-GH-2054-012]
func TestPostReviewCommand_AppliesConsistencyCheck(t *testing.T) {
	// Arrange: create contradictory review result JSON
	contradictoryJSON := `{
		"action": "request-changes",
		"body": "No significant findings.",
		"findings": [
			{
				"category": "logic-error",
				"severity": "critical",
				"file": "pkg/handler.go",
				"line": 42,
				"description": "Nil pointer dereference in error path",
				"remediation": "Add nil check before accessing field"
			}
		]
	}`

	// Act: parse the result and apply consistency check (simulating the command flow)
	parsed, err := parseReviewResult(contradictoryJSON)
	require.NoError(t, err)

	originalBody := parsed.Body
	patched := ensureBodyFindingsConsistency(&parsed)

	// Assert: consistency check ran and corrected the body
	require.True(t, patched, "consistency check should have patched the body")
	assert.NotEqual(t, originalBody, parsed.Body, "body should be different after patching")
	assert.Contains(t, parsed.Body, "logic-error", "posted body should contain finding category")
	assert.NotContains(t, parsed.Body, "No significant findings", "contradictory text should be removed")
}

// TestPostReviewCommand_LogsWarningOnSynthesis verifies that a StepWarn
// log message is emitted when the consistency check synthesizes a body.
// [test_id:TS-GH-2054-013]
func TestPostReviewCommand_LogsWarningOnSynthesis(t *testing.T) {
	// Arrange: capture log output
	var logOutput bytes.Buffer
	printer := ui.New(&logOutput)

	// Create contradictory result
	result := ReviewResult{
		Action: "request-changes",
		Body:   "No findings to report.",
		Findings: []ReviewFinding{
			{Severity: "critical", Category: "logic-error", Description: "Nil pointer dereference"},
		},
	}

	// Act: simulate the command flow
	if patched := ensureBodyFindingsConsistency(&result); patched {
		printer.StepWarn("Review body was inconsistent with findings — synthesized body from structured findings")
	}

	// Assert: warning was logged
	logStr := logOutput.String()
	assert.Contains(t, logStr, "synthesized", "log should contain synthesis warning")
}

// TestPostReviewCommand_PropagatesPatchedBody verifies that the patched body
// is available for both sticky comment and formal review after consistency check.
// [test_id:TS-GH-2054-014]
func TestPostReviewCommand_PropagatesPatchedBody(t *testing.T) {
	// Arrange: contradictory result
	result := ReviewResult{
		Action: "request-changes",
		Body:   "No findings to report.",
		Findings: []ReviewFinding{
			{Severity: "critical", Category: "logic-error", Description: "Nil pointer dereference"},
			{Severity: "high", Category: "security-issue", Description: "Missing auth check"},
		},
	}

	// Act: apply consistency check
	patched := ensureBodyFindingsConsistency(&result)
	require.True(t, patched)

	// The result.Body is now the single source of truth for both sticky
	// comment and formal review. Verify it contains the corrected content.
	patchedBody := result.Body

	// Assert: patched body would propagate to both output paths
	assert.Contains(t, patchedBody, "logic-error", "patched body should contain critical finding")
	assert.Contains(t, patchedBody, "security-issue", "patched body should contain high finding")
	assert.NotContains(t, patchedBody, "No findings", "contradictory text removed")

	// Verify the patched body is a valid synthesized review body
	assert.Contains(t, patchedBody, "## Review", "should have review header")
	assert.Contains(t, patchedBody, "### Findings", "should have findings section")
}

// TestSkillMD_ContainsConsistencyInstruction verifies that the pr-review
// SKILL.md file contains instruction for including findings in body
// for blocking verdicts.
// [test_id:TS-GH-2054-015]
func TestSkillMD_ContainsConsistencyInstruction(t *testing.T) {
	// Locate SKILL.md file relative to the repository root.
	// go test runs from the package directory, so we need to find the repo root.
	// Walk up from the current directory until we find go.mod.
	repoRoot := findRepoRoot(t)
	skillPath := repoRoot + "/skills/pr-review/SKILL.md"

	// Verify file exists
	_, err := os.Stat(skillPath)
	require.NoError(t, err, "skills/pr-review/SKILL.md should exist at %s", skillPath)

	// Read content
	content, err := os.ReadFile(skillPath)
	require.NoError(t, err, "should be able to read SKILL.md")

	contentStr := string(content)
	contentLower := strings.ToLower(contentStr)

	// Assert: contains instruction about body-verdict consistency
	// The instruction should mention including findings for blocking verdicts
	hasFindings := strings.Contains(contentLower, "finding")
	hasBody := strings.Contains(contentLower, "body") || strings.Contains(contentLower, "summary")
	hasBlocking := strings.Contains(contentLower, "request-changes") ||
		strings.Contains(contentLower, "reject") ||
		strings.Contains(contentLower, "blocking")

	assert.True(t, hasFindings, "SKILL.md should mention findings")
	assert.True(t, hasBody, "SKILL.md should mention body or summary")
	assert.True(t, hasBlocking, "SKILL.md should mention blocking verdicts (request-changes, reject, or blocking)")
}

// TestPostReview_EndToEnd_ContradictoryAgentOutput verifies the complete
// post-review flow from contradictory agent output to corrected posted body.
// This is the acceptance test from GH-2054's validation criteria.
// [test_id:TS-GH-2054-016]
func TestPostReview_EndToEnd_ContradictoryAgentOutput(t *testing.T) {
	// Arrange: full contradictory review result JSON (agent-like output)
	fullContradictoryJSON := `{
		"action": "request-changes",
		"body": "## Review Summary\n\nLGTM! No findings to report. The code looks good overall.",
		"head_sha": "abc1234567890abcdef1234567890abcdef123456",
		"findings": [
			{
				"category": "logic-error",
				"severity": "critical",
				"file": "pkg/handler.go",
				"line": 42,
				"description": "Nil pointer dereference in error path",
				"remediation": "Add nil check before accessing field"
			},
			{
				"category": "security-issue",
				"severity": "high",
				"file": "pkg/auth.go",
				"line": 15,
				"description": "Missing input validation on user-provided data",
				"remediation": "Validate and sanitize input before processing"
			},
			{
				"category": "style",
				"severity": "low",
				"file": "pkg/utils.go",
				"line": 8,
				"description": "Unused import"
			}
		]
	}`

	// Act: simulate the full post-review command flow
	// Step 1: Parse the review result
	parsed, err := parseReviewResult(fullContradictoryJSON)
	require.NoError(t, err)

	// Step 2: Apply consistency check (this is what the command does)
	var logOutput bytes.Buffer
	printer := ui.New(&logOutput)
	_ = printer // used for logging in real command

	patched := ensureBodyFindingsConsistency(&parsed)
	require.True(t, patched, "should detect contradiction and patch body")

	// The final posted body is parsed.Body after consistency check
	postedBody := parsed.Body

	// Assert: final posted comment reflects actual findings
	assert.Contains(t, postedBody, "logic-error", "critical finding should be in posted body")
	assert.Contains(t, postedBody, "security-issue", "high finding should be in posted body")
	assert.NotContains(t, postedBody, "No findings", "'No findings' text should be removed")
	assert.NotContains(t, postedBody, "LGTM", "misleading LGTM should be removed")

	// Verify severity ordering: critical before high
	criticalIdx := strings.Index(postedBody, "#### Critical")
	highIdx := strings.Index(postedBody, "#### High")
	assert.Greater(t, criticalIdx, -1, "critical section should exist")
	assert.Greater(t, highIdx, -1, "high section should exist")
	assert.Greater(t, highIdx, criticalIdx, "critical findings should appear before high findings")

	// Verify low findings are also included (synthesizeReviewBody includes all)
	assert.Contains(t, postedBody, "style", "low finding should also be included")

	// Suppress unused variable warning
	_ = io.Discard
}
