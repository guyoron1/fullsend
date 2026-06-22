package harness

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// GH-73-TC-030: Verify lint warns on missing role
func TestQF_Lint_WarnsOnMissingRole(t *testing.T) {
	h := &Harness{
		Agent: "agents/test.md",
		Slug:  "my-slug",
		// Role is intentionally empty
	}
	diags := h.Lint()

	assert.NotNil(t, diags, "should produce diagnostics")
	assert.Len(t, diags, 1, "should produce exactly one diagnostic")
	assert.Equal(t, SeverityWarning, diags[0].Severity, "diagnostic should be warning severity")
	assert.Equal(t, "role", diags[0].Field, "diagnostic should reference the 'role' field")
	assert.Contains(t, diags[0].Message, "required in a future version")
}

// GH-73-TC-031: Verify no diagnostics for valid harness
func TestQF_Lint_NoDiagnosticsForValidHarness(t *testing.T) {
	h := &Harness{
		Agent: "agents/test.md",
		Role:  "triage",
		Slug:  "my-slug",
	}
	diags := h.Lint()

	assert.Nil(t, diags, "should return nil for a fully valid harness")
}

// Supplemental: role set but slug empty should still pass lint
func TestQF_Lint_RoleSetSlugEmpty(t *testing.T) {
	h := &Harness{
		Agent: "agents/test.md",
		Role:  "coder",
	}
	diags := h.Lint()

	assert.Nil(t, diags, "lint only checks role, not slug")
}
