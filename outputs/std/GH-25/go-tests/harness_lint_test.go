//go:build e2e

package harness_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/harness"
)

/*
Harness Lint() Diagnostics Tests

STP Reference: outputs/stp/GH-25/GH-25_test_plan.md
Jira: GH-25
*/

func TestLint(t *testing.T) {
	// [test_id:TS-GH-25-015] harness with role set returns nil diagnostics
	t.Run("[test_id:TS-GH-25-015] should return nil for harness with role set", func(t *testing.T) {
		h := &harness.Harness{Role: "triage"}
		diags := h.Lint()
		assert.Nil(t, diags, "harness with role set should produce no diagnostics")
	})

	// [test_id:TS-GH-25-016] harness with empty role returns warning
	t.Run("[test_id:TS-GH-25-016] should return warning for harness with empty role", func(t *testing.T) {
		h := &harness.Harness{Role: ""}
		diags := h.Lint()

		require.Len(t, diags, 1, "expected exactly one diagnostic")
		assert.Equal(t, harness.SeverityWarning, diags[0].Severity,
			"diagnostic should be a warning")
		assert.Equal(t, "role", diags[0].Field,
			"diagnostic should reference the 'role' field")
		assert.Contains(t, diags[0].Message, "required in a future version",
			"warning should mention future version requirement")
	})

	// [test_id:TS-GH-25-017] harness with role and slug returns nil
	t.Run("[test_id:TS-GH-25-017] should return nil for harness with role and slug", func(t *testing.T) {
		h := &harness.Harness{Role: "triage", Slug: "triage-agent"}
		diags := h.Lint()
		assert.Nil(t, diags, "fully configured harness should produce no diagnostics")
	})

	// [test_id:TS-GH-25-021] returns nil not empty slice when no issues found
	t.Run("[test_id:TS-GH-25-021] should return nil not empty slice when no issues found", func(t *testing.T) {
		h := &harness.Harness{Role: "triage"}
		diags := h.Lint()

		// Go idiom: nil slice vs empty slice. Callers should be able to use
		// `if diags != nil` rather than `len(diags) > 0`.
		assert.Nil(t, diags, "Lint() should return nil, not an empty allocated slice")
		// Extra explicit check: ensure it's pointer-nil, not just empty
		var nilSlice []harness.Diagnostic
		assert.Equal(t, nilSlice, diags, "should be exactly nil, not []Diagnostic{}")
	})
}

func TestDiagnosticString(t *testing.T) {
	// [test_id:TS-GH-25-018] formats warning severity correctly
	t.Run("[test_id:TS-GH-25-018] should format warning severity correctly", func(t *testing.T) {
		d := harness.Diagnostic{
			Severity: harness.SeverityWarning,
			Field:    "role",
			Message:  "test",
		}
		assert.Equal(t, "warning: role: test", d.String())
	})

	// [test_id:TS-GH-25-019] formats error severity correctly
	t.Run("[test_id:TS-GH-25-019] should format error severity correctly", func(t *testing.T) {
		d := harness.Diagnostic{
			Severity: harness.SeverityError,
			Field:    "name",
			Message:  "missing",
		}
		assert.Equal(t, "error: name: missing", d.String())
	})

	// [test_id:TS-GH-25-020] formats unknown severity with fallback
	t.Run("[test_id:TS-GH-25-020] should format unknown severity with fallback", func(t *testing.T) {
		d := harness.Diagnostic{
			Severity: harness.DiagnosticSeverity(99),
			Field:    "x",
			Message:  "y",
		}
		expected := fmt.Sprintf("DiagnosticSeverity(99): x: y")
		assert.Equal(t, expected, d.String())
	})
}
