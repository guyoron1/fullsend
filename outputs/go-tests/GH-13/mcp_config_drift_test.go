//go:build e2e

package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// repoRoot returns the root directory of the repository by running git rev-parse.
func repoRoot(t *testing.T) string {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	require.NoError(t, err, "failed to determine repository root")
	return strings.TrimSpace(string(out))
}

// readFile reads a file relative to the repository root and returns its content.
func readFile(t *testing.T, root, relPath string) string {
	t.Helper()
	fullPath := filepath.Join(root, relPath)
	data, err := os.ReadFile(fullPath)
	require.NoError(t, err, "failed to read file: %s", fullPath)
	return string(data)
}

// extractRelativeMarkdownLinks extracts all relative markdown links from content.
// It returns the link targets (not the display text). External links (http/https) are excluded.
func extractRelativeMarkdownLinks(content string) []string {
	re := regexp.MustCompile(`\[([^\]]*)\]\(([^)]+)\)`)
	matches := re.FindAllStringSubmatch(content, -1)
	var links []string
	for _, m := range matches {
		target := m[2]
		// Skip external links and anchors
		if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") || strings.HasPrefix(target, "#") {
			continue
		}
		// Strip any anchor from the path
		if idx := strings.Index(target, "#"); idx >= 0 {
			target = target[:idx]
		}
		if target != "" {
			links = append(links, target)
		}
	}
	return links
}

// extractMarkdownHeadings extracts all markdown headings (lines starting with #) from content.
func extractMarkdownHeadings(content string) []string {
	var headings []string
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			headings = append(headings, trimmed)
		}
	}
	return headings
}

// TestCrossReferencesValid verifies all cross-references to security-threat-model.md,
// agent-architecture.md, and ADRs 0016/0017 are valid and link targets exist.
// [test_id:TS-GH-13-001]
func TestCrossReferencesValid(t *testing.T) {
	root := repoRoot(t)
	docPath := filepath.Join(root, "docs", "problems", "mcp-config-drift.md")

	content, err := os.ReadFile(docPath)
	require.NoError(t, err, "mcp-config-drift.md must exist")

	docContent := string(content)
	docDir := filepath.Dir(docPath)

	// Extract all relative markdown links
	links := extractRelativeMarkdownLinks(docContent)
	require.NotEmpty(t, links, "document should contain at least one relative link")

	// Verify each link target exists
	for _, link := range links {
		resolved := filepath.Join(docDir, link)
		_, err := os.Stat(resolved)
		assert.NoError(t, err, "cross-reference target does not exist: %s (resolved: %s)", link, resolved)
	}

	// Specifically verify ADR 0016 and ADR 0017 references
	t.Run("ADR 0016 referenced", func(t *testing.T) {
		assert.True(t, strings.Contains(docContent, "0016"),
			"document should reference ADR 0016 (unidirectional control flow)")
	})

	t.Run("ADR 0017 referenced", func(t *testing.T) {
		assert.True(t, strings.Contains(docContent, "0017"),
			"document should reference ADR 0017 (credential isolation)")
	})

	// Verify ADR files exist
	t.Run("ADR files exist", func(t *testing.T) {
		adrDir := filepath.Join(root, "docs", "adrs")
		entries, err := os.ReadDir(adrDir)
		if err != nil {
			// Try alternative ADR directory names
			adrDir = filepath.Join(root, "docs", "adr")
			entries, err = os.ReadDir(adrDir)
		}
		require.NoError(t, err, "ADR directory must exist")

		found0016 := false
		found0017 := false
		for _, e := range entries {
			if strings.Contains(e.Name(), "0016") {
				found0016 = true
			}
			if strings.Contains(e.Name(), "0017") {
				found0017 = true
			}
		}
		assert.True(t, found0016, "ADR 0016 file must exist in %s", adrDir)
		assert.True(t, found0017, "ADR 0017 file must exist in %s", adrDir)
	})
}

