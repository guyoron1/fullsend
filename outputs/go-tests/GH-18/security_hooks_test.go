//go:build e2e

package tests

/*
Security Hook Pipeline Tests — GH-18

STP Reference: outputs/stp/GH-18/GH-18_test_plan.md
Jira: GH-18
Requirement: Security hook pipeline generates correct Claude settings
*/

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/harness"
	"github.com/fullsend-ai/fullsend/internal/security"
)

// TestDefaultSettingsIncludeAllExpectedHooks verifies that GenerateClaudeSettings
// with a default Harness (no SecurityConfig overrides) produces settings JSON
// containing both PreToolUse and PostToolUse hook sections with all expected entries.
//
// [test_id:TS-GH-18-001]
func TestDefaultSettingsIncludeAllExpectedHooks(t *testing.T) {
	// Setup: Create a default harness with no security overrides
	h := &harness.Harness{Agent: "test-agent.md"}

	// Execute: Generate Claude settings
	data, err := security.GenerateClaudeSettings(h)
	require.NoError(t, err, "GenerateClaudeSettings should not return error with default config")

	// Parse the generated JSON
	var settings map[string]any
	require.NoError(t, json.Unmarshal(data, &settings), "Generated settings should be valid JSON")

	hooks, ok := settings["hooks"].(map[string]any)
	require.True(t, ok, "Settings should contain a 'hooks' map")

	// Assert: Both PreToolUse and PostToolUse sections exist
	assert.Contains(t, hooks, "PreToolUse", "Default settings should contain PreToolUse hooks")
	assert.Contains(t, hooks, "PostToolUse", "Default settings should contain PostToolUse hooks")

	// Verify we have the expected hook matchers in each section
	preTools, ok := hooks["PreToolUse"].([]any)
	require.True(t, ok, "PreToolUse should be an array")
	assert.NotEmpty(t, preTools, "PreToolUse should have at least one matcher")

	postTools, ok := hooks["PostToolUse"].([]any)
	require.True(t, ok, "PostToolUse should be an array")
	assert.NotEmpty(t, postTools, "PostToolUse should have at least one matcher")
}

// TestDefaultPreToolUseHookCountAndMatchers verifies that the PreToolUse section
// contains exactly 3 matchers (tirith, ssrf, canary) with correct matcher patterns.
// Tool allowlist is disabled by default and should NOT be present.
//
// [test_id:TS-GH-18-002]
func TestDefaultPreToolUseHookCountAndMatchers(t *testing.T) {
	// Setup: Create default harness
	h := &harness.Harness{Agent: "test-agent.md"}

	// Execute: Generate settings and extract PreToolUse hooks
	data, err := security.GenerateClaudeSettings(h)
	require.NoError(t, err)

	var settings map[string]any
	require.NoError(t, json.Unmarshal(data, &settings))

	hooks := settings["hooks"].(map[string]any)
	preTools := hooks["PreToolUse"].([]any)

	// Assert: Exactly 3 PreToolUse matchers (tirith=Bash, ssrf=Bash|WebFetch, canary=*)
	assert.Len(t, preTools, 3, "Default PreToolUse should have 3 matchers: tirith, ssrf, canary")

	// Verify matcher patterns
	matchers := make([]string, len(preTools))
	for i, pt := range preTools {
		m := pt.(map[string]any)
		matchers[i] = m["matcher"].(string)
	}

	assert.Equal(t, "Bash", matchers[0], "First PreToolUse matcher should be Bash (tirith)")
	assert.Equal(t, "Bash|WebFetch", matchers[1], "Second PreToolUse matcher should be Bash|WebFetch (ssrf)")
	assert.Equal(t, "*", matchers[2], "Third PreToolUse matcher should be * (canary)")
}

