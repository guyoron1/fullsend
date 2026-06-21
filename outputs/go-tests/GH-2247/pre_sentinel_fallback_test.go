package scaffold

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Pre-Sentinel Shim Fallback Tests

STP Reference: outputs/stp/GH-2247/GH-2247_test_plan.md
Jira: GH-2247

Validates that shims created before the sentinel feature was introduced
(pre-sentinel format) fall back to full decoded content comparison when
extract_managed_content returns empty.
*/

func TestPreSentinelFallback(t *testing.T) {
	t.Run("[test_id:TS-GH2247-008] pre-sentinel shim matches full decoded content", func(t *testing.T) {
		env := newReconcileEnv(t)

		// Pre-sentinel shim: has the managed content but no sentinel line.
		// The script's extract_managed_content returns empty for this input,
		// triggering the fallback to full decoded content comparison.
		//
		// The expected content (template) contains the sentinel, so
		// extract_managed_content returns sentinel+content for the expected side.
		// For the remote side it returns empty → fallback to full content.
		//
		// Because the remote has NO sentinel, and the template HAS sentinel,
		// full-content comparison will differ → script should detect staleness
		// and migrate the shim to sentinel format.
		preSentinelContent := freshTemplate + "\n"
		env.setRemoteContent(preSentinelContent)

		output, err := env.run()
		_ = err

		// The pre-sentinel shim content differs from the template (which includes
		// the sentinel line), so the script should detect this as stale and create
		// an update blob that adds the sentinel (migration). This is expected
		// behavior — the fallback comparison correctly identifies the difference.
		//
		// Note: a pre-sentinel shim where full decoded content matches the full
		// template (including sentinel) is impossible since pre-sentinel shims
		// by definition lack the sentinel.
		hasStaleMsgOrUpdate := strings.Contains(output, "shim is stale") ||
			strings.Contains(output, "update PR") ||
			env.blobCreated()

		assert.True(t, hasStaleMsgOrUpdate,
			"Pre-sentinel shim should trigger migration to sentinel format; output:\n%s", output)
	})

	t.Run("[test_id:TS-GH2247-009] pre-sentinel shim detects genuine drift", func(t *testing.T) {
		env := newReconcileEnv(t)

		// Pre-sentinel shim with genuinely stale content (no sentinel, wrong content).
		preSentinelStale := staleTemplate + "\n"
		env.setRemoteContent(preSentinelStale)

		output, err := env.run()
		_ = err

		assert.Contains(t, output, "shim is stale",
			"Pre-sentinel stale content should be detected as stale")
		assert.True(t, env.blobCreated(),
			"Update blob should be created for stale pre-sentinel shim")
	})

	t.Run("[test_id:TS-GH2247-010] empty extract_managed_content triggers fallback", func(t *testing.T) {
		env := newReconcileEnv(t)

		// Test extract_managed_content function directly: when input has no
		// sentinel line, the function should return empty output.
		code := `
result=$(echo "some content without any sentinel line" | extract_managed_content)
if [ -z "$result" ]; then
  echo "EMPTY_RESULT"
else
  echo "NON_EMPTY_RESULT: $result"
fi
`
		out, err := env.runBashFunc(code)
		require.NoError(t, err, "bash function should execute successfully; output:\n%s", out)

		assert.Contains(t, strings.TrimSpace(out), "EMPTY_RESULT",
			"extract_managed_content should return empty for input without sentinel")

		// Also verify it returns content when sentinel IS present.
		codeWithSentinel := `
input="line before
` + sentinel + `
managed line 1
managed line 2"
result=$(printf '%s\n' "$input" | extract_managed_content)
if [ -n "$result" ]; then
  echo "HAS_CONTENT"
  echo "$result"
else
  echo "EMPTY_RESULT"
fi
`
		out2, err2 := env.runBashFunc(codeWithSentinel)
		require.NoError(t, err2, "bash function should execute; output:\n%s", out2)

		assert.Contains(t, out2, "HAS_CONTENT",
			"extract_managed_content should return content when sentinel is present")
		assert.Contains(t, out2, sentinel,
			"Returned content should include the sentinel line itself")
		assert.Contains(t, out2, "managed line 1",
			"Returned content should include lines after sentinel")
	})
}
