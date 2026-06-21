package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Merge Retry on 409 — Functional Tests (Implemented)

STP Reference: outputs/stp/GH-2432/GH-2432_test_plan.md
STD Reference: outputs/std/GH-2432/GH-2432_test_description.yaml
Jira: GH-2432

Tests validate the retry logic in MergeChangeProposal when the GitHub
merge API returns HTTP 409 ("Head branch is out of date"). The fix
calls the update-branch API to sync the PR branch and retries the
merge up to 3 times with a 3-second delay between attempts.
*/

func TestMergeChangeProposal_RetryOn409(t *testing.T) {
	t.Run("[test_id:TS-GH-2432-001] should succeed after 409 with branch update", func(t *testing.T) {
		var mergeAttempts atomic.Int32
		var updateBranchCalls atomic.Int32

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodPut && r.URL.Path == "/repos/org/repo/pulls/42/merge":
				attempt := mergeAttempts.Add(1)
				if attempt == 1 {
					w.WriteHeader(http.StatusConflict)
					json.NewEncoder(w).Encode(map[string]string{
						"message": "Head branch was modified. Review and try the merge again.",
					})
					return
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"merged":  true,
					"message": "Pull Request successfully merged",
				})

			case r.Method == http.MethodPut && r.URL.Path == "/repos/org/repo/pulls/42/update-branch":
				updateBranchCalls.Add(1)
				w.WriteHeader(http.StatusAccepted)
				json.NewEncoder(w).Encode(map[string]string{
					"message": "Updating pull request branch.",
				})

			default:
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer srv.Close()

		client := newTestClient(t, srv)
		err := client.MergeChangeProposal(context.Background(), "org", "repo", 42)

		require.NoError(t, err, "MergeChangeProposal should succeed after 409 retry")
		assert.Equal(t, int32(2), mergeAttempts.Load(),
			"merge endpoint should receive exactly 2 PUT requests")
		assert.Equal(t, int32(1), updateBranchCalls.Load(),
			"update-branch should be called once between the failed and successful merge")
	})

	t.Run("[test_id:TS-GH-2432-002] should call update-branch before retry", func(t *testing.T) {
		var mu sync.Mutex
		var requestLog []string

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			requestLog = append(requestLog, r.Method+" "+r.URL.Path)
			reqIndex := len(requestLog)
			mu.Unlock()

			switch {
			case r.Method == http.MethodPut && r.URL.Path == "/repos/org/repo/pulls/7/merge":
				if reqIndex == 1 {
					w.WriteHeader(http.StatusConflict)
					json.NewEncoder(w).Encode(map[string]string{
						"message": "Head branch was modified.",
					})
					return
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]string{"sha": "abc123"})

			case r.Method == http.MethodPut && r.URL.Path == "/repos/org/repo/pulls/7/update-branch":
				w.WriteHeader(http.StatusAccepted)
				json.NewEncoder(w).Encode(map[string]string{
					"message": "Updating pull request branch.",
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

		mu.Lock()
		defer mu.Unlock()
		require.Len(t, requestLog, 3, "should have exactly 3 requests: merge(409), update-branch, merge(200)")

		assert.Contains(t, requestLog[0], "/merge",
			"first request should be a merge attempt")
		assert.Contains(t, requestLog[1], "/update-branch",
			"second request should be update-branch")
		assert.Contains(t, requestLog[2], "/merge",
			"third request should be the retry merge")
	})

	t.Run("[test_id:TS-GH-2432-003] should return error when update-branch fails", func(t *testing.T) {
		// NOTE: The implementation silently ignores update-branch errors.
		// When update-branch fails, the retry loop continues and the merge
		// is attempted again. After exhausting all 3 attempts, the
		// exhaustion error is returned.
		var mergeAttempts atomic.Int32

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodPut && r.URL.Path == "/repos/org/repo/pulls/7/merge":
				mergeAttempts.Add(1)
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(map[string]string{
					"message": "Head branch was modified.",
				})

			case r.Method == http.MethodPut && r.URL.Path == "/repos/org/repo/pulls/7/update-branch":
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{
					"message": "Internal server error",
				})

			default:
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer srv.Close()

		client := newTestClient(t, srv)
		err := client.MergeChangeProposal(context.Background(), "org", "repo", 7)

		require.Error(t, err, "should return error after retry exhaustion with failing update-branch")
		assert.Contains(t, err.Error(), "merge pull request #7",
			"error should reference the PR number")
		assert.Equal(t, int32(3), mergeAttempts.Load(),
			"should still attempt merge 3 times even with update-branch failures")
	})

	t.Run("[test_id:TS-GH-2432-010] should succeed on first attempt without retry", func(t *testing.T) {
		var mergeCallCount atomic.Int32
		var updateBranchCallCount atomic.Int32

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodPut && r.URL.Path == "/repos/org/repo/pulls/99/merge":
				mergeCallCount.Add(1)
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"merged":  true,
					"message": "Pull Request successfully merged",
				})

			case r.URL.Path == "/repos/org/repo/pulls/99/update-branch":
				updateBranchCallCount.Add(1)
				w.WriteHeader(http.StatusAccepted)

			default:
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer srv.Close()

		client := newTestClient(t, srv)
		err := client.MergeChangeProposal(context.Background(), "org", "repo", 99)

		require.NoError(t, err, "merge should succeed on first attempt")
		assert.Equal(t, int32(1), mergeCallCount.Load(),
			"merge endpoint should receive exactly 1 request")
		assert.Equal(t, int32(0), updateBranchCallCount.Load(),
			"update-branch should NOT be called when merge succeeds immediately")
	})
}