// TestDefaultPostToolUseChainStructure verifies that the PostToolUse section
// has the correct chain structure: a Bash|WebFetch|Read chain with 3 hooks
// (context_suppress, secret_redact, unicode) plus a separate * matcher for canary.
//
// [test_id:TS-GH-18-003]
func TestDefaultPostToolUseChainStructure(t *testing.T) {
	// Setup: Create default harness
	h := &harness.Harness{Agent: "test-agent.md"}

	// Execute: Generate settings and extract PostToolUse chain
	data, err := security.GenerateClaudeSettings(h)
	require.NoError(t, err)

	var settings map[string]any
	require.NoError(t, json.Unmarshal(data, &settings))

	hooks := settings["hooks"].(map[string]any)
	postTools := hooks["PostToolUse"].([]any)

	// Assert: 2 PostToolUse matchers (chain + canary)
	assert.Len(t, postTools, 2, "Default PostToolUse should have 2 matchers")

	// Verify first matcher is the Bash|WebFetch|Read chain with 3 hooks
	chainMatcher := postTools[0].(map[string]any)
	assert.Equal(t, "Bash|WebFetch|Read", chainMatcher["matcher"],
		"First PostToolUse matcher should be Bash|WebFetch|Read")

	chainedHooks := chainMatcher["hooks"].([]any)
	assert.Len(t, chainedHooks, 3,
		"Chain should have 3 hooks: context_suppress, secret_redact, unicode")

	// Verify ordering: context_suppress -> secret_redact -> unicode
	for i, hook := range chainedHooks {
		h := hook.(map[string]any)
		cmd := h["command"].(string)
		switch i {
		case 0:
			assert.Contains(t, cmd, "context_suppress_posttool.py",
				"First hook in chain should be context_suppress")
		case 1:
			assert.Contains(t, cmd, "secret_redact_posttool.py",
				"Second hook in chain should be secret_redact")
		case 2:
			assert.Contains(t, cmd, "unicode_posttool.py",
				"Third hook in chain should be unicode")
		}
	}

	// Verify second matcher is canary with * pattern
	canaryMatcher := postTools[1].(map[string]any)
	assert.Equal(t, "*", canaryMatcher["matcher"],
		"Second PostToolUse matcher should be * (canary)")
	canaryHooks := canaryMatcher["hooks"].([]any)
	assert.Len(t, canaryHooks, 1, "Canary matcher should have exactly 1 hook")
}

// TestSingleHookDisableLeavesOthersEnabled verifies that disabling a single
// hook (e.g., Tirith) removes only that hook while all others remain.
//
// [test_id:TS-GH-18-004]
func TestSingleHookDisableLeavesOthersEnabled(t *testing.T) {
	// Setup: Create config with only Tirith disabled
	disabled := false
	h := &harness.Harness{
		Agent: "test-agent.md",
		Security: &harness.SecurityConfig{
			SandboxHooks: &harness.SandboxHooks{
				Tirith: &harness.TirithConfig{Enabled: &disabled},
			},
		},
	}

	// Execute: Generate settings
	data, err := security.GenerateClaudeSettings(h)
	require.NoError(t, err)

	var settings map[string]any
	require.NoError(t, json.Unmarshal(data, &settings))

	hooks := settings["hooks"].(map[string]any)

	// Assert: PreToolUse should have 2 matchers (ssrf + canary, no tirith)
	preTools := hooks["PreToolUse"].([]any)
	assert.Len(t, preTools, 2, "With tirith disabled, PreToolUse should have 2 matchers")

	// Verify tirith Bash matcher is absent
	for _, pt := range preTools {
		m := pt.(map[string]any)
		matcher := m["matcher"].(string)
		if matcher == "Bash" {
			hookList := m["hooks"].([]any)
			for _, hook := range hookList {
				cmd := hook.(map[string]any)["command"].(string)
				assert.NotContains(t, cmd, "tirith_check.py",
					"Tirith hook should be absent when disabled")
			}
		}
	}

	// Assert: PostToolUse should still have full chain + canary
	postTools := hooks["PostToolUse"].([]any)
	assert.Len(t, postTools, 2, "PostToolUse should be unaffected by tirith disable")
}

