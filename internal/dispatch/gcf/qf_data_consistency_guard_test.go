package gcf

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =========================================================================
// Data Consistency Guard — EnsureOrgInMint
//
// STP Reference: outputs/stp/GH-74/GH-74_test_plan.md
// STD Reference: outputs/std/GH-74/GH-74_test_description.yaml
// Jira: GH-74 (Epic: GH-2433)
// =========================================================================

const (
	qfTestProjectID  = "proj1-test"
	qfTestRegion     = "us-central1"
	qfTestMintURL    = "https://fullsend-mint-test.run.app"
	qfTestFnName     = "fullsend-mint"
)

// qfFunctionInfo returns a FunctionInfo matching the standard test mint URL.
func qfFunctionInfo() *FunctionInfo {
	return &FunctionInfo{
		Name:  "projects/" + qfTestProjectID + "/locations/" + qfTestRegion + "/functions/" + qfTestFnName,
		State: "ACTIVE",
		URI:   qfTestMintURL,
	}
}

// qfProvisioner creates a Provisioner wired to the given fake client.
func qfProvisioner(fake GCFClient) *Provisioner {
	return NewProvisioner(Config{
		ProjectID: qfTestProjectID,
		Region:    qfTestRegion,
	}, fake)
}

// =========================================================================
// Requirement: Guard detects data inconsistency
// =========================================================================

// TestEnsureOrgInMint_GuardDetection validates that the data consistency
// guard correctly detects and blocks enrollment when ALLOWED_ORGS is empty
// but ROLE_APP_IDS has role-only entries.
func TestEnsureOrgInMint_GuardDetection(t *testing.T) {
	t.Run("[test_id:TS-GH-2433-001] should return error on empty ALLOWED_ORGS with role-only entries", func(t *testing.T) {
		fake := newFakeGCFClient()
		fake.functionInfo = qfFunctionInfo()
		fake.trafficEnvVars = map[string]string{
			"ALLOWED_ORGS": "",
			"ROLE_APP_IDS": `{"agent":"app-id-1","reviewer":"app-id-2"}`,
		}

		p := qfProvisioner(fake)
		err := p.EnsureOrgInMint(context.Background(), qfTestMintURL, "new-org")

		require.Error(t, err, "EnsureOrgInMint must return error on data inconsistency")
		assert.Contains(t, err.Error(), "data inconsistency",
			"Error message must mention data inconsistency")
	})

	t.Run("[test_id:TS-GH-2433-002] should not call UpdateServiceEnvVars when guard fires", func(t *testing.T) {
		fake := newFakeGCFClient()
		fake.functionInfo = qfFunctionInfo()
		fake.trafficEnvVars = map[string]string{
			"ALLOWED_ORGS": "",
			"ROLE_APP_IDS": `{"agent":"app-id-1","reviewer":"app-id-2"}`,
		}

		p := qfProvisioner(fake)
		_ = p.EnsureOrgInMint(context.Background(), qfTestMintURL, "new-org")

		assert.NotContains(t, fake.calls, "UpdateServiceEnvVars",
			"UpdateServiceEnvVars must NOT be called when guard fires")
	})

	t.Run("[test_id:TS-GH-2433-003] should permit enrollment when ALLOWED_ORGS populated", func(t *testing.T) {
		fake := newFakeGCFClient()
		fake.functionInfo = qfFunctionInfo()
		fake.trafficEnvVars = map[string]string{
			"ALLOWED_ORGS": "existing-org",
			"ROLE_APP_IDS": `{"agent":"app-id-1"}`,
		}

		p := qfProvisioner(fake)
		err := p.EnsureOrgInMint(context.Background(), qfTestMintURL, "new-org")

		require.NoError(t, err, "EnsureOrgInMint must succeed with populated ALLOWED_ORGS")
		assert.Contains(t, fake.calls, "UpdateServiceEnvVars",
			"UpdateServiceEnvVars must be called for normal enrollment")
	})
}

// =========================================================================
// Requirement: First enrollment succeeds
// =========================================================================

