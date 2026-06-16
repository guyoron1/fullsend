package gcf

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetProjectNumber_OmitsQuotaProjectHeader verifies that when
// GetProjectNumber is called on a LiveGCFClient whose embedded gcp.Client
// has a non-empty QuotaProject, the outgoing CRM API request does NOT
// include the x-goog-user-project header. The shallow copy with cleared
// QuotaProject prevents quota-project permission checks on the target
// project.
//
// Scenario: TS-GH-16-001 (P0, Unit, MVP)
func TestGetProjectNumber_OmitsQuotaProjectHeader(t *testing.T) {
	// SETUP: Create httptest server that captures request headers and
	// returns a valid project number.
	var capturedHeaders http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"projectNumber": "123456789"}`)
	}))
	defer srv.Close()

	// SETUP: Create LiveGCFClient with QuotaProject set, CRM endpoint
	// redirected to mock via rewriteTransport.
	client := newTestClient(srv)
	client.Client.QuotaProject = "test-quota-project"

	// TEST: Call GetProjectNumber.
	projectNum, err := client.GetProjectNumber(context.Background(), "test-project")

	// ASSERT: Call succeeds and returns expected project number.
	require.NoError(t, err)
	assert.Equal(t, "123456789", projectNum)

	// ASSERT: x-goog-user-project header is absent from CRM request.
	assert.Empty(t, capturedHeaders.Get("x-goog-user-project"),
		"CRM request must not include x-goog-user-project header")
}

// TestGetProjectNumber_ErrorOnHTTP403 verifies that GetProjectNumber returns
// a non-nil error when the CRM API responds with 403 Permission Denied.
//
// Scenario: TS-GH-16-002 (P1, Unit)
func TestGetProjectNumber_ErrorOnHTTP403(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":{"message":"Permission Denied"}}`, http.StatusForbidden)
	}))
	defer srv.Close()

	client := newTestClient(srv)

	_, err := client.GetProjectNumber(context.Background(), "test-project")

	require.Error(t, err, "GetProjectNumber should return error on 403")
}

// TestGetProjectNumber_HandlesEmptyProjectNumber verifies that
// GetProjectNumber handles the edge case of the CRM API returning an empty
// projectNumber field without panicking.
//
// Scenario: TS-GH-16-003 (P2, Unit)
func TestGetProjectNumber_HandlesEmptyProjectNumber(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"projectNumber": ""}`)
	}))
	defer srv.Close()

	client := newTestClient(srv)

	// The current implementation returns an error for empty project number.
	_, err := client.GetProjectNumber(context.Background(), "test-project")

	// ASSERT: No panic; returns error for empty projectNumber.
	require.Error(t, err, "GetProjectNumber should return error for empty projectNumber")
	assert.Contains(t, err.Error(), "empty project number")
}

// TestGetProjectNumber_DoesNotMutateOriginalQuotaProject verifies that the
// shallow copy mechanism in GetProjectNumber does not alter the original
// gcp.Client's QuotaProject field. After calling GetProjectNumber, the
// original client must retain its QuotaProject value.
//
// Scenario: TS-GH-16-004 (P0, Unit, MVP)
func TestGetProjectNumber_DoesNotMutateOriginalQuotaProject(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"projectNumber": "123456789"}`)
	}))
	defer srv.Close()

	client := newTestClient(srv)
	client.Client.QuotaProject = "test-quota-project"
	originalQuotaProject := client.Client.QuotaProject

	_, err := client.GetProjectNumber(context.Background(), "test-project")
	require.NoError(t, err)

	// ASSERT: Original QuotaProject is unchanged.
	assert.Equal(t, originalQuotaProject, client.Client.QuotaProject,
		"Original client QuotaProject must not be mutated by GetProjectNumber")
}

// TestSubsequentCallsRetainQuotaProject verifies that after calling
// GetProjectNumber (which internally clears QuotaProject on a copy),
// subsequent API calls through the original client still include the
// x-goog-user-project header with the correct QuotaProject value.
//
// Scenario: TS-GH-16-005 (P0, Unit, MVP)
func TestSubsequentCallsRetainQuotaProject(t *testing.T) {
	// Track headers for each request arriving at the server.
	var requestHeaders []http.Header

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestHeaders = append(requestHeaders, r.Header.Clone())
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"projectNumber": "123456789"}`)
	}))
	defer srv.Close()

	client := newTestClient(srv)
	client.Client.QuotaProject = "test-quota-project"
	ctx := context.Background()

	// CRM call (GetProjectNumber) — should omit x-goog-user-project.
	_, err := client.GetProjectNumber(ctx, "test-project")
	require.NoError(t, err)

	// Subsequent regular API call using the original client's DoRequest.
	resp, err := client.Client.DoRequest(ctx, http.MethodGet,
		"https://compute.googleapis.com/v1/some-other-api", "")
	require.NoError(t, err)
	resp.Body.Close()

	// ASSERT: At least two requests were made.
	require.GreaterOrEqual(t, len(requestHeaders), 2, "Expected at least 2 requests")

	// ASSERT: CRM request omitted x-goog-user-project.
	assert.Empty(t, requestHeaders[0].Get("x-goog-user-project"),
		"CRM request should not include x-goog-user-project header")

	// ASSERT: Subsequent request includes x-goog-user-project with original value.
	assert.Equal(t, "test-quota-project", requestHeaders[1].Get("x-goog-user-project"),
		"Subsequent API calls must retain x-goog-user-project header")
}

// TestGetProjectNumber_ErrorPropagationFromCopiedClient verifies that
// network-level errors are properly propagated through the shallow-copied
// client. When the CRM endpoint is unreachable, GetProjectNumber must
// return a meaningful error.
//
// Scenario: TS-GH-16-008 (P1, Unit)
func TestGetProjectNumber_ErrorPropagationFromCopiedClient(t *testing.T) {
	// Create and immediately close server to simulate unreachable endpoint.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	client := newTestClient(srv) // build client with server URL before closing
	srv.Close()                  // close server so all requests fail

	_, err := client.GetProjectNumber(context.Background(), "test-project")

	require.Error(t, err, "GetProjectNumber should return error on unreachable host")
}

// TestGetProjectNumber_403ErrorMessageIsDescriptive verifies that the error
// message from a 403 response contains enough diagnostic information for
// the user to identify and resolve the permission issue.
//
// Scenario: TS-GH-16-009 (P2, Unit)
func TestGetProjectNumber_403ErrorMessageIsDescriptive(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprintln(w, `{"error":{"message":"caller lacks cloudresourcemanager.projects.get"}}`)
	}))
	defer srv.Close()

	client := newTestClient(srv)

	_, err := client.GetProjectNumber(context.Background(), "test-project")
	require.Error(t, err)

	errMsg := strings.ToLower(err.Error())
	assert.True(t,
		strings.Contains(errMsg, "403") ||
			strings.Contains(errMsg, "permission") ||
			strings.Contains(errMsg, "forbidden"),
		"Error message should contain diagnostic info, got: %s", err.Error())
}
