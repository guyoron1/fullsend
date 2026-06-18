package github

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Non-Retryable HTTP Error Tests

STP Reference: outputs/stp/GH-24/GH-24_test_plan.md
STD Reference: outputs/std/GH-24/GH-24_test_description.yaml
Jira: GH-24

Note: do() returns (*http.Response, nil) for non-retryable status codes.
The status code error is surfaced by higher-level methods (get/put/post)
via checkStatus(). These tests verify that no retry occurs by checking
callCount == 1.
*/

// TestNonRetryableStatusCodesReturnImmediately validates that do() returns
// the response without retrying for non-retryable client error codes
// (400, 401, 404, 422).
// Covers: TS-GH-24-007
func TestNonRetryableStatusCodesReturnImmediately(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
	}{
		// [test_id:TS-GH-24-007] Verify isRetryable returns false for 400, 401, 404, 422
		{"400 Bad Request", http.StatusBadRequest, `{"message": "Bad Request"}`},
		{"401 Unauthorized", http.StatusUnauthorized, `{"message": "Bad credentials"}`},
		{"404 Not Found", http.StatusNotFound, `{"message": "Not Found"}`},
		{"422 Unprocessable Entity", http.StatusUnprocessableEntity, `{"message": "Validation Failed"}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var callCount atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				callCount.Add(1)
				w.WriteHeader(tc.statusCode)
				fmt.Fprint(w, tc.body)
			}))
			defer srv.Close()

			client := newTestClient(t, srv)
			resp, err := client.do(context.Background(), http.MethodGet, "/repos/org/repo/pulls/1", nil)
			require.NoError(t, err, "do() should not return error for non-retryable status codes")
			require.NotNil(t, resp)
			resp.Body.Close()
			assert.Equal(t, tc.statusCode, resp.StatusCode)
			assert.Equal(t, int32(1), callCount.Load(), "expected no retry for %d", tc.statusCode)
		})
	}
}

// TestNonRetryableResponseBodyPreserved validates that the response body
// is preserved and accessible for non-retryable status codes.
func TestNonRetryableResponseBodyPreserved(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprint(w, `{"message": "Validation Failed", "errors": [{"resource": "PullRequest", "field": "title", "code": "missing_field"}]}`)
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	_, err := client.get(context.Background(), "/repos/org/repo/pulls")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Validation Failed")
}
