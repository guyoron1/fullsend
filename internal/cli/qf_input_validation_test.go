package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Section 3.8 — Input Validation
// =============================================================================

// TS-GH73-054: Valid 40-char hex SHA passes
func TestQF_HexSHARe_Valid40Char(t *testing.T) {
	sha := "abc123def4567890abc123def4567890abcdef12"
	assert.True(t, hexSHARe.MatchString(sha), "valid 40-char hex SHA should pass")
}

// TS-GH73-055: Valid 64-char hex SHA (SHA-256) passes
func TestQF_HexSHARe_Valid64Char(t *testing.T) {
	sha := "abc123def4567890abc123def4567890abcdef12abc123def4567890abcdef12"
	assert.True(t, hexSHARe.MatchString(sha), "valid 64-char hex SHA should pass")
}

// TS-GH73-056: Short/malformed SHA fails
func TestQF_HexSHARe_ShortSHA(t *testing.T) {
	sha := "abc12"
	assert.False(t, hexSHARe.MatchString(sha), "short SHA should fail")
}

// TS-GH73-057: SHA with injection characters fails
func TestQF_HexSHARe_InjectionChars(t *testing.T) {
	sha := "abc123; rm -rf /"
	assert.False(t, hexSHARe.MatchString(sha), "SHA with injection chars should fail")
}

// TS-GH73-058: Empty SHA — the regex won't match, but empty is handled separately
func TestQF_HexSHARe_EmptySHA(t *testing.T) {
	// Empty SHA is valid as a sentinel (no SHA provided) —
	// the CLI checks sha != "" before applying regex validation
	sha := ""
	// Empty SHA bypasses regex entirely in the command handler
	assert.False(t, hexSHARe.MatchString(sha), "empty string should not match regex")
	// But the flow allows empty SHA (no commit pinning)
}

// TS-GH73-059: Reason with valid chars passes
func TestQF_ReasonRe_ValidChars(t *testing.T) {
	reason := "user-cancelled_v2"
	assert.True(t, reasonRe.MatchString(reason), "valid reason should pass")
}

// TS-GH73-060: Reason with injection fails
func TestQF_ReasonRe_Injection(t *testing.T) {
	reason := "reason <script>alert(1)</script>"
	assert.False(t, reasonRe.MatchString(reason), "reason with injection should fail")
}

// TS-GH73-061: Invalid repo format returns error (post-review command)
func TestQF_PostReviewCmd_InvalidRepo(t *testing.T) {
	cmd := newPostReviewCmd()
	cmd.SetArgs([]string{
		"--repo", "invalid-repo-format",
		"--pr", "1",
		"--token", "test-token",
		"--result", "-",
	})
	// Provide empty stdin to avoid blocking
	cmd.SetIn(nil)
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "owner/repo")
}

// TS-GH73-062: Negative PR number returns error
func TestQF_PostReviewCmd_NegativePR(t *testing.T) {
	cmd := newPostReviewCmd()
	cmd.SetArgs([]string{
		"--repo", "owner/repo",
		"--pr", "-1",
		"--token", "test-token",
		"--result", "-",
	})
	cmd.SetIn(nil)
	err := cmd.Execute()
	require.Error(t, err)
}

// TS-GH73-054 supplemental: reviewActionToEvent mapping tests
func TestQF_ReviewActionToEvent_Mappings(t *testing.T) {
	tests := []struct {
		action   string
		expected string
		ok       bool
	}{
		{"approve", "APPROVE", true},
		{"request-changes", "REQUEST_CHANGES", true},
		{"comment", "COMMENT", true},
		{"reject", "REQUEST_CHANGES", true},
		{"unknown", "", false},
		{"APPROVE", "APPROVE", true},
	}

	for _, tt := range tests {
		t.Run(tt.action, func(t *testing.T) {
			event, ok := reviewActionToEvent(tt.action)
			assert.Equal(t, tt.expected, event)
			assert.Equal(t, tt.ok, ok)
		})
	}
}
