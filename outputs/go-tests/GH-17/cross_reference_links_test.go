package qf_tests

/*
MCP Configuration Drift - Cross-Reference Link Integrity Tests

STP Reference: outputs/stp/GH-17/GH-17_test_plan.md
STD Reference: outputs/std/GH-17/GH-17_test_description.yaml
Jira: GH-17
*/

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

var linkRegex = regexp.MustCompile(`\]\(([^)]+)\)`)

// headingToSlug converts a markdown heading to a GitHub-compatible anchor slug.
// GitHub's algorithm: lowercase, strip non-alphanumeric (except hyphens and spaces),
// replace spaces with hyphens, collapse consecutive hyphens.
func headingToSlug(heading string) string {
	// Remove the leading # characters and trim
	heading = strings.TrimLeft(heading, "#")
	heading = strings.TrimSpace(heading)
	// Lowercase
	slug := strings.ToLower(heading)
	// Remove characters that are not alphanumeric, space, or hyphen
	var result strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == ' ' || r == '-' {
			result.WriteRune(r)
		}
	}
	// Replace spaces with hyphens
	slug = strings.ReplaceAll(result.String(), " ", "-")
	// Collapse consecutive hyphens
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	return slug
}

// extractHeadingSlugs reads a file and returns all heading slugs.
func extractHeadingSlugs(filePath string) ([]string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var slugs []string
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			slugs = append(slugs, headingToSlug(trimmed))
		}
	}
	return slugs, nil
}

// TestRelativeLinksResolve extracts all markdown relative links from
// mcp-config-drift.md and verifies each target file exists on disk.
// Links with anchor fragments are resolved to the file portion only.
//
// Test ID: TS-GH-17-004
// Priority: P0
func TestRelativeLinksResolve(t *testing.T) {
	content := readDocContent(t)
	docDir := filepath.Dir(docPath)

	matches := linkRegex.FindAllStringSubmatch(content, -1)
	require.NotEmpty(t, matches, "Document should contain at least one markdown link")

	relativeLinks := 0
	for _, match := range matches {
		target := match[1]
		// Skip external URLs
		if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
			continue
		}
		// Skip pure anchor links (same-document references)
		if strings.HasPrefix(target, "#") {
			continue
		}

		relativeLinks++

		// Strip anchor fragment to get file path
		filePart := strings.SplitN(target, "#", 2)[0]
		if filePart == "" {
			continue
		}

		resolvedPath := filepath.Join(docDir, filePart)
		t.Run(fmt.Sprintf("link_%s", filePart), func(t *testing.T) {
			_, err := os.Stat(resolvedPath)
			assert.NoError(t, err, "Relative link target %q should resolve to existing file at %s", target, resolvedPath)
		})
	}

	assert.Greater(t, relativeLinks, 0, "Document should contain at least one relative link")
}

// TestAnchorFragmentsReferenceValidHeadings verifies that for each markdown
// link containing an anchor fragment (e.g., file.md#heading-slug), the
// anchor corresponds to an actual heading in the target file.
//
// Test ID: TS-GH-17-005
// Priority: P0
func TestAnchorFragmentsReferenceValidHeadings(t *testing.T) {
	content := readDocContent(t)
	docDir := filepath.Dir(docPath)

	matches := linkRegex.FindAllStringSubmatch(content, -1)

	anchoredLinks := 0
	for _, match := range matches {
		target := match[1]
		// Skip external URLs and pure anchors
		if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
			continue
		}

		parts := strings.SplitN(target, "#", 2)
		if len(parts) != 2 || parts[1] == "" {
			continue // No anchor fragment
		}

		filePart := parts[0]
		anchor := parts[1]

		// For same-document anchors, use the doc itself
		targetFile := docPath
		if filePart != "" {
			targetFile = filepath.Join(docDir, filePart)
		}

		// Only check if the target file exists
		if _, err := os.Stat(targetFile); err != nil {
			continue // File doesn't exist — caught by TestRelativeLinksResolve
		}

		anchoredLinks++
		t.Run(fmt.Sprintf("anchor_%s#%s", filepath.Base(targetFile), anchor), func(t *testing.T) {
			slugs, err := extractHeadingSlugs(targetFile)
			require.NoError(t, err, "Failed to read target file %s for heading extraction", targetFile)

			found := false
			for _, slug := range slugs {
				if slug == anchor {
					found = true
					break
				}
			}
			assert.True(t, found,
				"Anchor #%s should match a heading in %s (available slugs: %v)",
				anchor, targetFile, slugs)
		})
	}

	// It's OK if there are no anchored links — some documents don't use them
	t.Logf("Checked %d anchored links", anchoredLinks)
}

// TestNoBrokenInternalLinks is a comprehensive negative test that scans
// the entire document for markdown link syntax and asserts zero broken
// links, combining file existence and anchor validation in a single pass.
//
// [NEGATIVE]
//
// Test ID: TS-GH-17-006
// Priority: P0
func TestNoBrokenInternalLinks(t *testing.T) {
	content := readDocContent(t)
	docDir := filepath.Dir(docPath)

	matches := linkRegex.FindAllStringSubmatch(content, -1)

	var brokenLinks []string

	for _, match := range matches {
		target := match[1]
		// Skip external URLs
		if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
			continue
		}

		parts := strings.SplitN(target, "#", 2)
		filePart := parts[0]
		anchor := ""
		if len(parts) == 2 {
			anchor = parts[1]
		}

		// For pure anchors, check within the same document
		targetFile := docPath
		if filePart != "" {
			targetFile = filepath.Join(docDir, filePart)
		}

		// Check file existence
		if _, err := os.Stat(targetFile); err != nil {
			if filePart != "" {
				brokenLinks = append(brokenLinks, fmt.Sprintf("file not found: %s", target))
			}
			continue
		}

		// Check anchor if present
		if anchor != "" {
			slugs, err := extractHeadingSlugs(targetFile)
			if err != nil {
				brokenLinks = append(brokenLinks, fmt.Sprintf("cannot read for anchor check: %s", target))
				continue
			}
			found := false
			for _, slug := range slugs {
				if slug == anchor {
					found = true
					break
				}
			}
			if !found {
				brokenLinks = append(brokenLinks, fmt.Sprintf("anchor not found: %s", target))
			}
		}
	}

	assert.Empty(t, brokenLinks,
		"Document should have zero broken internal links, but found: %v", brokenLinks)
}
