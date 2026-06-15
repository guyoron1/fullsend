# FullSend Test Plan

## **MCP Configuration Drift Problem Document - Quality Engineering Plan**

### **Metadata & Tracking**

- **Enhancement(s):** [PR #2011](https://github.com/fullsend-ai/fullsend/pull/2011) (fork: [GH-13](https://github.com/guyoron1/fullsend/pull/13))
- **Feature Tracking:** [GH-13](https://github.com/guyoron1/fullsend/pull/13)
- **Epic Tracking:** None
- **QE Owner(s):** TBD
- **Owning SIG:** N/A
- **Participating SIGs:** None

**Document Conventions (if applicable):** N/A

### **Feature Overview**

This PR introduces a problem document (`docs/problems/mcp-config-drift.md`) describing MCP configuration drift as a security threat vector for the FullSend harness. This is a documentation-only change with no code modifications; testing focuses on document accuracy, cross-reference integrity, and alignment with the existing codebase security architecture. For full context on attack scenarios and defense approaches, see the problem document itself.

---

### **I. Motivation and Requirements Review (QE Review Guidelines)**

This section documents the mandatory QE review process. The goal is to understand the feature's value,
technology, and testability before formal test planning.

#### **1. Requirement & User Story Review Checklist**

- [x] **Review Requirements**
  - Reviewed the relevant requirements.
  - The PR adds a problem document (`docs/problems/mcp-config-drift.md`) describing MCP configuration drift as a security threat vector. Requirements are implicit: the document must accurately describe the problem, attack scenarios, and defense considerations.
- [x] **Understand Value and Customer Use Cases**
  - Confirmed clear user stories and understood.
  - Understand the difference between community and product requirements.
  - **What is the value of the feature for customers**.
  - Ensured requirements contain relevant **customer use cases**.
  - Value: MCP configs define the agent tool surface; undetected drift enables privilege escalation, data exfiltration, and silent failures. This document provides the analytical foundation for implementing defenses in the harness.
- [x] **Testability**
  - Confirmed requirements are **testable and unambiguous**.
  - As a documentation change, testability focuses on: (1) factual accuracy of claims about existing code (hooks, security config), (2) cross-reference integrity (links to other docs, ADRs), (3) consistency with the codebase architecture.
- [x] **Acceptance Criteria**
  - Ensured acceptance criteria are **defined clearly** (clear user stories; product requirements clearly defined in Jira).
  - Acceptance criteria inferred from the PR: document is well-structured, accurately references existing security mechanisms, and follows the problem doc format used by other docs in `docs/problems/`.
- [x] **Non-Functional Requirements (NFRs)**
  - Confirmed coverage for NFRs, including Performance, Security, Usability, Downtime, Connectivity, Monitoring (alerts/metrics), Scalability, Portability (e.g., cloud support), and Docs.
  - This is a documentation-only change. NFRs are not directly applicable. Documentation quality (clarity, accuracy, completeness) is the primary non-functional concern.

#### **2. Known Limitations**

- This is a problem document only — no implementation is included. The defense approaches described (baseline-and-diff, immutable harness input, content-aware validation) are proposals, not implemented features.
- The document references ADR 0017 (credential isolation) and ADR 0016 (unidirectional control flow). Both ADRs exist in the repository; however, their content may diverge from upstream over time.
- No automated tooling exists to validate MCP config drift; this document is the first step toward building such tooling.

#### **3. Technology and Design Review**

- [x] **Developer Handoff/QE Kickoff**
  - A meeting where Dev/Arch walked QE through the design, architecture, and implementation details. **Critical for identifying untestable aspects early.**
  - No code changes to hand off. The document describes security architecture concepts that relate to existing harness security hooks in `internal/security/hooks.go` and harness configuration in `internal/harness/harness.go`.
- [x] **Technology Challenges**
  - Identified potential testing challenges related to the underlying technology.
  - The primary challenge is verifying that the document's claims about existing security hook behavior and SSRF validator coverage are accurate against the actual codebase.
- [x] **Test Environment Needs**
  - Determined necessary **test environment setups and tools**.
  - No special test environment needed. Validation requires access to the repository source code for cross-reference checking.
- [x] **API Extensions**
  - Reviewed new or modified APIs and their impact on testing.
  - No API changes. The document references existing APIs and hooks.
- [x] **Topology Considerations**
  - Evaluated multi-cluster, network topology, and architectural impacts.
  - N/A — documentation-only change.

### **II. Software Test Plan (STP)**

This STP serves as the **overall roadmap for testing**, detailing the scope, approach, resources, and schedule.

#### **1. Scope of Testing**

Testing validates the accuracy, completeness, and cross-reference integrity of the MCP configuration drift problem document. Since this is a documentation-only PR with no code changes, testing focuses on ensuring the document correctly describes the existing security architecture and that all internal and external references resolve correctly.

**Testing Goals**

**Functional Goals:**
- **P0:** Verify all cross-references to other problem docs, ADRs, and code components are accurate and resolve correctly
- **P0:** Verify the document's claims about existing security hooks (`ToolAllowlistPreToolHook`, SSRF validator) match the actual codebase behavior
- **P1:** Verify the README.md index entry is correctly placed and links to the new document

**Quality Goals:**
- **P1:** Verify the document follows the established problem doc format used by existing docs in `docs/problems/`
- **P2:** Verify attack scenarios are technically sound and consistent with the threat model

**Integration Goals:**
- **P1:** Verify the document correctly maps relationships to `security-threat-model.md`, `agent-architecture.md`, and `governance.md`

**Out of Scope (Testing Scope Exclusions)**

- [ ] **Implementation testing of defense approaches** — *Rationale:* The document proposes three defense approaches (baseline-and-diff, immutable harness input, content-aware validation) but none are implemented. Testing implementation is out of scope until code is written. — *PM/Lead Agreement:* TBD
- [ ] **Upstream PR #2011 validation** — *Rationale:* This PR is mirrored from upstream `fullsend-ai/fullsend`. Upstream review and validation is the responsibility of the upstream maintainers. — *PM/Lead Agreement:* TBD
- [ ] **MCP protocol conformance testing** — *Rationale:* MCP protocol behavior is external to FullSend. The document describes MCP configuration management, not MCP protocol implementation. — *PM/Lead Agreement:* TBD

#### **2. Test Strategy**

**Functional**

- [ ] **Functional Testing** — Validates that the feature works according to specified requirements and user stories
  - *Details:* Verify document content accuracy: cross-references resolve, code references match codebase, attack scenarios are technically sound.
- [ ] **Automation Testing** — Confirms test automation plan is in place for CI and regression coverage (all tests are expected to be automated)
  - *Details:* Link validation can be automated via markdown linting. Code reference accuracy can be partially automated by grepping for referenced symbols.
- [ ] **Regression Testing** — Verifies that new changes do not break existing functionality
  - *Details:* Verify README.md modification does not break existing problem doc links or table of contents structure.

**Non-Functional**

- [ ] **Performance Testing** — Validates feature performance meets requirements (latency, throughput, resource usage)
  - *Details:* N/A — documentation-only change.
- [ ] **Scale Testing** — Validates feature behavior under increased load and at production-like scale (e.g., large number of resources, nodes, or concurrent operations)
  - *Details:* N/A — documentation-only change.
- [ ] **Security Testing** — Verifies security requirements, RBAC, authentication, authorization, and vulnerability scanning
  - *Details:* Verify the document does not inadvertently disclose sensitive implementation details that could aid attackers (e.g., specific endpoint URLs, credential paths).
- [ ] **Usability Testing** — Validates user experience and accessibility requirements
  - *Details:* Verify document is well-organized, readable, and follows the established problem doc structure.
- [ ] **Monitoring** — Does the feature require metrics and/or alerts?
  - *Details:* N/A — documentation-only change.

**Integration & Compatibility**

- [ ] **Compatibility Testing** — Ensures feature works across supported platforms, versions, and configurations
  - *Details:* N/A — documentation-only change.
- [ ] **Upgrade Testing** — Validates upgrade paths from previous versions, data migration, and configuration preservation
  - *Details:* N/A — documentation-only change.
- [ ] **Dependencies** — Blocked by deliverables from other components/products. Identify what we need from other teams before we can test.
  - *Details:* No external dependencies. Referenced ADRs (0016, 0017) and problem docs should exist in the repository.
- [ ] **Cross Integrations** — Does the feature affect other features or require testing by other teams? Identify the impact we cause.
  - *Details:* The document references and extends the security threat model. The security-threat-model.md may need a reciprocal cross-reference added in the future.

**Infrastructure**

- [ ] **Cloud Testing** — Does the feature require multi-cloud platform testing? Consider cloud-specific features.
  - *Details:* N/A — documentation-only change.

#### **3. Test Environment**

- **Cluster Topology:** N/A — documentation-only change
- **Platform & Product Version(s):** GitHub Actions (FullSend 0.x)
- **CPU Virtualization:** N/A
- **Compute Resources:** N/A
- **Special Hardware:** N/A
- **Storage:** N/A
- **Network:** N/A
- **Required Operators:** N/A
- **Platform:** GitHub
- **Special Configurations:** Access to full repository source for cross-reference validation

#### **3.1. Testing Tools & Frameworks**

- **Test Framework:** N/A — manual review and automated link checking
- **CI/CD:** GitHub Actions (existing)
- **Other Tools:** `markdownlint` for link validation (if available)

#### **4. Entry Criteria**

The following conditions must be met before testing can begin:

- [ ] Requirements and design documents are **approved and merged**
- [ ] Test environment can be **set up and configured** (see Section II.3 - Test Environment)
- [ ] Access to the repository source code for cross-reference validation of security hook and harness architecture claims

#### **5. Risks**

- [ ] **Timeline/Schedule**
  - Risk: N/A — documentation review is lightweight
  - Mitigation: N/A
- [ ] **Test Coverage**
  - Risk: Document may reference code patterns or security mechanisms that have changed since the document was written
  - Mitigation: Cross-reference validation against current codebase using LSP analysis and grep
- [ ] **Test Environment**
  - Risk: N/A
  - Mitigation: N/A
- [ ] **Untestable Aspects**
  - Risk: The technical soundness of proposed defense approaches (baseline-and-diff, immutable harness input, content-aware validation) cannot be fully validated without implementation
  - Mitigation: Review against established security patterns and industry best practices
- [ ] **Resource Constraints**
  - Risk: N/A
  - Mitigation: N/A
- [ ] **Dependencies**
  - Risk: Referenced ADRs (0016, 0017) content may diverge from upstream over time
  - Mitigation: Periodically verify fork ADR content aligns with upstream repository (fullsend-ai/fullsend)
- [ ] **Document Staleness**
  - Risk: Document claims about existing security hooks may become outdated as the codebase evolves (e.g., hooks refactored or extended)
  - Mitigation: Tag problem document for periodic review when security hooks are modified
- [ ] **Other**
  - Risk: Upstream PR #2011 may diverge from this mirrored version over time
  - Mitigation: Track upstream changes and update the fork as needed

---

### **III. Test Scenarios & Traceability**

This section links requirements to test coverage, enabling reviewers to verify all requirements are tested.

#### **1. Requirements-to-Tests Mapping**

- **[GH-13]** — Document accurately describes MCP configuration drift as a security threat vector with correct cross-references to existing security architecture
  - *Test Scenario:* Verify cross-references to security-threat-model.md, agent-architecture.md, and ADRs 0016/0017 are valid and link targets exist
  - *Priority:* P0

- **[GH-13]** — Document claims about existing security hooks match actual codebase implementation
  - *Test Scenario:* Verify the document's claims about the tool allowlist hook's operating mechanism (filtering by tool names, not server endpoints) are accurate against the actual codebase
  - *Priority:* P0

- **[GH-13]** — Document claims about SSRF validator coverage are accurate
  - *Test Scenario:* Verify the document's claims about SSRF validation scope (covering Bash and WebFetch but not MCP connections) are accurate against the actual codebase
  - *Priority:* P0

- **[GH-13]** — README.md index entry is correctly placed and formatted
  - *Test Scenario:* Verify the MCP Configuration Drift entry in README.md links correctly to docs/problems/mcp-config-drift.md and maintains alphabetical/logical ordering with adjacent entries
  - *Priority:* P1

- **[GH-13]** — Document follows established problem doc structure and conventions
  - *Test Scenario:* Verify document structure matches the format of existing problem docs (security-threat-model.md, agent-architecture.md) including section headings, related-doc links, and open questions format
  - *Priority:* P1

- **[GH-13]** — Attack scenarios are technically sound and non-overlapping
  - *Test Scenario:* Verify each of the four attack scenarios (malicious server injection, endpoint replacement, permission escalation, gradual drift) describes a distinct threat vector consistent with the MCP protocol model
  - *Priority:* P1

- **[GH-13]** — Defense approaches correctly reference harness architecture
  - *Test Scenario:* Verify the document's references to harness architecture components (Harness struct, SecurityConfig) are accurate against the actual codebase
  - *Priority:* P1

- **[GH-13]** — Described injection pattern aligns with harness initialization flow
  - *Test Scenario:* Verify the MCP config injection pattern described in Approach 2 (immutable harness input) is consistent with the existing harness initialization flow
  - *Priority:* P2

- **[GH-13]** — Document does not disclose sensitive implementation details
  - *Test Scenario:* Verify the document does not contain specific endpoint URLs, credential paths, internal network topology, or other implementation details that could aid attackers
  - *Priority:* P1

- **[GH-13]** — Open Questions section is complete and actionable
  - *Test Scenario:* Verify the Open Questions section contains actionable, non-redundant questions that align with the identified attack scenarios and defense approaches
  - *Priority:* P2

- **[GH-13]** — Document does not introduce broken links or formatting errors
  - *Test Scenario:* Verify all relative markdown links in mcp-config-drift.md resolve to existing files and all markdown formatting renders correctly
  - *Priority:* P2

---

### **IV. Sign-off and Approval**

This Software Test Plan requires approval from the following stakeholders:

* **Reviewers:**
  - [TBD / @reviewer]
* **Approvers:**
  - [TBD / @approver]