func TestMergeChangeProposal_Non409Errors(t *testing.T) {
	t.Run("[test_id:TS-GH-2432-004] should return 422 error without retry", func(t *testing.T) {
		var mergeCallCount atomic.Int32
		var updateBranchCallCount atomic.Int32

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodPut && r.URL.Path == "/repos/org/repo/pulls/5/merge":
				mergeCallCount.Add(1)
				w.WriteHeader(http.StatusUnprocessableEntity)
				json.NewEncoder(w).Encode(map[string]string{
					"message": "Pull Request is not mergeable",
				})

			case r.URL.Path == "/repos/org/repo/pulls/5/update-branch":
				updateBranchCallCount.Add(1)
				w.WriteHeader(http.StatusAccepted)

			default:
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer srv.Close()

		client := newTestClient(t, srv)
		err := client.MergeChangeProposal(context.Background(), "org", "repo", 5)

		require.Error(t, err, "should return error for 422 response")
		assert.Contains(t, err.Error(), "not mergeable",
			"error should contain the GitHub API message")
		assert.Equal(t, int32(1), mergeCallCount.Load(),
			"should not retry non-409 errors — only 1 merge attempt expected")
		assert.Equal(t, int32(0), updateBranchCallCount.Load(),
			"update-branch should not be called for non-409 errors")
	})

	t.Run("[test_id:TS-GH-2432-005] should not call update-branch on non-409", func(t *testing.T) {
		var updateBranchCallCount atomic.Int32

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodPut && r.URL.Path == "/repos/org/repo/pulls/5/merge":
				w.WriteHeader(http.StatusUnprocessableEntity)
				json.NewEncoder(w).Encode(map[string]string{
					"message": "Pull Request is not mergeable",
				})

			case r.URL.Path == "/repos/org/repo/pulls/5/update-branch":
				updateBranchCallCount.Add(1)
				w.WriteHeader(http.StatusAccepted)

			default:
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer srv.Close()

		client := newTestClient(t, srv)
		err := client.MergeChangeProposal(context.Background(), "org", "repo", 5)

		require.Error(t, err, "should propagate the merge error")
		assert.Equal(t, int32(0), updateBranchCallCount.Load(),
			"update-branch endpoint should receive zero requests for non-409 errors")
	})
}

