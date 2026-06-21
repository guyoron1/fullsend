package cli

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/ui"
)

/*
Post-Review Regression Tests

STP Reference: outputs/stp/GH-1230/GH-1230_test_plan.md
Jira: GH-1230
Group 9: Existing post-review functionality regression (P1)

Verifies that the addition of sanitization does not break existing
post-review action flows (approve, request-changes, comment, failure).
*/

func TestPostReviewRegressionWithSanitization(t *testing.T) {
	printer := ui.New(io.Discard)

	t.Run("[test_id:TS-GH1230-022] should complete approve flow with sanitization", func(t *testing.T) {
		// Arrange: ReviewResult with approve action and clean body
		input := ReviewResult{
			Body:     "LGTM",
			Action:   "approve",
			HeadSHA:  "sha123",
			Findings: []ReviewFinding{},
		}

		// Act
		result := sanitizeReviewResult(input, printer)

		// Assert: approve flow data is preserved through sanitization
		assert.Equal(t, "LGTM", result.Body,
			"Clean approve body should be unchanged")
		assert.Equal(t, "approve", result.Action,
			"Approve action should be preserved through sanitization")
		assert.Equal(t, "sha123", result.HeadSHA,
			"HeadSHA should be preserved through sanitization")
	})

	t.Run("[test_id:TS-GH1230-023] should complete request-changes flow with sanitization", func(t *testing.T) {
		// Arrange: ReviewResult with request-changes action and findings
		input := ReviewResult{
			Body:   "Please fix the following issues",
			Action: "request-changes",
			Findings: []ReviewFinding{
				{
					Description: "Missing nil check on pointer dereference",
					Remediation: "Add guard: if ptr == nil { return ErrNilPointer }",
					Severity:    "high",
					Category:    "reliability",
					File:        "service.go",
					Line:        55,
				},
			},
		}

		// Act
		result := sanitizeReviewResult(input, printer)

		// Assert: request-changes flow data is preserved
		assert.Equal(t, "request-changes", result.Action,
			"Request-changes action should be preserved")
		assert.Equal(t, "Please fix the following issues", result.Body,
			"Clean body should be unchanged")
		require.Len(t, result.Findings, 1, "Should preserve findings")
		assert.Equal(t, input.Findings[0].Description, result.Findings[0].Description,
			"Clean finding description should be unchanged")
		assert.Equal(t, input.Findings[0].Remediation, result.Findings[0].Remediation,
			"Clean finding remediation should be unchanged")
	})

	t.Run("[test_id:TS-GH1230-024] should complete comment flow with sanitization", func(t *testing.T) {
		// Arrange: ReviewResult with comment action
		input := ReviewResult{
			Body:     "Some observations about the implementation",
			Action:   "comment",
			HeadSHA:  "sha456",
			Findings: []ReviewFinding{},
		}

		// Act
		result := sanitizeReviewResult(input, printer)

		// Assert: comment flow data is preserved
		assert.Equal(t, "comment", result.Action,
			"Comment action should be preserved through sanitization")
		assert.Equal(t, "Some observations about the implementation", result.Body,
			"Clean comment body should be unchanged")
	})

	t.Run("[test_id:TS-GH1230-025] should complete failure flow with sanitization", func(t *testing.T) {
		// Arrange: ReviewResult with failure action
		input := ReviewResult{
			Body:     "Agent failed: timeout after 300s",
			Action:   "failure",
			Reason:   "timeout",
			Findings: []ReviewFinding{},
		}

		// Act
		result := sanitizeReviewResult(input, printer)

		// Assert: failure flow data is preserved
		assert.Equal(t, "failure", result.Action,
			"Failure action should be preserved through sanitization")
		assert.Equal(t, "Agent failed: timeout after 300s", result.Body,
			"Failure body should be unchanged (no secrets present)")
		assert.Equal(t, "timeout", result.Reason,
			"Failure reason should be preserved through sanitization")
	})

	t.Run("[test_id:TS-GH1230-026] should not interfere with stale head SHA comparison", func(t *testing.T) {
		// Arrange: ReviewResult with a specific HeadSHA
		reviewedSHA := "abc123def456"
		currentSHA := "789ghi012jkl" // different — stale head condition
		input := ReviewResult{
			Body:     "Review of commit " + reviewedSHA,
			Action:   "comment",
			HeadSHA:  reviewedSHA,
			Findings: []ReviewFinding{},
		}

		// Act
		result := sanitizeReviewResult(input, printer)

		// Assert: HeadSHA is preserved for stale-head comparison
		assert.Equal(t, reviewedSHA, result.HeadSHA,
			"Sanitization must not modify HeadSHA used for stale-head detection")
		assert.NotEqual(t, currentSHA, result.HeadSHA,
			"HeadSHA should still differ from current SHA (stale head detectable)")
		assert.Contains(t, result.Body, reviewedSHA,
			"SHA reference in body should be preserved (not a secret pattern)")
	})
}
