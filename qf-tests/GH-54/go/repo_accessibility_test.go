package tests

/*
Repository Accessibility and Deprecation Tests

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
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestRepoAccessibility validates that the GH-54 evaluation handles
// external project accessibility and deprecation scenarios.
func TestRepoAccessibility(t *testing.T) {
	docContent := readEvalDocument(t)
	lowerContent := strings.ToLower(docContent)

	t.Run("[test_id:TS-GH-54-010] should document repository accessibility status for all three projects", func(t *testing.T) {
		// Verify repository name references are present for all three projects
		repos := []string{"gastown", "gascity", "goosetown"}
		for _, repo := range repos {
			assert.True(t,
				strings.Contains(lowerContent, repo),
				"Evaluation document should reference repository %q", repo,
			)
		}

		// Check for accessibility status language
		statusPattern := regexp.MustCompile(`(?i)(public|archived|accessible|available|private|deprecated|maintained)`)
		hasStatus := statusPattern.MatchString(docContent)
		assert.True(t, hasStatus,
			"Evaluation document should contain accessibility or maintenance status language (public, archived, accessible, available, private, deprecated, maintained)",
		)
	})

	t.Run("[test_id:TS-GH-54-011] should handle case where original Gastown is deprecated in favor of gascity", func(t *testing.T) {
		// Verify Gastown-gascity relationship documented using transition language
		transitionPattern := regexp.MustCompile(
			`(?i)(re.?architect|deprecat|successor|replac|evolution|fork).*(gastown|gascity)`,
		)
		reverseTransition := regexp.MustCompile(
			`(?i)(gastown|gascity).*(re.?architect|deprecat|successor|replac|evolution|fork)`,
		)
		hasRelationship := transitionPattern.MatchString(docContent) || reverseTransition.MatchString(docContent)
		assert.True(t, hasRelationship,
			"Document should discuss Gastown-gascity relationship using transition language (re-architect, deprecated, successor, replaced, evolution, fork)",
		)

		// Verify evaluation identifies current/maintained version
		currentPattern := regexp.MustCompile(`(?i)(current|maintained|active|primary).*(gascity|version)`)
		hasCurrentVersion := currentPattern.MatchString(docContent)
		assert.True(t, hasCurrentVersion,
			"Document should identify the current/maintained version of the project",
		)
	})
}
