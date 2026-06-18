package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
isTransientStatus and retryOnRepoRace Tests

STP Reference: outputs/stp/GH-24/GH-24_test_plan.md
STD Reference: outputs/std/GH-24/GH-24_test_description.yaml
Jira: GH-24
*/

// TestIsTransientStatusTrue404And409 validates that isTransientStatus returns
// true for 404 (async repo init) and 409 (branch ref conflict).
// Covers: TS-GH-24-021, TS-GH-24-022
func TestIsTransientStatusTrue404And409(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		testID     string
	}{
		// [test_id:TS-GH-24-021] Verify isTransientStatus returns true for 404
		{"404 Not Found", http.StatusNotFound, "TS-GH-24-021"},
		// [test_id:TS-GH-24-022] Verify isTransientStatus returns true for 409
		{"409 Conflict", http.StatusConflict, "TS-GH-24-022"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := isTransientStatus(tc.statusCode)
			assert.True(t, result, "[%s] expected isTransientStatus to return true for %d",
				tc.testID, tc.statusCode)
		})
	}
}

// TestIsTransientStatusFalse5xx validates that isTransientStatus returns false
// for all 5xx status codes, since 5xx retries are now handled by do().
// Covers: TS-GH-24-023
func TestIsTransientStatusFalse5xx(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		// [test_id:TS-GH-24-023] Verify isTransientStatus returns false for 500-504
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

// TestRetryOnRepoRaceRetries404 validates that retryOnRepoRace retries when
// it encounters a 404 (async repo initialization race condition).
// Covers: TS-GH-24-016
func TestRetryOnRepoRaceRetries404(t *testing.T) {
	// [test_id:TS-GH-24-016] Verify retryOnRepoRace retries on 404
	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		if n <= 1 {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"message": "Not Found"}`)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"sha": "abc123"})
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	err := client.retryOnRepoRace(context.Background(), "test-404", func() error {
		resp, err := client.do(context.Background(), http.MethodGet, "/repos/org/repo/contents/file.txt", nil)
		if err != nil {
			return err
		}
		if err := checkStatus(resp, http.StatusOK); err != nil {
			return err
		}
		resp.Body.Close()
		return nil
	})
	require.NoError(t, err)
	assert.Greater(t, callCount.Load(), int32(1), "expected retryOnRepoRace to retry on 404")
}

// TestRetryOnRepoRaceRetries409 validates that retryOnRepoRace retries when
// it encounters a 409 Conflict (branch ref conflict).
// Covers: TS-GH-24-017
func TestRetryOnRepoRaceRetries409(t *testing.T) {
	// [test_id:TS-GH-24-017] Verify retryOnRepoRace retries on 409
	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		if n <= 1 {
			w.WriteHeader(http.StatusConflict)
			fmt.Fprint(w, `{"message": "Conflict"}`)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"sha": "abc123"})
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	err := client.retryOnRepoRace(context.Background(), "test-409", func() error {
		resp, err := client.do(context.Background(), http.MethodGet, "/repos/org/repo/contents/file.txt", nil)
		if err != nil {
			return err
		}
		if err := checkStatus(resp, http.StatusOK); err != nil {
			return err
		}
		resp.Body.Close()
		return nil
	})
	require.NoError(t, err)
	assert.Greater(t, callCount.Load(), int32(1), "expected retryOnRepoRace to retry on 409")
}

// TestRetryOnRepoRaceDoesNotRetry500 validates that retryOnRepoRace does NOT
// retry when do() returns a 500 error (after do() has exhausted its retries).
// Covers: TS-GH-24-018
func TestRetryOnRepoRaceDoesNotRetry500(t *testing.T) {
	// [test_id:TS-GH-24-018] Verify retryOnRepoRace does not retry on 500
	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	err := client.retryOnRepoRace(context.Background(), "test-500", func() error {
		_, innerErr := client.do(context.Background(), http.MethodGet, "/repos/org/repo/contents/file.txt", nil)
		return innerErr
	})
	require.Error(t, err)
	// do() makes maxRetries=3 calls. retryOnRepoRace should NOT add more.
	assert.Equal(t, int32(maxRetries), callCount.Load(),
		"expected only do()-level retries (maxRetries calls), not retryOnRepoRace retries")
}

// TestRetryOnRepoRaceDoesNotRetry502 validates that retryOnRepoRace does NOT
// retry when do() returns a 502 error.
// Covers: TS-GH-24-019
func TestRetryOnRepoRaceDoesNotRetry502(t *testing.T) {
	// [test_id:TS-GH-24-019] Verify retryOnRepoRace does not retry on 502
	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	err := client.retryOnRepoRace(context.Background(), "test-502", func() error {
		_, innerErr := client.do(context.Background(), http.MethodGet, "/repos/org/repo/contents/file.txt", nil)
		return innerErr
	})
	require.Error(t, err)
	assert.Equal(t, int32(maxRetries), callCount.Load(),
		"expected only do()-level retries, not retryOnRepoRace retries for 502")
}

// TestRetryOnRepoRaceExhaustsAndReturnsWrappedError validates that
// retryOnRepoRace exhausts its retry attempts when the underlying operation
// persistently returns 404, and returns a wrapped error with context.
// Covers: TS-GH-24-020
func TestRetryOnRepoRaceExhaustsAndReturnsWrappedError(t *testing.T) {
	// [test_id:TS-GH-24-020] Verify retryOnRepoRace exhausts attempts and returns wrapped error
	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"message": "Not Found"}`)
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	err := client.retryOnRepoRace(context.Background(), "test-exhaust", func() error {
		resp, innerErr := client.do(context.Background(), http.MethodGet, "/repos/org/repo/contents/file.txt", nil)
		if innerErr != nil {
			return innerErr
		}
		if statusErr := checkStatus(resp, http.StatusOK); statusErr != nil {
			return statusErr
		}
		resp.Body.Close()
		return nil
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "after",
		"error should contain retry exhaustion context")
	assert.Contains(t, err.Error(), "attempts",
		"error should indicate number of attempts")
}
