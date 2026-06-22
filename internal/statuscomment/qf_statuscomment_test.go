package statuscomment

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/config"
	"github.com/fullsend-ai/fullsend/internal/forge"
)

// QualityFlow generated tests for GH-76: bound enrollment wait with timeout and backoff
// Covers: status comment lifecycle, orphan reconciliation, client factory token minting,
// comment placement heuristics, and edge cases.

// --- PostStart tests ---

func TestQF_PostStart_CorrectMarkerAndTimestamp(t *testing.T) {
	fc := forge.NewFakeClient()
	cfg := config.StatusNotificationConfig{
		Comment: config.CommentNotificationConfig{Start: "enabled"},
	}
	n := newTestNotifier(fc, cfg)

	err := n.PostStart(context.Background(), "Reviewing this PR")
	require.NoError(t, err)

	comments := fc.IssueComments["org/repo/7"]
	require.Len(t, comments, 1)
	assert.Contains(t, comments[0].Body, "<!-- fullsend:agent-status:run-42 -->")
	assert.Contains(t, comments[0].Body, "Reviewing this PR")
	assert.Contains(t, comments[0].Body, "Started 2:34 PM UTC")
	assert.Contains(t, comments[0].Body, "Commit: `a1b2c3d`")
	assert.Contains(t, comments[0].Body, "[View workflow run")
}

func TestQF_PostStart_ClientFactoryMintsFreshToken(t *testing.T) {
	fc1 := forge.NewFakeClient()
	fc2 := forge.NewFakeClient()
	fc2.AuthenticatedUser = "mint-bot[bot]"
	cfg := config.StatusNotificationConfig{}

	n := New(fc1, cfg, "org", "repo", 7, "https://ci/run/42", "a1b2c3d", "run-42")
	n.now = fixedTime

	factoryCalled := false
	n.SetClientFactory(func(ctx context.Context) (forge.Client, error) {
		factoryCalled = true
		return fc2, nil
	})

	err := n.PostStart(context.Background(), "Working")
	require.NoError(t, err)
	assert.True(t, factoryCalled, "factory should mint token before API call")
	assert.Len(t, fc2.IssueComments["org/repo/7"], 1, "comment posted via minted client")
	assert.Empty(t, fc1.IssueComments, "original client should not be used")
}

// --- PostCompletion placement tests ---

func TestQF_PostCompletion_UpdatesStartWhenLastOnTimeline(t *testing.T) {
	fc := forge.NewFakeClient()
	cfg := config.StatusNotificationConfig{
		Comment: config.CommentNotificationConfig{Start: "enabled", Completion: "enabled"},
	}
	n := newTestNotifier(fc, cfg)

	err := n.PostStart(context.Background(), "Reviewing this PR")
	require.NoError(t, err)

	n.now = func() time.Time { return fixedTime().Add(7 * time.Minute) }
	err = n.PostCompletion(context.Background(), "Reviewing this PR", "success")
	require.NoError(t, err)

	require.Len(t, fc.UpdatedComments, 1)
	assert.Contains(t, fc.UpdatedComments[0].Body, "Finished Reviewing this PR")
	assert.Contains(t, fc.UpdatedComments[0].Body, "Started 2:34 PM UTC")
	assert.Contains(t, fc.UpdatedComments[0].Body, "Completed 2:41 PM UTC")
}

func TestQF_PostCompletion_NewCommentWhenHumanIntervenes(t *testing.T) {
	fc := forge.NewFakeClient()
	cfg := config.StatusNotificationConfig{
		Comment: config.CommentNotificationConfig{Start: "enabled", Completion: "enabled"},
	}
	n := newTestNotifier(fc, cfg)

	err := n.PostStart(context.Background(), "Triaging issue")
	require.NoError(t, err)

	// Simulate human comment between start and completion.
	fc.IssueComments["org/repo/7"] = append(fc.IssueComments["org/repo/7"], forge.IssueComment{
		ID:     9999,
		Body:   "A human comment",
		Author: "some-human",
	})

	n.now = func() time.Time { return fixedTime().Add(5 * time.Minute) }
	err = n.PostCompletion(context.Background(), "Triaging issue", "success")
	require.NoError(t, err)

	assert.Empty(t, fc.UpdatedComments, "should not update when human activity intervenes")
	comments := fc.IssueComments["org/repo/7"]
	require.Len(t, comments, 3, "new comment should be posted after human activity")
	assert.Contains(t, comments[2].Body, "Finished Triaging issue")
}

