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
File Operations Integration Tests

STP Reference: outputs/stp/GH-24/GH-24_test_plan.md
STD Reference: outputs/std/GH-24/GH-24_test_description.yaml
Jira: GH-24
*/

// TestCreateOrUpdateFileSucceedsFirstAttempt validates the happy path where
// CreateOrUpdateFile succeeds on the first attempt without any retries.
// Covers: TS-GH-24-027
func TestCreateOrUpdateFileSucceedsFirstAttempt(t *testing.T) {
	// [test_id:TS-GH-24-027] Verify CreateOrUpdateFile succeeds on first attempt
	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		switch r.Method {
		case http.MethodGet:
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"sha": "abc123"})
		case http.MethodPut:
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{})
		}
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	err := client.CreateOrUpdateFile(context.Background(), "org", "repo", "file.txt", "create file", []byte("hello"))
	require.NoError(t, err)
	assert.Equal(t, int32(2), callCount.Load(), "expected exactly 2 requests (1 GET + 1 PUT)")
}

// TestCreateOrUpdateFileOnBranchRetries404 validates that
// CreateOrUpdateFileOnBranch retries when encountering a 404 (repo/branch not
// yet initialized) via the retryOnRepoRace wrapper. The 404 on PUT triggers
// an APIError which retryOnRepoRace catches and retries the whole operation.
// Covers: TS-GH-24-028
func TestCreateOrUpdateFileOnBranchRetries404(t *testing.T) {
	// [test_id:TS-GH-24-028] Verify CreateOrUpdateFileOnBranch retries on 404 via retryOnRepoRace
	var totalCalls atomic.Int32
	var putCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		totalCalls.Add(1)
		switch r.Method {
		case http.MethodGet:
			// GET always returns 404 (file not found) — this is fine,
			// CreateOrUpdateFileOnBranch treats it as "create new file".
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"message": "Not Found"}`)
		case http.MethodPut:
			n := putCount.Add(1)
			if n == 1 {
				// First PUT returns 404 (branch not ready yet)
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprint(w, `{"message": "Not Found"}`)
				return
			}
			// Second PUT succeeds
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{})
		}
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	err := client.CreateOrUpdateFileOnBranch(context.Background(), "org", "repo", "feature-branch", "file.txt", "update", []byte("content"))
	require.NoError(t, err)
	assert.Greater(t, putCount.Load(), int32(1),
		"expected retryOnRepoRace to retry after PUT returned 404")
}

// TestDeleteFileRetries409 validates that DeleteFile retries when encountering
// a 409 Conflict (branch ref conflict during concurrent operations) via
// retryOnRepoRace.
// Covers: TS-GH-24-029
func TestDeleteFileRetries409(t *testing.T) {
	// [test_id:TS-GH-24-029] Verify DeleteFile retries on 409 via retryOnRepoRace
	var callCount atomic.Int32
	var deleteCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		switch r.Method {
		case http.MethodGet:
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"sha": "abc123"})
		case http.MethodDelete:
			n := deleteCount.Add(1)
			if n == 1 {
				// First DELETE returns 409 (branch ref conflict)
				w.WriteHeader(http.StatusConflict)
				fmt.Fprint(w, `{"message": "Conflict"}`)
				return
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{})
		}
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	err := client.DeleteFile(context.Background(), "org", "repo", "file.txt", "delete file")
	require.NoError(t, err)
	assert.Greater(t, deleteCount.Load(), int32(1),
		"expected DeleteFile to retry on 409")
}

// TestPutFileWithRetryNonTransientPassthrough validates that putFileWithRetry
// does not retry on non-transient errors like 400 or 422, passing them
// through immediately to the caller.
// Covers: TS-GH-24-030
func TestPutFileWithRetryNonTransientPassthrough(t *testing.T) {
	// [test_id:TS-GH-24-030] Verify putFileWithRetry passes through non-transient errors
	tests := []struct {
		name       string
		statusCode int
	}{
		{"422 Unprocessable Entity", http.StatusUnprocessableEntity},
		{"400 Bad Request", http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var callCount atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				callCount.Add(1)
				w.WriteHeader(tc.statusCode)
				fmt.Fprintf(w, `{"message": "Error %d"}`, tc.statusCode)
			}))
			defer srv.Close()

			client := newTestClient(t, srv)
			payload := map[string]any{
				"message": "test",
				"content": "dGVzdA==",
			}
			err := client.putFileWithRetry(context.Background(), "/repos/org/repo/contents/file.txt", payload, "file.txt")
			require.Error(t, err)
			// Only 1 call to do() (no retry for non-transient)
			assert.Equal(t, int32(1), callCount.Load(),
				"expected no retry for non-transient %d error", tc.statusCode)
		})
	}
}

// TestCreateOrUpdateFileRetriesPUTNotGET validates that when PUT fails with
// 5xx at do()-level, only the PUT is retried, not the entire GET+PUT sequence.
func TestCreateOrUpdateFileRetriesPUTNotGET(t *testing.T) {
	var getCount atomic.Int32
	var putCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getCount.Add(1)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"sha": "abc123"})
		case http.MethodPut:
			n := putCount.Add(1)
			if n == 1 {
				w.WriteHeader(http.StatusBadGateway)
				return
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{})
		}
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	err := client.CreateOrUpdateFile(context.Background(), "org", "repo", "file.txt", "update", []byte("content"))
	require.NoError(t, err)
	assert.Equal(t, int32(1), getCount.Load(), "GET should be called exactly once")
	assert.Equal(t, int32(2), putCount.Load(), "PUT should be called twice (1 fail + 1 retry)")
}