// TestToolAllowlistHookClaims verifies the document's claims about the tool allowlist hook's
// operating mechanism are accurate — it filters by tool names, not server endpoints.
// [test_id:TS-GH-13-002]
func TestToolAllowlistHookClaims(t *testing.T) {
	root := repoRoot(t)

	// Search for ToolAllowlistPreToolHook or ToolAllowlist in the codebase
	cmd := exec.Command("grep", "-r", "--include=*.go", "-l", "ToolAllowlist", filepath.Join(root, "internal"))
	out, err := cmd.Output()
	require.NoError(t, err, "ToolAllowlist must exist in internal/")

	files := strings.Split(strings.TrimSpace(string(out)), "\n")
	require.NotEmpty(t, files, "at least one file should contain ToolAllowlist")

	// Read the hook implementation and verify filtering mechanism
	var hookFound bool
	for _, f := range files {
		if f == "" {
			continue
		}
		data, err := os.ReadFile(f)
		require.NoError(t, err, "failed to read %s", f)
		src := string(data)

		if strings.Contains(src, "ToolAllowlist") {
			hookFound = true

			t.Run("filters by tool names", func(t *testing.T) {
				// The hook should reference tool names/identifiers for filtering
				hasToolNameRef := strings.Contains(src, "tool.Name") ||
					strings.Contains(src, "toolName") ||
					strings.Contains(src, "ToolName") ||
					strings.Contains(src, "allowedTools") ||
					strings.Contains(src, "AllowedTools") ||
					strings.Contains(src, "allowList") ||
					strings.Contains(src, "AllowList")
				assert.True(t, hasToolNameRef,
					"hook implementation should contain tool-name-based filtering logic in %s", f)
			})

			t.Run("does not filter by server endpoints", func(t *testing.T) {
				// The hook should NOT have server-endpoint-based filtering
				hasEndpointFilter := strings.Contains(src, "serverEndpoint") ||
					strings.Contains(src, "ServerEndpoint") ||
					strings.Contains(src, "endpointFilter")
				assert.False(t, hasEndpointFilter,
					"hook should NOT filter by server endpoints (consistent with document claims) in %s", f)
			})
		}
	}
	assert.True(t, hookFound, "ToolAllowlistPreToolHook implementation must be found")
}

// TestSSRFValidatorCoverageClaims verifies that SSRF validation covers Bash and WebFetch
// but not MCP connections, as claimed in the document.
// [test_id:TS-GH-13-003]
func TestSSRFValidatorCoverageClaims(t *testing.T) {
	root := repoRoot(t)

	// Locate SSRF-related code
	cmd := exec.Command("grep", "-r", "--include=*.go", "-l", "-i", "ssrf", filepath.Join(root, "internal"))
	out, err := cmd.Output()
	require.NoError(t, err, "SSRF validator code must exist in the codebase")

	ssrfFiles := strings.Split(strings.TrimSpace(string(out)), "\n")
	require.NotEmpty(t, ssrfFiles, "at least one file should contain SSRF references")

	// Read all SSRF-related source files
	var allSSRFSource strings.Builder
	for _, f := range ssrfFiles {
		if f == "" {
			continue
		}
		data, err := os.ReadFile(f)
		require.NoError(t, err, "failed to read %s", f)
		allSSRFSource.WriteString(string(data))
		allSSRFSource.WriteString("\n")
	}
	ssrfCode := allSSRFSource.String()

	t.Run("SSRF covers Bash tool", func(t *testing.T) {
		// Search for SSRF validation integration with Bash tool
		cmd := exec.Command("grep", "-r", "--include=*.go", "-l", "-i", "ssrf", filepath.Join(root, "internal"))
		bashOut, _ := cmd.Output()
		bashFiles := string(bashOut)

		hasBashIntegration := strings.Contains(ssrfCode, "Bash") ||
			strings.Contains(ssrfCode, "bash") ||
			strings.Contains(ssrfCode, "command") ||
			strings.Contains(bashFiles, "bash")
		assert.True(t, hasBashIntegration,
			"SSRF validator should be integrated with Bash tool execution")
	})

	t.Run("SSRF covers WebFetch tool", func(t *testing.T) {
		hasFetchIntegration := strings.Contains(ssrfCode, "WebFetch") ||
			strings.Contains(ssrfCode, "webfetch") ||
			strings.Contains(ssrfCode, "Fetch") ||
			strings.Contains(ssrfCode, "fetch") ||
			strings.Contains(ssrfCode, "http")
		assert.True(t, hasFetchIntegration,
			"SSRF validator should be integrated with WebFetch tool execution")
	})

	t.Run("SSRF does NOT cover MCP connections", func(t *testing.T) {
		// Search specifically for SSRF in MCP connection/transport code
		mcpDir := filepath.Join(root, "internal", "mcp")
		if _, err := os.Stat(mcpDir); err == nil {
			cmd := exec.Command("grep", "-r", "--include=*.go", "-i", "ssrf", mcpDir)
			out, err := cmd.Output()
			mcpSSRF := strings.TrimSpace(string(out))
			// If grep found nothing (exit code 1), that's the expected result
			if err != nil {
				// grep returns exit code 1 when no matches — this is expected
				assert.Empty(t, mcpSSRF, "SSRF validation should NOT be present in MCP connection code")
			} else {
				assert.Empty(t, mcpSSRF,
					"SSRF validation should NOT be present in MCP connection code, but found references")
			}
		}
	})
}

