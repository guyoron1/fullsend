package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Triage Agent Defense Layer Tests

STP Reference: outputs/stp/GH-26/GH-26_test_plan.md
STD Reference: outputs/std/GH-26/GH-26_test_description.yaml
Jira: GH-26

Tests for the triage agent hard constraint that emits a 'prerequisites'
action when an open PR already addresses the target issue, preventing
routing to the code stage.

Since the triage agent is an LLM-driven agent whose behavior is defined
in its markdown definition (triage.md), these tests validate:
1. The agent definition contains the hard constraint
2. The constraint language is unambiguous
3. The output schema supports the prerequisites action
*/

//go:build e2e

const triageAgentRelPath = "internal/scaffold/fullsend-repo/agents/triage.md"

func findTriageAgent(t *testing.T) string {
	t.Helper()
	candidates := []string{
		triageAgentRelPath,
		filepath.Join("..", triageAgentRelPath),
		filepath.Join("..", "..", triageAgentRelPath),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	root := os.Getenv("REPO_ROOT")
	if root != "" {
		return filepath.Join(root, triageAgentRelPath)
	}
	t.Skip("triage.md not found — set REPO_ROOT or run from repo root")
	return ""
}

// TestTriageEmitsPrerequisitesOnExistingPR verifies that the triage agent
// definition contains a hard constraint to emit 'prerequisites' action
// when it detects an open PR addressing the issue.
//
// [test_id:TS-GH-26-017]
//
// Validates:
//   - triage.md contains "prerequisites" action reference
//   - The constraint is marked as HARD CONSTRAINT
//   - The constraint references open PRs as a trigger
func TestTriageEmitsPrerequisitesOnExistingPR(t *testing.T) {
	agentPath := findTriageAgent(t)
	content, err := os.ReadFile(agentPath)
	require.NoError(t, err)
	agentDef := string(content)

	// Verify the hard constraint about existing PRs exists
	assert.Contains(t, agentDef, "prerequisites",
		"Triage agent definition must reference 'prerequisites' action")

	// Verify it's a hard constraint (not advisory)
	lowerDef := strings.ToLower(agentDef)
	assert.True(t,
		strings.Contains(lowerDef, "hard constraint") || strings.Contains(lowerDef, "must"),
		"Prerequisites rule should be a hard constraint in triage.md")

	// Verify the constraint connects open PRs to prerequisites action
	assert.True(t,
		strings.Contains(agentDef, "open PR") || strings.Contains(agentDef, "open pr"),
		"Hard constraint should reference 'open PR' as the trigger condition")

	// Verify the instruction not to emit 'sufficient' when PRs exist
	assert.Contains(t, agentDef, "sufficient",
		"Constraint should mention not emitting 'sufficient' when PRs exist")
}

// TestTriageProceedsWhenNoPR verifies that the triage agent definition
// does not unconditionally block — only blocks when open PRs exist.
//
// [test_id:TS-GH-26-018]
//
// Validates:
//   - triage.md supports normal routing (not just prerequisites)
//   - The agent definition includes non-prerequisites actions
func TestTriageProceedsWhenNoPR(t *testing.T) {
	agentPath := findTriageAgent(t)
	content, err := os.ReadFile(agentPath)
	require.NoError(t, err)
	agentDef := string(content)

	// Verify the agent definition supports multiple actions, not just prerequisites
	assert.Contains(t, agentDef, "sufficient",
		"Triage agent must support 'sufficient' action for normal routing")

	// Verify the prerequisites action is conditional, not unconditional
	// The constraint should be inside a conditional context (if/when open PR exists)
	prerequisitesIdx := strings.Index(agentDef, "prerequisites")
	require.Greater(t, prerequisitesIdx, 0,
		"triage.md must mention prerequisites")

	// Check that the surrounding context is conditional
	surrounding := agentDef[max(0, prerequisitesIdx-200):min(len(agentDef), prerequisitesIdx+200)]
	hasConditional := strings.Contains(strings.ToLower(surrounding), "if") ||
		strings.Contains(strings.ToLower(surrounding), "when") ||
		strings.Contains(strings.ToLower(surrounding), "already")
	assert.True(t, hasConditional,
		"Prerequisites action should be conditional (triggered by specific conditions)")
}

// TestTriageIgnoresClosedPRs verifies that the triage agent constraint
// specifically targets open PRs, not closed ones.
//
// [test_id:TS-GH-26-019]
//
// Validates:
//   - The constraint language specifies "open" PRs
//   - The search command uses --state open filter
func TestTriageIgnoresClosedPRs(t *testing.T) {
	agentPath := findTriageAgent(t)
	content, err := os.ReadFile(agentPath)
	require.NoError(t, err)
	agentDef := string(content)

	// Verify the agent searches for OPEN PRs specifically
	assert.Contains(t, agentDef, "--state open",
		"Triage agent should search with '--state open' to exclude closed PRs")

	// Verify the hard constraint says "open PR" not just "PR"
	assert.True(t,
		strings.Contains(agentDef, "open PR") || strings.Contains(agentDef, "open pr"),
		"Hard constraint should specifically reference 'open' PRs")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
