//go:build e2e

package gcf

// GH-11 End-to-End Tests: Remove Quota Project from GCP Project Number Lookup
//
// These tests validate the behavioral change in GetProjectNumber where the
// x-goog-user-project header is omitted to avoid requiring the Cloud Resource
// Manager API to be enabled on the target project.
//
// STD Reference: outputs/std/GH-11/GH-11_test_description.yaml
// STP Reference: outputs/stp/GH-11/GH-11_test_plan.md
//
// Compilation prerequisite: gcp.Client must include a QuotaProject string
// field. See STP Known Limitations (I.2) and Entry Criteria (II.4).

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TS-GH-11-001: GetProjectNumber succeeds without quota project header.
//
// Validates the core behavioral change of GH-11: the CRM API request must
// omit the x-goog-user-project header so customers don't need to enable the
// Cloud Resource Manager API on their target project.
func TestGetProjectNumber_SuccessWithoutQuotaHeader(t *testing.T) {
	var capturedHeaders http.Header

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"projectNumber": "123456789"})
	}))
	defer srv.Close()

	client := newTestClient(srv)

	projectNumber, err := client.GetProjectNumber(context.Background(), "my-project")
	require.NoError(t, err, "GetProjectNumber should succeed for a valid project ID")

	assert.Equal(t, "123456789", projectNumber,
		"GetProjectNumber should return the correct project number from the API response")

	assert.Empty(t, capturedHeaders.Get("x-goog-user-project"),
		"Request to CRM API must not include x-goog-user-project header — "+
			"this is the core fix of GH-11")
}

// TS-GH-11-002: Original gcp.Client is not mutated after GetProjectNumber call.
//
// Verifies that the value copy of *c.Client isolates the QuotaProject
// mutation. If the original client is mutated, subsequent API calls from
// the same client could behave unexpectedly.
func TestGetProjectNumber_OriginalClientNotMutated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"projectNumber": "123456789"})
	}))
	defer srv.Close()

	client := newTestClient(srv)
	// Set a known QuotaProject value before the call.
	client.Client.QuotaProject = "original-project-id"

	_, err := client.GetProjectNumber(context.Background(), "test-project")
	require.NoError(t, err, "GetProjectNumber should complete without error")

	assert.Equal(t, "original-project-id", client.Client.QuotaProject,
		"GetProjectNumber must not mutate the original client's QuotaProject field — "+
			"it should create a value copy with QuotaProject cleared")
}

// TS-GH-11-003: GetProjectNumber returns error for HTTP 403 Forbidden.
//
// Validates that permission errors from the CRM API are surfaced clearly
// to help operators diagnose IAM issues during FullSend deployment.
func TestGetProjectNumber_Forbidden(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprintln(w, `{"error":{"message":"denied"}}`)
	}))
	defer srv.Close()

	client := newTestClient(srv)

	_, err := client.GetProjectNumber(context.Background(), "proj")
	require.Error(t, err, "GetProjectNumber should return an error for a 403 response")

	assert.Contains(t, err.Error(), "unexpected status 403",
		"Error message should include the HTTP status code for diagnostics")
}

// TS-GH-11-004: GetProjectNumber returns error for empty project number response.
//
// Validates that an empty projectNumber in a 200 response is detected
// and reported, rather than propagating an empty string downstream.
func TestGetProjectNumber_EmptyProjectNumber(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"projectNumber": ""})
	}))
	defer srv.Close()

	client := newTestClient(srv)

	_, err := client.GetProjectNumber(context.Background(), "proj")
	require.Error(t, err, "GetProjectNumber should return an error when projectNumber is empty")

	assert.Contains(t, err.Error(), "empty project number",
		"Error message should describe the empty project number condition")
}