// TestEnsureOrgInMint_FirstEnrollment validates that the guard allows
// genuine first enrollment (both ALLOWED_ORGS and ROLE_APP_IDS empty).
func TestEnsureOrgInMint_FirstEnrollment(t *testing.T) {
	t.Run("[test_id:TS-GH-2433-004] should succeed with empty state (first enrollment)", func(t *testing.T) {
		fake := newFakeGCFClient()
		fake.functionInfo = qfFunctionInfo()
		fake.trafficEnvVars = map[string]string{
			"ALLOWED_ORGS": "",
			"ROLE_APP_IDS": "",
		}

		p := qfProvisioner(fake)
		err := p.EnsureOrgInMint(context.Background(), qfTestMintURL, "first-org")

		require.NoError(t, err, "First enrollment must succeed when both env vars empty")
		assert.Contains(t, fake.lastUpdateServiceEnvVars["ALLOWED_ORGS"], "first-org",
			"New org must appear in ALLOWED_ORGS after enrollment")
	})

	t.Run("[test_id:TS-GH-2433-005] should call UpdateServiceEnvVars on first enrollment", func(t *testing.T) {
		fake := newFakeGCFClient()
		fake.functionInfo = qfFunctionInfo()
		fake.trafficEnvVars = map[string]string{
			"ALLOWED_ORGS": "",
			"ROLE_APP_IDS": "",
		}

		p := qfProvisioner(fake)
		err := p.EnsureOrgInMint(context.Background(), qfTestMintURL, "first-org")

		require.NoError(t, err)
		assert.Contains(t, fake.calls, "UpdateServiceEnvVars",
			"UpdateServiceEnvVars must be called during first enrollment")
	})
}

// =========================================================================
// Requirement: Guard bypass with populated ALLOWED_ORGS
// =========================================================================

// TestEnsureOrgInMint_GuardBypass validates that the guard is bypassed
// when ALLOWED_ORGS already has orgs.
func TestEnsureOrgInMint_GuardBypass(t *testing.T) {
	t.Run("[test_id:TS-GH-2433-006] should bypass guard with populated ALLOWED_ORGS", func(t *testing.T) {
		fake := newFakeGCFClient()
		fake.functionInfo = qfFunctionInfo()
		fake.trafficEnvVars = map[string]string{
			"ALLOWED_ORGS": "existing-org",
			"ROLE_APP_IDS": `{"agent":"app-id-1"}`,
		}

		p := qfProvisioner(fake)
		err := p.EnsureOrgInMint(context.Background(), qfTestMintURL, "new-org")

		require.NoError(t, err, "Guard must be bypassed with populated ALLOWED_ORGS")
		assert.Contains(t, fake.calls, "UpdateServiceEnvVars",
			"Enrollment must proceed when guard bypassed")
	})

	t.Run("[test_id:TS-GH-2433-007] should preserve existing orgs during new enrollment", func(t *testing.T) {
		fake := newFakeGCFClient()
		fake.functionInfo = qfFunctionInfo()
		fake.trafficEnvVars = map[string]string{
			"ALLOWED_ORGS": "existing-org",
			"ROLE_APP_IDS": `{"agent":"app-id-1"}`,
		}

		p := qfProvisioner(fake)
		err := p.EnsureOrgInMint(context.Background(), qfTestMintURL, "new-org")

		require.NoError(t, err)
		updatedOrgs := fake.lastUpdateServiceEnvVars["ALLOWED_ORGS"]
		assert.Contains(t, updatedOrgs, "existing-org",
			"Existing org must be preserved in ALLOWED_ORGS")
		assert.Contains(t, updatedOrgs, "new-org",
			"New org must be added to ALLOWED_ORGS")
	})
}

// =========================================================================
// Requirement: Legacy key filtering
// =========================================================================

