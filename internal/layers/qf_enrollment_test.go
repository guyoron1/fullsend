package layers

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/forge"
	"github.com/fullsend-ai/fullsend/internal/ui"
)

// QualityFlow generated tests for GH-76: bound enrollment wait with timeout and backoff
// Covers: enrollment timeout behavior, backoff calculation, context cancellation,
// Install/Uninstall lifecycle, and Analyze edge cases.

// --- nextInterval backoff tests ---

func TestQF_NextInterval_DoublesFromInitial(t *testing.T) {
	got := nextInterval(enrollmentPollInitial)
	assert.Equal(t, 4*time.Second, got, "should double from 2s to 4s")
}

func TestQF_NextInterval_DoublesFrom4sTo8s(t *testing.T) {
	got := nextInterval(4 * time.Second)
	assert.Equal(t, 8*time.Second, got, "should double from 4s to 8s")
}

func TestQF_NextInterval_CapsAtMax(t *testing.T) {
	got := nextInterval(8 * time.Second)
	assert.Equal(t, enrollmentPollMax, got, "should cap at 15s when doubling 8s exceeds max")
}

func TestQF_NextInterval_StaysAtMaxWhenAlreadyAtMax(t *testing.T) {
	got := nextInterval(enrollmentPollMax)
	assert.Equal(t, enrollmentPollMax, got, "should remain at max when already at max")
}

func TestQF_NextInterval_LargeValueCapsAtMax(t *testing.T) {
	got := nextInterval(1 * time.Minute)
	assert.Equal(t, enrollmentPollMax, got, "should cap at max even for very large inputs")
}

func TestQF_NextInterval_SubSecondInterval(t *testing.T) {
	got := nextInterval(500 * time.Millisecond)
	assert.Equal(t, 1*time.Second, got, "should double sub-second interval")
}

// --- awaitWorkflowRun timeout and context tests ---

func TestQF_AwaitWorkflowRun_ContextCancelled(t *testing.T) {
	// awaitWorkflowRun should return context.Canceled when context is cancelled.
	client := &forge.FakeClient{}
	var buf bytes.Buffer
	printer := ui.New(&buf)
	layer := NewEnrollmentLayer("test-org", client, []string{"repo-a"}, nil, printer)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := layer.awaitWorkflowRun(ctx, time.Now().UTC())
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestQF_AwaitWorkflowRun_ReturnsCompletedRun(t *testing.T) {
	dispatchTime := time.Now().UTC().Add(-30 * time.Second)
	client := &forge.FakeClient{
		WorkflowRuns: map[string]*forge.WorkflowRun{
			"test-org/.fullsend/repo-maintenance.yml": {
				ID:         5,
				Status:     "completed",
				Conclusion: "success",
				CreatedAt:  time.Now().UTC().Add(time.Second).Format(time.RFC3339),
				HTMLURL:    "https://github.com/test-org/.fullsend/actions/runs/5",
			},
		},
	}
	var buf bytes.Buffer
	printer := ui.New(&buf)
	layer := NewEnrollmentLayer("test-org", client, []string{"repo-a"}, nil, printer)

	run, err := layer.awaitWorkflowRun(context.Background(), dispatchTime)
	require.NoError(t, err)
	require.NotNil(t, run)
	assert.Equal(t, 5, run.ID)
	assert.Equal(t, "completed", run.Status)
	assert.Equal(t, "success", run.Conclusion)
}

func TestQF_AwaitWorkflowRun_SkipsOldRuns(t *testing.T) {
	// Runs created before dispatchTime should be ignored.
	dispatchTime := time.Now().UTC()
	client := &forge.FakeClient{
		WorkflowRuns: map[string]*forge.WorkflowRun{
			"test-org/.fullsend/repo-maintenance.yml": {
				ID:         1,
				Status:     "completed",
				Conclusion: "success",
				CreatedAt:  dispatchTime.Add(-10 * time.Minute).Format(time.RFC3339),
				HTMLURL:    "https://github.com/test-org/.fullsend/actions/runs/1",
			},
		},
	}
	var buf bytes.Buffer
	printer := ui.New(&buf)
	layer := NewEnrollmentLayer("test-org", client, []string{"repo-a"}, nil, printer)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := layer.awaitWorkflowRun(ctx, dispatchTime)
	require.Error(t, err)
	// Either timeout or context cancellation, but should not return the old run.
}

// --- Install lifecycle tests ---

func TestQF_Install_DispatchErrorIsFatal(t *testing.T) {
	client := &forge.FakeClient{
		Errors: map[string]error{
			"DispatchWorkflow": assert.AnError,
		},
	}
	layer, _ := newEnrollmentLayer(t, client, []string{"repo-a"}, nil)

	err := layer.Install(context.Background())
	require.Error(t, err, "dispatch error should be fatal")
	assert.Contains(t, err.Error(), "dispatching repo-maintenance")
}

func TestQF_Install_WorkflowFailureIsNonFatal(t *testing.T) {
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
	layer, buf := newEnrollmentLayer(t, client, []string{"repo-a"}, nil)

	err := layer.Install(context.Background())
	require.NoError(t, err, "workflow failure should be non-fatal (Install succeeds with warning)")

	output := buf.String()
	assert.Contains(t, output, "conclusion: failure")
}

func TestQF_Install_NoReposSkipsDispatch(t *testing.T) {
	client := &forge.FakeClient{}
	layer, buf := newEnrollmentLayer(t, client, nil, nil)

	err := layer.Install(context.Background())
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "no repositories to reconcile")
}

