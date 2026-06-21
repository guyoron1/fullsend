package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
User-Owned Header Preservation Tests

STP Reference: outputs/stp/GH-2247/GH-2247_test_plan.md
Jira: GH-2247

Validates that comment headers above the sentinel (e.g., copyright notices,
SPDX identifiers) are preserved during shim updates, and non-comment content
injection above the sentinel is rejected with a warning.
*/

func TestUserHeaderPreservation(t *testing.T) {
	t.Run("[test_id:TS-GH2247-014] comment header preserved above sentinel", func(t *testing.T) {
		env := newReconcileEnv(t)

		// Remote shim has copyright + SPDX comment lines above sentinel,
		// and stale managed content below sentinel (triggers update).
		remoteContent := "# Copyright 2026 Conforma\n" +
			"# SPDX-License-Identifier: Apache-2.0\n" +
			sentinel + "\n" +
			staleTemplate + "\n"
		env.setRemoteContent(remoteContent)

		blobCapture := filepath.Join(env.tmpDir, "blob-capture.json")
		enhanceMockGHForBlobCapture(env, blobCapture)

		output, err := env.run()
		_ = err

		// The script should detect stale content and update.
		assert.Contains(t, output, "shim is stale",
			"Script should detect stale managed content")

		// Read the captured blob and verify headers are preserved.
		blobData, readErr := os.ReadFile(blobCapture)
		require.NoError(t, readErr, "Blob capture file should exist")

		decoded := b64Decode(t, strings.TrimSpace(string(blobData)))

		// Copyright header preserved.
		assert.Contains(t, decoded, "# Copyright 2026 Conforma",
			"Copyright comment must be preserved in output blob")

		// SPDX header preserved.
		assert.Contains(t, decoded, "# SPDX-License-Identifier: Apache-2.0",
			"SPDX license header must be preserved in output blob")

		// Sentinel present after headers.
		assert.Contains(t, decoded, sentinel,
			"Sentinel line must be present after comment headers")

		// Fresh template content after sentinel.
		assert.Contains(t, decoded, freshTemplate,
			"Fresh template content must follow the sentinel")

		// Verify ordering: headers come before sentinel.
		headerIdx := strings.Index(decoded, "# Copyright 2026 Conforma")
		sentinelIdx := strings.Index(decoded, sentinel)
		assert.Less(t, headerIdx, sentinelIdx,
			"Comment headers must appear before the sentinel line")
	})

	t.Run("[test_id:TS-GH2247-015] non-comment content above sentinel rejected", func(t *testing.T) {
		env := newReconcileEnv(t)

		// Remote shim has non-comment YAML above sentinel — this is an
		// injection attempt that the script should reject.
		remoteContent := "name: injected-workflow\n" +
			sentinel + "\n" +
			staleTemplate + "\n"
		env.setRemoteContent(remoteContent)

		blobCapture := filepath.Join(env.tmpDir, "blob-capture.json")
		enhanceMockGHForBlobCapture(env, blobCapture)

		output, err := env.run()
		_ = err

		// Warning should be emitted about rejected header.
		assert.Contains(t, output, "non-comment content above sentinel was rejected",
			"Script must warn about rejected non-comment header")

		// Read the captured blob.
		blobData, readErr := os.ReadFile(blobCapture)
		require.NoError(t, readErr, "Blob capture file should exist")

		decoded := b64Decode(t, strings.TrimSpace(string(blobData)))

		// Injected YAML must NOT be in output.
		assert.NotContains(t, decoded, "injected-workflow",
			"Injected YAML content must be rejected from output blob")
		assert.NotContains(t, decoded, "name:",
			"No non-comment YAML keys should appear in output blob")

		// Sentinel and fresh content must still be present.
		assert.Contains(t, decoded, sentinel,
			"Sentinel must be present despite injection rejection")
		assert.Contains(t, decoded, freshTemplate,
			"Fresh template content must be present after rejection")
	})
}
