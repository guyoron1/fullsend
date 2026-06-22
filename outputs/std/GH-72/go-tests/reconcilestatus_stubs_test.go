package cli

// STD Test Stubs for GH-72: Reconcile-status CLI mint-url integration
// Suite: TS-GH72-007
//
// These stubs correspond to test cases TC-GH72-040 through TC-GH72-044.
// Production tests: internal/cli/reconcilestatus_test.go
// STP reference: outputs/stp/GH-72/GH-72_test_plan.md

import "testing"

// TC-GH72-040: Mint-url and role flags exist on reconcilestatus command
//
// Preconditions:
//   - reconcilestatus command created via newReconcileStatusCmd()
//
// Steps:
//  1. Look up --mint-url and --role flags on the command
//
// Expected:
//   - Both flags exist with empty default values
func TestReconcileStatusCmd_MintURLFlags_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-040")
}

// TC-GH72-041: FULLSEND_MINT_URL env var fallback
//
// Preconditions:
//   - FULLSEND_MINT_URL env var set to "https://mint.example.com"
//   - --role flag provided as "review"
//   - --mint-url flag NOT provided
//
// Steps:
//  1. Execute command with --repo, --number, --run-id, --role (no --mint-url)
//
// Expected:
//   - Command proceeds to OIDC exchange (fails due to missing ACTIONS_ID_TOKEN_REQUEST_URL)
//   - Error contains "minting status token" proving env var was picked up
func TestReconcileStatusCmd_MintURLFromEnv_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-041")
}

// TC-GH72-042: Error when --role missing with --mint-url
//
// Preconditions:
//   - --mint-url provided, --role NOT provided
//
// Steps:
//  1. Execute command with --mint-url but without --role
//
// Expected:
//   - Error returned: "--role is required when using --mint-url"
func TestReconcileStatusCmd_MintURLWithoutRole_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-042")
}

// TC-GH72-043: Deprecated --token flag still works
//
// Preconditions:
//   - httptest server returning empty JSON array (mocks GitHub API)
//   - FULLSEND_MINT_URL env var unset
//   - newForgeClient overridden to use test server
//
// Steps:
//  1. Execute command with --token test-token (deprecated flag)
//
// Expected:
//   - Command executes successfully (no error)
//   - --token flag is marked as deprecated
func TestReconcileStatusCmd_DeprecatedToken_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-043")
}

// TC-GH72-044: Error when neither --mint-url nor --token provided
//
// Preconditions:
//   - No --mint-url flag, no --token flag, no FULLSEND_MINT_URL env var
//
// Steps:
//  1. Execute command with only --repo, --number, --run-id
//
// Expected:
//   - Error: "--mint-url or FULLSEND_MINT_URL required"
func TestReconcileStatusCmd_NoAuth_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-044")
}
