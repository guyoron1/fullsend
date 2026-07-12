package layers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/forge"
)

// fastPollConfig returns a PollConfig suitable for unit tests: sub-millisecond
// intervals so tests complete quickly.
func fastPollConfig() PollConfig {
	return PollConfig{
		InitialInterval: 1 * time.Millisecond,
		MaxInterval:     1 * time.Millisecond,
		Timeout:         50 * time.Millisecond,
	}
}

func TestAwaitWorkflowCompletion_Success(t *testing.T) {
	client := forge.NewFakeClient()
	dispatchTime := time.Now().UTC().Add(-10 * time.Second)
	client.WorkflowRuns["test-org/.fullsend/repo-maintenance.yml"] = &forge.WorkflowRun{
		ID:         1,
		Status:     "completed",
		Conclusion: "success",
		HTMLURL:    "https://github.com/test-org/.fullsend/actions/runs/1",
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}

	run, err := AwaitWorkflowCompletion(
		context.Background(), client,
		"test-org", forge.ConfigRepoName, "repo-maintenance.yml",
		dispatchTime, fastPollConfig(), nil,
	)

	require.NoError(t, err)
	assert.Equal(t, 1, run.ID)
	assert.Equal(t, "success", run.Conclusion)
}

func TestAwaitWorkflowCompletion_Timeout(t *testing.T) {
	client := forge.NewFakeClient()
	dispatchTime := time.Now().UTC().Add(-10 * time.Second)
	// No workflow runs configured — will timeout.

	_, err := AwaitWorkflowCompletion(
		context.Background(), client,
		"test-org", forge.ConfigRepoName, "repo-maintenance.yml",
		dispatchTime, fastPollConfig(), nil,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
}

func TestAwaitWorkflowCompletion_ContextCancelled(t *testing.T) {
	client := forge.NewFakeClient()
	dispatchTime := time.Now().UTC().Add(-10 * time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := AwaitWorkflowCompletion(
		ctx, client,
		"test-org", forge.ConfigRepoName, "repo-maintenance.yml",
		dispatchTime, fastPollConfig(), nil,
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestAwaitWorkflowCompletion_ProgressCallback(t *testing.T) {
	client := forge.NewFakeClient()
	dispatchTime := time.Now().UTC().Add(-10 * time.Second)
	client.WorkflowRuns["test-org/.fullsend/repo-maintenance.yml"] = &forge.WorkflowRun{
		ID:        1,
		Status:    "in_progress",
		HTMLURL:   "https://github.com/test-org/.fullsend/actions/runs/1",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	var messages []string
	_, _ = AwaitWorkflowCompletion(
		context.Background(), client,
		"test-org", forge.ConfigRepoName, "repo-maintenance.yml",
		dispatchTime, fastPollConfig(),
		func(msg string) { messages = append(messages, msg) },
	)

	// Should have received at least one progress message about in_progress run.
	require.NotEmpty(t, messages)
	assert.Contains(t, messages[0], "in_progress")
}

func TestAwaitWorkflowCompletion_SkipsOldRuns(t *testing.T) {
	client := forge.NewFakeClient()
	dispatchTime := time.Now().UTC()
	// Run created before dispatch time — should be skipped.
	client.WorkflowRuns["test-org/.fullsend/repo-maintenance.yml"] = &forge.WorkflowRun{
		ID:         1,
		Status:     "completed",
		Conclusion: "success",
		HTMLURL:    "https://github.com/test-org/.fullsend/actions/runs/1",
		CreatedAt:  dispatchTime.Add(-1 * time.Minute).Format(time.RFC3339),
	}

	_, err := AwaitWorkflowCompletion(
		context.Background(), client,
		"test-org", forge.ConfigRepoName, "repo-maintenance.yml",
		dispatchTime, fastPollConfig(), nil,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
}

func TestAwaitWorkflowCompletion_RejectsZeroInterval(t *testing.T) {
	client := forge.NewFakeClient()
	dispatchTime := time.Now().UTC()

	_, err := AwaitWorkflowCompletion(
		context.Background(), client,
		"test-org", forge.ConfigRepoName, "repo-maintenance.yml",
		dispatchTime,
		PollConfig{InitialInterval: 0, MaxInterval: time.Millisecond, Timeout: time.Millisecond},
		nil,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "InitialInterval must be positive")
}

func TestAwaitWorkflowCompletion_RejectsZeroTimeout(t *testing.T) {
	client := forge.NewFakeClient()
	dispatchTime := time.Now().UTC()

	_, err := AwaitWorkflowCompletion(
		context.Background(), client,
		"test-org", forge.ConfigRepoName, "repo-maintenance.yml",
		dispatchTime,
		PollConfig{InitialInterval: time.Millisecond, MaxInterval: time.Millisecond, Timeout: 0},
		nil,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Timeout must be positive")
}

func TestNextInterval(t *testing.T) {
	assert.Equal(t, 4*time.Second, nextInterval(2*time.Second, 15*time.Second))
	assert.Equal(t, 8*time.Second, nextInterval(4*time.Second, 15*time.Second))
	assert.Equal(t, 15*time.Second, nextInterval(8*time.Second, 15*time.Second))
	assert.Equal(t, 15*time.Second, nextInterval(15*time.Second, 15*time.Second))
}

func TestDefaultPollConfig(t *testing.T) {
	cfg := DefaultPollConfig()
	assert.Equal(t, 2*time.Second, cfg.InitialInterval)
	assert.Equal(t, 15*time.Second, cfg.MaxInterval)
	assert.Equal(t, 3*time.Minute, cfg.Timeout)
}
