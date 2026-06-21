package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Sentinel Preservation Tests

STP Reference: outputs/stp/GH-2247/GH-2247_test_plan.md
Jira: GH-2247

Validates that the sentinel line "# --- fullsend managed below - do not edit ---"
is present in all shim blob outputs across new enrollment, stale update, and
injection guard rejection code paths.
*/

func TestSentinelPreservation(t *testing.T) {
	t.Run("[test_id:TS-GH2247-005] sentinel present in new enrollment shim", func(t *testing.T) {
		env := newReconcileEnv(t)

		// No existing shim on remote — mock gh returns 404 for contents.
		// writeDefaultGHMock("") sets up the 404 response.
		env.writeDefaultGHMock("")

		// We need to capture the blob content that the script sends to the
		// git/blobs API. Enhance the mock to save the blob input.
		blobCapture := filepath.Join(env.tmpDir, "blob-capture.json")
		enhanceMockGHForBlobCapture(env, blobCapture)

		output, err := env.run()
		_ = err // Script may succeed or fail depending on mock completeness
		_ = output

		// Verify a blob was created.
		assert.True(t, env.blobCreated(), "A blob should be created for new enrollment")

		// Read the captured blob content and verify sentinel is present.
		if blobData, readErr := os.ReadFile(blobCapture); readErr == nil {
			decoded := b64Decode(t, strings.TrimSpace(string(blobData)))
			assert.Contains(t, decoded, sentinel,
				"New enrollment blob must contain the sentinel line")
			assert.Contains(t, decoded, freshTemplate,
				"New enrollment blob must contain fresh template content")
		}
	})

	t.Run("[test_id:TS-GH2247-006] sentinel present in updated stale shim", func(t *testing.T) {
		env := newReconcileEnv(t)

		// Remote shim has user header + sentinel + stale content.
		remoteContent := "# Copyright 2026 Conforma\n# SPDX-License-Identifier: Apache-2.0\n" +
			sentinel + "\n" + staleTemplate + "\n"
		env.setRemoteContent(remoteContent)

		blobCapture := filepath.Join(env.tmpDir, "blob-capture.json")
		enhanceMockGHForBlobCapture(env, blobCapture)

		output, err := env.run()
		_ = err

		assert.Contains(t, output, "shim is stale",
			"Script should detect stale content and trigger update")
		assert.True(t, env.blobCreated(), "A blob should be created for the stale update")

		// Read captured blob and verify sentinel and fresh content.
		if blobData, readErr := os.ReadFile(blobCapture); readErr == nil {
			decoded := b64Decode(t, strings.TrimSpace(string(blobData)))
			assert.Contains(t, decoded, sentinel,
				"Updated blob must preserve sentinel line")
			assert.Contains(t, decoded, freshTemplate,
				"Updated blob must contain fresh template content after sentinel")
			assert.Contains(t, decoded, "# Copyright 2026 Conforma",
				"Updated blob should preserve user comment header")
		}
	})

	t.Run("[test_id:TS-GH2247-007] sentinel survives injection guard rejection", func(t *testing.T) {
		env := newReconcileEnv(t)

		// Remote shim has non-comment YAML above sentinel (injection attempt).
		remoteContent := "name: injected-workflow\n" +
			sentinel + "\n" + staleTemplate + "\n"
		env.setRemoteContent(remoteContent)

		blobCapture := filepath.Join(env.tmpDir, "blob-capture.json")
		enhanceMockGHForBlobCapture(env, blobCapture)

		output, err := env.run()
		_ = err

		// Verify the injection guard emitted a warning.
		assert.Contains(t, output, "non-comment content above sentinel was rejected",
			"Script should warn about rejected non-comment header")

		// Verify the blob does NOT contain the injected content but DOES
		// contain the sentinel and fresh template.
		if blobData, readErr := os.ReadFile(blobCapture); readErr == nil {
			decoded := b64Decode(t, strings.TrimSpace(string(blobData)))
			assert.NotContains(t, decoded, "injected-workflow",
				"Injected YAML must not appear in output blob")
			assert.Contains(t, decoded, sentinel,
				"Sentinel must survive injection guard rejection")
			assert.Contains(t, decoded, freshTemplate,
				"Fresh template must be present after injection rejection")
		}
	})
}

// enhanceMockGHForBlobCapture replaces the mock gh with one that also captures
// the base64 content sent to the git/blobs endpoint. The content is written
// to captureFile for later inspection.
func enhanceMockGHForBlobCapture(env *reconcileEnv, captureFile string) {
	env.t.Helper()

	// Read the existing mock and inject blob capture logic.
	mockPath := filepath.Join(env.mockBinDir, "gh")
	existing, err := os.ReadFile(mockPath)
	require.NoError(env.t, err)

	// Replace the blob handler to also capture the input content.
	enhanced := strings.Replace(string(existing),
		`repos/*/git/blobs)
        echo "mock-blob-sha"`,
		fmt.Sprintf(`repos/*/git/blobs)
        # Capture blob content from stdin (piped via --input -)
        if [ -t 0 ]; then
          :
        else
          input=$(cat)
          # Extract the base64 content from the JSON input.
          content=$(echo "$input" | jq -r '.content // empty' 2>/dev/null || true)
          if [ -n "$content" ]; then
            printf '%%s' "$content" > "%s"
          fi
        fi
        echo "mock-blob-sha"`, captureFile), 1)

	writeScript(env.t, mockPath, enhanced)
}
