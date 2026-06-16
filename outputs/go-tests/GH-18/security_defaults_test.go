//go:build e2e

package tests

/*
Security Configuration Nil-Safety and Default Tests — GH-18

STP Reference: outputs/stp/GH-18/GH-18_test_plan.md
Jira: GH-18
Requirement: Nil and zero-value configs default to fail-closed
*/

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fullsend-ai/fullsend/internal/harness"
)

// TestNilSecurityConfigDefaultsToFailClosed verifies that a Harness with nil
// SecurityConfig defaults to SecurityEnabled=true and FailModeClosed=true,
// implementing the fail-closed security principle.
//
// [test_id:TS-GH-18-007]
func TestNilSecurityConfigDefaultsToFailClosed(t *testing.T) {
	// Setup: Create harness with nil Security (no SecurityConfig at all)
	h := &harness.Harness{Agent: "test-agent.md"}

	// Execute & Assert: SecurityEnabled should return true
	assert.True(t, h.SecurityEnabled(),
		"Nil SecurityConfig should default to security enabled (fail-closed)")

	// Execute & Assert: FailModeClosed should return true
	assert.True(t, h.FailModeClosed(),
		"Nil SecurityConfig should default to fail-closed mode")
}

// TestNilEnabledPointerDefaultsToEnabled verifies that when the Enabled field
// is a nil pointer in SecurityConfig, BoolDefault interprets it as true.
//
// [test_id:TS-GH-18-008]
func TestNilEnabledPointerDefaultsToEnabled(t *testing.T) {
	// Setup: Create SecurityConfig with nil Enabled field (zero value for *bool)
	h := &harness.Harness{
		Agent: "test-agent.md",
		Security: &harness.SecurityConfig{
			// Enabled is nil by default (not set)
		},
	}

	// Execute & Assert: BoolDefault with nil should return the default (true)
	result := harness.BoolDefault(h.Security.Enabled, true)
	assert.True(t, result,
		"BoolDefault(nil, true) should return true")

	// Also verify through SecurityEnabled method
	assert.True(t, h.SecurityEnabled(),
		"Nil Enabled pointer should default to security enabled")
}

// TestEmptyFailModeDefaultsToClosed verifies that when FailMode is an empty
// string, FailModeClosed returns true, defaulting to the safest behavior.
//
// [test_id:TS-GH-18-009]
func TestEmptyFailModeDefaultsToClosed(t *testing.T) {
	// Setup: Create config with empty FailMode
	h := &harness.Harness{
		Agent: "test-agent.md",
		Security: &harness.SecurityConfig{
			FailMode: "", // empty string
		},
	}

	// Execute & Assert: Empty FailMode should default to closed
	assert.True(t, h.FailModeClosed(),
		"Empty FailMode should default to closed (deny)")
}

// TestExplicitFalseOverridesDefaultTrue verifies that when an Enabled pointer
// explicitly points to false, BoolDefault returns false, honoring the explicit
// override over the default.
//
// [test_id:TS-GH-18-010]
func TestExplicitFalseOverridesDefaultTrue(t *testing.T) {
	// Setup: Create pointer to false
	falseVal := false
	ptr := &falseVal

	// Execute & Assert: BoolDefault should respect explicit false
	result := harness.BoolDefault(ptr, true)
	assert.False(t, result,
		"BoolDefault(&false, true) should return false — explicit override")

	// Also verify through SecurityEnabled with explicit Enabled=false
	h := &harness.Harness{
		Agent: "test-agent.md",
		Security: &harness.SecurityConfig{
			Enabled: &falseVal,
		},
	}
	assert.False(t, h.SecurityEnabled(),
		"Explicit Enabled=false should disable security")
}
