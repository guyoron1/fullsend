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
Enrollment Happy Path (Regression Guard) Tests

STP Reference: outputs/stp/GH-2354/GH-2354_test_plan.md
STD Reference: outputs/std/GH-2354/GH-2354_test_description.yaml
Jira: GH-2354
Section: 4.4 Happy Path (Regression Guard)
*/

func TestEnrollmentHappyPath(t *testing.T) {
	t.Run("[test_id:TS-GH2354-011] Successful enrollment with PR discovery", func(t *testing.T) {
		// Full happy path: Install dispatches workflow, waits for completion,
		// and discovers enrollment PRs on enabled repos.
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
			PullRequests: map[string][]forge.ChangeProposal{
				"test-org/repo-a": {
					{Title: "chore: connect to fullsend agent pipeline",
						URL: "https://github.com/test-org/repo-a/pull/1"},
				},
			},
		}
		repos := []string{"repo-a", "repo-b"}
		layer, buf := newEnrollmentLayer(t, client, repos, nil)

		err := layer.Install(context.Background())

		require.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "dispatched repo-maintenance workflow")
		assert.Contains(t, output, "enrollment completed successfully")
		assert.Contains(t, output, "repo-a/pull/1")
	})

	t.Run("[test_id:TS-GH2354-012] Successful unenrollment with config update", func(t *testing.T) {
		// Full Uninstall flow: reads config.yaml, disables repos, dispatches
		// repo-maintenance, waits for completion, and reports unenrollment PRs.
		now := time.Now().UTC()
		cfgYAML := "version: \"1\"\ndispatch:\n  platform: github-actions\ndefaults:\n  roles: [triage]\n  max_implementation_retries: 2\n  auto_merge: false\nagents: []\nrepos:\n  repo-a:\n    enabled: true\n  repo-b:\n    enabled: true\n"
		client := &forge.FakeClient{
			FileContents: map[string][]byte{
				"test-org/.fullsend/config.yaml": []byte(cfgYAML),
			},
			WorkflowRuns: map[string]*forge.WorkflowRun{
				"test-org/.fullsend/repo-maintenance.yml": {
					ID:         42,
					Status:     "completed",
					Conclusion: "success",
					CreatedAt:  now.Add(time.Minute).Format(time.RFC3339),
					HTMLURL:    "https://github.com/test-org/.fullsend/actions/runs/42",
				},
			},
			PullRequests: map[string][]forge.ChangeProposal{
				"test-org/repo-a": {
					{Title: "chore: disconnect from fullsend agent pipeline",
						URL: "https://github.com/test-org/repo-a/pull/10"},
				},
				"test-org/repo-b": {
					{Title: "chore: disconnect from fullsend agent pipeline",
						URL: "https://github.com/test-org/repo-b/pull/11"},
				},
			},
		}
		layer, buf := newEnrollmentLayer(t, client, nil, []string{"repo-a", "repo-b"})

		err := layer.Uninstall(context.Background())

		require.NoError(t, err)

		// Verify config was updated with repos disabled.
		require.Len(t, client.CreatedFiles, 1)
		assert.Contains(t, string(client.CreatedFiles[0].Content), "enabled: false")
		assert.NotContains(t, string(client.CreatedFiles[0].Content), "enabled: true")

		output := buf.String()
		assert.Contains(t, output, "Unenrollment completed successfully")
		assert.Contains(t, output, "repo-a/pull/10")
		assert.Contains(t, output, "repo-b/pull/11")
	})

	t.Run("[test_id:TS-GH2354-013] No-op when no repos configured", func(t *testing.T) {
		// Install returns immediately when no repos are configured.
		client := &forge.FakeClient{}
		layer, buf := newEnrollmentLayer(t, client, nil, nil)

		err := layer.Install(context.Background())

		require.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "no repositories to reconcile")
	})
}