// TS-GH-11-005: provisionSelfManaged workflow succeeds with modified GetProjectNumber.
//
// Validates the integration between the modified GetProjectNumber and its
// caller (provisionSelfManaged). The project number returned by GetProjectNumber
// is used for subsequent WIF pool/provider operations.
func TestProvisionSelfManaged_SuccessWithProjectNumber(t *testing.T) {
	fake := newFakeGCFClient()
	fake.projectNumber = "123456789"
	// Configure GetFunction to return nil first (not found), then active after deploy.
	fake.functionInfoAfterCreate = &FunctionInfo{
		State:  "ACTIVE",
		URI:    "https://my-func-abc.run.app",
		Region: "us-central1",
	}

	srcDir := fakeFunctionSourceDir(t)

	p := newTestProvisioner(Config{
		ProjectID:         "test-project",
		Region:            "us-central1",
		GitHubOrgs:        []string{"my-org"},
		AgentPEMs:         singleRolePEMs(),
		AgentAppIDs:       singleRoleAppIDs(),
		FunctionSourceDir: srcDir,
	}, fake)

	result, err := p.provisionSelfManaged(context.Background())
	require.NoError(t, err, "provisionSelfManaged should succeed when GetProjectNumber returns a valid number")

	assert.Contains(t, result, "FULLSEND_MINT_URL",
		"Result map should contain FULLSEND_MINT_URL key")
	assert.NotEmpty(t, result["FULLSEND_MINT_URL"],
		"FULLSEND_MINT_URL should not be empty")

	// Verify GetProjectNumber was called.
	assert.Contains(t, fake.calls, "GetProjectNumber",
		"GetProjectNumber should be called during provisioning")
}

// TS-GH-11-006: provisionSelfManaged fails gracefully when GetProjectNumber errors.
//
// Validates fail-fast behavior: if the project number cannot be obtained,
// no downstream GCP operations (WIF pool creation, function deployment)
// should be attempted.
func TestProvisionSelfManaged_FailsOnProjectNumberError(t *testing.T) {
	fake := newFakeGCFClient()
	fake.errs["GetProjectNumber"] = fmt.Errorf("permission denied")

	srcDir := fakeFunctionSourceDir(t)

	p := newTestProvisioner(Config{
		ProjectID:         "test-project",
		Region:            "us-central1",
		GitHubOrgs:        []string{"my-org"},
		AgentPEMs:         singleRolePEMs(),
		AgentAppIDs:       singleRoleAppIDs(),
		FunctionSourceDir: srcDir,
	}, fake)

	_, err := p.provisionSelfManaged(context.Background())
	require.Error(t, err, "provisionSelfManaged should fail when GetProjectNumber returns an error")

	// Verify no downstream operations were attempted after the failure.
	for _, call := range fake.calls {
		switch call {
		case "GetProjectNumber":
			// Expected — this is the call that failed.
		default:
			t.Errorf("Unexpected downstream call %q after GetProjectNumber failure — "+
				"fail-fast behavior violated", call)
		}
	}
}

// TS-GH-11-007: Client value copy does not share mutable QuotaProject state.
//
// Validates that concurrent GetProjectNumber calls from the same client
// create independent copies and do not interfere with each other. Run with
// `go test -race` to detect data races in the copy mechanism.
func TestGetProjectNumber_ConcurrentClientIsolation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"projectNumber": "123456789"})
	}))
	defer srv.Close()

	client := newTestClient(srv)
	client.Client.QuotaProject = "shared-project"

	const goroutines = 10
	var wg sync.WaitGroup
	errs := make([]error, goroutines)

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			_, errs[idx] = client.GetProjectNumber(context.Background(), fmt.Sprintf("project-%d", idx))
		}(i)
	}
	wg.Wait()

	// All goroutines should complete without error.
	for i, err := range errs {
		assert.NoError(t, err, "Goroutine %d should complete without error", i)
	}

	// Original client state must be preserved after all concurrent calls.
	assert.Equal(t, "shared-project", client.Client.QuotaProject,
		"Original client QuotaProject must not be mutated by concurrent GetProjectNumber calls")
}
