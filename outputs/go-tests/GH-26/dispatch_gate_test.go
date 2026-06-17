package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

/*
Dispatch Workflow Pre-Flight Gate Tests

STP Reference: outputs/stp/GH-26/GH-26_test_plan.md
STD Reference: outputs/std/GH-26/GH-26_test_description.yaml
Jira: GH-26

Tests for the dispatch.yml workflow pre-flight check that prevents
code agent invocation when open PRs already exist for the target issue.
These tests validate the YAML structure of the dispatch workflow to
ensure the pr-check gate is properly configured.
*/

//go:build e2e

const dispatchYAMLRelPath = "internal/scaffold/fullsend-repo/.github/workflows/dispatch.yml"

// findDispatchYAML locates dispatch.yml from the repo root.
func findDispatchYAML(t *testing.T) string {
	t.Helper()
	candidates := []string{
		dispatchYAMLRelPath,
		filepath.Join("..", dispatchYAMLRelPath),
		filepath.Join("..", "..", dispatchYAMLRelPath),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	root := os.Getenv("REPO_ROOT")
	if root != "" {
		return filepath.Join(root, dispatchYAMLRelPath)
	}
	t.Skip("dispatch.yml not found — set REPO_ROOT or run from repo root")
	return ""
}

// workflowStep represents a step in a GitHub Actions workflow.
type workflowStep struct {
	Name string `yaml:"name"`
	ID   string `yaml:"id"`
	If   string `yaml:"if"`
	Run  string `yaml:"run"`
}

// workflowJob represents a job in a GitHub Actions workflow.
type workflowJob struct {
	Steps []workflowStep `yaml:"steps"`
}

// workflowFile represents a GitHub Actions workflow file.
type workflowFile struct {
	Jobs map[string]workflowJob `yaml:"jobs"`
}

func parseWorkflow(t *testing.T, path string) workflowFile {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err, "failed to read workflow file: %s", path)

	var wf workflowFile
	require.NoError(t, yaml.Unmarshal(data, &wf), "failed to parse YAML: %s", path)
	return wf
}

func findStepByID(steps []workflowStep, id string) *workflowStep {
	for i, s := range steps {
		if s.ID == id {
			return &steps[i]
		}
	}
	return nil
}

// TestDispatchBlocksCodeStageOnExistingPR verifies that the dispatch
// workflow pre-flight check blocks code stage invocation when an open
// PR exists for the target issue.
//
// [test_id:TS-GH-26-013]
//
// Validates:
//   - pr-check step exists in dispatch.yml
//   - pr-check step runs only for stage==code
//   - pr-check sets skipped=true when human PRs found
func TestDispatchBlocksCodeStageOnExistingPR(t *testing.T) {
	yamlPath := findDispatchYAML(t)
	wf := parseWorkflow(t, yamlPath)

	// Find the job that contains the pr-check step
	var prCheckStep *workflowStep
	for _, job := range wf.Jobs {
		step := findStepByID(job.Steps, "pr-check")
		if step != nil {
			prCheckStep = step
			break
		}
	}
	require.NotNil(t, prCheckStep, "dispatch.yml must have a step with id 'pr-check'")

	// The pr-check step must only run for code stage
	assert.Contains(t, prCheckStep.If, "code",
		"pr-check step should have 'if' condition gating on stage==code")

	// The step's run block must search for PRs and set skipped=true
	assert.Contains(t, prCheckStep.Run, "gh pr list",
		"pr-check step should search for PRs using gh pr list")
	assert.Contains(t, prCheckStep.Run, "skipped=true",
		"pr-check step should set skipped=true when PRs found")
}

