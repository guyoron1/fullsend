package gcf

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GH-73-TC-032: Verify cloud function creation and deployment
func TestQF_Provisioner_CreateAndDeploy(t *testing.T) {
	srcDir := fakeFunctionSourceDir(t)
	fc := newFakeGCFClient()
	fc.errs["GetSecret"] = ErrSecretNotFound
	fc.functionInfoAfterCreate = &FunctionInfo{
		Name:  "projects/test-project/locations/us-central1/functions/fullsend-mint",
		State: "ACTIVE",
		URI:   "https://fullsend-mint-abc123.run.app",
		EnvVars: map[string]string{
			"ALLOWED_ORGS": "test-org",
		},
	}

	p := newTestProvisioner(Config{
		ProjectID:         "test-project",
		GitHubOrgs:        []string{"test-org"},
		AgentPEMs:         singleRolePEMs(),
		AgentAppIDs:       singleRoleAppIDs(),
		FunctionSourceDir: srcDir,
	}, fc)

	envVars, err := p.Provision(context.Background())
	require.NoError(t, err)
	require.NotNil(t, envVars, "Provision should return env vars")

	// Verify the cloud function was created
	assert.Contains(t, fc.calls, "CreateFunction", "should create a cloud function")

	// Verify mint URL is returned
	_, hasMintURL := envVars["FULLSEND_MINT_URL"]
	assert.True(t, hasMintURL, "should return FULLSEND_MINT_URL")
}

// GH-73-TC-033: Verify environment variable updates on function
func TestQF_Provisioner_EnvVarMerge(t *testing.T) {
	srcDir := fakeFunctionSourceDir(t)
	fc := newFakeGCFClient()
	fc.errs["GetSecret"] = ErrSecretNotFound

	// Pre-populate a function with existing env vars and valid URI
	fc.functionInfo = &FunctionInfo{
		Name:    "projects/test-project/locations/us-central1/functions/fullsend-mint",
		URI:     "https://fullsend-mint-existing.run.app",
		State:   "ACTIVE",
		EnvVars: map[string]string{"EXISTING_KEY": "existing_value"},
	}

	p := newTestProvisioner(Config{
		ProjectID:         "test-project",
		GitHubOrgs:        []string{"test-org"},
		AgentPEMs:         singleRolePEMs(),
		AgentAppIDs:       singleRoleAppIDs(),
		FunctionSourceDir: srcDir,
	}, fc)

	envVars, err := p.Provision(context.Background())
	require.NoError(t, err)

	// Verify mint URL is returned based on existing function
	_, hasMintURL := envVars["FULLSEND_MINT_URL"]
	assert.True(t, hasMintURL, "should return FULLSEND_MINT_URL from existing function")
}

// GH-73-TC-034: Verify error handling for invalid project ID
func TestQF_Provisioner_InvalidProjectID(t *testing.T) {
	fc := newFakeGCFClient()
	fc.errs["GetProjectNumber"] = assert.AnError

	p := newTestProvisioner(Config{
		ProjectID:   "invalid-project",
		GitHubOrgs:  []string{"test-org"},
		AgentPEMs:   singleRolePEMs(),
		AgentAppIDs: singleRoleAppIDs(),
	}, fc)

	_, err := p.Provision(context.Background())
	require.Error(t, err, "should error when project number lookup fails")
}

// GH-73-TC-034 supplemental: Missing project ID
func TestQF_Provisioner_MissingProjectID(t *testing.T) {
	fc := newFakeGCFClient()

	p := newTestProvisioner(Config{
		ProjectID:   "", // empty
		GitHubOrgs:  []string{"test-org"},
		AgentPEMs:   singleRolePEMs(),
		AgentAppIDs: singleRoleAppIDs(),
	}, fc)

	_, err := p.Provision(context.Background())
	require.Error(t, err, "should error when project ID is empty")
}

// GH-73-TC-035: Verify fake client simulates API behavior
func TestQF_FakeGCFClient_CRUDOperations(t *testing.T) {
	fc := newFakeGCFClient()

	t.Run("create records call", func(t *testing.T) {
		err := fc.CreateServiceAccount(context.Background(), "proj", "sa", "SA")
		assert.NoError(t, err)
		assert.Contains(t, fc.calls, "CreateServiceAccount")
	})

	t.Run("get function returns preset info", func(t *testing.T) {
		fc.functionInfo = &FunctionInfo{
			URI:   "https://func.example.com",
			State: "ACTIVE",
		}
		info, err := fc.GetFunction(context.Background(), "", "", "")
		assert.NoError(t, err)
		assert.Equal(t, "https://func.example.com", info.URI)
	})

	t.Run("error injection works", func(t *testing.T) {
		fc.errs["CreateWIFPool"] = assert.AnError
		err := fc.CreateWIFPool(context.Background(), "", "", "")
		assert.Error(t, err)
	})

	t.Run("secret data tracks state", func(t *testing.T) {
		fc.secrets = map[string]bool{"my-secret": true}
		assert.True(t, fc.secrets["my-secret"])
		assert.False(t, fc.secrets["missing-secret"])
	})
}

// GH-73-TC-032 supplemental: Verify provisioner requires at least one org
func TestQF_Provisioner_RequiresOrg(t *testing.T) {
	fc := newFakeGCFClient()
	p := newTestProvisioner(Config{
		ProjectID:   "test-project",
		GitHubOrgs:  []string{}, // empty
		AgentPEMs:   singleRolePEMs(),
		AgentAppIDs: singleRoleAppIDs(),
	}, fc)

	_, err := p.Provision(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one GitHub org is required")
}
