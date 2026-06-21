package gcf

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Data Consistency Guard Tests

STP Reference: outputs/stp/GH-2433/GH-2433_test_plan.md
Jira: GH-2433

Tests covering the core data consistency guard in EnsureOrgInMint.
The guard prevents silent org unenrollment when ALLOWED_ORGS is empty
on a bootstrapped mint (ROLE_APP_IDS has role-only entries).
*/

// TestEnsureOrgInMint_DataInconsistency_ReturnsError verifies that the guard
// returns an error when ALLOWED_ORGS is empty but ROLE_APP_IDS contains
// role-only entries, indicating a bootstrapped mint with data loss.
// [test_id:TS-GH-2433-001]
func TestEnsureOrgInMint_DataInconsistency_ReturnsError(t *testing.T) {
	fake := newFakeGCFClient()
	fake.functionInfo = &FunctionInfo{
		URI: "https://fullsend-mint-test.run.app",
	}
	fake.trafficEnvVars = map[string]string{
		"ALLOWED_ORGS": "",
		"ROLE_APP_IDS": `{"coder":"app-id-123","reviewer":"app-id-456"}`,
	}

	p := NewProvisioner(Config{ProjectID: "my-project", Region: "us-central1"}, fake)

	err := p.EnsureOrgInMint(context.Background(), "https://fullsend-mint-test.run.app", "new-org")

	require.Error(t, err, "EnsureOrgInMint should return an error on data inconsistency")
	assert.Contains(t, err.Error(), "data inconsistency",
		"error message should indicate data inconsistency")
}

// TestEnsureOrgInMint_DataInconsistency_ErrorMessageContent verifies that the
// error message includes the role count and project ID for debugging context.
// [test_id:TS-GH-2433-002]
func TestEnsureOrgInMint_DataInconsistency_ErrorMessageContent(t *testing.T) {
	fake := newFakeGCFClient()
	fake.functionInfo = &FunctionInfo{
		URI: "https://fullsend-mint-test.run.app",
	}
	fake.trafficEnvVars = map[string]string{
		"ALLOWED_ORGS": "",
		"ROLE_APP_IDS": `{"coder":"id1","reviewer":"id2"}`,
	}

	p := NewProvisioner(Config{ProjectID: "my-project", Region: "us-central1"}, fake)

	err := p.EnsureOrgInMint(context.Background(), "https://fullsend-mint-test.run.app", "test-org")

	require.Error(t, err)
	errMsg := err.Error()
	assert.Contains(t, errMsg, "2",
		"error message should contain the count of role-only entries")
	assert.Contains(t, errMsg, "my-project",
		"error message should contain the project ID")
	assert.Contains(t, errMsg, "fullsend mint status",
		"error message should suggest running 'fullsend mint status'")
}

// TestEnsureOrgInMint_DataInconsistency_NoEnvVarWrite verifies that when
// the guard triggers, no env var writes occur — the function returns before
// reaching the write path.
// [test_id:TS-GH-2433-003]
func TestEnsureOrgInMint_DataInconsistency_NoEnvVarWrite(t *testing.T) {
	fake := newFakeGCFClient()
	fake.functionInfo = &FunctionInfo{
		URI: "https://fullsend-mint-test.run.app",
	}
	fake.trafficEnvVars = map[string]string{
		"ALLOWED_ORGS": "",
		"ROLE_APP_IDS": `{"coder":"app-id-123"}`,
	}

	p := NewProvisioner(Config{ProjectID: "my-project", Region: "us-central1"}, fake)

	err := p.EnsureOrgInMint(context.Background(), "https://fullsend-mint-test.run.app", "new-org")

	require.Error(t, err)

	// Verify no UpdateServiceEnvVars call was made.
	for _, call := range fake.calls {
		assert.NotEqual(t, "UpdateServiceEnvVars", call,
			"UpdateServiceEnvVars should not be called when guard triggers")
	}
	assert.Nil(t, fake.lastUpdateServiceEnvVars,
		"no env var update should have been captured")

	// Verify ALLOWED_ORGS remains empty in the traffic env vars.
	assert.Equal(t, "", fake.trafficEnvVars["ALLOWED_ORGS"],
		"ALLOWED_ORGS should remain empty after guard triggers")
}

// TestEnsureOrgInMint_MixedKeys_TriggersGuard verifies that the guard triggers
// when ROLE_APP_IDS contains a mix of legacy org-scoped and role-only keys,
// because even one role-only entry with empty ALLOWED_ORGS indicates data loss.
// [test_id:TS-GH-2433-007]
func TestEnsureOrgInMint_MixedKeys_TriggersGuard(t *testing.T) {
	fake := newFakeGCFClient()
	fake.functionInfo = &FunctionInfo{
		URI: "https://fullsend-mint-test.run.app",
	}
	fake.trafficEnvVars = map[string]string{
		"ALLOWED_ORGS": "",
		"ROLE_APP_IDS": `{"acme/coder":"app-id-123","reviewer":"app-id-456"}`,
	}

	p := NewProvisioner(Config{ProjectID: "my-project", Region: "us-central1"}, fake)

	err := p.EnsureOrgInMint(context.Background(), "https://fullsend-mint-test.run.app", "new-org")

	require.Error(t, err, "guard should trigger when role-only entries exist alongside legacy keys")
	assert.True(t, strings.Contains(err.Error(), "data inconsistency") ||
		strings.Contains(err.Error(), "ALLOWED_ORGS"),
		"error should reference data inconsistency")
}