func TestQF_PostCompletion_DeletesStartWhenCompletionDisabled(t *testing.T) {
	fc := forge.NewFakeClient()
	cfg := config.StatusNotificationConfig{
		Comment: config.CommentNotificationConfig{Start: "enabled", Completion: "disabled"},
	}
	n := newTestNotifier(fc, cfg)

	err := n.PostStart(context.Background(), "Working")
	require.NoError(t, err)
	require.NotZero(t, n.startCommentID)

	n.now = func() time.Time { return fixedTime().Add(time.Minute) }
	err = n.PostCompletion(context.Background(), "Working", "success")
	require.NoError(t, err)

	assert.Empty(t, fc.UpdatedComments, "should not update start comment")
	require.Len(t, fc.DeletedComments, 1, "should delete orphaned start comment")
}

func TestQF_PostCompletion_NoErrorWhenStartCommentNotFound(t *testing.T) {
	fc := forge.NewFakeClient()
	cfg := config.StatusNotificationConfig{
		Comment: config.CommentNotificationConfig{Start: "disabled", Completion: "enabled"},
	}
	n := newTestNotifier(fc, cfg)

	// Start disabled, so no start comment is created.
	err := n.PostStart(context.Background(), "Working")
	require.NoError(t, err)
	assert.Equal(t, 0, n.startCommentID)

	n.now = func() time.Time { return fixedTime().Add(time.Minute) }
	err = n.PostCompletion(context.Background(), "Working", "success")
	require.NoError(t, err)

	comments := fc.IssueComments["org/repo/7"]
	require.Len(t, comments, 1, "should post new completion comment")
	assert.Contains(t, comments[0].Body, "Finished Working")
}

// --- ReconcileOrphaned tests ---

func TestQF_ReconcileOrphaned_UpdatesOrphanedStartComment(t *testing.T) {
	fc := forge.NewFakeClient()
	fc.IssueComments = map[string][]forge.IssueComment{}
	setNow(t, time.Date(2026, 6, 3, 7, 12, 0, 0, time.UTC))

	fc.IssueComments["org/repo/7"] = []forge.IssueComment{
		{
			ID:     42,
			Body:   "<!-- fullsend:agent-status:run-99 -->\n\U0001f916 Code \u00b7 Started 6:43 AM UTC\nCommit: `abc1234` \u00b7 [View workflow run \u2192](https://ci/run/99)",
			Author: "fullsend-bot[bot]",
		},
	}

	err := ReconcileOrphaned(context.Background(), fc, "org", "repo", 7, "run-99", "https://ci/run/99", "abc1234def", ReasonTerminated)
	require.NoError(t, err)

	require.Len(t, fc.UpdatedComments, 1)
	body := fc.UpdatedComments[0].Body
	assert.Contains(t, body, "Code")
	assert.Contains(t, body, "Terminated")
	assert.Contains(t, body, "Started 6:43 AM UTC")
	assert.Contains(t, body, "Ended 7:12 AM UTC")
	assert.Contains(t, body, terminalTag)
}

