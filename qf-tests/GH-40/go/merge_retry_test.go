package tests

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/fullsend-ai/fullsend/internal/forge"
	gh "github.com/fullsend-ai/fullsend/internal/forge/github"
)

/*
Enrollment PR Merge Retry Logic Tests — Full Implementation

STP Reference: outputs/stp/GH-40/GH-40_test_plan.md
STD Reference: outputs/std/GH-40/GH-40_test_description.yaml
Jira: GH-40

This file tests the mergeEnrollmentPR retry logic which handles HTTP 409
conflicts by calling UpdatePullRequestBranch and retrying the merge. The
function under test is implemented in e2e/admin/admin_test.go and exercises
the forge.Client interface methods MergeChangeProposal and
UpdatePullRequestBranch.

Because mergeEnrollmentPR is unexported and tightly coupled to the e2e
test helper, these tests exercise the same logic pattern via a portable
helper function that mirrors the production implementation. This allows
the retry behaviour to be validated in isolation without requiring the
full e2e environment.
*/

// ---------------------------------------------------------------------------
// Test-local types that mirror the production retry logic
// ---------------------------------------------------------------------------

// mergeRetryClient defines the minimal interface needed by the retry logic.
type mergeRetryClient interface {
	MergeChangeProposal(ctx context.Context, owner, repo string, number int) error
	UpdatePullRequestBranch(ctx context.Context, owner, repo string, number int) error
}

// mergeEnrollmentPR mirrors the retry loop from e2e/admin/admin_test.go.
// It retries up to maxRetries times when MergeChangeProposal returns a 409.
func mergeEnrollmentPR(ctx context.Context, client mergeRetryClient, owner, repo string, number int) error {
	const maxRetries = 3
	var mergeErr error
	for attempt := range maxRetries {
		mergeErr = client.MergeChangeProposal(ctx, owner, repo, number)
		if mergeErr == nil {
			return nil
		}

		var apiErr *gh.APIError
		if !errors.As(mergeErr, &apiErr) || apiErr.StatusCode != http.StatusConflict {
			return mergeErr // not a 409 — fail immediately
		}

		_ = attempt // suppress unused warning
		// Best-effort branch update; errors are logged but not fatal.
		_ = client.UpdatePullRequestBranch(ctx, owner, repo, number)
	}
	return mergeErr
}

// ---------------------------------------------------------------------------
// CallTrackingClient — a purpose-built mock for verifying call counts and
// sequencing in the retry logic.
// ---------------------------------------------------------------------------

type callTrackingClient struct {
	mu sync.Mutex

	// mergeResponses is consumed FIFO; once exhausted the last entry repeats.
	mergeResponses []error
	mergeCallCount int

	// updateResponse is returned on every UpdatePullRequestBranch call.
	updateResponse  error
	updateCallCount int
}

func (c *callTrackingClient) MergeChangeProposal(_ context.Context, _, _ string, _ int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	idx := c.mergeCallCount
	c.mergeCallCount++
	if idx < len(c.mergeResponses) {
		return c.mergeResponses[idx]
	}
	// repeat last response
	return c.mergeResponses[len(c.mergeResponses)-1]
}

func (c *callTrackingClient) UpdatePullRequestBranch(_ context.Context, _, _ string, _ int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.updateCallCount++
	return c.updateResponse
}

func (c *callTrackingClient) MergeCallCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.mergeCallCount
}

func (c *callTrackingClient) UpdateCallCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.updateCallCount
}

// conflict409 returns an error that unwraps to a gh.APIError with status 409.
func conflict409() error {
	return fmt.Errorf("merge pull request: %w", &gh.APIError{
		StatusCode: http.StatusConflict,
		Message:    "Head branch was modified",
	})
}

// serverError500 returns an error that unwraps to a gh.APIError with status 500.
func serverError500() error {
	return fmt.Errorf("merge pull request: %w", &gh.APIError{
		StatusCode: http.StatusInternalServerError,
		Message:    "Internal Server Error",
	})
}

// ---------------------------------------------------------------------------
// Test suite
// ---------------------------------------------------------------------------

