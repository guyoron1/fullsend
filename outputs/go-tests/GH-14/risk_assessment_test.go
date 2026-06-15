//go:build e2e

package tests

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestRiskAssessmentSpectrumCoverage verifies four approaches cover the risk
// assessment spectrum from deterministic through semantic.
// [test_id:TS-GH-14-011] Tier 1 / P1
func TestRiskAssessmentSpectrumCoverage(t *testing.T) {
	content := readToolCallRiskDoc(t)
	contentLower := strings.ToLower(content)

	// Check for deterministic / rule-based approach
	hasDeterministic := strings.Contains(contentLower, "deterministic") ||
		strings.Contains(contentLower, "rule-based") ||
		strings.Contains(contentLower, "rule based")
	assert.True(t, hasDeterministic,
		"document should describe deterministic/rule-based approach")

	// Check for heuristic approach
	hasHeuristic := strings.Contains(contentLower, "heuristic") ||
		strings.Contains(contentLower, "pattern-based") ||
		strings.Contains(contentLower, "signature")
	assert.True(t, hasHeuristic,
		"document should describe heuristic approach")

	// Check for LLM-as-judge / semantic approach
	hasSemantic := strings.Contains(contentLower, "llm-as-judge") ||
		strings.Contains(contentLower, "llm as judge") ||
		strings.Contains(contentLower, "semantic") ||
		strings.Contains(contentLower, "llm judge")
	assert.True(t, hasSemantic,
		"document should describe LLM-as-judge/semantic approach")

	// Check for hybrid approach
	hasHybrid := strings.Contains(contentLower, "hybrid")
	assert.True(t, hasHybrid,
		"document should describe hybrid approach")

	// Verify at least four distinct approaches
	approachCount := 0
	if hasDeterministic {
		approachCount++
	}
	if hasHeuristic {
		approachCount++
	}
	if hasSemantic {
		approachCount++
	}
	if hasHybrid {
		approachCount++
	}
	assert.GreaterOrEqual(t, approachCount, 4,
		"document should describe at least four distinct risk assessment approaches")
}

// TestHybridApproachReferences verifies the hybrid approach section correctly
// references both deterministic and LLM-judge components.
// [test_id:TS-GH-14-012] Tier 1 / P1
func TestHybridApproachReferences(t *testing.T) {
	content := readToolCallRiskDoc(t)
	contentLower := strings.ToLower(content)

	// Locate the hybrid section — find "hybrid" and examine surrounding content
	hybridIdx := strings.Index(contentLower, "hybrid")
	assert.GreaterOrEqual(t, hybridIdx, 0,
		"document should contain a hybrid approach section")

	if hybridIdx < 0 {
		return // Can't continue without finding hybrid section
	}

	// Extract a window around the hybrid section for analysis
	// Look at the section from hybrid keyword to ~2000 chars ahead
	windowStart := hybridIdx
	windowEnd := hybridIdx + 2000
	if windowEnd > len(contentLower) {
		windowEnd = len(contentLower)
	}
	hybridSection := contentLower[windowStart:windowEnd]

	// Check for deterministic component reference
	hasDeterministicRef := strings.Contains(hybridSection, "deterministic") ||
		strings.Contains(hybridSection, "rule-based") ||
		strings.Contains(hybridSection, "rule based") ||
		strings.Contains(hybridSection, "static")
	assert.True(t, hasDeterministicRef,
		"hybrid approach section should reference deterministic/rule-based component")

	// Check for LLM-judge component reference
	hasLLMRef := strings.Contains(hybridSection, "llm") ||
		strings.Contains(hybridSection, "semantic") ||
		strings.Contains(hybridSection, "judge") ||
		strings.Contains(hybridSection, "model")
	assert.True(t, hasLLMRef,
		"hybrid approach section should reference LLM-judge/semantic component")
}
