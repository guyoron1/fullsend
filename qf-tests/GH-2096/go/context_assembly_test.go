package review

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Context Assembly Tests — GH-2096

Validates that security-prioritized context packages are assembled correctly,
with critical files placed before standard files for security and correctness
sub-agents, while non-security sub-agents receive unmodified context.

STP Reference: outputs/stp/GH-2096/GH-2096_test_plan.md
STD Scenarios: TS-GH-2096-008, TS-GH-2096-009, TS-GH-2096-010, TS-GH-2096-011
*/

// CriticalFile represents a file classified as security-critical by triage.
type CriticalFile struct {
	File   string `json:"file"`
	Reason string `json:"reason"`
}

// TriageResult holds the output of the security-triage sub-agent.
type TriageResult struct {
	SecurityCriticalFiles []CriticalFile `json:"security_critical_files"`
	StandardFiles         []string       `json:"standard_files"`
	Summary               string         `json:"summary"`
}

const (
	securityCriticalHeader = "## SECURITY-CRITICAL FILES"
	standardHeader         = "## STANDARD FILES"
)

// assembleSecurityContext builds a context package for the security or
// correctness sub-agent with critical files prioritized before standard files.
func assembleSecurityContext(triage TriageResult, diffs map[string]string) string {
	var sb strings.Builder

	sb.WriteString(securityCriticalHeader + "\n\n")
	for _, cf := range triage.SecurityCriticalFiles {
		sb.WriteString(fmt.Sprintf("### %s\nReason: %s\n\n", cf.File, cf.Reason))
		if diff, ok := diffs[cf.File]; ok {
			sb.WriteString(diff + "\n\n")
		}
	}

	sb.WriteString(standardHeader + "\n\n")
	for _, f := range triage.StandardFiles {
		sb.WriteString(fmt.Sprintf("### %s\n\n", f))
		if diff, ok := diffs[f]; ok {
			sb.WriteString(diff + "\n\n")
		}
	}

	return sb.String()
}

// assembleCorrectnessContext builds a context package for the correctness
// sub-agent. Uses the same prioritized ordering as security context.
func assembleCorrectnessContext(triage TriageResult, diffs map[string]string) string {
	return assembleSecurityContext(triage, diffs)
}

// assembleStandardContext builds a context package for non-security sub-agents.
// Files appear in their original order without prioritization.
func assembleStandardContext(allFiles []string, diffs map[string]string) string {
	var sb strings.Builder
	for _, f := range allFiles {
		sb.WriteString(fmt.Sprintf("### %s\n\n", f))
		if diff, ok := diffs[f]; ok {
			sb.WriteString(diff + "\n\n")
		}
	}
	return sb.String()
}

// assembleContext dispatches to the correct assembly function based on sub-agent type.
func assembleContext(agentType string, triage TriageResult, diffs map[string]string) string {
	switch agentType {
	case "security", "correctness":
		return assembleSecurityContext(triage, diffs)
	default:
		allFiles := make([]string, 0, len(triage.SecurityCriticalFiles)+len(triage.StandardFiles))
		for _, cf := range triage.SecurityCriticalFiles {
			allFiles = append(allFiles, cf.File)
		}
		allFiles = append(allFiles, triage.StandardFiles...)
		return assembleStandardContext(allFiles, diffs)
	}
}

