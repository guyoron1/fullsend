# Harness field integration checklist

When adding or modifying fields in the harness schema structs (`internal/harness/harness.go`, `internal/harness/forge.go`), the new field must be wired into up to four integration pipelines. Missing any of these is a common review gap — trace the code path of an existing sibling field (e.g., `PreScript`, `RunnerEnv`, `Skills`) to verify each one.

## 1. Expansion pipeline

**Files:** `internal/cli/run.go`, `internal/harness/harness.go`

Fields whose values may contain `${VAR}` references must be expanded at runtime and validated at load time.

- **Expand:** In `run.go`, the `expander` function calls `os.Expand(value, expander)` to substitute host environment variables. Currently applied to each value in `RunnerEnv` (run.go) and to `HostFiles[].Src` via `os.ExpandEnv` (run.go `bootstrapEnv`).
- **Validate:** In `harness.go`, `ValidateRunnerEnvWith` walks field values with `checkVarRefs`, which uses the `envVarRef` regex to find `${VAR}` patterns and confirms each variable is set in the host environment. Currently covers `RunnerEnv` values and `HostFiles[].Src`.

**When to add your field:** If users can write `${VAR}` in the field value and expect it to be substituted from the host environment.

**What to do:**
1. Add `os.Expand(value, expander)` (or `os.ExpandEnv`) for the field in `run.go`, after `ValidateRunnerEnvWith` and before `ValidateFilesExist`.
2. Add a `checkVarRefs` call for the field inside `ValidateRunnerEnvWith` in `harness.go`.

## 2. Environment construction

**Files:** `internal/cli/run.go`

Host-side commands (scripts that run on the runner, outside the sandbox) must inherit `RunnerEnv` so they see the same environment the harness author configured.

- **Pre-script** (run.go): `preCmd.Env = append(os.Environ(), envToList(h.RunnerEnv)...)`
- **Post-script** (run.go): `postCmd.Env = append(os.Environ(), envToList(h.RunnerEnv)...)`
- **Validation loop script** (run.go): `valCmd.Env = append(os.Environ(), append(envToList(h.RunnerEnv), ...)...)`

**When to add your field:** If the field references a script or command that runs on the host (outside the sandbox).

**What to do:** Merge `h.RunnerEnv` into the command's environment via `envToList(h.RunnerEnv)`, following the pattern of the existing script invocations above.

## 3. Composition carry-forward

**Files:** `internal/harness/compose.go`, `internal/harness/forge.go`

The harness supports two composition mechanisms: base inheritance and forge-platform overrides. New fields must be carried through all applicable merge sites or they are silently dropped during composition.

### Merge sites

| Site | Function | File | Purpose |
|------|----------|------|---------|
| Base composition | `mergeBaseIntoChild` | `compose.go` | Merges a base harness into a child harness |
| Forge base composition | `mergeForgeConfigInto` | `compose.go` | Merges a base `ForgeConfig` into a child `ForgeConfig` during base chain resolution |
| Forge resolution | `mergeForgeConfig` | `forge.go` | Merges a platform-specific `ForgeConfig` into the top-level `Harness` |

### Merge rules (ADR-0045)

| Field type | Rule | Example fields |
|------------|------|----------------|
| Scalars | Child overrides base if non-zero | `Agent`, `PreScript`, `PostScript`, `Model` |
| Slices | Concatenated (base + child) | `Skills`, `Plugins`, `Providers` |
| Maps | Merged; child keys win | `RunnerEnv` |
| Pointer structs | Child replaces if non-nil | `ValidationLoop`, `Security` |
| Security-sensitive | NOT merged (child must declare its own) | `AllowedRemoteResources`, `AllowRuntimeFetch` |

**When to add your field:** Always, for any field added to `Harness` or `ForgeConfig`.

**What to do:**
1. **`Harness` field:** Add the merge logic to `mergeBaseIntoChild` in `compose.go`, following the rule for the field's type (see table above).
2. **`ForgeConfig` field:** Add the merge logic to both `mergeForgeConfig` in `forge.go` (forge resolution) and `mergeForgeConfigInto` in `compose.go` (base composition).
3. If the field is security-sensitive (grants access or expands trust), explicitly document that it is NOT merged and add a comment explaining why, following the `AllowedRemoteResources` pattern.

## 4. Path and security validation

**Files:** `internal/harness/harness.go`

Fields that reference file paths or URLs go through a three-step validation pipeline.

| Step | Function | Purpose |
|------|----------|---------|
| Path resolution | `ResolveRelativeTo` | Resolves relative paths against the fullsend directory; rejects paths that escape via `../` |
| File existence | `ValidateFilesExist` | Confirms referenced files exist on disk (skips URLs and `${VAR}` paths) |
| Resource types | `ValidateResourceTypes` | Enforces that executable fields (scripts) are local paths (not URLs) and that declarative URL fields include `#sha256=...` integrity hashes |

**When to add your field:** If the field contains a file path or URL.

**What to do:**
1. Add the field to `ResolveRelativeTo` if it is a relative path that should be resolved against the base directory.
2. Add the field to `ValidateFilesExist` if the file must exist at load time.
3. Add the field to `ValidateResourceTypes`: executable fields must reject URLs; declarative URL fields must require integrity hashes.

## Quick reference

When adding a new field, check each row:

| Pipeline | Applies when | Key functions |
|----------|-------------|---------------|
| Expansion | Field value may contain `${VAR}` | `os.Expand`, `checkVarRefs` in `ValidateRunnerEnvWith` |
| Environment | Field is a host-side script | `envToList(h.RunnerEnv)` in `run.go` |
| Composition | Always | `mergeBaseIntoChild`, `mergeForgeConfigInto`, `mergeForgeConfig` |
| Path/security | Field is a path or URL | `ResolveRelativeTo`, `ValidateFilesExist`, `ValidateResourceTypes` |
