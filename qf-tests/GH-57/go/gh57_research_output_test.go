//go:build e2e

package tests

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// researchDocRelativePaths lists candidate locations where the GH-57 research
// summary might be written. Tests walk these in order until one is found.
var researchDocRelativePaths = []string{
	"outputs/research/GH-57/research_summary.md",
	"outputs/research/GH-57/GH-57_research_summary.md",
	"docs/research/GH-57.md",
	"research_summary.md",
}

// resolveResearchDocPath walks candidate paths relative to the repo root
// and returns the first that exists, or "" if none is found.
func resolveResearchDocPath(t *testing.T) string {
	t.Helper()

	repoRoot := os.Getenv("REPO_ROOT")
	if repoRoot == "" {
		// Fall back to current working directory
		var err error
		repoRoot, err = os.Getwd()
		require.NoError(t, err, "cannot determine repo root")
	}

	for _, rel := range researchDocRelativePaths {
		abs := filepath.Join(repoRoot, rel)
		if FileExists(abs) {
			return abs
		}
	}
	return ""
}

// TS-GH-57-001: Verify research summary document is produced with applicable insights.
//
// Validates that the research task produces a summary document containing
// applicable insights extracted from the latent.space article. The document
// must contain at least 3 distinct insights.
func TestResearchSummaryProducedWithInsights(t *testing.T) {
	// [test_id:TS-GH-57-001]

	// SETUP-01: Identify research output document path
	researchDocPath := resolveResearchDocPath(t)
	require.NotEmpty(t, researchDocPath,
		"research summary document not found at any candidate path: %v", researchDocRelativePaths)

	// TEST-01: Verify research summary document exists and is non-empty
	stat, err := os.Stat(researchDocPath)
	require.NoError(t, err, "cannot stat research summary document")
	require.Greater(t, stat.Size(), int64(0),
		"research summary document is empty")

	// TEST-02: Parse document for applicable insights
	docContent := ReadFileContent(t, researchDocPath)
	insightCount := CountMatches(docContent, insightPattern)
	assert.GreaterOrEqual(t, insightCount, 3,
		"research summary must contain at least 3 applicable insights, found %d", insightCount)

	// TEST-03: Verify insights reference the source article
	hasLatentSpace := strings.Contains(docContent, "latent.space")
	hasArticleTitle := strings.Contains(docContent, "Are Code Reviews Dead")
	assert.True(t, hasLatentSpace || hasArticleTitle,
		"research summary must reference the source article (latent.space or 'Are Code Reviews Dead')")
}

// TS-GH-57-002: Verify insights reference specific FullSend components.
//
// Validates that each insight in the research summary references specific
// FullSend components rather than making generic recommendations.
func TestInsightsReferenceFullSendComponents(t *testing.T) {
	// [test_id:TS-GH-57-002]

	// SETUP-01: Read research summary document
	researchDocPath := resolveResearchDocPath(t)
	require.NotEmpty(t, researchDocPath,
		"research summary document not found — cannot validate component references")
	docContent := ReadFileContent(t, researchDocPath)

	// SETUP-02: Known FullSend component names are defined in helpers

	// TEST-01: Extract insights from document
	insights := ExtractInsightSections(docContent)
	require.NotEmpty(t, insights, "no insight sections found in research summary")

	// TEST-02: Check each insight for FullSend component references
	for i, insight := range insights {
		refs := ContainsComponentReference(insight)
		assert.NotEmpty(t, refs,
			"insight #%d does not reference any known FullSend component; "+
				"expected references to components like: %s",
			i+1, strings.Join(knownFullSendComponents, ", "))
	}

	// TEST-03: Verify at least some component references use correct terminology
	allText := strings.ToLower(docContent)
	var foundTerms []string
	for _, comp := range knownFullSendComponents {
		if strings.Contains(allText, strings.ToLower(comp)) {
			foundTerms = append(foundTerms, comp)
		}
	}
	assert.NotEmpty(t, foundTerms,
		"research summary does not use any recognized FullSend terminology")
}

// ghIssue represents a GitHub issue returned by `gh issue list --json`.
type ghIssue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

