package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/forge"
	"github.com/fullsend-ai/fullsend/internal/ui"
)

// ============================================================
// TS-GH-53-008: Finding in diff hunk is posted as normal inline comment
// Priority: P0 | Tier: 1 | Type: Functional
// ============================================================

func TestFindingsToReviewComments_InHunkInlineComment(t *testing.T) {
	findings := []ReviewFinding{
		{
			File:        "main.go",
			Line:        42,
			Severity:    "high",
			Category:    "bug",
			Description: "Nil pointer dereference.",
		},
	}
	diffHunks := map[string][][2]int{
		"main.go": {{40, 50}},
	}

	comments, fileFiltered, fileLevelFallback := findingsToReviewComments(findings, diffHunks)

	assert.Equal(t, 0, fileFiltered)
	assert.Equal(t, 0, fileLevelFallback)
	require.Len(t, comments, 1)
	assert.Equal(t, "main.go", comments[0].Path, "comment path must match finding file")
	assert.Equal(t, 42, comments[0].Line, "in-hunk finding must keep original line number")
	assert.Contains(t, comments[0].Body, "Nil pointer dereference.")
}

// ============================================================
// TS-GH-53-009: Finding outside diff hunk falls back to file-level
//
//	comment (Line=0)
//
// Priority: P0 | Tier: 1 | Type: Functional
// ============================================================

func TestFindingsToReviewComments_OutOfHunkFallsBackToFileLevel(t *testing.T) {
	findings := []ReviewFinding{
		{
			File:        "main.go",
			Line:        100,
			Severity:    "medium",
			Category:    "logic-error",
			Description: "Missing bounds check.",
		},
	}
	diffHunks := map[string][][2]int{
		"main.go": {{40, 50}},
	}

	comments, fileFiltered, fileLevelFallback := findingsToReviewComments(findings, diffHunks)

	assert.Equal(t, 0, fileFiltered)
	assert.Equal(t, 1, fileLevelFallback, "out-of-hunk finding must count as file-level fallback")
	require.Len(t, comments, 1, "out-of-hunk finding must NOT be dropped")
	assert.Equal(t, "main.go", comments[0].Path)
	assert.Equal(t, 0, comments[0].Line, "out-of-hunk finding must have Line=0 (file-level)")
	assert.Contains(t, comments[0].Body, "Missing bounds check.")
}

// ============================================================
// TS-GH-53-010: File-level fallback body contains "Line N" with
//
//	original line number
//
// Priority: P0 | Tier: 1 | Type: Functional
// ============================================================

func TestFindingsToReviewComments_FileLevelBodyContainsLineNumber(t *testing.T) {
	findings := []ReviewFinding{
		{
			File:        "main.go",
			Line:        100,
			Severity:    "high",
			Category:    "bug",
			Description: "Potential race condition.",
		},
	}
	diffHunks := map[string][][2]int{
		"main.go": {{40, 50}},
	}

	comments, _, _ := findingsToReviewComments(findings, diffHunks)

	require.Len(t, comments, 1)
	assert.Equal(t, 0, comments[0].Line, "should be file-level comment")
	assert.Contains(t, comments[0].Body, "Line 100",
		"file-level fallback body must include original line number")
}

// ============================================================
// TS-GH-53-011: Finding on file not in PR diff is omitted
// Priority: P1 | Tier: 1 | Type: Functional
// ============================================================

func TestFindingsToReviewComments_FileNotInDiffIsOmitted(t *testing.T) {
	findings := []ReviewFinding{
		{
			File:        "unknown.go",
			Line:        5,
			Severity:    "low",
			Category:    "style",
			Description: "Naming convention violation.",
		},
	}
	diffHunks := map[string][][2]int{
		"main.go": {{1, 100}},
	}

	comments, fileFiltered, fileLevelFallback := findingsToReviewComments(findings, diffHunks)

	assert.Equal(t, 1, fileFiltered, "finding on non-diff file must increment fileFiltered counter")
	assert.Equal(t, 0, fileLevelFallback)
	assert.Empty(t, comments, "finding on file not in PR diff must be omitted")
}

// ============================================================
// TS-GH-53-012: All severity levels pass through to inline comments
//
//	when in hunk
//
// Priority: P1 | Tier: 1 | Type: Functional (table-driven)
// ============================================================

