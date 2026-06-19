package harness_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/forge"
	"github.com/fullsend-ai/fullsend/internal/harness"
)

// =============================================================================
// Test Helpers
// =============================================================================

// newFakeForgeClient creates a forge.FakeClient configured with the given
// directory entries and file contents. dirEntries maps filenames to their
// type ("file" or "dir"). fileContents maps filenames to their raw YAML content.
// fetchErrors maps filenames to errors returned when fetching that file.
func newFakeForgeClient(
	dirEntries map[string]string,
	fileContents map[string]string,
	fetchErrors map[string]error,
	listErr error,
) *forge.FakeClient {
	return forge.NewFakeClient(forge.FakeClientConfig{
		DirEntries:   dirEntries,
		FileContents: fileContents,
		FetchErrors:  fetchErrors,
		ListErr:      listErr,
	})
}

const (
	testRepo = "my-org/my-project"
	testDir  = "harness/agents"
)

// =============================================================================
// TS-GH-42-001: Remote agent discovery with correct identity fields (P0)
// =============================================================================

func TestDiscoverRemoteAgents_CorrectIdentity(t *testing.T) {
	tests := []struct {
		name             string
		filename         string
		yamlContent      string
		expectedRole     string
		expectedSlug     string
		expectedFilename string
	}{
		{
			name:     "builder agent with role and slug",
			filename: "builder.yaml",
			yamlContent: `role: "builder"
slug: "builder-agent"
base: "default"
`,
			expectedRole:     "builder",
			expectedSlug:     "builder-agent",
			expectedFilename: "builder.yaml",
		},
		{
			name:     "reviewer agent with role and slug",
			filename: "reviewer.yaml",
			yamlContent: `role: "reviewer"
slug: "reviewer-agent"
base: "default"
`,
			expectedRole:     "reviewer",
			expectedSlug:     "reviewer-agent",
			expectedFilename: "reviewer.yaml",
		},
		{
			name:     "deployer agent with yml extension",
			filename: "deployer.yml",
			yamlContent: `role: "deployer"
slug: "deploy-agent"
base: "default"
`,
			expectedRole:     "deployer",
			expectedSlug:     "deploy-agent",
			expectedFilename: "deployer.yml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			fakeClient := newFakeForgeClient(
				map[string]string{tt.filename: "file"},
				map[string]string{tt.filename: tt.yamlContent},
				nil,
				nil,
			)

			agents, err := harness.DiscoverRemoteAgents(ctx, fakeClient, testRepo, testDir)

			require.NoError(t, err)
			require.Len(t, agents, 1)
			assert.Equal(t, tt.expectedRole, agents[0].Role, "Role should match source YAML")
			assert.Equal(t, tt.expectedSlug, agents[0].Slug, "Slug should match source YAML")
			assert.Equal(t, tt.expectedFilename, agents[0].Filename, "Filename should match directory entry")
		})
	}
}

// =============================================================================
// TS-GH-42-002: Sort order verification (P0)
// =============================================================================

func TestDiscoverRemoteAgents_SortOrder(t *testing.T) {
	ctx := context.Background()
	fakeClient := newFakeForgeClient(
		map[string]string{
			"zebra.yaml":   "file",
			"alpha.yaml":   "file",
			"alpha-2.yaml": "file",
		},
		map[string]string{
			"zebra.yaml":   "role: \"zebra\"\nslug: \"z-agent\"\n",
			"alpha.yaml":   "role: \"alpha\"\nslug: \"a-agent\"\n",
			"alpha-2.yaml": "role: \"alpha\"\nslug: \"a2-agent\"\n",
		},
		nil,
		nil,
	)

	agents, err := harness.DiscoverRemoteAgents(ctx, fakeClient, testRepo, testDir)

	require.NoError(t, err)
	require.Len(t, agents, 3)

	// Primary sort by Role ascending
	assert.Equal(t, "alpha", agents[0].Role, "First agent should have lowest role")
	assert.Equal(t, "alpha", agents[1].Role, "Second agent should also be alpha")
	assert.Equal(t, "zebra", agents[2].Role, "Third agent should be zebra")

	// Secondary sort by Filename ascending for same role
	assert.Equal(t, "alpha-2.yaml", agents[0].Filename, "Within same role, sorted by filename ascending")
	assert.Equal(t, "alpha.yaml", agents[1].Filename, "Within same role, sorted by filename ascending")
}

