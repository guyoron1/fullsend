package scaffold

/*
ComparePathPresence Batch API Tests

STP Reference: outputs/stp/GH-2351/GH-2351_test_plan.md
Jira: GH-2351
*/

import (
	"context"
	"errors"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/forge"
)

func TestComparePathPresence(t *testing.T) {
	ctx := context.Background()

	t.Run("[test_id:TS-GH-2351-001] should return correct missing paths", func(t *testing.T) {
		client := &forge.FakeClient{
			FileContents: map[string][]byte{
				"owner/repo/path/a.txt": []byte("content-a"),
				"owner/repo/path/b.txt": []byte("content-b"),
			},
		}

		missing, err := ComparePathPresence(ctx, client, "owner", "repo",
			[]string{"path/a.txt", "path/b.txt", "path/c.txt"})

		require.NoError(t, err)
		assert.Equal(t, []string{"path/c.txt"}, missing)
	})

	t.Run("[test_id:TS-GH-2351-002] should report all paths present when all exist", func(t *testing.T) {
		client := &forge.FakeClient{
			FileContents: map[string][]byte{
				"owner/repo/path/a.txt": []byte("content-a"),
				"owner/repo/path/b.txt": []byte("content-b"),
			},
		}

		missing, err := ComparePathPresence(ctx, client, "owner", "repo",
			[]string{"path/a.txt", "path/b.txt"})

		require.NoError(t, err)
		assert.Empty(t, missing)
	})

	t.Run("[test_id:TS-GH-2351-003] should return sorted missing paths when some absent", func(t *testing.T) {
		client := &forge.FakeClient{
			FileContents: map[string][]byte{
				"owner/repo/path/b.txt": []byte("content-b"),
			},
		}

		missing, err := ComparePathPresence(ctx, client, "owner", "repo",
			[]string{"path/c.txt", "path/a.txt", "path/b.txt"})

		require.NoError(t, err)
		require.Len(t, missing, 2)
		assert.True(t, sort.StringsAreSorted(missing), "missing paths should be sorted")
		assert.Equal(t, []string{"path/a.txt", "path/c.txt"}, missing)
	})

	t.Run("[test_id:TS-GH-2351-004] should never call GetFileContent (batch regression guard)", func(t *testing.T) {
		client := &forge.FakeClient{
			FileContents: map[string][]byte{
				"owner/repo/path/a.txt": []byte("content-a"),
			},
			Errors: map[string]error{
				"GetFileContent": errors.New("GetFileContent must not be called"),
			},
		}

		missing, err := ComparePathPresence(ctx, client, "owner", "repo",
			[]string{"path/a.txt"})

		require.NoError(t, err, "should succeed because GetFileContent was never called")
		assert.Empty(t, missing)
	})

	t.Run("[test_id:TS-GH-2351-005] should propagate error from ListRepositoryFiles failure", func(t *testing.T) {
		client := &forge.FakeClient{
			Errors: map[string]error{
				"ListRepositoryFiles": errors.New("API rate limit exceeded"),
			},
		}

		_, err := ComparePathPresence(ctx, client, "owner", "repo",
			[]string{"path/a.txt"})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "API rate limit exceeded")
	})
}
