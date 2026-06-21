package cli

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/ui"
)

/*
Sanitize Finding Fields Tests

STP Reference: outputs/stp/GH-1230/GH-1230_test_plan.md
Jira: GH-1230
Group 3: Finding descriptions and remediations (P0)
Group 6: Mixed empty/non-empty finding fields (P1)
*/

func TestSanitizeFindingFields(t *testing.T) {
	printer := ui.New(io.Discard)

	t.Run("[test_id:TS-GH1230-006] should redact secret from finding description", func(t *testing.T) {
		// Arrange: ReviewResult with a GitHub PAT in finding description
		input := ReviewResult{
			Body:   "Review complete",
			Action: "comment",
			Findings: []ReviewFinding{
				{
					Description: "Hardcoded token ghp_ABCDEFghijklmnop1234567890abcdefghijklmn found",
					Remediation: "Use environment variables instead",
					Severity:    "high",
					Category:    "security",
					File:        "main.go",
				},
			},
		}

		// Act
		result := sanitizeReviewResult(input, printer)

		// Assert: secret redacted from description, remediation preserved
		assert.NotContains(t, result.Findings[0].Description, "ABCDEFghijklmnop1234567890abcdefghijklmn",
			"Full GitHub PAT payload should be redacted from finding description")
		assert.Contains(t, result.Findings[0].Description, "Hardcoded token",
			"Non-secret description content should be preserved")
		assert.Equal(t, "Use environment variables instead", result.Findings[0].Remediation,
			"Clean remediation should be unchanged")
	})

	t.Run("[test_id:TS-GH1230-007] should redact secret from finding remediation", func(t *testing.T) {
		// Arrange: ReviewResult with a GitHub PAT in finding remediation
		input := ReviewResult{
			Body:   "Review complete",
			Action: "comment",
			Findings: []ReviewFinding{
				{
					Description: "Hardcoded credentials detected",
					Remediation: "Replace ghp_ABCDEFghijklmnop1234567890abcdefghijklmn with env var",
					Severity:    "high",
					Category:    "security",
					File:        "config.go",
				},
			},
		}

		// Act
		result := sanitizeReviewResult(input, printer)

		// Assert: secret redacted from remediation, description preserved
		assert.NotContains(t, result.Findings[0].Remediation, "ABCDEFghijklmnop1234567890abcdefghijklmn",
			"Full GitHub PAT payload should be redacted from finding remediation")
		assert.Contains(t, result.Findings[0].Remediation, "with env var",
			"Non-secret remediation content should be preserved")
		assert.Equal(t, "Hardcoded credentials detected", result.Findings[0].Description,
			"Clean description should be unchanged")
	})

	t.Run("[test_id:TS-GH1230-008] should leave findings without secrets unchanged", func(t *testing.T) {
		// Arrange: ReviewResult with clean findings (no secrets)
		input := ReviewResult{
			Body:   "Review complete",
			Action: "approve",
			Findings: []ReviewFinding{
				{
					Description: "Consider using a constant for this magic number",
					Remediation: "Extract 42 to a named constant like maxRetries",
					Severity:    "low",
					Category:    "maintainability",
					File:        "handler.go",
					Line:        42,
				},
			},
		}

		// Act
		result := sanitizeReviewResult(input, printer)

		// Assert: findings are identical to input
		require.Len(t, result.Findings, 1, "Should still have one finding")
		assert.Equal(t, input.Findings[0].Description, result.Findings[0].Description,
			"Clean description should be unchanged")
		assert.Equal(t, input.Findings[0].Remediation, result.Findings[0].Remediation,
			"Clean remediation should be unchanged")
		assert.Equal(t, input.Findings[0].Severity, result.Findings[0].Severity,
			"Severity should be unchanged")
		assert.Equal(t, input.Findings[0].File, result.Findings[0].File,
			"File should be unchanged")
	})
}

func TestSanitizeFindingFieldEdgeCases(t *testing.T) {
	printer := ui.New(io.Discard)

	t.Run("[test_id:TS-GH1230-014] should sanitize secret in remediation when description is empty", func(t *testing.T) {
		// Arrange: finding with empty description, secret in remediation
		input := ReviewResult{
			Body:   "Review complete",
			Action: "comment",
			Findings: []ReviewFinding{
				{
					Description: "",
					Remediation: "Use ghp_ABCDEFghijklmnop1234567890abcdefghijklmn instead of hardcoded value",
					Severity:    "high",
					Category:    "security",
					File:        "auth.go",
				},
			},
		}

		// Act
		result := sanitizeReviewResult(input, printer)

		// Assert: empty description preserved, secret in remediation redacted
		assert.Empty(t, result.Findings[0].Description,
			"Empty description should remain empty")
		assert.NotContains(t, result.Findings[0].Remediation, "ABCDEFghijklmnop1234567890abcdefghijklmn",
			"Secret payload in remediation should be redacted even when description is empty")
	})

	t.Run("[test_id:TS-GH1230-015] should sanitize secret in description when remediation is empty", func(t *testing.T) {
		// Arrange: finding with secret in description, empty remediation
		input := ReviewResult{
			Body:   "Review complete",
			Action: "comment",
			Findings: []ReviewFinding{
				{
					Description: "Found leaked token ghp_ABCDEFghijklmnop1234567890abcdefghijklmn in source",
					Remediation: "",
					Severity:    "critical",
					Category:    "security",
					File:        "deploy.go",
				},
			},
		}

		// Act
		result := sanitizeReviewResult(input, printer)

		// Assert: secret in description redacted, empty remediation preserved
		assert.NotContains(t, result.Findings[0].Description, "ABCDEFghijklmnop1234567890abcdefghijklmn",
			"Secret payload in description should be redacted even when remediation is empty")
		assert.Empty(t, result.Findings[0].Remediation,
			"Empty remediation should remain empty")
	})

	t.Run("[test_id:TS-GH1230-016] should preserve finding field when entire content is a secret", func(t *testing.T) {
		// Arrange: finding where description is entirely a secret token
		input := ReviewResult{
			Body:   "Review complete",
			Action: "comment",
			Findings: []ReviewFinding{
				{
					Description: "ghp_ABCDEFghijklmnop1234567890abcdefghijklmn",
					Remediation: "Remove the token",
					Severity:    "critical",
					Category:    "security",
					File:        "leaked.go",
				},
			},
		}

		// Act
		result := sanitizeReviewResult(input, printer)

		// Assert: field is not empty — contains redaction marker
		assert.NotEmpty(t, result.Findings[0].Description,
			"Finding field should not be silently dropped when entire content is a secret")
		assert.NotContains(t, result.Findings[0].Description, "ABCDEFghijklmnop1234567890abcdefghijklmn",
			"The original secret payload should be redacted")
	})
}
