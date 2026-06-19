# My-Project Test Plan


## **Migrate Uninstall Flows to Harness-First Agent Discovery - Quality Engineering Plan**

### Metadata & Tracking

- **Enhancement:** [GH-43](https://github.com/guyoron1/fullsend/pull/43) (fork PR; upstream mirror: fullsend-ai/fullsend#2364)
- **Feature Tracking:** [GH-43](https://github.com/guyoron1/fullsend/pull/43)
- **Epic Tracking:** [fullsend-ai/fullsend#2364](https://github.com/fullsend-ai/fullsend/issues/2364)
- **QE Owner:** TBD
- **Owning SIG:** N/A
- **Participating SIGs:** N/A

**Document Conventions:** Standard QualityFlow STP conventions apply. Test scenario IDs follow the format TS-GH-43-NNN.

### Feature Overview

This feature refactors the uninstall flows in both the org-level (`runUninstall`) and GitHub-specific (`runGitHubUninstall`) CLI commands to use a new harness-first agent discovery mechanism. A new `discoverAgentSlugs` function implements a three-tier fallback strategy: (1) harness wrapper files via `DiscoverRemoteAgents`, (2) legacy `config.yaml` agents block with deprecation warning, (3) empty return for caller-managed defaults. The refactoring consolidates duplicated slug-resolution logic from two call sites into a single reusable module with deduplication, role-to-slug derivation, and partial-error resilience.

---

### Section I — Motivation and Requirements Review

#### I.1 — Requirement & User Story Review Checklist

- [ ] **Reviewed the relevant requirements.** -- Reviewed relevant requirements and user stories to ensure completeness and clarity.
  - PR #43 description references upstream fullsend-ai/fullsend#2364.
  - Requirements are implicit in the refactoring: consolidate slug discovery, prefer harness files, deprecate agents block.

- [ ] **Confirmed clear user stories and understood. Understand the value and customer use cases.** -- Confirmed clear user stories are understood. Understand the value and customer use cases.
  - Value: Prevents orphaned GitHub Apps during partial uninstalls by discovering agent slugs from harness wrapper files (the modern source of truth) rather than relying solely on config.yaml.
  - Customer use case: Org admins running `fullsend uninstall` get correct app cleanup regardless of whether their config uses harness files or legacy agents block.

- [ ] **Confirmed requirements are **testable and unambiguous**.** -- Confirmed requirements are testable and unambiguous.
  - Three-tier fallback is clearly defined and each tier is independently testable.
  - Deprecation warning text is specified and verifiable.

- [ ] **Ensured acceptance criteria are **defined clearly**.** -- Ensured acceptance criteria are defined clearly.
  - Acceptance criteria are implicit: harness slugs preferred over agents block, deprecation warning emitted on fallback, deduplication prevents duplicate app deletions, partial errors don't block discovery.

- [ ] **Confirmed coverage for NFRs.** -- Confirmed coverage for non-functional requirements (NFRs).
  - Performance: No significant impact; slug discovery adds one remote directory listing call.
  - Backward compatibility: Legacy agents block still works with deprecation warning.

#### I.2 — Known Limitations

- The `harness.DiscoverRemoteAgents` function is imported from the `harness` package but defined upstream; this repo's `harness` package only has the local `DiscoverAgents(dir)` variant. Integration depends on upstream availability.
- If the `.fullsend` config repo is completely deleted before uninstall, all three discovery tiers fail and the caller falls back to default naming convention — this is existing behavior preserved, not new.

#### I.3 — Technology and Design Review

- [ ] **Developer handoff session completed.** -- Developer handoff or design review session completed.
  - Handoff conducted via PR #43 review and upstream design discussion (fullsend-ai/fullsend#2364).
  - Key design decision: three-tier fallback strategy for agent slug discovery, consolidating duplicated logic into a single reusable module.
  - Participants: PR author (@guyoron1), upstream maintainers.

- [ ] **Technology challenges and risks identified.** -- Technology challenges and risks identified.
  - Dependency on `harness.DiscoverRemoteAgents` which performs remote API calls to list/read harness directory contents.
  - Partial error path: if some harness YAML files are malformed, valid agents are still returned.

- [ ] **Test environment needs documented.** -- Test environment needs documented.
  - Unit tests use `forge.NewFakeClient()` with `DirContents`, `FileContentsRef`, and `Installations` maps.
  - No real cluster or GitHub API access needed for existing tests.

- [ ] **API extensions or changes reviewed.** -- API extensions or changes reviewed.
  - No public API changes. Internal function signature: `discoverAgentSlugs(ctx, client, owner, configRepo, ref, appSet, cfg, printer) []string`.
  - `runUninstall` and `runGitHubUninstall` signatures unchanged.

- [ ] **Special topology or deployment constraints identified.** -- Special topology or deployment constraints identified.
  - None. CLI runs locally; forge client abstracts GitHub API interactions.

### Section II — Test Strategy and Planning

#### II.1 — Scope of Testing

This test plan covers the refactored agent slug discovery logic in the fullsend CLI uninstall flows. Testing validates that the three-tier fallback (harness files → config agents block → empty) works correctly across both the org-level and GitHub-specific uninstall code paths, including deduplication, role-based slug derivation, deprecation warnings, and partial error handling.

**Testing Goals:**

- **P0:** Verify harness-first discovery correctly resolves agent slugs from remote harness wrapper files.
- **P0:** Verify fallback to config.yaml agents block with deprecation warning when harness files are absent.
- **P1:** Verify default naming convention fallback when neither source provides slugs.
- **P1:** Verify slug deduplication across discovery sources.
- **P1:** Verify slug derivation from role when slug field is empty.
- **P1:** Verify partial error resilience when some harness files are malformed.
- **P2:** Verify both org-level and GitHub-specific uninstall produce consistent behavior via shared agent discovery.

**Out of Scope (Testing Scope Exclusions):**

- [ ] **GitHub App deletion mechanics** -- App deletion is handled by `appsetup` package, not modified in this PR. Already covered by existing tests.
- [ ] **Harness file parsing internals** -- `harness.DiscoverRemoteAgents` is tested in the `harness` package; this plan tests its integration via `discoverAgentSlugs`.
- [ ] **Forge client API interactions** -- Forge client is mocked; real GitHub API testing is out of scope.
- [ ] **Browser-based uninstall prompts** -- User interaction flow (stdin prompts, browser opening) is unchanged and tested elsewhere.

#### II.2 — Test Strategy

**Functional:**

- [ ] **Functional Testing**
  - Applicable: Yes
  - Validate three-tier fallback logic, deduplication, slug derivation, deprecation warnings, and error handling in `discoverAgentSlugs`.

- [ ] **Automation Testing**
  - Applicable: Yes
  - All tests are automated Go unit tests using `testing` + `testify`. No manual steps required.

- [ ] **Regression Testing**
  - Applicable: Yes
  - Existing `runUninstall` and `runGitHubUninstall` tests must continue passing. New tests cover harness-first discovery path.

**Non-Functional:**

- [ ] **Performance Testing**
  - Applicable: No
  - Slug discovery adds one directory listing; negligible performance impact.

- [ ] **Scale Testing**
  - Applicable: No
  - Number of agents is small (typically 2-5); scale is not a concern.

- [ ] **Security Testing**
  - Applicable: No
  - No new authentication or authorization paths introduced.

- [ ] **Usability Testing**
  - Applicable: No
  - CLI output changes are limited to deprecation warning text.

- [ ] **Monitoring**
  - Applicable: No
  - No new metrics or observability changes.

**Integration & Compatibility:**

- [ ] **Compatibility Testing**
  - Applicable: Yes
  - Backward compatibility with legacy config.yaml agents block must be preserved. See Section III group 9 (Backward compatibility) for detailed verification items.

- [ ] **Upgrade Testing**
  - Applicable: No
  - No upgrade path changes; slug discovery is runtime behavior.

- [ ] **Dependencies**
  - Applicable: Yes
  - Depends on `harness.DiscoverRemoteAgents` (upstream fullsend-ai/fullsend). Internal packages (`config.ParseOrgConfig`, `appsetup.AppSlug`) are existing project infrastructure, not cross-team dependencies.

- [ ] **Cross Integrations**
  - Applicable: No
  - Changes are internal to CLI uninstall flow.

**Infrastructure:**

- [ ] **Cloud Testing**
  - Applicable: No
  - CLI runs locally; no cloud-specific testing needed.

#### II.3 — Test Environment

- **Cluster Topology:** N/A — CLI unit tests, no cluster required
- **Platform Version:** Go 1.26+ (per go.mod)
- **CPU Virtualization:** N/A
- **Compute:** Standard CI runner — no special compute requirements for mock-based unit tests
- **Special Hardware:** None
- **Storage:** None
- **Network:** None — forge client is mocked
- **Operators:** None
- **Platform:** Linux/macOS CI (GitHub Actions)
- **Special Configs:** `forge.NewFakeClient()` with `DirContents`, `FileContentsRef`, `FileContents`, and `Installations` maps

#### II.3.1 — Testing Tools & Frameworks

No new or special tools required. Standard Go testing with `testify` assertions.

#### II.4 — Entry Criteria

- [ ] PR #43 merged or branch available for testing
- [ ] `harness.DiscoverRemoteAgents` function available in `internal/harness` package
- [ ] `go test ./internal/cli/...` compiles successfully
- [ ] Existing uninstall tests pass without modification

#### II.5 — Risks

- [ ] **Timeline**
  - Risk: Upstream `DiscoverRemoteAgents` not yet available in this repo's harness package
  - Mitigation: Function exists upstream (fullsend-ai/fullsend#2364); will be available after merge. If upstream merge is delayed, defer harness-first tier testing and test only config fallback path
  - Status: [ ] Resolved

- [ ] **Coverage**
  - Risk: Three-tier fallback has many permutations; edge cases may be missed
  - Mitigation: 8 dedicated unit tests for `discoverAgentSlugs` cover all tiers and edge cases
  - Status: [x] Resolved

- [ ] **Untestable**
  - Risk: Real GitHub API interaction for harness file discovery not tested at unit level
  - Mitigation: Integration tested via `forge.FakeClient`; real API tested in e2e suite
  - Status: [ ] Accepted

- [ ] **Dependencies**
  - Risk: `harness.DiscoverRemoteAgents` interface changes upstream could break `discoverAgentSlugs`
  - Mitigation: Function signature is stable; PR includes comprehensive tests that would catch breakage
  - Status: [ ] Monitoring


---

### Section III — Requirements-to-Tests Mapping

#### III.1 — Requirements Mapping

- **GH-43** — Harness-first agent slug discovery resolves slugs correctly
  - Verify harness wrapper files are preferred over config agents block
  - Verify slugs extracted from harness role and slug fields
  - Verify harness discovery skips agents block when harness provides slugs
  - **Tier:** [Tier 1]
  - **Priority:** P0

- **GH-43** — Fallback to config.yaml agents block with deprecation warning
  - Verify agents block slugs returned when no harness files exist
  - Verify deprecation warning emitted for agents block fallback
  - Verify agents block skipped when it has zero entries
  - **Tier:** [Tier 1]
  - **Priority:** P0

- **GH-43** — Default naming fallback when neither source provides slugs
  - Verify nil returned when no harness files and no config agents
  - Verify caller applies default role-based naming convention
  - **Tier:** [Tier 1]
  - **Priority:** P1

- **GH-43** — Slug derivation from role when slug field is empty
  - Verify slug derived from appSet and role via AppSlug convention
  - Verify derivation works for both harness and config agent sources
  - **Tier:** [Tier 1]
  - **Priority:** P1

- **GH-43** — Slug deduplication across discovery
  - Verify duplicate slugs from multiple harness files are deduplicated
  - Verify first occurrence wins in deduplication order
  - **Tier:** [Tier 1]
  - **Priority:** P2

- **GH-43** — Partial error resilience for malformed harness files
  - Verify valid agents returned when some harness files fail to parse
  - Verify warning emitted for unreadable harness files
  - Verify fallback to agents block skipped when valid harness slugs exist despite errors
  - **Tier:** [Tier 1]
  - **Priority:** P1

- **GH-43** — Org-level uninstall uses harness-discovered agents for app cleanup
  - Verify org-level uninstall uses harness-discovered slugs for app deletion
  - Verify org-level uninstall falls back to agents block with warning
  - Verify org-level uninstall applies default roles when discovery returns empty
  - **Tier:** [Tier 1]
  - **Priority:** P0

- **GH-43** — GitHub-specific uninstall uses harness-discovered agents for app cleanup
  - Verify GitHub-specific uninstall uses harness-discovered slugs
  - Verify GitHub-specific uninstall falls back to agents block with warning
  - Verify GitHub-specific uninstall applies default roles when discovery returns empty
  - **Tier:** [Tier 1]
  - **Priority:** P0

- **GH-43** — Backward compatibility with existing uninstall behavior
  - Verify existing uninstall tests pass without modification
  - Verify legacy slug-from-config path produces identical results
  - Verify error handling for missing config repo preserved
  - **Tier:** [Tier 1]
  - **Priority:** P2

---

### Section IV — Sign-off and Approval

This Software Test Plan requires approval from the following stakeholders:

* **Reviewers:**
  - TBD / @TBD
* **Approvers:**
  - guyoron1 / @guyoron1
