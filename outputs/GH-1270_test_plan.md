# Test Plan

**Issue:** [GH-1270 — Expand precommit-tools.yaml registry coverage](https://github.com/fullsend-ai/fullsend/issues/1270)
**Product:** fullsend
**Date:** 2026-06-25
**Status:** Draft

---

## I. Overview

### 1.1 Summary

GH-1270 tracks expanding the `.pre-commit-tools.yaml` registry introduced by PR #1055 to close coverage gaps found during an audit of pre-commit configs across target repos. The registry is a data-driven system that auto-detects and installs tool dependencies required by a repo's `.pre-commit-config.yaml` hooks before the authoritative pre-commit check runs in CI.

Three specific gaps were identified:

1. **`uv` match miss for local hooks** — The registry entry for `ty` matches `uvx` but not `uv run <tool>` entries.
2. **`tekwizely/pre-commit-golang` script hooks** — Hooks using `language: script` that shell out to Go binaries (`revive`, `gosec`, `gofumpt`, `goimports`, `gocritic`, `golint`) not in the registry.
3. **`shellcheck-py` variant** — Some repos use `shellcheck-py/shellcheck-py` (language: python, auto-managed) instead of `koalaman/shellcheck-precommit`. Documentation gap only.

### 1.2 Scope

| In Scope | Out of Scope |
|:---------|:-------------|
| `.pre-commit-tools.yaml` registry entries | Pre-commit framework internals |
| `resolve-precommit-tools.py` resolver logic | Target repo `.pre-commit-config.yaml` authoring |
| `install-precommit-tools.sh` installer behavior | GitHub Actions runner base image |
| Per-repo registry merge (`merge_registries`) | Org-level customized registry (L1 replacement) |
| `scaffold.go` executable file registration | Kubernetes/cluster-level testing |
| Pre/post script integration (pre-code, pre-fix, post-code, post-fix) | Gitleaks post-script (independent security gate) |

### 1.3 References

| Reference | Link |
|:----------|:-----|
| Parent Issue | [GH-1270](https://github.com/fullsend-ai/fullsend/issues/1270) |
| Introducing PR | [PR #1055](https://github.com/fullsend-ai/fullsend/pull/1055) — feat(scaffold): auto-detect and install pre-commit tool dependencies |
| Registry File | `internal/scaffold/fullsend-repo/scripts/.pre-commit-tools.yaml` |
| Resolver Script | `internal/scaffold/fullsend-repo/scripts/resolve-precommit-tools.py` |
| Installer Script | `internal/scaffold/fullsend-repo/scripts/install-precommit-tools.sh` |

---

## II. Regression Analysis

### 2.1 Call Graph (LSP-Traced)

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

### 2.2 Impacted Components

| Component | Files | Impact |
|:----------|:------|:-------|
| Registry | `.pre-commit-tools.yaml` | New tool entries must follow schema (hook_id, repo, match_entry, install block) |
| Resolver | `resolve-precommit-tools.py` | `entry_match_map` matching, `merge_registries()` merge semantics |
| Installer | `install-precommit-tools.sh` | Binary download + checksum verification, apt/pip/npm install paths |
| Scaffold | `scaffold.go` | `executableFiles` map determines file permissions at install time |
| Pre-scripts | `pre-code.sh`, `pre-fix.sh` | Tool resolution + install before agent runs |
| Post-scripts | `post-code.sh`, `post-fix.sh` | Tool resolution + install before authoritative pre-commit check |

### 2.3 Risk Assessment

| Risk | Severity | Mitigation |
|:-----|:---------|:-----------|
| Registry entry with wrong checksum blocks all pushes | High | Checksum verification test; fail-loud design is intentional |
| Resolver fails to match hook → warning but hooks fail at pre-commit time | Medium | Test resolver matching for all entry patterns (repo+hook_id, match_entry) |
| Per-repo merge introduces conflicting entries | Medium | Test merge_registries() override and exclude semantics |
| New executable scripts not registered → installed as 644, fail to execute | Medium | TestFileModeMatchesFilesystem catches this |
| install-precommit-tools.sh silent failure on unsupported arch | Low | Warning emitted; binary installs skipped gracefully |

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
| 21 | | Pre/post scripts integrate tool auto-detection without breaking CI | Verify post-code.sh resolves and installs tools before pre-commit check | Functional | P0 |
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

---

## IV. Test Summary

### 4.1 Test Counts by Type

| Test Type | Count |
|:----------|:------|
| Unit Tests | 22 |
| Functional | 10 |
| End-to-End | 2 |
| **Total** | **34** |

### 4.2 Test Counts by Priority

| Priority | Count |
|:---------|:------|
| P0 | 3 |
| P1 | 19 |
| P2 | 12 |
| **Total** | **34** |

### 4.3 Implementation Notes

- **Unit Tests** target `resolve-precommit-tools.py` (Python unittest/pytest) and `scaffold.go` (Go testing+testify). The resolver's `resolve()` and `merge_registries()` functions are pure functions ideal for unit testing with fixture YAML files.
- **Functional Tests** target `install-precommit-tools.sh` behavior using mock HTTP servers or fixture tarballs to validate checksum verification and install type handling.
- **End-to-End Tests** validate the full pipeline: `.pre-commit-config.yaml` → `resolve-precommit-tools.py` → JSON manifest → `install-precommit-tools.sh` → tools on PATH.

### 4.4 Existing Test Coverage

The following existing tests provide partial coverage and should be verified to still pass after registry expansion:

| Test | File | Covers |
|:-----|:-----|:-------|
| `TestFileModeMatchesFilesystem` | `internal/scaffold/scaffold_test.go` | Verifies `executableFiles` map stays in sync with filesystem |
| `TestCollectInstallFiles_*` | `internal/scaffold/installfiles_test.go` | Verifies scaffold install includes correct files with correct modes |
| `TestCollectVendoredAssets_*` | `internal/scaffold/vendorcontent_test.go` | Verifies vendor path includes correct files |

---

## V. Risks and Dependencies

| Item | Type | Description |
|:-----|:-----|:------------|
| PR #1055 merge status | Dependency | PR #1055 is already merged; registry expansion can proceed |
| `blocked` label on GH-1270 | Risk | Issue is labeled `blocked` — verify blocker is resolved before implementation |
| Go linter strategy decision (P3) | Dependency | tekwizely hooks require a strategy decision: recommend migration to golangci-lint vs. add registry entries |
| PyYAML availability | Risk | Resolver auto-installs `pyyaml==6.0.2` if missing; network access required on first run |
| Per-repo registry security | Risk | Per-repo `.pre-commit-tools.yaml` is read from base branch only (not PR head) to prevent supply-chain attacks via PR |