func TestMergeChangeProposal_RetryExhaustion(t *testing.T) {
	t.Run("[test_id:TS-GH-2432-006] should give up after 3 failed retries", func(t *testing.T) {
		var mergeCallCount atomic.Int32
		var updateBranchCallCount atomic.Int32

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodPut && r.URL.Path == "/repos/org/repo/pulls/7/merge":
				mergeCallCount.Add(1)
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(map[string]string{
					"message": "Head branch was modified.",
				})

			case r.Method == http.MethodPut && r.URL.Path == "/repos/org/repo/pulls/7/update-branch":
				updateBranchCallCount.Add(1)
				w.WriteHeader(http.StatusAccepted)
				json.NewEncoder(w).Encode(map[string]string{
					"message": "Updating pull request branch.",
				})

			default:
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer srv.Close()

		client := newTestClient(t, srv)
		err := client.MergeChangeProposal(context.Background(), "org", "repo", 7)

		require.Error(t, err, "should return error after exhausting all retry attempts")
		assert.Equal(t, int32(3), mergeCallCount.Load(),
			"should attempt merge exactly 3 times (initial + 2 retries)")
		assert.Equal(t, int32(3), updateBranchCallCount.Load(),
			"should call update-branch once after each 409 response")
	})

	t.Run("[test_id:TS-GH-2432-007] should include attempt count in error message", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodPut && r.URL.Path == "/repos/org/repo/pulls/7/merge":
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(map[string]string{
					"message": "Head branch was modified.",
				})

			case r.Method == http.MethodPut && r.URL.Path == "/repos/org/repo/pulls/7/update-branch":
				w.WriteHeader(http.StatusAccepted)
				json.NewEncoder(w).Encode(map[string]string{
					"message": "Updating pull request branch.",
				})

			default:
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer srv.Close()

		client := newTestClient(t, srv)
		err := client.MergeChangeProposal(context.Background(), "org", "repo", 7)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "3",
			"error message should include the attempt count")
		assert.Contains(t, err.Error(), "out of date",
			"error message should reference the out-of-date branch condition")
		assert.Contains(t, err.Error(), "merge pull request #7",
			"error message should identify the PR")
	})
}

func TestMergeChangeProposal_ContextCancellation(t *testing.T) {
	t.Run("[test_id:TS-GH-2432-008] should abort retry on cancelled context", func(t *testing.T) {
		var mergeCallCount atomic.Int32

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodPut && r.URL.Path == "/repos/org/repo/pulls/7/merge":
				mergeCallCount.Add(1)
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(map[string]string{
					"message": "Head branch was modified.",
				})

			case r.Method == http.MethodPut && r.URL.Path == "/repos/org/repo/pulls/7/update-branch":
				w.WriteHeader(http.StatusAccepted)
				json.NewEncoder(w).Encode(map[string]string{
					"message": "Updating pull request branch.",
				})

			default:
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer srv.Close()

		// Create a context that we cancel after a short delay, well before
		// the 3-second retry wait would complete.
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			// Cancel after 200ms — enough time for the first merge attempt
			// and update-branch call, but not enough for the 3s retry delay.
			time.Sleep(200 * time.Millisecond)
			cancel()
		}()

		client := newTestClient(t, srv)
		start := time.Now()
		err := client.MergeChangeProposal(ctx, "org", "repo", 7)
		elapsed := time.Since(start)

		require.Error(t, err, "should return error when context is cancelled")
		assert.Less(t, elapsed, 5*time.Second,
			"should return promptly after cancellation, not block for full retry timeout")
		assert.Equal(t, int32(1), mergeCallCount.Load(),
			"should only have attempted merge once before context cancellation aborted retry")
	})

	t.Run("[test_id:TS-GH-2432-009] should return context.Canceled error on cancellation", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodPut && r.URL.Path == "/repos/org/repo/pulls/7/merge":
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(map[string]string{
					"message": "Head branch was modified.",
				})

			case r.Method == http.MethodPut && r.URL.Path == "/repos/org/repo/pulls/7/update-branch":
				w.WriteHeader(http.StatusAccepted)
				json.NewEncoder(w).Encode(map[string]string{
					"message": "Updating pull request branch.",
				})

			default:
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer srv.Close()

		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(200 * time.Millisecond)
			cancel()
		}()

		client := newTestClient(t, srv)
		err := client.MergeChangeProposal(ctx, "org", "repo", 7)

		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled,
			"returned error should wrap context.Canceled so callers can distinguish cancellation from merge failure")
	})
}
