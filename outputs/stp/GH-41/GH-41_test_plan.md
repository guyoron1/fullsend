# My-Project Test Plan

## **Post Medium+ Findings as File-Level Comments When Line Is Outside Diff Hunk - Quality Engineering Plan**

### **Metadata & Tracking**

- **Enhancement(s):** [GH-41](https://github.com/guyoron1/fullsend/issues/41)
- **Feature Tracking:** [GH-41](https://github.com/guyoron1/fullsend/issues/41)
- **Epic Tracking:** GH-41 (standalone fix, mirror of upstream fullsend-ai/fullsend#2415)
- **QE Owner(s):** TBD
- **Owning SIG:** N/A
- **Participating SIGs:** None

**Document Conventions (if applicable):** N/A

### **Feature Overview**

This bug fix changes the review-comment posting logic in fullsend so that findings whose file is in the PR diff but whose line falls outside any diff hunk are posted as file-level comments instead of being silently dropped. Previously, these out-of-hunk findings were counted as "line-filtered" and omitted entirely, meaning reviewers could miss important findings. The fix modifies `findingsToReviewComments` in `internal/cli/postreview.go` to create file-level fallback comments (Line=0) that include the original line number in the body, and updates `CreatePullRequestReview` in `internal/forge/github/github.go` to set the GitHub API `subject_type: "file"` field when Line is 0.

---

### **I. Motivation and Requirements Review (QE Review Guidelines)**

This section documents the mandatory QE review process. The goal is to understand the feature's value,
technology, and testability before formal test planning.

#### **1. Requirement & User Story Review Checklist**

- [ ] **Review Requirements**
  - Reviewed the relevant requirements.
  - GH-41 describes a behavioral change: out-of-hunk findings should be posted as file-level comments rather than silently dropped. The issue body and PR diff clearly define the change scope.
- [ ] **Understand Value and Customer Use Cases**
  - Confirmed clear user stories and understood.
  - Understand the difference between community and product requirements.
  - **What is the value of the feature for customers**.
  - Ensured requirements contain relevant **customer use cases**.
  - Value: reviewers no longer lose visibility on findings that reference lines outside the changed diff region. This directly improves code review quality for all fullsend users.
- [ ] **Testability**
  - Confirmed requirements are **testable and unambiguous**.
  - The change is highly testable: `findingsToReviewComments` is a pure function that can be unit-tested with controlled inputs (findings + diffHunks map). The PR itself includes 4 new/updated test functions.
- [ ] **Acceptance Criteria**
  - Ensured acceptance criteria are **defined clearly** (clear user stories; product requirements clearly defined in Jira).
  - Acceptance criteria inferred from PR behavior: (1) out-of-hunk findings produce file-level comments with Line=0, (2) the comment body includes the original line number, (3) GitHub API payload includes `subject_type: "file"`.
- [ ] **Non-Functional Requirements (NFRs)**
  - Confirmed coverage for NFRs, including Performance, Security, Usability, Downtime, Connectivity, Monitoring (alerts/metrics), Scalability, Portability (e.g., cloud support), and Docs.
  - No significant NFR impact. The change adds a minor code path (file-level fallback) with negligible performance cost. No security, scalability, or monitoring changes.

#### **2. Known Limitations**

- File-level comments in GitHub do not display a line number annotation in the UI; the original line number is embedded in the comment body as a workaround.
- The `subject_type: "file"` field is GitHub-specific; other forge implementations (if any) would need their own file-level comment support.

#### **3. Technology and Design Review**

- [ ] **Developer Handoff/QE Kickoff**
  - A meeting where Dev/Arch walked QE through the design, architecture, and implementation details. **Critical for identifying untestable aspects early.**
  - PR #41 provides a clear diff. The change is localized to 4 files across 2 packages (`internal/cli`, `internal/forge`). LSP analysis confirms the call chain: `newPostReviewCmd` → `submitFormalReview` → `findingsToReviewComments`, and `submitFormalReview` → `CreatePullRequestReview`.
- [ ] **Technology Challenges**
  - Identified potential testing challenges related to the underlying technology.
  - No significant challenges. The core logic change is in a pure function (`findingsToReviewComments`) that is fully unit-testable. The GitHub API integration (`subject_type: "file"`) requires understanding of the GitHub Pull Request Review API.
- [ ] **Test Environment Needs**
  - Determined necessary **test environment setups and tools**.
  - Unit tests require only Go test infrastructure (go test + testify). End-to-end validation against the GitHub API requires a test repository with PR access.
- [ ] **API Extensions**
  - Reviewed new or modified APIs and their impact on testing.
  - `forge.ReviewComment.Line` field now has semantic meaning: Line=0 indicates a file-level comment. The GitHub implementation adds `SubjectType` to the internal `reviewComment` struct and conditionally sets `subject_type: "file"` in the API payload.
- [ ] **Topology Considerations**
  - Evaluated multi-cluster, network topology, and architectural impacts.
  - No topology impact. This is a client-side change in the CLI's review-posting flow.

### **II. Software Test Plan (STP)**

This STP serves as the **overall roadmap for testing**, detailing the scope, approach, resources, and schedule.

#### **1. Scope of Testing**

Testing covers the behavioral change in `findingsToReviewComments` (file-level fallback for out-of-hunk findings), the updated logging in `submitFormalReview`, and the `subject_type: "file"` handling in the GitHub forge implementation. The scope includes verifying that all severity levels fall back correctly, that in-hunk findings are unaffected, and that the GitHub API payload is correctly formed.

**Testing Goals**

- **P0:** Verify out-of-hunk findings are posted as file-level comments with correct body format (Line N prefix)
- **P0:** Verify in-hunk findings continue to be posted as line-level inline comments (no regression)
- **P0:** Verify GitHub API payload includes `subject_type: "file"` for Line=0 comments
- **P1:** Verify file-not-in-diff findings are still filtered out
- **P1:** Verify all severity levels (info through critical) fall back equally
- **P1:** Verify binary/empty-patch files bypass line filtering
- **P2:** Verify StepInfo log message reports fallback count

**Out of Scope (Testing Scope Exclusions)**

- [ ] **Sticky comment body rendering** — The sticky comment is unchanged by this PR; findings still appear in the body regardless of inline comment behavior.
  - *Rationale:* No code changes to sticky comment logic.
- [ ] **Non-GitHub forge implementations** — Only the GitHub forge is modified.
  - *Rationale:* Other forge backends (if any) are not affected by this change.
- [ ] **Review verdict logic** — The approve/request-changes decision is unaffected.
  - *Rationale:* Findings influence the verdict via the sticky comment, not inline comments.

#### **2. Test Strategy**

**Functional**

- [ ] **Functional Testing** — Validates that the feature works according to specified requirements and user stories
  - *Details:* Core testing of `findingsToReviewComments` with various input combinations: in-hunk findings, out-of-hunk findings, file-not-in-diff findings, binary files, mixed severities.
- [ ] **Automation Testing** — Confirms test automation plan is in place for CI and regression coverage (all tests are expected to be automated)
  - *Details:* All tests are Go unit tests using testify. The PR already includes 4 new/updated test functions that can be integrated into CI.
- [ ] **Regression Testing** — Verifies that new changes do not break existing functionality
  - *Details:* LSP analysis identified 22 callers of `submitFormalReview` and 19 references to `ReviewComment.Line`. Existing test coverage for these callers validates regression safety.

**Non-Functional**

- [ ] **Performance Testing** — Validates feature performance meets requirements (latency, throughput, resource usage)
  - *Details:* Not applicable. The file-level fallback adds negligible overhead (one `fmt.Sprintf` call per out-of-hunk finding).
- [ ] **Scale Testing** — Validates feature behavior under increased load and at production-like scale
  - *Details:* Not applicable for this bug fix scope.
- [ ] **Security Testing** — Verifies security requirements, RBAC, authentication, authorization, and vulnerability scanning
  - *Details:* Not applicable. No authentication or authorization changes.
- [ ] **Usability Testing** — Validates user experience and accessibility requirements
  - *Details:* Not applicable. The change improves visibility of findings (better UX) but requires no specific usability testing.
- [ ] **Monitoring** — Does the feature require metrics and/or alerts?
  - *Details:* Not applicable. The logging change (StepWarn→StepInfo) is informational only.

**Integration & Compatibility**

- [ ] **Compatibility Testing** — Ensures feature works across supported platforms, versions, and configurations
  - *Details:* The `subject_type: "file"` field is part of the GitHub Pull Request Review API. Compatibility with the GitHub API is the primary concern.
- [ ] **Upgrade Testing** — Validates upgrade paths from previous versions, data migration, and configuration preservation
  - *Details:* Not applicable. This is a behavioral change with no persistent state.
- [ ] **Dependencies** — Blocked by deliverables from other components/products
  - *Details:* No external dependencies. The change uses existing GitHub API capabilities.
- [ ] **Cross Integrations** — Does the feature affect other features or require testing by other teams?
  - *Details:* The `forge.ReviewComment` type is used by `internal/forge/fake.go` (test double). The fake client does not need changes since Line=0 is a valid value.

**Infrastructure**

- [ ] **Cloud Testing** — Does the feature require multi-cloud platform testing?
  - *Details:* Not applicable. This is a GitHub API client-side change.

#### **3. Test Environment**

- **Cluster Topology:** Not applicable (CLI tool, no cluster required)
- **Platform & Product Version(s):** Go 1.22+, fullsend current development branch
- **CPU Virtualization:** Not applicable
- **Compute Resources:** Standard CI runner
- **Special Hardware:** None
- **Storage:** None
- **Network:** GitHub API access required for E2E tests
- **Required Operators:** None
- **Platform:** Linux (CI), macOS/Windows (developer)
- **Special Configurations:** GitHub token with PR review permissions for E2E tests

#### **3.1. Testing Tools & Frameworks**

No new or special tools required. Standard Go test infrastructure (go test, testify) is used.

#### **4. Entry Criteria**

The following conditions must be met before testing can begin:

- [ ] Requirements and design documents are **approved and merged**
- [ ] Test environment can be **set up and configured** (see Section II.3 - Test Environment)
- [ ] PR #41 branch is available with all code changes
- [ ] Go test dependencies are installed (`go mod download`)

#### **5. Risks**

- [ ] **Timeline/Schedule**
  - Risk: Low risk. The change is small and well-scoped.
  - Mitigation: Tests are already written in the PR.
- [ ] **Test Coverage**
  - Risk: File-level comment rendering in GitHub UI may differ from expectations.
  - Mitigation: Verify with manual inspection of a real PR review containing file-level comments.
- [ ] **Test Environment**
  - Risk: E2E tests require GitHub API access which may be rate-limited.
  - Mitigation: Use a dedicated test repository with appropriate token scopes.
- [ ] **Untestable Aspects**
  - Risk: GitHub UI rendering of `subject_type: "file"` comments cannot be programmatically verified.
  - Mitigation: Manual verification during QE review.
- [ ] **Resource Constraints**
  - Risk: None identified.
  - Mitigation: N/A
- [ ] **Dependencies**
  - Risk: None identified. No external team dependencies.
  - Mitigation: N/A
- [ ] **Other**
  - Risk: None identified.
  - Mitigation: N/A

---

### **III. Test Scenarios & Traceability**

This section links requirements to test coverage, enabling reviewers to verify all requirements are tested.

#### **1. Requirements-to-Tests Mapping**

- **Requirement ID:** GH-41
  **Requirement Summary:** Out-of-hunk findings are posted as file-level comments instead of being silently dropped
  **Test Scenarios:**
  - Verify out-of-hunk finding posted as file-level comment
  - Verify finding with no file path is skipped
  - Verify file-level comments survive review re-submission
  **Tier:** Unit Tests / End-to-End
  **Priority:** P0

- **Requirement ID:**
  **Requirement Summary:** File-level fallback comments include the original line number in the body
  **Test Scenarios:**
  - Verify fallback body contains original line number
  - Verify body format matches '_Line N_ · description' pattern
  **Tier:** Unit Tests
  **Priority:** P0

- **Requirement ID:**
  **Requirement Summary:** In-hunk findings continue to be posted as line-level inline comments
  **Test Scenarios:**
  - Verify in-hunk finding retains correct line number
  - Verify in-hunk comment body unchanged from pre-change format
  **Tier:** Unit Tests
  **Priority:** P0

- **Requirement ID:**
  **Requirement Summary:** Findings referencing files not in the PR diff are still filtered out
  **Test Scenarios:**
  - Verify file-not-in-diff finding is omitted
  - Verify fileFiltered count incremented correctly
  **Tier:** Unit Tests
  **Priority:** P1

- **Requirement ID:**
  **Requirement Summary:** File-level fallback works for all severity levels
  **Test Scenarios:**
  - Verify all severities fall back to file-level equally
  - Verify case-insensitive severity handling in fallback
  **Tier:** Unit Tests
  **Priority:** P1

- **Requirement ID:**
  **Requirement Summary:** GitHub API receives subject_type:'file' for file-level comments
  **Test Scenarios:**
  - Verify API payload sets subject_type to file for Line=0
  - Verify API payload omits subject_type for Line>0
  - Verify GitHub API accepts file-level comment payload
  **Tier:** Unit Tests / End-to-End
  **Priority:** P0

- **Requirement ID:**
  **Requirement Summary:** Binary files and empty-patch files bypass line filtering
  **Test Scenarios:**
  - Verify binary file findings skip line-level filtering
  - Verify truncated-patch file findings posted without filtering
  **Tier:** Unit Tests
  **Priority:** P1

- **Requirement ID:**
  **Requirement Summary:** Fallback count is reported via StepInfo log message
  **Test Scenarios:**
  - Verify StepInfo log shows file-level fallback count
  - Verify no log emitted when fallback count is zero
  **Tier:** Unit Tests
  **Priority:** P2

---

### **IV. Sign-off and Approval**

This Software Test Plan requires approval from the following stakeholders:

* **Reviewers:**
  - [Name / @github-username]
  - [Name / @github-username]
* **Approvers:**
  - [Name / @github-username]
  - [Name / @github-username]
