//go:build e2e

package tests

/*
Output Pipeline Redaction Tests — GH-18

STP Reference: outputs/stp/GH-18/GH-18_test_plan.md
Jira: GH-18
Requirement: Output pipeline redacts API keys and tokens
*/

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fullsend-ai/fullsend/internal/security"
)

// TestAPIKeyRedactedFromOutput verifies that the output pipeline detects and
// redacts OpenAI-style API keys (sk-...) from agent-generated text.
//
// [test_id:TS-GH-18-015]
func TestAPIKeyRedactedFromOutput(t *testing.T) {
	// Setup: Create output text containing an API key
	apiKey := "sk-proj-abc123def456ghi789jkl012mno345pqr678"
	outputText := "The API key is " + apiKey + " and the request succeeded."

	// Execute: Run through output pipeline
	pipeline := security.OutputPipeline()
	result := pipeline.Scan(outputText)

	// Assert: API key is redacted
	assert.False(t, result.Safe, "Output with API key should be marked unsafe")
	assert.NotContains(t, result.Sanitized, apiKey,
		"API key should be redacted from output")

	// Non-secret text should be preserved
	assert.Contains(t, result.Sanitized, "The API key is",
		"Non-secret prefix text should be preserved")
	assert.Contains(t, result.Sanitized, "and the request succeeded.",
		"Non-secret suffix text should be preserved")

	// Verify finding is present
	hasFinding := false
	for _, f := range result.Findings {
		if f.Scanner == "secret_redactor" {
			hasFinding = true
			break
		}
	}
	assert.True(t, hasFinding, "Should have a secret_redactor finding for the API key")
}

// TestGitHubPATRedactedFromOutput verifies that GitHub Personal Access Tokens
// (ghp_...) are detected and redacted from agent output text.
//
// [test_id:TS-GH-18-016]
func TestGitHubPATRedactedFromOutput(t *testing.T) {
	// Setup: Create output with GitHub PAT
	pat := "ghp_FAKEtesttoken000000000000000000000000"
	outputText := "Authentication token: " + pat

	// Execute: Run through output pipeline
	pipeline := security.OutputPipeline()
	result := pipeline.Scan(outputText)

	// Assert: PAT is redacted
	assert.False(t, result.Safe, "Output with GitHub PAT should be marked unsafe")
	assert.NotContains(t, result.Sanitized, "ghp_FAKEtest",
		"GitHub PAT should be fully redacted")

	// Verify the correct finding type
	hasPATFinding := false
	for _, f := range result.Findings {
		if f.Name == "github_pat" {
			hasPATFinding = true
			break
		}
	}
	assert.True(t, hasPATFinding, "Should have a github_pat finding")
}

// TestCleanTextPassesThroughUnchanged verifies that output text without
// any secret patterns passes through completely unchanged.
//
// [test_id:TS-GH-18-017]
func TestCleanTextPassesThroughUnchanged(t *testing.T) {
	// Setup: Create clean output text with no secrets
	cleanText := "The deployment completed successfully. All 42 tests passed in 3.2 seconds."

	// Execute: Run through output pipeline
	pipeline := security.OutputPipeline()
	result := pipeline.Scan(cleanText)

	// Assert: Text is unchanged
	assert.True(t, result.Safe, "Clean text should be marked safe")
	assert.Empty(t, result.Sanitized,
		"Clean text should not be modified (empty Sanitized means unchanged)")
	assert.Empty(t, result.Findings,
		"Clean text should produce no findings")
}
