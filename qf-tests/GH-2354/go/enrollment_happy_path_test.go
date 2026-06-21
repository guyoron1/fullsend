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
Enrollment Happy Path Tests

STP Reference: outputs/stp/GH-2354/GH-2354_test_plan.md
Jira: GH-2354

These tests validate that enrollment install succeeds within expected time
when the workflow registers quickly, and reports success details including
workflow URL and reconciliation PRs.
*/

// TestEnrollmentHappyPath validates the happy-path enrollment flow.
func TestEnrollmentHappyPath(t *testing.T) {
	// Shared setup: a FakeClient with immediate workflow success and PRs.
	now := time.Now().UTC()
	workflowURL := "https://github.com/test-org/.fullsend/actions/runs/99"

	makeClient := func() *forge.FakeClient {
		return &forge.FakeClient{
			WorkflowRuns: map[string]*forge.WorkflowRun{
				"test-org/.fullsend/repo-maintenance.yml": {
					ID:         99,
					Status:     "completed",
					Conclusion: "success",
					CreatedAt:  now.Add(time.Minute).Format(time.RFC3339),
					HTMLURL:    workflowURL,
				},
			},
			PullRequests: map[string][]forge.ChangeProposal{
				"test-org/repo-a": {
					{
						Title: "chore: connect to fullsend agent pipeline",
						URL:   "https://github.com/test-org/repo-a/pull/42",
					},
				},
			},
		}
	}

	t.Run("should complete fast enrollment without delay", func(t *testing.T) {
		// [test_id:TS-GH-2354-009]
		client := makeClient()
		layer, _ := newEnrollmentLayer(t, client, []string{"repo-a"}, nil)

		start := time.Now()
		err := layer.Install(context.Background())
		elapsed := time.Since(start)

		// Assert: no error.
		require.NoError(t, err)
		// Assert: completes in under 5 seconds (first poll returns success).
		assert.Less(t, elapsed, 5*time.Second,
			"happy-path enrollment should complete in under 5 seconds, took %v", elapsed)
	})

	t.Run("should report success and workflow URL", func(t *testing.T) {
		// [test_id:TS-GH-2354-010]
		client := makeClient()
		layer, buf := newEnrollmentLayer(t, client, []string{"repo-a"}, nil)

		err := layer.Install(context.Background())
		require.NoError(t, err)

		output := buf.String()
		// Assert: output contains the workflow URL.
		assert.Contains(t, output, workflowURL,
			"output should contain the workflow run URL")
		assert.Contains(t, output, "https://github.com/",
			"output should contain a GitHub URL")
	})

	t.Run("should report reconciliation PRs", func(t *testing.T) {
		// [test_id:TS-GH-2354-011]
		client := makeClient()
		layer, buf := newEnrollmentLayer(t, client, []string{"repo-a"}, nil)

		err := layer.Install(context.Background())
		require.NoError(t, err)

		output := buf.String()
		// Assert: output mentions the reconciliation PR.
		assert.Contains(t, output, "repo-a/pull/42",
			"output should contain the reconciliation PR URL")
	})
}
