//go:build e2e

package scaffold_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Pre-Sentinel Shim Migration Tests

STP Reference: outputs/stp/GH-2247/GH-2247_test_plan.md
Jira: GH-2247

Requirement: Repos with pre-sentinel shims (no sentinel line) are correctly
detected as stale and migrated to the sentinel-based template without content
duplication.
*/

// ghMockForPreSentinel returns a gh mock case body for pre-sentinel migration tests.
func ghMockForPreSentinel(remoteB64 string) string {
	return `
case "$endpoint" in
  repos/test-org/test-repo/actions/variables/*)
    json='{"status":"404","message":"Not Found"}'
    rc=1
    ;;
  repos/test-org/test-repo/contents/.github/workflows/fullsend.yaml)
    json='{"content":"` + remoteB64 + `","sha":"file-sha"}'
    ;;
  repos/test-org/test-repo)
    json='{"default_branch":"main","private":false}'
    ;;
  repos/test-org/test-repo/git/ref/heads/*)
    json='{"object":{"sha":"base-sha"}}'
    ;;
  repos/test-org/test-repo/git/commits/base-sha)
    json='{"tree":{"sha":"base-tree-sha"}}'
    ;;
  repos/test-org/test-repo/git/blobs)
    json='{"sha":"blob-sha"}'
    ;;
  repos/test-org/test-repo/git/trees)
    json='{"sha":"tree-sha"}'
    ;;
  repos/test-org/test-repo/git/commits)
    json='{"sha":"desired-commit-sha"}'
    ;;
  repos/test-org/test-repo/git/refs)
    rc=1
    ;;
  repos/test-org/test-repo/git/refs/heads/*)
    rc=0
    ;;
  *)
    rc=0
    ;;
esac
`
}

func TestPreSentinelMigration(t *testing.T) {

	/*
	TS-GH-2247-017 — Verify that a shim without a sentinel line is correctly
	identified as stale and scheduled for migration.
	*/
	t.Run("[test_id:TS-GH-2247-017]_pre_sentinel_shim_flagged_stale", func(t *testing.T) {
		env := newTestEnv(t)

		// Pre-sentinel remote: old workflow content, no sentinel.
		preSentinelRemote := "stale shim template\n"
		preSentinelB64 := encodeB64(preSentinelRemote)

		env.installGhMock(t, ghMockForPreSentinel(preSentinelB64))

		output, _ := env.runReconcile(t)

		assert.Contains(t, output, "shim is stale",
			"pre-sentinel shim must be flagged as stale")
	})

	/*
	TS-GH-2247-018 — Verify that pre-sentinel migration does not duplicate
	old content in the output blob.
	*/
	t.Run("[test_id:TS-GH-2247-018]_migration_does_not_duplicate_content", func(t *testing.T) {
		env := newTestEnv(t)

		// Pre-sentinel remote with known content.
		preSentinelRemote := "name: fullsend\non:\n  push:\n    branches: [main]\n"
		preSentinelB64 := encodeB64(preSentinelRemote)

		env.installGhMock(t, ghMockForPreSentinel(preSentinelB64))

		_, _ = env.runReconcile(t)

		blob, ok := env.blobContent(t, "test-repo")
		require.True(t, ok, "blob should be created for pre-sentinel migration")

		// The old content should NOT be present in the migrated blob.
		assert.NotContains(t, blob, "name: fullsend",
			"old content should be replaced, not duplicated")

		// Fresh template content should be present.
		assert.Contains(t, blob, freshTemplate,
			"migrated blob should contain fresh template content")
	})

	/*
	TS-GH-2247-019 — Verify migrated blob has sentinel line and fresh template.
	After migration, the shim must be in canonical format.
	*/
	t.Run("[test_id:TS-GH-2247-019]_migrated_blob_has_sentinel_and_fresh_template", func(t *testing.T) {
		env := newTestEnv(t)

		preSentinelRemote := "old workflow content\n"
		preSentinelB64 := encodeB64(preSentinelRemote)

		env.installGhMock(t, ghMockForPreSentinel(preSentinelB64))

		_, _ = env.runReconcile(t)

		blob, ok := env.blobContent(t, "test-repo")
		require.True(t, ok, "blob should be created for pre-sentinel migration")

		// Verify sentinel is present.
		assert.Contains(t, blob, sentinelLine,
			"migrated blob must contain sentinel line")

		// Verify fresh template content is present after sentinel.
		assert.Contains(t, blob, freshTemplate,
			"migrated blob must contain fresh template content")

		// Verify sentinel appears before fresh template.
		sentinelIdx := strings.Index(blob, sentinelLine)
		templateIdx := strings.Index(blob, freshTemplate)
		assert.Less(t, sentinelIdx, templateIdx,
			"sentinel must appear before template content in migrated blob")
	})
}
