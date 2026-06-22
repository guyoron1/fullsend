package cli

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/ui"
)

// TS-GH-69-001: Verify secrets embedded in review body are redacted before posting.
func TestSanitizeReviewResult_SecretsInBodyAreRedacted(t *testing.T) {
	printer := ui.New(io.Discard)
	secret := "ghp_1234567890abcdefABCDEF1234567890abcd"
	result := ReviewResult{
		Body:   "Review looks good. Token: " + secret,
		Action: "approve",
	}

	sanitized := sanitizeReviewResult(result, printer)

	// ASSERT-01: Secret token is redacted from body.
	assert.NotContains(t, sanitized.Body, "ghp_1234567890",
		"secret should be redacted from body")
	assert.NotContains(t, sanitized.Body, secret,
		"full secret must not appear in sanitized body")

	// ASSERT-02: Non-secret text preserved.
	assert.Contains(t, sanitized.Body, "Review looks good",
		"surrounding text should be preserved")
}

// TS-GH-69-002: Verify clean review body (no secrets) passes through unchanged.
func TestSanitizeReviewResult_CleanBodyPassesThrough(t *testing.T) {
	printer := ui.New(io.Discard)
	result := ReviewResult{
		Body:     "This code looks great. No issues found.",
		Action:   "approve",
		Findings: []ReviewFinding{},
	}

	sanitized := sanitizeReviewResult(result, printer)

	// ASSERT-01: Clean body passes through unchanged.
	assert.Equal(t, result.Body, sanitized.Body,
		"clean body should pass through unchanged")
	assert.Equal(t, result.Action, sanitized.Action,
		"action should be preserved")
}

// TS-GH-69-008: Verify empty review body skips sanitization without error.
func TestSanitizeReviewResult_EmptyBodyHandledGracefully(t *testing.T) {
	printer := ui.New(io.Discard)
	result := ReviewResult{
		Body:     "",
		Action:   "comment",
		Findings: []ReviewFinding{},
	}

	// Should not panic.
	sanitized := sanitizeReviewResult(result, printer)

	// ASSERT-01: Empty body handled gracefully.
	assert.Empty(t, sanitized.Body,
		"empty body should remain empty after sanitization")
}

// TS-GH-69-009: Verify review with no findings sanitizes body only.
func TestSanitizeReviewResult_NoFindingsSanitizesBodyOnly(t *testing.T) {
	printer := ui.New(io.Discard)
	secret := "ghp_1234567890abcdefABCDEF1234567890abcd"
	result := ReviewResult{
		Body:     "LGTM. Token for CI: " + secret,
		Action:   "approve",
		Findings: []ReviewFinding{},
	}

	sanitized := sanitizeReviewResult(result, printer)

	// ASSERT-01: Body secret is redacted even with no findings.
	assert.NotContains(t, sanitized.Body, "ghp_1234567890",
		"body secret should be redacted")

	// Findings remain empty.
	require.Empty(t, sanitized.Findings,
		"findings should remain empty")
}