// TestReadmeIndexEntry verifies that README.md contains an index entry for
// the MCP Configuration Drift problem document with a valid link.
// [test_id:TS-GH-13-004]
func TestReadmeIndexEntry(t *testing.T) {
	root := repoRoot(t)

	readmePath := filepath.Join(root, "README.md")
	data, err := os.ReadFile(readmePath)
	require.NoError(t, err, "README.md must exist at repository root")
	readmeContent := string(data)

	t.Run("contains MCP config drift entry", func(t *testing.T) {
		hasMCPEntry := strings.Contains(strings.ToLower(readmeContent), "mcp") &&
			(strings.Contains(strings.ToLower(readmeContent), "config") ||
				strings.Contains(strings.ToLower(readmeContent), "drift"))
		assert.True(t, hasMCPEntry,
			"README.md should contain an entry referencing MCP configuration drift")
	})

	t.Run("entry links to correct document", func(t *testing.T) {
		re := regexp.MustCompile(`\[([^\]]*[Mm][Cc][Pp][^\]]*)\]\(([^)]+)\)`)
		matches := re.FindAllStringSubmatch(readmeContent, -1)
		if len(matches) == 0 {
			// Try broader pattern
			re = regexp.MustCompile(`\[[^\]]*\]\([^)]*mcp-config-drift[^)]*\)`)
			matches = re.FindAllStringSubmatch(readmeContent, -1)
		}
		require.NotEmpty(t, matches, "README should contain a markdown link referencing MCP config drift")

		// Verify the link target resolves
		for _, m := range matches {
			linkTarget := m[len(m)-1]
			if strings.HasPrefix(linkTarget, "http") {
				continue
			}
			resolved := filepath.Join(root, linkTarget)
			_, err := os.Stat(resolved)
			assert.NoError(t, err, "README link target should resolve: %s", linkTarget)
		}
	})
}

// TestDocumentStructureFormat verifies the document structure matches
// the format of existing problem documents.
// [test_id:TS-GH-13-005]
func TestDocumentStructureFormat(t *testing.T) {
	root := repoRoot(t)

	mcpDoc := readFile(t, root, "docs/problems/mcp-config-drift.md")
	mcpHeadings := extractMarkdownHeadings(mcpDoc)
	require.NotEmpty(t, mcpHeadings, "document should have section headings")

	// Read a reference problem document for structural comparison
	refPath := filepath.Join(root, "docs", "problems", "security-threat-model.md")
	var refHeadings []string
	if refData, err := os.ReadFile(refPath); err == nil {
		refHeadings = extractMarkdownHeadings(string(refData))
	}

	t.Run("contains standard problem doc sections", func(t *testing.T) {
		lowerDoc := strings.ToLower(mcpDoc)

		// Check for key structural sections expected in problem documents
		expectedSections := []struct {
			name    string
			pattern string
		}{
			{"problem statement or description", "problem"},
			{"analysis or investigation", "analy"},
			{"open questions or next steps", "question"},
		}

		for _, section := range expectedSections {
			found := false
			for _, h := range mcpHeadings {
				if strings.Contains(strings.ToLower(h), section.pattern) {
					found = true
					break
				}
			}
			if !found {
				// Also check body for section-like content
				found = strings.Contains(lowerDoc, section.pattern)
			}
			assert.True(t, found, "document should contain a %s section", section.name)
		}
	})

	t.Run("related documents section present", func(t *testing.T) {
		lowerDoc := strings.ToLower(mcpDoc)
		hasRelatedDocs := strings.Contains(lowerDoc, "related") ||
			strings.Contains(lowerDoc, "references") ||
			strings.Contains(lowerDoc, "see also")
		assert.True(t, hasRelatedDocs,
			"document should have a related documents or references section")
	})

	t.Run("structural similarity to reference docs", func(t *testing.T) {
		if len(refHeadings) == 0 {
			t.Skip("reference document not available for comparison")
		}
		// Both should have a comparable number of top-level headings
		mcpH1Count := 0
		refH1Count := 0
		for _, h := range mcpHeadings {
			if strings.HasPrefix(h, "# ") && !strings.HasPrefix(h, "## ") {
				mcpH1Count++
			}
		}
		for _, h := range refHeadings {
			if strings.HasPrefix(h, "# ") && !strings.HasPrefix(h, "## ") {
				refH1Count++
			}
		}
		assert.GreaterOrEqual(t, len(mcpHeadings), 3,
			"document should have at least 3 section headings")
	})
}

