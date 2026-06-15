package sandbox

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEnsureProvider_Success_NoConflict validates that EnsureProvider succeeds
// on first call when no provider with the same name exists.
// STD Scenario: TS-GH-10-001
func TestEnsureProvider_Success_NoConflict(t *testing.T) {
	binDir := t.TempDir()

	// Fake openshell that succeeds on provider create and logs calls.
	callLog := filepath.Join(binDir, "calls.log")
	fakeScript := `#!/bin/sh
echo "$@" >> "` + callLog + `"
if [ "$1" = "provider" ] && [ "$2" = "create" ]; then
  exit 0
fi
exit 0
`
	fakeBin := filepath.Join(binDir, "openshell")
	require.NoError(t, os.WriteFile(fakeBin, []byte(fakeScript), 0o755))

	t.Setenv("PATH", binDir)

	err := EnsureProvider("test-provider", "github-app", nil, nil)
	assert.NoError(t, err, "EnsureProvider should succeed when no provider conflict exists")

	// Verify openshell was called exactly once with provider create arguments.
	logData, readErr := os.ReadFile(callLog)
	require.NoError(t, readErr, "call log should exist")
	lines := strings.Split(strings.TrimSpace(string(logData)), "\n")
	assert.Len(t, lines, 1, "openshell should be called exactly once")
	assert.Contains(t, lines[0], "provider create", "call should be provider create")
}

// TestEnsureProvider_AlreadyExists_DeleteAndRecreate validates the idempotent
// recovery path: create fails with AlreadyExists, delete succeeds, recreate succeeds.
// STD Scenario: TS-GH-10-002
func TestEnsureProvider_AlreadyExists_DeleteAndRecreate(t *testing.T) {
	binDir := t.TempDir()
	stateFile := filepath.Join(binDir, "state")
	callLog := filepath.Join(binDir, "calls.log")

	// First "provider create" returns AlreadyExists, delete succeeds, second create succeeds.
	fakeScript := `#!/bin/sh
echo "$1 $2" >> "` + callLog + `"
if [ "$1" = "provider" ] && [ "$2" = "create" ]; then
  if [ ! -f "` + stateFile + `" ]; then
    echo "created" > "` + stateFile + `"
    echo "Error: × status: AlreadyExists, message: \"provider already exists\"" >&2
    exit 1
  fi
  exit 0
fi
if [ "$1" = "provider" ] && [ "$2" = "delete" ]; then
  exit 0
fi
exit 1
`
	fakeBin := filepath.Join(binDir, "openshell")
	require.NoError(t, os.WriteFile(fakeBin, []byte(fakeScript), 0o755))

	t.Setenv("PATH", binDir)

	err := EnsureProvider("github", "github-app", nil, nil)
	assert.NoError(t, err, "EnsureProvider should succeed after delete-and-recreate")

	// Verify openshell was called exactly 3 times: create, delete, create.
	logData, readErr := os.ReadFile(callLog)
	require.NoError(t, readErr, "call log should exist")
	lines := strings.Split(strings.TrimSpace(string(logData)), "\n")
	require.Len(t, lines, 3, "openshell should be called exactly 3 times")
	assert.Equal(t, "provider create", lines[0], "first call should be provider create")
	assert.Equal(t, "provider delete", lines[1], "second call should be provider delete")
	assert.Equal(t, "provider create", lines[2], "third call should be provider create")
}

// TestEnsureProvider_AlreadyExists_DeleteFails validates that EnsureProvider
// returns a clear error when the delete step fails during recreate.
// STD Scenario: TS-GH-10-003
func TestEnsureProvider_AlreadyExists_DeleteFails(t *testing.T) {
	binDir := t.TempDir()

	fakeScript := `#!/bin/sh
if [ "$1" = "provider" ] && [ "$2" = "create" ]; then
  echo "Error: × status: AlreadyExists, message: \"provider already exists\"" >&2
  exit 1
fi
if [ "$1" = "provider" ] && [ "$2" = "delete" ]; then
  echo "delete failed: permission denied" >&2
  exit 1
fi
exit 1
`
	fakeBin := filepath.Join(binDir, "openshell")
	require.NoError(t, os.WriteFile(fakeBin, []byte(fakeScript), 0o755))

	t.Setenv("PATH", binDir)

	err := EnsureProvider("github", "github-app", nil, nil)
	assert.Error(t, err, "EnsureProvider should return error when delete fails")
	assert.Contains(t, err.Error(), "provider delete", "error should mention provider delete")
	assert.Contains(t, err.Error(), "during recreate", "error should mention during recreate")
}

