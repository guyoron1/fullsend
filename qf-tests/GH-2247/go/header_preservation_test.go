//go:build e2e

package scaffold_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Header Preservation Tests

STP Reference: outputs/stp/GH-2247/GH-2247_test_plan.md
Jira: GH-2247

Requirement: User-owned content above the sentinel (e.g., license headers, SPDX
identifiers) is preserved when the managed portion is updated.
*/

func TestHeaderPreservation(t *testing.T) {

	/*
	TS-GH-2247-010 — Verify license header is preserved when updating a stale shim.
	Loss of license headers would create compliance issues.
	*/
	t.Run("[test_id:TS-GH-2247-010]_license_header_preserved_in_update", func(t *testing.T) {
		env := newTestEnv(t)

		// Remote has copyright + SPDX header above sentinel, and stale managed content.
		remoteContent := "# Copyright 2026 Conforma\n# SPDX-License-Identifier: Apache-2.0\n" +
			sentinelLine + "\nstale shim template\n"
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

		// Verify copyright is preserved above sentinel.
		assert.Contains(t, blob, "# Copyright 2026 Conforma",
			"copyright line must be preserved in update blob")
		assert.Contains(t, blob, "# SPDX-License-Identifier: Apache-2.0",
			"SPDX identifier must be preserved in update blob")

		// Verify header appears ABOVE sentinel.
		copyrightIdx := strings.Index(blob, "# Copyright")
		sentinelIdx := strings.Index(blob, sentinelLine)
		assert.Less(t, copyrightIdx, sentinelIdx,
			"copyright must appear above sentinel line")

		// Verify managed content was updated.
		assert.Contains(t, blob, freshTemplate,
			"managed content should be updated to fresh template")
		assert.NotContains(t, blob, "stale shim template",
			"stale managed content should be replaced")
	})

	/*
	TS-GH-2247-011 — Verify multi-line comment header (5+ lines) is fully
	preserved without truncation or corruption.
	*/
	t.Run("[test_id:TS-GH-2247-011]_multi_line_comment_header_preserved", func(t *testing.T) {
		env := newTestEnv(t)

		headerLines := []string{
			"# Copyright 2026 Example Corp",
			"# Licensed under the Apache License, Version 2.0",
			"# SPDX-License-Identifier: Apache-2.0",
			"# This file is managed by FullSend",
			"# See: https://example.com/docs",
			"# Maintainer: team@example.com",
		}
		header := strings.Join(headerLines, "\n") + "\n"
		remoteContent := header + sentinelLine + "\nstale shim template\n"
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

		// Verify all 6 header lines are preserved in order.
		for _, line := range headerLines {
			assert.Contains(t, blob, line,
				"header line should be preserved: %s", line)
		}

		// Verify ordering: each line should appear before the next.
		for i := 0; i < len(headerLines)-1; i++ {
			idxCurrent := strings.Index(blob, headerLines[i])
			idxNext := strings.Index(blob, headerLines[i+1])
			assert.Less(t, idxCurrent, idxNext,
				"header lines should maintain order: %q before %q",
				headerLines[i], headerLines[i+1])
		}

		// All header lines should be above the sentinel.
		sentinelIdx := strings.Index(blob, sentinelLine)
		lastHeaderIdx := strings.Index(blob, headerLines[len(headerLines)-1])
		assert.Less(t, lastHeaderIdx, sentinelIdx,
			"all header lines must appear above sentinel")
	})

	/*
	TS-GH-2247-012 — Verify blank-only header (whitespace lines above sentinel)
	is discarded rather than preserved.
	*/
	t.Run("[test_id:TS-GH-2247-012]_blank_only_header_discarded", func(t *testing.T) {
		env := newTestEnv(t)

		// Remote has only blank/whitespace lines above sentinel.
		remoteContent := "   \n\n  \n" + sentinelLine + "\nstale shim template\n"
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

		// Verify no blank lines appear before the sentinel.
		sentinelIdx := strings.Index(blob, sentinelLine)
		require.True(t, sentinelIdx >= 0, "blob should contain sentinel")

		beforeSentinel := blob[:sentinelIdx]
		trimmed := strings.TrimSpace(beforeSentinel)
		assert.Empty(t, trimmed,
			"blank-only header should be discarded; got content before sentinel: %q", beforeSentinel)
	})
}
