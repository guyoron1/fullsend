# FullSend Test Plan

## **fix(#2247): Compare Decoded Text in Shim Drift Detection - Quality Engineering Plan**

### Metadata & Tracking

- **Enhancement:** [GH-8](https://github.com/guyoron1/fullsend/pull/8)
- **Feature Tracking:** [GH-8](https://github.com/guyoron1/fullsend/pull/8) — fix(#2247): compare decoded text in shim drift detection
- **Epic Tracking:** [fullsend-ai#2247](https://github.com/fullsend-ai/fullsend/issues/2247)
- **QE Owner:** TBD
- **Owning SIG:** N/A
- **Participating SIGs:** N/A

**Document Conventions:** Standard QualityFlow STP conventions apply. Test IDs use the format `TS-GH-8-NNN`.

### Feature Overview

This bug fix corrects false-positive drift detection in the `reconcile-repos.sh` shim reconciliation script. The original implementation compared re-encoded base64 strings, which could differ due to Bash command substitution stripping trailing newlines, even when the decoded content was identical. This caused bogus update PRs that removed sentinel lines from enrolled repositories. The fix replaces base64-to-base64 comparison with decoded text comparison, stripping carriage returns before comparing. A regression test is included to prevent recurrence.

---

### Section I — Motivation & Requirements Review

#### I.1 — Requirement & User Story Review Checklist

- [ ] **Reviewed the relevant requirements.** -- Confirmed that the requirement documents and design specs have been reviewed for completeness.
  - Upstream issue fullsend-ai#2247 describes false-positive drift detection causing bogus update PRs (e.g., fullsend-ai#2101).
  - Root cause: Bash `$()` command substitution strips trailing newlines, causing re-encoded base64 to differ from the original.

- [ ] **Confirmed clear user stories and understood. Understand the value and customer use cases.** -- Validated that user stories are well-defined and the customer value proposition is understood.
  - User impact: enrolled repos received unnecessary "update fullsend shim workflow" PRs even when the shim content was already correct.
  - Value: eliminates noise PRs and prevents accidental removal of sentinel lines in shim workflows.

- [ ] **Confirmed requirements are **testable and unambiguous**.** -- Verified that each requirement can be validated through testing.
  - The fix is testable: provide logically identical content with different trailing newlines and verify no stale detection.
  - A regression test is included in the PR itself (`reconcile-repos-test.sh`).

- [ ] **Ensured acceptance criteria are **defined clearly**.** -- Checked that acceptance criteria leave no room for interpretation.
  - AC1: Identical shim content with different trailing newlines must NOT be flagged as stale.
  - AC2: Genuinely different shim content must still be correctly flagged as stale.
  - AC3: Existing enrollment and update flows must continue to work.

- [ ] **Confirmed coverage for NFRs.** -- Reviewed non-functional requirements (performance, security, reliability).
  - No performance regression expected; `base64 -d` decode is equivalent cost to the previous encode path.
  - No security implications; comparison logic is local to the script with no external input injection.

#### I.2 — Known Limitations

- The fix addresses only the trailing-newline encoding mismatch. Other potential base64 encoding differences (e.g., line-wrapping at 76 characters vs no wrapping) are handled by the existing `tr -d '\r\n'` strip but are not explicitly regression-tested.
- The regression test uses mocked `gh` commands; it does not validate against the real GitHub Contents API base64 encoding behavior.

#### I.3 — Technology and Design Review

- [ ] **Developer handoff completed.** -- Confirmed knowledge transfer from development to QE.
  - PR description clearly explains the root cause (Bash command substitution stripping trailing newlines) and the fix approach (decoded text comparison).

- [ ] **Technology challenges identified.** -- Reviewed technical risks and challenges for testing.
  - The script relies on platform `base64` command behavior which may vary between GNU coreutils and macOS/BSD. The `-w0` flag is GNU-specific.

- [ ] **Test environment needs assessed.** -- Evaluated infrastructure and environment requirements.
  - Tests run in Bash with mocked commands; no cluster or external infrastructure required.

- [ ] **API extensions reviewed.** -- Checked for new or modified API contracts.
  - No API changes. The fix is internal to the `reconcile-repos.sh` comparison logic.

- [ ] **Topology and deployment considerations reviewed.** -- Assessed multi-node, HA, or special topology needs.
  - N/A — this is a CI script, not a deployed service.

---

### Section II — Test Planning

#### II.1 — Scope of Testing

This test plan covers the shim drift detection logic in `reconcile-repos.sh`, specifically the comparison of managed (expected) shim content against the remote (GitHub-hosted) shim content during repo enrollment reconciliation. The fix changes the comparison from re-encoded base64 to decoded text, and this plan validates that the new comparison correctly distinguishes identical from truly different content.

**Testing Goals**

- **P0:** Verify that logically identical content with different base64 encoding (trailing newlines) is NOT flagged as stale.
- **P0:** Verify that genuinely different shim content IS correctly flagged as stale and triggers an update PR.
- **P1:** Verify that the enrollment workflow (enroll, skip, update) functions correctly end-to-end with the new comparison logic.
- **P2:** Verify handling of edge cases in content encoding (CR/LF variations, empty content).

**Out of Scope (Testing Scope Exclusions)**

- [ ] **GitHub Contents API behavior** -- The actual base64 encoding behavior of GitHub's API is platform-level; we test with mocked responses.
  - Rationale: Platform API behavior is owned by GitHub, not FullSend QE.
- [ ] **Bash `base64` cross-platform compatibility** -- GNU vs BSD `base64` flag differences.
  - Rationale: CI runs on Ubuntu (GNU coreutils); cross-platform shell compatibility is out of scope.
- [ ] **Branch protection and PR merge behavior** -- Whether the created PRs can be merged under branch protection rules.
  - Rationale: GitHub branch protection is platform-level functionality.

#### II.2 — Test Strategy

**Functional**

- [x] **Functional Testing** -- Validate core functionality works as expected.
  - Verify the decoded-text comparison correctly handles identical and different content scenarios.
- [x] **Automation Testing** -- Confirm tests can be automated and integrated into CI.
  - The regression test in `reconcile-repos-test.sh` is already automated and runs in CI.
- [x] **Regression Testing** -- Ensure existing functionality is not broken by changes.
  - The existing stale-shim atomic update test continues to pass; new regression test added for #2247.
- [ ] **Upgrade Testing** -- Validate upgrade paths.
  - N/A: No stateful upgrade; script is re-embedded on each build.

**Non-Functional**

- [ ] **Performance Testing** -- Evaluate performance impact of changes.
  - N/A: `base64 -d` decode has negligible performance difference from the previous approach.
- [ ] **Scale Testing** -- Test behavior under scale conditions.
  - N/A: The comparison runs once per enrolled repo; scale is bounded by config.yaml repo count.
- [ ] **Security Testing** -- Validate security posture.
  - N/A: No change to input validation, authentication, or authorization paths.
- [ ] **Usability Testing** -- Assess user experience impact.
  - N/A: No user-facing interface changes.
- [ ] **Monitoring** -- Verify observability and alerting.
  - N/A: Script output logging unchanged.

**Integration & Compatibility**

- [ ] **Compatibility Testing** -- Test across supported configurations.
  - N/A: Script targets GitHub Actions Ubuntu runners only.
- [ ] **Dependencies** -- Verify dependency compatibility.
  - Depends on `base64`, `tr`, `printf` from GNU coreutils (pre-installed on GitHub Actions runners).
- [ ] **Cross Integrations** -- Test interactions with other systems.
  - N/A: No new integrations introduced.

**Infrastructure**

- [ ] **Cloud Testing** -- Validate cloud-specific behavior.
  - N/A: No cloud infrastructure changes.

#### II.3 — Test Environment

- **Cluster Topology:** N/A — no cluster required; tests run in Bash
- **Platform Version:** GitHub Actions Ubuntu runner (latest)
- **CPU Virtualization:** N/A
- **Compute:** Standard GitHub Actions runner
- **Special Hardware:** None
- **Storage:** Ephemeral runner filesystem
- **Network:** GitHub API access (mocked in unit tests)
- **Operators:** None
- **Platform:** GitHub Actions CI
- **Special Configs:** Mocked `gh`, `yq`, `base64` commands in test PATH

#### II.3.1 — Testing Tools & Frameworks

No new or special tools required. Tests use standard Bash with mocked commands.

#### II.4 — Entry Criteria

- [ ] PR branch `fix/2247-shim-stale-comparison` builds successfully
- [ ] All existing `reconcile-repos-test.sh` tests pass before adding new test
- [ ] Go module compiles with embedded scaffold files (`go build ./...`)

#### II.5 — Risks

- [ ] **Timeline**
  - Risk: None — fix is small and self-contained.
  - Mitigation: N/A
  - Status: [ ] No risk identified

- [ ] **Coverage**
  - Risk: Mocked tests may not cover all real-world GitHub API base64 encoding variations.
  - Mitigation: The fix is defensive (decodes both sides and strips CR), covering known encoding variations.
  - Status: [ ] Acceptable risk

- [ ] **Environment**
  - Risk: Test environment differences between local dev and CI runners.
  - Mitigation: Tests use `/usr/bin/base64` explicitly for the mock, matching CI behavior.
  - Status: [ ] Mitigated

- [ ] **Untestable**
  - Risk: Cannot test actual GitHub Contents API encoding quirks without live API calls.
  - Mitigation: The decoded-text comparison is encoding-agnostic by design.
  - Status: [ ] Acceptable risk

- [ ] **Resources**
  - Risk: None — no additional resources required.
  - Mitigation: N/A
  - Status: [ ] No risk identified

- [ ] **Dependencies**
  - Risk: GNU coreutils `base64 -d` flag availability.
  - Mitigation: GitHub Actions runners always include GNU coreutils.
  - Status: [ ] No risk identified

- [ ] **Other**
  - Risk: None identified.
  - Mitigation: N/A
  - Status: [ ] No risk identified

---

### Section III — Requirements-to-Tests Mapping

#### III.1 — Requirements Mapping

- **Requirement ID:** GH-8
- **Requirement Summary:** Shim drift detection correctly identifies identical vs different content
- **Test Scenarios:**
  - TS-GH-8-001: Verify identical content with different trailing newlines is not flagged as stale
  - TS-GH-8-002: Verify genuinely stale shim content is correctly detected
  - TS-GH-8-003: Verify comparison handles CR/LF encoding variations
- **Tier:** [Functional]
- **Priority:** P0

- **Requirement ID:**
- **Requirement Summary:** Enrollment workflow functions correctly with decoded-text comparison
- **Test Scenarios:**
  - TS-GH-8-004: Verify enrollment skips repos with up-to-date shim
  - TS-GH-8-005: Verify enrollment creates update PR for stale shim
- **Tier:** [Functional]
- **Priority:** P1

- **Requirement ID:**
- **Requirement Summary:** Regression test validates fix for issue #2247
- **Test Scenarios:**
  - TS-GH-8-006: Verify reconcile-repos-test.sh regression test passes end-to-end
- **Tier:** [Functional]
- **Priority:** P0

---

### Section IV — Sign-off

| Role | Name | Date |
|:-----|:-----|:-----|
| QE Lead | TBD | |
| Dev Lead | guyoron1 | |
| Product Owner | TBD | |
