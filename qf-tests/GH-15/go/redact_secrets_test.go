//go:build e2e

package sandbox

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --------------------------------------------------------------------------
// TS-GH-15-009: Verify redactSecrets replaces single credential value
// --------------------------------------------------------------------------

func TestRedactSecrets_SingleSecret(t *testing.T) {
	input := "error: authentication failed with token my-secret-token for provider foo"
	secrets := []string{"my-secret-token"}

	result := redactSecrets(input, secrets)

	assert.NotContains(t, result, "my-secret-token")                           // [test_id:TS-GH-15-009]
	assert.Contains(t, result, "***")                                          // redaction marker applied
	assert.Contains(t, result, "error: authentication failed with token")      // non-secret context preserved
}

// --------------------------------------------------------------------------
// TS-GH-15-010: Verify redaction with multiple secret values
// --------------------------------------------------------------------------

func TestRedactSecrets_MultipleSecrets(t *testing.T) {
	input := "token=secret-tok-1 password=secret-pass-2"
	secrets := []string{"secret-tok-1", "secret-pass-2"}

	result := redactSecrets(input, secrets)

	assert.NotContains(t, result, "secret-tok-1")  // [test_id:TS-GH-15-010] first secret redacted
	assert.NotContains(t, result, "secret-pass-2")  // second secret redacted
	assert.Contains(t, result, "token=")             // non-secret key preserved
	assert.Contains(t, result, "password=")          // non-secret key preserved
}

// --------------------------------------------------------------------------
// TS-GH-15-011: Verify redaction with empty secrets list
// --------------------------------------------------------------------------

func TestRedactSecrets_EmptySecrets(t *testing.T) {
	input := "error: some failure message"
	secrets := []string{}

	result := redactSecrets(input, secrets)

	assert.Equal(t, input, result) // [test_id:TS-GH-15-011] unchanged when no secrets
}
