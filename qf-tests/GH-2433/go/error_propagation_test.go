package gcf

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Error Propagation Tests

STP Reference: outputs/stp/GH-2433/GH-2433_test_plan.md
Jira: GH-2433

Tests covering error propagation from EnsureOrgInMint through the
provisioning chain (provisionWithExistingMint). Ensures the data
consistency error is not swallowed and includes debugging context.
*/

// TestProvisionWithExistingMint_DataInconsistency_Aborts verifies that
// provisionWithExistingMint correctly propagates the data consistency error
// from EnsureOrgInMint and aborts the provisioning flow.
// [test_id:TS-GH-2433-010]
func TestProvisionWithExistingMint_DataInconsistency_Aborts(t *testing.T) {
	fake := newFakeGCFClient()
	fake.functionInfo = &FunctionInfo{
		URI: "https://fullsend-mint-test.run.app",
	}
	fake.trafficEnvVars = map[string]string{
		"ALLOWED_ORGS": "",
		"ROLE_APP_IDS": `{"coder":"app-id-123"}`,
	}

	p := newTestProvisioner(Config{
		ProjectID:  "my-project",
		Region:     "us-central1",
		MintURL:    "https://fullsend-mint-test.run.app",
		GitHubOrgs: []string{"new-org"},
		AgentAppIDs: map[string]string{"coder": "app-id-123"},
	}, fake)

	_, err := p.provisionWithExistingMint(context.Background())

	require.Error(t, err, "provisionWithExistingMint should propagate the guard error")
	assert.Contains(t, err.Error(), "data inconsistency",
		"error should reference data inconsistency from EnsureOrgInMint")
}

// TestProvisionWithExistingMint_ErrorWrapsOrgContext verifies that the error
// propagated from provisionWithExistingMint includes the org name for debugging.
// [test_id:TS-GH-2433-011]
func TestProvisionWithExistingMint_ErrorWrapsOrgContext(t *testing.T) {
	fake := newFakeGCFClient()
	fake.functionInfo = &FunctionInfo{
		URI: "https://fullsend-mint-test.run.app",
	}
	fake.trafficEnvVars = map[string]string{
		"ALLOWED_ORGS": "",
		"ROLE_APP_IDS": `{"coder":"app-id-123"}`,
	}

	p := newTestProvisioner(Config{
		ProjectID:  "my-project",
		Region:     "us-central1",
		MintURL:    "https://fullsend-mint-test.run.app",
		GitHubOrgs: []string{"debug-org"},
		AgentAppIDs: map[string]string{"coder": "app-id-123"},
	}, fake)

	_, err := p.provisionWithExistingMint(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "debug-org",
		"error should include the org name for debugging context")
}
