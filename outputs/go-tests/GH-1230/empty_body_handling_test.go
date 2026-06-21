package cli

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fullsend-ai/fullsend/internal/ui"
)

/*
Empty Body Handling Tests

STP Reference: outputs/stp/GH-1230/GH-1230_test_plan.md
Jira: GH-1230
Group 7: Empty review body handling (P2)
*/

func TestEmptyBodyHandling(t *testing.T) {
	printer := ui.New(io.Discard)

	t.Run("[test_id:TS-GH1230-017] should handle empty body without error", func(t *testing.T) {
		// Arrange: ReviewResult with empty body
		input := ReviewResult{
			Body:     "",
			Action:   "approve",
			Findings: []ReviewFinding{},
		}

		// Act: should not panic or error
		result := sanitizeReviewResult(input, printer)

		// Assert: body remains empty
		assert.Empty(t, result.Body,
			"Empty body should remain empty after sanitization")
		assert.Equal(t, "approve", result.Action,
			"Action should be preserved")
	})

	t.Run("[test_id:TS-GH1230-018] should handle failure action with empty body", func(t *testing.T) {
		// Arrange: ReviewResult with failure action and empty body
		input := ReviewResult{
			Body:     "",
			Action:   "failure",
			Reason:   "Agent timed out after 300s",
			Findings: []ReviewFinding{},
		}

		// Act: sanitization should not error on empty body in failure path
		result := sanitizeReviewResult(input, printer)

		// Assert: empty body preserved, action and reason unchanged
		assert.Empty(t, result.Body,
			"Empty body should remain empty for failure action")
		assert.Equal(t, "failure", result.Action,
			"Failure action should be preserved")
		assert.Equal(t, "Agent timed out after 300s", result.Reason,
			"Failure reason should be preserved")
	})
}
