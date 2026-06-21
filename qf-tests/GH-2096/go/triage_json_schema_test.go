package review

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Triage Output JSON Schema Tests — GH-2096

Validates that the triage JSON output is correctly parsed, rejects incomplete
input, and tolerates extra fields from non-deterministic LLM outputs.

STP Reference: outputs/stp/GH-2096/GH-2096_test_plan.md
STD Scenarios: TS-GH-2096-022, TS-GH-2096-023, TS-GH-2096-024
*/

func TestTriageJSONSchema(t *testing.T) {
	/*
		Preconditions:
			- Go development environment with Go 1.26+
			- fullsend repository with two-pass review strategy changes
	*/

	// TS-GH-2096-022: Verify valid triage JSON parsed by context assembly
	t.Run("valid triage JSON parsed by context assembly", func(t *testing.T) {
		validJSON := `{
			"security_critical_files": [
				{"file": "internal/mint/handler.go", "reason": "Token handling"},
				{"file": "internal/auth/oauth.go", "reason": "Auth logic"}
			],
			"standard_files": ["docs/README.md", "web/index.html"],
			"summary": "2 security-critical files, 2 standard files"
		}`

		result, err := parseTriageResponse(validJSON)
		require.NoError(t, err, "valid JSON must parse without error")
		require.NotNil(t, result, "result must not be nil")

		assert.Len(t, result.SecurityCriticalFiles, 2,
			"security_critical_files must have 2 entries")
		assert.Equal(t, "internal/mint/handler.go", result.SecurityCriticalFiles[0].File)
		assert.Equal(t, "Token handling", result.SecurityCriticalFiles[0].Reason)

		assert.Len(t, result.StandardFiles, 2,
			"standard_files must have 2 entries")
		assert.Equal(t, "docs/README.md", result.StandardFiles[0])

		assert.Contains(t, result.Summary, "2 security-critical",
			"summary must be parsed")
	})

	// TS-GH-2096-023: Verify rejection of triage JSON missing required fields
	t.Run("rejection of triage JSON missing required fields", func(t *testing.T) {
		incompleteCases := []struct {
			name string
			json string
		}{
			{
				"missing security_critical_files",
				`{"standard_files": ["a.go"]}`,
			},
			{
				"missing standard_files",
				`{"security_critical_files": [{"file":"a.go","reason":"x"}]}`,
			},
			{
				"both fields null",
				`{"security_critical_files": null, "standard_files": null}`,
			},
		}

		for _, tc := range incompleteCases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := parseTriageResponse(tc.json)
				assert.Error(t, err,
					"JSON with %s must trigger parse error", tc.name)
				assert.Nil(t, result,
					"result must be nil for incomplete JSON")
			})
		}
	})

	// TS-GH-2096-024: Verify handling of extra unexpected fields in triage JSON
	t.Run("handling of extra unexpected fields in triage JSON", func(t *testing.T) {
		extraFieldsJSON := `{
			"security_critical_files": [{"file": "a.go", "reason": "auth"}],
			"standard_files": ["b.go"],
			"summary": "1 critical",
			"confidence": 0.95,
			"model_notes": "Extra field from LLM",
			"reasoning_trace": "I classified based on..."
		}`

		result, err := parseTriageResponse(extraFieldsJSON)
		require.NoError(t, err,
			"JSON with extra fields must parse successfully")
		require.NotNil(t, result)

		// Expected fields extracted correctly
		assert.Len(t, result.SecurityCriticalFiles, 1,
			"expected fields must be extracted correctly")
		assert.Equal(t, "a.go", result.SecurityCriticalFiles[0].File)
		assert.Equal(t, "auth", result.SecurityCriticalFiles[0].Reason)
		assert.Len(t, result.StandardFiles, 1)
		assert.Equal(t, "b.go", result.StandardFiles[0])
		assert.Equal(t, "1 critical", result.Summary)
	})
}
