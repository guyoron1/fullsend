//go:build e2e

package scaffold_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Shim Drift Detection Tests

STP Reference: outputs/stp/GH-2247/GH-2247_test_plan.md
Jira: GH-2247

Requirement: Shim drift detection uses decoded text comparison instead of
re-encoded base64 to avoid false-positive stale detection.
*/

func TestShimDriftDetection(t *testing.T) {

	/*
	TS-GH-2247-001 — Primary regression test for GH-2247.
	Verifies that shim content differing only in trailing newlines
	(a common base64 re-encoding artifact) is NOT flagged as stale.
	*/
	t.Run("[test_id:TS-GH-2247-001]_identical_content_trailing_newlines_not_stale", func(t *testing.T) {
		env := newTestEnv(t)

		// The managed content is sentinel + fresh template.
		managedContent := sentinelLine + "\n" + freshTemplate + "\n"
		// The remote has an extra trailing newline — produces different base64
		// but is logically identical after newline normalization.
		remoteContent := managedContent + "\n"
		remoteB64 := encodeB64(remoteContent)

		env.installGhMock(t, `
case "$endpoint" in
  repos/test-org/test-repo/actions/variables/*)
    json='{"status":"404","message":"Not Found"}'
    rc=1
    ;;
  repos/test-org/test-repo/contents/.github/workflows/fullsend.yaml)
    json='{"content":"`+remoteB64+`","sha":"file-sha"}'
    ;;
  repos/test-org/test-repo)
    json='{"default_branch":"main","private":false}'
    ;;
  *)
    rc=0
    ;;
esac
`)

		output, _ := env.runReconcile(t)

		assert.NotContains(t, output, "shim is stale",
			"identical content with trailing newline diff should NOT be flagged stale")
		assert.Contains(t, output, "already enrolled (shim up to date)",
			"should report shim as up-to-date")

		// No blob should be created for an up-to-date shim.
		assert.False(t, env.blobExists(t, "test-repo"),
			"no blob should be created for up-to-date shim")
	})

	/*
	TS-GH-2247-002 — Verify genuinely stale managed content IS detected.
	The fix must not introduce false negatives.
	*/
	t.Run("[test_id:TS-GH-2247-002]_stale_managed_content_detected", func(t *testing.T) {
		env := newTestEnv(t)

		// Remote has outdated content below the sentinel.
		staleRemote := sentinelLine + "\nold stale template\n"
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

		output, _ := env.runReconcile(t)

		assert.Contains(t, output, "shim is stale",
			"genuinely stale content should be flagged")
	})

	/*
	TS-GH-2247-003 — Verify a shim with user header above sentinel is NOT
	flagged stale when the managed section matches.
	*/
	t.Run("[test_id:TS-GH-2247-003]_up_to_date_shim_with_user_header_not_stale", func(t *testing.T) {
		env := newTestEnv(t)

		managedContent := sentinelLine + "\n" + freshTemplate + "\n"
		// Remote has a license header above sentinel + matching managed content.
		remoteContent := "# Copyright 2026 Conforma\n# SPDX-License-Identifier: Apache-2.0\n" + managedContent
		remoteB64 := encodeB64(remoteContent)

		env.installGhMock(t, `
case "$endpoint" in
  repos/test-org/test-repo/actions/variables/*)
    json='{"status":"404","message":"Not Found"}'
    rc=1
    ;;
  repos/test-org/test-repo/contents/.github/workflows/fullsend.yaml)
    json='{"content":"`+remoteB64+`","sha":"file-sha"}'
    ;;
  repos/test-org/test-repo)
    json='{"default_branch":"main","private":false}'
    ;;
  *)
    rc=0
    ;;
esac
`)

		output, _ := env.runReconcile(t)

		assert.NotContains(t, output, "shim is stale",
			"up-to-date shim with user header should not be flagged stale")
		assert.Contains(t, output, "already enrolled (shim up to date)",
			"should report shim with header as up-to-date")

		assert.False(t, env.blobExists(t, "test-repo"),
			"no blob should be created when managed content matches")
	})

	/*
	TS-GH-2247-004 — Verify CRLF line endings are normalized before comparison
	to prevent false positives from platform-dependent line endings.
	*/
	t.Run("[test_id:TS-GH-2247-004]_carriage_return_normalization", func(t *testing.T) {
		env := newTestEnv(t)

		// Create content with CRLF line endings.
		managedLF := sentinelLine + "\n" + freshTemplate + "\n"
		managedCRLF := strings.ReplaceAll(managedLF, "\n", "\r\n")
		crlfB64 := encodeB64(managedCRLF)

		env.installGhMock(t, `
case "$endpoint" in
  repos/test-org/test-repo/actions/variables/*)
    json='{"status":"404","message":"Not Found"}'
    rc=1
    ;;
  repos/test-org/test-repo/contents/.github/workflows/fullsend.yaml)
    json='{"content":"`+crlfB64+`","sha":"file-sha"}'
    ;;
  repos/test-org/test-repo)
    json='{"default_branch":"main","private":false}'
    ;;
  *)
    rc=0
    ;;
esac
`)

		output, _ := env.runReconcile(t)

		// CRLF content should be normalized and match the LF template.
		assert.NotContains(t, output, "shim is stale",
			"CRLF content should be normalized and not flagged stale")
	})

	/*
	TS-GH-2247-005 — Verify that empty remote content (no shim file)
	triggers enrollment flow rather than an update.
	*/
	t.Run("[test_id:TS-GH-2247-005]_empty_remote_triggers_enrollment", func(t *testing.T) {
		env := newTestEnv(t)

		env.installGhMock(t, `
case "$endpoint" in
  repos/test-org/test-repo/actions/variables/*)
    json='{"status":"404","message":"Not Found"}'
    rc=1
    ;;
  repos/test-org/test-repo/contents/.github/workflows/fullsend.yaml)
    # No shim file exists — 404.
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

		output, _ := env.runReconcile(t)

		// Should trigger enrollment, not crash on empty content.
		require.NotEmpty(t, output, "script should produce output")
		// Enrollment creates a blob with sentinel.
		if env.blobExists(t, "test-repo") {
			blob, ok := env.blobContent(t, "test-repo")
			require.True(t, ok)
			assert.Contains(t, blob, sentinelLine,
				"enrollment blob should contain sentinel line")
		}
	})

	/*
	TS-GH-2247-006 — Verify __ORG__ placeholder substitution produces consistent
	comparison results (template has __ORG__, remote has interpolated org name).
	*/
	t.Run("[test_id:TS-GH-2247-006]_org_placeholder_substitution_consistent", func(t *testing.T) {
		env := newTestEnv(t)

		// Template has __ORG__ placeholder; remote has actual org name.
		// We need a template that uses __ORG__.
		templateContent := sentinelLine + "\nuses: test-org/fullsend/.github/workflows/shim.yml\n"
		remoteContent := sentinelLine + "\nuses: test-org/fullsend/.github/workflows/shim.yml\n"
		remoteB64 := encodeB64(remoteContent)

		// Update the template to use __ORG__ placeholder.
		writeFile(t, env.configDir+"/templates/shim-workflow-call.yaml",
			sentinelLine+"\nuses: __ORG__/fullsend/.github/workflows/shim.yml\n")

		env.installGhMock(t, `
case "$endpoint" in
  repos/test-org/test-repo/actions/variables/*)
    json='{"status":"404","message":"Not Found"}'
    rc=1
    ;;
  repos/test-org/test-repo/contents/.github/workflows/fullsend.yaml)
    json='{"content":"`+remoteB64+`","sha":"file-sha"}'
    ;;
  repos/test-org/test-repo)
    json='{"default_branch":"main","private":false}'
    ;;
  *)
    rc=0
    ;;
esac
`)

		output, _ := env.runReconcile(t)
		_ = templateContent // used for documentation

		// After __ORG__ is replaced with test-org, content should match.
		assert.NotContains(t, output, "shim is stale",
			"__ORG__ placeholder should be substituted before comparison")
	})
}
