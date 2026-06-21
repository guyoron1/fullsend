package gcf

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

/*
Malformed Input Tests

STP Reference: outputs/stp/GH-2433/GH-2433_test_plan.md
Jira: GH-2433

Tests covering edge cases with malformed or empty ROLE_APP_IDS values.
Ensures the guard handles JSON parse failures gracefully without panics
or false positive triggers.
*/

// TestEnsureOrgInMint_MalformedRoleAppIDs_Proceeds verifies that enrollment
// proceeds when ROLE_APP_IDS contains malformed JSON. The unmarshal error is
// intentionally ignored so that corrupted env vars do not block enrollment.
// [test_id:TS-GH-2433-015]
func TestEnsureOrgInMint_MalformedRoleAppIDs_Proceeds(t *testing.T) {
	fake := newFakeGCFClient()
	fake.functionInfo = &FunctionInfo{
		URI: "https://fullsend-mint-test.run.app",
	}
	fake.trafficEnvVars = map[string]string{
		"ALLOWED_ORGS": "",
		"ROLE_APP_IDS": "{not valid json",
	}

	p := NewProvisioner(Config{ProjectID: "my-project", Region: "us-central1"}, fake)

	// Should not panic — malformed JSON treated as empty (first enrollment path).
	require.NotPanics(t, func() {
		err := p.EnsureOrgInMint(context.Background(), "https://fullsend-mint-test.run.app", "new-org")
		require.NoError(t, err,
			"malformed JSON in ROLE_APP_IDS should not block enrollment")
	})
}

// TestEnsureOrgInMint_EmptyRoleAppIDsString_NoPanic verifies that an empty
// string ROLE_APP_IDS is handled gracefully without panicking or triggering
// the guard. This covers the edge case where the env var exists but is empty.
// [test_id:TS-GH-2433-016]
func TestEnsureOrgInMint_EmptyRoleAppIDsString_NoPanic(t *testing.T) {
	fake := newFakeGCFClient()
	fake.functionInfo = &FunctionInfo{
		URI: "https://fullsend-mint-test.run.app",
	}
	fake.trafficEnvVars = map[string]string{
		"ALLOWED_ORGS": "",
		"ROLE_APP_IDS": "",
	}

	p := NewProvisioner(Config{ProjectID: "my-project", Region: "us-central1"}, fake)

	// Should not panic — empty string is a valid value.
	require.NotPanics(t, func() {
		err := p.EnsureOrgInMint(context.Background(), "https://fullsend-mint-test.run.app", "new-org")
		require.NoError(t, err,
			"empty string ROLE_APP_IDS should be treated as first enrollment")
	})
}
