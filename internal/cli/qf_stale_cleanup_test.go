package cli

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/forge"
	"github.com/fullsend-ai/fullsend/internal/ui"
)

// =============================================================================
// Section 3.4 — Post-Review — Stale Review Cleanup
// =============================================================================

// TS-GH73-025: Bot has prior COMMENTED reviews — minimized
func TestQF_MinimizeStaleReviews_CommentedReviews(t *testing.T) {
	fc := forge.NewFakeClient()
	printer := ui.New(io.Discard)

	reviews := []forge.PullRequestReview{
		{ID: 100, NodeID: "node100", User: "bot-user", State: "COMMENTED", Body: "old review 1"},
		{ID: 101, NodeID: "node101", User: "bot-user", State: "COMMENTED", Body: "old review 2"},
	}

	minimizeStaleReviews(context.Background(), fc, "bot-user", reviews, printer)

	assert.Equal(t, 2, len(fc.MinimizedComments))
	assert.Equal(t, "OUTDATED", fc.MinimizedComments[0].Reason)
	assert.Equal(t, "OUTDATED", fc.MinimizedComments[1].Reason)
}

// TS-GH73-026: Bot has prior CR, new=APPROVE — dismissed
func TestQF_DismissStaleRequestChanges_CRToApprove(t *testing.T) {
	fc := forge.NewFakeClient()
	printer := ui.New(io.Discard)

	reviews := []forge.PullRequestReview{
		{ID: 200, NodeID: "node200", User: "bot-user", State: "CHANGES_REQUESTED", Body: "changes needed"},
	}

	dismissStaleRequestChanges(context.Background(), fc, "owner", "repo", 1, "APPROVE", "bot-user", reviews, printer)

	assert.Equal(t, 1, len(fc.DismissedReviews))
	assert.Contains(t, fc.DismissedReviews[0].Message, "Superseded")
}

// TS-GH73-027: Bot has prior CR, new=COMMENT — dismissed
func TestQF_DismissStaleRequestChanges_CRToComment(t *testing.T) {
	fc := forge.NewFakeClient()
	printer := ui.New(io.Discard)

	reviews := []forge.PullRequestReview{
		{ID: 200, NodeID: "node200", User: "bot-user", State: "CHANGES_REQUESTED", Body: "changes needed"},
	}

	dismissStaleRequestChanges(context.Background(), fc, "owner", "repo", 1, "COMMENT", "bot-user", reviews, printer)

	assert.Equal(t, 1, len(fc.DismissedReviews))
}

// TS-GH73-028: Bot has prior CR, new=REQUEST_CHANGES — NOT dismissed
func TestQF_DismissStaleRequestChanges_CRToCR(t *testing.T) {
	fc := forge.NewFakeClient()
	printer := ui.New(io.Discard)

	reviews := []forge.PullRequestReview{
		{ID: 200, NodeID: "node200", User: "bot-user", State: "CHANGES_REQUESTED", Body: "changes needed"},
	}

	dismissStaleRequestChanges(context.Background(), fc, "owner", "repo", 1, "REQUEST_CHANGES", "bot-user", reviews, printer)

	assert.Empty(t, fc.DismissedReviews)
}

// TS-GH73-029: Other user's CR reviews not dismissed
func TestQF_DismissStaleRequestChanges_OtherUserNotDismissed(t *testing.T) {
	fc := forge.NewFakeClient()
	printer := ui.New(io.Discard)

	reviews := []forge.PullRequestReview{
		{ID: 200, NodeID: "node200", User: "human-reviewer", State: "CHANGES_REQUESTED", Body: "changes needed"},
	}

	dismissStaleRequestChanges(context.Background(), fc, "owner", "repo", 1, "APPROVE", "bot-user", reviews, printer)

	assert.Empty(t, fc.DismissedReviews)
}

// TS-GH73-030: Multiple stale CR reviews by bot — all dismissed
func TestQF_DismissStaleRequestChanges_MultipleCR(t *testing.T) {
	fc := forge.NewFakeClient()
	printer := ui.New(io.Discard)

	reviews := []forge.PullRequestReview{
		{ID: 200, NodeID: "node200", User: "bot-user", State: "CHANGES_REQUESTED", Body: "cr 1"},
		{ID: 201, NodeID: "node201", User: "bot-user", State: "CHANGES_REQUESTED", Body: "cr 2"},
		{ID: 202, NodeID: "node202", User: "bot-user", State: "CHANGES_REQUESTED", Body: "cr 3"},
	}

	dismissStaleRequestChanges(context.Background(), fc, "owner", "repo", 1, "APPROVE", "bot-user", reviews, printer)

	assert.Equal(t, 3, len(fc.DismissedReviews))
}

// TS-GH73-031: MinimizeComment API error — soft-fail
func TestQF_MinimizeStaleReviews_APIError_SoftFail(t *testing.T) {
	fc := forge.NewFakeClient()
	fc.Errors = map[string]error{
		"MinimizeComment": assert.AnError,
	}
	printer := ui.New(io.Discard)

	reviews := []forge.PullRequestReview{
		{ID: 100, NodeID: "node100", User: "bot-user", State: "COMMENTED"},
	}

	// Should not panic
	require.NotPanics(t, func() {
		minimizeStaleReviews(context.Background(), fc, "bot-user", reviews, printer)
	})
}

// TS-GH73-032: GetAuthenticatedUser error — skips cleanup
func TestQF_SubmitFormalReview_AuthError_SkipsCleanup(t *testing.T) {
	fc := forge.NewFakeClient()
	fc.Errors = map[string]error{
		"GetAuthenticatedUser": assert.AnError,
	}
	printer := ui.New(io.Discard)

	err := submitFormalReview(
		context.Background(), fc, "owner", "repo", 1,
		"approve", "", "", nil, false, printer,
	)

	require.NoError(t, err)
	assert.Empty(t, fc.DismissedReviews)
	assert.Empty(t, fc.MinimizedComments)
}

// TS-GH73-033: ListPullRequestReviews error — skips cleanup
func TestQF_SubmitFormalReview_ListReviewsError_SkipsCleanup(t *testing.T) {
	fc := forge.NewFakeClient()
	fc.AuthenticatedUser = "bot-user"
	fc.Errors = map[string]error{
		"ListPullRequestReviews": assert.AnError,
	}
	printer := ui.New(io.Discard)

	err := submitFormalReview(
		context.Background(), fc, "owner", "repo", 1,
		"approve", "", "", nil, false, printer,
	)

	require.NoError(t, err)
	assert.Empty(t, fc.DismissedReviews)
	assert.Empty(t, fc.MinimizedComments)
}
