package cli

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/forge"
	"github.com/fullsend-ai/fullsend/internal/ui"
)

// TestFindingsToReviewComments_OutOfHunkFallbackToFileLevel verifies that a
// finding whose line falls outside every diff hunk is posted as a file-level
// comment with Line=0. This is the core behavioral change in GH-41.
// [test_id:TS-GH-41-001]
func TestFindingsToReviewComments_OutOfHunkFallbackToFileLevel(t *testing.T) {
	findings := []ReviewFinding{
		{File: "main.go", Line: 150, Severity: "high", Category: "bug", Description: "Potential nil dereference"},
	}
	diffHunks := map[string][][2]int{
		"main.go": {{10, 30}, {50, 70}},
	}

	comments, fileFiltered, fileLevelFallback := findingsToReviewComments(findings, diffHunks)

	require.Len(t, comments, 1, "out-of-hunk finding must not be dropped")
	assert.Equal(t, 0, comments[0].Line, "out-of-hunk finding should become file-level (Line=0)")
	assert.Equal(t, "main.go", comments[0].Path)
	assert.Equal(t, 0, fileFiltered)
	assert.Equal(t, 1, fileLevelFallback)
}

// TestFindingsToReviewComments_NoFilePathSkipped verifies that findings with
// an empty file path are silently skipped and produce no ReviewComment.
// [test_id:TS-GH-41-002]
func TestFindingsToReviewComments_NoFilePathSkipped(t *testing.T) {
	findings := []ReviewFinding{
		{File: "", Line: 10, Severity: "medium", Category: "style", Description: "General code smell"},
	}
	diffHunks := map[string][][2]int{
		"main.go": {{1, 100}},
	}

	comments, fileFiltered, fileLevelFallback := findingsToReviewComments(findings, diffHunks)

	assert.Empty(t, comments, "finding without file path must be skipped")
	assert.Equal(t, 0, fileFiltered)
	assert.Equal(t, 0, fileLevelFallback)
}

// TestFindingsToReviewComments_FallbackBodyContainsOriginalLine verifies that
// when a finding falls back to file-level, the comment body includes the
// original line number so reviewers retain location context.
// [test_id:TS-GH-41-004]
func TestFindingsToReviewComments_FallbackBodyContainsOriginalLine(t *testing.T) {
	findings := []ReviewFinding{
		{File: "main.go", Line: 150, Severity: "high", Category: "bug", Description: "Potential nil dereference"},
	}
	diffHunks := map[string][][2]int{
		"main.go": {{10, 30}, {50, 70}},
	}

	comments, _, _ := findingsToReviewComments(findings, diffHunks)

	require.Len(t, comments, 1)
	assert.Contains(t, comments[0].Body, "150", "fallback body must contain original line number")
}

// TestFindingsToReviewComments_FallbackBodyFormat verifies that file-level
// fallback comments use the exact format "_Line N_ . description".
// [test_id:TS-GH-41-005]
func TestFindingsToReviewComments_FallbackBodyFormat(t *testing.T) {
	findings := []ReviewFinding{
		{File: "main.go", Line: 42, Severity: "medium", Category: "style", Description: "Unused variable detected"},
	}
	diffHunks := map[string][][2]int{
		"main.go": {{10, 30}},
	}

	comments, _, _ := findingsToReviewComments(findings, diffHunks)

	require.Len(t, comments, 1)
	assert.Contains(t, comments[0].Body, "_Line 42_", "fallback body must start with '_Line N_' prefix")
	assert.Contains(t, comments[0].Body, "Unused variable detected")
	assert.Contains(t, comments[0].Body, "**[medium]** style")
}