// =============================================================================
// TS-GH-42-003: Invalid YAML error handling (P0)
// =============================================================================

func TestDiscoverRemoteAgents_InvalidYAML(t *testing.T) {
	ctx := context.Background()
	invalidFilename := "bad-agent.yaml"
	fakeClient := newFakeForgeClient(
		map[string]string{invalidFilename: "file"},
		map[string]string{invalidFilename: "role: \"valid\"\nslug: [invalid yaml {{{{\n"},
		nil,
		nil,
	)

	_, err := harness.DiscoverRemoteAgents(ctx, fakeClient, testRepo, testDir)

	require.Error(t, err, "Error should be returned for invalid YAML")
	assert.Contains(t, err.Error(), invalidFilename,
		"Error message should reference the failing file for debugging")
}

// =============================================================================
// TS-GH-42-004: Missing directory handling (P0)
// =============================================================================

func TestDiscoverRemoteAgents_MissingDirectory(t *testing.T) {
	ctx := context.Background()
	fakeClient := newFakeForgeClient(
		nil, // no directory entries — simulates not-found
		nil,
		nil,
		nil,
	)

	agents, err := harness.DiscoverRemoteAgents(ctx, fakeClient, testRepo, "nonexistent/dir")

	assert.Nil(t, agents, "Agents should be nil for missing directory")
	assert.NoError(t, err, "No error for missing directory — graceful handling")
}

// =============================================================================
// TS-GH-42-005: Directory listing error propagation (P0)
// =============================================================================

func TestDiscoverRemoteAgents_DirectoryListingError(t *testing.T) {
	ctx := context.Background()
	originalErr := fmt.Errorf("forge API rate limited")
	fakeClient := newFakeForgeClient(
		nil,
		nil,
		nil,
		originalErr, // error on directory listing
	)

	_, err := harness.DiscoverRemoteAgents(ctx, fakeClient, testRepo, testDir)

	require.Error(t, err, "Error should be propagated for directory listing failure")
	assert.ErrorIs(t, err, originalErr, "Original error should be preserved in error chain")
}

// =============================================================================
// TS-GH-42-006: YAML extension filter (P1)
// =============================================================================

func TestDiscoverRemoteAgents_YAMLExtensionFilter(t *testing.T) {
	tests := []struct {
		name          string
		dirEntries    map[string]string
		fileContents  map[string]string
		expectedCount int
	}{
		{
			name: "mixed file types — only .yaml and .yml processed",
			dirEntries: map[string]string{
				"agent-a.yaml": "file",
				"agent-b.yml":  "file",
				"readme.md":    "file",
				"config.json":  "file",
				"notes.txt":    "file",
			},
			fileContents: map[string]string{
				"agent-a.yaml": "role: \"a\"\nslug: \"a-agent\"\n",
				"agent-b.yml":  "role: \"b\"\nslug: \"b-agent\"\n",
			},
			expectedCount: 2,
		},
		{
			name: "only .yaml files",
			dirEntries: map[string]string{
				"agent.yaml": "file",
			},
			fileContents: map[string]string{
				"agent.yaml": "role: \"solo\"\nslug: \"solo-agent\"\n",
			},
			expectedCount: 1,
		},
		{
			name: "only .yml files",
			dirEntries: map[string]string{
				"agent.yml": "file",
			},
			fileContents: map[string]string{
				"agent.yml": "role: \"solo\"\nslug: \"solo-agent\"\n",
			},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			fakeClient := newFakeForgeClient(tt.dirEntries, tt.fileContents, nil, nil)

			agents, err := harness.DiscoverRemoteAgents(ctx, fakeClient, testRepo, testDir)

			require.NoError(t, err)
			assert.Len(t, agents, tt.expectedCount, "Only .yaml and .yml files should be processed")
		})
	}
}

