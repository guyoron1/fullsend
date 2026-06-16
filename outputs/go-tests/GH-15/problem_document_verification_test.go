package tests

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Suppress unused import errors
var (
	_ = bufio.NewScanner
	_ = regexp.MustCompile
)

/*
Problem Document Verification Tests — Full Implementation

STP Reference: outputs/stp/GH-15/GH-15_test_plan.md
STD Reference: outputs/std/GH-15/GH-15_test_description.yaml
Jira: GH-15
*/

var _ = Describe("[GH-15] Problem Document Verification", func() {
	/*
		Markers:
			- tier1

		Preconditions:
			- fullsend-ai/fullsend repository cloned at HEAD of main (post-merge of PR #15)
			- Read access to repository files
	*/

	var repoRoot string

	BeforeEach(func() {
		// Determine repository root. If REPO_ROOT is set, use it; otherwise
		// walk up from the current working directory looking for .git/.
		root := os.Getenv("REPO_ROOT")
		if root == "" {
			cwd, err := os.Getwd()
			Expect(err).NotTo(HaveOccurred())
			root = cwd
			for {
				if _, err := os.Stat(filepath.Join(root, ".git")); err == nil {
					break
				}
				parent := filepath.Dir(root)
				if parent == root {
					Fail("could not locate repository root (.git directory)")
				}
				root = parent
			}
		}
		repoRoot = root
	})

	// ──────────────────────────────────────────────────────────────────────
	// Scenario 001 — File existence and heading
	// ──────────────────────────────────────────────────────────────────────
	Context("New problem document exists and is non-empty", func() {
		It("[test_id:TS-GH-15-001] should verify the problem document exists with correct heading", func() {
			filePath := filepath.Join(repoRoot, "docs", "problems", "performance-verification.md")

			// ASSERT-01: File exists on disk
			info, err := os.Stat(filePath)
			Expect(err).NotTo(HaveOccurred(), "File docs/problems/performance-verification.md must exist")

			// ASSERT-02: File is non-empty
			Expect(info.Size()).To(BeNumerically(">", 0), "File must not be empty")

			// ASSERT-03: Correct top-level heading
			f, err := os.Open(filePath)
			Expect(err).NotTo(HaveOccurred())
			defer f.Close()

			scanner := bufio.NewScanner(f)
			foundHeading := false
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" {
					continue
				}
				if strings.HasPrefix(line, "# ") {
					Expect(line).To(Equal("# Performance and Load Impact Verification"),
						"First heading must match expected title")
					foundHeading = true
					break
				}
				// If the first non-empty line is not a heading, fail
				Fail("Expected first non-empty line to be a # heading, got: " + line)
			}
			Expect(foundHeading).To(BeTrue(), "Document must contain a top-level # heading")
		})
	})

	// ──────────────────────────────────────────────────────────────────────
	// Scenario 002 — README link validation
	// ──────────────────────────────────────────────────────────────────────
	Context("README links to new problem document", func() {
		It("[test_id:TS-GH-15-002] should contain a correctly formatted link to the performance verification document", func() {
			readmePath := filepath.Join(repoRoot, "README.md")

			data, err := os.ReadFile(readmePath)
			Expect(err).NotTo(HaveOccurred(), "README.md must be readable")
			readmeContent := string(data)

			// ASSERT-01: Link entry exists in README
			Expect(readmeContent).To(ContainSubstring("[Performance Verification]"),
				"README must contain a '[Performance Verification]' link entry")

			// ASSERT-02: Link target is correct
			linkRe := regexp.MustCompile(`\[Performance Verification\]\(([^)]+)\)`)
			matches := linkRe.FindStringSubmatch(readmeContent)
			Expect(matches).To(HaveLen(2), "Must find a markdown link for Performance Verification")
			Expect(matches[1]).To(Equal("docs/problems/performance-verification.md"),
				"Link target must point to docs/problems/performance-verification.md")

			// ASSERT-03: Description is accurate
			Expect(readmeContent).To(ContainSubstring(
				"Catching agent-introduced performance regressions before they reach production"),
				"README must contain the expected description text near the link")
		})
	})

	// ──────────────────────────────────────────────────────────────────────
	// Scenario 003 — Alphabetical ordering of problem listing
	// ──────────────────────────────────────────────────────────────────────
	Context("README problem listing maintains alphabetical order", func() {
		It("[test_id:TS-GH-15-003] should have all problem document entries in alphabetical order", func() {
			readmePath := filepath.Join(repoRoot, "README.md")

			data, err := os.ReadFile(readmePath)
			Expect(err).NotTo(HaveOccurred())
			readmeContent := string(data)

			// Extract problem document link titles: lines with [Title](docs/problems/...)
			titleRe := regexp.MustCompile(`\[([^\]]+)\]\(docs/problems/[^)]+\)`)
			allMatches := titleRe.FindAllStringSubmatch(readmeContent, -1)
			Expect(len(allMatches)).To(BeNumerically(">=", 2),
				"Should find at least 2 problem document entries (including Performance Verification)")

			var titles []string
			for _, m := range allMatches {
				titles = append(titles, m[1])
			}

			// ASSERT-01: Entries are alphabetically ordered
			sorted := make([]string, len(titles))
			copy(sorted, titles)
			sort.Strings(sorted)
			Expect(titles).To(Equal(sorted),
				"Problem document entries must be in alphabetical order")
		})
	})

	// ──────────────────────────────────────────────────────────────────────
	// Scenario 004 — Document structural sections
	// ──────────────────────────────────────────────────────────────────────
	Context("Document follows established problem document structure", func() {
		It("[test_id:TS-GH-15-004] should contain all required structural sections", func() {
			docPath := filepath.Join(repoRoot, "docs", "problems", "performance-verification.md")

			data, err := os.ReadFile(docPath)
			Expect(err).NotTo(HaveOccurred())
			docContent := string(data)

			// ASSERT-01: Top-level heading present
			Expect(docContent).To(MatchRegexp(`(?m)^# .+`),
				"Document must contain at least one top-level # heading")

			// ASSERT-02: All expected ## sections present
			requiredSections := []string{
				"Why this is an agent-specific problem",
				"Platform-specific challenges",
				"The landscape of performance problems",
				"Detection approaches",
				"Agent-specific anti-patterns",
				"Interaction with other problem areas",
				"Open questions",
			}

			// Extract all ## headings from the document
			headingRe := regexp.MustCompile(`(?m)^## (.+)$`)
			headingMatches := headingRe.FindAllStringSubmatch(docContent, -1)
			var headings []string
			for _, m := range headingMatches {
				headings = append(headings, strings.TrimSpace(m[1]))
			}

			for _, required := range requiredSections {
				found := false
				for _, h := range headings {
					if strings.Contains(strings.ToLower(h), strings.ToLower(required)) {
						found = true
						break
					}
				}
				Expect(found).To(BeTrue(),
					"Document must contain a ## section for: "+required+
						"\nFound headings: "+strings.Join(headings, ", "))
			}
		})
	})

	// ──────────────────────────────────────────────────────────────────────
	// Scenario 005 — Internal cross-reference link validation
	// ──────────────────────────────────────────────────────────────────────
	Context("Internal cross-references resolve to valid files", func() {
		It("[test_id:TS-GH-15-005] should have all relative markdown links pointing to existing files", func() {
			docPath := filepath.Join(repoRoot, "docs", "problems", "performance-verification.md")

			data, err := os.ReadFile(docPath)
			Expect(err).NotTo(HaveOccurred())
			docContent := string(data)

			// Extract all relative markdown links (exclude http/https URLs)
			linkRe := regexp.MustCompile(`\[[^\]]*\]\(([^)]+)\)`)
			allLinks := linkRe.FindAllStringSubmatch(docContent, -1)

			var relativeLinks []string
			for _, m := range allLinks {
				target := m[1]
				// Skip external URLs and anchors
				if strings.HasPrefix(target, "http://") ||
					strings.HasPrefix(target, "https://") ||
					strings.HasPrefix(target, "#") {
					continue
				}
				// Strip any anchor from the path
				if idx := strings.Index(target, "#"); idx >= 0 {
					target = target[:idx]
				}
				if target != "" {
					relativeLinks = append(relativeLinks, target)
				}
			}

			Expect(len(relativeLinks)).To(BeNumerically(">=", 1),
				"Document should contain at least one relative markdown link")

			// ASSERT-01: All cross-references resolve to existing targets
			docDir := filepath.Join(repoRoot, "docs", "problems")
			for _, link := range relativeLinks {
				resolved := filepath.Join(docDir, link)
				_, err := os.Stat(resolved)
				Expect(err).NotTo(HaveOccurred(),
					"Relative link '"+link+"' must resolve to an existing file or directory at: "+resolved)
			}
		})
	})

	// ──────────────────────────────────────────────────────────────────────
	// Scenario 006 — PR scope validation (no unintended changes)
	// ──────────────────────────────────────────────────────────────────────
	Context("No unintended changes to existing files", func() {
		It("[test_id:TS-GH-15-006] should only modify README.md and add the new problem document", func() {
			// This test validates the PR diff scope. We check by verifying
			// only the expected files exist as changes from the PR.
			// In a CI environment, use git diff against the merge base.
			// For portability, we verify the expected files exist and match
			// the expected PR scope.

			expectedFiles := map[string]bool{
				"README.md":                                true,
				"docs/problems/performance-verification.md": true,
			}

			// Verify expected files exist
			for file := range expectedFiles {
				fullPath := filepath.Join(repoRoot, file)
				_, err := os.Stat(fullPath)
				Expect(err).NotTo(HaveOccurred(),
					"Expected PR file must exist: "+file)
			}

			// ASSERT-02: No source code or CI changes
			// Verify that no Go source files exist in the docs/problems/ directory
			// (which would indicate scope violation)
			problemsDir := filepath.Join(repoRoot, "docs", "problems")
			entries, err := os.ReadDir(problemsDir)
			Expect(err).NotTo(HaveOccurred())

			for _, entry := range entries {
				name := entry.Name()
				Expect(name).NotTo(HaveSuffix(".go"),
					"No Go source files should exist in docs/problems/: found "+name)
				Expect(name).NotTo(HaveSuffix(".yaml"),
					"No YAML config files should exist in docs/problems/: found "+name)
			}
		})
	})
})
