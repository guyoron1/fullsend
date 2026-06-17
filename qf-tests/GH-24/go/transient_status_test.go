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
isTransientStatus and retryOnRepoRace Tests

STP Reference: outputs/stp/GH-24/GH-24_test_plan.md
Jira: GH-24
*/

/*
Preconditions:
    - None (pure function test)

Steps:
    1. Call isTransientStatus with 404 and 409 status codes

Expected:
    - isTransientStatus returns true for 404 (async repo init)
    - isTransientStatus returns true for 409 (branch ref conflict)
*/
func TestIsTransientStatusTrue404And409(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		// [test_id:TS-GH-24-008] Verify isTransientStatus returns true for 404 and 409
		{"404 Not Found", http.StatusNotFound},
		{"409 Conflict", http.StatusConflict},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := isTransientStatus(tc.statusCode)
			assert.True(t, result, "expected isTransientStatus to return true for %d", tc.statusCode)
		})
	}
}

/*
Preconditions:
    - None (pure function test)

Steps:
    1. Call isTransientStatus with 5xx codes (500, 502, 503, 504)

Expected:
    - isTransientStatus returns false for all 5xx codes (moved to do()-level retry)
*/
func TestIsTransientStatusFalse5xx(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		// [test_id:TS-GH-24-009] Verify isTransientStatus returns false for 500-504
		{"500 Internal Server Error", http.StatusInternalServerError},
		{"502 Bad Gateway", http.StatusBadGateway},
		{"503 Service Unavailable", http.StatusServiceUnavailable},
		{"504 Gateway Timeout", http.StatusGatewayTimeout},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := isTransientStatus(tc.statusCode)
			assert.False(t, result, "expected isTransientStatus to return false for %d", tc.statusCode)
		})
	}
}

/*
Preconditions:
    - Mock server configured to always return 503

Steps:
    1. Call retryOnRepoRace wrapping a do() call to the mock server
    2. Observe call count

Expected:
    - Only do()-level retries occur (maxRetries=3 total calls)
    - retryOnRepoRace does not add additional retry attempts for 5xx
*/
func TestRetryOnRepoRaceDoesNotRetry5xx(t *testing.T) {
	// [test_id:TS-GH-24-010] Verify retryOnRepoRace does not retry 5xx errors
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	err := client.retryOnRepoRace(context.Background(), "test-5xx", func() error {
		_, innerErr := client.do(context.Background(), http.MethodGet, "/repos/org/repo/contents/file.txt", nil)
		if innerErr != nil {
			return innerErr
		}
		return nil
	})
	require.Error(t, err)
	// do() makes maxRetries=3 total calls. retryOnRepoRace should NOT
	// add more attempts because 5xx is not a transient status.
	assert.Equal(t, 3, callCount, "expected only do()-level retries (3 calls), not retryOnRepoRace retries")
}
