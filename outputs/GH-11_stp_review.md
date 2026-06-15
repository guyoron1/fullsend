# STP Review Report: GH-11

## Review Summary

| Field | Value |
|:------|:------|
| **Jira ID** | GH-11 |
| **STP File** | `outputs/stp/GH-11/GH-11_test_plan.md` |
| **Verdict** | **APPROVED_WITH_FINDINGS** |
| **Weighted Score** | 82.65 / 100 |
| **Confidence** | MEDIUM |
| **Review Date** | 2026-06-15 |
| **Source Data** | GitHub PR #11 (Jira not configured) |

### Confidence Note

Jira was not configured (`JIRA_BASE_URL` empty). Review was conducted against GitHub PR data (title, body, diff, changed files). Confidence is reduced from HIGH to MEDIUM because formal acceptance criteria, priority, and labels from the issue tracker were unavailable. All STP claims were verified against source code and PR diff.

> WARNING: 45% of review rules are using generic defaults. Project-specific review precision is slightly reduced. `review_rules.yaml` not found at config dir; `repo_files_fetch` is disabled. Keys using defaults: `abstraction.internal_to_user_mappings`, `dependencies.infrastructure_not_dependency`, `dependencies.dependency_examples`, `strategy.requires_justification_for_y`, `metadata.version_source`, `scope.dependent_product`.

---

## Dimension Scores

| # | Dimension | Weight | Score | Weighted |
|:--|:----------|:-------|:------|:---------|
| 1 | Rule Compliance (A-P) | 25% | 85 | 21.25 |
| 2 | Requirement Coverage | 30% | 75 | 22.50 |
| 3 | Scenario Quality | 15% | 78 | 11.70 |
| 4 | Risk & Limitation Accuracy | 10% | 90 | 9.00 |
| 5 | Scope Boundary Assessment | 10% | 92 | 9.20 |
| 6 | Test Strategy Appropriateness | 5% | 92 | 4.60 |
| 7 | Metadata Accuracy | 5% | 88 | 4.40 |
| | **Total** | **100%** | | **82.65** |

---

## Findings

### MAJOR-001: Missing Critical Test Scenario for Header Absence Verification

- **Severity:** MAJOR
- **Dimension:** Requirement Coverage / Scenario Quality
- **Rule:** P (Fix Scope)
- **Actionable:** true

**Description:**

The core behavioral change in this bug fix is that the `x-goog-user-project` quota header is **not sent** in the CRM API request. This is the entire purpose of the PR ("removed the need to enable `cloudresourcemanager` permissions"). However, no test scenario explicitly verifies that the `x-goog-user-project` header is absent from the HTTP request sent to the mock server.

The STP itself identifies this gap in the Risks section (II.5, Test Coverage): "Existing tests do not verify the quota project header is omitted from requests" with mitigation "Add test assertion that verifies the request headers sent to the mock server." However, this risk-identified gap is not reflected in Section III as an actual test scenario.

**Source verification:** PR diff shows `noQuotaClient.QuotaProject = ""` is the fix. The STP's Risk section acknowledges the gap but Section III does not close it.

**Remediation:**

Add a P0 test scenario to Section III under the first requirement group:

```markdown
- *Test Scenario:* Verify `x-goog-user-project` header is NOT present in CRM API request [Functional]
- *Priority:* P0
```

This scenario should use the httptest mock server to capture and assert on request headers. The mock server handler should verify `r.Header.Get("x-goog-user-project") == ""`.

---

### MAJOR-002: Acceptance Criteria Are Implicit Rather Than Explicit

- **Severity:** MAJOR
- **Dimension:** Requirement Coverage
- **Rule:** N (Requirements Review)
- **Actionable:** true

**Description:**

Section I.1, Acceptance Criteria states: "Implicit acceptance: `GetProjectNumber` must succeed without requiring `cloudresourcemanager` API enabled on the target project. The original client must not be mutated."

