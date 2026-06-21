# Test Plan

## **[reconcile-repos.sh produces shim blob without sentinel, creating bogus update PR] - Quality Engineering Plan**

### Metadata & Tracking

- **Enhancement:** [GH-2247](https://github.com/fullsend-ai/fullsend/issues/2247)
- **Feature Tracking:** [GH-2247](https://github.com/fullsend-ai/fullsend/issues/2247)
- **Epic Tracking:** N/A
- **QE Owner:** TBD
- **Owning SIG:** N/A
- **Participating SIGs:** N/A

**Document Conventions:** Priority levels follow P0 (critical) > P1 (important) > P2 (edge case). Test types are classified as Unit Tests (mocked, no cluster), Functional (single feature with real or mocked integrations), or End-to-End (multi-feature workflows).

### Feature Overview

The `reconcile-repos.sh` script manages shim workflow enrollment across GitHub repositories. A bug in the shim drift detection logic caused false-positive staleness detection when logically identical content was encoded with different trailing newlines (e.g., from the GitHub content API). This produced bogus update PRs (such as PR #2101) that removed the sentinel line `# --- fullsend managed below - do not edit ---`, risking infinite reconciliation churn. The fix replaces base64-level comparison (`managed_content_b64`) with decoded text comparison via `extract_managed_content`, normalizing encoding differences before comparison.

---

### I. Motivation and Requirements Review

#### I.1 - Requirement & User Story Review Checklist

- [x] **Reviewed the relevant requirements.**
  - GH-2247 describes the root cause: `managed_content_b64()` re-encodes decoded content to base64 for comparison, but trailing newline differences between the template output and GitHub API response produce different base64 strings for identical text.
  - PR #2101 is the concrete symptom: a bogus PR removing the sentinel and YAML document separator.

- [x] **Confirmed clear user stories and understood. Understand the value and customer use cases.**
  - As a repo maintainer, I expect the reconcile bot to only create update PRs when the shim workflow has genuinely drifted from the template, not due to encoding artifacts.
  - Preventing infinite churn (PR removes sentinel -> next run detects missing sentinel -> opens another PR) is the core value.

- [x] **Confirmed requirements are **testable and unambiguous**.**
  - The fix is deterministic: compare decoded text instead of base64 strings. Testable by constructing inputs with varying trailing newlines and verifying comparison outcomes.

- [x] **Ensured acceptance criteria are **defined clearly**.**
  - Identical content with different trailing newlines must not be flagged as stale.
  - Genuinely different content must still be flagged as stale.
  - Sentinel line must be present in all generated shim blobs.

- [x] **Confirmed coverage for NFRs.**
  - No performance, scalability, or security NFRs identified. The fix is a comparison logic change with no runtime cost difference.

#### I.2 - Known Limitations

- The fix normalizes `\r` (carriage returns) via `tr -d '\r'` but does not normalize other whitespace differences (e.g., trailing spaces on individual lines). This is acceptable because the GitHub content API does not introduce such differences.
- The `extract_managed_content` function relies on exact string matching of the sentinel line. If the sentinel text is ever changed in the template without updating the `SENTINEL` variable, comparison will silently fall through to the full-content fallback.
- The existing test harness (`reconcile-repos-test.sh`) uses mock `gh` CLI commands. It does not test against real GitHub API responses, so encoding quirks specific to certain GitHub API versions are not covered.

#### I.3 - Technology and Design Review

- [x] **Developer handoff completed. Reviewed design and implementation approach.**
  - Fix is in `reconcile-repos.sh` lines 404-416. Replaces `managed_content_b64()` calls with inline decoded-text comparison using `base64 -d | tr -d '\r'` and `extract_managed_content`.
  - LSP analysis confirmed the Go-side scaffold code (`scaffold.go`, `enrollment.go`, `workflows.go`) is separate from the bash reconciliation path. The Go code uses `PrependManagedHeader` for initial scaffold installation, while `reconcile-repos.sh` handles ongoing drift detection.

- [x] **Identified technology challenges or constraints.**
  - Bash base64 encoding behavior varies across platforms (`base64 -w0` is GNU-specific). The script runs exclusively on GitHub Actions Ubuntu runners where GNU coreutils is standard.

- [x] **Assessed test environment needs.**
  - No cluster or special infrastructure required. All tests run in a mocked bash environment with stubbed `gh`, `yq`, and `base64` commands.

- [x] **Reviewed API extensions or changes.**
  - No API changes. The fix modifies internal comparison logic only.

- [x] **Assessed topology or deployment constraints.**
  - The script runs as a GitHub Actions workflow (`repo-maintenance.yml`). No topology constraints.

### II. Test Planning

#### II.1 - Scope of Testing

This test plan covers the shim drift detection and comparison logic in `reconcile-repos.sh`, specifically the fix that replaces base64-level comparison with decoded text comparison. Testing validates that encoding differences do not cause false-positive drift detection, that genuine drift is still detected, and that the sentinel line is preserved in all output paths.

**Testing Goals:**

- **P0:** Verify that logically identical shim content with encoding differences (trailing newlines, carriage returns) is correctly identified as up-to-date.
- **P0:** Verify that the sentinel line `# --- fullsend managed below - do not edit ---` is present in all generated shim blobs.
- **P1:** Verify that genuinely different content is correctly flagged as stale and triggers an update PR.
- **P1:** Verify that pre-sentinel shims (without sentinel line) fall back to full decoded content comparison.
- **P2:** Verify that user-owned comment headers above the sentinel are preserved and non-comment injection is rejected.

**Out of Scope (Testing Scope Exclusions):**

- [ ] **GitHub API base64 encoding behavior** -- Platform-level concern; tested by GitHub. We test our handling of API responses, not the API itself.
- [ ] **yq/jq YAML parsing correctness** -- Third-party tool behavior; tested by tool maintainers.
- [ ] **Branch protection and PR merge behavior** -- GitHub platform feature; not product-specific.
- [ ] **Go scaffold installation path (scaffold.go, workflows.go)** -- Separate code path from bash reconciliation; has its own test coverage.

#### II.2 - Test Strategy

**Functional:**

- [x] **Functional Testing** -- Applicable. Core focus: validate comparison logic produces correct stale/up-to-date decisions for various input combinations.
  - Covers decoded text comparison, sentinel extraction, fallback paths, and injection guard.

- [x] **Automation Testing** -- Applicable. All tests are automated in `reconcile-repos-test.sh` bash test harness.
  - Tests run in CI via `make test` or direct script invocation.

- [x] **Regression Testing** -- Applicable. Test 5 in the test harness is a direct regression test for GH-2247.
  - Validates the specific scenario (trailing newline difference) that caused PR #2101.

**Non-Functional:**

- [ ] **Performance Testing** -- Not applicable. Comparison logic change has negligible performance impact.

- [ ] **Scale Testing** -- Not applicable. Script processes repos sequentially; no scale concern for comparison logic.

- [ ] **Security Testing** -- Not applicable. No new attack surface. Existing injection guard (non-comment content rejection) is covered by existing tests.

- [ ] **Usability Testing** -- Not applicable. No user-facing UI changes.

- [ ] **Monitoring** -- Not applicable. No observability changes.

**Integration & Compatibility:**

- [ ] **Compatibility Testing** -- Not applicable. Bash script runs on fixed GitHub Actions Ubuntu runner.

- [ ] **Upgrade Testing** -- Not applicable. No version migration path for comparison logic.

- [ ] **Dependencies** -- Not applicable. No new dependencies introduced.

- [ ] **Cross Integrations** -- Not applicable. Fix is isolated to comparison logic within reconcile-repos.sh.

**Infrastructure:**

- [ ] **Cloud Testing** -- Not applicable. No cloud-specific behavior.

#### II.3 - Test Environment

- **Cluster Topology:** N/A (no cluster required)
- **Platform Version:** GitHub Actions Ubuntu runner (ubuntu-latest)
- **CPU Virtualization:** N/A
- **Compute:** Standard GitHub Actions runner
- **Special Hardware:** None
- **Storage:** Ephemeral tmpdir for test fixtures
- **Network:** Mocked (no real GitHub API calls)
- **Operators:** N/A
- **Platform:** Linux (GNU coreutils for base64, awk, grep)
- **Special Configs:** Mock `gh` CLI scripts, mock `yq`, test config.yaml with enabled/disabled repos

#### II.3.1 - Testing Tools & Frameworks

No new or special tools required. All tests use standard bash scripting with mock commands.

#### II.4 - Entry Criteria

- [x] Fix PR merged (or available on test branch) with changes to `reconcile-repos.sh` lines 404-416
- [x] `reconcile-repos-test.sh` updated with Test 5 (trailing newline regression test)
- [x] Mock `gh` CLI supports content API response simulation with configurable base64 content

#### II.5 - Risks

- [ ] **Timeline**
  - Risk: Test harness relies on GNU coreutils behavior (`base64 -w0`); macOS developers cannot run tests locally.
  - Mitigation: Tests run exclusively in CI on Ubuntu runners. Document this requirement.
  - Status: Low risk.

- [ ] **Coverage**
  - Risk: Tests use mocked GitHub API responses, which may not capture all real-world encoding variations.
  - Mitigation: Test 5 specifically models the encoding difference observed in the real bug (PR #2101). Additional encoding variations (e.g., CRLF) covered by carriage return normalization test.
  - Status: Acceptable.

- [ ] **Environment**
  - Risk: None identified. Test environment is simple (bash + mocks).
  - Mitigation: N/A.
  - Status: N/A.

- [ ] **Untestable**
  - Risk: Real GitHub content API encoding behavior cannot be tested without live API calls.
  - Mitigation: Mock responses model observed real-world behavior. The fix is defensive (normalizes before comparing) rather than targeting a specific encoding.
  - Status: Acceptable.

- [ ] **Resources**
  - Risk: None identified.
  - Mitigation: N/A.
  - Status: N/A.

- [ ] **Dependencies**
  - Risk: None identified. No external dependencies beyond GNU coreutils.
  - Mitigation: N/A.
  - Status: N/A.

- [ ] **Other**
  - Risk: If the sentinel string is changed in the template, the `SENTINEL` variable in the script must be updated in tandem, or comparison silently falls through to full-content comparison.
  - Mitigation: Document the coupling in code comments. Consider adding a consistency check in CI.
  - Status: Low risk.

---

### III. Test Execution

#### III.1 - Requirements-to-Tests Mapping

- **GH-2247** | Shim drift detection correctly identifies logically identical content as up-to-date
  - Verify identical content with extra trailing newline not flagged stale | Unit Tests | P0
  - Verify identical content with no trailing newline not flagged stale | Unit Tests | P0
  - Verify genuinely different content is flagged stale | Unit Tests | P0
  - Verify carriage return differences ignored in comparison | Unit Tests | P0

- **GH-2247** | Sentinel line is preserved in all shim blob outputs
  - Verify sentinel present in new enrollment shim | Unit Tests | P0
  - Verify sentinel present in updated stale shim | Unit Tests | P0
  - Verify sentinel survives injection guard rejection | Unit Tests | P0

- **GH-2247** | Pre-sentinel shim comparison falls back to full decoded content
  - Verify pre-sentinel shim matches full decoded content | Unit Tests | P1
  - Verify pre-sentinel shim detects genuine drift | Unit Tests | P1
  - Verify empty extract_managed_content triggers fallback | Unit Tests | P1

- **GH-2247** | Stale shim detection triggers update PR only for genuine content drift
  - Verify update PR created for genuine template change | Functional | P1
  - Verify no PR created when content matches | Functional | P1
  - Verify no blob created for false positive drift | Functional | P1

- **GH-2247** | User-owned header above sentinel is preserved during shim updates
  - Verify comment header preserved above sentinel | Unit Tests | P2
  - Verify non-comment content above sentinel rejected | Unit Tests | P2

- **GH-2247** | Base64 encoding/decoding round-trip does not corrupt shim content
  - Verify base64 round-trip preserves multi-line YAML | Unit Tests | P1
  - Verify GitHub API base64 line wrapping handled | Unit Tests | P1

---

### IV. Sign-off

| Role | Name | Date |
|:-----|:-----|:-----|
| QE Lead | TBD | |
| Dev Lead | TBD | |
| Product Owner | TBD | |
