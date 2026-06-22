package statuscomment

// STD Test Stubs for GH-72: StatusComment Notifier ClientFactory pattern
// Suite: TS-GH72-003
//
// These stubs correspond to test cases TC-GH72-009 through TC-GH72-018.
// Production tests: internal/statuscomment/statuscomment_test.go
// STP reference: outputs/stp/GH-72/GH-72_test_plan.md

import "testing"

// TC-GH72-009: ClientFactory called before PostStart API operations
//
// Preconditions:
//   - Notifier created with initial FakeClient fc1
//   - ClientFactory configured to return a different FakeClient fc2
//
// Steps:
//  1. Call PostStart on the Notifier
//
// Expected:
//   - factoryCalled flag is true
//   - Start comment appears on fc2 (factory-returned client)
//   - fc1 (original client) has no comments
func TestClientFactory_CalledBeforePostStart_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-009")
}

// TC-GH72-010: ClientFactory called before PostCompletion API operations
//
// Preconditions:
//   - PostStart already called successfully with default client
//   - ClientFactory set after PostStart to return fc2 with pre-populated comments
//
// Steps:
//  1. Call PostCompletion with "success" status
//
// Expected:
//   - completionFactoryCalled flag is true
//   - Completion operation uses the factory-minted client
func TestClientFactory_CalledBeforePostCompletion_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-010")
}

// TC-GH72-011: ClientFactory error propagated on PostStart
//
// Preconditions:
//   - ClientFactory configured to return error "mint service unavailable"
//
// Steps:
//  1. Call PostStart
//
// Expected:
//   - Error returned containing "mint service unavailable"
//   - No comment is created (static client not used as fallback)
func TestClientFactory_ErrorPropagated_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-011")
}

// TC-GH72-012: Static client used when no factory is set
//
// Preconditions:
//   - Notifier created with FakeClient, no factory set
//
// Steps:
//  1. Call PostStart
//
// Expected:
//   - Comment created on the static FakeClient (1 comment in issue comments)
func TestClientFactory_NilUsesStaticClient_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-012")
}

// TC-GH72-013: Completion-disabled path mints then deletes start comment
//
// Preconditions:
//   - Start comment exists (PostStart called with completion="disabled")
//   - ClientFactory returns fc2
//
// Steps:
//  1. Call PostCompletion with "success" status
//
// Expected:
//   - Factory is called (token refresh before cleanup)
//   - Start comment deleted via fc2.DeletedComments
func TestClientFactory_CompletionDisabled_DeletePath_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-013")
}

// TC-GH72-014: HasClientFactory reports factory presence
//
// Preconditions:
//   - Notifier created without factory
//
// Steps:
//  1. Check HasClientFactory before setting factory
//  2. Set factory, check HasClientFactory again
//
// Expected:
//   - Returns false before SetClientFactory
//   - Returns true after SetClientFactory
func TestHasClientFactory_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-014")
}

// TC-GH72-015: ClientFactory error on PostCompletion propagated
//
// Preconditions:
//   - PostStart succeeded, factory set to return error "token expired"
//
// Steps:
//  1. Call PostCompletion
//
// Expected:
//   - Error returned containing "token expired"
func TestClientFactory_ErrorOnPostCompletion_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-015")
}

// TC-GH72-016: Both disabled means no factory call
//
// Preconditions:
//   - Start and completion comments both disabled in config
//   - Factory configured to error (should never be called)
//
// Steps:
//  1. Call PostCompletion
//
// Expected:
//   - No error returned
//   - factoryCalled is false (factory never invoked)
func TestClientFactory_BothDisabled_NoMint_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-016")
}

// TC-GH72-017: Completion-disabled mint error is fail-open with warning
//
// Preconditions:
//   - Start comment exists, completion disabled
//   - Factory returns error "mint service down"
//   - WarnFunc configured to capture warnings
//
// Steps:
//  1. Call PostCompletion
//
// Expected:
//   - PostCompletion returns nil (fail-open behavior for cleanup)
//   - Warning emitted containing "mint service down"
func TestClientFactory_CompletionDisabled_MintError_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-017")
}

// TC-GH72-018: Completion-disabled delete error is fail-open with warning
//
// Preconditions:
//   - Start comment exists, completion disabled
//   - Factory returns fc2 with DeleteIssueComment error "forbidden"
//   - WarnFunc configured to capture warnings
//
// Steps:
//  1. Call PostCompletion
//
// Expected:
//   - PostCompletion returns nil (fail-open behavior for cleanup)
//   - Warning emitted containing "forbidden"
func TestClientFactory_CompletionDisabled_DeleteError_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-018")
}