// TestEnsureOrgInMint_LegacyKeys validates legacy org/role key filtering
// in ROLE_APP_IDS.
func TestEnsureOrgInMint_LegacyKeys(t *testing.T) {
	t.Run("[test_id:TS-GH-2433-008] legacy-only keys should not trigger guard", func(t *testing.T) {
		fake := newFakeGCFClient()
		fake.functionInfo = qfFunctionInfo()
		fake.trafficEnvVars = map[string]string{
			"ALLOWED_ORGS": "",
			"ROLE_APP_IDS": `{"org1/agent":"app-1","org2/reviewer":"app-2"}`,
		}

		p := qfProvisioner(fake)
		err := p.EnsureOrgInMint(context.Background(), qfTestMintURL, "new-org")

		require.NoError(t, err, "Legacy-only ROLE_APP_IDS keys must not trigger guard")
		assert.Contains(t, fake.calls, "UpdateServiceEnvVars",
			"Enrollment must proceed with legacy-only keys")
	})

	t.Run("[test_id:TS-GH-2433-009] mixed legacy and role-only keys should trigger guard", func(t *testing.T) {
		fake := newFakeGCFClient()
		fake.functionInfo = qfFunctionInfo()
		fake.trafficEnvVars = map[string]string{
			"ALLOWED_ORGS": "",
			"ROLE_APP_IDS": `{"org1/agent":"app-1","reviewer":"app-2"}`,
		}

		p := qfProvisioner(fake)
		err := p.EnsureOrgInMint(context.Background(), qfTestMintURL, "new-org")

		require.Error(t, err, "Mixed keys with role-only entries must trigger guard")
		assert.Contains(t, err.Error(), "data inconsistency",
			"Error must mention data inconsistency")
		assert.NotContains(t, fake.calls, "UpdateServiceEnvVars",
			"No env var update when guard fires with mixed keys")
	})
}

// =========================================================================
// Requirement: Error message content
// =========================================================================

// TestEnsureOrgInMint_ErrorContent validates that the guard error message
// contains actionable information.
func TestEnsureOrgInMint_ErrorContent(t *testing.T) {
	t.Run("[test_id:TS-GH-2433-010] error should contain role count and project ID", func(t *testing.T) {
		fake := newFakeGCFClient()
		fake.functionInfo = qfFunctionInfo()
		fake.trafficEnvVars = map[string]string{
			"ALLOWED_ORGS": "",
			"ROLE_APP_IDS": `{"agent":"app-1","reviewer":"app-2"}`,
		}

		p := qfProvisioner(fake)
		err := p.EnsureOrgInMint(context.Background(), qfTestMintURL, "new-org")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "2 configured roles",
			"Error must contain role count")
		assert.Contains(t, err.Error(), "proj1-test",
			"Error must contain project ID")
	})

	t.Run("[test_id:TS-GH-2433-011] error should contain suggested mint status command", func(t *testing.T) {
		fake := newFakeGCFClient()
		fake.functionInfo = qfFunctionInfo()
		fake.trafficEnvVars = map[string]string{
			"ALLOWED_ORGS": "",
			"ROLE_APP_IDS": `{"agent":"app-1","reviewer":"app-2"}`,
		}

		p := qfProvisioner(fake)
		err := p.EnsureOrgInMint(context.Background(), qfTestMintURL, "new-org")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "fullsend mint status --project=",
			"Error must contain suggested investigation command")
	})
}

// =========================================================================
// Requirement: ROLE_APP_IDS edge cases
// =========================================================================