// TestEnsureProvider_CreateFails_NotAlreadyExists validates that EnsureProvider
// returns the original error without attempting delete-and-recreate when the
// create failure is not an AlreadyExists error.
// STD Scenario: TS-GH-10-004
func TestEnsureProvider_CreateFails_NotAlreadyExists(t *testing.T) {
	binDir := t.TempDir()
	callLog := filepath.Join(binDir, "calls.log")

	fakeScript := `#!/bin/sh
echo "$1 $2" >> "` + callLog + `"
if [ "$1" = "provider" ] && [ "$2" = "create" ]; then
  echo "connection refused: cannot reach gateway" >&2
  exit 1
fi
exit 0
`
	fakeBin := filepath.Join(binDir, "openshell")
	require.NoError(t, os.WriteFile(fakeBin, []byte(fakeScript), 0o755))

	t.Setenv("PATH", binDir)

	err := EnsureProvider("test", "custom", nil, nil)
	assert.Error(t, err, "EnsureProvider should return error for non-AlreadyExists failure")
	assert.Contains(t, err.Error(), "provider create", "error should mention provider create")
	assert.Contains(t, err.Error(), "connection refused", "error should contain original error text")

	// Verify openshell was called exactly once (no delete attempt).
	logData, readErr := os.ReadFile(callLog)
	require.NoError(t, readErr, "call log should exist")
	lines := strings.Split(strings.TrimSpace(string(logData)), "\n")
	assert.Len(t, lines, 1, "openshell should be called exactly once (no delete attempt)")
}

// TestRedactSecrets_ReplacesAllSecrets validates that redactSecrets replaces
// all known secret values with "***" and preserves non-secret text.
// STD Scenario: TS-GH-10-005
func TestRedactSecrets_ReplacesAllSecrets(t *testing.T) {
	input := "error: auth failed with token=s3cr3t-t0k3n for user admin"
	secrets := []string{"s3cr3t-t0k3n"}

	result := redactSecrets(input, secrets)

	assert.Contains(t, result, "***", "output should contain redaction marker")
	assert.NotContains(t, result, "s3cr3t-t0k3n", "output must not contain secret value")
	assert.Contains(t, result, "error: auth failed with token=", "non-secret text should be preserved")
	assert.Contains(t, result, " for user admin", "non-secret text should be preserved")
}

// TestRedactSecrets_EmptySecretsList validates that redactSecrets returns
// the input unchanged when the secrets list is empty.
// STD Scenario: TS-GH-10-006
func TestRedactSecrets_EmptySecretsList(t *testing.T) {
	input := "error: something went wrong"

	result := redactSecrets(input, []string{})

	assert.Equal(t, input, result, "output should equal input when secrets list is empty")
}

// TestRedactSecrets_MultipleSecrets validates that redactSecrets replaces
// all different secret values when multiple secrets are present.
// STD Scenario: TS-GH-10-007
func TestRedactSecrets_MultipleSecrets(t *testing.T) {
	input := "token=abc123 password=xyz789 key=secret42"
	secrets := []string{"abc123", "xyz789", "secret42"}

	result := redactSecrets(input, secrets)

	assert.NotContains(t, result, "abc123", "first secret should be redacted")
	assert.NotContains(t, result, "xyz789", "second secret should be redacted")
	assert.NotContains(t, result, "secret42", "third secret should be redacted")
	assert.Equal(t, "token=*** password=*** key=***", result, "all secrets replaced with ***")
}

// TestEnsureProvider_RecreateCreateFails_RedactedError validates that when
// the second create (after successful delete) fails, the error message
// does not contain any secret values.
// STD Scenario: TS-GH-10-008
func TestEnsureProvider_RecreateCreateFails_RedactedError(t *testing.T) {
	binDir := t.TempDir()
	stateFile := filepath.Join(binDir, "state")

	secretValue := "s3cr3t-cr3d3nt1al"

	// First create → AlreadyExists, delete → success, second create → fail with secret in output.
	fakeScript := `#!/bin/sh
if [ "$1" = "provider" ] && [ "$2" = "create" ]; then
  if [ ! -f "` + stateFile + `" ]; then
    echo "created" > "` + stateFile + `"
    echo "Error: × status: AlreadyExists, message: \"provider already exists\"" >&2
    exit 1
  else
    echo "auth failed with token=` + secretValue + `" >&2
    exit 1
  fi
fi
if [ "$1" = "provider" ] && [ "$2" = "delete" ]; then
  exit 0
fi
exit 0
`
	fakeBin := filepath.Join(binDir, "openshell")
	require.NoError(t, os.WriteFile(fakeBin, []byte(fakeScript), 0o755))

	t.Setenv("PATH", binDir)

	credentials := map[string]string{
		"TOKEN": secretValue,
	}

	err := EnsureProvider("test-provider", "custom", credentials, nil)
	assert.Error(t, err, "EnsureProvider should return error when recreate fails")
	assert.NotContains(t, err.Error(), secretValue,
		"error message must not contain secret value — credentials would leak")
}
