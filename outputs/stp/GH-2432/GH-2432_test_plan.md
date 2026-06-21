# Test Plan

## **Retry PR Merge on 409 "Head Branch Out of Date" - Quality Engineering Plan**

### **Metadata & Tracking**

- **Enhancement(s):** [GH-2432](https://github.com/fullsend-ai/fullsend/issues/2432)
- **Feature Tracking:** [GH-2432](https://github.com/fullsend-ai/fullsend/issues/2432)
- **Epic Tracking:** N/A (standalone bug fix)
- **QE Owner(s):** TBD
- **Owning SIG:** N/A
- **Participating SIGs:** None

**Document Conventions (if applicable):** N/A

### **Feature Overview**

This fix addresses a flaky E2E test failure in `TestAdminInstallUninstall` where the enrollment PR merge step fails with a 409 "Head branch is out of date" error. The root cause is a race condition: the reconcile workflow pushes to the default branch between PR creation and the merge attempt, causing the enrollment PR's base to fall behind. The fix modifies `MergeChangeProposal` in the GitHub forge client to detect 409 errors, call the GitHub `update-branch` API to sync the PR branch, and retry the merge up to 3 times. Non-409 errors are returned immediately without retry.

---

### **I. Motivation and Requirements Review (QE Review Guidelines)**

This section documents the mandatory QE review process. The goal is to understand the feature's value,
technology, and testability before formal test planning.

#### **1. Requirement & User Story Review Checklist**

- [ ] **Review Requirements**
  - Reviewed the relevant requirements.
  - Issue GH-2432 describes the flaky 409 failure in E2E admin tests. Triage agent confirmed root cause: `MergeChangeProposal` does not handle 409 responses.
- [ ] **Understand Value and Customer Use Cases**
  - Confirmed clear user stories and understood.
  - Understand the difference between community and product requirements.
  - **What is the value of the feature for customers**.
  - Ensured requirements contain relevant **customer use cases**.
  - This fix prevents enrollment PR merges from failing intermittently due to base branch movement, improving CI/merge queue reliability. Users benefit from more stable onboarding workflows.
- [ ] **Testability**
  - Confirmed requirements are **testable and unambiguous**.
  - The retry logic is fully testable via HTTP mock servers (httptest). The 409 status code, update-branch call, and retry count are all observable in unit tests. E2E coverage exists in `TestAdminInstallUninstall`.
- [ ] **Acceptance Criteria**
  - Ensured acceptance criteria are **defined clearly** (clear user stories; product requirements clearly defined in Jira).
  - Acceptance criteria from issue: merge should succeed reliably; either update PR branch before merging or handle 409 by rebasing and retrying.
- [ ] **Non-Functional Requirements (NFRs)**
  - Confirmed coverage for NFRs, including Performance, Security, Usability, Downtime, Connectivity, Monitoring (alerts/metrics), Scalability, Portability (e.g., cloud support), and Docs.
  - The 3-second delay between retries is bounded (max 9 seconds total). Context cancellation is respected, preventing runaway retries. No security, scalability, or monitoring NFRs apply.

#### **2. Known Limitations**

- The retry mechanism adds up to 9 seconds of delay (3 retries x 3s wait) in the worst case before failing. This is acceptable for a merge operation but should not be used in latency-sensitive paths.
- The `update-branch` API is asynchronous on GitHub's side; the 3-second wait is a heuristic. Under heavy GitHub load, it may not be sufficient.
- PR #2434 (under review) takes a different approach than PR #2435 (merged): #2434 embeds the retry in `MergeChangeProposal` itself, while #2435 added an `UpdatePullRequestBranch` interface method called from the E2E test. Both approaches coexist.

#### **3. Technology and Design Review**

- [ ] **Developer Handoff/QE Kickoff**
  - A meeting where Dev/Arch walked QE through the design, architecture, and implementation details. **Critical for identifying untestable aspects early.**
  - Triage agent provided root cause analysis. Code change is self-contained in `MergeChangeProposal` with comprehensive unit tests in the PR.
- [ ] **Technology Challenges**
  - Identified potential testing challenges related to the underlying technology.
  - The race condition is non-deterministic in production. Unit tests simulate it deterministically via mock HTTP servers. E2E tests may still occasionally encounter the race if 3 retries are insufficient.
- [ ] **Test Environment Needs**
  - Determined necessary **test environment setups and tools**.
  - Unit tests: Go test runner with httptest (no external dependencies). E2E tests: GitHub org with repos for enrollment testing (existing halfsend infrastructure).
- [ ] **API Extensions**
  - Reviewed new or modified APIs and their impact on testing.
  - `MergeChangeProposal` signature unchanged. Internally now calls `PUT /repos/{owner}/{repo}/pulls/{number}/update-branch` on 409. No forge.Client interface changes in PR #2434.
- [ ] **Topology Considerations**
  - Evaluated multi-cluster, network topology, and architectural impacts.
  - N/A. Change is scoped to GitHub API interaction, no cluster topology impact.

---

### **II. Software Test Plan (STP)**

This STP serves as the **overall roadmap for testing**, detailing the scope, approach, resources, and schedule.

#### **1. Scope of Testing**

Testing validates the retry-on-409 logic added to `MergeChangeProposal` in the GitHub forge client. The scope includes: successful merge on first attempt (regression guard), 409 detection and branch-update retry, non-409 error passthrough, retry exhaustion, and context cancellation handling. E2E coverage validates that enrollment and uninstall PR merges succeed reliably under race conditions.

**Testing Goals**

**Functional Goals:**
- **P0:** Verify `MergeChangeProposal` retries merge after 409 by calling update-branch API and succeeds on subsequent attempt
- **P0:** Verify happy-path merge (no conflict) continues to work with no behavioral change
- **P1:** Verify non-409 errors (e.g., 422) are returned immediately without retry
- **P1:** Verify retry loop exhausts after 3 attempts with descriptive error message

**Quality Goals:**
- **P2:** Verify context cancellation during retry wait aborts promptly and returns context error

**Integration Goals:**
- **P1:** Verify E2E enrollment PR merge succeeds despite concurrent reconcile workflow activity
- **P2:** Verify E2E uninstall removal PR merge is resilient to the same 409 race

**Out of Scope (Testing Scope Exclusions)**

- [ ] GitHub API correctness (409 status code semantics, update-branch API behavior) -- *Rationale:* Platform-level; GitHub is responsible for their API contracts -- *PM/Lead Agreement:* N/A
- [ ] Kubernetes cluster behavior during enrollment -- *Rationale:* Orthogonal to the merge retry fix; tested by existing E2E infrastructure tests -- *PM/Lead Agreement:* N/A
- [ ] Performance benchmarking of retry delays -- *Rationale:* Fixed 3-second delay is a simple heuristic, not a performance-critical path -- *PM/Lead Agreement:* N/A

#### **2. Test Strategy**

**Functional**

- [x] **Functional Testing** -- Validates that the feature works according to specified requirements and user stories
  - *Details:* Unit tests cover all merge outcomes (success, 409-retry-success, 409-exhaustion, non-409 error). Each test uses httptest mock servers to simulate GitHub API responses deterministically.
- [x] **Automation Testing** -- Confirms test automation plan is in place for CI and regression coverage (all tests are expected to be automated)
  - *Details:* All unit tests are in `github_merge_test.go` and run in CI. E2E coverage via existing `TestAdminInstallUninstall` in `e2e/admin/admin_test.go`.
- [x] **Regression Testing** -- Verifies that new changes do not break existing functionality
  - *Details:* `TestMergeChangeProposal_Success` guards the happy-path. LSP analysis confirmed no callers outside e2e tests and the forge interface — change is backward-compatible.

**Non-Functional**

- [ ] **Performance Testing** -- Validates feature performance meets requirements (latency, throughput, resource usage)
  - *Details:* N/A. Retry adds at most 9 seconds to a merge operation, which is acceptable for CI workflows.
- [ ] **Scale Testing** -- Validates feature behavior under increased load and at production-like scale
  - *Details:* N/A. MergeChangeProposal is called at most twice per enrollment/uninstall flow.
- [ ] **Security Testing** -- Verifies security requirements, RBAC, authentication, authorization, and vulnerability scanning
  - *Details:* N/A. No new authentication paths or permission changes. Uses existing GitHub token.
- [ ] **Usability Testing** -- Validates user experience and accessibility requirements
  - *Details:* N/A. Internal forge client method, not user-facing.
- [ ] **Monitoring** -- Does the feature require metrics and/or alerts?
  - *Details:* N/A. No new metrics or alerts required for retry logic.

**Integration & Compatibility**

- [ ] **Compatibility Testing** -- Ensures feature works across supported platforms, versions, and configurations
  - *Details:* N/A. GitHub API v3 is stable; update-branch endpoint is generally available.
- [ ] **Upgrade Testing** -- Validates upgrade paths from previous versions, data migration, and configuration preservation
  - *Details:* N/A. No persistent state changes; behavioral change in a single function.
- [ ] **Dependencies** -- Blocked by deliverables from other components/products
  - *Details:* None. The fix depends only on the existing GitHub REST API.
- [ ] **Cross Integrations** -- Does the feature affect other features or require testing by other teams?
  - *Details:* The forge.Client interface is used throughout fullsend. LSP analysis confirmed MergeChangeProposal callers are limited to e2e tests — no cross-team impact.

**Infrastructure**

- [ ] **Cloud Testing** -- Does the feature require multi-cloud platform testing?
  - *Details:* N/A. GitHub API interaction is cloud-agnostic.

#### **3. Test Environment**

- **Cluster Topology:** N/A (no cluster required for unit tests)
- **Platform & Product Version(s):** Go 1.26+, fullsend current branch
- **CPU Virtualization:** N/A
- **Compute Resources:** Standard CI runner
- **Special Hardware:** N/A
- **Storage:** N/A
- **Network:** GitHub API access required for E2E tests
- **Required Operators:** N/A
- **Platform:** Linux (CI), macOS (local development)
- **Special Configurations:** E2E tests require `halfsend` GitHub org with test repos and valid `GH_TOKEN`

#### **3.1. Testing Tools & Frameworks**

- **Test Framework:** Standard (`testing` + `testify` — no new tools)
- **CI/CD:** Standard (existing GitHub Actions workflows)
- **Other Tools:** None

#### **4. Entry Criteria**

The following conditions must be met before testing can begin:

- [ ] Requirements and design documents are **approved and merged**
- [ ] Test environment can be **set up and configured** (see Section II.3 - Test Environment)
- [ ] PR #2434 is rebased on latest main and passes CI

#### **5. Risks**

- [ ] **Timeline/Schedule**
  - Risk: N/A — fix is small and self-contained
  - Mitigation: N/A
- [ ] **Test Coverage**
  - Risk: E2E race condition may not reproduce deterministically in CI
  - Mitigation: Unit tests provide deterministic coverage of all retry paths. E2E test acts as a smoke test for the overall flow.
- [ ] **Test Environment**
  - Risk: E2E tests depend on external GitHub API availability and halfsend org configuration
  - Mitigation: Unit tests are fully self-contained with httptest mocks. E2E environment is maintained by the team.
- [ ] **Untestable Aspects**
  - Risk: The exact timing of GitHub's asynchronous update-branch processing cannot be controlled
  - Mitigation: The 3-second delay is a conservative heuristic. If insufficient, the retry loop provides additional attempts.
- [ ] **Resource Constraints**
  - Risk: N/A
  - Mitigation: N/A
- [ ] **Dependencies**
  - Risk: PR #2435 (merged) and PR #2434 (open) take different approaches — potential conflict
  - Mitigation: Review both PRs for interaction. PR #2435 adds interface method + E2E retry; PR #2434 embeds retry in MergeChangeProposal itself.
- [ ] **Other**
  - Risk: N/A
  - Mitigation: N/A

---

### **III. Test Scenarios & Traceability**

This section links requirements to test coverage, enabling reviewers to verify all requirements are tested.

#### **1. Requirements-to-Tests Mapping**

- **[GH-2432]** -- PR merge retries on 409 "Head branch out of date" by updating PR branch
  - *Test Scenario:* Verify merge succeeds after 409 with branch update
  - *Tier:* [Functional]
  - *Priority:* P0

- **[GH-2432]** -- PR merge retries on 409 "Head branch out of date" by updating PR branch
  - *Test Scenario:* Verify update-branch called before retry
  - *Tier:* [Functional]
  - *Priority:* P0

- **[GH-2432]** -- PR merge retries on 409 "Head branch out of date" by updating PR branch
  - *Test Scenario:* Verify merge error for update-branch failure
  - *Tier:* [Functional]
  - *Priority:* P0

- **[GH-2432]** -- Non-409 merge errors are returned immediately without retry
  - *Test Scenario:* Verify 422 error returned without retry
  - *Tier:* [Functional]
  - *Priority:* P1

- **[GH-2432]** -- Non-409 merge errors are returned immediately without retry
  - *Test Scenario:* Verify update-branch not called on non-409
  - *Tier:* [Functional]
  - *Priority:* P1

- **[GH-2432]** -- Merge gives up after max retry attempts with descriptive error
  - *Test Scenario:* Verify exhaustion after 3 failed retries
  - *Tier:* [Functional]
  - *Priority:* P1

- **[GH-2432]** -- Merge gives up after max retry attempts with descriptive error
  - *Test Scenario:* Verify error message includes attempt count
  - *Tier:* [Functional]
  - *Priority:* P1

- **[GH-2432]** -- Context cancellation during retry wait is honored
  - *Test Scenario:* Verify cancelled context aborts retry
  - *Tier:* [Functional]
  - *Priority:* P2

- **[GH-2432]** -- Context cancellation during retry wait is honored
  - *Test Scenario:* Verify context error returned on cancellation
  - *Tier:* [Functional]
  - *Priority:* P2

- **[GH-2432]** -- Happy-path merge continues to work unchanged
  - *Test Scenario:* Verify successful merge on first attempt
  - *Tier:* [Functional]
  - *Priority:* P0

- **[GH-2432]** -- E2E enrollment PR merge succeeds despite concurrent base branch updates
  - *Test Scenario:* Verify enrollment install merges PR reliably
  - *Tier:* [End-to-End]
  - *Priority:* P1

- **[GH-2432]** -- E2E enrollment PR merge succeeds despite concurrent base branch updates
  - *Test Scenario:* Verify retry handles reconcile workflow race
  - *Tier:* [End-to-End]
  - *Priority:* P1

- **[GH-2432]** -- E2E uninstall PR merge is resilient to 409 race
  - *Test Scenario:* Verify uninstall merges removal PR reliably
  - *Tier:* [End-to-End]
  - *Priority:* P2

---

### **IV. Sign-off and Approval**

This Software Test Plan requires approval from the following stakeholders:

* **Reviewers:**
  - @ralphbean
  - [QE reviewer TBD]
* **Approvers:**
  - @ralphbean
  - [QE lead TBD]
