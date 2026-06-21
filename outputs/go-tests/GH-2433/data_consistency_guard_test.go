package gcf

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEnsureOrgInMint_DataConsistencyGuard groups all data consistency guard
// scenarios from STD GH-2433. Each subtest targets a distinct guard behavior.
func TestEnsureOrgInMint_DataConsistencyGuard(t *testing.T) {
	const expectedURL = "https://fullsend-mint-test.run.app"

	// TS-GH-2433-001: Guard returns error when ALLOWED_ORGS is empty but
	// ROLE_APP_IDS has role-only entries.
	t.Run("DataInconsistency_ReturnsError", func(t *testing.T) {
		fake := newFakeGCFClient()
		fake.functionInfo = &FunctionInfo{
			URI:     expectedURL,
			EnvVars: map[string]string{},
		}
		fake.trafficEnvVars = map[string]string{
			"ALLOWED_ORGS": "",
			"ROLE_APP_IDS": `{"agent":"app-id-1","reviewer":"app-id-2"}`,
		}

		p := NewProvisioner(Config{ProjectID: "proj1", Region: "us-central1"}, fake)
		err := p.EnsureOrgInMint(context.Background(), expectedURL, "new-org")

		require.Error(t, err, "EnsureOrgInMint must return an error when guard fires")
		assert.Contains(t, err.Error(), "data inconsistency",
			"error must mention data inconsistency")
		assert.NotContains(t, fake.calls, "UpdateServiceEnvVars",
			"no env var update should be attempted when guard fires")
	})

	// TS-GH-2433-002: Enrollment succeeds when both ALLOWED_ORGS and
	// ROLE_APP_IDS are empty (genuine first enrollment).
	t.Run("FirstEnrollment_Succeeds", func(t *testing.T) {
		fake := newFakeGCFClient()
		fake.functionInfo = &FunctionInfo{
			URI:     expectedURL,
			EnvVars: map[string]string{},
		}
		fake.trafficEnvVars = map[string]string{
			"ALLOWED_ORGS": "",
			"ROLE_APP_IDS": "",
		}

		p := NewProvisioner(Config{ProjectID: "proj1", Region: "us-central1"}, fake)
		err := p.EnsureOrgInMint(context.Background(), expectedURL, "first-org")

		require.NoError(t, err, "first enrollment must succeed")
		assert.Contains(t, fake.calls, "UpdateServiceEnvVars",
			"UpdateServiceEnvVars must be called to register the org")
		assert.Contains(t, fake.lastUpdateServiceEnvVars["ALLOWED_ORGS"], "first-org",
			"new org must appear in ALLOWED_ORGS")
	})

	// TS-GH-2433-003: Guard is bypassed when ALLOWED_ORGS has existing orgs.
	t.Run("PopulatedAllowedOrgs_BypassesGuard", func(t *testing.T) {
		fake := newFakeGCFClient()
		fake.functionInfo = &FunctionInfo{
			URI:     expectedURL,
			EnvVars: map[string]string{},
		}
		fake.trafficEnvVars = map[string]string{
			"ALLOWED_ORGS": "existing-org",
			"ROLE_APP_IDS": `{"agent":"app-id-1"}`,
		}

		p := NewProvisioner(Config{ProjectID: "proj1", Region: "us-central1"}, fake)
		err := p.EnsureOrgInMint(context.Background(), expectedURL, "new-org")

		require.NoError(t, err, "guard must not fire when ALLOWED_ORGS is populated")
		assert.Contains(t, fake.calls, "UpdateServiceEnvVars",
			"enrollment should proceed normally")
		require.NotNil(t, fake.lastUpdateServiceEnvVars)
		assert.Contains(t, fake.lastUpdateServiceEnvVars["ALLOWED_ORGS"], "existing-org",
			"existing org must be preserved")
		assert.Contains(t, fake.lastUpdateServiceEnvVars["ALLOWED_ORGS"], "new-org",
			"new org must be added")
	})

	// TS-GH-2433-004: Guard ignores legacy org/role keys (containing '/'),
	// only evaluates role-only keys.
	t.Run("LegacyKeys_Filtering", func(t *testing.T) {
		tests := []struct {
			name       string
			roleAppIDs string
			wantErr    bool
			errSubstr  string
		}{
			{
				name:       "only legacy keys - guard does not fire",
				roleAppIDs: `{"org1/agent":"app-1","org2/reviewer":"app-2"}`,
				wantErr:    false,
			},
			{
				name:       "mixed legacy and role-only keys - guard fires",
				roleAppIDs: `{"org1/agent":"app-1","reviewer":"app-2"}`,
				wantErr:    true,
				errSubstr:  "data inconsistency",
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				fake := newFakeGCFClient()
				fake.functionInfo = &FunctionInfo{
					URI:     expectedURL,
					EnvVars: map[string]string{},
				}
				fake.trafficEnvVars = map[string]string{
					"ALLOWED_ORGS": "",
					"ROLE_APP_IDS": tc.roleAppIDs,
				}

				p := NewProvisioner(Config{ProjectID: "proj1", Region: "us-central1"}, fake)
				err := p.EnsureOrgInMint(context.Background(), expectedURL, "new-org")

				if tc.wantErr {
					require.Error(t, err)
					assert.Contains(t, err.Error(), tc.errSubstr)
					assert.NotContains(t, fake.calls, "UpdateServiceEnvVars",
						"no update when guard fires")
				} else {
					require.NoError(t, err,
						"legacy-only keys should not trigger guard")
					assert.Contains(t, fake.calls, "UpdateServiceEnvVars",
						"enrollment should proceed when only legacy keys present")
				}
			})
		}
	})

	// TS-GH-2433-005: Error contains role count, project ID, and suggested
	// mint status command.
	t.Run("ErrorMessageContent", func(t *testing.T) {
		fake := newFakeGCFClient()
		fake.functionInfo = &FunctionInfo{
			URI:     expectedURL,
			EnvVars: map[string]string{},
		}
		fake.trafficEnvVars = map[string]string{
			"ALLOWED_ORGS": "",
			"ROLE_APP_IDS": `{"agent":"app-1","reviewer":"app-2"}`,
		}

		p := NewProvisioner(Config{ProjectID: "proj1", Region: "us-central1"}, fake)
		err := p.EnsureOrgInMint(context.Background(), expectedURL, "new-org")

		require.Error(t, err)
		errMsg := err.Error()
		assert.Contains(t, errMsg, "2 configured roles",
			"error must contain role count")
		assert.Contains(t, errMsg, "proj1",
			"error must contain project ID")
		assert.Contains(t, errMsg, "fullsend mint status --project=",
			"error must contain suggested investigation command")
	})

	// TS-GH-2433-006: Guard does not trigger on malformed JSON, empty string,
	// or missing ROLE_APP_IDS key.
	t.Run("RoleAppIDs_EdgeCases", func(t *testing.T) {
		tests := []struct {
			name           string
			trafficEnvVars map[string]string
		}{
			{
				name: "malformed JSON",
				trafficEnvVars: map[string]string{
					"ALLOWED_ORGS": "",
					"ROLE_APP_IDS": "{not-valid-json}",
				},
			},
			{
				name: "empty string",
				trafficEnvVars: map[string]string{
					"ALLOWED_ORGS": "",
					"ROLE_APP_IDS": "",
				},
			},
			{
				name: "missing key (not in map)",
				trafficEnvVars: map[string]string{
					"ALLOWED_ORGS": "",
					// ROLE_APP_IDS key absent entirely
				},
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				fake := newFakeGCFClient()
				fake.functionInfo = &FunctionInfo{
					URI:     expectedURL,
					EnvVars: map[string]string{},
				}
				fake.trafficEnvVars = tc.trafficEnvVars

				p := NewProvisioner(Config{ProjectID: "proj1", Region: "us-central1"}, fake)
				err := p.EnsureOrgInMint(context.Background(), expectedURL, "new-org")

				require.NoError(t, err,
					"edge case %q must not trigger guard", tc.name)
				assert.Contains(t, fake.calls, "UpdateServiceEnvVars",
					"enrollment should proceed for edge case %q", tc.name)
			})
		}
	})

	// TS-GH-2433-007: ROLE_APP_IDS is '{}' (empty JSON object) — enrollment
	// proceeds.
	t.Run("EmptyRoleAppIDsObject_Proceeds", func(t *testing.T) {
		fake := newFakeGCFClient()
		fake.functionInfo = &FunctionInfo{
			URI:     expectedURL,
			EnvVars: map[string]string{},
		}
		fake.trafficEnvVars = map[string]string{
			"ALLOWED_ORGS": "",
			"ROLE_APP_IDS": "{}",
		}

		p := NewProvisioner(Config{ProjectID: "proj1", Region: "us-central1"}, fake)
		err := p.EnsureOrgInMint(context.Background(), expectedURL, "new-org")

		require.NoError(t, err,
			"empty ROLE_APP_IDS object must not trigger guard")
		assert.Contains(t, fake.calls, "UpdateServiceEnvVars",
			"enrollment should proceed with empty ROLE_APP_IDS object")
	})

	// TS-GH-2433-008: provisionWithExistingMint and provisionSelfManaged
	// call EnsureOrgInMint and propagate guard errors.
	t.Run("ProvisionFlows_PropagateGuardError", func(t *testing.T) {
		t.Run("provisionWithExistingMint", func(t *testing.T) {
			fake := newFakeGCFClient()
			fake.functionInfo = &FunctionInfo{
				URI:     expectedURL,
				EnvVars: map[string]string{},
			}
			fake.trafficEnvVars = map[string]string{
				"ALLOWED_ORGS": "",
				"ROLE_APP_IDS": `{"agent":"app-1"}`,
			}

			p := NewProvisioner(Config{
				ProjectID:   "my-project",
				Region:      "us-central1",
				MintURL:     expectedURL,
				GitHubOrgs:  []string{"org1"},
				AgentPEMs:   map[string][]byte{"agent": []byte("pem-data")},
				AgentAppIDs: map[string]string{"agent": "app-1"},
			}, fake)
			_, err := p.provisionWithExistingMint(context.Background())

			require.Error(t, err, "provisionWithExistingMint must propagate guard error")
			assert.Contains(t, err.Error(), "data inconsistency",
				"error must originate from the data consistency guard")
		})

		t.Run("provisionSelfManaged", func(t *testing.T) {
			fake := newFakeGCFClient()
			// provisionSelfManaged deploys the full infrastructure first.
			// After deploy it calls EnsureOrgInMint. We set up the function
			// to appear ACTIVE after creation, with traffic env vars that
			// trigger the guard.
			fake.functionInfo = nil // first GetFunction → no existing fn → full deploy
			fake.functionInfoAfterCreate = &FunctionInfo{
				Name:  "projects/my-project/locations/us-central1/functions/fullsend-mint",
				State: "ACTIVE",
				URI:   expectedURL,
				EnvVars: map[string]string{
					"ALLOWED_ORGS": "",
					"ROLE_APP_IDS": `{"agent":"app-1"}`,
				},
			}
			fake.trafficEnvVars = map[string]string{
				"ALLOWED_ORGS": "",
				"ROLE_APP_IDS": `{"agent":"app-1"}`,
			}
			fake.errs["GetSecret"] = ErrSecretNotFound

			p := NewProvisioner(Config{
				ProjectID:         "my-project",
				Region:            "us-central1",
				GitHubOrgs:        []string{"org1"},
				AgentPEMs:         map[string][]byte{"agent": []byte("pem-data")},
				AgentAppIDs:       map[string]string{"agent": "app-1"},
				FunctionSourceDir: fakeFunctionSourceDir(t),
			}, fake)
			p.httpClient = healthyClient()

			_, err := p.provisionSelfManaged(context.Background())

			require.Error(t, err, "provisionSelfManaged must propagate guard error")
			assert.Contains(t, err.Error(), "data inconsistency",
				"error must originate from the data consistency guard")
		})
	})
}

