package qf_tests

/*
MCP Configuration Drift - CLAUDE.md Deletion Integrity Tests

STP Reference: outputs/stp/GH-17/GH-17_test_plan.md
STD Reference: outputs/std/GH-17/GH-17_test_description.yaml
Jira: GH-17
*/

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNoRemainingCLAUDEMDReferences is a negative test that scans all
// markdown and configuration files for references to the deleted CLAUDE.md
// file. References in .claude/ directory are excluded (internal config).
//
// [NEGATIVE]
//
// Test ID: TS-GH-17-016
// Priority: P1
func TestNoRemainingCLAUDEMDReferences(t *testing.T) {
	var staleRefs []string

	// File extensions to scan
	scanExtensions := map[string]bool{
		".md":   true,
		".yaml": true,
		".yml":  true,
		".json": true,
		".toml": true,
	}

	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Skip directories that should be excluded
		if info.IsDir() {
			name := info.Name()
			if name == ".git" || name == ".claude" || name == "node_modules" || name == ".fullsend" {
				return filepath.SkipDir
			}
			return nil
		}

		// Only scan relevant file types
		ext := filepath.Ext(path)
		if !scanExtensions[ext] {
			return nil
		}

		// Skip files in outputs/ directory (generated content may reference CLAUDE.md)
		if strings.HasPrefix(path, "outputs/") || strings.HasPrefix(path, "outputs"+string(filepath.Separator)) {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil // Skip unreadable files
		}

		content := string(data)
		// Look for references to CLAUDE.md as a link target or file reference
		// Match patterns like: (CLAUDE.md), "CLAUDE.md", `CLAUDE.md`, /CLAUDE.md
		if strings.Contains(content, "CLAUDE.md") {
			// Exclude lines that are about the deletion itself (e.g., changelog entries)
			lines := strings.Split(content, "\n")
			for lineNum, line := range lines {
				if strings.Contains(line, "CLAUDE.md") {
					// Check if this is a link target reference (not a mention of the concept)
					if strings.Contains(line, "(CLAUDE.md)") ||
						strings.Contains(line, "\"CLAUDE.md\"") ||
						strings.Contains(line, "`CLAUDE.md`") {
						staleRefs = append(staleRefs, path+":"+
							string(rune('0'+lineNum+1)))
					}
				}
			}
		}

		return nil
	})
	require.NoError(t, err, "Failed to walk repository file tree")

	assert.Empty(t, staleRefs,
		"No files should contain stale link/config references to deleted CLAUDE.md, but found references in: %v",
		staleRefs)
}

// TestRepositoryDocumentationIntegrity validates that CLAUDE.md does not
// exist at the repository root (confirming deletion) and that the core
// documentation structure remains intact.
//
// Test ID: TS-GH-17-017
// Priority: P1
func TestRepositoryDocumentationIntegrity(t *testing.T) {
	t.Run("CLAUDE.md does not exist at root", func(t *testing.T) {
		_, err := os.Stat("CLAUDE.md")
		assert.True(t, os.IsNotExist(err),
			"CLAUDE.md should not exist at repository root (deletion not carried out)")
	})

	t.Run("README.md exists at root", func(t *testing.T) {
		_, err := os.Stat("README.md")
		assert.NoError(t, err, "README.md must exist at repository root")
	})

	t.Run("docs directory exists", func(t *testing.T) {
		info, err := os.Stat("docs")
		assert.NoError(t, err, "docs/ directory must exist")
		if err == nil {
			assert.True(t, info.IsDir(), "docs must be a directory")
		}
	})
}
