# FullSend Test Plan

| Field | Value |
|:------|:------|
| **Ticket** | GH-10 |
| **Title** | fix(#2294): make EnsureProvider idempotent via delete-and-recreate |
| **Type** | Bug Fix |
| **Product** | FullSend |
| **Platform** | GitHub Actions |
| **Version** | 0.x |
| **Date** | 2026-06-15 |
| **Author** | QualityFlow |

---

## 1. Overview

This test plan covers the changes introduced in GH-10 (mirrored from upstream PR #2296), which fixes a bug where `EnsureProvider` in `internal/sandbox/sandbox.go` treated an `AlreadyExists` error from `openshell provider create` as a hard failure. This blocked subsequent agent runs without manual cleanup.

The fix makes `EnsureProvider` idempotent by detecting the `AlreadyExists` error, deleting the stale provider, and recreating it with current credentials. A `redactSecrets` helper function was also extracted to reduce duplication in error formatting paths.

### 1.1 Scope

**In Scope:**
- Idempotent behavior of `EnsureProvider` when a provider already exists
- Delete-and-recreate flow for stale providers
- Error handling for delete failures during recreate
- Error handling for non-AlreadyExists failures (unchanged behavior)
- Secret redaction in all error paths via the new `redactSecrets` helper
- Integration with `runAgent` in `internal/cli/run.go` (caller)

**Out of Scope:**
- `EnsureGateway` (not modified, already idempotent)
- `EnsureAvailable` (not modified)
- Sandbox create/delete/exec operations (not modified)
- openshell CLI behavior (external dependency)
- Provider credential expansion via `buildProviderArgs` (not modified)

### 1.2 Risk Assessment

| Risk | Severity | Mitigation |
|:-----|:---------|:-----------|
| Delete succeeds but recreate fails, leaving no provider | High | Error message includes context ("failed after delete") for debugging; caller (`runAgent`) surfaces the error to the user |
| AlreadyExists substring match is too broad | Low | The string `"AlreadyExists"` is a well-defined gRPC/openshell status; substring match is consistent with openshell error format |
| Secret values leak in error messages | High | All error paths now use `redactSecrets`; unit tests verify redaction |
| Race condition if two runs try to recreate simultaneously | Medium | EnsureProvider is called sequentially per provider in `runAgent`; concurrent fullsend runs on same gateway are not a supported pattern |

---

## 2. Requirements Mapping

| Req ID | Requirement | Source | Acceptance Criteria |
|:-------|:------------|:-------|:--------------------|
| REQ-001 | EnsureProvider must succeed when a provider with the same name already exists | PR body, issue #2294 | Calling EnsureProvider twice with the same name completes without error |
| REQ-002 | Stale credentials must be replaced when provider is recreated | PR body | After recreate, provider uses current (not cached) credentials |
| REQ-003 | Delete failure during recreate must produce a clear, redacted error | PR diff (line 68) | Error message contains "provider delete", "during recreate", and no secret values |
| REQ-004 | Non-AlreadyExists errors must still fail with original behavior | PR diff (line 77) | Error message contains "provider create" and the original (redacted) output |
| REQ-005 | Secret values must never appear in any error message | PR body, redactSecrets extraction | All error paths pass output through redactSecrets |

---

## 3. Test Scenarios

### 3.1 Unit Tests — `internal/sandbox/`

| Test ID | Scenario | Type | Requirement | Expected Result |
|:--------|:---------|:-----|:------------|:----------------|
| TS-GH-10-001 | EnsureProvider succeeds on first call (no conflict) | Unit | REQ-001 | Returns nil; openshell called once with correct args |
| TS-GH-10-002 | EnsureProvider detects AlreadyExists, deletes, and recreates successfully | Unit | REQ-001, REQ-002 | Returns nil; openshell called 3 times (create→delete→create) |
| TS-GH-10-003 | EnsureProvider returns error when delete fails during recreate | Unit | REQ-003 | Returns error containing "provider delete" and "during recreate" |
| TS-GH-10-004 | EnsureProvider returns error for non-AlreadyExists create failure | Unit | REQ-004 | Returns error containing "provider create" and original error text |
| TS-GH-10-005 | redactSecrets replaces all known secret values with "***" | Unit | REQ-005 | Output contains "***" in place of each secret; no secret values present |
| TS-GH-10-006 | redactSecrets with empty secrets list returns input unchanged | Unit | REQ-005 | Output equals input string |
| TS-GH-10-007 | redactSecrets with multiple secrets redacts all of them | Unit | REQ-005 | All secret values replaced |
| TS-GH-10-008 | Error from failed recreate (create after delete fails) includes redacted output | Unit | REQ-003, REQ-005 | Error contains "failed after delete" and no secret values |

### 3.2 Functional Tests — CLI Integration

| Test ID | Scenario | Type | Requirement | Expected Result |
|:--------|:---------|:-----|:------------|:----------------|
| TS-GH-10-009 | `runAgent` with provider that already exists completes successfully | Tier1 | REQ-001 | Agent run proceeds past provider setup without error |
| TS-GH-10-010 | `runAgent` with provider create failure (not AlreadyExists) aborts with clear error | Tier1 | REQ-004 | Agent run aborts; error message visible to user; no secret leakage |
| TS-GH-10-011 | Sequential `runAgent` calls reuse providers without manual cleanup | Tier1 | REQ-001, REQ-002 | Both runs complete; second run recreates provider with fresh credentials |

---

## 4. Regression Impact Analysis

### 4.1 Call Graph

```
internal/cli/run.go
  └─ newRunCmd()                    [CLI entry point]
       └─ runAgent()                [line 66]
            └─ sandbox.EnsureProvider()  [line 200] ← MODIFIED
                 ├─ buildProviderArgs()  [line 56, unchanged]
                 └─ redactSecrets()      [line 68, 72, 77] ← NEW
```

### 4.2 Affected Components

| Component | Package | Impact | Risk |
|:----------|:--------|:-------|:-----|
| Sandbox | `internal/sandbox/` | Direct — `EnsureProvider` modified, `redactSecrets` added | High |
| CLI | `internal/cli/` | Indirect — `runAgent` calls `EnsureProvider`; error propagation changed | Medium |
| Harness | `internal/harness/` | None — `LoadProviderDefs` only provides data to `EnsureProvider` | None |

### 4.3 Regression Test Coverage

The following existing tests exercise paths adjacent to the change and should continue to pass:

| Existing Test | File | Relevance |
|:-------------|:-----|:----------|
| `TestBuildProviderArgs_BareKeyCredentials` | `sandbox_test.go` | Validates credential handling upstream of EnsureProvider |
| `TestBuildProviderArgs_KeyRemapping` | `sandbox_test.go` | Validates credential key mapping |
| `TestBuildProviderArgs_EmptyCredential` | `sandbox_test.go` | Validates empty credential edge case |
| `TestEnsureAvailable_OpenshellNotInPath` | `sandbox_test.go` | Validates openshell availability check (unchanged) |

### 4.4 New Tests Added in This PR

| Test Function | Coverage |
|:-------------|:---------|
| `TestEnsureProvider_AlreadyExists_DeleteAndRecreate` | Happy path: AlreadyExists → delete → recreate succeeds |
| `TestEnsureProvider_AlreadyExists_DeleteFails` | Error path: AlreadyExists → delete fails → returns "during recreate" error |
| `TestEnsureProvider_CreateFails_NotAlreadyExists` | Error path: non-AlreadyExists error → returns original error |
| `TestRedactSecrets` | Utility: verifies secret replacement in strings |

---

## 5. Test Environment

| Requirement | Value |
|:------------|:------|
| **Platform** | GitHub Actions |
| **Go Version** | 1.23+ |
| **Test Framework** | `testing` + `testify` (assert, require) |
| **Build Command** | `go test ./internal/sandbox/...` |
| **External Dependencies** | None (tests use fake openshell scripts via `t.TempDir()` + `t.Setenv("PATH", ...)`) |

---

## 6. Test Execution Plan

### Phase 1: Unit Tests (Automated)

```bash
go test -v -count=1 ./internal/sandbox/... -run "TestEnsureProvider|TestRedactSecrets"
```

**Pass criteria:** All 4 new tests pass. All existing `TestBuildProviderArgs*` and `TestEnsureAvailable*` tests continue to pass.

### Phase 2: Integration Validation (Manual / E2E)

1. Run a fullsend agent against a gateway where the provider already exists from a prior run
2. Verify the run completes without manual cleanup
3. Verify credentials are refreshed (check provider config on gateway)
4. Run a fullsend agent with invalid credentials and verify the error message does not contain secret values

---

## 7. Summary

| Metric | Count |
|:-------|:------|
| **Total Test Scenarios** | 11 |
| **Unit Tests** | 8 |
| **Tier1 (Functional)** | 3 |
| **Requirements Covered** | 5/5 (100%) |
| **Changed Files** | 2 (`sandbox.go`, `sandbox_test.go`) |
| **Net Lines Changed** | +106 / -5 |
