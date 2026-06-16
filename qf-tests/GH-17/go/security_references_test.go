package qf_tests

/*
MCP Configuration Drift - Security Component Reference Tests

STP Reference: outputs/stp/GH-17/GH-17_test_plan.md
STD Reference: outputs/std/GH-17/GH-17_test_description.yaml
Jira: GH-17
*/

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestToolAllowlistHookExists verifies that the tool_allowlist_pretool.py
// security hook referenced in the document exists at its expected path.
//
// Test ID: TS-GH-17-009
// Priority: P1
func TestToolAllowlistHookExists(t *testing.T) {
	hookPath := "internal/security/hooks/tool_allowlist_pretool.py"
	_, err := os.Stat(hookPath)
	assert.NoError(t, err,
		"Security hook %s referenced in the problem document must exist", hookPath)
}

// TestSSRFValidatorFilesExist verifies that the SSRF validator files
// referenced in the document exist at their expected paths.
//
// Test ID: TS-GH-17-010
// Priority: P1
func TestSSRFValidatorFilesExist(t *testing.T) {
	ssrfPaths := []string{
		"internal/security/hooks/ssrf_pretool.py",
		"internal/security/ssrf.go",
	}

	for _, path := range ssrfPaths {
		t.Run(fmt.Sprintf("exists_%s", path), func(t *testing.T) {
			_, err := os.Stat(path)
			assert.NoError(t, err,
				"SSRF validator file %s referenced in the problem document must exist", path)
		})
	}
}

// TestNoReferencesToNonExistentComponents is a negative test scanning the
// document for backtick-wrapped code/file references and verifying none
// point to non-existent paths in the repository.
//
// [NEGATIVE]
//
// Test ID: TS-GH-17-011
// Priority: P1
func TestNoReferencesToNonExistentComponents(t *testing.T) {
	content := readDocContent(t)

	// Match backtick-wrapped references that look like file paths
	// (contain a slash and a file extension)
	codeRefRegex := regexp.MustCompile("`([^`]*?/[^`]*?\\.(?:py|go|json|yaml|yml|toml|md))`")
	matches := codeRefRegex.FindAllStringSubmatch(content, -1)

	var missingRefs []string
	for _, match := range matches {
		ref := match[1]
		// Skip references that are clearly not local file paths
		if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
			continue
		}
		// Skip module/import paths (contain dots in directory components like github.com)
		if strings.Contains(ref, ".com/") || strings.Contains(ref, ".org/") {
			continue
		}

		_, err := os.Stat(ref)
		if err != nil {
			missingRefs = append(missingRefs, ref)
		}
	}

	assert.Empty(t, missingRefs,
		"All backtick-wrapped file references in the document should resolve to existing files, but these are missing: %v",
		missingRefs)
}

// Ensure imports are used.
var _ = require.NoError
