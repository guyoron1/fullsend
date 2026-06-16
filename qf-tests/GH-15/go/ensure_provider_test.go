//go:build e2e

package sandbox

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeScript writes a shell script to path with executable permissions.
func writeScript(t *testing.T, path string, content string) {
	t.Helper()
	err := os.WriteFile(path, []byte(content), 0755)
	require.NoError(t, err)
}

// alreadyExistsScript: first 'provider create' returns AlreadyExists,
// 'provider delete' succeeds, second 'provider create' succeeds.
const alreadyExistsScript = `#!/bin/bash
STATE_FILE="/tmp/openshell_state_$$"
if [[ "$1" == "provider" && "$2" == "create" ]]; then
    if [ ! -f "$STATE_FILE" ]; then
        touch "$STATE_FILE"; echo "AlreadyExists" >&2; exit 1
    fi; exit 0
elif [[ "$1" == "provider" && "$2" == "delete" ]]; then
    exit 0
fi`

// deleteFailsScript: AlreadyExists on create, fails on delete.
const deleteFailsScript = `#!/bin/bash
if [[ "$1" == "provider" && "$2" == "create" ]]; then
    echo "AlreadyExists" >&2; exit 1
elif [[ "$1" == "provider" && "$2" == "delete" ]]; then
    echo "delete failed: internal error" >&2; exit 1
fi`

// retryCreateFailsScript: AlreadyExists, delete OK, retry create fails.
const retryCreateFailsScript = `#!/bin/bash
STATE_FILE="/tmp/openshell_state_$$"
if [[ "$1" == "provider" && "$2" == "create" ]]; then
    if [ ! -f "$STATE_FILE" ]; then
        touch "$STATE_FILE"; echo "AlreadyExists" >&2; exit 1
    fi; echo "retry create failed" >&2; exit 1
elif [[ "$1" == "provider" && "$2" == "delete" ]]; then
    exit 0
fi`

// captureArgsScript: AlreadyExists on first create, captures args on retry.
const captureArgsScript = `#!/bin/bash
STATE_FILE="/tmp/openshell_state_$$"
ARGS_FILE="/tmp/openshell_args_$$"
if [[ "$1" == "provider" && "$2" == "create" ]]; then
    if [ ! -f "$STATE_FILE" ]; then
        touch "$STATE_FILE"; echo "AlreadyExists" >&2; exit 1
    fi; echo "$@" > "$ARGS_FILE"; exit 0
elif [[ "$1" == "provider" && "$2" == "delete" ]]; then
    exit 0
fi`

// genericErrorScript: fails with a generic (non-AlreadyExists) error.
const genericErrorScript = `#!/bin/bash
if [[ "$1" == "provider" && "$2" == "create" ]]; then
    echo "connection refused" >&2; exit 1
fi`

// specificErrorScript: fails with a specific known error message.
const specificErrorScript = `#!/bin/bash
if [[ "$1" == "provider" && "$2" == "create" ]]; then
    echo "specific error from openshell" >&2; exit 1
fi`

// deleteFailsWithSecretScript: AlreadyExists on create, delete fails echoing
// the credential value from the SUPER_SECRET env var.
const deleteFailsWithSecretScript = `#!/bin/bash
if [[ "$1" == "provider" && "$2" == "create" ]]; then
    echo "AlreadyExists" >&2; exit 1
elif [[ "$1" == "provider" && "$2" == "delete" ]]; then
    echo "delete failed: auth=$SUPER_SECRET" >&2; exit 1
fi`

// retryCreateFailsWithSecretScript: AlreadyExists, delete OK, retry create
// fails echoing credential value from the SUPER_SECRET env var.
const retryCreateFailsWithSecretScript = `#!/bin/bash
STATE_FILE="/tmp/openshell_state_$$"
if [[ "$1" == "provider" && "$2" == "create" ]]; then
    if [ ! -f "$STATE_FILE" ]; then
        touch "$STATE_FILE"; echo "AlreadyExists" >&2; exit 1
    fi; echo "retry create failed: auth=$SUPER_SECRET" >&2; exit 1
elif [[ "$1" == "provider" && "$2" == "delete" ]]; then
    exit 0
fi`

// genericErrorWithSecretScript: generic (non-AlreadyExists) error that echoes
// credential value from the SUPER_SECRET env var.
const genericErrorWithSecretScript = `#!/bin/bash
if [[ "$1" == "provider" && "$2" == "create" ]]; then
    echo "connection refused: auth=$SUPER_SECRET" >&2; exit 1
fi`

