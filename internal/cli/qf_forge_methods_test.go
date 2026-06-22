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
// Section 3.10 — Forge Interface — New Methods
// =============================================================================

// TS-GH73-067: ListPullRequestFileDiffs returns files with patches
func TestQF_FakeClient_ListPullRequestFileDiffs(t *testing.T) {
	fc := forge.NewFakeClient()
	fc.PRFileDiffs = map[string][]forge.PullRequestFileDiff{
		"owner/repo/1": {
			{Path: "main.go", Patch: "@@ -1,5 +1,10 @@\n code"},
			{Path: "util.go", Patch: "@@ -10,3 +10,5 @@\n code"},
			{Path: "test.go", Patch: "@@ -1,1 +1,3 @@\n code"},
		},
	}

	diffs, err := fc.ListPullRequestFileDiffs(context.Background(), "owner", "repo", 1)

	require.NoError(t, err)
	assert.Len(t, diffs, 3)
	assert.NotEmpty(t, diffs[0].Patch)
}

// TS-GH73-068: ListPullRequestFileDiffs API error — graceful fallback
func TestQF_FakeClient_ListPullRequestFileDiffs_Error(t *testing.T) {
	fc := forge.NewFakeClient()
	fc.Errors = map[string]error{
		"ListPullRequestFileDiffs": assert.AnError,
	}

	_, err := fc.ListPullRequestFileDiffs(context.Background(), "owner", "repo", 1)

	require.Error(t, err)
}

// TS-GH73-069: ListPullRequestFileDiffs returns empty — inline comments disabled
func TestQF_SubmitFormalReview_EmptyFileDiffs(t *testing.T) {
	fc := forge.NewFakeClient()
	fc.AuthenticatedUser = "bot-user"
	fc.PullRequestHeadSHA = "abc1234567890abcdef1234567890abcdef1234ab"
	fc.PRFileDiffs = map[string][]forge.PullRequestFileDiff{
		"owner/repo/1": {},
	}
	printer := ui.New(io.Discard)

	findings := []ReviewFinding{
		{Severity: "high", Category: "bug", File: "main.go", Line: 5, Description: "issue"},
	}

	// With COMMENT and no inline-eligible comments → skipped
	err := submitFormalReview(
		context.Background(), fc, "owner", "repo", 1,
		"comment", "abc1234567890abcdef1234567890abcdef1234ab", "", findings, false, printer,
	)

	require.NoError(t, err)
	// Empty file diffs means no hunks → findings get file-filtered
	// For COMMENT verdict, no inline comments means review is skipped
}

// TS-GH73-070: DismissPullRequestReview success
func TestQF_FakeClient_DismissPullRequestReview(t *testing.T) {
	fc := forge.NewFakeClient()

	err := fc.DismissPullRequestReview(context.Background(), "owner", "repo", 1, 42, "Superseded by updated review")

	require.NoError(t, err)
	require.Len(t, fc.DismissedReviews, 1)
	assert.Equal(t, 42, fc.DismissedReviews[0].ReviewID)
	assert.Equal(t, "Superseded by updated review", fc.DismissedReviews[0].Message)
}

// TS-GH73-071: DismissPullRequestReview API error — soft-fail
func TestQF_FakeClient_DismissPullRequestReview_Error(t *testing.T) {
	fc := forge.NewFakeClient()
	fc.Errors = map[string]error{
		"DismissPullRequestReview": assert.AnError,
	}

	err := fc.DismissPullRequestReview(context.Background(), "owner", "repo", 1, 42, "msg")

	require.Error(t, err)
}

// TS-GH73-072: CreatePullRequestReview with inline comments
func TestQF_FakeClient_CreatePullRequestReview_InlineComments(t *testing.T) {
	fc := forge.NewFakeClient()

	comments := []forge.ReviewComment{
		{Path: "main.go", Line: 15, Body: "Issue found here"},
		{Path: "util.go", Line: 30, Body: "Another issue"},
	}

	err := fc.CreatePullRequestReview(context.Background(), "owner", "repo", 1, "COMMENT", "review body", "sha123", comments)

	require.NoError(t, err)
	require.Len(t, fc.CreatedReviews, 1)
	assert.Len(t, fc.CreatedReviews[0].Comments, 2)
	assert.Equal(t, "main.go", fc.CreatedReviews[0].Comments[0].Path)
}

// TS-GH73-073: ReviewComment with Line=0 — file-level comment
func TestQF_FakeClient_CreatePullRequestReview_FileLevelComment(t *testing.T) {
	fc := forge.NewFakeClient()

	comments := []forge.ReviewComment{
		{Path: "main.go", Line: 0, Body: "File-level finding"},
	}

	err := fc.CreatePullRequestReview(context.Background(), "owner", "repo", 1, "COMMENT", "", "sha123", comments)

	require.NoError(t, err)
	require.Len(t, fc.CreatedReviews, 1)
	assert.Equal(t, 0, fc.CreatedReviews[0].Comments[0].Line)
}
