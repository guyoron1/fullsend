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
Enrollment Progress Feedback Tests

STP Reference: outputs/stp/GH-2354/GH-2354_test_plan.md
Jira: GH-2354

These tests validate that enrollment provides progress feedback during each
polling phase, including elapsed time information. Progress messages are
emitted by awaitWorkflowRun when it finds an in_progress run or when
ListWorkflowRuns returns an error. We use FakeClient with an in_progress run
to trigger progress output, bounded by a short context timeout.
*/

// TestEnrollmentProgressFeedback validates progress output during polling.
func TestEnrollmentProgressFeedback(t *testing.T) {
	t.Run("should emit progress messages during polling", func(t *testing.T) {
		// [test_id:TS-GH-2354-007]
		// Setup: FakeClient with an in_progress workflow run. The polling loop
		// will find this run, see it's not completed, and log a progress message
		// with its status and elapsed time before continuing to poll.
		now := time.Now().UTC()
		client := &forge.FakeClient{
			WorkflowRuns: map[string]*forge.WorkflowRun{
				"test-org/.fullsend/repo-maintenance.yml": {
					ID:         10,
					Status:     "in_progress",
					Conclusion: "",
					CreatedAt:  now.Add(time.Minute).Format(time.RFC3339),
					HTMLURL:    "https://github.com/test-org/.fullsend/actions/runs/10",
				},
			},
		}
		layer, buf := newEnrollmentLayer(t, client, []string{"repo-a"}, nil)

		// Use a short context to allow a few poll iterations before cancellation.
		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		defer cancel()

		err := layer.Install(ctx)
		// Install treats awaitWorkflowRun errors as non-fatal.
		require.NoError(t, err)

		output := buf.String()
		// Assert: progress messages are present. The code emits
		// "workflow run <URL> (<status>, <elapsed> elapsed)" for in-progress runs.
		assert.Greater(t, len(output), 0,
			"printer buffer should contain output")
		assert.Contains(t, output, "in_progress",
			"progress output should mention workflow status")
	})

	t.Run("should report elapsed time in status updates", func(t *testing.T) {
		// [test_id:TS-GH-2354-008]
		// Same setup as above: in_progress run triggers progress messages
		// that include elapsed time.
		now := time.Now().UTC()
		client := &forge.FakeClient{
			WorkflowRuns: map[string]*forge.WorkflowRun{
				"test-org/.fullsend/repo-maintenance.yml": {
					ID:         10,
					Status:     "in_progress",
					Conclusion: "",
					CreatedAt:  now.Add(time.Minute).Format(time.RFC3339),
					HTMLURL:    "https://github.com/test-org/.fullsend/actions/runs/10",
				},
			},
		}
		layer, buf := newEnrollmentLayer(t, client, []string{"repo-a"}, nil)

		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		defer cancel()

		err := layer.Install(ctx)
		require.NoError(t, err)

		output := buf.String()
		// Assert: output contains elapsed time. The code formats elapsed as
		// time.Duration.Round(time.Second), which produces strings like "2s", "4s".
		// The "elapsed" keyword is also present in the format string.
		assert.Contains(t, output, "elapsed",
			"progress output should mention elapsed time")
		matched, _ := regexp.MatchString(`\d+s`, output)
		assert.True(t, matched,
			"progress output should contain a duration value like '2s' or '4s', got: %s", output)
	})
}