func TestQF_Install_ReportsEnrollmentPRsAfterSuccess(t *testing.T) {
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
				{Title: "chore: connect to fullsend agent pipeline", URL: "https://github.com/test-org/repo-a/pull/1"},
			},
			"test-org/repo-b": {
				{Title: "chore: connect to fullsend agent pipeline", URL: "https://github.com/test-org/repo-b/pull/2"},
			},
		},
	}
	repos := []string{"repo-a", "repo-b"}
	layer, buf := newEnrollmentLayer(t, client, repos, nil)

	err := layer.Install(context.Background())
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "enrollment completed successfully")
	assert.Contains(t, output, "repo-a/pull/1")
	assert.Contains(t, output, "repo-b/pull/2")
}

func TestQF_Install_ReportsRemovalPRsForDisabledRepos(t *testing.T) {
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
			"test-org/repo-x": {
				{Title: "chore: disconnect from fullsend agent pipeline", URL: "https://github.com/test-org/repo-x/pull/5"},
			},
		},
	}
	layer, buf := newEnrollmentLayer(t, client, nil, []string{"repo-x"})

	err := layer.Install(context.Background())
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "repo-x/pull/5")
}

// --- Uninstall lifecycle tests ---

func TestQF_Uninstall_NoReposSkipsUnenrollment(t *testing.T) {
	client := &forge.FakeClient{}
	layer, buf := newEnrollmentLayer(t, client, nil, nil)

	err := layer.Uninstall(context.Background())
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "no repositories to unenroll")
}

func TestQF_Uninstall_ConfigMissingReturnsNoError(t *testing.T) {
	// When config.yaml is not found, Uninstall should gracefully skip.
	client := &forge.FakeClient{
		FileContents: map[string][]byte{},
	}
	layer, buf := newEnrollmentLayer(t, client, nil, []string{"repo-a"})

	err := layer.Uninstall(context.Background())
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "config repo unavailable")
}

func TestQF_Uninstall_DispatchErrorIsNonFatal(t *testing.T) {
	cfgYAML := `version: "1"
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
	client := &forge.FakeClient{
		FileContents: map[string][]byte{
			"test-org/.fullsend/config.yaml": []byte(cfgYAML),
		},
		Errors: map[string]error{
			"DispatchWorkflow": assert.AnError,
		},
	}
	layer, buf := newEnrollmentLayer(t, client, nil, []string{"repo-a"})

	err := layer.Uninstall(context.Background())
	require.NoError(t, err, "Uninstall dispatch error should be non-fatal")

	output := buf.String()
	assert.Contains(t, output, "could not dispatch unenrollment workflow")
	assert.Contains(t, output, "manual cleanup")
}

func TestQF_Uninstall_DisablesAndReportsSuccess(t *testing.T) {
	now := time.Now().UTC()
	cfgYAML := `version: "1"
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
	}
	layer, buf := newEnrollmentLayer(t, client, nil, []string{"repo-a"})

	err := layer.Uninstall(context.Background())
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Disabled all repos in config")
	assert.Contains(t, output, "Unenrollment completed successfully")

	// Verify config was updated with repos disabled.
	require.Len(t, client.CreatedFiles, 1)
	assert.Contains(t, string(client.CreatedFiles[0].Content), "enabled: false")
	assert.NotContains(t, string(client.CreatedFiles[0].Content), "enabled: true")
}

// --- Analyze tests ---

func TestQF_Analyze_AllEnrolledReportsInstalled(t *testing.T) {
	client := &forge.FakeClient{
		FileContents: map[string][]byte{
			"test-org/repo-a/.github/workflows/fullsend.yaml": []byte("shim"),
			"test-org/repo-b/.github/workflows/fullsend.yaml": []byte("shim"),
		},
	}
	repos := []string{"repo-a", "repo-b"}
	layer, _ := newEnrollmentLayer(t, client, repos, nil)

	report, err := layer.Analyze(context.Background())
	require.NoError(t, err)

	assert.Equal(t, StatusInstalled, report.Status)
	assert.Len(t, report.Details, 2)
	assert.Empty(t, report.WouldInstall)
}

