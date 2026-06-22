package scaffold

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Base64 Round-Trip — Empty Content Edge Case

STP Reference: outputs/stp/GH-77/GH-77_test_plan.md
STD Reference: outputs/std/GH-77/GH-77_test_description.yaml
Jira: GH-77

Tests the edge case where empty content is encoded to base64 and decoded,
ensuring the decode-compare path handles empty input without panicking or
producing spurious output.

Existing coverage references (GH-2247):
  - Scenario 3 (TS-GH77-003): Covered by TestBase64RoundTrip/line-wrapped_base64_input_is_decoded_correctly
    in qf-tests/GH-2247/go/base64_roundtrip_test.go
  - Scenario 4 (TS-GH77-004): Covered by TestBase64RoundTrip/base64_round-trip_preserves_multi-line_YAML
    in qf-tests/GH-2247/go/base64_roundtrip_test.go
*/

func TestQF_Base64RoundTrip_EmptyContent(t *testing.T) {
	t.Run("[test_id:TS-GH77-005] should produce empty decoded text without errors", func(t *testing.T) {
		// Step TEST-01: Encode empty string to base64.
		encodeCmd := exec.Command("bash", "-c", `printf '' | base64 -w0`)
		encodedBytes, err := encodeCmd.Output()
		require.NoError(t, err, "base64 encoding of empty string should succeed")

		encoded := string(encodedBytes)
		// base64 of empty input is an empty string (no padding needed).
		// The command should succeed without error regardless.

		// Step TEST-02: Decode the base64 output.
		decodeCmd := exec.Command("bash", "-c", `printf '%s' "$ENCODED" | base64 -d`)
		decodeCmd.Env = append(decodeCmd.Environ(), "ENCODED="+encoded)
		decodedBytes, err := decodeCmd.Output()
		require.NoError(t, err, "base64 decoding of encoded empty string should succeed")

		// ASSERT-01: Empty input round-trips to empty output.
		assert.Empty(t, string(decodedBytes),
			"Decoded output of empty-input round-trip must be empty string")

		// Step TEST-03: Pipe empty string through full encode-decode-normalize path.
		// This matches the pipeline used in reconcile-repos.sh:
		//   printf '' | base64 -w0 | base64 -d | tr -d '\r'
		fullPipeCmd := exec.Command("bash", "-c", `printf '' | base64 -w0 | base64 -d | tr -d '\r'`)
		fullPipeOut, err := fullPipeCmd.Output()

		// ASSERT-02: No error during encode/decode of empty content.
		require.NoError(t, err,
			"Full encode-decode-normalize pipeline should succeed for empty input (exit code 0)")

		assert.Empty(t, string(fullPipeOut),
			"Full pipeline output for empty input must be empty string")
	})
}
