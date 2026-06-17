//go:build e2e

package harness

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Harness Lint() Diagnostic Method — Implementation Tests

STP Reference: outputs/stp/GH-23/GH-23_test_plan.md
STD Reference: outputs/std/GH-23/GH-23_test_description.yaml
Jira: GH-23

These tests verify the Lint() diagnostic method on the Harness struct
(ADR-0045 Phase 3), including DiagnosticSeverity string formatting and
Validate() regression.
*/

// TestLintMissingRole validates that Lint() returns a warning diagnostic when
// the Role field is empty. This is the core lint rule for Phase 3 — if it fails,
// harness authors receive no warning about the upcoming required role field.
// [test_id:TS-GH-23-001] [P0] [MVP]
func TestLintMissingRole(t *testing.T) {
	h := &Harness{} // Role is empty (zero-value)
	diagnostics := h.Lint()

	require.Len(t, diagnostics, 1, "Lint() should return exactly 1 diagnostic for missing role")
	assert.Equal(t, SeverityWarning, diagnostics[0].Severity, "diagnostic severity should be SeverityWarning")
	assert.Equal(t, "role", diagnostics[0].Field, "diagnostic field should be 'role'")
	assert.Contains(t, diagnostics[0].Message, "required in a future version", "diagnostic message should mention future requirement")
}

// TestLintValidHarnessWithRole validates that Lint() returns nil (not an empty
// slice) when the Role field is populated. Returning nil is the Go convention
// for "no issues found."
// [test_id:TS-GH-23-002] [P0] [MVP]
func TestLintValidHarnessWithRole(t *testing.T) {
	h := &Harness{Role: "triage"}
	diagnostics := h.Lint()

	assert.Nil(t, diagnostics, "Lint() should return nil (not empty slice) when Role is set")
}

// TestLintValidHarnessWithRoleAndSlug validates that Lint() returns nil when
// both Role and Slug fields are populated. This serves as a regression guard
// for the expanding rule set — additional populated fields must not trigger
// false positive warnings.
// [test_id:TS-GH-23-003] [P1]
func TestLintValidHarnessWithRoleAndSlug(t *testing.T) {
	h := &Harness{Role: "triage", Slug: "my-slug"}
	diagnostics := h.Lint()

	assert.Nil(t, diagnostics, "Lint() should return nil when both Role and Slug are set")
}

// TestDiagnosticStringWarning validates that Diagnostic.String() produces
// the expected human-readable format for SeverityWarning. The String() output
// is what users see in CLI output and logs.
// [test_id:TS-GH-23-004] [P1]
func TestDiagnosticStringWarning(t *testing.T) {
	d := Diagnostic{
		Severity: SeverityWarning,
		Field:    "role",
		Message:  "msg",
	}
	result := d.String()

	assert.Equal(t, "warning: role: msg", result, "String() should format warning severity as 'warning: field: message'")
}

// TestDiagnosticStringError validates that Diagnostic.String() produces
// the expected human-readable format for SeverityError. Error-level diagnostics
// must be clearly distinguishable from warnings.
// [test_id:TS-GH-23-005] [P1]
func TestDiagnosticStringError(t *testing.T) {
	d := Diagnostic{
		Severity: SeverityError,
		Field:    "role",
		Message:  "msg",
	}
	result := d.String()

	assert.Equal(t, "error: role: msg", result, "String() should format error severity as 'error: field: message'")
}

// TestDiagnosticStringUnknownSeverity validates that Diagnostic.String()
// handles an unrecognized DiagnosticSeverity value by falling back to the
// numeric format. This ensures forward compatibility as new severity levels
// are added.
// [test_id:TS-GH-23-006] [P2]
func TestDiagnosticStringUnknownSeverity(t *testing.T) {
	d := Diagnostic{
		Severity: DiagnosticSeverity(99),
		Field:    "x",
		Message:  "msg",
	}
	result := d.String()

	assert.Equal(t, "DiagnosticSeverity(99): x: msg", result, "String() should fall back to numeric format for unknown severity")
}

// TestValidateUnchangedAfterLintAddition is a regression test verifying that
// the addition of Lint() has not altered Validate() behavior. Validate()
// returns hard errors while Lint() returns non-fatal warnings — any accidental
// coupling between them could break existing validation logic.
// [test_id:TS-GH-23-007] [P0] [MVP]
func TestValidateUnchangedAfterLintAddition(t *testing.T) {
	t.Run("invalid harness returns error", func(t *testing.T) {
		h := &Harness{} // Missing required fields (agent is required)
		err := h.Validate()
		assert.Error(t, err, "Validate() should return error for structurally invalid harness")
	})

	t.Run("valid harness returns nil", func(t *testing.T) {
		h := &Harness{Agent: "test-harness", Role: "triage"}
		err := h.Validate()
		assert.NoError(t, err, "Validate() should return nil for valid harness")
	})
}
