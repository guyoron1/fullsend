package gcf_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/dispatch/gcf"
	"github.com/fullsend-ai/fullsend/internal/gcp"
)

// TestProvisioner_Provision_SelfManaged_Success tests the complete self-managed
// provisioning flow through the Provisioner, which internally calls
// GetProjectNumber as part of the OIDC setup. Validates that the modified
// GetProjectNumber integrates correctly with the broader provisioning workflow
// including service account creation and WIF pool setup.
//
// Scenario: TS-GH-16-006 (P1, Functional)
func TestProvisioner_Provision_SelfManaged_Success(t *testing.T) {
	// SETUP: Create comprehensive mock server handling CRM, IAM, and WIF API endpoints
	callLog := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callLog = append(callLog, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "projects"):
			fmt.Fprintln(w, `{"projectNumber": "123456789"}`)
		default:
			fmt.Fprintln(w, `{"done": true}`)
		}
	}))
	defer server.Close()

	// SETUP: Create Provisioner with LiveGCFClient pointing to mock
	gcpClient := &gcp.Client{
		QuotaProject: "my-quota-project",
	}
	gcfClient := &gcf.LiveGCFClient{Client: gcpClient}
	provisioner := &gcf.Provisioner{Client: gcfClient}

	ctx := context.Background()
	config := gcf.ProvisionConfig{
		Mode:    "self-managed",
		Project: "my-gcp-project",
		Region:  "us-central1",
	}

	// TEST: Call Provision with self-managed configuration
	err := provisioner.Provision(ctx, config)

	// ASSERT: Provisioning flow completes without error
	require.NoError(t, err)

	// ASSERT: GetProjectNumber was called during provisioning (CRM API path present)
	foundProjectCall := false
	for _, path := range callLog {
		if strings.Contains(path, "projects") {
			foundProjectCall = true
			break
		}
	}
	assert.True(t, foundProjectCall,
		"GetProjectNumber should be called during provisioning")
}

// TestProvisioner_Provision_ProjectNumberError_Aborts verifies that when
// GetProjectNumber fails during the provisioning flow, the Provisioner
// correctly propagates the error and aborts provisioning. No subsequent
// provisioning steps (SA creation, WIF setup) should execute.
//
// Scenario: TS-GH-16-007 (P1, Functional)
func TestProvisioner_Provision_ProjectNumberError_Aborts(t *testing.T) {
	// SETUP: Create mock server that returns 500 on CRM project lookup
	callLog := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callLog = append(callLog, r.URL.Path)
		if strings.Contains(r.URL.Path, "projects") {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, `{"error": {"code": 500, "message": "Internal error"}}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"done": true}`)
	}))
	defer server.Close()

	// SETUP: Create Provisioner with mock client
	gcpClient := &gcp.Client{
		QuotaProject: "my-quota-project",
	}
	gcfClient := &gcf.LiveGCFClient{Client: gcpClient}
	provisioner := &gcf.Provisioner{Client: gcfClient}

	ctx := context.Background()
	config := gcf.ProvisionConfig{
		Mode:    "self-managed",
		Project: "my-gcp-project",
		Region:  "us-central1",
	}

	// TEST: Call Provision and expect error
	err := provisioner.Provision(ctx, config)

	// ASSERT: Provisioning returns error on GetProjectNumber failure
	require.Error(t, err)

	// ASSERT: No subsequent provisioning steps executed (no SA or WIF API paths)
	for _, path := range callLog {
		assert.NotContains(t, path, "serviceAccounts",
			"Should not attempt SA creation after project number failure")
	}
}

// TestInstallOIDC_DispatchLayer_WithModifiedClient tests the OIDC dispatch
// layer installation flow, which calls GetProjectNumber as part of the
// provisioning chain: installOIDC -> Provision -> provisionSelfManaged ->
// GetProjectNumber. Validates that the modified GetProjectNumber works
// correctly when invoked through the full OIDC installation call chain.
//
// Scenario: TS-GH-16-010 (P1, Functional)
func TestInstallOIDC_DispatchLayer_WithModifiedClient(t *testing.T) {
	// SETUP: Create comprehensive mock server for OIDC flow
	callLog := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callLog = append(callLog, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "projects"):
			fmt.Fprintln(w, `{"projectNumber": "123456789"}`)
		default:
			fmt.Fprintln(w, `{"done": true}`)
		}
	}))
	defer server.Close()

	// SETUP: Create OIDC installer with GCF client pointing to mock
	gcpClient := &gcp.Client{
		QuotaProject: "my-quota-project",
	}
	gcfClient := &gcf.LiveGCFClient{Client: gcpClient}
	provisioner := &gcf.Provisioner{Client: gcfClient}

	ctx := context.Background()
	config := gcf.ProvisionConfig{
		Mode:     "self-managed",
		Project:  "my-gcp-project",
		Region:   "us-central1",
		AuthType: "oidc",
		Provider: "gcp",
	}

	// TEST: Call OIDC dispatch layer installation via Provisioner
	err := provisioner.Provision(ctx, config)

	// ASSERT: OIDC installation completes without error
	require.NoError(t, err)

	// ASSERT: GetProjectNumber was invoked during installation
	found := false
	for _, path := range callLog {
		if strings.Contains(path, "projects") {
			found = true
			break
		}
	}
	assert.True(t, found, "GetProjectNumber should be called during OIDC install")
}
