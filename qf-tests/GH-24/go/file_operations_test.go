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
File Operations 5xx Retry Tests

STP Reference: outputs/stp/GH-24/GH-24_test_plan.md
Jira: GH-24
*/

/*
Preconditions:
    - Mock server: GET returns 200 with SHA, first PUT returns 502, second PUT returns 200

Steps:
    1. Call CreateOrUpdateFile to trigger GET (SHA fetch) + PUT (update) sequence
    2. Observe GET and PUT call counts separately

Expected:
    - GET called exactly once (SHA not re-fetched during retry)
    - PUT called twice (1 fail + 1 success via do()-level retry)
*/
func TestCreateOrUpdateFileRetriesPUTNotGET(t *testing.T) {
	// [test_id:TS-GH-24-021] Verify CreateOrUpdateFile retries PUT without re-executing GET
	getCount := 0
	putCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getCount++
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"sha": "abc123"})
		case http.MethodPut:
			putCount++
			if putCount == 1 {
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
	assert.Equal(t, 1, getCount, "GET should be called exactly once")
	assert.Equal(t, 2, putCount, "PUT should be called twice (1 fail + 1 retry)")
}

/*
Preconditions:
    - Mock server: GET returns 200 with SHA, first DELETE returns 503, second DELETE returns 200

Steps:
    1. Call DeleteFile to trigger GET (SHA fetch) + DELETE sequence
    2. Observe total call count

Expected:
    - DeleteFile succeeds after retry
    - 3 total HTTP calls (GET + DELETE(503) + DELETE(200))
*/
func TestDeleteFileRetriesAtDoLevel(t *testing.T) {
	// [test_id:TS-GH-24-022] Verify DeleteFile retries at do() level
	callCount := 0
	deleteCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"sha": "abc123"})
			return
		}
		deleteCount++
		if deleteCount == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprint(w, `{"message": "Service Unavailable"}`)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{})
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	err := client.DeleteFile(context.Background(), "org", "repo", "file.txt", "delete file")
	require.NoError(t, err)
	assert.Equal(t, 3, callCount, "expected GET + DELETE(503) + DELETE(200) = 3 calls")
}

/*
Preconditions:
    - Mock server: GET returns 200 with SHA, first PUT returns 504, second PUT returns 200

Steps:
    1. Call CreateOrUpdateFileOnBranch targeting "feature-branch"
    2. Observe GET and PUT call counts

Expected:
    - GET called exactly once (branch-specific SHA fetch)
    - PUT called twice (1 fail + 1 retry)
    - Same single-layer retry pattern as CreateOrUpdateFile
*/
func TestCreateOrUpdateFileOnBranchSingleLayerRetry(t *testing.T) {
	// [test_id:TS-GH-24-023] Verify CreateOrUpdateFileOnBranch follows same retry pattern
	getCount := 0
	putCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getCount++
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"sha": "def456"})
		case http.MethodPut:
			putCount++
			if putCount == 1 {
				w.WriteHeader(http.StatusGatewayTimeout)
				return
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{})
		}
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	err := client.CreateOrUpdateFileOnBranch(context.Background(), "org", "repo", "feature-branch", "file.txt", "update", []byte("content"))
	require.NoError(t, err)
	assert.Equal(t, 1, getCount, "GET should be called exactly once")
	assert.Equal(t, 2, putCount, "PUT should be called twice (1 fail + 1 retry)")
}