// =============================================================================
// TS-GH-42-007: Skip subdirectories (P1)
// =============================================================================

func TestDiscoverRemoteAgents_SkipSubdirectories(t *testing.T) {
	ctx := context.Background()
	fakeClient := newFakeForgeClient(
		map[string]string{
			"agent.yaml": "file",
			"subdir":     "dir",
		},
		map[string]string{
			"agent.yaml": "role: \"agent\"\nslug: \"agent-1\"\n",
		},
		nil,
		nil,
	)

	agents, err := harness.DiscoverRemoteAgents(ctx, fakeClient, testRepo, testDir)

	require.NoError(t, err)
	assert.Len(t, agents, 1, "Only file entries should be processed, not directories")
	assert.Equal(t, "agent", agents[0].Role)
}

// =============================================================================
// TS-GH-42-008: Skip non-YAML files (P1)
// =============================================================================

func TestDiscoverRemoteAgents_SkipNonYAML(t *testing.T) {
	ctx := context.Background()
	fakeClient := newFakeForgeClient(
		map[string]string{
			"config.json": "file",
			"notes.txt":   "file",
		},
		nil, // no YAML content to fetch
		nil,
		nil,
	)

	agents, err := harness.DiscoverRemoteAgents(ctx, fakeClient, testRepo, testDir)

	assert.NoError(t, err, "No error for non-YAML files")
	assert.Empty(t, agents, "No agents from non-YAML files")
}

// =============================================================================
// TS-GH-42-009: Skip empty role+slug agents (P1)
// =============================================================================

func TestDiscoverRemoteAgents_SkipEmptyRoleSlug(t *testing.T) {
	ctx := context.Background()
	fakeClient := newFakeForgeClient(
		map[string]string{
			"empty-identity.yaml": "file",
		},
		map[string]string{
			"empty-identity.yaml": "role: \"\"\nslug: \"\"\nbase: \"default\"\n",
		},
		nil,
		nil,
	)

	agents, err := harness.DiscoverRemoteAgents(ctx, fakeClient, testRepo, testDir)

	assert.NoError(t, err)
	assert.Empty(t, agents, "Agents with empty role and slug should be excluded")
}

// =============================================================================
// TS-GH-42-010: Partial failure — valid agents + aggregated errors (P1)
// =============================================================================

func TestDiscoverRemoteAgents_PartialFailure(t *testing.T) {
	ctx := context.Background()
	fakeClient := newFakeForgeClient(
		map[string]string{
			"valid-1.yaml": "file",
			"invalid.yaml": "file",
			"valid-2.yaml": "file",
		},
		map[string]string{
			"valid-1.yaml": "role: \"agent-a\"\nslug: \"a\"\n",
			"invalid.yaml": "{{invalid yaml",
			"valid-2.yaml": "role: \"agent-b\"\nslug: \"b\"\n",
		},
		nil,
		nil,
	)

	agents, err := harness.DiscoverRemoteAgents(ctx, fakeClient, testRepo, testDir)

	assert.Len(t, agents, 2, "Valid agents should be returned despite errors in other files")
	assert.Error(t, err, "Error should be returned for malformed files")
}

// =============================================================================
// TS-GH-42-011: Single file fetch failure isolation (P1)
// =============================================================================

func TestDiscoverRemoteAgents_SingleFileFetchFailure(t *testing.T) {
	ctx := context.Background()
	fakeClient := newFakeForgeClient(
		map[string]string{
			"good.yaml": "file",
			"bad.yaml":  "file",
		},
		map[string]string{
			"good.yaml": "role: \"good\"\nslug: \"good-agent\"\n",
		},
		map[string]error{
			"bad.yaml": fmt.Errorf("network timeout"),
		},
		nil,
	)

	agents, err := harness.DiscoverRemoteAgents(ctx, fakeClient, testRepo, testDir)

	assert.NotEmpty(t, agents, "Agents from successful fetches should be returned")
	assert.Error(t, err, "Error should be returned for the failed fetch")
}

