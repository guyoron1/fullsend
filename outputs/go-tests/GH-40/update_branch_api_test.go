package tests

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/fullsend-ai/fullsend/internal/forge"
	gh "github.com/fullsend-ai/fullsend/internal/forge/github"
)

/*
UpdatePullRequestBranch API Method and FakeClient Compliance Tests — Full Implementation

STP Reference: outputs/stp/GH-40/GH-40_test_plan.md
STD Reference: outputs/std/GH-40/GH-40_test_description.yaml
Jira: GH-40

These tests validate:
  1. The GitHub LiveClient.UpdatePullRequestBranch method correctly calls
     PUT /repos/{owner}/{repo}/pulls/{number}/update-branch and handles
     202 Accepted and error responses.
  2. The FakeClient test double satisfies the forge.Client interface
     including the UpdatePullRequestBranch method.
*/

var _ = Describe("[GH-40] UpdatePullRequestBranch API method", func() {
	const (
		owner    = "test-org"
		repo     = "test-repo"
		prNumber = 42
	)

	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	// TS-GH-40-006 — Successful branch update returns no error
	Context("Branch update on valid PR", Ordered, func() {
		var (
			server         *httptest.Server
			capturedMethod string
			capturedPath   string
		)

		BeforeEach(func() {
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedMethod = r.Method
				capturedPath = r.URL.Path
				w.WriteHeader(http.StatusAccepted) // 202 Accepted
			}))
		})

		AfterEach(func() {
			server.Close()
		})

		It("[test_id:TS-GH-40-006] should call GitHub PUT update-branch endpoint and return success", func() {
			client := gh.New("fake-token").WithBaseURL(server.URL)

			err := client.UpdatePullRequestBranch(ctx, owner, repo, prNumber)

			Expect(err).ToNot(HaveOccurred())
			Expect(capturedMethod).To(Equal(http.MethodPut))
			Expect(capturedPath).To(Equal(
				fmt.Sprintf("/repos/%s/%s/pulls/%d/update-branch", owner, repo, prNumber),
			))
		})
	})

	// TS-GH-40-007 — Non-success status code returns an error
	Context("Error handling for failed branch update", Ordered, func() {
		var server *httptest.Server

		BeforeEach(func() {
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusUnprocessableEntity) // 422
			}))
		})

		AfterEach(func() {
			server.Close()
		})

		It("[test_id:TS-GH-40-007] should return error when GitHub API returns non-success status", func() {
			client := gh.New("fake-token").WithBaseURL(server.URL)

			err := client.UpdatePullRequestBranch(ctx, owner, repo, prNumber)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("update pull request branch"))
		})
	})
})

var _ = Describe("[GH-40] FakeClient interface compliance", func() {
	// TS-GH-40-008 — FakeClient satisfies forge.Client including UpdatePullRequestBranch
	Context("FakeClient implements UpdatePullRequestBranch", func() {
		It("[test_id:TS-GH-40-008] should satisfy the forge.Client interface including UpdatePullRequestBranch", func() {
			// Compile-time interface satisfaction check.
			var _ forge.Client = &forge.FakeClient{}

			// Runtime verification that the method is callable and returns
			// no error when no error is injected.
			fakeClient := &forge.FakeClient{}
			ctx := context.Background()
			err := fakeClient.UpdatePullRequestBranch(ctx, "owner", "repo", 1)
			Expect(err).ToNot(HaveOccurred())

			// Verify error injection works for UpdatePullRequestBranch.
			fakeClient.Errors = map[string]error{
				"UpdatePullRequestBranch": fmt.Errorf("injected error"),
			}
			err = fakeClient.UpdatePullRequestBranch(ctx, "owner", "repo", 1)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("injected error"))
		})
	})
})
