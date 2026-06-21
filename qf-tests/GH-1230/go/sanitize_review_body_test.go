package cli

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/ui"
)

/*
Sanitize Review Body Tests

STP Reference: outputs/stp/GH-1230/GH-1230_test_plan.md
Jira: GH-1230
Group 1: Review body sanitization (P0)
Group 2: Edge cases in review body sanitization (P2)
*/

func TestSanitizeReviewBody(t *testing.T) {
	printer := ui.New(io.Discard)

	t.Run("[test_id:TS-GH1230-001] should redact GitHub PAT from review body", func(t *testing.T) {
		// Arrange: ReviewResult with a full-length GitHub PAT in body
		input := ReviewResult{
			Body:     "Found issue: token ghp_ABCDEFghijklmnop1234567890abcdefghijklmn was exposed",
			Action:   "comment",
			Findings: []ReviewFinding{},
		}

		// Act
		result := sanitizeReviewResult(input, printer)

		// Assert: full PAT is redacted (mask() replaces with first 4 chars + "..."),
		// surrounding text preserved
		assert.NotContains(t, result.Body, "ABCDEFghijklmnop1234567890abcdefghijklmn",
			"Full GitHub PAT payload should be redacted from body")
		assert.Contains(t, result.Body, "ghp_...",
			"Masked token placeholder should be present")
		assert.Contains(t, result.Body, "Found issue:", "Non-secret prefix should be preserved")
		assert.Contains(t, result.Body, "was exposed", "Non-secret suffix should be preserved")
	})

	t.Run("[test_id:TS-GH1230-002] should redact multiple secret types from body", func(t *testing.T) {
		// Arrange: ReviewResult with both a GitHub PAT and an AWS key
		input := ReviewResult{
			Body:     "Token ghp_ABCDEFghijklmnop1234567890abcdefghijklmn and key AKIAIOSFODNN7EXAMPLE found in code",
			Action:   "comment",
			Findings: []ReviewFinding{},
		}

		// Act
		result := sanitizeReviewResult(input, printer)

		// Assert: both secret patterns are redacted (mask uses first 4 chars + "...")
		assert.NotContains(t, result.Body, "ABCDEFghijklmnop1234567890abcdefghijklmn",
			"GitHub PAT payload should be redacted")
		assert.NotContains(t, result.Body, "AKIAIOSFODNN7EXAMPLE",
			"Full AWS access key should be redacted")
		assert.Contains(t, result.Body, "ghp_...", "GitHub PAT masked placeholder should be present")
		assert.Contains(t, result.Body, "AKIA...", "AWS key masked placeholder should be present")
		assert.Contains(t, result.Body, "found in code", "Non-secret content between secrets should be preserved")
	})

	t.Run("[test_id:TS-GH1230-003] should pass clean body through unchanged", func(t *testing.T) {
		// Arrange: ReviewResult with clean body (no secrets)
		originalBody := "This code looks good. Consider adding error handling on line 42."
		input := ReviewResult{
			Body:     originalBody,
			Action:   "approve",
			Findings: []ReviewFinding{},
		}

		// Act
		result := sanitizeReviewResult(input, printer)

		// Assert: body is byte-for-byte identical
		assert.Equal(t, originalBody, result.Body, "Clean body should pass through unchanged")
	})

	t.Run("[test_id:TS-GH1230-004] should not over-redact partial token patterns", func(t *testing.T) {
		// Arrange: ReviewResult with a partial/invalid token pattern (too short to be real)
		originalBody := "Variable ghp_short is not a real token"
		input := ReviewResult{
			Body:     originalBody,
			Action:   "comment",
			Findings: []ReviewFinding{},
		}

		// Act
		result := sanitizeReviewResult(input, printer)

		// Assert: partial pattern is NOT redacted (no false positive)
		assert.Equal(t, originalBody, result.Body,
			"Partial token pattern should not be redacted; body should be unchanged")
	})

	t.Run("[test_id:TS-GH1230-005] should preserve non-obfuscation Unicode characters in body", func(t *testing.T) {
		// Arrange: ReviewResult with legitimate non-ASCII Unicode
		originalBody := "Review: 良いコード 🎉 résumé naïve"
		input := ReviewResult{
			Body:     originalBody,
			Action:   "comment",
			Findings: []ReviewFinding{},
		}

		// Act
		result := sanitizeReviewResult(input, printer)

		// Assert: legitimate Unicode is preserved
		require.NotEmpty(t, result.Body, "Body should not be empty after sanitization")
		assert.Contains(t, result.Body, "良いコード", "CJK characters should be preserved")
		assert.Contains(t, result.Body, "🎉", "Emoji should be preserved")
		assert.Contains(t, result.Body, "résumé", "Accented characters should be preserved")
		assert.Contains(t, result.Body, "naïve", "Diaeresis characters should be preserved")
	})
}
