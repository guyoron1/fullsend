//go:build e2e

package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// findReadmeOrDocsIndex locates the README.md or docs index file.
// Returns the path to the best available entry point.
func findReadmeOrDocsIndex(t *testing.T) (string, string) {
	t.Helper()
	root := repoRoot(t)

	// Try README.md first
	readmePath := filepath.Join(root, "README.md")
	if content, err := os.ReadFile(readmePath); err == nil {
		return readmePath, string(content)
	}

	// Try docs index
	docsIndex := filepath.Join(root, "docs", "README.md")
	if content, err := os.ReadFile(docsIndex); err == nil {
		return docsIndex, string(content)
	}

	t.Fatal("neither README.md nor docs/README.md found")
	return "", ""
}

// TestReadmeLinkTestingAgents verifies README contains a working link to
// testing-agents.md.
// [test_id:TS-GH-14-013] Tier 1 / P0 / MVP
func TestReadmeLinkTestingAgents(t *testing.T) {
	root := repoRoot(t)
	readmePath, content := findReadmeOrDocsIndex(t)

	// Search for link to testing-agents.md
	hasLink := strings.Contains(content, "testing-agents.md")
	assert.True(t, hasLink,
		"README should contain a link to testing-agents.md")

	if !hasLink {
		return
	}

	// Extract the actual link paths and verify the target exists
	links := extractRelativeMDLinks(content)
	readmeDir := filepath.Dir(readmePath)
	foundAndValid := false
	for _, link := range links {
		if strings.Contains(link, "testing-agents.md") {
			targetPath := filepath.Join(readmeDir, link)
			// Also try from repo root if link is repo-relative
			if _, err := os.Stat(targetPath); err == nil {
				foundAndValid = true
				break
			}
			targetPath = filepath.Join(root, link)
			if _, err := os.Stat(targetPath); err == nil {
				foundAndValid = true
				break
			}
		}
	}

	assert.True(t, foundAndValid,
		"link to testing-agents.md should resolve to an existing file")
}

// TestReadmeLinkToolCallRiskAssessment verifies README contains a working link
// to tool-call-risk-assessment.md.
// [test_id:TS-GH-14-014] Tier 1 / P0 / MVP
func TestReadmeLinkToolCallRiskAssessment(t *testing.T) {
	root := repoRoot(t)
	readmePath, content := findReadmeOrDocsIndex(t)

	// Search for link to tool-call-risk-assessment.md
	hasLink := strings.Contains(content, "tool-call-risk-assessment.md")
	assert.True(t, hasLink,
		"README should contain a link to tool-call-risk-assessment.md")

	if !hasLink {
		return
	}

	// Extract and verify link target exists
	links := extractRelativeMDLinks(content)
	readmeDir := filepath.Dir(readmePath)
	foundAndValid := false
	for _, link := range links {
		if strings.Contains(link, "tool-call-risk-assessment.md") {
			targetPath := filepath.Join(readmeDir, link)
			if _, err := os.Stat(targetPath); err == nil {
				foundAndValid = true
				break
			}
			targetPath = filepath.Join(root, link)
			if _, err := os.Stat(targetPath); err == nil {
				foundAndValid = true
				break
			}
		}
	}

	assert.True(t, foundAndValid,
		"link to tool-call-risk-assessment.md should resolve to an existing file")
}

// TestBrokenReadmeLinkDetection [NEGATIVE] verifies that a broken README link
// to a non-existent document is detected.
// [test_id:TS-GH-14-015] Tier 1 / P0 / MVP
func TestBrokenReadmeLinkDetection(t *testing.T) {
	root := repoRoot(t)

	// Simulate README content with a broken link
	content := "# README\nSee [Problem Doc](docs/problems/non-existent-problem.md) for details."

	links := extractRelativeMDLinks(content)
	require.NotEmpty(t, links, "should extract at least one link from test content")

	brokenLinks := []string{}
	for _, link := range links {
		// Resolve against repo root
		resolvedPath := filepath.Join(root, link)
		if _, err := os.Stat(resolvedPath); os.IsNotExist(err) {
			brokenLinks = append(brokenLinks, link)
		}
	}

	require.NotEmpty(t, brokenLinks,
		"should detect at least one broken link in simulated README")
	assert.Contains(t, brokenLinks[0], "non-existent-problem.md",
		"broken link to non-existent-problem.md should be reported")
}