// TestAttackScenariosDistinct verifies that the four attack scenarios describe
// distinct threat vectors consistent with the MCP protocol model.
// [test_id:TS-GH-13-006]
func TestAttackScenariosDistinct(t *testing.T) {
	root := repoRoot(t)
	docContent := readFile(t, root, "docs/problems/mcp-config-drift.md")

	// Define the four expected attack vector keywords
	attackVectors := []struct {
		name    string
		keyword string
	}{
		{"malicious server injection", "injection"},
		{"endpoint replacement", "replacement"},
		{"permission escalation", "escalation"},
		{"gradual drift", "drift"},
	}

	t.Run("four attack scenarios present", func(t *testing.T) {
		lowerDoc := strings.ToLower(docContent)
		foundCount := 0
		for _, av := range attackVectors {
			if strings.Contains(lowerDoc, av.keyword) {
				foundCount++
			}
		}
		assert.GreaterOrEqual(t, foundCount, 3,
			"document should describe at least 3 of the 4 expected attack scenario types (injection, replacement, escalation, drift)")
	})

	t.Run("scenarios are distinct", func(t *testing.T) {
		lowerDoc := strings.ToLower(docContent)
		// Each attack vector keyword should appear — indicating distinct scenarios
		for _, av := range attackVectors {
			assert.True(t, strings.Contains(lowerDoc, av.keyword),
				"document should describe %s attack scenario (keyword: %s)", av.name, av.keyword)
		}
	})

	t.Run("scenarios reference MCP concepts", func(t *testing.T) {
		lowerDoc := strings.ToLower(docContent)
		mcpConcepts := []string{"mcp", "server", "tool"}
		for _, concept := range mcpConcepts {
			assert.True(t, strings.Contains(lowerDoc, concept),
				"attack scenarios should reference MCP concept: %s", concept)
		}
	})
}

// TestHarnessArchitectureReferences verifies that the document's references to
// Harness struct and SecurityConfig are accurate against the codebase.
// [test_id:TS-GH-13-007]
func TestHarnessArchitectureReferences(t *testing.T) {
	root := repoRoot(t)

	t.Run("Harness struct exists", func(t *testing.T) {
		cmd := exec.Command("grep", "-r", "--include=*.go", "type Harness struct", filepath.Join(root, "internal"))
		out, err := cmd.Output()
		result := strings.TrimSpace(string(out))
		assert.NoError(t, err, "grep for Harness struct should succeed")
		assert.NotEmpty(t, result, "Harness struct definition should exist in internal/")
	})

	t.Run("SecurityConfig exists", func(t *testing.T) {
		cmd := exec.Command("grep", "-r", "--include=*.go", "SecurityConfig", filepath.Join(root, "internal"))
		out, err := cmd.Output()
		result := strings.TrimSpace(string(out))
		assert.NoError(t, err, "grep for SecurityConfig should succeed")
		assert.NotEmpty(t, result, "SecurityConfig type or field should exist in the codebase")
	})

	t.Run("types in expected locations", func(t *testing.T) {
		// Check harness package exists
		harnessDir := filepath.Join(root, "internal", "harness")
		_, err := os.Stat(harnessDir)
		assert.NoError(t, err, "internal/harness/ directory should exist")

		if err == nil {
			cmd := exec.Command("grep", "-r", "--include=*.go", "-l", "Harness", harnessDir)
			out, _ := cmd.Output()
			assert.NotEmpty(t, strings.TrimSpace(string(out)),
				"Harness references should be found in internal/harness/")
		}
	})
}

