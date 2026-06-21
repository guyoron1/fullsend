package gcf

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

/*
Legacy Key Handling Tests

STP Reference: outputs/stp/GH-2433/GH-2433_test_plan.md
Jira: GH-2433

Tests covering legacy org-scoped ROLE_APP_IDS key handling. Keys containing
'/' (e.g., "acme/coder") are org-scoped legacy format and should be filtered
out by RoleOnlyAppIDs, preventing false positive guard triggers.
*/

// TestEnsureOrgInMint_LegacyKeysOnly_Proceeds verifies that enrollment
// proceeds when ROLE_APP_IDS contains only legacy org-scoped keys. These
// are filtered out by RoleOnlyAppIDs, so the guard should not trigger.
// [test_id:TS-GH-2433-006]
func TestEnsureOrgInMint_LegacyKeysOnly_Proceeds(t *testing.T) {
	fake := newFakeGCFClient()
	fake.functionInfo = &FunctionInfo{
		URI: "https://fullsend-mint-test.run.app",
	}
	fake.trafficEnvVars = map[string]string{
		"ALLOWED_ORGS": "",
		"ROLE_APP_IDS": `{"acme/coder":"app-id-123","globex/reviewer":"app-id-456"}`,
	}

	p := NewProvisioner(Config{ProjectID: "my-project", Region: "us-central1"}, fake)

	err := p.EnsureOrgInMint(context.Background(), "https://fullsend-mint-test.run.app", "new-org")

	require.NoError(t, err, "guard should not trigger for legacy-only ROLE_APP_IDS")
}