// --------------------------------------------------------------------------
// TS-GH-15-001: Verify provider recreated when AlreadyExists returned
// --------------------------------------------------------------------------

func TestEnsureProvider_AlreadyExists_RecreatesProvider(t *testing.T) {
	tmpDir := t.TempDir()
	fakeOpenshell := filepath.Join(tmpDir, "openshell")
	writeScript(t, fakeOpenshell, alreadyExistsScript)
	t.Setenv("PATH", tmpDir+":"+os.Getenv("PATH"))

	err := EnsureProvider("test-provider", "gcp", map[string]string{
		"TOKEN": "test-token-value",
	}, nil)

	require.NoError(t, err) // [test_id:TS-GH-15-001]
}

// --------------------------------------------------------------------------
// TS-GH-15-002: Verify error when delete fails during recreate
// --------------------------------------------------------------------------

func TestEnsureProvider_AlreadyExists_DeleteFails(t *testing.T) {
	tmpDir := t.TempDir()
	writeScript(t, filepath.Join(tmpDir, "openshell"), deleteFailsScript)
	t.Setenv("PATH", tmpDir+":"+os.Getenv("PATH"))

	secretValue := "test-token-value"
	err := EnsureProvider("test-provider", "gcp", map[string]string{
		"TOKEN": secretValue,
	}, nil)

	require.Error(t, err)                                  // [test_id:TS-GH-15-002]
	assert.Contains(t, err.Error(), "delete")              // error indicates delete failure
	assert.NotContains(t, err.Error(), secretValue)        // no credential leakage
}

// --------------------------------------------------------------------------
// TS-GH-15-003: Verify error when retry create fails after delete
// --------------------------------------------------------------------------

func TestEnsureProvider_AlreadyExists_RetryCreateFails(t *testing.T) {
	tmpDir := t.TempDir()
	writeScript(t, filepath.Join(tmpDir, "openshell"), retryCreateFailsScript)
	t.Setenv("PATH", tmpDir+":"+os.Getenv("PATH"))

	secretValue := "test-token-value"
	err := EnsureProvider("test-provider", "gcp", map[string]string{
		"TOKEN": secretValue,
	}, nil)

	require.Error(t, err)                           // [test_id:TS-GH-15-003]
	assert.NotContains(t, err.Error(), secretValue)  // credentials redacted
}

// --------------------------------------------------------------------------
// TS-GH-15-004: Verify recreated provider uses current credentials
// --------------------------------------------------------------------------

func TestEnsureProvider_AlreadyExists_UsesFreshCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	writeScript(t, filepath.Join(tmpDir, "openshell"), captureArgsScript)
	t.Setenv("PATH", tmpDir+":"+os.Getenv("PATH"))

	freshToken := "fresh-token-abc123"
	err := EnsureProvider("test-provider", "gcp", map[string]string{
		"TOKEN": freshToken,
	}, nil)

	require.NoError(t, err) // [test_id:TS-GH-15-004]

	// Read the args file written by the capture script to verify the retry
	// create received the correct credential key (bare-key form means the
	// value is passed via environment, but the --credential KEY flag is present).
	argsFiles, _ := filepath.Glob("/tmp/openshell_args_*")
	require.NotEmpty(t, argsFiles, "capture script should have written args file")
	argsContent, readErr := os.ReadFile(argsFiles[0])
	require.NoError(t, readErr)
	assert.Contains(t, string(argsContent), "--credential")
	assert.Contains(t, string(argsContent), "TOKEN")
}

// --------------------------------------------------------------------------
// TS-GH-15-005: Verify non-AlreadyExists error returned without delete
// --------------------------------------------------------------------------

func TestEnsureProvider_NonAlreadyExistsError_NoDelete(t *testing.T) {
	tmpDir := t.TempDir()
	// Script that logs to a file when delete is called, to verify it was NOT called.
	deleteTrackingScript := `#!/bin/bash
DELETE_LOG="` + tmpDir + `/delete_called"
if [[ "$1" == "provider" && "$2" == "create" ]]; then
    echo "connection refused" >&2; exit 1
elif [[ "$1" == "provider" && "$2" == "delete" ]]; then
    touch "$DELETE_LOG"; exit 0
fi`
	writeScript(t, filepath.Join(tmpDir, "openshell"), deleteTrackingScript)
	t.Setenv("PATH", tmpDir+":"+os.Getenv("PATH"))

	err := EnsureProvider("test-provider", "gcp", map[string]string{
		"TOKEN": "test-token",
	}, nil)

	require.Error(t, err) // [test_id:TS-GH-15-005]

	// Verify delete was NOT called
	deleteLogFile := filepath.Join(tmpDir, "delete_called")
	_, statErr := os.Stat(deleteLogFile)
	assert.True(t, os.IsNotExist(statErr), "provider delete should not have been called")
}

