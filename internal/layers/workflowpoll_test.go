package layers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/forge"
)

func fastPollConfig() PollConfig {
	return PollConfig{
		InitialInterval: 1 * time.Millisecond,
		MaxInterval:     2 * time.Millisecond,
		Timeout:         500 * time.Millisecond,
	}
}

func TestAwaitWorkflowCompletion_Success(t *testing.T) {
	now := time.Now().UTC()
	client := &forge.FakeClient{
		WorkflowRuns: map[string]*forge.WorkflowRun{
			"org/.fullsend/repo-maintenance.yml": {
				ID:         1,
				Status:     "completed",
				Conclusion: "success",
				HTMLURL:    "https://example.com/runs/1",
				CreatedAt:  now.Add(time.Minute).Format(time.RFC3339),
			},
		},
	}

	run, err := AwaitWorkflowCompletion(
		context.Background(), client,
		"org", ".fullsend", "repo-maintenance.yml",
		now, fastPollConfig(), nil,
	)

	require.NoError(t, err)
	assert.Equal(t, 1, run.ID)
	assert.Equal(t, "success", run.Conclusion)
}

func TestAwaitWorkflowCompletion_Timeout(t *testing.T) {
	client := forge.NewFakeClient()
	now := time.Now().UTC()

	_, err := AwaitWorkflowCompletion(
		context.Background(), client,
		"org", ".fullsend", "repo-maintenance.yml",
		now, fastPollConfig(), nil,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
}

func TestAwaitWorkflowCompletion_ContextCancelled(t *testing.T) {
	client := forge.NewFakeClient()
	now := time.Now().UTC()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := AwaitWorkflowCompletion(
		ctx, client,
		"org", ".fullsend", "repo-maintenance.yml",
		now, fastPollConfig(), nil,
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestAwaitWorkflowCompletion_ProgressCallback(t *testing.T) {
	now := time.Now().UTC()
	client := &forge.FakeClient{
		WorkflowRuns: map[string]*forge.WorkflowRun{
			"org/.fullsend/ci.yml": {
				ID:        1,
				Status:    "completed",
				HTMLURL:   "https://example.com/runs/1",
				CreatedAt: now.Add(time.Minute).Format(time.RFC3339),
			},
		},
	}

	var messages []string
	run, err := AwaitWorkflowCompletion(
		context.Background(), client,
		"org", ".fullsend", "ci.yml",
		now, fastPollConfig(),
		func(msg string) { messages = append(messages, msg) },
	)

	require.NoError(t, err)
	assert.NotNil(t, run)
}

func TestNextInterval(t *testing.T) {
	assert.Equal(t, 4*time.Second, nextInterval(2*time.Second, 15*time.Second))
	assert.Equal(t, 15*time.Second, nextInterval(10*time.Second, 15*time.Second))
	assert.Equal(t, 15*time.Second, nextInterval(15*time.Second, 15*time.Second))
}
