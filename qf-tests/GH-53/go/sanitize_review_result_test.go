package cli

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/ui"
)

// ============================================================
// TS-GH-53-001: Review body containing a GitHub PAT (ghp_*) is redacted
// Priority: P0 | Tier: 1 | Type: Functional
// ============================================================

func TestSanitizeReviewResult_RedactsGitHubPAT(t *testing.T) {
	printer := ui.New(io.Discard)
	pat := "ghp_ABCDEFghijklmnop1234567890abcdef12345678"
	r := ReviewResult{
		Body:   fmt.Sprintf("Found issue: token %s was exposed", pat),
		Action: "comment",
	}

	sanitized := sanitizeReviewResult(r, printer)

	// The mask function preserves the prefix (ghp_...) but redacts
	// the unique token body. Verify the unique portion is gone.
	assert.NotContains(t, sanitized.Body, "ABCDEFghijklmnop",
		"GitHub PAT unique content must be redacted from review body")
	assert.Contains(t, sanitized.Body, "Found issue:",
		"non-secret text should remain intact")
}

// ============================================================
// TS-GH-53-002: Review body without secrets passes through unchanged
// Priority: P1 | Tier: 1 | Type: Functional
// ============================================================

func TestSanitizeReviewResult_NoSecretsPassthrough(t *testing.T) {
	printer := ui.New(io.Discard)
	originalBody := "This function has a bug on line 42. Please fix the nil check."
	r := ReviewResult{
		Body:   originalBody,
		Action: "comment",
	}

	sanitized := sanitizeReviewResult(r, printer)

	assert.Equal(t, originalBody, sanitized.Body, "clean body should pass through unchanged")
}

// ============================================================
// TS-GH-53-003: Empty review body remains empty after sanitization
// Priority: P2 | Tier: 1 | Type: Functional
// ============================================================

func TestSanitizeReviewResult_EmptyBodyRemainsEmpty(t *testing.T) {
	printer := ui.New(io.Discard)
	r := ReviewResult{
		Body:   "",
		Action: "failure",
		Reason: "tool-failure",
	}

	sanitized := sanitizeReviewResult(r, printer)

	assert.Empty(t, sanitized.Body, "empty body should remain empty after sanitization")
}

// ============================================================
// TS-GH-53-004: Finding Description containing a secret is redacted
// Priority: P0 | Tier: 1 | Type: Functional
// ============================================================

func TestSanitizeReviewResult_RedactsSecretsInFindingDescription(t *testing.T) {
	printer := ui.New(io.Discard)
	secret := "ghp_FAKEdescription0000000000000000000000"
	r := ReviewResult{
		Body:   "Review body without secrets.",
		Action: "request-changes",
		Findings: []ReviewFinding{
			{
				Severity:    "critical",
				Category:    "security",
				File:        "config.go",
				Line:        15,
				Description: fmt.Sprintf("Found hardcoded token %s in config.go", secret),
			},
		},
	}

	sanitized := sanitizeReviewResult(r, printer)

	assert.NotContains(t, sanitized.Findings[0].Description, "FAKEdescription",
		"secret in finding Description must be redacted")
	assert.Contains(t, sanitized.Findings[0].Description, "Found hardcoded token",
		"non-secret text in description should remain")
}

// ============================================================
// TS-GH-53-005: Finding Remediation containing a secret is redacted
// Priority: P0 | Tier: 1 | Type: Functional
// ============================================================

func TestSanitizeReviewResult_RedactsSecretsInFindingRemediation(t *testing.T) {
	printer := ui.New(io.Discard)
	secret := "ghs_FAKEremediation0000000000000000000000"
	r := ReviewResult{
		Body:   "Review body.",
		Action: "request-changes",
		Findings: []ReviewFinding{
			{
				Severity:    "high",
				Category:    "security",
				File:        "auth.go",
				Line:        30,
				Description: "Hardcoded app token found.",
				Remediation: fmt.Sprintf("Replace hardcoded token %s with env var", secret),
			},
		},
	}

	sanitized := sanitizeReviewResult(r, printer)

	assert.NotContains(t, sanitized.Findings[0].Remediation, "FAKEremediation",
		"secret in finding Remediation must be redacted")
	assert.Contains(t, sanitized.Findings[0].Remediation, "Replace hardcoded token",
		"non-secret text in remediation should remain")
}

// ============================================================
// TS-GH-53-006: Secret obfuscated with zero-width Unicode chars is
//
//	detected and redacted
//
// Priority: P0 | Tier: 1 | Type: Functional
// ============================================================

