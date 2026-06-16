# FullSend Test Plan

## **MCP Configuration Drift Problem Document - Quality Engineering Plan**

### **Metadata & Tracking**

- **Enhancement(s):** [GH-17](https://github.com/guyoron1/fullsend/issues/17)
- **Feature Tracking:** [GH-17](https://github.com/guyoron1/fullsend/issues/17)
- **Epic Tracking:** [GH-17](https://github.com/guyoron1/fullsend/issues/17)
- **QE Owner(s):** TBD
- **Owning SIG:** N/A
- **Participating SIGs:** N/A

**Document Conventions (if applicable):** N/A

### **Feature Overview**

This issue adds a problem document (`docs/problems/mcp-config-drift.md`) that explores MCP (Model Context Protocol) configuration drift — the risk that MCP server configurations silently change between agent runs, expanding the tool surface or redirecting tool calls to untrusted endpoints. The PR also adds an index entry in `README.md`. This is a documentation-only change with no code modifications; it describes attack scenarios and defense considerations for future implementation.

---

### **I. Motivation and Requirements Review (QE Review Guidelines)**

This section documents the mandatory QE review process. The goal is to understand the feature's value,
technology, and testability before formal test planning.

#### **1. Requirement & User Story Review Checklist**

- [ ] **Review Requirements**
  - Reviewed the relevant requirements.
  - This is a documentation-only PR. The requirement is to document the MCP configuration drift problem space, attack scenarios, and potential defense approaches. Mirrored from upstream PR #2011 (fullsend-ai/fullsend).
- [ ] **Understand Value and Customer Use Cases**
  - Confirmed clear user stories and understood.
  - Understand the difference between community and product requirements.
  - **What is the value of the feature for customers**.
  - Ensured requirements contain relevant **customer use cases**.
  - The document provides security analysis value by identifying attack vectors where MCP configs can be silently modified to inject malicious tool servers, replace endpoints, or escalate permissions. This informs future harness-level defenses.
- [ ] **Testability**
  - Confirmed requirements are **testable and unambiguous**.
  - As a documentation PR, testability is limited to document structure, content accuracy, and link integrity. No functional code is introduced. The defense approaches described (baseline hashing, immutable harness input, content-aware validation) are not yet implemented and therefore not testable as features.
- [ ] **Acceptance Criteria**
  - Ensured acceptance criteria are **defined clearly** (clear user stories; product requirements clearly defined in Jira).
  - Implicit acceptance criteria: (1) New problem doc renders correctly in Markdown, (2) README index entry links to the correct file, (3) Cross-references to existing docs (security-threat-model.md, agent-architecture.md, ADR 0017) are valid.
- [ ] **Non-Functional Requirements (NFRs)**
  - Confirmed coverage for NFRs, including Performance, Security, Usability, Downtime, Connectivity, Monitoring (alerts/metrics), Scalability, Portability (e.g., cloud support), and Docs.
  - No NFRs apply to this documentation change. Security implications are documented within the problem doc itself but no security controls are implemented in this PR.

#### **2. Known Limitations**

- This PR documents future defense considerations only — no MCP drift detection is implemented.
- The problem doc references ADR 0017 and security-threat-model.md; if those documents are relocated or renamed, cross-references will break.
- The document does not cover dynamic MCP server discovery scenarios in depth (listed as an open question).

#### **3. Technology and Design Review**

- [ ] **Developer Handoff/QE Kickoff**
  - A meeting where Dev/Arch walked QE through the design, architecture, and implementation details. **Critical for identifying untestable aspects early.**
  - No code changes to review. The problem doc is self-contained and describes the security design space for MCP configuration integrity.
- [ ] **Technology Challenges**
  - Identified potential testing challenges related to the underlying technology.
  - No technology challenges for this documentation PR. Future implementation of MCP drift detection (hashing, content validation) will introduce testability considerations when those features are built.
- [ ] **Test Environment Needs**
  - Determined necessary **test environment setups and tools**.
  - Standard GitHub environment sufficient. No cluster, sandbox, or special tooling needed for documentation validation.
- [ ] **API Extensions**
  - Reviewed new or modified APIs and their impact on testing.
  - No API changes. The document references existing APIs (ToolAllowlistPreToolHook, SSRFPreToolHook, GenerateClaudeSettings) but does not modify them.
- [ ] **Topology Considerations**
  - Evaluated multi-cluster, network topology, and architectural impacts.
  - No topology considerations. Documentation-only change.

### **II. Software Test Plan (STP)**

This STP serves as the **overall roadmap for testing**, detailing the scope, approach, resources, and schedule.

#### **1. Scope of Testing**

Testing scope covers the documentation artifacts introduced by this PR: the new problem document `docs/problems/mcp-config-drift.md` and the README index update. Validation focuses on document rendering, link integrity, and content accuracy with respect to the existing codebase components referenced (security hooks, harness configuration, sandbox module).

**Testing Goals**

- **P0:** Verify all internal cross-references in the problem doc resolve to existing files
- **P0:** Verify the README index entry correctly links to the new document
- **P1:** Verify the document accurately references existing security components (ToolAllowlistPreToolHook, SSRFPreToolHook, GenerateClaudeSettings)
- **P2:** Verify the document renders correctly as Markdown (headings, links, lists)

**Out of Scope (Testing Scope Exclusions)**

- [ ] **MCP drift detection implementation** — The document describes future defense approaches (baseline hashing, immutable harness input, content-aware validation) that are not implemented in this PR. Testing of those features will be covered when they are built.
- [ ] **Upstream PR validation** — This is a mirror of upstream PR #2011 on fullsend-ai/fullsend. Upstream review and validation is out of scope.
- [ ] **Security hook functional testing** — The existing security hooks (SSRF validator, tool allowlist) referenced in the document are not modified. Functional testing of those hooks is covered by their own test suites.

#### **2. Test Strategy**

**Functional**

- [ ] **Functional Testing** — Validates that the feature works according to specified requirements and user stories
  - *Details:* Verify document content, structure, and link integrity. Validate README index is correctly updated.
- [ ] **Automation Testing** — Confirms test automation plan is in place for CI and regression coverage (all tests are expected to be automated)
  - *Details:* Link validation can be automated via CI markdown lint checks. No custom test automation needed for documentation.
- [ ] **Regression Testing** — Verifies that new changes do not break existing functionality
  - *Details:* Verify README index ordering is preserved and no existing entries are displaced.

**Non-Functional**

- [ ] **Performance Testing** — Validates feature performance meets requirements (latency, throughput, resource usage)
  - *Details:* Not applicable — documentation-only change.
- [ ] **Scale Testing** — Validates feature behavior under increased load and at production-like scale (e.g., large number of resources, nodes, or concurrent operations)
  - *Details:* Not applicable — documentation-only change.
- [ ] **Security Testing** — Verifies security requirements, RBAC, authentication, authorization, and vulnerability scanning
  - *Details:* Not applicable for this PR. The document itself describes security considerations but no security controls are implemented.
- [ ] **Usability Testing** — Validates user experience and accessibility requirements
  - *Details:* Not applicable — documentation-only change.
- [ ] **Monitoring** — Does the feature require metrics and/or alerts?
  - *Details:* Not applicable — documentation-only change.

**Integration & Compatibility**

- [ ] **Compatibility Testing** — Ensures feature works across supported platforms, versions, and configurations
  - *Details:* Not applicable — Markdown renders consistently across platforms.
- [ ] **Upgrade Testing** — Validates upgrade paths from previous versions, data migration, and configuration preservation
  - *Details:* Not applicable — documentation-only change.
- [ ] **Dependencies** — Blocked by deliverables from other components/products. Identify what we need from other teams before we can test.
  - *Details:* No dependencies. The referenced documents (security-threat-model.md, agent-architecture.md, ADR 0017) already exist.
- [ ] **Cross Integrations** — Does the feature affect other features or require testing by other teams? Identify the impact we cause.
  - *Details:* No cross-integration impact. The document adds to the problem space documentation without modifying any features.

**Infrastructure**

- [ ] **Cloud Testing** — Does the feature require multi-cloud platform testing? Consider cloud-specific features.
  - *Details:* Not applicable — documentation-only change.

#### **3. Test Environment**

- **Cluster Topology:** None required (documentation validation only)
- **Platform & Product Version(s):** FullSend 0.x on GitHub Actions
- **CPU Virtualization:** Not applicable
- **Compute Resources:** Not applicable
- **Special Hardware:** Not applicable
- **Storage:** Not applicable
- **Network:** Not applicable
- **Required Operators:** Not applicable
- **Platform:** GitHub (for link validation and rendering)
- **Special Configurations:** None

#### **3.1. Testing Tools & Frameworks**

- **Test Framework:** None required
- **CI/CD:** Standard GitHub Actions CI
- **Other Tools:** None

#### **4. Entry Criteria**

The following conditions must be met before testing can begin:

- [ ] Requirements and design documents are **approved and merged**
- [ ] Test environment can be **set up and configured** (see Section II.3 - Test Environment)
- [ ] The problem document and README changes are available in the PR branch for review

#### **5. Risks**

- [ ] **Timeline/Schedule**
  - Risk: Minimal risk — documentation PRs have no deployment timeline pressure.
  - Mitigation: None needed.
- [ ] **Test Coverage**
  - Risk: Limited testable surface — this is a documentation-only PR with no functional code.
  - Mitigation: Focus on link integrity and content accuracy validation.
- [ ] **Test Environment**
  - Risk: No environment risks for documentation validation.
  - Mitigation: None needed.
- [ ] **Untestable Aspects**
  - Risk: The defense approaches described in the document (baseline hashing, immutable harness input, content-aware validation) cannot be validated as they are not implemented.
  - Mitigation: Track implementation of MCP drift detection as a separate feature with its own STP.
- [ ] **Resource Constraints**
  - Risk: No resource constraints for documentation review.
  - Mitigation: None needed.
- [ ] **Dependencies**
  - Risk: Cross-referenced documents (security-threat-model.md, agent-architecture.md, ADR 0017) could be moved or renamed, breaking links.
  - Mitigation: Validate all relative links resolve correctly at merge time.
- [ ] **Other**
  - Risk: No additional risks identified.
  - Mitigation: None.

---

### **III. Test Scenarios & Traceability**

This section links requirements to test coverage, enabling reviewers to verify all requirements are tested.

#### **1. Requirements-to-Tests Mapping**

- **GH-17** — MCP configuration drift problem document is complete and accurate
  - **Scenario:** Verify all cross-reference links in problem doc resolve to existing files
  - **Tier:** Functional
  - **Priority:** P0

- — README index entry is correctly added
  - **Scenario:** Verify README links to docs/problems/mcp-config-drift.md
  - **Tier:** Functional
  - **Priority:** P0

- — Document accurately references existing security components
  - **Scenario:** Verify references to ToolAllowlistPreToolHook, SSRFPreToolHook, and GenerateClaudeSettings match codebase
  - **Tier:** Functional
  - **Priority:** P1

- — Document structure follows problem doc conventions
  - **Scenario:** Verify document contains required sections (problem statement, attack scenarios, defense considerations, open questions)
  - **Tier:** Functional
  - **Priority:** P1

- — Cross-reference integrity with security documentation
  - **Scenario:** Verify links to security-threat-model.md, agent-architecture.md, and ADR 0017 resolve correctly
  - **Tier:** Functional
  - **Priority:** P0

- — Document content accuracy for existing defenses
  - **Scenario:** Verify description of existing defense mechanisms (SSRF validator, tool allowlist hook, credential isolation) matches current implementation
  - **Tier:** Functional
  - **Priority:** P1

---

### **IV. Sign-off and Approval**

This Software Test Plan requires approval from the following stakeholders:

* **Reviewers:**
  - [Name / @github-username]
  - [Name / @github-username]
* **Approvers:**
  - [Name / @github-username]
  - [Name / @github-username]
