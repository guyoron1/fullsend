package qf_tests

/*
MCP Configuration Drift - Defense Approach Quality Tests

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

// TestDefenseApproachesHaveTradeoffs validates that each numbered defense
// approach (### Approach N:) contains a **Trade-offs:** subsection.
//
// Test ID: TS-GH-17-014
// Priority: P2
func TestDefenseApproachesHaveTradeoffs(t *testing.T) {
	content := readDocContent(t)

	approachRegex := regexp.MustCompile(`### Approach \d+:`)
	approachHeaders := approachRegex.FindAllStringIndex(content, -1)
	require.NotEmpty(t, approachHeaders,
		"Document must contain at least one '### Approach N:' heading")

	for i, loc := range approachHeaders {
		// Extract the section from this approach header to the next one (or end)
		start := loc[0]
		end := len(content)
		if i+1 < len(approachHeaders) {
			end = approachHeaders[i+1][0]
		}
		section := content[start:end]

		// Extract the heading text for the test name
		headingEnd := strings.Index(section, "\n")
		if headingEnd < 0 {
			headingEnd = len(section)
		}
		heading := strings.TrimSpace(section[:headingEnd])

		t.Run(fmt.Sprintf("tradeoffs_in_%s", heading), func(t *testing.T) {
			assert.True(t, strings.Contains(section, "**Trade-offs"),
				"Defense approach section '%s' must contain a '**Trade-offs:**' subsection", heading)
		})
	}
}

// TestMultipleDefenseApproaches validates that the document presents at
// least 2 distinct defense approaches, ensuring the analysis covers
// multiple options.
//
// Test ID: TS-GH-17-015
// Priority: P2
func TestMultipleDefenseApproaches(t *testing.T) {
	content := readDocContent(t)

	approachRegex := regexp.MustCompile(`### Approach \d+:`)
	matches := approachRegex.FindAllString(content, -1)

	assert.GreaterOrEqual(t, len(matches), 2,
		"Document must present at least 2 distinct defense approaches (found %d)", len(matches))
}
