package statuscomment

// QualityFlow generated tests for GH-72
// Suite: TS-GH72-003 — StatusComment Notifier ClientFactory pattern
// STD: outputs/std/GH-72/GH-72_test_description.yaml

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

func qfFixedTime() time.Time {
	return time.Date(2026, 6, 3, 14, 34, 0, 0, time.UTC)
}

func newQFNotifier(fc *forge.FakeClient, cfg config.StatusNotificationConfig) *Notifier {
	fc.AuthenticatedUser = "fullsend-bot[bot]"
	n := New(fc, cfg, "org", "repo", 7, "https://ci/run/42", "a1b2c3d4e5f6789", "run-42")
	n.now = qfFixedTime
	return n
}

// TC-GH72-009: ClientFactory called before PostStart API operations
func TestQFClientFactory_CalledBeforePostStart(t *testing.T) {
	fc1 := forge.NewFakeClient()
	fc2 := forge.NewFakeClient()
	fc2.AuthenticatedUser = "mint-bot[bot]"
	cfg := config.StatusNotificationConfig{}

	n := New(fc1, cfg, "org", "repo", 7, "https://ci/run/42", "a1b2c3d", "run-42")
	n.now = qfFixedTime

	factoryCalled := false
	n.SetClientFactory(func(ctx context.Context) (forge.Client, error) {
		factoryCalled = true
		return fc2, nil
	})

	err := n.PostStart(context.Background(), "Working")
	require.NoError(t, err)
	assert.True(t, factoryCalled, "factory should be called before PostStart API calls")
	assert.Len(t, fc2.IssueComments["org/repo/7"], 1, "comment should be on factory-returned client")
	assert.Empty(t, fc1.IssueComments, "original client should not be used")
}

// TC-GH72-010: ClientFactory called before PostCompletion API operations
func TestQFClientFactory_CalledBeforePostCompletion(t *testing.T) {
	fc := forge.NewFakeClient()
	fc.AuthenticatedUser = "bot[bot]"
	cfg := config.StatusNotificationConfig{
		Comment: config.CommentNotificationConfig{Start: "enabled", Completion: "enabled"},
	}

	n := newQFNotifier(fc, cfg)
	err := n.PostStart(context.Background(), "Working")
	require.NoError(t, err)

	fc2 := forge.NewFakeClient()
	fc2.AuthenticatedUser = "bot[bot]"
	fc2.IssueComments = map[string][]forge.IssueComment{
		"org/repo/7": {fc.IssueComments["org/repo/7"][0]},
	}

	completionFactoryCalled := false
	n.SetClientFactory(func(ctx context.Context) (forge.Client, error) {
		completionFactoryCalled = true
		return fc2, nil
	})

	n.now = func() time.Time { return qfFixedTime().Add(5 * time.Minute) }
	err = n.PostCompletion(context.Background(), "Working", "success")
	require.NoError(t, err)
	assert.True(t, completionFactoryCalled, "factory should be called before PostCompletion API calls")
}

// TC-GH72-011: ClientFactory error propagated on PostStart
func TestQFClientFactory_ErrorPropagated(t *testing.T) {
	fc := forge.NewFakeClient()
	cfg := config.StatusNotificationConfig{}
	n := New(fc, cfg, "org", "repo", 7, "", "", "run-42")
	n.now = qfFixedTime

	n.SetClientFactory(func(ctx context.Context) (forge.Client, error) {
		return nil, fmt.Errorf("mint service unavailable")
	})

	err := n.PostStart(context.Background(), "Working")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mint service unavailable",
		"factory error should be propagated, not swallowed")
}

// TC-GH72-012: Static client used when no factory is set
func TestQFClientFactory_NilUsesStaticClient(t *testing.T) {
	fc := forge.NewFakeClient()
	cfg := config.StatusNotificationConfig{}
	n := newQFNotifier(fc, cfg)

	err := n.PostStart(context.Background(), "Working")
	require.NoError(t, err)
	assert.Len(t, fc.IssueComments["org/repo/7"], 1,
		"static client should be used when no factory set")
}

// TC-GH72-013: Completion-disabled path mints then deletes start comment
func TestQFClientFactory_CompletionDisabled_DeletePath(t *testing.T) {
	fc := forge.NewFakeClient()
	cfg := config.StatusNotificationConfig{
		Comment: config.CommentNotificationConfig{Start: "enabled", Completion: "disabled"},
	}
	n := newQFNotifier(fc, cfg)

	err := n.PostStart(context.Background(), "Working")
	require.NoError(t, err)
	require.Equal(t, 1, n.startCommentID)

	fc2 := forge.NewFakeClient()
	fc2.AuthenticatedUser = "fullsend-bot[bot]"
	fc2.IssueComments = map[string][]forge.IssueComment{
		"org/repo/7": {fc.IssueComments["org/repo/7"][0]},
	}

	factoryCalled := false
	n.SetClientFactory(func(ctx context.Context) (forge.Client, error) {
		factoryCalled = true
		return fc2, nil
	})

	n.now = func() time.Time { return qfFixedTime().Add(time.Minute) }
	err = n.PostCompletion(context.Background(), "Working", "success")
	require.NoError(t, err)
	assert.True(t, factoryCalled, "factory should be called even when completion disabled (for delete)")
	require.Len(t, fc2.DeletedComments, 1)
	assert.Equal(t, 1, fc2.DeletedComments[0])
}

