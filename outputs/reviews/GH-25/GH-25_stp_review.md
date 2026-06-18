# STP Review Report: GH-25

**Reviewed:** outputs/stp/GH-25/GH-25_test_plan.md
**Date:** 2026-06-18
**Reviewer:** QualityFlow Automated Review (v1.1.0)
**Review Rules Schema:** 1.1.0 (dynamic extraction, no static override)

---

## Verdict: NEEDS_REVISION

## Summary

| Metric | Value |
|:-------|:------|
| Dimensions reviewed | 7/7 |
| Critical findings | 2 |
| Major findings | 5 |
| Minor findings | 3 |
| Actionable findings | 9 |
| Confidence | MEDIUM |
| Weighted score | 67 |

## Dimension Scores

| Dimension | Weight | Pass Rate | Weighted |
|:----------|:-------|:----------|:---------|
| 1. Rule Compliance | 25% | 70% | 17.5 |
| 2. Requirement Coverage | 30% | 60% | 18.0 |
| 3. Scenario Quality | 15% | 75% | 11.3 |
| 4. Risk & Limitation Accuracy | 10% | 85% | 8.5 |
| 5. Scope Boundary Assessment | 10% | 60% | 6.0 |
| 6. Test Strategy Appropriateness | 5% | 40% | 2.0 |
| 7. Metadata Accuracy | 5% | 65% | 3.3 |
| **Total** | **100%** | | **66.6** |

---

## Findings by Dimension

### Dimension 1: Rule Compliance (Rules A-P)

| Rule | Status | Finding |
|:-----|:-------|:--------|
| A — Abstraction Level | WARN | Scenarios use Go-level identifiers (`forge.Client`, `FakeClient`, `LiveClient.do()`, `retryOnTransient`) throughout. While FullSend is developer-facing, the STP should describe *what* is tested at a capability level, not name internal types. See D1-A-001. |
| A.2 — Language Precision | PASS | Language is precise and measurable. No anthropomorphization or vague qualifiers found. |
| B — Section I Meta-Checklist | WARN | STP does not include a Section I meta-checklist (Requirements Review, Technology Review checkboxes with sub-items). The document uses a flat structure starting with Summary. No template was available for comparison. See D1-B-001. |
| C — Prerequisites vs Scenarios | PASS | All 30 scenarios describe testable behaviors, not configuration prerequisites. |
| D — Dependencies | PASS | N/A — Feature has no cross-team delivery dependencies. No dependencies section needed. |
| E — Upgrade Testing | PASS | Feature does not create persistent state. Upgrade testing is correctly omitted. |
| F — Version Derivation | PASS | Version "0.x" matches `project_context.versioning.current_version`. |
| G — Testing Tools | WARN | Test Environment (Section 5) lists standard project tools (`testing`, `testify`, `httptest`) that are baseline for all FullSend Go tests. See D1-G-001. |
| G.2 — Environment Specificity | PASS | Environment entries are feature-specific (HTTP mocking for LiveClient, FakeClient for unit tests). |
| H — Risk Deduplication | PASS | All 5 risks describe genuine uncertainties (truncated trees, empty repos, rate limiting). No duplication with environment requirements. |
| I — QE Kickoff Timing | PASS | N/A — No kickoff section; acceptable for this project type. |
| J — One Tier Per Row | PASS | All 30 scenarios specify exactly one tier (Tier1). No multi-tier rows. |
| K — Cross-Section Consistency | FAIL | **CRITICAL:** Scope lists 3 components (config.OrgConfig, cli/run.go + reconcilestatus.go, statuscomment) that have ZERO corresponding test scenarios in Section 3. See D1-K-001. |
| L — Section Content Validation | WARN | STP omits expected sections: Section I meta-checklist, Test Strategy (II.2) with checkboxes, Entry/Exit Criteria (II.4). Content is distributed in a non-standard structure. See D1-L-001. |
| M — Deletion Test | PASS | Content is concise and decision-relevant. Regression Analysis (Section 4) is detailed but provides valuable dependency-chain context for test design. |
| N — Link/Reference Validation | PASS | References to PR #25 and fullsend-ai/fullsend#2360 are consistent. Source code line references (e.g., `github.go:1020-1022`) are valid context pointers. |
| O — Untestable Aspects | PASS | No items are marked as untestable. All scenarios are actionable. |
| P — Testing Pyramid Efficiency | PASS | N/A — Issue type is performance improvement, not a bug/defect. Rule P skipped. |

