package scaffold

/*
ComparePathPresence Edge Case Tests

STP Reference: outputs/stp/GH-2351/GH-2351_test_plan.md
Jira: GH-2351
*/

import (
	"context"
	"errors"
	"sort"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/forge"
)

func TestComparePathPresenceEdgeCases(t *testing.T) {
	ctx := context.Background()

	t.Run("[test_id:TS-GH-2351-013] should short-circuit without API calls for empty expected list", func(t *testing.T) {
		client := &forge.FakeClient{
			Errors: map[string]error{
				"ListRepositoryFiles": errors.New("should not be called"),
			},
		}

		missing, err := ComparePathPresence(ctx, client, "owner", "repo", nil)

		require.NoError(t, err, "should succeed without calling ListRepositoryFiles")
		assert.Empty(t, missing)
	})

	t.Run("[test_id:TS-GH-2351-014] should return all-missing paths in sorted order", func(t *testing.T) {
		client := &forge.FakeClient{
			FileContents: map[string][]byte{
				"owner/repo/other.txt": []byte("content"),
			},
		}

		missing, err := ComparePathPresence(ctx, client, "owner", "repo",
			[]string{"z.txt", "a.txt", "m.txt"})

		require.NoError(t, err)
		require.Len(t, missing, 3, "all expected paths should be missing")
		assert.True(t, sort.StringsAreSorted(missing), "missing paths should be sorted")
		assert.Equal(t, []string{"a.txt", "m.txt", "z.txt"}, missing)
	})

	t.Run("[test_id:TS-GH-2351-015] should handle concurrent ListRepositoryFiles calls safely", func(t *testing.T) {
		client := &forge.FakeClient{
			FileContents: map[string][]byte{
				"owner/repo/file1.txt": []byte("content1"),
				"owner/repo/file2.txt": []byte("content2"),
			},
		}

		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				paths, err := client.ListRepositoryFiles(ctx, "owner", "repo")
				require.NoError(t, err)
				assert.Len(t, paths, 2)
			}()
		}
		wg.Wait()
	})
}
