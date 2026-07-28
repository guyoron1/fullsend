# Harness field integration checklist

When adding or modifying fields in the harness schema structs
(`internal/harness/harness.go`), every new field must be routed through
the integration pipeline described below. Existing sibling fields
(e.g. `PreScript`, `RunnerEnv`, `ValidationLoop.Script`) already go
through each of these steps. Trace one of them end-to-end before adding
your own.

## 1. Expansion pipeline

**Files:** `internal/cli/run.go`, `internal/harness/harness.go`

Fields whose values may contain `${VAR}` references must participate in
two steps during harness loading:

1. **Var-ref validation** (`ValidateRunnerEnvWith` /
   `checkVarRefs` in `harness.go`). Ensures every `${VAR}` reference
   resolves to a defined host environment variable before the run
   starts. Register the new field's value in the `checkVarRefs` loop
   so unset variables fail early.

2. **Expansion** (`os.Expand` in `run.go`). After validation, each
   value is expanded against the host environment (with
   `FULLSEND_DIR` injected). Add an `os.Expand` call for the new
   field alongside the existing `RunnerEnv` expansion block.

Current fields through this pipeline:

| Field | Validation | Expansion |
|---|---|---|
| `RunnerEnv` values | `checkVarRefs` (harness.go) | `os.Expand` (run.go) |
| `HostFiles[].Src` | `checkVarRefs` (harness.go) | `os.ExpandEnv` (run.go) |

## 2. Environment construction

**File:** `internal/cli/run.go`

Host-side commands (pre-script, post-script, validation script) must
see the harness's `RunnerEnv` variables. Each command's `Env` field is
set with:

```go
cmd.Env = append(os.Environ(), envToList(h.RunnerEnv)...)
```

If the new field introduces a host-side command (a script that runs on
the runner, not inside the sandbox), its `exec.Cmd` must merge
`RunnerEnv` via `envToList`.

Current merge sites:

| Command | Location |
|---|---|
| `PreScript` | run.go — `preCmd.Env` |
| `PostScript` | run.go — `postCmd.Env` |
| `ValidationLoop.Script` | run.go — `valCmd.Env` |

## 3. Composition carry-forward

**Files:** `internal/harness/compose.go`, `internal/harness/forge.go`

Three merge functions must know about new fields. If a field is not
handled, it silently drops during base composition or forge resolution.

### 3a. `mergeBaseIntoChild` (compose.go)

Merges a base harness into the child. Rules per ADR-0045:

- **Scalars** — child overrides if non-zero
- **Slices** (`Skills`, `Plugins`, `Providers`, `APIServers`) —
  base + child (concatenated)
- **Maps** (`RunnerEnv`) — base merged with child; child keys win
- **Pointer structs** (`ValidationLoop`, `Security`) — child
  replaces if non-nil
- **`HostFiles`** — concatenated, last-writer-wins dedup by `Dest`
- **`Forge`** — key-by-key merge via `mergeForgeBlocks`
- **`AllowedRemoteResources`** — NOT merged (security boundary)

Add the new field under the matching rule. If none fits, document the
merge semantics in the function comment.

### 3b. `mergeForgeConfigInto` (compose.go)

Merges per-platform `ForgeConfig` blocks during base composition. If
the new field also lives in `ForgeConfig` (`forge.go`), add it here
with the same merge rule used in 3a.

### 3c. `mergeForgeConfig` (forge.go)

Merges forge-specific overrides into the top-level harness during
`ResolveForge`. Same consideration as 3b — if the field is in
`ForgeConfig`, add it here.

Current `ForgeConfig` fields: `PreScript`, `PostScript`, `Skills`,
`RunnerEnv`, `ValidationLoop`.

## 4. Security pipeline

**File:** `internal/harness/harness.go`

### Resource type validation (`ValidateResourceTypes`)

New fields must be classified as either executable or declarative:

- **Executable fields** (scripts, inputs) — must be local paths; URLs
  are rejected. Add the field to the `execFields` list or an
  equivalent check.
- **Declarative fields** (agent definitions, policies) — URLs are
  allowed but must include `#sha256=...` integrity hashes. Add the
  field to the `declFields` list.

Current executable fields: `PreScript`, `PostScript`, `AgentInput`,
`ValidationLoop.Script`, `HostFiles[].Src`, `APIServers[].Script`.

Current declarative fields: `Agent`, `Policy`, `Base`, `Skills[]`.

### Path traversal protection (`ResolveRelativeTo`)

Relative paths are resolved against the fullsend directory and
rejected if they escape it (e.g. `../../etc/shadow`). If the new
field contains a file path, add it to `ResolveRelativeTo`.

### File existence validation (`ValidateFilesExist`)

After path resolution and URL-to-cache replacement, all file paths
are checked for existence. Add the new field to `ValidateFilesExist`
if it references a local file.

## Quick checklist

Use this when reviewing PRs that add harness struct fields:

- [ ] `${VAR}` references validated (`checkVarRefs`) and expanded
      (`os.Expand`) if the field supports variable interpolation
- [ ] Host-side commands merge `RunnerEnv` via `envToList`
- [ ] Field handled in all three merge functions (or documented as
      intentionally excluded)
- [ ] Field classified in `ValidateResourceTypes` (executable or
      declarative)
- [ ] Relative paths resolved in `ResolveRelativeTo`
- [ ] File existence checked in `ValidateFilesExist`