func TestContextAssembly(t *testing.T) {
	/*
		Preconditions:
			- Go development environment with Go 1.26+
			- fullsend repository with two-pass review strategy changes
			- Valid triage classification output available
	*/

	// Shared test fixtures
	triageResult := TriageResult{
		SecurityCriticalFiles: []CriticalFile{
			{File: "internal/mint/handler.go", Reason: "Token handling logic"},
			{File: "internal/mintcore/wif.go", Reason: "WIF verification"},
		},
		StandardFiles: []string{
			"docs/README.md",
			"web/index.html",
		},
		Summary: "2 security-critical files identified",
	}

	diffs := map[string]string{
		"internal/mint/handler.go": "+func HandleToken(ctx context.Context) error {",
		"internal/mintcore/wif.go": "+func VerifyWIF(claims *Claims) error {",
		"docs/README.md":           "+# Updated documentation",
		"web/index.html":           "+<div>Updated UI</div>",
	}

	// TS-GH-2096-008: Verify security sub-agent receives critical files first
	t.Run("security sub-agent receives critical files first", func(t *testing.T) {
		ctx := assembleSecurityContext(triageResult, diffs)
		require.NotEmpty(t, ctx, "context package must be non-empty")

		// Critical files must appear before standard files
		criticalIdx := strings.Index(ctx, "internal/mint/handler.go")
		standardIdx := strings.Index(ctx, "docs/README.md")
		require.NotEqual(t, -1, criticalIdx, "critical file must appear in context")
		require.NotEqual(t, -1, standardIdx, "standard file must appear in context")
		assert.Less(t, criticalIdx, standardIdx,
			"critical file must appear before standard file in security context")

		// All security-critical files present
		for _, cf := range triageResult.SecurityCriticalFiles {
			assert.Contains(t, ctx, cf.File,
				"security-critical file %q must appear in context", cf.File)
		}
	})

	// TS-GH-2096-009: Verify correctness sub-agent receives critical files first
	t.Run("correctness sub-agent receives critical files first", func(t *testing.T) {
		ctx := assembleCorrectnessContext(triageResult, diffs)
		require.NotEmpty(t, ctx)

		criticalIdx := strings.Index(ctx, "internal/mint/handler.go")
		standardIdx := strings.Index(ctx, "docs/README.md")
		assert.Less(t, criticalIdx, standardIdx,
			"correctness sub-agent must receive critical files before standard files")

		// Structure must match security sub-agent format
		secCtx := assembleSecurityContext(triageResult, diffs)
		assert.Equal(t, secCtx, ctx,
			"correctness context structure must match security context format")
	})

	// TS-GH-2096-010: Verify other sub-agents receive standard context
	t.Run("other sub-agents receive standard context", func(t *testing.T) {
		styleCtx := assembleContext("style", triageResult, diffs)
		require.NotEmpty(t, styleCtx)

		// Non-security agents should NOT have priority headers
		assert.NotContains(t, styleCtx, securityCriticalHeader,
			"style sub-agent must not receive security-critical header")
		assert.NotContains(t, styleCtx, standardHeader,
			"style sub-agent must not receive standard header")

		// All files should be present (both critical and standard)
		for _, cf := range triageResult.SecurityCriticalFiles {
			assert.Contains(t, styleCtx, cf.File,
				"style sub-agent must receive all files including %q", cf.File)
		}
		for _, f := range triageResult.StandardFiles {
			assert.Contains(t, styleCtx, f,
				"style sub-agent must receive all files including %q", f)
		}
	})

	// TS-GH-2096-011: Verify classification headers present in prioritized context
	t.Run("classification headers present in prioritized context", func(t *testing.T) {
		ctx := assembleSecurityContext(triageResult, diffs)

		assert.Contains(t, ctx, securityCriticalHeader,
			"prioritized context must contain SECURITY-CRITICAL header")
		assert.Contains(t, ctx, standardHeader,
			"prioritized context must contain STANDARD header")

		// Headers appear at correct positions
		criticalHeaderIdx := strings.Index(ctx, securityCriticalHeader)
		standardHeaderIdx := strings.Index(ctx, standardHeader)
		assert.Less(t, criticalHeaderIdx, standardHeaderIdx,
			"SECURITY-CRITICAL header must appear before STANDARD header")

		// First critical file appears after the critical header
		firstCriticalFile := strings.Index(ctx, triageResult.SecurityCriticalFiles[0].File)
		assert.Greater(t, firstCriticalFile, criticalHeaderIdx,
			"first critical file must appear after SECURITY-CRITICAL header")
	})
}
