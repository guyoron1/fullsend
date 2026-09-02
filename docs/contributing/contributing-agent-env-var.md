# Contributing a Configurable Env Var to a Default Agent

This tutorial walks through a concrete end-to-end example: taking a
hardcoded behavior in a default agent, extracting it into an environment
variable, wiring it through the harness, documenting the new knob, and
submitting the change upstream.

The example uses a fictional scenario — making the triage agent's
duplicate-check behavior configurable — but the process applies to any
default agent in the
[`fullsend-ai/agents`](https://github.com/fullsend-ai/agents) repository.

> **Scope.** This tutorial covers the simpler env-var approach for agent
> parameterization. A formal parameterization interface is tracked in
> [#579](https://github.com/fullsend-ai/fullsend/issues/579); when it
> lands, this tutorial will be updated to cover both approaches.

## Prerequisites

- A local clone of
  [`fullsend-ai/agents`](https://github.com/fullsend-ai/agents) (where
  default agent definitions, harnesses, and skills live).
- A local clone of
  [`fullsend-ai/fullsend`](https://github.com/fullsend-ai/fullsend)
  (where agent user docs live under `docs/agents/`).
- The fullsend CLI installed — see
  [Running agents locally](../guides/user/running-agents-locally.md).
- Familiarity with harness YAML structure — see
  [Configuring agent behavior](../guides/user/customizing-agents.md#harness-yaml-structure).

## Step 1: Identify where the behavior lives

Agent behavior is split across three layers. Before changing anything,
determine which layer owns the behavior you want to make configurable:

| Layer | Repository | What it controls |
|-------|-----------|------------------|
| **Agent definition** (`agents/<name>.md`) | `fullsend-ai/agents` | The prompt — what the agent does, how it reasons, what tools it uses |
| **Harness** (`harness/<name>.yaml`) | `fullsend-ai/agents` | Execution config — sandbox image, env vars, timeouts, skills, scripts |
| **Pre/post scripts** (`scripts/pre-<name>.sh`, `scripts/post-<name>.sh`) | `fullsend-ai/agents` | Host-side logic that runs before and after the sandbox |

In our example, the triage agent has a hardcoded instruction in
`agents/triage.md` that says:

```markdown
### Phase 2: Check for duplicates

Search open issues for duplicates. If you find a likely duplicate,
note it in the triage output.
```

We want to let users skip the duplicate check entirely (some repos
have their own dedup tooling). The behavior lives in the **agent
definition**, but the knob will be wired through the **harness** as an
environment variable.

## Step 2: Choose a variable name

Follow the naming convention from
[ADR 0049](../ADRs/0049-agent-configuration-env-var-convention.md):

```
{AGENT}_{SETTING_NAME}
```

- `{AGENT}` is the agent name in uppercase, derived from the harness
  filename (e.g., `TRIAGE`, `REVIEW`, `CODE`).
- `{SETTING_NAME}` is `SCREAMING_SNAKE_CASE` describing the setting.

For our example: **`TRIAGE_SKIP_DUPLICATE_CHECK`**.

Pick a sensible default so existing users are unaffected. Env vars
delivered via `.env` files with `expand: true` resolve unset host vars
to an empty string — so the agent must treat both unset and empty as
"use the default."

| Variable | Default | Valid values |
|----------|---------|-------------|
| `TRIAGE_SKIP_DUPLICATE_CHECK` | `false` | `true`, `false` |

## Step 3: Update the agent definition

Edit the agent prompt in `agents/triage.md` to read the variable and
branch on it. The agent already has access to all env vars delivered
via the harness `.env` file.

**Before:**

```markdown
### Phase 2: Check for duplicates

Search open issues for duplicates. If you find a likely duplicate,
note it in the triage output.
```

**After:**

```markdown
### Phase 2: Check for duplicates

If `$TRIAGE_SKIP_DUPLICATE_CHECK` is set to `true`, skip this phase
entirely and proceed to Phase 3.

Otherwise, search open issues for duplicates. If you find a likely
duplicate, note it in the triage output.
```

**Key principles:**

- The agent treats an unset or empty variable the same as `false` — the
  default behavior. This ensures backward compatibility.
- Stick to a canonical boolean value (`true` / `false`) and document it
  in the variable table. Since agents interpret the value via prompt
  logic rather than a strict parser, explicitly state the expected value
  rather than relying on the agent to infer equivalents like `1`, `yes`,
  or `TRUE`.

## Step 4: Wire the variable through the harness

The variable needs to reach two places:

1. **Inside the sandbox** (at inference time) — so the agent can read it.
2. **On the runner** (for pre/post scripts) — only if the scripts also
   need it.

### 4a. Add to the env file

The agent's `.env` file (e.g., `env/triage.env`) is copied into the
sandbox with `expand: true`, which resolves `${VAR}` references from
the host environment before the file enters the sandbox.

Add the variable:

```bash
# env/triage.env
TRIAGE_SKIP_DUPLICATE_CHECK=${TRIAGE_SKIP_DUPLICATE_CHECK}
```

> **Important:** Expansion uses Go's `os.Expand`, which supports `$VAR`
> and `${VAR}` only — no default values (`:-default`), substring
> operations, or other shell parameter expansion features
> ([ADR 0055](../ADRs/0055-unified-env-var-delivery.md)). If the host
> variable is unset, it resolves to an empty string. The agent must
> treat both unset and empty as "use the default" (see Step 3).

### 4b. Add to the harness `env:` block (if scripts need it)

If pre/post scripts also need the variable, add it to the harness
YAML's `env.runner` block. For our example, only the agent reads it,
so this step is optional:

```yaml
# harness/triage.yaml (excerpt)
env:
  runner:
    # ... existing vars ...
  sandbox:
    TRIAGE_SKIP_DUPLICATE_CHECK: "${TRIAGE_SKIP_DUPLICATE_CHECK}"
```

> **Note:** `env.sandbox` delivers variables directly into the sandbox
> environment. If the agent's `.env` file already carries the variable
> (step 4a), you do not need `env.sandbox` as well — either mechanism
> works. The `.env` file approach is more common in the existing agents.

### Where does the value come from at runtime?

In CI, the value flows through the GitHub Actions workflow:

```
GitHub repo variable / org secret
        │
        ▼
Workflow env: block  →  TRIAGE_SKIP_DUPLICATE_CHECK
        │
        ▼
Harness host_files   →  env/triage.env (expand: true)
        │
        ▼
Sandbox .env file    →  agent reads $TRIAGE_SKIP_DUPLICATE_CHECK
```

Users set the value in their CI workflow or as a GitHub repo variable.
They do **not** need to fork the harness or agent definition.

## Step 5: Test the change locally

Use `fullsend run` to verify the behavior with and without the new
variable. See [Running agents
locally](../guides/user/running-agents-locally.md) for full setup.

### Test 1: Default behavior (variable unset)

```bash
fullsend run triage \
  --fullsend-dir /tmp/fullsend-agents/ \
  --target-repo /tmp/target-repo/ \
  --env-file fullsend-gcp.env \
  --env-file fullsend-triage.env \
  --no-post-script
```

Verify the agent still performs the duplicate check (the default).

### Test 2: Variable set to `true`

Create an override env file:

```bash
# fullsend-triage-override.env
TRIAGE_SKIP_DUPLICATE_CHECK=true
```

```bash
fullsend run triage \
  --fullsend-dir /tmp/fullsend-agents/ \
  --target-repo /tmp/target-repo/ \
  --env-file fullsend-gcp.env \
  --env-file fullsend-triage.env \
  --env-file fullsend-triage-override.env \
  --no-post-script
```

Verify the agent skips the duplicate check.

> **Tip:** Use `--no-post-script` during testing to avoid side effects
> (posting comments, applying labels). Use `--keep-sandbox` to inspect
> the sandbox state after the run.

## Step 6: Document the variable

Update the agent's user-facing documentation in the `fullsend` repo at
`docs/agents/<agent>.md`. Every agent has a **Variables** subsection
under "Configuration and extension"
([ADR 0049](../ADRs/0049-agent-configuration-env-var-convention.md)).

**Before** (in `docs/agents/triage.md`):

```markdown
### Variables

None.
```

**After:**

```markdown
### Variables

| Variable | Description | Default | Valid values |
|----------|-------------|---------|--------------|
| `TRIAGE_SKIP_DUPLICATE_CHECK` | Skip the duplicate-issue check phase | `false` | `true`, `false` |
```

If the agent previously had no variables, replace `None.` with the
table. If it already has variables, add a new row.

## Step 7: Write or update tests

Agent behavior changes should have corresponding test coverage.
Depending on what you changed:

- **Agent prompt changes** — verify the behavior difference in a local
  `fullsend run` (step 5). If the agents repo has behaviour tests
  (`.feature` files), add a scenario that exercises the new variable.
- **Script changes** — if you modified a pre/post script to read the
  new variable, add a unit test or extend an existing one.
- **Harness changes** — if you added `env.sandbox` or `env.runner`
  entries, verify the harness passes validation:

  ```bash
  fullsend validate harness/triage.yaml --fullsend-dir /tmp/fullsend-agents/
  ```

## Step 8: Submit the PR

Your change likely spans two repositories:

| Repository | Files changed |
|-----------|---------------|
| `fullsend-ai/agents` | `agents/triage.md`, `env/triage.env`, optionally `harness/triage.yaml` |
| `fullsend-ai/fullsend` | `docs/agents/triage.md` (Variables table) |

Submit a PR to each repository. The agents-repo PR carries the
behavioral change; the fullsend-repo PR carries the documentation.
Reference the agents-repo PR from the fullsend-repo PR so reviewers
see the full picture.

### PR conventions

Follow the repo's commit and PR conventions:

- **Commit format:** Conventional Commits — see [COMMITS.md](../../COMMITS.md).
  For this type of change, use `refactor` (extracting a hardcoded behavior
  into a configurable knob) or `feat` (if the knob enables genuinely new
  user-facing functionality). When in doubt, prefer `refactor`.
- **PR title:** Matches the commit subject. Include the scope:
  `refactor(triage): make duplicate check configurable via env var`.
- **DCO:** If you are a human contributor, sign off your commits with
  `git commit -s`. Autonomous agent commits are exempt — see
  [AGENTS.md](../../AGENTS.md).
- **Linting:** Stage your changes before running `make lint`. Pre-commit
  only checks staged files.

## Summary of files touched

For a single env-var contribution to a default agent, you will typically
touch:

```
fullsend-ai/agents/
├── agents/<name>.md           # Agent prompt: add conditional on the var
├── env/<name>.env             # Env file: add the var with a default
├── harness/<name>.yaml        # Harness: add env.sandbox or env.runner
│                              #   (only if scripts need the var)
└── scripts/                   # Pre/post scripts (only if they branch
                               #   on the var)

fullsend-ai/fullsend/
└── docs/agents/<name>.md      # User docs: add row to Variables table
```

## Reference

- [ADR 0049 — Agent configuration env var convention](../ADRs/0049-agent-configuration-env-var-convention.md)
  — naming, defaults, and documentation requirements
- [ADR 0055 — Unified env var delivery](../ADRs/0055-unified-env-var-delivery.md)
  — `env.runner` and `env.sandbox` harness fields
- [Configuring agent behavior](../guides/user/customizing-agents.md)
  — harness YAML structure and layered resolution
- [Bring Your Own Agent](../guides/user/bring-your-own-agent.md)
  — config-driven agent registration and `base` composition
- [Building custom agents](../guides/user/building-custom-agents.md)
  — creating agents from scratch (when configuration is not enough)
- [Running agents locally](../guides/user/running-agents-locally.md)
  — local testing with `fullsend run`