### Dimension 2: Requirement Coverage

| Metric | Value |
|:-------|:------|
| Acceptance criteria covered | N/A (no formal AC in GitHub issue) |
| PR scope items with scenarios | 6/9 (67%) |
| Linked issues reflected | N/A |
| Negative scenarios present | YES (8 negative scenarios) |
| Coverage gaps found | 3 scope items |

**Coverage Assessment:**

The GitHub issue describes 4 core changes (ListRepositoryFiles, LiveClient implementation, pathpresence refactoring, test coverage). The actual PR diff reveals 64 changed files across 9+ packages. The STP correctly identifies the broader scope but fails to provide test scenarios for 3 of the 9 in-scope items.

**Gaps identified:**

| In-Scope Item | PR Changes | STP Scenarios | Status |
|:--------------|:-----------|:-------------|:-------|
| `forge.Client.ListRepositoryFiles` | +6 lines (interface) | 5 scenarios (3.1) | ✅ Covered |
| `github.LiveClient.ListRepositoryFiles` | +78 lines | 6 scenarios (3.2) | ✅ Covered |
| `forge.FakeClient.ListRepositoryFiles` | +18 lines | 4 scenarios (3.3) | ✅ Covered |
| `scaffold.ComparePathPresence` | +37 lines | 6 scenarios (3.4) | ✅ Covered |
| `harness.DiscoverRemoteAgents` | +76 lines | 6 scenarios (3.5) | ✅ Covered |
| `harness.Lint` | +52 lines | 3 scenarios (3.6) | ✅ Covered |
| `config.OrgConfig` (MintURL, dispatch) | +59 lines, +199 test lines | 0 scenarios | ❌ **MISSING** |
| `cli/run.go` + `cli/reconcilestatus.go` | +90 lines, +310 test lines | 0 scenarios | ❌ **MISSING** |
| `statuscomment` (expanded management) | +48 lines, +212 test lines | 0 scenarios | ❌ **MISSING** |

**Negative scenario distribution:** 8 of 30 scenarios are negative/error cases (27%), which is a healthy ratio. Covers: nonexistent repos, empty repos, truncated trees, API errors, injected errors, parse failures, empty roles.

### Dimension 3: Scenario Quality

| Metric | Value |
|:-------|:------|
| Total scenarios | 30 |
| Tier 1 | 30 |
| Tier 2 | 0 |
| P0 | 0 (not assigned) |
| P1 | 0 (not assigned) |
| P2 | 0 (not assigned) |
| Positive scenarios | 22 |
| Negative scenarios | 8 |

**Scenario-level findings:**

Scenario quality is generally strong — each scenario is specific, describes a distinct behavior, has clear expected results, and uses consistent ID formatting (`TS-GH-25-{NNN}`). The main gaps are structural:

1. **No priority assignments** (D3-PRI-001): None of the 30 scenarios have P0/P1/P2 priority. The primary positive scenarios for the feature's core capability (TS-GH-25-001, TS-GH-25-006, TS-GH-25-016) should be P0. Error-handling and edge-case scenarios should be P1/P2.

2. **All Tier 1** (D3-TIER-001): All 30 scenarios are Tier 1. This is reasonable for unit tests using mocks/fakes, but the STP should acknowledge that no integration-tier scenarios exist and justify why (e.g., "integration testing covered by upstream CI").

3. **Scenario specificity is good:** Examples like "Tree response is truncated (very large repo) → Returns error containing 'truncated'" and "All expected paths missing → Returns all paths sorted" are well-specified with measurable outcomes.

### Dimension 4: Risk & Limitation Accuracy

5 risks identified in Section 1.2, all relevant and well-structured:

