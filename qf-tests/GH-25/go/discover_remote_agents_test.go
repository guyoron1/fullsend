//go:build e2e

package harness_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/forge"
	"github.com/fullsend-ai/fullsend/internal/harness"
)

/*
DiscoverRemoteAgents Tests

STP Reference: outputs/stp/GH-25/GH-25_test_plan.md
Jira: GH-25
*/

const (
	testOwner = "test-org"
	testRepo  = "test-config"
	testRef   = "main"
)

// dirKey builds the FakeClient DirContents lookup key.
func dirKey(owner, repo, path, ref string) string {
	return fmt.Sprintf("%s/%s/%s@%s", owner, repo, path, ref)
}

// fileRefKey builds the FakeClient FileContentsRef lookup key.
func fileRefKey(owner, repo, path, ref string) string {
	return fmt.Sprintf("%s/%s/%s@%s", owner, repo, path, ref)
}

// yamlWithRoleAndSlug returns YAML content for a harness with role and slug.
func yamlWithRoleAndSlug(role, slug string) []byte {
	return []byte(fmt.Sprintf("role: %s\nslug: %s\n", role, slug))
}

// yamlWithRoleOnly returns YAML content for a harness with only role.
func yamlWithRoleOnly(role string) []byte {
	return []byte(fmt.Sprintf("role: %s\n", role))
}

// yamlWithSlugOnly returns YAML content for a harness with only slug.
func yamlWithSlugOnly(slug string) []byte {
	return []byte(fmt.Sprintf("slug: %s\n", slug))
}

// yamlEmpty returns YAML content for a harness with neither role nor slug.
func yamlEmpty() []byte {
	return []byte("description: no identity\n")
}

