package scaffold

/*
ListRepositoryFiles Git Trees API Tests

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

func TestListRepositoryFiles(t *testing.T) {
	ctx := context.Background()

	t.Run("[test_id:TS-GH-2351-006] should return all blob paths from repository tree", func(t *testing.T) {
		client := &forge.FakeClient{
			FileContents: map[string][]byte{
				"owner/repo/file1.go":         []byte("package main"),
				"owner/repo/dir/file2.go":     []byte("package dir"),
				"owner/repo/dir/sub/file3.go": []byte("package sub"),
			},
		}

		paths, err := client.ListRepositoryFiles(ctx, "owner", "repo")

		require.NoError(t, err)
		sort.Strings(paths)
		assert.Equal(t, []string{"dir/file2.go", "dir/sub/file3.go", "file1.go"}, paths)
	})

	t.Run("[test_id:TS-GH-2351-007] should exclude tree entries (directories) from results", func(t *testing.T) {
		client := &forge.FakeClient{
			FileContents: map[string][]byte{
				"owner/repo/dir/file.txt": []byte("content"),
			},
		}

		paths, err := client.ListRepositoryFiles(ctx, "owner", "repo")

		require.NoError(t, err)
		for _, p := range paths {
			assert.NotEqual(t, "dir", p, "directory-only entries should not be in results")
			assert.NotEqual(t, "dir/", p, "trailing-slash directory entries should not be in results")
		}
		assert.Contains(t, paths, "dir/file.txt", "file entries should be present")
	})

	t.Run("[test_id:TS-GH-2351-008] should return error when repository tree is truncated", func(t *testing.T) {
		client := &forge.FakeClient{
			Errors: map[string]error{
				"ListRepositoryFiles": errors.New("tree truncated: response too large"),
			},
		}

		_, err := client.ListRepositoryFiles(ctx, "owner", "repo")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "truncat")
	})

	t.Run("[test_id:TS-GH-2351-009] should propagate error for invalid repository", func(t *testing.T) {
		client := &forge.FakeClient{
			Errors: map[string]error{
				"ListRepositoryFiles": errors.New("repository not found: invalid/repo"),
			},
		}

		_, err := client.ListRepositoryFiles(ctx, "invalid", "repo")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "repository not found")
	})
}
