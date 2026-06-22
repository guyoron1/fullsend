package config

// QualityFlow generated tests for GH-72
// Suite: TS-GH72-006 — Config types for triage prerequisites
// STD: outputs/std/GH-72/GH-72_test_description.yaml

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TC-GH72-037: AllowTargets YAML parsing and defaults — nil config passes validation
func TestQFValidateCreateIssues_NilConfig(t *testing.T) {
	err := validateCreateIssues(nil)
	require.NoError(t, err, "nil CreateIssuesConfig should pass validation")
}

// TC-GH72-038: Validation rejects invalid repo format
func TestQFValidateCreateIssues_InvalidRepoFormat(t *testing.T) {
	cfg := &CreateIssuesConfig{
		AllowTargets: AllowTargets{
			Repos: []string{"invalid-format"},
		},
	}

	err := validateCreateIssues(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must contain owner/name",
		"error should identify the problematic repo format")
}

// TC-GH72-039: Validation rejects empty org
func TestQFValidateCreateIssues_EmptyOrg(t *testing.T) {
	cfg := &CreateIssuesConfig{
		AllowTargets: AllowTargets{
			Orgs: []string{""},
		},
	}

	err := validateCreateIssues(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty org",
		"error should catch empty org entries")
}
