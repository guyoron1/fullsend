# Test Plan

## **[fix(#2247): Compare Decoded Text in Shim Drift Detection] - Quality Engineering Plan**

### Metadata & Tracking

- **Enhancement:** [GH-77](https://github.com/guyoron1/fullsend/pull/77) — Mirror of upstream fullsend-ai/fullsend#2254
- **Feature Tracking:** [GH-77](https://github.com/guyoron1/fullsend/pull/77)
- **Epic Tracking:** [#2247](https://github.com/fullsend-ai/fullsend/issues/2247) — Shim drift false-positive detection
- **QE Owner:** TBD
- **Owning SIG:** N/A
- **Participating SIGs:** N/A

**Document Conventions:** Standard QualityFlow STP format. All test scenarios target the `reconcile-repos.sh` script and its test harness (`reconcile-repos-test.sh`). "Shim" refers to the `.github/workflows/fullsend.yaml` workflow file managed by the enrollment system.

### Feature Overview

This fix addresses issue #2247 where the shim drift detection logic in `reconcile-repos.sh` produced false-positive "stale" results for enrolled repositories. The root cause was that `managed_content_b64()` re-encoded extracted content to base64 for comparison, amplifying trivial whitespace differences (trailing newlines, CR/LF variations from the GitHub Content API) into mismatched base64 strings. The fix decodes both the expected and remote content to plain text, strips carriage returns, and compares the decoded strings directly. A new fallback path also handles pre-sentinel shims by comparing full decoded content when no sentinel line is found.

---

### Section I — Motivation & Requirements Review

#### I.1 — Requirement & User Story Review Checklist

- [ ] **Reviewed the relevant requirements.** -- Confirmed the requirement is based on issue #2247 (false-positive drift detection) and upstream PR fullsend-ai/fullsend#2254.
  - The issue describes a concrete bug: identical shim content flagged as stale due to encoding differences.
  - Root cause is well-documented: `managed_content_b64()` re-encodes to base64, amplifying trailing newline differences.

- [ ] **Confirmed clear user stories and understood. Understand the value and customer use cases.** -- The user story is: "As a repo maintainer, I expect that repos with up-to-date shims are not subjected to spurious update PRs."
  - Customer impact: false-positive drift creates unnecessary PRs and CI noise for enrolled repos.

- [ ] **Confirmed requirements are **testable and unambiguous**.** -- The fix is directly testable via the existing reconcile-repos-test.sh harness using mocked `gh` CLI responses.
  - Test 5 (added in this PR) directly validates the regression scenario.

- [ ] **Ensured acceptance criteria are **defined clearly**.** -- Acceptance criteria inferred from PR description and test assertions:
  - Identical content with different trailing newlines must not be flagged as stale.
  - Genuinely different content must still be flagged as stale.
  - No blob or PR should be created for encoding-only differences.

- [ ] **Confirmed coverage for NFRs.** -- Non-functional requirements are minimal for this bug fix.
  - Performance: no significant change (base64 decode is equivalent cost to re-encode).
  - Cross-platform: CR/LF normalization with `tr -d '\r'` ensures consistent behavior.

#### I.2 — Known Limitations

- The `managed_content_b64()` function remains in the script but is no longer called in the drift comparison path. It may be dead code pending cleanup.
- The `tr -d '\r'` normalization strips all carriage returns, which is correct for YAML workflow files but would be lossy for binary content (not applicable here).
- Pre-sentinel shim fallback compares full decoded content, which means any user-added header (comments or otherwise) in a pre-sentinel shim would cause a drift detection. This is acceptable because pre-sentinel shims predate the header-preservation feature.

#### I.3 — Technology and Design Review

- [ ] **Developer handoff completed; design reviewed with development team.** -- PR is a mirror of upstream fullsend-ai/fullsend#2254, authored by the maintainer.
  - Change is small (3 lines of production code replaced, 2 lines removed) and well-scoped.

- [ ] **Technology challenges and constraints identified.** -- No new technology introduced.
  - Fix uses standard shell utilities (`base64 -d`, `tr`, `printf`) available on all GitHub Actions runners.

- [ ] **Test environment needs assessed.** -- No special environment required.
  - Tests run via bash with a mock `gh` binary; no cluster, API, or network access needed.

- [ ] **API or interface extensions reviewed.** -- No API changes.
  - The script's external interface (exit codes, stdout messages) is unchanged.

- [ ] **Topology and deployment considerations reviewed.** -- Not applicable.
  - The reconcile script runs as a GitHub Actions workflow step; no topology constraints.

### Section II — Test Planning

#### II.1 — Scope of Testing

This test plan covers the shim drift detection logic in `reconcile-repos.sh`, specifically the comparison of expected vs. remote shim content for enrolled repositories. The fix changes the comparison from base64-encoded strings to decoded text strings, with CR/LF normalization.

**Testing Goals:**

- **P0:** Verify that identical content with encoding differences is correctly recognized as up-to-date (regression fix validation)
- **P0:** Verify that genuinely stale content is still detected and triggers an update PR (no regression in stale detection)
- **P1:** Verify pre-sentinel shim fallback path handles both matching and differing content
- **P1:** Verify no unnecessary blob writes or PR creations for up-to-date shims
- **P2:** Verify CR/LF normalization handles mixed line endings
- **P2:** Verify content-injection guard is unaffected by adjacent changes

**Out of Scope (Testing Scope Exclusions):**

- [ ] **GitHub Content API base64 encoding behavior** -- Platform-level concern; tested by GitHub.
- [ ] **base64 CLI utility correctness across OS versions** -- OS/coreutils responsibility.
- [ ] **Full enrollment workflow (end-to-end with real GitHub repos)** -- Covered by e2e/admin tests, not this STP.
- [ ] **Go scaffold embedding (go:embed)** -- Compile-time embedding; verified by existing scaffold_test.go.

#### II.2 — Test Strategy

**Functional:**

- [x] **Functional Testing** -- Applicable. Core drift comparison logic must be validated with multiple content variations (identical, different trailing newlines, genuinely stale, pre-sentinel).
- [x] **Automation Testing** -- Applicable. All tests are automated via `reconcile-repos-test.sh` bash harness with mock `gh` CLI.
- [x] **Regression Testing** -- Applicable. Test 5 is a dedicated regression test for issue #2247.

**Non-Functional:**

- [ ] **Performance Testing** -- Not applicable. The change replaces one shell pipeline with another of equivalent complexity.
- [ ] **Scale Testing** -- Not applicable. Script processes repos sequentially; no scale dimension affected.
- [ ] **Security Testing** -- Not applicable. Content-injection guard is unchanged; no new attack surface.
- [ ] **Usability Testing** -- Not applicable. No user-facing interface changes.
- [ ] **Monitoring** -- Not applicable. No observability changes.

**Integration & Compatibility:**

- [ ] **Compatibility Testing** -- Not applicable. Shell utilities used (`base64 -d`, `tr`) are POSIX-standard.
- [ ] **Upgrade Testing** -- Not applicable. No versioned state or migration path.
- [ ] **Dependencies** -- Not applicable. No new dependencies introduced.
- [ ] **Cross Integrations** -- Not applicable. Change is internal to reconcile script.

**Infrastructure:**

- [ ] **Cloud Testing** -- Not applicable. Script runs on standard GitHub Actions ubuntu runners.

#### II.3 — Test Environment

- **Cluster Topology:** N/A — no cluster required; tests run locally via bash
- **Platform Version:** GitHub Actions ubuntu-latest runner
- **CPU Virtualization:** N/A
- **Compute:** Standard GitHub Actions runner (2 vCPU, 7 GB RAM)
- **Special Hardware:** None
- **Storage:** Ephemeral runner disk (default)
- **Network:** No network access required; `gh` CLI is mocked
- **Operators:** N/A
- **Platform:** Linux (bash 5.x, coreutils base64, jq, yq)
- **Special Configs:** Mock `gh` binary injected via `$PATH` override; temporary directory for test artifacts

#### II.3.1 — Testing Tools & Frameworks

No new or special tools. Tests use standard bash scripting with mock binaries.

#### II.4 — Entry Criteria

- [ ] PR branch builds successfully (CI green)
- [ ] Existing reconcile-repos-test.sh tests 1-4 pass (no regression in existing tests)
- [ ] Mock `gh` binary correctly simulates GitHub Content API responses for test scenarios

#### II.5 — Risks

- [ ] **Timeline**
  - Risk: None identified; fix is small and well-scoped.
  - Mitigation: N/A
  - Status: [ ] Low risk

- [ ] **Coverage**
  - Risk: Edge cases in base64 encoding across different `base64` implementations (GNU vs BSD).
  - Mitigation: `base64 -d` is POSIX-standard; GitHub Actions uses GNU coreutils.
  - Status: [ ] Low risk

- [ ] **Environment**
  - Risk: None; tests run entirely locally with mocked dependencies.
  - Mitigation: N/A
  - Status: [ ] Low risk

- [ ] **Untestable**
  - Risk: Real GitHub Content API encoding variations cannot be fully replicated in mocks.
  - Mitigation: Test 5 simulates the specific encoding difference (extra trailing newline) that caused issue #2247.
  - Status: [ ] Accepted risk

- [ ] **Resources**
  - Risk: None; no special resources needed.
  - Mitigation: N/A
  - Status: [ ] Low risk

- [ ] **Dependencies**
  - Risk: None; no external dependencies.
  - Mitigation: N/A
  - Status: [ ] Low risk

- [ ] **Other**
  - Risk: `managed_content_b64()` function is now dead code in the drift path; may confuse future maintainers.
  - Mitigation: Consider removing or deprecating the function in a follow-up cleanup.
  - Status: [ ] Low risk

---

### Section III — Requirements-to-Tests Mapping

#### III.1 — Requirements Mapping

- **GH-77** — Shim drift detection correctly identifies identical content despite encoding differences
  - Verify identical content with different trailing newlines not flagged as stale — Functional — P0
  - Verify up-to-date shim produces "already enrolled" status — Functional — P0
  - Verify no blob or PR created for encoding-only differences — Functional — P0

- **GH-77** — Genuinely stale shim content is still detected and triggers an update PR
  - Verify stale shim triggers update PR creation — Functional — P0
  - Verify stale detection after template content change — Functional — P0
  - Verify error handling when update PR creation fails — Functional — P0

- **GH-77** — Pre-sentinel shim files fall back to full decoded content comparison
  - Verify pre-sentinel shim compares full decoded content — Functional — P1
  - Verify pre-sentinel shim with identical content not flagged stale — Functional — P1
  - Verify pre-sentinel shim with different content flagged stale — Functional — P1

- **GH-77** — Enrolled repos with up-to-date shims are skipped without creating unnecessary PRs or blob writes
  - Verify no blob created for up-to-date shim — Functional — P1
  - Verify skip counter incremented for current shim — Functional — P1

- **GH-77** — CR/LF normalization prevents cross-platform drift false positives
  - Verify CRLF content normalized before comparison — Functional — P2
  - Verify mixed line endings handled correctly — Functional — P2

- **GH-77** — Content-injection guard still rejects non-comment YAML above sentinel
  - Verify non-comment YAML above sentinel rejected — Functional — P2
  - Verify comment-only header preserved during update — Functional — P2

---

### Section IV — Sign-off

| Role | Name | Date |
|:-----|:-----|:-----|
| QE Lead | TBD | |
| Dev Lead | TBD | |
| PM | TBD | |