The acceptance criteria should be **explicitly** derived from source data, not labeled as "implicit." The PR description clearly states: "removed the need to enable `cloudresourcemanager` permissions on the GCP project." This IS the acceptance criterion and should be stated as such.

**Source verification:** PR body: "I found that doing this removed the need to enable `cloudresourcemanager` permissions on the GCP project."

**Remediation:**

Rewrite the Acceptance Criteria bullet to state explicit criteria derived from the PR:

```markdown
- [ ] **Acceptance Criteria**
  - Ensured acceptance criteria are **defined clearly**.
  - **AC-1:** `GetProjectNumber` must succeed without the `cloudresourcemanager` API enabled on the target GCP project (verified by omitting `x-goog-user-project` header).
  - **AC-2:** The original `gcp.Client` instance must not be mutated by the `GetProjectNumber` call.
  - **AC-3:** Error handling for forbidden and empty responses must remain functional.
```

---

### MINOR-001: Overlapping Test Scenarios Across Requirement Groups

- **Severity:** MINOR
- **Dimension:** Scenario Quality
- **Rule:** O (Test Scenarios)
- **Actionable:** true

**Description:**

The "forbidden response" error scenario appears in two separate requirement groups in Section III:

1. Under "GCP project number lookup succeeds without `cloudresourcemanager` API enabled": "Verify error when CRM API returns forbidden [Functional]" (P1)
2. Under "Project number lookup returns correct value and handles errors": "Verify appropriate error with status code for forbidden response [Functional]" (P1)

These describe the same test scenario with slightly different wording, inflating the test count and creating ambiguity about which scenario drives implementation.

**Remediation:**

Consolidate the forbidden-response scenario into a single requirement group (the error-handling group is the natural home). Remove the duplicate from the first group, or explicitly note they test different aspects (e.g., one tests the quota-header-absent path specifically, the other tests the error parsing).

---

### MINOR-002: Dependencies Section Conflates Code Prerequisites with External Dependencies

- **Severity:** MINOR
- **Dimension:** Rule Compliance
- **Rule:** D (Dependencies)
- **Actionable:** true

**Description:**

Section II.2 (Dependencies) states: "Depends on `gcp.Client` struct having a `QuotaProject` field (currently missing — see Known Limitations)."