// TestFindingsToReviewComments_InHunkRetainsLine verifies that findings whose
// line falls within a diff hunk retain the original line number and are not
// converted to file-level comments.
// [test_id:TS-GH-41-006]
func TestFindingsToReviewComments_InHunkRetainsLine(t *testing.T) {
	findings := []ReviewFinding{
		{File: "main.go", Line: 25, Severity: "high", Category: "bug", Description: "Missing error check"},
	}
	diffHunks := map[string][][2]int{
		"main.go": {{10, 30}},
	}

	comments, _, fileLevelFallback := findingsToReviewComments(findings, diffHunks)

	require.Len(t, comments, 1)
	assert.Equal(t, 25, comments[0].Line, "in-hunk finding must retain its original line number")
	assert.Equal(t, 0, fileLevelFallback)
}

// TestFindingsToReviewComments_InHunkBodyNoLinePrefix verifies that in-hunk
// findings do NOT get the "_Line N_" prefix in their body, since they display
// at the correct line in the diff view.
// [test_id:TS-GH-41-007]
func TestFindingsToReviewComments_InHunkBodyNoLinePrefix(t *testing.T) {
	findings := []ReviewFinding{
		{File: "main.go", Line: 25, Severity: "high", Category: "bug", Description: "Missing error check"},
	}
	diffHunks := map[string][][2]int{
		"main.go": {{10, 30}},
	}

	comments, _, _ := findingsToReviewComments(findings, diffHunks)

	require.Len(t, comments, 1)
	assert.NotContains(t, comments[0].Body, "_Line", "in-hunk comment body must not contain '_Line' prefix")
	assert.Contains(t, comments[0].Body, "Missing error check")
}

// TestFindingsToReviewComments_FileNotInDiffOmitted verifies that findings
// referencing files not present in diffHunks are filtered out entirely and
// increment the fileFiltered counter.
// [test_id:TS-GH-41-008]
func TestFindingsToReviewComments_FileNotInDiffOmitted(t *testing.T) {
	findings := []ReviewFinding{
		{File: "other_file.go", Line: 10, Severity: "high", Category: "bug", Description: "Issue in unrelated file"},
	}
	diffHunks := map[string][][2]int{
		"main.go": {{1, 100}},
	}

	comments, fileFiltered, fileLevelFallback := findingsToReviewComments(findings, diffHunks)

	assert.Empty(t, comments, "finding for file not in diff must be omitted")
	assert.Equal(t, 1, fileFiltered)
	assert.Equal(t, 0, fileLevelFallback)
}

// TestFindingsToReviewComments_FileFilteredCount verifies that the
// fileFiltered counter accurately reflects the number of findings whose
// file is not in the PR diff.
// [test_id:TS-GH-41-009]
func TestFindingsToReviewComments_FileFilteredCount(t *testing.T) {
	findings := []ReviewFinding{
		{File: "not-in-diff-1.go", Line: 10, Severity: "high", Category: "bug", Description: "Filtered 1"},
		{File: "not-in-diff-2.go", Line: 20, Severity: "medium", Category: "bug", Description: "Filtered 2"},
		{File: "in-diff.go", Line: 5, Severity: "low", Category: "style", Description: "Kept"},
	}
	diffHunks := map[string][][2]int{
		"in-diff.go": {{1, 50}},
	}

	comments, fileFiltered, _ := findingsToReviewComments(findings, diffHunks)

	require.Len(t, comments, 1)
	assert.Equal(t, 2, fileFiltered, "two findings for files not in diff should be counted")
}

// TestFindingsToReviewComments_AllSeveritiesFallbackEqually verifies that
// the file-level fallback applies uniformly to all severity levels.
// [test_id:TS-GH-41-010]
func TestFindingsToReviewComments_AllSeveritiesFallbackEqually(t *testing.T) {
	severities := []string{"info", "warning", "error", "critical"}

	for _, sev := range severities {
		t.Run(sev, func(t *testing.T) {
			findings := []ReviewFinding{
				{File: "main.go", Line: 150, Severity: sev, Category: "test", Description: "Out of hunk"},
			}
			diffHunks := map[string][][2]int{
				"main.go": {{10, 30}},
			}

			comments, _, fileLevelFallback := findingsToReviewComments(findings, diffHunks)

			require.Len(t, comments, 1, "severity %q must not be filtered", sev)
			assert.Equal(t, 0, comments[0].Line, "severity %q must fall back to file-level", sev)
			assert.Equal(t, 1, fileLevelFallback)
		})
	}
}

