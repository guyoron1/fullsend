package cli

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/ui"
)

/*
Clean Content Passthrough Tests

STP Reference: outputs/stp/GH-1230/GH-1230_test_plan.md
Jira: GH-1230
Group 5: Clean review content passes through (P1)
*/

func TestCleanContentPassthrough(t *testing.T) {
	printer := ui.New(io.Discard)

	t.Run("[test_id:TS-GH1230-012] should not modify clean body with markdown formatting", func(t *testing.T) {
		// Arrange: ReviewResult with rich markdown body (code blocks, links, formatting)
		originalBody := "## Review Summary\n\n" +
			"The implementation looks solid. A few observations:\n\n" +
			"```go\nfunc handleError(err error) {\n    log.Fatal(err)\n}\n```\n\n" +
			"- Consider using `errors.Wrap` for better stack traces\n" +
			"- See [Go error handling](https://blog.golang.org/error-handling) for patterns\n" +
			"- **Important**: The `defer` on line 42 should close the file handle\n\n" +
			"Overall: 👍 LGTM"
		input := ReviewResult{
			Body:     originalBody,
			Action:   "comment",
			Findings: []ReviewFinding{},
		}

		// Act
		result := sanitizeReviewResult(input, printer)

		// Assert: body is byte-for-byte identical
		assert.Equal(t, originalBody, result.Body,
			"Clean body with markdown formatting should pass through unchanged")
	})

	t.Run("[test_id:TS-GH1230-013] should not modify clean findings", func(t *testing.T) {
		// Arrange: ReviewResult with multiple clean findings
		input := ReviewResult{
			Body:   "Review complete with findings",
			Action: "request-changes",
			Findings: []ReviewFinding{
				{
					Description: "Consider using a constant for this magic number",
					Remediation: "Extract 42 to a named constant like maxRetries",
					Severity:    "low",
					Category:    "maintainability",
					File:        "handler.go",
					Line:        42,
				},
				{
					Description: "Missing error check on database query result",
					Remediation: "Add `if err != nil { return fmt.Errorf(\"query failed: %w\", err) }`",
					Severity:    "medium",
					Category:    "reliability",
					File:        "store.go",
					Line:        88,
				},
				{
					Description: "Function exceeds cyclomatic complexity threshold",
					Remediation: "Consider breaking processOrder into smaller helper functions",
					Severity:    "low",
					Category:    "maintainability",
					File:        "order.go",
					Line:        15,
				},
			},
		}

		// Act
		result := sanitizeReviewResult(input, printer)

		// Assert: all findings are identical to input
		require.Len(t, result.Findings, 3, "Should still have three findings")
		for i := range input.Findings {
			assert.Equal(t, input.Findings[i].Description, result.Findings[i].Description,
				"Finding %d description should be unchanged", i)
			assert.Equal(t, input.Findings[i].Remediation, result.Findings[i].Remediation,
				"Finding %d remediation should be unchanged", i)
			assert.Equal(t, input.Findings[i].Severity, result.Findings[i].Severity,
				"Finding %d severity should be unchanged", i)
			assert.Equal(t, input.Findings[i].File, result.Findings[i].File,
				"Finding %d file should be unchanged", i)
			assert.Equal(t, input.Findings[i].Line, result.Findings[i].Line,
				"Finding %d line should be unchanged", i)
		}
	})
}
