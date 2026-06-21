package cli

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Synthesize Review Body Tests

STP Reference: outputs/stp/GH-2054/GH-2054_test_plan.md
STD Reference: outputs/std/GH-2054/GH-2054_test_description.yaml
Jira: GH-2054

Tests for synthesizeReviewBody() which generates a markdown body from
structured findings, grouped by severity in descending order with proper
formatting for file locations, categories, and remediation text.
*/

func TestSynthesizeReviewBody_Generated(t *testing.T) {
	// =====================================================================
	// Group 2: Severity ordering and section rendering (P0)
	// =====================================================================

	t.Run("severity sections ordered critical to info", func(t *testing.T) {
		// [test_id:TS-GH-2054-005]
		findings := []ReviewFinding{
			{Severity: "info", Category: "docs", Description: "Info finding."},
			{Severity: "low", Category: "style", Description: "Low finding."},
			{Severity: "critical", Category: "logic-error", Description: "Critical finding."},
			{Severity: "medium", Category: "complexity", Description: "Medium finding."},
			{Severity: "high", Category: "missing-test", Description: "High finding."},
		}

		body := synthesizeReviewBody(findings)
		require.NotEmpty(t, body)

		critIdx := strings.Index(body, "#### Critical")
		highIdx := strings.Index(body, "#### High")
		medIdx := strings.Index(body, "#### Medium")
		lowIdx := strings.Index(body, "#### Low")
		infoIdx := strings.Index(body, "#### Info")

		assert.Greater(t, critIdx, -1, "Critical section should be present")
		assert.Greater(t, highIdx, -1, "High section should be present")
		assert.Greater(t, medIdx, -1, "Medium section should be present")
		assert.Greater(t, lowIdx, -1, "Low section should be present")
		assert.Greater(t, infoIdx, -1, "Info section should be present")

		assert.Greater(t, highIdx, critIdx, "Critical should appear before High")
		assert.Greater(t, medIdx, highIdx, "High should appear before Medium")
		assert.Greater(t, lowIdx, medIdx, "Medium should appear before Low")
		assert.Greater(t, infoIdx, lowIdx, "Low should appear before Info")
	})

	t.Run("only populated severity sections rendered", func(t *testing.T) {
		// [test_id:TS-GH-2054-006]
		findings := []ReviewFinding{
			{Severity: "critical", Category: "logic-error", Description: "Critical bug."},
			{Severity: "medium", Category: "complexity", Description: "Medium issue."},
		}

		body := synthesizeReviewBody(findings)
		require.NotEmpty(t, body)

		// Populated sections should be present
		assert.Contains(t, body, "#### Critical", "critical section should be rendered")
		assert.Contains(t, body, "#### Medium", "medium section should be rendered")

		// Unpopulated sections should be absent
		assert.NotContains(t, body, "#### High", "high section should not be rendered")
		assert.NotContains(t, body, "#### Low", "low section should not be rendered")
		assert.NotContains(t, body, "#### Info", "info section should not be rendered")
	})

	t.Run("remediation text included when present", func(t *testing.T) {
		// [test_id:TS-GH-2054-007]
		findings := []ReviewFinding{
			{
				Severity:    "critical",
				Category:    "logic-error",
				Description: "Off by one.",
				Remediation: "Use <= instead of <.",
			},
			{
				Severity:    "high",
				Category:    "missing-test",
				Description: "No test coverage.",
				// No remediation
			},
		}

		body := synthesizeReviewBody(findings)
		require.NotEmpty(t, body)

		assert.Contains(t, body, "Remediation: Use <= instead of <.", "remediation text should be included for findings that have it")
		assert.Contains(t, body, "No test coverage.", "finding without remediation should still render its description")
	})

	t.Run("body format matches pr-review skill template", func(t *testing.T) {
		// [test_id:TS-GH-2054-008]
		findings := []ReviewFinding{
			{
				Severity:    "critical",
				Category:    "logic-error",
				File:        "internal/cli/postreview.go",
				Line:        42,
				Description: "Nil pointer dereference.",
			},
			{
				Severity:    "high",
				Category:    "missing-test",
				File:        "internal/service.go",
				Line:        10,
				Description: "Missing test coverage.",
				Remediation: "Add a unit test.",
			},
		}

		body := synthesizeReviewBody(findings)

		// Verify top-level structure
		assert.Contains(t, body, "## Review", "body should start with ## Review")
		assert.Contains(t, body, "### Findings", "body should contain ### Findings heading")

		// Verify severity section headings use ####
		assert.Contains(t, body, "#### Critical", "severity headings should use #### format")
		assert.Contains(t, body, "#### High", "severity headings should use #### format")

		// Verify findings are bullet points with category in bold brackets
		assert.Contains(t, body, "- **[logic-error]**", "finding should be bullet with bold-bracketed category")
		assert.Contains(t, body, "- **[missing-test]**", "finding should be bullet with bold-bracketed category")

		// Verify description follows the dash separator
		assert.Contains(t, body, " — Nil pointer dereference.", "description should follow em dash")
	})

	// =====================================================================
	// Group 6: File location rendering (P1)
	// =====================================================================

	t.Run("file and line rendered in backtick block", func(t *testing.T) {
		// [test_id:TS-GH-2054-016]
		findings := []ReviewFinding{
			{
				Severity:    "critical",
				Category:    "logic-error",
				File:        "internal/cli/postreview.go",
				Line:        42,
				Description: "Bug found.",
			},
		}

		body := synthesizeReviewBody(findings)
		require.NotEmpty(t, body)

		assert.Contains(t, body, "`internal/cli/postreview.go:42`", "file and line should be rendered in backtick format")
	})

	t.Run("findings without file omit location block", func(t *testing.T) {
		// [test_id:TS-GH-2054-017]
		findings := []ReviewFinding{
			{
				Severity:    "critical",
				Category:    "architecture",
				File:        "",
				Line:        0,
				Description: "Major design flaw.",
			},
		}

		body := synthesizeReviewBody(findings)
		require.NotEmpty(t, body)

		// Description should be present
		assert.Contains(t, body, "Major design flaw.", "finding description should be rendered")
		// No backtick file location should be present
		assert.NotContains(t, body, "` —", "no backtick file reference should appear for findings without file")
		assert.NotContains(t, body, "``", "no empty backtick block")
	})

	t.Run("file without line number renders correctly", func(t *testing.T) {
		// [test_id:TS-GH-2054-018]
		findings := []ReviewFinding{
			{
				Severity:    "critical",
				Category:    "complexity",
				File:        "internal/cli/postreview.go",
				Line:        0,
				Description: "File too complex.",
			},
		}

		body := synthesizeReviewBody(findings)
		require.NotEmpty(t, body)

		// File path should be present in backticks
		assert.Contains(t, body, "`internal/cli/postreview.go`", "file path should be rendered in backticks")
		// No ":0" artifact
		assert.NotContains(t, body, ":0", "no ':0' artifact should appear for file without line number")
	})
}
