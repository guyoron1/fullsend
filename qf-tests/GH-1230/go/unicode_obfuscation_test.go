package cli

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fullsend-ai/fullsend/internal/ui"
)

/*
Unicode Obfuscation Bypass Prevention Tests

STP Reference: outputs/stp/GH-1230/GH-1230_test_plan.md
Jira: GH-1230
Group 4: Zero-width Unicode obfuscation bypass prevention (P2)
*/

func TestUnicodeObfuscationBypassPrevention(t *testing.T) {
	printer := ui.New(io.Discard)

	t.Run("[test_id:TS-GH1230-009] should detect zero-width char obfuscated token", func(t *testing.T) {
		// Arrange: Token with U+200B (zero-width space) inserted between chars
		// "g\u200Bh\u200Bp\u200B_" + rest of token
		obfuscatedToken := "g\u200Bh\u200Bp\u200B_ABCDEFghijklmnop1234567890abcdefghijklmn"
		input := ReviewResult{
			Body:     "Token " + obfuscatedToken,
			Action:   "comment",
			Findings: []ReviewFinding{},
		}

		// Act
		result := sanitizeReviewResult(input, printer)

		// Assert: token is detected and redacted despite zero-width char obfuscation
		assert.NotContains(t, result.Body, "ABCDEFghijklmnop1234567890abcdefghijklmn",
			"Token should be detected after zero-width chars are stripped by UnicodeNormalizer")
		assert.NotContains(t, result.Body, "ABCDEFghijklmnop1234567890abcdefghijklmn",
			"Token payload should not appear in sanitized output")
	})

	t.Run("[test_id:TS-GH1230-010] should detect bidirectional override obfuscated token", func(t *testing.T) {
		// Arrange: Token wrapped with U+202A (LRE) and U+202C (PDF) bidi overrides
		obfuscatedBody := "Token \u202Aghp_ABCDEFghijklmnop1234567890abcdefghijklmn\u202C found"
		input := ReviewResult{
			Body:     obfuscatedBody,
			Action:   "comment",
			Findings: []ReviewFinding{},
		}

		// Act
		result := sanitizeReviewResult(input, printer)

		// Assert: bidi-obfuscated token is detected and redacted
		assert.NotContains(t, result.Body, "ABCDEFghijklmnop1234567890abcdefghijklmn",
			"Token should be detected after bidi override chars are stripped")
		assert.NotContains(t, result.Body, "ABCDEFghijklmnop1234567890abcdefghijklmn",
			"Token payload should not appear in sanitized output")
	})

	t.Run("[test_id:TS-GH1230-011] should detect mixed invisible char injection", func(t *testing.T) {
		// Arrange: Token with mixed invisible chars: BOM (U+FEFF), ZWJ (U+200D), bidi (U+202A)
		obfuscatedToken := "g\uFEFFh\u200Dp\u202A_ABCDEFghijklmnop1234567890abcdefghijklmn"
		input := ReviewResult{
			Body:     "Token " + obfuscatedToken,
			Action:   "comment",
			Findings: []ReviewFinding{},
		}

		// Act
		result := sanitizeReviewResult(input, printer)

		// Assert: mixed-obfuscated token is detected and redacted
		assert.NotContains(t, result.Body, "ABCDEFghijklmnop1234567890abcdefghijklmn",
			"Token should be detected after all invisible char types are stripped")
		assert.NotContains(t, result.Body, "ABCDEFghijklmnop1234567890abcdefghijklmn",
			"Token payload should not appear in sanitized output")
	})
}
