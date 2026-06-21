# FullSend Test Plan

## **Two-Pass Review Strategy for Large PRs - Quality Engineering Plan**

### **Metadata & Tracking**

- **Enhancement(s):** [GH-2096](https://github.com/fullsend-ai/fullsend/issues/2096)
- **Feature Tracking:** [GH-2096](https://github.com/fullsend-ai/fullsend/issues/2096)
- **Epic Tracking:** [GH-898](https://github.com/fullsend-ai/fullsend/issues/898) (parent incident: review agent missed security findings on large PR)
- **QE Owner(s):** @ben-alkov
- **Owning SIG:** N/A
- **Participating SIGs:** N/A

**Document Conventions (if applicable):** N/A

### **Feature Overview**

On large PRs (≥50 changed files), the review agent currently spreads attention uniformly across all files, causing security-critical files to compete with boilerplate for context window and reasoning budget. This feature adds a two-pass review strategy: a lightweight haiku-model triage sub-agent first classifies changed files as security-critical or standard using path patterns and diff content heuristics, then the security and correctness sub-agents receive prioritized context packages with security-critical files appearing first. PRs under the threshold continue with standard uniform-attention review. This addresses the incident documented in GH-898 where the review agent missed a fail-open security bug and role escalation across 9 review rounds on a 52-file PR.

---

### **I. Motivation and Requirements Review (QE Review Guidelines)**

This section documents the mandatory QE review process. The goal is to understand the feature's value,
technology, and testability before formal test planning.

#### **1. Requirement & User Story Review Checklist**

- [ ] **Review Requirements**
  - Reviewed the relevant requirements.
  - GH-2096 provides clear problem statement with concrete incident evidence (PR #792, 52 files, 4 missed findings including fail-open security bug).
  - Related issues GH-898 (parent incident), GH-990 (partial verification), GH-946 (schema cross-checking) reviewed for context.
- [ ] **Understand Value and Customer Use Cases**
  - Confirmed clear user stories and understood.
  - Understand the difference between community and product requirements.
  - **What is the value of the feature for customers**.
  - Ensured requirements contain relevant **customer use cases**.
  - Value: Prevents security-critical code changes from being diluted in large PR reviews. Ensures auth, token, OIDC, and permission changes receive dedicated review attention.
  - Use case: Engineering teams submitting large architectural PRs (30-50+ files) need assurance that security-sensitive files receive thorough automated review.
- [ ] **Testability**
  - Confirmed requirements are **testable and unambiguous**.
  - Testable via: file count threshold verification, classification output validation, context package ordering inspection, and fallback behavior testing.
  - The 50-file threshold provides a clear, measurable activation condition.
- [ ] **Acceptance Criteria**
  - Ensured acceptance criteria are **defined clearly** (clear user stories; product requirements clearly defined in Jira).
  - Acceptance criteria derived from issue body and triage summary: threshold activation, path pattern matching, content heuristic classification, structured JSON output, fallback on failure, bypass for small PRs.
- [ ] **Non-Functional Requirements (NFRs)**
  - Confirmed coverage for NFRs, including Performance, Security, Usability, Downtime, Connectivity, Monitoring (alerts/metrics), Scalability, Portability (e.g., cloud support), and Docs.
  - Performance: Triage pass uses haiku model for speed — classification should complete quickly without blocking the review pipeline.
  - Security: The feature itself is a security improvement — erring on the side of inclusion prevents false negatives.

#### **2. Known Limitations**

- The 50-file threshold is a starting point that may need tuning based on triage pass performance in production.
- The triage pass uses path patterns and diff summary heuristics only — it cannot perform deep semantic analysis of file contents.
- The feature only affects the `security` and `correctness` sub-agents. Other review dimensions (`intent-coherence`, `style-conventions`, `docs-currency`, `cross-repo-contracts`) receive standard context regardless of triage results.
- Classification is based on the first ~20 lines of each file's diff, which may miss security-relevant changes deeper in the diff.

#### **3. Technology and Design Review**

- [ ] **Developer Handoff/QE Kickoff**
  - A meeting where Dev/Arch walked QE through the design, architecture, and implementation details. **Critical for identifying untestable aspects early.**
  - PR #2303 provides detailed implementation: new step 3c-1 in the pr-review SKILL.md orchestrator and a new security-triage sub-agent definition. Changes are to skill definition markdown files embedded via `go:embed` in the scaffold package.
- [ ] **Technology Challenges**
  - Identified potential testing challenges related to the underlying technology.
  - The triage sub-agent is a haiku-model LLM classifier — its outputs are non-deterministic. Testing must validate structural correctness (JSON schema, file completeness) rather than exact classification decisions.
- [ ] **Test Environment Needs**
  - Determined necessary **test environment setups and tools**.
  - Requires GitHub Actions environment with access to the `gh` CLI and ability to spawn Agent sub-processes. Test PRs with controlled file counts needed for threshold testing.
- [ ] **API Extensions**
  - Reviewed new or modified APIs and their impact on testing.
  - No API changes. The feature modifies internal orchestration logic within the pr-review skill definition. The security-triage sub-agent introduces a new JSON output contract (`security_critical_files`, `standard_files`, `summary`).
- [ ] **Topology Considerations**
  - Evaluated multi-cluster, network topology, and architectural impacts.
  - No topology impact. The feature operates entirely within the review agent's GitHub Actions sandbox.

### **II. Software Test Plan (STP)**

This STP serves as the **overall roadmap for testing**, detailing the scope, approach, resources, and schedule.

#### **1. Scope of Testing**

Testing validates the two-pass review strategy for the pr-review skill orchestrator. The scope covers the triage pass activation threshold, security-critical file classification via path patterns and content heuristics, prioritized context package assembly for security and correctness sub-agents, fallback behavior on triage failure, and threshold bypass for small PRs. Testing targets the skill definition files (`SKILL.md` and `sub-agents/security-triage.md`) and their integration with the review orchestration pipeline.

**Testing Goals**

- **P0:** Verify the triage pass correctly activates for PRs at or above the 50-file threshold and is bypassed for PRs below it
- **P0:** Verify the security-triage sub-agent produces structurally valid JSON output with complete file classification
- **P0:** Verify security and correctness sub-agents receive prioritized context with security-critical files first
- **P1:** Verify graceful fallback to uniform attention on triage failure (timeout, malformed response, empty response)
- **P1:** Verify path pattern and content heuristic classification accuracy for known security-critical file types
- **P1:** Verify end-to-end two-pass review completes successfully on a large PR with security-critical files

**Out of Scope (Testing Scope Exclusions)**

- [ ] **LLM classification accuracy benchmarking** — The triage sub-agent uses a non-deterministic haiku model; classification accuracy is validated structurally (complete, biased toward inclusion) rather than by correctness of individual decisions.
- [ ] **Review quality improvement measurement** — Measuring whether the two-pass strategy catches more security issues than uniform attention requires production A/B testing beyond STP scope.
- [ ] **Performance benchmarking of haiku triage latency** — The haiku model selection is a design decision; latency is expected to be acceptable and is not a testable requirement.
- [ ] **Other review dimensions (style, docs, intent)** — These sub-agents are explicitly unaffected by the triage classification.
- [ ] **Scaffold embedding and deployment** — The `go:embed` mechanism for distributing skill files is existing infrastructure tested by scaffold package tests.

#### **2. Test Strategy**

**Functional**

- [ ] **Functional Testing** — Validates that the feature works according to specified requirements and user stories
  - *Details:* Core functional tests verify threshold activation/bypass, classification completeness, context prioritization ordering, and fallback behavior. Tests use controlled PR file lists and mock sub-agent responses.
- [ ] **Automation Testing** — Confirms test automation plan is in place for CI and regression coverage (all tests are expected to be automated)
  - *Details:* All tests will be automated in Go using the testify framework. Unit tests for classification logic; functional tests for orchestrator behavior.
- [ ] **Regression Testing** — Verifies that new changes do not break existing functionality
  - *Details:* Existing scaffold tests (`TestFullsendRepoFilesExist`, `TestReviewWorkflowContent`) verify the pr-review skill files are correctly embedded. The new sub-agent file must be included in scaffold content validation.

**Non-Functional**

- [ ] **Performance Testing** — Validates feature performance meets requirements (latency, throughput, resource usage)
  - *Details:* Not applicable. The triage pass adds one haiku-model call which is expected to be fast. No performance SLA defined.
- [ ] **Scale Testing** — Validates feature behavior under increased load and at production-like scale
  - *Details:* Edge case testing covers PRs where all files are security-critical (maximum triage output) and very large file lists.
- [ ] **Security Testing** — Verifies security requirements, RBAC, authentication, authorization, and vulnerability scanning
  - *Details:* The feature itself improves security review coverage. Testing verifies conservative classification bias (false positives acceptable, false negatives not).
- [ ] **Usability Testing** — Validates user experience and accessibility requirements
  - *Details:* Not applicable. The feature is internal to the review agent orchestration — no user-facing interface changes.
- [ ] **Monitoring** — Does the feature require metrics and/or alerts?
  - *Details:* The triage pass logs an info-level note on fallback. No new metrics or alerts required.

**Integration & Compatibility**

- [ ] **Compatibility Testing** — Ensures feature works across supported platforms, versions, and configurations
  - *Details:* The feature operates within the pr-review skill definition. Compatibility is ensured by existing scaffold embedding tests.
- [ ] **Upgrade Testing** — Validates upgrade paths from previous versions, data migration, and configuration preservation
  - *Details:* Not applicable. Skill definitions are stateless — no data migration required. Repos receiving updated scaffold content will get the new triage step automatically.
- [ ] **Dependencies** — Blocked by deliverables from other components/products
  - *Details:* Depends on the Agent tool supporting haiku model selection for sub-agent spawning (existing capability).
- [ ] **Cross Integrations** — Does the feature affect other features or require testing by other teams?
  - *Details:* The triage step runs before dimension sub-agent dispatch (step 4) and must not alter the dispatch behavior for non-security dimensions. Integration test TS-GH-2096-020 covers this.

**Infrastructure**

- [ ] **Cloud Testing** — Does the feature require multi-cloud platform testing?
  - *Details:* Not applicable. The feature runs within GitHub Actions sandboxes — no cloud-specific behavior.

#### **3. Test Environment**

- **Cluster Topology:** N/A (no cluster required — GitHub Actions runner)
- **Platform & Product Version(s):** FullSend 0.x on GitHub Actions
- **CPU Virtualization:** N/A
- **Compute Resources:** Standard GitHub Actions runner (ubuntu-latest)
- **Special Hardware:** None
- **Storage:** Standard runner disk
- **Network:** GitHub API access for PR data fetching
- **Required Operators:** None
- **Platform:** GitHub Actions with fullsend harness
- **Special Configurations:** Agent tool with haiku model support; `gh` CLI authenticated

#### **3.1. Testing Tools & Frameworks**

- **Test Framework:** Go testing with testify (standard — no new tools)
- **CI/CD:** GitHub Actions (standard)
- **Other Tools:** None — standard tooling only

#### **4. Entry Criteria**

The following conditions must be met before testing can begin:

- [ ] Requirements and design documents are **approved and merged**
- [ ] Test environment can be **set up and configured** (see Section II.3 - Test Environment)
- [ ] PR #2303 is merged and the updated skill definitions are available in the scaffold embed
- [ ] The `security-triage` sub-agent definition is correctly embedded by `go:embed` and accessible via `FullsendRepoFile()`

#### **5. Risks**

- [ ] **Timeline/Schedule**
  - Risk: The 50-file threshold may need tuning post-merge, requiring follow-up changes
  - Mitigation: Threshold is parameterized in the skill definition and can be adjusted without structural changes
- [ ] **Test Coverage**
  - Risk: Non-deterministic haiku model outputs make exact classification testing unreliable
  - Mitigation: Tests validate structural properties (completeness, JSON schema, file coverage) rather than exact classification decisions
- [ ] **Test Environment**
  - Risk: Testing the full two-pass flow requires spawning a real haiku sub-agent in CI
  - Mitigation: Unit and functional tests use mocked sub-agent responses; E2E test runs in dedicated GitHub Actions workflow
- [ ] **Untestable Aspects**
  - Risk: Cannot verify whether the two-pass strategy actually improves security finding rates without production data
  - Mitigation: Structural tests verify the mechanism works correctly; effectiveness measurement is out of scope (tracked separately)
- [ ] **Resource Constraints**
  - Risk: None identified
  - Mitigation: N/A
- [ ] **Dependencies**
  - Risk: Agent tool's haiku model support must correctly handle the Explore subagent_type for read-only classification
  - Mitigation: Existing Agent tool capability — validated by other sub-agents already using model overrides
- [ ] **Other**
  - Risk: The triage pass diff summary (first ~20 lines per file) may not capture security-relevant changes in large files
  - Mitigation: Conservative classification bias ensures ambiguous files are included as security-critical

---

### **III. Test Scenarios & Traceability**

This section links requirements to test coverage, enabling reviewers to verify all requirements are tested.

#### **1. Requirements-to-Tests Mapping**

- **Requirement ID:** GH-2096
- **Requirement Summary:** Triage pass activates for PRs with ≥50 changed files and classifies files as security-critical or standard
- **Test Scenarios:**
  - TS-GH-2096-001: Verify triage pass triggers for PR with ≥50 files
  - TS-GH-2096-002: Verify triage bypass for PR under 50 files
  - TS-GH-2096-003: Verify triage at exact 50-file boundary
- **Tier:** Functional
- **Priority:** P0

- **Requirement ID:**
- **Requirement Summary:** Security-prioritized context assembly for security and correctness sub-agents
- **Test Scenarios:**
  - TS-GH-2096-004: Verify security files appear first in security sub-agent context
  - TS-GH-2096-005: Verify security files appear first in correctness sub-agent context
  - TS-GH-2096-006: Verify other sub-agents receive standard context without prioritization
- **Tier:** Functional
- **Priority:** P0

- **Requirement ID:**
- **Requirement Summary:** Security-triage sub-agent returns structured JSON classification
- **Test Scenarios:**
  - TS-GH-2096-007: Verify JSON output contains all required fields (security_critical_files, standard_files, summary)
  - TS-GH-2096-008: Verify every changed file classified in exactly one category
  - TS-GH-2096-009: Verify error handling for malformed JSON response
- **Tier:** Unit Tests
- **Priority:** P0

- **Requirement ID:**
- **Requirement Summary:** File classification uses path patterns and content heuristics with conservative bias
- **Test Scenarios:**
  - TS-GH-2096-010: Verify security path patterns trigger classification (mint, auth, oidc, workflows)
  - TS-GH-2096-011: Verify content heuristics detect auth-related changes in diff summary
  - TS-GH-2096-012: Verify conservative classification errs on inclusion for ambiguous files
- **Tier:** Unit Tests
- **Priority:** P0

- **Requirement ID:**
- **Requirement Summary:** Graceful fallback on triage failure preserves uniform attention behavior
- **Test Scenarios:**
  - TS-GH-2096-013: Verify fallback treats all files as security-critical on sub-agent failure
  - TS-GH-2096-014: Verify fallback on triage timeout
  - TS-GH-2096-015: Verify fallback on empty triage response
- **Tier:** Functional
- **Priority:** P1

- **Requirement ID:**
- **Requirement Summary:** Edge cases in file classification produce correct behavior
- **Test Scenarios:**
  - TS-GH-2096-016: Verify behavior when all files are security-critical (full context for all)
  - TS-GH-2096-017: Verify behavior when no files match security patterns (standard review)
  - TS-GH-2096-018: Verify triage with workflow files containing permissions blocks
- **Tier:** Functional
- **Priority:** P1

- **Requirement ID:**
- **Requirement Summary:** Two-pass review integrates correctly with existing review orchestration
- **Test Scenarios:**
  - TS-GH-2096-019: Verify end-to-end two-pass review on large PR with security-critical files
  - TS-GH-2096-020: Verify triage does not affect dimension sub-agent dispatch order
- **Tier:** End-to-End
- **Priority:** P1

---

### **IV. Sign-off and Approval**

This Software Test Plan requires approval from the following stakeholders:

* **Reviewers:**
  - @ben-alkov
* **Approvers:**
  - @ben-alkov
