# Telemetry

Fullsend produces structured telemetry for every agent run. This guide
covers how to enable telemetry, choose a backend, and read traces.

For span structure, environment variable reference, and attribute details,
see the [Distributed Tracing Reference](../infrastructure/distributed-tracing.md).
For local development and adding new spans, see
[Telemetry Internals](../dev/telemetry-internals.md).

Decided in [ADR 0050](../../ADRs/0050-distributed-tracing-instrumentation.md).

## Zero-configuration baseline (Level 1)

Every `fullsend run` produces one file in the run output directory with no
configuration required:

- **`run-telemetry.jsonl`** — OTLP JSON spans covering the run lifecycle
  (sandbox creation, agent iterations, validation) with timestamps, durations,
  trace IDs, and token/cost attributes.

This file is written on every run unless `OTEL_SDK_DISABLED=true`, which
suppresses all telemetry output including the local file. It contains
metadata only — no prompts, completions, or source code content.

## Prerequisites

Level 1 requires nothing. To enable OTLP export (Level 2 and Level 3) you need:

- An **OTLP/HTTP-capable backend** and its endpoint URL — e.g. Jaeger, Tempo,
  Grafana, MLflow ≥ 3.6, or any OpenTelemetry Collector.
- Any **backend authentication** (bearer token or basic auth) for the
  `OTEL_EXPORTER_OTLP_TRACES_HEADERS` variable.
- **Network reachability** from where runs execute (your machine or CI runners)
  to the backend endpoint.
- For a backend behind a **private CA** (e.g. an internal MLflow): the CA
  certificate bundle, pointed to by `OTEL_EXPORTER_OTLP_CERTIFICATE`. Local
  and bring-your-own-workflow runs only — the managed workflows do not yet
  pass a CA bundle through.

## Enabling OTLP export (Level 2)

To send metadata spans to an OpenTelemetry-compatible backend, set one of the
standard OTEL environment variables:

```bash
# Signal-specific (takes precedence, used as-is — no /v1/traces appended)
export OTEL_EXPORTER_OTLP_TRACES_ENDPOINT="https://your-backend:4318/v1/traces"

# Base URL (SDK appends /v1/traces automatically)
export OTEL_EXPORTER_OTLP_ENDPOINT="https://your-backend:4318"
```

When an endpoint is configured, spans are exported via OTLP/HTTP. Any backend
that speaks OTLP works: Jaeger, Grafana Tempo, MLflow, Arize Phoenix,
Langfuse, SigNoz, Honeycomb, Datadog, etc.

If the endpoint is unreachable, the CLI continues normally — local files are
still produced and the run is not affected.

