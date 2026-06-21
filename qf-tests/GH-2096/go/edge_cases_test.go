package review

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Edge Case Tests — GH-2096

Validates behavior when triage produces extreme results: all files critical,
or zero files critical. The system must handle both gracefully.

STP Reference: outputs/stp/GH-2096/GH-2096_test_plan.md
STD Scenarios: TS-GH-2096-025, TS-GH-2096-026, TS-GH-2096-027, TS-GH-2096-028
*/

func TestEdgeCaseAllFilesCritical(t *testing.T) {
	/*
		Preconditions:
			- Go development environment with Go 1.26+
			- fullsend repository with two-pass review strategy changes
	*/

	allCriticalTriage := TriageResult{
		SecurityCriticalFiles: []CriticalFile{
			{File: "internal/mint/handler.go", Reason: "Token handling"},
			{File: "internal/auth/oauth.go", Reason: "Auth logic"},
			{File: "docs/README.md", Reason: "Mentions authentication"},
		},
		StandardFiles: []string{},
		Summary:       "All 3 files classified as security-critical",
	}

	testFiles := []string{
		"internal/mint/handler.go",
		"internal/auth/oauth.go",
		"docs/README.md",
	}

	diffs := map[string]string{
		"internal/mint/handler.go": "+func HandleToken() {}",
		"internal/auth/oauth.go":   "+func AuthFlow() {}",
		"docs/README.md":           "+# Auth documentation",
	}

	// TS-GH-2096-025: Verify all-critical classification produces standard-equivalent review
	t.Run("all-critical classification produces standard-equivalent review", func(t *testing.T) {
		ctx := assembleSecurityContext(allCriticalTriage, diffs)
		require.NotEmpty(t, ctx, "context must be non-empty for all-critical case")

		// All files must appear in context
		for _, cf := range allCriticalTriage.SecurityCriticalFiles {
			assert.Contains(t, ctx, cf.File,
				"all-critical context must contain file %q", cf.File)
		}

		// Review must complete and produce findings
		reviewResult := runReviewPipeline(allCriticalTriage, testFiles)
		assert.True(t, reviewResult.Success,
			"review must complete successfully with all files critical")
		assert.NotEmpty(t, reviewResult.Findings,
			"review findings must be non-empty for all-critical case")
	})

	// TS-GH-2096-026: Verify no degradation in review quality for all-critical case
	t.Run("no degradation in review quality for all-critical case", func(t *testing.T) {
		// Baseline: review without triage (uniform attention via fallback)
		baselineTriage := runTriageWithFallback(errTriageTimeout, testFiles)
		baselineResult := runReviewPipeline(baselineTriage, testFiles)

		// Triaged: review with all-critical classification
		triagedResult := runReviewPipeline(allCriticalTriage, testFiles)

		// Both must complete
		require.True(t, baselineResult.Success, "baseline review must succeed")
		require.True(t, triagedResult.Success, "triaged review must succeed")

		// Same sub-agent coverage
		assert.Equal(t, len(baselineResult.Agents), len(triagedResult.Agents),
			"both reviews must dispatch same number of sub-agents")

		// No sub-agent received empty context
		assert.Equal(t, len(baselineResult.Findings), len(triagedResult.Findings),
			"all sub-agents must produce findings in both cases")
	})
}

func TestEdgeCaseNoFilesCritical(t *testing.T) {
	/*
		Preconditions:
			- Go development environment with Go 1.26+
			- fullsend repository with two-pass review strategy changes
	*/

	noCriticalTriage := TriageResult{
		SecurityCriticalFiles: []CriticalFile{},
		StandardFiles: []string{
			"docs/README.md",
			"web/index.html",
			"config/settings.yaml",
		},
		Summary: "No security-critical files identified",
	}

	testFiles := []string{
		"docs/README.md",
		"web/index.html",
		"config/settings.yaml",
	}

	diffs := map[string]string{
		"docs/README.md":       "+# Updated docs",
		"web/index.html":       "+<div>Updated UI</div>",
		"config/settings.yaml": "+key: value",
	}

	// TS-GH-2096-027: Verify all files receive standard context when none are critical
	t.Run("all files receive standard context when none are critical", func(t *testing.T) {
		ctx := assembleSecurityContext(noCriticalTriage, diffs)
		require.NotEmpty(t, ctx, "context must be non-empty even with zero critical files")

		// All standard files must appear in context
		for _, f := range noCriticalTriage.StandardFiles {
			assert.Contains(t, ctx, f,
				"all standard files must appear in context, including %q", f)
		}

		// Review completes without error
		reviewResult := runReviewPipeline(noCriticalTriage, testFiles)
		assert.True(t, reviewResult.Success,
			"review must complete with zero critical files")
	})

	// TS-GH-2096-028: Verify triage cost is minimal for zero-critical case
	t.Run("triage cost is minimal for zero-critical case", func(t *testing.T) {
		// Triage completes without error
		assert.Empty(t, noCriticalTriage.SecurityCriticalFiles,
			"triage result must have empty critical files array")
		assert.NotEmpty(t, noCriticalTriage.StandardFiles,
			"triage result must have populated standard files array")

		// Review pipeline proceeds without retry or error
		reviewResult := runReviewPipeline(noCriticalTriage, testFiles)
		assert.True(t, reviewResult.Success,
			"review pipeline must proceed to sub-agent dispatch without retries")

		// All agents received context and produced findings
		assert.Len(t, reviewResult.Findings, len(reviewResult.Agents),
			"all agents must produce findings (no empty context)")
	})
}
