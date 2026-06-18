package github

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
isRetryable 5xx Status Code Tests

STP Reference: outputs/stp/GH-24/GH-24_test_plan.md
STD Reference: outputs/std/GH-24/GH-24_test_description.yaml
Jira: GH-24
*/

// TestIsRetryableReturnsTrue5xx validates that isRetryable returns true for
// all 5xx status codes in the retryable range (500-504).
// Covers: TS-GH-24-001 through TS-GH-24-005
func TestIsRetryableReturnsTrue5xx(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		testID     string
	}{
		// [test_id:TS-GH-24-001] Verify isRetryable returns true for HTTP 500
		{"500 Internal Server Error", http.StatusInternalServerError, "TS-GH-24-001"},
		// [test_id:TS-GH-24-005] Verify isRetryable returns true for HTTP 501
		{"501 Not Implemented", 501, "TS-GH-24-005"},
		// [test_id:TS-GH-24-002] Verify isRetryable returns true for HTTP 502
		{"502 Bad Gateway", http.StatusBadGateway, "TS-GH-24-002"},
		// [test_id:TS-GH-24-003] Verify isRetryable returns true for HTTP 503
		{"503 Service Unavailable", http.StatusServiceUnavailable, "TS-GH-24-003"},
		// [test_id:TS-GH-24-004] Verify isRetryable returns true for HTTP 504
		{"504 Gateway Timeout", http.StatusGatewayTimeout, "TS-GH-24-004"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := &http.Response{
				StatusCode: tc.statusCode,
				Body:       http.NoBody,
			}
			result, _ := isRetryable(resp)
			assert.True(t, result, "[%s] expected isRetryable to return true for %d", tc.testID, tc.statusCode)
		})
	}
}

// TestIsRetryableDrainsBodyOn5xx verifies that isRetryable drains the response
// body when it encounters a 5xx status code. This prevents connection leaks
// from undrained response bodies.
// Covers: TS-GH-24-006
func TestIsRetryableDrainsBodyOn5xx(t *testing.T) {
	// [test_id:TS-GH-24-006] Verify isRetryable drains response body on 5xx
	bodyContent := "test-body-data-that-should-be-drained"
	body := io.NopCloser(strings.NewReader(bodyContent))

	resp := &http.Response{
		StatusCode: http.StatusBadGateway,
		Body:       body,
	}

	retryable, _ := isRetryable(resp)
	require.True(t, retryable, "expected isRetryable to return true for 502")

	// After isRetryable, the body should be fully drained.
	buf := make([]byte, 1)
	n, err := resp.Body.Read(buf)
	assert.Equal(t, 0, n, "expected 0 bytes remaining after body drain")
	assert.Equal(t, io.EOF, err, "expected io.EOF after body drain")
}

// TestIsRetryableReturnsFalseNonRetryable validates that isRetryable returns
// false for HTTP status codes that should NOT trigger retries.
// Covers: TS-GH-24-007
func TestIsRetryableReturnsFalseNonRetryable(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		// [test_id:TS-GH-24-007] Verify isRetryable returns false for 400, 401, 404, 422
		{"400 Bad Request", http.StatusBadRequest},
		{"401 Unauthorized", http.StatusUnauthorized},
		{"404 Not Found", http.StatusNotFound},
		{"422 Unprocessable Entity", http.StatusUnprocessableEntity},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := &http.Response{
				StatusCode: tc.statusCode,
				Header:     http.Header{},
				Body:       io.NopCloser(strings.NewReader("")),
			}
			result, _ := isRetryable(resp)
			assert.False(t, result, "expected isRetryable to return false for %d", tc.statusCode)
		})
	}
}
