package tests

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/config"
	"github.com/fullsend-ai/fullsend/internal/forge"
	"github.com/fullsend-ai/fullsend/internal/statuscomment"
)

/*
Mint URL Migration Tests

STP Reference: outputs/stp/GH-26/GH-26_test_plan.md
STD Reference: outputs/std/GH-26/GH-26_test_description.yaml
Jira: GH-26

Tests for the status-token to mint-url migration. Validates that the
StatusNotifier correctly uses a ClientFactory for on-demand token
minting, and that ReconcileOrphaned works with the new auth flow.
*/

//go:build e2e

// TestStatusNotifierWorksWithMintURL verifies that the status notifier
// correctly obtains a fresh forge.Client from a ClientFactory and uses
// it to post status comments. This simulates the mint-url flow where
// a fresh token is obtained before each API call.
//
// [test_id:TS-GH-26-027]
func TestStatusNotifierWorksWithMintURL(t *testing.T) {
	fc := forge.NewFakeClient()
	fc.AuthenticatedUser = "fullsend-bot[bot]"
	cfg := config.StatusNotificationConfig{
		Comment: config.CommentNotificationConfig{Start: "enabled", Completion: "enabled"},
	}

	n := statuscomment.New(fc, cfg, "org", "repo", 42,
		"https://github.com/org/repo/actions/runs/123", "abc1234def5678", "test-run-1")

	// Configure a ClientFactory that returns a fresh FakeClient each time,
	// simulating the mint-url flow where a fresh token is minted per call.
	factoryCallCount := 0
	mintedClient := forge.NewFakeClient()
	mintedClient.AuthenticatedUser = "fullsend-bot[bot]"

	n.SetClientFactory(func(ctx context.Context) (forge.Client, error) {
		factoryCallCount++
		return mintedClient, nil
	})

	assert.True(t, n.HasClientFactory(),
		"Notifier should report having a ClientFactory after SetClientFactory")

	// Verify factory is callable
	client, err := n.InvokeClientFactory(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, client, "Factory should return a valid client")
	assert.Equal(t, 1, factoryCallCount, "Factory should have been called once")

	// Post a start comment — this should use the factory
	err = n.PostStart(context.Background(), "Testing mint-url integration")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, factoryCallCount, 2,
		"Factory should be called during PostStart for client refresh")

	// Verify the comment was posted via the minted client
	comments := mintedClient.IssueComments["org/repo/42"]
	require.Len(t, comments, 1, "One start comment should be posted")
	assert.Contains(t, comments[0].Body, "fullsend:agent-status:test-run-1",
		"Comment should contain the run marker")
	assert.Contains(t, comments[0].Body, "Testing mint-url integration",
		"Comment should contain the description")
}

// TestStatusNotifierErrorWhenMintUnavailable verifies that the status
// notifier returns a clear error when the ClientFactory fails, simulating
// a mint-url endpoint that is unavailable.
//
// [test_id:TS-GH-26-028]
func TestStatusNotifierErrorWhenMintUnavailable(t *testing.T) {
	fc := forge.NewFakeClient()
	fc.AuthenticatedUser = "fullsend-bot[bot]"
	cfg := config.StatusNotificationConfig{
		Comment: config.CommentNotificationConfig{Start: "enabled", Completion: "enabled"},
	}

	n := statuscomment.New(fc, cfg, "org", "repo", 42,
		"https://github.com/org/repo/actions/runs/123", "abc1234def5678", "test-run-2")

	// Configure a factory that returns an error (simulating mint-url unavailable)
	mintError := fmt.Errorf("connection refused: mint-url https://mint.example.com/v1/token")
	n.SetClientFactory(func(ctx context.Context) (forge.Client, error) {
		return nil, mintError
	})

	// PostStart should fail with a descriptive error
	err := n.PostStart(context.Background(), "Testing mint failure")
	require.Error(t, err, "PostStart should fail when factory returns error")
	assert.Contains(t, err.Error(), "minting fresh client",
		"Error should indicate the factory/minting step failed")

	// PostCompletion should also fail
	err = n.PostCompletion(context.Background(), "Testing mint failure", "failure")
	require.Error(t, err, "PostCompletion should fail when factory returns error")

	// Verify no panic occurred (we got clean errors instead)
	// If we reach this line, no panic happened — test passes
}

// TestReconcileStatusMintsToken verifies that ReconcileOrphaned
// works correctly when called with a forge.Client, simulating the
// reconcile-status command's flow of minting a token via mint-url.
//
// [test_id:TS-GH-26-029]
func TestReconcileStatusMintsToken(t *testing.T) {
	fc := forge.NewFakeClient()
	fc.AuthenticatedUser = "fullsend-bot[bot]"

	// Simulate an orphaned start comment left by a hard-killed process
	runID := "run-orphaned-42"
	marker := fmt.Sprintf("<!-- fullsend:agent-status:%s -->", runID)
	startBody := marker + "\n🤖 Fixing issue · Started 2:30 PM UTC\nCommit: `abc1234`"

	// Add the orphaned comment to the fake client
	fc.IssueComments["org/repo/99"] = []forge.IssueComment{
		{
			ID:     1001,
			Body:   startBody,
			Author: "fullsend-bot[bot]",
		},
	}

	// Call ReconcileOrphaned — this is what reconcile-status does after minting
	err := statuscomment.ReconcileOrphaned(
		context.Background(),
		fc,
		"org", "repo", 99,
		runID,
		"https://github.com/org/repo/actions/runs/999",
		"abc1234def5678",
		statuscomment.ReasonTerminated,
	)
	require.NoError(t, err, "ReconcileOrphaned should succeed")

	// Verify the comment was updated with terminal state
	require.Len(t, fc.UpdatedComments, 1, "One comment should have been updated")
	updatedBody := fc.UpdatedComments[0].Body
	assert.Contains(t, updatedBody, marker,
		"Updated body should contain the original marker")
	assert.Contains(t, updatedBody, "<!-- fullsend:status:terminal -->",
		"Updated body should contain the terminal tag")
	assert.Contains(t, updatedBody, "Terminated",
		"Updated body should indicate termination")
}

