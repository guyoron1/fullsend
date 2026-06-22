package scaffold

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

/*
Content-Injection Guard Tests — YAML Injection Prevention

STP Reference: outputs/stp/GH-77/GH-77_test_plan.md
Jira: GH-77

These tests verify that the content-injection guard in shim_with_header_b64()
correctly rejects non-comment YAML above the sentinel line while preserving
legitimate comment-only headers (e.g., license headers).
*/

func TestContentInjectionGuard(t *testing.T) {
	t.Run("[test_id:TS-GH77-014] should reject non-comment YAML above sentinel", func(t *testing.T) {
		h := newReconcileHarness(t)

		// Remote shim has non-comment YAML key injected above sentinel.
		remoteContent := "name: injected-workflow\n# --- fullsend managed below - do not edit ---\nstale shim template\n"
		remoteB64 := b64encode(remoteContent)

		h.writeGHMock(ghMockOpts{
			prBlock: `
case "$2" in
  list)
    for arg in "$@"; do
      if [[ "$arg" == "fullsend/onboard" ]]; then
        echo "https://github.com/test-org/test-repo/pull/99"
      fi
    done
    exit 0 ;;
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

		// Verify blob was created (content was stale).
		assert.True(t, h.blobExists("test-repo"),
			"blob should be created for injection-guarded shim update")

		blobDecoded := h.blobContent("test-repo")

		// Verify injected YAML was stripped.
		assert.NotContains(t, blobDecoded, "injected-workflow",
			"injected YAML key should be stripped from blob content")

		// Verify warning was emitted.
		assert.Contains(t, output, "non-comment content above sentinel was rejected",
			"warning log should be emitted for rejected content")

		// Verify blob still contains valid template.
		assert.Contains(t, blobDecoded, "# --- fullsend managed below - do not edit ---",
			"sentinel line should be present in the updated blob")
		assert.Contains(t, blobDecoded, "fresh shim template",
			"fresh template content should be present after guard")
	})

	t.Run("[test_id:TS-GH77-015] should preserve comment-only header during update", func(t *testing.T) {
		h := newReconcileHarness(t)

		// Remote shim has comment-only header (license) + sentinel + stale managed content.
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

		// Verify stale detection.
		assert.Contains(t, output, "shim is stale",
			"stale managed content should be detected")

		// Verify blob was created with preserved header.
		assert.True(t, h.blobExists("test-repo"),
			"blob should be created for stale shim update")
		blobDecoded := h.blobContent("test-repo")

		assert.Contains(t, blobDecoded, "# Copyright 2026 Conforma",
			"comment header should be preserved in updated blob")
		assert.Contains(t, blobDecoded, "# SPDX-License-Identifier: Apache-2.0",
			"SPDX identifier should be preserved")
		assert.Contains(t, blobDecoded, "# --- fullsend managed below - do not edit ---",
			"sentinel line should be present")
		assert.Contains(t, blobDecoded, "fresh shim template",
			"managed section should be updated with fresh template")
		assert.NotContains(t, blobDecoded, "stale shim template",
			"old managed content should be replaced")
	})
}
