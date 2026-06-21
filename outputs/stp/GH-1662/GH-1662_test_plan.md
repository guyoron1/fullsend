# FullSend Test Plan

## **Require Authorization on All Agent Dispatch Paths - Quality Engineering Plan**

### Metadata & Tracking

- **Enhancement(s):** [GH-1662](https://github.com/fullsend-ai/fullsend/issues/1662)
- **Feature Tracking:** [PR #1688](https://github.com/fullsend-ai/fullsend/pull/1688)
- **Epic Tracking:** GH-1662
- **QE Owner(s):** TBD
- **Owning SIG:** N/A
- **Participating SIGs:** None

### Document Conventions

N/A

### Feature Overview

This feature enforces the `is_authorized` check on all agent slash commands (`/fs-triage`, `/fs-code`, `/fs-review`) and automatic PR event triggers (`pull_request_target.opened/synchronize/ready_for_review`), closing cost-exposure and abuse-surface gaps. Previously, only `/fs-fix`, `/fs-retro`, and `/fs-prioritize` required authorization. The change introduces a new `is_event_actor_authorized()` helper for non-comment triggers and intentionally leaves auto-triage on `issues.opened/edited` ungated to preserve the drive-by bug reporter workflow. Bot-to-bot orchestration via label-based triggers is unaffected.

---

### Section I - Motivation and Requirements Review

#### I.1 - Requirement & User Story Review Checklist

- [ ] **Review Requirements**
  - Reviewed the relevant requirements.
  - ADR 0051 documents the decision to require authorization on all agent dispatch paths. Requirements are sourced from the issue body, ADR text, and PR #1688 implementation.

- [ ] **Understand Value and Customer Use Cases**
  - Confirmed clear user stories and understood. Understand the value and customer use cases.
  - Primary value: prevent unauthorized users from triggering inference runs at org expense. Closes cost-exposure and prompt-injection abuse-surface gaps on public repos.

- [ ] **Testability**
  - Confirmed requirements are **testable and unambiguous**.
  - Authorization logic is implemented as shell functions (`is_authorized`, `is_event_actor_authorized`) in dispatch workflow YAML. Testable via workflow dispatch simulation and by validating the YAML routing logic.

- [ ] **Acceptance Criteria**
  - Ensured acceptance criteria are **defined clearly**.
  - Acceptance criteria derived from ADR 0051: all slash commands gated, PR event triggers gated, `issues.opened/edited` ungated, bot-to-bot workflows preserved, visible feedback for unauthorized users.

- [ ] **Non-Functional Requirements**
  - Confirmed coverage for NFRs including Performance, Security, Usability, Downtime, Connectivity, Monitoring, Scalability, Portability, and Docs.
  - Security is the primary NFR: authorization boundary is a platform-level security control. Usability NFR: unauthorized users receive visible feedback. Documentation updated across all agent docs.

#### I.2 - Known Limitations

- Authorization relies on GitHub's `author_association` field, which only distinguishes OWNER, MEMBER, COLLABORATOR, CONTRIBUTOR, FIRST_TIMER, FIRST_TIME_CONTRIBUTOR, and NONE. Fine-grained per-user or per-team authorization is not supported.
- Auto-triage on `issues.opened/edited` remains ungated. Abuse mitigation for this path is deferred to per-user rate limiting ([GH-1687](https://github.com/fullsend-ai/fullsend/issues/1687)).
- No visible feedback mechanism is implemented in this PR for unauthorized slash command attempts. The ADR requires it but defers the exact mechanism to implementation.
- The `is_event_actor_authorized()` helper reads from `github.event.pull_request.author_association` only; issue-level event actor association is not checked (auto-triage is ungated by design).

#### I.3 - Technology and Design Review

- [ ] **Developer Handoff/QE Kickoff**
  - Developer handoff completed via ADR 0051 and PR #1688 description. Architectural decision documented with full rationale.

- [ ] **Technology Challenges**
  - Changes are in GitHub Actions workflow YAML (bash scripting), not Go code. Testing requires understanding of GitHub Actions event payloads and `author_association` field semantics.

- [ ] **Test Environment Needs**
  - Requires a GitHub repository with configurable collaborator roles, or simulation of GitHub event payloads with varying `author_association` values.

- [ ] **API Extensions**
  - No new APIs. New `is_event_actor_authorized()` bash helper added to dispatch workflow. New `PR_AUTHOR_ASSOC` environment variable passed from GitHub event context.

- [ ] **Topology Considerations**
  - Changes apply to both per-repo (`dispatch.yml` in scaffold) and per-org (`reusable-dispatch.yml`) install modes. Both paths must be tested.

### Section II - Test Planning

#### II.1 - Scope of Testing

This test plan covers the authorization enforcement on all agent dispatch paths in the FullSend platform. Testing will verify that slash commands and PR event triggers correctly gate on user authorization, that intentionally ungated paths remain functional, and that bot-to-bot workflows are preserved.

**Testing Goals:**

**Functional Goals:**
- **P0:** Verify all slash commands (`/fs-triage`, `/fs-code`, `/fs-review`) enforce `is_authorized` for non-Bot users
- **P0:** Verify `pull_request_target` event triggers enforce `is_event_actor_authorized`
- **P0:** Verify `issues.opened/edited` auto-triage remains ungated
- **P1:** Verify bot-to-bot label-based workflows are unaffected by authorization gates
- **P1:** Verify previously-gated commands (`/fs-fix`, `/fs-retro`, `/fs-prioritize`) remain correctly gated

**Quality Goals:**
- **P1:** Verify fail-closed behavior when `author_association` is empty or missing

**Integration Goals:**
- **P1:** Verify per-repo scaffold `dispatch.yml` and per-org `reusable-dispatch.yml` have consistent authorization logic
- **P2:** Verify documentation accurately reflects authorization requirements

**Out of Scope (Testing Scope Exclusions):**

- [ ] GitHub Actions platform reliability -- *Rationale:* GitHub Actions execution is a platform concern, not a product concern. -- *PM/Lead Agreement:* TBD
- [ ] Per-user rate limiting for auto-triage -- *Rationale:* Deferred to GH-1687; not implemented in this change. -- *PM/Lead Agreement:* TBD
- [ ] GitHub `author_association` field accuracy -- *Rationale:* GitHub platform responsibility; FullSend consumes the field as-is. -- *PM/Lead Agreement:* TBD
- [ ] Visible feedback UX for unauthorized users -- *Rationale:* ADR 0051 requires it but the exact mechanism is an implementation detail not addressed in PR #1688. -- *PM/Lead Agreement:* TBD

#### II.2 - Test Strategy

**Functional:**

- [x] **Functional Testing**
  - Validate authorization gates on all slash command and event trigger dispatch paths. Cover authorized (OWNER, MEMBER, COLLABORATOR) and unauthorized (CONTRIBUTOR, NONE, FIRST_TIMER) associations.

- [x] **Automation Testing**
  - Dispatch routing logic is testable via Go tests that parse and validate the scaffold YAML templates. Existing `workflow_call_alignment_test.go` provides a pattern.

- [x] **Regression Testing**
  - Verify previously-gated commands remain gated and that ungated paths (`issues.opened`, `issues.labeled`) are not accidentally gated.

**Non-Functional:**

- [ ] **Performance Testing**
  - N/A -- Authorization check is a simple string match; no performance concern.

- [ ] **Scale Testing**
  - N/A -- No scale dimension to this change.

- [x] **Security Testing**
  - Core focus of this feature. Verify that unauthorized users cannot trigger agent runs via slash commands or PR events.

- [ ] **Usability Testing**
  - N/A -- No UI changes. Visible feedback mechanism deferred.

- [ ] **Monitoring**
  - N/A -- No new monitoring surfaces introduced.

**Integration & Compatibility:**

- [x] **Compatibility Testing**
  - Verify both per-repo and per-org install modes apply authorization consistently.

- [ ] **Upgrade Testing**
  - N/A -- Workflow files are overwritten during scaffold sync; no upgrade path concerns.

- [x] **Dependencies**
  - Depends on GitHub `author_association` field being populated correctly in event payloads.

- [ ] **Cross Integrations**
  - N/A -- No cross-product integrations affected.

**Infrastructure:**

- [ ] **Cloud Testing**
  - N/A -- No cloud infrastructure changes.

#### II.3 - Test Environment

- **Cluster Topology:** N/A -- No cluster required. Tests operate on workflow YAML files and GitHub API event payloads.
- **Platform & Product Version(s):** GitHub Actions (current), Go 1.23+
- **CPU Virtualization:** N/A
- **Compute Resources:** Standard CI runner
- **Special Hardware:** None
- **Storage:** N/A
- **Network:** GitHub API access required for integration tests
- **Required Operators:** None
- **Platform:** GitHub Actions
- **Special Configurations:** Test GitHub org with users of varying `author_association` levels (OWNER, MEMBER, COLLABORATOR, CONTRIBUTOR, NONE) for integration tests

#### II.3.1 - Testing Tools & Frameworks

N/A -- Standard Go testing tools only.

#### II.4 - Entry Criteria

- [ ] Requirements and design documents are **approved and merged**
- [ ] Test environment can be **set up and configured** (see Section II.3 - Test Environment)
- [ ] ADR 0051 is accepted and merged
- [ ] PR #1688 changes are merged to main branch
- [ ] Scaffold `dispatch.yml` and `reusable-dispatch.yml` both contain `is_event_actor_authorized` helper

#### II.5 - Risks

- [ ] **Timeline/Schedule**
  - *Risk:* Low risk -- changes are self-contained in workflow YAML
  - *Mitigation:* Feature is already implemented in PR #1688
  - *Impact:* Minimal

- [ ] **Test Coverage**
  - *Risk:* Integration tests with real GitHub event payloads may be difficult to automate for all `author_association` values
  - *Mitigation:* Use Go unit tests to parse and validate YAML dispatch logic; supplement with manual integration testing
  - *Impact:* Medium -- some coverage gaps in live event simulation

- [ ] **Test Environment**
  - *Risk:* Requires GitHub org with users of specific roles for integration tests
  - *Mitigation:* Use existing test org or mock event payloads
  - *Impact:* Low

- [ ] **Untestable Aspects**
  - *Risk:* GitHub's `author_association` field behavior for edge cases (deleted users, suspended accounts) cannot be fully tested
  - *Mitigation:* Test fail-closed behavior with empty/missing association values
  - *Impact:* Low

- [ ] **Resource Constraints**
  - *Risk:* N/A -- no special resources needed
  - *Mitigation:* N/A
  - *Impact:* None

- [ ] **Dependencies**
  - *Risk:* Depends on GitHub correctly populating `author_association` in all event types
  - *Mitigation:* Document dependency; test with known-good event payloads
  - *Impact:* Low -- GitHub API behavior is well-documented

- [ ] **Other**
  - *Risk:* N/A
  - *Mitigation:* N/A
  - *Impact:* None

---

### Section III - Requirements-to-Tests Mapping

- **[GH-1662]** -- Slash commands enforce authorization for non-Bot commenters
  - *Test Scenario:* Verify `/fs-triage` is blocked for unauthorized commenters (CONTRIBUTOR, NONE)
  - *Tier:* Functional
  - *Priority:* P0

- **[GH-1662]** -- Slash commands enforce authorization for non-Bot commenters
  - *Test Scenario:* Verify `/fs-code` is blocked for unauthorized commenters
  - *Tier:* Functional
  - *Priority:* P0

- **[GH-1662]** -- Slash commands enforce authorization for non-Bot commenters
  - *Test Scenario:* Verify `/fs-review` is blocked for unauthorized commenters
  - *Tier:* Functional
  - *Priority:* P0

- **[GH-1662]** -- Slash commands permit authorized users (OWNER, MEMBER, COLLABORATOR)
  - *Test Scenario:* Verify `/fs-triage`, `/fs-code`, `/fs-review` dispatch correctly for authorized users
  - *Tier:* Functional
  - *Priority:* P0

- **[GH-1662]** -- PR event triggers enforce actor authorization
  - *Test Scenario:* Verify `pull_request_target.opened` is blocked for non-member PR authors
  - *Tier:* Functional
  - *Priority:* P0

- **[GH-1662]** -- PR event triggers enforce actor authorization
  - *Test Scenario:* Verify `pull_request_target.synchronize` is blocked for non-member PR authors
  - *Tier:* Functional
  - *Priority:* P0

- **[GH-1662]** -- PR event triggers enforce actor authorization
  - *Test Scenario:* Verify `pull_request_target.ready_for_review` is blocked for non-member PR authors
  - *Tier:* Functional
  - *Priority:* P0

- **[GH-1662]** -- PR event triggers permit authorized PR authors
  - *Test Scenario:* Verify `pull_request_target` events dispatch review for OWNER/MEMBER/COLLABORATOR PR authors
  - *Tier:* Functional
  - *Priority:* P0

- **[GH-1662]** -- Auto-triage on issue creation remains ungated
  - *Test Scenario:* Verify `issues.opened` triggers triage without authorization check
  - *Tier:* Functional
  - *Priority:* P0

- **[GH-1662]** -- Auto-triage on issue edit remains ungated
  - *Test Scenario:* Verify `issues.edited` triggers triage without authorization check
  - *Tier:* Functional
  - *Priority:* P0

- **[GH-1662]** -- Bot-to-bot label-based workflows are preserved
  - *Test Scenario:* Verify `issues.labeled` with `ready-to-code` dispatches code stage without authorization gate
  - *Tier:* Functional
  - *Priority:* P1

- **[GH-1662]** -- Bot-to-bot label-based workflows are preserved
  - *Test Scenario:* Verify `issues.labeled` with `ready-for-review` dispatches review stage without authorization gate
  - *Tier:* Functional
  - *Priority:* P1

- **[GH-1662]** -- Previously-gated commands remain correctly gated
  - *Test Scenario:* Verify `/fs-fix`, `/fs-retro`, `/fs-prioritize` still enforce `is_authorized`
  - *Tier:* Functional
  - *Priority:* P1

- **[GH-1662]** -- Bot comments are rejected on all slash commands
  - *Test Scenario:* Verify Bot `COMMENT_USER_TYPE` short-circuits slash command dispatch to no-op
  - *Tier:* Functional
  - *Priority:* P1

- **[GH-1662]** -- Fail-closed behavior for missing or empty association
  - *Test Scenario:* Verify empty `author_association` results in denied dispatch
  - *Tier:* Functional
  - *Priority:* P0

- **[GH-1662]** -- Fail-closed behavior for missing or empty association
  - *Test Scenario:* Verify unknown `author_association` value (e.g., `FIRST_TIMER`) results in denied dispatch
  - *Tier:* Functional
  - *Priority:* P1

- **[GH-1662]** -- Per-repo and per-org dispatch consistency
  - *Test Scenario:* Verify scaffold `dispatch.yml` and `reusable-dispatch.yml` have identical authorization logic
  - *Tier:* Functional
  - *Priority:* P1

- **[GH-1662]** -- `pull_request_target.closed` remains ungated
  - *Test Scenario:* Verify retro stage fires on PR close without authorization check
  - *Tier:* Functional
  - *Priority:* P1

- **[GH-1662]** -- End-to-end unauthorized user slash command flow
  - *Test Scenario:* Verify external contributor posts `/fs-code` and no agent run is triggered
  - *Tier:* End-to-End
  - *Priority:* P0

- **[GH-1662]** -- End-to-end authorized user slash command flow
  - *Test Scenario:* Verify org member posts `/fs-triage` and triage agent dispatches successfully
  - *Tier:* End-to-End
  - *Priority:* P0

- **[GH-1662]** -- End-to-end external PR author flow
  - *Test Scenario:* Verify external contributor opens PR and review agent does not auto-trigger
  - *Tier:* End-to-End
  - *Priority:* P1

- **[GH-1662]** -- End-to-end auto-triage for external issue reporter
  - *Test Scenario:* Verify external user opens issue and triage agent fires without authorization
  - *Tier:* End-to-End
  - *Priority:* P1

---

### Section IV - Sign-off

| Role | Name | Date |
|:-----|:-----|:-----|
| QE Lead | TBD | |
| Dev Lead | TBD | |
| PM | TBD | |
