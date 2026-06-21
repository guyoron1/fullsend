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
Enrollment Timeout and Bounded Wait Tests

STP Reference: outputs/stp/GH-2354/GH-2354_test_plan.md
STD Reference: outputs/std/GH-2354/GH-2354_test_description.yaml
Jira: GH-2354
Section: 4.1 Timeout and Bounded Wait
*/

func TestEnrollmentTimeout(t *testing.T) {
	t.Run("[test_id:TS-GH2354-001] Install completes within timeout on fast registration", func(t *testing.T) {
		// Scenario 1: Happy path — FakeClient returns a completed workflow run,
		// Install should finish quickly without hitting the timeout.
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
		repos := []string{"repo-a", "repo-b"}
		layer, buf := newEnrollmentLayer(t, client, repos, nil)

		start := time.Now()
		err := layer.Install(context.Background())
		elapsed := time.Since(start)

		require.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "enrollment completed successfully")
		assert.Less(t, elapsed, enrollmentWaitTimeout,
			"Install should complete well before the timeout")
	})

	t.Run("[test_id:TS-GH2354-002] Install times out with actionable error on slow registration", func(t *testing.T) {
		// Scenario 2: No workflow runs ever appear — Install should time out
		// with a non-fatal warning and actionable guidance.
		client := &forge.FakeClient{}
		repos := []string{"repo-a"}
		layer, buf := newEnrollmentLayer(t, client, repos, nil)

		err := layer.Install(context.Background())

		require.NoError(t, err, "timeout should be non-fatal")
		output := buf.String()
		assert.Contains(t, output, "could not confirm enrollment")
		assert.Contains(t, output, "re-run install if needed")
	})

	t.Run("[test_id:TS-GH2354-003] Uninstall times out with same bounded behavior", func(t *testing.T) {
		// Scenario 3: Uninstall shares awaitWorkflowRun with Install.
		// When the workflow never completes, Uninstall emits a timeout warning.
		cfgYAML := "version: \"1\"\ndispatch:\n  platform: github-actions\ndefaults:\n  roles: [triage]\n  max_implementation_retries: 2\n  auto_merge: false\nagents: []\nrepos:\n  repo-a:\n    enabled: true\n"
		client := &forge.FakeClient{
			FileContents: map[string][]byte{
				"test-org/.fullsend/config.yaml": []byte(cfgYAML),
			},
		}
		layer, buf := newEnrollmentLayer(t, client, nil, []string{"repo-a"})

		err := layer.Uninstall(context.Background())

		require.NoError(t, err, "timeout should be non-fatal")
		output := buf.String()
		assert.Contains(t, output, "could not confirm unenrollment")
	})

	t.Run("[test_id:TS-GH2354-004] Install respects context cancellation during wait", func(t *testing.T) {
		// Scenario 4: When the context is cancelled, Install returns promptly
		// without blocking until the full timeout.
		client := &forge.FakeClient{}
		repos := []string{"repo-a"}
		layer, buf := newEnrollmentLayer(t, client, repos, nil)
		ctx, cancel := context.WithCancel(context.Background())

		// Cancel context immediately to force early exit from awaitWorkflowRun.
		cancel()
		start := time.Now()
		err := layer.Install(ctx)
		elapsed := time.Since(start)

		require.NoError(t, err, "cancellation should be non-fatal")
		output := buf.String()
		assert.Contains(t, output, "could not confirm enrollment")
		assert.Less(t, elapsed, 10*time.Second,
			"should return promptly on cancellation, not wait for full timeout")
	})
}
