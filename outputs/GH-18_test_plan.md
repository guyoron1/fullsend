# Test Plan

## **Tool Call Risk Assessment — Security Supply Chain Threat Model - Quality Engineering Plan**

### **Metadata & Tracking**

- **Enhancement(s):** [GH-18](https://github.com/guyoron1/fullsend/pull/18)
- **Feature Tracking:** [GH-18](https://github.com/guyoron1/fullsend/pull/18)
- **Epic Tracking:** GH-18
- **QE Owner(s):** QualityFlow (automated)
- **Owning SIG:** N/A
- **Participating SIGs:** N/A

**Document Conventions (if applicable):** N/A

### **Feature Overview**

This PR introduces a new problem document exploring tool call risk assessment — evaluating the risk level of individual tool invocations in agent workflows beyond static pattern matching. It proposes four approaches: LLM-as-judge, learned behavioral baselines, declarative tool call policies, and a hybrid approach. The PR also includes Go unit tests validating the existing security hook pipeline configuration, input/output pipeline integrity, context injection detection, and fail-closed behavior in the `internal/security` and `internal/harness` packages.

---

### **I. Motivation and Requirements Review (QE Review Guidelines)**

This section documents the mandatory QE review process. The goal is to understand the feature's value,
technology, and testability before formal test planning.

#### **1. Requirement & User Story Review Checklist**

- [ ] **Review Requirements**
  - Reviewed the relevant requirements.
  - PR #18 adds a problem document (`docs/problems/tool-call-risk-assessment.md`) and Go test files covering security hook pipeline, input/output pipeline, context injection, and provider loading.
- [ ] **Understand Value and Customer Use Cases**
  - Confirmed clear user stories and understood.
  - Understand the difference between community and product requirements.
  - **What is the value of the feature for customers**.
  - Ensured requirements contain relevant **customer use cases**.
  - The tool call risk assessment problem addresses the gap between static pattern matching and context-dependent security evaluation in agent workflows. Test coverage validates that existing security infrastructure (hook pipeline, scanners, redactors) behaves correctly.
- [ ] **Testability**
  - Confirmed requirements are **testable and unambiguous**.
  - All security functions under test (`GenerateClaudeSettings`, `InputPipeline`, `OutputPipeline`, `NewContextInjectionScanner`, `LoadProviderDefs`, `SecurityEnabled`, `FailModeClosed`, `BoolDefault`, `HasCriticalFindings`) are pure Go functions with deterministic inputs/outputs, making them highly testable.
- [ ] **Acceptance Criteria**
  - Ensured acceptance criteria are **defined clearly** (clear user stories; product requirements clearly defined in Jira).
  - Acceptance criteria are implicit in the test assertions: default hooks are enabled, toggles isolate correctly, nil configs default to fail-closed, pipeline ordering is normalizer-before-scanner, secrets are redacted, and injection patterns are detected with correct severity.
- [ ] **Non-Functional Requirements (NFRs)**
  - Confirmed coverage for NFRs, including Performance, Security, Usability, Downtime, Connectivity, Monitoring (alerts/metrics), Scalability, Portability (e.g., cloud support), and Docs.
  - Security is the primary NFR: fail-closed defaults, nil-safety, injection detection severity classification, and secret redaction are all validated. Performance and scale are not applicable to these unit-level security functions.

#### **2. Known Limitations**

- The problem document is exploratory — no implementation of tool call risk assessment exists yet. Test coverage addresses only the existing security infrastructure, not the proposed approaches.
- LSP analysis was limited by sandbox network restrictions (Go module dependencies could not be downloaded), but symbol resolution succeeded for the key packages.
- The `CLAUDE.md` file was deleted in this PR; its conventions may need to be relocated.

#### **3. Technology and Design Review**

- [ ] **Developer Handoff/QE Kickoff**
  - A meeting where Dev/Arch walked QE through the design, architecture, and implementation details. **Critical for identifying untestable aspects early.**
  - PR #18 is a documentation and test PR. The security architecture is documented in `docs/problems/tool-call-risk-assessment.md` and the code is in `internal/security/` and `internal/harness/`.
- [ ] **Technology Challenges**
  - Identified potential testing challenges related to the underlying technology.
  - No significant challenges. All tests use standard Go/Ginkgo patterns with in-memory structs and temporary directories.
- [ ] **Test Environment Needs**
  - Determined necessary **test environment setups and tools**.
  - Standard Go test environment with Ginkgo v2 and Gomega. No cluster or external services required.
- [ ] **API Extensions**
  - Reviewed new or modified APIs and their impact on testing.
  - No API changes. Tests exercise existing internal Go APIs (`security.GenerateClaudeSettings`, `security.InputPipeline`, `harness.LoadProviderDefs`, etc.).
- [ ] **Topology Considerations**
  - Evaluated multi-cluster, network topology, and architectural impacts.
  - Not applicable. All tests are unit-level and run in a single process.

### **II. Software Test Plan (STP)**

This STP serves as the **overall roadmap for testing**, detailing the scope, approach, resources, and schedule.

#### **1. Scope of Testing**

Testing validates the correctness of fullsend's security infrastructure: the hook pipeline configuration generator, input/output scanning pipelines, context injection detection, secret redaction, Unicode normalization ordering, fail-closed defaults, provider definition loading, and tool allowlist toggle behavior.

**Testing Goals**

- **P0 — Functional Goals:**
  - Verify security hook pipeline generates correct Claude settings JSON with all default hooks enabled
  - Verify nil and zero-value security configs default to fail-closed, security-enabled state
  - Verify input pipeline chains Unicode normalizer before injection scanner to catch evasion
  - Verify context injection scanner classifies patterns with correct severity levels
- **P1 — Functional Goals:**
  - Verify output pipeline redacts API keys and GitHub PATs from agent output
  - Verify pipeline finding aggregation and `HasCriticalFindings` helper work correctly
  - Verify model provider definitions load with credential mapping and validation
  - Verify tool allowlist hook is disabled by default and can be explicitly enabled

**Out of Scope (Testing Scope Exclusions)**

- [ ] Implementation of the proposed tool call risk assessment approaches (LLM-as-judge, behavioral baseline, declarative policies) -- *Rationale:* These are exploratory proposals in the problem document; no implementation exists yet -- *PM/Lead Agreement:* N/A
- [ ] End-to-end testing of security hooks in a live sandbox environment -- *Rationale:* Tests validate the configuration generation logic, not runtime hook execution -- *PM/Lead Agreement:* N/A
- [ ] Performance benchmarking of scanner pipelines -- *Rationale:* Unit-level functions with negligible performance characteristics -- *PM/Lead Agreement:* N/A

#### **2. Test Strategy**

**Functional**

- [x] **Functional Testing** — Validates that the feature works according to specified requirements and user stories
  - *Details:* 33 unit test scenarios covering 9 requirements across security hook pipeline, input/output pipelines, context injection, provider loading, and configuration defaults.
- [x] **Automation Testing** — Confirms test automation plan is in place for CI and regression coverage (all tests are expected to be automated)
  - *Details:* All tests are automated Go/Ginkgo unit tests in `qf-tests/GH-18/go/`. Tests run via `go test` or `ginkgo` in CI.
- [x] **Regression Testing** — Verifies that new changes do not break existing functionality
  - *Details:* LSP analysis traced `GenerateClaudeSettings` callers to `bootstrapSecurityHooks` in `internal/cli/run.go:1338` and existing test suites in `internal/security/hooks_test.go` (10 test functions) and `internal/harness/harness_test.go`.

**Non-Functional**

- [ ] **Performance Testing** — Validates feature performance meets requirements (latency, throughput, resource usage)
  - *Details:* Not applicable. Unit-level functions with negligible latency.
- [ ] **Scale Testing** — Validates feature behavior under increased load and at production-like scale
  - *Details:* Not applicable. No scale dimension for configuration generation functions.
- [x] **Security Testing** — Verifies security requirements, RBAC, authentication, authorization, and vulnerability scanning
  - *Details:* Core focus of this STP. Tests validate fail-closed defaults, injection detection, secret redaction, and nil-safety — all security-critical behaviors.
- [ ] **Usability Testing** — Validates user experience and accessibility requirements
  - *Details:* Not applicable. Internal library APIs, not user-facing.
- [ ] **Monitoring** — Does the feature require metrics and/or alerts?
  - *Details:* Not applicable. No monitoring requirements for these functions.

**Integration & Compatibility**

- [ ] **Compatibility Testing** — Ensures feature works across supported platforms, versions, and configurations
  - *Details:* Not applicable. Pure Go functions with no platform-specific dependencies.
- [ ] **Upgrade Testing** — Validates upgrade paths from previous versions, data migration, and configuration preservation
  - *Details:* Not applicable. No data migration or upgrade paths involved.
- [ ] **Dependencies** — Blocked by deliverables from other components/products
  - *Details:* Tests depend on `internal/harness` and `internal/security` packages. Both are available and stable.
- [ ] **Cross Integrations** — Does the feature affect other features or require testing by other teams?
  - *Details:* `GenerateClaudeSettings` is called by `bootstrapSecurityHooks` in `internal/cli/run.go`. Changes to hook configuration logic could affect sandbox security setup.

**Infrastructure**

- [ ] **Cloud Testing** — Does the feature require multi-cloud platform testing?
  - *Details:* Not applicable. No cloud-specific dependencies.

#### **3. Test Environment**

- **Cluster Topology:** N/A (unit tests, no cluster required)
- **Platform & Product Version(s):** Go 1.22+, Ginkgo v2, Gomega
- **CPU Virtualization:** N/A
- **Compute Resources:** Standard CI runner
- **Special Hardware:** N/A
- **Storage:** Temporary directories for provider definition tests
- **Network:** N/A (no network access required)
- **Required Operators:** N/A
- **Platform:** Linux (CI runner)
- **Special Configurations:** N/A

#### **3.1. Testing Tools & Frameworks**

- **Test Framework:** N/A (standard Ginkgo v2 + Gomega)
- **CI/CD:** N/A (standard CI pipeline)
- **Other Tools:** N/A

#### **4. Entry Criteria**

The following conditions must be met before testing can begin:

- [ ] Requirements and design documents are **approved and merged**
- [ ] Test environment can be **set up and configured** (see Section II.3 - Test Environment)
- [ ] `internal/security` and `internal/harness` packages compile successfully with all dependencies

#### **5. Risks**

- [ ] **Timeline/Schedule**
  - Risk: N/A — tests are already written and included in the PR
  - Mitigation: N/A
- [ ] **Test Coverage**
  - Risk: Tests cover existing security infrastructure but not the proposed risk assessment approaches described in the problem document
  - Mitigation: Future PRs implementing risk assessment approaches will require their own STPs with dedicated test coverage
- [ ] **Test Environment**
  - Risk: Go module dependencies may be unavailable in restricted sandbox environments
  - Mitigation: Ensure CI runners have network access to Go module proxy or use vendored dependencies
- [ ] **Untestable Aspects**
  - Risk: Runtime behavior of security hooks in a live Claude Code sandbox cannot be validated via unit tests
  - Mitigation: Integration/E2E tests in a sandbox environment should be considered separately
- [ ] **Resource Constraints**
  - Risk: N/A — unit tests require minimal resources
  - Mitigation: N/A
- [ ] **Dependencies**
  - Risk: Tests import `internal/harness` and `internal/security` packages; breaking changes to these packages would break tests
  - Mitigation: Tests are co-located in the same repo; CI will catch compilation failures
- [ ] **Other**
  - Risk: Deletion of `CLAUDE.md` in this PR removes repository conventions that developers may rely on
  - Mitigation: Verify conventions are documented elsewhere or restore the file

---

### **III. Test Scenarios & Traceability**

This section links requirements to test coverage, enabling reviewers to verify all requirements are tested.

#### **1. Requirements-to-Tests Mapping**

- **[GH-18]** — Security hook pipeline generates correct Claude settings with all default hooks enabled
  - *Test Scenario:* Verify default settings include all expected hooks
  - *Test Scenario:* Verify default PreToolUse hook count and matchers
  - *Test Scenario:* Verify default PostToolUse chain structure
  - *Tier:* Unit Tests
  - *Priority:* P0

- **[GH-18]** — Individual security hook toggles disable only the targeted hook without affecting others
  - *Test Scenario:* Verify single hook disable leaves others enabled
  - *Test Scenario:* Verify all hooks disabled produces empty hook map
  - *Test Scenario:* Verify re-enabling disabled hook restores it
  - *Tier:* Unit Tests
  - *Priority:* P0

- **[GH-18]** — Security configuration handles nil and zero-value configs safely with fail-closed defaults
  - *Test Scenario:* Verify nil SecurityConfig defaults to fail-closed
  - *Test Scenario:* Verify nil Enabled pointer defaults to enabled
  - *Test Scenario:* Verify empty FailMode defaults to closed
  - *Test Scenario:* Verify explicit false overrides default true
  - *Tier:* Unit Tests
  - *Priority:* P0

- **[GH-18]** — Input pipeline chains Unicode normalizer before injection scanner to detect evasion attempts
  - *Test Scenario:* Verify normalizer runs before injection scanner
  - *Test Scenario:* Verify injection hidden by zero-width chars detected
  - *Test Scenario:* Verify sanitized output propagates between stages
  - *Test Scenario:* Verify clean input passes through safely
  - *Tier:* Unit Tests
  - *Priority:* P0

- **[GH-18]** — Output pipeline redacts API keys and tokens from agent-generated text
  - *Test Scenario:* Verify API key redacted from output
  - *Test Scenario:* Verify GitHub PAT redacted from output
  - *Test Scenario:* Verify clean text passes through unchanged
  - *Tier:* Unit Tests
  - *Priority:* P1

- **[GH-18]** — Context injection scanner detects known injection patterns with correct severity classification
  - *Test Scenario:* Verify instruction override detected as critical
  - *Test Scenario:* Verify credential exfiltration detected as critical
  - *Test Scenario:* Verify hidden HTML comment detected as high
  - *Test Scenario:* Verify clean text returns safe result
  - *Test Scenario:* Verify empty string handled without panic
  - *Tier:* Unit Tests
  - *Priority:* P0

- **[GH-18]** — Pipeline aggregates findings from all scanners and enforces fail-closed safety
  - *Test Scenario:* Verify safe result when all scanners pass
  - *Test Scenario:* Verify unsafe result when any scanner triggers
  - *Test Scenario:* Verify findings aggregated from multiple scanners
  - *Test Scenario:* Verify HasCriticalFindings identifies critical severity
  - *Test Scenario:* Verify nil findings returns false for critical check
  - *Tier:* Unit Tests
  - *Priority:* P1

- **[GH-18]** — Model provider definitions load correctly with credential mapping and validation
  - *Test Scenario:* Verify multiple providers loaded from directory
  - *Test Scenario:* Verify credentials mapped correctly per provider
  - *Test Scenario:* Verify error for missing required name field
  - *Test Scenario:* Verify error for missing required type field
  - *Tier:* Unit Tests
  - *Priority:* P1

- **[GH-18]** — Tool allowlist hook can be explicitly enabled via toggle (disabled by default)
  - *Test Scenario:* Verify allowlist hook added when enabled
  - *Test Scenario:* Verify allowlist hook absent by default
  - *Tier:* Unit Tests
  - *Priority:* P1

---

### **IV. Sign-off and Approval**

This Software Test Plan requires approval from the following stakeholders:

* **Reviewers:**
  - [TBD / @reviewer]
* **Approvers:**
  - [TBD / @approver]
