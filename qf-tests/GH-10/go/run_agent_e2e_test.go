//go:build e2e

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/ui"
)

// setupFakeOpenshell creates a fake openshell script in a temp directory and
// prepends it to PATH. The script tracks calls in a log file and simulates
// provider and sandbox behaviors needed for runAgent integration tests.
func setupFakeOpenshell(t *testing.T, script string) (binDir string, callLogPath string) {
	t.Helper()
	binDir = t.TempDir()
	callLogPath = filepath.Join(binDir, "calls.log")

	fakeBin := filepath.Join(binDir, "openshell")
	require.NoError(t, os.WriteFile(fakeBin, []byte(script), 0o755))

	// Prepend the fake binary directory so it shadows the real openshell.
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))
	return binDir, callLogPath
}

// setupMinimalHarness creates a minimal .fullsend directory structure with a
// harness YAML for the given agent name. Returns the fullsend dir path.
func setupMinimalHarness(t *testing.T, agentName string) string {
	t.Helper()
	fullsendDir := t.TempDir()

	harnessDir := filepath.Join(fullsendDir, "harness")
	require.NoError(t, os.MkdirAll(harnessDir, 0o755))

	harnessYAML := `agent: "` + agentName + `"
image: "ubuntu:latest"
providers:
  - name: "test-provider"
    type: "github-app"
    credentials: {}
    config: {}
timeout_minutes: 5
`
	harnessFile := filepath.Join(harnessDir, agentName+".yaml")
	require.NoError(t, os.WriteFile(harnessFile, []byte(harnessYAML), 0o644))

	return fullsendDir
}

// TestRunAgent_ProviderAlreadyExists_Succeeds validates that runAgent handles
// pre-existing providers transparently via EnsureProvider's idempotent recovery.
// This is the user-visible fix for issue #2294.
// STD Scenario: TS-GH-10-009
func TestRunAgent_ProviderAlreadyExists_Succeeds(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "provider_state")

	// Fake openshell: first provider create → AlreadyExists, delete → ok,
	// second create → ok. Gateway and sandbox operations succeed.
	script := `#!/bin/sh
if [ "$1" = "provider" ] && [ "$2" = "create" ]; then
  if [ ! -f "` + stateFile + `" ]; then
    echo "done" > "` + stateFile + `"
    echo "Error: × status: AlreadyExists" >&2
    exit 1
  fi
  exit 0
fi
if [ "$1" = "provider" ] && [ "$2" = "delete" ]; then
  exit 0
fi
if [ "$1" = "gateway" ]; then
  exit 0
fi
if [ "$1" = "sandbox" ]; then
  if [ "$2" = "get" ]; then
    echo "Ready"
  fi
  exit 0
fi
exit 0
`

	setupFakeOpenshell(t, script)
	fullsendDir := setupMinimalHarness(t, "test-agent")
	targetRepo := t.TempDir()

	printer := ui.New(os.Stdout)
	err := runAgent("test-agent", fullsendDir, "", targetRepo, "", nil, true, printer)
	assert.NoError(t, err, "runAgent should succeed when provider already exists (idempotent recovery)")
}

// TestRunAgent_ProviderCreateFails_AbortsWithClearError validates that runAgent
// aborts with a clear, redacted error when provider creation fails with a
// non-recoverable error (not AlreadyExists).
// STD Scenario: TS-GH-10-010
func TestRunAgent_ProviderCreateFails_AbortsWithClearError(t *testing.T) {
	secretValue := "gh-app-private-key-abc123"

	// Fake openshell: provider create fails with a non-AlreadyExists error
	// whose output contains a secret value. Gateway works fine.
	script := `#!/bin/sh
if [ "$1" = "provider" ] && [ "$2" = "create" ]; then
  echo "connection refused: cannot reach gateway, key=` + secretValue + `" >&2
  exit 1
fi
if [ "$1" = "gateway" ]; then
  exit 0
fi
if [ "$1" = "sandbox" ] && [ "$2" = "get" ]; then
  echo "Ready"
  exit 0
fi
exit 0
`

	setupFakeOpenshell(t, script)
	fullsendDir := setupMinimalHarness(t, "test-agent")
	targetRepo := t.TempDir()

	printer := ui.New(os.Stdout)
	err := runAgent("test-agent", fullsendDir, "", targetRepo, "", nil, true, printer)
	assert.Error(t, err, "runAgent should return error when provider creation fails")
	assert.NotContains(t, err.Error(), secretValue,
		"error message must not contain secret values")
}

// TestRunAgent_SequentialCalls_ReuseProviders validates that two sequential
// runAgent calls both succeed. The second call encounters the provider from
// the first run, triggers delete-and-recreate, and proceeds.
// STD Scenario: TS-GH-10-011
func TestRunAgent_SequentialCalls_ReuseProviders(t *testing.T) {
	providerState := filepath.Join(t.TempDir(), "provider_exists")

	// Fake openshell: tracks whether a provider has been created. First create
	// always succeeds. Subsequent creates return AlreadyExists until delete clears
	// the state, then create succeeds again.
	script := `#!/bin/sh
if [ "$1" = "provider" ] && [ "$2" = "create" ]; then
  if [ -f "` + providerState + `" ]; then
    # Provider exists from previous run — simulate AlreadyExists.
    echo "Error: × status: AlreadyExists" >&2
    exit 1
  fi
  echo "exists" > "` + providerState + `"
  exit 0
fi
if [ "$1" = "provider" ] && [ "$2" = "delete" ]; then
  rm -f "` + providerState + `"
  exit 0
fi
if [ "$1" = "gateway" ]; then
  exit 0
fi
if [ "$1" = "sandbox" ]; then
  if [ "$2" = "get" ]; then
    echo "Ready"
  fi
  exit 0
fi
exit 0
`

	setupFakeOpenshell(t, script)
	fullsendDir := setupMinimalHarness(t, "test-agent")
	targetRepo := t.TempDir()

	printer := ui.New(os.Stdout)

	// First run: provider does not exist, created fresh.
	err1 := runAgent("test-agent", fullsendDir, "", targetRepo, "", nil, true, printer)
	assert.NoError(t, err1, "first runAgent call should succeed (fresh provider creation)")

	// Second run: provider exists from first run, delete-and-recreate fires.
	err2 := runAgent("test-agent", fullsendDir, "", targetRepo, "", nil, true, printer)
	assert.NoError(t, err2, "second runAgent call should succeed (idempotent recovery)")
}
