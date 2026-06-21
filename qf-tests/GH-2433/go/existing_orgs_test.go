package gcf

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Existing Org Enrollment Tests

STP Reference: outputs/stp/GH-2433/GH-2433_test_plan.md
Jira: GH-2433

Tests covering enrollment behavior when ALLOWED_ORGS is already populated.
The guard should not trigger when ALLOWED_ORGS contains data, and enrollment
should correctly append new orgs or handle duplicates idempotently.
*/

// TestEnsureOrgInMint_PopulatedAllowedOrgs_Succeeds verifies that enrollment
// succeeds when ALLOWED_ORGS already contains existing orgs. The guard only
// checks for empty ALLOWED_ORGS, so populated state should proceed normally.
// [test_id:TS-GH-2433-008]
func TestEnsureOrgInMint_PopulatedAllowedOrgs_Succeeds(t *testing.T) {
	fake := newFakeGCFClient()
	fake.functionInfo = &FunctionInfo{
		URI: "https://fullsend-mint-test.run.app",
	}
	fake.trafficEnvVars = map[string]string{
		"ALLOWED_ORGS": "existing-org",
		"ROLE_APP_IDS": `{"coder":"app-id-123"}`,
	}

	p := NewProvisioner(Config{ProjectID: "my-project", Region: "us-central1"}, fake)

	err := p.EnsureOrgInMint(context.Background(), "https://fullsend-mint-test.run.app", "new-org")

	require.NoError(t, err, "enrollment should succeed when ALLOWED_ORGS is populated")

	// Verify new org is appended alongside existing org.
	require.NotNil(t, fake.lastUpdateServiceEnvVars,
		"env var update should have been captured")
	allowedOrgs := fake.lastUpdateServiceEnvVars["ALLOWED_ORGS"]
	assert.Contains(t, allowedOrgs, "existing-org",
		"existing org should be preserved in ALLOWED_ORGS")
	assert.Contains(t, allowedOrgs, "new-org",
		"new org should be added to ALLOWED_ORGS")
}

// TestEnsureOrgInMint_DuplicateOrg_Idempotent verifies that enrolling an org
// already present in ALLOWED_ORGS is idempotent — no error, no duplication.
// [test_id:TS-GH-2433-009]
func TestEnsureOrgInMint_DuplicateOrg_Idempotent(t *testing.T) {
	fake := newFakeGCFClient()
	fake.functionInfo = &FunctionInfo{
		URI: "https://fullsend-mint-test.run.app",
	}
	fake.trafficEnvVars = map[string]string{
		"ALLOWED_ORGS": "existing-org",
		"ROLE_APP_IDS": `{"coder":"app-id-123"}`,
	}

	p := NewProvisioner(Config{ProjectID: "my-project", Region: "us-central1"}, fake)

	err := p.EnsureOrgInMint(context.Background(), "https://fullsend-mint-test.run.app", "existing-org")

	require.NoError(t, err, "re-enrolling an existing org should not produce an error")

	// Verify no UpdateServiceEnvVars call was made (org already present).
	for _, call := range fake.calls {
		assert.NotEqual(t, "UpdateServiceEnvVars", call,
			"UpdateServiceEnvVars should not be called when org is already enrolled")
	}

	// Verify org appears exactly once if env vars were checked (idempotent).
	allowedOrgs := fake.trafficEnvVars["ALLOWED_ORGS"]
	count := strings.Count(allowedOrgs, "existing-org")
	assert.Equal(t, 1, count,
		"existing-org should appear exactly once in ALLOWED_ORGS")
}