// TestEnsureOrgInMint_RoleAppIDEdgeCases validates guard behavior on
// malformed, empty, or missing ROLE_APP_IDS.
func TestEnsureOrgInMint_RoleAppIDEdgeCases(t *testing.T) {
	t.Run("[test_id:TS-GH-2433-012] malformed ROLE_APP_IDS JSON should not trigger guard", func(t *testing.T) {
		fake := newFakeGCFClient()
		fake.functionInfo = qfFunctionInfo()
		fake.trafficEnvVars = map[string]string{
			"ALLOWED_ORGS": "",
			"ROLE_APP_IDS": "{not-valid-json}",
		}

		p := qfProvisioner(fake)
		err := p.EnsureOrgInMint(context.Background(), qfTestMintURL, "new-org")

		require.NoError(t, err, "Malformed JSON must not trigger guard")
		assert.Contains(t, fake.calls, "UpdateServiceEnvVars",
			"Enrollment must proceed despite malformed JSON")
	})

	t.Run("[test_id:TS-GH-2433-013] empty ROLE_APP_IDS should not trigger guard", func(t *testing.T) {
		fake := newFakeGCFClient()
		fake.functionInfo = qfFunctionInfo()
		fake.trafficEnvVars = map[string]string{
			"ALLOWED_ORGS": "",
			"ROLE_APP_IDS": "",
		}

		p := qfProvisioner(fake)
		err := p.EnsureOrgInMint(context.Background(), qfTestMintURL, "new-org")

		require.NoError(t, err, "Empty ROLE_APP_IDS must not trigger guard")
	})

	t.Run("[test_id:TS-GH-2433-013] missing ROLE_APP_IDS key should not trigger guard", func(t *testing.T) {
		fake := newFakeGCFClient()
		fake.functionInfo = qfFunctionInfo()
		fake.trafficEnvVars = map[string]string{
			"ALLOWED_ORGS": "",
			// ROLE_APP_IDS key absent
		}

		p := qfProvisioner(fake)
		err := p.EnsureOrgInMint(context.Background(), qfTestMintURL, "new-org")

		require.NoError(t, err, "Missing ROLE_APP_IDS key must not trigger guard")
	})

	t.Run("[test_id:TS-GH-2433-014] empty JSON object ROLE_APP_IDS should not trigger guard", func(t *testing.T) {
		fake := newFakeGCFClient()
		fake.functionInfo = qfFunctionInfo()
		fake.trafficEnvVars = map[string]string{
			"ALLOWED_ORGS": "",
			"ROLE_APP_IDS": "{}",
		}

		p := qfProvisioner(fake)
		err := p.EnsureOrgInMint(context.Background(), qfTestMintURL, "new-org")

		require.NoError(t, err, "Empty JSON object must not trigger guard")
		assert.Contains(t, fake.calls, "UpdateServiceEnvVars",
			"Enrollment must proceed with empty JSON object ROLE_APP_IDS")
	})
}

// =========================================================================
// Requirement: Error propagation through provisioning flows
// =========================================================================

// TestProvisionWithExistingMint_GuardPropagation verifies that the guard
// error propagates through provisionWithExistingMint.
func TestProvisionWithExistingMint_GuardPropagation(t *testing.T) {
	t.Run("[test_id:TS-GH-2433-015] provisionWithExistingMint should propagate guard error", func(t *testing.T) {
		fake := newFakeGCFClient()
		fake.functionInfo = qfFunctionInfo()
		fake.trafficEnvVars = map[string]string{
			"ALLOWED_ORGS": "",
			"ROLE_APP_IDS": `{"agent":"app-1"}`,
		}

		p := NewProvisioner(Config{
			ProjectID:   qfTestProjectID,
			Region:      qfTestRegion,
			MintURL:     qfTestMintURL,
			GitHubOrgs:  []string{"org1"},
			AgentPEMs:   map[string][]byte{"agent": []byte("pem-data")},
			AgentAppIDs: map[string]string{"agent": "app-1"},
		}, fake)

		_, err := p.provisionWithExistingMint(context.Background())

		require.Error(t, err, "provisionWithExistingMint must propagate guard error")
		assert.Contains(t, err.Error(), "data inconsistency",
			"Error must originate from data consistency guard")
	})
}

// TestProvisionSelfManaged_GuardPropagation verifies that the guard error
// propagates through provisionSelfManaged after infrastructure deployment.
func TestProvisionSelfManaged_GuardPropagation(t *testing.T) {
	t.Run("[test_id:TS-GH-2433-016] provisionSelfManaged should propagate guard error", func(t *testing.T) {
		fake := newFakeGCFClient()
		// No existing function — triggers full deploy path.
		fake.functionInfo = nil
		// After create, the function exists but has inconsistent state.
		fake.functionInfoAfterCreate = &FunctionInfo{
			Name:  "projects/" + qfTestProjectID + "/locations/" + qfTestRegion + "/functions/" + qfTestFnName,
			State: "ACTIVE",
			URI:   qfTestMintURL,
			EnvVars: map[string]string{
				"ALLOWED_ORGS": "",
				"ROLE_APP_IDS": `{"agent":"app-1"}`,
			},
		}
		fake.trafficEnvVars = map[string]string{
			"ALLOWED_ORGS": "",
			"ROLE_APP_IDS": `{"agent":"app-1"}`,
		}

		srcDir := fakeFunctionSourceDir(t)

		p := newTestProvisioner(Config{
			ProjectID:         qfTestProjectID,
			Region:            qfTestRegion,
			GitHubOrgs:        []string{"org1"},
			AgentPEMs:         map[string][]byte{"agent": []byte("pem-data")},
			AgentAppIDs:       map[string]string{"agent": "app-1"},
			FunctionSourceDir: srcDir,
		}, fake)

		_, err := p.provisionSelfManaged(context.Background())

		require.Error(t, err, "provisionSelfManaged must propagate guard error")
		assert.Contains(t, err.Error(), "data inconsistency",
			"Error must originate from data consistency guard")
	})
}

