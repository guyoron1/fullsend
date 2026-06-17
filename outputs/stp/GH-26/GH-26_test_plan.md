# FullSend Test Plan

## **Defense-in-Depth Against Duplicate Code Agent PRs — Quality Engineering Plan**

### **Metadata & Tracking**

- **Enhancement(s):** [GH-26](https://github.com/guyoron1/fullsend/pull/26) (mirror of [fullsend-ai/fullsend#2373](https://github.com/fullsend-ai/fullsend/pull/2373))
- **Feature Tracking:** GH-26
- **Epic Tracking:** Upstream issues #1312, #1320, #1321
- **QE Owner(s):** TBD
- **Owning SIG:** N/A
- **Participating SIGs:** None

**Document Conventions (if applicable):** N/A

### **Feature Overview**

This change implements a three-layer defense system to prevent duplicate code agent PRs from being created when an open human PR already addresses a GitHub issue. The defenses operate at the code agent layer (`pre-code.sh` skip logic and `reusable-code.yml` step gating), the triage agent layer (hard constraint in the agent definition), and the dispatch layer (`dispatch.yml` pre-flight PR check). Each layer operates independently so that a failure in one layer does not allow duplicate PRs through. The change also migrates status comment authentication from a static `status-token` input to on-demand token minting via `mint-url`.

---

### **I. Motivation and Requirements Review (QE Review Guidelines)**

This section documents the mandatory QE review process. The goal is to understand the feature's value, technology, and testability before formal test planning.

#### **1. Requirement & User Story Review Checklist**

- [ ] **Review Requirements**
  - Reviewed the relevant requirements.
  - PR body clearly describes the 2026-05-21 incident where 5 duplicate PRs were created, motivating all three defense layers.
  - Upstream issues #1312, #1320, #1321 provide per-layer requirements.
- [ ] **Understand Value and Customer Use Cases**
  - Confirmed clear user stories and understood.
  - Understand the difference between community and product requirements.
  - **What is the value of the feature for customers**.
  - Ensured requirements contain relevant **customer use cases**.
  - Prevents wasted CI compute and developer confusion from duplicate automated PRs. Directly impacts platform reliability for all FullSend-enrolled repositories.
- [ ] **Testability**
  - Confirmed requirements are **testable and unambiguous**.
  - `pre-code.sh` has a companion `pre-code-test.sh` with mock gh infrastructure. Workflow step conditionals are testable via GitHub Actions. Triage agent behavior is testable via structured JSON output validation.
- [ ] **Acceptance Criteria**
  - Ensured acceptance criteria are **defined clearly** (clear user stories; product requirements clearly defined in Jira).
  - AC derived from PR description: (1) pre-code.sh sets `skipped=true` when human PR exists, (2) all post-validation workflow steps gate on skip output, (3) triage agent emits `prerequisites` action, (4) dispatch pre-flight blocks code stage.
- [ ] **Non-Functional Requirements (NFRs)**
  - Confirmed coverage for NFRs, including Performance, Security, Usability, Downtime, Connectivity, Monitoring (alerts/metrics), Scalability, Portability (e.g., cloud support), and Docs.
  - Race condition handling between concurrent dispatch events is the primary NFR concern. The `--force` override addresses usability for intentional duplicate work.

#### **2. Known Limitations**

- The pre-code.sh PR search uses `in:body,title` which may produce false positives if the issue number appears coincidentally in unrelated PRs.
- Bot author filtering is hardcoded to `fullsend-ai[bot]` and `fullsend-ai-coder[bot]` — additional bot accounts require code changes.
- The dispatch layer pr-check only runs for `stage == 'code'` — other stages are unguarded against duplicates.
- The triage agent constraint is a prompt-level instruction (not programmatic enforcement), so LLM compliance is probabilistic.
- The `--force` override bypasses all three layers, which could be misused.

#### **3. Technology and Design Review**

- [ ] **Developer Handoff/QE Kickoff**
  - A meeting where Dev/Arch walked QE through the design, architecture, and implementation details. **Critical for identifying untestable aspects early.**
  - Three-layer architecture documented in PR body with clear separation of concerns. Shell script, workflow YAML, and agent definition changes are independent.
- [ ] **Technology Challenges**
  - Identified potential testing challenges related to the underlying technology.
  - Testing GitHub Actions workflow conditionals requires either real workflow runs or YAML parsing validation. Race conditions between concurrent dispatches are difficult to reproduce deterministically.
- [ ] **Test Environment Needs**
  - Determined necessary **test environment setups and tools**.
  - Mock gh binary infrastructure (pre-code-test.sh pattern), GitHub Actions runner for workflow tests, and agent sandbox for triage agent tests.
- [ ] **API Extensions**
  - Reviewed new or modified APIs and their impact on testing.
  - `action.yml` replaces `status-token` input with `mint-url`. CLI replaces `--status-token` flag with `--mint-url`. `reconcile-status` replaces `--token` with `--mint-url` and adds `--role`.
- [ ] **Topology Considerations**
  - Evaluated multi-cluster, network topology, and architectural impacts.
  - No topology changes. All layers run on single GitHub Actions runners within enrolled repositories.

### **II. Software Test Plan (STP)**

This STP serves as the **overall roadmap for testing**, detailing the scope, approach, resources, and schedule.

#### **1. Scope of Testing**

Testing covers the three independent defense layers against duplicate code agent PRs: the pre-code.sh script (input validation + existing PR detection + skip signaling), the dispatch.yml pre-flight check (PR search before code stage invocation), and the triage agent hard constraint (prerequisites action on existing PR). Additionally, the status-token to mint-url migration in action.yml, run.go, and reconcile-status is in scope.

**Testing Goals**

**Functional Goals:**
- **P0:** Verify each defense layer independently detects and blocks duplicate PR creation when an open human PR exists for the target issue.
- **P0:** Verify all post-validation workflow steps in reusable-code.yml are correctly gated on the validate step's skip output.
- **P1:** Verify force override (`--force` / `CODE_FORCE=true`) bypasses the pre-code duplicate check.
- **P1:** Verify bot-authored PRs are correctly excluded from duplicate detection at both pre-code and dispatch layers.

**Quality Goals:**
- **P1:** Verify status comment functionality is preserved after migration from status-token to mint-url.
- **P1:** Verify reconcile-status command correctly mints tokens on-demand for orphaned comment cleanup.

**Integration Goals:**
- **P1:** Verify the three defense layers collectively prevent duplicates even under concurrent dispatch conditions.

**Out of Scope (Testing Scope Exclusions)**

- [ ] GitHub Actions platform behavior (workflow scheduling, concurrency groups) — *Rationale:* Platform-level behavior tested by GitHub. — *PM/Lead Agreement:* TBD
- [ ] `gh` CLI correctness for PR search queries — *Rationale:* External tool tested by GitHub CLI team. — *PM/Lead Agreement:* TBD
- [ ] LLM response quality for triage agent — *Rationale:* Agent prompt compliance is probabilistic and not deterministically testable. Structural JSON output validation is in scope. — *PM/Lead Agreement:* TBD
- [ ] Vendored install workflow changes (hashFiles gate for sparse checkout) — *Rationale:* Covered by separate ADR-0047 test plan. — *PM/Lead Agreement:* TBD

#### **2. Test Strategy**

**Functional**

- [x] **Functional Testing** — Validates that the feature works according to specified requirements and user stories
  - *Details:* Core testing of all three defense layers: pre-code.sh skip logic, dispatch pr-check gate, triage agent prerequisites constraint. Each layer tested independently with positive and negative cases.
- [x] **Automation Testing** — Confirms test automation plan is in place for CI and regression coverage (all tests are expected to be automated)
  - *Details:* pre-code-test.sh (existing bash test harness with mock gh), Go unit tests for run.go/reconcilestatus.go (testify), workflow YAML validation in CI.
- [x] **Regression Testing** — Verifies that new changes do not break existing functionality
  - *Details:* LSP analysis identified callers of setupStatusNotifier (10 test functions), Lint() (referenced in run.go and lock.go), and DiscoverRemoteAgents (used by admin.go and discover_slugs.go). Existing tests must continue to pass.

**Non-Functional**

- [ ] **Performance Testing** — Validates feature performance meets requirements (latency, throughput, resource usage)
  - *Details:* N/A — defense checks are lightweight GitHub API calls with negligible performance impact.
- [ ] **Scale Testing** — Validates feature behavior under increased load and at production-like scale
  - *Details:* N/A — single-issue scope per invocation; no scale concerns.
- [x] **Security Testing** — Verifies security requirements, RBAC, authentication, authorization, and vulnerability scanning
  - *Details:* Verify mint-url token minting replaces static status-token without credential leakage. Verify `::add-mask::` removal does not expose tokens in logs.
- [ ] **Usability Testing** — Validates user experience and accessibility requirements
  - *Details:* N/A — changes are infrastructure-level, not user-facing UI.
- [ ] **Monitoring** — Does the feature require metrics and/or alerts?
  - *Details:* `::notice::` annotations in workflow logs provide observability for skip events. No new metrics required.

**Integration & Compatibility**

- [ ] **Compatibility Testing** — Ensures feature works across supported platforms, versions, and configurations
  - *Details:* N/A — GitHub Actions is the sole platform.
- [ ] **Upgrade Testing** — Validates upgrade paths from previous versions, data migration, and configuration preservation
  - *Details:* N/A — scaffold sync handles workflow updates for enrolled repos.
- [x] **Dependencies** — Blocked by deliverables from other components/products
  - *Details:* Depends on `gh` CLI availability in GitHub Actions runners. Depends on mint service availability for status-token migration.
- [x] **Cross Integrations** — Does the feature affect other features or require testing by other teams?
  - *Details:* Changes to action.yml affect all agent types (code, triage, review, fix, retro). The mint-url migration impacts all reusable workflows. Scaffold sync propagates dispatch.yml and triage.md changes to enrolled repos.

**Infrastructure**

- [ ] **Cloud Testing** — Does the feature require multi-cloud platform testing?
  - *Details:* N/A — GitHub Actions only.

#### **3. Test Environment**

- **Cluster Topology:** N/A (no cluster required — GitHub Actions runner environment)
- **Platform & Product Version(s):** GitHub Actions (ubuntu-latest), Go 1.23+, FullSend 0.x
- **CPU Virtualization:** N/A
- **Compute Resources:** Standard GitHub Actions runner (2-core, 7 GB RAM)
- **Special Hardware:** N/A
- **Storage:** N/A
- **Network:** GitHub API access required for `gh` CLI operations
- **Required Operators:** N/A
- **Platform:** GitHub Actions with `gh` CLI pre-installed, `jq` available
- **Special Configurations:** Mock `gh` binary for unit tests (pre-code-test.sh pattern), fake forge client for Go tests

#### **3.1. Testing Tools & Frameworks**

- **Test Framework:** N/A (standard Go testing + testify, bash test scripts)
- **CI/CD:** N/A (standard GitHub Actions)
- **Other Tools:** N/A (standard tools only)

#### **4. Entry Criteria**

The following conditions must be met before testing can begin:

- [ ] Requirements and design documents are **approved and merged**
- [ ] Test environment can be **set up and configured** (see Section II.3 - Test Environment)
- [ ] Mock `gh` binary infrastructure is functional (pre-code-test.sh verified)
- [ ] Mint service endpoint is available for mint-url integration tests
- [ ] PR #26 changes are rebased on latest main

#### **5. Risks**

- [ ] **Timeline/Schedule**
  - Risk: Multi-layer testing across shell scripts, workflows, and agent definitions requires diverse test infrastructure.
  - Mitigation: Leverage existing pre-code-test.sh mock pattern and Go test infrastructure. Parallelize layer testing.
- [ ] **Test Coverage**
  - Risk: Race conditions between concurrent dispatch events are difficult to reproduce deterministically.
  - Mitigation: Focus on independent layer verification. Document concurrent scenarios as manual validation items.
- [ ] **Test Environment**
  - Risk: Workflow step conditional testing requires real GitHub Actions runs or sophisticated YAML parsing.
  - Mitigation: Use Go tests for CLI/API behavior; validate workflow YAML structure statically.
- [ ] **Untestable Aspects**
  - Risk: Triage agent hard constraint is a prompt-level instruction — LLM compliance cannot be guaranteed.
  - Mitigation: Validate structured JSON output schema (action field must be "prerequisites" when PR exists). Accept probabilistic compliance.
- [ ] **Resource Constraints**
  - Risk: N/A — standard CI resources sufficient.
  - Mitigation: N/A.
- [ ] **Dependencies**
  - Risk: Mint service availability is required for status-token migration tests.
  - Mitigation: Use mocked mint service for unit tests; integration test against staging mint endpoint.
- [ ] **Other**
  - Risk: False positives from `in:body,title` PR search matching unrelated PRs containing the issue number.
  - Mitigation: Include test scenarios with coincidental number matches to validate filtering behavior.

---

### **III. Test Scenarios & Traceability**

This section links requirements to test coverage, enabling reviewers to verify all requirements are tested.

#### **1. Requirements-to-Tests Mapping**

- **[GH-26]** — Code agent pre-script detects existing human PRs and skips automated implementation
  - *Test Scenario:* TS-GH-26-001: Verify pre-code skips when human PR exists (Unit Tests)
  - *Test Scenario:* TS-GH-26-002: Verify skip comment posted on issue (Unit Tests)
  - *Test Scenario:* TS-GH-26-003: Verify pr-open label applied on skip (Unit Tests)
  - *Test Scenario:* TS-GH-26-004: Verify skipped=true written to GITHUB_OUTPUT (Unit Tests)
  - *Priority:* P0

- **[GH-26]** — Code agent pre-script allows execution when no existing PRs are found
  - *Test Scenario:* TS-GH-26-005: Verify agent proceeds with no existing PRs (Unit Tests)
  - *Test Scenario:* TS-GH-26-006: Verify skipped=false written to GITHUB_OUTPUT (Unit Tests)
  - *Priority:* P0

- **[GH-26]** — Code agent pre-script supports force override to bypass duplicate check
  - *Test Scenario:* TS-GH-26-007: Verify --force flag bypasses PR check (Unit Tests)
  - *Test Scenario:* TS-GH-26-008: Verify CODE_FORCE=true bypasses PR check (Unit Tests)
  - *Test Scenario:* TS-GH-26-009: Verify force override with existing PRs proceeds (Unit Tests)
  - *Priority:* P1

- **[GH-26]** — Code agent pre-script filters out bot-authored PRs from duplicate detection
  - *Test Scenario:* TS-GH-26-010: Verify bot PRs excluded from duplicate check (Unit Tests)
  - *Test Scenario:* TS-GH-26-011: Verify mixed bot and human PRs detected correctly (Unit Tests)
  - *Test Scenario:* TS-GH-26-012: Verify only-bot PRs do not trigger skip (Unit Tests)
  - *Priority:* P1

- **[GH-26]** — Dispatch workflow pre-flight check prevents code agent invocation when open PRs exist
  - *Test Scenario:* TS-GH-26-013: Verify dispatch blocks code stage on existing PR (Functional)
  - *Test Scenario:* TS-GH-26-014: Verify dispatch allows non-code stages (Functional)
  - *Test Scenario:* TS-GH-26-015: Verify dispatch proceeds when no PRs found (Functional)
  - *Test Scenario:* TS-GH-26-016: Verify stage output includes pr-check gate (Functional)
  - *Priority:* P0

- **[GH-26]** — Triage agent stops with prerequisites action when an open PR addresses the issue
  - *Test Scenario:* TS-GH-26-017: Verify triage emits prerequisites on existing PR (Functional)
  - *Test Scenario:* TS-GH-26-018: Verify triage proceeds when no open PR exists (Functional)
  - *Test Scenario:* TS-GH-26-019: Verify closed PR does not trigger prerequisite (Functional)
  - *Priority:* P0

- **[GH-26]** — Defense layers operate independently — each layer can catch duplicates even if others fail
  - *Test Scenario:* TS-GH-26-020: Verify pre-code catches duplicate alone (End-to-End)
  - *Test Scenario:* TS-GH-26-021: Verify dispatch catches duplicate alone (End-to-End)
  - *Test Scenario:* TS-GH-26-022: Verify triage catches duplicate alone (End-to-End)
  - *Test Scenario:* TS-GH-26-023: Verify concurrent triggers handled by layered defense (End-to-End)
  - *Priority:* P1

- **[GH-26]** — Reusable workflow steps correctly gated on validate step skip output
  - *Test Scenario:* TS-GH-26-024: Verify GCP setup skipped when validate skips (Functional)
  - *Test Scenario:* TS-GH-26-025: Verify agent run skipped when validate skips (Functional)
  - *Test Scenario:* TS-GH-26-026: Verify all gated steps run when not skipped (Functional)
  - *Priority:* P0

- **[GH-26]** — Status comment token migration from status-token to mint-url preserves functionality
  - *Test Scenario:* TS-GH-26-027: Verify status notifier works with mint-url (Unit Tests)
  - *Test Scenario:* TS-GH-26-028: Verify error when mint-url unavailable (Unit Tests)
  - *Priority:* P1

- **[GH-26]** — Reconcile status command uses mint-url for token acquisition
  - *Test Scenario:* TS-GH-26-029: Verify reconcile-status mints token via URL (Unit Tests)
  - *Test Scenario:* TS-GH-26-030: Verify orphaned comment finalized with mint-url (Functional)
  - *Priority:* P1

---

### **IV. Sign-off and Approval**

This Software Test Plan requires approval from the following stakeholders:

* **Reviewers:**
  - [TBD / @tbd]
* **Approvers:**
  - [TBD / @tbd]
