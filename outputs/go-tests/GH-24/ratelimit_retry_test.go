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
Jira: GH-24
*/

/*
Preconditions:
    - None (pure function test)

Steps:
    1. Create HTTP response with 429 status code
    2. Call isRetryable with the response

Expected:
    - isRetryable returns true for 429
*/
func TestIsRetryable429TooManyRequests(t *testing.T) {
	// [test_id:TS-GH-24-005] Verify 429 Too Many Requests triggers retry
	resp := &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Body:       http.NoBody,
	}
	result, _ := isRetryable(resp)
	assert.True(t, result, "expected isRetryable to return true for 429")
}

/*
Preconditions:
    - None (pure function test)

Steps:
    1. Create HTTP 403 response with "secondary rate limit" body
    2. Call isRetryable with response

Expected:
    - isRetryable returns true for 403 with secondary rate limit body
*/
func TestIsRetryable403SecondaryRateLimit(t *testing.T) {
	// [test_id:TS-GH-24-006] Verify 403 with secondary rate limit body triggers retry
	resp := &http.Response{
		StatusCode: http.StatusForbidden,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader("You have exceeded a secondary rate limit")),
	}
	result, _ := isRetryable(resp)
	assert.True(t, result, "expected isRetryable to return true for 403 with secondary rate limit body")
}

/*
Preconditions:
    - None (pure function test)

Steps:
    1. Create HTTP 403 response without rate limit indicators in body
    2. Call isRetryable with response

Expected:
    - isRetryable returns false for plain 403
*/
func TestIsRetryable403NoRateLimitIndicator(t *testing.T) {
	// [test_id:TS-GH-24-007] Verify 403 without rate limit indicators is not retried
	resp := &http.Response{
		StatusCode: http.StatusForbidden,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader("Resource not accessible by personal access token")),
	}
	result, _ := isRetryable(resp)
	assert.False(t, result, "expected isRetryable to return false for 403 without rate limit")
}
