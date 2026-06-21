# FullSend Test Plan

## **Review Agent Cross-File Content Verification - Quality Engineering Plan**

### **Metadata & Tracking**

- **Enhancement(s):** [GH-1835](https://github.com/fullsend-ai/fullsend/issues/1835)
- **Feature Tracking:** [GH-1835](https://github.com/fullsend-ai/fullsend/issues/1835)
- **Epic Tracking:** [GH-1835](https://github.com/fullsend-ai/fullsend/issues/1835)
- **QE Owner(s):** @ben-alkov
- **Owning SIG:** N/A
- **Participating SIGs:** N/A

**Document Conventions (if applicable):** N/A

### **Feature Overview**

The review agent hallucinated file contents on an external PR, claiming a Dockerfile contained `--nogpgcheck` when it never did. The root cause was that the code-review skill and correctness sub-agent had no explicit requirement to read files outside the PR diff before asserting what they contain. This fix adds cross-file verification requirements to the code-review SKILL.md (steps 2 and 4) and the correctness sub-agent definition, requiring the agent to read any file it references in a finding and to state inability to verify rather than assuming contents when files are inaccessible.

---

### **I. Motivation and Requirements Review (QE Review Guidelines)**

This section documents the mandatory QE review process. The goal is to understand the feature's value,
technology, and testability before formal test planning.

#### **1. Requirement & User Story Review Checklist**

- [ ] **Review Requirements**
  - Reviewed the relevant requirements.
  - Issue GH-1835 describes a specific false positive from the review agent on PR konflux-ci/konflux-test#833 where the agent fabricated Dockerfile contents.
  - Triage comment identifies the root cause: code-review skill step 2 only requires reading files in the diff, not files referenced in findings.
  - Related issues: #1445, #1774, #1656 cover adjacent but distinct failure modes.
- [ ] **Understand Value and Customer Use Cases**
  - Confirmed clear user stories and understood.
  - Understand the difference between community and product requirements.
  - **What is the value of the feature for customers**.
  - Ensured requirements contain relevant **customer use cases**.
  - Trust in automated review agent findings is critical for adoption. False positives from hallucinated file contents cause unnecessary rework and erode confidence in the review pipeline.
  - RICE score: 4.8 (Reach: 2, Impact: 1.5, Confidence: 0.8, Effort: 0.5).
- [ ] **Testability**
  - Confirmed requirements are **testable and unambiguous**.
  - Validation criteria are well-defined: zero findings should assert specific file contents without evidence of the agent having read that file. Verifiable by checking agent traces for file-read tool calls.
- [ ] **Acceptance Criteria**
  - Ensured acceptance criteria are **defined clearly** (clear user stories; product requirements clearly defined in Jira).
  - On next 5 review agent runs, zero findings should assert specific file contents without evidence of a file read. False positive rate for cross-file contradiction findings should drop.
- [ ] **Non-Functional Requirements (NFRs)**
  - Confirmed coverage for NFRs, including Performance, Security, Usability, Downtime, Connectivity, Monitoring (alerts/metrics), Scalability, Portability (e.g., cloud support), and Docs.
  - No performance, security, or scalability impact — changes are prompt-level instructions only. No runtime code changes.

#### **2. Known Limitations**

- The fix relies on prompt-level instructions which are advisory, not enforced programmatically. LLM compliance with the cross-file verification instruction cannot be guaranteed with 100% reliability.
- Files in external repositories (not the repo under review) may still be inaccessible to the review agent depending on token permissions. The agent will state it cannot verify rather than hallucinate, but coverage of cross-repo references remains limited.
- Existing scaffolded repositories will not receive the updated skill files until the next `fullsend scaffold` sync or repo reconciliation run.

#### **3. Technology and Design Review**

- [ ] **Developer Handoff/QE Kickoff**
  - A meeting where Dev/Arch walked QE through the design, architecture, and implementation details. **Critical for identifying untestable aspects early.**
  - Fix is prompt engineering — two Markdown skill files modified. No architectural changes. Code agent (fullsend-ai-coder) implemented the fix with commit `4e21a60`.
- [ ] **Technology Challenges**
  - Identified potential testing challenges related to the underlying technology.
  - Testing prompt-level behavior changes requires end-to-end agent runs and trace inspection. Non-deterministic LLM behavior means tests must validate patterns across multiple runs.
- [ ] **Test Environment Needs**
  - Determined necessary **test environment setups and tools**.
  - Requires a GitHub repository with PRs that reference files not in the diff. Agent runtime environment (GitHub Actions sandbox) is needed for E2E validation.
- [ ] **API Extensions**
  - Reviewed new or modified APIs and their impact on testing.
  - No API changes. Changes are to embedded scaffold templates consumed via `embed.FS` → `FullsendRepoFile()` → `CollectInstallFiles()`.
- [ ] **Topology Considerations**
  - Evaluated multi-cluster, network topology, and architectural impacts.
  - No topology impact. Changes propagate through scaffold sync to enrolled repositories.

### **II. Software Test Plan (STP)**

This STP serves as the **overall roadmap for testing**, detailing the scope, approach, resources, and schedule.

#### **1. Scope of Testing**

Testing validates that the review agent correctly reads files before asserting their contents in findings, that the self-check mechanism in step 4 catches unverified assertions, and that the correctness sub-agent follows the same cross-file verification protocol. Additionally, testing verifies that the updated skill templates propagate correctly through the scaffold embed pipeline.

**Testing Goals**

- **P0:** Verify that the review agent reads files before asserting their contents in cross-file findings — eliminates the root cause of the hallucination bug.
- **P0:** Verify that the self-check in step 4 catches and removes findings with unverified file content assertions.
- **P0:** Verify zero false positive findings from hallucinated file contents across review agent runs.
- **P1:** Verify the correctness sub-agent reads files before reporting cross-file contradictions.
- **P1:** Verify the agent explicitly states inability to verify when files are inaccessible rather than assuming contents.
- **P1:** Verify scaffold output includes the updated cross-file verification instructions.

**Out of Scope (Testing Scope Exclusions)**

- [ ] **Dockerfile linting or static analysis** — The original incident involved a Dockerfile, but the fix is to review agent behavior, not to Dockerfile tooling. Agreed: QE / 2026-06-21.
- [ ] **External repository file access permissions** — Token permissions and cross-repo access are managed by the platform team. Agreed: QE / 2026-06-21.
- [ ] **Review comment formatting and markdown structure** — Cosmetic output is not affected by this change. Agreed: QE / 2026-06-21.
- [ ] **Review agent performance or latency** — Prompt-only changes are not expected to impact agent execution time. Agreed: QE / 2026-06-21.

#### **2. Test Strategy**

**Functional**

- [ ] **Functional Testing** — Validates that the feature works according to specified requirements and user stories
  - *Details:* Validate cross-file verification behavior in code-review and correctness sub-agent. Test that file reads precede content assertions, self-check catches violations, and fallback messaging works for inaccessible files.
- [ ] **Automation Testing** — Confirms test automation plan is in place for CI and regression coverage (all tests are expected to be automated)
  - *Details:* Unit tests for scaffold template content validation (Go test suite). E2E tests for agent behavior validation via trace inspection.
- [ ] **Regression Testing** — Verifies that new changes do not break existing functionality
  - *Details:* Existing scaffold tests (`TestFullsendRepoFilesExist`, `TestCodeAgentContent`, etc.) verify template embedding. LSP analysis confirms `FullsendRepoFile()` has 39 callers — all must continue to work.

**Non-Functional**

- [ ] **Performance Testing** — Validates feature performance meets requirements (latency, throughput, resource usage)
  - *Details:* Not applicable. Prompt-level instruction changes do not affect agent execution performance.
- [ ] **Scale Testing** — Validates feature behavior under increased load and at production-like scale
  - *Details:* Not applicable. No runtime code changes.
- [ ] **Security Testing** — Verifies security requirements, RBAC, authentication, authorization, and vulnerability scanning
  - *Details:* Not applicable. No security-sensitive changes. The fix prevents information fabrication, which is a trust concern rather than a security vulnerability.
- [ ] **Usability Testing** — Validates user experience and accessibility requirements
  - *Details:* Not applicable. No user-facing interface changes.
- [ ] **Monitoring** — Does the feature require metrics and/or alerts?
  - *Details:* Not applicable. No new metrics or alerts required. Existing agent trace logging captures file-read tool calls for post-hoc analysis.

**Integration & Compatibility**

- [ ] **Compatibility Testing** — Ensures feature works across supported platforms, versions, and configurations
  - *Details:* Verify updated templates work with existing scaffold sync mechanism. No platform version dependencies.
- [ ] **Upgrade Testing** — Validates upgrade paths from previous versions, data migration, and configuration preservation
  - *Details:* Not applicable. Template changes are forward-only; no migration needed. Repos receive updates on next scaffold sync.
- [ ] **Dependencies** — Blocked by deliverables from other components/products
  - *Details:* No external dependencies. The fix is self-contained within the fullsend scaffold templates.
- [ ] **Cross Integrations** — Does the feature affect other features or require testing by other teams?
  - *Details:* The pr-review skill orchestrates correctness sub-agent. Both files are modified — verify the orchestrator correctly delegates to the updated sub-agent.

**Infrastructure**

- [ ] **Cloud Testing** — Does the feature require multi-cloud platform testing?
  - *Details:* Not applicable. Agent runs in GitHub Actions; no cloud-specific features involved.

#### **3. Test Environment**

- **Cluster Topology:** None required (agent runs in GitHub Actions sandbox)
- **Platform & Product Version(s):** FullSend 0.x on GitHub Actions
- **CPU Virtualization:** N/A
- **Compute Resources:** Standard GitHub Actions runner
- **Special Hardware:** None
- **Storage:** Standard runner disk
- **Network:** GitHub API access required for PR review operations
- **Required Operators:** None
- **Platform:** GitHub Actions
- **Special Configurations:** `GH_TOKEN` with repo read access for cross-file verification testing

#### **3.1. Testing Tools & Frameworks**

- **Other Tools:** Agent trace inspection tooling for E2E validation (to correlate file-read tool calls with content assertions in review findings)

#### **4. Entry Criteria**

The following conditions must be met before testing can begin:

- [ ] Requirements and design documents are **approved and merged**
- [ ] Test environment can be **set up and configured** (see Section II.3 - Test Environment)
- [ ] PR #2443 is merged and skill templates are updated in the scaffold
- [ ] At least one enrolled repository has received the updated skill files via scaffold sync

#### **5. Risks**

- [ ] **Timeline/Schedule**
  - Risk: Agent behavior validation requires multiple review runs which take time and consume API credits.
  - Mitigation: Use targeted test PRs with known cross-file reference patterns to accelerate validation.
- [ ] **Test Coverage**
  - Risk: LLM non-determinism means the agent may occasionally fail to follow prompt instructions even after the fix.
  - Mitigation: Validate across 5+ review runs as specified in acceptance criteria. Statistical validation rather than single-run pass/fail.
- [ ] **Test Environment**
  - Risk: E2E tests require a repository with PRs containing cross-file references — test data setup is needed.
  - Mitigation: Use existing test repositories (e.g., konflux-ci/konflux-test) or create synthetic test PRs.
- [ ] **Untestable Aspects**
  - Risk: Cannot programmatically guarantee 100% LLM compliance with prompt instructions.
  - Mitigation: Focus on trace-level validation (file-read tool calls present before content assertions) rather than output-level validation only.
- [ ] **Resource Constraints**
  - Risk: No significant resource constraints identified.
  - Mitigation: N/A.
- [ ] **Dependencies**
  - Risk: Scaffold sync timing — enrolled repos may not immediately receive the updated templates.
  - Mitigation: Test directly on the scaffold output first, then validate on synced repos.
- [ ] **Other**
  - Risk: Related issues (#1445, #1774, #1656) may interact with this fix in unexpected ways.
  - Mitigation: Review adjacent fixes for conflicts. Validate that the cross-file verification instruction does not interfere with existing review behavior.

---

### **III. Test Scenarios & Traceability**

This section links requirements to test coverage, enabling reviewers to verify all requirements are tested.

#### **1. Requirements-to-Tests Mapping**

- **GH-1835** — Review agent reads files before asserting their contents in findings
  - Verify agent reads file before citing its contents — Functional — P0
  - Verify no finding asserts unread file contents — Functional — P0
  - Verify agent handles cross-repo file reference gracefully — Functional — P0

- **GH-1835** — Review agent self-checks cross-file findings before finalizing
  - Verify self-check catches unverified file assertion — Functional — P0
  - Verify finding removed when file read contradicts assertion — Functional — P0
  - Verify self-check passes for verified cross-file finding — Functional — P0

- **GH-1835** — Correctness sub-agent validates file contents before cross-file contradiction findings
  - Verify correctness sub-agent reads files before contradiction finding — Functional — P1
  - Verify real contradiction detected after file read — Functional — P1
  - Verify no false contradiction when file lacks claimed content — Functional — P1

- **GH-1835** — Agent states inability to verify when files are unreadable rather than assuming contents
  - Verify agent states unverifiable for inaccessible file — Functional — P1
  - Verify no content assumption for cross-repo files — Functional — P1

- **GH-1835** — Scaffold installs updated skill templates to enrolled repositories
  - Verify scaffold output includes cross-file verification instructions — Unit Tests — P1
  - Verify embedded template content matches source files — Unit Tests — P1

- **GH-1835** — No false positive findings from hallucinated file contents
  - Verify zero hallucinated file content assertions in review run — End-to-End — P0
  - Verify agent trace shows read calls for all cited files — End-to-End — P0
  - Verify review accuracy on PR with cross-file references — End-to-End — P0

---

### **IV. Sign-off and Approval**

This Software Test Plan requires approval from the following stakeholders:

* **Reviewers:**
  - @ben-alkov
  - [Name / @github-username]
* **Approvers:**
  - [Name / @github-username]
  - [Name / @github-username]