| Risk | Assessment |
|:-----|:-----------|
| Truncated tree response | ✅ Real uncertainty, medium likelihood, test scenario TS-GH-25-004 covers it |
| Empty repository | ✅ Real edge case, covered by TS-GH-25-002 |
| API rate limiting | ✅ Real concern, mitigation references existing retry logic |
| FakeClient divergence | ✅ Valid testing risk, contract tests address it |
| ComparePathPresence regression | ✅ Valid concern, 6 test cases + TS-GH-25-021 |

No limitations section is present, but no obvious feature limitations are documented in the source issue either. The risks have actionable mitigations and traceable test coverage.

**One gap:** The PR introduces significant config and CLI changes (OrgConfig, dispatch mode, MintURL field) that carry their own risks (backward compatibility, config migration). These risks are not documented since the corresponding scope items lack scenarios.

### Dimension 5: Scope Boundary Assessment

**Scope alignment with project boundaries:**
All in-scope components (forge, scaffold, harness, config, cli, statuscomment) map to FullSend's `in_scope_resources` (Agent, Sandbox, Harness, Skill, Scaffold, Forge, Mint). No out-of-boundary components are tested. ✅

**Scope alignment with source data:**
The GitHub issue summary describes only the core forge/scaffold changes (4 items), but the actual PR diff contains 64 files across 9+ packages. The STP correctly broadens scope to match the PR diff, not just the issue summary. However, this creates a **scope inflation** pattern where 3 items are listed in scope but never tested. See D5-SCOPE-001.

**Out-of-scope assessment:**
Out-of-scope items are well-justified:
- Upstream PR (tested separately) ✅
- Workflow YAML changes (infrastructure) ✅
- Documentation-only files ✅
- Scaffold template files (static content) ✅
- External dependencies ✅

### Dimension 6: Test Strategy Appropriateness

The STP has a "Test Execution Strategy" section (Section 6) but **no Test Strategy section** with classification checkboxes (Functional, Automation, Performance, Security, Upgrade, etc.).

Section 6 describes *how* to run tests (`go test ./internal/forge/... ./internal/scaffold/... ./internal/harness/...`) and pass criteria, but does not classify *what types* of testing apply to this feature.

| Expected Strategy Item | Applicable? | STP Status |
|:-----------------------|:------------|:-----------|
| Functional Testing | Yes (always) | Not classified |
| Automation Testing | Yes (always) | Implied but not explicit |
| Performance Testing | Possibly (perf optimization feature) | Not addressed |
| Security Testing | No | Not addressed |
| Upgrade Testing | No (no persistent state) | Not addressed |
| Regression Testing | Yes (refactoring ComparePathPresence) | Not addressed |

**Notable gap:** This is a **performance optimization** PR (title: `perf(#2351)`). The STP does not address whether the performance improvement should be verified (e.g., measuring API call count reduction from 100+ to 3). Scenario TS-GH-25-005 tests "API call count is exactly 3" which partially covers this, but the strategy-level classification is absent.

### Dimension 7: Metadata Accuracy

