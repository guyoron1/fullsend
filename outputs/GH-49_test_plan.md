# My-Project Test Plan

## **Migrate loadKnownSlugs to Harness-First Discovery - Quality Engineering Plan**

### Metadata & Tracking

- **Enhancement:** [GH-49](https://github.com/guyoron1/fullsend/pull/49)
- **Feature Tracking:** [GH-49](https://github.com/guyoron1/fullsend/pull/49)
- **Epic Tracking:** [GH-49](https://github.com/guyoron1/fullsend/pull/49)
- **QE Owner:** Unassigned
- **Owning SIG:** N/A
- **Participating SIGs:** N/A

**Document Conventions:** Standard QE test plan conventions apply. Test IDs follow the format TS-GH-49-NNN.

### Feature Overview

This feature migrates the `loadKnownSlugs` function in the admin CLI from a legacy file-based lookup (reading agent slugs from `config.yaml` `agents:` block) to a harness-first agent discovery model using `harness.DiscoverRemoteAgents`. The new implementation scans harness wrapper files in the config repo for role/slug fields, preferring them over the legacy source. When harness discovery returns no valid agents, the function gracefully falls back to the legacy `config.yaml` path and logs a deprecation warning. This refactor affects the `runAppSetup` call chain, which is invoked from `newInstallCmd`, `runPerRepoInstall`, and `runGitHubSetupPerOrg`.

---

### Section I - Motivation & Requirements Review

#### I.1 - Requirement & User Story Review Checklist

- [ ] **Reviewed the relevant requirements.**
  - PR mirrors upstream fullsend-ai/fullsend#2361; requirement is to prefer harness wrapper files over legacy config.yaml for agent slug discovery.
  - `loadKnownSlugs` signature changed to accept `configRepo`, `ref`, and `printer` parameters.

- [ ] **Confirmed clear user stories and understood. Understand the value and customer use cases.**
  - As a platform admin running `fullsend install`, agent slugs should be discovered from harness wrapper files automatically, without requiring manual config.yaml maintenance.
  - Deprecation path provides clear migration signal to teams still using legacy format.

- [ ] **Confirmed requirements are **testable and unambiguous**.**
  - All behaviors are testable via mock `forge.Client` — harness preference, fallback, warnings, duplicate handling, and error resilience are all deterministic.

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

- `harness.DiscoverRemoteAgents` is referenced in the PR diff but not yet defined in the harness package in this fork — it is part of the upstream PR being mirrored. Tests use `forge.FakeClient` to simulate the remote discovery behavior.
- The function only reads top-level `role` and `slug` fields from harness files; base chain resolution is not performed.
- No cluster interaction is required — all operations use the forge client API to read remote file contents.

#### I.3 - Technology and Design Review

- [ ] **Developer handoff completed; design and implementation reviewed.**
  - PR adds 41 lines to `internal/cli/admin.go` and 188 lines of tests to `internal/cli/admin_test.go`.
  - New dependency on `internal/harness` package for `DiscoverRemoteAgents` and `AgentInfo` type.

- [ ] **Identified technology challenges or new dependencies.**
  - Depends on `harness.DiscoverRemoteAgents` which must be available in the harness package (upstream dependency).
  - Uses `forge.FakeClient` with `DirContents` and `FileContentsRef` maps for test mocking.

- [ ] **Test environment needs assessed.**
  - No cluster required; all tests run with mock forge client.

- [ ] **API extensions or changes reviewed.**
  - `loadKnownSlugs` function signature changed: added `configRepo`, `ref`, and `printer` parameters.
  - Original function renamed to `loadKnownSlugsLegacy` with original signature preserved.

- [ ] **Topology or special infrastructure needs identified.**
  - None; purely in-process function with mocked external dependencies.

---

### Section II - Test Planning

#### II.1 - Scope of Testing

This test plan covers the refactored `loadKnownSlugs` function and its integration with the `runAppSetup` call chain. Testing validates the harness-first discovery preference, legacy fallback behavior, warning/logging behavior, and error resilience.

**Testing Goals:**

- **P0:** Verify harness-first discovery returns correct slugs when harness files contain valid role+slug fields.
- **P0:** Verify graceful fallback to legacy config.yaml when harness discovery yields no agents.
- **P1:** Verify deprecation warnings are logged when legacy path is used.
- **P1:** Verify entries with incomplete role/slug fields are handled correctly with appropriate warnings.
- **P1:** Verify duplicate role handling (first occurrence wins).
- **P2:** Verify resilience to partial read errors and malformed configuration.

**Out of Scope (Testing Scope Exclusions):**

- [ ] **Upstream harness.DiscoverRemoteAgents implementation** -- Tested by upstream fullsend-ai/fullsend; this plan covers the integration point only.
- [ ] **Forge client network behavior** -- Platform-level concern; tests use mock forge client.
- [ ] **End-to-end install workflow** -- Full install flow is out of scope; focus is on slug discovery logic.
- [ ] **Harness file parsing (LoadRaw)** -- Covered by existing harness package tests.

#### II.2 - Test Strategy

**Functional:**

- [x] **Functional Testing** -- Verify loadKnownSlugs behavior across all discovery paths (harness-first, legacy fallback, error cases).
- [x] **Automation Testing** -- All scenarios implemented as Go unit tests using forge.FakeClient mocks.
- [x] **Regression Testing** -- Verify callers (runAppSetup from newInstallCmd, runPerRepoInstall, runGitHubSetupPerOrg) continue to work with updated function signature.
- [ ] **Upgrade Testing** -- Not applicable; no persistent state migration.

**Non-Functional:**

- [ ] **Performance Testing** -- Not applicable; function called once per install.
- [ ] **Scale Testing** -- Not applicable; operates on small number of harness files.
- [ ] **Security Testing** -- Not applicable; no authentication or authorization changes.
- [ ] **Usability Testing** -- Not applicable; no user-facing UI changes.
- [ ] **Monitoring** -- Not applicable; no new metrics or observability changes.

**Integration & Compatibility:**

- [x] **Compatibility Testing** -- Verify backward compatibility: legacy config.yaml format continues to work via fallback.
- [x] **Dependencies** -- Verify integration with harness.DiscoverRemoteAgents and forge.Client interfaces.
- [ ] **Cross Integrations** -- Not applicable; changes are internal to admin CLI.

**Infrastructure:**

- [ ] **Cloud Testing** -- Not applicable; no cloud-specific behavior.

#### II.3 - Test Environment

- **Cluster Topology:** Not required; unit test execution only
- **Platform Version:** Go 1.22+ (per go.mod)
- **CPU Virtualization:** Not applicable
- **Compute:** Standard CI runner
- **Special Hardware:** None
- **Storage:** Not applicable
- **Network:** Not applicable (mock forge client)
- **Operators:** None
- **Platform:** Linux/macOS CI environment
- **Special Configs:** forge.FakeClient with DirContents and FileContentsRef maps configured per test case

#### II.3.1 - Testing Tools & Frameworks

No new or special tools required. Standard Go testing with testify assertions.

#### II.4 - Entry Criteria

- [ ] `harness.DiscoverRemoteAgents` function is available in the harness package
- [ ] `forge.FakeClient` supports `DirContents` and `FileContentsRef` maps for test mocking
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
  - Mitigation: `forge.FakeClient.Errors` map simulates hard errors; partial errors tested via missing FileContentsRef entries
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
  - TS-GH-49-001: Verify harness files with valid role+slug are used over config.yaml agents block | Functional | P0
  - TS-GH-49-002: Verify config.yaml agents block is not consulted when harness discovery succeeds | Functional | P0

- | Fallback to legacy config.yaml when harness discovery yields no agents
  - TS-GH-49-003: Verify fallback to config.yaml when no harness directory exists | Functional | P0
  - TS-GH-49-004: Verify fallback when harness files lack role/slug fields | Functional | P1
  - TS-GH-49-005: Verify nil returned when neither harness nor config.yaml provides agents | Functional | P1

- | Deprecation warning emitted for legacy path usage
  - TS-GH-49-006: Verify deprecation warning logged when config.yaml agents block is used | Functional | P1
  - TS-GH-49-007: Verify no deprecation warning when harness discovery succeeds | Functional | P1

- | Incomplete harness entries handled with appropriate warnings
  - TS-GH-49-008: Verify entry with role but no slug is skipped with warning | Functional | P1
  - TS-GH-49-009: Verify entry with empty role and empty slug is silently skipped | Functional | P2

- | Duplicate role handling preserves deterministic behavior
  - TS-GH-49-010: Verify duplicate roles keep first occurrence (sorted by Role then Filename) | Functional | P1
  - TS-GH-49-011: Verify info message logged for duplicate role detection | Functional | P2

- | Error resilience in harness discovery
  - TS-GH-49-012: Verify partial read errors still return successfully parsed agents | Functional | P1
  - TS-GH-49-013: Verify hard discovery error falls back to legacy | Functional | P1
  - TS-GH-49-014: Verify warning logged when harness discovery encounters errors | Functional | P2

- | Malformed configuration handling
  - TS-GH-49-015: Verify malformed config.yaml returns nil without panic | Functional | P2

- | Integration with runAppSetup call chain
  - TS-GH-49-016: Verify runAppSetup passes correct parameters to loadKnownSlugs | Functional | P0
  - TS-GH-49-017: Verify filterSlugsByAppSet correctly filters harness-discovered slugs | Functional | P1

---

### Section IV - Sign-off

| Role | Name | Date |
|:-----|:-----|:-----|
| QE Lead | | |
| Dev Lead | | |
| PM | | |
