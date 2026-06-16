---
name: lsp-tracer
description: Trace call graphs using gopls CLI to identify regression impact
model: claude-opus-4-6
---

# LSP Tracer Skill

**Phase:** Pre-Processing
**User-Invocable:** false

## Purpose

Trace call graphs using **gopls CLI commands** to identify regression impact.
This skill uses direct `gopls` CLI invocations via Bash — it does NOT depend
on the Claude Code LSP tool (which is unavailable in FullSend agent mode).

## When to Use

Invoked by the **regression-analyzer** subagent (or directly by stp-builder)
to trace code dependencies from changed files.

## Tools Required

- Bash (for `gopls` CLI commands)
- Grep (for fallback text-based discovery)
- Read (for reading source files)

## Prerequisites

Before tracing, verify gopls is available and the repo has a go.mod.
**CRITICAL:** Always prepend `/usr/local/go/bin` to PATH — gopls needs the
`go` binary but it may not be in PATH inside the FullSend sandbox.

```bash
export PATH="/usr/local/go/bin:$PATH" && which go && which gopls && gopls version && echo "gopls READY" || echo "gopls NOT AVAILABLE"
ls $SOURCE_REPO_DIR/go.mod 2>/dev/null && echo "Go module found" || echo "No go.mod"
```

If gopls is not available, fall back to Grep/Read-based analysis.

**Every gopls command in this skill MUST start with:**
```bash
export PATH="/usr/local/go/bin:$PATH" && cd $SOURCE_REPO_DIR && gopls ...
```

## gopls CLI Operations

### 1. Find Symbols (workspace search)

Search for symbols by name across the repo:

```bash
cd $SOURCE_REPO_DIR && gopls symbols ./pkg/path/to/file.go 2>/dev/null
```

This lists all symbols (functions, types, methods) in a file with their
line numbers and kinds.

To search across multiple files for a symbol name:

```bash
cd $SOURCE_REPO_DIR && grep -rn "func.*SymbolName" --include="*.go" pkg/
```

### 2. Go to Definition

Find where a symbol is defined:

```bash
cd $SOURCE_REPO_DIR && gopls definition ./pkg/path/to/file.go:<line>:<column> 2>/dev/null
```

Output format: `file:line:col-endcol: definition text`

### 3. Find References

Find all places a symbol is referenced:

```bash
cd $SOURCE_REPO_DIR && gopls references ./pkg/path/to/file.go:<line>:<column> 2>/dev/null
```

Output: one `file:line:col` per line for each reference location.

### 4. Find Implementations

Find all implementations of an interface method:

```bash
cd $SOURCE_REPO_DIR && gopls implementations ./pkg/path/to/file.go:<line>:<column> 2>/dev/null
```

### 5. Call Hierarchy (Incoming Calls)

Find who calls a function:

```bash
cd $SOURCE_REPO_DIR && gopls call_hierarchy ./pkg/path/to/file.go:<line>:<column> 2>/dev/null
```

This shows the call hierarchy for the symbol at the given position.

## Input

```yaml
# Repo path: $SOURCE_REPO_DIR or read from repositories.yaml
symbols_to_trace:
  - name: HandleCPUHotplug
    file: pkg/virt-controller/vm/vm.go
    line: 105
    character: 6
  - name: CPUHotplugSpec
    file: api/v1/types.go
    line: 250
    character: 6

# Optional: If symbols_to_trace is empty but feature_candidates is provided
feature_candidates:
  explicit_mentions:
    - VirtualMachine
    - multiarch
  component_hints:
    - component: virt-handler
      package_path: pkg/virt-handler/
    - component: node-labeller
      package_path: pkg/virt-handler/node-labeller/
  acceptance_criteria:
    - VM can run on ARM nodes
```

## Alternative Entry Point Discovery (when no PR data)

If `symbols_to_trace` is empty but `feature_candidates` is provided:

### 1. Discovery from explicit_mentions

For each candidate in `explicit_mentions`:

```bash
cd $SOURCE_REPO_DIR && grep -rn "func.*VirtualMachine" --include="*.go" pkg/ | head -20
```

Then for each discovered function, get its symbols:

```bash
cd $SOURCE_REPO_DIR && gopls symbols ./pkg/path/to/discovered_file.go 2>/dev/null
```

### 2. Discovery from component_hints

For each component_hint:
1. Map to package path using `{project_context.config_dir}/components.yaml`
2. List Go files in the package
3. Run `gopls symbols` on main files to find exported functions

```bash
cd $SOURCE_REPO_DIR && gopls symbols ./pkg/virt-handler/node-labeller/node_labeller.go 2>/dev/null
```

### 3. Discovery from acceptance_criteria

Parse each acceptance criteria item for technical terms, then grep:

