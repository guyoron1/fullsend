# Test Plan — GH-2354

**Title:** Enrollment: long serial wait when activating repo-maintenance workflow
**Issue:** [GH-2354](https://github.com/fullsend-ai/fullsend/issues/2354)
**Author:** QualityFlow (auto-generated)
**Date:** 2026-06-21
**Product:** fullsend
**Status:** Open
**Priority:** Medium
**Component:** component/install

---

## 1. Overview

### 1.1 Problem Statement

After scaffold install, the enrollment layer waits for repo-maintenance workflow
registration and dispatch with chained polling/retry loops. The `awaitWorkflowRun`
method polls up to ~3 minutes with exponential backoff (2s → 15s cap). Combined
with upstream workflow dispatch and completion, install can block for extended
periods when GitHub is slow to register workflows, with no user-facing progress or
early termination.

### 1.2 Scope

This test plan covers changes to the enrollment workflow wait logic in
`internal/layers/enrollment.go` and its callers. The fix should ensure:

- Bounded, predictable wait times with configurable timeout
- Progress indicators during each polling phase
- Fail-fast with actionable error messages on timeout
- No regressions to happy-path enrollment or unenrollment flows

### 1.3 Related References

| Reference | Description |
|:----------|:------------|
| [GH-2354](https://github.com/fullsend-ai/fullsend/issues/2354) | Parent issue — enrollment long serial wait |
| [PR #1954](https://github.com/fullsend-ai/fullsend/pull/1954) | Origin PR — `--vendor` flag introducing enrollment changes |
| `internal/layers/enrollment.go` | Core enrollment layer implementation |
| `internal/layers/layers.go` | Layer stack orchestration (`InstallAll`, `UninstallAll`) |
| `internal/forge/forge.go` | Forge client interface (`DispatchWorkflow`, `ListWorkflowRuns`) |

---

## 2. Regression Analysis

### 2.1 LSP Call Graph Summary

Analysis performed via gopls LSP on `/sandbox/workspace/pr-repo`.

| Symbol | File | Line | Relationship |
|:-------|:-----|:-----|:-------------|
| `EnrollmentLayer.Install` | `internal/layers/enrollment.go` | 81 | Entry point — 8 direct test callers + `InstallAll` in `layers.go:109` |
| `EnrollmentLayer.awaitWorkflowRun` | `internal/layers/enrollment.go` | 121 | Called by `Install` (line 98) and `Uninstall` (line 286) |
| `nextInterval` | `internal/layers/enrollment.go` | 173 | Exponential backoff helper — called by `awaitWorkflowRun` |
| `EnrollmentLayer.Uninstall` | `internal/layers/enrollment.go` | 230 | Shares `awaitWorkflowRun` — same timeout behavior |
| `Stack.InstallAll` | `internal/layers/layers.go` | 104 | Orchestrator — calls `Install` on each layer in order |
| `forge.Client.DispatchWorkflow` | `internal/forge/forge.go` | 262 | Interface method — dispatches workflow via GitHub API |
| `forge.Client.ListWorkflowRuns` | `internal/forge/forge.go` | 296 | Interface method — polls for workflow run status |
| `forge.Client.GetWorkflowRunLogs` | `internal/forge/forge.go` | 300 | Interface method — fetches logs on failure |

### 2.2 Impacted Features

| Feature | Relationship | Why It Might Break |
|:--------|:-------------|:-------------------|
| Enrollment install flow | Direct — `Install()` calls `awaitWorkflowRun` | Timeout/backoff changes affect wait behavior |
| Enrollment uninstall flow | Direct — `Uninstall()` calls `awaitWorkflowRun` | Same shared polling logic |
| Layer stack orchestration | Indirect — `InstallAll()` calls `Install()` | Timeout changes propagate to full install pipeline |
| Progress/UI output | Direct — `ui.StepInfo` calls in `awaitWorkflowRun` | Progress indicator changes affect user output |
| Context cancellation | Direct — `ctx.Done()` select in `awaitWorkflowRun` | Cancellation behavior must be preserved |

### 2.3 Existing Test Coverage

The following tests exist in `internal/layers/enrollment_test.go`:

| Test | Covers |
|:-----|:-------|
| `TestEnrollmentLayer_Install_DispatchesWorkflow` | Happy path — dispatch + successful completion |
| `TestEnrollmentLayer_Install_ReportsEnrollmentPRs` | PR discovery after successful enrollment |
| `TestEnrollmentLayer_Install_ReportsRemovalPRs` | PR discovery for disabled repos |
| `TestEnrollmentLayer_Install_NoRepos` | Early return when no repos configured |
| `TestEnrollmentLayer_Install_DispatchError` | Dispatch failure error handling |
| `TestEnrollmentLayer_Install_WorkflowWarning` | Non-success workflow conclusion |
| `TestEnrollmentLayer_Install_ContextCancelled` | Context cancellation during wait |
| `TestBuildLayerStack_NilEnabledRepos_SkipsDisabledRepos` | Layer stack construction (in `admin_test.go`) |

---

## 3. Requirements Mapping

### 3.1 Validated Requirements

| Req ID | Requirement Summary | Source | Evidence | Priority |
|:-------|:-------------------|:-------|:---------|:---------|
| GH-2354 | Enrollment wait completes within bounded, predictable timeout | Regression analysis | `awaitWorkflowRun` polls with `enrollmentWaitTimeout` (3 min); callers `Install` and `Uninstall` both depend on this bound | P0 |
| | Timeout produces actionable error with guidance | Regression analysis | Timeout error at line 129-133 must include remediation steps (check workflow, re-run install) | P0 |
| | Progress indicators emitted during each polling phase | Regression analysis | `ui.StepInfo` at line 146 and 164 — user needs visibility into wait state | P1 |
| | Exponential backoff respects configured bounds | Regression analysis | `nextInterval` doubles from `enrollmentPollInitial` (2s) to `enrollmentPollMax` (15s) | P1 |
| | Context cancellation terminates wait immediately | Regression analysis | `ctx.Done()` select at line 137 — must not block beyond cancellation | P0 |
| | Uninstall wait shares same bounded behavior | Regression analysis | `Uninstall` calls `awaitWorkflowRun` at line 286 — same timeout applies | P1 |
| | Non-fatal timeout does not block install pipeline | Regression analysis | `Install` returns `nil` on timeout (line 101) — `InstallAll` must continue | P1 |
| | Workflow log retrieval on non-success conclusion | Regression analysis | `showWorkflowLogs` called at line 108 — diagnostic output on failure | P2 |

### 3.2 Rejected Requirements

| Requirement | Reason | Gate Failed |
|:------------|:-------|:------------|
| GitHub API rate limiting during polling | Platform-level — GitHub API rate limits are tested by GitHub | Requirement Level Validation |
| Workflow registration timing in GitHub Actions | Platform-level — GitHub Actions workflow registration is external | Requirement Level Validation |
| Repo-maintenance workflow script correctness | Separate component — tested by `scripts/reconcile-repos.sh` tests | Scope Boundary |

---

## 4. Test Scenarios

### 4.1 Timeout and Bounded Wait

| ID | Scenario | Steps | Expected Result | Priority |
|:---|:---------|:------|:----------------|:---------|
| TC-01 | Install completes within timeout on fast registration | Mock `ListWorkflowRuns` to return completed run after 2 polls | Install succeeds, output contains "enrollment completed successfully", total elapsed < `enrollmentWaitTimeout` | P0 |
| TC-02 | Install times out with actionable error on slow registration | Mock `ListWorkflowRuns` to return empty/error for duration exceeding `enrollmentWaitTimeout` | Install returns `nil` (non-fatal), output contains "timed out" message with guidance to "check the workflow in .fullsend and re-run install if needed" | P0 |
| TC-03 | Uninstall times out with same bounded behavior | Mock `ListWorkflowRuns` to never return completed run | Uninstall returns `nil` (non-fatal), output contains timeout warning, total elapsed ≤ `enrollmentWaitTimeout` + tolerance | P1 |
| TC-04 | Install respects context cancellation during wait | Cancel context after 1 second while `awaitWorkflowRun` is polling | Install returns `nil` (non-fatal), output contains cancellation warning, returns promptly after cancellation | P0 |

### 4.2 Exponential Backoff

| ID | Scenario | Steps | Expected Result | Priority |
|:---|:---------|:------|:----------------|:---------|
| TC-05 | Polling interval doubles from initial to max | Mock `ListWorkflowRuns` to return non-completed run, track poll intervals | Intervals follow 2s → 4s → 8s → 15s → 15s pattern (`enrollmentPollInitial` → `enrollmentPollMax`) | P1 |
| TC-06 | `nextInterval` caps at `enrollmentPollMax` | Call `nextInterval` with value ≥ `enrollmentPollMax` | Returns `enrollmentPollMax` (15s), never exceeds cap | P1 |
| TC-07 | `nextInterval` doubles sub-max values | Call `nextInterval(2s)`, `nextInterval(4s)`, `nextInterval(8s)` | Returns 4s, 8s, 15s (capped) respectively | P1 |

### 4.3 Progress Indicators

| ID | Scenario | Steps | Expected Result | Priority |
|:---|:---------|:------|:----------------|:---------|
| TC-08 | Progress messages emitted during workflow registration wait | Mock `ListWorkflowRuns` to return error (workflow not registered yet) | Output contains "waiting for workflow registration" with elapsed time | P1 |
| TC-09 | Progress messages emitted for in-progress workflow | Mock `ListWorkflowRuns` to return run with `status: "in_progress"` | Output contains workflow run URL, status, and elapsed time | P1 |
| TC-10 | No progress spam on immediate completion | Mock `ListWorkflowRuns` to return completed run on first poll | Output contains "enrollment completed successfully" without intermediate progress messages | P2 |

### 4.4 Happy Path (Regression Guard)

| ID | Scenario | Steps | Expected Result | Priority |
|:---|:---------|:------|:----------------|:---------|
| TC-11 | Successful enrollment with PR discovery | Mock successful dispatch + completed run + PRs on enabled repos | Output contains "dispatched", "enrollment completed successfully", and PR URLs for enrolled repos | P0 |
| TC-12 | Successful unenrollment with config update | Mock config read/write + successful dispatch + completed run | Config updated with all repos disabled, dispatch succeeds, output contains "Unenrollment completed" and PR URLs | P1 |
| TC-13 | No-op when no repos configured | Create layer with empty `enabledRepos` and `disabledRepos` | Output contains "no repositories to reconcile", no dispatch attempted | P1 |

### 4.5 Error Handling

| ID | Scenario | Steps | Expected Result | Priority |
|:---|:---------|:------|:----------------|:---------|
| TC-14 | Dispatch failure returns error | Mock `DispatchWorkflow` to return error | Install returns error wrapping "dispatching repo-maintenance", no polling attempted | P0 |
| TC-15 | Non-success workflow conclusion shows logs | Mock completed run with `conclusion: "failure"` + workflow logs | Output contains "completed with conclusion: failure" and workflow log content | P1 |
| TC-16 | Log fetch failure is non-fatal | Mock completed run with failure + `GetWorkflowRunLogs` returns error | Output contains conclusion warning, "could not fetch workflow logs" info, no panic | P2 |
| TC-17 | Workflow run with unparseable `CreatedAt` is skipped | Mock run with invalid `CreatedAt` timestamp | Run is skipped, polling continues to next interval | P2 |

### 4.6 Layer Stack Integration

| ID | Scenario | Steps | Expected Result | Priority |
|:---|:---------|:------|:----------------|:---------|
| TC-18 | `InstallAll` continues after enrollment timeout | Build stack with enrollment layer + subsequent layers, mock enrollment timeout | Enrollment emits warning (non-fatal), subsequent layers execute normally | P1 |
| TC-19 | `InstallAll` stops on enrollment dispatch error | Build stack with enrollment layer, mock dispatch error | `InstallAll` returns error with "layer enrollment:" prefix, subsequent layers skipped | P1 |

---

## 5. Test Classification

### 5.1 Unit Tests

Tests targeting individual functions with mocked dependencies.

| Test ID | Target Function | Mock Surface |
|:--------|:---------------|:-------------|
| TC-05 | `nextInterval` | None (pure function) |
| TC-06 | `nextInterval` | None (pure function) |
| TC-07 | `nextInterval` | None (pure function) |
| TC-01 | `awaitWorkflowRun` | `forge.FakeClient` |
| TC-02 | `awaitWorkflowRun` | `forge.FakeClient` |
| TC-04 | `awaitWorkflowRun` | `forge.FakeClient` + context |
| TC-08 | `awaitWorkflowRun` | `forge.FakeClient` + `ui.Printer` buffer |
| TC-09 | `awaitWorkflowRun` | `forge.FakeClient` + `ui.Printer` buffer |
| TC-10 | `awaitWorkflowRun` | `forge.FakeClient` |
| TC-17 | `awaitWorkflowRun` | `forge.FakeClient` |

### 5.2 Functional Tests

Tests targeting method-level behavior with mocked forge client.

| Test ID | Target Method | Mock Surface |
|:--------|:-------------|:-------------|
| TC-03 | `Uninstall` | `forge.FakeClient` |
| TC-11 | `Install` | `forge.FakeClient` with workflow runs + PRs |
| TC-12 | `Uninstall` | `forge.FakeClient` with config + workflow runs + PRs |
| TC-13 | `Install` | `forge.FakeClient` (minimal) |
| TC-14 | `Install` | `forge.FakeClient` with dispatch error |
| TC-15 | `Install` | `forge.FakeClient` with failed run + logs |
| TC-16 | `Install` | `forge.FakeClient` with failed run + log error |
| TC-18 | `InstallAll` | `forge.FakeClient` + layer stack |
| TC-19 | `InstallAll` | `forge.FakeClient` + layer stack |

---

## 6. Test Environment

| Component | Details |
|:----------|:--------|
| Language | Go |
| Test Framework | `testing` (stdlib) |
| Assertion Library | `github.com/stretchr/testify` (`assert`, `require`) |
| Mock Client | `forge.FakeClient` (in-repo fake at `internal/forge/fake.go`) |
| UI Capture | `bytes.Buffer` via `ui.New(&buf)` |
| Package Convention | Same-package tests (`package layers`) |
| Test File | `internal/layers/enrollment_test.go` |

---

## 7. Key Constants Under Test

| Constant | Value | Purpose |
|:---------|:------|:--------|
| `enrollmentWaitTimeout` | 3 min | Maximum time to wait for workflow run |
| `enrollmentPollInitial` | 2 sec | Initial polling interval |
| `enrollmentPollMax` | 15 sec | Maximum polling interval (backoff cap) |
| `repoMaintenanceWorkflow` | `repo-maintenance.yml` | Workflow file dispatched for enrollment |
| `shimWorkflowPath` | `.github/workflows/fullsend.yaml` | Shim workflow checked during analyze |

---

## 8. Coverage Summary

| Category | Count |
|:---------|:------|
| Total test scenarios | 19 |
| P0 (Critical) | 5 |
| P1 (Major) | 10 |
| P2 (Minor) | 4 |
| Unit tests | 10 |
| Functional tests | 9 |
| Requirements validated | 8 |
| Requirements rejected | 3 |

---

*Generated by QualityFlow STP Builder — 2026-06-21*