// =========================================================================
// Requirement: Concurrent enrollment isolation
// =========================================================================

// TestEnsureOrgInMint_ConcurrentIsolation validates that the guard
// fires independently per goroutine.
func TestEnsureOrgInMint_ConcurrentIsolation(t *testing.T) {
	t.Run("[test_id:TS-GH-2433-017] concurrent enrollments should isolate guard evaluation", func(t *testing.T) {
		// Stale client: empty ALLOWED_ORGS + role-only entries → guard fires
		staleFake := newFakeGCFClient()
		staleFake.functionInfo = qfFunctionInfo()
		staleFake.trafficEnvVars = map[string]string{
			"ALLOWED_ORGS": "",
			"ROLE_APP_IDS": `{"agent":"app-1"}`,
		}

		// Fresh client: populated ALLOWED_ORGS → guard bypassed
		freshFake := newFakeGCFClient()
		freshFake.functionInfo = qfFunctionInfo()
		freshFake.trafficEnvVars = map[string]string{
			"ALLOWED_ORGS": "existing-org",
			"ROLE_APP_IDS": `{"agent":"app-1"}`,
		}

		staleP := qfProvisioner(staleFake)
		freshP := qfProvisioner(freshFake)

		var (
			wg       sync.WaitGroup
			mu       sync.Mutex
			staleErr error
			freshErr error
		)

		wg.Add(2)
		go func() {
			defer wg.Done()
			err := staleP.EnsureOrgInMint(context.Background(), qfTestMintURL, "new-org")
			mu.Lock()
			staleErr = err
			mu.Unlock()
		}()
		go func() {
			defer wg.Done()
			err := freshP.EnsureOrgInMint(context.Background(), qfTestMintURL, "new-org")
			mu.Lock()
			freshErr = err
			mu.Unlock()
		}()
		wg.Wait()

		assert.Error(t, staleErr, "Stale-read goroutine must receive guard error")
		assert.NoError(t, freshErr, "Fresh goroutine must succeed")
	})

	t.Run("[test_id:TS-GH-2433-018] stale-read goroutine should fail while fresh succeeds", func(t *testing.T) {
		staleFake := newFakeGCFClient()
		staleFake.functionInfo = qfFunctionInfo()
		staleFake.trafficEnvVars = map[string]string{
			"ALLOWED_ORGS": "",
			"ROLE_APP_IDS": `{"agent":"app-1"}`,
		}

		freshFake := newFakeGCFClient()
		freshFake.functionInfo = qfFunctionInfo()
		freshFake.trafficEnvVars = map[string]string{
			"ALLOWED_ORGS": "existing-org",
			"ROLE_APP_IDS": `{"agent":"app-1"}`,
		}

		staleP := qfProvisioner(staleFake)
		freshP := qfProvisioner(freshFake)

		var (
			wg       sync.WaitGroup
			mu       sync.Mutex
			staleErr error
			freshErr error
		)

		wg.Add(2)
		go func() {
			defer wg.Done()
			err := staleP.EnsureOrgInMint(context.Background(), qfTestMintURL, "new-org")
			mu.Lock()
			staleErr = err
			mu.Unlock()
		}()
		go func() {
			defer wg.Done()
			err := freshP.EnsureOrgInMint(context.Background(), qfTestMintURL, "new-org")
			mu.Lock()
			freshErr = err
			mu.Unlock()
		}()
		wg.Wait()

		require.Error(t, staleErr)
		assert.Contains(t, staleErr.Error(), "data inconsistency",
			"Stale-read goroutine must report data inconsistency")
		assert.NoError(t, freshErr,
			"Fresh goroutine must succeed without cross-goroutine contamination")
	})
}