// TC-GH72-014: HasClientFactory reports factory presence
func TestQFHasClientFactory(t *testing.T) {
	fc := forge.NewFakeClient()
	cfg := config.StatusNotificationConfig{}
	n := newQFNotifier(fc, cfg)

	assert.False(t, n.HasClientFactory(), "should be false when no factory set")

	n.SetClientFactory(func(ctx context.Context) (forge.Client, error) {
		return fc, nil
	})
	assert.True(t, n.HasClientFactory(), "should be true after SetClientFactory")
}

// TC-GH72-015: ClientFactory error on PostCompletion propagated
func TestQFClientFactory_ErrorOnPostCompletion(t *testing.T) {
	fc := forge.NewFakeClient()
	cfg := config.StatusNotificationConfig{
		Comment: config.CommentNotificationConfig{Start: "enabled", Completion: "enabled"},
	}
	n := newQFNotifier(fc, cfg)

	err := n.PostStart(context.Background(), "Working")
	require.NoError(t, err)

	n.SetClientFactory(func(ctx context.Context) (forge.Client, error) {
		return nil, fmt.Errorf("token expired")
	})

	n.now = func() time.Time { return qfFixedTime().Add(5 * time.Minute) }
	err = n.PostCompletion(context.Background(), "Working", "success")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "token expired")
}

// TC-GH72-016: Both disabled means no factory call
func TestQFClientFactory_BothDisabled_NoMint(t *testing.T) {
	fc := forge.NewFakeClient()
	cfg := config.StatusNotificationConfig{
		Comment: config.CommentNotificationConfig{Start: "disabled", Completion: "disabled"},
	}
	n := newQFNotifier(fc, cfg)

	factoryCalled := false
	n.SetClientFactory(func(ctx context.Context) (forge.Client, error) {
		factoryCalled = true
		return nil, fmt.Errorf("should not be called")
	})

	err := n.PostCompletion(context.Background(), "Working", "success")
	require.NoError(t, err, "should not error when no API call is needed")
	assert.False(t, factoryCalled, "factory should not be called when both disabled and no start comment")
}

// TC-GH72-017: Completion-disabled mint error is fail-open with warning
func TestQFClientFactory_CompletionDisabled_MintError(t *testing.T) {
	fc := forge.NewFakeClient()
	cfg := config.StatusNotificationConfig{
		Comment: config.CommentNotificationConfig{Start: "enabled", Completion: "disabled"},
	}
	n := newQFNotifier(fc, cfg)

	err := n.PostStart(context.Background(), "Working")
	require.NoError(t, err)
	require.NotZero(t, n.startCommentID)

	var warnings []string
	n.SetWarnFunc(func(format string, args ...any) {
		warnings = append(warnings, fmt.Sprintf(format, args...))
	})
	n.SetClientFactory(func(ctx context.Context) (forge.Client, error) {
		return nil, fmt.Errorf("mint service down")
	})

	err = n.PostCompletion(context.Background(), "Working", "success")
	require.NoError(t, err, "should not return error — fail-open on cleanup")
	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "mint service down")
}

// TC-GH72-018: Completion-disabled delete error is fail-open with warning
func TestQFClientFactory_CompletionDisabled_DeleteError(t *testing.T) {
	fc := forge.NewFakeClient()
	cfg := config.StatusNotificationConfig{
		Comment: config.CommentNotificationConfig{Start: "enabled", Completion: "disabled"},
	}
	n := newQFNotifier(fc, cfg)

	err := n.PostStart(context.Background(), "Working")
	require.NoError(t, err)
	require.NotZero(t, n.startCommentID)

	fc2 := forge.NewFakeClient()
	fc2.Errors["DeleteIssueComment"] = fmt.Errorf("forbidden")

	var warnings []string
	n.SetWarnFunc(func(format string, args ...any) {
		warnings = append(warnings, fmt.Sprintf(format, args...))
	})
	n.SetClientFactory(func(ctx context.Context) (forge.Client, error) {
		return fc2, nil
	})

	err = n.PostCompletion(context.Background(), "Working", "success")
	require.NoError(t, err, "should not return error — fail-open on cleanup")
	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "forbidden")
}
