package qf_tests

/*
MCP Configuration Drift - README Index Tests

STP Reference: outputs/stp/GH-17/GH-17_test_plan.md
STD Reference: outputs/std/GH-17/GH-17_test_description.yaml
Jira: GH-17
*/

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const readmePath = "README.md"

// TestREADMEContainsLinkToMCPConfigDrift verifies that README.md contains
// a markdown link pointing to docs/problems/mcp-config-drift.md or a
// relative path that resolves to it.
//
// Test ID: TS-GH-17-007
// Priority: P0
func TestREADMEContainsLinkToMCPConfigDrift(t *testing.T) {
	data, err := os.ReadFile(readmePath)
	require.NoError(t, err, "Failed to read %s", readmePath)

	readmeContent := string(data)

	assert.True(t, strings.Contains(readmeContent, "mcp-config-drift"),
		"README.md must contain a reference to 'mcp-config-drift'")
}

// TestREADMELinkTargetExists extracts the link target for the MCP config
// drift entry from README.md and verifies the target file exists on disk.
//
// Test ID: TS-GH-17-008
// Priority: P0
func TestREADMELinkTargetExists(t *testing.T) {
	data, err := os.ReadFile(readmePath)
	require.NoError(t, err, "Failed to read %s", readmePath)

	readmeContent := string(data)

	// Find markdown links containing "mcp-config-drift"
	re := regexp.MustCompile(`\]\(([^)]*mcp-config-drift[^)]*)\)`)
	matches := re.FindAllStringSubmatch(readmeContent, -1)
	require.NotEmpty(t, matches,
		"README.md must contain a markdown link referencing mcp-config-drift")

	for _, match := range matches {
		linkTarget := match[1]
		// Strip anchor fragment
		filePart := strings.SplitN(linkTarget, "#", 2)[0]
		if filePart == "" {
			continue
		}

		// Skip external URLs
		if strings.HasPrefix(filePart, "http://") || strings.HasPrefix(filePart, "https://") {
			continue
		}

		resolvedPath := filepath.Clean(filePart)
		t.Run("link_target_exists_"+filepath.Base(resolvedPath), func(t *testing.T) {
			_, err := os.Stat(resolvedPath)
			assert.NoError(t, err,
				"README link target %q should exist on disk at %s", linkTarget, resolvedPath)
		})
	}
}
