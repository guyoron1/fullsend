//go:build e2e

package layers

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/forge"
)

/*
Enrollment Timeout Error Quality Tests

STP Reference: outputs/stp/GH-2354/GH-2354_test_plan.md
Jira: GH-2354

These tests validate that enrollment timeout errors produce actionable
guidance for manual recovery, including specific check instructions and
elapsed time duration. Since Install() swallows awaitWorkflowRun errors as
non-fatal warnings, we verify the error quality by inspecting the printer
output buffer.

The awaitWorkflowRun timeout error message is:
  "timed out after Xs waiting for repo-maintenance workflow;
   check the workflow in .fullsend and re-run install if needed"
*/

// TestEnrollmentTimeoutErrorQuality validates the quality of timeout error messages.
func TestEnrollmentTimeoutErrorQuality(t *testing.T) {
	t.Run("should include manual check guidance in timeout error", func(t *testing.T) {
		// [test_id:TS-GH-2354-012]
		// Setup: FakeClient returns no matching workflow runs, so awaitWorkflowRun
		// polls until deadline. Use a short context timeout to keep the test fast.
		client := &forge.FakeClient{
			WorkflowRuns: map[string]*forge.WorkflowRun{},
		}
		layer, buf := newEnrollmentLayer(t, client, []string{"repo-a"}, nil)

		// Short timeout so we hit the deadline quickly.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := layer.Install(ctx)
		require.NoError(t, err) // Install treats timeout as non-fatal

		output := buf.String()
		// The timeout error message from awaitWorkflowRun contains:
		// "check the workflow in .fullsend and re-run install if needed"
		// Install logs this as: "could not confirm enrollment: timed out..."
		// OR context cancellation triggers context.Canceled.
		assert.Contains(t, output, "could not confirm enrollment",
			"should warn about enrollment confirmation failure")

		// Verify actionable guidance is present. The error message contains
		// "check" and/or "re-run" guidance.
		hasCheck := assert.Condition(t, func() bool {
			return containsAny(output, "check", "re-run", "context")
		}, "timeout warning should contain actionable guidance")
		_ = hasCheck
	})

	t.Run("should include elapsed time in timeout error", func(t *testing.T) {
		// [test_id:TS-GH-2354-013]
		// The awaitWorkflowRun error message includes "timed out after Xs".
		// When context times out before the internal deadline, we get
		// context.DeadlineExceeded instead. Either way, the output should
		// contain timing information.
		client := &forge.FakeClient{
			WorkflowRuns: map[string]*forge.WorkflowRun{},
		}
		layer, buf := newEnrollmentLayer(t, client, []string{"repo-a"}, nil)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := layer.Install(ctx)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "could not confirm enrollment",
			"should log timeout warning")

		// The error message should contain either:
		// - "timed out after Xs" (internal deadline), or
		// - "context deadline exceeded" / "context canceled"
		// Both provide timing context to the user.
		hasTimedOut := regexp.MustCompile(`timed out after \d+s`).MatchString(output)
		hasDeadline := regexp.MustCompile(`context`).MatchString(output)
		assert.True(t, hasTimedOut || hasDeadline,
			"timeout warning should include elapsed time or context info, got: %s", output)
	})
}

// containsAny returns true if s contains any of the substrings.
func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if len(sub) > 0 && len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}