// TestAllHooksDisabledProducesEmptyHookMap verifies that disabling all hooks
// produces settings with no PreToolUse or PostToolUse entries.
//
// [test_id:TS-GH-18-005]
func TestAllHooksDisabledProducesEmptyHookMap(t *testing.T) {
	// Setup: Disable every hook toggle
	disabled := false
	h := &harness.Harness{
		Agent: "test-agent.md",
		Security: &harness.SecurityConfig{
			SandboxHooks: &harness.SandboxHooks{
				Tirith:                  &harness.TirithConfig{Enabled: &disabled},
				SSRFPreTool:             &disabled,
				SecretRedactPostTool:    &disabled,
				UnicodePostTool:         &disabled,
				ContextSuppressPostTool: &disabled,
				CanaryPreTool:           &disabled,
				CanaryPostTool:          &disabled,
				// ToolAllowlistPreTool omitted — already disabled by default
			},
		},
	}

	// Execute: Generate settings
	data, err := security.GenerateClaudeSettings(h)
	require.NoError(t, err)

	var settings map[string]any
	require.NoError(t, json.Unmarshal(data, &settings))

	hooks := settings["hooks"].(map[string]any)

	// Assert: No PreToolUse or PostToolUse entries
	assert.NotContains(t, hooks, "PreToolUse",
		"Hook map should not contain PreToolUse when all hooks disabled")
	assert.NotContains(t, hooks, "PostToolUse",
		"Hook map should not contain PostToolUse when all hooks disabled")
}

// TestReEnablingDisabledHookRestoresIt verifies that a hook can be disabled
// and then re-enabled, and it appears correctly in the generated settings.
//
// [test_id:TS-GH-18-006]
func TestReEnablingDisabledHookRestoresIt(t *testing.T) {
	// Setup: First create a config with ssrf disabled
	disabled := false
	h1 := &harness.Harness{
		Agent: "test-agent.md",
		Security: &harness.SecurityConfig{
			SandboxHooks: &harness.SandboxHooks{
				SSRFPreTool: &disabled,
			},
		},
	}

	// Verify it's disabled
	data1, err := security.GenerateClaudeSettings(h1)
	require.NoError(t, err)

	var settings1 map[string]any
	require.NoError(t, json.Unmarshal(data1, &settings1))
	hooks1 := settings1["hooks"].(map[string]any)
	preTools1 := hooks1["PreToolUse"].([]any)
	assert.Len(t, preTools1, 2, "With ssrf disabled, PreToolUse should have 2 matchers")

	// Now re-enable ssrf
	enabled := true
	h2 := &harness.Harness{
		Agent: "test-agent.md",
		Security: &harness.SecurityConfig{
			SandboxHooks: &harness.SandboxHooks{
				SSRFPreTool: &enabled,
			},
		},
	}

	// Execute: Generate settings with re-enabled hook
	data2, err := security.GenerateClaudeSettings(h2)
	require.NoError(t, err)

	var settings2 map[string]any
	require.NoError(t, json.Unmarshal(data2, &settings2))
	hooks2 := settings2["hooks"].(map[string]any)
	preTools2 := hooks2["PreToolUse"].([]any)

	// Assert: ssrf hook is restored — back to 3 PreToolUse matchers
	assert.Len(t, preTools2, 3,
		"Re-enabling ssrf should restore PreToolUse to 3 matchers")

	// Verify ssrf matcher is present
	hasSSRF := false
	for _, pt := range preTools2 {
		m := pt.(map[string]any)
		if m["matcher"].(string) == "Bash|WebFetch" {
			hookList := m["hooks"].([]any)
			for _, hook := range hookList {
				cmd := hook.(map[string]any)["command"].(string)
				if strings.Contains(cmd, "ssrf_pretool.py") {
					hasSSRF = true
				}
			}
		}
	}
	assert.True(t, hasSSRF, "Re-enabled ssrf hook should be present in settings")
}
