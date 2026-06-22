package scaffold

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

/*
Shim Drift Detection Tests — Encoding-Insensitive Comparison

STP Reference: outputs/stp/GH-77/GH-77_test_plan.md
Jira: GH-77

These tests verify the fix for issue #2247: the old managed_content_b64()
comparison re-encoded content to base64, amplifying trivial trailing newline
differences into mismatched base64 strings. The fix compares decoded text
instead of re-encoded base64.
*/

func TestShimDriftDetection(t *testing.T) {
	t.Run("[test_id:TS-GH77-001] should not flag identical content with different trailing newlines as stale", func(t *testing.T) {
		h := newReconcileHarness(t)

		// The remote has the same template content but with an extra trailing newline,
		// which produces different base64 from shim_content_b64(). This simulates
		// encoding differences from the GitHub Content API.
		templateContent := "# --- fullsend managed below - do not edit ---\nfresh shim template\n"
		remoteContent := templateContent + "\n" // extra trailing newline
		remoteB64 := b64encode(remoteContent)

		h.writeGHMock(ghMockOpts{
			prBlock: `exit 0`,
			apiCases: fmt.Sprintf(`repos/test-org/test-repo/contents/*)
    json='{"content":"%s","sha":"file-sha"}'
    ;;
`, remoteB64),
		})

		output, exitCode := h.run()

		assert.Equal(t, 0, exitCode, "script should exit successfully")
		assert.Contains(t, output, "already enrolled (shim up to date)",
			"identical content with trailing newline difference should be recognized as up-to-date")
		assert.NotContains(t, output, "shim is stale",
			"identical content should NOT be flagged as stale")
		assert.False(t, h.blobExists("test-repo"),
			"no blob should be created for encoding-only differences")
	})

	t.Run("[test_id:TS-GH77-002] should produce already enrolled status for up-to-date shim", func(t *testing.T) {
		h := newReconcileHarness(t)

		// Remote shim includes user header + sentinel + matching managed portion.
		remoteContent := "# Copyright 2026 Conforma\n# SPDX-License-Identifier: Apache-2.0\n# --- fullsend managed below - do not edit ---\nfresh shim template\n"
		remoteB64 := b64encode(remoteContent)

		h.writeGHMock(ghMockOpts{
			prBlock: `exit 0`,
			apiCases: fmt.Sprintf(`repos/test-org/test-repo/contents/*)
    json='{"content":"%s","sha":"file-sha"}'
    ;;
`, remoteB64),
		})

		output, exitCode := h.run()

		assert.Equal(t, 0, exitCode, "script should exit successfully")
		assert.Contains(t, output, "already enrolled (shim up to date)",
			"up-to-date shim should be recognized as current")
		assert.Contains(t, output, "Skipped (already reconciled): 1",
			"SKIPPED counter should be incremented")
		assert.False(t, h.blobExists("test-repo"),
			"no blob or PR should be created for up-to-date shim")
	})

	t.Run("[test_id:TS-GH77-003] should not create blob or PR for encoding-only differences", func(t *testing.T) {
		h := newReconcileHarness(t)

		// Remote has identical text but with trailing newline variation,
		// causing different base64 encoding.
		templateContent := "# --- fullsend managed below - do not edit ---\nfresh shim template\n"
		remoteContent := templateContent + "\n" // trailing newline diff
		remoteB64 := b64encode(remoteContent)

		h.writeGHMock(ghMockOpts{
			prBlock: `exit 0`,
			apiCases: fmt.Sprintf(`repos/test-org/test-repo/contents/*)
    json='{"content":"%s","sha":"file-sha"}'
    ;;
`, remoteB64),
		})

		output, exitCode := h.run()

		assert.Equal(t, 0, exitCode, "script should exit successfully")

		// Verify no blob creation.
		assert.False(t, h.blobExists("test-repo"),
			"no blob-input JSON should exist for encoding-only differences")

		// Verify no git/blobs endpoint called.
		ghLog := h.ghCallsLog()
		assert.NotContains(t, ghLog, "git/blobs",
			"no git/blobs API call should be made")

		// Verify no PR creation.
		assert.NotContains(t, ghLog, "pr create",
			"no PR creation should occur for encoding-only differences")

		// Verify the repo was recognized as up-to-date.
		assert.Contains(t, output, "already enrolled (shim up to date)")
	})
}
