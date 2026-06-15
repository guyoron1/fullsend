//go:build e2e

package tests

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mdLinkRegex matches markdown links of the form [text](path).
var mdLinkRegex = regexp.MustCompile(`\[.*?\]\(([^)]+)\)`)

// extractRelativeMDLinks extracts relative markdown file links from content.
// It filters out URLs (http/https), anchors (#), and non-.md links.
func extractRelativeMDLinks(content string) []string {
	matches := mdLinkRegex.FindAllStringSubmatch(content, -1)
	var links []string
	for _, match := range matches {
		link := match[1]
		// Skip external URLs
		if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
			continue
		}
		// Skip pure anchors
		if strings.HasPrefix(link, "#") {
			continue
		}
		// Strip anchor from link if present
		if idx := strings.Index(link, "#"); idx >= 0 {
			link = link[:idx]
		}
		if link == "" {
			continue
		}
		// Only include .md links
		if strings.HasSuffix(link, ".md") {
			links = append(links, link)
		}
	}
	return links
}

// TestInternalLinksResolve verifies all internal markdown links in testing-agents.md
// resolve to existing files.
// [test_id:TS-GH-14-004] Tier 1 / P0 / MVP
func TestInternalLinksResolve(t *testing.T) {
	root := repoRoot(t)
	docPath := filepath.Join(root, "docs", "problems", "testing-agents.md")

	content, err := os.ReadFile(docPath)
	require.NoError(t, err, "failed to read testing-agents.md")

	docDir := filepath.Dir(docPath)
	links := extractRelativeMDLinks(string(content))

	if len(links) == 0 {
		t.Log("No internal markdown links found in testing-agents.md — skipping resolution check")
		return
	}

	for _, link := range links {
		resolvedPath := filepath.Join(docDir, link)
		_, err := os.Stat(resolvedPath)
		assert.NoError(t, err, "internal link %q should resolve to an existing file at %s", link, resolvedPath)
	}
}

// TestBrokenCrossReferenceDetection [NEGATIVE] validates that a broken cross-reference
// link pointing to a non-existent file is detected and reported.
// [test_id:TS-GH-14-005] Tier 1 / P1
func TestBrokenCrossReferenceDetection(t *testing.T) {
	root := repoRoot(t)
	content := "See [details](non-existent-file.md) for more info."

	links := extractRelativeMDLinks(content)
	require.NotEmpty(t, links, "should extract at least one link from test content")

	brokenLinks := []string{}
	for _, link := range links {
		// Resolve against the repo root (simulating a document in the root)
		resolvedPath := filepath.Join(root, link)
		if _, err := os.Stat(resolvedPath); os.IsNotExist(err) {
			brokenLinks = append(brokenLinks, link)
		}
	}

	require.NotEmpty(t, brokenLinks, "should detect at least one broken link")
	assert.Contains(t, brokenLinks, "non-existent-file.md",
		"broken link to non-existent-file.md should be reported")
}
