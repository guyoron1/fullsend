package github_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ghclient "github.com/fullsend-ai/fullsend/internal/forge/github"
)

/*
MergeChangeProposal Retry Logic Tests

STP Reference: outputs/stp/GH-2432/GH-2432_test_plan.md
STD Reference: outputs/std/GH-2432/GH-2432_test_description.yaml
Jira: GH-2432

These tests validate the retry-on-409 behavior added to MergeChangeProposal
in internal/forge/github/github.go. Each test uses httptest.NewServer to
mock GitHub API responses and validate retry, error propagation, and
context cancellation behavior.

Shared Preconditions:
    - Go 1.23+ toolchain available
    - httptest mock server for GitHub API simulation
    - LiveClient instantiated with mock server URL
*/

// newClient creates a LiveClient pointed at the given httptest server.
func newClient(t *testing.T, srv *httptest.Server) *ghclient.LiveClient {
	t.Helper()
	return ghclient.New("test-token").WithBaseURL(srv.URL)
}

// TestMergeChangeProposal_SuccessOnFirstAttempt validates that MergeChangeProposal
// completes successfully when the GitHub API returns 200 OK on the first merge
// attempt. No retry logic should be triggered and no update-branch call should
// be made.
//
// Requirement: REQ-001
// Priority: P0
func TestMergeChangeProposal_SuccessOnFirstAttempt(t *testing.T) {
	// [test_id:TS-GH-2432-001]
	var mergeCallCount atomic.Int32
	var updateBranchCalled atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/merge"):
			mergeCallCount.Add(1)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"sha": "abc123"})

		case strings.HasSuffix(r.URL.Path, "/update-branch"):
			updateBranchCalled.Add(1)
			w.WriteHeader(http.StatusAccepted)

		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := newClient(t, srv)
	err := client.MergeChangeProposal(context.Background(), "org", "repo", 42)

	// ASSERT-01: Merge returns no error
	require.NoError(t, err, "core merge path is broken")

	// ASSERT-02: update-branch was never called
	assert.Equal(t, int32(0), updateBranchCalled.Load(),
		"unnecessary branch update on successful merge")

	// ASSERT-03: Merge endpoint called exactly once
	assert.Equal(t, int32(1), mergeCallCount.Load(),
		"unnecessary retry on successful merge")
}

// TestMergeChangeProposal_409TriggersRetry validates the core retry-on-409
// logic: when the first merge attempt returns a 409 "Head branch is out of
// date" error, the function should call update-branch, then retry the merge.
// The second attempt succeeds.
//
// Requirement: REQ-001
// Priority: P0
func TestMergeChangeProposal_409TriggersRetry(t *testing.T) {
	// [test_id:TS-GH-2432-002]
	var mergeCallCount atomic.Int32
	var updateBranchCallCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/merge"):
			attempt := mergeCallCount.Add(1)
			if attempt == 1 {
				// First merge attempt: 409 conflict
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(map[string]string{
					"message": "Head branch is out of date",
				})
				return
			}
			// Subsequent attempts: success
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"sha": "def456"})

		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/update-branch"):
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

	client := newClient(t, srv)
	err := client.MergeChangeProposal(context.Background(), "org", "repo", 7)

	// ASSERT-01: Merge returns no error (retry succeeded)
	require.NoError(t, err, "retry-on-409 mechanism is broken")

	// ASSERT-02: update-branch was called exactly once
	assert.Equal(t, int32(1), updateBranchCallCount.Load(),
		"branch update not triggered on 409")

	// ASSERT-03: Merge endpoint was called exactly twice
	assert.Equal(t, int32(2), mergeCallCount.Load(),
		"incorrect number of merge attempts")
}

// TestMergeChangeProposal_Non409NotRetried validates that HTTP errors other
// than 409 (e.g., 422 "not mergeable") are returned immediately without
// triggering the retry loop or calling update-branch.
//
// Requirement: REQ-003
// Priority: P1
func TestMergeChangeProposal_Non409NotRetried(t *testing.T) {
	// [test_id:TS-GH-2432-003]
	var mergeCallCount atomic.Int32
	var updateBranchCalled atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/merge"):
			mergeCallCount.Add(1)
			w.WriteHeader(http.StatusUnprocessableEntity) // 422
			json.NewEncoder(w).Encode(map[string]string{
				"message": "Pull request is not mergeable",
			})

		case strings.HasSuffix(r.URL.Path, "/update-branch"):
			updateBranchCalled.Add(1)
			w.WriteHeader(http.StatusAccepted)

		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := newClient(t, srv)
	err := client.MergeChangeProposal(context.Background(), "org", "repo", 7)

	// ASSERT-01: Function returns an error
	require.Error(t, err, "non-retriable errors are being swallowed")

	// ASSERT-02: Error message contains original failure reason
	assert.Contains(t, err.Error(), "not mergeable",
		"error context lost during retry handling")

	// ASSERT-03: Merge endpoint called exactly once (no retry)
	assert.Equal(t, int32(1), mergeCallCount.Load(),
		"non-409 errors are being incorrectly retried")

	// ASSERT-04: update-branch was never called
	assert.Equal(t, int32(0), updateBranchCalled.Load(),
		"unnecessary branch update on non-409 error")
}

