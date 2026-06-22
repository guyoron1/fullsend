package scaffold

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Pre-Sentinel Fallback — Sentinel Existence Guard

STP Reference: outputs/stp/GH-77/GH-77_test_plan.md
STD Reference: outputs/std/GH-77/GH-77_test_description.yaml
Jira: GH-77

Validates that the fallback path (full content comparison) is NOT triggered
when the sentinel line exists in the remote shim. When sentinel exists,
only the managed section (after sentinel) should be compared.

Existing coverage references (GH-2247):
  - Scenario 6 (TS-GH77-006): Covered by TestPreSentinelFallback/empty_extract_managed_content_triggers_fallback
    in qf-tests/GH-2247/go/pre_sentinel_fallback_test.go
  - Scenario 7 (TS-GH77-007): Covered by TestPreSentinelFallback/empty_extract_managed_content_triggers_fallback
    in qf-tests/GH-2247/go/pre_sentinel_fallback_test.go
  - Scenario 8 (TS-GH77-008): Covered by TestPreSentinelFallback/pre-sentinel_shim_matches_full_decoded_content
    in qf-tests/GH-2247/go/pre_sentinel_fallback_test.go
  - Scenario 9 (TS-GH77-009): Covered by TestPreSentinelFallback/pre-sentinel_shim_detects_genuine_drift
    in qf-tests/GH-2247/go/pre_sentinel_fallback_test.go
*/

func TestQF_PreSentinelFallback_SentinelGuard(t *testing.T) {
	t.Run("[test_id:TS-GH77-010] should not trigger fallback when sentinel exists", func(t *testing.T) {
		env := newReconcileEnv(t)

		// Step TEST-01: Verify extract_managed_content returns non-empty for
		// sentinel-containing input.
		codeWithSentinel := `
input="user custom header line
# Copyright 2026 Conforma
` + sentinel + `
` + freshTemplate + `"
result=$(printf '%s\n' "$input" | extract_managed_content)
if [ -n "$result" ]; then
  echo "HAS_MANAGED_CONTENT"
  echo "$result"
else
  echo "EMPTY_MANAGED_CONTENT"
fi
`
		out, err := env.runBashFunc(codeWithSentinel)
		require.NoError(t, err, "extract_managed_content should execute; output:\n%s", out)

		// ASSERT-01: extract_managed_content returns non-empty for sentinel input.
		assert.Contains(t, out, "HAS_MANAGED_CONTENT",
			"extract_managed_content must return non-empty when sentinel is present in input")
		assert.Contains(t, out, sentinel,
			"Returned content should include the sentinel line itself")

		// Step TEST-02/03: Set remote content with sentinel + matching managed
		// section but a different user header above the sentinel. If the fallback
		// were incorrectly triggered, the full-content comparison would see the
		// different header and flag it as stale. The correct behavior compares
		// only the managed section (after sentinel), which matches.
		differentHeaderSameManaged := "# Different copyright header\n" +
			"# SPDX-License-Identifier: MIT\n" +
			sentinel + "\n" + freshTemplate + "\n"

		env.setRemoteContent(differentHeaderSameManaged)

		output, err := env.run()
		require.NoError(t, err, "reconcile-repos.sh should exit 0; output:\n%s", output)

		// ASSERT-02: Different header with same managed content is NOT flagged stale.
		// This confirms the comparison uses only the managed section, not the full file.
		assert.Contains(t, output, "already enrolled (shim up to date)",
			"User header changes above sentinel should not trigger drift when managed content matches")

		// Verify no unnecessary API calls were made.
		assert.False(t, env.blobCreated(),
			"No blob should be created when only the user header differs")

		// Verify no git/blobs endpoint was hit.
		for _, call := range env.ghCalls() {
			assert.False(t, strings.Contains(call, "git/blobs"),
				"No git/blobs API call should be made for header-only differences; call: %s", call)
		}
	})
}
