# Test Plan

## **Require Authorization on All Agent Dispatch Paths - Quality Engineering Plan**

### **Metadata & Tracking**

- **Enhancement(s):** [GH-1662](https://github.com/fullsend-ai/fullsend/issues/1662)
- **Feature Tracking:** [GH-1662](https://github.com/fullsend-ai/fullsend/issues/1662)
- **Epic Tracking:** GH-1662
- **QE Owner(s):** @ascerra
- **Owning SIG:** N/A
- **Participating SIGs:** N/A

**Document Conventions (if applicable):** N/A

### **Feature Overview**

This feature enforces a comment author authorization check (OWNER, MEMBER, or COLLABORATOR association) on all agent slash commands (`/fs-triage`, `/fs-code`, `/fs-review`) and a PR actor authorization check on automatic PR event triggers (`pull_request_target.opened/synchronize/ready_for_review`) in the dispatch routing logic. Previously, only `/fs-fix`, `/fs-retro`, and `/fs-prioritize` were gated. Auto-triage on `issues.opened/edited` is intentionally left ungated to preserve the drive-by bug reporter workflow. The change is documented in ADR 0051 and implemented in both per-repo (`reusable-dispatch.yml`) and per-org scaffold (`dispatch.yml`) workflow files.

---

### **I. Motivation and Requirements Review (QE Review Guidelines)**

This section documents the mandatory QE review process. The goal is to understand the feature's value,
technology, and testability before formal test planning.

#### **1. Requirement & User Story Review Checklist**

- [x] **Review Requirements**
  - Reviewed the relevant requirements.
  - GH-1662 clearly defines which dispatch paths are ungated and the security/cost risks.
  - ADR 0051 documents the architectural decision and rationale for each path.
- [x] **Understand Value and Customer Use Cases**
  - Confirmed clear user stories and understood.
  - Understand the difference between community and product requirements.
  - **What is the value of the feature for customers**.
  - Ensured requirements contain relevant **customer use cases**.
  - Closes a cost-exposure and abuse-surface gap where any GitHub user could trigger inference runs via ungated slash commands on public repos.
  - Preserves auto-triage for external contributors (key value prop for drive-by bug reporters).
- [x] **Testability**
  - Confirmed requirements are **testable and unambiguous**.
  - Authorization behavior is directly testable via dispatch routing — each slash command and event trigger either sets STAGE or does not based on association.
  - The `is_event_actor_authorized` shell function is independently testable with specific input values.
- [x] **Acceptance Criteria**
  - Ensured acceptance criteria are **defined clearly** (clear user stories; product requirements clearly defined in Jira).
  - Issue body specifies four design questions the ADR must address: auto-triage carve-out, bot-to-bot preservation, unauthorized feedback, and per-repo configurability interaction.
  - All four are addressed in ADR 0051.
- [x] **Non-Functional Requirements (NFRs)**
  - Confirmed coverage for NFRs, including Performance, Security, Usability, Downtime, Connectivity, Monitoring (alerts/metrics), Scalability, Portability (e.g., cloud support), and Docs.
  - Security is the primary NFR — authorization gates prevent unauthorized inference cost and reduce prompt injection attack surface.
  - Usability NFR: unauthorized users should receive visible feedback when slash commands are rejected.

#### **2. Known Limitations**

- Visible feedback for unauthorized slash command attempts (reaction/comment) is specified in ADR 0051 but not implemented in PR #1688 — the dispatch currently silently skips setting STAGE.
- Per-user rate limiting for the ungated `issues.opened` auto-triage path is deferred to #1687.
- `docs/architecture.md` references ADR 0051 but links to file `0050` — possible link mismatch needs verification.

#### **3. Technology and Design Review**

- [x] **Developer Handoff/QE Kickoff**
  - A meeting where Dev/Arch walked QE through the design, architecture, and implementation details. **Critical for identifying untestable aspects early.**
  - PR #1688 authored by fullsend-ai-coder agent; ADR 0051 provides full design context.
- [x] **Technology Challenges**
  - Identified potential testing challenges related to the underlying technology.
  - Testing dispatch routing requires simulating GitHub webhook events with specific `author_association` values — may require workflow-level integration tests or shell function unit tests.
- [x] **Test Environment Needs**
  - Determined necessary **test environment setups and tools**.
  - Tests require GitHub Actions environment or equivalent to validate dispatch routing behavior.
  - Shell function unit tests can run in any bash environment.
- [x] **API Extensions**
  - Reviewed new or modified APIs and their impact on testing.
  - New `PR_AUTHOR_ASSOC` environment variable plumbed from `github.event.pull_request.author_association`. New `is_event_actor_authorized()` shell helper function.
- [x] **Topology Considerations**
  - Evaluated multi-cluster, network topology, and architectural impacts.
  - No topology impact — changes are in workflow dispatch routing only.

### **II. Software Test Plan (STP)**

This STP serves as the **overall roadmap for testing**, detailing the scope, approach, resources, and schedule.

#### **1. Scope of Testing**

Testing covers the authorization enforcement on all agent dispatch paths in both per-repo (`reusable-dispatch.yml`) and per-org scaffold (`dispatch.yml`) workflow files. This includes verifying that `/fs-triage`, `/fs-code`, and `/fs-review` slash commands require comment author authorization, that PR event triggers use actor authorization, that `issues.opened/edited` auto-triage remains ungated, and that bot-to-bot label handoffs are unaffected.

**Testing Goals**

**Functional Goals:**

- **P0:** Verify all slash commands enforce authorization — unauthorized users (NONE, CONTRIBUTOR, FIRST_TIME_CONTRIBUTOR) cannot trigger `/fs-triage`, `/fs-code`, or `/fs-review`.
- **P0:** Verify PR event triggers enforce actor authorization — PRs by non-members do not auto-trigger review.
- **P0:** Verify auto-triage on `issues.opened/edited` remains ungated for external users.
- **P1:** Verify authorized users (OWNER, MEMBER, COLLABORATOR) can invoke all slash commands.
- **P1:** Verify bot-to-bot label handoffs are unaffected by the new authorization gates.

**Quality Goals:**

- **P1:** Verify the PR actor authorization check correctly handles all association types including edge cases (empty string, unexpected values).
- **P1:** Verify per-repo and per-org dispatch templates have consistent authorization behavior.

**Integration Goals:**

- **P2:** Verify unauthorized slash command feedback mechanism (pending implementation).

**Out of Scope (Testing Scope Exclusions)**

- [ ] GitHub Actions platform behavior (webhook delivery, event field population) -- *Rationale:* GitHub platform is tested by GitHub; we test our routing logic only. -- *PM/Lead Agreement:* TBD
- [ ] Per-user rate limiting for ungated auto-triage path -- *Rationale:* Deferred to #1687; not part of this change. -- *PM/Lead Agreement:* TBD
- [ ] GitHub `author_association` field correctness -- *Rationale:* Platform-level behavior; we trust the field value and test our response to it. -- *PM/Lead Agreement:* TBD

#### **2. Test Strategy**

**Functional**

- [x] **Functional Testing** — Validates that the feature works according to specified requirements and user stories
  - *Details:* Verify each slash command and event trigger path respects the authorization gate. Test authorized and unauthorized users for each dispatch path.
- [x] **Automation Testing** — Confirms test automation plan is in place for CI and regression coverage (all tests are expected to be automated)
  - *Details:* Existing `TestDispatchWorkflowContent` in `scaffold_test.go` validates dispatch file content including `is_authorized` strings. Shell function tests can be automated in CI.
- [x] **Regression Testing** — Verifies that new changes do not break existing functionality
  - *Details:* Verify that previously-gated commands (`/fs-fix`, `/fs-retro`, `/fs-prioritize`) remain correctly gated. Verify label-based handoffs still work.

**Non-Functional**

- [ ] **Performance Testing** — Validates feature performance meets requirements (latency, throughput, resource usage)
  - *Details:* N/A — authorization check is a trivial shell case statement with no performance impact.
- [ ] **Scale Testing** — Validates feature behavior under increased load and at production-like scale
  - *Details:* N/A — no scale dimension to authorization checks.
- [x] **Security Testing** — Verifies security requirements, RBAC, authentication, authorization, and vulnerability scanning
  - *Details:* Core focus of this feature. Verify all association types are correctly accepted or rejected. Verify no bypass paths exist.
- [ ] **Usability Testing** — Validates user experience and accessibility requirements
  - *Details:* Verify unauthorized users receive visible feedback (pending implementation).
- [ ] **Monitoring** — Does the feature require metrics and/or alerts?
  - *Details:* N/A — no new monitoring requirements for dispatch authorization.

**Integration & Compatibility**

- [x] **Compatibility Testing** — Ensures feature works across supported platforms, versions, and configurations
  - *Details:* Verify per-repo (`reusable-dispatch.yml`) and per-org scaffold (`dispatch.yml`) templates have identical authorization behavior for all dispatch paths.
- [ ] **Upgrade Testing** — Validates upgrade paths from previous versions, data migration, and configuration preservation
  - *Details:* N/A — workflow file changes are deployed atomically via scaffold install.
- [ ] **Dependencies** — Blocked by deliverables from other components/products
  - *Details:* Depends on GitHub providing `author_association` field on events (stable GitHub API feature).
- [ ] **Cross Integrations** — Does the feature affect other features or require testing by other teams?
  - *Details:* Affects all agent stages (triage, code, review). Triage auto-trigger behavior changes for PR events. Bot-to-bot handoffs via labels are unaffected.

**Infrastructure**

- [ ] **Cloud Testing** — Does the feature require multi-cloud platform testing?
  - *Details:* N/A — dispatch routing is cloud-agnostic.

#### **3. Test Environment**

- **Cluster Topology:** N/A (no cluster required — tests validate workflow routing logic)
- **Platform & Product Version(s):** GitHub Actions runner (ubuntu-latest)
- **CPU Virtualization:** N/A
- **Compute Resources:** Standard GitHub Actions runner
- **Special Hardware:** N/A
- **Storage:** N/A
- **Network:** GitHub API access required for integration tests
- **Required Operators:** N/A
- **Platform:** GitHub Actions
- **Special Configurations:** Test GitHub org with users of varying association levels (OWNER, MEMBER, COLLABORATOR, CONTRIBUTOR, NONE)

#### **3.1. Testing Tools & Frameworks**

- **Other Tools:** bash/shell for actor authorization function unit tests

#### **4. Entry Criteria**

The following conditions must be met before testing can begin:

- [ ] Requirements and design documents are **approved and merged**
- [ ] Test environment can be **set up and configured** (see Section II.3 - Test Environment)
- [ ] ADR 0051 is accepted and merged
- [ ] PR #1688 changes are merged to main branch
- [ ] Test GitHub org has users with OWNER, MEMBER, COLLABORATOR, CONTRIBUTOR, and NONE associations available

#### **5. Risks**

- [ ] **Timeline/Schedule**
  - Risk: Visible feedback mechanism for unauthorized users is not yet implemented — testing that scenario is blocked.
  - Mitigation: Track as follow-up; test current behavior (silent skip) and verify feedback when implemented.
- [ ] **Test Coverage**
  - Risk: Integration testing of actual GitHub webhook dispatch requires real GitHub events, which are difficult to simulate in unit tests.
  - Mitigation: Use scaffold content tests (`TestDispatchWorkflowContent`) for structural validation; manual or e2e tests for runtime behavior.
- [ ] **Test Environment**
  - Risk: Test org may not have users with all required association levels (OWNER, MEMBER, COLLABORATOR, CONTRIBUTOR, NONE) pre-configured.
  - Mitigation: Create dedicated test users with each association level before test execution begins.
- [ ] **Untestable Aspects**
  - Risk: Cannot directly unit-test GitHub Actions `run:` blocks — they execute in the Actions runtime.
  - Mitigation: Extract testable shell functions; validate workflow content via string assertions in Go tests.
- [ ] **Resource Constraints**
  - Risk: N/A — no additional resource requirements.
  - Mitigation: N/A
- [ ] **Dependencies**
  - Risk: Depends on GitHub `author_association` field being populated correctly for all event types.
  - Mitigation: This is a stable GitHub API feature; document expected values in test setup.
- [ ] **Other**
  - Risk: ADR reference mismatch in `docs/architecture.md` (links to 0050 file instead of 0051).
  - Mitigation: Fix link in a follow-up commit before merge.

---

### **III. Test Scenarios & Traceability**

This section links requirements to test coverage, enabling reviewers to verify all requirements are tested.

#### **1. Requirements-to-Tests Mapping**

- **[GH-1662]** -- All slash commands enforce authorization before dispatching agent runs
  - *Test Scenario:* Verify authorized user triggers /fs-triage successfully
  - *Tier:* Functional
  - *Priority:* P0
  - *Test Scenario:* Verify unauthorized user cannot trigger /fs-triage
  - *Tier:* Functional
  - *Priority:* P0
  - *Test Scenario:* Verify unauthorized user cannot trigger /fs-code
  - *Tier:* Functional
  - *Priority:* P0
  - *Test Scenario:* Verify unauthorized user cannot trigger /fs-review
  - *Tier:* Functional
  - *Priority:* P0
  - *Test Scenario:* Verify CONTRIBUTOR association is rejected for slash commands
  - *Tier:* Functional
  - *Priority:* P0

- **[GH-1662]** -- PR event triggers enforce actor authorization for auto-review
  - *Test Scenario:* Verify member PR triggers auto-review
  - *Tier:* Functional
  - *Priority:* P0
  - *Test Scenario:* Verify external contributor PR skips auto-review
  - *Tier:* Functional
  - *Priority:* P0
  - *Test Scenario:* Verify PR synchronize by non-member skips review
  - *Tier:* Functional
  - *Priority:* P0

- **[GH-1662]** -- Auto-triage on issues.opened/edited remains ungated
  - *Test Scenario:* Verify external user issue triggers auto-triage
  - *Tier:* Functional
  - *Priority:* P0
  - *Test Scenario:* Verify edited issue re-triggers triage without auth
  - *Tier:* Functional
  - *Priority:* P0

- **[GH-1662]** -- Bot-to-bot agent handoffs via labels are unaffected by authorization gates
  - *Test Scenario:* Verify label-based handoff triggers downstream agent
  - *Tier:* Functional
  - *Priority:* P1
  - *Test Scenario:* Verify bot slash command is blocked by non-Bot check
  - *Tier:* Functional
  - *Priority:* P1

- **[GH-1662]** -- Authorized users can invoke all slash commands successfully
  - *Test Scenario:* Verify OWNER can invoke /fs-triage, /fs-code, /fs-review, /fs-fix, /fs-retro, and /fs-prioritize
  - *Tier:* End-to-End
  - *Priority:* P1
  - *Test Scenario:* Verify MEMBER can invoke /fs-triage, /fs-code, /fs-review, /fs-fix, /fs-retro, and /fs-prioritize
  - *Tier:* End-to-End
  - *Priority:* P1
  - *Test Scenario:* Verify COLLABORATOR can invoke /fs-triage, /fs-code, /fs-review, /fs-fix, /fs-retro, and /fs-prioritize
  - *Tier:* End-to-End
  - *Priority:* P1

- **[GH-1662]** -- Per-repo and per-org dispatch templates are consistent in authorization behavior
  - *Test Scenario:* Verify per-repo dispatch has identical auth gates
  - *Tier:* Unit Tests
  - *Priority:* P1
  - *Test Scenario:* Verify per-org scaffold dispatch has identical auth gates
  - *Tier:* Unit Tests
  - *Priority:* P1

- **[GH-1662]** -- Previously gated commands remain correctly gated after dispatch changes
  - *Test Scenario:* Verify /fs-fix still requires authorization after dispatch routing changes
  - *Tier:* Functional
  - *Priority:* P1
  - *Test Scenario:* Verify /fs-retro still requires authorization after dispatch routing changes
  - *Tier:* Functional
  - *Priority:* P1
  - *Test Scenario:* Verify /fs-prioritize still requires authorization after dispatch routing changes
  - *Tier:* Functional
  - *Priority:* P1

- **[GH-1662]** -- Unauthorized slash command attempts produce visible feedback
  - *Test Scenario:* Verify unauthorized command produces user-visible response
  - *Tier:* End-to-End
  - *Priority:* P2
  - *Test Scenario:* Verify silent skip for unauthorized PR event trigger
  - *Tier:* Functional
  - *Priority:* P2

- **[GH-1662]** -- is_event_actor_authorized correctly validates all association types
  - *Test Scenario:* Verify OWNER association returns authorized
  - *Tier:* Unit Tests
  - *Priority:* P1
  - *Test Scenario:* Verify empty association string returns unauthorized
  - *Tier:* Unit Tests
  - *Priority:* P1
  - *Test Scenario:* Verify FIRST_TIME_CONTRIBUTOR is rejected
  - *Tier:* Unit Tests
  - *Priority:* P1
  - *Test Scenario:* Verify NONE association is rejected
  - *Tier:* Unit Tests
  - *Priority:* P1

---

### **IV. Sign-off and Approval**

This Software Test Plan requires approval from the following stakeholders:

* **Reviewers:**
  - @ascerra
  - TBD
* **Approvers:**
  - TBD
  - TBD