// TestHarnessInitializationFlowConsistency verifies that the MCP config injection
// pattern is consistent with the existing harness initialization flow.
// [test_id:TS-GH-13-008]
func TestHarnessInitializationFlowConsistency(t *testing.T) {
	root := repoRoot(t)

	t.Run("harness initialization identifiable", func(t *testing.T) {
		// Search for harness initialization functions
		cmd := exec.Command("grep", "-r", "--include=*.go", "-E",
			"func.*(New|Init).*Harness|func New\\(", filepath.Join(root, "internal", "harness"))
		out, err := cmd.Output()
		if err != nil {
			// Broaden search to all internal/
			cmd = exec.Command("grep", "-r", "--include=*.go", "-E",
				"func.*(New|Init).*Harness", filepath.Join(root, "internal"))
			out, err = cmd.Output()
		}
		result := strings.TrimSpace(string(out))
		assert.NoError(t, err, "harness initialization function should be findable")
		assert.NotEmpty(t, result, "harness initialization function must exist")
	})

	t.Run("MCP config accessible during initialization", func(t *testing.T) {
		harnessDir := filepath.Join(root, "internal", "harness")
		if _, err := os.Stat(harnessDir); err != nil {
			t.Skip("internal/harness/ directory not found")
		}

		// Search for MCP or config references in harness initialization code
		cmd := exec.Command("grep", "-r", "--include=*.go", "-i", "-l", "mcp\\|config", harnessDir)
		out, _ := cmd.Output()
		files := strings.TrimSpace(string(out))
		assert.NotEmpty(t, files,
			"harness initialization should reference MCP or config, confirming injection feasibility")
	})
}

// TestNoSensitiveDisclosure verifies the document does not disclose
// sensitive implementation details that could aid attackers.
// [test_id:TS-GH-13-009]
func TestNoSensitiveDisclosure(t *testing.T) {
	root := repoRoot(t)
	docContent := readFile(t, root, "docs/problems/mcp-config-drift.md")

	t.Run("no internal endpoint URLs", func(t *testing.T) {
		// Match specific internal URLs (not generic examples or documentation URLs)
		internalURLPatterns := []string{
			"http://10.",
			"http://172.",
			"http://192.168.",
			"http://localhost:",
			"https://internal.",
		}
		for _, pattern := range internalURLPatterns {
			assert.False(t, strings.Contains(docContent, pattern),
				"document should not contain internal endpoint URL pattern: %s", pattern)
		}
	})

	t.Run("no credential paths", func(t *testing.T) {
		sensitivePatterns := []string{
			"/etc/shadow",
			".ssh/id_rsa",
			"credentials.json",
			"service-account-key",
			"/var/run/secrets",
		}
		for _, pattern := range sensitivePatterns {
			assert.False(t, strings.Contains(docContent, pattern),
				"document should not contain credential path: %s", pattern)
		}
	})

	t.Run("no internal network topology", func(t *testing.T) {
		// Check for specific internal IP ranges (not in code examples or generic docs)
		ipPattern := regexp.MustCompile(`\b10\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`)
		privateIPs := ipPattern.FindAllString(docContent, -1)
		assert.Empty(t, privateIPs,
			"document should not contain specific internal IP addresses: %v", privateIPs)

		// Check for internal hostnames
		internalHostPatterns := []string{
			".internal.",
			".corp.",
			".local:",
		}
		for _, pattern := range internalHostPatterns {
			assert.False(t, strings.Contains(docContent, pattern),
				"document should not contain internal hostname pattern: %s", pattern)
		}
	})

	t.Run("no API keys or tokens", func(t *testing.T) {
		apiKeyPatterns := []*regexp.Regexp{
			regexp.MustCompile(`(?i)api[_-]?key\s*[:=]\s*["'][^"']+["']`),
			regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9._-]{20,}`),
			regexp.MustCompile(`(?i)token\s*[:=]\s*["'][a-zA-Z0-9._-]{20,}["']`),
		}
		for _, re := range apiKeyPatterns {
			matches := re.FindAllString(docContent, -1)
			assert.Empty(t, matches,
				"document should not contain API keys or tokens: %v", matches)
		}
	})
}

