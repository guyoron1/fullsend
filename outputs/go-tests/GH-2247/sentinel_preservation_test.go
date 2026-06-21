//go:build e2e

package scaffold_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Sentinel Preservation Tests

STP Reference: outputs/stp/GH-2247/GH-2247_test_plan.md
Jira: GH-2247

Requirement: The sentinel line ("# --- fullsend managed below - do not edit ---")
must never be stripped from the shim blob written to enrolled repos.
*/

func TestSentinelPreservation(t *testing.T) {

	/*
	TS-GH-2247-007 — Verify sentinel present in blob when updating a stale shim.
	The original bug (PR #2101) removed sentinel lines.
	*/
	t.Run("[test_id:TS-GH-2247-007]_sentinel_present_in_stale_shim_update", func(t *testing.T) {
		env := newTestEnv(t)

		// Remote has sentinel but outdated managed content.
		staleRemote := sentinelLine + "\nold outdated template\n"
		staleB64 := encodeB64(staleRemote)

		env.installGhMock(t, `
case "$endpoint" in
  repos/test-org/test-repo/actions/variables/*)
    json='{"status":"404","message":"Not Found"}'
    rc=1
    ;;
  repos/test-org/test-repo/contents/.github/workflows/fullsend.yaml)
    json='{"content":"`+staleB64+`","sha":"file-sha"}'
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
`)

		_, _ = env.runReconcile(t)

		blob, ok := env.blobContent(t, "test-repo")
		require.True(t, ok, "blob should be created for stale shim update")

		assert.Contains(t, blob, sentinelLine,
			"update blob must contain sentinel line")

		// Sentinel should appear before managed content.
		sentinelIdx := strings.Index(blob, sentinelLine)
		contentIdx := strings.Index(blob, freshTemplate)
		require.True(t, contentIdx >= 0, "blob should contain fresh template")
		assert.Less(t, sentinelIdx, contentIdx,
			"sentinel must appear before managed content")
	})

	/*
	TS-GH-2247-008 — Verify sentinel is added when migrating a pre-sentinel shim.
	Pre-sentinel shims are legacy artifacts that lack the sentinel convention.
	*/
	t.Run("[test_id:TS-GH-2247-008]_sentinel_present_in_pre_sentinel_migration", func(t *testing.T) {
		env := newTestEnv(t)

		// Pre-sentinel remote: old content without any sentinel line.
		preSentinelRemote := "stale shim template\n"
		preSentinelB64 := encodeB64(preSentinelRemote)

		env.installGhMock(t, `
case "$endpoint" in
  repos/test-org/test-repo/actions/variables/*)
    json='{"status":"404","message":"Not Found"}'
    rc=1
    ;;
  repos/test-org/test-repo/contents/.github/workflows/fullsend.yaml)
    json='{"content":"`+preSentinelB64+`","sha":"file-sha"}'
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
`)

		_, _ = env.runReconcile(t)

		blob, ok := env.blobContent(t, "test-repo")
		require.True(t, ok, "blob should be created for pre-sentinel migration")

		assert.Contains(t, blob, sentinelLine,
			"migrated blob must contain sentinel line")
	})

	/*
	TS-GH-2247-009 — Verify sentinel present in blob for new repo enrollment.
	New enrollments must include sentinel from day one.
	*/
	t.Run("[test_id:TS-GH-2247-009]_sentinel_present_in_new_enrollment", func(t *testing.T) {
		env := newTestEnv(t)

		// No shim file exists (404 from GitHub API).
		env.installGhMock(t, `
case "$endpoint" in
  repos/test-org/test-repo/actions/variables/*)
    json='{"status":"404","message":"Not Found"}'
    rc=1
    ;;
  repos/test-org/test-repo/contents/.github/workflows/fullsend.yaml)
    rc=1
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
`)

		_, _ = env.runReconcile(t)

		blob, ok := env.blobContent(t, "test-repo")
		require.True(t, ok, "blob should be created for new enrollment")

		assert.Contains(t, blob, sentinelLine,
			"new enrollment blob must contain sentinel line")
	})
}
