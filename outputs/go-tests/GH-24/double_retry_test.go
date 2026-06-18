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
Double-Retry Prevention Tests

STP Reference: outputs/stp/GH-24/GH-24_test_plan.md
STD Reference: outputs/std/GH-24/GH-24_test_description.yaml
Jira: GH-24
*/

// TestCreateOrUpdateFileNoDoubleRetry504 validates that CreateOrUpdateFile
// with a 504 on PUT results in exactly 3 HTTP calls: GET(200) + PUT(504) +
// PUT(200). The retry happens only at do() level, not retryOnRepoRace.
// Covers: TS-GH-24-013
func TestCreateOrUpdateFileNoDoubleRetry504(t *testing.T) {
	// [test_id:TS-GH-24-013] Verify CreateOrUpdateFile with 504 retries only at do() level
	var callCount atomic.Int32
	var putCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"sha": "abc123"})
			return
		}
		n := putCount.Add(1)
		if n == 1 {
			w.WriteHeader(http.StatusGatewayTimeout)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{})
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	err := client.CreateOrUpdateFile(context.Background(), "org", "repo", "file.txt", "update file", []byte("content"))
	require.NoError(t, err)
	assert.Equal(t, int32(3), callCount.Load(), "expected GET + PUT(504) + PUT(200) = 3 calls")
}

// TestCreateOrUpdateFileSingleLayerRetryAll5xx validates that CreateOrUpdateFile
// handles all 5xx status codes (500-504) with retries occurring only at the
// do() level, not duplicated by retryOnRepoRace.
// Covers: TS-GH-24-014
func TestCreateOrUpdateFileSingleLayerRetryAll5xx(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		// [test_id:TS-GH-24-014] Verify single-layer retry for all 5xx codes
		{"500 Internal Server Error", http.StatusInternalServerError},
		{"501 Not Implemented", 501},
		{"502 Bad Gateway", http.StatusBadGateway},
		{"503 Service Unavailable", http.StatusServiceUnavailable},
		{"504 Gateway Timeout", http.StatusGatewayTimeout},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var callCount atomic.Int32
			var putCount atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				callCount.Add(1)
				if r.Method == http.MethodGet {
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(map[string]any{"sha": "abc123"})
					return
				}
				n := putCount.Add(1)
				if n == 1 {
					w.WriteHeader(tc.statusCode)
					return
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]any{})
			}))
			defer srv.Close()

			client := newTestClient(t, srv)
			err := client.CreateOrUpdateFile(context.Background(), "org", "repo", "file.txt", "update", []byte("content"))
			require.NoError(t, err)
			assert.Equal(t, int32(3), callCount.Load(), "expected 3 calls for %d", tc.statusCode)
		})
	}
}

// TestPersistent5xxExhaustsDoRetryOnly validates that when do() exhausts all
// retries on a persistent 5xx error, retryOnRepoRace does not attempt
// additional retries. The error propagates directly to the caller.
// Covers: TS-GH-24-015
func TestPersistent5xxExhaustsDoRetryOnly(t *testing.T) {
	// [test_id:TS-GH-24-015] Verify retryOnRepoRace does not re-invoke on do()-exhausted 5xx
	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"sha": "abc123"})
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprint(w, `{"message": "Service Unavailable"}`)
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	err := client.CreateOrUpdateFile(context.Background(), "org", "repo", "file.txt", "update", []byte("content"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "retryable error after 3 attempts")
	// 1 GET + maxRetries PUT attempts = 1 + 3 = 4 total calls
	// If retryOnRepoRace added retries, total would be much higher
	assert.Equal(t, int32(1+maxRetries), callCount.Load(),
		"expected 1 GET + maxRetries PUTs (no retryOnRepoRace multiplier)")
}