| Field | STP Value | Source Value | Status |
|:------|:----------|:-------------|:-------|
| Ticket | GH-25 | GH-25 | ✅ Match |
| Title | perf(#2351): batch path-existence checks via Git Trees API | Same | ✅ Match |
| Version | 0.x | project.yaml: 0.x | ✅ Match |
| Product | FullSend | project.yaml: FullSend | ✅ Match |
| Platform | GitHub Actions | project.yaml: GitHub Actions | ✅ Match |
| Status | Draft | Expected for new STP | ✅ Correct |
| Date | 2026-06-18 | Current date | ✅ Correct |
| Author | QualityFlow | Expected | ✅ Correct |
| SIG / Ownership | Not present | Not in issue labels | ⚠️ Missing field |
| Enhancement Links | Not present | N/A | ⚠️ Missing field |

---

## Findings Detail

### Critical Findings

#### D1-K-001: Scope-Scenario Coverage Gap

- **finding_id:** D1-K-001
- **severity:** CRITICAL
- **dimension:** Rule Compliance
- **rule:** K — Cross-Section Consistency
- **description:** Three in-scope items listed in Section 1.1 have zero corresponding test scenarios in Section 3. The STP scope includes `config.OrgConfig changes (new MintURL field, dispatch mode)`, `cli/run.go and cli/reconcilestatus.go — updated status/dispatch logic`, and `statuscomment — expanded status comment management`, but none of these appear in the test scenario tables.
- **evidence:** Section 1.1 Scope lists 9 in-scope items. Section 3 contains 6 subsections (3.1–3.6) covering only the first 6 scope items. PR data confirms significant code changes: config.go (+59/+199 test lines), run.go (+42/+207 test lines), reconcilestatus.go (+48/+103 test lines), statuscomment.go (+48/+212 test lines).
- **remediation:** Either (a) add test scenario subsections 3.7, 3.8, 3.9 covering OrgConfig, CLI dispatch/status, and statuscomment scenarios respectively, OR (b) move these items to Out of Scope with justification if they are covered by existing upstream tests.
- **actionable:** true

#### D2-COV-001: Below-Threshold Scope Coverage Rate

- **finding_id:** D2-COV-001
- **severity:** CRITICAL
- **dimension:** Requirement Coverage
- **rule:** Coverage Threshold
- **description:** Only 6 of 9 in-scope items (67%) have corresponding test scenarios. This is below the 70% minimum coverage threshold. The missing items represent 255 lines of production code changes and 522 lines of test code in the PR.
- **evidence:** Scope item count: 9. Scenario coverage: forge (3 items covered), scaffold (1 covered), harness (2 covered), config (0 covered), cli (0 covered), statuscomment (0 covered). Coverage rate: 67%.
- **remediation:** Add test scenarios for the 3 uncovered scope items to bring coverage above 70%, or remove them from scope with justification.
- **actionable:** true

### Major Findings

#### D1-A-001: Internal Go Identifiers in Scenario Descriptions

- **finding_id:** D1-A-001
- **severity:** MAJOR
- **dimension:** Rule Compliance
- **rule:** A — Abstraction Level
- **description:** Test scenarios use Go-level type and method names (`forge.Client`, `FakeClient`, `LiveClient`, `retryOnTransient`, `LiveClient.do()`, `forge.ErrNotFound`) as scenario descriptions. While FullSend is a developer tool, the STP should describe capabilities being tested, not internal type hierarchies.
- **evidence:** Section 3.1 header: "forge.Client.ListRepositoryFiles — Interface Contract". Scenario TS-GH-25-011: "Rate limit (429) during tree fetch → Retry logic in do() handles it transparently". TS-GH-25-012: "FakeClient with populated FileContents returns matching paths".
- **remediation:** Rewrite scenario descriptions at capability level. Example: "List files in repo with rate limiting → Operation succeeds after transient failure" instead of "Rate limit (429) during tree fetch → Retry logic in do() handles it transparently". Keep Go identifiers in a "Component" or "Code Reference" column for traceability, not in the scenario description.
- **actionable:** true

#### D1-B-001: Missing Template Structure (Section I Meta-Checklist)

- **finding_id:** D1-B-001
- **severity:** MAJOR
- **dimension:** Rule Compliance
- **rule:** B — Section I Meta-Checklist
- **description:** The STP does not include the expected Section I meta-checklist with Requirements Review and Technology Review checkboxes. It also omits Test Strategy classification checkboxes (II.2), Entry/Exit Criteria (II.4), and structured Risk checkboxes (II.5). The document uses a simplified flat structure.
- **evidence:** STP sections: 1. Summary, 2. Requirements Mapping, 3. Test Scenarios, 4. Regression Analysis, 5. Test Environment, 6. Test Execution Strategy, 7. Test Counts, 8. Approval. No Section I/II structure with checkboxes.
- **remediation:** Restructure the STP to include: (I.1) Requirements Review checklist with sub-items, (I.2) Known Limitations, (I.3) Technology Review checklist, (II.1) Scope/Out of Scope/Goals, (II.2) Test Strategy checkboxes, (II.3) Test Environment, (II.4) Entry/Exit Criteria, (II.5) Risks with checkboxes, (III) Requirements-to-Tests Mapping.
- **actionable:** true

#### D3-PRI-001: No Priority Assignments on Scenarios

- **finding_id:** D3-PRI-001
- **severity:** MAJOR
- **dimension:** Scenario Quality
- **rule:** Priority Validation
- **description:** None of the 30 test scenarios have P0/P1/P2 priority assignments. Without priorities, the team cannot triage which scenarios to execute first in time-constrained situations.
- **evidence:** All 6 scenario tables (Sections 3.1–3.6) have columns: ID, Scenario, Tier, Requirement, Expected Result. No Priority column exists.
- **remediation:** Add a Priority column to each scenario table. Assign P0 to core happy-path scenarios (TS-GH-25-001, TS-GH-25-006, TS-GH-25-016), P1 to error handling and contract verification scenarios, P2 to edge cases and supplementary coverage.
- **actionable:** true

#### D6-STR-001: Missing Test Strategy Classifications

- **finding_id:** D6-STR-001
- **severity:** MAJOR
- **dimension:** Test Strategy Appropriateness
- **rule:** Strategy Classification
- **description:** The STP has no Test Strategy section with classification checkboxes. The "Test Execution Strategy" (Section 6) describes how to run tests but does not classify what types of testing apply. Notably, this is a performance optimization PR (`perf(#2351)`) but performance verification is not explicitly classified in the strategy.
- **evidence:** Section 6 contains only execution commands and pass criteria. No Functional/Automation/Performance/Security/Upgrade/Regression classification exists.
- **remediation:** Add a Test Strategy section (II.2) with checkboxes: Functional Testing (Y — core functionality), Automation Testing (Y — all scenarios automated), Performance Testing (Y — verify API call reduction from 100+ to 3), Regression Testing (Y — ComparePathPresence refactored), Security Testing (N/A), Upgrade Testing (N/A — no persistent state), with feature-specific sub-items for each.
- **actionable:** true

#### D5-SCOPE-001: Scope Inflation Without Coverage

- **finding_id:** D5-SCOPE-001
- **severity:** MAJOR
- **dimension:** Scope Boundary Assessment
- **rule:** Scope-Scenario Alignment
- **description:** The STP scope was correctly broadened beyond the issue summary to match the actual PR diff, but 3 of the broadened items were added to scope without corresponding test scenarios. This creates a scope promise that the STP does not fulfill.
- **evidence:** In-scope items config.OrgConfig, cli/run.go + reconcilestatus.go, and statuscomment are listed in Section 1.1 but have no subsections in Section 3. The PR's generated QF tests include `org_config_test.go` and `mint_url_migration_test.go`, confirming these components warrant test planning.
- **remediation:** For each uncovered scope item, either: (a) add a scenario subsection in Section 3 with at least 3 scenarios covering happy path, error, and edge cases, or (b) move to Out of Scope with explicit justification (e.g., "Covered by existing unit tests in the PR").
- **actionable:** true

### Minor Findings

#### D1-G-001: Standard Tools Listed in Test Environment

- **finding_id:** D1-G-001
- **severity:** MINOR
- **dimension:** Rule Compliance
- **rule:** G — Testing Tools
- **description:** Test Environment (Section 5) lists standard project tools (`testing`, `testify`, `httptest`, `FakeClient`) that are baseline for all FullSend Go tests. Per Rule G, standard tools should not be listed unless feature-specific.
- **evidence:** Section 5 table lists: "Go 1.22+", "testing + testify", "net/http/httptest", "forge.FakeClient", "GitHub Actions", "go test ./..."
- **remediation:** Remove standard tools from the table or add a note that only non-standard tools should be listed. If no non-standard tools are needed, state "Standard project test infrastructure (no additional tools required)."
- **actionable:** true

#### D3-TIER-001: No Tier Distribution

- **finding_id:** D3-TIER-001
- **severity:** MINOR
- **dimension:** Scenario Quality
- **rule:** Tier Distribution
- **description:** All 30 scenarios are Tier 1. While this is reasonable for unit tests using mocks/fakes, the STP should explicitly acknowledge the absence of integration-tier scenarios and justify it.
- **evidence:** All scenario tables show Tier column = "Tier1". Section 6.2 mentions "Integration testing would require a test repository..." and states "covered by upstream repo's CI and out of scope." This justification exists but is in the wrong section.
- **remediation:** Move the integration testing justification from Section 6.2 into the Out of Scope section (or a Test Strategy section), making it a deliberate scope decision rather than an afterthought.
- **actionable:** true

#### D7-META-001: Missing Optional Metadata Fields

- **finding_id:** D7-META-001
- **severity:** MINOR
- **dimension:** Metadata Accuracy
- **rule:** Metadata Completeness
- **description:** The metadata table omits SIG/ownership and enhancement link fields. While these are optional for the FullSend project (no SIG structure, no enhancement proposals), the omission should be explicit.
- **evidence:** Metadata table has 8 fields (Ticket, Title, Author, Date, Version, Product, Platform, Status). No SIG or Enhancement rows.
- **remediation:** Add "Owning Team: N/A" and "Enhancement: N/A" rows to the metadata table for completeness, or note that these fields are not applicable to this project.
- **actionable:** true

---

## Recommendations

1. **[CRITICAL]** Add test scenarios for OrgConfig, CLI dispatch/status, and statuscomment to cover the 3 in-scope items currently without scenarios, bringing scope coverage above 70%. — **Remediation:** Create Sections 3.7 (OrgConfig — MintURL field, dispatch mode), 3.8 (CLI status/dispatch — run.go, reconcilestatus.go), 3.9 (Status Comment management) with 3-5 scenarios each. — **Actionable:** yes

2. **[CRITICAL]** Resolve the 67% scope-to-scenario coverage rate by either adding missing scenarios or moving uncovered items to Out of Scope with justification. — **Remediation:** See D1-K-001 and D2-COV-001 above. — **Actionable:** yes

3. **[MAJOR]** Restructure the STP to follow the expected template format with Section I meta-checklist, Section II test strategy, and checkbox-based classifications. — **Remediation:** See D1-B-001. — **Actionable:** yes

4. **[MAJOR]** Rewrite scenario descriptions at capability level, removing internal Go type names from scenario text. — **Remediation:** See D1-A-001. — **Actionable:** yes

5. **[MAJOR]** Add P0/P1/P2 priority assignments to all 30 scenarios. — **Remediation:** See D3-PRI-001. — **Actionable:** yes

6. **[MAJOR]** Add a Test Strategy section with classification checkboxes, explicitly addressing Performance Testing given this is a perf optimization PR. — **Remediation:** See D6-STR-001. — **Actionable:** yes

7. **[MAJOR]** Ensure scope item coverage is complete — each scope item must have at least one scenario or be explicitly excluded. — **Remediation:** See D5-SCOPE-001. — **Actionable:** yes

8. **[MINOR]** Remove standard tools from Test Environment or note they are project defaults. — **Remediation:** See D1-G-001. — **Actionable:** yes

9. **[MINOR]** Move integration testing justification to a scope/strategy section. — **Remediation:** See D3-TIER-001. — **Actionable:** yes

10. **[MINOR]** Add missing optional metadata fields (Owning Team, Enhancement) as N/A. — **Remediation:** See D7-META-001. — **Actionable:** yes

---

## Confidence Notes

| Factor | Status |
|:-------|:-------|
| Jira source data available | NO (GitHub issue used as proxy) |
| Linked issues fetched | NO |
| PR data referenced in STP | YES (64 files, +5498/-185 lines analyzed) |
| All STP sections present | NO (missing Section I meta-checklist, II.2 strategy, II.4 criteria) |
| Template comparison possible | NO (no STP template file found in project config) |
| Project review rules loaded | YES (dynamically extracted from config files) |

**Confidence rationale:** MEDIUM. GitHub issue data was available as a proxy for Jira, providing the source-of-truth for scope and requirement comparison. Full PR diff data (64 files) enabled comprehensive scope verification. However, confidence is reduced because: (1) no formal Jira acceptance criteria exist for precise coverage measurement, (2) no STP template was available for structural comparison, and (3) review rules were dynamically extracted with moderate default ratio (~0.45). The scope-scenario coverage gap was verified against actual PR file changes, giving high confidence in the critical findings.
