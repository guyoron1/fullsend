package harness

// QualityFlow generated tests for GH-72
// Suite: TS-GH72-004 — Harness Lint non-fatal diagnostics
// STD: outputs/std/GH-72/GH-72_test_description.yaml

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TC-GH72-019: Lint returns nil when role is set
func TestQFLint_RoleSet(t *testing.T) {
	h := &Harness{Role: "triage"}
	assert.Nil(t, h.Lint(), "no diagnostics when role is set")
}

// TC-GH72-020: Lint warns on missing role field
func TestQFLint_RoleEmpty(t *testing.T) {
	h := &Harness{}
	diags := h.Lint()
	assert.NotNil(t, diags)
	assert.Len(t, diags, 1)
	assert.Equal(t, SeverityWarning, diags[0].Severity)
	assert.Equal(t, "role", diags[0].Field)
	assert.Contains(t, diags[0].Message, "required in a future version",
		"diagnostic should warn about future requirement")
}

// TC-GH72-021: Lint returns nil when role and slug both set
func TestQFLint_RoleAndSlugSet(t *testing.T) {
	h := &Harness{Role: "triage", Slug: "my-slug"}
	assert.Nil(t, h.Lint(), "no diagnostics when both role and slug are set")
}

// TC-GH72-022: Diagnostic String formatting for warning
func TestQFDiagnostic_String_Warning(t *testing.T) {
	d := Diagnostic{Severity: SeverityWarning, Field: "role", Message: "msg"}
	assert.Equal(t, "warning: role: msg", d.String())
}

// TC-GH72-023: Diagnostic String formatting for error
func TestQFDiagnostic_String_Error(t *testing.T) {
	d := Diagnostic{Severity: SeverityError, Field: "role", Message: "msg"}
	assert.Equal(t, "error: role: msg", d.String())
}

// TC-GH72-024: Diagnostic String formatting for unknown severity
func TestQFDiagnostic_String_UnknownSeverity(t *testing.T) {
	d := Diagnostic{Severity: DiagnosticSeverity(99), Field: "x", Message: "msg"}
	assert.Equal(t, "DiagnosticSeverity(99): x: msg", d.String(),
		"unknown severity should use Go stringer format")
}
