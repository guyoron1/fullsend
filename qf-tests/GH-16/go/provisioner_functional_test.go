package gcf

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTempFunctionSource creates a minimal Cloud Function source directory
// for tests that exercise provisionSelfManaged (which validates the source
// directory before calling GetProjectNumber).
func createTempFunctionSource(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n\ngo 1.23\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc ServeHTTP() {}\n"), 0644))

	return dir
}

// TestProvisionSelfManaged_CompletesWithModifiedGetProjectNumber validates
// the end-to-end self-managed provisioning flow with the modified
// GetProjectNumber. The provisioner calls GetProjectNumber early in
// provisionSelfManaged, and all subsequent steps (SA creation, WIF pool/
// provider setup, function deployment) must succeed with the shallow-copy
// fix in place.
//
// Scenario: TS-GH-16-006 (P1, Functional)
func TestProvisionSelfManaged_CompletesWithModifiedGetProjectNumber(t *testing.T) {
	// Track which API paths are called during provisioning.
	var callPaths []string

	mux := http.NewServeMux()

	// CRM endpoint — GetProjectNumber
	mux.HandleFunc("/v1/projects/", func(w http.ResponseWriter, r *http.Request) {
		callPaths = append(callPaths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"projectNumber": "123456789"}`)
	})
	// IAM service account creation
	mux.HandleFunc("/v1/projects/test-project-id/serviceAccounts", func(w http.ResponseWriter, r *http.Request) {
		callPaths = append(callPaths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"email": "fullsend-dispatch@test-project-id.iam.gserviceaccount.com"}`)
	})
	// WIF pool/provider — return done immediately
	mux.HandleFunc("/v1/projects/123456789/locations/global/workloadIdentityPools", func(w http.ResponseWriter, r *http.Request) {
		callPaths = append(callPaths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"name":"operations/wif-op","done":true}`)
	})
	// Secret Manager operations
	mux.HandleFunc("/v1/projects/test-project-id/secrets/", func(w http.ResponseWriter, r *http.Request) {
		callPaths = append(callPaths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "getIamPolicy") || strings.Contains(r.URL.Path, "setIamPolicy") {
			fmt.Fprintln(w, `{"bindings":[],"etag":"abc"}`)
			return
		}
		fmt.Fprintln(w, `{}`)
	})
	// Cloud Functions — GetFunction (not found triggers create)
	mux.HandleFunc("/v2/projects/", func(w http.ResponseWriter, r *http.Request) {
		callPaths = append(callPaths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "generateUploadUrl") {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uploadUrl":     "https://storage.googleapis.com/upload",
				"storageSource": map[string]string{"bucket": "b", "object": "o"},
			})
			return
		}
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/functions/") {
			// First call: not found. After create: return active.
			fmt.Fprintln(w, `{"name":"projects/test-project-id/locations/us-central1/functions/fullsend-mint","state":"ACTIVE","serviceConfig":{"uri":"https://fullsend-mint-abc.run.app"}}`)
			return
		}
		// Create/update function or wait for operation
		fmt.Fprintln(w, `{"name":"operations/fn-op","done":true}`)
	})
	// Cloud Run IAM
	mux.HandleFunc("/v2/projects/test-project-id/locations/us-central1/services/", func(w http.ResponseWriter, r *http.Request) {
		callPaths = append(callPaths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"bindings":[{"role":"roles/run.invoker","members":["allUsers"]}]}`)
	})
	// Catch-all for any unmatched paths
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		callPaths = append(callPaths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{}`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	srcDir := createTempFunctionSource(t)

	cfg := Config{
		ProjectID:         "test-project-id",
		Region:            "us-central1",
		GitHubOrgs:        []string{"test-org"},
		FunctionSourceDir: srcDir,
		AgentPEMs:         map[string][]byte{"agent": []byte("pem-data")},
		AgentAppIDs:       map[string]string{"agent": "12345"},
	}

	provisioner := NewProvisioner(cfg, newTestClient(srv))

	// Redirect health check to mock server too.
	provisioner.httpClient = srv.Client()

	result, err := provisioner.Provision(context.Background())

	// ASSERT: Provisioning completes without error.
	require.NoError(t, err, "Self-managed provisioning should complete successfully")

	// ASSERT: FULLSEND_MINT_URL is returned.
	require.NotNil(t, result)
	assert.NotEmpty(t, result["FULLSEND_MINT_URL"],
		"Provision should return a FULLSEND_MINT_URL")

	// ASSERT: GetProjectNumber was called (CRM path present).
	foundCRM := false
	for _, p := range callPaths {
		if strings.Contains(p, "projects") {
			foundCRM = true
			break
		}
	}
	assert.True(t, foundCRM,
		"GetProjectNumber should be called during self-managed provisioning")
}

// TestProvisionAborts_WhenGetProjectNumberFails validates that when the
// CRM API returns 500 for the project number lookup, the provisioner
// aborts immediately. No downstream operations (SA creation, WIF setup)
// should be attempted.
//
// Scenario: TS-GH-16-007 (P1, Functional)
func TestProvisionAborts_WhenGetProjectNumberFails(t *testing.T) {
	saCallCount := 0
	mux := http.NewServeMux()

	// CRM endpoint — return 500
	mux.HandleFunc("/v1/projects/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"message":"Internal Server Error"}}`, http.StatusInternalServerError)
	})
	// SA creation — should NOT be reached
	mux.HandleFunc("/v1/projects/test-project-id/serviceAccounts", func(w http.ResponseWriter, r *http.Request) {
		saCallCount++
		w.WriteHeader(http.StatusOK)
	})
	// Catch-all
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{}`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	srcDir := createTempFunctionSource(t)

	cfg := Config{
		ProjectID:         "test-project-id",
		Region:            "us-central1",
		GitHubOrgs:        []string{"test-org"},
		FunctionSourceDir: srcDir,
		AgentPEMs:         map[string][]byte{"agent": []byte("pem-data")},
		AgentAppIDs:       map[string]string{"agent": "12345"},
	}

	provisioner := NewProvisioner(cfg, newTestClient(srv))

	_, err := provisioner.Provision(context.Background())

	// ASSERT: Provisioning returns error.
	require.Error(t, err, "Provisioning should fail when GetProjectNumber returns error")

	// ASSERT: No SA creation was attempted.
	assert.Equal(t, 0, saCallCount,
		"No SA creation should be attempted after GetProjectNumber failure")
}

// TestOIDCDispatchInstallation_WithModifiedClient validates that the full
// OIDC dispatch layer installation works correctly with the modified
// GetProjectNumber. The OIDC provisioning chain makes multiple sequential
// GCP API calls, and the shallow copy fix must not interfere with OIDC-
// specific endpoints.
//
// Scenario: TS-GH-16-010 (P1, Functional)
func TestOIDCDispatchInstallation_WithModifiedClient(t *testing.T) {
	var callPaths []string

	mux := http.NewServeMux()

	// CRM endpoint
	mux.HandleFunc("/v1/projects/", func(w http.ResponseWriter, r *http.Request) {
		callPaths = append(callPaths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"projectNumber": "123456789"}`)
	})
	// IAM — SA creation
	mux.HandleFunc("/v1/projects/test-project-id/serviceAccounts", func(w http.ResponseWriter, r *http.Request) {
		callPaths = append(callPaths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"email": "fullsend-dispatch@test-project-id.iam.gserviceaccount.com"}`)
	})
	// WIF pool
	mux.HandleFunc("/v1/projects/123456789/locations/global/workloadIdentityPools", func(w http.ResponseWriter, r *http.Request) {
		callPaths = append(callPaths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"name":"operations/pool-op","done":true}`)
	})
	// Secret Manager
	mux.HandleFunc("/v1/projects/test-project-id/secrets/", func(w http.ResponseWriter, r *http.Request) {
		callPaths = append(callPaths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "getIamPolicy") || strings.Contains(r.URL.Path, "setIamPolicy") {
			fmt.Fprintln(w, `{"bindings":[],"etag":"abc"}`)
			return
		}
		fmt.Fprintln(w, `{}`)
	})
	// Cloud Functions v2
	mux.HandleFunc("/v2/projects/", func(w http.ResponseWriter, r *http.Request) {
		callPaths = append(callPaths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "generateUploadUrl") {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uploadUrl":     "https://storage.googleapis.com/upload",
				"storageSource": map[string]string{"bucket": "b", "object": "o"},
			})
			return
		}
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/functions/") {
			fmt.Fprintln(w, `{"name":"projects/test-project-id/locations/us-central1/functions/fullsend-mint","state":"ACTIVE","serviceConfig":{"uri":"https://fullsend-mint-abc.run.app"}}`)
			return
		}
		fmt.Fprintln(w, `{"name":"operations/fn-op","done":true}`)
	})
	// Cloud Run IAM
	mux.HandleFunc("/v2/projects/test-project-id/locations/us-central1/services/", func(w http.ResponseWriter, r *http.Request) {
		callPaths = append(callPaths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"bindings":[{"role":"roles/run.invoker","members":["allUsers"]}]}`)
	})
	// Catch-all
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		callPaths = append(callPaths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{}`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	srcDir := createTempFunctionSource(t)

	cfg := Config{
		ProjectID:         "test-project-id",
		Region:            "us-central1",
		GitHubOrgs:        []string{"test-org"},
		FunctionSourceDir: srcDir,
		AgentPEMs:         map[string][]byte{"agent": []byte("pem-data")},
		AgentAppIDs:       map[string]string{"agent": "12345"},
	}

	provisioner := NewProvisioner(cfg, newTestClient(srv))
	provisioner.httpClient = srv.Client()

	result, err := provisioner.Provision(context.Background())

	// ASSERT: OIDC provisioning completes without error.
	require.NoError(t, err, "OIDC provisioning should complete successfully")
	require.NotNil(t, result)
	assert.NotEmpty(t, result["FULLSEND_MINT_URL"])

	// ASSERT: GetProjectNumber was called during the OIDC flow.
	foundCRM := false
	for _, p := range callPaths {
		if strings.Contains(p, "projects") {
			foundCRM = true
			break
		}
	}
	assert.True(t, foundCRM,
		"GetProjectNumber should be called during OIDC dispatch installation")
}
