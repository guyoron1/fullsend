package qf_tests

/*
MCP Configuration Drift - Document Structure Tests

STP Reference: outputs/stp/GH-17/GH-17_test_plan.md
STD Reference: outputs/std/GH-17/GH-17_test_description.yaml
Jira: GH-17
*/

import (
	"bufio"
	"os"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const docPath = "docs/problems/mcp-config-drift.md"

// readDocContent is a shared helper that reads the problem document.
func readDocContent(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(docPath)
	require.NoError(t, err, "Failed to read %s", docPath)
	return string(data)
}

// TestDocumentContainsRequiredSections verifies that docs/problems/mcp-config-drift.md
// contains all mandatory sections expected in a FullSend problem document:
// a top-level heading, a problem statement section, defense considerations,
// and open questions.
//
// Test ID: TS-GH-17-001
// Priority: P0
func TestDocumentContainsRequiredSections(t *testing.T) {
	content := readDocContent(t)

	t.Run("has H1 heading", func(t *testing.T) {
		assert.True(t, strings.Contains(content, "# MCP Configuration Drift"),
			"Document must contain H1 heading '# MCP Configuration Drift'")
	})

	t.Run("has problem section", func(t *testing.T) {
		assert.True(t, strings.Contains(content, "## The problem"),
			"Document must contain '## The problem' section")
	})

	t.Run("has defense considerations section", func(t *testing.T) {
		assert.True(t, strings.Contains(content, "## Defense considerations"),
			"Document must contain '## Defense considerations' section")
	})

	t.Run("has open questions section", func(t *testing.T) {
		assert.True(t, strings.Contains(content, "## Open questions"),
			"Document must contain '## Open questions' section")
	})
}

// TestDocumentFollowsProblemDocStructure verifies the document follows the
// established problem document pattern: Related links, Attack scenarios
// subsection, and numbered defense approaches with trade-offs.
//
// Test ID: TS-GH-17-002
// Priority: P0
func TestDocumentFollowsProblemDocStructure(t *testing.T) {
	content := readDocContent(t)

	t.Run("has Related links block", func(t *testing.T) {
		assert.True(t, strings.Contains(content, "**Related:**"),
			"Document must contain a '**Related:**' links block")
	})

	t.Run("has Attack scenarios subsection", func(t *testing.T) {
		assert.True(t, strings.Contains(content, "### Attack scenarios"),
			"Document must contain '### Attack scenarios' subsection")
	})

	t.Run("has numbered defense approaches", func(t *testing.T) {
		approachRegex := regexp.MustCompile(`### Approach \d+:`)
		matches := approachRegex.FindAllString(content, -1)
		assert.GreaterOrEqual(t, len(matches), 2,
			"Document must contain at least 2 numbered '### Approach N:' headings (found %d)", len(matches))
	})
}

// TestDocumentNotEmptyOrMalformed is a negative test validating the document
// is not empty, is valid UTF-8, has minimum content length, and contains
// prose paragraphs beyond just headings.
//
// [NEGATIVE]
//
// Test ID: TS-GH-17-003
// Priority: P0
func TestDocumentNotEmptyOrMalformed(t *testing.T) {
	info, err := os.Stat(docPath)
	require.NoError(t, err, "Failed to stat %s", docPath)

	data, err := os.ReadFile(docPath)
	require.NoError(t, err, "Failed to read %s", docPath)

	t.Run("file size greater than 1000 bytes", func(t *testing.T) {
		assert.Greater(t, info.Size(), int64(1000),
			"Document must be greater than 1000 bytes (got %d)", info.Size())
	})

	t.Run("content is valid UTF-8", func(t *testing.T) {
		assert.True(t, utf8.Valid(data),
			"Document content must be valid UTF-8")
	})

	t.Run("document has prose content beyond headings", func(t *testing.T) {
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		proseLineCount := 0
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			// Skip empty lines and heading lines
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if len(line) > 20 {
				proseLineCount++
			}
		}
		assert.GreaterOrEqual(t, proseLineCount, 10,
			"Document must contain at least 10 non-heading prose lines with >20 characters (found %d)", proseLineCount)
	})
}
