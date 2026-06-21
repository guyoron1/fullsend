package tests

/*
Gastown Evaluation Document Completeness Tests

STP Reference: outputs/stp/GH-54/GH-54_test_plan.md
STD Reference: outputs/std/GH-54/GH-54_test_description.yaml
Jira: GH-54

Markers:
    - tier1

Preconditions:
    - GitHub Actions runner with internet access
    - Go 1.23+ toolchain installed
    - Evaluation document produced by GH-54 research task
*/

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// evalDocOutputDir returns the output directory where evaluation documents
// are expected. It checks the EVAL_OUTPUT_DIR environment variable first,
// then falls back to a relative path from the repo root.
func evalDocOutputDir(t *testing.T) string {
	t.Helper()
	if dir := os.Getenv("EVAL_OUTPUT_DIR"); dir != "" {
		return dir
	}
	// Default: look relative to repo root
	return filepath.Join("..", "..", "outputs")
}

// findEvalDocument locates the evaluation document in the output directory.
// It searches for files matching common evaluation document name patterns.
func findEvalDocument(t *testing.T) string {
	t.Helper()
	outputDir := evalDocOutputDir(t)

	// Try several common patterns for evaluation documents
	patterns := []string{
		filepath.Join(outputDir, "*evaluation*"),
		filepath.Join(outputDir, "*eval*"),
		filepath.Join(outputDir, "GH-54*"),
		filepath.Join(outputDir, "*gastown*"),
	}

	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil || info.IsDir() {
				continue
			}
			return match
		}
	}

	t.Fatalf("Could not find evaluation document in %s. Set EVAL_OUTPUT_DIR to the directory containing the evaluation document.", outputDir)
	return ""
}

// readEvalDocument reads the evaluation document content and returns it as a string.
func readEvalDocument(t *testing.T) string {
	t.Helper()
	docPath := findEvalDocument(t)
	content, err := os.ReadFile(docPath)
	require.NoError(t, err, "Failed to read evaluation document at %s", docPath)
	require.NotEmpty(t, content, "Evaluation document at %s is empty", docPath)
	return string(content)
}

// TestEvalDocumentCompleteness validates that the GH-54 evaluation
// deliverable covers all required projects and analysis areas.
func TestEvalDocumentCompleteness(t *testing.T) {
	docContent := readEvalDocument(t)
	lowerContent := strings.ToLower(docContent)

	t.Run("[test_id:TS-GH-54-001] should cover all three projects (Gastown, gascity, goosetown)", func(t *testing.T) {
		// Verify document references all three projects
		expectedProjects := []string{"gastown", "gascity", "goosetown"}

		for _, proj := range expectedProjects {
			assert.True(t,
				strings.Contains(lowerContent, proj),
				"Evaluation document must reference project %q but it was not found", proj,
			)
		}

		// Verify each project has substantive coverage (not just a passing mention)
		// by checking that each project name appears at least twice
		for _, proj := range expectedProjects {
			count := strings.Count(lowerContent, proj)
			assert.GreaterOrEqual(t, count, 2,
				"Project %q should have substantive coverage (found %d references, expected at least 2)", proj, count,
			)
		}
	})

	t.Run("[test_id:TS-GH-54-002] should include architecture analysis for each project", func(t *testing.T) {
		// Verify architecture analysis sections exist for each project
		projects := []string{"Gastown", "gascity", "goosetown"}
		architectureKeywords := []string{"architecture", "design", "structure", "pattern"}

		for _, proj := range projects {
			projLower := strings.ToLower(proj)
			hasArchAnalysis := false
			for _, keyword := range architectureKeywords {
				// Check if the project name appears near architecture keywords
				pattern := `(?i)` + projLower + `[\s\S]{0,300}` + keyword
				reversePattern := `(?i)` + keyword + `[\s\S]{0,300}` + projLower
				if regexp.MustCompile(pattern).MatchString(docContent) ||
					regexp.MustCompile(reversePattern).MatchString(docContent) {
					hasArchAnalysis = true
					break
				}
			}
			assert.True(t, hasArchAnalysis,
				"Evaluation should include architecture analysis for %s (no architecture/design/structure/pattern keywords found near %s)", proj, proj,
			)
		}
	})

	t.Run("[test_id:TS-GH-54-003] should map Gastown capabilities to FullSend problem areas", func(t *testing.T) {
		// Check for FullSend problem area references
		fullsendAreas := []string{"agent", "sandbox", "forge", "orchestration", "harness"}
		matchCount := 0
		for _, area := range fullsendAreas {
			if strings.Contains(lowerContent, area) {
				matchCount++
			}
		}
		assert.GreaterOrEqual(t, matchCount, 2,
			"At least 2 of 5 FullSend problem areas (agent, sandbox, forge, orchestration, harness) should be referenced; found %d", matchCount,
		)

		// Verify capability mapping structure — relevance assessment language
		// connecting external projects to FullSend
		relevancePattern := regexp.MustCompile(`(?i)(relevance|overlap|complement|map|integrate).*(fullsend|FullSend)`)
		reverseRelevance := regexp.MustCompile(`(?i)(fullsend|FullSend).*(relevance|overlap|complement|map|integrate)`)
		hasMapping := relevancePattern.MatchString(docContent) || reverseRelevance.MatchString(docContent)
		assert.True(t, hasMapping,
			"Document should contain relevance assessment language connecting external projects to FullSend",
		)
	})
}
