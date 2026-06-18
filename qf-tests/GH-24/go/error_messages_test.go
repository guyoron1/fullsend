package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Retry Exhaustion Error Message Tests

STP Reference: outputs/stp/GH-24/GH-24_test_plan.md
STD Reference: outputs/std/GH-24/GH-24_test_description.yaml
Jira: GH-24
*/

// TestRetryExhaustionErrorContainsRetryableError validates that the error
// message contains "retryable error" and does NOT contain the old
// "rate limited" text.
// Covers: TS-GH-24-011, TS-GH-24-024
func TestRetryExhaustionErrorContainsRetryableError(t *testing.T) {
	// [test_id:TS-GH-24-011] Verify error message reads 'retryable error after N attempts'
	// [test_id:TS-GH-24-024] Verify error message contains 'retryable error' (not 'rate limited')
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	_, err := client.do(context.Background(), http.MethodGet, "/repos/org/repo/pulls/1", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "retryable error",
		"error message should contain 'retryable error'")
	assert.True(t, !strings.Contains(err.Error(), "rate limited"),
		"error message should NOT contain old 'rate limited' text")
	assert.Contains(t, err.Error(), "attempts",
		"error message should include attempt count")
}

// TestRetryExhaustionErrorContainsMethodAndPath validates that the error
// message includes the HTTP method and request path for debugging.
// Covers: TS-GH-24-025
func TestRetryExhaustionErrorContainsMethodAndPath(t *testing.T) {
	// [test_id:TS-GH-24-025] Verify error includes method, path, and delay information
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	_, err := client.do(context.Background(), http.MethodGet, "/repos/org/repo/contents/file", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "GET",
		"error message should include HTTP method")
	assert.Contains(t, err.Error(), "/repos/org/repo/contents/file",
		"error message should include request path")
	assert.Contains(t, err.Error(), "delay",
		"error message should include delay information")
}

// TestRetryExhaustionErrorIncludesRetryAfter validates that when a 5xx
// response includes a Retry-After header, the error message incorporates it.
// Covers: TS-GH-24-026
func TestRetryExhaustionErrorIncludesRetryAfter(t *testing.T) {
	// [test_id:TS-GH-24-026] Verify error includes Retry-After header value when present
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "1")
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	_, err := client.do(context.Background(), http.MethodGet, "/repos/org/repo/contents/file", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Retry-After",
		"error message should include Retry-After header information")
}
