# My-Project Test Plan

## **Remote Harness Agent Discovery via Forge API - Quality Engineering Plan**

### Metadata & Tracking

- **Enhancement:** [GH-42](https://github.com/fullsend-ai/fullsend/pull/42)
- **Feature Tracking:** [GH-42](https://github.com/fullsend-ai/fullsend/pull/42) — feat(harness): add remote harness agent discovery via forge API
- **Epic Tracking:** N/A
- **QE Owner:** Unassigned
- **Owning SIG:** N/A
- **Participating SIGs:** N/A

**Document Conventions:** Standard QualityFlow STP conventions apply. Test IDs use the format TS-GH-42-NNN.

### Feature Overview

This feature adds remote agent discovery to the fullsend harness subsystem, enabling the harness to find agents deployed in remote config repositories via the forge API. The new remote discovery capability mirrors the existing local agent discovery but reads harness YAML files from a remote repository using the forge API client. The implementation includes a refactoring of the harness file loading path to share YAML parsing logic between local and remote discovery. For full implementation details, see PR #42.

---

### Section I: Motivation & Requirements

#### I.1 - Requirement & User Story Review Checklist

- [ ] **Reviewed the relevant requirements.** -- PR description and upstream issue reference reviewed.
  - GH-42 mirrors upstream [fullsend-ai/fullsend#2327](https://github.com/fullsend-ai/fullsend/pull/2327). The requirement is to discover agent identity (role, slug) from harness files in remote config repos via the forge API.
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
  - PR review served as QE kickoff for this small-scope feature. Design, architecture, and implementation reviewed via PR #42. The PR introduces one new source file for remote discovery, one modified file for shared parsing logic, and one new test file with 15 test cases.
- [ ] **Technology Challenges** -- Technical risks identified.
  - Remote discovery depends on the forge API client interface. A fake client implementation is used for testing, avoiding external service dependencies. The forge client interface may evolve, requiring test updates (see Risks II.5).
- [ ] **Test Environment Needs** -- Environment requirements assessed.
  - Unit tests only require Go test runner with mocked forge client. No cluster or external service needed.
- [ ] **API Extensions** -- API surface changes reviewed.
  - New exported function `DiscoverRemoteAgents` added to `internal/harness` package. New unexported helper `parseRaw` extracted from `LoadRaw` — no breaking API change.
- [ ] **Topology** -- Deployment topology assessed.
  - No topology changes. Remote discovery is invoked at harness resolution time, before sandbox creation.

### Section II: Test Planning

#### II.1 - Scope of Testing

This test plan covers remote agent discovery from external config repositories and backward compatibility of the harness file loading refactoring. Testing validates correct agent discovery from remote repositories, error handling for partial failures, file filtering logic, deterministic sort ordering, and backward compatibility of the harness file loading interface.

**Testing Goals:**

- **P0:** Verify remote agent discovery returns correct agent identity from valid harness files
- **P0:** Verify harness file loading refactoring does not break existing consumers
- **P1:** Verify partial failure error handling (valid agents returned alongside aggregated errors)
- **P1:** Verify file filtering (YAML only, no directories, no non-YAML files)
- **P1:** Verify deterministic sort order (by Role, then Filename)
- **P2:** Verify graceful handling of missing harness directory (empty result, no error)

**Out of Scope (Testing Scope Exclusions):**

- [ ] **Forge API client implementation** -- Forge API transport and authentication are tested by the `internal/forge` package, not by this feature.
- [ ] **Base chain resolution for remote harnesses** -- Remote discovery intentionally skips base resolution; this is a known limitation, not a test gap.
- [ ] **Local agent discovery** -- Existing local discovery function has its own test suite; only regression impact of the shared parsing refactoring is in scope.
- [ ] **End-to-end forge API integration** -- Remote API calls are mocked via a fake client; live forge integration is out of scope for this plan.

#### II.2 - Test Strategy

**Functional:**

- [x] **Functional Testing** -- Applicable.
  - Verify remote agent discovery returns correct agents for valid harness files with role and slug fields. Verify filtering, sorting, and error collection behavior.
- [x] **Automation Testing** -- Applicable.
  - All tests are automated Go unit tests using standard assertion libraries with a fake forge client.
- [x] **Regression Testing** -- Applicable.
  - Verify harness file loading continues to work correctly after the shared parsing refactoring. LSP analysis confirms the file loading interface is consumed by 8 callers across the CLI and harness packages.

**Non-Functional:**

- [ ] **Performance Testing** -- Not applicable for this feature scope.
- [ ] **Scale Testing** -- Not applicable; remote discovery processes files sequentially.
- [ ] **Security Testing** -- Not applicable; no new auth or permission surfaces introduced.
- [ ] **Usability Testing** -- Not applicable; internal API only.
- [ ] **Monitoring** -- Not applicable; no new observability surfaces.

**Integration & Compatibility:**

- [ ] **Compatibility Testing** -- Not applicable; no version-dependent behavior.
- [ ] **Upgrade Testing** -- Not applicable; no persisted state or migration paths.
- [ ] **Dependencies** -- Not applicable. No team delivery blockers identified. The forge API client interface is a code-level dependency, not a cross-team delivery gate. Tests are fully self-contained using a fake client implementation. See Technology Challenges (I.3) for technical dependency details.
- [ ] **Cross Integrations** -- Not applicable for initial feature scope.

**Infrastructure:**

- [ ] **Cloud Testing** -- Not applicable; feature is platform-agnostic.

#### II.3 - Test Environment

- **Cluster Topology:** Not required — unit tests only, no cluster interaction
- **Platform Version:** Go 1.22+ (per go.mod)
- **CPU Virtualization:** N/A — unit tests only, no VM operations
- **Compute:** Standard CI runner — no special compute requirements for unit tests
- **Special Hardware:** None — pure software logic with no hardware dependencies
- **Storage:** N/A — no persistent storage operations; all data is in-memory
- **Network:** N/A — forge API is mocked; no real network calls in test scope
- **Operators:** None — feature operates at library level, no operator interaction
- **Platform:** Linux (CI environment)
- **Special Configs:** None — default Go test environment is sufficient

#### II.3.1 - Testing Tools & Frameworks

No new or special tools required. Standard Go test runner with `testify` assertions and `forge.FakeClient` mock.

#### II.4 - Entry Criteria

- [ ] PR #42 is merged to main branch
- [ ] `go test ./internal/harness/...` passes with no failures
- [ ] Harness file loading refactoring does not introduce regressions in existing consumers

#### II.5 - Risks

- [ ] **Timeline**
  - Risk: Feature is mirrored from upstream; upstream changes may diverge from this PR.
  - Mitigation: Track upstream [fullsend-ai/fullsend#2327](https://github.com/fullsend-ai/fullsend/pull/2327) for changes.
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

- **Requirement ID:** GH-42-01
- **Requirement Summary:** As a harness consumer, I want remote agent discovery so that agents in external config repositories are available for resolution with correct identity fields.
- **Test Scenarios:**
  - Verify discovery returns agents with correct role, slug, and filename (positive)
  - Verify discovery returns agents sorted by role then filename (positive)
  - Verify error when forge API returns invalid YAML (negative)
- **Tier:** Functional
- **Priority:** P0

- **Requirement ID:** GH-42-02
- **Requirement Summary:** As a harness consumer, I want remote discovery to handle missing directories gracefully so that the system does not fail when a harness directory is absent.
- **Test Scenarios:**
  - Verify empty result and no error returned when directory not found (positive)
  - Verify directory listing errors propagate with context (negative)
- **Tier:** Functional
- **Priority:** P0

- **Requirement ID:** GH-42-03
- **Requirement Summary:** As a harness consumer, I want remote discovery to process only valid harness files so that non-harness content is excluded from results.
- **Test Scenarios:**
  - Verify only .yaml and .yml files are processed (positive)
  - Verify subdirectories are skipped (positive)
  - Verify non-YAML files are skipped (positive)
  - Verify files with empty role and slug are skipped (positive)
- **Tier:** Functional
- **Priority:** P1

- **Requirement ID:** GH-42-04
- **Requirement Summary:** As a harness consumer, I want remote discovery to return valid results alongside errors so that partial failures do not discard successfully discovered agents.
- **Test Scenarios:**
  - Verify valid agents returned alongside aggregated errors for malformed files (positive)
  - Verify single-file fetch failure does not block other file processing (positive)
  - Verify error messages identify the failing filename (negative)
- **Tier:** Functional
- **Priority:** P1

- **Requirement ID:** GH-42-05
- **Requirement Summary:** As a harness consumer, I want agent identity fields to be extracted accurately from remote harness files so that role and slug values match the source YAML.
- **Test Scenarios:**
  - Verify agent with role only (no slug) is included (positive)
  - Verify agent with slug only (no role) is included (positive)
  - Verify path field is empty for remote agents (positive)
  - Verify path prefix in directory entry is stripped to bare filename (positive)
- **Tier:** Functional
- **Priority:** P1

- **Requirement ID:** GH-42-06
- **Requirement Summary:** As a harness API consumer, I want the file loading interface to remain unchanged after internal refactoring so that existing callers continue to function without modification.
- **Test Scenarios:**
  - Verify harness file loading returns expected unvalidated structure (regression)
  - Verify harness file loading preserves configuration mappings (regression)
  - Verify harness file loading reports errors for invalid paths (regression)
  - Verify all existing harness consumers continue to compile and function (regression)
- **Tier:** Functional
- **Priority:** P0

- **Requirement ID:** GH-42-07
- **Requirement Summary:** As a harness consumer, I want remote discovery to integrate reliably with the forge API client so that discovery works correctly in end-to-end workflows.
- **Test Scenarios:**
  - Verify discovery works end-to-end with fake forge client (positive)
  - Verify behavior with empty harness directory (edge case)
  - Verify concurrent discovery calls do not interfere (negative)
- **Tier:** Functional
- **Priority:** P2

---

### Section IV: Sign-off

* **Reviewers:** [Unassigned]
* **Approvers:** [Unassigned]
* **Date:** 2026-06-19
* **Status:** Draft — pending review
