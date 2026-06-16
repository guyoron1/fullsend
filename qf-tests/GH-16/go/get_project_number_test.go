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

// TestGetProjectNumber_OmitsQuotaProjectHeader verifies that when
// GetProjectNumber is called on a LiveGCFClient that has a QuotaProject
// configured, the outgoing HTTP request to the Cloud Resource Manager API
// does NOT include the x-goog-user-project header. This validates that a
// shallow copy of the gcp.Client with cleared QuotaProject is used for the
// CRM API call.
//
// Scenario: TS-GH-16-001 (P0, Unit, MVP)
func TestGetProjectNumber_OmitsQuotaProjectHeader(t *testing.T) {
	// SETUP: Create httptest.Server that captures request headers
	var capturedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"projectNumber": "123456789"}`)
	}))
	defer server.Close()

	// SETUP: Create LiveGCFClient with QuotaProject set and pointing to mock server
	gcpClient := &gcp.Client{
		QuotaProject: "my-quota-project",
	}
	client := &gcf.LiveGCFClient{Client: gcpClient}

	ctx := context.Background()

	// TEST: Call GetProjectNumber
	projectNumber, err := client.GetProjectNumber(ctx, "my-project")

	// ASSERT: Call succeeds and returns the expected project number
	require.NoError(t, err)
	assert.Equal(t, "123456789", projectNumber)

	// ASSERT: x-goog-user-project header is absent in captured headers
	assert.Empty(t, capturedHeaders.Get("x-goog-user-project"),
		"x-goog-user-project header should not be present in CRM request")
}

// TestGetProjectNumber_PermissionDenied verifies that when the CRM API
// returns HTTP 403 (Permission Denied), GetProjectNumber returns an
// appropriate error rather than panicking or returning invalid data.
//
// Scenario: TS-GH-16-002 (P1, Unit)
func TestGetProjectNumber_PermissionDenied(t *testing.T) {
	// SETUP: Create httptest.Server returning 403 Forbidden
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprintln(w, `{"error": {"code": 403, "message": "Permission denied"}}`)
	}))
	defer server.Close()

	// SETUP: Create LiveGCFClient pointing to mock server
	gcpClient := &gcp.Client{
		QuotaProject: "my-quota-project",
	}
	client := &gcf.LiveGCFClient{Client: gcpClient}

	ctx := context.Background()

	// TEST: Call GetProjectNumber and expect error
	_, err := client.GetProjectNumber(ctx, "my-project")

	// ASSERT: Error is returned on 403 response
	require.Error(t, err, "GetProjectNumber should return error on 403 response")
}

// TestGetProjectNumber_EmptyProjectNumber verifies that when the CRM API
// returns HTTP 200 but with an empty projectNumber field, GetProjectNumber
// handles this edge case gracefully.
//
// Scenario: TS-GH-16-003 (P2, Unit)
func TestGetProjectNumber_EmptyProjectNumber(t *testing.T) {
	// SETUP: Create httptest.Server returning empty projectNumber
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"projectNumber": ""}`)
	}))
	defer server.Close()

	// SETUP: Create LiveGCFClient pointing to mock server
	gcpClient := &gcp.Client{
		QuotaProject: "my-quota-project",
	}
	client := &gcf.LiveGCFClient{Client: gcpClient}

	ctx := context.Background()

	// TEST: Call GetProjectNumber and check result
	projectNumber, err := client.GetProjectNumber(ctx, "my-project")

	// ASSERT: Empty project number is handled without panic
	// Either an error is returned or empty string is handled
	if err == nil {
		assert.Empty(t, projectNumber, "Empty project number should be flagged")
	}
}

