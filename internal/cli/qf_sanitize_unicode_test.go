package cli

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fullsend-ai/fullsend/internal/ui"
)

// TS-GH-69-006: Verify secrets obfuscated with zero-width characters are detected and redacted.
func TestSanitizeReviewResult_ZeroWidthObfuscatedSecretDetected(t *testing.T) {
	printer := ui.New(io.Discard)

	// Build a GitHub PAT with zero-width non-joiner (U+200C) characters
	// interleaved to attempt obfuscation bypass.
	plain := "ghp_1234567890abcdefABCDEF1234567890abcd"
	var obfuscated string
	for _, c := range plain {
		obfuscated += string(c) + "\u200C"
	}

	result := ReviewResult{
		Body:     "Token: " + obfuscated,
		Action:   "comment",
		Findings: []ReviewFinding{},
	}

	sanitized := sanitizeReviewResult(result, printer)

	// ASSERT-01: Obfuscated secret is detected and redacted after
	// UnicodeNormalizer strips zero-width characters. The mask()
	// function preserves the first 4 chars as "ghp_..." so we assert
	// on the secret payload rather than the prefix.
	assert.NotContains(t, sanitized.Body, "1234567890abcdef",
		"obfuscated secret payload should be absent after normalization + redaction")
	assert.NotContains(t, sanitized.Body, plain,
		"full plaintext secret must not appear in sanitized body")

	// Zero-width characters themselves should be removed.
	assert.NotContains(t, sanitized.Body, "\u200C",
		"zero-width non-joiner characters should be stripped")
}

// TS-GH-69-007: Verify secrets obfuscated with fullwidth characters are detected and redacted.
func TestSanitizeReviewResult_FullwidthObfuscatedSecretDetected(t *testing.T) {
	printer := ui.New(io.Discard)

	// Use fullwidth 'g' (U+FF47) to obfuscate the GitHub PAT prefix.
	// NFKC normalization should convert it back to ASCII 'g'.
	body := "Token: \uFF47hp_1234567890abcdefABCDEF1234567890abcd"
	result := ReviewResult{
		Body:     body,
		Action:   "comment",
		Findings: []ReviewFinding{},
	}

	sanitized := sanitizeReviewResult(result, printer)

	// ASSERT-01: Fullwidth-obfuscated secret is detected and redacted
	// after NFKC normalization converts fullwidth chars to ASCII.
	assert.NotContains(t, sanitized.Body, "1234567890abcdef",
		"secret content should be absent after NFKC normalization + redaction")
}
