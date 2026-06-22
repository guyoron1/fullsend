package scaffold

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

/*
Up-to-Date Shim Skip Behavior Tests

STP Reference: outputs/stp/GH-77/GH-77_test_plan.md
Jira: GH-77

These tests verify that up-to-date shims are correctly skipped: no blob
creation, no API writes, and the SKIPPED counter is incremented.
*/

func TestUpToDateShimSkipBehavior(t *testing.T) {
	t.Run("[test_id:TS-GH77-010] should not create blob for up-to-date shim", func(t *testing.T) {
		h := newReconcileHarness(t)

		// Remote content exactly matches the template.
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

		assert.Equal(t, 0, exitCode)
		assert.Contains(t, output, "already enrolled (shim up to date)")

		// Verify no blob creation.
		assert.False(t, h.blobExists("test-repo"),
			"no blob-input JSON should exist for up-to-date shim")

		// Verify no git/blobs endpoint called.
		ghLog := h.ghCallsLog()
		assert.NotContains(t, ghLog, "git/blobs",
			"no git/blobs API call should be made for up-to-date shim")
	})

	t.Run("[test_id:TS-GH77-011] should increment skip counter for current shim", func(t *testing.T) {
		h := newReconcileHarness(t)

		// Configure two repos: one up-to-date, one stale.
		h.writeConfig(`version: 1
repos:
  uptodate-repo:
    enabled: true
  stale-repo:
    enabled: true
`)
		h.writeMockYQ([]string{"uptodate-repo", "stale-repo"}, nil)

		uptodateContent := "# --- fullsend managed below - do not edit ---\nfresh shim template\n"
		uptodateB64 := b64encode(uptodateContent)
		staleContent := "# --- fullsend managed below - do not edit ---\nstale shim template\n"
		staleB64 := b64encode(staleContent)

		h.writeGHMock(ghMockOpts{
			prBlock: `
case "$2" in
  list) exit 0 ;;
  create) echo "https://github.com/test-org/mock/pull/99"; exit 0 ;;
  close) exit 0 ;;
esac
exit 0`,
			apiCases: fmt.Sprintf(`repos/test-org/uptodate-repo/contents/*)
    json='{"content":"%s","sha":"file-sha"}'
    ;;
  repos/test-org/stale-repo/contents/*)
    json='{"content":"%s","sha":"file-sha"}'
    ;;
`, uptodateB64, staleB64),
		})

		output, _ := h.run()

		// Verify the summary shows at least 1 skipped repo.
		assert.Contains(t, output, "Skipped (already reconciled): 1",
			"SKIPPED counter should reflect the up-to-date repo")

		// Verify updated counter for the stale repo.
		assert.Contains(t, output, "Updated (stale shim): 1",
			"UPDATED counter should reflect the stale repo")
	})
}
