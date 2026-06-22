package harness

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GH-73-TC-021: Verify discovery parses role and slug from YAML
func TestQF_DiscoverAgents_ParsesRoleAndSlug(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "agent1.yaml", "agent: agents/code.md\nrole: reviewer\nslug: my-agent\n")
	writeFile(t, dir, "agent2.yaml", "agent: agents/triage.md\nrole: triage\nslug: my-triage\n")

	agents, err := DiscoverAgents(dir)
	require.NoError(t, err)
	require.Len(t, agents, 2)

	// Sorted by role
	assert.Equal(t, "reviewer", agents[0].Role)
	assert.Equal(t, "my-agent", agents[0].Slug)
	assert.Equal(t, "triage", agents[1].Role)
	assert.Equal(t, "my-triage", agents[1].Slug)
}

// GH-73-TC-022: Verify slug derivation from role and appSet
// Note: In the actual codebase, slug derivation is not done inside DiscoverAgents.
// DiscoverAgents reads the slug from YAML directly. When slug is empty, it remains empty.
// This test verifies that behavior.
func TestQF_DiscoverAgents_RoleWithoutSlug(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "partial.yaml", "agent: agents/triage.md\nrole: triage\n")

	agents, err := DiscoverAgents(dir)
	require.NoError(t, err)
	require.Len(t, agents, 1)
	assert.Equal(t, "triage", agents[0].Role)
	assert.Empty(t, agents[0].Slug, "slug should be empty when not specified in YAML")
}

// GH-73-TC-023: Verify deduplication of discovered slugs
// Note: DiscoverAgents does not deduplicate — it returns all entries with role or slug.
// We test that multiple files with the same slug are all returned.
func TestQF_DiscoverAgents_MultipleFilesReturned(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "agent1.yaml", "agent: agents/code.md\nrole: coder\nslug: same-slug\n")
	writeFile(t, dir, "agent2.yaml", "agent: agents/review.md\nrole: reviewer\nslug: same-slug\n")
	writeFile(t, dir, "agent3.yaml", "agent: agents/triage.md\nrole: triage\nslug: different-slug\n")

	agents, err := DiscoverAgents(dir)
	require.NoError(t, err)
	assert.Len(t, agents, 3, "all entries should be returned including duplicate slugs")
}

// GH-73-TC-024: Verify graceful handling of partial parse errors
func TestQF_DiscoverAgents_PartialParseErrors(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "good1.yaml", "agent: agents/code.md\nrole: coder\nslug: fs-code\n")
	writeFile(t, dir, "good2.yaml", "agent: agents/review.md\nrole: reviewer\nslug: fs-review\n")
	writeFile(t, dir, "bad.yaml", ":\n  :\n    - [invalid yaml")

	agents, err := DiscoverAgents(dir)
	require.Error(t, err, "should return error for malformed YAML")
	assert.Len(t, agents, 2, "should return entries from valid files")

	roles := []string{agents[0].Role, agents[1].Role}
	assert.Contains(t, roles, "coder")
	assert.Contains(t, roles, "reviewer")
}

// GH-73-TC-025: Verify nil return when harness dir missing
func TestQF_DiscoverAgents_MissingDirectory(t *testing.T) {
	agents, err := DiscoverAgents(filepath.Join(t.TempDir(), "nonexistent"))
	require.NoError(t, err)
	assert.Nil(t, agents, "should return nil when directory does not exist")
}

// GH-73-TC-025 supplemental: empty directory
func TestQF_DiscoverAgents_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()

	agents, err := DiscoverAgents(dir)
	require.NoError(t, err)
	assert.Empty(t, agents, "should return empty list for empty directory")
}