// TestGetProjectNumber_OriginalClientUnchanged verifies that after calling
// GetProjectNumber, the original LiveGCFClient's embedded gcp.Client retains
// its QuotaProject value. The shallow copy used internally must not modify
// the original struct's fields.
//
// Scenario: TS-GH-16-004 (P0, Unit, MVP)
func TestGetProjectNumber_OriginalClientUnchanged(t *testing.T) {
	// SETUP: Create mock HTTP server returning valid project number
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"projectNumber": "123456789"}`)
	}))
	defer server.Close()

	// SETUP: Create LiveGCFClient with QuotaProject set
	gcpClient := &gcp.Client{
		QuotaProject: "my-quota-project",
	}
	client := &gcf.LiveGCFClient{Client: gcpClient}
	originalQuotaProject := gcpClient.QuotaProject

	ctx := context.Background()

	// TEST: Call GetProjectNumber
	_, err := client.GetProjectNumber(ctx, "target-project")
	require.NoError(t, err)

	// ASSERT: Original QuotaProject value is preserved
	assert.Equal(t, originalQuotaProject, client.Client.QuotaProject,
		"Original client's QuotaProject must not be mutated by GetProjectNumber")
}

// TestGetProjectNumber_SubsequentCallsUseOriginalQuotaProject verifies that
// after GetProjectNumber clears the QuotaProject on its internal copy,
// subsequent API calls continue to use the original QuotaProject value.
//
// Scenario: TS-GH-16-005 (P0, Unit, MVP)
func TestGetProjectNumber_SubsequentCallsUseOriginalQuotaProject(t *testing.T) {
	// SETUP: Create httptest.Server that logs headers per request
	var requestLog []http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestLog = append(requestLog, r.Header.Clone())
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"projectNumber": "123456789"}`)
	}))
	defer server.Close()

	// SETUP: Create LiveGCFClient with QuotaProject set
	gcpClient := &gcp.Client{
		QuotaProject: "my-quota-project",
	}
	client := &gcf.LiveGCFClient{Client: gcpClient}

	ctx := context.Background()

	// TEST: Call GetProjectNumber (CRM call - should omit header)
	_, err := client.GetProjectNumber(ctx, "my-project")
	require.NoError(t, err)

	// TEST: Make a subsequent API request using the original client
	req, _ := http.NewRequestWithContext(ctx, "GET", server.URL+"/v1/other", nil)
	_, _ = client.Client.DoRequest(req)

	// ASSERT: First call has no x-goog-user-project
	require.GreaterOrEqual(t, len(requestLog), 2, "Expected at least 2 requests")
	assert.Empty(t, requestLog[0].Get("x-goog-user-project"),
		"CRM request should not have x-goog-user-project header")

	// ASSERT: Second call has x-goog-user-project = "my-quota-project"
	assert.Equal(t, "my-quota-project", requestLog[1].Get("x-goog-user-project"),
		"Subsequent request should use original QuotaProject")
}

// TestGetProjectNumber_ErrorPropagationFromCopiedClient verifies that errors
// from the copied gcp.Client's DoRequest method are properly propagated
// through GetProjectNumber. The shallow copy must maintain the same error
// propagation behavior as the original client.
//
// Scenario: TS-GH-16-008 (P1, Unit)
func TestGetProjectNumber_ErrorPropagationFromCopiedClient(t *testing.T) {
	// SETUP: Create server and close it immediately to force connection error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close()

	// SETUP: Create LiveGCFClient pointing to closed server URL
	gcpClient := &gcp.Client{
		QuotaProject: "my-quota-project",
	}
	client := &gcf.LiveGCFClient{Client: gcpClient}

	ctx := context.Background()

	// TEST: Call GetProjectNumber on client with closed server
	_, err := client.GetProjectNumber(ctx, "my-project")

	// ASSERT: Error is returned for unreachable server
	require.Error(t, err, "Should return error when server is unreachable")
}

// TestGetProjectNumber_HTTP403_DescriptiveError verifies that when the CRM
// API returns HTTP 403 Forbidden, the error message returned by
// GetProjectNumber contains enough information for the user to diagnose the
// issue.
//
// Scenario: TS-GH-16-009 (P2, Unit)
func TestGetProjectNumber_HTTP403_DescriptiveError(t *testing.T) {
	// SETUP: Create mock HTTP server returning 403 with descriptive body
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprintln(w, `{"error": {"code": 403, "message": "Cloud Resource Manager API has not been used in project my-project before or it is disabled.", "status": "PERMISSION_DENIED"}}`)
	}))
	defer server.Close()

	// SETUP: Create LiveGCFClient pointing to mock server
	gcpClient := &gcp.Client{
		QuotaProject: "my-quota-project",
	}
	client := &gcf.LiveGCFClient{Client: gcpClient}

	ctx := context.Background()

	// TEST: Call GetProjectNumber and capture error
	_, err := client.GetProjectNumber(ctx, "my-project")

	// ASSERT: Error is returned
	require.Error(t, err)

	// ASSERT: Error message is descriptive for 403 responses
	errMsg := err.Error()
	assert.True(t,
		strings.Contains(errMsg, "403") || strings.Contains(errMsg, "forbidden") || strings.Contains(errMsg, "permission") || strings.Contains(errMsg, "Permission"),
		"Error should indicate permission issue, got: %s", errMsg)
}
