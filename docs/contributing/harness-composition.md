# Harness Composition

When changing merge or compose functions in `internal/harness/`, be aware that
these functions have an architectural counterpart: diff functions that extract
the delta between a composed result and its base. The diff functions were
removed with the scaffold agent extraction (see
[ADR 0045](../ADRs/0045-forge-portable-harness-schema.md), Consequences →
"Bidirectional composition"), but the bidirectional invariant remains
architecturally important — any future re-addition of diff-based operations
must mirror the current merge semantics.

## Why this matters

The `fullsend agent migrate-customizations` command
([cli-internals](../guides/dev/cli-internals.md)) converts deprecated
`customized/` directory overlays into config-driven agents with `base:`
composition. The current implementation moves files from `customized/` to
regular directories via `rewriteCustomizedPaths` and registers the agent in
config — it does not use diff functions today. However,
[ADR 0064](../ADRs/0064-deprecate-customized-directory-overlay.md) envisions a
diff-based migration flow where overrides are extracted by diffing a customized
harness against its upstream base. If diff functions are re-added and do not
mirror the merge logic, overrides would be lost or duplicated — silently
corrupting harness customizations.

This constraint is documented in
[ADR 0045](../ADRs/0045-forge-portable-harness-schema.md) (Consequences →
"Bidirectional composition") but is not visible during normal documentation
reads because ADRs are subject to immutability policy.

## Compose/merge functions and their expected diff counterparts

The merge functions below live in `internal/harness/compose.go` (for `base:`
composition) and `internal/harness/forge.go` (for runtime forge-config
application). The diff counterparts listed in the right column **do not
currently exist** — they were removed with the scaffold agent extraction. The
names shown are the expected counterparts based on ADR 0045; if diff functions
are re-added, they should follow this naming and scope.

| Compose / Merge (compose.go, forge.go) | Expected diff counterpart (not currently present) | Scope |
|---|---|---|
| `mergeBaseIntoChild` | `DiffHarness` | Top-level harness fields |
| `mergeForgeConfigInto` | `diffForgeConfig` | Per-platform forge config in `base:` composition |
| `mergeForgeConfig` | `diffForgeConfig` | Forge config applied at runtime |
| `mergeForgeBlocks` | (per-key iteration over `diffForgeConfig`) | Map of platform → ForgeConfig |
| Field-level helpers (e.g. `mergeSkills`, `mergeHostFiles`) | Corresponding diff helpers | Individual collection fields |

## Checklist for changes

When modifying merge or compose logic:

1. **Check whether diff functions exist.** Look for `internal/harness/diff.go`
   or functions matching `Diff*`/`diff*` in the harness package. If they do not
   exist, note in your commit message that the diff side is absent and skip
   steps 2–4.
2. **Identify the paired function.** Use the table above or search for the
   counterpart by name pattern (`merge*` ↔ `diff*`, `Diff*`).
3. **Mirror the change.** If you add a field to a merge function, add the
   corresponding extraction to the diff function. If you change merge
   semantics (e.g. from whole-replace to field-level merge), update the
   diff to produce the correct delta under the new semantics.
4. **Test the round-trip.** Verify that `compose(base, diff(composed, base))`
   produces the original `composed` result. Existing tests in
   `compose_test.go` cover the merge side; diff tests should follow the
   same patterns.
5. **Check field-level helpers.** Changes to helpers like `mergeSkills` or
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
`DiffHarness` was "removed with the scaffold agent extraction." The diff
functions do not currently exist in this repository. The bidirectional
constraint remains architecturally important — any future re-addition of diff
functions must mirror the current merge semantics.

See also: [Issue #662](https://github.com/guyoron1/fullsend/issues/662)
tracks a broader harness field integration checklist covering expansion,
environment construction, composition carry-forward, and security pipelines.