func TestDiscoverRemoteAgents(t *testing.T) {
	ctx := context.Background()

	// [test_id:TS-GH-25-022] should return agents sorted by role then filename
	t.Run("[test_id:TS-GH-25-022] should return agents sorted by role then filename", func(t *testing.T) {
		fake := forge.NewFakeClient()
		fake.DirContents = map[string][]forge.DirectoryEntry{
			dirKey(testOwner, testRepo, "harness", testRef): {
				{Name: "review.yaml", Path: "harness/review.yaml", Type: "file"},
				{Name: "coder.yaml", Path: "harness/coder.yaml", Type: "file"},
				{Name: "triage.yaml", Path: "harness/triage.yaml", Type: "file"},
			},
		}
		fake.FileContentsRef = map[string][]byte{
			fileRefKey(testOwner, testRepo, "harness/review.yaml", testRef):  yamlWithRoleAndSlug("review", "review-agent"),
			fileRefKey(testOwner, testRepo, "harness/coder.yaml", testRef):   yamlWithRoleAndSlug("coder", "coder-agent"),
			fileRefKey(testOwner, testRepo, "harness/triage.yaml", testRef):  yamlWithRoleAndSlug("triage", "triage-agent"),
		}

		agents, err := harness.DiscoverRemoteAgents(ctx, fake, testOwner, testRepo, testRef)

		require.NoError(t, err)
		require.Len(t, agents, 3)
		// Should be sorted by Role: coder < review < triage
		assert.Equal(t, "coder", agents[0].Role)
		assert.Equal(t, "review", agents[1].Role)
		assert.Equal(t, "triage", agents[2].Role)
	})

	// [test_id:TS-GH-25-023] should return nil nil when no harness directory exists
	t.Run("[test_id:TS-GH-25-023] should return nil nil when no harness directory exists", func(t *testing.T) {
		fake := forge.NewFakeClient()
		// No DirContents entry → FakeClient returns ErrNotFound

		agents, err := harness.DiscoverRemoteAgents(ctx, fake, testOwner, testRepo, testRef)

		assert.Nil(t, agents, "should return nil agents when harness/ does not exist")
		assert.Nil(t, err, "should return nil error when harness/ does not exist")
	})

	// [test_id:TS-GH-25-024] should skip files without role or slug
	t.Run("[test_id:TS-GH-25-024] should skip files without role or slug", func(t *testing.T) {
		fake := forge.NewFakeClient()
		fake.DirContents = map[string][]forge.DirectoryEntry{
			dirKey(testOwner, testRepo, "harness", testRef): {
				{Name: "triage.yaml", Path: "harness/triage.yaml", Type: "file"},
				{Name: "empty.yaml", Path: "harness/empty.yaml", Type: "file"},
				{Name: "also-empty.yaml", Path: "harness/also-empty.yaml", Type: "file"},
			},
		}
		fake.FileContentsRef = map[string][]byte{
			fileRefKey(testOwner, testRepo, "harness/triage.yaml", testRef):     yamlWithRoleOnly("triage"),
			fileRefKey(testOwner, testRepo, "harness/empty.yaml", testRef):      yamlEmpty(),
			fileRefKey(testOwner, testRepo, "harness/also-empty.yaml", testRef): yamlEmpty(),
		}

		agents, err := harness.DiscoverRemoteAgents(ctx, fake, testOwner, testRepo, testRef)

		require.NoError(t, err)
		assert.Len(t, agents, 1, "only files with role or slug should be returned")
		assert.Equal(t, "triage", agents[0].Role)
	})

	// [test_id:TS-GH-25-025] should include file with role only
	t.Run("[test_id:TS-GH-25-025] should include file with role only", func(t *testing.T) {
		fake := forge.NewFakeClient()
		fake.DirContents = map[string][]forge.DirectoryEntry{
			dirKey(testOwner, testRepo, "harness", testRef): {
				{Name: "triage.yaml", Path: "harness/triage.yaml", Type: "file"},
			},
		}
		fake.FileContentsRef = map[string][]byte{
			fileRefKey(testOwner, testRepo, "harness/triage.yaml", testRef): yamlWithRoleOnly("triage"),
		}

		agents, err := harness.DiscoverRemoteAgents(ctx, fake, testOwner, testRepo, testRef)

		require.NoError(t, err)
		require.Len(t, agents, 1)
		assert.Equal(t, "triage", agents[0].Role)
		assert.Empty(t, agents[0].Slug, "slug should be empty for role-only file")
	})

	// [test_id:TS-GH-25-026] should include file with slug only
	t.Run("[test_id:TS-GH-25-026] should include file with slug only", func(t *testing.T) {
		fake := forge.NewFakeClient()
		fake.DirContents = map[string][]forge.DirectoryEntry{
			dirKey(testOwner, testRepo, "harness", testRef): {
				{Name: "my-agent.yaml", Path: "harness/my-agent.yaml", Type: "file"},
			},
		}
		fake.FileContentsRef = map[string][]byte{
			fileRefKey(testOwner, testRepo, "harness/my-agent.yaml", testRef): yamlWithSlugOnly("my-agent"),
		}

		agents, err := harness.DiscoverRemoteAgents(ctx, fake, testOwner, testRepo, testRef)

		require.NoError(t, err)
		require.Len(t, agents, 1)
		assert.Equal(t, "my-agent", agents[0].Slug)
		assert.Empty(t, agents[0].Role, "role should be empty for slug-only file")
	})

	// [test_id:TS-GH-25-027] should return multi-error with valid files on malformed YAML
	t.Run("[test_id:TS-GH-25-027] should return multi-error with valid files on malformed YAML", func(t *testing.T) {
		fake := forge.NewFakeClient()
		fake.DirContents = map[string][]forge.DirectoryEntry{
			dirKey(testOwner, testRepo, "harness", testRef): {
				{Name: "good.yaml", Path: "harness/good.yaml", Type: "file"},
				{Name: "bad.yaml", Path: "harness/bad.yaml", Type: "file"},
			},
		}
		fake.FileContentsRef = map[string][]byte{
			fileRefKey(testOwner, testRepo, "harness/good.yaml", testRef): yamlWithRoleOnly("triage"),
			fileRefKey(testOwner, testRepo, "harness/bad.yaml", testRef):  []byte(":::invalid yaml{{{"),
		}

		agents, err := harness.DiscoverRemoteAgents(ctx, fake, testOwner, testRepo, testRef)

		// Should have both a result and an error (partial success)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "bad.yaml", "error should mention the bad file")
		assert.Len(t, agents, 1, "valid files should still be returned")
		assert.Equal(t, "triage", agents[0].Role)
	})

	// [test_id:TS-GH-25-028] should return multi-error on GetFileContentAtRef failure
	t.Run("[test_id:TS-GH-25-028] should return multi-error on GetFileContentAtRef failure", func(t *testing.T) {
		fake := forge.NewFakeClient()
		fake.DirContents = map[string][]forge.DirectoryEntry{
			dirKey(testOwner, testRepo, "harness", testRef): {
				{Name: "good.yaml", Path: "harness/good.yaml", Type: "file"},
				{Name: "missing.yaml", Path: "harness/missing.yaml", Type: "file"},
			},
		}
		// Only provide content for the good file; missing.yaml will trigger ErrNotFound
		fake.FileContentsRef = map[string][]byte{
			fileRefKey(testOwner, testRepo, "harness/good.yaml", testRef): yamlWithRoleOnly("coder"),
		}

		agents, err := harness.DiscoverRemoteAgents(ctx, fake, testOwner, testRepo, testRef)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing.yaml",
			"error should mention the file that failed to fetch")
		assert.Len(t, agents, 1, "valid files should still be returned")
		assert.Equal(t, "coder", agents[0].Role)
	})

	// [test_id:TS-GH-25-029] should return empty slice for empty harness directory
	t.Run("[test_id:TS-GH-25-029] should return empty slice for empty harness directory", func(t *testing.T) {
		fake := forge.NewFakeClient()
		fake.DirContents = map[string][]forge.DirectoryEntry{
			dirKey(testOwner, testRepo, "harness", testRef): {}, // empty
		}

		agents, err := harness.DiscoverRemoteAgents(ctx, fake, testOwner, testRepo, testRef)

		require.NoError(t, err)
		assert.Empty(t, agents, "empty harness/ directory should return empty slice")
	})

	// [test_id:TS-GH-25-030] should discover .yml extension files
	t.Run("[test_id:TS-GH-25-030] should discover .yml extension files", func(t *testing.T) {
		fake := forge.NewFakeClient()
		fake.DirContents = map[string][]forge.DirectoryEntry{
			dirKey(testOwner, testRepo, "harness", testRef): {
				{Name: "agent.yml", Path: "harness/agent.yml", Type: "file"},
			},
		}
		fake.FileContentsRef = map[string][]byte{
			fileRefKey(testOwner, testRepo, "harness/agent.yml", testRef): yamlWithRoleOnly("triage"),
		}

		agents, err := harness.DiscoverRemoteAgents(ctx, fake, testOwner, testRepo, testRef)

		require.NoError(t, err)
		require.Len(t, agents, 1)
		assert.Equal(t, "triage", agents[0].Role)
		assert.Equal(t, "agent.yml", agents[0].Filename)
	})

	// [test_id:TS-GH-25-031] should skip non-YAML files
	t.Run("[test_id:TS-GH-25-031] should skip non-YAML files", func(t *testing.T) {
		fake := forge.NewFakeClient()
		fake.DirContents = map[string][]forge.DirectoryEntry{
			dirKey(testOwner, testRepo, "harness", testRef): {
				{Name: "triage.yaml", Path: "harness/triage.yaml", Type: "file"},
				{Name: "README.md", Path: "harness/README.md", Type: "file"},
				{Name: "notes.txt", Path: "harness/notes.txt", Type: "file"},
				{Name: "coder.yml", Path: "harness/coder.yml", Type: "file"},
			},
		}
		fake.FileContentsRef = map[string][]byte{
			fileRefKey(testOwner, testRepo, "harness/triage.yaml", testRef): yamlWithRoleOnly("triage"),
			fileRefKey(testOwner, testRepo, "harness/coder.yml", testRef):   yamlWithRoleOnly("coder"),
		}

		agents, err := harness.DiscoverRemoteAgents(ctx, fake, testOwner, testRepo, testRef)

		require.NoError(t, err)
		assert.Len(t, agents, 2, "only .yaml and .yml files should be processed")
	})

	// [test_id:TS-GH-25-032] should skip subdirectories in harness directory
	t.Run("[test_id:TS-GH-25-032] should skip subdirectories in harness directory", func(t *testing.T) {
		fake := forge.NewFakeClient()
		fake.DirContents = map[string][]forge.DirectoryEntry{
			dirKey(testOwner, testRepo, "harness", testRef): {
				{Name: "triage.yaml", Path: "harness/triage.yaml", Type: "file"},
				{Name: "templates", Path: "harness/templates", Type: "dir"},
				{Name: "archive", Path: "harness/archive", Type: "dir"},
			},
		}
		fake.FileContentsRef = map[string][]byte{
			fileRefKey(testOwner, testRepo, "harness/triage.yaml", testRef): yamlWithRoleOnly("triage"),
		}

		agents, err := harness.DiscoverRemoteAgents(ctx, fake, testOwner, testRepo, testRef)

		require.NoError(t, err)
		assert.Len(t, agents, 1, "only file-type entries should be processed")
	})

	// [test_id:TS-GH-25-033] should sort same role by filename for deterministic output
	t.Run("[test_id:TS-GH-25-033] should sort same role by filename for deterministic output", func(t *testing.T) {
		fake := forge.NewFakeClient()
		fake.DirContents = map[string][]forge.DirectoryEntry{
			dirKey(testOwner, testRepo, "harness", testRef): {
				{Name: "z-coder.yaml", Path: "harness/z-coder.yaml", Type: "file"},
				{Name: "a-coder.yaml", Path: "harness/a-coder.yaml", Type: "file"},
				{Name: "m-coder.yaml", Path: "harness/m-coder.yaml", Type: "file"},
			},
		}
		fake.FileContentsRef = map[string][]byte{
			fileRefKey(testOwner, testRepo, "harness/z-coder.yaml", testRef): yamlWithRoleOnly("coder"),
			fileRefKey(testOwner, testRepo, "harness/a-coder.yaml", testRef): yamlWithRoleOnly("coder"),
			fileRefKey(testOwner, testRepo, "harness/m-coder.yaml", testRef): yamlWithRoleOnly("coder"),
		}

		agents, err := harness.DiscoverRemoteAgents(ctx, fake, testOwner, testRepo, testRef)

		require.NoError(t, err)
		require.Len(t, agents, 3)
		// Same role → sorted by filename
		assert.Equal(t, "a-coder.yaml", agents[0].Filename)
		assert.Equal(t, "m-coder.yaml", agents[1].Filename)
		assert.Equal(t, "z-coder.yaml", agents[2].Filename)
	})

	// [test_id:TS-GH-25-034] should have empty Path for remote agents
	t.Run("[test_id:TS-GH-25-034] should have empty Path for remote agents", func(t *testing.T) {
		fake := forge.NewFakeClient()
		fake.DirContents = map[string][]forge.DirectoryEntry{
			dirKey(testOwner, testRepo, "harness", testRef): {
				{Name: "triage.yaml", Path: "harness/triage.yaml", Type: "file"},
			},
		}
		fake.FileContentsRef = map[string][]byte{
			fileRefKey(testOwner, testRepo, "harness/triage.yaml", testRef): yamlWithRoleOnly("triage"),
		}

		agents, err := harness.DiscoverRemoteAgents(ctx, fake, testOwner, testRepo, testRef)

		require.NoError(t, err)
		require.Len(t, agents, 1)
		assert.Empty(t, agents[0].Path, "remote agents should have empty Path (no local filesystem)")
	})

	// [test_id:TS-GH-25-035] should strip path prefix to bare filename
	t.Run("[test_id:TS-GH-25-035] should strip path prefix to bare filename", func(t *testing.T) {
		fake := forge.NewFakeClient()
		fake.DirContents = map[string][]forge.DirectoryEntry{
			dirKey(testOwner, testRepo, "harness", testRef): {
				{Name: "triage.yaml", Path: "harness/triage.yaml", Type: "file"},
			},
		}
		fake.FileContentsRef = map[string][]byte{
			fileRefKey(testOwner, testRepo, "harness/triage.yaml", testRef): yamlWithRoleOnly("triage"),
		}

		agents, err := harness.DiscoverRemoteAgents(ctx, fake, testOwner, testRepo, testRef)

		require.NoError(t, err)
		require.Len(t, agents, 1)
		assert.Equal(t, "triage.yaml", agents[0].Filename,
			"filename should be bare name without harness/ prefix")
	})

	// [test_id:TS-GH-25-036] should propagate ListDirectoryContents error
	t.Run("[test_id:TS-GH-25-036] should propagate ListDirectoryContents error", func(t *testing.T) {
		fake := forge.NewFakeClient()
		listDirErr := fmt.Errorf("internal server error")
		fake.Errors = map[string]error{
			"ListDirectoryContents": listDirErr,
		}

		agents, err := harness.DiscoverRemoteAgents(ctx, fake, testOwner, testRepo, testRef)

		require.Error(t, err)
		assert.Nil(t, agents)
		assert.Contains(t, err.Error(), "listing harness directory",
			"error should contain descriptive wrapping")
	})
}