// TS-GH-57-003: Verify follow-up issues are filed for actionable recommendations.
//
// Validates that for each actionable recommendation a corresponding GitHub
// issue has been filed in the FullSend repository, referencing GH-57.
func TestFollowUpIssuesFiledForRecommendations(t *testing.T) {
	// [test_id:TS-GH-57-003]

	// SETUP-01: Read research summary to count actionable recommendations
	researchDocPath := resolveResearchDocPath(t)
	require.NotEmpty(t, researchDocPath,
		"research summary not found — cannot determine expected follow-up issue count")
	docContent := ReadFileContent(t, researchDocPath)
	insightCount := CountMatches(docContent, insightPattern)
	require.Greater(t, insightCount, 0,
		"no actionable insights found in research summary")

	// SETUP-02: Verify gh CLI is available
	_, err := exec.LookPath("gh")
	require.NoError(t, err, "gh CLI not found in PATH — cannot verify follow-up issues")

	// TEST-01: List follow-up issues referencing GH-57
	cmd := exec.Command("gh", "issue", "list",
		"--repo", "fullsend-ai/fullsend",
		"--search", "GH-57",
		"--json", "number,title,body",
		"--limit", "50",
	)
	out, err := cmd.Output()
	require.NoError(t, err,
		"failed to list GitHub issues: %s", string(out))

	var followUpIssues []ghIssue
	require.NoError(t, json.Unmarshal(out, &followUpIssues),
		"failed to parse GitHub issue JSON")

	// TEST-02: Verify issue count matches recommendation count
	assert.GreaterOrEqual(t, len(followUpIssues), insightCount,
		"expected at least %d follow-up issues (one per actionable recommendation), found %d",
		insightCount, len(followUpIssues))

	// TEST-03: Verify each issue references GH-57
	for _, issue := range followUpIssues {
		combined := issue.Title + " " + issue.Body
		assert.True(t,
			strings.Contains(combined, "GH-57") || strings.Contains(combined, "gh-57"),
			"follow-up issue #%d (%q) does not reference GH-57",
			issue.Number, issue.Title)
	}

	// TEST-04: Verify each issue describes a specific recommendation
	for _, issue := range followUpIssues {
		assert.Greater(t, len(issue.Body), 50,
			"follow-up issue #%d (%q) has an insufficient description (body length: %d)",
			issue.Number, issue.Title, len(issue.Body))
	}
}

// TS-GH-57-004: Verify research output does not duplicate existing capabilities.
//
// Negative test: validates that recommendations don't propose features FullSend
// already provides (pr-review agent, code-review skill, harness, dispatch, etc.).
func TestResearchOutputNoExistingCapabilityDuplication(t *testing.T) {
	// [test_id:TS-GH-57-004]

	// SETUP-01: Read research summary document
	researchDocPath := resolveResearchDocPath(t)
	require.NotEmpty(t, researchDocPath,
		"research summary not found — cannot check for capability duplication")
	docContent := ReadFileContent(t, researchDocPath)

	// SETUP-02: existing capabilities are defined in helpers

	// TEST-01: Extract individual recommendations
	recommendations := ExtractInsightSections(docContent)
	require.NotEmpty(t, recommendations,
		"no recommendations found in research summary")

	// TEST-02: Compare each recommendation against existing capabilities
	for i, rec := range recommendations {
		recLower := strings.ToLower(rec)

		for _, cap := range existingCapabilities {
			capNameLower := strings.ToLower(cap.Name)

			// Check if the recommendation appears to propose reimplementing
			// an existing capability rather than enhancing it.
			if strings.Contains(recLower, capNameLower) {
				// If it mentions the capability, it should frame it as an
				// enhancement (words like "improve", "enhance", "extend",
				// "augment", "build on") rather than a new implementation
				// (words like "implement", "create", "build", "add").
				enhancementTerms := []string{
					"improve", "enhance", "extend", "augment", "build on",
					"refine", "optimize", "upgrade", "enrich",
				}
				reimplementTerms := []string{
					"implement from scratch", "create a new", "build a new",
					"replace the existing", "rewrite",
				}

				hasReimplementLanguage := false
				for _, term := range reimplementTerms {
					if strings.Contains(recLower, term) {
						hasReimplementLanguage = true
						break
					}
				}

				if hasReimplementLanguage {
					// Confirm no enhancement language to soften it
					hasEnhancementLanguage := false
					for _, term := range enhancementTerms {
						if strings.Contains(recLower, term) {
							hasEnhancementLanguage = true
							break
						}
					}
					assert.True(t, hasEnhancementLanguage,
						"recommendation #%d proposes reimplementing existing capability %q "+
							"without framing as an enhancement; existing capability: %s",
						i+1, cap.Name, cap.Description)
				}
			}
		}
	}

	// TEST-03: Log which existing capabilities are referenced (informational)
	for _, cap := range existingCapabilities {
		if strings.Contains(strings.ToLower(docContent), strings.ToLower(cap.Name)) {
			t.Logf("INFO: existing capability %q is referenced in research summary", cap.Name)
		}
	}
}