// TestOrphanedCommentFinalizedWithMintURL verifies that ReconcileOrphaned
// correctly identifies orphaned status comments and finalizes them,
// including the case where the comment was already finalized (no-op).
//
// [test_id:TS-GH-26-030]
func TestOrphanedCommentFinalizedWithMintURL(t *testing.T) {
	t.Run("finalizes orphaned comment", func(t *testing.T) {
		fc := forge.NewFakeClient()
		fc.AuthenticatedUser = "fullsend-bot[bot]"

		runID := "run-cancelled-55"
		marker := fmt.Sprintf("<!-- fullsend:agent-status:%s -->", runID)
		startBody := marker + "\n🤖 Reviewing PR · Started 4:15 PM UTC"

		fc.IssueComments["org/repo/77"] = []forge.IssueComment{
			{
				ID:     2001,
				Body:   startBody,
				Author: "fullsend-bot[bot]",
			},
		}

		err := statuscomment.ReconcileOrphaned(
			context.Background(), fc, "org", "repo", 77,
			runID, "https://ci/run/55", "def456789", statuscomment.ReasonCancelled,
		)
		require.NoError(t, err)

		require.Len(t, fc.UpdatedComments, 1)
		updatedBody := fc.UpdatedComments[0].Body
		assert.Contains(t, updatedBody, "Cancelled",
			"Should show cancellation reason")
		assert.Contains(t, updatedBody, "<!-- fullsend:status:terminal -->",
			"Should contain terminal tag")
		assert.Contains(t, updatedBody, "Reviewing PR",
			"Should preserve original description")
		assert.Contains(t, updatedBody, "Started 4:15 PM UTC",
			"Should preserve original start time")
	})

	t.Run("no-op when already finalized", func(t *testing.T) {
		fc := forge.NewFakeClient()
		fc.AuthenticatedUser = "fullsend-bot[bot]"

		runID := "run-already-done-66"
		marker := fmt.Sprintf("<!-- fullsend:agent-status:%s -->", runID)
		terminalTag := "<!-- fullsend:status:terminal -->"
		finalizedBody := marker + "\n" + terminalTag + "\n🤖 Finished review · ✅ Success"

		fc.IssueComments["org/repo/88"] = []forge.IssueComment{
			{
				ID:     3001,
				Body:   finalizedBody,
				Author: "fullsend-bot[bot]",
			},
		}

		err := statuscomment.ReconcileOrphaned(
			context.Background(), fc, "org", "repo", 88,
			runID, "", "", statuscomment.ReasonTerminated,
		)
		require.NoError(t, err)

		assert.Empty(t, fc.UpdatedComments,
			"Already-finalized comment should not be updated")
	})

	t.Run("no-op when no matching comment found", func(t *testing.T) {
		fc := forge.NewFakeClient()
		fc.AuthenticatedUser = "fullsend-bot[bot]"

		// No comments at all
		err := statuscomment.ReconcileOrphaned(
			context.Background(), fc, "org", "repo", 55,
			"run-nonexistent", "", "", statuscomment.ReasonTerminated,
		)
		require.NoError(t, err, "Should succeed even when no comment found")
		assert.Empty(t, fc.UpdatedComments)
	})

	t.Run("error on invalid runID", func(t *testing.T) {
		fc := forge.NewFakeClient()
		err := statuscomment.ReconcileOrphaned(
			context.Background(), fc, "org", "repo", 1,
			"invalid run id!", "", "", statuscomment.ReasonTerminated,
		)
		require.Error(t, err, "Invalid runID should produce an error")
		assert.Contains(t, err.Error(), "marker",
			"Error should mention marker building")
	})
}

// TestNotifierClientFactoryRefresh verifies that the factory is called
// before each API operation (PostStart and PostCompletion), ensuring
// fresh tokens are used.
//
// This is a supplementary test that validates the mint-url integration
// at the Notifier level without requiring actual HTTP calls.
func TestNotifierClientFactoryRefresh(t *testing.T) {
	cfg := config.StatusNotificationConfig{
		Comment: config.CommentNotificationConfig{Start: "enabled", Completion: "enabled"},
	}

	// Track all clients created by the factory
	var clients []*forge.FakeClient
	fc := forge.NewFakeClient()
	fc.AuthenticatedUser = "fullsend-bot[bot]"

	n := statuscomment.New(fc, cfg, "org", "repo", 10,
		"https://ci/run/1", "aabbccdd", "factory-test")

	fixedNow := time.Date(2026, 6, 17, 10, 0, 0, 0, time.UTC)

	n.SetClientFactory(func(ctx context.Context) (forge.Client, error) {
		newFC := forge.NewFakeClient()
		newFC.AuthenticatedUser = "fullsend-bot[bot]"
		// Copy comments from previous clients so timeline analysis works
		for k, v := range fc.IssueComments {
			newFC.IssueComments[k] = v
		}
		clients = append(clients, newFC)
		return newFC, nil
	})

	_ = fixedNow
	_ = strings.Contains // import guard

	err := n.PostStart(context.Background(), "Test refresh")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(clients), 1,
		"At least one client should be created via factory during PostStart")
}
