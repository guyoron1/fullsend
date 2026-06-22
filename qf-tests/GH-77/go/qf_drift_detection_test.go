package scaffold

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Drift Detection — Mixed Line Ending Normalization

STP Reference: outputs/stp/GH-77/GH-77_test_plan.md
STD Reference: outputs/std/GH-77/GH-77_test_description.yaml
Jira: GH-77

Validates that content with mixed line endings (some lines CRLF, some LF)
is correctly normalized before comparison, preventing false drift detection.

Existing coverage references (GH-2247):
  - Scenario 1 (TS-GH77-001): Covered by TestDriftDetection_EncodingNormalization/identical_content_with_extra_trailing_newline_not_flagged_stale
    in qf-tests/GH-2247/go/drift_detection_test.go
  - Scenario 2 (TS-GH77-002): Covered by TestDriftDetection_EncodingNormalization/genuinely_different_content_is_flagged_stale
    in qf-tests/GH-2247/go/drift_detection_test.go
  - Scenario 15 (TS-GH77-015): Covered by TestDriftDetection_EncodingNormalization/carriage_return_differences_ignored_in_comparison
    in qf-tests/GH-2247/go/drift_detection_test.go
*/

func TestQF_DriftDetection_MixedLineEndings(t *testing.T) {
	t.Run("[test_id:TS-GH77-016] should handle mixed CRLF/LF line endings correctly", func(t *testing.T) {
		// Setup: Create test environment with mocked gh CLI.
		env := newReconcileEnv(t)

		// Build remote content with mixed line endings:
		// - Sentinel line ends with CRLF
		// - Managed content line ends with LF only
		// The text is otherwise identical to the template.
		mixedEndingsContent := sentinel + "\r\n" + freshTemplate + "\n"

		env.setRemoteContent(mixedEndingsContent)

		// Execute: Run reconcile script.
		output, err := env.run()
		require.NoError(t, err, "reconcile-repos.sh should exit 0; output:\n%s", output)

		// ASSERT-01: Mixed line endings do not cause false drift.
		assert.Contains(t, output, "already enrolled (shim up to date)",
			"Script should recognize mixed-ending content as up-to-date after tr -d '\\r' normalization")

		// ASSERT-02: No blob API call for mixed-ending identical content.
		assert.False(t, env.blobCreated(),
			"No blob should be created when content differs only in line ending style")
	})

	t.Run("[test_id:TS-GH77-016-negative] mixed endings with genuinely different text is still detected as stale", func(t *testing.T) {
		// Verify the normalization does not mask genuine content differences.
		env := newReconcileEnv(t)

		// Mixed line endings AND genuinely different managed content.
		mixedEndingsStale := sentinel + "\r\n" + staleTemplate + "\n"

		env.setRemoteContent(mixedEndingsStale)

		output, err := env.run()
		_ = err

		assert.Contains(t, output, "shim is stale",
			"Genuinely different content with mixed line endings should still be detected as stale")
		assert.True(t, env.blobCreated(),
			"A blob should be created for genuinely stale content regardless of line ending style")
	})
}
