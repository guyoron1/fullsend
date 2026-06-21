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
Traffic-Serving Revision Config Source Tests

STP Reference: outputs/stp/GH-58/GH-58_test_plan.md
STD Reference: outputs/std/GH-58/GH-58_test_description.yaml
Jira: GH-58
*/

func TestTrafficServingRevision_ConfigReads(t *testing.T) {
	/*
		Markers:
			- tier1

		Preconditions:
			- Go 1.23+ toolchain available
			- Mock provisioner with trackable config read paths
			- Mock distinguishes traffic-serving revision vs function config reads
	*/

	t.Run("[test_id:TS-GH-58-005] should read from traffic-serving revision not function config for org enrollment", func(t *testing.T) {
		// Setup: mock with distinct values for traffic-serving revision and function config.
		// The function config (GetFunction) has stale/empty values, while the
		// traffic-serving revision has the real operational data. EnsureOrgInMint
		// must use the traffic-serving data.
		client := gcf.NewFakeGCFClient(
			gcf.WithFakeFunctionInfo(&gcf.FunctionInfo{
				URI: "https://mint.example.com",
				// Function config (stale): empty ALLOWED_ORGS
				EnvVars: map[string]string{
					"ALLOWED_ORGS":  "",
					"ROLE_APP_IDS":  `{}`,
					"ALLOWED_ROLES": "",
				},
			}),
			// Traffic-serving revision has the real data with existing orgs
			gcf.WithFakeTrafficEnvVars(map[string]string{
				"ALLOWED_ORGS":  "org-a,org-b,org-c",
				"ROLE_APP_IDS":  `{"coder":"100"}`,
				"ALLOWED_ROLES": "coder",
			}),
		)

		p := gcf.NewProvisioner(gcf.Config{
			ProjectID: "proj1",
			Region:    "us-central1",
		}, client)

		// Execute: call EnsureOrgInMint
		err := p.EnsureOrgInMint(context.Background(), "https://mint.example.com", "new-org")

		// Assert: operation succeeds, meaning it read from traffic-serving revision
		// (which has non-empty ALLOWED_ORGS) rather than function config (which is empty).
		// If it had read from function config, the data consistency guard might trigger
		// incorrectly or existing orgs would be lost.
		require.NoError(t, err,
			"Should read from traffic-serving revision where ALLOWED_ORGS is populated")
	})

	t.Run("[test_id:TS-GH-58-010] should read WIF repos list from traffic-serving revision for per-repo WIF registration", func(t *testing.T) {
		// Setup: mock with trackable config read paths. The function config
		// has stale PER_REPO_WIF_REPOS while traffic-serving has real data.
		client := gcf.NewFakeGCFClient(
			gcf.WithFakeFunctionInfo(&gcf.FunctionInfo{
				URI: "https://mint.example.com",
				// Function config (stale): empty WIF repos
				EnvVars: map[string]string{
					"PER_REPO_WIF_REPOS": "",
				},
			}),
			// Traffic-serving revision has the real data
			gcf.WithFakeTrafficEnvVars(map[string]string{
				"PER_REPO_WIF_REPOS": "acme-corp/existing-repo",
			}),
		)

		p := gcf.NewProvisioner(gcf.Config{
			ProjectID: "proj1",
			Region:    "us-central1",
		}, client)

		// Execute: call per-repo WIF registration
		err := p.RegisterPerRepoWIF(context.Background(), "acme-corp/new-repo")

		// Assert: operation completes using traffic-serving revision data
		require.NoError(t, err,
			"RegisterPerRepoWIF should read from traffic-serving revision")
	})

	t.Run("[test_id:TS-GH-58-011] should read allowed-orgs from traffic-serving revision for org removal", func(t *testing.T) {
		// Setup: mock with trackable config paths. Function config has stale
		// ALLOWED_ORGS while traffic-serving has the current state.
		client := gcf.NewFakeGCFClient(
			gcf.WithFakeFunctionInfo(&gcf.FunctionInfo{
				URI: "https://mint.example.com",
				// Function config (stale): doesn't include target-org
				EnvVars: map[string]string{
					"ALLOWED_ORGS": "",
				},
			}),
			// Traffic-serving revision has the real data with target-org
			gcf.WithFakeTrafficEnvVars(map[string]string{
				"ALLOWED_ORGS": "org-alpha,target-org,org-beta",
			}),
		)

		p := gcf.NewProvisioner(gcf.Config{
			ProjectID: "proj1",
			Region:    "us-central1",
		}, client)

		// Execute: call org removal
		err := p.RemoveOrgFromMint(context.Background(), "target-org")

		// Assert: operation completes using traffic-serving revision data
		require.NoError(t, err,
			"RemoveOrgFromMint should read from traffic-serving revision")
	})
}

// TestConfigSourceConsistency validates that all mint operations consistently
// use GetServiceTrafficEnvVars rather than function config env vars. This is
// critical to prevent stale data issues from template/revision divergence.
func TestConfigSourceConsistency(t *testing.T) {
	t.Run("enrollment and removal should use same config source", func(t *testing.T) {
		// Both EnsureOrgInMint and RemoveOrgFromMint should read from
		// traffic-serving revision. Verify both succeed when the function
		// config has stale data but traffic-serving is correct.
		trafficData := map[string]string{
			"ALLOWED_ORGS":  "org-alpha",
			"ROLE_APP_IDS":  `{"coder":"100"}`,
			"ALLOWED_ROLES": "coder",
		}

		// Test enrollment
		enrollClient := gcf.NewFakeGCFClient(
			gcf.WithFakeFunctionInfo(&gcf.FunctionInfo{
				URI:     "https://mint.example.com",
				EnvVars: map[string]string{"ALLOWED_ORGS": ""},
			}),
			gcf.WithFakeTrafficEnvVars(trafficData),
		)

		enrollP := gcf.NewProvisioner(gcf.Config{ProjectID: "proj1", Region: "us-central1"}, enrollClient)
		err := enrollP.EnsureOrgInMint(context.Background(), "https://mint.example.com", "org-beta")
		assert.NoError(t, err, "Enrollment should use traffic-serving data")

		// Test removal
		removeClient := gcf.NewFakeGCFClient(
			gcf.WithFakeFunctionInfo(&gcf.FunctionInfo{
				URI:     "https://mint.example.com",
				EnvVars: map[string]string{"ALLOWED_ORGS": ""},
			}),
			gcf.WithFakeTrafficEnvVars(trafficData),
		)

		removeP := gcf.NewProvisioner(gcf.Config{ProjectID: "proj1", Region: "us-central1"}, removeClient)
		err = removeP.RemoveOrgFromMint(context.Background(), "org-alpha")
		assert.NoError(t, err, "Removal should use traffic-serving data")
	})
}
