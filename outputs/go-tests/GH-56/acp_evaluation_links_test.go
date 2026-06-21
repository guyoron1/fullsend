//go:build e2e

package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLandscapeToDetailCrossLink validates that the cross-link from the ACP
// entry in landscape.md to the detailed analysis resolves correctly.
// [test_id:TS-GH-56-004]
func TestLandscapeToDetailCrossLink(t *testing.T) {
	landscapePath := "docs/landscape.md"

	// Setup: read docs/landscape.md
	landscapeBytes, err := os.ReadFile(landscapePath)
	require.NoError(t, err, "failed to read docs/landscape.md")
	landscapeContent := string(landscapeBytes)

	t.Run("should have landscape-to-detail cross-link that resolves", func(t *testing.T) {
		// TEST-01: Extract cross-links from landscape ACP entry
		// Look for markdown links matching agent-infrastructure pattern
		linkPattern := regexp.MustCompile(`\[([^\]]+)\]\(([^)]*agent-infrastructure[^)]*)\)`)
		matches := linkPattern.FindAllStringSubmatch(landscapeContent, -1)
		require.NotEmpty(t, matches,
			"landscape.md must contain at least one link to agent-infrastructure.md")

		for _, match := range matches {
			linkPath := match[2]

			// Strip any anchor fragment for file existence check
			filePath := linkPath
			if idx := strings.Index(filePath, "#"); idx != -1 {
				filePath = filePath[:idx]
			}

			// TEST-02: Resolve relative link path from landscape.md location
			resolvedPath := filepath.Join(filepath.Dir(landscapePath), filePath)

			// TEST-03: Verify target file exists
			_, err := os.Stat(resolvedPath)
			assert.NoError(t, err,
				"Cross-link target %q (resolved to %q) must exist", linkPath, resolvedPath)
		}
	})
}

// TestAnchorTargetExists validates that cross-link anchor fragments
// map to valid headings in the destination document.
// [test_id:TS-GH-56-005]
func TestAnchorTargetExists(t *testing.T) {
	landscapePath := "docs/landscape.md"

	// Setup: read both documentation files
	landscapeBytes, err := os.ReadFile(landscapePath)
	require.NoError(t, err, "failed to read docs/landscape.md")
	landscapeContent := string(landscapeBytes)

	detailBytes, err := os.ReadFile("docs/problems/agent-infrastructure.md")
	require.NoError(t, err, "failed to read docs/problems/agent-infrastructure.md")
	detailContent := string(detailBytes)

	t.Run("should have anchor target that exists in destination doc", func(t *testing.T) {
		// TEST-01: Extract anchor fragment from cross-links
		linkPattern := regexp.MustCompile(`\[([^\]]+)\]\(([^)]*agent-infrastructure[^)]*)\)`)
		matches := linkPattern.FindAllStringSubmatch(landscapeContent, -1)
		require.NotEmpty(t, matches, "landscape.md must contain links to agent-infrastructure.md")

		for _, match := range matches {
			linkPath := match[2]
			anchorIdx := strings.Index(linkPath, "#")
			if anchorIdx == -1 {
				// No anchor in this link, skip anchor validation
				continue
			}
			anchor := linkPath[anchorIdx+1:]

			// TEST-02: Extract all headings from destination document
			headingPattern := regexp.MustCompile(`(?m)^#{1,6}\s+(.+)$`)
			headings := headingPattern.FindAllStringSubmatch(detailContent, -1)
			require.NotEmpty(t, headings, "destination document must contain headings")

			// TEST-03: Convert headings to GitHub-style slugs
			slugs := make([]string, 0, len(headings))
			for _, h := range headings {
				slug := markdownSlug(h[1])
				slugs = append(slugs, slug)
			}

			// TEST-04: Check anchor exists in slug list
			assert.Contains(t, slugs, anchor,
				"Anchor %q must map to a valid heading in agent-infrastructure.md. Available slugs: %v",
				anchor, slugs)
		}
	})
}

// TestBrokenAnchorDetection validates that broken anchors are detected
// and reported with clear, actionable error messages.
// [test_id:TS-GH-56-006]
func TestBrokenAnchorDetection(t *testing.T) {
	// Setup: read agent-infrastructure.md to get valid headings
	detailBytes, err := os.ReadFile("docs/problems/agent-infrastructure.md")
	require.NoError(t, err, "failed to read docs/problems/agent-infrastructure.md")
	detailContent := string(detailBytes)

	t.Run("should detect and report broken anchors clearly", func(t *testing.T) {
		// TEST-01: Create test case with intentionally broken anchor
		brokenAnchor := "non-existent-section-that-should-not-exist"

		// TEST-02: Run anchor validation on broken link
		headingPattern := regexp.MustCompile(`(?m)^#{1,6}\s+(.+)$`)
		headings := headingPattern.FindAllStringSubmatch(detailContent, -1)
		require.NotEmpty(t, headings, "destination document must contain headings")

		slugs := make([]string, 0, len(headings))
		for _, h := range headings {
			slug := markdownSlug(h[1])
			slugs = append(slugs, slug)
		}

		found, errMsg := validateAnchor(brokenAnchor, slugs)

		// TEST-03: Verify broken anchor is detected
		assert.False(t, found,
			"Validation must detect broken anchor %q", brokenAnchor)

		// Verify error message is actionable
		assert.NotEmpty(t, errMsg,
			"Error message must not be empty for broken anchor")
		assert.Contains(t, errMsg, brokenAnchor,
			"Error message must identify the specific broken anchor")
	})
}

// validateAnchor checks whether an anchor exists in the list of heading slugs.
// Returns (found, errorMessage). If not found, errorMessage contains the broken
// anchor name and the available slugs for remediation.
func validateAnchor(anchor string, slugs []string) (bool, string) {
	for _, slug := range slugs {
		if slug == anchor {
			return true, ""
		}
	}
	return false, fmt.Sprintf(
		"broken anchor %q: no matching heading found. Available anchors: %v",
		anchor, slugs)
}

// markdownSlug converts a markdown heading text to a GitHub-style anchor slug.
// It lowercases the text, replaces spaces with hyphens, and strips special characters.
func markdownSlug(heading string) string {
	slug := strings.ToLower(strings.TrimSpace(heading))
	// Remove characters that are not alphanumeric, spaces, or hyphens
	re := regexp.MustCompile(`[^\w\s-]`)
	slug = re.ReplaceAllString(slug, "")
	// Replace spaces with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")
	// Collapse multiple hyphens
	re = regexp.MustCompile(`-+`)
	slug = re.ReplaceAllString(slug, "-")
	return slug
}
