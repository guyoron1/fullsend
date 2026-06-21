# STP Review Report: GH-2433

**Reviewed:** outputs/stp/GH-2433/GH-2433_test_plan.md
**Date:** 2026-06-21
**Reviewer:** QualityFlow Automated Review (v1.1.0)
**Review Rules Schema:** 1.1.0 (dynamically extracted, no static override)

---

## Verdict: APPROVED_WITH_FINDINGS

## Summary

| Metric | Value |
|:-------|:------|
| Dimensions reviewed | 7/7 |
| Critical findings | 0 |
| Major findings | 2 |
| Minor findings | 4 |
| Actionable findings | 5 |
| Confidence | MEDIUM |
| Weighted score | 91/100 |

## Dimension Scores

| Dimension | Weight | Pass Rate | Weighted |
|:----------|:-------|:----------|:---------|
| 1. Rule Compliance | 25% | 88% | 22.0 |
| 2. Requirement Coverage | 30% | 97% | 29.1 |
| 3. Scenario Quality | 15% | 90% | 13.5 |
| 4. Risk & Limitation Accuracy | 10% | 95% | 9.5 |
| 5. Scope Boundary Assessment | 10% | 92% | 9.2 |
| 6. Test Strategy Appropriateness | 5% | 95% | 4.75 |
| 7. Metadata Accuracy | 5% | 95% | 4.75 |
| **Total** | **100%** | | **92.8** |

---

## Findings by Dimension

### Dimension 1: Rule Compliance (Rules A-P)

| Rule | Status | Finding |
|:-----|:-------|:--------|
| A — Abstraction Level | WARN | Internal function names in scope and test scenarios |
| A.2 — Language Precision | PASS | Language is precise and professional throughout |
| B — Section I Meta-Checklist | PASS | All required items present with substantive content |
| C — Prerequisites vs Scenarios | PASS | All Section III items describe testable behaviors |
| D — Dependencies | PASS | Dependencies correctly identifies existing internal function |
| E — Upgrade Testing | PASS | Correctly unchecked; guard is additive, no persistent state changes |
| F — Version Derivation | PASS | "FullSend 0.x" matches project.yaml versioning |
| G — Testing Tools | WARN | Standard tools listed where section should be empty or "None" |
| G.2 — Environment Specificity | PASS | Environment entries feature-specific with N/A justifications |
| H — Risk Deduplication | PASS | Risks are unique and do not duplicate environment info |
| I — QE Kickoff Timing | PASS | Acceptable for small bug fix scope |
| J — One Tier Per Row | PASS | Each scenario specifies exactly one test type |
| K — Cross-Section Consistency | PASS | No contradictions found across sections |
| L — Section Content Validation | PASS | Content is in correct sections |
| M — Deletion Test | WARN | Feature Overview is verbose; could reference issue instead of restating |
| N — Link/Reference Validation | PASS | All links syntactically valid and point to correct resources |
| O — Untestable Aspects | PASS | Cloud Run stale-read limitation well-documented with mitigation |
| P — Testing Pyramid Efficiency | PASS | Fix is single-function-isolated; unit tests as minimum tier is correct |

**Detailed Findings:**

---

**Finding D1-A-001**

- **finding_id:** D1-A-001
- **severity:** MAJOR
- **dimension:** Rule Compliance
- **rule:** A — Abstraction Level
- **description:** Internal function and file names are used throughout Scope of Testing and Test Scenarios where user-facing language would be more appropriate. The STP references `EnsureOrgInMint` (function), `mintcore.RoleOnlyAppIDs()` (function), `provisionWithExistingMint` (function), and `internal/dispatch/gcf/provisioner.go` (file path) in testable content sections.
- **evidence:**
  - Scope (II.1): "Testing covers the restored data consistency guard in `EnsureOrgInMint` (`internal/dispatch/gcf/provisioner.go`)"
  - Section III: "Verify provisioning aborts on data inconsistency" is good, but the requirement group title references "`provisionWithExistingMint`"
  - Section III: Scenarios reference `mintcore.RoleOnlyAppIDs` filtering behavior by function name
