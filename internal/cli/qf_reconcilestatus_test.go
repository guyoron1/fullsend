package cli

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/forge"
	gh "github.com/fullsend-ai/fullsend/internal/forge/github"
	"github.com/fullsend-ai/fullsend/internal/ui"
)

// QualityFlow generated tests for GH-76: bound enrollment wait with timeout and backoff
// Covers: reconcile-status CLI command validation, flag parsing, mint-url and
// deprecated --token flag handling.

func TestQF_ReconcileStatus_NumberNotPositive(t *testing.T) {
	cmd := newReconcileStatusCmd()
	cmd.SetArgs([]string{"--repo", "org/repo", "--number", "0", "--run-id", "run-1"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--number must be a positive integer")
}

func TestQF_ReconcileStatus_NegativeNumber(t *testing.T) {
	cmd := newReconcileStatusCmd()
	cmd.SetArgs([]string{"--repo", "org/repo", "--number", "-1", "--run-id", "run-1"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--number must be a positive integer")
}

func TestQF_ReconcileStatus_InvalidRepoFormat(t *testing.T) {
	cmd := newReconcileStatusCmd()
	cmd.SetArgs([]string{"--repo", "noslash", "--number", "7", "--run-id", "run-1"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--repo must be in owner/repo format")
}

func TestQF_ReconcileStatus_EmptyOwner(t *testing.T) {
	cmd := newReconcileStatusCmd()
	cmd.SetArgs([]string{"--repo", "/repo", "--number", "7", "--run-id", "run-1"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--repo must be in owner/repo format")
}

func TestQF_ReconcileStatus_EmptyRepoName(t *testing.T) {
	cmd := newReconcileStatusCmd()
	cmd.SetArgs([]string{"--repo", "org/", "--number", "7", "--run-id", "run-1"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--repo must be in owner/repo format")
}

func TestQF_ReconcileStatus_NoMintURLOrToken(t *testing.T) {
	t.Setenv("FULLSEND_MINT_URL", "")

	cmd := newReconcileStatusCmd()
	cmd.SetArgs([]string{"--repo", "org/repo", "--number", "7", "--run-id", "run-1"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--mint-url or FULLSEND_MINT_URL required")
}

func TestQF_ReconcileStatus_MintURLWithoutRole(t *testing.T) {
	cmd := newReconcileStatusCmd()
	cmd.SetArgs([]string{
		"--repo", "org/repo",
		"--number", "7",
		"--run-id", "run-1",
		"--mint-url", "https://mint.example.com",
	})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--role is required when using --mint-url")
}

func TestQF_ReconcileStatus_DeprecatedTokenStillWorks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[]"))
	}))
	defer srv.Close()

	origNew := newForgeClient
	newForgeClient = func(token string) forge.Client {
		return gh.New(token).WithBaseURL(srv.URL)
	}
	defer func() { newForgeClient = origNew }()

	t.Setenv("FULLSEND_MINT_URL", "")

	cmd := newReconcileStatusCmd()
	cmd.SetArgs([]string{
		"--repo", "org/repo",
		"--number", "7",
		"--run-id", "run-1",
		"--token", "test-token",
	})

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestQF_ReconcileStatus_TokenFlagIsDeprecated(t *testing.T) {
	cmd := newReconcileStatusCmd()
	f := cmd.Flags().Lookup("token")
	require.NotNil(t, f, "--token flag should exist for backwards compat")
	assert.NotEmpty(t, f.Deprecated, "--token should be marked deprecated")
}

func TestQF_ReconcileStatus_MintURLFromEnv(t *testing.T) {
	t.Setenv("FULLSEND_MINT_URL", "https://mint.example.com")

	cmd := newReconcileStatusCmd()
	cmd.SetArgs([]string{
		"--repo", "org/repo",
		"--number", "7",
		"--run-id", "run-1",
		"--role", "review",
	})

	err := cmd.Execute()
	// Will fail at OIDC exchange but proves the env var was picked up.
	require.Error(t, err)
	assert.Contains(t, err.Error(), "minting status token")
}

func TestQF_ReconcileStatus_ReasonDefaultTerminated(t *testing.T) {
	cmd := newReconcileStatusCmd()
	reason := cmd.Flags().Lookup("reason")
	require.NotNil(t, reason)
	assert.Equal(t, "terminated", reason.DefValue)
}

func TestQF_ReconcileStatus_CancelledReason(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[]"))
	}))
	defer srv.Close()

	origNew := newForgeClient
	newForgeClient = func(token string) forge.Client {
		return gh.New(token).WithBaseURL(srv.URL)
	}
	defer func() { newForgeClient = origNew }()

	t.Setenv("FULLSEND_MINT_URL", "")

	cmd := newReconcileStatusCmd()
	cmd.SetArgs([]string{
		"--repo", "org/repo",
		"--number", "7",
		"--run-id", "run-1",
		"--reason", "cancelled",
		"--token", "test-token",
	})

	err := cmd.Execute()
	require.NoError(t, err)
}

// --- setupStatusNotifier tests (TS-GH-76-019, TS-GH-76-020, TS-GH-76-021) ---

func TestQF_SetupStatusNotifier_MintURLFromFlag(t *testing.T) {
	// TS-GH-76-019: setupStatusNotifier uses --mint-url flag value to
	// configure the status notification client with a client factory.
	tmpDir := t.TempDir()
	printer := ui.New(io.Discard)

	sOpts := statusOpts{
		statusRepo: "org/repo",
		statusNum:  7,
		mintURL:    "https://mint.example.com",
	}

	t.Setenv("GITHUB_RUN_ID", "run-42")

	n, err := setupStatusNotifier(tmpDir, "review", sOpts, printer)
	require.NoError(t, err)
	assert.NotNil(t, n)
	assert.True(t, n.HasClientFactory(),
		"client factory should be set when --mint-url is provided via flag")
}

func TestQF_SetupStatusNotifier_FallsBackToMintURLEnv(t *testing.T) {
	// TS-GH-76-020: When --mint-url flag is not provided, setupStatusNotifier
	// falls back to FULLSEND_MINT_URL environment variable.
	tmpDir := t.TempDir()
	printer := ui.New(io.Discard)

	sOpts := statusOpts{
		statusRepo: "org/repo",
		statusNum:  7,
		// mintURL deliberately empty — should fall back to env
	}

	t.Setenv("FULLSEND_MINT_URL", "https://mint.example.com")
	t.Setenv("GITHUB_RUN_ID", "run-42")

	n, err := setupStatusNotifier(tmpDir, "code", sOpts, printer)
	require.NoError(t, err)
	assert.NotNil(t, n)
	assert.True(t, n.HasClientFactory(),
		"client factory should be set from FULLSEND_MINT_URL env var")
}

func TestQF_SetupStatusNotifier_ErrorWhenNoMintSource(t *testing.T) {
	// TS-GH-76-021: Returns error when neither --mint-url flag nor
	// FULLSEND_MINT_URL environment variable is set.
	tmpDir := t.TempDir()
	printer := ui.New(io.Discard)

	sOpts := statusOpts{
		statusRepo: "org/repo",
		statusNum:  7,
	}

	t.Setenv("FULLSEND_MINT_URL", "")
	t.Setenv("GITHUB_RUN_ID", "run-42")

	_, err := setupStatusNotifier(tmpDir, "review", sOpts, printer)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no mint URL available",
		"should indicate mint URL is missing")
}

// --- Run command flag acceptance (TS-GH-76-026) ---

func TestQF_RunCommand_AcceptsMintURLFlag(t *testing.T) {
	// TS-GH-76-026: The run command's CLI accepts --mint-url parameter.
	cmd := newRunCmd()

	f := cmd.Flags().Lookup("mint-url")
	require.NotNil(t, f, "run command should expose --mint-url flag")
	assert.Equal(t, "", f.DefValue, "default should be empty (env fallback)")
}

func TestQF_RunCommand_DeprecatedStatusTokenFlag(t *testing.T) {
	// TS-GH-76-018 (run variant): The --status-token flag is deprecated
	// but still present for backwards compatibility.
	cmd := newRunCmd()

	f := cmd.Flags().Lookup("status-token")
	require.NotNil(t, f, "run command should have --status-token for backwards compat")
	assert.NotEmpty(t, f.Deprecated, "--status-token should be marked deprecated")
}

// --- E2E flow: setupStatusNotifier with config.yaml (TS-GH-76-027) ---

func TestQF_SetupStatusNotifier_LoadsConfigYAML(t *testing.T) {
	// TS-GH-76-027: End-to-end flow from config loading through notifier setup.
	tmpDir := t.TempDir()
	printer := ui.New(io.Discard)

	configData := `defaults:
  status_notifications:
    comment:
      start: enabled
      completion: enabled
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(configData), 0o644))

	sOpts := statusOpts{
		statusRepo: "org/repo",
		statusNum:  7,
		mintURL:    "https://mint.example.com",
	}

	t.Setenv("GITHUB_RUN_ID", "run-42")

	n, err := setupStatusNotifier(tmpDir, "review", sOpts, printer)
	require.NoError(t, err)
	assert.NotNil(t, n)
	assert.True(t, n.HasClientFactory(),
		"notifier should be fully configured with config.yaml and mint URL")
}

func TestQF_SetupStatusNotifier_DeprecatedTokenNoFactory(t *testing.T) {
	// TS-GH-76-018 (setup variant): When using deprecated --status-token,
	// no client factory is set (static token used directly).
	tmpDir := t.TempDir()
	printer := ui.New(io.Discard)

	sOpts := statusOpts{
		statusRepo:  "org/repo",
		statusNum:   7,
		statusToken: "test-static-token",
	}

	t.Setenv("FULLSEND_MINT_URL", "")
	t.Setenv("GITHUB_RUN_ID", "run-42")

	n, err := setupStatusNotifier(tmpDir, "code", sOpts, printer)
	require.NoError(t, err)
	assert.NotNil(t, n)
	assert.False(t, n.HasClientFactory(),
		"static token should not set client factory")
}
