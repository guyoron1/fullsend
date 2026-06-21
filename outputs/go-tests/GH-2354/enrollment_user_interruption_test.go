//go:build e2e

package layers

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/forge"
)

/*
Enrollment User Interruption Tests

STP Reference: outputs/stp/GH-2354/GH-2354_test_plan.md
Jira: GH-2354

These tests validate that enrollment handles user interruption (Ctrl+C /
context cancellation) gracefully during polling, treating it as a non-fatal
condition with no goroutine leaks. Install() treats awaitWorkflowRun errors
(including context.Canceled) as non-fatal warnings, so we verify the
behaviour via output assertions and timing.
*/

// TestEnrollmentUserInterruption validates graceful handling of context
// cancellation during enrollment polling.
func TestEnrollmentUserInterruption(t *testing.T) {
	t.Run("should stop polling on user interruption", func(t *testing.T) {
		// [test_id:TS-GH-2354-014]
		// Setup: No workflow runs → polling loop runs indefinitely.
		// Cancel context after a short delay to simulate Ctrl+C.
		client := &forge.FakeClient{
			WorkflowRuns: map[string]*forge.WorkflowRun{},
		}
		layer, buf := newEnrollmentLayer(t, client, []string{"repo-a"}, nil)

		ctx, cancel := context.WithCancel(context.Background())
		// Cancel after 1 second to simulate user interruption.
		go func() {
			time.Sleep(1 * time.Second)
			cancel()
		}()

		start := time.Now()
		err := layer.Install(ctx)
		elapsed := time.Since(start)

		// Install treats this as non-fatal.
		require.NoError(t, err)
		// Assert: returns promptly after cancellation (within a few seconds).
		assert.Less(t, elapsed, 5*time.Second,
			"enrollment should stop promptly after context cancellation")
		// Assert: warning was logged about the failure.
		output := buf.String()
		assert.Contains(t, output, "could not confirm enrollment",
			"should warn about enrollment confirmation failure on cancellation")
	})

	t.Run("should treat interruption as non-fatal", func(t *testing.T) {
		// [test_id:TS-GH-2354-015]
		// Install() returns nil even when awaitWorkflowRun returns
		// context.Canceled. This is by design: enrollment is best-effort.
		client := &forge.FakeClient{
			WorkflowRuns: map[string]*forge.WorkflowRun{},
		}
		layer, buf := newEnrollmentLayer(t, client, []string{"repo-a"}, nil)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// Assert: no panic.
		require.NotPanics(t, func() {
			err := layer.Install(ctx)
			// Install returns nil (non-fatal).
			assert.NoError(t, err, "context cancellation should be non-fatal")
		})

		output := buf.String()
		// The warning should reference context cancellation.
		assert.Contains(t, output, "could not confirm enrollment",
			"should warn about enrollment failure, not crash")
	})

	t.Run("should exit cleanly with no hanging processes", func(t *testing.T) {
		// [test_id:TS-GH-2354-016]
		// Record baseline goroutine count, run enrollment with cancellation,
		// then verify goroutines settle back to baseline.
		baseline := runtime.NumGoroutine()

		client := &forge.FakeClient{
			WorkflowRuns: map[string]*forge.WorkflowRun{},
		}
		layer, _ := newEnrollmentLayer(t, client, []string{"repo-a"}, nil)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := layer.Install(ctx)
		require.NoError(t, err)

		// Allow goroutines to settle.
		time.Sleep(200 * time.Millisecond)

		current := runtime.NumGoroutine()
		// Allow a small margin for background goroutines from the test framework.
		assert.LessOrEqual(t, current, baseline+2,
			"goroutine count should return near baseline (%d) after cancellation, got %d",
			baseline, current)
	})
}