// TestFindingsToReviewComments_CaseInsensitiveSeverity verifies that severity
// comparison is case-insensitive and mixed-case values all produce file-level
// fallback comments equally.
// [test_id:TS-GH-41-011]
func TestFindingsToReviewComments_CaseInsensitiveSeverity(t *testing.T) {
	caseVariants := []string{"HIGH", "High", "high"}

	for _, sev := range caseVariants {
		t.Run(sev, func(t *testing.T) {
			findings := []ReviewFinding{
				{File: "main.go", Line: 150, Severity: sev, Category: "bug", Description: "Out of hunk"},
			}
			diffHunks := map[string][][2]int{
				"main.go": {{10, 30}},
			}

			comments, _, fileLevelFallback := findingsToReviewComments(findings, diffHunks)

			require.Len(t, comments, 1, "severity %q must produce a comment", sev)
			assert.Equal(t, 0, comments[0].Line, "severity %q must fall back to file-level", sev)
			assert.Equal(t, 1, fileLevelFallback)
		})
	}
}

// TestFindingsToReviewComments_Line0ImpliesFileLevelSubjectType verifies that
// findingsToReviewComments outputs Line=0 for out-of-hunk findings, which the
// GitHub implementation translates to subject_type="file" in the API payload.
// [test_id:TS-GH-41-012]
func TestFindingsToReviewComments_Line0ImpliesFileLevelSubjectType(t *testing.T) {
	findings := []ReviewFinding{
		{File: "main.go", Line: 150, Severity: "high", Category: "bug", Description: "Potential nil dereference"},
	}
	diffHunks := map[string][][2]int{
		"main.go": {{10, 30}},
	}

	comments, _, _ := findingsToReviewComments(findings, diffHunks)

	require.Len(t, comments, 1)
	assert.Equal(t, 0, comments[0].Line, "Line=0 signals file-level; GitHub client sets subject_type='file'")
	assert.Equal(t, "main.go", comments[0].Path)
}

// TestFindingsToReviewComments_InHunkLinePositiveNoSubjectType verifies that
// in-hunk findings produce Line>0, which the GitHub implementation handles
// by omitting subject_type from the API payload.
// [test_id:TS-GH-41-013]
func TestFindingsToReviewComments_InHunkLinePositiveNoSubjectType(t *testing.T) {
	findings := []ReviewFinding{
		{File: "main.go", Line: 25, Severity: "high", Category: "bug", Description: "Missing error check"},
	}
	diffHunks := map[string][][2]int{
		"main.go": {{10, 30}},
	}

	comments, _, _ := findingsToReviewComments(findings, diffHunks)

	require.Len(t, comments, 1)
	assert.Greater(t, comments[0].Line, 0, "in-hunk finding must have positive Line (no subject_type in API)")
	assert.Equal(t, 25, comments[0].Line)
}

// TestFindingsToReviewComments_BinaryFileSkipsLineFiltering verifies that
// findings for binary files (present in diffHunks with nil/empty hunk list)
// bypass line-level filtering entirely and pass through.
// [test_id:TS-GH-41-015]
func TestFindingsToReviewComments_BinaryFileSkipsLineFiltering(t *testing.T) {
	findings := []ReviewFinding{
		{File: "binary.png", Line: 1, Severity: "high", Category: "bug", Description: "On binary file"},
	}
	diffHunks := map[string][][2]int{
		"binary.png": nil,
	}

	comments, fileFiltered, fileLevelFallback := findingsToReviewComments(findings, diffHunks)

	require.Len(t, comments, 1, "binary file finding must not be filtered")
	assert.Equal(t, "binary.png", comments[0].Path)
	assert.Equal(t, 1, comments[0].Line, "binary file finding should retain original line")
	assert.Equal(t, 0, fileFiltered)
	assert.Equal(t, 0, fileLevelFallback)
}

