package scaffold

/*
FakeClient.ListRepositoryFiles Tests

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

func TestFakeClientListRepositoryFiles(t *testing.T) {
	ctx := context.Background()

	t.Run("[test_id:TS-GH-2351-010] should return correct relative paths from FileContents", func(t *testing.T) {
		client := &forge.FakeClient{
			FileContents: map[string][]byte{
				"myorg/myrepo/src/main.go": []byte("package main"),
				"myorg/myrepo/README.md":   []byte("# readme"),
			},
		}

		paths, err := client.ListRepositoryFiles(ctx, "myorg", "myrepo")

		require.NoError(t, err)
		sort.Strings(paths)
		assert.Equal(t, []string{"README.md", "src/main.go"}, paths,
			"paths should have owner/repo prefix stripped")
	})

	t.Run("[test_id:TS-GH-2351-011] should return empty list for empty FileContents map", func(t *testing.T) {
		client := &forge.FakeClient{
			FileContents: map[string][]byte{},
		}

		paths, err := client.ListRepositoryFiles(ctx, "owner", "repo")

		require.NoError(t, err)
		assert.Empty(t, paths, "empty FileContents should yield nil or empty result")
	})

	t.Run("[test_id:TS-GH-2351-012] should respect error injection via Errors map", func(t *testing.T) {
		client := &forge.FakeClient{
			Errors: map[string]error{
				"ListRepositoryFiles": errors.New("injected test error"),
			},
		}

		paths, err := client.ListRepositoryFiles(ctx, "owner", "repo")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "injected test error")
		assert.Nil(t, paths)
	})
}
