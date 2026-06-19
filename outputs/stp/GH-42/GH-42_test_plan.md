# My-Project Test Plan

## **Remote Harness Agent Discovery via Forge API - Quality Engineering Plan**

### Metadata & Tracking

- **Enhancement:** [GH-42](https://github.com/guyoron1/fullsend/pull/42)
- **Feature Tracking:** [GH-42](https://github.com/guyoron1/fullsend/pull/42) — feat(harness): add remote harness agent discovery via forge API
- **Epic Tracking:** N/A
- **QE Owner:** Unassigned
- **Owning SIG:** N/A
- **Participating SIGs:** N/A

**Document Conventions:** Standard QualityFlow STP conventions apply. Test IDs use the format TS-GH-42-NNN.

### Feature Overview

This feature adds remote agent discovery to the fullsend harness subsystem, enabling the harness to find agents deployed in remote config repositories via the forge API. The new `DiscoverRemoteAgents` function mirrors the existing local `DiscoverAgents` function but reads harness YAML files from a remote repository using `forge.Client.ListDirectoryContents` and `forge.Client.GetFileContentAtRef`. The implementation includes a refactoring of `LoadRaw` to extract a shared `parseRaw` helper function that both local and remote discovery paths use for YAML unmarshalling.

---

### Section I: Motivation & Requirements

#### I.1 - Requirement & User Story Review Checklist

- [ ] **Reviewed the relevant requirements.** -- PR description and upstream issue reference reviewed.
  - GH-42 mirrors upstream fullsend-ai/fullsend#2327. The requirement is to discover agent identity (role, slug) from harness files in remote config repos via the forge API.
- [ ] **Confirmed clear user stories and understood. Understand the value and customer use cases.** -- User value assessed.
  - Enables harness to discover agents deployed outside the local repository, supporting distributed agent configuration workflows.
- [ ] **Confirmed requirements are **testable and unambiguous**.** -- Testability assessed.
  - Function signature and behavior are well-defined. Comprehensive unit tests (15 cases) are included in the PR. Functional behavior is deterministic (sorted output, clear error semantics).
- [ ] **Ensured acceptance criteria are **defined clearly**.** -- Acceptance criteria reviewed.
  - Implicit acceptance criteria derived from implementation: returns sorted agents, skips empty role+slug, collects per-file errors into multi-error, returns nil/nil for missing directory.
- [ ] **Confirmed coverage for NFRs.** -- Non-functional requirements reviewed.
  - Performance: sequential file fetches via forge API; no parallelism requirement identified. Reliability: partial failure returns valid results alongside multi-error.

#### I.2 - Known Limitations

- Remote discovery does not resolve base chains or validate harness files — it only extracts role and slug identity fields.
- The `Path` field in `AgentInfo` is always empty for remotely discovered agents (no local filesystem path exists).
- File fetches from the forge API are sequential; large harness directories may have higher latency compared to local discovery.

#### I.3 - Technology and Design Review

- [ ] **Developer Handoff** -- Implementation details reviewed.
  - Reviewed PR diff: 1 new file (`discover_remote.go`, 76 lines), 1 modified file (`harness.go`, refactored `LoadRaw` to use new `parseRaw`), 1 new test file (226 lines, 15 test cases).
- [ ] **Technology Challenges** -- Technical risks identified.
  - Depends on `forge.Client` interface methods (`ListDirectoryContents`, `GetFileContentAtRef`). A `FakeClient` is used for testing, avoiding external dependencies.
- [ ] **Test Environment Needs** -- Environment requirements assessed.
  - Unit tests only require Go test runner with mocked forge client. No cluster or external service needed.
- [ ] **API Extensions** -- API surface changes reviewed.
  - New exported function `DiscoverRemoteAgents` added to `internal/harness` package. New unexported helper `parseRaw` extracted from `LoadRaw` — no breaking API change.
- [ ] **Topology** -- Deployment topology assessed.
  - No topology changes. Remote discovery is invoked at harness resolution time, before sandbox creation.

### Section II: Test Planning

#### II.1 - Scope of Testing

This test plan covers the new `DiscoverRemoteAgents` function and the `parseRaw` refactoring of `LoadRaw`. Testing validates correct agent discovery from remote repositories, error handling for partial failures, file filtering logic, deterministic sort ordering, and backward compatibility of the `LoadRaw` refactoring.

**Testing Goals:**

- **P0:** Verify remote agent discovery returns correct agent identity from valid harness files
- **P0:** Verify `parseRaw` refactoring does not break existing `LoadRaw` callers
- **P1:** Verify partial failure error handling (valid agents returned alongside multi-error)
- **P1:** Verify file filtering (YAML only, no directories, no non-YAML files)
- **P1:** Verify deterministic sort order (by Role, then Filename)
- **P2:** Verify graceful handling of missing harness directory (nil, nil return)

**Out of Scope (Testing Scope Exclusions):**

- [ ] **Forge API client implementation** -- Forge API transport and authentication are tested by the `internal/forge` package, not by this feature.
- [ ] **Base chain resolution for remote harnesses** -- Remote discovery intentionally skips base resolution; this is a known limitation, not a test gap.
- [ ] **Local agent discovery (`DiscoverAgents`)** -- Existing function with its own test suite; only regression impact of shared `parseRaw` is in scope.
- [ ] **End-to-end forge API integration** -- Remote API calls are mocked via `FakeClient`; live forge integration is out of scope for this plan.

#### II.2 - Test Strategy

**Functional:**

- [x] **Functional Testing** -- Applicable.
  - Verify `DiscoverRemoteAgents` returns correct agents for valid harness files with role and slug fields. Verify filtering, sorting, and error collection behavior.
- [x] **Automation Testing** -- Applicable.
  - All tests are automated Go unit tests using `testify/assert` and `testify/require` with `forge.FakeClient`.
- [x] **Regression Testing** -- Applicable.
  - Verify `LoadRaw` continues to work correctly after `parseRaw` extraction. LSP analysis confirms `LoadRaw` is called by 8 callers across `cli/lock.go`, `cli/run.go`, `harness/compose.go`, `harness/discover.go`, and `harness/harness.go`.

**Non-Functional:**

- [ ] **Performance Testing** -- Not applicable for this feature scope.
- [ ] **Scale Testing** -- Not applicable; remote discovery processes files sequentially.
- [ ] **Security Testing** -- Not applicable; no new auth or permission surfaces introduced.
- [ ] **Usability Testing** -- Not applicable; internal API only.
- [ ] **Monitoring** -- Not applicable; no new observability surfaces.

**Integration & Compatibility:**

- [ ] **Compatibility Testing** -- Not applicable; no version-dependent behavior.
- [ ] **Upgrade Testing** -- Not applicable; no persisted state or migration paths.
- [x] **Dependencies** -- Applicable.
  - Depends on `forge.Client` interface. Tests use `forge.NewFakeClient()` to mock dependencies.
- [ ] **Cross Integrations** -- Not applicable for initial feature scope.

**Infrastructure:**

- [ ] **Cloud Testing** -- Not applicable; feature is platform-agnostic.

#### II.3 - Test Environment

- **Cluster Topology:** Not required — unit tests only
- **Platform Version:** Go 1.22+ (per go.mod)
- **CPU Virtualization:** N/A
- **Compute:** Standard CI runner
- **Special Hardware:** None
- **Storage:** N/A
- **Network:** N/A (forge API is mocked)
- **Operators:** None
- **Platform:** Linux (CI environment)
- **Special Configs:** None

#### II.3.1 - Testing Tools & Frameworks

No new or special tools required. Standard Go test runner with `testify` assertions and `forge.FakeClient` mock.

#### II.4 - Entry Criteria

- [ ] PR #42 is merged to main branch
- [ ] `go test ./internal/harness/...` passes with no failures
- [ ] `parseRaw` refactoring does not introduce regressions in existing `LoadRaw` callers

#### II.5 - Risks

- [ ] **Timeline**
  - Risk: Feature is mirrored from upstream; upstream changes may diverge from this PR.
  - Mitigation: Track upstream fullsend-ai/fullsend#2327 for changes.
  - Status: [ ] Open
- [ ] **Coverage**
  - Risk: Remote discovery only tests with `FakeClient`; real forge API behavior may differ.
  - Mitigation: `FakeClient` implements the same `forge.Client` interface; integration tests in upstream repo cover real API.
  - Status: [ ] Open
- [ ] **Environment**
  - Risk: None identified — tests run in standard Go test environment.
  - Mitigation: N/A
  - Status: [x] Resolved
- [ ] **Untestable**
  - Risk: Live forge API latency and rate limiting cannot be tested in unit tests.
  - Mitigation: Accepted limitation; covered by upstream integration tests.
  - Status: [ ] Open
- [ ] **Resources**
  - Risk: None identified.
  - Mitigation: N/A
  - Status: [x] Resolved
- [ ] **Dependencies**
  - Risk: `forge.Client` interface may change, breaking `DiscoverRemoteAgents` signature.
  - Mitigation: Interface is defined in the same repository; compile-time checks catch breakage.
  - Status: [ ] Open
- [ ] **Other**
  - Risk: None identified.
  - Mitigation: N/A
  - Status: [x] Resolved

---

### Section III: Requirements-to-Tests Mapping

#### III.1 - Requirements Mapping

- **Requirement ID:** GH-42
- **Requirement Summary:** Remote agent discovery returns correct agent identity from valid harness files
- **Test Scenarios:**
  - Verify discovery returns agents with correct role, slug, and filename (positive)
  - Verify discovery returns agents sorted by role then filename (positive)
  - Verify error when forge API returns invalid YAML (negative)
- **Tier:** Functional
- **Priority:** P0

- **Requirement ID:**
- **Requirement Summary:** Remote discovery handles missing harness directory gracefully
- **Test Scenarios:**
  - Verify nil agents and nil error returned when directory not found (positive)
  - Verify ListDirectoryContents error propagates with context (negative)
- **Tier:** Functional
- **Priority:** P0

- **Requirement ID:**
- **Requirement Summary:** Remote discovery filters files correctly
- **Test Scenarios:**
  - Verify only .yaml and .yml files are processed (positive)
  - Verify subdirectories are skipped (positive)
  - Verify non-YAML files are skipped (positive)
  - Verify files with empty role and slug are skipped (positive)
- **Tier:** Functional
- **Priority:** P1

- **Requirement ID:**
- **Requirement Summary:** Remote discovery handles partial failures with multi-error
- **Test Scenarios:**
  - Verify valid agents returned alongside multi-error for malformed files (positive)
  - Verify GetFileContentAtRef failure for one file does not block others (positive)
  - Verify error message identifies the failing filename (negative)
- **Tier:** Functional
- **Priority:** P1

- **Requirement ID:**
- **Requirement Summary:** Agent identity fields are correctly extracted from remote harness files
- **Test Scenarios:**
  - Verify agent with role only (no slug) is included (positive)
  - Verify agent with slug only (no role) is included (positive)
  - Verify Path field is empty for remote agents (positive)
  - Verify path prefix in directory entry is stripped to bare filename (positive)
- **Tier:** Functional
- **Priority:** P1

- **Requirement ID:**
- **Requirement Summary:** parseRaw refactoring preserves LoadRaw backward compatibility
- **Test Scenarios:**
  - Verify LoadRaw returns unvalidated harness (regression)
  - Verify LoadRaw preserves forge map (regression)
  - Verify LoadRaw returns error for missing file (regression)
  - Verify all existing LoadRaw callers compile without changes (regression)
- **Tier:** Functional
- **Priority:** P0

- **Requirement ID:**
- **Requirement Summary:** Remote discovery integrates correctly with forge.Client interface
- **Test Scenarios:**
  - Verify discovery works end-to-end with FakeClient mock (positive)
  - Verify behavior with empty harness directory (edge case)
  - Verify concurrent discovery calls do not interfere (negative)
- **Tier:** Functional
- **Priority:** P1

---

### Section IV: Sign-off

| Role | Name | Date | Signature |
|:-----|:-----|:-----|:----------|
| QE Lead | | | |
| Dev Lead | | | |
| PM | | | |