Per Rule D, the Dependencies checkbox is for deliverables blocked by **other teams or external components**. The missing `QuotaProject` field is a code change needed within the same repository (and the same PR, since it's mirrored from upstream). This is more accurately an **Entry Criterion** (which it already is in Section II.4) or a **Known Limitation** (which it already is in Section I.2).

**Remediation:**

Change the Dependencies *Details* to: "No external team dependencies. The `QuotaProject` field prerequisite is tracked as an Entry Criterion (II.4) and Known Limitation (I.2)." Or mark Dependencies as unchecked with "N/A — no external team dependencies."

---

### MINOR-003: Feature Overview Uses Internal Component Names

- **Severity:** MINOR
- **Dimension:** Rule Compliance
- **Rule:** A (Abstraction)
- **Actionable:** true

**Description:**

The Feature Overview mentions `gcp.Client`, `GetProjectNumber`, and "GCF dispatch provisioner" — these are internal implementation details. While acceptable for a narrow bug-fix STP (and they appear in appropriate locations like Technology Challenges and Known Limitations), the Feature Overview is a user/stakeholder-facing section that should describe the change in terms of its user impact.

**Remediation:**

Consider rephrasing the Feature Overview opening to lead with the user impact:

```markdown
This bug fix simplifies GCP project permissions for FullSend deployments by removing the
requirement for the Cloud Resource Manager API to be enabled on the target GCP project.
Internally, the `GetProjectNumber` method is modified to omit the `x-goog-user-project`
quota header when calling the GCP Cloud Resource Manager API.
```

This preserves the technical detail while leading with the value proposition.

---

## Rule Compliance Detail (Rules A-P)

| Rule | Description | Status | Notes |
|:-----|:------------|:-------|:------|
| A | Abstraction | MINOR | Internal names in Feature Overview; acceptable in Tech Review & Known Limitations |
| B | Scenario Naming | PASS | Clear, descriptive scenario names |
| C | N/A Justification | PASS | All N/A items justified |
| D | Dependencies | MINOR | Conflates code prerequisite with external dependency |
| E | Upgrade Testing | PASS | Appropriately N/A for bug fix |
| F | Entry Criteria | PASS | Includes compilation prerequisite — good catch |
| G | Risks | PASS | Well-identified with mitigations |
| H | Testing Tools | PASS | Standard Go testing + httptest matches project config |
| I | Test Environment | PASS | Appropriate for unit-test scope |
| J | Strategy Checkboxes | PASS | Functional [x], Automation [x], Regression [x]; N/F items justified |
| K | Scope | PASS | In/out of scope well-defined with rationale |
| L | Metadata | PASS | TBD for QE Owner acceptable for initial STP |
| M | Feature Overview | PASS | Accurate, matches PR diff |
| N | Requirements Review | MAJOR | Acceptance criteria labeled "implicit" |
| O | Test Scenarios | MINOR | Overlapping scenarios across groups |
| P | Fix Scope (Bug) | MAJOR | Missing core behavioral verification scenario |

## Source Data Verification

The following STP claims were verified against source code and PR data:

| STP Claim | Verified? | Evidence |
|:----------|:----------|:---------|
| `gcp.Client` has no `QuotaProject` field | **YES** | `internal/gcp/client.go:20-25` — struct has only `httpClient` and `tokenFunc` |
| `LiveGCFClient` embeds `*gcp.Client` | **YES** | `internal/dispatch/gcf/gcp.go:95-98` |
| PR creates value copy of `*c.Client` | **YES** | PR diff line: `noQuotaClient := *c.Client` |
| `GetProjectNumber` called by `provisionSelfManaged` | **YES** | `provisioner.go:310` |
| `provisionSelfManaged` is at line 310 | **YES** | Confirmed via grep |
| 3 existing test cases in `gcp_test.go` | **YES** | `TestLiveGCFClient_GetProjectNumber` at line 700 with success, empty, error subtests |
| `GCFClient` interface includes `GetProjectNumber` | **YES** | `gcp.go:90` |
| Code will not compile as written | **YES** | `gcp.Client` struct lacks `QuotaProject` field; line 893 references `noQuotaClient.QuotaProject` |
| PR is mirrored from upstream PR #2231 | **YES** | PR body: "Mirrored from upstream PR #2231" |
| Single file modified | **YES** | PR changed files: `internal/dispatch/gcf/gcp.go` (5 additions, 1 deletion) |

**All STP claims verified.** No factual inaccuracies found. The STP demonstrates excellent source-code awareness, particularly in identifying the compilation issue that is not visible from the PR diff alone.

## Strengths

1. **Excellent compilation issue detection** — The STP correctly identified that `gcp.Client` lacks the `QuotaProject` field, which is a genuine blocking issue. This finding is confirmed by source code inspection and adds significant QE value.

2. **Well-structured risks** — All risks have corresponding mitigations. The upstream sync risk and shared pointer risk are technically accurate and demonstrate strong engineering understanding.

3. **Appropriate scope boundaries** — Out-of-scope items (CRM API availability, OAuth2/ADC, WIF provisioning) are correctly identified with clear rationale aligned to the project's scope validation gate.

4. **Correct test strategy** — The strategy appropriately marks Functional, Automation, and Regression as in-scope while justifying N/A for non-functional categories given the narrow bug-fix scope.

5. **Good entry criteria** — Including the compilation prerequisite as an entry criterion prevents premature test execution.

---

*Review generated by QualityFlow STP Reviewer*
*Reviewed: `outputs/stp/GH-11/GH-11_test_plan.md`*
*Source: GitHub PR #11 (`fix(gcp): remove the project from the number call`)*
