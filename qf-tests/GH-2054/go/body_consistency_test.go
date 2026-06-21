package cli

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Body-Verdict Consistency Check Tests

STP Reference: outputs/stp/GH-2054/GH-2054_test_plan.md
STD Reference: outputs/std/GH-2054/GH-2054_test_description.yaml
Jira: GH-2054

Tests for ensureBodyFindingsConsistency() which detects contradictions
between the review body text and structured findings, and replaces the
body when a blocking verdict has critical/high findings that the body
does not reference.
*/

func TestEnsureBodyFindingsConsistency_Generated(t *testing.T) {
	// =====================================================================
	// Group 1: Body replaced when verdict contradicts summary (P0)
	// =====================================================================

	t.Run("replaces contradictory body when verdict is request-changes with critical findings", func(t *testing.T) {
		// [test_id:TS-GH-2054-001]
		result := &ReviewResult{
			Action: "request-changes",
			Body:   "## Review\n### Findings\nNo findings.",
			Findings: []ReviewFinding{
				{
					Severity:    "critical",
					Category:    "logic-error",
					File:        "pipeline.yaml",
					Line:        42,
					Description: "CEL expression uses wrong operator.",
					Remediation: "Use && instead of ||.",
				},
			},
		}

		replaced := ensureBodyFindingsConsistency(result)

		assert.True(t, replaced, "should return true when body contradicts verdict with critical findings")
		assert.Contains(t, result.Body, "CEL expression uses wrong operator.", "body should contain the critical finding description")
		assert.NotContains(t, result.Body, "No findings", "original contradictory text should be replaced")
	})

	t.Run("synthesized body contains all critical and high findings", func(t *testing.T) {
		// [test_id:TS-GH-2054-002]
		result := &ReviewResult{
			Action: "request-changes",
			Body:   "## Review\n### Findings\nNo findings.",
			Findings: []ReviewFinding{
				{Severity: "critical", Category: "logic-error", File: "a.go", Line: 10, Description: "Critical bug one."},
				{Severity: "critical", Category: "security", File: "b.go", Line: 20, Description: "Critical bug two."},
				{Severity: "high", Category: "missing-test", File: "c.go", Line: 30, Description: "High severity one."},
				{Severity: "high", Category: "auth-bypass", File: "d.go", Line: 40, Description: "High severity two."},
				{Severity: "low", Category: "style", File: "e.go", Line: 50, Description: "Low nitpick."},
			},
		}

		replaced := ensureBodyFindingsConsistency(result)
		require.True(t, replaced)

		// Every critical finding description must appear
		assert.Contains(t, result.Body, "Critical bug one.")
		assert.Contains(t, result.Body, "Critical bug two.")
		// Every high finding description must appear
		assert.Contains(t, result.Body, "High severity one.")
		assert.Contains(t, result.Body, "High severity two.")
	})

	t.Run("result.Body mutated in place after replacement", func(t *testing.T) {
		// [test_id:TS-GH-2054-003]
		result := &ReviewResult{
			Action: "request-changes",
			Body:   "## Review\n### Findings\nNo findings.",
			Findings: []ReviewFinding{
				{Severity: "critical", Category: "logic-error", Description: "Major bug."},
			},
		}
		originalBody := result.Body

		replaced := ensureBodyFindingsConsistency(result)

		assert.True(t, replaced, "should return true indicating replacement")
		assert.NotEqual(t, originalBody, result.Body, "result.Body should be mutated in place")
		assert.NotEmpty(t, result.Body, "mutated body should not be empty")
		assert.Contains(t, result.Body, "Major bug.", "mutated body should contain synthesized finding content")
	})

	t.Run("no replacement when findings array is empty", func(t *testing.T) {
		// [test_id:TS-GH-2054-004]
		originalBody := "## Review\n### Findings\nNo findings."
		result := &ReviewResult{
			Action:   "request-changes",
			Body:     originalBody,
			Findings: []ReviewFinding{},
		}

		replaced := ensureBodyFindingsConsistency(result)

		assert.False(t, replaced, "should not replace when findings array is empty")
		assert.Equal(t, originalBody, result.Body, "body should be preserved unchanged")
	})

	// =====================================================================
	// Group 3: No-op when body already references findings (P1)
	// =====================================================================

	t.Run("no replacement when category already present in body", func(t *testing.T) {
		// [test_id:TS-GH-2054-009]
		originalBody := "## Review\n### Findings\n#### Critical\n- **[logic-error]** Bad CEL expression."
		result := &ReviewResult{
			Action: "request-changes",
			Body:   originalBody,
			Findings: []ReviewFinding{
				{Severity: "critical", Category: "logic-error", Description: "Bad CEL expression."},
			},
		}

		replaced := ensureBodyFindingsConsistency(result)

		assert.False(t, replaced, "body already references the finding category, should not be patched")
		assert.Equal(t, originalBody, result.Body, "body should be preserved")
	})

	t.Run("case-insensitive category matching", func(t *testing.T) {
		// [test_id:TS-GH-2054-010]
		originalBody := "## Review\n#### Critical\n- **[Logic-Error]** Bad expression."
		result := &ReviewResult{
			Action: "request-changes",
			Body:   originalBody,
			Findings: []ReviewFinding{
				{Severity: "critical", Category: "logic-error", Description: "Bad expression."},
			},
		}

		replaced := ensureBodyFindingsConsistency(result)

		assert.False(t, replaced, "case-insensitive category match should detect the reference")
		assert.Equal(t, originalBody, result.Body, "body should be preserved when case-insensitive match succeeds")
	})

	t.Run("partial category match behavior — substring matching", func(t *testing.T) {
		// [test_id:TS-GH-2054-011]
		// The implementation uses strings.Contains for matching, so a body
		// mentioning "error" WILL match "logic-error" via substring. This
		// test documents the actual implementation behavior.
		result := &ReviewResult{
			Action: "request-changes",
			Body:   "## Review\n### Findings\nSome generic error discussion.",
			Findings: []ReviewFinding{
				{Severity: "critical", Category: "logic-error", Description: "Specific logic issue."},
			},
		}

		replaced := ensureBodyFindingsConsistency(result)

		// The implementation uses substring matching (strings.Contains),
		// so "logic-error" is found within the body via substring match.
		// "error" in the body doesn't match, but "logic-error" is not in
		// the body either in this case. The body says "error" but the
		// category is "logic-error" — body doesn't contain "logic-error".
		assert.True(t, replaced, "body does not contain the full category 'logic-error', so replacement triggers")
	})

	// =====================================================================
	// Group 4: Non-blocking verdicts do not trigger check (P1)
	// =====================================================================

	t.Run("no replacement for approve action", func(t *testing.T) {
		// [test_id:TS-GH-2054-012]
		originalBody := "## Review\n### Findings\nNo findings."
		result := &ReviewResult{
			Action: "approve",
			Body:   originalBody,
			Findings: []ReviewFinding{
				{Severity: "critical", Category: "security", Description: "Auth bypass."},
			},
		}

		replaced := ensureBodyFindingsConsistency(result)

		assert.False(t, replaced, "approve action should never trigger body replacement")
		assert.Equal(t, originalBody, result.Body, "body should not be modified for approve action")
	})

	t.Run("no replacement for comment action", func(t *testing.T) {
		// [test_id:TS-GH-2054-013]
		originalBody := "## Review\n### Findings\nNo findings."
		result := &ReviewResult{
			Action: "comment",
			Body:   originalBody,
			Findings: []ReviewFinding{
				{Severity: "high", Category: "security", Description: "Auth bypass."},
			},
		}

		replaced := ensureBodyFindingsConsistency(result)

		assert.False(t, replaced, "comment action should never trigger body replacement")
		assert.Equal(t, originalBody, result.Body, "body should not be modified for comment action")
	})

	// =====================================================================
	// Group 5: Low/medium-only findings do not trigger check (P1)
	// =====================================================================

	t.Run("no replacement with only low-severity findings", func(t *testing.T) {
		// [test_id:TS-GH-2054-014]
		originalBody := "## Review\n### Findings\nNo findings."
		result := &ReviewResult{
			Action: "request-changes",
			Body:   originalBody,
			Findings: []ReviewFinding{
				{Severity: "low", Category: "style", Description: "Nitpick one."},
				{Severity: "low", Category: "docs", Description: "Nitpick two."},
			},
		}

		replaced := ensureBodyFindingsConsistency(result)

		assert.False(t, replaced, "low-severity-only findings should not trigger replacement")
		assert.Equal(t, originalBody, result.Body, "body should not be modified for low-severity findings")
	})

	t.Run("no replacement with mixed low and medium findings", func(t *testing.T) {
		// [test_id:TS-GH-2054-015]
		originalBody := "## Review\n### Findings\nNo findings."
		result := &ReviewResult{
			Action: "request-changes",
			Body:   originalBody,
			Findings: []ReviewFinding{
				{Severity: "low", Category: "style", Description: "Nitpick."},
				{Severity: "medium", Category: "docs", Description: "Missing docs."},
			},
		}

		replaced := ensureBodyFindingsConsistency(result)

		assert.False(t, replaced, "mixed low/medium findings should not trigger replacement")
		assert.Equal(t, originalBody, result.Body, "body should not be modified")
	})

	// =====================================================================
	// Group 7: Reject action alias (P1)
	// =====================================================================

	t.Run("reject action triggers body replacement", func(t *testing.T) {
		// [test_id:TS-GH-2054-019]
		result := &ReviewResult{
			Action: "reject",
			Body:   "## Review\n### Findings\nNo findings.",
			Findings: []ReviewFinding{
				{Severity: "critical", Category: "auth-bypass", File: "auth.go", Line: 99, Description: "Auth bypass vulnerability."},
			},
		}

		replaced := ensureBodyFindingsConsistency(result)

		assert.True(t, replaced, "reject maps to REQUEST_CHANGES, should trigger replacement")
		assert.Contains(t, result.Body, "auth-bypass", "replacement body should contain the finding category")
	})

	t.Run("reject body contains synthesized findings", func(t *testing.T) {
		// [test_id:TS-GH-2054-020]
		result := &ReviewResult{
			Action: "reject",
			Body:   "## Review\n### Findings\nNo findings.",
			Findings: []ReviewFinding{
				{Severity: "critical", Category: "logic-error", File: "main.go", Line: 10, Description: "Critical logic flaw."},
				{Severity: "high", Category: "missing-test", File: "svc.go", Line: 20, Description: "Missing test coverage."},
				{Severity: "low", Category: "style", Description: "Style nitpick."},
			},
		}

		replaced := ensureBodyFindingsConsistency(result)
		require.True(t, replaced)

		// All critical and high findings must be present
		assert.Contains(t, result.Body, "Critical logic flaw.")
		assert.Contains(t, result.Body, "Missing test coverage.")
		// Low findings are also included (synthesizeReviewBody includes ALL findings)
		assert.Contains(t, result.Body, "Style nitpick.")

		// Verify proper severity section formatting
		assert.Contains(t, result.Body, "#### Critical")
		assert.Contains(t, result.Body, "#### High")

		// Verify severity ordering (critical before high)
		critIdx := strings.Index(result.Body, "#### Critical")
		highIdx := strings.Index(result.Body, "#### High")
		assert.Greater(t, highIdx, critIdx, "Critical should appear before High")
	})

	// =====================================================================
	// Group 8: Edge cases — nil/empty inputs (P2)
	// =====================================================================

	t.Run("nil result returns false without panic", func(t *testing.T) {
		// [test_id:TS-GH-2054-021]
		assert.NotPanics(t, func() {
			replaced := ensureBodyFindingsConsistency(nil)
			assert.False(t, replaced, "nil input should return false")
		})
	})

	t.Run("unknown action value returns false", func(t *testing.T) {
		// [test_id:TS-GH-2054-022]
		originalBody := "## Review\n### Findings\nNo findings."
		result := &ReviewResult{
			Action: "unknown",
			Body:   originalBody,
			Findings: []ReviewFinding{
				{Severity: "critical", Category: "logic-error", Description: "Critical bug."},
			},
		}

		replaced := ensureBodyFindingsConsistency(result)

		assert.False(t, replaced, "unknown action should not trigger replacement")
		assert.Equal(t, originalBody, result.Body, "body should not be modified for unknown action")
	})
}