func TestQF_Analyze_MissingShimReportsNotInstalled(t *testing.T) {
	client := &forge.FakeClient{
		FileContents: map[string][]byte{},
	}
	layer, _ := newEnrollmentLayer(t, client, []string{"repo-a"}, nil)

	report, err := layer.Analyze(context.Background())
	require.NoError(t, err)

	assert.Equal(t, StatusNotInstalled, report.Status)
	require.Len(t, report.WouldInstall, 1)
	assert.Contains(t, report.WouldInstall[0], "repo-a")
}

func TestQF_Analyze_PartialReportsDegraded(t *testing.T) {
	client := &forge.FakeClient{
		FileContents: map[string][]byte{
			"test-org/repo-a/.github/workflows/fullsend.yaml": []byte("shim"),
		},
	}
	layer, _ := newEnrollmentLayer(t, client, []string{"repo-a", "repo-b"}, nil)

	report, err := layer.Analyze(context.Background())
	require.NoError(t, err)

	assert.Equal(t, StatusDegraded, report.Status)
	assert.Len(t, report.Details, 1) // repo-a enrolled
	assert.Len(t, report.WouldInstall, 1)
	assert.Contains(t, report.WouldInstall[0], "repo-b")
}

func TestQF_Analyze_PerRepoGuardSkipsOrgAnalysis(t *testing.T) {
	client := forge.NewFakeClient()
	client.VariableValues["test-org/repo-a/FULLSEND_PER_REPO_INSTALL"] = "true"
	layer, _ := newEnrollmentLayer(t, client, []string{"repo-a"}, nil)

	report, err := layer.Analyze(context.Background())
	require.NoError(t, err)

	assert.Equal(t, StatusInstalled, report.Status)
	assert.Contains(t, report.Details[0], "per-repo install, skipped")
	assert.Empty(t, report.WouldInstall)
}

func TestQF_Analyze_GuardCheckFailureSurfacesWarning(t *testing.T) {
	client := forge.NewFakeClient()
	client.Errors["GetRepoVariable"] = assert.AnError
	layer, _ := newEnrollmentLayer(t, client, []string{"repo-a"}, nil)

	report, err := layer.Analyze(context.Background())
	require.NoError(t, err)

	assert.Equal(t, StatusDegraded, report.Status)
	require.Len(t, report.Details, 2)
	assert.Contains(t, report.Details[0], "failed guard check")
}

func TestQF_Analyze_StaleShimOnDisabledRepoGeneratesRemoval(t *testing.T) {
	client := &forge.FakeClient{
		FileContents: map[string][]byte{
			"test-org/repo-x/.github/workflows/fullsend.yaml": []byte("shim"),
		},
	}
	layer, _ := newEnrollmentLayer(t, client, nil, []string{"repo-x"})

	report, err := layer.Analyze(context.Background())
	require.NoError(t, err)

	assert.Equal(t, StatusDegraded, report.Status)
	require.Len(t, report.WouldFix, 1)
	assert.Contains(t, report.WouldFix[0], "removal PR for repo-x")
}

// --- RequiredScopes tests ---

func TestQF_RequiredScopes_Install(t *testing.T) {
	layer, _ := newEnrollmentLayer(t, &forge.FakeClient{}, nil, nil)
	scopes := layer.RequiredScopes(OpInstall)
	assert.Equal(t, []string{"repo"}, scopes)
}

func TestQF_RequiredScopes_Uninstall_WithDisabledRepos(t *testing.T) {
	layer, _ := newEnrollmentLayer(t, &forge.FakeClient{}, nil, []string{"repo-a"})
	scopes := layer.RequiredScopes(OpUninstall)
	assert.Equal(t, []string{"repo"}, scopes)
}

func TestQF_RequiredScopes_Uninstall_NoDisabledRepos(t *testing.T) {
	layer, _ := newEnrollmentLayer(t, &forge.FakeClient{}, nil, nil)
	scopes := layer.RequiredScopes(OpUninstall)
	assert.Nil(t, scopes)
}

func TestQF_RequiredScopes_Analyze(t *testing.T) {
	layer, _ := newEnrollmentLayer(t, &forge.FakeClient{}, nil, nil)
	scopes := layer.RequiredScopes(OpAnalyze)
	assert.Equal(t, []string{"repo"}, scopes)
}

// --- Name test ---

func TestQF_Name(t *testing.T) {
	layer, _ := newEnrollmentLayer(t, &forge.FakeClient{}, nil, nil)
	assert.Equal(t, "enrollment", layer.Name())
}