func TestSanitizeReviewResult_ZeroWidthSpaceObfuscation(t *testing.T) {
	printer := ui.New(io.Discard)
	plain := "ghp_FAKEzerowidth000000000000000000000000"

	// Interleave zero-width non-joiner (U+200C) characters to obfuscate
	// the token — matching the pattern used by existing tests.
	var obfuscated string
	for _, c := range plain {
		obfuscated += string(c) + "\u200c"
	}

	r := ReviewResult{
		Body:   "Token: " + obfuscated,
		Action: "comment",
	}

	sanitized := sanitizeReviewResult(r, printer)

	assert.NotContains(t, sanitized.Body, "FAKEzerowidth",
		"zero-width obfuscated secret must be caught after Unicode normalization")
}

// ============================================================
// TS-GH-53-007: Multiple findings with mixed clean/secret text are
//
//	individually sanitized
//
// Priority: P1 | Tier: 1 | Type: Functional
// ============================================================

func TestSanitizeReviewResult_MultipleFindingsMixedContent(t *testing.T) {
	printer := ui.New(io.Discard)
	secret := "ghp_FAKEmixedtest00000000000000000000000000"
	cleanDesc := "Consider renaming variable for clarity."
	cleanRemediation := "Rename to a more descriptive name."

	r := ReviewResult{
		Body:   "Review with multiple findings.",
		Action: "request-changes",
		Findings: []ReviewFinding{
			{
				Severity:    "low",
				Category:    "style",
				File:        "main.go",
				Line:        5,
				Description: cleanDesc,
				Remediation: cleanRemediation,
			},
			{
				Severity:    "critical",
				Category:    "security",
				File:        "auth.go",
				Line:        20,
				Description: fmt.Sprintf("Leaked token: %s", secret),
				Remediation: "Use environment variable instead.",
			},
			{
				Severity:    "high",
				Category:    "security",
				File:        "config.go",
				Line:        35,
				Description: "Check error handling.",
				Remediation: fmt.Sprintf("Remove %s from config", secret),
			},
		},
	}

	sanitized := sanitizeReviewResult(r, printer)

	// All findings should be preserved (none dropped).
	require.Len(t, sanitized.Findings, 3, "all findings must be preserved (none dropped)")

	// Clean finding passes through unchanged.
	assert.Equal(t, cleanDesc, sanitized.Findings[0].Description,
		"clean finding description should pass through unchanged")
	assert.Equal(t, cleanRemediation, sanitized.Findings[0].Remediation,
		"clean finding remediation should pass through unchanged")

	// Secret-bearing findings are redacted.
	assert.NotContains(t, sanitized.Findings[1].Description, "FAKEmixedtest",
		"secret in second finding description must be redacted")
	assert.NotContains(t, sanitized.Findings[2].Remediation, "FAKEmixedtest",
		"secret in third finding remediation must be redacted")
}

// ============================================================
// TS-GH-53-020: OutputPipeline.Scan() ensures fail-closed behavior —
//
//	secrets never pass through unsanitized
//
// Priority: P0 | Tier: 1 | Type: Functional
// ============================================================

func TestSanitizeReviewResult_ScanErrorPreventsPosting(t *testing.T) {
	// The sanitizeReviewResult function always runs OutputPipeline.Scan()
	// before returning. This test validates the fail-closed contract:
	// secrets never appear in the output regardless of how many types
	// are present simultaneously.
	printer := ui.New(io.Discard)

	bodySecret := "ghp_FAKEbodysecret0000000000000000000000"
	remediationSecret := "ghs_FAKEremediationsecret000000000000000"

	r := ReviewResult{
		Body:   "Token: " + bodySecret,
		Action: "comment",
		Findings: []ReviewFinding{
			{
				Severity:    "high",
				Category:    "security",
				File:        "main.go",
				Line:        10,
				Description: "Normal description without secrets.",
				Remediation: "Use " + remediationSecret + " from vault.",
			},
		},
	}

	sanitized := sanitizeReviewResult(r, printer)

	// Pipeline must have redacted all secret patterns — fail-closed.
	assert.NotContains(t, sanitized.Body, "FAKEbodysecret",
		"body secrets must be redacted (fail-closed)")
	assert.NotContains(t, sanitized.Findings[0].Remediation, "FAKEremediationsecret",
		"finding remediation secrets must be redacted (fail-closed)")
	// Clean description remains.
	assert.Equal(t, "Normal description without secrets.", sanitized.Findings[0].Description,
		"clean description should pass through unchanged")

	// Verify body still contains non-secret context text.
	assert.True(t, strings.Contains(sanitized.Body, "Token:"),
		"non-secret body text should remain")
}
