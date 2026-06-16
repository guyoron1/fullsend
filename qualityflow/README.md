# QualityFlow — AI-powered QE test planning and code generation

QualityFlow extends FullSend with a complete QE (Quality Engineering) pipeline that generates test plans, test descriptions, and working test implementations from Jira tickets.

## Where QualityFlow fits in the SDLC

FullSend already covers triage, code generation, and code review. QualityFlow adds the missing **test planning and verification** stage, completing the development lifecycle:

```
┌─────────────────────────────────────────────────────────┐
│                   FullSend SDLC                         │
│                                                         │
│  Triage → Prioritize → Code → Review → Test → Retro    │
│                                         ^^^^            │
│                                     QualityFlow         │
└─────────────────────────────────────────────────────────┘
```

Together, FullSend + QualityFlow provide a closed loop: code changes generate test plans, test plans generate test code, and test results feed back into the next cycle.

## Lifecycle

QualityFlow adds 8 agents:

```
Jira ticket
  │
  ├─ stp-builder          → Software Test Plan (STP) from Jira + PR data
  ├─ stp-reviewer         → Automated QE review of the STP
  ├─ stp-refiner          → Iterative fix loop until STP is APPROVED
  │
  ├─ std-builder          → STD YAML + Go/Python test stubs from STP
  ├─ std-reviewer         → Automated QE review of the STD
  ├─ std-refiner          → Iterative fix loop until STD is APPROVED
  │
  ├─ go-test-generator    → Working Go/Ginkgo tier 1 tests from STD
  └─ python-test-generator → Working Python/pytest tier 2 tests from STD
```

## Architecture

QualityFlow separates **tool** from **project config**:

- **Tool** (this directory) — agents, skills, harness, policies, validation scripts. Ships upstream in the fullsend repo. Generic, project-agnostic.
- **Project config** (`config/`) — routing, components, patterns, templates, reference tests. Team-specific. Lives in your org's `.fullsend` repo or is customized after `fullsend admin install`.

The `config/` directory ships with `config/projects/example/` — a skeleton showing the required YAML structure. Teams copy it to create their own project config.

### How config reaches the sandbox

Each harness YAML declares `agent_input: config`. At runtime, fullsend SCP's the `config/` directory into `/tmp/workspace/agent-input/` inside the sandbox. The `QF_CONFIG_DIR` env var (set in `env/qf-credentials.env`) tells the agents where to find it.

```
Host                              Sandbox
qualityflow/config/  ──SCP──>  /tmp/workspace/agent-input/
                                  ├── routing.yaml
                                  ├── _defaults.yaml
                                  ├── _schema.yaml
                                  └── projects/my-project/
                                       ├── project.yaml
                                       ├── repositories.yaml
                                       └── ...
```

## Usage

```bash
# Set credentials
export JIRA_TICKET=MYPROJ-12345
export JIRA_BASE_URL=https://your-jira.atlassian.net
export JIRA_API_TOKEN=<token>
export JIRA_USER_EMAIL=<email>
export GH_TOKEN=<github-token>

# Run an agent
fullsend run qualityflow/stp-builder \
  --fullsend-dir /path/to/.fullsend \
  --target-repo /path/to/your/project-source
```

`--target-repo` points to the project's source code repository (e.g., the repo under test). QualityFlow config comes from the `config/` directory inside the fullsend-dir, not from the target repo.

## Adding your project

1. Copy the example skeleton:
   ```bash
   cp -r qualityflow/config/projects/example qualityflow/config/projects/my-project
   ```

2. Edit each YAML file with your project's real values (see comments in each file).

3. Add route(s) in `qualityflow/config/routing.yaml`:
   ```yaml
   routes:
     - prefix: "MYPROJ"
       project: "my-project"
   ```

4. Run the pipeline:
   ```bash
   export JIRA_TICKET=MYPROJ-100
   fullsend run qualityflow/stp-builder ...
   ```

## Network policies

| Policy | Used by | Allows |
|--------|---------|--------|
| `qf-full.yaml` | stp-builder, stp-reviewer, stp-refiner | Jira API, GitHub API, Vertex AI |
| `qf-vertex.yaml` | std-builder, std-reviewer, std-refiner | Vertex AI only |
| `qf-codegen.yaml` | go-test-generator, python-test-generator | Vertex AI, GitHub API, Go/Python registries |

## Directory layout

```
qualityflow/
├── agents/      8 agent prompts
├── config/      Project config framework + example skeleton
│   ├── routing.yaml
│   ├── _defaults.yaml
│   ├── _schema.yaml
│   └── projects/
│       └── example/    ← Copy this for your project
├── harness/     8 harness YAMLs (wiring)
├── images/      Container images (Dockerfile for sandbox)
├── plugins/     LSP plugin config (gopls for Go analysis)
├── policies/    3 network policies
├── scripts/     7 validation + setup scripts
├── skills/     24 reusable skills
├── env/         Credential templates (variable expansion, no secrets)
└── README.md
```

## Prerequisites

- Jira API token with read access to your project
- GitHub token with repo read access (for PR diffs and repo file fetch)
- GCP credentials for Vertex AI (Claude via Vertex)