// TestDispatchAllowsNonCodeStages verifies that the dispatch pr-check
// only applies to stage=code and does not block other stages.
//
// [test_id:TS-GH-26-014]
//
// Validates:
//   - pr-check step condition explicitly checks for 'code' stage
//   - Non-code stages (triage, review, fix) are not gated by pr-check
func TestDispatchAllowsNonCodeStages(t *testing.T) {
	yamlPath := findDispatchYAML(t)
	wf := parseWorkflow(t, yamlPath)

	var prCheckStep *workflowStep
	for _, job := range wf.Jobs {
		step := findStepByID(job.Steps, "pr-check")
		if step != nil {
			prCheckStep = step
			break
		}
	}
	require.NotNil(t, prCheckStep, "dispatch.yml must have a step with id 'pr-check'")

	// The condition must be specific to 'code' stage
	assert.Contains(t, prCheckStep.If, "'code'",
		"pr-check condition should specifically check for stage 'code'")

	// Verify that downstream steps check pr-check.outputs.skipped
	// but also include stage != '' check (allowing non-code stages to proceed)
	for _, job := range wf.Jobs {
		for _, step := range job.Steps {
			if strings.Contains(step.If, "pr-check.outputs.skipped") {
				// Steps referencing pr-check should also check stage
				assert.Contains(t, step.If, "stage",
					"Steps gating on pr-check should also reference stage output for: %s", step.Name)
			}
		}
	}
}

// TestDispatchProceedsWhenNoPRs verifies that dispatch allows code
// stage invocation when no open PRs are found for the target issue.
//
// [test_id:TS-GH-26-015]
//
// Validates:
//   - pr-check step logic: when gh pr list returns empty, skipped is not set
//   - Downstream steps proceed when pr-check.outputs.skipped is not 'true'
func TestDispatchProceedsWhenNoPRs(t *testing.T) {
	yamlPath := findDispatchYAML(t)
	wf := parseWorkflow(t, yamlPath)

	var prCheckStep *workflowStep
	for _, job := range wf.Jobs {
		step := findStepByID(job.Steps, "pr-check")
		if step != nil {
			prCheckStep = step
			break
		}
	}
	require.NotNil(t, prCheckStep)

	// The pr-check run block should only set skipped=true conditionally
	// (i.e., inside an if-block checking for non-empty PR results).
	// It should NOT unconditionally set skipped=true.
	lines := strings.Split(prCheckStep.Run, "\n")
	skippedSetUnconditionally := false
	insideIf := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "if ") || strings.HasPrefix(trimmed, "if [") {
			insideIf = true
		}
		if trimmed == "fi" {
			insideIf = false
		}
		if strings.Contains(trimmed, "skipped=true") && !insideIf {
			skippedSetUnconditionally = true
		}
	}
	assert.False(t, skippedSetUnconditionally,
		"skipped=true should only be set conditionally (inside if-block), not unconditionally")
}

// TestDispatchStageOutputIncludesPRCheckGate verifies that the dispatch.yml
// workflow YAML contains the pr-check step with correct conditional gating.
//
// [test_id:TS-GH-26-016]
//
// Validates:
//   - dispatch.yml has a pr-check step
//   - pr-check runs only for code stage
//   - Downstream steps (mint token, workflow dispatch) check pr-check output
func TestDispatchStageOutputIncludesPRCheckGate(t *testing.T) {
	yamlPath := findDispatchYAML(t)
	wf := parseWorkflow(t, yamlPath)

	// Verify pr-check step exists
	var prCheckFound bool
	var downstreamGated int
	for _, job := range wf.Jobs {
		for _, step := range job.Steps {
			if step.ID == "pr-check" {
				prCheckFound = true
				// Must gate on code stage
				assert.Contains(t, step.If, "code",
					"pr-check must gate on stage==code")
			}
			// Count downstream steps that reference pr-check output
			if step.ID != "pr-check" && strings.Contains(step.If, "pr-check.outputs.skipped") {
				downstreamGated++
			}
		}
	}

	assert.True(t, prCheckFound, "dispatch.yml must contain a step with id 'pr-check'")
	assert.Greater(t, downstreamGated, 0,
		"At least one downstream step must gate on pr-check.outputs.skipped")
}
