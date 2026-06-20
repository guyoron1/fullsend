# My-Project Test Plan

## **Migrate Agent Slug Discovery to Harness-First Model - Quality Engineering Plan**

### Metadata & Tracking

- **Enhancement:** [fullsend-ai/fullsend#2361](https://github.com/fullsend-ai/fullsend/pull/2361)
- **Feature Tracking:** N/A — no separate feature tracking issue exists for this refactoring
- **Epic Tracking:** N/A — no separate epic tracking issue exists for this refactoring
- **QE Owner:** Unassigned
- **Owning SIG:** N/A
- **Participating SIGs:** N/A

**Document Conventions:** Standard QE test plan conventions apply. Test IDs follow the format TS-GH-49-NNN.

### Feature Overview

This feature migrates agent slug discovery in the admin CLI from a legacy file-based lookup (reading agent slugs from `config.yaml` `agents:` block) to a harness-first agent discovery model. The new implementation scans harness wrapper files in the config repo for role/slug fields, preferring them over the legacy source. When harness discovery returns no valid agents, the system gracefully falls back to the legacy `config.yaml` path and logs a deprecation warning. This refactor affects the install setup flow, which is invoked from the install command, per-repo install, and GitHub org setup paths.

---

### Section I - Motivation & Requirements Review

#### I.1 - Requirement & User Story Review Checklist

- [ ] **Reviewed the relevant requirements.**
  - PR mirrors upstream fullsend-ai/fullsend#2361; requirement is to prefer harness wrapper files over legacy config.yaml for agent slug discovery.
  - Agent slug discovery function signature updated to accept config repository, reference, and printer parameters.

- [ ] **Confirmed clear user stories and understood. Understand the value and customer use cases.**
  - As a platform admin running `fullsend install`, agent slugs should be discovered from harness wrapper files automatically, without requiring manual config.yaml maintenance.
  - Deprecation path provides clear migration signal to teams still using legacy format.

- [ ] **Confirmed requirements are **testable and unambiguous**.**
  - All behaviors are testable via mock forge client — harness preference, fallback, warnings, duplicate handling, and error resilience are all deterministic.

- [ ] **Ensured acceptance criteria are **defined clearly**.**
  - Harness files with valid role+slug fields are used preferentially.
  - Legacy config.yaml is used as fallback when harness discovery yields no agents.
  - Deprecation warning is emitted when legacy path is exercised.
  - Entries with missing role or slug are skipped with a warning.
  - Duplicate roles keep the first occurrence.

- [ ] **Confirmed coverage for NFRs.**
  - No performance NFRs identified; function is called once during install setup.
  - Backward compatibility preserved via fallback to legacy path.

#### I.2 - Known Limitations

- The harness agent discovery function is referenced in the PR diff but not yet defined in the harness package in this fork — it is part of the upstream PR being mirrored. Tests use a mock forge client to simulate the remote discovery behavior.
- The function only reads top-level `role` and `slug` fields from harness files; base chain resolution is not performed.
- No cluster interaction is required — all operations use the forge client API to read remote file contents.

#### I.3 - Technology and Design Review

- [ ] **Developer handoff completed; design and implementation reviewed.**
  - PR adds 41 lines to `internal/cli/admin.go` and 188 lines of tests to `internal/cli/admin_test.go`.
  - New dependency on `internal/harness` package for agent discovery and agent info types.
  - QE kickoff aligned with upstream PR review cycle for fullsend-ai/fullsend#2361.

- [ ] **Identified technology challenges or new dependencies.**
  - Depends on harness agent discovery being available in the harness package (upstream dependency from fullsend-ai/fullsend#2361).
  - Tests use a mock forge client with configurable directory contents and file references for test mocking.

- [ ] **Test environment needs assessed.**
  - No cluster required; all tests run with mock forge client.

- [ ] **API extensions or changes reviewed.**
  - Agent slug discovery function signature changed: added config repository, reference, and printer parameters.
  - Original function preserved as a legacy variant with the original signature for backward compatibility.

- [ ] **Topology or special infrastructure needs identified.**
  - None; purely in-process function with mocked external dependencies.

---

### Section II - Test Planning

#### II.1 - Scope of Testing

This test plan covers agent slug discovery during `fullsend install`, validating that harness wrapper files are preferred over legacy `config.yaml` for determining which agents to install. Testing validates the harness-first discovery preference, legacy fallback behavior, warning/logging behavior, and error resilience.

**Testing Goals:**

- **P0:** Verify agent slugs are discovered from harness wrapper files when valid role and slug fields are present.
- **P0:** Verify graceful fallback to legacy `config.yaml` agents block when harness discovery yields no agents.
- **P1:** Verify deprecation warnings are logged when the legacy discovery path is used.
- **P1:** Verify entries with incomplete role or slug fields are handled correctly with appropriate warnings.
- **P1:** Verify duplicate role handling (first occurrence wins).
- **P2:** Verify resilience to partial read errors and malformed configuration.

**Out of Scope (Testing Scope Exclusions):**

- [ ] **Upstream harness.DiscoverRemoteAgents implementation** -- Tested by upstream fullsend-ai/fullsend; this plan covers the integration point only.
- [ ] **Forge client network behavior** -- Platform-level concern; tests use mock forge client.
- [ ] **End-to-end install workflow** -- Full install flow is out of scope; focus is on slug discovery logic.
- [ ] **Harness file parsing (LoadRaw)** -- Covered by existing harness package tests.

#### II.2 - Test Strategy

**Functional:**

- [x] **Functional Testing** -- Verify agent slug discovery behavior across all discovery paths (harness-first, legacy fallback, error cases).
- [x] **Automation Testing** -- All scenarios implemented as Go unit tests using mock forge client.
- [x] **Regression Testing** -- Verify install, per-repo install, and GitHub setup callers continue to work with updated slug discovery.
- [ ] **Upgrade Testing** -- Not applicable; no persistent state migration.

**Non-Functional:**

- [ ] **Performance Testing** -- Not applicable; function called once per install.
- [ ] **Scale Testing** -- Not applicable; operates on small number of harness files.
- [ ] **Security Testing** -- Not applicable; no authentication or authorization changes.
- [ ] **Usability Testing** -- Not applicable; no user-facing UI changes.
- [ ] **Monitoring** -- Not applicable; no new metrics or observability changes.

**Integration & Compatibility:**

- [x] **Compatibility Testing** -- Verify backward compatibility: legacy config.yaml format continues to work via fallback.
- [x] **Dependencies** -- Depends on upstream fullsend-ai/fullsend#2361 being merged to make harness agent discovery available in the harness package.
- [ ] **Cross Integrations** -- Not applicable; changes are internal to admin CLI.

**Infrastructure:**

- [ ] **Cloud Testing** -- Not applicable; no cloud-specific behavior.

#### II.3 - Test Environment

- **Cluster Topology:** Not required; unit test execution only
- **Platform Version:** Go 1.22+ (per go.mod)
- **Compute:** Standard CI runner (Linux/macOS)
- **Special Infrastructure:** No special infrastructure required. Tests execute in-process with a mock forge client configured per test case to simulate harness file contents and discovery responses.

#### II.3.1 - Testing Tools & Frameworks

No new or special tools required. Standard Go testing with testify assertions.

#### II.4 - Entry Criteria

- [ ] Harness agent discovery function is available in the harness package (upstream PR merged)
- [ ] Mock forge client supports configurable directory contents and file references for test mocking
- [ ] PR branch compiles successfully with all dependencies resolved

#### II.5 - Risks

- [ ] **Timeline**
  - Risk: Upstream `harness.DiscoverRemoteAgents` may not be merged when this PR lands
  - Mitigation: PR is a mirror of upstream #2361; coordinate merge timing
  - Status: [ ] Open

- [ ] **Coverage**
  - Risk: Mock-based tests may not catch real forge client edge cases
  - Mitigation: 9 test cases cover all major paths; integration testing in CI validates real client
  - Status: [ ] Acceptable

- [ ] **Environment**
  - Risk: None identified; no cluster dependency
  - Mitigation: N/A
  - Status: [x] No risk

- [ ] **Untestable**
  - Risk: Real network errors from forge client cannot be unit tested
  - Mitigation: Mock forge client error map simulates hard errors; partial errors tested via missing file reference entries
  - Status: [ ] Mitigated

- [ ] **Resources**
  - Risk: None identified
  - Mitigation: N/A
  - Status: [x] No risk

- [ ] **Dependencies**
  - Risk: Depends on upstream harness package exporting `DiscoverRemoteAgents`
  - Mitigation: Function is defined in upstream PR #2361; this PR mirrors that change
  - Status: [ ] Open

- [ ] **Other**
  - Risk: None identified
  - Mitigation: N/A
  - Status: [x] No risk

---

### Section III - Requirements-to-Tests Mapping

#### III.1 - Requirements Mapping

- **GH-49** | Harness-first agent discovery is preferred over legacy config.yaml
  - TS-GH-49-001: Verify harness files with valid role+slug are used over config.yaml agents block | Functional | P0 | Unit
  - TS-GH-49-002: Verify config.yaml agents block is not consulted when harness discovery succeeds | Functional | P0 | Unit

- | Fallback to legacy config.yaml when harness discovery yields no agents
  - TS-GH-49-003: Verify fallback to config.yaml when no harness directory exists | Functional | P0 | Unit
  - TS-GH-49-004: Verify fallback to config.yaml agents block when harness files contain no role/slug fields | Functional | P1 | Unit
  - TS-GH-49-005: Verify nil returned when neither harness nor config.yaml provides agents | Functional | P1 | Unit

- | Deprecation warning emitted for legacy path usage
  - TS-GH-49-006: Verify deprecation warning logged when config.yaml agents block is used | Functional | P1 | Unit
  - TS-GH-49-007: Verify no deprecation warning when harness discovery succeeds | Functional | P1 | Unit

- | Incomplete harness entries handled with appropriate warnings
  - TS-GH-49-008: Verify entry with role but no slug is skipped and a warning is logged to the printer output | Functional | P1 | Unit
  - TS-GH-49-009: Verify entry with empty role and empty slug is silently skipped (no output produced) | Functional | P2 | Unit

- | Duplicate role handling preserves deterministic behavior
  - TS-GH-49-010: Verify duplicate roles keep first occurrence (sorted by Role then Filename) | Functional | P1 | Unit
  - TS-GH-49-011: Verify info message logged for duplicate role detection | Functional | P2 | Unit

- | Error resilience in harness discovery
  - TS-GH-49-012: Verify partial read errors still return successfully parsed agents | Functional | P1 | Unit
  - TS-GH-49-013: Verify hard discovery error falls back to legacy config.yaml path | Functional | P1 | Unit
  - TS-GH-49-014: Verify warning logged when harness discovery encounters errors | Functional | P2 | Unit

- | Malformed configuration handling
  - TS-GH-49-015: Verify malformed config.yaml returns nil without panic | Functional | P2 | Unit

- | Integration with install setup call chain
  - TS-GH-49-016: Verify install setup uses harness-discovered agent slugs when initiating app configuration | Functional | P0 | Unit
  - TS-GH-49-017: Verify agent slug filtering by app-set works correctly with harness-discovered slugs | Functional | P1 | Unit

---

### Section IV - Sign-off

| Role | Name | Date |
|:-----|:-----|:-----|
| QE Lead | | |
| Dev Lead | | |
| PM | | |
