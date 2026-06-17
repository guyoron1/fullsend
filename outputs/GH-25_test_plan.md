# FullSend Test Plan

| Field | Value |
|:------|:------|
| **Ticket** | GH-25 |
| **Title** | perf(#2351): batch path-existence checks via Git Trees API |
| **Author** | guyoron1 |
| **Status** | Open |
| **Branch** | `agent/2351-batch-path-presence` |
| **Date** | 2026-06-17 |
| **Product** | FullSend |
| **Platform** | GitHub Actions |
| **Version** | 0.x |

---

## 1. Summary

This PR adds `forge.Client.ListRepositoryFiles` to retrieve all file paths in a
repository's default branch with a single Git Trees API call (refs -> commit ->
tree?recursive=1). It replaces the O(N) `GetFileContent` pattern used by
`ComparePathPresence`, reducing 100+ sequential API calls to 3 fixed calls
regardless of path count.

Additionally, it introduces:
- Harness `Lint()` diagnostic infrastructure (Phase 3 of ADR-0045)
- Remote harness agent discovery via forge API (`DiscoverRemoteAgents`)
- `parseRaw()` helper for byte-based YAML parsing of harness files
- Mint-URL based token acquisition replacing deprecated static `status-token`
- `OrgConfig` enhancements for `CreateIssues` and `MintURL` fields
- Status comment reconciliation with mint-URL support

---

## 2. Requirements

| ID | Requirement | Source |
|:---|:-----------|:-------|
| REQ-001 | `ListRepositoryFiles` retrieves all file paths in a repo's default branch using Git Trees API (refs -> commit -> tree?recursive=1) | PR body, `forge.go:195-199` |
| REQ-002 | `ComparePathPresence` uses batched file listing (single API call) instead of per-path `GetFileContent` | PR body, `pathpresence.go` |
| REQ-003 | `FakeClient` implements `ListRepositoryFiles` for testing | `fake.go:403-419` |
| REQ-004 | `Harness.Lint()` returns non-fatal `[]Diagnostic` warnings without affecting `Validate()` | `lint.go`, ADR-0045 Phase 3 |
| REQ-005 | `Lint()` warns when `role` is empty on a harness | `lint.go:42-47` |
| REQ-006 | `DiscoverRemoteAgents` discovers agent identity from remote config repo harness files via forge API | `discover_remote.go` |
| REQ-007 | `parseRaw()` helper parses harness YAML from raw bytes without file I/O | `harness.go` refactor |
| REQ-008 | CLI `--mint-url` replaces deprecated `--status-token` for status comment authentication | `run.go`, `reconcilestatus.go`, `action.yml` |
| REQ-009 | `OrgConfig` supports `CreateIssues` configuration for cross-repo issue creation | `config.go` |
| REQ-010 | Status comment reconciliation supports mint-URL token minting | `reconcilestatus.go`, `statuscomment.go` |

---

## 3. Test Scenarios

### 3.1 forge.Client.ListRepositoryFiles (REQ-001, REQ-003)

| ID | Scenario | Expected Result | Tier |
|:---|:---------|:---------------|:-----|
| TS-GH-25-001 | `ListRepositoryFiles` on a repository with files returns all blob paths | Returns `[]string` of all file paths; no tree/directory entries included | Tier1 |
| TS-GH-25-002 | `ListRepositoryFiles` follows the ref chain: default branch -> commit SHA -> tree SHA -> recursive tree | Exactly 3 API calls issued (get repo, get ref, get commit, get tree) | Tier1 |
| TS-GH-25-003 | `ListRepositoryFiles` on a non-existent repository returns `ErrNotFound` | Error wraps `forge.ErrNotFound` | Tier1 |
| TS-GH-25-004 | `ListRepositoryFiles` on a truncated tree (repo too large) returns an error | Returns error containing "truncated" | Tier1 |
| TS-GH-25-005 | `ListRepositoryFiles` on an empty repository returns empty slice | Returns `[]string{}`, no error | Tier1 |
| TS-GH-25-006 | `ListRepositoryFiles` retries on transient failures during ref resolution | Uses `retryOnTransient` for the branch ref API call | Tier1 |
| TS-GH-25-007 | `FakeClient.ListRepositoryFiles` returns paths from `FileContents` map keyed by `owner/repo/path` | Paths returned match keys with `owner/repo/` prefix stripped | Unit |
| TS-GH-25-008 | `FakeClient.ListRepositoryFiles` with injected error returns the error | Error from `Errors["ListRepositoryFiles"]` propagated | Unit |