func TestQF_ReconcileOrphaned_SkipsAlreadyTerminal(t *testing.T) {
	fc := forge.NewFakeClient()
	fc.IssueComments = map[string][]forge.IssueComment{}

	fc.IssueComments["org/repo/7"] = []forge.IssueComment{
		{
			ID:     42,
			Body:   "<!-- fullsend:agent-status:run-99 -->\n<!-- fullsend:status:terminal -->\nCompleted",
			Author: "fullsend-bot[bot]",
		},
	}

	err := ReconcileOrphaned(context.Background(), fc, "org", "repo", 7, "run-99", "", "", ReasonTerminated)
	require.NoError(t, err)

	assert.Empty(t, fc.UpdatedComments, "should not update already-terminal comment")
}

func TestQF_ReconcileOrphaned_NoMatchingCommentIsOK(t *testing.T) {
	fc := forge.NewFakeClient()

	err := ReconcileOrphaned(context.Background(), fc, "org", "repo", 7, "run-99", "", "", ReasonTerminated)
	require.NoError(t, err)
	assert.Empty(t, fc.UpdatedComments)
}

func TestQF_ReconcileOrphaned_InvalidRunIDReturnsError(t *testing.T) {
	fc := forge.NewFakeClient()
	err := ReconcileOrphaned(context.Background(), fc, "org", "repo", 7, "-->bad", "", "", ReasonTerminated)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid run ID")
}

func TestQF_ReconcileOrphaned_CancelledReasonProducesCancelledLabel(t *testing.T) {
	fc := forge.NewFakeClient()
	fc.IssueComments = map[string][]forge.IssueComment{}
	setNow(t, time.Date(2026, 6, 3, 14, 47, 0, 0, time.UTC))

	fc.IssueComments["org/repo/7"] = []forge.IssueComment{
		{
			ID:     42,
			Body:   "<!-- fullsend:agent-status:run-99 -->\n\U0001f916 Reviewing this PR \u00b7 Started 2:34 PM UTC",
			Author: "fullsend-bot[bot]",
		},
	}

	err := ReconcileOrphaned(context.Background(), fc, "org", "repo", 7, "run-99", "https://ci/run/99", "abc1234def", ReasonCancelled)
	require.NoError(t, err)

	require.Len(t, fc.UpdatedComments, 1)
	body := fc.UpdatedComments[0].Body
	assert.Contains(t, body, "Cancelled")
	assert.Contains(t, body, terminalTag)
}

func TestQF_ReconcileOrphaned_TerminatedReasonProducesTerminatedLabel(t *testing.T) {
	fc := forge.NewFakeClient()
	fc.IssueComments = map[string][]forge.IssueComment{}
	setNow(t, time.Date(2026, 6, 3, 14, 47, 0, 0, time.UTC))

	fc.IssueComments["org/repo/7"] = []forge.IssueComment{
		{
			ID:     42,
			Body:   "<!-- fullsend:agent-status:run-99 -->\n\U0001f916 Code \u00b7 Started 6:43 AM UTC",
			Author: "fullsend-bot[bot]",
		},
	}

	err := ReconcileOrphaned(context.Background(), fc, "org", "repo", 7, "run-99", "https://ci/run/99", "abc1234def", ReasonTerminated)
	require.NoError(t, err)

	require.Len(t, fc.UpdatedComments, 1)
	body := fc.UpdatedComments[0].Body
	assert.Contains(t, body, "Terminated")
	assert.Contains(t, body, terminalTag)
}

// --- ClientFactory lifecycle tests ---

func TestQF_ClientFactory_ErrorOnPostCompletionPropagated(t *testing.T) {
	fc := forge.NewFakeClient()
	cfg := config.StatusNotificationConfig{
		Comment: config.CommentNotificationConfig{Start: "enabled", Completion: "enabled"},
	}
	n := newTestNotifier(fc, cfg)

	err := n.PostStart(context.Background(), "Working")
	require.NoError(t, err)

	n.SetClientFactory(func(ctx context.Context) (forge.Client, error) {
		return nil, fmt.Errorf("token expired")
	})

	n.now = func() time.Time { return fixedTime().Add(5 * time.Minute) }
	err = n.PostCompletion(context.Background(), "Working", "success")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "token expired")
}

