//go:build e2e

package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/fullsend-ai/fullsend/internal/dispatch/gcf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Error Handling and Edge Case Tests

STP Reference: outputs/stp/GH-58/GH-58_test_plan.md
STD Reference: outputs/std/GH-58/GH-58_test_description.yaml
Jira: GH-58
*/

func TestEnsureOrgInMint_ErrorHandling(t *testing.T) {
	/*
		Markers:
			- tier1

		Preconditions:
			- Go 1.23+ toolchain available
			- Mock provisioner infrastructure available
			- All tests use mock infrastructure — no GCP credentials required
	*/

	t.Run("[test_id:TS-GH-58-012] should proceed without error on malformed app ID registry", func(t *testing.T) {
		// Setup: mock with invalid JSON in ROLE_APP_IDS.
		// The guard attempts json.Unmarshal on ROLE_APP_IDS; malformed JSON
		// should result in roleAppIDMap being nil/empty, so RoleOnlyAppIDs
		// returns nil/empty, and the guard does NOT trigger.
		client := gcf.NewFakeGCFClient(
			gcf.WithFakeFunctionInfo(&gcf.FunctionInfo{
				URI:     "https://mint.example.com",
				EnvVars: map[string]string{},
			}),
			gcf.WithFakeTrafficEnvVars(map[string]string{
				"ALLOWED_ORGS": "",
				"ROLE_APP_IDS": `{invalid json content here`,
			}),
		)

		p := gcf.NewProvisioner(gcf.Config{
			ProjectID: "proj1",
			Region:    "us-central1",
		}, client)

		// Execute: call EnsureOrgInMint
		err := p.EnsureOrgInMint(context.Background(), "https://mint.example.com", "new-org")

		// Assert: enrollment proceeds — malformed JSON is treated as empty
		// (json.Unmarshal error is silently ignored with _ = ...)
		require.NoError(t, err,
			"Enrollment should proceed when ROLE_APP_IDS contains malformed JSON (treated as empty)")
	})

	t.Run("[test_id:TS-GH-58-013] [NEGATIVE] should fail with a clear error on API failure", func(t *testing.T) {
		// Setup: mock that returns API error when reading traffic-serving revision
		client := gcf.NewFakeGCFClient(
			gcf.WithFakeFunctionInfo(&gcf.FunctionInfo{
				URI:     "https://mint.example.com",
				EnvVars: map[string]string{},
			}),
			gcf.WithFakeErrors(map[string]error{
				"GetServiceTrafficEnvVars": fmt.Errorf("Cloud Run API unavailable"),
			}),
		)

		p := gcf.NewProvisioner(gcf.Config{
			ProjectID: "proj1",
			Region:    "us-central1",
		}, client)

		// Execute: call EnsureOrgInMint
		err := p.EnsureOrgInMint(context.Background(), "https://mint.example.com", "new-org")

		// Assert: error returned with clear message about config read failure
		require.Error(t, err, "Should fail when traffic-serving revision API returns error")
		assert.Contains(t, err.Error(), "traffic-serving env vars",
			"Error should indicate the config read failure source")
		assert.NotContains(t, err.Error(), "data inconsistency",
			"API errors should be distinguishable from data inconsistency errors")
	})

	t.Run("[test_id:TS-GH-58-014] should handle corrupt allowed-orgs data gracefully without panicking", func(t *testing.T) {
		// Setup: mock with corrupt/malformed ALLOWED_ORGS data (null bytes, binary data)
		client := gcf.NewFakeGCFClient(
			gcf.WithFakeFunctionInfo(&gcf.FunctionInfo{
				URI:     "https://mint.example.com",
				EnvVars: map[string]string{},
			}),
			gcf.WithFakeTrafficEnvVars(map[string]string{
				"ALLOWED_ORGS":  "\x00\xff\xfe invalid data",
				"ROLE_APP_IDS":  `{"coder":"100"}`,
				"ALLOWED_ROLES": "coder",
			}),
		)

		p := gcf.NewProvisioner(gcf.Config{
			ProjectID: "proj1",
			Region:    "us-central1",
		}, client)

		// Execute: call EnsureOrgInMint — must not panic
		require.NotPanics(t, func() {
			// The function uses strings.Split on ALLOWED_ORGS, which handles
			// arbitrary string data without panicking. The corrupt data becomes
			// one of the entries in the split result.
			_ = p.EnsureOrgInMint(context.Background(), "https://mint.example.com", "new-org")
		}, "Should handle corrupt ALLOWED_ORGS data without panicking")
	})

	t.Run("[test_id:TS-GH-58-015] [NEGATIVE] should fail with a clear error on missing traffic-serving revision", func(t *testing.T) {
		// Setup: mock that simulates service with no traffic-serving revision.
		// GetFunction succeeds but GetServiceTrafficEnvVars returns error
		// because there's no serving revision.
		client := gcf.NewFakeGCFClient(
			gcf.WithFakeFunctionInfo(&gcf.FunctionInfo{
				URI:     "https://mint.example.com",
				EnvVars: map[string]string{},
			}),
			gcf.WithFakeErrors(map[string]error{
				"GetServiceTrafficEnvVars": fmt.Errorf("no traffic-serving revision found for service"),
			}),
		)

		p := gcf.NewProvisioner(gcf.Config{
			ProjectID: "proj1",
			Region:    "us-central1",
		}, client)

		// Execute: call EnsureOrgInMint
		err := p.EnsureOrgInMint(context.Background(), "https://mint.example.com", "new-org")

		// Assert: error about missing revision
		require.Error(t, err, "Should fail when no traffic-serving revision exists")
		assert.Contains(t, err.Error(), "traffic-serving",
			"Error should reference the traffic-serving revision")
	})
}
