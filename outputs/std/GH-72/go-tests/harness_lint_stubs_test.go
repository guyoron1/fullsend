package harness

// STD Test Stubs for GH-72: Harness Lint non-fatal diagnostics
// Suite: TS-GH72-004
//
// These stubs correspond to test cases TC-GH72-019 through TC-GH72-024.
// Production tests: internal/harness/lint_test.go
// STP reference: outputs/stp/GH-72/GH-72_test_plan.md

import "testing"

// TC-GH72-019: Lint returns nil when role is set
//
// Preconditions:
//   - Harness struct with Role="triage"
//
// Steps:
//  1. Call Lint() on the harness
//
// Expected:
//   - Returns nil (no diagnostics emitted)
func TestLint_RoleSet_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-019")
}

// TC-GH72-020: Lint warns on missing role field
//
// Preconditions:
//   - Harness struct with empty Role field
//
// Steps:
//  1. Call Lint() on the harness
//
// Expected:
//   - Returns 1 Diagnostic with SeverityWarning, Field="role"
//   - Message contains "required in a future version"
func TestLint_RoleEmpty_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-020")
}

// TC-GH72-021: Lint returns nil when role and slug both set
//
// Preconditions:
//   - Harness struct with Role="triage", Slug="my-slug"
//
// Steps:
//  1. Call Lint() on the harness
//
// Expected:
//   - Returns nil (no diagnostics)
func TestLint_RoleAndSlugSet_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-021")
}

// TC-GH72-022: Diagnostic String formatting for warning
//
// Preconditions:
//   - Diagnostic with SeverityWarning, Field="role", Message="msg"
//
// Steps:
//  1. Call String() on the Diagnostic
//
// Expected:
//   - Returns "warning: role: msg"
func TestDiagnosticString_Warning_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-022")
}

// TC-GH72-023: Diagnostic String formatting for error
//
// Preconditions:
//   - Diagnostic with SeverityError, Field="role", Message="msg"
//
// Steps:
//  1. Call String() on the Diagnostic
//
// Expected:
//   - Returns "error: role: msg"
func TestDiagnosticString_Error_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-023")
}

// TC-GH72-024: Diagnostic String formatting for unknown severity
//
// Preconditions:
//   - Diagnostic with DiagnosticSeverity(99), Field="x", Message="msg"
//
// Steps:
//  1. Call String() on the Diagnostic
//
// Expected:
//   - Returns "DiagnosticSeverity(99): x: msg" (Go stringer fallback)
func TestDiagnosticString_UnknownSeverity_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-024")
}
