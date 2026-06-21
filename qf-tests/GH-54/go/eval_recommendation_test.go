package tests

/*
Evaluation Recommendation Tests

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
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestEvalRecommendation validates that the GH-54 evaluation produces
// an actionable recommendation with supporting evidence.
func TestEvalRecommendation(t *testing.T) {
	docContent := readEvalDocument(t)

	t.Run("[test_id:TS-GH-54-007] should conclude with adopt/defer/reject recommendation", func(t *testing.T) {
		// Search for recommendation section heading
		sectionPattern := regexp.MustCompile(`(?im)^#+\s*(recommendation|conclusion|verdict|decision)`)
		hasSection := sectionPattern.MatchString(docContent)
		assert.True(t, hasSection,
			"Evaluation document should contain a Recommendation, Conclusion, Verdict, or Decision section heading",
		)

		// Verify recommendation uses standard language
		recKeywords := regexp.MustCompile(`(?i)(\badopt\b|\bdefer\b|\breject\b|\brecommend\b)`)
		hasKeywords := recKeywords.MatchString(docContent)
		assert.True(t, hasKeywords,
			"Evaluation document should contain at least one recommendation keyword (adopt/defer/reject/recommend)",
		)
	})

	t.Run("[test_id:TS-GH-54-008] should include justification referencing FullSend architecture", func(t *testing.T) {
		// Verify justification references FullSend components or architecture
		// Look for recommendation/conclusion keywords near FullSend component names
		justificationPattern := regexp.MustCompile(
			`(?i)(recommend|conclusion|verdict)[\s\S]{0,500}(fullsend|FullSend|forge|harness|sandbox|agent)`,
		)
		hasJustification := justificationPattern.MatchString(docContent)
		assert.True(t, hasJustification,
			"Recommendation section should contain justification referencing FullSend components (forge, harness, sandbox, agent) within 500 chars of recommendation language",
		)
	})

	t.Run("[test_id:TS-GH-54-009] should identify follow-up implementation issues if adoption recommended", func(t *testing.T) {
		// Search for follow-up or next steps section
		followUpPattern := regexp.MustCompile(
			`(?i)(follow.?up|next\s+steps|implementation\s+(plan|issue|task)|action\s+item)`,
		)
		hasFollowUp := followUpPattern.MatchString(docContent)
		assert.True(t, hasFollowUp,
			"Evaluation document should contain follow-up actions, next steps, or implementation planning language",
		)
	})
}