// TestEnsureOrgInMint_CLIIntegration groups CLI-level tests for the data
// consistency guard (TS-GH-2433-009). These verify that guard errors
// surface through the provisioning API boundary with actionable content.
func TestEnsureOrgInMint_CLIIntegration(t *testing.T) {
	const expectedURL = "https://fullsend-mint-test.run.app"

	// TS-GH-2433-009: The provisioner returns an error with actionable
	// information when the data inconsistency guard fires, suitable for
	// CLI consumption.
	t.Run("MintEnroll_SurfacesGuardError", func(t *testing.T) {
		fake := newFakeGCFClient()
		fake.functionInfo = &FunctionInfo{
			URI:     expectedURL,
			EnvVars: map[string]string{},
		}
		fake.trafficEnvVars = map[string]string{
			"ALLOWED_ORGS": "",
			"ROLE_APP_IDS": `{"agent":"app-1","reviewer":"app-2"}`,
		}

		p := NewProvisioner(Config{ProjectID: "proj1", Region: "us-central1"}, fake)
		err := p.EnsureOrgInMint(context.Background(), expectedURL, "org1")

		require.Error(t, err, "guard error must be surfaced")
		errMsg := err.Error()
		assert.Contains(t, errMsg, "data inconsistency",
			"CLI must see 'data inconsistency' in the error")
		assert.Contains(t, errMsg, "fullsend mint status",
			"CLI must see suggested investigation command")
	})
}

