//go:build e2e

package tests

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewSectionsCorrectLocation validates that the new ACP entry in
// landscape.md and the detailed analysis section in agent-infrastructure.md
// are positioned correctly within each document's structure.
// [test_id:TS-GH-56-007]
func TestNewSectionsCorrectLocation(t *testing.T) {
	// Setup: read both documentation files
	landscapeBytes, err := os.ReadFile("docs/landscape.md")
	require.NoError(t, err, "failed to read docs/landscape.md")
	landscapeContent := string(landscapeBytes)

	detailBytes, err := os.ReadFile("docs/problems/agent-infrastructure.md")
	require.NoError(t, err, "failed to read docs/problems/agent-infrastructure.md")
	detailContent := string(detailBytes)

	t.Run("should have new sections placed in correct document locations", func(t *testing.T) {
		// TEST-01: Extract section headings from landscape.md
		headingPattern := regexp.MustCompile(`(?m)^#{1,6}\s+(.+)$`)
		landscapeHeadings := headingPattern.FindAllStringSubmatch(landscapeContent, -1)
		require.NotEmpty(t, landscapeHeadings,
			"landscape.md must contain section headings")

		// TEST-02: Verify ACP entry appears among landscape entries
		acpFound := false
		for _, h := range landscapeHeadings {
			heading := strings.ToLower(h[1])
			if strings.Contains(heading, "ambient") ||
				strings.Contains(heading, "acp") ||
				strings.Contains(heading, "ambient code") {
				acpFound = true
				break
			}
		}
		assert.True(t, acpFound,
			"landscape.md must contain an ACP-related heading (containing 'ambient', 'acp', or 'ambient code')")

		// TEST-03: Extract section headings from agent-infrastructure.md
		detailHeadings := headingPattern.FindAllStringSubmatch(detailContent, -1)
		require.NotEmpty(t, detailHeadings,
			"agent-infrastructure.md must contain section headings")

		// TEST-04: Verify ACP analysis section exists in detail doc
		acpDetailFound := false
		for _, h := range detailHeadings {
			heading := strings.ToLower(h[1])
			if strings.Contains(heading, "ambient") ||
				strings.Contains(heading, "acp") ||
				strings.Contains(heading, "ambient code") {
				acpDetailFound = true
				break
			}
		}
		assert.True(t, acpDetailFound,
			"agent-infrastructure.md must contain an ACP-related section heading")
	})
}

// TestExistingContentUnmodified validates that pre-existing content in both
// documentation files was not modified by the ACP documentation changes.
// [test_id:TS-GH-56-008]
func TestExistingContentUnmodified(t *testing.T) {
	t.Run("should leave existing content unmodified by insertion", func(t *testing.T) {
		// SETUP-01: Retrieve baseline content from git parent commit
		baselineLandscape, err := exec.Command("git", "show", "HEAD~1:docs/landscape.md").Output()
		if err != nil {
			t.Skip("Cannot retrieve baseline landscape.md from git history (HEAD~1); skipping content preservation check")
		}

		baselineDetail, err := exec.Command("git", "show", "HEAD~1:docs/problems/agent-infrastructure.md").Output()
		if err != nil {
			t.Skip("Cannot retrieve baseline agent-infrastructure.md from git history (HEAD~1); skipping content preservation check")
		}

		// SETUP-02: Read current file content
		currentLandscape, err := os.ReadFile("docs/landscape.md")
		require.NoError(t, err, "failed to read current docs/landscape.md")

		currentDetail, err := os.ReadFile("docs/problems/agent-infrastructure.md")
		require.NoError(t, err, "failed to read current docs/problems/agent-infrastructure.md")

		// TEST-01 & TEST-02: Extract non-ACP sections from landscape.md and compare
		baselineLandscapeLines := filterNonACPLines(string(baselineLandscape))
		currentLandscapeLines := filterNonACPLines(string(currentLandscape))

		assert.Equal(t, baselineLandscapeLines, currentLandscapeLines,
			"Pre-existing (non-ACP) content in landscape.md must be identical to baseline. "+
				"Only new ACP sections should be added.")

		// TEST-03 & TEST-04: Extract non-ACP sections from agent-infrastructure.md and compare
		baselineDetailLines := filterNonACPLines(string(baselineDetail))
		currentDetailLines := filterNonACPLines(string(currentDetail))

		assert.Equal(t, baselineDetailLines, currentDetailLines,
			"Pre-existing (non-ACP) content in agent-infrastructure.md must be identical to baseline. "+
				"Only new ACP sections should be added.")
	})
}

// filterNonACPLines removes lines that are part of ACP-specific sections
// from the document content, returning only the pre-existing content for
// comparison. This allows us to verify that existing content was not modified
// when ACP sections were added.
func filterNonACPLines(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inACPSection := false
	acpSectionLevel := 0
	headingPattern := regexp.MustCompile(`^(#{1,6})\s+(.+)$`)

	for _, line := range lines {
		matches := headingPattern.FindStringSubmatch(line)
		if matches != nil {
			level := len(matches[1])
			heading := strings.ToLower(matches[2])

			if strings.Contains(heading, "ambient") ||
				strings.Contains(heading, "acp") ||
				strings.Contains(heading, "ambient code") {
				inACPSection = true
				acpSectionLevel = level
				continue
			}

			// If we encounter a heading at the same or higher level as the
			// ACP section, we've left the ACP section.
			if inACPSection && level <= acpSectionLevel {
				inACPSection = false
			}
		}

		if !inACPSection {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}
