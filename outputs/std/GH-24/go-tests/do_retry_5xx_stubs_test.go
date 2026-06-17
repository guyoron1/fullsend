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
do() 5xx Retry Behavior Tests

STP Reference: outputs/stp/GH-24/GH-24_test_plan.md
Jira: GH-24
*/

/*
Preconditions:
    - Mock server configured to return 502 on first call, 200 on second

Steps:
    1. Call do() with GET request to mock server
    2. Observe retry behavior

Expected:
    - do() returns no error after successful retry
    - Exactly 2 HTTP calls are made (1 fail + 1 success)
*/
func TestDoRetries502AndSucceeds(t *testing.T) {
	t.Skip("Phase 1: Design only - awaiting implementation")

	// [test_id:TS-GH-24-002] Verify do() retries a 502 and succeeds on next attempt
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"sha": "abc123"}`)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.do(context.Background(), http.MethodGet, "/repos/org/repo/pulls/1", nil)
	require.NoError(t, err)
	assert.Equal(t, 2, callCount, "expected exactly 2 HTTP calls (1 fail + 1 success)")
}

/*
Preconditions:
    - Mock server configured to always return 503

Steps:
    1. Call do() with GET request to mock server
    2. Observe retry exhaustion behavior

Expected:
    - do() returns error after retry exhaustion
    - Error message contains "retryable error after 3 attempts"
    - Exactly 4 HTTP calls made (1 initial + 3 retries)
*/
func TestDoExhaustsRetriesOnPersistent5xx(t *testing.T) {
	t.Skip("Phase 1: Design only - awaiting implementation")

	// [test_id:TS-GH-24-003] Verify do() exhausts retries on persistent 5xx
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.do(context.Background(), http.MethodGet, "/repos/org/repo/pulls/1", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "retryable error after 3 attempts")
	assert.Equal(t, 4, callCount, "expected 1 initial + 3 retries = 4 total calls")
}
