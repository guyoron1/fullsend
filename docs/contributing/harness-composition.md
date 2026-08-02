# Harness Composition

When changing merge or compose functions in `internal/harness/`, you must check
whether corresponding diff functions exist and update them in lockstep.

## The bidirectional invariant

[ADR 0045](../ADRs/0045-forge-portable-harness-schema.md) introduced `base:`
composition with field-level merge semantics. The merge functions have an
architectural inverse: diff functions that extract the delta between a composed
result and its base. ADR 0045 (Consequences, "Bidirectional composition") notes
that `DiffHarness` was removed with the scaffold agent extraction, but the
constraint is that any re-introduced diff function must mirror the current
merge semantics.

**Why this matters:** If diff functions are re-added (for example to support
extracting overrides from a composed harness), they must produce deltas that
round-trip correctly: `compose(base, diff(composed, base))` must equal the
original `composed` result. A mismatch silently corrupts harness
customizations. PR #5450 demonstrated this failure mode: merge logic was
updated without touching the corresponding diff functions, causing a
round-trip regression that took 6 iterations to diagnose.

## Current merge functions

These are the merge functions that define the forward direction of
composition. Any future diff function must be the inverse of the
corresponding merge function listed here.

### `base:` composition (`compose.go`)

| Function | Purpose | Merge semantics |
|---|---|---|
| `mergeBaseIntoChild` | Top-level harness merge | Scalars: child overrides; slices: concatenated; maps: merged (child wins); pointer structs: child replaces if non-nil |
| `mergeSkills` | Skill path deduplication | Base + child, child overrides base by basename |
| `mergeHostFiles` | Host file deduplication | Base + child, child overrides base by dest path |
| `mergeForgeBlocks` | Per-platform forge merge | Key-by-key merge; each platform uses `mergeForgeConfigInto` |
| `mergeForgeConfigInto` | Per-platform forge config merge | Scalars: child overrides; skills: base + child; runner_env: merged (child wins); validation_loop: child replaces (with preflight_check carry-forward) |

### Runtime forge resolution (`forge.go`)

| Function | Purpose | Merge semantics |
|---|---|---|
| `mergeForgeConfig` | Applies platform-specific forge config to harness at runtime | Skills: appended; runner_env: merged (forge wins); validation_loop: forge replaces entirely (no preflight_check carry-forward) |

Note: `mergeForgeConfig` (runtime) and `mergeForgeConfigInto` (composition)
have different semantics. `mergeForgeConfigInto` prepends base skills and
carries forward `PreflightCheck` from base when the child omits it (see
[#5074](https://github.com/fullsend-ai/fullsend/pull/5074)).
`mergeForgeConfig` appends forge skills and replaces `ValidationLoop` entirely.
A diff counterpart would need to account for which merge function produced the
composed result.

### Shared helpers

| Function | Location | Purpose |
|---|---|---|
| `mergeEnvFrom` | `harness.go` | Merges `EnvConfig` sub-maps independently; caller controls precedence |

## Checklist for changes

When modifying merge or compose logic in `internal/harness/`:

1. **Check whether diff functions exist.** Search for
   `internal/harness/diff.go` or functions matching `Diff*`/`diff*` in the
   harness package. As of this writing they do not exist (removed with the
   scaffold agent extraction per ADR 0045).
2. **If diff functions exist:** identify the paired function using the tables
   above or by name pattern (`merge*` / `Merge*` maps to `diff*` / `Diff*`).
   Mirror your change in the diff counterpart and verify the round-trip
   property.
3. **If diff functions do not exist:** note their absence in your commit
   message and skip the remaining steps. No diff-side update is needed.
4. **Test the round-trip.** Verify that
   `compose(base, diff(composed, base))` produces the original `composed`
   result. Existing tests in `compose_test.go` cover the merge side; diff
   tests should follow the same patterns.
5. **Check field-level helpers.** Changes to helpers like `mergeSkills`,
   `mergeHostFiles`, or `mergeForgeConfigInto` may also need corresponding
   diff helpers.

## When the diff side is unaffected

Not every compose change requires a diff change. If the modification is purely
internal (e.g. performance optimization that preserves semantics) or affects
only fields that use whole-replace semantics in both directions, the diff side
may already be correct. Document in your commit message why the diff side is
unaffected.

## Historical context

[ADR 0045](../ADRs/0045-forge-portable-harness-schema.md) introduced `base:`
composition and its inverse. The diff functions (`DiffHarness` and
counterparts) were removed with the scaffold agent extraction. The
bidirectional constraint remains documented in ADR 0045 but is not visible
during normal documentation reads because ADRs are subject to immutability
policy.

See also: [Issue #662](https://github.com/guyoron1/fullsend/issues/662)
tracks a broader harness field integration checklist covering expansion,
environment construction, composition carry-forward, and security pipelines.
