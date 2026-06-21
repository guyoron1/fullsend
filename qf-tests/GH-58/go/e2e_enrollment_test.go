//go:build e2e

package tests

import (
	"context"
	"testing"

	"github.com/fullsend-ai/fullsend/internal/dispatch/gcf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
End-to-End Enrollment Tests

STP Reference: outputs/stp/GH-58/GH-58_test_plan.md
STD Reference: outputs/std/GH-58/GH-58_test_description.yaml
Jira: GH-58

Note: This is a Tier 2 test. tier2_tests is currently disabled for
this project. When enabled, this test exercises the full enrollment
path through the provisioner with comprehensive assertions.
*/

func TestMintEnrollOrg_EndToEnd(t *testing.T) {
	/*
		Markers:
			- tier2
			- e2e

		Preconditions:
			- Go 1.23+ toolchain available
			- fullsend binary or provisioner API available
			- Mock GCP services configured
	*/

	t.Run("[test_id:TS-GH-58-016] should succeed end-to-end with guard protecting against stale reads", func(t *testing.T) {
		// This test exercises the full enrollment flow through the provisioner,
		// validating that the data consistency guard is active and the
		// traffic-serving revision is used for all config reads.
		//
		// In a full e2e setup, this would invoke the CLI binary. For the
		// mock-based version, we exercise the provisioner through its complete
		// enrollment path including guard checks.

		// Setup: configure mock with realistic production-like state
		// - Existing orgs enrolled
		// - Role-only app IDs configured (guard relevant)
		// - Traffic-serving revision has current data
		client := gcf.NewFakeGCFClient(
			gcf.WithFakeFunctionInfo(&gcf.FunctionInfo{
				URI: "https://mint-prod.us-central1.run.app",
				EnvVars: map[string]string{
					"ALLOWED_ORGS":           "org-alpha,org-beta",
					"ROLE_APP_IDS":           `{"coder":"100","reviewer":"200"}`,
					"ALLOWED_ROLES":          "coder,reviewer",
					"ALLOWED_WORKFLOW_FILES": "*",
				},
			}),
			gcf.WithFakeTrafficEnvVars(map[string]string{
				"ALLOWED_ORGS":           "org-alpha,org-beta",
				"ROLE_APP_IDS":           `{"coder":"100","reviewer":"200"}`,
				"ALLOWED_ROLES":          "coder,reviewer",
				"ALLOWED_WORKFLOW_FILES": "*",
			}),
		)

		p := gcf.NewProvisioner(gcf.Config{
			ProjectID: "prod-project-123",
			Region:    "us-central1",
		}, client)

		// Execute: enroll a new org (the core operation)
		err := p.EnsureOrgInMint(
			context.Background(),
			"https://mint-prod.us-central1.run.app",
			"test-org",
		)

		// Assert: enrollment succeeds end-to-end
		require.NoError(t, err, "End-to-end enrollment should succeed")

		// Verify guard was exercised: enrollment with existing orgs + role-only keys
		// means the guard checked for data inconsistency and correctly allowed
		// the operation (because ALLOWED_ORGS is not empty).

		// Verify idempotency: re-enrollment should also succeed
		err = p.EnsureOrgInMint(
			context.Background(),
			"https://mint-prod.us-central1.run.app",
			"test-org",
		)
		assert.NoError(t, err, "Re-enrollment should be idempotent")

		// Verify guard blocks when it should: simulate stale read scenario
		staleClient := gcf.NewFakeGCFClient(
			gcf.WithFakeFunctionInfo(&gcf.FunctionInfo{
				URI:     "https://mint-prod.us-central1.run.app",
				EnvVars: map[string]string{},
			}),
			gcf.WithFakeTrafficEnvVars(map[string]string{
				"ALLOWED_ORGS": "",
				"ROLE_APP_IDS": `{"coder":"100","reviewer":"200"}`,
			}),
		)

		staleP := gcf.NewProvisioner(gcf.Config{
			ProjectID: "prod-project-123",
			Region:    "us-central1",
		}, staleClient)

		staleErr := staleP.EnsureOrgInMint(
			context.Background(),
			"https://mint-prod.us-central1.run.app",
			"another-org",
		)
		require.Error(t, staleErr, "Guard should block enrollment when stale read detected")
		assert.Contains(t, staleErr.Error(), "data inconsistency",
			"Guard error should indicate data inconsistency")
	})
}
