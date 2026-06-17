package github_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Non-Retryable HTTP Error Tests

STP Reference: outputs/stp/GH-24/GH-24_test_plan.md
Jira: GH-24
*/

/*
Preconditions:
    - Mock server configured to return 400

Steps:
    1. Call do() with GET request
    2. Observe call count and error

Expected:
    - do() returns error immediately
    - Exactly 1 HTTP call is made (no retry)
*/
func TestNonRetryable400ReturnsImmediately(t *testing.T) {
	t.Skip("Phase 1: Design only - awaiting implementation")

	// [test_id:TS-GH-24-017] Verify 400 Bad Request returns immediately
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.do(context.Background(), http.MethodGet, "/repos/org/repo/pulls/1", nil)
	require.Error(t, err)
	assert.Equal(t, 1, callCount, "expected no retry for 400")
}

/*
Preconditions:
    - Mock server configured to return 401

Steps:
    1. Call do() with GET request
    2. Observe call count and error

Expected:
    - do() returns error immediately
    - Exactly 1 HTTP call is made (no retry)
*/
func TestNonRetryable401ReturnsImmediately(t *testing.T) {
	t.Skip("Phase 1: Design only - awaiting implementation")

	// [test_id:TS-GH-24-018] Verify 401 Unauthorized returns immediately
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.do(context.Background(), http.MethodGet, "/repos/org/repo/pulls/1", nil)
	require.Error(t, err)
	assert.Equal(t, 1, callCount, "expected no retry for 401")
}

/*
Preconditions:
    - Mock server configured to return 422

Steps:
    1. Call do() with GET request
    2. Observe call count and error

Expected:
    - do() returns error immediately
    - Exactly 1 HTTP call is made (no retry)
*/
func TestNonRetryable422ReturnsImmediately(t *testing.T) {
	t.Skip("Phase 1: Design only - awaiting implementation")

	// [test_id:TS-GH-24-019] Verify 422 Unprocessable Entity returns immediately
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusUnprocessableEntity)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.do(context.Background(), http.MethodGet, "/repos/org/repo/pulls/1", nil)
	require.Error(t, err)
	assert.Equal(t, 1, callCount, "expected no retry for 422")
}

/*
Preconditions:
    - Mock server configured to return 422 with detailed JSON error body

Steps:
    1. Call do() with GET request
    2. Inspect the error for response body content

Expected:
    - Error contains response body content ("Validation Failed")
    - Response body details are preserved, not discarded
*/
func TestNonRetryableResponseBodyPreserved(t *testing.T) {
	t.Skip("Phase 1: Design only - awaiting implementation")

	// [test_id:TS-GH-24-020] Verify response body is preserved for non-retryable responses
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprint(w, `{"message": "Validation Failed", "errors": [{"resource": "PullRequest", "field": "title", "code": "missing_field"}]}`)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.do(context.Background(), http.MethodGet, "/repos/org/repo/pulls", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Validation Failed")
}
