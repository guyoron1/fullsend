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
// Section 3.3 — Post-Review — Formal Review Submission
// =============================================================================

// TS-GH73-014: Submit APPROVE review
func TestQF_SubmitFormalReview_Approve(t *testing.T) {
	fc := forge.NewFakeClient()
	fc.AuthenticatedUser = "bot-user"
	fc.PullRequestHeadSHA = "abc1234567890abcdef1234567890abcdef1234ab"
	printer := ui.New(io.Discard)

	err := submitFormalReview(
		context.Background(), fc, "owner", "repo", 1,
		"approve", "abc1234567890abcdef1234567890abcdef1234ab", "", nil, false, printer,
	)

	require.NoError(t, err)
	require.NotEmpty(t, fc.CreatedReviews)
	assert.Equal(t, "APPROVE", fc.CreatedReviews[0].Event)
	assert.Empty(t, fc.CreatedReviews[0].Body)
}

// TS-GH73-015: Submit REQUEST_CHANGES with comment URL
func TestQF_SubmitFormalReview_RequestChangesWithURL(t *testing.T) {
	fc := forge.NewFakeClient()
	fc.AuthenticatedUser = "bot-user"
	fc.PullRequestHeadSHA = "abc1234567890abcdef1234567890abcdef1234ab"
	printer := ui.New(io.Discard)
	commentURL := "https://github.com/owner/repo/pull/1#issuecomment-123"

	err := submitFormalReview(
		context.Background(), fc, "owner", "repo", 1,
		"request-changes", "abc1234567890abcdef1234567890abcdef1234ab", commentURL, nil, false, printer,
	)

	require.NoError(t, err)
	require.NotEmpty(t, fc.CreatedReviews)
	assert.Equal(t, "REQUEST_CHANGES", fc.CreatedReviews[0].Event)
	assert.Contains(t, fc.CreatedReviews[0].Body, commentURL)
}

// TS-GH73-016: Submit REQUEST_CHANGES without comment URL — fallback body
func TestQF_SubmitFormalReview_RequestChangesNoURL(t *testing.T) {
	fc := forge.NewFakeClient()
	fc.AuthenticatedUser = "bot-user"
	fc.PullRequestHeadSHA = "abc1234567890abcdef1234567890abcdef1234ab"
	printer := ui.New(io.Discard)

	err := submitFormalReview(
		context.Background(), fc, "owner", "repo", 1,
		"request-changes", "abc1234567890abcdef1234567890abcdef1234ab", "", nil, false, printer,
	)

	require.NoError(t, err)
	require.NotEmpty(t, fc.CreatedReviews)
	assert.Contains(t, fc.CreatedReviews[0].Body, "See the review comment above")
}

// TS-GH73-017: Submit with action='reject' maps to REQUEST_CHANGES
func TestQF_SubmitFormalReview_RejectMapsToRequestChanges(t *testing.T) {
	fc := forge.NewFakeClient()
	fc.AuthenticatedUser = "bot-user"
	fc.PullRequestHeadSHA = "abc1234567890abcdef1234567890abcdef1234ab"
	printer := ui.New(io.Discard)

	err := submitFormalReview(
		context.Background(), fc, "owner", "repo", 1,
		"reject", "abc1234567890abcdef1234567890abcdef1234ab", "", nil, false, printer,
	)

	require.NoError(t, err)
	require.NotEmpty(t, fc.CreatedReviews)
	assert.Equal(t, "REQUEST_CHANGES", fc.CreatedReviews[0].Event)
}

// TS-GH73-018: Submit COMMENT with no inline findings — no-op
func TestQF_SubmitFormalReview_CommentNoFindings(t *testing.T) {
	fc := forge.NewFakeClient()
	fc.AuthenticatedUser = "bot-user"
	fc.PullRequestHeadSHA = "abc1234567890abcdef1234567890abcdef1234ab"
	printer := ui.New(io.Discard)

	err := submitFormalReview(
		context.Background(), fc, "owner", "repo", 1,
		"comment", "abc1234567890abcdef1234567890abcdef1234ab", "", nil, false, printer,
	)

	require.NoError(t, err)
	assert.Empty(t, fc.CreatedReviews, "COMMENT review should be skipped without inline findings")
}

