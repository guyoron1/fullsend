//go:build e2e

package scaffold_test

import (
	"context"
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/forge"
	"github.com/fullsend-ai/fullsend/internal/scaffold"
)

/*
ComparePathPresence Tests

STP Reference: outputs/stp/GH-25/GH-25_test_plan.md
Jira: GH-25
*/

func TestComparePathPresence(t *testing.T) {
	ctx := context.Background()
	const owner = "test-org"
	const repo = "test-repo"

	// [test_id:TS-GH-25-009] all expected paths exist
	t.Run("[test_id:TS-GH-25-009] should return nil when all expected paths exist", func(t *testing.T) {
		fake := forge.NewFakeClient()
		fake.FileContents = map[string][]byte{
			owner + "/" + repo + "/README.md":        []byte("readme"),
			owner + "/" + repo + "/.github/CODEOWNERS": []byte("* @team"),
			owner + "/" + repo + "/action.yml":       []byte("name: test"),
		}

		expected := []string{"README.md", ".github/CODEOWNERS", "action.yml"}
		missing, err := scaffold.ComparePathPresence(ctx, fake, owner, repo, expected)

		require.NoError(t, err)
		assert.Nil(t, missing, "no paths should be missing when all exist")
	})

	// [test_id:TS-GH-25-010] some expected paths are missing
	t.Run("[test_id:TS-GH-25-010] should return sorted missing paths when some are absent", func(t *testing.T) {
		fake := forge.NewFakeClient()
		fake.FileContents = map[string][]byte{
			owner + "/" + repo + "/README.md": []byte("readme"),
			// .github/CODEOWNERS and action.yml are missing
		}

		expected := []string{"README.md", "action.yml", ".github/CODEOWNERS"}
		missing, err := scaffold.ComparePathPresence(ctx, fake, owner, repo, expected)

		require.NoError(t, err)
		assert.Len(t, missing, 2)
		assert.Contains(t, missing, "action.yml")
		assert.Contains(t, missing, ".github/CODEOWNERS")
		assert.True(t, sort.StringsAreSorted(missing), "missing paths should be sorted")
	})

	// [test_id:TS-GH-25-011] all expected paths are missing
	t.Run("[test_id:TS-GH-25-011] should return all paths as missing when none exist", func(t *testing.T) {
		fake := forge.NewFakeClient()
		fake.FileContents = map[string][]byte{
			// empty — no matching paths
		}

		expected := []string{"z-file.txt", "a-file.txt", "m-file.txt"}
		missing, err := scaffold.ComparePathPresence(ctx, fake, owner, repo, expected)

		require.NoError(t, err)
		assert.Equal(t, []string{"a-file.txt", "m-file.txt", "z-file.txt"}, missing,
			"all expected paths should be reported missing in sorted order")
	})

	// [test_id:TS-GH-25-012] empty expected paths returns immediately
	t.Run("[test_id:TS-GH-25-012] should return nil nil for empty expected paths", func(t *testing.T) {
		fake := forge.NewFakeClient()
		// FakeClient should NOT be called; if it is, something is wrong
		fake.Errors = map[string]error{
			"ListRepositoryFiles": fmt.Errorf("should not be called"),
		}

		missing, err := scaffold.ComparePathPresence(ctx, fake, owner, repo, []string{})

		assert.Nil(t, missing)
		assert.Nil(t, err)
	})

	// [test_id:TS-GH-25-013] propagates ListRepositoryFiles error with context
	t.Run("[test_id:TS-GH-25-013] should propagate ListRepositoryFiles error with context", func(t *testing.T) {
		originalErr := fmt.Errorf("connection refused")
		fake := forge.NewFakeClient()
		fake.Errors = map[string]error{
			"ListRepositoryFiles": originalErr,
		}

		_, err := scaffold.ComparePathPresence(ctx, fake, owner, repo, []string{"some/path"})

		require.Error(t, err)
		assert.ErrorIs(t, err, originalErr, "original error should be in chain")
		assert.Contains(t, err.Error(), "listing repository files",
			"error should be wrapped with descriptive context")
	})

	// [test_id:TS-GH-25-014] uses batch ListRepositoryFiles not per-path GetFileContent
	t.Run("[test_id:TS-GH-25-014] should use batch ListRepositoryFiles not per-path GetFileContent", func(t *testing.T) {
		fake := forge.NewFakeClient()
		// Valid ListRepositoryFiles data
		fake.FileContents = map[string][]byte{
			owner + "/" + repo + "/file-a.go": []byte("a"),
			owner + "/" + repo + "/file-b.go": []byte("b"),
		}
		// Inject error on GetFileContent — if ComparePathPresence calls it, test fails
		fake.Errors = map[string]error{
			"GetFileContent": fmt.Errorf("FATAL: should not call GetFileContent"),
		}

		expected := []string{"file-a.go", "file-c.go"}
		missing, err := scaffold.ComparePathPresence(ctx, fake, owner, repo, expected)

		require.NoError(t, err, "should succeed using ListRepositoryFiles despite GetFileContent error")
		assert.Equal(t, []string{"file-c.go"}, missing)
	})
}
