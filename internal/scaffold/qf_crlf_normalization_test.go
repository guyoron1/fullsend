package scaffold

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

/*
CR/LF Normalization Tests — Cross-Platform Drift Prevention

STP Reference: outputs/stp/GH-77/GH-77_test_plan.md
Jira: GH-77

These tests verify that the tr -d '\r' normalization in reconcile-repos.sh
correctly handles Windows-style line endings, preventing false-positive
drift detection from CR/LF differences.
*/

func TestCRLFNormalization(t *testing.T) {
	t.Run("[test_id:TS-GH77-012] should normalize CRLF content before comparison", func(t *testing.T) {
		h := newReconcileHarness(t)

		// Remote content has CRLF line endings throughout.
		remoteContent := "# --- fullsend managed below - do not edit ---\r\nfresh shim template\r\n"
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
			"CRLF content should be recognized as up-to-date after normalization")
		assert.NotContains(t, output, "shim is stale",
			"CRLF differences should not cause false drift")
	})

	t.Run("[test_id:TS-GH77-013] should handle mixed line endings correctly", func(t *testing.T) {
		h := newReconcileHarness(t)

		// Remote content has mixed line endings: first line CRLF, second line LF.
		remoteContent := "# --- fullsend managed below - do not edit ---\r\nfresh shim template\n"
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
		assert.NotContains(t, output, "shim is stale",
			"mixed line endings should not cause false drift")
		assert.Contains(t, output, "already enrolled (shim up to date)",
			"mixed-ending content should be recognized as up-to-date")
	})
}