// TestFindingsToReviewComments_TruncatedPatchSkipsLineFiltering verifies that
// findings for files with truncated patches (empty hunk list) bypass line-level
// filtering and pass through, same as binary files.
// [test_id:TS-GH-41-016]
func TestFindingsToReviewComments_TruncatedPatchSkipsLineFiltering(t *testing.T) {
	findings := []ReviewFinding{
		{File: "large.go", Line: 999, Severity: "medium", Category: "style", Description: "On truncated-patch file"},
	}
	diffHunks := map[string][][2]int{
		"large.go": nil,
	}

	comments, fileFiltered, fileLevelFallback := findingsToReviewComments(findings, diffHunks)

	require.Len(t, comments, 1, "truncated-patch file finding must not be filtered")
	assert.Equal(t, "large.go", comments[0].Path)
	assert.Equal(t, 999, comments[0].Line, "truncated-patch finding should retain original line")
	assert.Equal(t, 0, fileFiltered)
	assert.Equal(t, 0, fileLevelFallback)
}

// TestSubmitFormalReview_LogsFileLevelFallbackCount verifies that when
// out-of-hunk findings fall back to file-level comments, the printer output
// includes a StepInfo message with the fallback count.
// [test_id:TS-GH-41-017]
func TestSubmitFormalReview_LogsFileLevelFallbackCount(t *testing.T) {
	fc := forge.NewFakeClient()
	fc.AuthenticatedUser = "fullsend-bot"
	fc.PRFileDiffs = map[string][]forge.PullRequestFileDiff{
		"acme/repo/1": {
			{Path: "main.go", Patch: "@@ -10,20 +10,25 @@ func main() {"},
		},
	}

	var out bytes.Buffer
	printer := ui.New(&out)

	findings := []ReviewFinding{
		{File: "main.go", Line: 10, Severity: "high", Category: "bug", Description: "In hunk"},
		{File: "main.go", Line: 100, Severity: "medium", Category: "style", Description: "Outside hunk"},
	}

	err := submitFormalReview(context.Background(), fc, "acme", "repo", 1, "request-changes", "", "", findings, false, printer)
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, "file-level comment", "printer output must mention file-level fallback")
	assert.Contains(t, output, "1 finding(s) posted as file-level comment(s)", "printer output must contain the correct fallback count")
}

// TestSubmitFormalReview_NoFallbackLogWhenCountZero verifies that when all
// findings are in-hunk (no fallbacks), the printer output does not contain
// any file-level fallback message.
// [test_id:TS-GH-41-018]
func TestSubmitFormalReview_NoFallbackLogWhenCountZero(t *testing.T) {
	fc := forge.NewFakeClient()
	fc.AuthenticatedUser = "fullsend-bot"
	fc.PRFileDiffs = map[string][]forge.PullRequestFileDiff{
		"acme/repo/1": {
			{Path: "main.go", Patch: "@@ -10,20 +10,25 @@ func main() {"},
		},
	}

	var out bytes.Buffer
	printer := ui.New(&out)

	findings := []ReviewFinding{
		{File: "main.go", Line: 15, Severity: "high", Category: "bug", Description: "In hunk"},
		{File: "main.go", Line: 20, Severity: "medium", Category: "style", Description: "Also in hunk"},
	}

	err := submitFormalReview(context.Background(), fc, "acme", "repo", 1, "request-changes", "", "", findings, false, printer)
	require.NoError(t, err)

	output := out.String()
	assert.NotContains(t, output, "file-level comment", "no fallback log when all findings are in-hunk")
}
