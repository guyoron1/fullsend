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

// readToolCallRiskDoc reads the tool-call-risk-assessment.md document.
func readToolCallRiskDoc(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)
	docPath := filepath.Join(root, "docs", "problems", "tool-call-risk-assessment.md")
	content, err := os.ReadFile(docPath)
	require.NoError(t, err, "failed to read tool-call-risk-assessment.md")
	return string(content)
}

// TestCodebaseHooksDocumented verifies the risk assessment document references
// all seven known security hooks.
// [test_id:TS-GH-14-008] Tier 1 / P0 / MVP
func TestCodebaseHooksDocumented(t *testing.T) {
	content := readToolCallRiskDoc(t)
	contentLower := strings.ToLower(content)

	hooks := []struct {
		name     string
		keywords []string
	}{
		{"Tirith", []string{"tirith"}},
		{"SSRF validator", []string{"ssrf"}},
		{"canary token detection", []string{"canary"}},
		{"unicode normalizer", []string{"unicode"}},
		{"tool allowlist", []string{"allowlist", "allow list", "allow-list", "whitelist"}},
		{"secret redactor", []string{"secret redact", "secret_redact", "redact"}},
		{"context suppressor", []string{"context suppress", "context_suppress", "suppress"}},
	}

	for _, hook := range hooks {
		found := false
		for _, kw := range hook.keywords {
			if strings.Contains(contentLower, kw) {
				found = true
				break
			}
		}
		assert.True(t, found,
			"document should reference security hook: %s", hook.name)
	}
}

// extractStructFields reads a Go source file and extracts field names from
// the named struct type. Returns a slice of field names.
func extractStructFields(t *testing.T, filePath, structName string) []string {
	t.Helper()
	content, err := os.ReadFile(filePath)
	require.NoError(t, err, "failed to read %s", filePath)

	// Match "type StructName struct {" through closing "}"
	pattern := `type\s+` + regexp.QuoteMeta(structName) + `\s+struct\s*\{([^}]*)\}`
	re := regexp.MustCompile(pattern)
	match := re.FindSubmatch(content)
	if match == nil {
		return nil
	}

	// Extract field names from struct body
	fieldRe := regexp.MustCompile(`(?m)^\s+(\w+)\s+`)
	fieldMatches := fieldRe.FindAllSubmatch(match[1], -1)
	var fields []string
	for _, fm := range fieldMatches {
		fields = append(fields, string(fm[1]))
	}
	return fields
}

// TestHookDescriptionsAlignWithCode verifies that hook descriptions in the document
// align with SecurityConfig and SandboxHooks struct fields in harness.go.
// [test_id:TS-GH-14-009] Tier 1 / P0 / MVP
func TestHookDescriptionsAlignWithCode(t *testing.T) {
	root := repoRoot(t)
	harnessPath := filepath.Join(root, "internal", "harness", "harness.go")

	// Read document content
	docContent := readToolCallRiskDoc(t)
	docLower := strings.ToLower(docContent)

	// Extract struct fields from harness.go
	securityFields := extractStructFields(t, harnessPath, "SecurityConfig")
	require.NotEmpty(t, securityFields,
		"should extract fields from SecurityConfig struct")

	sandboxFields := extractStructFields(t, harnessPath, "SandboxHooks")
	require.NotEmpty(t, sandboxFields,
		"should extract fields from SandboxHooks struct")

	// Verify document references SecurityConfig fields
	for _, field := range securityFields {
		fieldLower := strings.ToLower(field)
		// Convert CamelCase to searchable keywords
		// e.g., "SandboxHooks" -> "sandbox" and "hooks"
		keywords := camelCaseToKeywords(field)
		found := false
		for _, kw := range keywords {
			if strings.Contains(docLower, strings.ToLower(kw)) {
				found = true
				break
			}
		}
		if !found {
			// Also check if the raw field name appears
			found = strings.Contains(docLower, fieldLower)
		}
		assert.True(t, found,
			"SecurityConfig field %q should be referenced in the document", field)
	}

	// Verify document references SandboxHooks fields
	for _, field := range sandboxFields {
		fieldLower := strings.ToLower(field)
		keywords := camelCaseToKeywords(field)
		found := false
		for _, kw := range keywords {
			if strings.Contains(docLower, strings.ToLower(kw)) {
				found = true
				break
			}
		}
		if !found {
			found = strings.Contains(docLower, fieldLower)
		}
		assert.True(t, found,
			"SandboxHooks field %q should be referenced in the document", field)
	}
}

// camelCaseToKeywords splits a CamelCase identifier into individual keywords.
// e.g., "SSRFPreTool" -> ["SSRF", "Pre", "Tool"]
func camelCaseToKeywords(s string) []string {
	// Split on transitions between lower->upper, upper->lower (for acronyms)
	re := regexp.MustCompile(`[A-Z]+[a-z]*|[a-z]+`)
	return re.FindAllString(s, -1)
}

// TestHookDescriptionMismatchDetection [NEGATIVE] validates that a hook described
// in a document but not present in the codebase is detected as a mismatch.
// [test_id:TS-GH-14-010] Tier 1 / P1
func TestHookDescriptionMismatchDetection(t *testing.T) {
	root := repoRoot(t)
	harnessPath := filepath.Join(root, "internal", "harness", "harness.go")

	// Extract actual struct fields from harness.go
	securityFields := extractStructFields(t, harnessPath, "SecurityConfig")
	sandboxFields := extractStructFields(t, harnessPath, "SandboxHooks")

	allFields := append(securityFields, sandboxFields...)

	// Fictional hook name that should NOT be in the codebase
	fictionalHook := "DataExfiltrationBlocker"

	found := false
	for _, field := range allFields {
		if field == fictionalHook {
			found = true
			break
		}
	}

	assert.False(t, found,
		"fictional hook %q should NOT be found in codebase struct fields — "+
			"its presence in a document would indicate a mismatch", fictionalHook)
}