```bash
cd $SOURCE_REPO_DIR && grep -rn "func Migrate" --include="*.go" pkg/ | head -10
```

## Tracing Workflow

For each symbol to trace:

### Step 1: Locate the symbol

```bash
cd $SOURCE_REPO_DIR && gopls symbols ./path/to/file.go 2>/dev/null | grep "SymbolName"
```

### Step 2: Find references (who uses this)

```bash
cd $SOURCE_REPO_DIR && gopls references ./path/to/file.go:<line>:<col> 2>/dev/null
```

### Step 3: Go to definitions of callers

For each reference found, get its definition:

```bash
cd $SOURCE_REPO_DIR && gopls definition ./caller/file.go:<line>:<col> 2>/dev/null
```

### Step 4: Build the call chain

Repeat Steps 2-3 up the call chain until you reach:
- Test files (note them as test impact)
- Standard library calls
- External dependencies
- Maximum depth (3 levels for performance)

## Output Format

```yaml
call_graph:
  - symbol: HandleCPUHotplug
    file: pkg/virt-controller/vm/vm.go
    line: 105

    incoming_calls:  # Who calls this function
      - caller: ReconcileVM
        file: pkg/virt-controller/vm/vm.go
        line: 45
        relationship: direct
      - caller: ProcessVMUpdate
        file: pkg/virt-controller/vm/update.go
        line: 89
        relationship: direct

    outgoing_calls:  # What this function calls
      - callee: ValidateCPUChange
        file: pkg/virt-controller/vm/validation.go
        line: 120
        relationship: direct

    references:  # All code that references this symbol
      - file: pkg/virt-controller/vm/vm_test.go
        line: 456
        context: test
      - file: tests/hotplug_test.go
        line: 78
        context: test

dependency_chains:
  - chain_name: CPU Hotplug > Migration
    path:
      - symbol: HandleCPUHotplug
        file: pkg/virt-controller/vm/vm.go
      - symbol: UpdateVMISpec
        file: pkg/virt-handler/vmi/vmi.go
      - symbol: PrepareMigration
        file: pkg/virt-handler/migration/migration.go
    impact: Migration may be affected by CPU hotplug changes

summary:
  symbols_traced: 5
  total_callers: 12
  total_callees: 8
  total_references: 45
  max_chain_depth: 3
  tool_used: gopls-cli
```

## Depth Limits

- **Maximum Call Chain Depth:** 3 levels (to stay within context limits)
- **Maximum References:** 30 per symbol
- **Stop Conditions:**
  - Reaching test files (note as test impact)
  - Reaching standard library
  - Reaching external dependencies

## Path Normalization

**Repository Base:** `$SOURCE_REPO_DIR`

When reporting paths, use relative paths from repo root:
- Absolute: `$SOURCE_REPO_DIR/pkg/virt-controller/vm/vm.go`
- Relative: `pkg/virt-controller/vm/vm.go`

## Feature Mapping

Map code locations to features by reading `{project_context.config_dir}/components.yaml` `path_to_feature` mapping.

| Path Pattern | Feature |
|:-------------|:--------|
| `pkg/virt-controller/vm/` | VM Lifecycle |
| `pkg/virt-handler/migration/` | Live Migration |
| `pkg/virt-controller/migration/` | Live Migration |
| `pkg/*/hotplug/` | Hot-plug |
| `pkg/virt-controller/snapshot/` | Snapshots |
| `pkg/virt-controller/clone/` | Clone |
| `pkg/network/` | Networking |
| `pkg/storage/` | Storage |
| `pkg/instancetype/` | Instance Types |

## Example Trace

Input:
```yaml
symbols_to_trace:
  - name: HandleCPUHotplug
    file: pkg/virt-controller/vm/vm.go
    line: 105
    character: 6
```

gopls CLI Calls:
```bash
# 1. Get symbols from the file
cd $SOURCE_REPO_DIR && gopls symbols ./pkg/virt-controller/vm/vm.go 2>/dev/null | grep "HandleCPUHotplug"

# 2. Find all references
cd $SOURCE_REPO_DIR && gopls references ./pkg/virt-controller/vm/vm.go:105:6 2>/dev/null

# 3. For each caller, go to its definition
cd $SOURCE_REPO_DIR && gopls definition ./pkg/virt-handler/vmi/vmi.go:230:6 2>/dev/null
```

Output:
```yaml
dependency_chains:
  - chain_name: CPU Hotplug > VMI Update > Migration
    path:
      - HandleCPUHotplug (vm.go:105)
      - UpdateVMISpec (vmi.go:230)
      - PrepareMigration (migration.go:45)
    impact: Live migration depends on VMI spec updates
    recommended_test: Verify migration works after CPU hotplug
```