### 3.2 ComparePathPresence (REQ-002)

| ID | Scenario | Expected Result | Tier |
|:---|:---------|:---------------|:-----|
| TS-GH-25-009 | All expected paths exist in the repository | Returns `nil` missing slice, no error | Unit |
| TS-GH-25-010 | Some expected paths are missing | Returns sorted `[]string` of missing paths | Unit |
| TS-GH-25-011 | All expected paths are missing | Returns sorted slice of all expected paths | Unit |
| TS-GH-25-012 | Empty expected paths slice | Returns `nil, nil` immediately (no API call) | Unit |
| TS-GH-25-013 | `ListRepositoryFiles` returns an error | Error propagated with "listing repository files" context | Unit |
| TS-GH-25-014 | `ComparePathPresence` uses `ListRepositoryFiles` (batch) not per-path `GetFileContent` | Injecting error on `GetFileContent` does not affect result; only `ListRepositoryFiles` is called | Unit |

### 3.3 Harness Lint() Diagnostics (REQ-004, REQ-005)

| ID | Scenario | Expected Result | Tier |
|:---|:---------|:---------------|:-----|
| TS-GH-25-015 | `Lint()` on harness with `role` set returns `nil` | No diagnostics returned | Unit |
| TS-GH-25-016 | `Lint()` on harness with empty `role` returns warning diagnostic | One `SeverityWarning` diagnostic with `Field: "role"` and message containing "required in a future version" | Unit |
| TS-GH-25-017 | `Lint()` on harness with both `role` and `slug` set returns `nil` | No diagnostics returned | Unit |
| TS-GH-25-018 | `Diagnostic.String()` formats warning severity correctly | Returns `"warning: <field>: <message>"` | Unit |
| TS-GH-25-019 | `Diagnostic.String()` formats error severity correctly | Returns `"error: <field>: <message>"` | Unit |
| TS-GH-25-020 | `Diagnostic.String()` formats unknown severity | Returns `"DiagnosticSeverity(N): <field>: <message>"` | Unit |
| TS-GH-25-021 | `Lint()` returns `nil` (not empty slice) when no issues found | `diags == nil` is true, not just `len(diags) == 0` | Unit |

### 3.4 DiscoverRemoteAgents (REQ-006, REQ-007)

| ID | Scenario | Expected Result | Tier |
|:---|:---------|:---------------|:-----|
| TS-GH-25-022 | Multiple harness files in remote `harness/` directory | Returns `[]AgentInfo` sorted by Role then Filename | Unit |
| TS-GH-25-023 | No `harness/` directory exists (`ErrNotFound`) | Returns `(nil, nil)` | Unit |
| TS-GH-25-024 | Files without `role` or `slug` are skipped | Only files with at least one of role/slug are returned | Unit |
| TS-GH-25-025 | File with `role` only (no `slug`) is included | AgentInfo has Role set, Slug empty | Unit |
| TS-GH-25-026 | File with `slug` only (no `role`) is included | AgentInfo has Slug set, Role empty | Unit |
| TS-GH-25-027 | Malformed YAML in one file returns multi-error with valid files | Error contains bad filename; valid AgentInfo still returned | Unit |
| TS-GH-25-028 | `GetFileContentAtRef` failure for one file returns multi-error | Error contains missing filename; valid AgentInfo still returned | Unit |
| TS-GH-25-029 | Empty `harness/` directory | Returns empty slice, no error | Unit |
| TS-GH-25-030 | `.yml` extension files are discovered | Files with `.yml` suffix parsed and returned | Unit |
| TS-GH-25-031 | Non-YAML files (`.md`, `.txt`) are skipped | Only `.yaml`/`.yml` files processed | Unit |
| TS-GH-25-032 | Subdirectories in `harness/` are skipped | Only entries with `Type: "file"` processed | Unit |
| TS-GH-25-033 | Same role sorted by filename for deterministic output | When two agents share a role, sorted alphabetically by Filename | Unit |
| TS-GH-25-034 | Path field in returned AgentInfo is empty (remote agents have no local path) | `AgentInfo.Path` is empty string | Unit |
| TS-GH-25-035 | Path prefix in directory entry is stripped to bare filename | `harness/triage.yaml` entry -> `Filename: "triage.yaml"` | Unit |
| TS-GH-25-036 | `ListDirectoryContents` error propagates | Returns error containing "listing harness directory" | Unit |

### 3.5 Mint-URL Status Token Migration (REQ-008, REQ-010)

