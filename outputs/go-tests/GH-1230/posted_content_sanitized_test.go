package cli

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/ui"
)

/*
Posted Review Content Sanitization Tests

STP Reference: outputs/stp/GH-1230/GH-1230_test_plan.md
Jira: GH-1230
Group 8: Posted review content secret-free (P1)

These tests verify that sanitizeReviewResult produces output suitable for
posting to the forge API — no secrets should survive sanitization.
*/

func TestPostedContentIsSanitized(t *testing.T) {
	printer := ui.New(io.Discard)

	t.Run("[test_id:TS-GH1230-019] should produce secret-free body for PR comment posting", func(t *testing.T) {
		// Arrange: ReviewResult with a GitHub PAT in body, simulating agent output
		input := ReviewResult{
			Body:     "Analysis complete. Found token ghp_ABCDEFghijklmnop1234567890abcdefghijklmn in config.yaml",
			Action:   "comment",
			HeadSHA:  "abc123",
			Findings: []ReviewFinding{},
		}

		// Act: sanitize before posting
		result := sanitizeReviewResult(input, printer)

		// Assert: the body that would be posted to forge is secret-free
		assert.NotContains(t, result.Body, "ABCDEFghijklmnop1234567890abcdefghijklmn",
			"Body destined for PR comment should not contain GitHub PAT")
		assert.Contains(t, result.Body, "Analysis complete",
			"Non-secret content should be preserved for readability")
		assert.Equal(t, "comment", result.Action, "Action should be preserved")
		assert.Equal(t, "abc123", result.HeadSHA, "HeadSHA should be preserved")
	})

	t.Run("[test_id:TS-GH1230-020] should produce secret-free findings for formal review posting", func(t *testing.T) {
		// Arrange: ReviewResult with secrets in both finding description and remediation
		input := ReviewResult{
			Body:   "Review with findings",
			Action: "request-changes",
			Findings: []ReviewFinding{
				{
					Description: "Leaked token ghp_ABCDEFghijklmnop1234567890abcdefghijklmn in source",
					Remediation: "Replace ghp_XYZDEFghijklmnop1234567890abcdefghijklmn with env var",
					Severity:    "critical",
					Category:    "security",
					File:        "auth.go",
					Line:        10,
				},
				{
					Description: "AWS key AKIAIOSFODNN7EXAMPLE hardcoded",
					Remediation: "Use AWS IAM roles or secrets manager",
					Severity:    "critical",
					Category:    "security",
					File:        "deploy.go",
					Line:        25,
				},
			},
		}

		// Act
		result := sanitizeReviewResult(input, printer)

		// Assert: all finding fields destined for forge are secret-free
		require.Len(t, result.Findings, 2, "Should preserve finding count")
		for i, f := range result.Findings {
			assert.NotContains(t, f.Description, "ABCDEFghijklmnop1234567890abcdefghijklmn",
				"Finding %d description should not contain full GitHub PAT payload", i)
			assert.NotContains(t, f.Description, "AKIAIOSFODNN7EXAMPLE",
				"Finding %d description should not contain full AWS key", i)
			assert.NotContains(t, f.Remediation, "XYZDEFghijklmnop1234567890abcdefghijklmn",
				"Finding %d remediation should not contain full GitHub PAT payload", i)
		}
	})

	t.Run("[test_id:TS-GH1230-021] should produce secret-free body for sticky comment posting", func(t *testing.T) {
		// Arrange: ReviewResult with secrets that would go through sticky comment path
		// The sticky comment path uses the same sanitized body, so we verify
		// sanitizeReviewResult produces clean output regardless of posting mechanism.
		input := ReviewResult{
			Body:     "Sticky update: found credential ghp_ABCDEFghijklmnop1234567890abcdefghijklmn leaked",
			Action:   "comment",
			HeadSHA:  "def456",
			Findings: []ReviewFinding{},
		}

		// Act
		result := sanitizeReviewResult(input, printer)

		// Assert: body is clean for sticky comment posting
		assert.NotContains(t, result.Body, "ABCDEFghijklmnop1234567890abcdefghijklmn",
			"Body for sticky comment should not contain GitHub PAT")
		assert.Contains(t, result.Body, "Sticky update",
			"Non-secret content should be preserved for sticky comment")
	})
}
