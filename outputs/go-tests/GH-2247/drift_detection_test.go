package scaffold

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Shim Drift Detection Tests — Encoding Normalization

STP Reference: outputs/stp/GH-2247/GH-2247_test_plan.md
Jira: GH-2247

Validates that the decoded text comparison in reconcile-repos.sh correctly
identifies logically identical content as up-to-date, regardless of encoding
differences (trailing newlines, carriage returns).
*/

func TestDriftDetection_EncodingNormalization(t *testing.T) {
	t.Run("[test_id:TS-GH2247-001] identical content with extra trailing newline not flagged stale", func(t *testing.T) {
		env := newReconcileEnv(t)

		// Remote content is identical to template but has an extra trailing newline.
		// This produces different base64 but the decoded text (after normalization)
		// should match. This is the root cause of GH-2247.
		templateContent := sentinel + "\n" + freshTemplate + "\n"
		remoteContent := sentinel + "\n" + freshTemplate + "\n\n" // extra trailing newline

		// Verify the base64 representations are indeed different (precondition).
		templateB64 := base64.StdEncoding.EncodeToString([]byte(templateContent))
		remoteB64 := base64.StdEncoding.EncodeToString([]byte(remoteContent))
		require.NotEqual(t, templateB64, remoteB64, "precondition: base64 should differ due to extra newline")

		env.setRemoteContent(remoteContent)
		output, err := env.run()
		require.NoError(t, err, "reconcile-repos.sh should exit 0; output:\n%s", output)

		assert.Contains(t, output, "already enrolled (shim up to date)",
			"Script should recognize identical content as up-to-date")
		assert.NotContains(t, output, "shim is stale",
			"Script should NOT flag identical content as stale")
		assert.False(t, env.blobCreated(),
			"No blob should be created for identical content")
	})

	t.Run("[test_id:TS-GH2247-002] identical content with no trailing newline not flagged stale", func(t *testing.T) {
		env := newReconcileEnv(t)

		// Remote content has no trailing newline — raw bytes end immediately
		// after the last content character.
		remoteContent := sentinel + "\n" + freshTemplate // no trailing \n

		env.setRemoteContent(remoteContent)
		output, err := env.run()
		require.NoError(t, err, "reconcile-repos.sh should exit 0; output:\n%s", output)

		assert.Contains(t, output, "already enrolled (shim up to date)",
			"Script should recognize content without trailing newline as matching")
		assert.False(t, env.blobCreated(),
			"No blob should be created")
	})

	t.Run("[test_id:TS-GH2247-003] genuinely different content is flagged stale", func(t *testing.T) {
		env := newReconcileEnv(t)

		// Remote content has genuinely different managed content.
		remoteContent := sentinel + "\n" + staleTemplate + "\n"
		env.setRemoteContent(remoteContent)

		output, err := env.run()
		// The script may exit 0 even when creating an update PR.
		_ = err

		assert.Contains(t, output, "shim is stale",
			"Script should detect genuinely different content as stale")
		assert.True(t, env.blobCreated(),
			"A blob should be created for the update PR")
	})

	t.Run("[test_id:TS-GH2247-004] carriage return differences ignored in comparison", func(t *testing.T) {
		env := newReconcileEnv(t)

		// Remote content has CRLF line endings instead of LF.
		// The script normalizes with tr -d '\r' before comparison.
		remoteContent := sentinel + "\r\n" + freshTemplate + "\r\n"
		env.setRemoteContent(remoteContent)

		output, err := env.run()
		require.NoError(t, err, "reconcile-repos.sh should exit 0; output:\n%s", output)

		assert.NotContains(t, output, "shim is stale",
			"CRLF differences should not trigger false positive drift detection")

		// Verify the script did not create any blob for this false positive.
		for _, call := range env.ghCalls() {
			assert.False(t, strings.Contains(call, "git/blobs"),
				"No blob API call should be made for CRLF-only differences")
		}
	})
}