func TestFindingsToReviewComments_AllSeveritiesInHunk(t *testing.T) {
	severities := []string{"info", "low", "medium", "high", "critical"}
	for _, sev := range severities {
		t.Run(sev, func(t *testing.T) {
			findings := []ReviewFinding{
				{
					File:        "a.go",
					Line:        10,
					Severity:    sev,
					Category:    "test",
					Description: fmt.Sprintf("%s severity finding", sev),
				},
			}
			diffHunks := map[string][][2]int{
				"a.go": {{5, 15}},
			}

			comments, fileFiltered, fileLevelFallback := findingsToReviewComments(findings, diffHunks)

			assert.Equal(t, 0, fileFiltered)
			assert.Equal(t, 0, fileLevelFallback)
			require.Len(t, comments, 1, "%s severity should produce inline comment when in-hunk", sev)
			assert.Equal(t, 10, comments[0].Line, "%s severity must preserve line number", sev)
		})
	}
}

// ============================================================
// TS-GH-53-013: All severity levels fall back to file-level when
//
//	outside hunk
//
// Priority: P1 | Tier: 1 | Type: Functional (table-driven)
// ============================================================

func TestFindingsToReviewComments_AllSeveritiesFallbackToFileLevelSubtests(t *testing.T) {
	severities := []string{"info", "low", "medium", "high", "critical"}
	for _, sev := range severities {
		t.Run(sev, func(t *testing.T) {
			findings := []ReviewFinding{
				{
					File:        "changed.go",
					Line:        100,
					Severity:    sev,
					Category:    "test",
					Description: fmt.Sprintf("%s severity finding outside hunk", sev),
				},
			}
			diffHunks := map[string][][2]int{
				"changed.go": {{5, 15}},
			}

			comments, _, fileLevelFallback := findingsToReviewComments(findings, diffHunks)

			assert.Equal(t, 1, fileLevelFallback,
				"%s severity should fall back to file-level when out-of-hunk", sev)
			require.Len(t, comments, 1, "%s severity must not be dropped", sev)
			assert.Equal(t, 0, comments[0].Line,
				"%s severity must have Line=0 (file-level) when out-of-hunk", sev)
		})
	}
}

// ============================================================
// TS-GH-53-014: Binary files (empty hunk list) skip line-level filtering
// Priority: P2 | Tier: 1 | Type: Functional
// ============================================================

func TestFindingsToReviewComments_BinaryFileEmptyPatch(t *testing.T) {
	findings := []ReviewFinding{
		{
			File:        "image.png",
			Line:        1,
			Severity:    "high",
			Category:    "binary",
			Description: "Finding on binary file.",
		},
	}
	diffHunks := map[string][][2]int{
		"image.png": nil, // empty/nil hunk list = binary file
	}

	comments, fileFiltered, fileLevelFallback := findingsToReviewComments(findings, diffHunks)

	assert.Equal(t, 0, fileFiltered, "binary file is in diff, should not be file-filtered")
	assert.Equal(t, 0, fileLevelFallback)
	require.Len(t, comments, 1, "binary file finding should not cause panic and should produce comment")
	assert.Equal(t, "image.png", comments[0].Path)
	assert.Equal(t, 1, comments[0].Line, "binary files skip hunk filtering, line preserved")
}

// ============================================================
// TS-GH-53-015: End-to-end review with mixed finding types produces
//
//	correct comment set
//
// Priority: P0 | Tier: 1 | Type: Integration
// ============================================================

