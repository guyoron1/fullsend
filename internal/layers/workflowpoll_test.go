package layers

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/forge"
)

// errorCountClient wraps FakeClient, returning transient errors on the first
// N calls to ListWorkflowRuns before delegating to the embedded fake.
type errorCountClient struct {
	*forge.FakeClient
	errorsLeft int
	mu         sync.Mutex
}

func (c *errorCountClient) ListWorkflowRuns(ctx context.Context, owner, repo, wf string) ([]forge.WorkflowRun, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.errorsLeft > 0 {
		c.errorsLeft--
		return nil, fmt.Errorf("transient API error")
	}
	return c.FakeClient.ListWorkflowRuns(ctx, owner, repo, wf)
}

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

func TestAwaitWorkflowCompletion_InProgressRunTimesOut(t *testing.T) {
	now := time.Now().UTC()
	client := &forge.FakeClient{
		WorkflowRuns: map[string]*forge.WorkflowRun{
			"org/.fullsend/ci.yml": {
				ID:        1,
				Status:    "in_progress",
				HTMLURL:   "https://example.com/runs/1",
				CreatedAt: now.Add(time.Minute).Format(time.RFC3339),
			},
		},
	}

	var messages []string
	_, err := AwaitWorkflowCompletion(
		context.Background(), client,
		"org", ".fullsend", "ci.yml",
		now, fastPollConfig(),
		func(msg string) { messages = append(messages, msg) },
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
	// Should have received at least one in-progress status message.
	require.NotEmpty(t, messages)
	assert.Contains(t, messages[0], "in_progress")
}

func TestAwaitWorkflowCompletion_NoMatchingRunByDispatchTime(t *testing.T) {
	now := time.Now().UTC()
	// Run was created BEFORE dispatchTime — should not match.
	client := &forge.FakeClient{
		WorkflowRuns: map[string]*forge.WorkflowRun{
			"org/.fullsend/ci.yml": {
				ID:        1,
				Status:    "completed",
				HTMLURL:   "https://example.com/runs/1",
				CreatedAt: now.Add(-time.Hour).Format(time.RFC3339),
			},
		},
	}

	_, err := AwaitWorkflowCompletion(
		context.Background(), client,
		"org", ".fullsend", "ci.yml",
		now, fastPollConfig(), nil,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
}

func TestAwaitWorkflowCompletion_ErrorThenSuccess(t *testing.T) {
	now := time.Now().UTC()
	inner := &forge.FakeClient{
		WorkflowRuns: map[string]*forge.WorkflowRun{
			"org/.fullsend/ci.yml": {
				ID:         1,
				Status:     "completed",
				Conclusion: "success",
				HTMLURL:    "https://example.com/runs/1",
				CreatedAt:  now.Add(time.Minute).Format(time.RFC3339),
			},
		},
	}
	client := &errorCountClient{FakeClient: inner, errorsLeft: 2}

	var messages []string
	run, err := AwaitWorkflowCompletion(
		context.Background(), client,
		"org", ".fullsend", "ci.yml",
		now, fastPollConfig(),
		func(msg string) { messages = append(messages, msg) },
	)

	require.NoError(t, err)
	assert.Equal(t, 1, run.ID)
	// The 2 transient errors should have produced progress messages.
	assert.GreaterOrEqual(t, len(messages), 2)
}

func TestNextInterval(t *testing.T) {
	assert.Equal(t, 4*time.Second, nextInterval(2*time.Second, 15*time.Second))
	assert.Equal(t, 15*time.Second, nextInterval(10*time.Second, 15*time.Second))
	assert.Equal(t, 15*time.Second, nextInterval(15*time.Second, 15*time.Second))
}