// =============================================================================
// TS-GH-42-012: Error message identifies failing filename (P1)
// =============================================================================

func TestDiscoverRemoteAgents_ErrorIdentifiesFilename(t *testing.T) {
	ctx := context.Background()
	failingFile := "bad-agent.yaml"
	fakeClient := newFakeForgeClient(
		map[string]string{
			failingFile: "file",
		},
		nil,
		map[string]error{
			failingFile: fmt.Errorf("permission denied"),
		},
		nil,
	)

	_, err := harness.DiscoverRemoteAgents(ctx, fakeClient, testRepo, testDir)

	require.Error(t, err)
	assert.Contains(t, err.Error(), failingFile,
		"Error message should contain the failing filename for debugging")
}

// =============================================================================
// TS-GH-42-013: Role-only agent included (P1)
// =============================================================================

func TestDiscoverRemoteAgents_RoleOnlyAgent(t *testing.T) {
	ctx := context.Background()
	fakeClient := newFakeForgeClient(
		map[string]string{"role-only.yaml": "file"},
		map[string]string{"role-only.yaml": "role: \"builder\"\nbase: \"default\"\n"},
		nil,
		nil,
	)

	agents, err := harness.DiscoverRemoteAgents(ctx, fakeClient, testRepo, testDir)

	require.NoError(t, err)
	require.Len(t, agents, 1)
	assert.Equal(t, "builder", agents[0].Role, "Role should be correctly extracted")
	assert.Empty(t, agents[0].Slug, "Slug should be empty for role-only agent")
}

// =============================================================================
// TS-GH-42-014: Slug-only agent included (P1)
// =============================================================================

func TestDiscoverRemoteAgents_SlugOnlyAgent(t *testing.T) {
	ctx := context.Background()
	fakeClient := newFakeForgeClient(
		map[string]string{"slug-only.yaml": "file"},
		map[string]string{"slug-only.yaml": "slug: \"custom-agent\"\nbase: \"default\"\n"},
		nil,
		nil,
	)

	agents, err := harness.DiscoverRemoteAgents(ctx, fakeClient, testRepo, testDir)

	require.NoError(t, err)
	require.Len(t, agents, 1)
	assert.Equal(t, "custom-agent", agents[0].Slug, "Slug should be correctly extracted")
	assert.Empty(t, agents[0].Role, "Role should be empty for slug-only agent")
}

// =============================================================================
// TS-GH-42-015: Path field empty for remote agents (P1)
// =============================================================================

func TestDiscoverRemoteAgents_PathEmpty(t *testing.T) {
	ctx := context.Background()
	fakeClient := newFakeForgeClient(
		map[string]string{
			"agent-1.yaml": "file",
			"agent-2.yaml": "file",
		},
		map[string]string{
			"agent-1.yaml": "role: \"a\"\nslug: \"a-agent\"\n",
			"agent-2.yaml": "role: \"b\"\nslug: \"b-agent\"\n",
		},
		nil,
		nil,
	)

	agents, err := harness.DiscoverRemoteAgents(ctx, fakeClient, testRepo, testDir)

	require.NoError(t, err)
	require.Len(t, agents, 2)
	for i, agent := range agents {
		assert.Empty(t, agent.Path,
			"Agent[%d] Path should be empty for remote agents (no local filesystem path)", i)
	}
}

// =============================================================================
// TS-GH-42-016: Path prefix stripped to bare filename (P1)
// =============================================================================

