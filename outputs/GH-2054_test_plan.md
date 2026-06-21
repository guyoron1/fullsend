# FullSend Test Plan

## **Review Agent Summary Comment Should Reflect Inline Findings and Verdict - Quality Engineering Plan**

### **Metadata & Tracking**

- **Enhancement(s):** [GH-2054](https://github.com/fullsend-ai/fullsend/issues/2054)
- **Feature Tracking:** [GH-2054](https://github.com/fullsend-ai/fullsend/issues/2054)
- **Epic Tracking:** GH-2054
- **QE Owner(s):** TBD
- **Owning SIG:** N/A
- **Participating SIGs:** N/A

**Document Conventions (if applicable):** N/A

### **Feature Overview**

The review agent's sticky summary comment can contradict its own formal review verdict and inline findings. Specifically, the summary says "No findings" while the formal review is `CHANGES_REQUESTED` with critical inline comments. This bug fix adds a consistency safety net in the `post-review` CLI command (`ensureBodyFindingsConsistency`) that detects when the review body omits significant findings despite a blocking verdict, and replaces the body with one synthesized from the structured findings array. The pr-review skill is also updated to instruct the agent to produce consistent output at the source.

---

### **I. Motivation and Requirements Review (QE Review Guidelines)**

This section documents the mandatory QE review process. The goal is to understand the feature's value,
technology, and testability before formal test planning.

#### **1. Requirement & User Story Review Checklist**

- [ ] **Review Requirements**
  - Reviewed the relevant requirements.
  - GH-2054 describes a clear, reproducible bug observed on PR #7193 in konflux-ci/konflux-ci. The issue body includes root cause hypothesis, proposed change, and validation criteria. Linked PR #2055 was closed (fragile regex approach); PR #2189 is the active fix.
- [ ] **Understand Value and Customer Use Cases**
  - Confirmed clear user stories and understood.
  - Understand the difference between community and product requirements.
  - **What is the value of the feature for customers**.
  - Ensured requirements contain relevant **customer use cases**.
  - The review agent runs across all orgs using the platform. A misleading "No findings" summary when critical issues exist could cause reviewers to merge code with unaddressed bugs. Fixing this ensures trust in the automated review process.
- [ ] **Testability**
  - Confirmed requirements are **testable and unambiguous**.
  - The core fix is in pure Go functions (`ensureBodyFindingsConsistency`, `synthesizeReviewBody`) that accept structured input and return deterministic output. All scenarios are directly unit-testable. PR #2189 includes 12 test cases covering the key paths.
- [ ] **Acceptance Criteria**
  - Ensured acceptance criteria are **defined clearly** (clear user stories; product requirements clearly defined in Jira).
  - Validation criteria from GH-2054: "On the next 5 review agent runs that submit CHANGES_REQUESTED with inline findings, verify that the summary PR comment lists those findings. The summary should never say 'No findings' when the verdict is CHANGES_REQUESTED and inline comments contain critical or high-severity issues."
- [ ] **Non-Functional Requirements (NFRs)**
  - Confirmed coverage for NFRs, including Performance, Security, Usability, Downtime, Connectivity, Monitoring (alerts/metrics), Scalability, Portability (e.g., cloud support), and Docs.
  - No significant NFR impact. The consistency check runs once per review post with negligible overhead (string operations on a small body). A warning is logged when the body is synthesized, providing observability.

#### **2. Known Limitations**

- The CLI safety net is a fallback — if the agent produces a contradictory body, the CLI patches it, but the root cause (agent-side logic) is addressed via a SKILL.md instruction update rather than a hard enforcement mechanism.
- Category-based consistency detection relies on finding hyphenated category tokens (e.g., "logic-error") in the body. If the agent uses a completely different phrasing that doesn't include the category token, the check may trigger unnecessarily and replace a valid body.
- The fix only covers `request-changes` and `reject` actions. A `comment` action with critical findings in the array but an inconsistent body will not be patched (by design — comment verdicts are non-blocking).

#### **3. Technology and Design Review**

- [ ] **Developer Handoff/QE Kickoff**
  - A meeting where Dev/Arch walked QE through the design, architecture, and implementation details. **Critical for identifying untestable aspects early.**
  - PR #2189 includes comprehensive code comments explaining the design decision to synthesize the full body (rather than regex-patch individual phrases as in the closed PR #2055). The approach is documented inline in `postreview.go`.
- [ ] **Technology Challenges**
  - Identified potential testing challenges related to the underlying technology.
  - No significant technology challenges. The fix uses standard Go string operations and integrates with the existing `ReviewResult` struct. Testing requires only the existing testify framework.
- [ ] **Test Environment Needs**
  - Determined necessary **test environment setups and tools**.
  - Unit tests run without external dependencies. Functional tests require the `fullsend post-review` CLI with a mocked or stubbed GitHub API (forge.Client interface).
- [ ] **API Extensions**
  - Reviewed new or modified APIs and their impact on testing.
  - No new APIs. The fix adds internal functions to the existing `post-review` CLI command. The `ReviewResult` struct is unchanged. The only external-facing change is that the sticky comment body may now be synthesized when a contradiction is detected.
- [ ] **Topology Considerations**
  - Evaluated multi-cluster, network topology, and architectural impacts.
  - N/A — this is a CLI-side fix that runs in the GitHub Actions sandbox. No cluster, network, or topology considerations.

### **II. Software Test Plan (STP)**

This STP serves as the **overall roadmap for testing**, detailing the scope, approach, resources, and schedule.

#### **1. Scope of Testing**

Testing covers the body-verdict consistency enforcement logic added to the `post-review` CLI command in `internal/cli/postreview.go`. The scope includes the `ensureBodyFindingsConsistency` function and its helper `synthesizeReviewBody`, the integration point where the consistency check runs in the `newPostReviewCmd` flow, and the SKILL.md update that instructs the agent to produce consistent output.

**Testing Goals**

- **P0:** Verify that `ensureBodyFindingsConsistency` correctly detects and replaces a contradictory body when action is `request-changes`/`reject` with critical/high findings that are not referenced in the body.
- **P0:** Verify that `synthesizeReviewBody` produces correctly formatted markdown with findings grouped by severity in the correct order (critical > high > medium > low > info).
- **P0:** Verify no-op behavior for all expected safe conditions (approve, comment, only low/medium findings, body already references categories, nil/empty inputs).
- **P1:** Verify the consistency check integrates correctly with the `post-review` CLI command flow (runs after `parseReviewResult`, before `sticky.Post`).
- **P1:** Verify the SKILL.md update provides clear guidance to prevent the agent from producing contradictory output.

**Out of Scope (Testing Scope Exclusions)**

- [ ] **GitHub API behavior** — The `forge.Client` interface and GitHub API interactions are tested elsewhere. This STP covers only the body consistency logic.
  - Rationale: Platform-level concern; tested by the forge package's own test suite.
- [ ] **Sticky comment posting mechanics** — The `sticky.Post` function and comment update logic are out of scope.
  - Rationale: Existing functionality with its own test coverage in `internal/sticky/`.
- [ ] **Agent-side review generation** — The pr-review skill's internal logic for generating the review body is not tested here.
  - Rationale: The SKILL.md update is a documentation change; agent behavior validation is an integration concern beyond this STP's scope.
- [ ] **Stale-head detection and formal review submission** — These are separate code paths in `postreview.go` unaffected by this change.
  - Rationale: No code changes in those paths.

#### **2. Test Strategy**

**Functional**

- [x] **Functional Testing** — Validates that the feature works according to specified requirements and user stories
  - *Details:* Core focus of this STP. Unit tests validate `ensureBodyFindingsConsistency` and `synthesizeReviewBody` with various input combinations. Functional tests validate the CLI integration point.
- [x] **Automation Testing** — Confirms test automation plan is in place for CI and regression coverage (all tests are expected to be automated)
  - *Details:* All tests are Go unit tests using testify, running in `go test ./internal/cli/...`. PR #2189 includes 12 automated test cases. No manual testing required.
- [x] **Regression Testing** — Verifies that new changes do not break existing functionality
  - *Details:* LSP call graph analysis confirms `ensureBodyFindingsConsistency` is called only from `newPostReviewCmd` (line 94). `reviewActionToEvent` is shared with `submitFormalReview` (line 300) — existing tests cover that path. The `sticky.Post` call at line 132 receives the patched body transparently.

**Non-Functional**

- [ ] **Performance Testing** — Validates feature performance meets requirements (latency, throughput, resource usage)
  - *Details:* N/A — the consistency check performs string operations on a small body (typically < 5KB). No measurable performance impact.
- [ ] **Scale Testing** — Validates feature behavior under increased load and at production-like scale
  - *Details:* N/A — the function processes a single `ReviewResult` per invocation with a bounded number of findings.
- [ ] **Security Testing** — Verifies security requirements, RBAC, authentication, authorization, and vulnerability scanning
  - *Details:* N/A — no new authentication, authorization, or user-facing input handling. The function operates on trusted internal data structures.
- [ ] **Usability Testing** — Validates user experience and accessibility requirements
  - *Details:* N/A — the synthesized body follows the same markdown format defined in the pr-review skill. No UX change for end users.
- [ ] **Monitoring** — Does the feature require metrics and/or alerts?
  - *Details:* A `StepWarn` log message is emitted when the body is synthesized. No new metrics or alerts are required.

**Integration & Compatibility**

- [ ] **Compatibility Testing** — Ensures feature works across supported platforms, versions, and configurations
  - *Details:* N/A — pure Go logic with no platform-specific dependencies.
- [ ] **Upgrade Testing** — Validates upgrade paths from previous versions, data migration, and configuration preservation
  - *Details:* N/A — no persistent state or configuration changes. The fix is stateless.
- [ ] **Dependencies** — Blocked by deliverables from other components/products
  - *Details:* No external dependencies. Uses only existing `ReviewResult` struct and `reviewActionToEvent` function.
- [ ] **Cross Integrations** — Does the feature affect other features or require testing by other teams?
  - *Details:* The pr-review skill's SKILL.md is updated with a body-verdict consistency instruction. This is a documentation-level change that affects agent behavior but requires no code-level cross-integration testing.

**Infrastructure**

- [ ] **Cloud Testing** — Does the feature require multi-cloud platform testing?
  - *Details:* N/A — CLI runs in GitHub Actions sandbox, no cloud-specific behavior.

#### **3. Test Environment**

- **Cluster Topology:** N/A — no cluster required (unit tests only)
- **Platform & Product Version(s):** FullSend 0.x on GitHub Actions
- **CPU Virtualization:** N/A
- **Compute Resources:** Standard CI runner (GitHub Actions)
- **Special Hardware:** None
- **Storage:** N/A
- **Network:** N/A
- **Required Operators:** None
- **Platform:** GitHub Actions (Linux runner)
- **Special Configurations:** None

#### **3.1. Testing Tools & Frameworks**

- **Test Framework:** Go testing + testify (standard — no new tools)
- **CI/CD:** GitHub Actions (standard — no new tools)
- **Other Tools:** None

#### **4. Entry Criteria**

The following conditions must be met before testing can begin:

- [ ] Requirements and design documents are **approved and merged**
- [ ] Test environment can be **set up and configured** (see Section II.3 - Test Environment)
- [ ] PR #2189 is rebased on current main and all existing tests pass
- [ ] `go test ./internal/cli/...` completes without errors in the CI environment

#### **5. Risks**

- [ ] **Timeline/Schedule**
  - Risk: Low risk — PR #2189 is already open with passing tests.
  - Mitigation: Tests are already implemented in the PR; merge when approved.
- [ ] **Test Coverage**
  - Risk: Category-based detection may have edge cases where a valid body is unnecessarily replaced (false positive) or an invalid body passes (false negative with non-standard category tokens).
  - Mitigation: 12 test cases cover the primary scenarios. Add fuzz testing for category matching if edge cases are discovered in production.
- [ ] **Test Environment**
  - Risk: Sandbox Go module cache permission errors (noted in PR #2055 and #2189) may prevent test execution.
  - Mitigation: Run authoritative test suite on CI runner where module cache is properly configured.
- [ ] **Untestable Aspects**
  - Risk: The SKILL.md instruction update's effectiveness depends on LLM behavior, which is non-deterministic and cannot be unit-tested.
  - Mitigation: The CLI safety net provides a hard guarantee regardless of agent behavior. Monitor via `StepWarn` log output.
- [ ] **Resource Constraints**
  - Risk: None — tests require only a standard CI runner.
  - Mitigation: N/A.
- [ ] **Dependencies**
  - Risk: None — no external dependencies for the consistency check logic.
  - Mitigation: N/A.
- [ ] **Other**
  - Risk: The closed PR #2055 used the same branch name (`agent/2054-review-summary-consistency`). PR #2189 force-pushed a new approach on the same branch. Verify no stale CI artifacts from the old approach.
  - Mitigation: Confirm PR #2189 CI runs are against the current commit, not cached results from #2055.

---

### **III. Test Scenarios & Traceability**

This section links requirements to test coverage, enabling reviewers to verify all requirements are tested.

#### **1. Requirements-to-Tests Mapping**

- **GH-2054** — Body-verdict consistency enforcement
  - *Scenario:* Verify body replaced when findings contradict summary
  - *Tier:* Unit Tests
  - *Priority:* P0
  - *Evidence:* `ensureBodyFindingsConsistency` (line 524) detects body/verdict mismatch and calls `synthesizeReviewBody` (line 560)

- — Body-verdict consistency enforcement (continued)
  - *Scenario:* Verify no-op for approve/comment actions
  - *Tier:* Unit Tests
  - *Priority:* P0
  - *Evidence:* `reviewActionToEvent` (line 529) gates on `REQUEST_CHANGES` event; approve/comment bypass the check

- — Body-verdict consistency enforcement (continued)
  - *Scenario:* Verify reject action triggers consistency check
  - *Tier:* Unit Tests
  - *Priority:* P1
  - *Evidence:* `reviewActionToEvent` maps "reject" to `REQUEST_CHANGES` (line 187), enabling the consistency check

- — Body-verdict consistency enforcement (continued)
  - *Scenario:* Verify error handling for nil/empty input
  - *Tier:* Unit Tests
  - *Priority:* P1
  - *Evidence:* Guard clauses at lines 525-526 return false for nil result or empty findings

- **GH-2054** — Synthesized body formatting
  - *Scenario:* Verify synthesized body groups findings by severity
  - *Tier:* Unit Tests
  - *Priority:* P0
  - *Evidence:* `synthesizeReviewBody` (line 568) uses severity order array `["critical", "high", "medium", "low", "info"]` (line 570)

- — Synthesized body formatting (continued)
  - *Scenario:* Verify findings include category, file location, and remediation
  - *Tier:* Unit Tests
  - *Priority:* P1
  - *Evidence:* `synthesizeReviewBody` renders `file:line` backtick format (lines 593-598) and remediation (lines 600-602)

- — Synthesized body formatting (continued)
  - *Scenario:* Verify findings without file locations render correctly
  - *Tier:* Unit Tests
  - *Priority:* P1
  - *Evidence:* File rendering is conditional on `f.File != ""` (line 593); findings without files skip the backtick block

- **GH-2054** — Category-based consistency detection
  - *Scenario:* Verify no-op when body already references finding categories
  - *Tier:* Unit Tests
  - *Priority:* P0
  - *Evidence:* Category check at lines 551-554 returns false when `bodyLower` contains the category token

- — Category-based consistency detection (continued)
  - *Scenario:* Verify case-insensitive category matching
  - *Tier:* Unit Tests
  - *Priority:* P1
  - *Evidence:* Both body and category are lowercased via `strings.ToLower` (lines 551, 553)

- — Category-based consistency detection (continued)
  - *Scenario:* Verify no-op for only low/medium severity findings
  - *Tier:* Unit Tests
  - *Priority:* P1
  - *Evidence:* Only "critical" and "high" severities are collected as significant (lines 536-539); low/medium findings alone produce an empty `significant` slice

- **GH-2054** — Post-review CLI integration
  - *Scenario:* Verify post-review command applies consistency check before posting
  - *Tier:* Functional
  - *Priority:* P0
  - *Evidence:* `ensureBodyFindingsConsistency` is called at line 94 in `newPostReviewCmd`, after `parseReviewResult` (line 85) and before `sticky.Post` (line 132). LSP incoming calls confirm this is the only production caller.

- — Post-review CLI integration (continued)
  - *Scenario:* Verify warning logged when body is synthesized
  - *Tier:* Functional
  - *Priority:* P1
  - *Evidence:* `printer.StepWarn` is called at line 95 when `patched` is true

- — Post-review CLI integration (continued)
  - *Scenario:* Verify consistency check integrates with sticky comment flow
  - *Tier:* Functional
  - *Priority:* P1
  - *Evidence:* The patched `parsed.Body` is passed to `sticky.Post` at line 132 and to `submitFormalReview` at line 137 (via `parsed.Action` and `parsed.Findings`)

- **GH-2054** — Agent-level body-verdict alignment
  - *Scenario:* Verify SKILL.md instructs findings inclusion for blocking verdicts
  - *Tier:* Functional
  - *Priority:* P1
  - *Evidence:* SKILL.md diff adds explicit instruction at lines 697-702: "When the action is `request-changes` or `reject`, the body MUST list the findings that drove that verdict."

- — Agent-level body-verdict alignment (continued)
  - *Scenario:* Verify end-to-end review with contradictory agent output
  - *Tier:* Functional
  - *Priority:* P1
  - *Evidence:* End-to-end flow: agent produces JSON with body="No findings" + action="request-changes" + critical findings → CLI detects contradiction → body replaced → sticky comment reflects findings

---

### **IV. Sign-off and Approval**

This Software Test Plan requires approval from the following stakeholders:

* **Reviewers:**
  - [TBD / @reviewer]
* **Approvers:**
  - [TBD / @approver]
