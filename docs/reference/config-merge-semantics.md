# Config Merge Semantics

How each configuration field is resolved across layers, so contributors
can predict the effective value without reading implementation code.

For content-file layering (agents, skills, harness definitions, etc.),
see [Customizing Agents — Layered Configuration Resolution](../guides/user/customizing-agents.md#layered-configuration-resolution).
This document covers `config.yaml` field resolution only.

## Layering overview

Fullsend has three configuration tiers. Each tier is independent — there
is no field-level YAML merging between tiers.

```
fullsend code defaults  →  config.yaml (org or per-repo)  →  content overrides
     (read-only)               (writable layer)               (customized/)
```

- **Code defaults** are compiled into the CLI (`NewOrgConfig`,
  `NewPerRepoConfig`) and into workflow expressions (`// false`,
  fallback values). They are read-through — never edited directly.
- **`config.yaml`** is the only writable layer. In per-org mode it lives
  in the `.fullsend` repo root. In per-repo mode it lives at
  `.fullsend/config.yaml` in the target repository.
- **Content overrides** (`customized/` directories) use file-level
  replacement, not field-level merging — documented in the
  [customizing agents guide](../guides/user/customizing-agents.md#how-override-resolution-works).

## Per-repo config (`.fullsend/config.yaml`)

Used in per-repo installation mode. The config is self-contained — there
is no parent org config to fall back to.

| Field | YAML key | Type | Merge rule | Default | Runtime consumer |
|---|---|---|---|---|---|
| Version | `version` | string | **Required.** Must be `"1"`. | None (must be set) | Validated at parse time |
| Kill switch | `kill_switch` | bool | **Scalar override.** `true` halts all dispatch. | `false` | `reusable-dispatch.yml` reads via `yq '.kill_switch // false'` |
| Roles | `roles` | string list | **Scalar override.** Replaces the code default entirely — not merged or appended. | `["triage", "coder", "review", "fix", "retro", "prioritize"]` | `reusable-dispatch.yml` reads via `yq '.roles[]'`; stages whose role is absent are skipped |

### Notes

- **No parent fallback.** Per-repo config does not inherit from or merge
  with any org-level config. Each field is either set in the file or
  takes its code default.
- **`roles` is a full replacement.** Setting `roles: [triage, coder]`
  disables review, fix, retro, and prioritize. There is no way to
  "append to defaults" — list the complete set you want.
- **Only `config.yaml` is writable.** The code defaults (`PerRepoDefaultRoles()`,
  `Version: "1"`) are read-through; changing them requires a fullsend
  release.

## Org config (`config.yaml` in `.fullsend` repo)

Used in per-org installation mode. Fields are grouped by resolution
behavior.

### Top-level fields

| Field | YAML key | Type | Merge rule | Default | Runtime consumer |
|---|---|---|---|---|---|
| Version | `version` | string | **Required.** Must be `"1"`. | None (must be set) | Validated at parse time |
| Kill switch | `kill_switch` | bool | **Scalar override.** | `false` | `dispatch.yml` reads via `yq '.kill_switch // false'` |
| Allowed remote resources | `allowed_remote_resources` | string list | **Scalar override.** URL prefix allowlist for remote content fetching. | `["https://raw.githubusercontent.com/fullsend-ai/fullsend/"]` | Harness runner |

### Dispatch block

| Field | YAML path | Type | Merge rule | Default | Runtime consumer |
|---|---|---|---|---|---|
| Platform | `dispatch.platform` | string | **Required.** Must be `"github-actions"`. | `"github-actions"` | Validated at parse time |
| Mode | `dispatch.mode` | string | **Scalar override.** `"oidc-mint"` or empty. | `""` (empty) | Dispatch workflow |
| Mint URL | `dispatch.mint_url` | string | **Scalar override.** Informational when mode is `oidc-mint`. | `""` (empty) | Informational |

### Inference block

| Field | YAML path | Type | Merge rule | Default | Runtime consumer |
|---|---|---|---|---|---|
| Provider | `inference.provider` | string | **Scalar override.** Must be `"vertex"` if set. | `""` (omitted when empty) | GCP setup |

### Defaults block

These fields set org-wide defaults. They apply uniformly to all enrolled
repos — there is no per-repo override mechanism for these fields in the
org config (see [Per-repo entries](#per-repo-entries) below).

| Field | YAML path | Type | Merge rule | Default | Runtime consumer |
|---|---|---|---|---|---|
| Roles | `defaults.roles` | string list | **Scalar override.** Full replacement of code default. | `["fullsend", "triage", "coder", "review", "retro", "prioritize"]` | `dispatch.yml` reads via `yq '.defaults.roles[]'`; stages whose role is absent are skipped |
| Max retries | `defaults.max_implementation_retries` | int | **Scalar override.** Must be >= 0. | `2` | Set at install time; not read by dispatch workflows |
| Auto merge | `defaults.auto_merge` | bool | **Scalar override.** | `false` | Set at install time; not read by dispatch workflows |
| Status notifications | `defaults.status_notifications` | object | **Scalar override.** When present, the entire object replaces the code default (no field-level merge within the object). | `null` (omitted) | `fullsend run` reads at runtime for status comments |
| — Comment start | `defaults.status_notifications.comment.start` | string | **Scalar override.** `"enabled"` or `"disabled"`. | `"enabled"` (when parent object is set) | Status comment posting |
| — Comment completion | `defaults.status_notifications.comment.completion` | string | **Scalar override.** `"enabled"` or `"disabled"`. | `"enabled"` (when parent object is set) | Status comment posting |

### Agents block

| Field | YAML path | Type | Merge rule | Default | Runtime consumer |
|---|---|---|---|---|---|
| Agents list | `agents` | list of objects | **Scalar override.** The full list is set at install time. Each entry has `role`, `name`, `slug`. | Generated from installed GitHub Apps | Agent identity resolution |

### Per-repo entries

The `repos` map holds per-repo enrollment status. It does **not**
override `defaults` fields — each entry controls only whether the repo is
enrolled.

| Field | YAML path | Type | Merge rule | Default | Runtime consumer |
|---|---|---|---|---|---|
| Enabled | `repos.<name>.enabled` | bool | **Scalar override.** Controls enrollment. | `false` | Enrollment validation, repo maintenance |
| Roles | `repos.<name>.roles` | string list | **Present in schema but not consumed by dispatch.** Dispatch workflows read `defaults.roles`, not per-repo roles. | `[]` (empty, omitempty) | Not consumed at runtime |

> **Design note:** `StatusNotifications` is intentionally absent from
> per-repo entries. Notification style is an org-wide UX decision
> (consistent appearance across all repos), unlike roles and auto_merge
> which are operationally per-repo. See the `RepoConfig` comment in
> `internal/config/config.go`.

## Resolution summary

Every field in both config types uses **scalar override** — the value in
`config.yaml` replaces the code default entirely. There is no deep merge,
no list append, and no field-level inheritance between tiers.

| Merge rule | Meaning | Used by |
|---|---|---|
| **Scalar override** | Value in config replaces the code default wholesale. For lists, the entire list is replaced — no append/prepend. For objects (e.g., `status_notifications`), the entire object replaces the default. | All fields |
| **Required** | Must be present and set to a specific value. No default to fall back to. | `version` |
| **Not consumed** | Field exists in the schema but is not read at runtime. | `repos.<name>.roles` |

## See also

- [Customizing Agents](../guides/user/customizing-agents.md) — content-file
  layering and override resolution
- [ADR 0003](../ADRs/0003-org-config-repo-convention.md) — org config repo
  convention and inheritance model
- [ADR 0033](../ADRs/0033-per-repo-installation-mode.md) — per-repo
  installation mode and config workspace
- [ADR 0035](../ADRs/0035-layered-content-resolution.md) — layered content
  resolution via `customized/` directories
