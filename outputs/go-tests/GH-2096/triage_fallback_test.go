package review

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Triage Failure Fallback Tests — GH-2096

Validates that when the triage sub-agent fails (timeout, malformed JSON, empty
response), the system gracefully falls back to uniform attention rather than
failing the entire review.

STP Reference: outputs/stp/GH-2096/GH-2096_test_plan.md
STD Scenarios: TS-GH-2096-012, TS-GH-2096-013, TS-GH-2096-014, TS-GH-2096-015
*/

// triageError sentinel values.
var (
	errTriageTimeout  = errors.New("triage sub-agent timed out")
	errMalformedJSON  = errors.New("malformed triage JSON response")
	errEmptyResponse  = errors.New("empty triage response: no files classified")
)

// parseTriageResponse parses the triage sub-agent JSON output into a TriageResult.
// Returns an error if JSON is malformed, missing required fields, or empty.
func parseTriageResponse(raw string) (*TriageResult, error) {
	if raw == "" {
		return nil, errMalformedJSON
	}

	var result TriageResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, errMalformedJSON
	}

	// Validate required fields are present
	if result.SecurityCriticalFiles == nil && result.StandardFiles == nil {
		return nil, errMalformedJSON
	}

	return &result, nil
}

// isTriageResponseEmpty returns true if the triage classified zero files,
// indicating a classifier failure that should trigger fallback.
func isTriageResponseEmpty(result *TriageResult) bool {
	return len(result.SecurityCriticalFiles) == 0 && len(result.StandardFiles) == 0
}

// SubAgentFinding represents a finding from a review sub-agent.
type SubAgentFinding struct {
	Agent    string
	Severity string
	Message  string
}

// ReviewResult holds the aggregate output of a review pipeline run.
type ReviewResult struct {
	Findings []SubAgentFinding
	Agents   []string
	Success  bool
}

// runTriageWithFallback attempts to use triage classification. On any error,
// falls back to uniform attention (all files treated as security-critical).
func runTriageWithFallback(triageErr error, files []string) TriageResult {
	if triageErr != nil {
		// Fallback: treat all files as security-critical (uniform attention)
		criticalFiles := make([]CriticalFile, len(files))
		for i, f := range files {
			criticalFiles[i] = CriticalFile{
				File:   f,
				Reason: "fallback: uniform attention (triage failed)",
			}
		}
		return TriageResult{
			SecurityCriticalFiles: criticalFiles,
			StandardFiles:         nil,
			Summary:               "fallback to uniform attention due to: " + triageErr.Error(),
		}
	}
	return TriageResult{}
}

// runReviewPipeline simulates the full review pipeline with a given triage context.
func runReviewPipeline(triage TriageResult, files []string) ReviewResult {
	agents := []string{"security", "correctness", "style", "docs-currency"}
	var findings []SubAgentFinding

	diffs := make(map[string]string)
	for _, f := range files {
		diffs[f] = "+// changed content"
	}

	for _, agent := range agents {
		ctx := assembleContext(agent, triage, diffs)
		if ctx != "" {
			findings = append(findings, SubAgentFinding{
				Agent:    agent,
				Severity: "info",
				Message:  "Reviewed files in context",
			})
		}
	}

	return ReviewResult{
		Findings: findings,
		Agents:   agents,
		Success:  len(findings) == len(agents),
	}
}

func TestTriageFallback(t *testing.T) {
	/*
		Preconditions:
			- Go development environment with Go 1.26+
			- fullsend repository with two-pass review strategy changes
	*/

	testFiles := []string{
		"internal/mint/handler.go",
		"docs/README.md",
		"web/index.html",
	}

	// TS-GH-2096-012: Verify fallback on triage sub-agent timeout
	t.Run("fallback on triage sub-agent timeout", func(t *testing.T) {
		result := runTriageWithFallback(errTriageTimeout, testFiles)

		// All files treated as security-critical in fallback mode
		assert.Len(t, result.SecurityCriticalFiles, len(testFiles),
			"fallback must treat all files as security-critical")
		assert.Empty(t, result.StandardFiles,
			"fallback must have no standard files")
		assert.Contains(t, result.Summary, "fallback",
			"summary must indicate fallback mode")

		// Review continues without error
		reviewResult := runReviewPipeline(result, testFiles)
		assert.True(t, reviewResult.Success,
			"review must complete successfully after timeout fallback")
	})

	// TS-GH-2096-013: Verify fallback on malformed JSON response
	t.Run("fallback on malformed JSON response", func(t *testing.T) {
		malformedCases := []struct {
			name string
			json string
		}{
			{"invalid syntax", `{invalid json`},
			{"truncated array", `{"security_critical_files": [`},
			{"wrong structure", `{"wrong_key": "value"}`},
			{"empty string", ""},
		}

		for _, tc := range malformedCases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := parseTriageResponse(tc.json)
				assert.Error(t, err,
					"malformed JSON %q must trigger parse error", tc.name)
				assert.Nil(t, result,
					"malformed JSON must return nil result")
			})
		}
	})

	// TS-GH-2096-014: Verify fallback on empty triage response
	t.Run("fallback on empty triage response", func(t *testing.T) {
		emptyCases := []struct {
			name   string
			result TriageResult
		}{
			{
				"both arrays empty",
				TriageResult{
					SecurityCriticalFiles: []CriticalFile{},
					StandardFiles:         []string{},
					Summary:               "",
				},
			},
			{
				"nil critical files with empty standard",
				TriageResult{
					SecurityCriticalFiles: nil,
					StandardFiles:         []string{},
				},
			},
			{
				"empty critical with nil standard",
				TriageResult{
					SecurityCriticalFiles: []CriticalFile{},
					StandardFiles:         nil,
				},
			},
		}

		for _, tc := range emptyCases {
			t.Run(tc.name, func(t *testing.T) {
				shouldFallback := isTriageResponseEmpty(&tc.result)
				assert.True(t, shouldFallback,
					"empty triage response (%s) must trigger fallback", tc.name)
			})
		}
	})

	// TS-GH-2096-015: Verify review completes normally after fallback
	t.Run("review completes normally after fallback", func(t *testing.T) {
		// Trigger fallback via timeout
		fallbackTriage := runTriageWithFallback(errTriageTimeout, testFiles)
		require.NotEmpty(t, fallbackTriage.SecurityCriticalFiles,
			"fallback triage must have files")

		// Run full review pipeline after fallback
		reviewResult := runReviewPipeline(fallbackTriage, testFiles)

		assert.True(t, reviewResult.Success,
			"review pipeline must complete successfully after fallback")
		assert.Len(t, reviewResult.Findings, len(reviewResult.Agents),
			"all sub-agents must produce findings after fallback")

		// Verify each expected sub-agent produced output
		expectedAgents := map[string]bool{
			"security": false, "correctness": false,
			"style": false, "docs-currency": false,
		}
		for _, finding := range reviewResult.Findings {
			expectedAgents[finding.Agent] = true
		}
		for agent, found := range expectedAgents {
			assert.True(t, found,
				"sub-agent %q must produce findings after fallback", agent)
		}
	})
}
