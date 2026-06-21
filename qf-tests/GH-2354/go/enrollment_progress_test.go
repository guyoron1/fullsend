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
Enrollment Progress Indicator Tests

STP Reference: outputs/stp/GH-2354/GH-2354_test_plan.md
STD Reference: outputs/std/GH-2354/GH-2354_test_description.yaml
Jira: GH-2354
Section: 4.3 Progress Indicators
*/

func TestEnrollmentProgress(t *testing.T) {
	t.Run("[test_id:TS-GH2354-008] Progress messages emitted during workflow registration wait", func(t *testing.T) {
		// When ListWorkflowRuns returns an error (workflow not yet registered),
		// awaitWorkflowRun should emit progress messages showing "waiting for
		// workflow registration" with elapsed time.
		client := &forge.FakeClient{
			Errors: map[string]error{
				"ListWorkflowRuns": fmt.Errorf("workflow not found"),
			},
		}
		repos := []string{"repo-a"}
		layer, buf := newEnrollmentLayer(t, client, repos, nil)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = layer.Install(ctx)

		output := buf.String()
		assert.Contains(t, output, "waiting for workflow registration")
		assert.Contains(t, output, "elapsed")
	})

	t.Run("[test_id:TS-GH2354-009] Progress messages emitted for in-progress workflow", func(t *testing.T) {
		// When ListWorkflowRuns returns a run with status "in_progress",
		// awaitWorkflowRun should emit the workflow run URL and status.
		now := time.Now().UTC()
		client := &forge.FakeClient{
			WorkflowRuns: map[string]*forge.WorkflowRun{
				"test-org/.fullsend/repo-maintenance.yml": {
					ID:         1,
					Status:     "in_progress",
					Conclusion: "",
					CreatedAt:  now.Add(time.Minute).Format(time.RFC3339),
					HTMLURL:    "https://github.com/test-org/.fullsend/actions/runs/1",
				},
			},
		}
		repos := []string{"repo-a"}
		layer, buf := newEnrollmentLayer(t, client, repos, nil)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = layer.Install(ctx)

		output := buf.String()
		assert.Contains(t, output, "actions/runs/1")
		assert.Contains(t, output, "in_progress")
	})

	t.Run("[test_id:TS-GH2354-010] No progress spam on immediate completion", func(t *testing.T) {
		// When the workflow completes on the first poll, no intermediate
		// "waiting..." messages should appear.
		now := time.Now().UTC()
		client := &forge.FakeClient{
			WorkflowRuns: map[string]*forge.WorkflowRun{
				"test-org/.fullsend/repo-maintenance.yml": {
					ID:         1,
					Status:     "completed",
					Conclusion: "success",
					CreatedAt:  now.Add(time.Minute).Format(time.RFC3339),
					HTMLURL:    "https://github.com/test-org/.fullsend/actions/runs/1",
				},
			},
		}
		repos := []string{"repo-a"}
		layer, buf := newEnrollmentLayer(t, client, repos, nil)

		err := layer.Install(context.Background())
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "enrollment completed successfully")
		assert.NotContains(t, output, "waiting for workflow registration")
	})
}
