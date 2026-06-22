package scaffold

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

/*
Pre-Sentinel Shim Fallback Tests — Full Decoded Content Comparison

STP Reference: outputs/stp/GH-77/GH-77_test_plan.md
Jira: GH-77

These tests verify behavior when the remote shim has no sentinel line
(pre-sentinel shim from before the header-preservation feature). The
script falls back to comparing full decoded content.
*/

func TestPreSentinelShimFallback(t *testing.T) {
	t.Run("[test_id:TS-GH77-007] should compare full decoded content for pre-sentinel shim", func(t *testing.T) {
		h := newReconcileHarness(t)

		// Pre-sentinel shim: no sentinel line, stale content.
		remoteContent := "stale shim template\n"
		remoteB64 := b64encode(remoteContent)

		h.writeGHMock(ghMockOpts{
			prBlock: `
case "$2" in
  list)
    # Check --head flag for existing PR.
    for arg in "$@"; do
      if [[ "$arg" == "fullsend/onboard" ]]; then
        echo "https://github.com/test-org/test-repo/pull/42"
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

		assert.Contains(t, output, "shim is stale",
			"pre-sentinel shim with different content should be flagged as stale")

		// Verify blob is created with sentinel + fresh template (migration).
		assert.True(t, h.blobExists("test-repo"),
			"blob should be created for pre-sentinel shim update")
		blobDecoded := h.blobContent("test-repo")
		assert.Contains(t, blobDecoded, "# --- fullsend managed below - do not edit ---",
			"updated blob should include sentinel line (migration to new format)")
		assert.Contains(t, blobDecoded, "fresh shim template",
			"updated blob should contain fresh template content")
		assert.NotContains(t, blobDecoded, "stale shim template",
			"old content should NOT be duplicated in the blob")
	})

	t.Run("[test_id:TS-GH77-008] should not flag pre-sentinel shim with identical content as stale", func(t *testing.T) {
		h := newReconcileHarness(t)

		// Pre-sentinel shim whose content matches the full template
		// (sentinel + fresh template). No user header.
		remoteContent := "# --- fullsend managed below - do not edit ---\nfresh shim template\n"
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
			"matching pre-sentinel shim should be recognized as current")
		assert.False(t, h.blobExists("test-repo"),
			"no blob should be created for matching pre-sentinel shim")
	})

	t.Run("[test_id:TS-GH77-009] should flag pre-sentinel shim with different content as stale", func(t *testing.T) {
		h := newReconcileHarness(t)

		// Pre-sentinel shim with completely different body.
		remoteContent := "old workflow template v0\n"
		remoteB64 := b64encode(remoteContent)

		h.writeGHMock(ghMockOpts{
			prBlock: `
case "$2" in
  list)
    for arg in "$@"; do
      if [[ "$arg" == "fullsend/onboard" ]]; then
        echo "https://github.com/test-org/test-repo/pull/42"
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

		assert.Contains(t, output, "shim is stale",
			"diverged pre-sentinel shim should be flagged as stale")

		// Verify blob has fresh template.
		assert.True(t, h.blobExists("test-repo"),
			"blob should be created for diverged pre-sentinel shim")
		blobDecoded := h.blobContent("test-repo")
		assert.Contains(t, blobDecoded, "fresh shim template",
			"blob should contain fresh template content")
	})
}
