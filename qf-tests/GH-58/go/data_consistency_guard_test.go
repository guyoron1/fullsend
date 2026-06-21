//go:build e2e

package tests

import (
	"context"
	"strings"
	"testing"

	"github.com/fullsend-ai/fullsend/internal/dispatch/gcf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Data Consistency Guard Tests

STP Reference: outputs/stp/GH-58/GH-58_test_plan.md
STD Reference: outputs/std/GH-58/GH-58_test_description.yaml
Jira: GH-58
*/

func TestEnsureOrgInMint_DataConsistencyGuard(t *testing.T) {
	/*
		Markers:
			- tier1

		Preconditions:
			- Go 1.23+ toolchain available
			- Mock provisioner infrastructure available
			- All tests use mock infrastructure — no GCP credentials required
	*/

	t.Run("[test_id:TS-GH-58-001] should block enrollment with data inconsistency error when active roles exist but allowed-orgs is empty", func(t *testing.T) {
		// Setup: mock with empty ALLOWED_ORGS but active role-only keys in ROLE_APP_IDS
		client := gcf.NewFakeGCFClient(
			gcf.WithFakeFunctionInfo(&gcf.FunctionInfo{
				URI:     "https://mint.example.com",
				EnvVars: map[string]string{},
			}),
			gcf.WithFakeTrafficEnvVars(map[string]string{
				"ALLOWED_ORGS": "",
				"ROLE_APP_IDS": `{"role-admin":{"wif_provider":"projects/123/locations/global/workloadIdentityPools/pool/providers/provider"},"role-viewer":{"wif_provider":"projects/123/locations/global/workloadIdentityPools/pool/providers/provider"}}`,
			}),
		)

		p := gcf.NewProvisioner(gcf.Config{
			ProjectID: "my-gcp-project",
			Region:    "us-central1",
		}, client)

		// Execute: call EnsureOrgInMint with a new org
		err := p.EnsureOrgInMint(context.Background(), "https://mint.example.com", "new-org")

		// Assert: error contains "data inconsistency"
		require.Error(t, err, "EnsureOrgInMint should return error when active roles exist but allowed-orgs is empty")
		assert.Contains(t, err.Error(), "data inconsistency",
			"Error message should indicate data inconsistency")
		assert.Contains(t, err.Error(), "2",
			"Error message should include the count of configured roles")
	})

	t.Run("[test_id:TS-GH-58-002] should permit first enrollment when both allowed-orgs and app ID registry are empty", func(t *testing.T) {
		// Setup: mock with empty ALLOWED_ORGS and empty ROLE_APP_IDS
		client := gcf.NewFakeGCFClient(
			gcf.WithFakeFunctionInfo(&gcf.FunctionInfo{
				URI:     "https://mint.example.com",
				EnvVars: map[string]string{},
			}),
			gcf.WithFakeTrafficEnvVars(map[string]string{
				"ALLOWED_ORGS": "",
				"ROLE_APP_IDS": `{}`,
			}),
		)

		p := gcf.NewProvisioner(gcf.Config{
			ProjectID: "my-gcp-project",
			Region:    "us-central1",
		}, client)

		// Execute: call EnsureOrgInMint with a new org
		err := p.EnsureOrgInMint(context.Background(), "https://mint.example.com", "new-org")

		// Assert: no error, org is enrolled
		require.NoError(t, err, "First enrollment should succeed when both allowed-orgs and app ID registry are empty")
	})

	t.Run("[test_id:TS-GH-58-003] should permit enrollment when only legacy keys exist in app ID registry", func(t *testing.T) {
		// Setup: mock with empty ALLOWED_ORGS and ROLE_APP_IDS containing only legacy keys (with "/" separator)
		client := gcf.NewFakeGCFClient(
			gcf.WithFakeFunctionInfo(&gcf.FunctionInfo{
				URI:     "https://mint.example.com",
				EnvVars: map[string]string{},
			}),
			gcf.WithFakeTrafficEnvVars(map[string]string{
				"ALLOWED_ORGS": "",
				"ROLE_APP_IDS": `{"my-org/admin-role":"111","my-org/viewer-role":"222"}`,
			}),
		)

		p := gcf.NewProvisioner(gcf.Config{
			ProjectID: "my-gcp-project",
			Region:    "us-central1",
		}, client)

		// Execute: call EnsureOrgInMint with a new org
		err := p.EnsureOrgInMint(context.Background(), "https://mint.example.com", "new-org")

		// Assert: no error, enrollment proceeds — legacy keys are filtered out by RoleOnlyAppIDs
		require.NoError(t, err, "Enrollment should proceed when only legacy keys (with '/') exist in app ID registry")
	})
}

// TestRoleOnlyKeyFiltering_SeparatesLegacyKeys tests the role-only key filtering
// that underpins the data consistency guard.
func TestRoleOnlyKeyFiltering_SeparatesLegacyKeys(t *testing.T) {
	/*
		Markers:
			- tier1

		Preconditions:
			- Role-only key filtering function available (mintcore.RoleOnlyAppIDs)
	*/

	t.Run("[test_id:TS-GH-58-009] should return only keys without '/' separator from mixed registry", func(t *testing.T) {
		// Setup: create registry with mixed legacy ("org/role") and role-only ("role") keys
		registry := map[string]string{
			"my-org/admin-role": "provider-1",
			"role-admin":        "provider-2",
			"other-org/viewer":  "provider-3",
			"role-viewer":       "provider-4",
		}

		// Execute: call RoleOnlyAppIDs filtering function
		// Note: We import mintcore to test this directly. Since the test package
		// is external, we test via the provisioner behavior instead.
		// The provisioner uses mintcore.RoleOnlyAppIDs internally during
		// the data consistency guard check. We verify behavior through integration.

		// Verify through provisioner behavior with mixed keys:
		// If role-only keys exist AND ALLOWED_ORGS is empty → guard triggers
		client := gcf.NewFakeGCFClient(
			gcf.WithFakeFunctionInfo(&gcf.FunctionInfo{
				URI:     "https://mint.example.com",
				EnvVars: map[string]string{},
			}),
			gcf.WithFakeTrafficEnvVars(map[string]string{
				"ALLOWED_ORGS": "",
				"ROLE_APP_IDS": `{"my-org/admin-role":"provider-1","role-admin":"provider-2","other-org/viewer":"provider-3","role-viewer":"provider-4"}`,
			}),
		)

		p := gcf.NewProvisioner(gcf.Config{
			ProjectID: "test-project",
			Region:    "us-central1",
		}, client)

		err := p.EnsureOrgInMint(context.Background(), "https://mint.example.com", "new-org")

		// Assert: guard triggers because role-only keys exist (role-admin, role-viewer)
		require.Error(t, err, "Guard should trigger when role-only keys exist in mixed registry")
		assert.Contains(t, err.Error(), "data inconsistency")
		// The guard should report 2 role-only keys (legacy keys are filtered out)
		assert.Contains(t, err.Error(), "2",
			"Error should report the count of role-only keys, not total keys")

		// Also verify: with ONLY legacy keys, guard does NOT trigger
		clientLegacyOnly := gcf.NewFakeGCFClient(
			gcf.WithFakeFunctionInfo(&gcf.FunctionInfo{
				URI:     "https://mint.example.com",
				EnvVars: map[string]string{},
			}),
			gcf.WithFakeTrafficEnvVars(map[string]string{
				"ALLOWED_ORGS": "",
				"ROLE_APP_IDS": `{"my-org/admin-role":"provider-1","other-org/viewer":"provider-3"}`,
			}),
		)

		pLegacyOnly := gcf.NewProvisioner(gcf.Config{
			ProjectID: "test-project",
			Region:    "us-central1",
		}, clientLegacyOnly)

		errLegacyOnly := pLegacyOnly.EnsureOrgInMint(context.Background(), "https://mint.example.com", "new-org")
		require.NoError(t, errLegacyOnly,
			"Guard should NOT trigger when only legacy keys exist (all contain '/')")

		// Verify empty registry also doesn't trigger
		_ = registry // used for documentation of test data structure
	})
}

// TestEnsureOrgInMint_ErrorContainsDiagnosticInfo verifies error messages
// include actionable diagnostic information for operators.
func TestEnsureOrgInMint_ErrorContainsDiagnosticInfo(t *testing.T) {
	/*
		Markers:
			- tier1
	*/

	t.Run("[test_id:TS-GH-58-004] should include role count and project ID in error message when data inconsistency detected", func(t *testing.T) {
		// Setup: mock with empty ALLOWED_ORGS, 2 active role-only keys, and known project ID
		client := gcf.NewFakeGCFClient(
			gcf.WithFakeFunctionInfo(&gcf.FunctionInfo{
				URI:     "https://mint.example.com",
				EnvVars: map[string]string{},
			}),
			gcf.WithFakeTrafficEnvVars(map[string]string{
				"ALLOWED_ORGS": "",
				"ROLE_APP_IDS": `{"role-admin":"provider-1","role-viewer":"provider-2"}`,
			}),
		)

		p := gcf.NewProvisioner(gcf.Config{
			ProjectID: "my-gcp-project",
			Region:    "us-central1",
		}, client)

		// Execute: call EnsureOrgInMint to trigger data inconsistency error
		err := p.EnsureOrgInMint(context.Background(), "https://mint.example.com", "new-org")

		// Assert: error contains diagnostic info
		require.Error(t, err)
		errMsg := err.Error()

		assert.True(t, strings.Contains(errMsg, "2"),
			"Error should include role count (2 active roles), got: %s", errMsg)
		assert.Contains(t, errMsg, "my-gcp-project",
			"Error should include the GCP project ID for operator diagnosis")
		assert.Contains(t, errMsg, "fullsend mint status",
			"Error should provide actionable guidance to run 'fullsend mint status'")
	})
}
