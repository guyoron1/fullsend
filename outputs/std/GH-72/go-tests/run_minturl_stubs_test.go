package cli

// STD Test Stubs for GH-72: Run command mint-url integration
// Suite: TS-GH72-008
//
// These stubs correspond to test cases TC-GH72-045 through TC-GH72-050.
// Production tests: internal/cli/run_test.go
// STP reference: outputs/stp/GH-72/GH-72_test_plan.md

import "testing"

// TC-GH72-045: Client factory set from --mint-url flag
//
// Preconditions:
//   - statusOpts with mintURL="https://mint.example.com"
//   - GITHUB_RUN_ID env var set to "run-42"
//   - tmpDir created for fullsend directory
//
// Steps:
//  1. Call setupStatusNotifier(tmpDir, "review", sOpts, printer)
//
// Expected:
//   - Returns non-nil Notifier
//   - Notifier.HasClientFactory() returns true
func TestSetupStatusNotifier_MintURL_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-045")
}

// TC-GH72-046: FULLSEND_MINT_URL env var picked up
//
// Preconditions:
//   - FULLSEND_MINT_URL env var set to "https://mint.example.com"
//   - statusOpts without mintURL (empty string)
//   - GITHUB_RUN_ID env var set
//
// Steps:
//  1. Call setupStatusNotifier with empty mintURL in opts
//
// Expected:
//   - Returns Notifier with HasClientFactory() == true
//   - Env var used as fallback for missing --mint-url flag
func TestSetupStatusNotifier_MintURLFromEnv_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-046")
}

// TC-GH72-047: Error when no mint-url or token available
//
// Preconditions:
//   - No mintURL in opts, no FULLSEND_MINT_URL env var, no statusToken
//   - GITHUB_RUN_ID env var set
//
// Steps:
//  1. Call setupStatusNotifier with empty opts
//
// Expected:
//   - Error returned: "no mint URL available"
//   - No Notifier created
func TestSetupStatusNotifier_NoMintURL_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-047")
}

// TC-GH72-048: Deprecated static token creates client without factory
//
// Preconditions:
//   - statusOpts with statusToken="test-static-token", no mintURL
//   - FULLSEND_MINT_URL env var unset
//   - GITHUB_RUN_ID env var set
//
// Steps:
//  1. Call setupStatusNotifier with static token in opts
//
// Expected:
//   - Returns non-nil Notifier
//   - Notifier.HasClientFactory() returns false (static client, no factory)
func TestSetupStatusNotifier_DeprecatedToken_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-048")
}

// TC-GH72-049: Run command has --mint-url flag
//
// Preconditions:
//   - Run command created via newRunCmd()
//
// Steps:
//  1. Look up --mint-url flag on the command
//
// Expected:
//   - Flag exists with empty default value
func TestRunCommand_HasMintURLFlag_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-049")
}

// TC-GH72-050: Run command --status-token flag is marked deprecated
//
// Preconditions:
//   - Run command created via newRunCmd()
//
// Steps:
//  1. Look up --status-token flag on the command
//
// Expected:
//   - Flag exists with non-empty Deprecated field
func TestRunCommand_StatusTokenFlagDeprecated_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-050")
}
