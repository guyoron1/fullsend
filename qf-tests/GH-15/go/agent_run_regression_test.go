//go:build e2e

package sandbox

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// allProvidersAlreadyExistScript: AlreadyExists for all creates, succeeds
// on deletes and retry creates. Uses per-provider state files.
const allProvidersAlreadyExistScript = `#!/bin/bash
# Extract provider name from --name flag
NAME=""
for i in "$@"; do
    if [ "$PREV" = "--name" ]; then NAME="$i"; fi
    PREV="$i"
done
STATE_FILE="/tmp/openshell_state_${NAME}_$$"

if [[ "$1" == "provider" && "$2" == "create" ]]; then
    if [ ! -f "$STATE_FILE" ]; then
        touch "$STATE_FILE"; echo "AlreadyExists" >&2; exit 1
    fi; exit 0
elif [[ "$1" == "provider" && "$2" == "delete" ]]; then
    exit 0
fi`

// nonIdempotentErrorScript: fails with a non-AlreadyExists error for
// regression tests.
const nonIdempotentErrorScript = `#!/bin/bash
if [[ "$1" == "provider" && "$2" == "create" ]]; then
    # Extract provider name from --name flag
    NAME=""
    for i in "$@"; do
        if [ "$PREV" = "--name" ]; then NAME="$i"; fi
        PREV="$i"
    done
    echo "gateway unreachable for provider $NAME" >&2; exit 1
fi`

// --------------------------------------------------------------------------
// TS-GH-15-013: Verify agent run succeeds with pre-existing providers
//
// This regression test validates that when multiple providers already exist,
// the idempotent delete-and-recreate path succeeds for each one. This
// exercises the same code path that runAgent calls via EnsureProvider.
// --------------------------------------------------------------------------

func TestRunAgent_PreExistingProviders_Succeeds(t *testing.T) {
	tmpDir := t.TempDir()
	writeScript(t, filepath.Join(tmpDir, "openshell"), allProvidersAlreadyExistScript)
	t.Setenv("PATH", tmpDir+":"+os.Getenv("PATH"))

	// Simulate multiple provider definitions as seen in a real harness config.
	providers := []struct {
		name         string
		providerType string
		credentials  map[string]string
	}{
		{name: "anthropic", providerType: "anthropic", credentials: map[string]string{"ANTHROPIC_API_KEY": "key-1"}},
		{name: "github", providerType: "github", credentials: map[string]string{"GITHUB_TOKEN": "ghp-test"}},
		{name: "custom-llm", providerType: "openai", credentials: map[string]string{"OPENAI_API_KEY": "sk-test"}},
	}

	for _, p := range providers {
		err := EnsureProvider(p.name, p.providerType, p.credentials, nil)
		require.NoError(t, err, "EnsureProvider should succeed for pre-existing provider %q", p.name) // [test_id:TS-GH-15-013]
	}
}

// --------------------------------------------------------------------------
// TS-GH-15-014: Verify agent run fails fast on non-idempotent error
//
// This regression test validates that when a provider creation fails with a
// non-AlreadyExists error, the workflow reports the error immediately with
// actionable context (provider name).
// --------------------------------------------------------------------------

func TestRunAgent_NonIdempotentError_FailsFast(t *testing.T) {
	tmpDir := t.TempDir()
	writeScript(t, filepath.Join(tmpDir, "openshell"), nonIdempotentErrorScript)
	t.Setenv("PATH", tmpDir+":"+os.Getenv("PATH"))

	providerName := "failing-provider"
	err := EnsureProvider(providerName, "gcp", map[string]string{
		"TOKEN": "test-token",
	}, nil)

	require.Error(t, err)                              // [test_id:TS-GH-15-014]
	assert.Contains(t, err.Error(), providerName)       // error is actionable (contains provider name)
}
