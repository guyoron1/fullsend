package github

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
do() 5xx Retry Behavior Tests

STP Reference: outputs/stp/GH-24/GH-24_test_plan.md
STD Reference: outputs/std/GH-24/GH-24_test_description.yaml
Jira: GH-24
*/

// TestDoRetries502AndSucceeds validates that do() retries a 502 Bad Gateway
// and succeeds when the subsequent attempt returns 200.
// Covers: TS-GH-24-008
func TestDoRetries502AndSucceeds(t *testing.T) {
	// [test_id:TS-GH-24-008] Verify do() retries and succeeds after transient 502
	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		if n == 1 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"sha": "abc123"}`)
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	resp, err := client.do(context.Background(), http.MethodGet, "/repos/org/repo/pulls/1", nil)
	require.NoError(t, err)
	require.NotNil(t, resp)
	resp.Body.Close()
	assert.Equal(t, int32(2), callCount.Load(), "expected exactly 2 HTTP calls (1 fail + 1 success)")
}

// TestDoRetries503AndSucceeds validates that do() retries after receiving 503
// Service Unavailable and succeeds on a subsequent attempt returning 200.
// Covers: TS-GH-24-009
func TestDoRetries503AndSucceeds(t *testing.T) {
	// [test_id:TS-GH-24-009] Verify do() retries and succeeds after transient 503
	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		if n == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"ok": true}`)
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	resp, err := client.do(context.Background(), http.MethodGet, "/repos/org/repo/contents/file", nil)
	require.NoError(t, err)
	require.NotNil(t, resp)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, int32(2), callCount.Load(), "expected exactly 2 HTTP calls")
}

// TestDoExhaustsRetriesOnPersistent500 validates that do() exhausts all retry
// attempts when the server persistently returns 500 and returns an error.
// Covers: TS-GH-24-010
func TestDoExhaustsRetriesOnPersistent500(t *testing.T) {
	// [test_id:TS-GH-24-010] Verify do() exhausts retries and returns error after persistent 500
	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	_, err := client.do(context.Background(), http.MethodGet, "/repos/org/repo/pulls/1", nil)
	require.Error(t, err)
	// maxRetries=3, so exactly 3 attempts
	assert.Equal(t, int32(maxRetries), callCount.Load(), "expected maxRetries total calls")
}

// TestDoRespectsContextCancellation validates that do() stops retrying and
// returns a context error when the context is cancelled during backoff.
// Covers: TS-GH-24-012
func TestDoRespectsContextCancellation(t *testing.T) {
	// [test_id:TS-GH-24-012] Verify do() respects context cancellation during retry backoff
	var callCount atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		w.WriteHeader(http.StatusBadGateway)
		// Cancel context after first request so do() is cancelled during backoff
		if n == 1 {
			cancel()
		}
	}))
	defer srv.Close()
	defer cancel() // idempotent

	client := newTestClient(t, srv)
	_, err := client.do(ctx, http.MethodGet, "/repos/org/repo/contents/file.txt", nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled, "expected context.Canceled error")
	// Should have made 1 request, then been cancelled during backoff
	assert.LessOrEqual(t, callCount.Load(), int32(2),
		"should not make many requests after cancellation")
}

// TestDoBackoffHonorsRetryAfterFor429 validates that the backoff delay for
// rate-limited requests (429) respects the Retry-After header value.
// Covers: TS-GH-24-034
func TestDoBackoffHonorsRetryAfterFor429(t *testing.T) {
	// [test_id:TS-GH-24-034] Verify rate limit backoff timing uses Retry-After
	var callCount atomic.Int32
	var timestamps []time.Time

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timestamps = append(timestamps, time.Now())
		n := callCount.Add(1)
		if n == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	resp, err := client.do(context.Background(), http.MethodGet, "/repos/org/repo/pulls/1", nil)
	require.NoError(t, err)
	resp.Body.Close()

	require.Len(t, timestamps, 2, "expected exactly 2 requests")
	elapsed := timestamps[1].Sub(timestamps[0])
	// Retry-After: 1 means at least 1 second backoff
	assert.GreaterOrEqual(t, elapsed, 900*time.Millisecond,
		"backoff should respect Retry-After header (>= ~1s)")
}