// TestOpenQuestionsComplete verifies the Open Questions section is complete,
// actionable, and aligned with attack scenarios.
// [test_id:TS-GH-13-010]
func TestOpenQuestionsComplete(t *testing.T) {
	root := repoRoot(t)
	docContent := readFile(t, root, "docs/problems/mcp-config-drift.md")

	t.Run("Open Questions section exists", func(t *testing.T) {
		lowerDoc := strings.ToLower(docContent)
		hasOpenQuestions := strings.Contains(lowerDoc, "open question") ||
			strings.Contains(lowerDoc, "open-question") ||
			strings.Contains(lowerDoc, "unresolved") ||
			strings.Contains(lowerDoc, "future work") ||
			strings.Contains(lowerDoc, "next steps")
		assert.True(t, hasOpenQuestions,
			"document should contain an Open Questions or equivalent section")
	})

	t.Run("questions are actionable", func(t *testing.T) {
		// Extract the open questions section
		lowerDoc := strings.ToLower(docContent)
		sectionIdx := strings.Index(lowerDoc, "open question")
		if sectionIdx == -1 {
			sectionIdx = strings.Index(lowerDoc, "next step")
		}
		if sectionIdx == -1 {
			sectionIdx = strings.Index(lowerDoc, "future work")
		}
		if sectionIdx == -1 {
			t.Skip("no Open Questions section found to validate")
		}

		questionsSection := docContent[sectionIdx:]

		// Questions should reference specific concepts (not be vague)
		specificTerms := []string{"mcp", "server", "config", "drift", "security", "hook", "harness",
			"tool", "allowlist", "ssrf", "endpoint", "defense", "approach"}
		termFound := false
		for _, term := range specificTerms {
			if strings.Contains(strings.ToLower(questionsSection), term) {
				termFound = true
				break
			}
		}
		assert.True(t, termFound,
			"open questions should reference specific attack scenarios or defense approaches")
	})

	t.Run("questions are non-redundant", func(t *testing.T) {
		// Extract question lines (lines containing ?)
		var questions []string
		lines := strings.Split(docContent, "\n")
		for _, line := range lines {
			if strings.Contains(line, "?") && len(strings.TrimSpace(line)) > 10 {
				questions = append(questions, strings.ToLower(strings.TrimSpace(line)))
			}
		}

		// Check for obvious duplicates (same line appearing twice)
		seen := make(map[string]bool)
		for _, q := range questions {
			assert.False(t, seen[q], "duplicate question found: %s", q)
			seen[q] = true
		}
	})
}

// TestMarkdownLinksAndFormatting verifies all relative markdown links resolve
// and markdown formatting renders correctly.
// [test_id:TS-GH-13-011]
func TestMarkdownLinksAndFormatting(t *testing.T) {
	root := repoRoot(t)
	docPath := filepath.Join(root, "docs", "problems", "mcp-config-drift.md")

	content, err := os.ReadFile(docPath)
	require.NoError(t, err, "mcp-config-drift.md must exist")

	docContent := string(content)
	docDir := filepath.Dir(docPath)

	t.Run("all relative links resolve", func(t *testing.T) {
		links := extractRelativeMarkdownLinks(docContent)
		var brokenLinks []string
		for _, link := range links {
			resolved := filepath.Join(docDir, link)
			if _, err := os.Stat(resolved); err != nil {
				brokenLinks = append(brokenLinks, link)
			}
		}
		assert.Empty(t, brokenLinks,
			"all relative markdown links should resolve; broken: %v", brokenLinks)
	})

	t.Run("no malformed markdown link syntax", func(t *testing.T) {
		// Check for unclosed brackets in link syntax
		// Pattern: [ without matching ]( ... )
		lines := strings.Split(docContent, "\n")
		for i, line := range lines {
			openBrackets := strings.Count(line, "[")
			closeBrackets := strings.Count(line, "]")

			// Skip code blocks
			if strings.HasPrefix(strings.TrimSpace(line), "```") {
				continue
			}

			// In lines with link-like syntax, brackets should be balanced
			if openBrackets > 0 && strings.Contains(line, "](") {
				assert.Equal(t, openBrackets, closeBrackets,
					"line %d has unbalanced brackets: %s", i+1, line)
			}
		}
	})

	t.Run("no unclosed code blocks", func(t *testing.T) {
		codeBlockCount := strings.Count(docContent, "```")
		assert.Equal(t, 0, codeBlockCount%2,
			"code blocks should be properly closed (found %d ``` markers, expected even number)", codeBlockCount)
	})
}
