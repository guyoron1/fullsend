# FullSend Test Plan

## **Shim Drift Detection False-Positive Prevention - Quality Engineering Plan**

### Metadata & Tracking

- **Enhancement:** [GH-2247](https://github.com/fullsend-ai/fullsend/issues/2247)
- **Feature Tracking:** [GH-2247](https://github.com/fullsend-ai/fullsend/issues/2247) — reconcile-repos.sh produces shim blob without sentinel, creating bogus update PR
- **Epic Tracking:** N/A (standalone bug fix)
- **QE Owner:** TBD
- **Owning SIG:** N/A
- **Participating SIGs:** N/A

**Document Conventions:** Test scenario IDs follow the format TS-GH-2247-NNN.

### Feature Overview

The `reconcile-repos.sh` script manages shim workflow enrollment across repositories in the fullsend platform. A bug in the stale-shim detection logic caused false-positive drift detection: the comparison between remote and expected shim content operated on re-encoded base64 strings, which differed due to trailing newline handling in Bash command substitution — even when the decoded text was identical. This led to bogus update PRs (e.g., PR #2101) that would remove sentinel lines from the shim workflow. The fix (PR #2254) replaces the base64-to-base64 comparison with decoded text comparison, normalizing whitespace before diffing. This test plan covers verification of the corrected comparison logic, sentinel preservation, ORG interpolation consistency, and enrollment correctness.

---

### Section I — Motivation and Requirements Review

#### I.1 — Requirement & User Story Review Checklist

- [x] **Reviewed the relevant requirements.**
  - Issue GH-2247 clearly describes the false-positive drift detection bug, root cause analysis (base64 re-encoding differences from trailing newlines), and the expected behavior (no bogus PRs).
  - Triage agent provided updated root cause hypothesis confirmed by maintainer: pre-interpolated vs post-interpolated template comparison, not base64 encoding differences per se.

- [x] **Confirmed clear user stories and understood. Understand the value and customer use cases.**
  - The reconcile bot manages shim workflows for all enrolled repos. False-positive drift creates infinite PR churn — each merge triggers another "fix" PR. This directly impacts maintainer productivity and CI resources.

- [x] **Confirmed requirements are **testable and unambiguous**.**
  - The requirement is clear: identical shim content (modulo trailing whitespace) must not trigger an update PR. Genuinely changed content must still trigger updates. Both are directly testable via the existing bash test harness with mocked `gh` CLI.

- [x] **Ensured acceptance criteria are **defined clearly**.**
  - Acceptance criteria inferred from issue: (1) reconcile-repos.sh must not create a PR when remote shim matches expected content modulo trailing newlines, (2) PR #2101-style bogus PRs must not recur, (3) genuine drift must still be detected.

- [x] **Confirmed coverage for NFRs.**
  - No performance or scale NFRs. The comparison runs once per repo per reconciliation cycle. Correctness is the sole NFR.

#### I.2 — Known Limitations

- The fix compares decoded text after `tr -d '\r'` normalization. Content differences limited to carriage returns will not be detected as drift. This is acceptable because GitHub normalizes line endings.
- The test harness uses mocked `gh` CLI calls. Behavioral differences between the mock and real GitHub Content API (e.g., base64 line wrapping) could mask edge cases.
- Pre-sentinel shims (legacy repos enrolled before the sentinel was added) use a full-content fallback comparison that is less precise than sentinel-bounded comparison.

#### I.3 — Technology and Design Review

- [x] **Completed developer handoff or design review.**
  - PR #2254 reviewed. The fix modifies 10 lines in `reconcile-repos.sh` (lines 404-419) replacing `managed_content_b64` calls with decoded text comparison via `base64 -d | tr -d '\r'` and `extract_managed_content`.

- [x] **Identified technology challenges or constraints.**
  - Bash command substitution strips trailing newlines, making base64 round-trip comparisons unreliable. The fix avoids this by comparing decoded strings directly.

- [x] **Confirmed test environment needs are understood.**
  - Tests run in bash with mocked `gh`/`yq`/`base64` commands. No cluster or external services required.

- [x] **Reviewed API or interface extensions.**
  - No API changes. The `extract_managed_content` function (existing) is reused for text-mode extraction. No new interfaces added.

- [x] **Confirmed topology or deployment requirements.**
  - N/A. The reconcile script runs as a GitHub Actions workflow on ubuntu-latest runners. No special topology needed.

---

### Section II — Test Planning

#### II.1 — Scope of Testing

This test plan covers the shim drift detection comparison logic in `reconcile-repos.sh`, specifically the transition from base64-encoded comparison to decoded text comparison. Testing focuses on the comparison function behavior, sentinel preservation, ORG placeholder interpolation, and the enrollment/update PR creation paths.

**Testing Goals:**

- **P0:** Verify identical shim content is not falsely flagged as stale (regression prevention for GH-2247)
- **P0:** Verify genuinely stale shim content is correctly detected and triggers an update PR
- **P0:** Verify sentinel lines are preserved in all generated shim content
- **P1:** Verify `__ORG__` placeholder interpolation is consistent between expected and remote comparison paths
- **P1:** Verify pre-sentinel shim fallback comparison works correctly
- **P1:** Verify new enrollment produces correct shim content

**Out of Scope (Testing Scope Exclusions):**

- [ ] **GitHub Content API base64 encoding behavior** — Platform behavior owned by GitHub; we test our handling of its output, not the API itself.
- [ ] **GitHub Actions runner environment** — Runner OS, preinstalled tools, and network connectivity are platform concerns.
- [ ] **Branch protection rule enforcement** — GitHub-managed feature; we test PR creation, not merge policies.
- [ ] **yq YAML parsing correctness** — Third-party tool; we test our usage patterns via mocked responses.

#### II.2 — Test Strategy

**Functional:**

- [x] **Functional Testing** — Applicable
  - Validate the corrected comparison logic (decoded text vs. base64), sentinel preservation, ORG interpolation, and enrollment paths using the bash test harness with mocked GitHub API.
- [x] **Automation Testing** — Applicable
  - All tests are automated in `reconcile-repos-test.sh`. PR #2254 adds Test 5 (trailing newline regression test). Additional scenarios will extend this harness.
- [x] **Regression Testing** — Applicable
  - Test 5 in `reconcile-repos-test.sh` is a direct regression test for GH-2247. It verifies that logically identical content with different trailing newlines is not flagged as stale.

**Non-Functional:**

- [ ] **Performance Testing** — Not applicable
  - Comparison runs once per repo; no performance-sensitive paths.
- [ ] **Scale Testing** — Not applicable
  - Reconciliation processes repos sequentially; no scale concerns for the comparison logic.
- [ ] **Security Testing** — Not applicable
  - No new attack surface. Content injection guard is pre-existing and unchanged.
- [ ] **Usability Testing** — Not applicable
  - No user-facing interface changes; script output messages only.
- [ ] **Monitoring** — Not applicable
  - No new metrics or observability changes.

**Integration & Compatibility:**

- [ ] **Compatibility Testing** — Not applicable
  - No version-specific behavior; script uses POSIX-standard `base64 -d` and `tr`.
- [ ] **Upgrade Testing** — Not applicable
  - No stateful upgrade path for the reconcile script.
- [ ] **Dependencies** — Not applicable
  - No new dependencies introduced.
- [ ] **Cross Integrations** — Not applicable
  - Fix is self-contained within reconcile-repos.sh.

**Infrastructure:**

- [ ] **Cloud Testing** — Not applicable
  - Script runs on GitHub Actions runners; no cloud-specific testing needed.

#### II.3 — Test Environment

- **Cluster Topology:** N/A — no cluster required; tests run in bash shell
- **Platform Version:** GitHub Actions ubuntu-latest runner
- **CPU Virtualization:** N/A
- **Compute:** Standard GitHub Actions runner (2-core, 7 GB RAM)
- **Special Hardware:** None
- **Storage:** Ephemeral runner disk (default)
- **Network:** Mocked — all `gh` API calls intercepted by mock scripts
- **Operators:** N/A
- **Platform:** GitHub Actions with `gh` CLI pre-installed
- **Special Configs:** `GITHUB_REPOSITORY_OWNER` environment variable must be set; mock `gh`/`yq`/`base64` binaries prepended to PATH

#### II.3.1 — Testing Tools & Frameworks

No new or special tools required. Tests use the existing bash test harness (`reconcile-repos-test.sh`) with mock `gh` CLI scripts.

#### II.4 — Entry Criteria

- [x] PR #2254 merged or branch `agent/2247-fix-shim-stale-comparison` available for testing
- [x] `reconcile-repos-test.sh` executable and passing existing tests (Tests 1-4)
- [x] Mock `gh` CLI scripts correctly simulate GitHub Content API responses (base64-encoded content with `.content` field)
- [x] `GITHUB_REPOSITORY_OWNER` environment variable set to test org name

#### II.5 — Risks

- [ ] **Timeline**
  - *Risk:* Low — fix is already implemented in PR #2254 with regression test included.
  - *Mitigation:* Test plan validates existing test coverage and identifies gaps.
  - *Status:* Open

- [ ] **Coverage**
  - *Risk:* Medium — the existing regression test (Test 5) covers only the trailing newline case. Other comparison edge cases (empty content, ORG special characters, pre-sentinel fallback) are not yet covered.
  - *Mitigation:* This test plan identifies 25 scenarios; coverage gaps should be addressed by extending `reconcile-repos-test.sh`.
  - *Status:* Open

- [ ] **Environment**
  - *Risk:* Low — test environment is a simple bash shell with mocks. No complex infrastructure.
  - *Mitigation:* Mocks validated against real GitHub API response format.
  - *Status:* Closed

- [ ] **Untestable**
  - *Risk:* Low — the actual GitHub Content API base64 encoding behavior (line wrapping, padding) cannot be tested in mocks. The fix's `tr -d '\r\n'` normalization should handle known variations.
  - *Mitigation:* Manual verification against real GitHub API response for one enrolled repo.
  - *Status:* Open

- [ ] **Resources**
  - *Risk:* Low — no special resources required.
  - *Mitigation:* N/A.
  - *Status:* Closed

- [ ] **Dependencies**
  - *Risk:* Low — no external dependencies beyond `base64`, `tr`, and `gh` CLI.
  - *Mitigation:* Mock scripts cover all dependency interactions.
  - *Status:* Closed

- [ ] **Other**
  - *Risk:* The `extract_managed_content` function behavior is assumed to be correct. If it has edge cases (e.g., multiple sentinel markers), the comparison could silently fail.
  - *Mitigation:* Add test for shim with multiple sentinel markers (should extract content after the last sentinel).
  - *Status:* Open

---

### Section III — Requirements-to-Tests Mapping

#### III.1 — Test Scenarios

- **Requirement ID:** GH-2247
- **Requirement Summary:** Shim drift detection correctly identifies identical content as up-to-date (false-positive prevention)
- **Test Scenarios:**
  - TS-GH-2247-001: Verify identical shim not flagged as stale
  - TS-GH-2247-002: Verify identical content with trailing newline variation detected as current
  - TS-GH-2247-003: Verify no update PR created for up-to-date shim
  - TS-GH-2247-004: Verify false-positive detection after freshly enrolled repo re-run
- **Tier:** Functional
- **Priority:** P0

---

- **Requirement ID:** GH-2247
- **Requirement Summary:** Shim drift detection correctly identifies genuinely stale content and triggers update PR
- **Test Scenarios:**
  - TS-GH-2247-005: Verify stale shim triggers update PR creation
  - TS-GH-2247-006: Verify update PR contains correct updated shim content
  - TS-GH-2247-007: Verify drift detected when template changes between runs
- **Tier:** Functional
- **Priority:** P0

---

- **Requirement ID:** GH-2247
- **Requirement Summary:** Shim content generation preserves sentinel lines in all code paths
- **Test Scenarios:**
  - TS-GH-2247-008: Verify generated shim blob contains sentinel comment
  - TS-GH-2247-009: Verify sentinel preserved through base64 encode-decode round-trip
  - TS-GH-2247-010: Verify error when sentinel missing from template
- **Tier:** Functional
- **Priority:** P0

---

- **Requirement ID:** GH-2247
- **Requirement Summary:** Base64 encoding/decoding round-trip handles trailing newline variations
- **Test Scenarios:**
  - TS-GH-2247-011: Verify decoded comparison matches despite trailing newline differences
  - TS-GH-2247-012: Verify comparison resilient to carriage return in remote content
  - TS-GH-2247-013: Verify empty content handled gracefully
- **Tier:** Functional
- **Priority:** P1

---

- **Requirement ID:** GH-2247
- **Requirement Summary:** Pre-sentinel shim comparison falls back to full decoded content comparison
- **Test Scenarios:**
  - TS-GH-2247-014: Verify fallback to full content when no sentinel found
  - TS-GH-2247-015: Verify fallback detects genuine drift in pre-sentinel shim
  - TS-GH-2247-016: Verify fallback identifies identical pre-sentinel content as current
- **Tier:** Functional
- **Priority:** P1

---

- **Requirement ID:** GH-2247
- **Requirement Summary:** Reconcile script does not create PRs when no actual drift exists
- **Test Scenarios:**
  - TS-GH-2247-017: Verify no PR created for up-to-date enrolled repo
  - TS-GH-2247-018: Verify skip count incremented for up-to-date repos
  - TS-GH-2247-019: Verify no blob API call made for identical content
- **Tier:** Functional
- **Priority:** P0

---

- **Requirement ID:** GH-2247
- **Requirement Summary:** ORG placeholder interpolation is consistent between expected and remote paths
- **Test Scenarios:**
  - TS-GH-2247-020: Verify ORG substitution in expected matches deployed content
  - TS-GH-2247-021: Verify comparison consistent when ORG contains special characters
  - TS-GH-2247-022: Verify error when ORG environment variable is unset
- **Tier:** Functional
- **Priority:** P1

---

- **Requirement ID:** GH-2247
- **Requirement Summary:** Shim enrollment for new repos produces correct content with sentinel
- **Test Scenarios:**
  - TS-GH-2247-023: Verify new enrollment creates shim with sentinel and ORG interpolated
  - TS-GH-2247-024: Verify enrollment PR created with correct title and body
  - TS-GH-2247-025: Verify enrollment skipped for private repos
- **Tier:** Functional
- **Priority:** P1

---

### Section IV — Sign-off

| Role | Name | Date | Signature |
|:-----|:-----|:-----|:----------|
| QE Lead | TBD | | |
| Dev Lead | TBD | | |
| Product Owner | TBD | | |
