//go:build e2e

package tests

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestEvalFrameworkCoverage verifies that each eval framework section describes
// capabilities and gaps for promptfoo, deepeval, and lightspeed-evaluation.
// [test_id:TS-GH-14-006] Tier 1 / P2
func TestEvalFrameworkCoverage(t *testing.T) {
	content := readTestingAgentsDoc(t)
	contentLower := strings.ToLower(content)

	frameworks := []struct {
		name     string
		keywords []string
	}{
		{"promptfoo", []string{"promptfoo"}},
		{"deepeval", []string{"deepeval"}},
		{"lightspeed-evaluation", []string{"lightspeed-evaluation", "lightspeed evaluation", "lightspeed_evaluation"}},
	}

	for _, fw := range frameworks {
		found := false
		for _, kw := range fw.keywords {
			if strings.Contains(contentLower, kw) {
				found = true
				break
			}
		}
		assert.True(t, found,
			"%s section should be documented in testing-agents.md", fw.name)
	}
}

// TestInputExpansionPattern verifies the input expansion from seed sets pattern
// is documented.
// [test_id:TS-GH-14-007] Tier 1 / P2
func TestInputExpansionPattern(t *testing.T) {
	content := readTestingAgentsDoc(t)
	contentLower := strings.ToLower(content)

	// Check for seed-related keywords indicating input expansion pattern
	hasSeed := strings.Contains(contentLower, "seed")
	hasExpansion := strings.Contains(contentLower, "expansion") ||
		strings.Contains(contentLower, "expand") ||
		strings.Contains(contentLower, "augment") ||
		strings.Contains(contentLower, "generate") ||
		strings.Contains(contentLower, "synthetic")

	assert.True(t, hasSeed, "document should reference seed sets")
	assert.True(t, hasExpansion,
		"document should describe expansion/augmentation of seed test cases")
}
