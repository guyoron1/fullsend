package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// Section 3.5 — Post-Review — Inline Comment Mapping
// =============================================================================

// TS-GH73-034: Finding with file + line in diff hunk — inline comment
func TestQF_FindingsToReviewComments_InHunk(t *testing.T) {
	findings := []ReviewFinding{
		{Severity: "high", Category: "bug", File: "main.go", Line: 15, Description: "issue"},
	}
	diffHunks := map[string][][2]int{
		"main.go": {{10, 20}},
	}

	comments, _, _ := findingsToReviewComments(findings, diffHunks)

	assert.Len(t, comments, 1)
	assert.Equal(t, "main.go", comments[0].Path)
	assert.Equal(t, 15, comments[0].Line)
}

// TS-GH73-035: Finding without file path — omitted
func TestQF_FindingsToReviewComments_NoFile(t *testing.T) {
	findings := []ReviewFinding{
		{Severity: "high", Category: "bug", File: "", Line: 15, Description: "issue"},
	}
	diffHunks := map[string][][2]int{
		"main.go": {{10, 20}},
	}

	comments, _, _ := findingsToReviewComments(findings, diffHunks)

	assert.Empty(t, comments)
}

// TS-GH73-036: Finding with line=0 — omitted
func TestQF_FindingsToReviewComments_LineZero(t *testing.T) {
	findings := []ReviewFinding{
		{Severity: "high", Category: "bug", File: "main.go", Line: 0, Description: "issue"},
	}
	diffHunks := map[string][][2]int{
		"main.go": {{10, 20}},
	}

	comments, _, _ := findingsToReviewComments(findings, diffHunks)

	assert.Empty(t, comments)
}

// TS-GH73-037: Finding on file not in PR diff — filtered out
func TestQF_FindingsToReviewComments_FileNotInDiff(t *testing.T) {
	findings := []ReviewFinding{
		{Severity: "high", Category: "bug", File: "other.go", Line: 10, Description: "issue"},
	}
	diffHunks := map[string][][2]int{
		"main.go": {{10, 20}},
		"util.go": {{5, 15}},
	}

	comments, fileFiltered, _ := findingsToReviewComments(findings, diffHunks)

	assert.Empty(t, comments)
	assert.Equal(t, 1, fileFiltered)
}

// TS-GH73-038: Finding on file in diff but line outside hunk — file-level fallback
func TestQF_FindingsToReviewComments_OutsideHunk(t *testing.T) {
	findings := []ReviewFinding{
		{Severity: "high", Category: "bug", File: "main.go", Line: 50, Description: "issue outside hunk"},
	}
	diffHunks := map[string][][2]int{
		"main.go": {{10, 20}},
	}

	comments, _, fileLevelFallback := findingsToReviewComments(findings, diffHunks)

	assert.Len(t, comments, 1)
	assert.Equal(t, 0, comments[0].Line, "file-level comment should have Line=0")
	assert.Contains(t, comments[0].Body, "Line 50")
	assert.Equal(t, 1, fileLevelFallback)
}

// TS-GH73-039: Binary file — line filtering skipped
func TestQF_FindingsToReviewComments_BinaryFile(t *testing.T) {
	findings := []ReviewFinding{
		{Severity: "info", Category: "binary", File: "image.png", Line: 1, Description: "binary file change"},
	}
	// Binary files have the file in diff but with empty hunks (nil/empty slice)
	diffHunks := map[string][][2]int{
		"image.png": {},
	}

	comments, _, _ := findingsToReviewComments(findings, diffHunks)

	assert.Len(t, comments, 1)
}

// TS-GH73-040: Multiple findings across files — each mapped correctly
func TestQF_FindingsToReviewComments_MultipleFiles(t *testing.T) {
	findings := []ReviewFinding{
		{Severity: "high", Category: "bug", File: "main.go", Line: 15, Description: "issue 1"},
		{Severity: "medium", Category: "style", File: "util.go", Line: 10, Description: "issue 2"},
		{Severity: "low", Category: "perf", File: "main.go", Line: 12, Description: "issue 3"},
	}
	diffHunks := map[string][][2]int{
		"main.go": {{10, 20}},
		"util.go": {{5, 15}},
	}

	comments, _, _ := findingsToReviewComments(findings, diffHunks)

	assert.Len(t, comments, 3)
	assert.Equal(t, "main.go", comments[0].Path)
	assert.Equal(t, "util.go", comments[1].Path)
	assert.Equal(t, "main.go", comments[2].Path)
}

// TS-GH73-041: All severities pass through
func TestQF_FindingsToReviewComments_AllSeverities(t *testing.T) {
	severities := []string{"info", "low", "medium", "high", "critical"}
	var findings []ReviewFinding
	for i, sev := range severities {
		findings = append(findings, ReviewFinding{
			Severity: sev, Category: "test", File: "main.go",
			Line: 10 + i, Description: sev + " issue",
		})
	}
	diffHunks := map[string][][2]int{
		"main.go": {{10, 20}},
	}

	comments, _, _ := findingsToReviewComments(findings, diffHunks)

	assert.Len(t, comments, 5)
}

// TS-GH73-042: Finding with remediation — body includes 'Suggested fix:'
func TestQF_FormatFindingComment_WithRemediation(t *testing.T) {
	f := ReviewFinding{
		Severity:    "high",
		Category:    "concurrency",
		Description: "Race condition detected",
		Remediation: "Use sync.Mutex instead",
	}

	body := formatFindingComment(f)

	assert.Contains(t, body, "Suggested fix:")
	assert.Contains(t, body, "Use sync.Mutex instead")
}

// TS-GH73-043: Finding without remediation — no 'Suggested fix:'
func TestQF_FormatFindingComment_WithoutRemediation(t *testing.T) {
	f := ReviewFinding{
		Severity:    "medium",
		Category:    "style",
		Description: "Naming convention violated",
		Remediation: "",
	}

	body := formatFindingComment(f)

	assert.NotContains(t, body, "Suggested fix:")
}
