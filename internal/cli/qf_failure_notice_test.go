package cli

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/forge"
	"github.com/fullsend-ai/fullsend/internal/sticky"
	"github.com/fullsend-ai/fullsend/internal/ui"
)

// =============================================================================
// Section 3.7 — Post-Review — Failure Notices
// =============================================================================

// TS-GH73-050: Failure with custom body — posted as-is
func TestQF_PostFailureNotice_CustomBody(t *testing.T) {
	fc := forge.NewFakeClient()
	printer := ui.New(io.Discard)
	cfg := sticky.Config{Marker: reviewMarker}

	parsed := ReviewResult{
		Action: "failure",
		Body:   "Custom failure message",
	}

	err := postFailureNotice(context.Background(), fc, "owner", "repo", 1, parsed, cfg, printer)
	require.NoError(t, err)

	// Verify the comment was posted (via sticky.Post which creates an issue comment)
	require.NotEmpty(t, fc.IssueComments["owner/repo/1"])
	postedComment := fc.IssueComments["owner/repo/1"][len(fc.IssueComments["owner/repo/1"])-1].Body
	assert.Contains(t, postedComment, "Custom failure message")
}

// TS-GH73-051: Failure without body, with reason — 'NOT reviewed' notice
func TestQF_PostFailureNotice_WithReason(t *testing.T) {
	fc := forge.NewFakeClient()
	printer := ui.New(io.Discard)
	cfg := sticky.Config{Marker: reviewMarker}

	parsed := ReviewResult{
		Action: "failure",
		Body:   "",
		Reason: "timeout",
	}

	err := postFailureNotice(context.Background(), fc, "owner", "repo", 1, parsed, cfg, printer)
	require.NoError(t, err)

	require.NotEmpty(t, fc.IssueComments["owner/repo/1"])
	postedComment := fc.IssueComments["owner/repo/1"][len(fc.IssueComments["owner/repo/1"])-1].Body
	assert.Contains(t, postedComment, "NOT reviewed")
	assert.Contains(t, postedComment, "timeout")
}

// TS-GH73-052: Failure without body, empty reason — defaults to 'unknown'
func TestQF_PostFailureNotice_EmptyReason(t *testing.T) {
	fc := forge.NewFakeClient()
	printer := ui.New(io.Discard)
	cfg := sticky.Config{Marker: reviewMarker}

	parsed := ReviewResult{
		Action: "failure",
		Body:   "",
		Reason: "",
	}

	err := postFailureNotice(context.Background(), fc, "owner", "repo", 1, parsed, cfg, printer)
	require.NoError(t, err)

	require.NotEmpty(t, fc.IssueComments["owner/repo/1"])
	postedComment := fc.IssueComments["owner/repo/1"][len(fc.IssueComments["owner/repo/1"])-1].Body
	assert.Contains(t, postedComment, "unknown")
}

// TS-GH73-053: Follow-up issue creation (disabled) — no-op
func TestQF_PostApprovedFollowUpIssues_Disabled(t *testing.T) {
	printer := ui.New(io.Discard)

	parsed := ReviewResult{Action: "approve", Body: "looks good"}

	err := postApprovedFollowUpIssues(context.Background(), "owner", "repo", 1, parsed, printer)
	require.NoError(t, err)
}