| ID | Scenario | Expected Result | Tier |
|:---|:---------|:---------------|:-----|
| TS-GH-25-037 | `fullsend run` with `--mint-url` mints a fresh token for status comments | Status comment uses minted token; no `--status-token` required | Tier1 |
| TS-GH-25-038 | `fullsend run` with deprecated `--status-token` emits deprecation warning | Warning message printed to stderr; command still succeeds | Tier1 |
| TS-GH-25-039 | `fullsend run` with both `--mint-url` and `--status-token` prefers mint-url | Mint-URL is used; status-token is ignored | Tier1 |
| TS-GH-25-040 | `reconcile-status` with `--mint-url` and `--role` mints token successfully | Token minted and used for reconciliation | Tier1 |
| TS-GH-25-041 | `reconcile-status` with `--mint-url` but missing `--role` returns error | Error: "--role is required when using --mint-url" | Tier1 |
| TS-GH-25-042 | `reconcile-status` with deprecated `--token` emits warning | Warning printed to stderr; reconciliation proceeds | Tier1 |
| TS-GH-25-043 | `reconcile-status` with neither `--mint-url` nor `--token` returns error | Error: "--mint-url or FULLSEND_MINT_URL required" | Tier1 |
| TS-GH-25-044 | Action.yml passes `mint-url` input to binary via `MINT_URL` env var | Environment variable set correctly in composite action step | Tier1 |
| TS-GH-25-045 | Finalize orphaned status comment step requires mint-url or status-token | Step `if` condition checks `inputs.mint-url != '' \|\| inputs.status-token != ''` | Tier1 |

### 3.6 OrgConfig CreateIssues (REQ-009)

| ID | Scenario | Expected Result | Tier |
|:---|:---------|:---------------|:-----|
| TS-GH-25-046 | `OrgConfig` with `create_issues.allow_targets` parses correctly | `AllowTargets.Orgs` and `AllowTargets.Repos` populated from YAML | Unit |
| TS-GH-25-047 | `OrgConfig` without `create_issues` section uses empty defaults | `CreateIssues` field is zero-value; no panic | Unit |
| TS-GH-25-048 | `MintURL` field parsed from `dispatch.mint_url` in config | `OrgConfig.Dispatch.MintURL` contains the configured URL | Unit |

### 3.7 Harness Scaffold Integration (Cross-cutting)

| ID | Scenario | Expected Result | Tier |
|:---|:---------|:---------------|:-----|
| TS-GH-25-049 | Scaffold integration test validates harness files against schema | All generated harness wrapper files pass `Validate()` | Tier1 |
| TS-GH-25-050 | `parseRaw()` parses valid YAML bytes into `Harness` struct | Returns populated `*Harness`, no error | Unit |
| TS-GH-25-051 | `parseRaw()` with invalid YAML returns parse error | Returns `nil`, error from `yaml.Unmarshal` | Unit |

---

## 4. Regression Impact Analysis

### 4.1 LSP Call Graph Analysis

The following dependency chains were identified using LSP analysis:

**`forge.Client.ListRepositoryFiles` (new interface method)**
- Defined: `internal/forge/forge.go:199`
- Implemented by: `github.LiveClient` (`internal/forge/github/github.go:957`)
- Implemented by: `forge.FakeClient` (`internal/forge/fake.go:403`)
- Called by: `scaffold.ComparePathPresence` (`internal/scaffold/pathpresence.go:20`)
- Test coverage: `internal/forge/fake_test.go`, `internal/scaffold/pathpresence_test.go`

**`scaffold.ComparePathPresence` (refactored function)**
- Defined: `internal/scaffold/pathpresence.go:15`
- Callers: Test-only at this point (6 test functions in `pathpresence_test.go`)
- No production callers yet — function is new infrastructure for future scaffold operations
- Risk: Low — no existing production code paths affected

**`harness.DiscoverRemoteAgents` (new function)**
- Defined: `internal/harness/discover_remote.go:24`
- Callers: Test-only (15 test sub-cases in `discover_remote_test.go`)
- Depends on: `forge.Client.ListDirectoryContents`, `forge.Client.GetFileContentAtRef`, `harness.parseRaw`
- Risk: Low — new function with no production callers; designed for Phase 3 migration

**`harness.Lint()` (new method)**
- Defined: `internal/harness/lint.go:40`
- Operates on: `*Harness` struct (250 references across 21 files)
- Callers: Test-only (3 test sub-cases in `lint_test.go`)
- Risk: Very low — additive method, does not modify `Validate()` behavior

### 4.2 Regression Risk Areas

