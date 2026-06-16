package qf_tests

/*
MCP Configuration Drift - Attack Scenario Quality Tests

STP Reference: outputs/stp/GH-17/GH-17_test_plan.md
STD Reference: outputs/std/GH-17/GH-17_test_description.yaml
Jira: GH-17
*/

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDistinctAttackScenarios validates that the document contains at least
// 3 distinct attack scenarios under the Attack scenarios subsection, each
// with a unique bold numbered label (e.g., **Scenario 1:**).
//
// Test ID: TS-GH-17-012
// Priority: P2
func TestDistinctAttackScenarios(t *testing.T) {
	content := readDocContent(t)

	scenarioRegex := regexp.MustCompile(`\*\*Scenario \d+`)
	matches := scenarioRegex.FindAllString(content, -1)

	assert.GreaterOrEqual(t, len(matches), 3,
		"Document must contain at least 3 distinct attack scenarios (found %d)", len(matches))

	// Verify uniqueness of scenario numbers
	seen := make(map[string]bool)
	for _, m := range matches {
		if seen[m] {
			t.Errorf("Duplicate scenario label found: %s", m)
		}
		seen[m] = true
	}
}

// TestEachScenarioHasDescription validates that each attack scenario label
// is followed by a substantive description of at least 50 characters.
//
// Test ID: TS-GH-17-013
// Priority: P2
func TestEachScenarioHasDescription(t *testing.T) {
	content := readDocContent(t)

	// Split by scenario labels to extract each scenario's content
	scenarioSplitter := regexp.MustCompile(`\*\*Scenario \d+[^*]*\*\*`)
	labels := scenarioSplitter.FindAllString(content, -1)
	require.NotEmpty(t, labels, "Document must contain at least one scenario label")

	parts := scenarioSplitter.Split(content, -1)
	// parts[0] is content before the first scenario, parts[1..] are after each label

	for i, label := range labels {
		scenarioContent := ""
		if i+1 < len(parts) {
			scenarioContent = parts[i+1]
		}

		t.Run(fmt.Sprintf("scenario_%s", strings.TrimSpace(label)), func(t *testing.T) {
			// Get the text until the next scenario or section heading
			lines := strings.Split(scenarioContent, "\n")
			var descriptionBuilder strings.Builder
			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				// Stop at next scenario or major heading
				if strings.HasPrefix(trimmed, "**Scenario") || strings.HasPrefix(trimmed, "## ") || strings.HasPrefix(trimmed, "### ") {
					break
				}
				descriptionBuilder.WriteString(trimmed)
				descriptionBuilder.WriteString(" ")
			}
			description := strings.TrimSpace(descriptionBuilder.String())
			assert.GreaterOrEqual(t, len(description), 50,
				"Scenario %s must have a description of at least 50 characters (got %d)",
				label, len(description))
		})
	}
}