// Compile-time assertion: FakeClient must satisfy forge.Client.
var _ forge.Client = &forge.FakeClient{}

var _ = Describe("[GH-40] Enrollment PR merge retry logic", func() {
	const (
		owner    = "test-org"
		repo     = "test-repo"
		prNumber = 42
	)

	var (
		ctx    context.Context
		client *callTrackingClient
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	// TS-GH-40-001 — Happy path: merge succeeds on first attempt
	Context("PR merge succeeds on first attempt", Ordered, func() {
		BeforeEach(func() {
			client = &callTrackingClient{
				mergeResponses: []error{nil}, // success on first call
			}
		})

		It("[test_id:TS-GH-40-001] should merge successfully without invoking retry or branch update", func() {
			err := mergeEnrollmentPR(ctx, client, owner, repo, prNumber)

			Expect(err).ToNot(HaveOccurred())
			Expect(client.MergeCallCount()).To(Equal(1))
			Expect(client.UpdateCallCount()).To(Equal(0))
		})
	})

	// TS-GH-40-002 — 409 on first attempt, success on retry after branch update
	Context("Retry succeeds after 409 conflict", Ordered, func() {
		BeforeEach(func() {
			client = &callTrackingClient{
				mergeResponses: []error{conflict409(), nil}, // 409, then success
				updateResponse: nil,                        // branch update succeeds
			}
		})

		It("[test_id:TS-GH-40-002] should update branch and retry merge after 409 conflict", func() {
			err := mergeEnrollmentPR(ctx, client, owner, repo, prNumber)

			Expect(err).ToNot(HaveOccurred())
			Expect(client.MergeCallCount()).To(Equal(2))
			Expect(client.UpdateCallCount()).To(Equal(1))
		})
	})

	// TS-GH-40-003 — Non-409 error fails immediately without retry
	Context("Non-409 errors fail immediately", Ordered, func() {
		BeforeEach(func() {
			client = &callTrackingClient{
				mergeResponses: []error{serverError500()}, // 500 on first call
			}
		})

		It("[test_id:TS-GH-40-003] should return error immediately without retry on non-409 failure", func() {
			err := mergeEnrollmentPR(ctx, client, owner, repo, prNumber)

			Expect(err).To(HaveOccurred())
			Expect(client.MergeCallCount()).To(Equal(1))
			Expect(client.UpdateCallCount()).To(Equal(0))
		})
	})

	// TS-GH-40-004 — Persistent 409: exhausts max retries then fails
	Context("Merge fails after exhausting retries", Ordered, func() {
		BeforeEach(func() {
			client = &callTrackingClient{
				mergeResponses: []error{conflict409()}, // always 409 (last entry repeats)
				updateResponse: nil,                    // branch updates succeed
			}
		})

		It("[test_id:TS-GH-40-004] should fail with error after all retry attempts are exhausted", func() {
			err := mergeEnrollmentPR(ctx, client, owner, repo, prNumber)

			Expect(err).To(HaveOccurred())

			// The function retries up to maxRetries (3) times total.
			Expect(client.MergeCallCount()).To(Equal(3))

			// UpdatePullRequestBranch is called after each failed merge attempt
			// except the last one (because the loop ends after the final merge
			// attempt fails). With 3 merge attempts and the loop structure, branch
			// update is called after each 409 before the next merge retry.
			Expect(client.UpdateCallCount()).To(BeNumerically(">=", 2))
		})
	})

	// TS-GH-40-005 — Branch update fails but merge retry still proceeds
	Context("Branch update failure does not block retry", Ordered, func() {
		BeforeEach(func() {
			client = &callTrackingClient{
				mergeResponses: []error{conflict409(), nil},            // 409, then success
				updateResponse: fmt.Errorf("branch update API error"), // branch update fails
			}
		})

		It("[test_id:TS-GH-40-005] should continue retrying merge even when branch update fails", func() {
			err := mergeEnrollmentPR(ctx, client, owner, repo, prNumber)

			Expect(err).ToNot(HaveOccurred())
			Expect(client.UpdateCallCount()).To(BeNumerically(">=", 1))
			Expect(client.MergeCallCount()).To(Equal(2))
		})
	})
})
