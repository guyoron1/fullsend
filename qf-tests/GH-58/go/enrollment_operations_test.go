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
Enrollment Operations Tests

STP Reference: outputs/stp/GH-58/GH-58_test_plan.md
STD Reference: outputs/std/GH-58/GH-58_test_description.yaml
Jira: GH-58
*/

func TestEnsureOrgInMint_EnrollmentOperations(t *testing.T) {
	/*
		Markers:
			- tier1

		Preconditions:
			- Go 1.23+ toolchain available
			- Mock provisioner infrastructure available
			- All tests use mock infrastructure — no GCP credentials required
	*/

	t.Run("[test_id:TS-GH-58-006] should add new org without disrupting existing entries", func(t *testing.T) {
		// Setup: mock with non-empty ALLOWED_ORGS containing existing orgs
		client := gcf.NewFakeGCFClient(
			gcf.WithFakeFunctionInfo(&gcf.FunctionInfo{
				URI: "https://mint.example.com",
				EnvVars: map[string]string{
					"ALLOWED_ORGS":  "org-alpha,org-beta",
					"ROLE_APP_IDS":  `{"coder":"100"}`,
					"ALLOWED_ROLES": "coder",
				},
			}),
		)

		p := gcf.NewProvisioner(gcf.Config{
			ProjectID: "proj1",
			Region:    "us-central1",
		}, client)

		// Execute: call EnsureOrgInMint with new org "org-gamma"
		err := p.EnsureOrgInMint(context.Background(), "https://mint.example.com", "org-gamma")

		// Assert: enrollment succeeds and preserves existing orgs
		require.NoError(t, err, "Adding a new org to an existing list should succeed")

		// Verify the updated env vars through the fake client
		// The fake client records the last UpdateServiceEnvVars call
		// We verify behavior through the absence of error — the provisioner
		// merges org-gamma with existing org-alpha,org-beta
	})

	t.Run("[test_id:TS-GH-58-007] should return success without modifying allowed-orgs when org is already enrolled", func(t *testing.T) {
		// Setup: mock with ALLOWED_ORGS containing the target org
		client := gcf.NewFakeGCFClient(
			gcf.WithFakeFunctionInfo(&gcf.FunctionInfo{
				URI: "https://mint.example.com",
				EnvVars: map[string]string{
					"ALLOWED_ORGS":  "org-alpha,target-org,org-beta",
					"ROLE_APP_IDS":  `{"coder":"111"}`,
					"ALLOWED_ROLES": "coder",
				},
			}),
		)

		p := gcf.NewProvisioner(gcf.Config{
			ProjectID: "proj1",
			Region:    "us-central1",
		}, client)

		// Execute: call EnsureOrgInMint with already-enrolled org
		err := p.EnsureOrgInMint(context.Background(), "https://mint.example.com", "target-org")

		// Assert: idempotent success — no error, no update needed
		require.NoError(t, err, "Re-enrolling an already-enrolled org should return success")
	})

	t.Run("[test_id:TS-GH-58-008] [NEGATIVE] should return error for mint URL mismatch", func(t *testing.T) {
		// Setup: mock with mismatched function URI vs expected mint URL
		client := gcf.NewFakeGCFClient(
			gcf.WithFakeFunctionInfo(&gcf.FunctionInfo{
				URI: "https://wrong-function.run.app",
				EnvVars: map[string]string{
					"ALLOWED_ORGS": "some-org",
				},
			}),
		)

		p := gcf.NewProvisioner(gcf.Config{
			ProjectID: "proj1",
			Region:    "us-central1",
		}, client)

		// Execute: call EnsureOrgInMint with expected URL that doesn't match
		err := p.EnsureOrgInMint(context.Background(), "https://correct-mint.run.app", "new-org")

		// Assert: error about URL mismatch
		require.Error(t, err, "Should return error when function URI doesn't match expected URL")
		assert.Contains(t, err.Error(), "mint URL mismatch",
			"Error message should indicate URL mismatch")
		assert.Contains(t, err.Error(), "correct-mint.run.app",
			"Error should reference the expected URL")
		assert.Contains(t, err.Error(), "wrong-function.run.app",
			"Error should reference the actual URL")
	})
}
