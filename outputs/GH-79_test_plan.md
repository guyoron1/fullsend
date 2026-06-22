# Test Plan

## **ADR 0051: Require Authorization on All Agent Dispatch Paths - Quality Engineering Plan**

### **Metadata & Tracking**

- **Enhancement:** [GH-79](https://github.com/guyoron1/fullsend/issues/79)
- **Feature Tracking:** [GH-79 — feat(#1662): ADR 0051 + implement is_authorized on all agent dispatch paths](https://github.com/guyoron1/fullsend/issues/79)
- **Epic Tracking:** [upstream fullsend-ai/fullsend#1688](https://github.com/fullsend-ai/fullsend/pull/1688)
- **QE Owner:** TBD
- **Document Conventions:** `[Functional]` = single-feature isolated test; `[End-to-End]` = multi-feature workflow or integration test

### **Feature Overview**

This feature enforces `is_authorized` authorization checks on all agent dispatch paths, closing a security gap identified in ADR 0051. Previously, only `/fs-fix`, `/fs-retro`, and `/fs-prioritize` slash commands gated on the caller's `author_association`; the `/fs-triage`, `/fs-code`, and `/fs-review` commands and automatic `pull_request_target` event triggers were ungated. This change adds consistent authorization checks across all dispatch paths to prevent unauthorized users from triggering agent inference runs, reducing cost exposure and abuse surface.

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
  - Authorization behavior is testable: the `is_authorized()` and `is_event_actor_authorized()` functions return deterministic results based on `author_association` values.
  - Dispatch routing logic is exercised via workflow YAML with well-defined input/output contracts.

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
- The `is_event_actor_authorized()` helper is only used for `pull_request_target` events; `issue_comment` events continue to use the existing `is_authorized()` helper that reads `COMMENT_AUTHOR_ASSOC`.

#### **I.3 - Technology and Design Review**

- [ ] **Developer handoff completed.** -- Design discussion and knowledge transfer done.
  - ADR 0051 accepted and reviewed. Implementation mirrors existing `/fs-fix` guard pattern for consistency.
  - New `is_event_actor_authorized()` helper introduced for non-comment event triggers.

- [ ] **Technology challenges identified and mitigated.** -- Technical risks assessed.
  - No new technology introduced. The change extends existing bash helper functions in the dispatch workflow YAML.
  - `forge.Client` interface (referenced in 36+ files) is not modified, reducing blast radius.

- [ ] **Test environment needs identified.** -- Special infrastructure or access requirements documented.
  - Testing requires simulating GitHub webhook events with varying `author_association` values.
  - E2E tests need a GitHub org with controllable membership for live dispatch testing.

- [ ] **API extensions reviewed.** -- New or modified APIs are documented and tested.
  - No new APIs. Changes are in GitHub Actions workflow YAML and CLI internals.
  - `config.ValidRoles()` unchanged; `PerRepoDefaultRoles()` and `PerRepoConfig` added for per-repo install flow.

- [ ] **Topology and deployment considerations reviewed.** -- Impact on deployment modes assessed.
  - Per-org and per-repo install modes both affected. The dispatch workflow is shared across both modes via `reusable-dispatch.yml`.

---

### **II. Software Test Plan (STP)**

#### **II.1 - Scope of Testing**

This test plan covers the authorization enforcement on all agent dispatch paths in the `reusable-dispatch.yml` workflow, the new `is_event_actor_authorized()` helper, the updated CLI admin and config packages, and the per-repo installation flow changes. Testing validates that unauthorized users are blocked from triggering agent runs while authorized users retain full access.

**Testing Goals**

- **P0:** Verify all slash commands (`/fs-triage`, `/fs-code`, `/fs-review`, `/fs-fix`, `/fs-retro`, `/fs-prioritize`) enforce `is_authorized` before dispatch.
- **P0:** Verify `pull_request_target` events check `is_event_actor_authorized` with PR author association.
- **P1:** Verify CLI admin per-repo install flow works with new config structures (`PerRepoConfig`, `PerRepoDefaultRoles`).
- **P1:** Verify provisioner correctly handles org/role authorization in mint enrollment.
- **P2:** Verify edge cases in dispatch routing (Bot users, `needs-info` label re-triage, fork PR blocking).

**Out of Scope (Testing Scope Exclusions)**

- [ ] **GitHub Actions platform behavior** -- GitHub's webhook delivery, event payload structure, and `author_association` computation are GitHub platform responsibilities, not product-level concerns.
- [ ] **Kubernetes platform primitives** -- Raw pod scheduling, RBAC engine, and namespace isolation are platform-level tests.
- [ ] **Inference provider behavior** -- Vertex AI or other inference provider availability and response quality are external dependencies.

#### **II.2 - Test Strategy**

**Functional**

- [x] **Functional Testing** -- Core authorization enforcement on dispatch paths.
  - Validate `is_authorized()` accepts OWNER, MEMBER, COLLABORATOR and rejects all other associations.
  - Validate `is_event_actor_authorized()` for PR author association checks.
  - Validate each slash command dispatch path enforces authorization.
  - Validate `PerRepoConfig` parsing, validation, and marshaling.

- [x] **Automation Testing** -- All tests automated in Go test suite.
  - Unit tests for `config.ValidRoles()`, `PerRepoDefaultRoles()`, `ParsePerRepoConfig()`.
  - Unit tests for `cli.run`, `cli.admin`, `cli.mint_setup`, `cli.discover_slugs`.
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
  - Dispatch routing already emits stage output via `GITHUB_OUTPUT`.

**Integration & Compatibility**

- [x] **Compatibility Testing** -- Per-org and per-repo install modes.
  - Verify `reusable-dispatch.yml` works for both install modes.
  - Verify `PerRepoConfig` roles validation is consistent with `OrgConfig` roles.

- [x] **Dependencies** -- forge.Client interface stability.
  - Verify `forge.Client` implementations (GitHub, Fake) satisfy updated interface.
  - Verify `forge.Fake` test double covers new methods.

- [ ] **Cross Integrations** -- Not applicable.
  - No new cross-service integrations introduced.

**Infrastructure**

- [ ] **Cloud Testing** -- Not applicable.
  - GCP provisioner changes are tested via `fakeclient` mock, not live infrastructure.

#### **II.3 - Test Environment**

- **Cluster Topology:** N/A -- no Kubernetes cluster required for unit/functional tests
- **Platform Version:** Go 1.26.0 (per go.mod)
- **CPU Virtualization:** N/A
- **Compute:** Standard CI runner (ubuntu-latest)
- **Special Hardware:** None
- **Storage:** Standard filesystem for test fixtures
- **Network:** GitHub API access for E2E tests; mocked for unit tests
- **Operators:** N/A
- **Platform:** GitHub Actions (workflow dispatch testing)
- **Special Configs:** GitHub org with controllable membership for E2E dispatch tests

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
  - *Risk:* E2E dispatch tests require a GitHub org with controllable user membership.
  - *Mitigation:* Use existing `guyoron1` test org with bot and external user accounts.
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
  - *Risk:* `forge.Client` interface referenced in 36+ files; changes could cause widespread compilation failures.
  - *Mitigation:* Interface is not modified in this PR; only new implementations (`forge.Fake`) and consumers added.
  - *Status:* [ ] Mitigated

- [ ] **Other**
  - *Risk:* `issues.opened` remaining ungated may be re-evaluated in future ADRs.
  - *Mitigation:* Current behavior is intentional per ADR 0051; test plan covers current decision.
  - *Status:* [ ] Accepted

---

### **III. Test Scenarios & Traceability**

#### **III.1 - Requirements-to-Tests Mapping**

- **[GH-79]** -- Slash command authorization: `/fs-triage`, `/fs-code`, `/fs-review` enforce `is_authorized` before dispatch, matching existing `/fs-fix`, `/fs-retro`, `/fs-prioritize` behavior.
  - *Test Scenario:* Verify authorized user (MEMBER) can trigger `/fs-triage` dispatch [Functional]
  - *Test Scenario:* Verify authorized user (COLLABORATOR) can trigger `/fs-code` dispatch [Functional]
  - *Test Scenario:* Verify authorized user (OWNER) can trigger `/fs-review` dispatch [Functional]
  - *Test Scenario:* Verify unauthorized user (NONE) is blocked from all slash commands [Functional]
  - *Test Scenario:* Verify Bot user type is excluded from slash command dispatch [Functional]
  - *Priority:* P0

- **[GH-79]** -- PR event authorization: `pull_request_target` opened/synchronize/ready_for_review events check `is_event_actor_authorized` with PR author association.
  - *Test Scenario:* Verify PR from authorized author (MEMBER) triggers review dispatch [Functional]
  - *Test Scenario:* Verify PR from unauthorized author (NONE) is blocked from review dispatch [Functional]
  - *Test Scenario:* Verify `is_event_actor_authorized` accepts OWNER, MEMBER, COLLABORATOR [Functional]
  - *Test Scenario:* Verify `is_event_actor_authorized` rejects NONE, FIRST_TIME_CONTRIBUTOR [Functional]
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

- **[GH-79]** -- PerRepoConfig parsing and validation: new `PerRepoConfig` struct supports per-repo installation with roles and kill switch.
  - *Test Scenario:* Verify PerRepoConfig parses valid YAML correctly [Functional]
  - *Test Scenario:* Verify PerRepoConfig rejects invalid role names [Functional]
  - *Test Scenario:* Verify PerRepoConfig marshal roundtrip preserves data [Functional]
  - *Test Scenario:* Verify PerRepoDefaultRoles returns expected default roles [Functional]
  - *Priority:* P1

- **[GH-79]** -- OrgConfig role validation: `ValidRoles()` returns all recognized agent roles including new dispatch-gated roles.
  - *Test Scenario:* Verify ValidRoles includes all seven agent roles [Functional]
  - *Test Scenario:* Verify OrgConfig.Validate rejects unknown roles [Functional]
  - *Test Scenario:* Verify role-check step skips dispatch when stage role not in configured roles [Functional]
  - *Priority:* P1

- **[GH-79]** -- Kill switch enforcement: dispatch is halted when `kill_switch: true` in `.fullsend/config.yaml`.
  - *Test Scenario:* Verify kill switch halts all dispatch stages [Functional]
  - *Test Scenario:* Verify dispatch proceeds when kill switch is false [Functional]
  - *Priority:* P1

- **[GH-79]** -- Provisioner mint enrollment with authorization: provisioner correctly handles org/role authorization when enrolling new orgs.
  - *Test Scenario:* Verify provisioner stores agent PEM for authorized roles [Functional]
  - *Test Scenario:* Verify provisioner adds role to mint with correct app ID [Functional]
  - *Test Scenario:* Verify provisioner registers per-repo WIF provider [Functional]
  - *Test Scenario:* Verify provisioner discovers existing mint configuration [Functional]
  - *Priority:* P1

- **[GH-79]** -- Fake forge client for testing: new `forge.Fake` implementation enables isolated testing of authorization-dependent code paths.
  - *Test Scenario:* Verify Fake client satisfies forge.Client interface [Functional]
  - *Test Scenario:* Verify Fake client returns configured test responses [Functional]
  - *Priority:* P2

- **[GH-79]** -- End-to-end dispatch authorization flow: complete slash command lifecycle from comment to agent execution with authorization enforcement.
  - *Test Scenario:* Verify authorized user slash command triggers full dispatch pipeline [End-to-End]
  - *Test Scenario:* Verify unauthorized user slash command produces no dispatch output [End-to-End]
  - *Test Scenario:* Verify PR from external contributor does not trigger review agent [End-to-End]
  - *Priority:* P0

- **[GH-79]** -- CLI admin per-repo install flow: end-to-end per-repo installation creates config, sets up dispatch, and validates roles.
  - *Test Scenario:* Verify per-repo install creates valid PerRepoConfig [End-to-End]
  - *Test Scenario:* Verify per-repo install with custom roles propagates to dispatch [End-to-End]
  - *Priority:* P1

---

### **IV. Sign-off and Approval**

| Role | Name | Date | Signature |
|:-----|:-----|:-----|:----------|
| QE Lead | | | |
| Dev Lead | | | |
| PM | | | |
