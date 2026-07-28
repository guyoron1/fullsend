# Per-repo configuration reference

Per-repo mode stores configuration in `.fullsend/config.yaml` within the
target repository. This document describes every field, its default, and
how it resolves at runtime.

## Config resolution model

Per-repo config uses a two-tier resolution: the **overlay** (your
`.fullsend/config.yaml`) and **code defaults** (compiled into fullsend).

```
.fullsend/config.yaml   (overlay — writable)
        ↓ falls back to
code defaults            (read-only, compiled into fullsend)
```

Each field is a **scalar override**: if the field is present in the
overlay, its value replaces the code default entirely. There is no
deep merge, no partial list append, no field-level inheritance. If
the overlay file is missing, all code defaults apply.

Unlike content layering ([ADR 0035][adr-0035]), which resolves
*files* across three tiers (upstream defaults, org overrides,
per-repo overrides), config field resolution uses only two tiers
and operates on individual YAML fields, not files.

**Only the overlay is writable.** Code defaults cannot be changed
without a fullsend release. Content customization (agents, skills,
harness definitions) uses a separate file-level layering model
documented in [Customizing Agents][customizing-agents].

## Per-field reference

| Field | YAML key | Type | Code default | Merge rule |
|-------|----------|------|--------------|------------|
| Version | `version` | string | `"1"` | Required in overlay |
| Kill switch | `kill_switch` | bool | `false` | Scalar override |
| Roles | `roles` | string[] | *see below* | Scalar override (entire list) |

### `version`

- **Required.** Must be `"1"`.
- Validated at install time by `PerRepoConfig.Validate()`.
- Not read at dispatch time.

### `kill_switch`

- **Optional.** Defaults to `false` when absent.
- Checked in `reusable-dispatch.yml` before stage routing. When
  `true`, all agent dispatch halts with an error.
- When the overlay file itself is missing, defaults to `false`
  (dispatch proceeds normally).

### `roles`

- **Optional.** When absent or empty, all stages are allowed.
- When present, acts as a **whitelist** — only stages whose required
  role appears in the list will dispatch. The entire list is replaced;
  individual roles cannot be added or removed incrementally.
- Stages map to required roles as follows:

  | Stage | Required role |
  |-------|---------------|
  | `triage` | `triage` |
  | `code` | `coder` |
  | `review` | `review` |
  | `fix` | `coder` |
  | `retro` | `fullsend` |
  | `prioritize` | `fullsend` |

- The CLI default (`fullsend admin install` / `fullsend github setup`)
  writes `[triage, coder, review, fix, retro, prioritize]`. This
  excludes the `fullsend` role because per-repo mode does not use a
  separate dispatch app. As a consequence, `retro` and `prioritize`
  stages are **not dispatched** with the CLI default roles — add
  `fullsend` to the list to enable them.
- When the overlay file is missing entirely, the role check is skipped
  and all stages dispatch regardless.

## Example

Minimal overlay enabling triage, code, and review only:

```yaml
version: "1"
roles: [triage, coder, review]
```

Full overlay with all stages enabled (including retro and prioritize):

```yaml
version: "1"
roles: [fullsend, triage, coder, review, fix, retro, prioritize]
```

Emergency stop — halt all dispatch:

```yaml
version: "1"
kill_switch: true
```

## Comparison with per-org config

In per-org mode, `config.yaml` lives in the `.fullsend` config repo
and uses the `OrgConfig` schema. Per-org adds fields that are not
available in per-repo mode:

| Field | Per-repo | Per-org |
|-------|----------|---------|
| `kill_switch` | `.kill_switch` | `.kill_switch` |
| Roles | `.roles` (flat list) | `.defaults.roles` |
| Auto-merge | Not available | `.defaults.auto_merge` |
| Max retries | Not available (set in harness YAML) | `.defaults.max_implementation_retries` |
| Status notifications | Not available | `.defaults.status_notifications` |

Status notifications are intentionally org-level only — notification
style is an org-wide UX decision, not a per-repo operational choice.

## See also

- [ADR 0033: Per-repo installation mode][adr-0033] — architecture and
  credential models
- [Installation guide — Per-repo installation][install-per-repo] —
  install commands and flags
- [Customizing Agents][customizing-agents] — content layering
  (file-level resolution via ADR 0035)
- [Private repositories][private-repos] — role recommendations for
  private repos

[adr-0033]: ../ADRs/0033-per-repo-installation-mode.md
[adr-0035]: ../ADRs/0035-layered-content-resolution.md
[install-per-repo]: installation.md#per-repo-installation
[customizing-agents]: ../guides/user/customizing-agents.md#layered-configuration-resolution
[private-repos]: ../guides/infrastructure/private-repositories.md