For precedence rules, protocol constraints, and other operational details, see
the [environment variable reference](../infrastructure/distributed-tracing.md#environment-variables).

### MLflow example

MLflow ≥ 3.6 ingests OTLP/HTTP natively at `{server}/v1/traces` and routes
traces to an experiment via a required header:

```bash
export OTEL_EXPORTER_OTLP_TRACES_ENDPOINT="https://mlflow.example.com/v1/traces"
export OTEL_EXPORTER_OTLP_TRACES_HEADERS="x-mlflow-experiment-id=42"
```

Header values are URL-decoded, so spaces are percent-encoded — for a
Basic-auth-fronted instance:

```bash
export OTEL_EXPORTER_OTLP_TRACES_HEADERS="authorization=Basic%20${CREDS_B64},x-mlflow-experiment-id=42"
```

> **Cost columns:** MLflow's per-trace cost is its own estimate — extracted
> input/output token counts priced against MLflow's internal model table. It
> excludes cache-creation/cache-read tokens, which dominate agent-run cost.
> The authoritative figure is the runtime-reported `fullsend.cost_usd` on
> `agent` spans (also in `run-telemetry.jsonl`).

## Enabling content capture (Level 3)

> **Planned:** Level 3 content capture is not yet implemented. This section
> documents the contract decided in
> [ADR 0050](../../ADRs/0050-distributed-tracing-instrumentation.md).

By default, spans contain metadata only (timing, token counts, tool names,
errors). To include full prompt/completion content in spans:

```bash
export OTEL_INSTRUMENTATION_GENAI_CAPTURE_MESSAGE_CONTENT=true
```

This follows the [OTEL GenAI semantic conventions](https://github.com/open-telemetry/semantic-conventions/blob/v1.37.0/docs/gen-ai/gen-ai-spans.md)
which mandate that content capture is opt-in. When enabled, spans include:

- System prompts and user messages
- Tool arguments and results (file contents, command output)
- Agent reasoning/thinking text
- Completion text

**Warning:** Only enable content capture when your backend's access controls
are appropriate for the sensitivity of the data. Content may include
proprietary source code, issue descriptions with PII, or credentials visible
in tool outputs.

## GHA workflow configuration

### Managed workflows

All agent stages (triage, code, review, fix, retro, prioritize, harness)
forward OTEL configuration. To enable export, set on the org (or repo)
that hosts the fullsend caller workflows:

1. Actions **variable** `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` — the backend's
   full traces URL (e.g. `https://mlflow.example.com/v1/traces`).
2. Actions **secret** `OTEL_EXPORTER_OTLP_TRACES_HEADERS` — the complete
   header string, auth and routing included (e.g.
   `Authorization=Bearer%20<token>,x-mlflow-experiment-id=42`).
3. Optional: Actions **variable** `OTEL_RESOURCE_ATTRIBUTES` — static
   `k=v,k=v` trace tags. The value is used verbatim: `${{ github.* }}`
   expressions evaluate only in workflow YAML, not in variables.

Installations scaffolded before OTEL support was added must also forward the
secret (add `OTEL_EXPORTER_OTLP_TRACES_HEADERS` under `secrets:`) until the
scaffold is re-synced: in the `.fullsend` repo's stage workflows (per-org),
or in the fullsend shim workflow's dispatch job (per-repo).

### Bring your own workflow

Add the environment variables to any job that runs `fullsend run`:

```yaml
env:
  OTEL_EXPORTER_OTLP_TRACES_ENDPOINT: "${{ vars.OTEL_EXPORTER_OTLP_TRACES_ENDPOINT }}"
  OTEL_EXPORTER_OTLP_TRACES_HEADERS: "${{ secrets.OTEL_EXPORTER_OTLP_TRACES_HEADERS }}"
```

Any variable and secret names work here — the values reach the exporter
as-is. Consult your backend's documentation for the endpoint URL and
authentication mechanism.

### Organizing traces for an org

Two conventions keep a shared backend navigable as repos onboard:

1. **One backend bucket per org.** On MLflow, create one experiment per org
   (e.g. `fullsend-<org>`) and point the org's header secret at its id. The
   backend's per-bucket access controls then align with org boundaries.
2. **Slice inside the bucket with resource attributes.** Standard OTel
   resource env is honored, so workflows can tag every trace with repo,
   agent, and environment:

   ```yaml
   env:
     OTEL_RESOURCE_ATTRIBUTES: "fullsend.repo=${{ github.repository }},fullsend.agent=triage,deployment.environment=prod"
   ```

   The example is inline workflow `env:`, where `${{ github.* }}` evaluates.
   On the managed path, set the `OTEL_RESOURCE_ATTRIBUTES` Actions variable
   to a static value instead — variables are not expression-expanded.

   These become filterable trace tags (enable them as columns in MLflow's
   Traces table). `fullsend.work_item_id` is on the root `run` span, so runs
   for the same issue correlate by filtering on the root span.

## Choosing a backend

Any OTLP-compatible backend works. Choosing an LLM-aware backend (MLflow,
Phoenix, Langfuse) activates GenAI dashboards — token cost rollups,
prompt/completion inspection, agent-specific views — without any CLI-side
configuration change. The `gen_ai.*` span attributes are recognized
automatically.

For production deployments, consult your backend's documentation for:
- High-availability configuration
- Authentication and access control
- Data retention policies
- Cost considerations for high-volume trace ingestion
