# Harness Composition — Compose/Diff Bidirectional Invariant

When changing merge or compose functions in `internal/harness/`, you must also
update the corresponding diff functions. These two sides form a bidirectional
pair: compose merges a base harness into a child, and diff extracts the delta
between a composed result and its base. Breaking one side without updating the
other silently corrupts the round-trip property that the
`migrate-customizations` command depends on
([ADR 0064](../ADRs/0064-deprecate-customized-directory-overlay.md)).

## Why this matters

The `fullsend agent migrate-customizations` command
([cli-internals](../guides/dev/cli-internals.md)) converts deprecated
`customized/` directory overlays into config-driven agents with `base:`
composition. To do this, it must diff a customized harness against its upstream
base and express only the overrides. If the diff logic does not mirror the merge
logic, overrides are lost or duplicated during migration — silently corrupting
harness customizations.

This constraint is documented in
[ADR 0045](../ADRs/0045-forge-portable-harness-schema.md) (Consequences →
"Bidirectional composition") but is not visible during normal documentation
reads because ADRs are subject to immutability policy.

## Current paired functions

Changes to any function in the left column must be mirrored in the right column,
and vice versa.

| Compose / Merge (compose.go, forge.go) | Diff (diff.go when present) | Scope |
|---|---|---|
| `mergeBaseIntoChild` | `DiffHarness` | Top-level harness fields |
| `mergeForgeConfigInto` | `diffForgeConfig` | Per-platform forge config in `base:` composition |
| `mergeForgeConfig` | `diffForgeConfig` | Forge config applied at runtime |
| `mergeForgeBlocks` | (per-key iteration over `diffForgeConfig`) | Map of platform → ForgeConfig |
| Field-level helpers (e.g. `mergeSkills`, `mergeHostFiles`) | Corresponding diff helpers | Individual collection fields |

The merge functions live in `internal/harness/compose.go` (for `base:`
composition) and `internal/harness/forge.go` (for runtime forge-config
application). Diff counterparts live in `internal/harness/diff.go` when present.

## Checklist for changes

When modifying merge or compose logic:

1. **Identify the paired function.** Use the table above or search for the
   counterpart by name pattern (`merge*` ↔ `diff*`, `Diff*`).
2. **Mirror the change.** If you add a field to a merge function, add the
   corresponding extraction to the diff function. If you change merge
   semantics (e.g. from whole-replace to field-level merge), update the
   diff to produce the correct delta under the new semantics.
3. **Test the round-trip.** Verify that `compose(base, diff(composed, base))`
   produces the original `composed` result. Existing tests in
   `compose_test.go` cover the merge side; diff tests should follow the
   same patterns.
4. **Check field-level helpers.** Changes to helpers like `mergeSkills` or
   `mergeHostFiles` may also need corresponding diff helpers.

## When the diff side is unaffected

Not every compose change requires a diff change. If the modification is purely
internal (e.g. performance optimization that preserves semantics) or affects
only fields that use whole-replace semantics in both directions, the diff side
may already be correct. In that case, document in your commit message why the
diff side is unaffected.

## Historical context

[ADR 0045](../ADRs/0045-forge-portable-harness-schema.md) introduced `base:`
composition and its inverse. The ADR's Consequences section states that
`DiffHarness` was "removed with the scaffold agent extraction," but the diff
functions may be re-added or may exist in the upstream repository. The
bidirectional constraint remains architecturally important regardless of
whether the diff functions are currently present — any future re-addition must
mirror the current merge semantics.

See also: [Issue #662](https://github.com/guyoron1/fullsend/issues/662)
tracks a broader harness field integration checklist covering expansion,
environment construction, composition carry-forward, and security pipelines.
