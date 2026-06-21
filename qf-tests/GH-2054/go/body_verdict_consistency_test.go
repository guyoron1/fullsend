//go:build e2e

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Body-Verdict Consistency Enforcement Tests

STP Reference: outputs/stp/GH-2054/GH-2054_test_plan.md
STD Reference: outputs/std/GH-2054/GH-2054_test_description.yaml
Jira: GH-2054
*/

// TestEnsureBodyFindingsConsistency_ReplacesContradictoryBody verifies that
// ensureBodyFindingsConsistency replaces the body when findings contradict
// the summary. This is the core bug fix for GH-2054.
// [test_id:TS-GH-2054-001]
func TestEnsureBodyFindingsConsistency_ReplacesContradictoryBody(t *testing.T) {
	// Arrange: ReviewResult with action=request-changes,
	// critical findings, but body says "No findings"
	result := ReviewResult{
		Action: "request-changes",
		Body:   "## Review Summary\n\nNo findings to report.",
		Findings: []ReviewFinding{
			{
				Severity:    "critical",
				Category:    "logic-error",
				File:        "pkg/handler.go",
				Line:        42,
				Description: "Nil pointer dereference in error path",
				Remediation: "Add nil check before accessing field",
			},
		},
	}

	// Act: call ensureBodyFindingsConsistency
	patched := ensureBodyFindingsConsistency(&result)

	// Assert: body is replaced with synthesized content
	require.True(t, patched, "body should be replaced when contradictory")
	assert.NotContains(t, result.Body, "No findings", "contradictory text should be removed")
	assert.Contains(t, result.Body, "logic-error", "replaced body should include the finding category")
	assert.Contains(t, result.Body, "Nil pointer dereference", "replaced body should include finding description")
	assert.Contains(t, result.Body, "pkg/handler.go:42", "replaced body should include file:line")
	assert.Contains(t, result.Body, "Remediation:", "replaced body should include remediation")
}

// TestEnsureBodyFindingsConsistency_NoOpForApproveComment verifies that
// approve and comment actions are not modified by the consistency check.
// [test_id:TS-GH-2054-002]
func TestEnsureBodyFindingsConsistency_NoOpForApproveComment(t *testing.T) {
	for _, action := range []string{"approve", "comment"} {
		t.Run(action, func(t *testing.T) {
			originalBody := "Some review notes for the author."
			result := ReviewResult{
				Action: action,
				Body:   originalBody,
				Findings: []ReviewFinding{
					{
						Severity:    "critical",
						Category:    "logic-error",
						Description: "Nil pointer dereference",
					},
				},
			}

			// Act
			patched := ensureBodyFindingsConsistency(&result)

			// Assert: body is unchanged for non-blocking verdicts
			assert.False(t, patched, "%s action should not trigger patching", action)
			assert.Equal(t, originalBody, result.Body, "body should be unchanged for %s action", action)
		})
	}
}

// TestEnsureBodyFindingsConsistency_RejectAction verifies that the reject
// action triggers consistency check since it maps to REQUEST_CHANGES.
// [test_id:TS-GH-2054-003]
func TestEnsureBodyFindingsConsistency_RejectActionTriggersCheck(t *testing.T) {
	// Arrange: ReviewResult with action=reject and critical findings
	result := ReviewResult{
		Action: "reject",
		Body:   "No significant issues.",
		Findings: []ReviewFinding{
			{
				Severity:    "critical",
				Category:    "security-issue",
				Description: "Hardcoded credentials detected",
			},
		},
	}

	// Act
	patched := ensureBodyFindingsConsistency(&result)

	// Assert: body is replaced (reject is a blocking verdict)
	assert.True(t, patched, "reject maps to REQUEST_CHANGES, should trigger patching")
	assert.Contains(t, result.Body, "security-issue", "body should contain the finding category")
}

// TestEnsureBodyFindingsConsistency_NilEmptyInput verifies defensive handling
// of nil and empty inputs without panicking.
// [test_id:TS-GH-2054-004]
func TestEnsureBodyFindingsConsistency_NilAndEmptyInput(t *testing.T) {
	criticalFinding := []ReviewFinding{
		{Severity: "critical", Category: "logic-error", Description: "Bug"},
	}

	tests := []struct {
		name   string
		result *ReviewResult
	}{
		{"nil result", nil},
		{"nil findings", &ReviewResult{Action: "request-changes", Body: "body", Findings: nil}},
		{"empty findings", &ReviewResult{Action: "request-changes", Body: "body", Findings: []ReviewFinding{}}},
		{"empty body with findings", &ReviewResult{Action: "request-changes", Body: "", Findings: criticalFinding}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act + Assert: no panic
			assert.NotPanics(t, func() {
				ensureBodyFindingsConsistency(tt.result)
			}, "should not panic for %s", tt.name)
		})
	}
}
