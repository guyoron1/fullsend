# Test Plan

## **[fix(#2247): Compare Decoded Text in Shim Drift Detection] - Quality Engineering Plan**

### Metadata & Tracking

- **Enhancement:** [GH-77 / fullsend-ai/fullsend#2254](https://github.com/fullsend-ai/fullsend/pull/2254) (fork PR: [guyoron1/fullsend#77](https://github.com/guyoron1/fullsend/pull/77))
- **Feature Tracking:** [GH-77](https://github.com/fullsend-ai/fullsend/pull/2254) — fix(#2247): compare decoded text in shim drift detection
- **Epic Tracking:** [GH-2247](https://github.com/fullsend-ai/fullsend/issues/2247) — Shim drift false-positive from trailing newline encoding differences
- **QE Owner:** Unassigned
- **Owning SIG:** Dispatch
- **Participating SIGs:** N/A

**Document Conventions:** Standard QualityFlow STP format. "Verify" denotes a positive validation; "Validate" denotes a constraint or negative check.

### Feature Overview

This fix addresses false-positive shim drift detection in the `reconcile-repos.sh` enrollment script. The previous implementation compared re-encoded base64 strings (via `managed_content_b64`), which produced spurious "stale" results when the remote content from GitHub's content API differed only by trailing newlines. The fix decodes both expected and remote base64 content to plaintext, strips carriage returns, and compares the decoded text directly. A fallback path compares full decoded content for pre-sentinel shims that lack a managed-content marker.

---

### Section I — Motivation & Requirements Review

#### I.1 — Requirement & User Story Review Checklist

- [ ] **Reviewed the relevant requirements.**
  - GH-77 fixes issue GH-2247: shim drift detection produced false-positive "stale" results due to base64 encoding differences from trailing newlines.
  - The root cause is that `managed_content_b64()` re-encoded decoded content to base64 for comparison, and trailing newline variations caused different base64 output for semantically identical content.

- [ ] **Confirmed clear user stories and understood. Understand the value and customer use cases.**
  - As a repository administrator with fullsend enrolled, I expect that the reconcile script does not create unnecessary update PRs when my shim workflow file is already up to date.
  - The false-positive drift caused noise PRs on every reconciliation cycle for affected repositories.

- [ ] **Confirmed requirements are **testable and unambiguous**.**
  - The fix is a well-scoped change to the comparison block (lines 404-416 of `reconcile-repos.sh`). Behavior is directly testable via the existing shell test harness (`reconcile-repos-test.sh` Test 5) and the generated Go unit tests.

- [ ] **Ensured acceptance criteria are **defined clearly**.**
  - Content that is identical except for trailing newlines must NOT be flagged as stale.
  - Content that is genuinely different MUST still be flagged as stale.
  - Pre-sentinel shims (no managed-content marker) must compare full decoded content.
  - Carriage returns must be stripped before comparison (cross-platform safety).

- [ ] **Confirmed coverage for NFRs.**
  - No performance, scale, or security NFRs apply. The comparison logic runs once per enrolled repo during reconciliation — no hot path.

#### I.2 — Known Limitations

- The fix relies on `base64 -d` and `tr -d '\r'` being available in the shell environment. These are standard coreutils but could behave differently on non-GNU systems (e.g., macOS `base64` uses `-D` instead of `-d`). The reconcile script runs in GitHub Actions (Ubuntu), so this is not a practical concern.
- The comparison normalizes `\r` but does not normalize other whitespace variations (e.g., trailing spaces within lines). This is intentional — only encoding-level differences are normalized.

#### I.3 — Technology and Design Review

- [ ] **Developer handoff completed; design reviewed with development team.**
  - PR mirrors upstream fullsend-ai/fullsend#2254. The fix is a 12-line change to the comparison block in `reconcile-repos.sh`, replacing `managed_content_b64()` calls with inline `base64 -d | tr -d '\r'` decoding.
  - QE engaged post-implementation; scope is a well-defined bug fix with clear acceptance criteria, making post-implementation test planning appropriate.

- [ ] **Technology challenges and constraints identified.**
  - No new technology introduced. The fix uses standard shell utilities (`base64`, `tr`, `printf`) already present in the script.

- [ ] **Test environment needs are understood and documented.**
  - Tests run in a shell environment with mocked `gh` CLI. No cluster or external service required.

- [ ] **API extensions and changes reviewed.**
  - No API changes. The fix is internal to the reconcile script's comparison logic.

- [ ] **Topology and deployment model impact assessed.**
  - No topology impact. The reconcile script runs as a single GitHub Actions workflow.

### Section II — Test Planning

#### II.1 — Scope of Testing

This test plan covers the shim drift detection comparison logic in `reconcile-repos.sh`, specifically the change from base64-to-base64 comparison to decoded-text comparison. Testing validates that encoding-neutral comparison eliminates false-positive drift while preserving detection of genuine content changes.

**Testing Goals:**

- **P0:** Verify false-positive drift from trailing newline differences is eliminated
- **P0:** Verify genuine content drift is still correctly detected and triggers update PRs
- **P1:** Verify base64 round-trip integrity for the new decode-compare path
- **P1:** Verify sentinel-based extraction works correctly on decoded text
- **P1:** Verify pre-sentinel fallback compares full decoded content
- **P2:** Verify carriage return normalization and user header preservation

**Out of Scope (Testing Scope Exclusions):**

- [ ] **GitHub content API encoding behavior** — Platform-level; not within project scope. GitHub's base64 encoding is an external dependency.
- [ ] **`base64` CLI correctness** — Coreutils testing; OS/distro responsibility.
- [ ] **PR creation mechanics** — The `gh pr create` flow is tested elsewhere; this plan covers only the drift *detection* logic.
- [ ] **Shim template content** — Template correctness is orthogonal to the comparison fix.

#### II.2 — Test Strategy

**Functional:**

- [x] **Functional Testing** — Applicable
  - Validate the decoded-text comparison logic produces correct stale/up-to-date decisions for various input combinations (trailing newlines, CR/LF, sentinel presence).
- [x] **Automation Testing** — Applicable
  - Shell test (Test 5 in `reconcile-repos-test.sh`) and Go unit tests in `qf-tests/GH-2247/go/` run in CI.
- [x] **Regression Testing** — Applicable
  - Existing Tests 1-4 in `reconcile-repos-test.sh` ensure no regression in enrollment, unenrollment, header preservation, and injection guard.

**Non-Functional:**

- [ ] **Performance Testing** — Not Applicable
  - Comparison runs once per repo per reconciliation cycle; no performance concern.
- [ ] **Scale Testing** — Not Applicable
  - No scale dimension; each repo comparison is independent.
- [ ] **Security Testing** — Not Applicable
  - No security surface change; content-injection guard is unchanged.
- [ ] **Usability Testing** — Not Applicable
  - No user-facing interface change.
- [ ] **Monitoring** — Not Applicable
  - No new metrics or observability changes.

**Integration & Compatibility:**

- [ ] **Compatibility Testing** — Not Applicable
  - Shell script runs in fixed GitHub Actions Ubuntu environment.
- [ ] **Upgrade Testing** — Not Applicable
  - No upgrade path; script is deployed atomically via scaffold.
- [ ] **Dependencies** — Not Applicable
  - No new dependencies introduced.
- [ ] **Cross Integrations** — Not Applicable
  - No cross-feature integration points affected.

**Infrastructure:**

- [ ] **Cloud Testing** — Not Applicable
  - No cloud-specific behavior.

#### II.3 — Test Environment

- **Cluster Topology:** N/A — no cluster required; tests run in shell and Go test environments
- **Platform Version:** Ubuntu (GitHub Actions runner)
- **CPU Virtualization:** N/A
- **Compute:** Standard GitHub Actions runner
- **Special Hardware:** None
- **Storage:** Local filesystem (tmpdir for test artifacts)
- **Network:** Mocked `gh` CLI — no real network calls
- **Operators:** N/A
- **Platform:** GitHub Actions
- **Special Configs:** Mocked `gh` binary in `$PATH` for shell tests; `testscript` pattern for Go tests

#### II.3.1 — Testing Tools & Frameworks

No new or special tools required. Standard Go `testing` + `testify` and bash test harness.

#### II.4 — Entry Criteria

- [ ] PR #77 merged or branch available for testing
- [ ] `reconcile-repos-test.sh` passes all 5 tests (including new Test 5)
- [ ] Go test files in `qf-tests/GH-2247/go/` compile and pass
- [ ] Existing reconcile tests (Tests 1-4) show no regression

#### II.5 — Risks

- [ ] **Timeline**
  - Specific Risk: None — fix is small and well-scoped.
  - Mitigation: N/A
  - Status: Low risk

- [ ] **Coverage**
  - Specific Risk: Edge cases in base64 encoding beyond trailing newlines (e.g., padding differences, line wrapping) may not be fully covered.
  - Mitigation: Go unit tests cover base64 round-trip with various content patterns including multi-line YAML, empty content, and special characters.
  - Status: Mitigated

- [ ] **Environment**
  - Specific Risk: Shell behavior differences between GNU and non-GNU `base64` utilities.
  - Mitigation: Reconcile script runs exclusively in GitHub Actions Ubuntu runners where GNU coreutils are standard.
  - Status: Mitigated

- [ ] **Untestable**
  - Specific Risk: Actual GitHub content API encoding variations cannot be reproduced deterministically in tests.
  - Mitigation: Tests simulate the known failure mode (extra trailing newline) and additional encoding variations.
  - Status: Accepted

- [ ] **Resources**
  - Specific Risk: None — no special resources needed.
  - Mitigation: N/A
  - Status: Low risk

- [ ] **Dependencies**
  - Specific Risk: None — no external dependencies changed.
  - Mitigation: N/A
  - Status: Low risk

- [ ] **Other**
  - Specific Risk: The `managed_content_b64()` function is now unused in the comparison path but remains in the script. Dead code could cause confusion.
  - Mitigation: Function may still be used elsewhere or removed in a follow-up cleanup.
  - Status: Accepted

---

### Section III — Requirements-to-Tests Mapping

#### III.1 — Requirements Mapping

- **GH-77** — Shim drift detection correctly identifies identical content regardless of encoding differences
  - Verify identical content with different trailing newlines is not flagged as stale — Functional (Unit) — P0
  - Verify comparison logic returns stale for genuinely different content — Functional (Unit) — P0
  - Verify GitHub API base64 line-wrapping does not cause false drift — Functional (Unit) — P1

- **GH-77** — Base64 encode/decode round-trip preserves content integrity for drift comparison
  - Verify base64 round-trip preserves multi-line YAML — Functional (Unit) — P1
  - Verify base64 round-trip of empty content produces empty decoded text without errors — Functional (Unit) — P2

- **GH-77** — Sentinel-based managed content extraction works on decoded text
  - Verify managed content extracted from sentinel onward — Functional (Unit) — P1
  - Verify empty result when no sentinel present — Functional (Unit) — P1

- **GH-77** — Pre-sentinel shim fallback compares full decoded content
  - Verify full content comparison for pre-sentinel shims — Functional (Unit) — P1
  - Verify pre-sentinel drift detected for different content — Functional (Unit) — P1
  - Verify fallback does not trigger when sentinel exists — Functional (Unit) — P1

- **GH-77** — User-owned headers above sentinel are preserved during shim updates
  - Verify comment headers preserved after drift update — Functional (Unit) — P2
  - Verify non-comment header injection rejected — Functional (Unit) — P2

- **GH-77** — Genuine shim drift triggers update PR creation while up-to-date shims are skipped
  - Verify stale detection triggers PR creation workflow — Functional (Integration) — P0
  - Verify up-to-date shim skips PR creation — Functional (Integration) — P0

- **GH-77** — Carriage return normalization prevents platform-specific comparison failures
  - Verify CRLF and LF content compared as equivalent — Functional (Unit) — P2
  - Verify mixed line endings handled correctly — Functional (Unit) — P2

- **GH-77** — Existing reconcile functionality is not regressed by the comparison logic change
  - Verify repository enrollment workflow completes successfully — Regression (Integration) — P1
  - Verify repository unenrollment removes shim correctly — Regression (Integration) — P1
  - Verify user-owned headers are preserved during shim update — Regression (Integration) — P1
  - Verify content-injection guard still rejects non-comment content above sentinel — Regression (Integration) — P1

---

### Section IV — Sign-off

- **Reviewers:** TBD
- **Approvers:** TBD
- **Date:** 2026-06-22
