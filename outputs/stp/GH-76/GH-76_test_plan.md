# Test Plan

## **Bound Enrollment Wait with Timeout and Backoff - Quality Engineering Plan**

### Metadata & Tracking

- **Enhancement:** [GH-76](https://github.com/guyoron1/fullsend/pull/76)
- **Feature Tracking:** [GH-76](https://github.com/guyoron1/fullsend/pull/76) — perf(#2354): bound enrollment wait with timeout and backoff
- **Epic Tracking:** [GH-2354](https://github.com/fullsend-ai/fullsend/issues/2354) — Enrollment wait timeout (upstream)
- **QE Owner:** QualityFlow (auto-generated)
- **Owning SIG:** N/A
- **Participating SIGs:** N/A

**Document Conventions:** This document follows the QualityFlow STP template. Test tiers use "Functional" for single-feature tests and "End-to-End" for multi-feature workflow tests.

### Feature Overview

This PR adds bounded timeout and exponential backoff to the enrollment wait loop in the fullsend CLI. Previously, `awaitWorkflowRun` could block indefinitely when the repo-maintenance workflow failed to appear or complete. The change introduces a 3-minute timeout (`enrollmentWaitTimeout`), an initial 2-second poll interval (`enrollmentPollInitial`), and a 15-second backoff cap (`enrollmentPollMax`) with exponential doubling via `nextInterval()`. Additionally, the PR migrates status comment token acquisition from static `--status-token` to on-demand minting via `--mint-url` / `FULLSEND_MINT_URL`, and adds a new `reconcile-status` CLI command for finalizing orphaned status comments.

---

### I. Motivation & Requirements

#### I.1 Requirement & User Story Review Checklist

- [ ] **Reviewed the relevant requirements.**
  - PR #76 description and upstream issue #2354 reviewed. The requirement is to prevent unbounded blocking during enrollment workflow polling.
  - Enrollment wait previously had no upper bound; this caused silent hangs when workflow dispatch failed or was delayed.

- [ ] **Confirmed clear user stories and understood. Understand the value and customer use cases.**
  - User story: As a fullsend operator running `fullsend install`, I want the enrollment wait to be bounded so the CLI does not hang indefinitely if the repo-maintenance workflow is slow or fails.
  - Value: Improves operator experience by providing timely feedback and preventing resource waste on stalled operations.

- [ ] **Confirmed requirements are **testable and unambiguous**.**
  - Timeout value (3 min) and backoff parameters (2s initial, 15s max, 2x doubling) are explicit constants, directly testable.
  - Status comment lifecycle (start, completion, orphan reconciliation) has clear state machine semantics.

- [ ] **Ensured acceptance criteria are **defined clearly**.**
  - AC1: `awaitWorkflowRun` returns a timeout error after `enrollmentWaitTimeout` elapses.
  - AC2: Polling interval doubles from `enrollmentPollInitial` to `enrollmentPollMax` and caps.
  - AC3: Context cancellation is respected within the poll loop.
  - AC4: `reconcile-status` command finalizes orphaned start comments.
  - AC5: `--mint-url` replaces `--status-token` for token acquisition.

- [ ] **Confirmed coverage for NFRs.**
  - Performance: Backoff reduces API call rate under load (exponential decay from 2s to 15s).
  - Reliability: Timeout prevents indefinite hangs; errors are non-fatal (Install continues).
  - Security: Mint-based token acquisition avoids long-lived static tokens.

#### I.2 Known Limitations

- The 3-minute timeout is a compile-time constant and is not user-configurable. Environments with unusually slow GitHub Actions runners may hit the timeout during normal operation.
- `reconcile-status` requires `--mint-url` or `FULLSEND_MINT_URL`; the deprecated `--token` flag will be removed in a future release.
- Orphan reconciliation relies on HTML comment markers in issue comments; external tools that strip HTML comments may break detection.

#### I.3 Technology and Design Review

- [ ] **Developer handoff completed and any technology challenges are understood.**
  - Implementation uses standard Go `time` package for backoff; no new dependencies introduced.
  - `nextInterval` is a pure function with deterministic doubling behavior.

- [ ] **Technology challenges identified and addressed.**
  - GitHub Actions workflow dispatch is eventually consistent; the poll loop accounts for initial registration delay with informational messages.

- [ ] **Test environment needs identified.**
  - Unit tests use `forge.FakeClient` for mocked GitHub API interactions; no real cluster or API access needed.
  - Status comment tests use mock `forge.Client` implementations.

- [ ] **API extensions reviewed.**
  - New CLI command `reconcile-status` added with flags: `--repo`, `--number`, `--run-id`, `--run-url`, `--sha`, `--reason`, `--mint-url`, `--role`.
  - `--status-token` deprecated across `run` and `reconcile-status` commands in favor of `--mint-url`.

- [ ] **Topology and deployment considerations reviewed.**
  - Changes are CLI-side only; no changes to sandbox, gateway, or deployed infrastructure.
  - Mint URL is resolved from flag or `FULLSEND_MINT_URL` environment variable.

---

### II. Strategy & Logistics

#### II.1 Scope of Testing

This test plan covers the enrollment wait timeout/backoff mechanism, the `nextInterval` backoff function, the `reconcile-status` CLI command, and the status comment notification lifecycle including orphan reconciliation.

**Testing Goals:**

- **P0:** Verify enrollment wait times out after the configured deadline and returns a descriptive error.
- **P0:** Verify exponential backoff doubles the interval and caps at `enrollmentPollMax`.
- **P0:** Verify `reconcile-status` finalizes orphaned start comments correctly.
- **P1:** Verify context cancellation exits the poll loop promptly.
- **P1:** Verify status comment placement logic (update-in-place vs. new comment).
- **P1:** Verify mint-based token acquisition for status operations.
- **P2:** Verify graceful handling when workflow listing returns transient errors.

**Out of Scope (Testing Scope Exclusions):**

- [ ] **GitHub Actions workflow dispatch reliability** — Platform-level concern; we test our polling and timeout, not GitHub's dispatch mechanism.
- [ ] **Mint service availability and token generation** — Tested by the mint service's own test suite; we test the client integration.
- [ ] **Sandbox creation, bootstrap, and agent execution** — Unchanged by this PR; covered by existing e2e tests.
- [ ] **OIDC token refresh and GCP authentication** — Unmodified code paths; existing test coverage sufficient.

#### II.2 Test Strategy

**Functional:**

- [x] **Functional Testing** — Applicable
  - Enrollment timeout behavior, backoff calculation, context cancellation, status comment lifecycle, orphan reconciliation, CLI flag parsing.
- [x] **Automation Testing** — Applicable
  - All tests are automated Go unit tests using `testing` + `testify`; executed in CI via `go test`.
- [x] **Regression Testing** — Applicable
  - Existing enrollment tests (`TestEnrollmentLayer_Install_*`, `TestEnrollmentLayer_Uninstall_*`, `TestEnrollmentLayer_Analyze_*`) validate that timeout/backoff changes don't break existing behavior.

**Non-Functional:**

- [ ] **Performance Testing** — Not Applicable
  - Backoff parameters are constants; no dynamic performance tuning to validate.
- [ ] **Scale Testing** — Not Applicable
  - Single workflow poll loop; no multi-resource scaling concern.
- [ ] **Security Testing** — Not Applicable
  - Mint URL token flow is tested functionally; no new attack surface introduced.
- [ ] **Usability Testing** — Not Applicable
  - CLI output changes are informational messages; no interactive UI.
- [ ] **Monitoring** — Not Applicable
  - No new metrics or observability endpoints added.

**Integration & Compatibility:**

- [x] **Compatibility Testing** — Applicable
  - `--status-token` deprecation path must remain functional alongside `--mint-url`.
- [ ] **Upgrade Testing** — Not Applicable
  - CLI binary replacement; no stateful upgrade path.
- [x] **Dependencies** — Applicable
  - `forge.Client` interface contract must be preserved; `FakeClient` tests verify this.
- [ ] **Cross Integrations** — Not Applicable
  - No cross-component integration changes.

**Infrastructure:**

- [ ] **Cloud Testing** — Not Applicable
  - All tests run locally with mocked GitHub API.

#### II.3 Test Environment

- **Cluster Topology:** N/A — unit tests only, no cluster required
- **Platform Version:** Go 1.26.0 (as specified in go.mod)
- **CPU Virtualization:** N/A
- **Compute:** Standard CI runner
- **Special Hardware:** None
- **Storage:** Local filesystem for test artifacts
- **Network:** No external network access required (mocked API)
- **Operators:** N/A
- **Platform:** Linux (CI), macOS (local development)
- **Special Configs:** `forge.FakeClient` configured with `WorkflowRuns`, `FileContents`, `PullRequests`, `VariableValues`, and `Errors` maps

#### II.3.1 Testing Tools & Frameworks

No new or special tools required. Standard `go test` with `testify` assertions.

#### II.4 Entry Criteria

- [ ] PR #76 merged or branch available for testing
- [ ] `go build ./...` succeeds without errors
- [ ] `go vet ./...` reports no issues
- [ ] All pre-existing tests pass (`go test ./internal/layers/... ./internal/statuscomment/... ./internal/cli/...`)

#### II.5 Risks

- [ ] **Timeline**
  - Risk: Tight timeline if upstream #2359 requires further iteration.
  - Mitigation: PR is a mirror of upstream; changes are stable.
  - Status: [ ] Resolved

- [ ] **Coverage**
  - Risk: Real GitHub Actions timing cannot be tested in unit tests.
  - Mitigation: `FakeClient` simulates all workflow states; timeout logic is deterministic.
  - Status: [ ] Accepted

- [ ] **Environment**
  - Risk: None — all tests run with mocked dependencies.
  - Mitigation: N/A
  - Status: [x] Resolved

- [ ] **Untestable**
  - Risk: Actual exponential backoff wall-clock timing is not validated (tests use fake time).
  - Mitigation: `nextInterval` is a pure function tested independently; time progression is implicit.
  - Status: [ ] Accepted

- [ ] **Resources**
  - Risk: None — standard Go test infrastructure.
  - Mitigation: N/A
  - Status: [x] Resolved

- [ ] **Dependencies**
  - Risk: `forge.FakeClient` must accurately model `forge.Client` interface changes.
  - Mitigation: Compile-time interface check (`var _ Layer = (*EnrollmentLayer)(nil)`).
  - Status: [x] Resolved

- [ ] **Other**
  - Risk: Deprecated `--token` flag removal in future release may break existing CI configurations.
  - Mitigation: Deprecation warning emitted; documented migration path to `--mint-url`.
  - Status: [ ] Monitoring

---

### III. Test Deliverables

#### III.1 Requirements-to-Tests Mapping

- **Requirement ID:** GH-76
  **Requirement Summary:** Enrollment wait is bounded by timeout with exponential backoff
  **Test Scenarios:**
  - Verify `awaitWorkflowRun` returns timeout error after deadline elapses (positive)
  - Verify `awaitWorkflowRun` returns completed run before deadline (positive)
  - Verify timeout error message includes elapsed duration (positive)
  - Verify context cancellation exits poll loop immediately (positive)
  - Verify error on timeout is non-fatal (Install succeeds with warning) (positive)
  - Verify unbounded poll does not occur when no runs appear (negative)
  **Tier:** Unit Tests
  **Priority:** P0

- **Requirement Summary:** Polling interval follows exponential backoff with cap
  **Test Scenarios:**
  - Verify `nextInterval` doubles 2s to 4s (positive)
  - Verify `nextInterval` doubles 4s to 8s (positive)
  - Verify `nextInterval` caps at `enrollmentPollMax` (15s) (positive)
  - Verify `nextInterval` stays at max when already at max (positive)
  - Verify poll loop uses increasing intervals between API calls (positive)
  **Tier:** Unit Tests
  **Priority:** P0

- **Requirement Summary:** Status comments are posted at agent start and updated on completion
  **Test Scenarios:**
  - Verify start comment is posted with correct marker and timestamp (positive)
  - Verify completion comment updates start comment in place when it is last (positive)
  - Verify new completion comment posted when other activity follows start (positive)
  - Verify completion deletes start comment when completion notifications disabled (positive)
  - Verify client factory mints fresh token before each API call (positive)
  - Verify graceful handling when start comment not found on timeline (negative)
  **Tier:** Unit Tests
  **Priority:** P0

- **Requirement Summary:** Orphaned status comments are reconciled after hard process kill
  **Test Scenarios:**
  - Verify orphaned start comment is updated to "Interrupted" state (positive)
  - Verify already-terminal comment is left unchanged (positive)
  - Verify no error when no matching comment exists (positive)
  - Verify cancelled reason produces "Cancelled" label (positive)
  - Verify terminated reason produces "Terminated" label (positive)
  - Verify invalid run ID is rejected (negative)
  **Tier:** Unit Tests
  **Priority:** P1

- **Requirement Summary:** `reconcile-status` CLI command finalizes orphaned comments
  **Test Scenarios:**
  - Verify command calls `ReconcileOrphaned` with correct parameters (positive)
  - Verify `--mint-url` flag mints token for API access (positive)
  - Verify deprecated `--token` flag still works with warning (positive)
  - Verify error when `--number` is not positive (negative)
  - Verify error when `--repo` is not in owner/repo format (negative)
  - Verify error when neither `--mint-url` nor `--token` provided (negative)
  **Tier:** Unit Tests
  **Priority:** P1

- **Requirement Summary:** Enrollment Install and Uninstall use bounded wait
  **Test Scenarios:**
  - Verify Install dispatches workflow and waits for completion (positive)
  - Verify Install reports enrollment PRs after successful workflow (positive)
  - Verify Install reports removal PRs for disabled repos (positive)
  - Verify Install with no repos skips dispatch (positive)
  - Verify Install dispatch error is fatal (negative)
  - Verify Install workflow failure is non-fatal with warning (positive)
  - Verify Uninstall disables all repos and dispatches workflow (positive)
  - Verify Uninstall handles missing config gracefully (positive)
  - Verify Uninstall dispatch error is non-fatal (negative)
  **Tier:** Functional
  **Priority:** P0

- **Requirement Summary:** Status notification token acquisition uses mint service
  **Test Scenarios:**
  - Verify `setupStatusNotifier` creates factory with mint URL (positive)
  - Verify `setupStatusNotifier` reads `FULLSEND_MINT_URL` from environment (positive)
  - Verify deprecated `--status-token` creates static client with warning (positive)
  - Verify error when no mint URL and no token available (negative)
  **Tier:** Unit Tests
  **Priority:** P1

- **Requirement Summary:** Enrollment Analyze detects per-repo guard and drift
  **Test Scenarios:**
  - Verify all-enrolled repos report StatusInstalled (positive)
  - Verify missing shim reports StatusNotInstalled (positive)
  - Verify partial enrollment reports StatusDegraded (positive)
  - Verify per-repo guard variable skips org-level analysis (positive)
  - Verify stale shim on disabled repo generates removal recommendation (positive)
  - Verify guard check failure surfaces warning (negative)
  **Tier:** Functional
  **Priority:** P1

---

### IV. Sign-off

| Role | Name | Date |
|:-----|:-----|:-----|
| QE Lead | | |
| Dev Lead | | |
| PM | | |
