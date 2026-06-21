package gcf

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
First Enrollment Tests

STP Reference: outputs/stp/GH-2433/GH-2433_test_plan.md
Jira: GH-2433

Tests covering the first enrollment path where both ALLOWED_ORGS and
ROLE_APP_IDS are empty, representing a legitimate fresh mint bootstrap.
The guard must not produce false positives in this case.
*/

// TestEnsureOrgInMint_FirstEnrollment_Succeeds verifies that enrollment
// succeeds when both ALLOWED_ORGS and ROLE_APP_IDS are empty, representing
// a genuine first enrollment on a fresh mint.
// [test_id:TS-GH-2433-004]
func TestEnsureOrgInMint_FirstEnrollment_Succeeds(t *testing.T) {
	fake := newFakeGCFClient()
	fake.functionInfo = &FunctionInfo{
		URI: "https://fullsend-mint-test.run.app",
	}
	fake.trafficEnvVars = map[string]string{
		"ALLOWED_ORGS": "",
		"ROLE_APP_IDS": "",
	}

	p := NewProvisioner(Config{ProjectID: "my-project", Region: "us-central1"}, fake)

	err := p.EnsureOrgInMint(context.Background(), "https://fullsend-mint-test.run.app", "first-org")

	require.NoError(t, err, "first enrollment on a fresh mint should succeed")
}

// TestEnsureOrgInMint_FirstEnrollment_WritesAllowedOrgs verifies that after
// a successful first enrollment, ALLOWED_ORGS is written with the new org.
// [test_id:TS-GH-2433-005]
func TestEnsureOrgInMint_FirstEnrollment_WritesAllowedOrgs(t *testing.T) {
	fake := newFakeGCFClient()
	fake.functionInfo = &FunctionInfo{
		URI: "https://fullsend-mint-test.run.app",
	}
	fake.trafficEnvVars = map[string]string{
		"ALLOWED_ORGS": "",
		"ROLE_APP_IDS": "",
	}

	p := NewProvisioner(Config{ProjectID: "my-project", Region: "us-central1"}, fake)

	err := p.EnsureOrgInMint(context.Background(), "https://fullsend-mint-test.run.app", "new-org")
	require.NoError(t, err)

	// Verify UpdateServiceEnvVars was called.
	updateCalls := 0
	for _, call := range fake.calls {
		if call == "UpdateServiceEnvVars" {
			updateCalls++
		}
	}
	assert.Equal(t, 1, updateCalls,
		"exactly one UpdateServiceEnvVars call should be made")

	// Verify ALLOWED_ORGS contains the new org.
	require.NotNil(t, fake.lastUpdateServiceEnvVars,
		"env var update should have been captured")
	assert.Contains(t, fake.lastUpdateServiceEnvVars["ALLOWED_ORGS"], "new-org",
		"ALLOWED_ORGS should contain the enrolled org")
}