// TS-GH73-019: Submit COMMENT with inline-eligible findings
func TestQF_SubmitFormalReview_CommentWithFindings(t *testing.T) {
	fc := forge.NewFakeClient()
	fc.AuthenticatedUser = "bot-user"
	fc.PullRequestHeadSHA = "abc1234567890abcdef1234567890abcdef1234ab"
	fc.PRFileDiffs = map[string][]forge.PullRequestFileDiff{
		"owner/repo/1": {
			{Path: "main.go", Patch: "@@ -1,5 +1,10 @@\n some code"},
		},
	}
	printer := ui.New(io.Discard)

	findings := []ReviewFinding{
		{Severity: "high", Category: "bug", File: "main.go", Line: 5, Description: "issue found", Actionable: true},
	}

	err := submitFormalReview(
		context.Background(), fc, "owner", "repo", 1,
		"comment", "abc1234567890abcdef1234567890abcdef1234ab", "", findings, false, printer,
	)

	require.NoError(t, err)
	require.NotEmpty(t, fc.CreatedReviews)
	assert.Equal(t, "COMMENT", fc.CreatedReviews[0].Event)
	assert.NotEmpty(t, fc.CreatedReviews[0].Comments)
}

// TS-GH73-021: Unknown action string skips formal review
func TestQF_SubmitFormalReview_UnknownAction(t *testing.T) {
	fc := forge.NewFakeClient()
	fc.AuthenticatedUser = "bot-user"
	printer := ui.New(io.Discard)

	err := submitFormalReview(
		context.Background(), fc, "owner", "repo", 1,
		"unknown_action", "", "", nil, false, printer,
	)

	require.NoError(t, err)
	assert.Empty(t, fc.CreatedReviews)
}

// TS-GH73-022: Dry-run mode makes no API calls
func TestQF_SubmitFormalReview_DryRun(t *testing.T) {
	fc := forge.NewFakeClient()
	printer := ui.New(io.Discard)

	err := submitFormalReview(
		context.Background(), fc, "owner", "repo", 1,
		"approve", "", "", nil, true, printer,
	)

	require.NoError(t, err)
	assert.Empty(t, fc.CreatedReviews)
}

// TS-GH73-023: Commit SHA passed to review API
func TestQF_SubmitFormalReview_CommitSHA(t *testing.T) {
	fc := forge.NewFakeClient()
	fc.AuthenticatedUser = "bot-user"
	fc.PullRequestHeadSHA = "abc1234567890abcdef1234567890abcdef1234ab"
	printer := ui.New(io.Discard)

	commitSHA := "abc1234567890abcdef1234567890abcdef1234ab"

	err := submitFormalReview(
		context.Background(), fc, "owner", "repo", 1,
		"approve", commitSHA, "", nil, false, printer,
	)

	require.NoError(t, err)
	require.NotEmpty(t, fc.CreatedReviews)
	assert.Equal(t, commitSHA, fc.CreatedReviews[0].CommitSHA)
}

// TS-GH73-024: Empty commit SHA
func TestQF_SubmitFormalReview_EmptyCommitSHA(t *testing.T) {
	fc := forge.NewFakeClient()
	fc.AuthenticatedUser = "bot-user"
	fc.PullRequestHeadSHA = "abc1234567890abcdef1234567890abcdef1234ab"
	printer := ui.New(io.Discard)

	err := submitFormalReview(
		context.Background(), fc, "owner", "repo", 1,
		"approve", "", "", nil, false, printer,
	)

	require.NoError(t, err)
	require.NotEmpty(t, fc.CreatedReviews)
	assert.Empty(t, fc.CreatedReviews[0].CommitSHA)
}