func TestQF_ClientFactory_NilUsesStaticClient(t *testing.T) {
	fc := forge.NewFakeClient()
	cfg := config.StatusNotificationConfig{}
	n := newTestNotifier(fc, cfg)

	// No factory set — should use static client.
	err := n.PostStart(context.Background(), "Working")
	require.NoError(t, err)
	assert.Len(t, fc.IssueComments["org/repo/7"], 1)
}

func TestQF_HasClientFactory_ReflectsState(t *testing.T) {
	fc := forge.NewFakeClient()
	cfg := config.StatusNotificationConfig{}
	n := newTestNotifier(fc, cfg)

	assert.False(t, n.HasClientFactory())

	n.SetClientFactory(func(ctx context.Context) (forge.Client, error) {
		return fc, nil
	})
	assert.True(t, n.HasClientFactory())
}

// --- Utility function tests ---

func TestQF_IsSafeURL_AcceptsValidHTTPS(t *testing.T) {
	assert.True(t, isSafeURL("https://github.com/org/repo/actions/runs/123"))
}

func TestQF_IsSafeURL_RejectsHTTP(t *testing.T) {
	assert.False(t, isSafeURL("http://example.com/run"))
}

func TestQF_IsSafeURL_RejectsJavascript(t *testing.T) {
	assert.False(t, isSafeURL("javascript:alert(1)"))
}

func TestQF_IsSafeURL_RejectsParenInURL(t *testing.T) {
	assert.False(t, isSafeURL("https://example.com/run)"))
}

func TestQF_IsSafeURL_RejectsNewlineInURL(t *testing.T) {
	assert.False(t, isSafeURL("https://example.com/run\ninjected"))
}

func TestQF_ShortSHA_TruncatesLongSHA(t *testing.T) {
	assert.Equal(t, "a1b2c3d", shortSHA("a1b2c3d4e5f6789"))
}

func TestQF_ShortSHA_PreservesShortSHA(t *testing.T) {
	assert.Equal(t, "abc", shortSHA("abc"))
}

func TestQF_ShortSHA_RejectsNonHex(t *testing.T) {
	assert.Equal(t, "", shortSHA("not-a-sha"))
}

func TestQF_ShortSHA_RejectsEmpty(t *testing.T) {
	assert.Equal(t, "", shortSHA(""))
}

func TestQF_BuildMarker_ValidRunID(t *testing.T) {
	m, err := buildMarker("run-42")
	require.NoError(t, err)
	assert.Equal(t, "<!-- fullsend:agent-status:run-42 -->", m)
}

func TestQF_BuildMarker_InvalidRunID(t *testing.T) {
	_, err := buildMarker("-->injected")
	assert.Error(t, err)
}

func TestQF_BuildMarker_EmptyRunID(t *testing.T) {
	_, err := buildMarker("")
	assert.Error(t, err)
}

func TestQF_ReasonLabel_Terminated(t *testing.T) {
	statusLabel, heading := reasonLabel(ReasonTerminated, "Code")
	assert.Contains(t, statusLabel, "Terminated")
	assert.Equal(t, "Code", heading)
}

func TestQF_ReasonLabel_Cancelled(t *testing.T) {
	statusLabel, heading := reasonLabel(ReasonCancelled, "Review")
	assert.Contains(t, statusLabel, "Cancelled")
	assert.Equal(t, "Review", heading)
}

func TestQF_ReasonLabel_TerminatedNoDescription(t *testing.T) {
	statusLabel, heading := reasonLabel(ReasonTerminated, "")
	assert.Contains(t, statusLabel, "Terminated")
	assert.Equal(t, "Agent run interrupted", heading)
}

func TestQF_ReasonLabel_CancelledNoDescription(t *testing.T) {
	statusLabel, heading := reasonLabel(ReasonCancelled, "")
	assert.Contains(t, statusLabel, "Cancelled")
	assert.Equal(t, "Agent run cancelled", heading)
}
