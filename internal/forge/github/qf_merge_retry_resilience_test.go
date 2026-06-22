package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMergeChangeProposal_UpdateBranchAPIFailureResilience validates that
// MergeChangeProposal continues retrying the merge even when the update-branch
// API call fails (e.g., returns 403 Forbidden). The implementation silently
// ignores update-branch errors and proceeds to the next merge attempt.
//
// STD Reference: outputs/std/GH-75/GH-75_test_description.yaml
// Scenario:      TS-06 (test_id: TS-GH-75-006)
// Jira:          GH-75
func TestMergeChangeProposal_UpdateBranchAPIFailureResilience(t *testing.T) {
	t.Run("[test_id:TS-GH-75-006] update-branch 403 failure does not abort retry loop", func(t *testing.T) {
		// SETUP-01: Create httptest server with route handler.
		// - PUT /merge: attempt 1 → 409, attempt 2 → 200
		// - PUT /update-branch: → 403 Forbidden
		var mergeAttempts atomic.Int32
		var updateCalls atomic.Int32

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodPut && r.URL.Path == "/repos/org/repo/pulls/7/merge":
				attempt := mergeAttempts.Add(1)
				if attempt == 1 {
					w.WriteHeader(http.StatusConflict)
					json.NewEncoder(w).Encode(map[string]string{
						"message": "Head branch is out of date",
					})
					return
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]string{"sha": "merged123"})

			case r.Method == http.MethodPut && r.URL.Path == "/repos/org/repo/pulls/7/update-branch":
				updateCalls.Add(1)
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]string{
					"message": "Resource not accessible by integration",
				})

			default:
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer srv.Close()

		// SETUP-02: Create test client pointing at httptest server.
		client := newTestClient(t, srv)

		// TEST-01: Call MergeChangeProposal — should handle 409 → update-branch
		// failure → retry merge → success.
		err := client.MergeChangeProposal(context.Background(), "org", "repo", 7)

		// ASSERT-01: Method returns no error (merge eventually succeeds).
		require.NoError(t, err)

		// ASSERT-02: Merge endpoint was called at least twice.
		assert.GreaterOrEqual(t, mergeAttempts.Load(), int32(2),
			"should have retried merge after update-branch failure")

		// ASSERT-03: Update-branch endpoint was called at least once.
		assert.GreaterOrEqual(t, updateCalls.Load(), int32(1),
			"should have attempted to update branch")
	})

	t.Run("[test_id:TS-GH-75-006] update-branch 500 failure does not abort retry loop", func(t *testing.T) {
		// Variant: update-branch returns 500 Internal Server Error instead of 403.
		// The retry logic must be equally resilient.
		var mergeAttempts atomic.Int32
		var updateCalls atomic.Int32

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodPut && r.URL.Path == "/repos/org/repo/pulls/7/merge":
				attempt := mergeAttempts.Add(1)
				if attempt == 1 {
					w.WriteHeader(http.StatusConflict)
					json.NewEncoder(w).Encode(map[string]string{
						"message": "Head branch is out of date",
					})
					return
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]string{"sha": "merged456"})

			case r.Method == http.MethodPut && r.URL.Path == "/repos/org/repo/pulls/7/update-branch":
				updateCalls.Add(1)
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{
					"message": "Internal Server Error",
				})

			default:
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer srv.Close()

		client := newTestClient(t, srv)
		err := client.MergeChangeProposal(context.Background(), "org", "repo", 7)

		require.NoError(t, err)
		assert.GreaterOrEqual(t, mergeAttempts.Load(), int32(2),
			"should have retried merge after update-branch 500")
		assert.GreaterOrEqual(t, updateCalls.Load(), int32(1),
			"should have attempted to update branch")
	})
}
