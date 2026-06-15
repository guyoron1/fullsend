# FullSend Test Plan

## **Add Testing-Agents Problem Document - Quality Engineering Plan**

### **Metadata & Tracking**

- **Enhancement(s):** [GH-14](https://github.com/fullsend-ai/fullsend/pull/14)
- **Feature Tracking:** [GH-14](https://github.com/fullsend-ai/fullsend/pull/14)
- **Epic Tracking:** GH-14
- **QE Owner(s):** TBD
- **Owning SIG:** N/A
- **Participating SIGs:** None

**Document Conventions (if applicable):** N/A

### **Feature Overview**

This PR introduces a problem exploration document (`docs/problems/testing-agents.md`) that addresses how to verify agent behavior hasn't regressed when instructions change. The document covers golden-set evaluation, behavioral contract testing, canary deployments, mutation testing for natural-language instructions, and eval framework integration (promptfoo, deepeval, lightspeed-evaluation). A companion PR (#2009) adds a related document on tool call risk assessment, proposing LLM-as-judge pre-tool hooks, behavioral baselines, and declarative tool call policies that complement the existing security hook architecture (Tirith, SSRF validator, canary, unicode normalizer).

---

### **I. Motivation and Requirements Review (QE Review Guidelines)**

This section documents the mandatory QE review process. The goal is to understand the feature's value,
technology, and testability before formal test planning.

#### **1. Requirement & User Story Review Checklist**

- [ ] **Review Requirements**
  - Reviewed the relevant requirements.
  - GH-14 is a documentation-only PR that adds a problem exploration document. No code changes, no API changes, no configuration changes. Requirements center on document accuracy, completeness, and internal consistency.
- [ ] **Understand Value and Customer Use Cases**
  - Confirmed clear user stories and understood.
  - Understand the difference between community and product requirements.
  - **What is the value of the feature for customers**.
  - Ensured requirements contain relevant **customer use cases**.
  - This document establishes the intellectual foundation for CI-for-prompts in FullSend. It enables the team to make informed decisions about agent instruction testing strategies, directly impacting platform reliability for end users who depend on consistent agent behavior.
- [ ] **Testability**
  - Confirmed requirements are **testable and unambiguous**.
  - Document content is testable through link validation, structural completeness checks, and accuracy verification against the codebase's actual security hook architecture (discovered via LSP analysis of `internal/security/hooks.go` and `internal/harness/harness.go`).
- [ ] **Acceptance Criteria**
  - Ensured acceptance criteria are **defined clearly** (clear user stories; product requirements clearly defined in Jira).
  - No explicit acceptance criteria in the PR body. Implicit criteria: document is well-structured, internally consistent, cross-references resolve, and descriptions of existing hooks match the codebase.
- [ ] **Non-Functional Requirements (NFRs)**
  - Confirmed coverage for NFRs, including Performance, Security, Usability, Downtime, Connectivity, Monitoring (alerts/metrics), Scalability, Portability (e.g., cloud support), and Docs.
  - NFRs are not directly applicable to a documentation PR. The document itself discusses security testing approaches (adversarial robustness, prompt injection resistance) which are NFR-adjacent design considerations, not testable NFRs for this PR.

#### **2. Known Limitations**

- This is a problem exploration document, not a specification or implementation. It describes approaches and trade-offs but does not prescribe a specific solution.
- No code changes are included; testing is limited to document validation rather than functional verification.
- The PR was merged before this STP was generated; testing is retrospective rather than gating.
- Eval framework descriptions (promptfoo, deepeval, lightspeed-evaluation) may become outdated as those tools evolve.

#### **3. Technology and Design Review**

- [ ] **Developer Handoff/QE Kickoff**
  - A meeting where Dev/Arch walked QE through the design, architecture, and implementation details. **Critical for identifying untestable aspects early.**
  - No formal handoff required for a documentation PR. PR review comments between @twaugh and @ralphbean demonstrate collaborative design discussion, including the addition of eval framework coverage based on review feedback.
- [ ] **Technology Challenges**
  - Identified potential testing challenges related to the underlying technology.
  - Primary challenge is validating that the document's descriptions of existing security hooks (Tirith, SSRF, canary, unicode, tool allowlist) accurately reflect the codebase. LSP analysis confirmed 8 hook variables and `GenerateClaudeSettings` in `internal/security/hooks.go`, along with `SecurityConfig` struct hierarchy in `internal/harness/harness.go`.
- [ ] **Test Environment Needs**
  - Determined necessary **test environment setups and tools**.
  - No special environment needed. Document validation requires only the repository checkout and standard markdown tooling.
- [ ] **API Extensions**
  - Reviewed new or modified APIs and their impact on testing.
  - No API changes. The documents reference existing APIs and hook architectures but do not modify them.
- [ ] **Topology Considerations**
  - Evaluated multi-cluster, network topology, and architectural impacts.
  - Not applicable for documentation changes.

### **II. Software Test Plan (STP)**

This STP serves as the **overall roadmap for testing**, detailing the scope, approach, resources, and schedule.

#### **1. Scope of Testing**

Testing scope covers validation of the two problem exploration documents added by PR #14 and companion PR #2009. This includes verifying document structural completeness, internal link validity, cross-reference accuracy, and alignment between documented security hook descriptions and the actual codebase architecture.

**Testing Goals**

*Functional Goals:*

- **P0:** Verify README index entries correctly link to both new problem documents
- **P0:** Verify tool call risk assessment document accurately describes existing security hook architecture (Tirith, SSRF, canary, unicode normalizer, tool allowlist) as implemented in `internal/security/hooks.go`
- **P1:** Verify testing-agents document covers all four proposed testing approaches (golden-set, behavioral contracts, canary, mutation testing)
- **P1:** Verify all internal cross-references between problem documents resolve to valid targets
- **P2:** Verify eval framework descriptions (promptfoo, deepeval, lightspeed-evaluation) are technically accurate

*Quality Goals:*

- **P1:** Verify documents are internally consistent (no contradictions between sections)

*Integration Goals:*

- **P1:** Verify proposed approaches reference correct existing codebase components

**Out of Scope (Testing Scope Exclusions)**

- [ ] Implementation of any testing approach described in the documents -- *Rationale:* These are problem exploration documents, not implementation specs. Implementation testing will be covered when code PRs are submitted. -- *PM/Lead Agreement:* TBD
- [ ] Eval framework functionality testing (promptfoo, deepeval, lightspeed-evaluation) -- *Rationale:* Third-party tools are out of scope for FullSend product QE. -- *PM/Lead Agreement:* TBD
- [ ] GitHub Actions CI pipeline configuration -- *Rationale:* Infrastructure-level concern tested by platform team. -- *PM/Lead Agreement:* TBD
- [ ] Go code compilation or unit test execution -- *Rationale:* No Go code was changed in these PRs. -- *PM/Lead Agreement:* TBD

#### **2. Test Strategy**

**Functional**

- [ ] **Functional Testing** -- Validates that the feature works according to specified requirements and user stories
  - *Details:* Validate document content accuracy, structural completeness, and link integrity. Verify descriptions of existing security hooks match codebase reality.
- [ ] **Automation Testing** -- Confirms test automation plan is in place for CI and regression coverage (all tests are expected to be automated)
  - *Details:* Link validation can be automated via markdown lint tooling. Content accuracy checks against codebase require manual or semi-automated review.
- [ ] **Regression Testing** -- Verifies that new changes do not break existing functionality
  - *Details:* Verify README.md modifications do not break existing index entries or formatting. Verify no existing problem document cross-references are invalidated.

**Non-Functional**

- [ ] **Performance Testing** -- Validates feature performance meets requirements (latency, throughput, resource usage)
  - *Details:* N/A -- documentation changes have no performance impact.
- [ ] **Scale Testing** -- Validates feature behavior under increased load and at production-like scale (e.g., large number of resources, nodes, or concurrent operations)
  - *Details:* N/A -- documentation changes have no scale impact.
- [ ] **Security Testing** -- Verifies security requirements, RBAC, authentication, authorization, and vulnerability scanning
  - *Details:* The tool call risk assessment document discusses security architecture. Verify that no sensitive information (internal URLs, credentials, private infrastructure details) is exposed in the public documentation.
- [ ] **Usability Testing** -- Validates user experience and accessibility requirements
  - *Details:* Verify documents are well-organized, clearly written, and navigable via the README index.
- [ ] **Monitoring** -- Does the feature require metrics and/or alerts?
  - *Details:* N/A -- no monitoring requirements for documentation.

**Integration & Compatibility**

- [ ] **Compatibility Testing** -- Ensures feature works across supported platforms, versions, and configurations
  - *Details:* N/A -- markdown documents are platform-independent.
- [ ] **Upgrade Testing** -- Validates upgrade paths from previous versions, data migration, and configuration preservation
  - *Details:* N/A -- no upgrade path for documentation.
- [ ] **Dependencies** -- Blocked by deliverables from other components/products. Identify what we need from other teams before we can test.
  - *Details:* No external dependencies. Documents reference existing codebase components only.
- [ ] **Cross Integrations** -- Does the feature affect other features or require testing by other teams? Identify the impact we cause.
  - *Details:* The testing-agents document references code-review.md, agent-architecture.md, security-threat-model.md, governance.md, repo-readiness.md, and agent-infrastructure.md. Changes to those documents could require updates to testing-agents.md cross-references.

**Infrastructure**

- [ ] **Cloud Testing** -- Does the feature require multi-cloud platform testing? Consider cloud-specific features.
  - *Details:* N/A -- documentation changes have no cloud infrastructure requirements.

#### **3. Test Environment**

- **Cluster Topology:** N/A (no cluster required for documentation validation)
- **Platform & Product Version(s):** FullSend 0.x on GitHub Actions
- **CPU Virtualization:** N/A
- **Compute Resources:** N/A
- **Special Hardware:** N/A
- **Storage:** N/A
- **Network:** N/A
- **Required Operators:** N/A
- **Platform:** GitHub Actions (for CI-based link validation)
- **Special Configurations:** None

#### **3.1. Testing Tools & Frameworks**

- **Test Framework:** N/A -- standard markdown linting only
- **CI/CD:** N/A -- standard GitHub Actions
- **Other Tools:** N/A

#### **4. Entry Criteria**

The following conditions must be met before testing can begin:

- [ ] Requirements and design documents are **approved and merged**
- [ ] Test environment can be **set up and configured** (see Section II.3 - Test Environment)
- [ ] Repository checkout is available with both PR #14 and PR #2009 changes merged

#### **5. Risks**

- [ ] **Timeline/Schedule**
  - Risk: PR #14 is already merged; retrospective testing has limited ability to influence changes.
  - Mitigation: Document issues as follow-up PRs if inaccuracies are found.
- [ ] **Test Coverage**
  - Risk: Document accuracy validation against codebase is partially manual and may miss subtle misalignments between documentation and implementation.
  - Mitigation: Use LSP analysis results to systematically verify hook descriptions against `SecurityConfig` struct fields and `GenerateClaudeSettings` function.
- [ ] **Test Environment**
  - Risk: N/A -- minimal environment requirements for documentation testing.
  - Mitigation: N/A
- [ ] **Untestable Aspects**
  - Risk: Subjective document qualities (clarity, usefulness, completeness of trade-off analysis) are not objectively testable.
  - Mitigation: Rely on peer review process (PR comments from @ralphbean and @twaugh demonstrate this occurred).
- [ ] **Resource Constraints**
  - Risk: N/A -- documentation validation requires minimal resources.
  - Mitigation: N/A
- [ ] **Dependencies**
  - Risk: Cross-referenced problem documents (code-review.md, security-threat-model.md, etc.) could change, breaking references.
  - Mitigation: Automated link checking in CI can detect broken cross-references.
- [ ] **Other**
  - Risk: Eval framework descriptions may become outdated as promptfoo, deepeval, and lightspeed-evaluation release new versions.
  - Mitigation: Periodic review of external tool descriptions; the document itself notes "the tooling is maturing quickly" and recommends periodic re-evaluation.

---

### **III. Test Scenarios & Traceability**

This section links requirements to test coverage, enabling reviewers to verify all requirements are tested.

#### **1. Requirements-to-Tests Mapping**

- **[GH-14]** -- Document accurately describes agent testing approaches including golden-set evaluation, behavioral contract testing, canary deployments, and mutation testing
  - *Test Scenario:* Verify all four testing approaches are documented with trade-offs
  - *Priority:* P1
  - *Test Scenario:* Verify CI pipeline section references all five pipeline stages
  - *Priority:* P1
  - *Test Scenario:* Verify error when approach section is missing or incomplete
  - *Priority:* P2

- **[GH-14]** -- Cross-references between problem documents resolve to valid targets
  - *Test Scenario:* Verify all internal document links resolve correctly
  - *Priority:* P0
  - *Test Scenario:* Verify broken cross-reference is detected and reported
  - *Priority:* P1

- **[GH-14]** -- Eval framework integration patterns are documented for promptfoo, deepeval, and lightspeed-evaluation
  - *Test Scenario:* Verify each framework section describes capabilities and gaps
  - *Priority:* P2
  - *Test Scenario:* Verify input expansion from seed sets pattern is documented
  - *Priority:* P2

- **[GH-14]** -- Tool call risk assessment document accurately describes existing security hook architecture
  - *Test Scenario:* Verify document references match codebase hooks (Tirith, SSRF, canary, unicode, tool allowlist, secret redactor, context suppressor)
  - *Priority:* P0
  - *Test Scenario:* Verify hook descriptions align with SecurityConfig and SandboxHooks struct fields in harness.go
  - *Priority:* P0
  - *Test Scenario:* Verify error when hook description mismatches codebase implementation
  - *Priority:* P1

- **[GH-14]** -- Proposed risk assessment approaches are internally consistent and logically complete
  - *Test Scenario:* Verify four approaches cover the risk assessment spectrum (deterministic through semantic)
  - *Priority:* P1
  - *Test Scenario:* Verify hybrid approach correctly references deterministic and LLM-judge components
  - *Priority:* P1

- **[GH-14]** -- README index entries correctly link to new problem documents
  - *Test Scenario:* Verify README link to testing-agents.md resolves
  - *Priority:* P0
  - *Test Scenario:* Verify README link to tool-call-risk-assessment.md resolves
  - *Priority:* P0
  - *Test Scenario:* Verify broken README link is detected
  - *Priority:* P0

---

### **IV. Sign-off and Approval**

This Software Test Plan requires approval from the following stakeholders:

* **Reviewers:**
  - [TBD / @tbd]
  - [TBD / @tbd]
* **Approvers:**
  - [TBD / @tbd]
  - [TBD / @tbd]
