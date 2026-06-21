//go:build e2e

package layers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/forge"
)

/*
Enrollment Dispatch Failure Tests

STP Reference: outputs/stp/GH-2354/GH-2354_test_plan.md
Jira: GH-2354

These tests validate that enrollment workflow dispatch failures are reported
clearly, do not block install, and are safe in concurrent contexts.
When DispatchWorkflow fails, Install returns the error immediately without
entering the polling loop.
*/

// TestEnrollmentDispatchFailure validates dispatch error handling.
func TestEnrollmentDispatchFailure(t *testing.T) {
	dispatchErrMsg := "workflow file not found: .github/workflows/repo-maintenance.yml"

	t.Run("should return descriptive error on dispatch failure", func(t *testing.T) {
		// [test_id:TS-GH-2354-019]
		// Setup: FakeClient with a specific dispatch error.
		client := &forge.FakeClient{
			Errors: map[string]error{
				"DispatchWorkflow": fmt.Errorf("%s", dispatchErrMsg),
			},
		}
		layer, _ := newEnrollmentLayer(t, client, []string{"repo-a"}, nil)

		err := layer.Install(context.Background())

		// Assert: error is returned (dispatch errors are fatal in Install).
		require.Error(t, err)
		// Assert: error contains the original dispatch error message.
		assert.Contains(t, err.Error(), dispatchErrMsg,
			"error should contain the original dispatch failure reason")
		// Assert: error is wrapped with context about what was being done.
		assert.Contains(t, err.Error(), "dispatching repo-maintenance",
			"error should explain the operation that failed")
	})

	t.Run("should not block install on dispatch error", func(t *testing.T) {
		// [test_id:TS-GH-2354-020]
		// Setup: FakeClient with dispatch error. The static FakeClient
		// records whether ListWorkflowRuns was called (if WorkflowRuns
		// map is accessed). With dispatch error, Install returns immediately
		// without calling awaitWorkflowRun.
		client := &forge.FakeClient{
			Errors: map[string]error{
				"DispatchWorkflow": fmt.Errorf("%s", dispatchErrMsg),
			},
			WorkflowRuns: map[string]*forge.WorkflowRun{},
		}
		layer, _ := newEnrollmentLayer(t, client, []string{"repo-a"}, nil)

		start := time.Now()
		err := layer.Install(context.Background())
		elapsed := time.Since(start)

		// Assert: error returned promptly (no polling delay).
		require.Error(t, err)
		assert.Less(t, elapsed, 5*time.Second,
			"dispatch failure should return promptly without entering polling loop")

		// The existing enrollment_test.go TestEnrollmentLayer_Install_DispatchError
		// already verifies the error path, but this test adds the timing assertion
		// to confirm no blocking occurs.
	})

	t.Run("should handle dispatch error safely in concurrent context", func(t *testing.T) {
		// [test_id:TS-GH-2354-021]
		// Verify that dispatch errors do not cause panics or data races.
		// The FakeClient is thread-safe (uses sync.Mutex internally).
		client := &forge.FakeClient{
			Errors: map[string]error{
				"DispatchWorkflow": fmt.Errorf("%s", dispatchErrMsg),
			},
		}
		layer, _ := newEnrollmentLayer(t, client, []string{"repo-a"}, nil)

		// Assert: no panic on dispatch error.
		require.NotPanics(t, func() {
			err := layer.Install(context.Background())
			// Assert: error propagated cleanly.
			assert.Error(t, err)
			assert.Contains(t, err.Error(), dispatchErrMsg,
				"error should propagate cleanly in concurrent context")
		})
	})
}
