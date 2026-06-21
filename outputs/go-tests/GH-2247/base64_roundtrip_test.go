package scaffold

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Base64 Encoding Round-Trip Integrity Tests

STP Reference: outputs/stp/GH-2247/GH-2247_test_plan.md
Jira: GH-2247

Validates that base64 encode/decode round-trips preserve content byte-for-byte.
This tests the data transformation pipeline preceding the comparison logic.
*/

func TestBase64RoundTrip(t *testing.T) {
	t.Run("[test_id:TS-GH2247-016] base64 round-trip preserves multi-line YAML", func(t *testing.T) {
		// Multi-line YAML with indentation, colons, and dashes — representative
		// of a real shim workflow file.
		multilineYAML := `name: test-workflow
on:
  workflow_call:
    inputs:
      target:
        type: string
jobs:
  test:
    runs-on: ubuntu-latest
`
		// Encode with base64 -w0 (no wrapping) and decode back — should be
		// byte-identical to the original.
		encodeCmd := exec.Command("bash", "-c",
			`printf '%s' "$INPUT" | base64 -w0 | base64 -d`)
		encodeCmd.Env = append(encodeCmd.Environ(), "INPUT="+multilineYAML)
		decoded, err := encodeCmd.Output()
		require.NoError(t, err, "base64 encode/decode should succeed")

		assert.Equal(t, multilineYAML, string(decoded),
			"Decoded content must be byte-identical to original multi-line YAML")

		// Verify YAML structure is preserved.
		assert.Contains(t, string(decoded), "  workflow_call:",
			"Indentation must be preserved")
		assert.Contains(t, string(decoded), "    inputs:",
			"Nested indentation must be preserved")
		assert.Contains(t, string(decoded), "runs-on: ubuntu-latest",
			"Colons and values must be preserved")
	})

	t.Run("[test_id:TS-GH2247-017] line-wrapped base64 input is decoded correctly", func(t *testing.T) {
		// Generate a long enough string that standard base64 encoding (76-char
		// line wrapping) produces multiple lines.
		longContent := strings.Repeat("# This is a long line of content for testing base64 wrapping behavior\n", 10)

		// Encode with default wrapping (76 chars per line).
		wrapCmd := exec.Command("bash", "-c",
			`printf '%s' "$INPUT" | base64`)
		wrapCmd.Env = append(wrapCmd.Environ(), "INPUT="+longContent)
		wrappedB64, err := wrapCmd.Output()
		require.NoError(t, err, "wrapped base64 encode should succeed")

		// Verify it actually has line breaks (precondition).
		assert.Contains(t, string(wrappedB64), "\n",
			"Precondition: wrapped base64 should contain newlines")

		// Encode without wrapping.
		nowrapCmd := exec.Command("bash", "-c",
			`printf '%s' "$INPUT" | base64 -w0`)
		nowrapCmd.Env = append(nowrapCmd.Environ(), "INPUT="+longContent)
		unwrappedB64, err := nowrapCmd.Output()
		require.NoError(t, err, "unwrapped base64 encode should succeed")

		// Verify no line breaks in unwrapped output (precondition).
		assert.NotContains(t, string(unwrappedB64), "\n",
			"Precondition: unwrapped base64 should not contain newlines")

		// Decode both and verify they produce identical output.
		decodeWrapped := exec.Command("bash", "-c",
			`printf '%s' "$B64" | base64 -d`)
		decodeWrapped.Env = append(decodeWrapped.Environ(), "B64="+string(wrappedB64))
		decodedWrapped, err := decodeWrapped.Output()
		require.NoError(t, err, "decoding wrapped base64 should succeed")

		decodeUnwrapped := exec.Command("bash", "-c",
			`printf '%s' "$B64" | base64 -d`)
		decodeUnwrapped.Env = append(decodeUnwrapped.Environ(), "B64="+string(unwrappedB64))
		decodedUnwrapped, err := decodeUnwrapped.Output()
		require.NoError(t, err, "decoding unwrapped base64 should succeed")

		assert.Equal(t, string(decodedWrapped), string(decodedUnwrapped),
			"Wrapped and unwrapped base64 must decode to identical content")
		assert.Equal(t, longContent, string(decodedWrapped),
			"Decoded wrapped base64 must equal original content")
		assert.Equal(t, longContent, string(decodedUnwrapped),
			"Decoded unwrapped base64 must equal original content")
	})
}
