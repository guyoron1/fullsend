# Test Plan

## **Authorization Enforcement on Agent Dispatch Paths - Quality Engineering Plan**

### **Metadata & Tracking**

- **Enhancement:** [GH-79](https://github.com/fullsend-ai/fullsend/issues/79)
- **Feature Tracking:** [GH-79 — Authorization enforcement on all agent dispatch paths](https://github.com/fullsend-ai/fullsend/issues/79)
- **Epic Tracking:** [fullsend-ai/fullsend#1688](https://github.com/fullsend-ai/fullsend/pull/1688)
- **ADR Reference:** ADR 0051 — Require Authorization on All Agent Dispatch Paths
- **QE Owner:** TBD
- **Document Conventions:** `[Functional]` = single-feature isolated test; `[End-to-End]` = multi-feature workflow or integration test

### **Feature Overview**

This feature enforces authorization checks on all agent dispatch paths, ensuring only authorized users (org members, collaborators) can trigger agent runs via slash commands or PR events. This closes a security gap where several dispatch paths were ungated, reducing cost exposure and abuse surface.

---

### **I. Motivation and Requirements Review (QE Review Guidelines)**

#### **I.1 - Requirement & User Story Review Checklist**

- [ ] **Reviewed the relevant requirements.** -- Confirmed the requirements are documented and understood by the QE team.
  - ADR 0051 documents the security gap: `/fs-triage`, `/fs-code`, `/fs-review` and automatic PR triggers lacked `is_authorized` checks, allowing any GitHub user to trigger agent runs on public repos.
  - The decision requires all dispatch paths to check `author_association` against OWNER, MEMBER, or COLLABORATOR before dispatching.

- [ ] **Confirmed clear user stories and understood. Understand the value and customer use cases.** -- Value proposition and user impact are clear.
  - Value: prevents unauthorized users from triggering expensive agent inference runs (cost exposure) and reduces the attack surface for prompt injection (security).
  - User impact: external contributors can no longer trigger agents via slash commands or by opening PRs; only org members/collaborators can dispatch agent work.

- [ ] **Confirmed requirements are **testable and unambiguous**.** -- Requirements can be verified through testing.
  - Authorization behavior is testable: dispatch paths produce deterministic results based on the caller's association level.
  - Dispatch routing logic has well-defined input/output contracts verifiable through functional tests.

- [ ] **Ensured acceptance criteria are **defined clearly**.** -- Acceptance criteria exist and are measurable.
  - AC1: All slash commands (`/fs-triage`, `/fs-code`, `/fs-review`, `/fs-fix`, `/fs-retro`, `/fs-prioritize`) must check `is_authorized` before setting a STAGE.
  - AC2: `pull_request_target` events (opened, synchronize, ready_for_review) must check `is_event_actor_authorized` with the PR author's association.
  - AC3: `issues.opened`/`issues.edited` remains ungated per ADR decision (triage is low-cost).

- [ ] **Confirmed coverage for NFRs.** -- Non-functional requirements (performance, security, reliability) are identified.
  - Security: primary driver -- closes unauthorized dispatch paths.
  - Performance: no regression expected; authorization checks are simple string comparisons.
  - Reliability: dispatch routing must not silently skip stages for authorized users.

#### **I.2 - Known Limitations**

- The `issues.opened` and `issues.edited` events intentionally remain ungated for triage, as documented in ADR 0051. Triage is considered low-cost and blocking it would prevent community issue filing from being triaged.
- Authorization relies on GitHub's `author_association` field, which may not reflect real-time permission changes (e.g., a user removed from an org may still show MEMBER until GitHub refreshes the association).
- PR event authorization uses a separate helper from comment-based authorization; `issue_comment` events use the commenter's association while `pull_request_target` events use the PR author's association.
- External contributor PRs no longer receive automatic review agent runs. Maintainers must manually trigger review via label or slash command, which may increase maintainer workload for active open-source projects.
- The `pull_request_target.closed` event dispatches the retro stage without an explicit authorization check; PR closure requires write access or PR authorship, which provides implicit authorization. See Risks (II.5) for the edge case where PR authors can close their own PRs.

#### **I.3 - Technology and Design Review**

- [ ] **Developer handoff completed.** -- Design discussion and knowledge transfer done.
  - ADR 0051 accepted and reviewed. Implementation mirrors existing authorization guard pattern for consistency.
  - New authorization helper introduced for PR event triggers (distinct from comment-based authorization).
  - QE engaged during ADR design phase; test plan authored alongside implementation.

- [ ] **Technology challenges identified and mitigated.** -- Technical risks assessed.
  - No new technology introduced. The change extends existing authorization helpers in the dispatch workflow.
  - The forge client interface (referenced in 36+ files) is not modified, reducing blast radius. New test double implementation and consumers are added but the interface contract is unchanged.

- [ ] **Test environment needs identified.** -- Special infrastructure or access requirements documented.
  - Testing requires simulating GitHub webhook events with varying `author_association` values.
  - E2E tests need a GitHub org with controllable membership for live dispatch testing.

- [ ] **API extensions reviewed.** -- New or modified APIs are documented and tested.
  - No user-facing API changes. Internal configuration API extended to support per-repo installation mode with new role defaults and config structures.

- [ ] **Topology and deployment considerations reviewed.** -- Impact on deployment modes assessed.
  - Per-org and per-repo install modes both affected. The dispatch workflow is shared across both modes via `reusable-dispatch.yml`.

---

### **II. Software Test Plan (STP)**

#### **II.1 - Scope of Testing**

This test plan covers authorization enforcement on all agent dispatch paths, including slash command dispatch, PR event dispatch, the updated CLI admin and config packages, and per-repo installation flow changes. Testing validates that unauthorized users are blocked from triggering agent runs, that authorized users retain full access, and that unauthorized users receive visible feedback when their commands are not executed.

**Testing Goals**

- **P0:** Verify all slash commands (`/fs-triage`, `/fs-code`, `/fs-review`, `/fs-fix`, `/fs-retro`, `/fs-prioritize`) enforce authorization before dispatch.
- **P0:** Verify PR events (opened, synchronize, ready_for_review) check the PR author's authorization before dispatching agents.
- **P0:** Verify unauthorized slash commands produce visible feedback (reaction or comment) so users know the command was received but not executed.
- **P1:** Verify CLI admin per-repo install flow works with new configuration structures and default roles.
- **P1:** Verify provisioner correctly handles org/role authorization in mint enrollment.
- **P2:** Verify edge cases in dispatch routing (Bot users, needs-info label re-triage, fork PR blocking, missing or malformed association values).

**Out of Scope (Testing Scope Exclusions)**

- [ ] **GitHub Actions platform behavior** -- GitHub's webhook delivery, event payload structure, and `author_association` computation are GitHub platform responsibilities, not product-level concerns.
- [ ] **Kubernetes platform primitives** -- Raw pod scheduling, RBAC engine, and namespace isolation are platform-level tests.
- [ ] **Inference provider behavior** -- Vertex AI or other inference provider availability and response quality are external dependencies.
- [ ] **ADRs 0047–0050 (vendored installs, automatic updates, env var convention, distributed tracing)** -- Bundled in same PR but tracked under separate test plans with independent validation.
- [ ] **Token model migration (status-token to mint-url)** -- Infrastructure change bundled in this PR; validated separately as part of the mint enrollment workflow.
- [ ] **Triage-result schema changes (blocked → prerequisites)** -- Schema evolution tracked independently; no authorization impact.

#### **II.2 - Test Strategy**

**Functional**

- [x] **Functional Testing** -- Core authorization enforcement on dispatch paths.
  - Validate comment-based authorization accepts OWNER, MEMBER, COLLABORATOR and rejects all other associations.
  - Validate PR event authorization checks the PR author's association level.
  - Validate each slash command dispatch path enforces authorization before setting a stage.
  - Validate per-repo configuration parsing, validation, and marshaling.

- [x] **Automation Testing** -- All tests automated in Go test suite.
  - Unit tests for role validation, default role generation, and per-repo configuration parsing.
  - Unit tests for CLI run, admin, mint setup, and slug discovery commands.
  - Integration tests for provisioner authorization flows.

- [x] **Regression Testing** -- Verify existing dispatch behavior not broken.
  - Existing `/fs-fix`, `/fs-retro`, `/fs-prioritize` guards unchanged.
  - `needs-info` label re-triage path preserves existing NONE + issue-author logic.
  - `issues.labeled` dispatch (ready-to-code, ready-for-review) unaffected.

- [ ] **Upgrade Testing** -- Not applicable for this change.
  - Workflow changes deploy atomically via `@v0` tag reference; no rolling upgrade path.

**Non-Functional**

- [ ] **Performance Testing** -- Not applicable.
  - Authorization checks are simple string comparisons with negligible latency impact.

- [ ] **Scale Testing** -- Not applicable.
  - No new resource-intensive operations introduced.

- [x] **Security Testing** -- Primary motivation for this feature.
  - Verify external users (NONE association) cannot trigger any slash command.
  - Verify Bot users are excluded from slash command dispatch.
  - Verify fork PRs are blocked from fix agent dispatch.

- [ ] **Usability Testing** -- Not applicable.
  - No user-facing UI changes.

- [ ] **Monitoring** -- Not applicable.
  - Dispatch routing already emits stage output; no additional monitoring instrumentation needed.

**Integration & Compatibility**

- [x] **Compatibility Testing** -- Per-org and per-repo install modes.
  - Verify `reusable-dispatch.yml` works for both install modes.
  - Verify per-repo roles validation is consistent with organization-level roles.

- [ ] **Dependencies** -- No external team delivery dependencies identified.
  - Forge client interface stability is an internal code concern addressed in Technology Challenges (I.3).

- [ ] **Cross Integrations** -- Not applicable.
  - No new cross-service integrations introduced.

**Infrastructure**

- [ ] **Cloud Testing** -- Not applicable.
  - GCP provisioner changes are tested via mock; live infrastructure validation is out of scope for this test plan. See Risk: Mock Coverage Gap (II.5).

#### **II.3 - Test Environment**

- **Cluster Topology:** N/A -- no Kubernetes cluster required; all tests run in CI
- **Platform Version:** Go 1.26.0 (per go.mod)
- **CPU Virtualization:** N/A
- **Compute:** CI runner with GitHub API access for dispatch event simulation (ubuntu-latest)
- **Special Hardware:** None
- **Storage:** Filesystem for test fixtures (per-repo config YAML files, role definitions)
- **Network:** GitHub API access for E2E dispatch tests; mocked for unit/functional tests
- **Operators:** N/A
- **Platform:** GitHub Actions (workflow dispatch testing)
- **Special Configs:** GitHub org with controllable membership to simulate authorized/unauthorized dispatch scenarios for E2E tests

#### **II.3.1 - Testing Tools & Frameworks**

No new or special testing tools required. Standard Go testing with testify assertions.

#### **II.4 - Entry Criteria**

- [ ] All PR commits merged and CI passing on branch
- [ ] ADR 0051 accepted and documented
- [ ] `go vet` and `go build` pass without errors
- [ ] Existing test suite passes (no regressions)
- [ ] `reusable-dispatch.yml` linting passes

#### **II.5 - Risks**

- [ ] **Timeline**
  - *Risk:* Large PR (100 files, +16589/-2316) may delay review and test completion.
  - *Mitigation:* PR bundles multiple upstream changes; authorization changes are isolated in dispatch workflow and CLI packages.
  - *Status:* [ ] Monitoring

- [ ] **Coverage**
  - *Risk:* Workflow YAML authorization logic cannot be unit-tested directly; requires E2E dispatch simulation.
  - *Mitigation:* CLI-level tests cover config parsing and role validation; E2E tests cover live dispatch behavior.
  - *Status:* [ ] Monitoring

- [ ] **Environment**
  - *Risk:* Test org membership may not be configurable in all CI environments, preventing E2E dispatch tests from running.
  - *Mitigation:* Use existing test org with bot and external user accounts; fall back to mock-based testing if live org unavailable.
  - *Status:* [ ] Monitoring

- [ ] **Untestable**
  - *Risk:* GitHub's `author_association` refresh timing is non-deterministic.
  - *Mitigation:* Document as known limitation; test with stable association values.
  - *Status:* [ ] Accepted

- [ ] **Resources**
  - *Risk:* None identified.
  - *Mitigation:* N/A
  - *Status:* [ ] N/A

- [ ] **Dependencies**
  - *Risk:* Forge client interface referenced in 36+ files; changes could cause widespread compilation failures.
  - *Mitigation:* Interface is not modified in this PR; only new test double and consumers added.
  - *Status:* [ ] Mitigated

- [ ] **Retro Path**
  - *Risk:* PR closure dispatches retro without explicit authorization check; PR authors (including external contributors) can close their own PRs, potentially triggering unauthorized retro runs.
  - *Mitigation:* PR closure requires write access or PR authorship; implicit authorization is considered acceptable per current design. Documented in Known Limitations.
  - *Status:* [ ] Accepted

- [ ] **Mock Coverage Gap**
  - *Risk:* Provisioner authorization changes tested only via mock; live GCP enrollment behavior is not validated in this test plan.
  - *Mitigation:* Mock-based tests verify authorization logic; live enrollment validated separately in infrastructure test suite.
  - *Status:* [ ] Accepted

- [ ] **Other**
  - *Risk:* `issues.opened` remaining ungated may be re-evaluated in future ADRs.
  - *Mitigation:* Current behavior is intentional per ADR 0051; test plan covers current decision.
  - *Status:* [ ] Accepted

---

### **III. Test Scenarios & Traceability**

#### **III.1 - Requirements-to-Tests Mapping**

- **[GH-79]** -- Slash command authorization: all slash commands enforce authorization before dispatch, matching existing guard pattern across `/fs-triage`, `/fs-code`, `/fs-review`, `/fs-fix`, `/fs-retro`, `/fs-prioritize`.
  - *Test Scenario:* Verify authorized user (MEMBER) can trigger `/fs-triage` dispatch [Functional]
  - *Test Scenario:* Verify authorized user (COLLABORATOR) can trigger `/fs-code` dispatch [Functional]
  - *Test Scenario:* Verify authorized user (OWNER) can trigger `/fs-review` dispatch [Functional]
  - *Test Scenario:* Verify unauthorized user (NONE) is blocked from all slash commands [Functional]
  - *Test Scenario:* Verify Bot user type is excluded from slash command dispatch [Functional]
  - *Priority:* P0

- **[GH-79]** -- PR event authorization: opened, synchronize, and ready_for_review events check PR author authorization before dispatching agents.
  - *Test Scenario:* Verify PR from authorized author (MEMBER) triggers review dispatch [Functional]
  - *Test Scenario:* Verify PR from unauthorized author (NONE) is blocked from review dispatch [Functional]
  - *Test Scenario:* Verify PR event authorization accepts OWNER, MEMBER, COLLABORATOR associations [Functional]
  - *Test Scenario:* Verify PR event authorization rejects NONE and FIRST_TIME_CONTRIBUTOR associations [Functional]
  - *Priority:* P0

- **[GH-79]** -- Issues.opened triage remains ungated: triage dispatch fires for any issue opener regardless of association, per ADR 0051 decision.
  - *Test Scenario:* Verify issues.opened triggers triage without authorization check [Functional]
  - *Test Scenario:* Verify issues.edited triggers triage without authorization check [Functional]
  - *Priority:* P1

- **[GH-79]** -- Needs-info re-triage authorization: comments on `needs-info` labeled issues allow NONE association only if commenter is the issue author.
  - *Test Scenario:* Verify issue author with NONE association can re-trigger triage on needs-info issue [Functional]
  - *Test Scenario:* Verify non-author with NONE association is blocked from re-triggering triage [Functional]
  - *Test Scenario:* Verify non-Bot user with non-NONE association can re-trigger triage [Functional]
  - *Priority:* P1

- **[GH-79]** -- Fork PR blocking for fix agent: fix dispatch is blocked when PR head repo differs from base repo.
  - *Test Scenario:* Verify fork PR is blocked from fix agent dispatch [Functional]
  - *Test Scenario:* Verify same-repo PR is allowed for fix agent dispatch [Functional]
  - *Priority:* P1

- **[GH-79]** -- Per-repo configuration parsing and validation: per-repo installation configuration supports roles and kill switch.
  - *Test Scenario:* Verify per-repo configuration accepts valid role definitions [Functional]
  - *Test Scenario:* Verify per-repo configuration rejects invalid role names [Functional]
  - *Test Scenario:* Verify per-repo configuration roundtrip preserves data integrity [Functional]
  - *Test Scenario:* Verify default roles for per-repo installation match expected set [Functional]
  - *Priority:* P1

- **[GH-79]** -- Organization role validation: valid roles include all recognized agent roles including dispatch-gated roles.
  - *Test Scenario:* Verify role validation recognizes all seven agent roles [Functional]
  - *Test Scenario:* Verify organization configuration rejects unknown role names [Functional]
  - *Test Scenario:* Verify dispatch is skipped when the stage role is not in configured roles [Functional]
  - *Priority:* P1

- **[GH-79]** -- Kill switch enforcement: dispatch is halted when kill switch is enabled in configuration.
  - *Test Scenario:* Verify kill switch halts all dispatch stages [Functional]
  - *Test Scenario:* Verify dispatch proceeds when kill switch is disabled [Functional]
  - *Priority:* P0

- **[GH-79]** -- Provisioner mint enrollment with authorization: provisioner correctly handles org/role authorization when enrolling new orgs.
  - *Test Scenario:* Verify provisioner stores agent PEM for authorized roles [Functional]
  - *Test Scenario:* Verify provisioner adds role to mint with correct app ID [Functional]
  - *Test Scenario:* Verify provisioner registers per-repo WIF provider [Functional]
  - *Test Scenario:* Verify provisioner discovers existing mint configuration [Functional]
  - *Priority:* P1

- **[GH-79]** -- Test double for forge client: test mock enables isolated testing of authorization-dependent code paths.
  - *Test Scenario:* Verify test mock implements all required forge client operations [Functional]
  - *Test Scenario:* Verify test mock returns configured test responses [Functional]
  - *Priority:* P2

- **[GH-79]** -- Unauthorized user feedback: ADR 0051 mandates visible feedback (reaction or comment) when unauthorized users invoke slash commands, so users know the command was received but not executed.
  - *Test Scenario:* Verify unauthorized slash command produces visible feedback indicating command was received but not executed [Functional]
  - *Test Scenario:* Verify unauthorized PR event produces no dispatch but logs the rejection [Functional]
  - *Priority:* P0

- **[GH-79]** -- Retro path authorization edge case: PR closure dispatches retro stage; verify authorization boundaries for the close event.
  - *Test Scenario:* Verify PR closure by authorized user triggers retro dispatch [Functional]
  - *Test Scenario:* Verify PR closure by external contributor does not trigger unauthorized retro agent run [Functional]
  - *Priority:* P1

- **[GH-79]** -- Authorization boundary edge cases: verify behavior at authorization check boundaries.
  - *Test Scenario:* Verify authorization check handles missing association value gracefully [Functional]
  - *Test Scenario:* Verify authorization check is case-sensitive per GitHub API contract [Functional]
  - *Test Scenario:* Verify authorization check handles empty association string without error [Functional]
  - *Priority:* P2

- **[GH-79]** -- End-to-end dispatch authorization flow: complete slash command lifecycle from comment to agent execution with authorization enforcement.
  - *Test Scenario:* Verify authorized user slash command triggers full dispatch pipeline [End-to-End]
  - *Test Scenario:* Verify unauthorized user slash command produces visible feedback and no dispatch output [End-to-End]
  - *Test Scenario:* Verify PR from external contributor does not trigger review agent [End-to-End]
  - *Test Scenario:* Verify unauthorized user receives reaction or comment indicating command was not executed [End-to-End]
  - *Priority:* P0

- **[GH-79]** -- CLI admin per-repo install flow: end-to-end per-repo installation creates config, sets up dispatch, and validates roles.
  - *Test Scenario:* Verify per-repo install creates valid configuration [End-to-End]
  - *Test Scenario:* Verify per-repo install with custom roles propagates to dispatch [End-to-End]
  - *Priority:* P1

---

### **IV. Sign-off and Approval**

| Role | Name | Date | Signature |
|:-----|:-----|:-----|:----------|
| QE Lead | | | |
| Dev Lead | | | |
| PM | | | |
