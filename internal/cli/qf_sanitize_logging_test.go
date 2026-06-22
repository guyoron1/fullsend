package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fullsend-ai/fullsend/internal/ui"
)

// TS-GH-69-010: Verify redaction warning is logged when secrets are found in body.
func TestSanitizeReviewResult_WarningLoggedOnRedaction(t *testing.T) {
	var buf bytes.Buffer
	printer := ui.New(&buf)

	secret := "ghp_1234567890abcdefABCDEF1234567890abcd"
	result := ReviewResult{
		Body:     "Token: " + secret,
		Action:   "comment",
		Findings: []ReviewFinding{},
	}

	_ = sanitizeReviewResult(result, printer)

	// ASSERT-01: Redaction warning logged.
	output := buf.String()
	assert.True(t, strings.Contains(strings.ToLower(output), "redact") || strings.Contains(strings.ToLower(output), "secret"),
		"printer output should contain a sanitization/redaction warning; got: %s", output)
}

// TS-GH-69-011: Verify no warning is logged when review content is clean.
func TestSanitizeReviewResult_NoWarningWhenClean(t *testing.T) {
	var buf bytes.Buffer
	printer := ui.New(&buf)

	result := ReviewResult{
		Body:     "LGTM, no issues found.",
		Action:   "approve",
		Findings: []ReviewFinding{},
	}

	_ = sanitizeReviewResult(result, printer)

	// ASSERT-01: No spurious warning on clean content.
	output := buf.String()
	assert.NotContains(t, strings.ToLower(output), "redact",
		"no redaction warning should be printed for clean content")
	assert.NotContains(t, strings.ToLower(output), "secret",
		"no secret-related warning should be printed for clean content")
}
