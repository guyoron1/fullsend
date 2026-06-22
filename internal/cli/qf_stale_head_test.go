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
// Section 3.2 — Post-Review — Stale Head Detection
// =============================================================================

// TS-GH73-008: PR HEAD matches reviewed SHA — stale=false
func TestQF_CheckStaleHead_Matching(t *testing.T) {
	fc := forge.NewFakeClient()
	fc.PullRequestHeadSHA = "abc1234567890abcdef1234567890abcdef123456"
	printer := ui.New(io.Discard)

	stale, currentSHA, err := checkStaleHead(
		context.Background(), fc, "owner", "repo", 1,
		"abc1234567890abcdef1234567890abcdef123456", false, printer,
	)

	require.NoError(t, err)
	assert.False(t, stale)
	assert.Equal(t, "abc1234567890abcdef1234567890abcdef123456", currentSHA)
}

// TS-GH73-009: PR HEAD differs from reviewed SHA — stale=true
func TestQF_CheckStaleHead_Differs(t *testing.T) {
	fc := forge.NewFakeClient()
	fc.PullRequestHeadSHA = "def4567890abcdef1234567890abcdef1234567890"
	printer := ui.New(io.Discard)

	stale, currentSHA, err := checkStaleHead(
		context.Background(), fc, "owner", "repo", 1,
		"abc1234567890abcdef1234567890abcdef123456", false, printer,
	)

	require.NoError(t, err)
	assert.True(t, stale)
	assert.Equal(t, "def4567890abcdef1234567890abcdef1234567890", currentSHA)
}

// TS-GH73-010: Dry-run mode — stale=false without API call
func TestQF_CheckStaleHead_DryRun(t *testing.T) {
	fc := forge.NewFakeClient()
	fc.PullRequestHeadSHA = "def4567890abcdef1234567890abcdef1234567890"
	printer := ui.New(io.Discard)

	stale, _, err := checkStaleHead(
		context.Background(), fc, "owner", "repo", 1,
		"abc1234567890abcdef1234567890abcdef123456", true, printer,
	)

	require.NoError(t, err)
	assert.False(t, stale, "dry-run should always return stale=false")
}

// TS-GH73-011: Case-insensitive SHA comparison
func TestQF_CheckStaleHead_CaseInsensitive(t *testing.T) {
	fc := forge.NewFakeClient()
	fc.PullRequestHeadSHA = "ABC1234567890ABCDEF1234567890ABCDEF123456"
	printer := ui.New(io.Discard)

	stale, _, err := checkStaleHead(
		context.Background(), fc, "owner", "repo", 1,
		"abc1234567890abcdef1234567890abcdef123456", false, printer,
	)

	require.NoError(t, err)
	assert.False(t, stale, "SHAs differing only in case should match")
}

// TS-GH73-013: staleHeadError returns StaleHeadExitCode (10)
func TestQF_StaleHeadError_ExitCode(t *testing.T) {
	reviewedSHA := "abc1234567890abcdef1234567890abcdef123456"
	currentSHA := "def4567890abcdef1234567890abcdef1234567890"

	err := &staleHeadError{reviewedSHA: reviewedSHA, currentSHA: currentSHA}

	assert.Equal(t, StaleHeadExitCode, err.ExitCode())
	assert.Equal(t, 10, err.ExitCode())
	assert.Contains(t, err.Error(), reviewedSHA)
	assert.Contains(t, err.Error(), currentSHA)
}