// TS-GH-57-005: Verify quality gate rejects documents with fewer than 3 insights.
//
// Boundary test: validates that the insight count threshold correctly rejects
// documents below the minimum and accepts documents at the threshold.
func TestResearchSummaryRejectsInsufficientInsights(t *testing.T) {
	// [test_id:TS-GH-57-005]

	const insightThreshold = 3

	t.Run("zero insights are rejected", func(t *testing.T) {
		// SETUP: Create temp doc with 0 insights
		docPath := createTempDocWithInsights(t, 0)
		content := ReadFileContent(t, docPath)
		count := CountMatches(content, insightPattern)
		assert.False(t, meetsInsightThreshold(count, insightThreshold),
			"quality gate should reject document with 0 insights")
	})

	t.Run("one insight is rejected", func(t *testing.T) {
		docPath := createTempDocWithInsights(t, 1)
		content := ReadFileContent(t, docPath)
		count := CountMatches(content, insightPattern)
		assert.False(t, meetsInsightThreshold(count, insightThreshold),
			"quality gate should reject document with 1 insight")
	})

	t.Run("two insights are rejected", func(t *testing.T) {
		// TEST-01 & TEST-02: boundary at exactly 2
		docPath := createTempDocWithInsights(t, 2)
		content := ReadFileContent(t, docPath)
		count := CountMatches(content, insightPattern)
		assert.Equal(t, 2, count, "fixture should contain exactly 2 insight sections")
		assert.False(t, meetsInsightThreshold(count, insightThreshold),
			"quality gate should reject document with 2 insights (below threshold of %d)", insightThreshold)
	})

	t.Run("three insights are accepted", func(t *testing.T) {
		// TEST-03 & TEST-04: boundary at exactly 3
		docPath := createTempDocWithInsights(t, 3)
		content := ReadFileContent(t, docPath)
		count := CountMatches(content, insightPattern)
		assert.Equal(t, 3, count, "fixture should contain exactly 3 insight sections")
		assert.True(t, meetsInsightThreshold(count, insightThreshold),
			"quality gate should accept document with exactly %d insights", insightThreshold)
	})

	t.Run("four insights are accepted", func(t *testing.T) {
		docPath := createTempDocWithInsights(t, 4)
		content := ReadFileContent(t, docPath)
		count := CountMatches(content, insightPattern)
		assert.GreaterOrEqual(t, count, insightThreshold,
			"quality gate should accept document with %d insights (above threshold)", count)
		assert.True(t, meetsInsightThreshold(count, insightThreshold))
	})

	t.Run("threshold function unit tests", func(t *testing.T) {
		assert.False(t, meetsInsightThreshold(0, 3), "0 < 3")
		assert.False(t, meetsInsightThreshold(1, 3), "1 < 3")
		assert.False(t, meetsInsightThreshold(2, 3), "2 < 3")
		assert.True(t, meetsInsightThreshold(3, 3), "3 == 3")
		assert.True(t, meetsInsightThreshold(4, 3), "4 > 3")
		assert.True(t, meetsInsightThreshold(100, 3), "100 > 3")
	})
}

func init() {
	// Suppress unused import warnings — these are used conditionally.
	_ = fmt.Sprintf
	_ = filepath.Join
}
