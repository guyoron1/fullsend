# Test Plan

## **Cross-File Verification for Review Agent Findings - Quality Engineering Plan**

### **Metadata & Tracking**

- **Enhancement(s):** [GH-70](https://github.com/guyoron1/fullsend/pull/70)
- **Feature Tracking:** [GH-1835](https://github.com/guyoron1/fullsend/issues/1835)
- **Epic Tracking:** [GH-1835](https://github.com/guyoron1/fullsend/issues/1835)
- **QE Owner(s):** QualityFlow (auto-generated)
- **Owning SIG:** N/A
- **Participating SIGs:** N/A

**Document Conventions (if applicable):** Test IDs follow the format `TS-GH-1835-NNN`. Tier classification uses Functional only (no Unit Test or End-to-End scenarios identified).

### **Feature Overview**

This fix addresses GH-1835, an incident where the review agent asserted file contents in findings without having read the files. The change adds "Cross-file verification" instructions to two skill files: the code-review `SKILL.md` (Steps 2 and 4) and the pr-review correctness sub-agent definition (`correctness.md`). Both files are embedded in the scaffold binary via Go's `embed.FS` and deployed to enrolled repositories through the layered content system. The fix ensures agents must read files before asserting their contents and provides graceful degradation language when files are unreadable.

---

### **I. Motivation and Requirements Review (QE Review Guidelines)**

This section documents the mandatory QE review process. The goal is to understand the feature's value,
technology, and testability before formal test planning.

#### **1. Requirement & User Story Review Checklist**

- [ ] **Review Requirements**
  - Reviewed the relevant requirements.
  - GH-1835 describes an incident where the review agent produced findings asserting file contents it never read. The fix adds cross-file verification mandates to skill definitions.
- [ ] **Understand Value and Customer Use Cases**
  - Confirmed clear user stories and understood.
  - Understand the difference between community and product requirements.
  - **What is the value of the feature for customers**.
  - Ensured requirements contain relevant **customer use cases**.
  - Prevents false-positive review findings that erode developer trust in AI-assisted code review. Users benefit from accurate, evidence-backed review feedback.
- [ ] **Testability**
  - Confirmed requirements are **testable and unambiguous**.
  - Changes are to embedded markdown skill files with specific, searchable phrases. Testable via substring assertions on `FullsendRepoFile()` output.
- [ ] **Acceptance Criteria**
  - Ensured acceptance criteria are **defined clearly** (clear user stories; product requirements clearly defined in Jira).
  - SKILL.md must contain "Cross-file verification" instruction and "Cross-file finding self-check" gate. correctness.md must contain "Cross-file verification" section with MUST-read mandate.
- [ ] **Non-Functional Requirements (NFRs)**
  - Confirmed coverage for NFRs, including Performance, Security, Usability, Downtime, Connectivity, Monitoring (alerts/metrics), Scalability, Portability (e.g., cloud support), and Docs.
  - No performance/scalability NFRs — change is to static embedded content. Security NFR: prevents agents from asserting unverified content (agent safety improvement).

#### **2. Known Limitations**

- Cross-file verification instructions are advisory — they guide LLM behavior but cannot mechanistically enforce file reads at runtime. Compliance depends on the model following the instructions.
- The fix applies only to the code-review skill and correctness sub-agent. Other sub-agents (challenger, security, intent-coherence, style-conventions, docs-currency, cross-repo-contracts) are not modified by this PR.

#### **3. Technology and Design Review**

- [ ] **Developer Handoff/QE Kickoff**
  - A meeting where Dev/Arch walked QE through the design, architecture, and implementation details. **Critical for identifying untestable aspects early.**
  - Mirror of upstream fullsend-ai/fullsend#2443. Design is straightforward: add instructional text to two embedded skill files.
- [ ] **Technology Challenges**
  - Identified potential testing challenges related to the underlying technology.
  - Skill files are embedded via Go's `embed.FS` in the `internal/scaffold` package. Tests must use `FullsendRepoFile()` and `WalkFullsendRepoAll()` to access embedded content, not filesystem reads.
- [ ] **Test Environment Needs**
  - Determined necessary **test environment setups and tools**.
  - Standard Go test environment with `go test`. No cluster or external services required. Tests run against the compiled scaffold binary's embedded filesystem.
- [ ] **API Extensions**
  - Reviewed new or modified APIs and their impact on testing.
  - No API changes. The modification is to embedded skill file content only.
- [ ] **Topology Considerations**
  - Evaluated multi-cluster, network topology, and architectural impacts.
  - No topology impact. Skill files are deployed identically to all enrolled repositories via the scaffold layered content system.

### **II. Software Test Plan (STP)**

This STP serves as the **overall roadmap for testing**, detailing the scope, approach, resources, and schedule.

#### **1. Scope of Testing**

Testing validates that the cross-file verification instructions are correctly present in both skill files (SKILL.md and correctness.md), that the instructions use consistent language, and that the modified files are properly embedded in the scaffold binary and deployable via layered content.

**Testing Goals**

- **P0:** Verify SKILL.md contains cross-file verification instruction (Step 2) and self-check gate (Step 4) with correct mandate language
- **P0:** Verify correctness.md contains cross-file verification section with MUST-read mandate and unverified-content prohibition
- **P1:** Verify consistent graceful degradation language across both skill files
- **P1:** Verify modified files are embedded in scaffold and present in layered content deployment

**Out of Scope (Testing Scope Exclusions)**

- [ ] **Runtime LLM compliance verification** — Cannot test whether the LLM actually follows cross-file verification instructions during a live review; instructions are advisory.
- [ ] **Other sub-agent definitions** — Only correctness.md is modified; challenger, security, intent-coherence, style-conventions, docs-currency, and cross-repo-contracts sub-agents are unaffected.
- [ ] **End-to-end review workflow** — This STP covers content presence and deployment, not full review agent execution.
- [ ] **Upstream content parity verification** — This fork mirrors upstream fullsend-ai/fullsend#2443; content correctness is validated by upstream CI.

#### **2. Test Strategy**

**Functional**

- [ ] **Functional Testing** — Validates that the feature works according to specified requirements and user stories
  - *Details:* Applicable. Verify cross-file verification text is present in both skill files with correct phrasing.
- [ ] **Automation Testing** — Confirms test automation plan is in place for CI and regression coverage (all tests are expected to be automated)
  - *Details:* Applicable. All tests are Go unit tests using `testing` + `testify`, runnable via `go test ./qf-tests/GH-1835/go/...`.
- [ ] **Regression Testing** — Verifies that new changes do not break existing functionality
  - *Details:* Applicable. LSP analysis shows `FullsendRepoFile` has 72 references across 11 files. Existing scaffold tests in `internal/scaffold/scaffold_test.go` provide baseline regression coverage.

**Non-Functional**

- [ ] **Performance Testing** — Validates feature performance meets requirements (latency, throughput, resource usage)
  - *Details:* Not applicable. Changes are to static embedded content with no runtime performance impact.
- [ ] **Scale Testing** — Validates feature behavior under increased load and at production-like scale
  - *Details:* Not applicable. No scaling dimension for embedded markdown content.
- [ ] **Security Testing** — Verifies security requirements, RBAC, authentication, authorization, and vulnerability scanning
  - *Details:* Not directly applicable. The fix itself is a security-adjacent improvement (prevents agents from asserting unverified content).
- [ ] **Usability Testing** — Validates user experience and accessibility requirements
  - *Details:* Not applicable. Changes are to agent instruction files, not user-facing UI.
- [ ] **Monitoring** — Does the feature require metrics and/or alerts?
  - *Details:* Not applicable. No new metrics or alerts.

**Integration & Compatibility**

- [ ] **Compatibility Testing** — Ensures feature works across supported platforms, versions, and configurations
  - *Details:* Not applicable. Embedded content is platform-independent.
- [ ] **Upgrade Testing** — Validates upgrade paths from previous versions, data migration, and configuration preservation
  - *Details:* Not applicable. New skill content is additive; no migration needed.
- [ ] **Dependencies** — Blocked by deliverables from other components/products
  - *Details:* None. Self-contained change to embedded files.
- [ ] **Cross Integrations** — Does the feature affect other features or require testing by other teams?
  - *Details:* Layered content deployment affects all enrolled repos. `WalkLayeredContent` (10 references across 3 files) ensures updated files reach enrolled repositories.

**Infrastructure**

- [ ] **Cloud Testing** — Does the feature require multi-cloud platform testing?
  - *Details:* Not applicable. Embedded content is cloud-agnostic.

#### **3. Test Environment**

- **Cluster Topology:** None required (unit tests run locally)
- **Platform & Product Version(s):** Go 1.26.0 (per go.mod)
- **CPU Virtualization:** N/A
- **Compute Resources:** Standard CI runner
- **Special Hardware:** None
- **Storage:** N/A
- **Network:** N/A
- **Required Operators:** None
- **Platform:** Linux (CI runner)
- **Special Configurations:** None — tests use the scaffold `embed.FS` compiled into the test binary

#### **3.1. Testing Tools & Frameworks**

- None — uses standard project test infrastructure (`testing` + `testify`, GitHub Actions).

#### **4. Entry Criteria**

The following conditions must be met before testing can begin:

- [ ] Requirements and design documents are **approved and merged**
- [ ] Test environment can be **set up and configured** (see Section II.3 - Test Environment)
- [ ] Cross-file verification text is present in both `skills/code-review/SKILL.md` and `skills/pr-review/sub-agents/correctness.md` source files
- [ ] Scaffold binary compiles successfully with updated embedded content

#### **5. Risks**

- [ ] **Timeline/Schedule**
  - Risk: Low. Straightforward content changes with well-defined test assertions.
  - Mitigation: Tests are simple substring assertions with minimal development effort.
- [ ] **Test Coverage**
  - Risk: Tests validate content presence but cannot verify runtime LLM compliance with instructions.
  - Mitigation: Content tests ensure the instructions exist; behavioral compliance is monitored through review quality metrics.
- [ ] **Test Environment**
  - Risk: None. Tests run in standard Go environment with no external dependencies.
  - Mitigation: N/A
- [ ] **Untestable Aspects**
  - Risk: Whether agents actually follow the cross-file verification instructions at inference time cannot be mechanistically tested.
  - Mitigation: Observational monitoring of review findings for unverified assertions post-deployment.
- [ ] **Resource Constraints**
  - Risk: None. Tests are lightweight and fast.
  - Mitigation: N/A
- [ ] **Dependencies**
  - Risk: None. No external dependencies.
  - Mitigation: N/A
- [ ] **Other**
  - Risk: Instruction text could be inadvertently removed or modified in future PRs.
  - Mitigation: Tests serve as regression guards — any removal triggers test failure in CI.

---

### **III. Test Scenarios & Traceability**

This section links requirements to test coverage, enabling reviewers to verify all requirements are tested.

#### **1. Requirements-to-Tests Mapping**

- **Requirement ID:** GH-1835
- **Requirement Summary:** Code-review skill mandates reading files before asserting their contents in findings
- **Test Scenarios:**
  - [TS-GH-1835-001] Verify SKILL.md contains cross-file verification instruction in step 2
  - [TS-GH-1835-002] Verify SKILL.md contains self-check gate in step 4
  - [TS-GH-1835-003] Verify SKILL.md contains unreadable file fallback language
- **Tier:** Functional
- **Priority:** P0

- **Requirement Summary:** Code-review skill self-check gate validates file reads before finalizing findings
- **Test Scenarios:**
  - [TS-GH-1835-011] Verify self-check requires re-read if file not read in step 2
  - [TS-GH-1835-012] Verify reframe instruction and prohibition against asserting unverified contents
- **Tier:** Functional
- **Priority:** P1

- **Requirement Summary:** Correctness sub-agent enforces read-before-assert for cross-file findings
- **Test Scenarios:**
  - [TS-GH-1835-004] Verify correctness.md contains cross-file verification section and MUST-read mandate
  - [TS-GH-1835-005] Verify correctness.md prohibits presenting unverified file contents as fact
- **Tier:** Functional
- **Priority:** P0

- **Requirement Summary:** Both skill files provide consistent graceful degradation language for unreadable files
- **Test Scenarios:**
  - [TS-GH-1835-009] Verify consistent "unable to verify the contents" fallback language in both files
  - [TS-GH-1835-010] Verify neither file contains prohibited "assume the contents" phrasing
  - [TS-GH-1835-013] Verify SKILL.md does not contain deprecated instruction patterns from before the cross-file verification fix
- **Tier:** Functional
- **Priority:** P2

- **Requirement Summary:** Modified skill files are embedded in scaffold binary and deployable via layered content
- **Test Scenarios:**
  - [TS-GH-1835-006] Verify SKILL.md included in embedded filesystem via WalkFullsendRepoAll
  - [TS-GH-1835-007] Verify correctness.md included in scaffold embed via FullsendRepoFile
  - [TS-GH-1835-008] Verify updated files deployed via WalkLayeredContent with cross-file verification instructions
- **Tier:** Functional
- **Priority:** P1

---

### **IV. Sign-off and Approval**

This Software Test Plan requires approval from the following stakeholders:

* **Reviewers:**
  - TBD — pending QE lead assignment
* **Approvers:**
  - TBD — pending QE lead assignment