// TestMergeChangeProposal_ExhaustsRetries validates that when the GitHub API
// returns 409 on every merge attempt, the retry loop terminates after the
// maximum number of attempts and returns an error rather than looping
// indefinitely.
//
// Requirement: REQ-002
// Priority: P1
func TestMergeChangeProposal_ExhaustsRetries(t *testing.T) {
	// [test_id:TS-GH-2432-004]
	var mergeCallCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/merge"):
			mergeCallCount.Add(1)
			w.WriteHeader(http.StatusConflict) // 409
			json.NewEncoder(w).Encode(map[string]string{
				"message": "Head branch is out of date",
			})

		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/update-branch"):
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

	client := newClient(t, srv)
	err := client.MergeChangeProposal(context.Background(), "org", "repo", 7)

	// ASSERT-01: Function returns an error
	require.Error(t, err,
		"infinite retry loop — function never returns on persistent 409")

	// ASSERT-02: Merge was attempted more than once
	assert.Greater(t, mergeCallCount.Load(), int32(1),
		"retry logic not executing")

	// ASSERT-03: Error message references the PR number
	assert.Contains(t, err.Error(), fmt.Sprintf("#%d", 7),
		"error message unhelpful for debugging — should reference PR number")
}

// TestMergeChangeProposal_ContextCancelled validates that when a context is
// cancelled during the retry delay (between the 409 response and the next
// merge attempt), the function returns the context error promptly without
// hanging.
//
// Requirement: REQ-004
// Priority: P2
func TestMergeChangeProposal_ContextCancelled(t *testing.T) {
	// [test_id:TS-GH-2432-008]
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/merge"):
			w.WriteHeader(http.StatusConflict) // 409
			json.NewEncoder(w).Encode(map[string]string{
				"message": "Head branch is out of date",
			})

		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/update-branch"):
			w.WriteHeader(http.StatusAccepted)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	// Create context with a short timeout so it cancels during retry delay.
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	client := newClient(t, srv)
	err := client.MergeChangeProposal(ctx, "org", "repo", 7)

	// ASSERT-01: Function returns a context error
	require.Error(t, err,
		"retry loop ignores context cancellation — tests could hang")
	assert.True(t,
		errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled),
		"expected context error, got: %v", err)
}

// TestMergeChangeProposal_UpdateBranchFailsRetryProceeds validates that when
// the update-branch API call fails (returns an error), the function still
// proceeds to retry the merge. The update-branch failure should not abort
// the retry loop since the merge might succeed anyway.
//
// Requirement: REQ-001
// Priority: P2
func TestMergeChangeProposal_UpdateBranchFailsRetryProceeds(t *testing.T) {
	// [test_id:TS-GH-2432-009]
	var mergeCallCount atomic.Int32
	var updateBranchCallCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/merge"):
			attempt := mergeCallCount.Add(1)
			if attempt == 1 {
				w.WriteHeader(http.StatusConflict) // 409
				json.NewEncoder(w).Encode(map[string]string{
					"message": "Head branch is out of date",
				})
				return
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"sha": "ghi789"})

		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/update-branch"):
			updateBranchCallCount.Add(1)
			w.WriteHeader(http.StatusInternalServerError) // 500
			json.NewEncoder(w).Encode(map[string]string{
				"message": "Internal Server Error",
			})

		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := newClient(t, srv)
	err := client.MergeChangeProposal(context.Background(), "org", "repo", 7)

	// ASSERT-01: Function returns nil (merge eventually succeeded)
	require.NoError(t, err,
		"update-branch failure aborts retry loop — fragile retry logic")

	// ASSERT-02: update-branch was attempted
	assert.GreaterOrEqual(t, updateBranchCallCount.Load(), int32(1),
		"update-branch not being called at all")

	// ASSERT-03: Merge was retried after update-branch failure
	assert.Equal(t, int32(2), mergeCallCount.Load(),
		"merge retry skipped after update-branch failure")
}
