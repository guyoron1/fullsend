package cli

// QualityFlow generated tests for GH-72
// Suite: TS-GH72-007 — Reconcile-status command mint-url authentication
// STD: outputs/std/GH-72/GH-72_test_description.yaml

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/forge"
	gh "github.com/fullsend-ai/fullsend/internal/forge/github"
)

// TC-GH72-040: Mint-url flag and role flags exist on reconcilestatus command
func TestQFNewReconcileStatusCmd_MintURLFlags(t *testing.T) {
	cmd := newReconcileStatusCmd()

	for _, name := range []string{"mint-url", "role"} {
		f := cmd.Flags().Lookup(name)
		require.NotNil(t, f, "flag %q should exist", name)
	}

	mintURL := cmd.Flags().Lookup("mint-url")
	assert.Equal(t, "", mintURL.DefValue, "mint-url should default to empty")

	role := cmd.Flags().Lookup("role")
	assert.Equal(t, "", role.DefValue, "role should default to empty")
}

// TC-GH72-041: FULLSEND_MINT_URL env var fallback when --mint-url not provided
func TestQFNewReconcileStatusCmd_MintURLFromEnv(t *testing.T) {
	t.Setenv("FULLSEND_MINT_URL", "https://mint.example.com")

	cmd := newReconcileStatusCmd()
	cmd.SetArgs([]string{"--repo", "org/repo", "--number", "7", "--run-id", "run-1", "--role", "review"})
	err := cmd.Execute()
	// Will fail at OIDC exchange (no ACTIONS_ID_TOKEN_REQUEST_URL),
	// but proves the env var was picked up and --role validation passed.
	require.Error(t, err)
	assert.Contains(t, err.Error(), "minting status token",
		"error proves env var was picked up for token minting")
}

// TC-GH72-042: Error when --role missing with --mint-url
func TestQFNewReconcileStatusCmd_ValidationErrors_MintURLWithoutRole(t *testing.T) {
	cmd := newReconcileStatusCmd()
	cmd.SetArgs([]string{
		"--repo", "org/repo",
		"--number", "7",
		"--run-id", "run-1",
		"--mint-url", "https://mint.example.com",
	})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--role is required when using --mint-url",
		"should produce clear validation error")
}

// TC-GH72-043: Deprecated --token flag still works for backward compatibility
func TestQFNewReconcileStatusCmd_DeprecatedTokenExecution(t *testing.T) {
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
	require.NoError(t, err, "deprecated --token flag should still function")

	f := cmd.Flags().Lookup("token")
	assert.NotEmpty(t, f.Deprecated, "--token flag should be marked as deprecated")
}

// TC-GH72-044: Error when neither --mint-url nor --token provided
func TestQFNewReconcileStatusCmd_ValidationErrors_MissingMintURL(t *testing.T) {
	t.Setenv("FULLSEND_MINT_URL", "")

	cmd := newReconcileStatusCmd()
	cmd.SetArgs([]string{
		"--repo", "org/repo",
		"--number", "7",
		"--run-id", "run-1",
	})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--mint-url or FULLSEND_MINT_URL required",
		"should fail with clear authentication error")
}