func TestDiscoverRemoteAgents_PathPrefixStripped(t *testing.T) {
	ctx := context.Background()
	fakeClient := newFakeForgeClient(
		map[string]string{
			"harness/agents/builder.yaml": "file",
		},
		map[string]string{
			"harness/agents/builder.yaml": "role: \"builder\"\nslug: \"b\"\n",
		},
		nil,
		nil,
	)

	agents, err := harness.DiscoverRemoteAgents(ctx, fakeClient, testRepo, testDir)

	require.NoError(t, err)
	require.Len(t, agents, 1)
	assert.Equal(t, "builder.yaml", agents[0].Filename,
		"Path prefix should be stripped — Filename should be bare filename only")
}

// =============================================================================
// TS-GH-42-017: LoadRaw backward compatibility — unvalidated structure (P0)
// =============================================================================

func TestLoadRaw_BackwardCompat_UnvalidatedStructure(t *testing.T) {
	content := []byte(`role: "test-agent"
slug: "test"
base: "default"
config:
  timeout: 300
`)
	tmpFile := filepath.Join(t.TempDir(), "test-harness.yaml")
	require.NoError(t, os.WriteFile(tmpFile, content, 0644))

	result, err := harness.LoadRaw(tmpFile)

	require.NoError(t, err, "LoadRaw should succeed for valid harness file")
	require.NotNil(t, result, "Result should not be nil")
	assert.Equal(t, "test-agent", result.Role, "Role field should be populated")
	assert.Equal(t, "test", result.Slug, "Slug field should be populated")
	assert.Equal(t, "default", result.Base, "Base field should be populated")
}

// =============================================================================
// TS-GH-42-018: LoadRaw backward compatibility — config mappings (P0)
// =============================================================================

func TestLoadRaw_BackwardCompat_ConfigMappings(t *testing.T) {
	content := []byte(`role: "complex-agent"
slug: "complex"
config:
  timeout: 300
  retries: 3
  labels:
    env: "prod"
    tier: "premium"
`)
	tmpFile := filepath.Join(t.TempDir(), "complex-harness.yaml")
	require.NoError(t, os.WriteFile(tmpFile, content, 0644))

	result, err := harness.LoadRaw(tmpFile)

	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify nested config maps are preserved exactly
	require.NotNil(t, result.Config, "Config section should be preserved")

	// Verify top-level config values
	timeout, ok := result.Config["timeout"]
	assert.True(t, ok, "timeout key should exist in config")
	assert.Equal(t, 300, timeout, "timeout value should be preserved")

	retries, ok := result.Config["retries"]
	assert.True(t, ok, "retries key should exist in config")
	assert.Equal(t, 3, retries, "retries value should be preserved")

	// Verify nested map
	labels, ok := result.Config["labels"]
	assert.True(t, ok, "labels key should exist in config")
	labelsMap, ok := labels.(map[string]interface{})
	require.True(t, ok, "labels should be a nested map")
	assert.Equal(t, "prod", labelsMap["env"], "Nested env label should be preserved")
	assert.Equal(t, "premium", labelsMap["tier"], "Nested tier label should be preserved")
}

// =============================================================================
// TS-GH-42-019: LoadRaw backward compatibility — invalid path (P0)
// =============================================================================

func TestLoadRaw_BackwardCompat_InvalidPath(t *testing.T) {
	_, err := harness.LoadRaw("/nonexistent/path/harness.yaml")

	require.Error(t, err, "Error should be returned for non-existent file")
	assert.True(t, errors.Is(err, os.ErrNotExist),
		"Error should wrap os.ErrNotExist for callers' file-existence checks")
}

// =============================================================================
// TS-GH-42-020: LoadRaw backward compatibility — consumers compile (P0)
// NOTE: This is a build verification test. Compilation of this file itself
// validates that the harness package API is compatible. The test body verifies
// the function signature is callable with expected parameters.
// =============================================================================