// TestEnsureOrgInMint_ConcurrentEnrollment tests the guard behavior under
// concurrent enrollment scenarios (TS-GH-2433-010).
func TestEnsureOrgInMint_ConcurrentEnrollment(t *testing.T) {
	const expectedURL = "https://fullsend-mint-test.run.app"

	// TS-GH-2433-010: Guard fires independently per goroutine when stale
	// data is encountered. Non-stale goroutines proceed normally.
	t.Run("GuardPerGoroutine", func(t *testing.T) {
		// concurrentFake allows per-goroutine env var responses to simulate
		// stale reads for some goroutines but not others.
		type result struct {
			org string
			err error
		}

		// We run two scenarios concurrently:
		// 1. "stale" goroutine sees empty ALLOWED_ORGS + role-only ROLE_APP_IDS → guard fires
		// 2. "fresh" goroutine sees populated ALLOWED_ORGS → guard does not fire
		scenarios := []struct {
			org            string
			trafficEnvVars map[string]string
			wantErr        bool
		}{
			{
				org: "stale-org",
				trafficEnvVars: map[string]string{
					"ALLOWED_ORGS": "",
					"ROLE_APP_IDS": `{"agent":"app-1"}`,
				},
				wantErr: true,
			},
			{
				org: "fresh-org",
				trafficEnvVars: map[string]string{
					"ALLOWED_ORGS": "existing-org",
					"ROLE_APP_IDS": `{"agent":"app-1"}`,
				},
				wantErr: false,
			},
		}

		var (
			mu      sync.Mutex
			results []result
			wg      sync.WaitGroup
		)

		for _, sc := range scenarios {
			wg.Add(1)
			go func(sc struct {
				org            string
				trafficEnvVars map[string]string
				wantErr        bool
			}) {
				defer wg.Done()

				// Each goroutine gets its own fake client to avoid data races.
				fake := newFakeGCFClient()
				fake.functionInfo = &FunctionInfo{
					URI:     expectedURL,
					EnvVars: map[string]string{},
				}
				fake.trafficEnvVars = sc.trafficEnvVars

				p := NewProvisioner(Config{ProjectID: "proj1", Region: "us-central1"}, fake)
				err := p.EnsureOrgInMint(context.Background(), expectedURL, sc.org)

				mu.Lock()
				results = append(results, result{org: sc.org, err: err})
				mu.Unlock()
			}(sc)
		}
		wg.Wait()

		require.Len(t, results, 2, "both goroutines must complete")

		for _, r := range results {
			switch r.org {
			case "stale-org":
				require.Error(t, r.err, "stale-read goroutine must receive guard error")
				assert.Contains(t, r.err.Error(), "data inconsistency")
			case "fresh-org":
				require.NoError(t, r.err, "non-stale goroutine must succeed")
			default:
				t.Fatalf("unexpected org in results: %s", r.org)
			}
		}
	})
}

// Silence the unused import warning for json and fmt — they are used in the
// provisioning flow tests and error assertions above.
var (
	_ = json.Marshal
	_ = fmt.Sprintf
)
