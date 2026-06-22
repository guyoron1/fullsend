package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Section 3.1 — Post-Review — Review Result Parsing
// =============================================================================

// TS-GH73-001: Parse valid JSON with body and action
func TestQF_ParseReviewResult_ValidJSON(t *testing.T) {
	input := `{"body":"Review looks good","action":"approve"}`

	result, err := parseReviewResult(input)

	require.NoError(t, err)
	assert.Equal(t, "Review looks good", result.Body)
	assert.Equal(t, "approve", result.Action)
}

// TS-GH73-002: Parse plain text input (non-JSON)
func TestQF_ParseReviewResult_PlainText(t *testing.T) {
	input := "This is a review comment"

	result, err := parseReviewResult(input)

	require.NoError(t, err)
	assert.Equal(t, "This is a review comment", result.Body)
	assert.Equal(t, "comment", result.Action)
}

// TS-GH73-003: Parse JSON with missing action field defaults to "comment"
func TestQF_ParseReviewResult_MissingAction(t *testing.T) {
	input := `{"body":"Some review text"}`

	result, err := parseReviewResult(input)

	require.NoError(t, err)
	assert.Equal(t, "comment", result.Action)
	assert.Equal(t, "Some review text", result.Body)
}

// TS-GH73-004: Parse JSON with empty body and non-failure action returns error
// (Already covered in qf_postreview_test.go — included here for completeness)
func TestQF_ParseReviewResult_EmptyBodyNonFailure(t *testing.T) {
	input := `{"body":"","action":"approve"}`

	_, err := parseReviewResult(input)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty body")
}

// TS-GH73-005: Parse JSON with action='failure' and empty body succeeds
func TestQF_ParseReviewResult_FailureEmptyBody(t *testing.T) {
	input := `{"body":"","action":"failure"}`

	result, err := parseReviewResult(input)

	require.NoError(t, err)
	assert.Equal(t, "failure", result.Action)
	assert.Empty(t, result.Body)
}

// TS-GH73-006: Parse JSON with head_sha field
func TestQF_ParseReviewResult_HeadSHA(t *testing.T) {
	expectedSHA := "abc123def4567890abc123def4567890abc123de"
	input := `{"body":"review","action":"comment","head_sha":"` + expectedSHA + `"}`

	result, err := parseReviewResult(input)

	require.NoError(t, err)
	assert.Equal(t, expectedSHA, result.HeadSHA)
}

// TS-GH73-007: Parse JSON with findings array
func TestQF_ParseReviewResult_FindingsArray(t *testing.T) {
	input := `{
		"body": "review",
		"action": "comment",
		"findings": [
			{"file":"main.go","line":42,"severity":"high","category":"bug","description":"null pointer"},
			{"file":"util.go","line":10,"severity":"medium","category":"style","description":"naming"}
		]
	}`

	result, err := parseReviewResult(input)

	require.NoError(t, err)
	assert.Len(t, result.Findings, 2)
	assert.Equal(t, "main.go", result.Findings[0].File)
	assert.Equal(t, 42, result.Findings[0].Line)
	assert.Equal(t, "high", result.Findings[0].Severity)
	assert.Equal(t, "util.go", result.Findings[1].File)
	assert.Equal(t, 10, result.Findings[1].Line)
}