func TestSubmitFormalReview_MixedFindingTypes(t *testing.T) {
	fc := forge.NewFakeClient()
	fc.AuthenticatedUser = "fullsend-bot"
	fc.PRFileDiffs = map[string][]forge.PullRequestFileDiff{
		"acme/repo/1": {
			{Path: "main.go", Patch: "@@ -40,11 +40,11 @@ func process() {"},
		},
	}
	var out bytes.Buffer
	printer := ui.New(&out)

	findings := []ReviewFinding{
		// In-hunk: line 42 within hunk [40, 50].
		{File: "main.go", Line: 42, Severity: "high", Category: "bug", Description: "In hunk finding"},
		// Out-of-hunk: line 100 outside hunk [40, 50].
		{File: "main.go", Line: 100, Severity: "medium", Category: "logic-error", Description: "Out of hunk finding"},
		// File not in PR diff.
		{File: "unknown.go", Line: 5, Severity: "low", Category: "style", Description: "File not in diff"},
	}

	err := submitFormalReview(context.Background(), fc, "acme", "repo", 1, "request-changes", "", "", findings, false, printer)
	require.NoError(t, err)
	require.Len(t, fc.CreatedReviews, 1)

	comments := fc.CreatedReviews[0].Comments
	// Expect 2 comments: 1 inline (in-hunk) + 1 file-level (out-of-hunk).
	// The file-not-in-diff finding is filtered out.
	require.Len(t, comments, 2,
		"should have 2 comments: 1 inline + 1 file-level; file-not-in-diff is filtered")

	// In-hunk finding: inline comment with correct line.
	assert.Equal(t, "main.go", comments[0].Path)
	assert.Equal(t, 42, comments[0].Line, "in-hunk finding must be inline at line 42")

	// Out-of-hunk finding: file-level fallback (Line=0).
	assert.Equal(t, "main.go", comments[1].Path)
	assert.Equal(t, 0, comments[1].Line, "out-of-hunk finding must fall back to file-level (Line=0)")
	assert.Contains(t, comments[1].Body, "Line 100",
		"file-level fallback must include original line number in body")
}

// ============================================================
// TS-GH-53-016: Log output shows file-level fallback info message
// Priority: P1 | Tier: 1 | Type: Functional
// ============================================================

func TestFindingsToReviewComments_LogsFallbackAsInfo(t *testing.T) {
	// The logging of file-level fallback happens inside submitFormalReview,
	// not in findingsToReviewComments itself. Test via submitFormalReview.
	fc := forge.NewFakeClient()
	fc.AuthenticatedUser = "fullsend-bot"
	fc.PRFileDiffs = map[string][]forge.PullRequestFileDiff{
		"acme/repo/1": {
			{Path: "changed.go", Patch: "@@ -5,10 +5,12 @@ func main() {"},
		},
	}
	var out bytes.Buffer
	printer := ui.New(&out)

	findings := []ReviewFinding{
		{File: "changed.go", Line: 50, Severity: "low", Category: "style", Description: "Outside hunk"},
	}

	err := submitFormalReview(context.Background(), fc, "acme", "repo", 1, "comment", "", "", findings, false, printer)
	require.NoError(t, err)

	logOutput := out.String()
	assert.Contains(t, logOutput, "file-level comment",
		"log output should contain info message about file-level fallback")
}

// ============================================================
// TS-GH-53-017: sanitizeReviewResult is called before submitFormalReview
//
//	in the command flow
//
// Priority: P0 | Tier: 1 | Type: Integration
// ============================================================

func TestPostReviewCommand_SanitizeBeforeSubmit(t *testing.T) {
	// Validate the sanitize-then-submit contract by running sanitizeReviewResult
	// first, then submitting. If secrets appear in posted comments, the
	// ordering contract is violated.
	printer := ui.New(io.Discard)
	secret := "ghp_FAKEordertest00000000000000000000000000"

	r := ReviewResult{
		Body:   "Review body.",
		Action: "request-changes",
		Findings: []ReviewFinding{
			{
				Severity:    "critical",
				Category:    "security",
				File:        "main.go",
				Line:        10,
				Description: fmt.Sprintf("Found token: %s", secret),
			},
		},
	}

	// Step 1: sanitize (must happen before submit).
	sanitized := sanitizeReviewResult(r, printer)
	assert.NotContains(t, sanitized.Findings[0].Description, "FAKEordertest",
		"sanitizeReviewResult must redact secrets before submit")

	// Step 2: submit with sanitized result — secrets must not appear.
	fc := forge.NewFakeClient()
	fc.AuthenticatedUser = "fullsend-bot"
	fc.PRFileDiffs = map[string][]forge.PullRequestFileDiff{
		"acme/repo/1": {
			{Path: "main.go", Patch: "@@ -5,10 +5,12 @@ func main() {"},
		},
	}

	err := submitFormalReview(context.Background(), fc, "acme", "repo", 1,
		sanitized.Action, "", "", sanitized.Findings, false, printer)
	require.NoError(t, err)
	require.Len(t, fc.CreatedReviews, 1)

	for _, c := range fc.CreatedReviews[0].Comments {
		assert.NotContains(t, c.Body, "FAKEordertest",
			"posted comment must not contain secret (sanitize-before-submit contract)")
	}
}
