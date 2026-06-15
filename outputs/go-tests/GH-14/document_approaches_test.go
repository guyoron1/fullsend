//go:build e2e

package tests

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// repoRoot returns the repository root directory.
// It walks up from the current file's directory until it finds go.mod.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller failed")
	dir := filepath.Dir(thisFile)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repository root (no go.mod found)")
		}
		dir = parent
	}
}

// readTestingAgentsDoc reads the testing-agents.md document and returns its content.
func readTestingAgentsDoc(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)
	docPath := filepath.Join(root, "docs", "problems", "testing-agents.md")
	content, err := os.ReadFile(docPath)
	require.NoError(t, err, "failed to read testing-agents.md")
	return string(content)
}

// TestDocumentApproachesCoverage verifies all four testing approaches are documented
// with trade-offs analysis.
// [test_id:TS-GH-14-001] Tier 1 / P1
func TestDocumentApproachesCoverage(t *testing.T) {
	content := readTestingAgentsDoc(t)

	// Check for golden-set evaluation section
	hasGoldenSet := strings.Contains(strings.ToLower(content), "golden-set") ||
		strings.Contains(strings.ToLower(content), "golden set")
	assert.True(t, hasGoldenSet, "document should contain golden-set evaluation section")

	// Check for behavioral contract testing section
	hasBehavioral := strings.Contains(strings.ToLower(content), "behavioral contract") ||
		strings.Contains(strings.ToLower(content), "contract test")
	assert.True(t, hasBehavioral, "document should contain behavioral contract testing section")

	// Check for canary deployment section
	hasCanary := strings.Contains(strings.ToLower(content), "canary")
	assert.True(t, hasCanary, "document should contain canary deployment section")

	// Check for mutation testing section
	hasMutation := strings.Contains(strings.ToLower(content), "mutation test")
	assert.True(t, hasMutation, "document should contain mutation testing section")

	// Verify trade-off analysis is present
	hasTradeoffs := strings.Contains(strings.ToLower(content), "trade-off") ||
		strings.Contains(strings.ToLower(content), "tradeoff") ||
		strings.Contains(strings.ToLower(content), "trade off") ||
		strings.Contains(strings.ToLower(content), "pros") ||
		strings.Contains(strings.ToLower(content), "cons") ||
		strings.Contains(strings.ToLower(content), "advantage") ||
		strings.Contains(strings.ToLower(content), "disadvantage")
	assert.True(t, hasTradeoffs, "document should include trade-off or pros/cons analysis")
}

// TestCIPipelineStages verifies the CI pipeline section references all five pipeline stages.
// [test_id:TS-GH-14-002] Tier 1 / P1
func TestCIPipelineStages(t *testing.T) {
	content := readTestingAgentsDoc(t)
	contentLower := strings.ToLower(content)

	stages := map[string][]string{
		"prompt-design":    {"prompt-design", "prompt design"},
		"eval-run":         {"eval-run", "eval run"},
		"score-threshold":  {"score-threshold", "score threshold"},
		"regression-gate":  {"regression-gate", "regression gate"},
		"deploy-canary":    {"deploy-canary", "canary"},
	}

	for stageName, keywords := range stages {
		found := false
		for _, kw := range keywords {
			if strings.Contains(contentLower, kw) {
				found = true
				break
			}
		}
		assert.True(t, found, "CI pipeline section should reference %s stage", stageName)
	}
}

// TestMissingApproachSection [NEGATIVE] validates that a document missing one of the
// four testing approach sections is detected as incomplete.
// [test_id:TS-GH-14-003] Tier 1 / P2
func TestMissingApproachSection(t *testing.T) {
	// Simulate content missing mutation testing section
	content := `## Golden-Set Evaluation
Golden-set evaluation uses curated test cases.

## Behavioral Contract Testing
Behavioral contract testing verifies invariants.

## Canary Deployments
Canary deployments test in production with limited blast radius.
`

	approaches := []struct {
		name    string
		keyword string
	}{
		{"golden-set", "golden-set"},
		{"behavioral contract", "behavioral contract"},
		{"canary", "canary"},
		{"mutation testing", "mutation test"},
	}

	contentLower := strings.ToLower(content)
	missingApproaches := []string{}
	for _, approach := range approaches {
		if !strings.Contains(contentLower, approach.keyword) {
			missingApproaches = append(missingApproaches, approach.name)
		}
	}

	// Expect that mutation testing is detected as missing
	require.NotEmpty(t, missingApproaches, "validation should detect missing approaches")
	assert.Contains(t, missingApproaches, "mutation testing",
		"validation should identify mutation testing as the missing section")
}
