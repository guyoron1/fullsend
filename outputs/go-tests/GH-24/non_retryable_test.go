package github

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

Note: do() returns (*http.Response, nil) for non-retryable status codes.
The status code error is surfaced by higher-level methods (get/put/post)
via checkStatus(). These tests verify that no retry occurs by checking
callCount == 1.
*/

/*
Preconditions:
    - Mock server configured to return 400

Steps:
    1. Call do() with GET request
    2. Observe call count — should be exactly 1 (no retry)

Expected:
    - do() returns the response without retrying
    - Exactly 1 HTTP call is made (no retry)
*/
func TestNonRetryable400ReturnsImmediately(t *testing.T) {
	// [test_id:TS-GH-24-017] Verify 400 Bad Request returns immediately
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"message": "Bad Request"}`)
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	resp, err := client.do(context.Background(), http.MethodGet, "/repos/org/repo/pulls/1", nil)
	require.NoError(t, err, "do() should not return error for non-retryable status codes")
	require.NotNil(t, resp)
	resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, 1, callCount, "expected no retry for 400")
}

/*
Preconditions:
    - Mock server configured to return 401

Steps:
    1. Call do() with GET request
    2. Observe call count — should be exactly 1 (no retry)

Expected:
    - do() returns the response without retrying
    - Exactly 1 HTTP call is made (no retry)
*/
func TestNonRetryable401ReturnsImmediately(t *testing.T) {
	// [test_id:TS-GH-24-018] Verify 401 Unauthorized returns immediately
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"message": "Bad credentials"}`)
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	resp, err := client.do(context.Background(), http.MethodGet, "/repos/org/repo/pulls/1", nil)
	require.NoError(t, err, "do() should not return error for non-retryable status codes")
	require.NotNil(t, resp)
	resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	assert.Equal(t, 1, callCount, "expected no retry for 401")
}

/*
Preconditions:
    - Mock server configured to return 422

Steps:
    1. Call do() with GET request
    2. Observe call count — should be exactly 1 (no retry)

Expected:
    - do() returns the response without retrying
    - Exactly 1 HTTP call is made (no retry)
*/
func TestNonRetryable422ReturnsImmediately(t *testing.T) {
	// [test_id:TS-GH-24-019] Verify 422 Unprocessable Entity returns immediately
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprint(w, `{"message": "Validation Failed"}`)
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	resp, err := client.do(context.Background(), http.MethodGet, "/repos/org/repo/pulls/1", nil)
	require.NoError(t, err, "do() should not return error for non-retryable status codes")
	require.NotNil(t, resp)
	resp.Body.Close()
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
	assert.Equal(t, 1, callCount, "expected no retry for 422")
}

/*
Preconditions:
    - Mock server configured to return 422 with detailed JSON error body

Steps:
    1. Call do() with GET request
    2. Inspect the response for preserved body content

Expected:
    - Response body content is preserved ("Validation Failed")
    - Response body details are accessible, not discarded
*/
func TestNonRetryableResponseBodyPreserved(t *testing.T) {
	// [test_id:TS-GH-24-020] Verify response body is preserved for non-retryable responses
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprint(w, `{"message": "Validation Failed", "errors": [{"resource": "PullRequest", "field": "title", "code": "missing_field"}]}`)
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	// do() returns the response directly for non-retryable status codes.
	// Use get() which wraps do() and calls checkStatus() to extract the error.
	_, err := client.get(context.Background(), "/repos/org/repo/pulls")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Validation Failed")
}
