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
Enrollment Error Handling Tests

STP Reference: outputs/stp/GH-2354/GH-2354_test_plan.md
STD Reference: outputs/std/GH-2354/GH-2354_test_description.yaml
Jira: GH-2354
Section: 4.5 Error Handling
*/

func TestEnrollmentErrorHandling(t *testing.T) {
	t.Run("[test_id:TS-GH2354-014] Dispatch failure returns error", func(t *testing.T) {
		// When DispatchWorkflow fails, Install should propagate the error
		// wrapping "dispatching repo-maintenance" and not proceed to polling.
		client := &forge.FakeClient{
			Errors: map[string]error{
				"DispatchWorkflow": assert.AnError,
			},
		}
		repos := []string{"repo-a"}
		layer, _ := newEnrollmentLayer(t, client, repos, nil)

		err := layer.Install(context.Background())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "dispatching repo-maintenance")
	})

	t.Run("[test_id:TS-GH2354-015] Non-success workflow conclusion shows logs", func(t *testing.T) {
		// When the workflow completes with a failure conclusion, Install
		// should emit a warning with the conclusion and fetch workflow logs.
		now := time.Now().UTC()
		client := &forge.FakeClient{
			WorkflowRuns: map[string]*forge.WorkflowRun{
				"test-org/.fullsend/repo-maintenance.yml": {
					ID:         1,
					Status:     "completed",
					Conclusion: "failure",
					CreatedAt:  now.Add(time.Minute).Format(time.RFC3339),
					HTMLURL:    "https://github.com/test-org/.fullsend/actions/runs/1",
				},
			},
		}
		repos := []string{"repo-a"}
		layer, buf := newEnrollmentLayer(t, client, repos, nil)

		err := layer.Install(context.Background())

		require.NoError(t, err, "non-success conclusion is non-fatal")
		output := buf.String()
		assert.Contains(t, output, "conclusion: failure")
	})

	t.Run("[test_id:TS-GH2354-016] Log fetch failure is non-fatal", func(t *testing.T) {
		// When GetWorkflowRunLogs fails after a failed workflow run,
		// the error is handled gracefully with an informational message.
		now := time.Now().UTC()
		client := &forge.FakeClient{
			WorkflowRuns: map[string]*forge.WorkflowRun{
				"test-org/.fullsend/repo-maintenance.yml": {
					ID:         1,
					Status:     "completed",
					Conclusion: "failure",
					CreatedAt:  now.Add(time.Minute).Format(time.RFC3339),
					HTMLURL:    "https://github.com/test-org/.fullsend/actions/runs/1",
				},
			},
			Errors: map[string]error{
				"GetWorkflowRunLogs": fmt.Errorf("logs unavailable"),
			},
		}
		repos := []string{"repo-a"}
		layer, buf := newEnrollmentLayer(t, client, repos, nil)

		err := layer.Install(context.Background())

		require.NoError(t, err, "log fetch failure should not crash install")
		output := buf.String()
		assert.Contains(t, output, "could not fetch workflow logs")
	})

	t.Run("[test_id:TS-GH2354-017] Workflow run with unparseable CreatedAt is skipped", func(t *testing.T) {
		// When a workflow run has an invalid CreatedAt timestamp,
		// awaitWorkflowRun skips it and continues polling.
		client := &forge.FakeClient{
			WorkflowRuns: map[string]*forge.WorkflowRun{
				"test-org/.fullsend/repo-maintenance.yml": {
					ID:         1,
					Status:     "completed",
					Conclusion: "success",
					CreatedAt:  "not-a-valid-timestamp",
					HTMLURL:    "https://github.com/test-org/.fullsend/actions/runs/1",
				},
			},
		}
		repos := []string{"repo-a"}
		layer, buf := newEnrollmentLayer(t, client, repos, nil)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := layer.Install(ctx)

		require.NoError(t, err, "unparseable timestamp should not panic")
		output := buf.String()
		assert.Contains(t, output, "could not confirm enrollment",
			"should time out because the only run was skipped")
	})
}
