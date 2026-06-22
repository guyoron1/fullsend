package cli

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEnsureBodyFindingsConsistency_QF covers the 17 STD scenarios for
// the ensureBodyFindingsConsistency and synthesizeReviewBody functions.
// Source: outputs/std/GH-78/GH-78_test_description.yaml

func TestEnsureBodyFindingsConsistency_QF(t *testing.T) {

	// TS-GH-78-001: Contradictory body replaced for request-changes with critical findings.
	t.Run("TS-GH-78-001 contradictory body replaced for request-changes with critical findings", func(t *testing.T) {
		result := ReviewResult{
			Action: "request-changes",
			Body:   "No findings to report.",
			Findings: []ReviewFinding{
				{
					Category:    "logic-error",
					Severity:    "critical",
					File:        "cmd/run.go",
					Line:        42,
					Description: "Pointer dereference without nil guard",
					Remediation: "Add nil check before dereference",
				},
			},
		}

		patched := ensureBodyFindingsConsistency(&result)

		require.True(t, patched, "function must return true when body is replaced")
		assert.NotContains(t, result.Body, "No findings to report.", "original contradictory body must be overwritten")
		assert.Contains(t, result.Body, "logic-error", "synthesized body must contain the critical finding category")
		assert.Contains(t, result.Body, "Pointer dereference without nil guard", "synthesized body must contain the finding description")
	})

	// TS-GH-78-002: Severity sections ordered critical > high > medium > low > info.
	t.Run("TS-GH-78-002 severity sections ordered critical high medium low info", func(t *testing.T) {
		result := ReviewResult{
			Action: "request-changes",
			Body:   "No issues found.",
			Findings: []ReviewFinding{
				{Category: "perf-issue", Severity: "low", Description: "Slow loop"},
				{Category: "logic-error", Severity: "critical", Description: "Nil deref"},
				{Category: "style-issue", Severity: "info", Description: "Naming"},
				{Category: "auth-bypass", Severity: "high", Description: "Missing auth"},
				{Category: "data-race", Severity: "medium", Description: "Race condition"},
			},
		}

		patched := ensureBodyFindingsConsistency(&result)
		require.True(t, patched)

		body := result.Body
		critIdx := strings.Index(body, "Critical")
		highIdx := strings.Index(body, "High")
		medIdx := strings.Index(body, "Medium")
		lowIdx := strings.Index(body, "Low")
		infoIdx := strings.Index(body, "Info")

		require.NotEqual(t, -1, critIdx, "Critical section must be present")
		require.NotEqual(t, -1, highIdx, "High section must be present")
		require.NotEqual(t, -1, medIdx, "Medium section must be present")
		require.NotEqual(t, -1, lowIdx, "Low section must be present")
		require.NotEqual(t, -1, infoIdx, "Info section must be present")

		assert.Less(t, critIdx, highIdx, "Critical must appear before High")
		assert.Less(t, highIdx, medIdx, "High must appear before Medium")
		assert.Less(t, medIdx, lowIdx, "Medium must appear before Low")
		assert.Less(t, lowIdx, infoIdx, "Low must appear before Info")
	})

	// TS-GH-78-003: Synthesized body includes correct headings and bullet format.
	t.Run("TS-GH-78-003 synthesized body includes headings and bullet format", func(t *testing.T) {
		result := ReviewResult{
			Action: "request-changes",
			Body:   "LGTM",
			Findings: []ReviewFinding{
				{
					Category:    "logic-error",
					Severity:    "critical",
					File:        "pkg/handler.go",
					Line:        55,
					Description: "Dereference of potentially nil pointer",
				},
			},
		}

		patched := ensureBodyFindingsConsistency(&result)
		require.True(t, patched)

		assert.Contains(t, result.Body, "## Review", "body must contain Review heading")
		assert.Contains(t, result.Body, "### Findings", "body must contain Findings heading")
		assert.Contains(t, result.Body, "#### Critical", "body must contain severity sub-section")
		assert.Contains(t, result.Body, "- **[logic-error]**", "finding must be rendered as bullet with category")
		assert.Contains(t, result.Body, "Dereference of potentially nil pointer", "finding description must be present")
	})

	// TS-GH-78-004: Reject action triggers body replacement with critical findings.
	t.Run("TS-GH-78-004 reject action triggers body replacement", func(t *testing.T) {
		result := ReviewResult{
			Action: "reject",
			Body:   "Looks good overall.",
			Findings: []ReviewFinding{
				{
					Category:    "security-vuln",
					Severity:    "critical",
					Description: "Unsanitized input in query",
				},
			},
		}

		patched := ensureBodyFindingsConsistency(&result)
		require.True(t, patched, "reject maps to REQUEST_CHANGES and must trigger replacement")
		assert.Contains(t, result.Body, "security-vuln", "synthesized body must contain finding category")
		assert.Contains(t, result.Body, "Unsanitized input in query", "synthesized body must contain finding description")
	})

	// TS-GH-78-005: No-op when body contains finding category string.
	t.Run("TS-GH-78-005 no-op when body contains finding category", func(t *testing.T) {
		originalBody := "Found a logic-error in the handler that needs fixing."
		result := ReviewResult{
			Action: "request-changes",
			Body:   originalBody,
			Findings: []ReviewFinding{
				{
					Category:    "logic-error",
					Severity:    "critical",
					Description: "Handler does not check for nil",
				},
			},
		}

		patched := ensureBodyFindingsConsistency(&result)
		assert.False(t, patched, "body references finding category, should not be replaced")
		assert.Equal(t, originalBody, result.Body, "body must remain unchanged")
	})

	// TS-GH-78-006: Case-insensitive category matching prevents unnecessary replacement.
	t.Run("TS-GH-78-006 case-insensitive category matching", func(t *testing.T) {
		originalBody := "There is a Logic-Error in the code."
		result := ReviewResult{
			Action: "request-changes",
			Body:   originalBody,
			Findings: []ReviewFinding{
				{
					Category:    "logic-error",
					Severity:    "critical",
					Description: "Missing nil check",
				},
			},
		}

		patched := ensureBodyFindingsConsistency(&result)
		assert.False(t, patched, "case-insensitive match must detect the category reference")
	})

	// TS-GH-78-007: Approve action never triggers body replacement.
	t.Run("TS-GH-78-007 approve action never triggers replacement", func(t *testing.T) {
		originalBody := "No issues."
		result := ReviewResult{
			Action: "approve",
			Body:   originalBody,
			Findings: []ReviewFinding{
				{
					Category:    "logic-error",
					Severity:    "critical",
					Description: "Possible nil deref",
				},
			},
		}

		patched := ensureBodyFindingsConsistency(&result)
		assert.False(t, patched, "approve action must not trigger body replacement")
		assert.Equal(t, originalBody, result.Body, "body must remain unchanged")
	})

	// TS-GH-78-008: Comment action never triggers body replacement.
	t.Run("TS-GH-78-008 comment action never triggers replacement", func(t *testing.T) {
		originalBody := "Everything looks fine."
		result := ReviewResult{
			Action: "comment",
			Body:   originalBody,
			Findings: []ReviewFinding{
				{
					Category:    "perf-issue",
					Severity:    "high",
					Description: "N+1 query detected",
				},
			},
		}

		patched := ensureBodyFindingsConsistency(&result)
		assert.False(t, patched, "comment action must not trigger body replacement")
		assert.Equal(t, originalBody, result.Body, "body must remain unchanged")
	})

	// TS-GH-78-009: Low/medium-only findings do not trigger replacement.
	t.Run("TS-GH-78-009 low-medium-only findings do not trigger replacement", func(t *testing.T) {
		originalBody := "No significant issues."
		result := ReviewResult{
			Action: "request-changes",
			Body:   originalBody,
			Findings: []ReviewFinding{
				{Category: "style-issue", Severity: "low", Description: "Variable name does not follow convention"},
				{Category: "perf-issue", Severity: "medium", Description: "Could use buffer pool"},
			},
		}

		patched := ensureBodyFindingsConsistency(&result)
		assert.False(t, patched, "only low/medium findings must not trigger replacement")
		assert.Equal(t, originalBody, result.Body, "body must remain unchanged")
	})

	// TS-GH-78-010: File:line rendered in backtick block in synthesized body.
	t.Run("TS-GH-78-010 file-line rendered in backtick format", func(t *testing.T) {
		result := ReviewResult{
			Action: "request-changes",
			Body:   "LGTM",
			Findings: []ReviewFinding{
				{
					Category:    "logic-error",
					Severity:    "critical",
					File:        "pkg/processor.go",
					Line:        127,
					Description: "Loop bounds incorrect",
				},
			},
		}

		patched := ensureBodyFindingsConsistency(&result)
		require.True(t, patched)
		assert.Contains(t, result.Body, "`pkg/processor.go:127`", "file:line must be rendered in backtick format")
	})

	// TS-GH-78-011: Findings without file path render without backtick location.
	t.Run("TS-GH-78-011 findings without file render without location block", func(t *testing.T) {
		result := ReviewResult{
			Action: "request-changes",
			Body:   "All clear.",
			Findings: []ReviewFinding{
				{
					Category:    "architecture",
					Severity:    "high",
					Description: "No global error handler defined",
				},
			},
		}

		patched := ensureBodyFindingsConsistency(&result)
		require.True(t, patched)
		assert.Contains(t, result.Body, "architecture", "finding category must be present")
		assert.Contains(t, result.Body, "No global error handler defined", "finding description must be present")
		// No backtick-wrapped location should appear for findings without a file.
		assert.NotContains(t, result.Body, "``", "no empty backtick blocks should appear")
	})

	// TS-GH-78-012: Remediation text rendered for findings that have it.
	t.Run("TS-GH-78-012 remediation text rendered for findings", func(t *testing.T) {
		result := ReviewResult{
			Action: "request-changes",
			Body:   "Ship it.",
			Findings: []ReviewFinding{
				{
					Category:    "logic-error",
					Severity:    "critical",
					File:        "pkg/calc.go",
					Line:        33,
					Description: "Divisor not validated",
					Remediation: "Add a zero-check guard before the division",
				},
			},
		}

		patched := ensureBodyFindingsConsistency(&result)
		require.True(t, patched)
		assert.Contains(t, result.Body, "Add a zero-check guard before the division",
			"remediation text must be included in synthesized body")
	})

	// TS-GH-78-013: Unpopulated severity sections are absent from output.
	t.Run("TS-GH-78-013 unpopulated severity sections absent from output", func(t *testing.T) {
		result := ReviewResult{
			Action: "request-changes",
			Body:   "Nothing to see here.",
			Findings: []ReviewFinding{
				{Category: "logic-error", Severity: "critical", Description: "Nil deref"},
				{Category: "perf-issue", Severity: "low", Description: "Unnecessary alloc"},
			},
		}

		patched := ensureBodyFindingsConsistency(&result)
		require.True(t, patched)

		assert.Contains(t, result.Body, "#### Critical", "Critical section must be present")
		assert.Contains(t, result.Body, "#### Low", "Low section must be present")
		assert.NotContains(t, result.Body, "#### High", "High section must be absent (no high findings)")
		assert.NotContains(t, result.Body, "#### Medium", "Medium section must be absent (no medium findings)")
		assert.NotContains(t, result.Body, "#### Info", "Info section must be absent (no info findings)")
	})

	// TS-GH-78-014: Nil input returns false without panic.
	t.Run("TS-GH-78-014 nil input returns false without panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			patched := ensureBodyFindingsConsistency(nil)
			assert.False(t, patched, "nil input must return false")
		})
	})

	// TS-GH-78-015: Empty findings returns false.
	t.Run("TS-GH-78-015 empty findings returns false", func(t *testing.T) {
		originalBody := "No findings."
		result := ReviewResult{
			Action:   "request-changes",
			Body:     originalBody,
			Findings: []ReviewFinding{},
		}

		patched := ensureBodyFindingsConsistency(&result)
		assert.False(t, patched, "empty findings array must return false")
		assert.Equal(t, originalBody, result.Body, "body must remain unchanged")
	})

	// TS-GH-78-016: Unknown action returns false without modification.
	t.Run("TS-GH-78-016 unknown action returns false", func(t *testing.T) {
		originalBody := "No issues."
		result := ReviewResult{
			Action: "unknown-action",
			Body:   originalBody,
			Findings: []ReviewFinding{
				{Category: "logic-error", Severity: "critical", Description: "Serious issue"},
			},
		}

		patched := ensureBodyFindingsConsistency(&result)
		assert.False(t, patched, "unknown action must not trigger replacement")
		assert.Equal(t, originalBody, result.Body, "body must remain unchanged")
	})

	// TS-GH-78-017: File without line number renders cleanly (no ":0" artifact).
	t.Run("TS-GH-78-017 file without line number renders cleanly", func(t *testing.T) {
		result := ReviewResult{
			Action: "request-changes",
			Body:   "Clean code.",
			Findings: []ReviewFinding{
				{
					Category:    "logic-error",
					Severity:    "critical",
					File:        "pkg/handler.go",
					Line:        0,
					Description: "Function falls through",
				},
			},
		}

		patched := ensureBodyFindingsConsistency(&result)
		require.True(t, patched)
		assert.Contains(t, result.Body, "pkg/handler.go", "file path must be present in body")
		assert.NotContains(t, result.Body, ":0", "no ':0' artifact should appear for zero line number")
	})
}
