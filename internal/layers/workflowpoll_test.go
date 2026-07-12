package layers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/forge"
)

func TestNextBackoff(t *testing.T) {
	tests := []struct {
		name    string
		current time.Duration
		max     time.Duration
		want    time.Duration
	}{
		{
			name:    "doubles interval",
			current: 2 * time.Second,
			max:     0,
			want:    4 * time.Second,
		},
		{
			name:    "caps at max",
			current: 10 * time.Second,
			max:     15 * time.Second,
			want:    15 * time.Second,
		},
		{
			name:    "already at max",
			current: 15 * time.Second,
			max:     15 * time.Second,
			want:    15 * time.Second,
		},
		{
			name:    "no cap when max is zero",
			current: 1 * time.Minute,
			max:     0,
			want:    2 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nextBackoff(tt.current, tt.max)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAwaitWorkflowCompletion_CompletedRun(t *testing.T) {
	now := time.Now().UTC()
	client := &forge.FakeClient{
		WorkflowRuns: map[string]*forge.WorkflowRun{
			"org/repo/workflow.yml": {
				ID:         1,
				Status:     "completed",
				Conclusion: "success",
				CreatedAt:  now.Add(time.Minute).Format(time.RFC3339),
				HTMLURL:    "https://example.com/runs/1",
			},
		},
	}

	cfg := WorkflowPollConfig{
		InitialInterval: 1 * time.Millisecond,
		MaxInterval:     10 * time.Millisecond,
		MaxAttempts:     5,
	}

	run, err := AwaitWorkflowCompletion(
		context.Background(), client, "org", "repo", "workflow.yml",
		now, cfg, nil,
	)
	require.NoError(t, err)
	require.NotNil(t, run)
	assert.Equal(t, 1, run.ID)
	assert.Equal(t, "success", run.Conclusion)
}

func TestAwaitWorkflowCompletion_Timeout(t *testing.T) {
	client := &forge.FakeClient{}

	cfg := WorkflowPollConfig{
		InitialInterval: 1 * time.Millisecond,
		MaxInterval:     10 * time.Millisecond,
		MaxAttempts:     3,
	}

	run, err := AwaitWorkflowCompletion(
		context.Background(), client, "org", "repo", "workflow.yml",
		time.Now().UTC(), cfg, nil,
	)
	assert.Nil(t, run)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
}

func TestAwaitWorkflowCompletion_ContextCancelled(t *testing.T) {
	client := &forge.FakeClient{}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cfg := WorkflowPollConfig{
		InitialInterval: 1 * time.Millisecond,
		MaxInterval:     10 * time.Millisecond,
		MaxAttempts:     5,
	}

	run, err := AwaitWorkflowCompletion(ctx, client, "org", "repo", "workflow.yml",
		time.Now().UTC(), cfg, nil,
	)
	assert.Nil(t, run)
	require.ErrorIs(t, err, context.Canceled)
}

func TestAwaitWorkflowCompletion_LogsProgress(t *testing.T) {
	now := time.Now().UTC()
	client := &forge.FakeClient{
		WorkflowRuns: map[string]*forge.WorkflowRun{
			"org/repo/workflow.yml": {
				ID:        1,
				Status:    "in_progress",
				CreatedAt: now.Add(time.Minute).Format(time.RFC3339),
				HTMLURL:   "https://example.com/runs/1",
			},
		},
	}

	cfg := WorkflowPollConfig{
		InitialInterval: 1 * time.Millisecond,
		MaxInterval:     10 * time.Millisecond,
		MaxAttempts:     2,
	}

	var msgs []string
	_, _ = AwaitWorkflowCompletion(
		context.Background(), client, "org", "repo", "workflow.yml",
		now, cfg, func(msg string) { msgs = append(msgs, msg) },
	)
	assert.NotEmpty(t, msgs)
	assert.Contains(t, msgs[0], "in_progress")
}

func TestAwaitWorkflowCompletion_IgnoresOlderRuns(t *testing.T) {
	now := time.Now().UTC()
	client := &forge.FakeClient{
		WorkflowRuns: map[string]*forge.WorkflowRun{
			"org/repo/workflow.yml": {
				ID:         1,
				Status:     "completed",
				Conclusion: "success",
				CreatedAt:  now.Add(-time.Hour).Format(time.RFC3339), // before dispatch
				HTMLURL:    "https://example.com/runs/1",
			},
		},
	}

	cfg := WorkflowPollConfig{
		InitialInterval: 1 * time.Millisecond,
		MaxInterval:     10 * time.Millisecond,
		MaxAttempts:     2,
	}

	run, err := AwaitWorkflowCompletion(
		context.Background(), client, "org", "repo", "workflow.yml",
		now, cfg, nil,
	)
	assert.Nil(t, run)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
}
