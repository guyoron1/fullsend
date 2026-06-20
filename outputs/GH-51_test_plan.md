# FullSend Test Plan

## **Inject CLAUDE.md Pointer for Repos with AGENTS.md - Quality Engineering Plan**

### **Metadata & Tracking**

- **Enhancement(s):** [GH-51](https://github.com/guyoron1/fullsend/issues/51)
- **Feature Tracking:** [PR #51](https://github.com/guyoron1/fullsend/pull/51)
- **Epic Tracking:** GH-51
- **QE Owner(s):** TBD
- **Owning SIG:** N/A
- **Participating SIGs:** None

**Document Conventions (if applicable):** N/A

### **Feature Overview**

This feature adds automatic injection of a minimal CLAUDE.md pointer file when FullSend runs a Claude Code agent against a target repository that has AGENTS.md but no CLAUDE.md. Claude Code auto-loads CLAUDE.md into its system context but does not read AGENTS.md by default; without this bridge file, agents are effectively context-blind in repos that only ship AGENTS.md. The injected file is excluded from git tracking so it never appears in commits.

---

### **I. Motivation and Requirements Review (QE Review Guidelines)**

This section documents the mandatory QE review process. The goal is to understand the feature's value,
technology, and testability before formal test planning.

#### **1. Requirement & User Story Review Checklist**

- [ ] **Review Requirements**
  - Reviewed the relevant requirements.
  - GH-51 describes the feature clearly: inject CLAUDE.md when repo has AGENTS.md but no CLAUDE.md, scoped to Claude Code runtime only.
- [ ] **Understand Value and Customer Use Cases**
  - Confirmed clear user stories and understood.
  - Understand the difference between community and product requirements.
  - **What is the value of the feature for customers**.
  - Ensured requirements contain relevant **customer use cases**.
  - Customers using AGENTS.md for agent guidance will now have Claude Code automatically discover those rules without requiring a separate CLAUDE.md file.
- [ ] **Testability**
  - Confirmed requirements are **testable and unambiguous**.
  - Feature is fully testable: `hasClaudeMD` and `doInjectClaudeMDPointer` are unit-testable with mocks; integration path is testable in sandbox setup flow.
- [ ] **Acceptance Criteria**
  - Ensured acceptance criteria are **defined clearly** (clear user stories; product requirements clearly defined in Jira).
  - AC derived from issue description: when repo has AGENTS.md but no CLAUDE.md and runtime is Claude, inject pointer. PR includes 11 unit tests validating all paths.
- [ ] **Non-Functional Requirements (NFRs)**
  - Confirmed coverage for NFRs, including Performance, Security, Usability, Downtime, Connectivity, Monitoring (alerts/metrics), Scalability, Portability (e.g., cloud support), and Docs.
  - Minimal NFR impact: single file write + git exclude operation during sandbox bootstrap. No performance, security, or scalability concerns identified.

#### **2. Known Limitations**

- The CLAUDE.md detection checks only four casing variants (CLAUDE.md, claude.md, Claude.md, .claude.md). Non-standard casings or nested CLAUDE.md files are not detected.
- The injected CLAUDE.md is a static pointer; it does not dynamically reflect changes to AGENTS.md content during the agent run.
- If the sandbox `printf` command is unavailable or restricted, the injection will silently fail with a warning.

#### **3. Technology and Design Review**

- [ ] **Developer Handoff/QE Kickoff**
  - A meeting where Dev/Arch walked QE through the design, architecture, and implementation details. **Critical for identifying untestable aspects early.**
  - PR #51 provides clear implementation with inline comments. `doInjectClaudeMDPointer` is extracted for testability with a `sandboxExecFunc` type.
- [ ] **Technology Challenges**
  - Identified potential testing challenges related to the underlying technology.
  - No significant challenges. Feature uses existing sandbox.Exec infrastructure and filesystem operations.
- [ ] **Test Environment Needs**
  - Determined necessary **test environment setups and tools**.
  - Standard Go test environment with `t.TempDir()` for filesystem tests. No special sandbox or cluster required for unit tests.
- [ ] **API Extensions**
  - Reviewed new or modified APIs and their impact on testing.
  - No public API changes. Internal functions added: `hasClaudeMD`, `injectClaudeMDPointer`, `doInjectClaudeMDPointer`. New type: `sandboxExecFunc`.
- [ ] **Topology Considerations**
  - Evaluated multi-cluster, network topology, and architectural impacts.
  - No topology impact. Feature operates within the local sandbox bootstrap flow.

### **II. Software Test Plan (STP)**

This STP serves as the **overall roadmap for testing**, detailing the scope, approach, resources, and schedule.

#### **1. Scope of Testing**

Testing covers the CLAUDE.md pointer injection feature within the FullSend CLI `runAgent` function. This includes the `hasClaudeMD` file detection, the `injectClaudeMDPointer`/`doInjectClaudeMDPointer` injection logic, guard conditions (runtime check, AGENTS.md availability, CLAUDE.md absence), git exclude behavior, and error handling for write and exclude failures.

**Testing Goals**

**Functional Goals**

- **P0:** Verify CLAUDE.md pointer is correctly injected when all three guard conditions are met (Claude runtime, AGENTS.md available, no CLAUDE.md)
- **P1:** Verify all guard conditions correctly prevent injection when not satisfied
- **P1:** Verify `hasClaudeMD` correctly detects all four supported casing variants
- **P1:** Verify error handling allows agent run to continue when injection fails

**Quality Goals**

- **P1:** Verify injected CLAUDE.md is excluded from git tracking
- **P2:** Verify graceful degradation when git exclude command fails

**Integration Goals**

- **P1:** Verify `agentsMDAvailable` flag correctly propagates from org AGENTS.md injection to CLAUDE.md injection logic

**Out of Scope (Testing Scope Exclusions)**

- [ ] Sandbox container filesystem permissions -- *Rationale:* Platform-level infrastructure tested by sandbox team -- *PM/Lead Agreement:* TBD
- [ ] Git exclude mechanism internals (.git/info/exclude) -- *Rationale:* Git platform behavior tested by git upstream -- *PM/Lead Agreement:* TBD
- [ ] Claude Code's CLAUDE.md loading behavior -- *Rationale:* External tool behavior outside FullSend's control -- *PM/Lead Agreement:* TBD

#### **2. Test Strategy**

**Functional**

- [x] **Functional Testing** -- Validates that the feature works according to specified requirements and user stories
  - *Details:* Unit tests for `hasClaudeMD` casing detection; unit tests for `doInjectClaudeMDPointer` with mock exec; functional tests for guard condition logic in `runAgent`.
- [x] **Automation Testing** -- Confirms test automation plan is in place for CI and regression coverage (all tests are expected to be automated)
  - *Details:* All 19 test scenarios are automatable. PR already includes 11 Go unit tests. Remaining functional scenarios will be automated in the standard Go test suite.
- [x] **Regression Testing** -- Verifies that new changes do not break existing functionality
  - *Details:* LSP analysis confirms `runAgent` is called from `newRunCmd` and 10 existing test functions. The `agentsMDAvailable` variable change and new injection step must not regress existing AGENTS.md injection behavior.

**Non-Functional**

- [ ] **Performance Testing** -- Validates feature performance meets requirements (latency, throughput, resource usage)
  - *Details:* N/A. Single file write + git exclude adds negligible latency to sandbox bootstrap.
- [ ] **Scale Testing** -- Validates feature behavior under increased load and at production-like scale
  - *Details:* N/A. Feature operates once per agent run, not at scale.
- [ ] **Security Testing** -- Verifies security requirements, RBAC, authentication, authorization, and vulnerability scanning
  - *Details:* N/A. Injected content is a static constant string (`claudeMDPointerContent`). No user input is written to the file.
- [ ] **Usability Testing** -- Validates user experience and accessibility requirements
  - *Details:* N/A. Feature is transparent to the user; only observable via `StepDone` log message.
- [ ] **Monitoring** -- Does the feature require metrics and/or alerts?
  - *Details:* N/A. No metrics or alerts required. Warning messages via `printer.StepWarn` provide sufficient observability.

**Integration & Compatibility**

- [ ] **Compatibility Testing** -- Ensures feature works across supported platforms, versions, and configurations
  - *Details:* N/A. Feature uses standard Go `os.Stat` and `fmt.Sprintf` — no platform-specific behavior.
- [ ] **Upgrade Testing** -- Validates upgrade paths from previous versions, data migration, and configuration preservation
  - *Details:* N/A. No persistent state or data migration involved.
- [ ] **Dependencies** -- Blocked by deliverables from other components/products
  - *Details:* Depends on existing `sandbox.Exec` and `sandbox.UploadFile` APIs. No new external dependencies.
- [ ] **Cross Integrations** -- Does the feature affect other features or require testing by other teams?
  - *Details:* Interacts with the AGENTS.md injection flow (step 8a in `runAgent`). The `agentsMDAvailable` flag is shared state between the two features.

**Infrastructure**

- [ ] **Cloud Testing** -- Does the feature require multi-cloud platform testing?
  - *Details:* N/A. Feature is cloud-agnostic.

#### **3. Test Environment**

- **Cluster Topology:** N/A (no cluster required for unit/functional tests)
- **Platform & Product Version(s):** Go 1.23+, GitHub Actions
- **CPU Virtualization:** N/A
- **Compute Resources:** Standard CI runner
- **Special Hardware:** None
- **Storage:** Local filesystem (t.TempDir for tests)
- **Network:** N/A
- **Required Operators:** None
- **Platform:** GitHub Actions
- **Special Configurations:** None

#### **3.1. Testing Tools & Frameworks**

- **Test Framework:** Standard (Go testing, testify)
- **CI/CD:** Standard (GitHub Actions)
- **Other Tools:** None

#### **4. Entry Criteria**

The following conditions must be met before testing can begin:

- [ ] Requirements and design documents are **approved and merged**
- [ ] Test environment can be **set up and configured** (see Section II.3 - Test Environment)
- [ ] PR #51 code changes are merged to main branch

#### **5. Risks**

- [ ] **Timeline/Schedule**
  - Risk: N/A. Feature is small and self-contained.
  - Mitigation: N/A.
- [ ] **Test Coverage**
  - Risk: Guard condition combinations may not cover all edge cases (e.g., race between AGENTS.md injection failure and CLAUDE.md injection check).
  - Mitigation: PR includes tests for success, write-failure, and exclude-failure paths. Combinatorial guard condition testing covers all three conditions.
- [ ] **Test Environment**
  - Risk: N/A. Tests use standard Go testing infrastructure.
  - Mitigation: N/A.
- [ ] **Untestable Aspects**
  - Risk: Cannot directly verify that Claude Code reads the injected CLAUDE.md in production sandbox environments.
  - Mitigation: Verify file content and placement; Claude Code's CLAUDE.md loading is documented behavior.
- [ ] **Resource Constraints**
  - Risk: N/A. No special resources required.
  - Mitigation: N/A.
- [ ] **Dependencies**
  - Risk: Depends on `sandbox.Exec` behavior remaining stable.
  - Mitigation: `doInjectClaudeMDPointer` uses injected `sandboxExecFunc` for testability, isolating from actual sandbox implementation.
- [ ] **Other**
  - Risk: Filesystem case sensitivity differences between macOS (case-insensitive) and Linux (case-sensitive) could affect `hasClaudeMD` behavior.
  - Mitigation: Tests use `t.TempDir()` which respects OS behavior; CI runs on Linux matching production.

---

### **III. Test Scenarios & Traceability**

This section links requirements to test coverage, enabling reviewers to verify all requirements are tested.

#### **1. Requirements-to-Tests Mapping**

- **[GH-51]** -- CLAUDE.md pointer is injected when target repo has AGENTS.md but no CLAUDE.md and runtime is Claude Code
  - *Test Scenario:* Verify CLAUDE.md pointer injected for Claude runtime with AGENTS.md only
  - *Tier:* Functional
  - *Priority:* P0
  - *Test Scenario:* Verify injected CLAUDE.md content references AGENTS.md
  - *Tier:* Unit Tests
  - *Priority:* P0
  - *Test Scenario:* Verify no injection when runtime is not Claude
  - *Tier:* Functional
  - *Priority:* P0

- **[GH-51]** -- CLAUDE.md detection handles all common casing variants (CLAUDE.md, claude.md, Claude.md, .claude.md)
  - *Test Scenario:* Verify detection of CLAUDE.md uppercase
  - *Tier:* Unit Tests
  - *Priority:* P1
  - *Test Scenario:* Verify detection of claude.md lowercase
  - *Tier:* Unit Tests
  - *Priority:* P1
  - *Test Scenario:* Verify detection of .claude.md dot-prefixed
  - *Tier:* Unit Tests
  - *Priority:* P1
  - *Test Scenario:* Verify false when no CLAUDE.md variants exist
  - *Tier:* Unit Tests
  - *Priority:* P1

- **[GH-51]** -- Injected CLAUDE.md is excluded from git tracking in the sandbox
  - *Test Scenario:* Verify CLAUDE.md added to git exclude after injection
  - *Tier:* Functional
  - *Priority:* P1
  - *Test Scenario:* Verify injected file hidden from git status
  - *Tier:* Functional
  - *Priority:* P1

- **[GH-51]** -- CLAUDE.md injection is skipped for non-Claude runtimes
  - *Test Scenario:* Verify no injection for non-Claude agent runtime
  - *Tier:* Functional
  - *Priority:* P1

- **[GH-51]** -- CLAUDE.md injection is skipped when repo already has CLAUDE.md
  - *Test Scenario:* Verify no injection when CLAUDE.md already exists
  - *Tier:* Functional
  - *Priority:* P1
  - *Test Scenario:* Verify skip applies to all casing variants
  - *Tier:* Unit Tests
  - *Priority:* P1

- **[GH-51]** -- CLAUDE.md injection is skipped when no AGENTS.md is available (neither repo nor org)
  - *Test Scenario:* Verify no injection without AGENTS.md
  - *Tier:* Functional
  - *Priority:* P1

- **[GH-51]** -- Injection failure is handled gracefully without aborting agent run
  - *Test Scenario:* Verify warning logged on CLAUDE.md write failure
  - *Tier:* Unit Tests
  - *Priority:* P1
  - *Test Scenario:* Verify agent run continues after injection failure
  - *Tier:* Functional
  - *Priority:* P1

- **[GH-51]** -- Git exclude failure after successful write is handled gracefully
  - *Test Scenario:* Verify warning on exclude failure after successful write
  - *Tier:* Unit Tests
  - *Priority:* P2
  - *Test Scenario:* Verify CLAUDE.md preserved despite exclude failure
  - *Tier:* Unit Tests
  - *Priority:* P2

- **[GH-51]** -- Existing AGENTS.md injection flow correctly propagates agentsMDAvailable flag to CLAUDE.md logic
  - *Test Scenario:* Verify CLAUDE.md injected after org AGENTS.md injection
  - *Tier:* Functional
  - *Priority:* P1
  - *Test Scenario:* Verify no CLAUDE.md when org AGENTS.md injection fails
  - *Tier:* Functional
  - *Priority:* P1

---

### **IV. Sign-off and Approval**

This Software Test Plan requires approval from the following stakeholders:

* **Reviewers:**
  - [TBD]
* **Approvers:**
  - [TBD]
