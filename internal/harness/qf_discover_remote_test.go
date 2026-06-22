package harness

// QualityFlow generated tests for GH-72
// Suite: TS-GH72-005 — DiscoverRemoteAgents harness discovery via forge API
// STD: outputs/std/GH-72/GH-72_test_description.yaml

import (
	"context"
	"fmt"
	"testing"

	"github.com/fullsend-ai/fullsend/internal/forge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQFDiscoverRemoteAgents(t *testing.T) {
	ctx := context.Background()
	const (
		owner = "acme"
		repo  = ".fullsend"
		ref   = "main"
	)

	harnessKey := func() string {
		return fmt.Sprintf("%s/%s/harness@%s", owner, repo, ref)
	}
	fileKey := func(name string) string {
		return fmt.Sprintf("%s/%s/harness/%s@%s", owner, repo, name, ref)
	}

	// TC-GH72-025: Multiple harnesses discovered and sorted by role
	t.Run("multiple_harnesses_sorted_by_role", func(t *testing.T) {
		fc := forge.NewFakeClient()
		fc.DirContents[harnessKey()] = []forge.DirectoryEntry{
			{Path: "triage.yaml", Type: "file"},
			{Path: "code.yaml", Type: "file"},
			{Path: "review.yaml", Type: "file"},
		}
		fc.FileContentsRef[fileKey("triage.yaml")] = []byte("agent: agents/triage.md\nrole: triage\nslug: fs-triage\n")
		fc.FileContentsRef[fileKey("code.yaml")] = []byte("agent: agents/code.md\nrole: coder\nslug: fs-coder\n")
		fc.FileContentsRef[fileKey("review.yaml")] = []byte("agent: agents/review.md\nrole: review\nslug: fs-review\n")

		agents, err := DiscoverRemoteAgents(ctx, fc, owner, repo, ref)
		require.NoError(t, err)
		require.Len(t, agents, 3)
		assert.Equal(t, "coder", agents[0].Role)
		assert.Equal(t, "fs-coder", agents[0].Slug)
		assert.Equal(t, "code.yaml", agents[0].Filename)
		assert.Equal(t, "review", agents[1].Role)
		assert.Equal(t, "triage", agents[2].Role)
	})

	// TC-GH72-026: Missing harness directory returns nil,nil
	t.Run("no_harness_directory_returns_nil_nil", func(t *testing.T) {
		fc := forge.NewFakeClient()

		agents, err := DiscoverRemoteAgents(ctx, fc, owner, repo, ref)
		require.NoError(t, err, "not-found is not an error")
		assert.Nil(t, agents)
	})

	// TC-GH72-027: Files without role or slug are skipped
	t.Run("skips_files_without_role_or_slug", func(t *testing.T) {
		fc := forge.NewFakeClient()
		fc.DirContents[harnessKey()] = []forge.DirectoryEntry{
			{Path: "legacy.yaml", Type: "file"},
			{Path: "modern.yaml", Type: "file"},
		}
		fc.FileContentsRef[fileKey("legacy.yaml")] = []byte("agent: agents/legacy.md\n")
		fc.FileContentsRef[fileKey("modern.yaml")] = []byte("agent: agents/modern.md\nrole: triage\nslug: fs-triage\n")

		agents, err := DiscoverRemoteAgents(ctx, fc, owner, repo, ref)
		require.NoError(t, err)
		require.Len(t, agents, 1, "legacy.yaml without role/slug should be excluded")
		assert.Equal(t, "triage", agents[0].Role)
	})

	// TC-GH72-028: Malformed YAML returns partial results with multi-error
	t.Run("malformed_YAML_returns_multi-error_with_valid_files", func(t *testing.T) {
		fc := forge.NewFakeClient()
		fc.DirContents[harnessKey()] = []forge.DirectoryEntry{
			{Path: "good.yaml", Type: "file"},
			{Path: "bad.yaml", Type: "file"},
		}
		fc.FileContentsRef[fileKey("good.yaml")] = []byte("agent: agents/good.md\nrole: triage\nslug: fs-triage\n")
		fc.FileContentsRef[fileKey("bad.yaml")] = []byte(":\n  :\n    - [invalid yaml")

		agents, err := DiscoverRemoteAgents(ctx, fc, owner, repo, ref)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "bad.yaml", "error should identify the problematic file")
		require.Len(t, agents, 1, "valid files should still be returned")
		assert.Equal(t, "triage", agents[0].Role)
	})

	// TC-GH72-029: Non-YAML files and subdirectories are skipped
	t.Run("skips_subdirectories", func(t *testing.T) {
		fc := forge.NewFakeClient()
		fc.DirContents[harnessKey()] = []forge.DirectoryEntry{
			{Path: "triage.yaml", Type: "file"},
			{Path: "subdir", Type: "dir"},
		}
		fc.FileContentsRef[fileKey("triage.yaml")] = []byte("agent: agents/triage.md\nrole: triage\nslug: fs-triage\n")

		agents, err := DiscoverRemoteAgents(ctx, fc, owner, repo, ref)
		require.NoError(t, err)
		require.Len(t, agents, 1, "only YAML files should be processed; subdirectory ignored")
	})

	// TC-GH72-030: ListDirectoryContents error propagates
	t.Run("ListDirectoryContents_error_propagates", func(t *testing.T) {
		fc := forge.NewFakeClient()
		fc.Errors["ListDirectoryContents"] = fmt.Errorf("network error")

		agents, err := DiscoverRemoteAgents(ctx, fc, owner, repo, ref)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "listing harness directory")
		assert.Nil(t, agents)
	})

	// TC-GH72-031: Same role sorted by filename
	t.Run("same_role_sorted_by_filename", func(t *testing.T) {
		fc := forge.NewFakeClient()
		fc.DirContents[harnessKey()] = []forge.DirectoryEntry{
			{Path: "fix.yaml", Type: "file"},
			{Path: "code.yaml", Type: "file"},
		}
		fc.FileContentsRef[fileKey("fix.yaml")] = []byte("agent: agents/fix.md\nrole: coder\nslug: fs-coder\n")
		fc.FileContentsRef[fileKey("code.yaml")] = []byte("agent: agents/code.md\nrole: coder\nslug: fs-coder-2\n")

		agents, err := DiscoverRemoteAgents(ctx, fc, owner, repo, ref)
		require.NoError(t, err)
		require.Len(t, agents, 2)
		assert.Equal(t, "code.yaml", agents[0].Filename, "code.yaml should sort before fix.yaml")
		assert.Equal(t, "fix.yaml", agents[1].Filename)
	})

	// TC-GH72-032: Role-only file (no slug) is included
	t.Run("role_only_without_slug_is_included", func(t *testing.T) {
		fc := forge.NewFakeClient()
		fc.DirContents[harnessKey()] = []forge.DirectoryEntry{
			{Path: "partial.yaml", Type: "file"},
		}
		fc.FileContentsRef[fileKey("partial.yaml")] = []byte("agent: agents/partial.md\nrole: triage\n")

		agents, err := DiscoverRemoteAgents(ctx, fc, owner, repo, ref)
		require.NoError(t, err)
		require.Len(t, agents, 1)
		assert.Equal(t, "triage", agents[0].Role)
		assert.Empty(t, agents[0].Slug, "slug should be empty when not set")
	})

	// TC-GH72-033: Slug-only file (no role) is included
	t.Run("slug_only_without_role_is_included", func(t *testing.T) {
		fc := forge.NewFakeClient()
		fc.DirContents[harnessKey()] = []forge.DirectoryEntry{
			{Path: "slug-only.yaml", Type: "file"},
		}
		fc.FileContentsRef[fileKey("slug-only.yaml")] = []byte("agent: agents/slug.md\nslug: fs-triage\n")

		agents, err := DiscoverRemoteAgents(ctx, fc, owner, repo, ref)
		require.NoError(t, err)
		require.Len(t, agents, 1)
		assert.Equal(t, "fs-triage", agents[0].Slug)
		assert.Empty(t, agents[0].Role, "role should be empty when not set")
	})

	// TC-GH72-034: .yml extension files are discovered
	t.Run("yml_extension_is_discovered", func(t *testing.T) {
		fc := forge.NewFakeClient()
		fc.DirContents[harnessKey()] = []forge.DirectoryEntry{
			{Path: "agent.yml", Type: "file"},
		}
		fc.FileContentsRef[fileKey("agent.yml")] = []byte("agent: agents/agent.md\nrole: triage\nslug: fs-triage\n")

		agents, err := DiscoverRemoteAgents(ctx, fc, owner, repo, ref)
		require.NoError(t, err)
		require.Len(t, agents, 1)
		assert.Equal(t, "agent.yml", agents[0].Filename)
	})

	// TC-GH72-035: Empty harness directory returns empty list
	t.Run("empty_harness_directory_returns_empty_list", func(t *testing.T) {
		fc := forge.NewFakeClient()
		fc.DirContents[harnessKey()] = []forge.DirectoryEntry{}

		agents, err := DiscoverRemoteAgents(ctx, fc, owner, repo, ref)
		require.NoError(t, err)
		assert.Empty(t, agents, "empty directory should return empty but not nil")
	})

	// TC-GH72-036: Path field is empty for remote agents
	t.Run("path_field_is_empty_for_remote_agents", func(t *testing.T) {
		fc := forge.NewFakeClient()
		fc.DirContents[harnessKey()] = []forge.DirectoryEntry{
			{Path: "triage.yaml", Type: "file"},
		}
		fc.FileContentsRef[fileKey("triage.yaml")] = []byte("agent: agents/triage.md\nrole: triage\nslug: fs-triage\n")

		agents, err := DiscoverRemoteAgents(ctx, fc, owner, repo, ref)
		require.NoError(t, err)
		require.Len(t, agents, 1)
		assert.Empty(t, agents[0].Path, "Path should be empty for remotely discovered agents")
	})
}
