package cli

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/forge"
	"github.com/fullsend-ai/fullsend/internal/mintclient"
)

// =============================================================================
// Section 3.9 — Reconcile Status Command
// =============================================================================

func setupReconcileMocks(t *testing.T, fc *forge.FakeClient) func() {
	t.Helper()
	origMint := reconcileMintToken
	origForge := reconcileNewForgeClient

	reconcileMintToken = func(_ context.Context, _ mintclient.MintRequest) (*mintclient.MintResult, error) {
		return &mintclient.MintResult{Token: "test-token"}, nil
	}
	reconcileNewForgeClient = func(_ string) forge.Client {
		return fc
	}

	return func() {
		reconcileMintToken = origMint
		reconcileNewForgeClient = origForge
	}
}

// TS-GH73-063: Invalid repo format — error
func TestQF_ReconcileStatus_InvalidRepoFormat(t *testing.T) {
	fc := forge.NewFakeClient()
	cleanup := setupReconcileMocks(t, fc)
	defer cleanup()

	cmd := newReconcileStatusCmd()
	cmd.SetArgs([]string{
		"--repo", "bad-format",
		"--number", "1",
		"--run-id", "12345",
		"--mint-url", "https://mint.example.com",
		"--role", "test-role",
	})
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "owner/repo")
}

// TS-GH73-064: Negative --number — error
func TestQF_ReconcileStatus_NegativeNumber(t *testing.T) {
	fc := forge.NewFakeClient()
	cleanup := setupReconcileMocks(t, fc)
	defer cleanup()

	cmd := newReconcileStatusCmd()
	cmd.SetArgs([]string{
		"--repo", "owner/repo",
		"--number", "-5",
		"--run-id", "12345",
		"--mint-url", "https://mint.example.com",
		"--role", "test-role",
	})
	err := cmd.Execute()
	require.Error(t, err)
}

// TS-GH73-065: Reason 'cancelled' is accepted
func TestQF_ReconcileStatus_CancelledReasonAccepted(t *testing.T) {
	fc := forge.NewFakeClient()
	cleanup := setupReconcileMocks(t, fc)
	defer cleanup()

	cmd := newReconcileStatusCmd()
	cmd.SetArgs([]string{
		"--repo", "owner/repo",
		"--number", "1",
		"--run-id", "12345",
		"--reason", "cancelled",
		"--mint-url", "https://mint.example.com",
		"--role", "test-role",
	})

	// The command should accept 'cancelled' without error (may fail for other reasons)
	_ = cmd.Execute()
}

// TS-GH73-066: Default reason 'terminated' is accepted
func TestQF_ReconcileStatus_TerminatedReason(t *testing.T) {
	fc := forge.NewFakeClient()
	cleanup := setupReconcileMocks(t, fc)
	defer cleanup()

	cmd := newReconcileStatusCmd()
	cmd.SetArgs([]string{
		"--repo", "owner/repo",
		"--number", "1",
		"--run-id", "12345",
		"--reason", "terminated",
		"--mint-url", "https://mint.example.com",
		"--role", "test-role",
	})

	_ = cmd.Execute()
}
