package github_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/forge"
	ghclient "github.com/fullsend-ai/fullsend/internal/forge/github"
)

/*
Forge Client Interface Compliance Tests

STP Reference: outputs/stp/GH-2432/GH-2432_test_plan.md
STD Reference: outputs/std/GH-2432/GH-2432_test_description.yaml
Jira: GH-2432

These tests verify that the MergeChangeProposal changes do not break the
forge.Client interface contract. LiveClient must continue to satisfy the
interface, and FakeClient must continue to work for integration tests.

Shared Preconditions:
    - Go 1.23+ toolchain available
    - Source code compiles without errors
*/

// TestLiveClient_ImplementsForgeClient validates that the LiveClient type
// still satisfies the forge.Client interface after the MergeChangeProposal
// changes. The method signature must remain unchanged.
//
// Requirement: REQ-001
// Priority: P1
func TestLiveClient_ImplementsForgeClient(t *testing.T) {
	// [test_id:TS-GH-2432-005]

	// Compile-time interface compliance check.
	// If LiveClient ever stops implementing forge.Client, this line will
	// cause a compilation error, catching interface-breaking changes at
	// build time rather than test time.
	var _ forge.Client = (*ghclient.LiveClient)(nil)

	// Runtime verification: ensure we can assign a real instance.
	client := ghclient.New("test-token")
	var iface forge.Client = client
	assert.NotNil(t, iface, "LiveClient should be assignable to forge.Client")
}

// TestFakeClient_MergeChangeProposal validates that the FakeClient mock
// implementation of MergeChangeProposal continues to work correctly,
// delegating to the configured error function and returning expected results.
//
// Requirement: REQ-001
// Priority: P2
func TestFakeClient_MergeChangeProposal(t *testing.T) {
	// [test_id:TS-GH-2432-006]

	t.Run("returns nil when no error configured", func(t *testing.T) {
		fakeClient := forge.NewFakeClient()
		ctx := context.Background()

		err := fakeClient.MergeChangeProposal(ctx, "org", "repo", 42)

		// ASSERT-01: FakeClient returns nil when no error configured
		require.NoError(t, err,
			"FakeClient broken — integration tests affected")
	})

	t.Run("returns configured error", func(t *testing.T) {
		expectedErr := errors.New("merge blocked by policy")
		fakeClient := forge.NewFakeClient()
		fakeClient.Errors["MergeChangeProposal"] = expectedErr
		ctx := context.Background()

		err := fakeClient.MergeChangeProposal(ctx, "org", "repo", 42)

		// ASSERT-02: FakeClient returns configured error
		require.Error(t, err,
			"FakeClient error injection broken")
		assert.ErrorIs(t, err, expectedErr,
			"FakeClient should return the exact error that was configured")
	})

	t.Run("interface compliance", func(t *testing.T) {
		// Verify FakeClient satisfies forge.Client interface
		var _ forge.Client = (*forge.FakeClient)(nil)

		fakeClient := forge.NewFakeClient()
		var iface forge.Client = fakeClient
		assert.NotNil(t, iface,
			"FakeClient should be assignable to forge.Client")
	})
}