| Area | Risk | Rationale |
|:-----|:-----|:----------|
| `forge.Client` interface | **Medium** | New `ListRepositoryFiles` method added — all implementations (LiveClient, FakeClient, any external mocks) must implement it. Compile-time check via `var _ Client = (*)` guards this. |
| `ComparePathPresence` | **Low** | New function, no existing callers to break. |
| `Harness.Lint()` | **Very Low** | Additive method on existing struct. `Validate()` unchanged. |
| `DiscoverRemoteAgents` | **Low** | New function. Depends on existing forge API methods that are already tested. |
| `action.yml` mint-url migration | **Medium** | Existing `status-token` input deprecated. Workflows passing `status-token` still work but get deprecation warning. New `mint-url` input requires mint service availability. |
| `reconcile-status` CLI | **Medium** | Token acquisition logic refactored. Deprecated `--token` flag still functional but emits warning. Missing `--role` with `--mint-url` now errors. |
| `OrgConfig` struct changes | **Low** | New fields added with `omitempty`; existing configs without new fields parse without error. |
| `harness.parseRaw` refactor | **Low** | `LoadRaw` refactored to call `parseRaw` internally. Same behavior, just extracted. |

---

## 5. Components Affected

| Component | Package Path | Changes |
|:----------|:------------|:--------|
| Code Generation (Forge) | `internal/forge/` | New `ListRepositoryFiles` interface method + FakeClient implementation |
| Code Generation (Forge/GitHub) | `internal/forge/github/` | `LiveClient.ListRepositoryFiles` using Git Trees API |
| Repo Scaffolding | `internal/scaffold/` | New `ComparePathPresence` + `pathpresence_test.go` |
| Agent Harness | `internal/harness/` | `Lint()`, `DiscoverRemoteAgents`, `parseRaw`, scaffold integration test |
| CLI Commands | `internal/cli/` | `run.go` (mint-url), `reconcilestatus.go` (mint-url + role), `admin.go`, `github.go` |
| Configuration | `internal/config/` | `CreateIssues`, `MintURL` fields in OrgConfig |
| Status Comments | `internal/statuscomment/` | Mint-URL token support |

---

## 6. Out of Scope

The following are explicitly out of scope for this test plan:

- **Upstream fullsend-ai/fullsend repo testing** — this is a mirror PR; upstream has its own test pipeline
- **End-to-end GitHub API integration tests** — `ListRepositoryFiles` LiveClient tested via unit tests with httptest mocking
- **Phase 4 of ADR-0045** — requiring `role` in `Validate()`, removing `agents:` block (future work)
- **Wiring `Lint()` into `fullsend run`/`fullsend lock`** — PR 3 in the plan (not in this PR)
- **Migrating `loadKnownSlugs`/uninstall to `DiscoverRemoteAgents`** — PRs 4-5 in the plan (not in this PR)
- **Documentation-only changes** (ADR updates, plan docs, triage docs, guides) — informational, not testable
- **Workflow YAML changes** (reusable-*.yml status-token -> mint-url) — CI config, tested via action.yml integration

---

## 7. Test Execution Summary

| Tier | Count | Description |
|:-----|:------|:-----------|
| Unit | 33 | Pure function/method tests with mock/fake dependencies |
| Tier1 | 18 | Functional tests requiring CLI flag parsing, action.yml integration, scaffold integration |
| **Total** | **51** | |

---

## 8. Existing Test Coverage

The PR already includes comprehensive test files:

| Test File | Tests | Status |
|:----------|:------|:-------|
| `internal/forge/fake_test.go` | `ListRepositoryFiles` fake behavior | Included in PR |
| `internal/scaffold/pathpresence_test.go` | 6 test functions covering all `ComparePathPresence` paths | Included in PR |
| `internal/harness/lint_test.go` | 6 test sub-cases for `Lint()` and `Diagnostic.String()` | Included in PR |
| `internal/harness/discover_remote_test.go` | 15 test sub-cases covering all `DiscoverRemoteAgents` paths | Included in PR |
| `internal/harness/scaffold_integration_test.go` | Integration test for scaffold harness generation | Included in PR |
| `internal/cli/run_test.go` | Extended with mint-url flag tests | Included in PR |
| `internal/cli/reconcilestatus_test.go` | Extended with mint-url/role/token tests | Included in PR |
| `internal/config/config_test.go` | Extended with `CreateIssues` and `MintURL` parsing tests | Included in PR |
| `internal/statuscomment/statuscomment_test.go` | Extended with mint-URL token support tests | Included in PR |

---

*Generated by QualityFlow STP Builder | 2026-06-17*
