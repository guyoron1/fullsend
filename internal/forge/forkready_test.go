package forge

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAwaitForkReady_ImmediateSuccess(t *testing.T) {
	fc := NewFakeClient()
	fc.DefaultBranches = map[string]string{
		"org/repo": "main",
	}

	err := AwaitForkReady(context.Background(), fc, "org", "repo")
	require.NoError(t, err)
}

func TestAwaitForkReady_SucceedsAfterRetries(t *testing.T) {
	fc := NewFakeClient()
	// DefaultBranches starts nil → GetDefaultBranch returns ErrNotFound.

	// Use a counting wrapper to track calls and make the branch
	// available after N failures.
	var calls atomic.Int32
	readyAfter := int32(3)
	go func() {
		for {
			if calls.Load() >= readyAfter {
				fc.mu.Lock()
				if fc.DefaultBranches == nil {
					fc.DefaultBranches = make(map[string]string)
				}
				fc.DefaultBranches["org/fork-repo"] = "main"
				fc.mu.Unlock()
				return
			}
		}
	}()

	wrapper := &countingClient{Client: fc, calls: &calls}
	err := AwaitForkReady(context.Background(), wrapper, "org", "fork-repo")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, calls.Load(), readyAfter)
}

func TestAwaitForkReady_ContextCancelled(t *testing.T) {
	fc := NewFakeClient()
	// DefaultBranches is nil → GetDefaultBranch always returns ErrNotFound.

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := AwaitForkReady(ctx, fc, "org", "repo")
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

// countingClient wraps a forge.Client and counts GetDefaultBranch calls.
type countingClient struct {
	Client
	calls *atomic.Int32
}

func (c *countingClient) GetDefaultBranch(ctx context.Context, owner, repo string) (string, error) {
	c.calls.Add(1)
	return c.Client.GetDefaultBranch(ctx, owner, repo)
}

func TestFakeClient_CreateForkInOrg(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		fc := NewFakeClient()
		fc.Repos = []Repository{
			{Name: "upstream", FullName: "owner/upstream", DefaultBranch: "main"},
		}

		fork, err := fc.CreateForkInOrg(ctx, "owner", "upstream", "myorg")
		require.NoError(t, err)
		assert.Equal(t, "upstream", fork.Name)
		assert.Equal(t, "myorg/upstream", fork.FullName)
		assert.Equal(t, "main", fork.DefaultBranch)
		assert.True(t, fork.Fork)

		require.Len(t, fc.CreatedForks, 1)
		assert.Equal(t, "myorg/upstream", fc.CreatedForks[0].FullName)
	})

	t.Run("source not found", func(t *testing.T) {
		fc := NewFakeClient()

		_, err := fc.CreateForkInOrg(ctx, "owner", "missing", "myorg")
		require.Error(t, err)
		assert.True(t, IsNotFound(err))
	})

	t.Run("error injection", func(t *testing.T) {
		fc := NewFakeClient()
		fc.Errors["CreateForkInOrg"] = assert.AnError

		_, err := fc.CreateForkInOrg(ctx, "owner", "repo", "myorg")
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
	})
}

func TestFakeClient_GetDefaultBranch(t *testing.T) {
	ctx := context.Background()

	t.Run("found", func(t *testing.T) {
		fc := NewFakeClient()
		fc.DefaultBranches = map[string]string{
			"owner/repo": "main",
		}

		branch, err := fc.GetDefaultBranch(ctx, "owner", "repo")
		require.NoError(t, err)
		assert.Equal(t, "main", branch)
	})

	t.Run("not found", func(t *testing.T) {
		fc := NewFakeClient()

		_, err := fc.GetDefaultBranch(ctx, "owner", "repo")
		require.Error(t, err)
		assert.True(t, IsNotFound(err))
	})

	t.Run("error injection", func(t *testing.T) {
		fc := NewFakeClient()
		fc.Errors["GetDefaultBranch"] = assert.AnError

		_, err := fc.GetDefaultBranch(ctx, "owner", "repo")
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
	})
}
