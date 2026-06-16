# FullSend Test Plan

## **MCP Configuration Drift Problem Document - Quality Engineering Plan**

### **Metadata & Tracking**

- **Enhancement(s):** [GH-17](https://github.com/fullsend-ai/fullsend/pull/17) — docs(problems): add MCP configuration drift problem doc
- **Feature Tracking:** [GH-17](https://github.com/fullsend-ai/fullsend/pull/17)
- **Epic Tracking:** N/A (standalone problem document)
- **QE Owner:** QualityFlow (automated)
- **Owning SIG:** sig-security
- **Participating SIGs:** sig-architecture

**Document Conventions:** This STP covers a documentation-only change. Test scenarios validate document structure, cross-reference integrity, and accuracy of codebase references rather than runtime behavior.

### **Feature Overview**

This PR introduces a new problem document (`docs/problems/mcp-config-drift.md`) that analyzes MCP (Model Context Protocol) configuration drift — a security concern where MCP server configurations fall out of sync across environments, potentially expanding an agent's tool surface without approval. The document defines four attack scenarios, three defense approaches with trade-offs, and relates the problem to FullSend's existing security hooks and harness architecture. The PR also removes the top-level `CLAUDE.md` file and updates `README.md` to index the new document.

---

### **I. Motivation and Requirements Review (QE Review Guidelines)**

#### 1. Requirement & User Story Review Checklist

- [ ] **Reviewed the relevant requirements.** -- Reviewed the associated Jira issue and/or other requirements documentation to understand what is expected.
  - GH-17 adds a problem document analyzing MCP configuration drift as a security vector. The PR body describes the change and references upstream PR #2011.
- [ ] **Confirmed clear user stories and understood. Understand the value and customer use cases.** -- Confirmed that the user stories are clear and understood. Understand the value proposition and customer use cases.
  - The value is documenting a security gap in FullSend's agent harness: MCP configs define the agent tool surface and are not currently monitored for unauthorized changes. This affects any organization using FullSend with MCP integrations.
- [ ] **Confirmed requirements are **testable and unambiguous**.** -- Verified requirements are specific enough to derive test cases from, without ambiguity.
  - Requirements are testable: document structure, link resolution, and codebase reference accuracy can all be validated programmatically.
- [ ] **Ensured acceptance criteria are **defined clearly**.** -- Verified acceptance criteria exist and are precise enough for pass/fail determination.
  - Acceptance criteria are implicit: the document must follow the problem document format, all cross-references must resolve, and the README index must be updated.
- [ ] **Confirmed coverage for NFRs.** -- Checked for non-functional requirements (performance, security, usability, etc.) and ensured they are addressed.
  - No runtime NFRs apply. The document itself is an NFR artifact — it captures security design considerations for future implementation.

#### 2. Known Limitations

- This is a design exploration document, not a specification. The attack scenarios and defense approaches are not yet implemented in code.
- Cross-reference link validation can only verify file existence, not semantic correctness of the referenced content.
- Security component references (e.g., `ToolAllowlistPreToolHook`) are validated against the current codebase state; future refactoring may invalidate them.

#### 3. Technology and Design Review

- [ ] **Developer handoff completed.** -- Met with the developer(s) to understand the implementation approach, architecture, and any areas of concern.
  - PR authored by guyoron1. Implementation is documentation-only — a new markdown file in `docs/problems/` with README index update.
- [ ] **Identified technology challenges.** -- Reviewed the technology stack and identified any new or complex technologies that may impact testing.
  - No new technologies. Standard markdown documentation following the existing problem document pattern.
- [ ] **Reviewed test environment needs.** -- Assessed whether new infrastructure, tools, or configurations are needed for testing.
  - No special test environment needed. Tests run as Go unit tests that read and validate files from the repository.
- [ ] **Reviewed API or interface extensions.** -- Checked for new or modified APIs, CLIs, or interfaces that need test coverage.
  - No API or interface changes. The PR modifies documentation files only.
- [ ] **Reviewed topology requirements.** -- Assessed whether specific cluster or network topologies are needed for testing.
  - No topology requirements. All tests are file-based and run locally.

### **II. Software Test Plan (STP)**

#### 1. Scope of Testing

This test plan covers validation of the MCP configuration drift problem document introduced in GH-17. Testing focuses on document structural completeness, cross-reference link integrity, README index accuracy, and correctness of references to existing security components in the FullSend codebase. The deletion of `CLAUDE.md` is also validated to ensure no broken references remain.

**Testing Goals:**

- **P0:** Verify problem document contains all required sections and follows the established format
- **P0:** Verify all cross-reference links resolve to existing files in the repository
- **P0:** Verify README.md index entry links to the new document correctly
- **P1:** Verify security component references in the document match actual code symbols
- **P1:** Verify CLAUDE.md deletion does not leave broken references

**Out of Scope (Testing Scope Exclusions):**

- [ ] **Runtime MCP drift detection behavior** -- No drift detection is implemented yet; this PR is a design document only.
- [ ] **Security hook functional testing** -- Existing security hooks (SSRF, ToolAllowlist, etc.) have their own test suites and are not modified by this PR.
- [ ] **Harness configuration loading** -- `internal/harness/harness.go` is not modified; existing `harness_test.go` covers this.
- [ ] **Upstream PR #2011 validation** -- The upstream PR in fullsend-ai/fullsend is outside the scope of this fork's test plan.

#### 2. Test Strategy

**Functional:**

- [ ] **Functional Testing** -- Applicable. Validate document structure, content sections, and cross-reference integrity through file-based assertions.
- [ ] **Automation Testing** -- Applicable. All tests are automated as Go test functions using Ginkgo/Gomega for file content validation.
- [ ] **Regression Testing** -- Applicable. Verify CLAUDE.md deletion does not introduce broken references elsewhere in the repository.

**Non-Functional:**

- [ ] **Performance Testing** -- Not applicable. Documentation change with no runtime impact.
- [ ] **Scale Testing** -- Not applicable. No scalability concerns for static documentation.
- [ ] **Security Testing** -- Not directly applicable. The document itself analyzes security concerns but introduces no new attack surface.
- [ ] **Usability Testing** -- Not applicable. No UI or CLI changes.
- [ ] **Monitoring** -- Not applicable. No observable metrics affected.

**Integration & Compatibility:**

- [ ] **Compatibility Testing** -- Not applicable. Markdown documentation is format-agnostic.
- [ ] **Upgrade Testing** -- Not applicable. No version-sensitive changes.
- [ ] **Dependencies** -- Not applicable. No blocked deliverables from other teams.
- [ ] **Cross Integrations** -- Not applicable. No cross-component runtime interactions.

**Infrastructure:**

- [ ] **Cloud Testing** -- Not applicable. No cloud infrastructure involved.

#### 3. Test Environment

- **Cluster Topology:** None required — all tests run locally
- **Platform Version:** GitHub Actions (standard runner)
- **CPU Virtualization:** N/A
- **Compute:** Standard CI runner
- **Special Hardware:** None
- **Storage:** Local filesystem access to repository
- **Network:** None required
- **Operators:** None
- **Platform:** Go 1.23+ with Ginkgo/Gomega test framework
- **Special Configs:** Repository must be fully cloned (not shallow) for cross-reference validation

##### 3.1. Testing Tools & Frameworks

No new or special tools required. Standard Go test infrastructure with Ginkgo/Gomega.

#### 4. Entry Criteria

- [ ] PR branch is mergeable and CI passes lint checks (`make lint`)
- [ ] All referenced files in cross-reference links exist in the repository
- [ ] Go test dependencies (Ginkgo, Gomega) are available in the test environment
- [ ] Referenced code symbols (`ToolAllowlistPreToolHook`, `GenerateClaudeSettings`) exist in the codebase

#### 5. Risks

- [ ] **Timeline**
  - Risk: Document may reference files or ADRs that don't exist yet in this fork (mirrored from upstream)
  - Mitigation: Cross-reference tests will catch missing targets; known missing files can be excluded
  - Status: [ ] Monitoring

- [ ] **Coverage**
  - Risk: Semantic accuracy of security component descriptions cannot be fully validated through automated tests
  - Mitigation: Manual review of security claims against code behavior during PR review
  - Status: [ ] Accepted

- [ ] **Environment**
  - Risk: Shallow git clones in CI may miss files needed for cross-reference validation
  - Mitigation: Ensure full clone depth in CI configuration
  - Status: [ ] Monitoring

- [ ] **Untestable**
  - Risk: The correctness of the threat analysis itself (are the attack scenarios realistic?) is not testable
  - Mitigation: Peer review by security-focused team members
  - Status: [ ] Accepted

- [ ] **Resources**
  - Risk: No dedicated QE resource for documentation validation
  - Mitigation: Automated tests provide baseline coverage; QualityFlow pipeline generates and maintains tests
  - Status: [ ] Mitigated

- [ ] **Dependencies**
  - Risk: Document references ADR 0016 and ADR 0017 which must exist in `docs/ADRs/`
  - Mitigation: Cross-reference link tests will fail explicitly if ADRs are missing
  - Status: [ ] Monitoring

- [ ] **Other**
  - Risk: CLAUDE.md deletion may affect developer onboarding workflows that reference it
  - Mitigation: Grep-based test verifies no remaining references to deleted file
  - Status: [ ] Monitoring

---

### **III. Test Scenarios & Traceability**

#### **1. Requirements-to-Tests Mapping**

- **GH-17** — As a contributor, I want the MCP config drift problem document to follow the established format so that it is consistent with other problem documents
  - TS-GH-17-001: Verify document contains all required sections (positive) — [Functional] — P0
  - TS-GH-17-002: Verify document follows problem document structure (positive) — [Functional] — P0
  - TS-GH-17-003: Verify document is not empty or malformed (negative) — [Functional] — P0

- **GH-17** — As a reader, I want all cross-reference links in the problem document to resolve to existing files so that I can navigate related content
  - TS-GH-17-004: Verify all relative links resolve to existing files (positive) — [Functional] — P0
  - TS-GH-17-005: Verify anchor fragments reference valid headings (positive) — [Functional] — P0
  - TS-GH-17-006: Verify no broken internal links exist (negative) — [Functional] — P0

- **GH-17** — As a reader, I want the README.md index to include the MCP config drift entry so that I can discover the document
  - TS-GH-17-007: Verify README contains link to mcp-config-drift.md (positive) — [Functional] — P0
  - TS-GH-17-008: Verify linked target file exists on disk (positive) — [Functional] — P0

- **GH-17** — As a contributor, I want security component references to accurately reflect the codebase so that the document remains trustworthy
  - TS-GH-17-009: Verify ToolAllowlistPreToolHook reference matches code (positive) — [Functional] — P1
  - TS-GH-17-010: Verify SSRF validator reference matches code (positive) — [Functional] — P1
  - TS-GH-17-011: Verify no references to non-existent components (negative) — [Functional] — P1

- **GH-17** — As a reader, I want the attack scenarios to be distinct and cover the MCP threat surface so that I understand the security risks
  - TS-GH-17-012: Verify document contains distinct attack scenarios (positive) — [Functional] — P2
  - TS-GH-17-013: Verify each scenario has a clear description (positive) — [Functional] — P2

- **GH-17** — As a reader, I want defense approaches to include trade-offs analysis so that I can evaluate implementation options
  - TS-GH-17-014: Verify each defense approach has trade-offs section (positive) — [Functional] — P2
  - TS-GH-17-015: Verify multiple defense approaches are presented (positive) — [Functional] — P2

- **GH-17** — As a contributor, I want the CLAUDE.md removal to not break repository documentation references so that existing workflows remain functional
  - TS-GH-17-016: Verify no remaining references to deleted CLAUDE.md (negative) — [Functional] — P1
  - TS-GH-17-017: Verify repository documentation integrity after deletion (positive) — [Functional] — P1

---

### **IV. Sign-off and Approval**

This Software Test Plan requires approval from the following stakeholders:

* **Reviewers:**
  - [Name / @github-username]
  - [Name / @github-username]
* **Approvers:**
  - [Name / @github-username]
  - [Name / @github-username]
