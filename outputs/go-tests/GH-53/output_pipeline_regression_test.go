package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/security"
)

// ============================================================
// TS-GH-53-018: OutputPipeline in run.go continues to function after
//
//	security package changes
//
// Priority: P2 | Tier: 2 | Type: Regression
// ============================================================

func TestOutputPipeline_RunGoConsumer_Regression(t *testing.T) {
	pipeline := security.OutputPipeline()
	require.NotNil(t, pipeline, "OutputPipeline() must return a valid pipeline")

	// Scan clean input — should pass through unchanged.
	cleanInput := "This is a normal log line with no secrets."
	result := pipeline.Scan(cleanInput)
	assert.True(t, result.Safe, "clean input should be marked safe")
	assert.Empty(t, result.Sanitized,
		"clean input should produce empty Sanitized (no changes)")

	// Scan input with a known secret — should produce sanitized output.
	secret := "ghp_FAKErunregression0000000000000000000"
	secretInput := "token=" + secret
	secretResult := pipeline.Scan(secretInput)
	assert.False(t, secretResult.Safe, "secret input should be marked unsafe")
	assert.NotEmpty(t, secretResult.Sanitized,
		"OutputPipeline must produce sanitized output for secret input")
	assert.NotContains(t, secretResult.Sanitized, "FAKErunregression",
		"OutputPipeline must redact GitHub PAT content from output")
	assert.NotEmpty(t, secretResult.Findings,
		"OutputPipeline must report findings for detected secret")
}

// ============================================================
// TS-GH-53-019: OutputPipeline in scan.go continues to function after
//
//	security package changes
//
// Priority: P2 | Tier: 2 | Type: Regression
// ============================================================

func TestOutputPipeline_ScanGoConsumer_Regression(t *testing.T) {
	pipeline := security.OutputPipeline()
	require.NotNil(t, pipeline, "OutputPipeline() must return a valid pipeline")

	// Verify pipeline handles multiple sequential scans (stateless).
	inputs := []struct {
		name      string
		text      string
		hasSecret bool
	}{
		{"clean_text", "Normal review comment.", false},
		{"github_pat", "Found ghp_FAKEscanregression00000000000000000000 in code.", true},
		{"clean_after_secret", "Another clean comment.", false},
	}

	for _, tc := range inputs {
		t.Run(tc.name, func(t *testing.T) {
			result := pipeline.Scan(tc.text)
			if tc.hasSecret {
				assert.False(t, result.Safe, "pipeline should detect secret")
				assert.NotEmpty(t, result.Findings, "pipeline should report findings")
				assert.NotEmpty(t, result.Sanitized, "pipeline should produce sanitized output")
			} else {
				assert.True(t, result.Safe, "clean input should be safe")
				assert.Empty(t, result.Sanitized, "clean input should not be modified")
			}
		})
	}
}
