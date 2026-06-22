# Test Plan

## **Two-Pass Review Strategy for Large PRs - Quality Engineering Plan**

### Metadata & Tracking

- **Enhancement:** [GH-73](https://github.com/guyoron1/fullsend/issues/73)
- **Feature Tracking:** [GH-73](https://github.com/guyoron1/fullsend/issues/73) — Mirror of upstream fullsend-ai/fullsend#2303
- **Epic Tracking:** N/A
- **QE Owner:** Unassigned
- **Owning SIG:** N/A
- **Participating SIGs:** N/A

**Document Conventions:** All test tiers follow the auto-detected strategy. Unit Tests use Go `testing` + `testify`. Functional and End-to-End tests exercise CLI commands and layer integrations with fake forge clients.

### Feature Overview

This feature introduces a two-pass review strategy for large PRs to improve review quality and coverage. The PR includes significant enhancements across the fullsend CLI, binary management, forge abstraction, harness system, enrollment layers, and GCF dispatch infrastructure. Key additions include release binary download with checksum verification, remote agent discovery from config repos, vendor source root resolution, harness lint diagnostics, enhanced post-review inline comment handling, mint role provisioning, and status reconciliation for orphaned agent processes.

---

### Section I — Motivation and Requirements Review

#### I.1 — Requirement & User Story Review Checklist

- [ ] **Reviewed the relevant requirements.** -- Confirmed the feature requirements are documented.
  - GH-73 mirrors upstream fullsend-ai/fullsend#2303, describing a two-pass review strategy for large PRs
  - The issue body is minimal; functional scope was derived from code analysis and LSP regression tracing
- [ ] **Confirmed clear user stories and understood. Understand the value and customer use cases.** -- Understood the customer value and use cases.
  - Value: improved review quality for large PRs by splitting review into two passes
  - Users: CI/CD pipelines running fullsend agents for automated code review
- [ ] **Confirmed requirements are **testable and unambiguous**.** -- Assessed testability of each requirement.
  - All 11 validated requirements are testable via unit tests or functional tests with fake clients
  - LSP analysis confirmed concrete function entry points for each requirement
- [ ] **Ensured acceptance criteria are **defined clearly**.** -- Reviewed acceptance criteria clarity.
  - No explicit acceptance criteria in the issue; criteria derived from code behavior and regression analysis
  - Each requirement maps to specific Go functions with well-defined input/output contracts
- [ ] **Confirmed coverage for NFRs.** -- Evaluated non-functional requirements.
  - Binary download enforces 200MB compressed / 500MB uncompressed size limits
  - SHA256 checksum verification ensures binary integrity
  - Path traversal protections in tar extraction (rejects `..` and absolute paths)

#### I.2 — Known Limitations

- The issue body is minimal ("Adds a two-pass review strategy for large PRs"); detailed requirements were inferred from code changes
- No explicit acceptance criteria defined in GH-73; test scenarios are derived from regression analysis
- The PR bundles many independent changes (15,748 additions) beyond the stated two-pass review feature, including infrastructure improvements, new CLI commands, and refactored provisioning
- Auto-detected project context (`config_dir: null`) — no project-specific tier definitions, patterns, or component mappings available

#### I.3 — Technology and Design Review

- [ ] **Developer handoff completed; technical approach reviewed.** -- Assessed developer collaboration.
  - PR is a mirror of upstream #2303; no direct developer handoff available
  - Code analysis via LSP provided sufficient understanding of architecture
- [ ] **Technology challenges identified and addressed.** -- Reviewed technical challenges.
  - Cross-compilation for sandbox binaries (macOS host → Linux sandbox) handled by `binary.ResolveForRun`
  - Remote source tree fetching introduces network dependency with size limits and checksum verification
- [ ] **Test environment needs identified.** -- Confirmed environment requirements.
  - Unit tests require Go 1.26+ with testify; no external services needed
  - Functional tests require fake forge clients (already implemented in `forge/fake.go`)
- [ ] **API extensions and contract changes reviewed.** -- Evaluated API surface changes.
  - Forge `Client` interface extended with `ListDirectoryContents`, `GetFileContentAtRef`, `ListPullRequestFileDiffs`
  - New `ReviewComment` struct and `DismissPullRequestReview` method added
- [ ] **Topology and deployment requirements reviewed.** -- Assessed deployment topology.
  - No topology changes; all changes are CLI-side and run in existing sandbox infrastructure

### Section II — Test Planning

#### II.1 — Scope of Testing

This test plan covers all functional changes introduced in GH-73, focusing on the CLI layer (agent run lifecycle, post-review, reconcile-status, mint setup, vendor), binary management (download, checksum, vendor root), forge abstraction (new API methods, fake client), harness system (remote discovery, lint), enrollment/vendor layers, and GCF dispatch provisioning.

**Testing Goals:**

- **P0:** Verify binary download integrity (checksum verification, size limits, tar extraction safety)
- **P0:** Verify agent run lifecycle completes through all bootstrap phases
- **P1:** Verify post-review CLI handles stale-head detection, inline comments, and diff hunk filtering
- **P1:** Verify remote agent discovery correctly parses harness YAML and derives slugs
- **P1:** Verify mint role provisioning across all input modes (slug+PEM, existing secret)
- **P1:** Verify enrollment and vendor layers handle cross-platform binary installation
- **P1:** Verify GCF provisioner creates and manages cloud functions
- **P1:** Verify invalid inputs are rejected gracefully across all CLI commands
- **P2:** Verify harness lint diagnostics detect missing role field
- **P2:** Verify status reconciliation finalizes orphaned comments idempotently

**Out of Scope (Testing Scope Exclusions):**

- [ ] **GitHub Actions workflow YAML validation** — CI/CD infrastructure tested by platform pipeline
- [ ] **Documentation rendering** — Markdown rendering is a platform-level concern
- [ ] **Dependabot configuration** — GitHub platform feature, not product-level test
- [ ] **Upstream fullsend-ai/fullsend#2303 end-to-end integration** — Mirror PR; upstream tests cover integration

#### II.2 — Test Strategy

**Functional:**

- [x] **Functional Testing** — Applicable
  - Validate CLI commands (post-review, run, reconcile-status, mint add-role, vendor) produce correct outputs and side effects
  - Verify forge client methods return expected data for valid and invalid inputs
- [x] **Automation Testing** — Applicable
  - All tests are automated using Go `testing` package with `testify` assertions
  - Tests use `httptest` servers, fake forge clients, and in-memory tar archives
- [x] **Regression Testing** — Applicable
  - LSP-traced regression chains confirm impacted call paths: `runAgent` → `bootstrapCommon` → `ResolveForRun` → `DownloadRelease`
  - `submitFormalReview` → `findingsToReviewComments` chain verified for inline comment changes

**Non-Functional:**

- [ ] **Performance Testing** — Not applicable
  - No performance-sensitive changes; download size limits provide implicit bounds
- [ ] **Scale Testing** — Not applicable
  - No scale-sensitive changes in this PR
- [x] **Security Testing** — Applicable
  - Binary checksum verification prevents supply-chain attacks
  - Tar extraction rejects path traversal (`..` and absolute paths)
  - Download size limits prevent denial-of-service via oversized artifacts
- [ ] **Usability Testing** — Not applicable
  - CLI interface changes are backward-compatible
- [ ] **Monitoring** — Not applicable
  - No monitoring changes

**Integration & Compatibility:**

- [ ] **Compatibility Testing** — Not applicable
  - No cross-version compatibility concerns
- [ ] **Upgrade Testing** — Not applicable
  - No upgrade path changes
- [x] **Dependencies** — Applicable
  - New forge interface methods must be implemented by all Client implementations
  - `ResolveVendorRoot` fallback chain depends on `ModuleRoot()` and GitHub release API
- [ ] **Cross Integrations** — Not applicable
  - No cross-product integrations

**Infrastructure:**

- [ ] **Cloud Testing** — Not applicable
  - GCF provisioner tests use fake client, not real cloud infrastructure

#### II.3 — Test Environment

- **Cluster Topology:** N/A — unit and functional tests run locally
- **Platform Version:** Go 1.26.0 (per go.mod)
- **CPU Virtualization:** N/A
- **Compute:** Standard CI runner (Linux amd64)
- **Special Hardware:** N/A
- **Storage:** Local filesystem for temp dirs and extracted archives
- **Network:** `httptest` servers for HTTP mocking; no external network required
- **Operators:** N/A
- **Platform:** Linux (sandbox target); macOS (cross-compilation source)
- **Special Configs:** `FULLSEND_SANDBOX_ARCH` env var for cross-compilation override

#### II.3.1 — Testing Tools & Frameworks

No new or special tools required. Standard Go testing infrastructure with `testify` and `httptest`.

#### II.4 — Entry Criteria

- [ ] Go 1.26+ toolchain available on CI runner
- [ ] All Go module dependencies resolved (`go mod download`)
- [ ] Testify assertion library available
- [ ] PR branch builds without compilation errors

#### II.5 — Risks

- [ ] **Timeline**
  - Risk: Large PR (15,748 additions) may require extended review cycles
  - Mitigation: Focus testing on P0/P1 requirements first; P2 items can follow
  - Status: [ ] Open
- [ ] **Coverage**
  - Risk: Bundled changes may have untested interactions between new components
  - Mitigation: LSP regression analysis identified key call chains; tests follow traced paths
  - Status: [ ] Open
- [ ] **Environment**
  - Risk: Cross-compilation tests may behave differently on arm64 vs amd64
  - Mitigation: `FULLSEND_SANDBOX_ARCH` override allows explicit architecture targeting
  - Status: [ ] Open
- [ ] **Untestable**
  - Risk: Browser-based GitHub App manifest flow (mint add-role --org) cannot be unit tested
  - Mitigation: Test hooks (`mintAddRoleResolveToken`, `mintAddRoleAppSetup`) enable isolated testing
  - Status: [ ] Mitigated
- [ ] **Resources**
  - Risk: No QE owner assigned
  - Mitigation: Assign QE owner before test execution
  - Status: [ ] Open
- [ ] **Dependencies**
  - Risk: `DownloadRelease` depends on GitHub Releases API availability
  - Mitigation: Tests use `httptest` server with `ReleaseBaseURL` override; no real API calls
  - Status: [ ] Mitigated
- [ ] **Other**
  - Risk: Minimal issue description limits requirement traceability
  - Mitigation: Requirements derived from code analysis and LSP regression tracing
  - Status: [ ] Accepted

---

### Section III — Requirements-to-Tests Mapping

#### III.1 — Requirements Mapping

- **GH-73** — Agent sandbox run lifecycle completes successfully with all bootstrap phases
  - Verify agent run completes full lifecycle — End-to-End — P0
  - Verify sandbox cleanup after successful run — Functional — P0
  - Verify run fails gracefully when openshell unavailable — Unit Tests — P0
  - Verify run aborts on bootstrap failure — Unit Tests — P0
  - Verify validation loop retries on failure — Functional — P0

- **GH-73** — Binary download and checksum verification ensures integrity of cross-compiled binaries
  - Verify release download with valid checksum — Unit Tests — P0
  - Verify rejection of tampered archive — Unit Tests — P0
  - Verify rejection of oversized download — Unit Tests — P0
  - Verify latest release tag resolution — Unit Tests — P0
  - Verify source tree extraction strips root prefix — Unit Tests — P0

- **GH-73** — Vendor source root resolution falls back through local checkout, module root, and remote fetch
  - Verify explicit source dir takes precedence — Unit Tests — P1
  - Verify fallback to ModuleRoot — Unit Tests — P1
  - Verify fallback to GitHub source fetch — Unit Tests — P1
  - Verify error for dev build without checkout — Unit Tests — P1

- **GH-73** — Post-review CLI correctly handles stale-head detection and inline diff comments
  - Verify stale-head detection discards review — Unit Tests — P1
  - Verify inline comments map to diff hunks — Unit Tests — P1
  - Verify file-level fallback for out-of-hunk lines — Unit Tests — P1
  - Verify stale reviews are minimized — Unit Tests — P1
  - Verify COMMENT review skipped without inline findings — Unit Tests — P1
  - Verify error for empty review body — Unit Tests — P1

- **GH-73** — Remote agent discovery identifies roles and slugs from harness files in config repos
  - Verify discovery parses role and slug from YAML — Unit Tests — P1
  - Verify slug derivation from role and appSet — Unit Tests — P1
  - Verify deduplication of discovered slugs — Unit Tests — P1
  - Verify graceful handling of partial parse errors — Unit Tests — P1
  - Verify nil return when harness dir missing — Unit Tests — P1

- **GH-73** — Mint setup and role provisioning operates correctly with browser, PEM, and existing-secret modes
  - Verify add-role with slug and PEM file — Functional — P1
  - Verify add-role with existing PEM secret — Functional — P1
  - Verify error for missing project flag — Unit Tests — P1
  - Verify mutual exclusivity of input modes — Unit Tests — P1

- **GH-73** — Harness lint diagnostics detect missing role field and emit appropriate severity
  - Verify lint warns on missing role — Unit Tests — P2
  - Verify no diagnostics for valid harness — Unit Tests — P2

- **GH-73** — GCF provisioner and fake client correctly provision and manage cloud functions
  - Verify cloud function creation and deployment — Functional — P1
  - Verify environment variable updates on function — Functional — P1
  - Verify error handling for invalid project ID — Unit Tests — P1
  - Verify fake client simulates API behavior — Unit Tests — P1

- **GH-73** — Enrollment and vendor layers handle vendored binary installation and workflow generation
  - Verify enrollment provisions new repository — Functional — P1
  - Verify vendored binary installs cross-platform — Functional — P1
  - Verify workflow YAML renders correctly — Unit Tests — P1
  - Verify error for unsupported architecture — Unit Tests — P1

- **GH-73** — Status reconciliation finalizes orphaned status comments from hard-killed agent processes
  - Verify orphaned comment finalized to interrupted — Unit Tests — P2
  - Verify idempotent on already-finalized comment — Unit Tests — P2
  - Verify cancelled reason handled correctly — Unit Tests — P2

- **GH-73** — Invalid inputs and error conditions are handled gracefully across CLI commands
  - Verify rejection of invalid repo format — Unit Tests — P1
  - Verify rejection of negative PR numbers — Unit Tests — P1
  - Verify rejection of missing required tokens — Unit Tests — P1
  - Verify rejection of invalid SHA format — Unit Tests — P1

---

### Section IV — Sign-off

| Role | Name | Date |
|:-----|:-----|:-----|
| QE Lead | _Pending_ | |
| Dev Lead | _Pending_ | |
| PM | _Pending_ | |
