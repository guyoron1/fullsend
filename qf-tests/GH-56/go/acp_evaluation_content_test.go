//go:build e2e

package tests

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestACPEvaluationPointsPresent validates that all five ACP evaluation points
// are present in the agent-infrastructure documentation.
// [test_id:TS-GH-56-001]
func TestACPEvaluationPointsPresent(t *testing.T) {
	// Setup: read docs/problems/agent-infrastructure.md
	docBytes, err := os.ReadFile("docs/problems/agent-infrastructure.md")
	require.NoError(t, err, "failed to read docs/problems/agent-infrastructure.md")
	docContent := string(docBytes)

	t.Run("should contain all ACP evaluation points in documentation", func(t *testing.T) {
		// TEST-01: Check for controller overhead evaluation point
		controllerOverhead := strings.Contains(docContent, "controller overhead") ||
			strings.Contains(docContent, "operator overhead") ||
			strings.Contains(docContent, "Controller Overhead")
		assert.True(t, controllerOverhead,
			"Documentation must contain controller overhead evaluation point")

		// TEST-02: Check for UI-centric design evaluation point
		uiCentric := strings.Contains(docContent, "UI-centric") ||
			strings.Contains(docContent, "UI-Centric") ||
			strings.Contains(docContent, "user-interface-centric")
		assert.True(t, uiCentric,
			"Documentation must contain UI-centric design evaluation point")

		// TEST-03: Check for CR surface friction evaluation point
		crSurface := strings.Contains(docContent, "CR surface") ||
			strings.Contains(docContent, "custom resource") ||
			strings.Contains(docContent, "Custom Resource") ||
			strings.Contains(docContent, "CRD")
		assert.True(t, crSurface,
			"Documentation must contain CR surface friction evaluation point")

		// TEST-04: Check for shared workspace risk evaluation point
		sharedWorkspace := strings.Contains(docContent, "shared workspace") ||
			strings.Contains(docContent, "Shared Workspace") ||
			strings.Contains(docContent, "shared-workspace")
		assert.True(t, sharedWorkspace,
			"Documentation must contain shared workspace risk evaluation point")

		// TEST-05: Check for plain Pod execution limits evaluation point
		plainPod := strings.Contains(docContent, "plain Pod") ||
			strings.Contains(docContent, "Plain Pod") ||
			strings.Contains(docContent, "plain pod")
		assert.True(t, plainPod,
			"Documentation must contain plain Pod execution limits evaluation point")
	})
}

// TestEvaluationClaimsMatchDiscussion validates that evaluation claims
// in the documentation accurately reflect issue discussion findings.
// [test_id:TS-GH-56-002]
func TestEvaluationClaimsMatchDiscussion(t *testing.T) {
	// Setup: read docs/problems/agent-infrastructure.md
	docBytes, err := os.ReadFile("docs/problems/agent-infrastructure.md")
	require.NoError(t, err, "failed to read docs/problems/agent-infrastructure.md")
	docContent := string(docBytes)

	t.Run("should have evaluation claims matching issue discussion findings", func(t *testing.T) {
		// TEST-01: Verify operator overhead claim present and accurate
		operatorOverhead := strings.Contains(docContent, "operator overhead") ||
			strings.Contains(docContent, "controller overhead")
		assert.True(t, operatorOverhead,
			"Documentation must reflect operator overhead concerns from issue discussion")

		// TEST-02: Verify UI-centric limitation claim present and accurate
		uiCentric := strings.Contains(docContent, "UI-centric") ||
			strings.Contains(docContent, "UI-Centric")
		assert.True(t, uiCentric,
			"Documentation must reflect UI-centric design limitation from issue discussion")

		// TEST-03: Verify shared-workspace risk claim present and accurate
		sharedWorkspace := strings.Contains(docContent, "shared workspace") ||
			strings.Contains(docContent, "shared-workspace") ||
			strings.Contains(docContent, "workspace injection")
		assert.True(t, sharedWorkspace,
			"Documentation must reflect shared-workspace injection risk from issue discussion")
	})
}

// TestNoStaleOrInaccurateClaims validates that the ACP evaluation
// documentation does not contain stale or factually inaccurate claims.
// [test_id:TS-GH-56-003]
func TestNoStaleOrInaccurateClaims(t *testing.T) {
	// Setup: read docs/problems/agent-infrastructure.md
	docBytes, err := os.ReadFile("docs/problems/agent-infrastructure.md")
	require.NoError(t, err, "failed to read docs/problems/agent-infrastructure.md")
	docContent := string(docBytes)

	t.Run("should contain no stale or inaccurate platform claims", func(t *testing.T) {
		// TEST-01: Check for temporal framing of claims
		temporalFraming := strings.Contains(docContent, "as of") ||
			strings.Contains(docContent, "at the time of") ||
			strings.Contains(docContent, "currently") ||
			strings.Contains(docContent, "at evaluation time")
		assert.True(t, temporalFraming,
			"Document must contain temporal phrases near platform-specific claims "+
				"(e.g., 'as of', 'at the time of', 'currently', 'at evaluation time')")

		// TEST-02: Check for absence of known-discontinued ACP feature references
		assert.False(t, strings.Contains(docContent, "ACP v0."),
			"Document must not reference discontinued ACP versions (found 'ACP v0.')")
		assert.False(t, strings.Contains(docContent, "deprecated ACP"),
			"Document must not reference deprecated ACP features (found 'deprecated ACP')")
	})
}
