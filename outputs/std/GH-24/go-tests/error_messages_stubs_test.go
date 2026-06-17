package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Retry Exhaustion Error Message Tests

STP Reference: outputs/stp/GH-24/GH-24_test_plan.md
Jira: GH-24
*/

/*
Preconditions:
    - Mock server configured to always return 502

Steps:
    1. Call do() until retries are exhausted
    2. Inspect the error message

Expected:
    - Error message contains "retryable error after 3 attempts"
*/
func TestRetryExhaustionErrorContainsAttemptCount(t *testing.T) {
	t.Skip("Phase 1: Design only - awaiting implementation")

	// [test_id:TS-GH-24-014] Verify error contains "retryable error after 3 attempts"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.do(context.Background(), http.MethodGet, "/repos/org/repo/pulls/1", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "retryable error after 3 attempts")
}

/*
Preconditions:
    - Mock server configured to always return 503

Steps:
    1. Call do() with specific HTTP method (GET) and path
    2. Wait for retry exhaustion
    3. Inspect the error message

Expected:
    - Error message contains the HTTP method "GET"
    - Error message contains the request path "/repos/org/repo/pulls/1"
*/
func TestRetryExhaustionErrorContainsMethodAndPath(t *testing.T) {
	t.Skip("Phase 1: Design only - awaiting implementation")

	// [test_id:TS-GH-24-015] Verify error contains HTTP method and path
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.do(context.Background(), http.MethodGet, "/repos/org/repo/pulls/1", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "GET")
	assert.Contains(t, err.Error(), "/repos/org/repo/pulls/1")
}

/*
Preconditions:
    - Mock server configured to return 503 with Retry-After: 60 header

Steps:
    1. Call do() until retries are exhausted
    2. Inspect the error message for Retry-After information

Expected:
    - Error message includes Retry-After header information
*/
func TestRetryExhaustionErrorIncludesRetryAfter(t *testing.T) {
	t.Skip("Phase 1: Design only - awaiting implementation")

	// [test_id:TS-GH-24-016] Verify error includes Retry-After header value
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.do(context.Background(), http.MethodGet, "/repos/org/repo/pulls/1", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Retry-After")
}
