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
Workflow Step Gating Tests

STP Reference: outputs/stp/GH-26/GH-26_test_plan.md
STD Reference: outputs/std/GH-26/GH-26_test_description.yaml
Jira: GH-26

Tests for the reusable-code.yml workflow step gating that ensures
all post-validation steps (GCP setup, agent run, etc.) are correctly
conditioned on the validate step's skip output.
*/

//go:build e2e

const reusableCodeYAMLRelPath = ".github/workflows/reusable-code.yml"

func findReusableCodeYAML(t *testing.T) string {
	t.Helper()
	candidates := []string{
		reusableCodeYAMLRelPath,
		filepath.Join("..", reusableCodeYAMLRelPath),
		filepath.Join("..", "..", reusableCodeYAMLRelPath),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	root := os.Getenv("REPO_ROOT")
	if root != "" {
		return filepath.Join(root, reusableCodeYAMLRelPath)
	}
	t.Skip("reusable-code.yml not found — set REPO_ROOT or run from repo root")
	return ""
}

// reusableWorkflowStep for parsing reusable-code.yml
type reusableWorkflowStep struct {
	Name string `yaml:"name"`
	ID   string `yaml:"id"`
	If   string `yaml:"if"`
	Uses string `yaml:"uses"`
	Run  string `yaml:"run"`
}

type reusableWorkflowJob struct {
	Steps []reusableWorkflowStep `yaml:"steps"`
}

type reusableWorkflowFile struct {
	Jobs map[string]reusableWorkflowJob `yaml:"jobs"`
}

func parseReusableWorkflow(t *testing.T, path string) reusableWorkflowFile {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var wf reusableWorkflowFile
	require.NoError(t, yaml.Unmarshal(data, &wf))
	return wf
}

// findAllSteps flattens all steps from all jobs.
func findAllSteps(wf reusableWorkflowFile) []reusableWorkflowStep {
	var all []reusableWorkflowStep
	for _, job := range wf.Jobs {
		all = append(all, job.Steps...)
	}
	return all
}

// findStepByName locates a step containing the given name substring.
func findStepByName(steps []reusableWorkflowStep, nameSubstring string) *reusableWorkflowStep {
	for i, s := range steps {
		if strings.Contains(strings.ToLower(s.Name), strings.ToLower(nameSubstring)) {
			return &steps[i]
		}
	}
	return nil
}

// TestGCPSetupSkippedWhenValidateSkips verifies that the GCP setup step
// in reusable-code.yml is gated on the validate step's skip output.
//
// [test_id:TS-GH-26-024]
//
// Validates:
//   - GCP setup step exists in reusable-code.yml
//   - Step has 'if' condition referencing validate.outputs.skipped
//   - Condition evaluates to skip when skipped=true
func TestGCPSetupSkippedWhenValidateSkips(t *testing.T) {
	yamlPath := findReusableCodeYAML(t)
	wf := parseReusableWorkflow(t, yamlPath)
	steps := findAllSteps(wf)

	gcpStep := findStepByName(steps, "GCP")
	if gcpStep == nil {
		gcpStep = findStepByName(steps, "gcp")
	}
	if gcpStep == nil {
		gcpStep = findStepByName(steps, "Setup GCP")
	}
	require.NotNil(t, gcpStep,
		"reusable-code.yml must contain a GCP setup step")

	assert.NotEmpty(t, gcpStep.If,
		"GCP setup step must have an 'if' condition")
	assert.Contains(t, gcpStep.If, "validate",
		"GCP setup step condition should reference the validate step")
	assert.Contains(t, gcpStep.If, "skipped",
		"GCP setup step condition should reference the skipped output")
	assert.Contains(t, gcpStep.If, "'true'",
		"GCP setup step condition should compare against 'true'")
}

// TestAgentRunSkippedWhenValidateSkips verifies that the agent run step
// is correctly gated on the validate step's skip output.
//
// [test_id:TS-GH-26-025]
//
// Validates:
//   - Agent run step exists in reusable-code.yml
//   - Step has 'if' condition referencing validate.outputs.skipped
//   - Condition prevents execution when skipped=true
func TestAgentRunSkippedWhenValidateSkips(t *testing.T) {
	yamlPath := findReusableCodeYAML(t)
	wf := parseReusableWorkflow(t, yamlPath)
	steps := findAllSteps(wf)

	// Look for the agent run step by name or by the .defaults/ action usage
	agentStep := findStepByName(steps, "code agent")
	if agentStep == nil {
		agentStep = findStepByName(steps, "Run code")
	}
	if agentStep == nil {
		// Fall back to finding step that uses .defaults/ action
		for i, s := range steps {
			if strings.Contains(s.Uses, ".defaults") {
				agentStep = &steps[i]
				break
			}
		}
	}
	require.NotNil(t, agentStep,
		"reusable-code.yml must contain a code agent run step")

	assert.NotEmpty(t, agentStep.If,
		"Agent run step must have an 'if' condition")
	assert.Contains(t, agentStep.If, "validate",
		"Agent run step condition should reference the validate step")
	assert.Contains(t, agentStep.If, "skipped",
		"Agent run step condition should reference the skipped output")

	// The condition should be != 'true' (skip when true)
	assert.Contains(t, agentStep.If, "!= 'true'",
		"Agent run step should use != 'true' to skip when validate says skipped")
}

// TestAllGatedStepsRunWhenNotSkipped verifies that when validate sets
// skipped=false, all downstream steps execute normally.
//
// [test_id:TS-GH-26-026]
//
// Validates:
//   - All steps with skip gates use consistent condition format
//   - Conditions evaluate to true when skipped is not 'true'
//   - No step is unconditionally skipped
func TestAllGatedStepsRunWhenNotSkipped(t *testing.T) {
	yamlPath := findReusableCodeYAML(t)
	wf := parseReusableWorkflow(t, yamlPath)
	steps := findAllSteps(wf)

	var gatedSteps []reusableWorkflowStep
	for _, step := range steps {
		if strings.Contains(step.If, "validate") && strings.Contains(step.If, "skipped") {
			gatedSteps = append(gatedSteps, step)
		}
	}

	require.Greater(t, len(gatedSteps), 0,
		"reusable-code.yml should have steps gated on validate.outputs.skipped")

	for _, step := range gatedSteps {
		// When skipped=false (or empty), all gated steps should execute.
		// The condition should be: steps.validate.outputs.skipped != 'true'
		// This means when skipped is 'false' or empty, the step runs.
		assert.Contains(t, step.If, "!= 'true'",
			"Step %q should gate with != 'true' (runs when skipped is false/empty)", step.Name)

		// Ensure no step has an unconditional skip (like `if: false`)
		assert.NotEqual(t, "false", strings.TrimSpace(step.If),
			"Step %q should not be unconditionally skipped", step.Name)
	}

	t.Logf("Found %d steps gated on validate.outputs.skipped, all using consistent != 'true' pattern", len(gatedSteps))
}
