# Test Plan

## **[Batch Path-Existence Checks via Git Trees API] - Quality Engineering Plan**

### Metadata & Tracking

- **Enhancement:** [GH-72](https://github.com/guyoron1/fullsend/issues/72) — perf(#2351): batch path-existence checks via Git Trees API
- **Feature Tracking:** [GH-72](https://github.com/guyoron1/fullsend/issues/72)
- **Epic Tracking:** [upstream #2360](https://github.com/fullsend-ai/fullsend/pull/2360)
- **QE Owner:** QualityFlow (auto-generated)
- **Owning SIG:** N/A
- **Participating SIGs:** N/A

**Document Conventions:** Standard Go testing conventions using `testing` stdlib and `testify` assertions. Test files follow `*_test.go` naming in the same package.

### Feature Overview

This PR introduces a performance optimization that replaces O(N) individual GitHub API calls for path-existence checks with a single O(1) Git Trees API call via a new `ListRepositoryFiles` method on the `forge.Client` interface. It also migrates status-comment authentication from static tokens to just-in-time minted tokens via a `ClientFactory` pattern, deprecating `--status-token` / `--token` flags in favor of `--mint-url`. Additionally, it implements ADR-0045 Phase 3 features including a `Lint()` method for non-fatal harness diagnostics, `DiscoverRemoteAgents()` for remote config repo discovery, and new config types (`AllowTargets`, `CreateIssuesConfig`) for triage prerequisites.

---

### I. Motivation and Requirements

#### I.1 — Requirement & User Story Review Checklist

- [ ] **Reviewed the relevant requirements.**
  - GH-72 mirrors upstream fullsend-ai/fullsend#2360, specifying batch path-existence checks using the Git Trees API.
  - PR description and linked upstream issue provide clear scope: replace per-path API calls with batch tree listing.

- [ ] **Confirmed clear user stories and understood. Understand the value and customer use cases.**
  - Value: reduces GitHub API usage from O(N) calls to O(1) per path-presence check, improving scaffold/install performance.
  - Mint token migration improves security by using short-lived tokens instead of static credentials.
  - Harness Lint enables non-fatal warnings for gradual schema migration (ADR-0045 Phase 3).

- [ ] **Confirmed requirements are **testable and unambiguous**.**
  - Batch path presence: testable via `FakeClient` mock with deterministic file sets.
  - Mint integration: testable via `ClientFactory` injection and `httptest` servers.
  - Lint diagnostics: testable via direct struct instantiation.

- [ ] **Ensured acceptance criteria are **defined clearly**.**
  - PR includes comprehensive test suites for all new functionality (30+ test functions).
  - `ComparePathPresence` verifies O(1) behavior by injecting error on `GetFileContent`.

- [ ] **Confirmed coverage for NFRs.**
  - Performance: batch API call reduces latency and rate-limit consumption.
  - Security: mint-based tokens are short-lived, reducing credential exposure window.
  - Backward compatibility: deprecated `--token` flag still functions with warning.

#### I.2 — Known Limitations

- `ListRepositoryFiles` returns an error for repositories whose Git tree is too large (truncated response from GitHub API). This is a GitHub platform limitation for repos with >100k files.
- `DiscoverRemoteAgents` is implemented but not yet integrated into a production calling flow — it is infrastructure for future harness-first discovery.
- Mint token integration depends on external OIDC/WIF infrastructure (`ACTIONS_ID_TOKEN_REQUEST_URL`); tests mock this boundary.

#### I.3 — Technology and Design Review

- [ ] **Developer handoff completed and design reviewed.**
  - PR adds new `forge.Client` interface method (`ListRepositoryFiles`), requiring all implementations (live, fake) to implement it.
  - `ClientFactory` pattern in `statuscomment.Notifier` is a well-understood dependency injection approach.

- [ ] **Technology challenges identified and mitigated.**
  - Git Trees API truncation for very large repos is handled with explicit error return.
  - gopls cold-start latency observed during LSP analysis; not a product concern.

- [ ] **Test environment needs identified.**
  - All tests use mocks (`FakeClient`, `httptest`); no external services required.
  - CI workflows reference `mint-url` input but actual minting requires WIF infrastructure.

- [ ] **API extensions and interface changes reviewed.**
  - `forge.Client` interface gains `ListRepositoryFiles(ctx, owner, repo) ([]string, error)`.
  - `forge.FakeClient` updated with `ListRepositoryFiles` implementation.
  - `statuscomment.Notifier` gains `SetClientFactory`, `HasClientFactory`, `refreshClient`.

- [ ] **Topology and deployment impact assessed.**
  - No topology changes. All modifications are library-level.
  - CI workflow changes (`action.yml`, reusable workflows) affect all agent types uniformly.

---

### II. Test Planning

#### II.1 — Scope of Testing

This test plan covers four change themes in GH-72: (1) batch path-existence checking via Git Trees API, (2) mint-based token integration for status comments, (3) ADR-0045 Phase 3 harness features (Lint, DiscoverRemoteAgents), and (4) config type expansion for triage prerequisites.

**Testing Goals:**

- **P0:** Verify `ComparePathPresence` correctly identifies missing and present paths using batch listing.
- **P0:** Verify `ClientFactory` pattern in status comment `Notifier` mints fresh tokens before each API call.
- **P1:** Verify `reconcilestatus` and `run` commands correctly handle `--mint-url` flag and env var fallback.
- **P1:** Verify `DiscoverRemoteAgents` correctly discovers, filters, and sorts harness files from remote repos.
- **P1:** Verify all error paths return descriptive errors and deprecated flags emit warnings.
- **P2:** Verify `Lint()` produces correct diagnostics and config types parse/validate correctly.

**Out of Scope (Testing Scope Exclusions):**

- [ ] **GitHub API rate limiting and quota management** — Platform-level concern managed by forge client layer, not this feature.
- [ ] **OIDC token exchange for workload identity federation** — Infrastructure concern handled by mintclient and cloud provider.
- [ ] **End-to-end CI workflow execution** — Requires production GitHub Actions environment; workflow YAML changes are validated structurally.
- [ ] **Upstream fullsend-ai/fullsend repo behavior** — This is a mirror PR; upstream testing is separate.

#### II.2 — Test Strategy

**Functional:**

- [x] **Functional Testing** — Applicable.
  - Unit tests for all new functions: `ComparePathPresence`, `ListRepositoryFiles`, `ClientFactory`, `Lint`, `DiscoverRemoteAgents`, config constructors/validators.
  - CLI command tests for `reconcilestatus` and `run` with `httptest` servers.

- [x] **Automation Testing** — Applicable.
  - All tests are automated Go tests using `testing` + `testify`.
  - No manual testing required.

- [x] **Regression Testing** — Applicable.
  - Existing `PostStart`/`PostCompletion` tests updated to cover `refreshClient` integration.
  - `LoadRaw` refactored to use `parseRaw`; existing behavior preserved.

**Non-Functional:**

- [ ] **Performance Testing** — Not applicable.
  - Performance improvement is architectural (O(N) → O(1) API calls); no benchmark tests in scope.

- [ ] **Scale Testing** — Not applicable.
  - Truncated tree error handling covers the scale boundary; no load testing needed.

- [ ] **Security Testing** — Not applicable.
  - Token masking (`::add-mask::`) and short-lived minting are security improvements but tested functionally.

- [ ] **Usability Testing** — Not applicable.
  - CLI flag changes are developer-facing; deprecation warnings provide migration guidance.

- [ ] **Monitoring** — Not applicable.
  - No new metrics or observability changes.

**Integration & Compatibility:**

- [x] **Compatibility Testing** — Applicable.
  - Deprecated `--token` flag backward compatibility verified in tests.
  - `forge.Client` interface addition is backward-compatible (new method only).

- [ ] **Upgrade Testing** — Not applicable.
  - No data migration or state upgrade required.

- [x] **Dependencies** — Applicable.
  - `mintclient` package is a new dependency for status comment authentication.
  - `forge.FakeClient` updated to support new interface method.

- [ ] **Cross Integrations** — Not applicable.
  - Changes are internal to fullsend; no cross-product integrations.

**Infrastructure:**

- [ ] **Cloud Testing** — Not applicable.
  - No cloud-specific functionality; all tests run locally with mocks.

#### II.3 — Test Environment

- **Cluster Topology:** N/A — no cluster required; all tests use mocks
- **Platform Version:** Go 1.26.0 (per go.mod)
- **CPU Virtualization:** N/A
- **Compute:** Standard CI runner
- **Special Hardware:** None
- **Storage:** Local filesystem only
- **Network:** `httptest` servers for HTTP API simulation
- **Operators:** N/A
- **Platform:** Linux (CI), macOS/Linux (local development)
- **Special Configs:** `FULLSEND_MINT_URL` env var for mint integration tests

#### II.3.1 — Testing Tools & Frameworks

No new or special tools required. Standard `go test` with `testify` assertions.

#### II.4 — Entry Criteria

- [ ] All PR commits are merged and code compiles without errors
- [ ] `go vet` and `go build` pass cleanly
- [ ] `FakeClient` implements updated `forge.Client` interface (including `ListRepositoryFiles`)
- [ ] `FULLSEND_MINT_URL` documentation available for operators

#### II.5 — Risks

- [ ] **Timeline**
  - Risk: Multi-concern PR (4 themes) increases review and integration time.
  - Mitigation: Each theme is independently testable with isolated test suites.
  - Status: [ ] Monitoring

- [ ] **Coverage**
  - Risk: `DiscoverRemoteAgents` is not yet called from production code; test coverage cannot verify integration behavior.
  - Mitigation: Comprehensive unit tests with `FakeClient`; integration testing deferred to Phase 3 completion.
  - Status: [ ] Accepted

- [ ] **Environment**
  - Risk: Mint token tests cannot exercise real OIDC exchange in CI without WIF infrastructure.
  - Mitigation: Mock boundary at `mintclient.MintToken`; real integration tested in staging environment.
  - Status: [ ] Accepted

- [ ] **Untestable**
  - Risk: CI workflow YAML changes (`action.yml`, reusable workflows) cannot be unit-tested.
  - Mitigation: Structural review of YAML changes; end-to-end validation via CI pipeline execution.
  - Status: [ ] Accepted

- [ ] **Resources**
  - Risk: None identified — all tests run with standard Go tooling.
  - Mitigation: N/A
  - Status: [x] No risk

- [ ] **Dependencies**
  - Risk: `mintclient` package availability and API stability.
  - Mitigation: Package is internal to the fullsend module; versioned together.
  - Status: [x] No risk

- [ ] **Other**
  - Risk: GitHub Git Trees API may change truncation behavior or limits.
  - Mitigation: Explicit `truncated` field check with clear error message.
  - Status: [ ] Monitoring

---

### III. Test Coverage

#### III.1 — Requirements-to-Tests Mapping

- **GH-72** — Batch path-existence checks operate correctly using the Git Trees API
  - Verify batch path check identifies all present paths — Unit Tests — P0
  - Verify batch path check detects missing paths — Unit Tests — P0
  - Verify empty expected list returns no missing — Unit Tests — P0
  - Verify single API call used instead of per-path — Unit Tests — P0

- Git Trees API handles edge cases and error conditions gracefully
  - Verify error on truncated repository tree — Unit Tests — P1
  - Verify error propagation from forge client — Unit Tests — P1
  - Verify FakeClient implements ListRepositoryFiles — Unit Tests — P1

- Status comment notifications work with mint-based token refresh
  - Verify factory called before PostStart — Unit Tests — P0
  - Verify factory called before PostCompletion — Unit Tests — P0
  - Verify factory error propagated on PostStart — Unit Tests — P0
  - Verify static client used when no factory set — Unit Tests — P0
  - Verify completion-disabled path mints then deletes — Unit Tests — P0

- Reconcile-status command supports mint-url authentication
  - Verify mint-url flag mints token and reconciles — Functional — P1
  - Verify error when role missing with mint-url — Unit Tests — P1
  - Verify deprecated token flag still works — Functional — P1
  - Verify FULLSEND_MINT_URL env var fallback — Unit Tests — P1

- Run command integrates mint-url for status comment authentication
  - Verify client factory set from mint-url flag — Unit Tests — P1
  - Verify FULLSEND_MINT_URL env var picked up — Unit Tests — P1
  - Verify error when no mint-url or token available — Unit Tests — P1
  - Verify deprecated static token creates client directly — Unit Tests — P1

- Harness Lint() produces non-fatal diagnostics without breaking Validate()
  - Verify Lint warns on missing role field — Unit Tests — P2
  - Verify Lint returns nil when role is set — Unit Tests — P2
  - Verify Diagnostic string formatting — Unit Tests — P2

- Remote agent discovery works via forge API for harness files
  - Verify discovery of multiple harnesses sorted by role — Unit Tests — P1
  - Verify nil returned for missing harness directory — Unit Tests — P1
  - Verify malformed YAML returns partial results with error — Unit Tests — P1
  - Verify skipping files without role or slug — Unit Tests — P1
  - Verify non-YAML files and subdirectories skipped — Unit Tests — P1

- Config types support create-issues allow-targets validation
  - Verify AllowTargets YAML parsing and defaults — Unit Tests — P2
  - Verify validation rejects invalid repo format — Unit Tests — P2
  - Verify validation rejects empty org — Unit Tests — P2

- CI workflows correctly pass mint-url instead of static status-token
  - Verify action.yml passes mint-url to binary — End-to-End — P1
  - Verify deprecation warning emitted for status-token — Functional — P1
  - Verify token masking in GitHub Actions output — Functional — P1

- Negative: invalid inputs and error conditions handled across all new interfaces
  - Verify error for invalid repo format in status flags — Unit Tests — P1
  - Verify error for mint token acquisition failure — Unit Tests — P1
  - Verify ListDirectoryContents error propagation — Unit Tests — P1

---

### IV. Sign-off

| Role | Name | Date |
|:-----|:-----|:-----|
| QE Lead | | |
| Dev Lead | | |
| PM | | |
