//go:build e2e

package tests

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// knownFullSendComponents is the canonical list of FullSend component names
// used by multiple test scenarios.
var knownFullSendComponents = []string{
	"harness",
	"skills",
	"dispatch",
	"pr-review agent",
	"code-review skill",
	"scaffold",
	"mint",
	"forge",
	"sandbox",
	"agent",
	"workflow",
	"plugin",
}

// existingCapabilities lists capabilities that FullSend already provides.
// Used by the negative-duplication test to verify recommendations propose
// enhancements rather than re-implementations.
var existingCapabilities = []struct {
	Name        string
	Description string
}{
	{"pr-review agent", "Automated PR review orchestration with sub-agent dispatch"},
	{"code-review skill", "Standalone code review procedure across six dimensions"},
	{"agent dispatch", "Agent lifecycle management via GitHub Actions"},
	{"harness configuration", "Claude Code harness with skills, hooks, and AGENTS.md"},
	{"scaffold", "Repository scaffolding and agent enrollment"},
	{"mint", "Token management for GitHub org/repo authentication"},
	{"forge", "Sandbox image building and caching"},
}

// insightPattern matches insight section headings in research summary documents.
// It matches lines like "## Insight ..." or numbered list items like "1. [Title]".
var insightPattern = regexp.MustCompile(`(?m)(?:^##\s+Insight|^\d+\.\s+\[)`)

// FileExists returns true if the file at path exists.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ReadFileContent reads a file and returns its content as a string.
// It fails the test immediately if the file cannot be read.
func ReadFileContent(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err, "failed to read file: %s", path)
	return string(data)
}

// CountMatches returns the number of matches of the given regex pattern in text.
func CountMatches(text string, pattern *regexp.Regexp) int {
	return len(pattern.FindAllString(text, -1))
}

// ContainsComponentReference checks whether text mentions any of the known
// FullSend components (case-insensitive).
func ContainsComponentReference(text string) []string {
	lower := strings.ToLower(text)
	var found []string
	for _, comp := range knownFullSendComponents {
		if strings.Contains(lower, strings.ToLower(comp)) {
			found = append(found, comp)
		}
	}
	return found
}

// ExtractInsightSections splits a research document into individual insight
// sections. It looks for "## Insight" headings or numbered list items with
// bracket titles, and returns each section's content.
func ExtractInsightSections(content string) []string {
	// Split on insight-like headings
	parts := insightPattern.Split(content, -1)
	var sections []string
	for i, part := range parts {
		if i == 0 {
			continue // preamble before first insight
		}
		trimmed := strings.TrimSpace(part)
		if len(trimmed) > 0 {
			sections = append(sections, trimmed)
		}
	}
	return sections
}

// meetsInsightThreshold returns true when the insight count meets or exceeds
// the required minimum.
func meetsInsightThreshold(count, threshold int) bool {
	return count >= threshold
}

// createTempDocWithInsights writes a temporary markdown file with the given
// number of insight sections and returns the file path.
func createTempDocWithInsights(t *testing.T, numInsights int) string {
	t.Helper()

	var sb strings.Builder
	sb.WriteString("# Research Summary: Are Code Reviews Dead?\n")
	sb.WriteString("## Source: latent.space article\n")
	sb.WriteString("## Applicable Insights:\n")

	for i := 1; i <= numInsights; i++ {
		sb.WriteString("\n## Insight " + strings.Repeat("I", i) + "\n")
		sb.WriteString("- Description: Finding number " + string(rune('0'+i)) + " from the article\n")
		sb.WriteString("- Applicability to FullSend: Applicable to harness\n")
	}

	tmpFile, err := os.CreateTemp(t.TempDir(), "research-*.md")
	require.NoError(t, err)
	_, err = tmpFile.WriteString(sb.String())
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())
	return tmpFile.Name()
}
