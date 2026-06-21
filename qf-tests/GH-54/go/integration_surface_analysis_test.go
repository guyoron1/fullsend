package tests

/*
Integration Surface Analysis Tests

STP Reference: outputs/stp/GH-54/GH-54_test_plan.md
STD Reference: outputs/std/GH-54/GH-54_test_description.yaml
Jira: GH-54

Markers:
    - tier1

Preconditions:
    - GitHub Actions runner with internet access
    - Go 1.23+ toolchain installed
    - Evaluation document produced by GH-54 research task
*/

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestIntegrationSurfaceAnalysis validates that the GH-54 evaluation
// identifies regression-sensitive FullSend integration surfaces.
func TestIntegrationSurfaceAnalysis(t *testing.T) {
	docContent := readEvalDocument(t)

	t.Run("[test_id:TS-GH-54-004] should identify forge.Client interface as primary integration surface", func(t *testing.T) {
		// Search for forge.Client or forge interface references
		hasForgeClient := strings.Contains(docContent, "forge.Client")
		hasForge := strings.Contains(strings.ToLower(docContent), "forge")
		assert.True(t, hasForgeClient || hasForge,
			"Document should reference forge.Client or forge interface as an integration surface",
		)

		// Verify forge is discussed in integration or impact context
		forgeIntegration := regexp.MustCompile(`(?i)forge.*(integration|surface|impact|regression|abstraction)`)
		integrationForge := regexp.MustCompile(`(?i)(integration|surface|impact).*(forge)`)
		hasContext := forgeIntegration.MatchString(docContent) || integrationForge.MatchString(docContent)
		assert.True(t, hasContext,
			"forge.Client should be discussed in integration, impact, or abstraction context",
		)
	})

	t.Run("[test_id:TS-GH-54-005] should assess impact on harness/sandbox execution layer", func(t *testing.T) {
		lowerContent := strings.ToLower(docContent)

		// Search for harness/sandbox references
		hasHarness := strings.Contains(lowerContent, "harness")
		hasSandbox := strings.Contains(lowerContent, "sandbox")

		// Verify discussed in execution/impact context
		impactPattern := regexp.MustCompile(`(?i)(harness|sandbox).*(impact|execution|layer|integration)`)
		hasImpactContext := impactPattern.MatchString(docContent)

		assert.True(t, (hasHarness && hasSandbox) || hasImpactContext,
			"Document should reference harness and sandbox in execution/impact context",
		)
	})

	t.Run("[test_id:TS-GH-54-006] should document potential config.OrgConfig changes", func(t *testing.T) {
		// Search for config.OrgConfig or configuration references
		hasOrgConfig := strings.Contains(docContent, "OrgConfig")
		hasPerRepoConfig := strings.Contains(docContent, "PerRepoConfig")
		hasConfigOrg := strings.Contains(docContent, "config.Org")
		hasConfigRef := strings.Contains(strings.ToLower(docContent), "configuration")

		assert.True(t, hasOrgConfig || hasPerRepoConfig || hasConfigOrg || hasConfigRef,
			"Document should reference config.OrgConfig, PerRepoConfig, or configuration management",
		)
	})
}
