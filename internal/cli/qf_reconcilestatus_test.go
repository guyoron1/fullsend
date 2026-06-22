package cli

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/forge"
	gh "github.com/fullsend-ai/fullsend/internal/forge/github"
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