func TestLoadRaw_BackwardCompat_ConsumersCompile(t *testing.T) {
	// This test verifies that LoadRaw's function signature hasn't changed.
	// If the parseRaw refactoring altered the return type or parameters,
	// this file would fail to compile — catching the regression at build time.
	var loadFn func(string) (*harness.RawHarness, error) = harness.LoadRaw
	assert.NotNil(t, loadFn, "LoadRaw should be a callable function with unchanged signature")
}

// =============================================================================
// TS-GH-42-021: End-to-end fake client flow (P2)
// =============================================================================

func TestDiscoverRemoteAgents_E2E_FakeClient(t *testing.T) {
	ctx := context.Background()
	fakeClient := newFakeForgeClient(
		map[string]string{
			"agent-alpha.yaml": "file",
			"agent-beta.yaml":  "file",
			"readme.md":        "file", // should be ignored
		},
		map[string]string{
			"agent-alpha.yaml": "role: \"alpha\"\nslug: \"alpha-agent\"\nbase: \"default\"\n",
			"agent-beta.yaml":  "role: \"beta\"\nslug: \"beta-agent\"\nbase: \"default\"\n",
		},
		nil,
		nil,
	)

	agents, err := harness.DiscoverRemoteAgents(ctx, fakeClient, testRepo, testDir)

	require.NoError(t, err, "End-to-end discovery should succeed")
	require.Len(t, agents, 2, "Only YAML files should be processed")

	// Verify sorted by role ascending
	assert.Equal(t, "alpha", agents[0].Role, "Alpha should come first")
	assert.Equal(t, "alpha-agent", agents[0].Slug)
	assert.Equal(t, "agent-alpha.yaml", agents[0].Filename)
	assert.Empty(t, agents[0].Path, "Remote agents have no local path")

	assert.Equal(t, "beta", agents[1].Role, "Beta should come second")
	assert.Equal(t, "beta-agent", agents[1].Slug)
	assert.Equal(t, "agent-beta.yaml", agents[1].Filename)
	assert.Empty(t, agents[1].Path, "Remote agents have no local path")
}

// =============================================================================
// TS-GH-42-022: Empty directory handling (P2)
// =============================================================================

func TestDiscoverRemoteAgents_EmptyDirectory(t *testing.T) {
	ctx := context.Background()
	fakeClient := newFakeForgeClient(
		map[string]string{}, // empty directory listing
		nil,
		nil,
		nil,
	)

	agents, err := harness.DiscoverRemoteAgents(ctx, fakeClient, testRepo, testDir)

	assert.NoError(t, err, "No error for empty directory")
	assert.Empty(t, agents, "No agents from empty directory")
}

// =============================================================================
// TS-GH-42-023: Concurrent calls safety (P2)
// =============================================================================

func TestDiscoverRemoteAgents_ConcurrentCalls(t *testing.T) {
	const numGoroutines = 10
	ctx := context.Background()

	var wg sync.WaitGroup
	results := make([][]harness.AgentInfo, numGoroutines)
	errs := make([]error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			roleName := fmt.Sprintf("agent-%d", idx)
			filename := fmt.Sprintf("agent-%d.yaml", idx)
			content := fmt.Sprintf("role: %q\nslug: %q\n", roleName, roleName)

			client := newFakeForgeClient(
				map[string]string{filename: "file"},
				map[string]string{filename: content},
				nil,
				nil,
			)

			agents, err := harness.DiscoverRemoteAgents(ctx, client, testRepo, testDir)
			results[idx] = agents
			errs[idx] = err
		}(i)
	}

	wg.Wait()

	for i := 0; i < numGoroutines; i++ {
		assert.NoError(t, errs[i], "Goroutine %d should not return error", i)
		require.Len(t, results[i], 1, "Goroutine %d should return exactly one agent", i)

		expectedRole := fmt.Sprintf("agent-%d", i)
		assert.Equal(t, expectedRole, results[i][0].Role,
			"Goroutine %d result should be independent — no cross-contamination", i)
	}
}

// Ensure unused imports don't cause compile errors — these are used in string
// operations within error validation tests above.
var _ = strings.Contains
var _ = fmt.Sprintf
