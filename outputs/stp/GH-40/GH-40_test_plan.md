# My-Project Test Plan

## **Retry Enrollment PR Merge on 409 With Branch Update - Quality Engineering Plan**

### **Metadata & Tracking**

- **Enhancement(s):** [GH-40](https://github.com/guyoron1/fullsend/pull/40)
- **Feature Tracking:** [GH-40](https://github.com/guyoron1/fullsend/pull/40)
- **Epic Tracking:** [upstream fullsend-ai/fullsend#2435](https://github.com/fullsend-ai/fullsend/pull/2435)
- **QE Owner(s):** TBD
- **Owning SIG:** N/A
- **Participating SIGs:** N/A

**Document Conventions (if applicable):** N/A

### **Feature Overview**

This change adds retry logic to the enrollment PR merge step in the fullsend admin e2e workflow. When an enrollment PR merge fails with HTTP 409 (branch out of date), the system now updates the PR branch by merging the base branch into it and retries the merge up to 3 times. A new `UpdatePullRequestBranch` method is added to the `forge.Client` interface with implementations in both `LiveClient` (calling GitHub's PUT `/pulls/{number}/update-branch` endpoint) and `FakeClient`. This prevents flaky enrollment failures caused by base branch advancement between PR creation and merge.

---

### **I. Motivation and Requirements Review (QE Review Guidelines)**

This section documents the mandatory QE review process. The goal is to understand the feature's value,
technology, and testability before formal test planning.

#### **1. Requirement & User Story Review Checklist**

- [ ] **Review Requirements**
  - Reviewed the relevant requirements.
  - PR description and upstream issue fullsend-ai/fullsend#2435 reviewed. The change addresses HTTP 409 "Head branch is out of date" errors during enrollment PR merge.
- [ ] **Understand Value and Customer Use Cases**
  - Confirmed clear user stories and understood.
  - Understand the difference between community and product requirements.
  - **What is the value of the feature for customers**.
  - Ensured requirements contain relevant **customer use cases**.
  - Value: eliminates flaky enrollment failures that require manual intervention when the base branch advances between PR creation and merge. Directly improves reliability of the `fullsend admin enroll` workflow.
- [ ] **Testability**
  - Confirmed requirements are **testable and unambiguous**.
  - The retry behavior is directly testable via the e2e enrollment test. The new `UpdatePullRequestBranch` API method can be unit tested independently. The `FakeClient` stub enables isolated testing without GitHub API calls.
- [ ] **Acceptance Criteria**
  - Ensured acceptance criteria are **defined clearly** (clear user stories; product requirements clearly defined in Jira).
  - AC derived from PR: (1) Merge retries up to 3 times on 409, (2) non-409 errors fail immediately, (3) branch is updated between retries, (4) branch update failure is logged but doesn't block retry.
- [ ] **Non-Functional Requirements (NFRs)**
  - Confirmed coverage for NFRs, including Performance, Security, Usability, Downtime, Connectivity, Monitoring (alerts/metrics), Scalability, Portability (e.g., cloud support), and Docs.
  - Retry introduces a 5-second sleep between attempts (max 15 seconds total delay). No security, scalability, or monitoring implications identified.

#### **2. Known Limitations**

- Retry is capped at 3 attempts. If the base branch continues to advance rapidly (e.g., multiple concurrent reconcile pushes), the merge may still fail after exhausting retries.
- The 5-second wait between retries is a fixed value and not configurable.
- `UpdatePullRequestBranch` relies on GitHub's async 202 response; there is no confirmation the branch update completed before the retry.

#### **3. Technology and Design Review**

- [ ] **Developer Handoff/QE Kickoff**
  - A meeting where Dev/Arch walked QE through the design, architecture, and implementation details. **Critical for identifying untestable aspects early.**
  - Change is self-contained: retry loop in `mergeEnrollmentPR` e2e helper and new `UpdatePullRequestBranch` method on `forge.Client` interface.
- [ ] **Technology Challenges**
  - Identified potential testing challenges related to the underlying technology.
  - Simulating a 409 in e2e requires the base branch to advance after PR creation but before merge. This is a timing-dependent condition that may be difficult to reproduce deterministically.
- [ ] **Test Environment Needs**
  - Determined necessary **test environment setups and tools**.
  - Standard e2e test environment with GitHub API access. No additional infrastructure required.
- [ ] **API Extensions**
  - Reviewed new or modified APIs and their impact on testing.
  - New method `UpdatePullRequestBranch(ctx, owner, repo, number)` added to `forge.Client` interface. Calls GitHub PUT `/repos/{owner}/{repo}/pulls/{number}/update-branch`. Expects HTTP 202 Accepted.
- [ ] **Topology Considerations**
  - Evaluated multi-cluster, network topology, and architectural impacts.
  - No topology impact. Change is limited to GitHub API interaction layer.

### **II. Software Test Plan (STP)**

This STP serves as the **overall roadmap for testing**, detailing the scope, approach, resources, and schedule.

#### **1. Scope of Testing**

Testing covers the new retry-on-409 logic in the enrollment PR merge flow, the new `UpdatePullRequestBranch` API method on the `forge.Client` interface, and the `FakeClient` interface compliance. The scope includes verifying correct retry behavior, error classification (409 vs non-409), branch update invocation, and graceful handling of branch update failures.

**Testing Goals**

- **P0:** Verify enrollment PR merge succeeds when a 409 conflict triggers the retry path (branch update + re-merge)
- **P0:** Verify the new `UpdatePullRequestBranch` method correctly calls the GitHub API
- **P1:** Verify non-409 errors cause immediate failure without retry
- **P1:** Verify merge fails gracefully after exhausting all retry attempts
- **P1:** Verify branch update failures are tolerated and do not block subsequent retry attempts

**Out of Scope (Testing Scope Exclusions)**

- [ ] **GitHub API availability and rate limiting** — Platform-level concern; GitHub API reliability is not within project testing scope
- [ ] **Concurrent enrollment across multiple repos** — Not addressed by this change; existing behavior unchanged
- [ ] **Base branch protection rules** — GitHub platform feature; branch protection enforcement is outside project scope

#### **2. Test Strategy**

**Functional**

- [ ] **Functional Testing** — Validates that the feature works according to specified requirements and user stories
  - *Details:* Verify retry logic triggers on 409, verify branch update is called, verify non-409 errors are not retried, verify max retry exhaustion behavior
- [ ] **Automation Testing** — Confirms test automation plan is in place for CI and regression coverage (all tests are expected to be automated)
  - *Details:* Changes are in the e2e test suite (`e2e/admin/admin_test.go`). Unit tests for `UpdatePullRequestBranch` use the existing `FakeClient` mock infrastructure
- [ ] **Regression Testing** — Verifies that new changes do not break existing functionality
  - *Details:* Existing enrollment e2e flow (Phase 2 in `TestAdminFlow`) exercises the modified `mergeEnrollmentPR` function. The retry logic is additive and backward-compatible

**Non-Functional**

- [ ] **Performance Testing** — Validates feature performance meets requirements (latency, throughput, resource usage)
  - *Details:* Not applicable. Retry adds at most 15 seconds (3 x 5s sleep) in the worst case, which is acceptable for an enrollment operation
- [ ] **Scale Testing** — Validates feature behavior under increased load and at production-like scale
  - *Details:* Not applicable. Enrollment is a one-time per-repo operation
- [ ] **Security Testing** — Verifies security requirements, RBAC, authentication, authorization, and vulnerability scanning
  - *Details:* Not applicable. No new authentication or authorization paths introduced; uses existing GitHub token
- [ ] **Usability Testing** — Validates user experience and accessibility requirements
  - *Details:* Not applicable. Change is in automated infrastructure, not user-facing UI
- [ ] **Monitoring** — Does the feature require metrics and/or alerts?
  - *Details:* Not applicable. Retry attempts are logged via `t.Logf` in tests; no production monitoring changes

**Integration & Compatibility**

- [ ] **Compatibility Testing** — Ensures feature works across supported platforms, versions, and configurations
  - *Details:* `UpdatePullRequestBranch` uses GitHub REST API v3; compatible with all GitHub Enterprise and github.com instances
- [ ] **Upgrade Testing** — Validates upgrade paths from previous versions, data migration, and configuration preservation
  - *Details:* Not applicable. No persistent state or configuration changes
- [ ] **Dependencies** — Blocked by deliverables from other components/products
  - *Details:* Depends on GitHub API endpoint `PUT /repos/{owner}/{repo}/pulls/{number}/update-branch` being available (GA endpoint)
- [ ] **Cross Integrations** — Does the feature affect other features or require testing by other teams?
  - *Details:* The `forge.Client` interface change requires all implementations to add `UpdatePullRequestBranch`. `FakeClient` is updated in this PR. No other forge implementations exist currently

**Infrastructure**

- [ ] **Cloud Testing** — Does the feature require multi-cloud platform testing?
  - *Details:* Not applicable. GitHub API interaction is cloud-agnostic

#### **3. Test Environment**

- **Cluster Topology:** N/A (no cluster required; tests run against GitHub API)
- **Platform & Product Version(s):** Go 1.22+, fullsend current development branch
- **CPU Virtualization:** N/A
- **Compute Resources:** Standard CI runner
- **Special Hardware:** None
- **Storage:** None
- **Network:** Outbound HTTPS access to GitHub API (api.github.com)
- **Required Operators:** None
- **Platform:** Linux (CI runner)
- **Special Configurations:** GitHub token with repo and pull request write permissions

#### **3.1. Testing Tools & Frameworks**

No new or special tools required. Standard Go testing with `testing` and `testify` packages.

#### **4. Entry Criteria**

The following conditions must be met before testing can begin:

- [ ] Requirements and design documents are **approved and merged**
- [ ] Test environment can be **set up and configured** (see Section II.3 - Test Environment)
- [ ] GitHub API token with pull request write and repo permissions is available
- [ ] Test organization and test repository are provisioned for e2e testing

#### **5. Risks**

- [ ] **Timeline/Schedule**
  - Risk: None identified. Change is small and self-contained.
  - Mitigation: N/A
- [ ] **Test Coverage**
  - Risk: 409 conflict condition is timing-dependent and may not occur naturally in e2e tests.
  - Mitigation: Unit test the retry logic with mocked 409 responses via `FakeClient`. The e2e test validates the happy path (merge succeeds, with or without retry).
- [ ] **Test Environment**
  - Risk: GitHub API rate limiting could affect test reliability.
  - Mitigation: Use dedicated test org with sufficient API quota. The retry adds at most 3 additional API calls per enrollment.
- [ ] **Untestable Aspects**
  - Risk: The 5-second sleep between retries makes deterministic timing assertions difficult.
  - Mitigation: Accept timing as non-deterministic; validate behavior (retry count, error classification) rather than timing.
- [ ] **Resource Constraints**
  - Risk: None identified.
  - Mitigation: N/A
- [ ] **Dependencies**
  - Risk: GitHub `PUT /pulls/{number}/update-branch` endpoint returns 202 Accepted asynchronously; branch update may not complete before retry.
  - Mitigation: The 5-second sleep provides a buffer. If insufficient, increase sleep or add a branch status check.
- [ ] **Other**
  - Risk: None identified.
  - Mitigation: N/A

---

### **III. Test Scenarios & Traceability**

This section links requirements to test coverage, enabling reviewers to verify all requirements are tested.

#### **1. Requirements-to-Tests Mapping**

- **Requirement ID:** GH-40
  **Requirement Summary:** Enrollment PR merge handles 409 conflict with retry and branch update
  **Test Scenarios:**
  - Verify PR merge succeeds on first attempt without retry
  - Verify retry succeeds after 409 conflict with branch update
  - Verify non-409 errors fail immediately without retry
  - Verify merge fails after exhausting maximum retries
  - Verify branch update failure does not block merge retry
  **Tier:** Functional
  **Priority:** P0

- **Requirement ID:**
  **Requirement Summary:** UpdatePullRequestBranch API method calls GitHub endpoint correctly
  **Test Scenarios:**
  - Verify branch update returns success on valid PR
  - Verify error handling for failed branch update request
  **Tier:** Unit Tests
  **Priority:** P0

- **Requirement ID:**
  **Requirement Summary:** Client interface implementations comply with new method
  **Test Scenarios:**
  - Verify FakeClient implements UpdatePullRequestBranch
  **Tier:** Unit Tests
  **Priority:** P1

---

### **IV. Sign-off and Approval**

This Software Test Plan requires approval from the following stakeholders:

* **Reviewers:**
  - [TBD / @reviewer]
* **Approvers:**
  - [TBD / @approver]
