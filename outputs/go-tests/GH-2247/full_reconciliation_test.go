//go:build e2e

package scaffold_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Full Reconciliation Tests (Functional)

STP Reference: outputs/stp/GH-2247/GH-2247_test_plan.md
Jira: GH-2247

Requirement: End-to-end reconciliation run does not open update PRs for repos
whose shim content is logically up-to-date. Genuinely stale repos are correctly
updated.
*/

func TestFullReconciliation(t *testing.T) {

	/*
	TS-GH-2247-020 — Primary functional regression test for GH-2247.
	Verifies that a full reconciliation run skips up-to-date repos without
	creating branches or PRs.
	*/
	t.Run("[test_id:TS-GH-2247-020]_skips_up_to_date_repos", func(t *testing.T) {
		env := newTestEnv(t)

		// Configure two repos, both up-to-date.
		env.setConfig(t, `version: 1
repos:
  repo-a:
    enabled: true
  repo-b:
    enabled: true
`)
		env.installYqMock(t, []string{"repo-a", "repo-b"}, nil)

		// Both repos have current content (sentinel + fresh template).
		currentContent := sentinelLine + "\n" + freshTemplate + "\n"
		currentB64 := encodeB64(currentContent)

		env.installGhMock(t, `
case "$endpoint" in
  repos/test-org/*/actions/variables/*)
    json='{"status":"404","message":"Not Found"}'
    rc=1
    ;;
  repos/test-org/*/contents/.github/workflows/fullsend.yaml)
    json='{"content":"`+currentB64+`","sha":"file-sha"}'
    ;;
  repos/test-org/*)
    json='{"default_branch":"main","private":false}'
    ;;
  *)
    rc=0
    ;;
esac
`)

		output, _ := env.runReconcile(t)

		ghLog := env.ghLogContents(t)

		// No branch creation API calls.
		assert.NotContains(t, ghLog, "git/refs",
			"no branch creation should occur for up-to-date repos")

		// No PR creation calls.
		assert.NotContains(t, ghLog, "pr create",
			"no PR should be created for up-to-date repos")

		// Script should report repos as up-to-date.
		assert.Contains(t, output, "already enrolled (shim up to date)",
			"script should log up-to-date status")

		// No blobs should be created.
		assert.False(t, env.blobExists(t, "repo-a"),
			"no blob for up-to-date repo-a")
		assert.False(t, env.blobExists(t, "repo-b"),
			"no blob for up-to-date repo-b")
	})

	/*
	TS-GH-2247-021 — Verify reconciliation correctly updates genuinely stale repos.
	The fix must not introduce false negatives.
	*/
	t.Run("[test_id:TS-GH-2247-021]_updates_genuinely_stale_repos", func(t *testing.T) {
		env := newTestEnv(t)

		env.setConfig(t, `version: 1
repos:
  stale-repo:
    enabled: true
`)
		env.installYqMock(t, []string{"stale-repo"}, nil)

		// Stale repo has outdated managed content.
		staleContent := sentinelLine + "\nold outdated template\n"
		staleB64 := encodeB64(staleContent)

		env.installGhMock(t, `
case "$endpoint" in
  repos/test-org/stale-repo/actions/variables/*)
    json='{"status":"404","message":"Not Found"}'
    rc=1
    ;;
  repos/test-org/stale-repo/contents/.github/workflows/fullsend.yaml)
    json='{"content":"`+staleB64+`","sha":"file-sha"}'
    ;;
  repos/test-org/stale-repo)
    json='{"default_branch":"main","private":false}'
    ;;
  repos/test-org/stale-repo/git/ref/heads/*)
    json='{"object":{"sha":"base-sha"}}'
    ;;
  repos/test-org/stale-repo/git/commits/base-sha)
    json='{"tree":{"sha":"base-tree-sha"}}'
    ;;
  repos/test-org/stale-repo/git/blobs)
    json='{"sha":"blob-sha"}'
    ;;
  repos/test-org/stale-repo/git/trees)
    json='{"sha":"tree-sha"}'
    ;;
  repos/test-org/stale-repo/git/commits)
    json='{"sha":"desired-commit-sha"}'
    ;;
  repos/test-org/stale-repo/git/refs)
    rc=1
    ;;
  repos/test-org/stale-repo/git/refs/heads/*)
    rc=0
    ;;
  *)
    rc=0
    ;;
esac
`)

		output, _ := env.runReconcile(t)

		// Stale repo should be flagged.
		assert.Contains(t, output, "shim is stale",
			"stale repo should be flagged for update")

		// Blob should be created with sentinel and fresh content.
		blob, ok := env.blobContent(t, "stale-repo")
		require.True(t, ok, "blob should be created for stale repo")
		assert.Contains(t, blob, sentinelLine,
			"update blob must contain sentinel")
		assert.Contains(t, blob, freshTemplate,
			"update blob must contain fresh template")
	})

	/*
	TS-GH-2247-022 — Verify no blob content is generated when shim is current.
	The script should short-circuit comparison and skip blob generation.
	*/
	t.Run("[test_id:TS-GH-2247-022]_no_blob_created_when_current", func(t *testing.T) {
		env := newTestEnv(t)

		// Current content — exact match to template.
		currentContent := sentinelLine + "\n" + freshTemplate + "\n"
		currentB64 := encodeB64(currentContent)

		env.installGhMock(t, `
case "$endpoint" in
  repos/test-org/test-repo/actions/variables/*)
    json='{"status":"404","message":"Not Found"}'
    rc=1
    ;;
  repos/test-org/test-repo/contents/.github/workflows/fullsend.yaml)
    json='{"content":"`+currentB64+`","sha":"file-sha"}'
    ;;
  repos/test-org/test-repo)
    json='{"default_branch":"main","private":false}'
    ;;
  *)
    rc=0
    ;;
esac
`)

		_, _ = env.runReconcile(t)

		// No blob should be generated at all.
		assert.False(t, env.blobExists(t, "test-repo"),
			"no blob should be generated for current shim")

		// No file creation API calls in gh log.
		ghLog := env.ghLogContents(t)
		assert.NotContains(t, ghLog, "git/blobs",
			"no blob API calls should be made for current shim")
	})

	/*
	TS-GH-2247-023 — Verify reconciliation handles mixed repo states correctly.
	Four repos: up-to-date, stale, pre-sentinel, and new enrollment.
	*/
	t.Run("[test_id:TS-GH-2247-023]_handles_mixed_repo_states", func(t *testing.T) {
		env := newTestEnv(t)

		env.setConfig(t, `version: 1
repos:
  repo-a:
    enabled: true
  repo-b:
    enabled: true
  repo-c:
    enabled: true
  repo-d:
    enabled: true
`)
		env.installYqMock(t, []string{"repo-a", "repo-b", "repo-c", "repo-d"}, nil)

		// repo-a: current (up-to-date)
		currentContent := sentinelLine + "\n" + freshTemplate + "\n"
		currentB64 := encodeB64(currentContent)

		// repo-b: stale (outdated managed content)
		staleContent := sentinelLine + "\nold outdated template\n"
		staleB64 := encodeB64(staleContent)

		// repo-c: pre-sentinel (no sentinel line)
		preSentinelContent := "legacy content without sentinel\n"
		preSentinelB64 := encodeB64(preSentinelContent)

		// repo-d: new (404 — no shim file exists)

		env.installGhMock(t, `
case "$endpoint" in
  repos/test-org/*/actions/variables/*)
    json='{"status":"404","message":"Not Found"}'
    rc=1
    ;;
  repos/test-org/repo-a/contents/.github/workflows/fullsend.yaml)
    json='{"content":"`+currentB64+`","sha":"file-sha"}'
    ;;
  repos/test-org/repo-b/contents/.github/workflows/fullsend.yaml)
    json='{"content":"`+staleB64+`","sha":"file-sha"}'
    ;;
  repos/test-org/repo-c/contents/.github/workflows/fullsend.yaml)
    json='{"content":"`+preSentinelB64+`","sha":"file-sha"}'
    ;;
  repos/test-org/repo-d/contents/.github/workflows/fullsend.yaml)
    rc=1
    ;;
  repos/test-org/*/git/ref/heads/*)
    json='{"object":{"sha":"base-sha"}}'
    ;;
  repos/test-org/*/git/commits/base-sha)
    json='{"tree":{"sha":"base-tree-sha"}}'
    ;;
  repos/test-org/*/git/blobs)
    json='{"sha":"blob-sha"}'
    ;;
  repos/test-org/*/git/trees)
    json='{"sha":"tree-sha"}'
    ;;
  repos/test-org/*/git/commits)
    json='{"sha":"desired-commit-sha"}'
    ;;
  repos/test-org/*/git/refs)
    rc=1
    ;;
  repos/test-org/*/git/refs/heads/*)
    rc=0
    ;;
  repos/test-org/*)
    json='{"default_branch":"main","private":false}'
    ;;
  *)
    rc=0
    ;;
esac
`)

		_, _ = env.runReconcile(t)

		// repo-a: skipped (no blob created).
		assert.False(t, env.blobExists(t, "repo-a"),
			"up-to-date repo-a should not have a blob")

		// repo-b: updated (blob created with sentinel + fresh template).
		blobB, okB := env.blobContent(t, "repo-b")
		require.True(t, okB, "stale repo-b should have a blob")
		assert.Contains(t, blobB, sentinelLine,
			"repo-b update blob must have sentinel")
		assert.Contains(t, blobB, freshTemplate,
			"repo-b update blob must have fresh template")

		// repo-c: migrated (blob created with sentinel).
		blobC, okC := env.blobContent(t, "repo-c")
		require.True(t, okC, "pre-sentinel repo-c should have a blob")
		assert.Contains(t, blobC, sentinelLine,
			"repo-c migration blob must have sentinel")
		assert.NotContains(t, blobC, "legacy content without sentinel",
			"repo-c migration should not preserve old content")

		// repo-d: enrolled (blob created with full template).
		blobD, okD := env.blobContent(t, "repo-d")
		require.True(t, okD, "new repo-d should have a blob")
		assert.Contains(t, blobD, sentinelLine,
			"repo-d enrollment blob must have sentinel")
		assert.Contains(t, blobD, freshTemplate,
			"repo-d enrollment blob must have fresh template")

		// Verify blob content ordering for repos with blobs.
		for _, tc := range []struct {
			name string
			blob string
		}{
			{"repo-b", blobB},
			{"repo-c", blobC},
			{"repo-d", blobD},
		} {
			sentIdx := strings.Index(tc.blob, sentinelLine)
			tmplIdx := strings.Index(tc.blob, freshTemplate)
			if sentIdx >= 0 && tmplIdx >= 0 {
				assert.Less(t, sentIdx, tmplIdx,
					"%s: sentinel must appear before template", tc.name)
			}
		}
	})
}
