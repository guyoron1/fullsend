package scaffold

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

/*
Stale Shim Detection Tests — Genuine Drift Triggers Update PR

STP Reference: outputs/stp/GH-77/GH-77_test_plan.md
Jira: GH-77

These tests verify that genuinely stale shims (where the managed content
has actually changed) are correctly detected and trigger update PRs.
*/

func TestStaleShimDetection(t *testing.T) {
	t.Run("[test_id:TS-GH77-004] should trigger update PR for genuinely stale shim", func(t *testing.T) {
		h := newReconcileHarness(t)

		// Remote shim has user header + sentinel + stale managed content.
		remoteContent := "# Copyright 2026 Conforma\n# SPDX-License-Identifier: Apache-2.0\n# --- fullsend managed below - do not edit ---\nstale shim template\n"
		remoteB64 := b64encode(remoteContent)

		h.writeGHMock(ghMockOpts{
			prBlock: `
case "$2" in
  list) exit 0 ;;
  create) echo "https://github.com/test-org/test-repo/pull/99"; exit 0 ;;
  close) exit 0 ;;
esac
exit 0`,
			apiCases: fmt.Sprintf(`repos/test-org/test-repo/contents/*)
    json='{"content":"%s","sha":"file-sha"}'
    ;;
`, remoteB64),
		})

		output, _ := h.run()

		assert.Contains(t, output, "shim is stale",
			"genuinely stale shim should be detected")

		// Verify blob is created with fresh template content.
		assert.True(t, h.blobExists("test-repo"),
			"blob should be created for stale shim update")
		blobDecoded := h.blobContent("test-repo")
		assert.Contains(t, blobDecoded, "fresh shim template",
			"blob should contain the updated template content")

		// Verify user header is preserved.
		assert.Contains(t, blobDecoded, "# Copyright 2026 Conforma",
			"user license header should be preserved in the updated blob")

		// Verify PR was created.
		assert.Contains(t, output, "Created shim update PR")

		// Verify UPDATED counter.
		assert.Contains(t, output, "Updated (stale shim): 1")
	})

	t.Run("[test_id:TS-GH77-005] should detect stale shim after template content change", func(t *testing.T) {
		h := newReconcileHarness(t)

		// Remote has correct sentinel but different managed body (old version).
		remoteContent := "# --- fullsend managed below - do not edit ---\nold workflow version v1\n"
		remoteB64 := b64encode(remoteContent)

		h.writeGHMock(ghMockOpts{
			prBlock: `
case "$2" in
  list) exit 0 ;;
  create) echo "https://github.com/test-org/test-repo/pull/99"; exit 0 ;;
  close) exit 0 ;;
esac
exit 0`,
			apiCases: fmt.Sprintf(`repos/test-org/test-repo/contents/*)
    json='{"content":"%s","sha":"file-sha"}'
    ;;
`, remoteB64),
		})

		output, _ := h.run()

		assert.Contains(t, output, "shim is stale",
			"template body change should be detected as drift")
	})

	t.Run("[test_id:TS-GH77-006] should handle error when update PR creation fails", func(t *testing.T) {
		h := newReconcileHarness(t)

		// Remote has stale content to trigger update path.
		remoteContent := "# --- fullsend managed below - do not edit ---\nstale shim template\n"
		remoteB64 := b64encode(remoteContent)

		h.writeGHMock(ghMockOpts{
			prBlock: `
case "$2" in
  list) exit 0 ;;
  create)
    echo "Permission denied" >&2
    exit 1 ;;
  close) exit 0 ;;
esac
exit 0`,
			apiCases: fmt.Sprintf(`repos/test-org/test-repo/contents/*)
    json='{"content":"%s","sha":"file-sha"}'
    ;;
`, remoteB64),
		})

		output, exitCode := h.run()

		assert.NotEqual(t, 0, exitCode,
			"script should exit with non-zero code when PR creation fails")
		assert.Contains(t, output, "::error::Failed to create",
			"error annotation should be emitted for failed PR creation")
		assert.Contains(t, output, "Failed: 1",
			"FAILED counter should be incremented")
	})
}