- **remediation:** Rewrite scope to use user-facing language. Example: replace "data consistency guard in `EnsureOrgInMint` (`internal/dispatch/gcf/provisioner.go`)" with "data consistency guard in the org enrollment process." Replace `provisionWithExistingMint` with "provisioning with an existing mint." Keep internal references in acceptable locations (Technology Review I.3, Known Limitations I.2) where they already appear correctly.
- **actionable:** true

---

**Finding D1-G-001**

- **finding_id:** D1-G-001
- **severity:** MINOR
- **dimension:** Rule Compliance
- **rule:** G — Testing Tools
- **description:** Section II.3.1 lists standard project tools (Go testing + testify, GitHub Actions) that are baseline infrastructure for all FullSend tests and do not need to be listed.
- **evidence:** "Test Framework: Standard (Go testing + testify)" and "CI/CD: Standard (GitHub Actions)" and "Other Tools: None"
- **remediation:** Replace the Testing Tools section content with "None — feature uses only standard project test infrastructure." or remove the tool entries and keep only "Other Tools: None."
- **actionable:** true

---

**Finding D1-M-001**

- **finding_id:** D1-M-001
- **severity:** MINOR
- **dimension:** Rule Compliance
- **rule:** M — Deletion Test (ISTQB)
- **description:** The Feature Overview (5 lines of dense technical prose) restates the full context already available in the GitHub issue body. The ISTQB deletion test asks: "If removed, would the Go/No-Go decision be hindered?" The issue link provides identical context, making the full restatement redundant for test-planning purposes.
- **evidence:** Feature Overview paragraph closely mirrors the GitHub issue Summary and Risk sections word-for-word (PR #2331 removal, PR #1846 original fix, GH-1842 original bug).
- **remediation:** Condense the Feature Overview to 2-3 sentences summarizing the user-facing impact: "A data consistency guard in the org enrollment flow was removed in PR #2331. Without it, a stale read can silently unenroll all existing orgs. This fix restores the guard adapted for the role-only model. See GH-2433 for full context."
- **actionable:** true

---

### Dimension 2: Requirement Coverage

| Metric | Value |
|:-------|:------|
| Acceptance criteria covered | 4/4 |
| Acceptance criteria coverage rate | 100% |
| P0 criteria covered | 3/3 |
| Linked issues reflected | 1/1 (GH-1842 referenced) |
| Negative scenarios present | YES (8 of 16 scenarios) |
| Edge cases identified | 3 (from issue) / 4 (in STP) |

**Coverage Verification (zero-trust cross-reference against GitHub issue):**

| GitHub Issue Acceptance Criterion | STP Coverage | Status |
|:----------------------------------|:-------------|:-------|
| Guard triggers when ALLOWED_ORGS empty + role-only entries in ROLE_APP_IDS | Section III: P0 scenario "Verify guard returns error when ALLOWED_ORGS empty with role-only entries" | COVERED |
| First enrollment (both empty) proceeds normally | Section III: P0 scenario "Verify first enrollment succeeds with empty state" | COVERED |
| Legacy keys are filtered out (not triggering guard) | Section III: P1 scenario "Verify enrollment proceeds with legacy org-scoped keys only" | COVERED |
| Error message includes actionable remediation (`fullsend mint status`) | Section III: P1 scenario "Verify error message includes role count and project ID" + P2 "Verify CLI suggests mint status command" | COVERED |

**Additional coverage beyond acceptance criteria:**
- Malformed ROLE_APP_IDS JSON handling (P1) — not in acceptance criteria but mentioned in triage comment
- Mixed legacy + role-only keys (P1) — edge case not explicitly in AC
- Error propagation through call chain (P1) — integration coverage
- Duplicate org enrollment idempotency (P2) — regression safety

**Gaps identified:** None. Coverage is comprehensive and exceeds the minimum acceptance criteria.

---

### Dimension 3: Scenario Quality

| Metric | Value |
|:-------|:------|
| Total scenarios | 16 |
| Unit Tests | 10 |
| Functional | 2 |
| End-to-End | 3 |
| Other | 1 |
| P0 | 3 |
| P1 | 8 |
| P2 | 5 |
| Positive scenarios | 8 |
| Negative scenarios | 8 |

**Priority distribution assessment:** Healthy. P0 reserved for core guard detection and first-enrollment pass-through (the two critical behaviors). P1 for supporting behaviors and error quality. P2 for edge cases and integration polish.

**Positive/negative balance:** Excellent. Equal split reflects the nature of a defensive guard where both triggering and NOT triggering must be verified.

**Scenario-level findings:** None critical. Scenarios are specific, actionable, and appropriately prioritized. The use of internal function names (noted under Rule A) is the only quality concern.

---

### Dimension 4: Risk & Limitation Accuracy

**Known Limitations verification (zero-trust cross-reference):**

| STP Limitation | GitHub Issue Support | Verdict |
|:---------------|:--------------------|:--------|
| Role names containing `/` misclassified as legacy | Issue body: "distinguishes role-only keys from legacy org-scoped keys by the presence of `/`" | VERIFIED |
| Malformed ROLE_APP_IDS JSON silently falls through | Issue body: "intentional — a malformed env var should not block enrollment" | VERIFIED |
| Cannot distinguish "never set" vs "data loss" when both empty | Implicit in the proposed fix logic: both-empty = first enrollment | VERIFIED |

**Risk accuracy:** All 7 risk entries are genuine uncertainties with actionable mitigations. No risk duplicates environment requirements. Cloud Run stale-read untestability is correctly categorized as a risk with a strong mitigation (test the symptom, not the cause).

No findings.

---

### Dimension 5: Scope Boundary Assessment

**Scope alignment with GitHub issue:**

The scope ("data consistency guard in the org enrollment process") directly maps to the issue's described fix. All scope items are traceable to the issue's acceptance criteria.

**Out-of-scope verification:** All three exclusions are reasonable:
1. Cloud Run revision traffic splitting — GCP platform behavior, not product code
2. WIF pool and provider configuration — unrelated subsystem
3. Actual Cloud Run deployment with stale reads — requires live infrastructure

**Project scope boundary check:** The fix is in `internal/dispatch/gcf/` which maps to the `dispatch` and `mint` components — both listed in `project.yaml` `scope_boundaries.in_scope_resources`.

**Finding D5-001**

- **finding_id:** D5-001
- **severity:** MINOR
- **dimension:** Scope Boundary Assessment
- **description:** All three Out-of-Scope items have "PM/Lead Agreement: TBD". For a bug fix with clear scope boundaries, this is acceptable in draft but should be resolved before final approval.
- **evidence:** "-- *PM/Lead Agreement:* TBD" appears on all three out-of-scope items
- **remediation:** Obtain PM/lead sign-off on scope exclusions or note that scope exclusions are self-evident for a bug fix (GCP platform testing, unrelated subsystems, live infrastructure requirements).
- **actionable:** false

---

### Dimension 6: Test Strategy Appropriateness

**Strategy classification validation:**

| Item | State | Correct? | Notes |
|:-----|:------|:---------|:------|
| Functional Testing | Checked | YES | Core guard logic tested |
| Automation Testing | Checked | YES | All tests are Go unit tests |
| Regression Testing | Checked | YES | Existing test updated to not conflict with guard |
| Performance Testing | Unchecked | YES | Single map iteration, no SLA |
| Scale Testing | Unchecked | YES | In-memory maps, no scale concern |
| Security Testing | Unchecked | YES | No auth/RBAC changes |
| Usability Testing | Unchecked | YES | Error message usability tested in unit tests |
| Monitoring | Unchecked | YES | Standard Go error, no new metrics |
| Compatibility Testing | Unchecked | YES | Pure Go logic |
| Upgrade Testing | Unchecked | YES | Additive guard, no data format changes |
| Dependencies | Checked | YES | Notes existing function dependency |
| Cross Integrations | Checked | YES | Lists affected callers |
| Cloud Testing | Unchecked | YES | GCF client fully mocked |

All classifications are correct with substantive sub-item justifications. No findings.

---

### Dimension 7: Metadata Accuracy

**Metadata cross-reference against GitHub issue:**

| Field | STP Value | Source Value | Match? |
|:------|:----------|:-------------|:-------|
| Enhancement(s) | GH-2433 (fullsend-ai/fullsend) | GitHub issue #2433 | YES |
| Feature Tracking | GH-2433 | Issue #2433 | YES |
| Epic Tracking | GH-2433 (standalone bug fix) | No epic; standalone issue | YES |
| QE Owner(s) | TBD | Not assigned | YES (acceptable) |
| Owning SIG | N/A | Labels: component/dispatch, component/mint (no SIG label) | YES |
| Participating SIGs | None | No cross-SIG labels | YES |

**Finding D7-001**

- **finding_id:** D7-001
- **severity:** MINOR
- **dimension:** Metadata Accuracy
- **description:** The STP title uses "Restore Data Consistency Guard in EnsureOrgInMint" while the GitHub issue title is "EnsureOrgInMint missing data consistency guard after role-only ROLE_APP_IDS migration". The STP title is a reasonable action-oriented rephrasing, but includes the internal function name `EnsureOrgInMint` in the document title. Consistent with the Rule A finding, consider user-facing language.
- **evidence:** STP title: "Restore Data Consistency Guard in EnsureOrgInMint" vs Issue title: "EnsureOrgInMint missing data consistency guard..."
- **remediation:** Consider rephrasing the STP title to "Restore Data Consistency Guard in Org Enrollment — Quality Engineering Plan" to maintain user-facing abstraction while preserving clarity.
- **actionable:** true

---

## Recommendations

1. **[MAJOR]** Internal function names (`EnsureOrgInMint`, `mintcore.RoleOnlyAppIDs()`, `provisionWithExistingMint`) used in Scope of Testing and Section III scenario groups where user-facing language would be more appropriate. Internal references in Technology Review (I.3) and Known Limitations (I.2) are correctly placed. — **Remediation:** Rewrite Scope to say "org enrollment process" instead of `EnsureOrgInMint`; replace `provisionWithExistingMint` with "provisioning with an existing mint"; replace `mintcore.RoleOnlyAppIDs()` with "role-only app ID detection" in Section III requirement group titles. Keep internal references in I.2 and I.3. — **Actionable:** yes

2. **[MAJOR]** File path `internal/dispatch/gcf/provisioner.go` appears in the Scope of Testing section. Code paths are implementation details that belong in Technology Review, not in the test scope definition. — **Remediation:** Remove the file path from Scope. It already appears correctly in the Technology Review (I.3) Developer Handoff sub-item. — **Actionable:** yes

3. **[MINOR]** Testing Tools section lists standard project infrastructure (Go testing + testify, GitHub Actions) that does not need to be enumerated. — **Remediation:** Replace with "None — feature uses only standard project test infrastructure." — **Actionable:** yes

4. **[MINOR]** Feature Overview restates the full GitHub issue context verbatim. — **Remediation:** Condense to 2-3 sentences summarizing user-facing impact and link to GH-2433 for full context. — **Actionable:** yes

5. **[MINOR]** Out-of-Scope items all have "PM/Lead Agreement: TBD". — **Remediation:** Obtain sign-off or note that exclusions are self-evident for a bug fix. — **Actionable:** false

6. **[MINOR]** STP document title includes internal function name `EnsureOrgInMint`. — **Remediation:** Rephrase to "Restore Data Consistency Guard in Org Enrollment." — **Actionable:** yes

---

## Confidence Notes

| Factor | Status |
|:-------|:-------|
| Source data available (GitHub issue) | YES |
| Linked issues fetched | PARTIAL (GH-1842 referenced but not fetched) |
| PR data referenced in STP | YES (PR #2436 / fork PR #63) |
| All STP sections present | YES |
| Template comparison possible | NO (no STP template in config or repo_rules) |
| Project review rules loaded | YES (dynamically extracted, ~45% defaults) |

**Confidence rationale:** MEDIUM. GitHub issue data provides full acceptance criteria and context for zero-trust verification. However, no STP template was available for structural comparison (Rule B checked against general conventions only), and review rules are operating at ~45% default ratio. The strong source data availability partially compensates — all requirement coverage and metadata checks were verified against actual issue data.

**Review precision note:** 45% of review rules are using generic defaults. Project-specific review precision is reduced. To improve: add a `review_rules.yaml` to `config/projects/fullsend/` or enable `repo_files_fetch` to fetch the STP template from the upstream repository.