// --------------------------------------------------------------------------
// TS-GH-15-006: Verify original error message preserved in output
// --------------------------------------------------------------------------

func TestEnsureProvider_NonAlreadyExistsError_PreservesMessage(t *testing.T) {
	tmpDir := t.TempDir()
	writeScript(t, filepath.Join(tmpDir, "openshell"), specificErrorScript)
	t.Setenv("PATH", tmpDir+":"+os.Getenv("PATH"))

	err := EnsureProvider("test-provider", "gcp", map[string]string{
		"TOKEN": "test-token",
	}, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "specific error from openshell") // [test_id:TS-GH-15-006]
	assert.Contains(t, err.Error(), "test-provider")                 // error context includes provider name
}

// --------------------------------------------------------------------------
// TS-GH-15-007: Verify delete error does not leak credentials
// --------------------------------------------------------------------------

func TestEnsureProvider_DeleteFails_RedactsCredentials(t *testing.T) {
	secretToken := "super-secret-token-12345"
	tmpDir := t.TempDir()
	writeScript(t, filepath.Join(tmpDir, "openshell"), deleteFailsWithSecretScript)
	t.Setenv("PATH", tmpDir+":"+os.Getenv("PATH"))
	t.Setenv("SUPER_SECRET", secretToken)

	err := EnsureProvider("test-provider", "gcp", map[string]string{
		"SUPER_SECRET": "${SUPER_SECRET}",
	}, nil)

	require.Error(t, err)
	assert.NotContains(t, err.Error(), secretToken) // [test_id:TS-GH-15-007]
	assert.Contains(t, err.Error(), "***")          // redaction marker present
}

// --------------------------------------------------------------------------
// TS-GH-15-008: Verify retry create error does not leak credentials
// --------------------------------------------------------------------------

func TestEnsureProvider_RetryCreateFails_RedactsCredentials(t *testing.T) {
	secretToken := "super-secret-token-12345"
	tmpDir := t.TempDir()
	writeScript(t, filepath.Join(tmpDir, "openshell"), retryCreateFailsWithSecretScript)
	t.Setenv("PATH", tmpDir+":"+os.Getenv("PATH"))
	t.Setenv("SUPER_SECRET", secretToken)

	err := EnsureProvider("test-provider", "gcp", map[string]string{
		"SUPER_SECRET": "${SUPER_SECRET}",
	}, nil)

	require.Error(t, err)
	assert.NotContains(t, err.Error(), secretToken) // [test_id:TS-GH-15-008]
	assert.Contains(t, err.Error(), "***")          // redaction marker present
}

// --------------------------------------------------------------------------
// TS-GH-15-012: Verify credentials redacted in non-AlreadyExists error
// --------------------------------------------------------------------------

func TestEnsureProvider_NonAlreadyExists_RedactsCredentials(t *testing.T) {
	secretToken := "super-secret-token-12345"
	tmpDir := t.TempDir()
	writeScript(t, filepath.Join(tmpDir, "openshell"), genericErrorWithSecretScript)
	t.Setenv("PATH", tmpDir+":"+os.Getenv("PATH"))
	t.Setenv("SUPER_SECRET", secretToken)

	err := EnsureProvider("test-provider", "gcp", map[string]string{
		"SUPER_SECRET": "${SUPER_SECRET}",
	}, nil)

	require.Error(t, err)
	assert.NotContains(t, err.Error(), secretToken) // [test_id:TS-GH-15-012]
}

// --------------------------------------------------------------------------
// Helpers: count calls for verifying invocation sequences
// --------------------------------------------------------------------------

// verifyNoDeleteCall is a helper used by tests that need to confirm the
// delete path was not triggered. The fake openshell scripts in those tests
// write a marker file on delete; this function checks its absence.
func verifyNoDeleteCall(t *testing.T, tmpDir string) {
	t.Helper()
	deleteLogFile := filepath.Join(tmpDir, "delete_called")
	_, err := os.Stat(deleteLogFile)
	assert.True(t, os.IsNotExist(err), "provider delete should not have been called")
}

// containsAny returns true if s contains any of the given substrings.
func containsAny(s string, substrs []string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
