//go:build e2e

package tests

/*
Tool Allowlist Hook Toggle Tests — GH-18

STP Reference: outputs/stp/GH-18/GH-18_test_plan.md
Jira: GH-18
Requirement: Tool allowlist hook toggle
*/

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/harness"
	"github.com/fullsend-ai/fullsend/internal/security"
)

// TestAllowlistHookAddedWhenEnabled verifies that when the tool allowlist
// toggle is explicitly enabled, the allowlist hook appears in the generated
// Claude settings.
//
// [test_id:TS-GH-18-032]
func TestAllowlistHookAddedWhenEnabled(t *testing.T) {
	// Setup: Create config with allowlist explicitly enabled
	enabled := true
	h := &harness.Harness{
		Agent: "test-agent.md",
		Security: &harness.SecurityConfig{
			SandboxHooks: &harness.SandboxHooks{
				ToolAllowlistPreTool: &harness.ToolAllowlistConfig{
					Enabled: &enabled,
				},
			},
		},
	}

	// Execute: Generate Claude settings
	data, err := security.GenerateClaudeSettings(h)
	require.NoError(t, err)

	var settings map[string]any
	require.NoError(t, json.Unmarshal(data, &settings))

	hooks := settings["hooks"].(map[string]any)
	preTools := hooks["PreToolUse"].([]any)

	// Assert: PreToolUse should have 4 matchers (tirith + ssrf + canary + allowlist)
	assert.Len(t, preTools, 4,
		"With allowlist enabled, PreToolUse should have 4 matchers")

	// Verify the allowlist hook is present (last matcher with * pattern)
	lastMatcher := preTools[3].(map[string]any)
	assert.Equal(t, "*", lastMatcher["matcher"],
		"Allowlist matcher should use * pattern")

	hookList := lastMatcher["hooks"].([]any)
	require.Len(t, hookList, 1)
	cmd := hookList[0].(map[string]any)["command"].(string)
	assert.Contains(t, cmd, "tool_allowlist_pretool.py",
		"Allowlist hook command should reference tool_allowlist_pretool.py")
}

// TestAllowlistHookAbsentByDefault verifies that with default configuration
// (no explicit allowlist toggle), the allowlist hook is NOT present.
//
// [test_id:TS-GH-18-033]
func TestAllowlistHookAbsentByDefault(t *testing.T) {
	// Setup: Create default config (no allowlist toggle)
	h := &harness.Harness{Agent: "test-agent.md"}

	// Execute: Generate Claude settings
	data, err := security.GenerateClaudeSettings(h)
	require.NoError(t, err)

	var settings map[string]any
	require.NoError(t, json.Unmarshal(data, &settings))

	hooks := settings["hooks"].(map[string]any)
	preTools := hooks["PreToolUse"].([]any)

	// Assert: PreToolUse should have only 3 matchers (no allowlist)
	assert.Len(t, preTools, 3,
		"Default PreToolUse should have 3 matchers (no allowlist)")

	// Verify no matcher references tool_allowlist
	for _, pt := range preTools {
		m := pt.(map[string]any)
		hookList := m["hooks"].([]any)
		for _, hook := range hookList {
			cmd := hook.(map[string]any)["command"].(string)
			assert.NotContains(t, cmd, "tool_allowlist_pretool.py",
				"Default settings should NOT contain tool allowlist hook")
		}
	}
}
