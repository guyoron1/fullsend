package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Double-Retry Prevention Tests

STP Reference: outputs/stp/GH-24/GH-24_test_plan.md
Jira: GH-24
*/

/*
Preconditions:
    - Mock server: GET returns 200 (SHA fetch), first PUT returns 504, second PUT returns 200

Steps:
    1. Call CreateOrUpdateFile to trigger GET + PUT sequence
    2. Observe total HTTP call count

Expected:
    - Exactly 3 HTTP calls: GET(200) + PUT(504) + PUT(200)
    - CreateOrUpdateFile returns success
*/
func TestCreateOrUpdateFileNoDoubleRetry504(t *testing.T) {
	// [test_id:TS-GH-24-011] Verify CreateOrUpdateFile with 504 results in exactly 3 HTTP calls
	callCount := 0
	putCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"sha": "abc123"})
			return
		}
		putCount++
		if putCount == 1 {
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
	assert.Equal(t, 3, callCount, "expected GET + PUT(504) + PUT(200) = 3 calls")
}

/*
Preconditions:
    - Mock server: GET returns 200, first PUT returns parameterized 5xx, second PUT returns 200

Steps:
    1. For each 5xx code (500, 502, 503, 504), call CreateOrUpdateFile
    2. Verify consistent call count across all codes

Expected:
    - 3 HTTP calls for each 5xx code (single-layer retry pattern)
*/
func TestCreateOrUpdateFileSingleLayerRetryAll5xx(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		// [test_id:TS-GH-24-012] Verify single-layer retry for all 5xx codes
		{"500 Internal Server Error", http.StatusInternalServerError},
		{"502 Bad Gateway", http.StatusBadGateway},
		{"503 Service Unavailable", http.StatusServiceUnavailable},
		{"504 Gateway Timeout", http.StatusGatewayTimeout},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			callCount := 0
			putCount := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				callCount++
				if r.Method == http.MethodGet {
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(map[string]any{"sha": "abc123"})
					return
				}
				putCount++
				if putCount == 1 {
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
			assert.Equal(t, 3, callCount, "expected 3 calls for %d", tc.statusCode)
		})
	}
}

/*
Preconditions:
    - Mock server: GET returns 200 (SHA fetch), all PUTs return 503

Steps:
    1. Call CreateOrUpdateFile with persistent 5xx on PUT
    2. Observe total call count and error

Expected:
    - do() makes maxRetries=3 PUT attempts
    - 1 GET + 3 PUTs = 4 total calls
    - Error contains "retryable error after 3 attempts"
    - retryOnRepoRace does not add additional attempts
*/
func TestPersistent5xxExhaustsDoRetryOnly(t *testing.T) {
	// [test_id:TS-GH-24-013] Verify persistent 5xx exhausts do() retries without retryOnRepoRace
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
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
	// 1 GET + 3 PUT attempts (maxRetries=3) = 4 total calls
	assert.Equal(t, 4, callCount, "expected 1 GET + 3 PUTs (maxRetries=3)")
}
