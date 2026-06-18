package github

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

/*
Rate Limit Retry Tests (429, 403 Secondary)

STP Reference: outputs/stp/GH-24/GH-24_test_plan.md
STD Reference: outputs/std/GH-24/GH-24_test_description.yaml
Jira: GH-24
*/

// TestIsRetryable429TooManyRequests validates that isRetryable returns true
// for 429 Too Many Requests (GitHub primary rate limit).
// Covers: TS-GH-24-031
func TestIsRetryable429TooManyRequests(t *testing.T) {
	// [test_id:TS-GH-24-031] Verify isRetryable still returns true for 429
	resp := &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Body:       http.NoBody,
	}
	result, _ := isRetryable(resp)
	assert.True(t, result, "expected isRetryable to return true for 429")
}

// TestIsRetryable403WithRetryAfterHeader validates that isRetryable returns
// true for 403 with Retry-After header (GitHub secondary rate limit signal).
// Covers: TS-GH-24-032
func TestIsRetryable403WithRetryAfterHeader(t *testing.T) {
	// [test_id:TS-GH-24-032] Verify isRetryable returns true for 403 with Retry-After header
	resp := &http.Response{
		StatusCode: http.StatusForbidden,
		Header:     http.Header{"Retry-After": {"60"}},
		Body:       http.NoBody,
	}
	result, _ := isRetryable(resp)
	assert.True(t, result, "expected isRetryable to return true for 403 with Retry-After")
}

// TestIsRetryable403WithoutRetryAfterNotRetried validates that isRetryable
// returns false for 403 without any rate limit indicators.
func TestIsRetryable403WithoutRetryAfterNotRetried(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusForbidden,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader("Resource not accessible by personal access token")),
	}
	result, _ := isRetryable(resp)
	assert.False(t, result, "expected isRetryable to return false for 403 without rate limit")
}

// TestIsRetryable403SecondaryRateLimitInBody validates that isRetryable detects
// GitHub's secondary rate limit signal in the response body for 403 responses
// without a Retry-After header.
// Covers: TS-GH-24-033
func TestIsRetryable403SecondaryRateLimitInBody(t *testing.T) {
	// [test_id:TS-GH-24-033] Verify isRetryable detects secondary rate limit in body
	resp := &http.Response{
		StatusCode: http.StatusForbidden,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader(`{"message":"You have exceeded a secondary rate limit"}`)),
	}
	result, _ := isRetryable(resp)
	assert.True(t, result, "expected isRetryable to return true for 403 with secondary rate limit in body")
}
