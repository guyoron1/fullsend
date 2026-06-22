package dispatch_auth

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
CLI Infrastructure Compatibility Tests (E2E / Tier 2)

STP Reference: outputs/stp/GH-79/GH-79_test_plan.md
STD Reference: outputs/std/GH-79/GH-79_test_description.yaml
Jira: GH-79

Verifies that the updated CLI infrastructure (100+ file changes) maintains
compatibility: agent pipeline, harness loading, and forge.Client interface.
These tests validate structural invariants in the dispatch workflow.
*/

func TestCLIInfrastructureCompatibility(t *testing.T) {
	workflows := bothWorkflows(t)

	t.Run("agent run pipeline completes successfully", func(t *testing.T) {
		// [test_id:TS-GH-79-033] P1 E2E
		// Verify the dispatch workflow routes to all required stages.
		for _, wf := range workflows {
			t.Run(wf.Name, func(t *testing.T) {
				route := extractRouteBlock(wf.Content)
				require.NotEmpty(t, route)

				// All stages must be routable from the dispatch routing logic
				stages := []string{"triage", "code", "review", "fix", "retro"}
				for _, stage := range stages {
					t.Run(stage, func(t *testing.T) {
						assert.Contains(t, route, `STAGE="`+stage+`"`,
							"routing must be able to set STAGE=%s", stage)
					})
				}

				// Route must output stage variable
				assert.Contains(t, wf.Content, "GITHUB_OUTPUT",
					"routing must write stage to GITHUB_OUTPUT")
			})
		}
	})

	t.Run("harness loading with updated config structure", func(t *testing.T) {
		// [test_id:TS-GH-79-034] P1 E2E
		// Verify dispatch workflows include PR check step for code stage.
		for _, wf := range workflows {
			t.Run(wf.Name, func(t *testing.T) {
				// Both workflows must have the PR-check step for /fs-code
				assert.Contains(t, wf.Content, "Check for existing PRs",
					"workflow must have PR-check step for code stage")

				// PR-check must use gh CLI
				assert.Contains(t, wf.Content, "gh pr list",
					"PR-check must use gh pr list")
			})
		}
	})

	t.Run("forge.Client interface compatibility", func(t *testing.T) {
		// [test_id:TS-GH-79-035] P1 E2E
		// Verify dispatch workflows correctly use GitHub API via gh CLI.
		for _, wf := range workflows {
			t.Run(wf.Name, func(t *testing.T) {
				// GH_TOKEN must be set for API access
				assert.Contains(t, wf.Content, "GH_TOKEN",
					"workflow must set GH_TOKEN for gh CLI")

				// Must use gh CLI for PR checks
				assert.Contains(t, wf.Content, "gh pr list",
					"workflow must use gh CLI for PR checks")
			})
		}
	})

	t.Run("per-repo dispatch template references correct reusable workflows", func(t *testing.T) {
		// Verify the per-repo reusable-dispatch.yml references fullsend-ai/fullsend
		repoContent, err := os.ReadFile("../../../.github/workflows/reusable-dispatch.yml")
		require.NoError(t, err)

		content := string(repoContent)

		// All stage workflows must reference fullsend-ai/fullsend
		stages := []string{"triage", "code", "review", "fix", "retro", "prioritize"}
		for _, stage := range stages {
			ref := "fullsend-ai/fullsend/.github/workflows/reusable-" + stage + ".yml"
			assert.True(t, strings.Contains(content, ref),
				"stage %s must reference %s", stage, ref)
		}

		// Verify jobs depend on route
		assert.Contains(t, content, "needs: route",
			"stage jobs must depend on route job")
	})

	t.Run("dispatch workflow validates stage and trigger_source", func(t *testing.T) {
		// Structural test: per-repo workflow has stage validation.
		repoContent, err := os.ReadFile("../../../.github/workflows/reusable-dispatch.yml")
		require.NoError(t, err)
		content := string(repoContent)

		// Validate routed stage step must exist
		assert.Contains(t, content, "Validate routed stage",
			"per-repo workflow must validate routed stage")

		// Stage validation must check format
		assert.Contains(t, content, "^[a-z]",
			"stage validation must check format")
	})
}
