# FullSend Test Plan

| Field | Value |
|:------|:------|
| **Ticket** | GH-18 |
| **Title** | Expand supply chain threat to cover model-as-toolchain risk |
| **Author** | ralphbean |
| **Status** | Merged |
| **Date** | 2026-06-16 |
| **Product** | FullSend |
| **Platform** | GitHub Actions |
| **Version** | 0.x |

---

## 1. Summary

This test plan covers GH-18, which expands Threat 4 (supply chain attacks) in the
FullSend security threat model documentation to address the "model as toolchain" risk.
The PR introduces the Thompson's "Reflections on Trusting Trust" analog for agentic
development, adds a trust boundary table, proposes defenses (model diversity, authorship
provenance, intent attestation, property-based testing, periodic model rotation), and
documents how this threat cuts across all other threat categories.

**Change scope:** Documentation only — `docs/problems/security-threat-model.md` (+51/−3 lines).

**Testing rationale:** Although the PR modifies only documentation, the threat model
describes security controls that are implemented in code. This test plan validates that
existing security controls referenced in the expanded threat model continue to function
correctly, ensuring the documented security posture reflects the actual system behavior.

---

## 2. PR Analysis

| Attribute | Detail |
|:----------|:-------|
| **PR** | [#18](https://github.com/fullsend-ai/fullsend/pull/18) |
| **Branch** | `worktree-thompson-trust-threat-model` → `main` |
| **Files changed** | 1 (`docs/problems/security-threat-model.md`) |
| **Lines** | +51 / −3 |
| **Change type** | Documentation enhancement |

### Key Changes

1. **Threat 4 reframing** — Supply chain attacks section expanded from a brief summary to a comprehensive analysis of model-as-toolchain risk
2. **Trust boundary table** — New table distinguishing Source→Binary (covered), Intent→Source human (covered), and Intent→Source agent (partially covered)
3. **Model-as-toolchain defenses** — Five defense strategies documented: model diversity, authorship provenance, intent attestation, property-based testing, model rotation
4. **Cross-cutting analysis** — New section mapping model-as-toolchain risk across prompt injection, insider threat, drift, and agent-to-agent injection
5. **Open questions** — Five new open questions about model integrity verification, diversity costs, provenance requirements, intent attestation practicality, and trust claim extension

### Review Insights

A contributor comment (arewm) validated the trust boundary framing and identified
concrete integration points with devaipod for authorship provenance via SLSA build
provenance predicates, and model diversity via sidecar profiles.

---

## 3. Regression Analysis (LSP)

LSP analysis was performed against the Go source tree to trace the security components
referenced in the expanded threat model documentation.

### 3a. Security Hook Pipeline

**File:** `internal/security/hooks.go`

| Symbol | Type | Line | Purpose |
|:-------|:-----|:-----|:--------|
| `GenerateClaudeSettings` | Function | 57 | Generates security hook configuration for agent sandboxes |
| `HookFiles` | Function | 161 | Returns embedded hook script files |
| `tirithEnabled` | Function | 201 | Toggle: Tirith policy evaluation hook |
| `ssrfPreToolEnabled` | Function | 208 | Toggle: SSRF pre-tool validation hook |
| `secretRedactPostToolEnabled` | Function | 215 | Toggle: Secret redaction post-tool hook |
| `unicodePostToolEnabled` | Function | 222 | Toggle: Unicode normalization post-tool hook |
| `contextSuppressPostToolEnabled` | Function | 229 | Toggle: Context suppression post-tool hook |
| `canaryPreToolEnabled` | Function | 236 | Toggle: Canary pre-tool detection hook |
| `canaryPostToolEnabled` | Function | 243 | Toggle: Canary post-tool detection hook |
| `toolAllowlistPreToolEnabled` | Function | 250 | Toggle: Tool allowlist pre-tool hook |

**Call chain:** `bootstrapSecurityHooks` (internal/cli/run.go:1308) → `GenerateClaudeSettings` (hooks.go:57)

### 3b. Security Scanner Pipeline

**File:** `internal/security/scanner.go`

| Symbol | Type | Line | Purpose |
|:-------|:-----|:-----|:--------|
| `Pipeline` | Struct | 44 | Chains scanners sequentially, fail-closed for safety |
| `InputPipeline` | Function | 86 | UnicodeNormalizer → ContextInjectionScanner |
| `OutputPipeline` | Function | 96 | SecretRedactor chain |
| `HasCriticalFindings` | Function | 103 | Checks for critical severity findings |

### 3c. Injection Scanner

**File:** `internal/security/injection.go`

| Symbol | Type | Line | Purpose |
|:-------|:-----|:-----|:--------|
| `ContextInjectionScanner` | Struct | 13 | Detects prompt injection patterns |
| `NewContextInjectionScanner` | Function | 25 | Constructor |
| `ShouldScan` | Function | 75 | Determines if a file should be scanned |

**Callers of `NewContextInjectionScanner`:** `InputPipeline` (scanner.go:89), `scan.go:223`, scanner_test.go

### 3d. Harness Security Configuration

**File:** `internal/harness/harness.go`

| Symbol | Type | Line | Purpose |
|:-------|:-----|:-----|:--------|
| `SecurityConfig` | Struct | 82 | Top-level security configuration |
| `SandboxHooks` | Struct | 113 | Sandbox-level hook toggles |
| `HostScanners` | Struct | 94 | Host-level scanner toggles |
| `ProviderDef` | Struct | 38 | Model provider definition (supports diversity) |
| `SecurityEnabled` | Method | 157 | Checks if security is enabled |
| `FailModeClosed` | Method | 165 | Checks fail-closed mode |

### 3e. Dependency Chains

```
bootstrapSecurityHooks (cli/run.go)
  └─ GenerateClaudeSettings (security/hooks.go)
       ├─ tirithEnabled
       ├─ ssrfPreToolEnabled
       ├─ secretRedactPostToolEnabled
       ├─ unicodePostToolEnabled
       ├─ contextSuppressPostToolEnabled
       ├─ canaryPreToolEnabled
       ├─ canaryPostToolEnabled
       └─ toolAllowlistPreToolEnabled

InputPipeline (security/scanner.go)
  ├─ NewUnicodeNormalizer
  └─ NewContextInjectionScanner (security/injection.go)

OutputPipeline (security/scanner.go)
  └─ NewSecretRedactor
```

---

## 4. Requirements-to-Tests Mapping

### Validated Requirements

| # | Requirement ID | Requirement Summary | Source | Evidence | Test Scenario | Tier | Priority |
|:--|:---------------|:-------------------|:-------|:---------|:-------------|:-----|:---------|
| 1 | GH-18 | Security hook pipeline generates correct configuration for all toggle combinations | regression_analysis | `GenerateClaudeSettings` → 8 toggle functions, called from `bootstrapSecurityHooks` | Verify hook settings generated for all toggle combinations | Tier1 | P0 |
| 2 | | Input pipeline chains scanners in correct order (normalize → detect) | regression_analysis | `InputPipeline` → `UnicodeNormalizer` → `ContextInjectionScanner` | Verify input pipeline scanner ordering | Tier1 | P0 |
| 3 | | Output pipeline redacts secrets from agent output | regression_analysis | `OutputPipeline` → `SecretRedactor` | Verify output pipeline redacts sensitive content | Tier1 | P0 |
| 4 | | Context injection scanner detects prompt injection patterns | regression_analysis | `ContextInjectionScanner.Scan` with `defaultPatterns` | Verify injection patterns detected in untrusted input | Tier1 | P0 |
| 5 | | Pipeline fails closed when any scanner marks input unsafe | regression_analysis | `Pipeline.Scan` — fail-closed aggregation logic (scanner.go:66) | Verify pipeline marks result unsafe on any scanner failure | Tier1 | P0 |
| 6 | | Harness supports multiple provider configurations for model diversity | regression_analysis | `ProviderDef` struct, `LoadProviderDefs` function | Verify multiple model providers can be configured | Tier1 | P1 |
| 7 | | Security configuration validates correctly with fail-closed default | regression_analysis | `SecurityEnabled`, `FailModeClosed`, `validateSecurity` | Verify security defaults to fail-closed mode | Tier1 | P1 |
| 8 | | Individual sandbox hook toggles enable/disable correctly | regression_analysis | 8 `*Enabled` functions with `boolDefault` | Verify each hook toggle respects explicit enable/disable | Tier1 | P1 |
| 9 | | Unicode normalizer strips invisible characters before injection scan | regression_analysis | `InputPipeline` ordering: normalizer first, scanner second | Verify invisible Unicode stripped before injection detection | Tier1 | P0 |
| 10 | | Scanner pipeline propagates sanitized output between stages | regression_analysis | `Pipeline.Scan` passes `result.Sanitized` to next scanner | Verify sanitized text feeds into subsequent scanner | Tier1 | P1 |

### Negative / Error Scenarios

| # | Requirement ID | Requirement Summary | Source | Evidence | Test Scenario | Tier | Priority |
|:--|:---------------|:-------------------|:-------|:---------|:-------------|:-----|:---------|
| 11 | GH-18 | Pipeline handles scanner returning no findings gracefully | regression_analysis | `Pipeline.Scan` — empty findings aggregation | Verify pipeline returns safe when no findings | Tier1 | P1 |
| 12 | | Hook generation handles nil security config | regression_analysis | `GenerateClaudeSettings` receives `*harness.Harness` | Verify hook generation handles missing security config | Tier1 | P1 |
| 13 | | Injection scanner handles empty input | regression_analysis | `ContextInjectionScanner.Scan` with empty string | Verify scanner handles empty string input | Tier1 | P2 |
| 14 | | Pipeline handles scanner returning critical findings | regression_analysis | `HasCriticalFindings` checks severity=="critical" | Verify critical findings correctly identified | Tier1 | P1 |

### Rejected Requirements

| Requirement Summary | Reason | Gate Failed |
|:-------------------|:-------|:------------|
| Verify SLSA provenance chain for builds | Platform-level supply chain verification — tested by build platform team | Requirement Level Validation |
| Verify hermetic build isolation | Build infrastructure concern — tested by build system team | Requirement Level Validation |
| Verify Enterprise Contract policy evaluation | External policy engine — tested by policy platform team | Requirement Level Validation |
| Verify model training data integrity | Model provider responsibility — outside FullSend product scope | Requirement Level Validation |

---

## 5. Test Scenarios

### TS-GH-18-001: Security Hook Pipeline Configuration

**Tier:** Tier1 (Unit)
**Priority:** P0
**Requirement:** #1

| Scenario | Type | Description |
|:---------|:-----|:------------|
| TS-GH-18-001a | Positive | Verify all hooks enabled with default security config |
| TS-GH-18-001b | Positive | Verify individual hook disabled when toggle set false |
| TS-GH-18-001c | Positive | Verify all hooks disabled when all toggles false |
| TS-GH-18-001d | Negative | Verify hook generation with nil security config |
| TS-GH-18-001e | Positive | Verify tool allowlist hook enabled when toggle set true |

### TS-GH-18-002: Input Pipeline Integrity

**Tier:** Tier1 (Unit)
**Priority:** P0
**Requirement:** #2, #9

| Scenario | Type | Description |
|:---------|:-----|:------------|
| TS-GH-18-002a | Positive | Verify input pipeline normalizes then scans |
| TS-GH-18-002b | Positive | Verify invisible Unicode stripped before injection scan |
| TS-GH-18-002c | Positive | Verify sanitized output propagated between stages |
| TS-GH-18-002d | Negative | Verify pipeline fails closed on injection detection |

### TS-GH-18-003: Output Pipeline Redaction

**Tier:** Tier1 (Unit)
**Priority:** P0
**Requirement:** #3

| Scenario | Type | Description |
|:---------|:-----|:------------|
| TS-GH-18-003a | Positive | Verify API keys redacted from agent output |
| TS-GH-18-003b | Positive | Verify tokens redacted from agent output |
| TS-GH-18-003c | Negative | Verify clean text passes through unchanged |

### TS-GH-18-004: Context Injection Detection

**Tier:** Tier1 (Unit)
**Priority:** P0
**Requirement:** #4

| Scenario | Type | Description |
|:---------|:-----|:------------|
| TS-GH-18-004a | Positive | Verify known injection patterns detected |
| TS-GH-18-004b | Positive | Verify severity correctly assigned per pattern |
| TS-GH-18-004c | Negative | Verify clean text returns safe result |
| TS-GH-18-004d | Negative | Verify empty input returns safe result |

### TS-GH-18-005: Pipeline Fail-Closed Behavior

**Tier:** Tier1 (Unit)
**Priority:** P0
**Requirement:** #5, #14

| Scenario | Type | Description |
|:---------|:-----|:------------|
| TS-GH-18-005a | Positive | Verify pipeline safe when all scanners safe |
| TS-GH-18-005b | Negative | Verify pipeline unsafe when any scanner unsafe |
| TS-GH-18-005c | Positive | Verify critical findings correctly flagged |
| TS-GH-18-005d | Positive | Verify findings aggregated across all scanners |

### TS-GH-18-006: Model Provider Diversity Support

**Tier:** Tier1 (Unit)
**Priority:** P1
**Requirement:** #6

| Scenario | Type | Description |
|:---------|:-----|:------------|
| TS-GH-18-006a | Positive | Verify multiple providers loaded from config |
| TS-GH-18-006b | Positive | Verify provider credentials mapped correctly |
| TS-GH-18-006c | Negative | Verify error for invalid provider definition |

### TS-GH-18-007: Security Configuration Defaults

**Tier:** Tier1 (Unit)
**Priority:** P1
**Requirement:** #7, #8

| Scenario | Type | Description |
|:---------|:-----|:------------|
| TS-GH-18-007a | Positive | Verify security defaults to fail-closed |
| TS-GH-18-007b | Positive | Verify security enabled by default |
| TS-GH-18-007c | Positive | Verify each hook toggle respects explicit value |
| TS-GH-18-007d | Positive | Verify boolean defaults applied for nil toggles |

---

## 6. Test Count Summary

| Tier | Count |
|:-----|:------|
| Tier1 (Unit) | 26 |
| Tier2 (E2E) | 0 |
| **Total** | **26** |

| Priority | Count |
|:---------|:------|
| P0 | 16 |
| P1 | 10 |
| P2 | 0 |

---

## 7. Environment Requirements

| Requirement | Value |
|:------------|:------|
| Platform | GitHub Actions |
| Go version | 1.23+ |
| Test framework | `testing` + `testify` |
| Build command | `go test ./...` |
| CLI tools | `fullsend`, `gh`, `go` |

---

## 8. Out of Scope

The following items are explicitly out of scope for this test plan:

- **SLSA provenance verification** — Tested by build platform team
- **Hermetic build isolation** — Build infrastructure concern
- **Model training data integrity** — Model provider responsibility
- **Enterprise Contract policy evaluation** — External policy engine testing
- **Kubernetes platform primitives** — Not applicable (GitHub Actions platform)
- **End-to-end integration tests** — No code changes to integrate; documentation-only PR

---

## 9. Notes

1. **Documentation-driven testing:** This PR modifies only the security threat model
   document, but the expanded content references specific security controls implemented
   in the codebase. The test scenarios validate those controls continue to function as
   documented.

2. **Model-as-toolchain coverage:** The threat model introduces the concept of model
   diversity as a defense. The existing `ProviderDef` and `LoadProviderDefs` mechanisms
   in `internal/harness/harness.go` provide the infrastructure for this defense. Test
   scenarios TS-GH-18-006a-c validate this capability.

3. **LSP-traced call chains:** All test scenarios are grounded in LSP-traced call chains
   from the security subsystem. The primary entry point `bootstrapSecurityHooks` →
   `GenerateClaudeSettings` was traced through 8 toggle functions with 11 references
   across 3 files.

4. **Existing test coverage:** The hooks_test.go file already contains 9 test functions
   covering `GenerateClaudeSettings` toggle combinations. New scenarios should complement
   rather than duplicate existing coverage.

---

*Generated by QualityFlow STP Builder | 2026-06-16*
