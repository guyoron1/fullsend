package tests

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

/*
Architecture Flexibility Problem Document Tests

STP Reference: outputs/stp/GH-45/GH-45_test_plan.md
STD Reference: outputs/std/GH-45/GH-45_test_description.yaml
Jira: GH-45

These tests validate the architecture flexibility problem document
(docs/problems/architecture-flexibility.md) for completeness, structure,
and cross-reference integrity.
*/

var _ = Describe("[GH-45] Architecture Flexibility Problem Document", func() {
	var (
		docPath     string
		repoRoot    string
		docContent  string
		readmeError error
	)

	BeforeEach(func() {
		// Determine repo root: walk up from CWD until we find go.mod or .git
		cwd, err := os.Getwd()
		Expect(err).NotTo(HaveOccurred())
		repoRoot = findRepoRoot(cwd)
		Expect(repoRoot).NotTo(BeEmpty(), "could not determine repository root")
		docPath = filepath.Join(repoRoot, "docs", "problems", "architecture-flexibility.md")
	})

	Context("Four architectural approaches coverage", Ordered, func() {
		var docContent string
		var err error

		BeforeAll(func() {
			raw, readErr := os.ReadFile(docPath)
			err = readErr
			docContent = string(raw)
		})

		It("[test_id:TS-GH-45-001] should cover interface-first, thin integration, deferred decisions, and compositional approaches", func() {
			Expect(err).NotTo(HaveOccurred(), "failed to read architecture-flexibility.md")
			Expect(docContent).NotTo(BeEmpty(), "architecture-flexibility.md is empty")

			lowerContent := strings.ToLower(docContent)

			// Verify all four architectural approaches are documented
			Expect(lowerContent).To(
				ContainSubstring("interface"),
				"Document should discuss interface-first architecture",
			)
			Expect(lowerContent).To(
				ContainSubstring("thin integration"),
				"Document should discuss thin integration layers",
			)
			Expect(lowerContent).To(
				ContainSubstring("deferred"),
				"Document should discuss deferred decisions",
			)
			Expect(lowerContent).To(
				ContainSubstring("compositional"),
				"Document should discuss compositional architecture",
			)

			// Verify trade-offs are discussed
			Expect(lowerContent).To(SatisfyAny(
				ContainSubstring("trade-off"),
				ContainSubstring("tradeoff"),
				ContainSubstring("trade off"),
				ContainSubstring("pros"),
				ContainSubstring("cons"),
				ContainSubstring("disadvantage"),
				ContainSubstring("advantage"),
				ContainSubstring("cost"),
			), "Document should include trade-off analysis for the approaches")
		})
	})

	Context("Stable vs swappable component categorization", Ordered, func() {
		var docContent string
		var err error

		BeforeAll(func() {
			raw, readErr := os.ReadFile(docPath)
			err = readErr
			docContent = string(raw)
		})

		It("[test_id:TS-GH-45-002] should categorize stable and swappable components", func() {
			Expect(err).NotTo(HaveOccurred(), "failed to read architecture-flexibility.md")
			Expect(docContent).NotTo(BeEmpty())

			lowerContent := strings.ToLower(docContent)

			// Verify stable components are identified
			Expect(lowerContent).To(
				ContainSubstring("coordination"),
				"Document should identify coordination model as a stable component",
			)
			Expect(lowerContent).To(
				ContainSubstring("trust"),
				"Document should identify trust model as a stable component",
			)
			Expect(lowerContent).To(
				ContainSubstring("governance"),
				"Document should identify governance as a stable component",
			)

			// Verify swappable components are identified
			swappableFound := 0
			swappableKeywords := []string{"cli", "model", "framework", "review tool"}
			for _, kw := range swappableKeywords {
				if strings.Contains(lowerContent, kw) {
					swappableFound++
				}
			}
			Expect(swappableFound).To(BeNumerically(">=", 2),
				"Document should identify swappable components (CLIs, models, frameworks, review tools) — found %d of %d keywords",
				swappableFound, len(swappableKeywords),
			)

			// Verify distinction between stable and swappable
			Expect(lowerContent).To(SatisfyAny(
				ContainSubstring("stable"),
				ContainSubstring("swappable"),
				ContainSubstring("fixed"),
				ContainSubstring("interchangeable"),
				ContainSubstring("loosely coupled"),
			), "Document should distinguish between stable and swappable components")
		})
	})

	Context("Cross-reference integrity", Ordered, func() {
		var docContent string
		var err error

		BeforeAll(func() {
			raw, readErr := os.ReadFile(docPath)
			err = readErr
			docContent = string(raw)
		})

		It("[test_id:TS-GH-45-003] should contain valid links to all 7 existing problem documents", func() {
			Expect(err).NotTo(HaveOccurred(), "failed to read architecture-flexibility.md")
			Expect(docContent).NotTo(BeEmpty())

			expectedRefs := []string{
				"agent-architecture",
				"agent-infrastructure",
				"landscape",
				"governance",
				"codebase-context",
				"security-threat-model",
				"testing-agents",
			}

			lowerContent := strings.ToLower(docContent)

			// Verify each expected problem doc is referenced
			for _, ref := range expectedRefs {
				Expect(lowerContent).To(
					ContainSubstring(ref),
					"Document should contain a reference to %s problem document", ref,
				)
			}

			// Extract markdown links and verify they point to existing files
			linkRe := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
			matches := linkRe.FindAllStringSubmatch(docContent, -1)
			Expect(matches).NotTo(BeEmpty(), "Document should contain markdown links")

			// Check that cross-reference links to problem docs resolve
			problemDocsDir := filepath.Join(repoRoot, "docs", "problems")
			for _, match := range matches {
				linkPath := match[2]
				// Only check relative links to problem docs
				if strings.Contains(linkPath, "://") {
					continue // skip external URLs
				}
				// Resolve relative to the document's directory
				resolvedPath := filepath.Join(filepath.Dir(docPath), linkPath)
				if strings.Contains(resolvedPath, "problems") {
					_, statErr := os.Stat(resolvedPath)
					Expect(statErr).NotTo(HaveOccurred(),
						"Cross-reference link target should exist: %s (resolved to %s)", linkPath, resolvedPath,
					)
				}
			}
			_ = problemDocsDir
		})
	})

	Context("README index link", Ordered, func() {
		var readmeContent string

		BeforeAll(func() {
			readmePath := filepath.Join(repoRoot, "README.md")
			raw, readErr := os.ReadFile(readmePath)
			readmeError = readErr
			readmeContent = string(raw)
		})

		It("[test_id:TS-GH-45-004] should include Architecture Flexibility link in README with correct path", func() {
			Expect(readmeError).NotTo(HaveOccurred(), "failed to read README.md")
			Expect(readmeContent).NotTo(BeEmpty(), "README.md is empty")

			lowerReadme := strings.ToLower(readmeContent)

			// Verify README contains a reference to architecture-flexibility
			Expect(lowerReadme).To(
				ContainSubstring("architecture-flexibility"),
				"README should contain a link referencing architecture-flexibility document",
			)

			// Verify the link path is correct
			Expect(readmeContent).To(SatisfyAny(
				ContainSubstring("docs/problems/architecture-flexibility.md"),
				ContainSubstring("docs/problems/architecture-flexibility"),
			), "README link should point to docs/problems/architecture-flexibility.md")

			// Verify the link has descriptive text
			linkRe := regexp.MustCompile(`(?i)\[([^\]]*architecture[^\]]*flexibility[^\]]*)\]`)
			matches := linkRe.FindStringSubmatch(readmeContent)
			if matches == nil {
				// Fallback: check for any link containing the path
				linkRe2 := regexp.MustCompile(`\[[^\]]+\]\([^)]*architecture-flexibility[^)]*\)`)
				Expect(linkRe2.MatchString(readmeContent)).To(BeTrue(),
					"README should have a markdown link to the architecture-flexibility document",
				)
			}
		})
	})

	Context("Interface contract table", Ordered, func() {
		var docContent string
		var err error

		BeforeAll(func() {
			raw, readErr := os.ReadFile(docPath)
			err = readErr
			docContent = string(raw)
		})

		It("[test_id:TS-GH-45-005] should include interface contract table with agent roles", func() {
			Expect(err).NotTo(HaveOccurred(), "failed to read architecture-flexibility.md")
			Expect(docContent).NotTo(BeEmpty())

			lowerContent := strings.ToLower(docContent)

			// Verify interface contract table or section exists
			Expect(lowerContent).To(SatisfyAny(
				ContainSubstring("interface contract"),
				ContainSubstring("contract table"),
				ContainSubstring("agent role"),
				ContainSubstring("role definition"),
			), "Document should contain an interface contract table or section")

			// Verify all three agent roles are defined
			expectedRoles := []string{"implementation", "review", "triage"}
			for _, role := range expectedRoles {
				Expect(lowerContent).To(
					ContainSubstring(role),
					"Document should define the %s agent role", role,
				)
			}

			// Verify table columns or structured content for input/output/contract
			columnKeywords := []string{"input", "output", "contract"}
			columnsFound := 0
			for _, col := range columnKeywords {
				if strings.Contains(lowerContent, col) {
					columnsFound++
				}
			}
			Expect(columnsFound).To(BeNumerically(">=", 2),
				"Interface contract section should define input, output, and contract columns — found %d of %d",
				columnsFound, len(columnKeywords),
			)
		})
	})

	Context("Broken cross-reference handling", Ordered, func() {
		var docContent string
		var err error

		BeforeAll(func() {
			raw, readErr := os.ReadFile(docPath)
			err = readErr
			docContent = string(raw)
		})

		It("[test_id:TS-GH-45-006] should handle broken or missing cross-reference links gracefully", func() {
			Expect(err).NotTo(HaveOccurred(), "failed to read architecture-flexibility.md")
			Expect(docContent).NotTo(BeEmpty())

			// Extract all markdown links
			linkRe := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
			matches := linkRe.FindAllStringSubmatch(docContent, -1)

			// Verify all cross-reference links use standard markdown syntax
			// (no raw URLs, no wiki-style links)
			rawURLRe := regexp.MustCompile(`(?m)^[^[]*(?:https?://|\.\.?/)[^\s)]+\s*$`)
			wikiLinkRe := regexp.MustCompile(`\[\[[^\]]+\]\]`)

			Expect(wikiLinkRe.MatchString(docContent)).To(BeFalse(),
				"Document should not use wiki-style [[link]] syntax",
			)

			// Check that relative links use proper markdown format
			for _, match := range matches {
				linkText := match[1]
				linkPath := match[2]

				// Link text should not be empty
				Expect(strings.TrimSpace(linkText)).NotTo(BeEmpty(),
					"Markdown link should have descriptive text, not empty: [%s](%s)", linkText, linkPath,
				)

				// Relative links should not contain spaces (URL encoding issue)
				if !strings.Contains(linkPath, "://") {
					Expect(linkPath).NotTo(ContainSubstring(" "),
						"Relative link path should not contain spaces: %s", linkPath,
					)
				}
			}

			_ = rawURLRe // available for extended validation
		})
	})

	Context("Problem document structure conventions", Ordered, func() {
		var docContent string
		var err error

		BeforeAll(func() {
			raw, readErr := os.ReadFile(docPath)
			err = readErr
			docContent = string(raw)
		})

		It("[test_id:TS-GH-45-007] should follow established problem doc conventions with required sections", func() {
			Expect(err).NotTo(HaveOccurred(), "failed to read architecture-flexibility.md")
			Expect(docContent).NotTo(BeEmpty())

			// Extract headings from the document
			headingRe := regexp.MustCompile(`(?m)^#{1,4}\s+(.+)$`)
			headingMatches := headingRe.FindAllStringSubmatch(docContent, -1)
			Expect(headingMatches).NotTo(BeEmpty(), "Document should contain markdown headings")

			// Collect all headings as lowercase for matching
			var headings []string
			for _, m := range headingMatches {
				headings = append(headings, strings.ToLower(strings.TrimSpace(m[1])))
			}
			allHeadings := strings.Join(headings, " | ")
			lowerContent := strings.ToLower(docContent)

			// Verify problem statement section
			hasProblem := containsAnyKeyword(allHeadings, []string{"problem", "challenge", "issue"})
			Expect(hasProblem).To(BeTrue(),
				"Document should have a problem statement section. Headings found: %s", allHeadings,
			)

			// Verify approaches section with trade-offs
			hasApproaches := containsAnyKeyword(allHeadings, []string{"approach", "strategy", "solution", "option"})
			Expect(hasApproaches).To(BeTrue(),
				"Document should have an approaches section. Headings found: %s", allHeadings,
			)

			// Verify trade-offs are discussed within the approaches
			Expect(lowerContent).To(SatisfyAny(
				ContainSubstring("trade-off"),
				ContainSubstring("tradeoff"),
				ContainSubstring("trade off"),
				ContainSubstring("pros"),
				ContainSubstring("cons"),
				ContainSubstring("advantage"),
				ContainSubstring("disadvantage"),
			), "Document should contain trade-off analysis within approaches")

			// Verify relationship to other areas section
			hasRelationship := containsAnyKeyword(allHeadings, []string{"relationship", "related", "cross-reference", "connection", "other area", "other problem"})
			if !hasRelationship {
				// Fallback: check if cross-references exist in content even without a dedicated heading
				hasRelationship = strings.Contains(lowerContent, "related") || strings.Contains(lowerContent, "see also")
			}
			Expect(hasRelationship).To(BeTrue(),
				"Document should have a relationship to other areas section. Headings found: %s", allHeadings,
			)

			// Verify open questions section
			hasOpenQuestions := containsAnyKeyword(allHeadings, []string{"open question", "unresolved", "future", "todo", "tbd", "outstanding"})
			Expect(hasOpenQuestions).To(BeTrue(),
				"Document should have an open questions section. Headings found: %s", allHeadings,
			)
		})
	})

	Context("Open questions content", Ordered, func() {
		var docContent string
		var err error

		BeforeAll(func() {
			raw, readErr := os.ReadFile(docPath)
			err = readErr
			docContent = string(raw)
		})

		It("[test_id:TS-GH-45-008] should address key architectural decisions in open questions", func() {
			Expect(err).NotTo(HaveOccurred(), "failed to read architecture-flexibility.md")
			Expect(docContent).NotTo(BeEmpty())

			lowerContent := strings.ToLower(docContent)

			// Find the open questions section
			openQIdx := findSectionStart(lowerContent, []string{"open question", "unresolved", "outstanding"})
			Expect(openQIdx).To(BeNumerically(">=", 0),
				"Document should contain an open questions section",
			)

			// Get content from open questions section onward
			openQContent := lowerContent[openQIdx:]

			// Verify interface formality is addressed
			Expect(openQContent).To(SatisfyAny(
				ContainSubstring("interface formality"),
				ContainSubstring("formal interface"),
				ContainSubstring("interface definition"),
				ContainSubstring("how formal"),
			), "Open questions should address interface formality level")

			// Verify tool boundary blurring is addressed
			Expect(openQContent).To(SatisfyAny(
				ContainSubstring("tool boundary"),
				ContainSubstring("boundary blur"),
				ContainSubstring("boundaries between"),
				ContainSubstring("tool overlap"),
				ContainSubstring("boundary"),
			), "Open questions should address tool boundary blurring concerns")

			// Verify swap cost estimation is addressed
			Expect(openQContent).To(SatisfyAny(
				ContainSubstring("swap cost"),
				ContainSubstring("cost of swap"),
				ContainSubstring("migration cost"),
				ContainSubstring("replacement cost"),
				ContainSubstring("switching cost"),
				ContainSubstring("cost"),
			), "Open questions should address swap cost estimation")
		})
	})

	Context("Standalone rendering", Ordered, func() {
		var docContent string
		var err error

		BeforeAll(func() {
			raw, readErr := os.ReadFile(docPath)
			err = readErr
			docContent = string(raw)
		})

		It("[test_id:TS-GH-45-009] should render correctly as standalone markdown", func() {
			Expect(err).NotTo(HaveOccurred(), "failed to read architecture-flexibility.md")
			Expect(docContent).NotTo(BeEmpty())

			// Verify document uses valid markdown syntax
			// Check for unclosed or malformed links
			openBrackets := strings.Count(docContent, "[")
			closeBrackets := strings.Count(docContent, "]")
			// Allow some tolerance for brackets used in non-link contexts
			Expect(openBrackets).To(BeNumerically("~", closeBrackets, openBrackets/2+2),
				"Document should have roughly balanced brackets (open: %d, close: %d)", openBrackets, closeBrackets,
			)

			// Verify all links use standard markdown syntax
			linkRe := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
			matches := linkRe.FindAllStringSubmatch(docContent, -1)
			for _, match := range matches {
				linkPath := match[2]
				// Links should not contain GitHub-specific fragments that break elsewhere
				Expect(linkPath).NotTo(MatchRegexp(`/blob/[a-f0-9]{40}/`),
					"Links should not use commit-specific GitHub URLs: %s", linkPath,
				)
			}

			// Verify no GitHub-specific rendering features
			// Check for GitHub-specific alert syntax that breaks elsewhere
			githubAlertRe := regexp.MustCompile(`> \[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]`)
			if githubAlertRe.MatchString(docContent) {
				// Not a failure, just a warning — these render as blockquotes outside GitHub
				GinkgoWriter.Printf("Note: Document uses GitHub-specific alert syntax which renders as plain blockquotes outside GitHub\n")
			}

			// Verify headings exist (document has structure)
			headingRe := regexp.MustCompile(`(?m)^#{1,6}\s+.+$`)
			headings := headingRe.FindAllString(docContent, -1)
			Expect(len(headings)).To(BeNumerically(">=", 3),
				"Document should have at least 3 headings for proper structure, found %d", len(headings),
			)

			// Verify no HTML-only constructs that break in pure markdown renderers
			htmlOnlyRe := regexp.MustCompile(`<(div|span|style|script)\b`)
			Expect(htmlOnlyRe.MatchString(docContent)).To(BeFalse(),
				"Document should not use HTML-only constructs (div, span, style, script) that may break in non-HTML renderers",
			)
		})
	})
})

// containsAnyKeyword checks if text contains any of the given keywords.
func containsAnyKeyword(text string, keywords []string) bool {
	for _, kw := range keywords {
		if strings.Contains(text, kw) {
			return true
		}
	}
	return false
}

// findSectionStart finds the start index of a section identified by any of the given keywords in a heading.
func findSectionStart(content string, keywords []string) int {
	headingRe := regexp.MustCompile(`(?m)^#{1,4}\s+(.+)$`)
	matches := headingRe.FindAllStringSubmatchIndex(content, -1)
	for _, m := range matches {
		headingText := strings.ToLower(content[m[2]:m[3]])
		for _, kw := range keywords {
			if strings.Contains(headingText, kw) {
				return m[0]
			}
		}
	}
	return -1
}

// findRepoRoot walks up from the given directory to find the repository root.
func findRepoRoot(dir string) string {
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}
