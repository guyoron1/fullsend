package cli

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/ui"
)

// TS-GH-69-003: Verify secrets in finding descriptions are redacted.
func TestSanitizeReviewResult_SecretsInFindingDescriptionRedacted(t *testing.T) {
	printer := ui.New(io.Discard)
	secret := "ghp_1234567890abcdefABCDEF1234567890abcd"
	result := ReviewResult{
		Body:   "Review complete",
		Action: "comment",
		Findings: []ReviewFinding{
			{
				Severity:    "high",
				Category:    "security",
				File:        "config.go",
				Line:        42,
				Description: "Hardcoded token found: " + secret,
				Remediation: "",
			},
		},
	}

	sanitized := sanitizeReviewResult(result, printer)

	// ASSERT-01: Secret redacted from finding description.
	assert.NotContains(t, sanitized.Findings[0].Description, "ghp_1234567890",
		"secret should be redacted from finding description")

	// ASSERT-02: Finding metadata unchanged.
	assert.Contains(t, sanitized.Findings[0].Description, "Hardcoded token found",
		"context text should be preserved in description")
	assert.Equal(t, "config.go", sanitized.Findings[0].File,
		"file field should be unchanged")
	assert.Equal(t, 42, sanitized.Findings[0].Line,
		"line field should be unchanged")
	assert.Equal(t, "high", sanitized.Findings[0].Severity,
		"severity field should be unchanged")
	assert.Equal(t, "security", sanitized.Findings[0].Category,
		"category field should be unchanged")
}

// TS-GH-69-004: Verify secrets in finding remediations are redacted.
func TestSanitizeReviewResult_SecretsInFindingRemediationRedacted(t *testing.T) {
	printer := ui.New(io.Discard)
	secret := "ghp_1234567890abcdefABCDEF1234567890abcd"
	result := ReviewResult{
		Body:   "Issues found",
		Action: "request-changes",
		Findings: []ReviewFinding{
			{
				Severity:    "critical",
				Category:    "security",
				File:        "auth.go",
				Line:        15,
				Description: "Hardcoded credential detected",
				Remediation: "Replace " + secret + " with env var",
			},
		},
	}

	sanitized := sanitizeReviewResult(result, printer)

	// ASSERT-01: Secret redacted from finding remediation.
	assert.NotContains(t, sanitized.Findings[0].Remediation, "ghp_1234567890",
		"secret should be redacted from finding remediation")
}

// TS-GH-69-005: Verify clean findings pass through unchanged.
func TestSanitizeReviewResult_CleanFindingsPassThrough(t *testing.T) {
	printer := ui.New(io.Discard)
	result := ReviewResult{
		Body:   "Found some issues",
		Action: "request-changes",
		Findings: []ReviewFinding{
			{
				Severity:    "medium",
				Category:    "style",
				File:        "handler.go",
				Line:        25,
				Description: "Consider using early return to reduce nesting",
				Remediation: "Refactor: if err != nil { return err }",
			},
		},
	}

	sanitized := sanitizeReviewResult(result, printer)

	// ASSERT-01: Clean findings pass through unchanged.
	require.Len(t, sanitized.Findings, 1)
	assert.Equal(t, result.Findings[0].Description, sanitized.Findings[0].Description,
		"clean finding description should pass through unchanged")
	assert.Equal(t, result.Findings[0].Remediation, sanitized.Findings[0].Remediation,
		"clean finding remediation should pass through unchanged")
	assert.Equal(t, result.Findings[0].File, sanitized.Findings[0].File)
	assert.Equal(t, result.Findings[0].Line, sanitized.Findings[0].Line)
	assert.Equal(t, result.Findings[0].Severity, sanitized.Findings[0].Severity)
	assert.Equal(t, result.Findings[0].Category, sanitized.Findings[0].Category)
}
