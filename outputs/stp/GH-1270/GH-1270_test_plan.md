# Test Plan

**Issue:** [GH-1270 — Expand precommit-tools.yaml registry coverage](https://github.com/fullsend-ai/fullsend/issues/1270)
**Product:** fullsend
**Date:** 2026-06-25
**Status:** Draft

---

## I. Requirements Review

### I.1 Requirements Review Checklist

- [x] **Review Requirements** — Three specific gaps identified in GH-1270: uv match miss, tekwizely script hooks, shellcheck-py variant
- [x] **Understand Value** — Expanding the pre-commit-tools registry ensures CI pipelines auto-detect and install all required tool dependencies without manual intervention
- [x] **Testability** — All three gaps produce observable behavior (resolver match/no-match, warning messages, install actions) that can be verified
- [x] **Acceptance Criteria** — Derived from issue body: (1) uv match_entry resolves "uv run" entries, (2) tekwizely hooks produce actionable warnings, (3) shellcheck-py variant is handled correctly
- [x] **Non-Functional Requirements** — Supply-chain safety via checksum verification; no performance SLAs for CI tooling

### I.2 Known Limitations

- Go linter strategy (P3) is deferred pending a design decision — tekwizely hooks require choosing between registry entries and migration to golangci-lint. Test scenarios will be added once the approach is selected.
- No Jira instance configured; requirements derived from GitHub issue body text.

### I.3 Technology Review

**Call Graph (LSP-Traced)**

The following dependency chains were traced using LSP analysis on the Go source:

```
scaffold.go::executableFiles
  └─ scaffold.go::FileMode()
       ├─ installfiles.go::CollectInstallFiles()
       │    ├─ layers/workflows.go::Install()
       │    ├─ installfiles.go::ManagedPaths()
       │    └─ installfiles_test.go::TestCollectInstallFiles_*
       ├─ vendorcontent.go::CollectVendoredAssets()
       │    ├─ cli/vendor.go::prepareVendorFiles()
       │    ├─ layers/vendorbinary.go::reportSourceAlignment()
       │    └─ vendorcontent_test.go::TestCollectVendoredAssets_*
       └─ scaffold_test.go::TestFileModeMatchesFilesystem()
```

**Key finding:** Adding new scripts to `executableFiles` in `scaffold.go` propagates through `FileMode()` to both `CollectInstallFiles` (install path) and `CollectVendoredAssets` (vendor path). Both paths must handle the new entries correctly to avoid scaffold install or vendor drift.

**Impacted Components**

| Component | Files | Impact |
|:----------|:------|:-------|
| Registry | `.pre-commit-tools.yaml` | New tool entries must follow schema (hook_id, repo, match_entry, install block) |
| Resolver | `resolve-precommit-tools.py` | `entry_match_map` matching, `merge_registries()` merge semantics |
| Installer | `install-precommit-tools.sh` | Binary download + checksum verification, apt/pip/npm install paths |
| Scaffold | `scaffold.go` | `executableFiles` map determines file permissions at install time |
| Pre-scripts | `pre-code.sh`, `pre-fix.sh` | Tool resolution + install before agent runs |
| Post-scripts | `post-code.sh`, `post-fix.sh` | Tool resolution + install before authoritative pre-commit check |

**References**

| Reference | Link |
|:----------|:-----|
| Parent Issue | [GH-1270](https://github.com/fullsend-ai/fullsend/issues/1270) |
| Introducing PR | [PR #1055](https://github.com/fullsend-ai/fullsend/pull/1055) (merged 2026-06-25) — feat(scaffold): auto-detect and install pre-commit tool dependencies |
| Registry File | `internal/scaffold/fullsend-repo/scripts/.pre-commit-tools.yaml` |
| Resolver Script | `internal/scaffold/fullsend-repo/scripts/resolve-precommit-tools.py` |
| Installer Script | `internal/scaffold/fullsend-repo/scripts/install-precommit-tools.sh` |

---

## II. Test Strategy

### II.1 Scope of Testing

While GH-1270 identifies three specific gaps, this test plan covers the entire pre-commit-tools subsystem to ensure registry expansion does not introduce regressions in existing functionality.

| In Scope | Out of Scope |
|:---------|:-------------|
| `.pre-commit-tools.yaml` registry entries | Pre-commit framework internals |
| `resolve-precommit-tools.py` resolver logic | Target repo `.pre-commit-config.yaml` authoring |
| `install-precommit-tools.sh` installer behavior | GitHub Actions runner base image |
| Per-repo registry merge (`merge_registries`) | Org-level customized registry (L1 replacement) |
| `scaffold.go` executable file registration | Kubernetes/cluster-level testing |
| Pre/post script integration (pre-code, pre-fix, post-code, post-fix) | Gitleaks post-script (independent security gate) |

### II.2 Test Strategy Classification

- [x] **Functional Testing** — Core feature validation: resolver matching, installer behavior, registry merge semantics
- [x] **Automation Testing** — All tests are automated; no manual testing required
- [x] **Security Testing** — Supply-chain safety via checksum verification; per-repo registry reads from base branch only
- [x] **Regression Testing** — Existing tests (TestFileModeMatchesFilesystem, TestCollectInstallFiles, TestCollectVendoredAssets) must continue passing
- [ ] **Performance Testing** — N/A; no latency or throughput requirements for CI tooling
- [ ] **Usability Testing** — N/A; no UI components
- [ ] **Upgrade Testing** — N/A; CI tooling with no persistent state across upgrades
- [ ] **Monitoring Testing** — N/A; no metrics or alerting requirements

### II.3 Test Environment

- **CI Runner:** GitHub Actions Ubuntu runner with Python 3.x and Go 1.26+
- **Dependencies:** PyYAML 6.0.2 (auto-installed by resolver if missing), pre-commit framework
- **Fixture Data:** Mock `.pre-commit-config.yaml` files, fixture tarballs for binary install tests, mock HTTP servers for checksum verification tests

### II.4 Entry/Exit Criteria

**Entry Criteria:**
- [x] PR #1055 merged (confirmed: merged 2026-06-25)
- [ ] `blocked` label removed from GH-1270 (stale — blocker is resolved, label should be removed)
- [x] Dev environment with `.pre-commit-config.yaml` samples available

**Exit Criteria:**
- [ ] All P0 scenarios pass (checksum verification, CI pipeline integration)
- [ ] All P1 scenarios pass (resolver matching, installer behavior, registry merge)
- [ ] No regressions in existing TestFileModeMatchesFilesystem
- [ ] No regressions in TestCollectInstallFiles and TestCollectVendoredAssets

### II.5 Risks

| Item | Severity | Description | Mitigation |
|:-----|:---------|:------------|:-----------|
| Registry entry with wrong checksum blocks all pushes | High | Fail-loud design is intentional — a bad checksum hard-stops the pipeline | Checksum verification test scenarios (rows 12, 25); binary checksums validated before merge |
| Resolver fails to match hook | Medium | Unmatched hook produces a warning but hooks fail at pre-commit time | Test resolver matching for all entry patterns (repo+hook_id, match_entry) |
| Per-repo merge introduces conflicting entries | Medium | Override semantics may produce unexpected results | Test merge_registries() override and exclude semantics |
| New executable scripts not registered | Medium | Scripts installed as 644 instead of 755, fail to execute | TestFileModeMatchesFilesystem catches permission mismatches |
| install-precommit-tools.sh silent failure on unsupported arch | Low | Binary installs skipped on unsupported architectures | Warning emitted; graceful degradation tested |
| PyYAML availability | Low | Resolver auto-installs `pyyaml==6.0.2` if missing; network access required on first run | CI runners have network access; pip install is idempotent |
| Per-repo registry security | Low | Per-repo `.pre-commit-tools.yaml` could be weaponized via PR | Read from base branch only (not PR head) to prevent supply-chain attacks |
| Registry changes deployed without comprehensive test coverage initially | Medium | Expanded registry could surface latent bugs in hook matching | Full subsystem test coverage via this plan mitigates risk |

### II.6 Dependencies (Resolved)

| Dependency | Status | Notes |
|:-----------|:-------|:------|
| PR #1055 | ✅ Merged (2026-06-25) | Registry expansion can proceed |
| `blocked` label on GH-1270 | ⚠️ Stale | PR #1055 merged resolves the blocker; label should be removed from the issue |
| Go linter strategy decision (P3) | ⏳ Pending | tekwizely hooks require a strategy decision; deferred — test scenarios will be added once approach is selected |

---

## III. Requirements-to-Tests Mapping

| # | Requirement ID | Requirement Summary | Test Scenario | Test Type | Priority |
|:--|:---------------|:-------------------|:--------------|:----------|:---------|
| 1 | GH-1270 | Registry match_entry correctly resolves local hooks using "uv run" entry format | Verify resolver matches "uv" match_entry for "uv run mypy" hook entry | Unit Tests | P1 |
| 2 | | | Verify resolver does not match partial substrings (e.g., "uv" does not match "uvx-other") | Unit Tests | P1 |
| 3 | | | Verify resolver returns no match for unknown entry command | Unit Tests | P2 |
| 4 | | Registry supports deduplication when multiple hooks resolve to same tool | Verify seen_names deduplication when both "uvx" and "uv" hooks resolve to uv | Unit Tests | P1 |
| 5 | | | Verify only one install entry emitted for duplicated tool name | Unit Tests | P1 |
| 6 | | Resolver emits actionable warnings for unregistered system hooks | Verify warning for language:system hook not in registry includes command name | Unit Tests | P2 |
| 7 | | | Verify warning for language:golang hook mentions Go toolchain requirement | Unit Tests | P2 |
| 8 | | | Verify no warning for language:python hooks (auto-managed by pre-commit) | Unit Tests | P2 |
| 9 | | Registry handles skip_install to prevent double-installation | Verify tool with skip_install:true is recognized but not installed | Unit Tests | P1 |
| 10 | | | Verify skip_install tool does not appear in resolved manifest output | Unit Tests | P1 |
| 11 | | Install script handles all install types with supply-chain safety | Verify binary install with valid checksum succeeds | Functional | P1 |
| 12 | | | Verify binary install with mismatched checksum exits non-zero | Functional | P0 |
| 13 | | | Verify pip install without version pin is rejected | Unit Tests | P1 |
| 14 | | | Verify npm install without version pin is rejected | Unit Tests | P1 |
| 15 | | | Verify unsupported architecture emits warning and skips binary | Unit Tests | P2 |
| 16 | | Per-repo registry merge correctly extends, overrides, and excludes | Verify additive merge appends new entries to upstream registry | Unit Tests | P1 |
| 17 | | | Verify matching (repo, hook_id) key overrides upstream entry | Unit Tests | P1 |
| 18 | | | Verify exclude:true suppresses matching upstream entry | Unit Tests | P1 |
| 19 | | | Verify invalid per-repo entry (missing hook_id) emits warning | Unit Tests | P2 |
| 20 | | | Verify empty per-repo registry falls back to upstream only | Unit Tests | P2 |
| 21 | | Pre/post scripts integrate tool auto-detection without breaking CI | Verify post-code.sh resolves and installs tools before pre-commit check | Functional | P1 |
| 22 | | | Verify pre-code.sh installs tools and adds ~/.local/bin to PATH and GITHUB_PATH | Functional | P1 |
| 23 | | | Verify graceful degradation when resolve script fails (warning, no abort) | Functional | P1 |
| 24 | | | Verify graceful handling when .pre-commit-config.yaml is absent | Functional | P2 |
| 25 | | Binary checksum verification fails loudly on mismatch | Verify sha256sum failure causes exit 1 (hard stop, not skip) | Functional | P0 |
| 26 | | | Verify successful checksum allows install to proceed | Functional | P1 |
| 27 | | Scaffold registers new scripts as executable | Verify install-precommit-tools.sh gets 100755 mode via FileMode() | Unit Tests | P1 |
| 28 | | | Verify resolve-precommit-tools.py gets 100755 mode via FileMode() | Unit Tests | P1 |
| 29 | | | Verify TestFileModeMatchesFilesystem passes with new entries | Unit Tests | P1 |
| 30 | | Resolver handles malformed input gracefully | Verify resolver returns empty tools for invalid YAML in .pre-commit-config.yaml | Unit Tests | P2 |
| 31 | | | Verify resolver returns empty tools for missing repos field | Unit Tests | P2 |
| 32 | | | Verify resolver handles non-list repos field | Unit Tests | P2 |
| 33 | | End-to-end: full resolution + install pipeline works for multi-tool repo | Verify resolver → manifest → installer pipeline for repo with lychee + uv + actionlint hooks | End-to-End | P1 |
| 34 | | | Verify pipeline handles repo with no matching hooks (empty manifest, no install) | End-to-End | P2 |
| 35 | | Resolver correctly handles shellcheck-py variant (auto-managed) | Verify no warning emitted for shellcheck-py/shellcheck-py hook (language: python, auto-managed) | Unit Tests | P2 |
| 36 | | | Verify resolver identifies shellcheck-py as auto-managed and does not flag it for install | Unit Tests | P2 |

---

## IV. Test Summary

### 4.1 Test Counts by Type

| Test Type | Count |
|:----------|:------|
| Unit Tests | 24 |
| Functional | 10 |
| End-to-End | 2 |
| **Total** | **36** |

### 4.2 Test Counts by Priority

| Priority | Count |
|:---------|:------|
| P0 | 2 |
| P1 | 20 |
| P2 | 14 |
| **Total** | **36** |

### 4.3 Implementation Notes

- **Unit Tests** target `resolve-precommit-tools.py` and `scaffold.go`. The resolver's `resolve()` and `merge_registries()` functions are pure functions ideal for unit testing with fixture YAML files.
- **Functional Tests** target `install-precommit-tools.sh` behavior using mock HTTP servers or fixture tarballs to validate checksum verification and install type handling.
- **End-to-End Tests** validate the full pipeline: `.pre-commit-config.yaml` → `resolve-precommit-tools.py` → JSON manifest → `install-precommit-tools.sh` → tools on PATH.

### 4.4 Existing Test Coverage

The following existing tests provide partial coverage and should be verified to still pass after registry expansion:

| Test | File | Covers |
|:-----|:-----|:-------|
| `TestFileModeMatchesFilesystem` | `internal/scaffold/scaffold_test.go` | Verifies `executableFiles` map stays in sync with filesystem |
| `TestCollectInstallFiles_*` | `internal/scaffold/installfiles_test.go` | Verifies scaffold install includes correct files with correct modes |
| `TestCollectVendoredAssets_*` | `internal/scaffold/vendorcontent_test.go` | Verifies vendor path includes correct files |
