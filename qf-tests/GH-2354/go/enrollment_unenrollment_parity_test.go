//go:build e2e

package layers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/forge"
)

/*
Enrollment Unenrollment Parity Tests

STP Reference: outputs/stp/GH-2354/GH-2354_test_plan.md
Jira: GH-2354

These tests validate that the unenrollment (uninstall) workflow uses the
same bounded timeout and exponential backoff as enrollment install. Both
code paths share the awaitWorkflowRun function, so parity is enforced at
the code level. These tests confirm that parity through behavioural testing.
*/

// unenrollConfigYAML is a minimal config.yaml for unenrollment tests.
const unenrollConfigYAML = `version: "1"
dispatch:
  platform: github-actions
defaults:
  roles: [triage]
  max_implementation_retries: 2
  auto_merge: false
agents: []
repos:
  repo-a:
    enabled: true
`

// TestUnenrollmentParity validates that unenrollment uses the same timeout
// and backoff behaviour as enrollment.
func TestUnenrollmentParity(t *testing.T) {
	t.Run("should use bounded timeout for unenrollment", func(t *testing.T) {
		// [test_id:TS-GH-2354-017]
		// Setup: FakeClient with config but no workflow runs.
		// Unenrollment dispatches the workflow and then awaits it.
		// With no matching runs, awaitWorkflowRun will poll until timeout.
		client := forge.NewFakeClient()
		client.FileContents["test-org/.fullsend/config.yaml"] = []byte(unenrollConfigYAML)
		// No workflow runs → awaitWorkflowRun will timeout

		layer, buf := newEnrollmentLayer(t, client, nil, []string{"repo-a"})

		// Use a short context timeout to avoid waiting 3 minutes.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		start := time.Now()
		err := layer.Uninstall(ctx)
		elapsed := time.Since(start)

		// Uninstall treats awaitWorkflowRun errors as non-fatal.
		require.NoError(t, err)

		output := buf.String()
		// Verify the timeout/cancellation warning was emitted.
		assert.Contains(t, output, "could not confirm unenrollment",
			"should warn about unenrollment confirmation failure")

		// Verify the operation was bounded by the context timeout.
		assert.Less(t, elapsed, 10*time.Second,
			"unenrollment should complete within the context timeout")
	})

	t.Run("should match enrollment backoff pattern", func(t *testing.T) {
		// [test_id:TS-GH-2354-018]
		// Both Install and Uninstall call awaitWorkflowRun, which uses
		// nextInterval for backoff. Since nextInterval is a shared helper,
		// parity is guaranteed at the code level.
		//
		// This test verifies that the same constants are used by checking
		// that nextInterval produces the same sequence regardless of caller.
		// We also verify that unenrollment exercises the polling path.

		// Verify backoff constants are shared (they're package-level consts).
		assert.Equal(t, 2*time.Second, enrollmentPollInitial,
			"initial poll interval should be 2s for both enroll and unenroll")
		assert.Equal(t, 15*time.Second, enrollmentPollMax,
			"max poll interval should be 15s for both enroll and unenroll")
		assert.Equal(t, 3*time.Minute, enrollmentWaitTimeout,
			"wait timeout should be 3m for both enroll and unenroll")

		// Verify the backoff sequence is consistent.
		interval := enrollmentPollInitial
		expectedSequence := []time.Duration{
			4 * time.Second,
			8 * time.Second,
			enrollmentPollMax,
			enrollmentPollMax,
		}
		for i, expected := range expectedSequence {
			interval = nextInterval(interval)
			assert.Equal(t, expected, interval,
				"backoff step %d should be %v (shared by enroll and unenroll)", i+1, expected)
		}

		// Integration check: verify unenrollment actually enters the polling path.
		client := forge.NewFakeClient()
		client.FileContents["test-org/.fullsend/config.yaml"] = []byte(unenrollConfigYAML)
		// In-progress run to trigger polling progress messages.
		now := time.Now().UTC()
		client.WorkflowRuns["test-org/.fullsend/repo-maintenance.yml"] = &forge.WorkflowRun{
			ID:         5,
			Status:     "in_progress",
			Conclusion: "",
			CreatedAt:  now.Add(time.Minute).Format(time.RFC3339),
			HTMLURL:    "https://github.com/test-org/.fullsend/actions/runs/5",
		}

		layer, buf := newEnrollmentLayer(t, client, nil, []string{"repo-a"})

		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		defer cancel()

		err := layer.Uninstall(ctx)
		require.NoError(t, err)

		output := buf.String()
		// Verify that polling progress was emitted (same as enrollment).
		assert.Contains(t, output, "in_progress",
			"unenrollment should emit polling progress like enrollment")
		assert.Contains(t, output, "elapsed",
			"unenrollment should report elapsed time like enrollment")
	})
}
