package cli

// QualityFlow generated tests for GH-72
// Suite: TS-GH72-008 — Run command mint-url for status comment authentication
// STD: outputs/std/GH-72/GH-72_test_description.yaml

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/ui"
)

// TC-GH72-045: Client factory set from --mint-url flag
func TestQFSetupStatusNotifier_MintURL(t *testing.T) {
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
		"client factory should be set when mint URL provided")
}

// TC-GH72-046: FULLSEND_MINT_URL env var picked up by run command
func TestQFSetupStatusNotifier_MintURLFromEnv(t *testing.T) {
	tmpDir := t.TempDir()
	printer := ui.New(io.Discard)

	sOpts := statusOpts{
		statusRepo: "org/repo",
		statusNum:  7,
	}

	t.Setenv("FULLSEND_MINT_URL", "https://mint.example.com")
	t.Setenv("GITHUB_RUN_ID", "run-42")

	n, err := setupStatusNotifier(tmpDir, "code", sOpts, printer)
	require.NoError(t, err)
	assert.NotNil(t, n)
	assert.True(t, n.HasClientFactory(),
		"client factory should be set from FULLSEND_MINT_URL env var")
}

// TC-GH72-047: Error when no mint-url or token available
func TestQFSetupStatusNotifier_NoMintURL(t *testing.T) {
	tmpDir := t.TempDir()
	printer := ui.New(io.Discard)

	sOpts := statusOpts{
		statusRepo: "org/repo",
		statusNum:  7,
	}

	t.Setenv("GITHUB_RUN_ID", "run-42")
	t.Setenv("FULLSEND_MINT_URL", "")
	t.Setenv("GITHUB_TOKEN", "")

	_, err := setupStatusNotifier(tmpDir, "review", sOpts, printer)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no mint URL available",
		"should error when neither mint URL nor token available")
}

// TC-GH72-048: Deprecated static token creates client directly without factory
func TestQFSetupStatusNotifier_DeprecatedToken(t *testing.T) {
	tmpDir := t.TempDir()
	printer := ui.New(io.Discard)

	sOpts := statusOpts{
		statusRepo:  "org/repo",
		statusNum:   7,
		statusToken: "test-static-token",
	}

	t.Setenv("GITHUB_RUN_ID", "run-42")
	t.Setenv("FULLSEND_MINT_URL", "")

	n, err := setupStatusNotifier(tmpDir, "code", sOpts, printer)
	require.NoError(t, err)
	assert.NotNil(t, n)
	assert.False(t, n.HasClientFactory(),
		"client factory should not be set when using deprecated static token")
}

// TC-GH72-049: Run command has --mint-url flag
func TestQFRunCommand_HasMintURLFlag(t *testing.T) {
	cmd := newRunCmd()

	f := cmd.Flags().Lookup("mint-url")
	require.NotNil(t, f, "run command should have --mint-url flag")
	assert.Equal(t, "", f.DefValue)
}

// TC-GH72-050: Run command --status-token flag is marked deprecated
func TestQFRunCommand_StatusTokenFlagDeprecated(t *testing.T) {
	cmd := newRunCmd()

	f := cmd.Flags().Lookup("status-token")
	require.NotNil(t, f, "run command should have --status-token flag for backwards compatibility")
	assert.NotEmpty(t, f.Deprecated, "--status-token flag should be marked deprecated")
}
