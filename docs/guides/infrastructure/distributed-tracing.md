# Distributed Tracing Reference

Environment variables, span structure, and attribute reference for fullsend's
OpenTelemetry-based tracing. For a guide on enabling telemetry and choosing a
backend, see the [Telemetry user guide](../user/telemetry.md). For local
development and adding new spans, see
[Telemetry Internals](../dev/telemetry-internals.md).

Decided in [ADR 0050](../../ADRs/0050-distributed-tracing-instrumentation.md).

## Environment variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` | Signal-specific OTLP endpoint URL (used as-is — no `/v1/traces` appended). Takes precedence over the base URL. | Unset (no export) |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Base OTLP endpoint URL (SDK appends `/v1/traces` automatically). | Unset (no export) |
| `OTEL_EXPORTER_OTLP_TRACES_HEADERS` | Signal-specific HTTP headers for the traces endpoint. Takes precedence over base headers. | Unset |
| `OTEL_EXPORTER_OTLP_HEADERS` | Base HTTP headers for all OTLP signals. | Unset |
| `OTEL_SDK_DISABLED` | Set to `true` to disable all telemetry output (local file *and* OTLP export). | `false` |
| `OTEL_TRACES_EXPORTER` | Set to `none` to disable OTLP export only; the local `run-telemetry.jsonl` file is still written. | SDK default |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | Must be `http/protobuf` (the only supported protocol). Any other value skips export with a warning. | `http/protobuf` |
| `OTEL_EXPORTER_OTLP_CERTIFICATE` | Path to a PEM certificate bundle for backends behind a private CA. | System trust store |
| `OTEL_INSTRUMENTATION_GENAI_CAPTURE_MESSAGE_CONTENT` | Set to `true` to include full prompt/completion content in spans (Level 3). | `false` |
| `OTEL_RESOURCE_ATTRIBUTES` | Static `k=v,k=v` resource attributes added to all spans. | Unset |
| `TRACEPARENT` | W3C Trace Context header for cross-run correlation. | Unset |

**Precedence:** `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` >
`OTEL_EXPORTER_OTLP_ENDPOINT`. Headers follow the same pattern:
`OTEL_EXPORTER_OTLP_TRACES_HEADERS` > `OTEL_EXPORTER_OTLP_HEADERS`.

## Operational details

- **Export timing:** spans are exported live via the OTel SDK's batch
  processor as they complete. On shutdown, the provider flushes remaining
  spans within a 5-second budget. A dead endpoint does not block the run.
- **Crashed runs:** completed spans that were already flushed mid-run reach
  the backend; spans still in the batch buffer are lost. The local
  `run-telemetry.jsonl` (written synchronously per span) remains the
  forensic record.
- **Sampling:** when the run continues an inbound `TRACEPARENT` whose W3C
  sampled flag is unset (`-00`), the upstream sampling decision is respected:
  nothing is exported. The local file is still written.
- **Protocol:** OTLP over `http/protobuf` only. Setting
  `OTEL_EXPORTER_OTLP_PROTOCOL` (or the traces-specific variant) to anything
  else — e.g. `grpc` — skips export with a warning rather than posting
  protobuf at a gRPC endpoint.
- **Validation:** a malformed endpoint value skips export with a warning; it
  is never silently replaced with the SDK's `localhost:4318` default.
- **Kill switches:** `OTEL_SDK_DISABLED=true` disables all telemetry output
  (OTLP export *and* the local file). `OTEL_TRACES_EXPORTER=none` disables
  only the OTLP export; the local file is still written.
- **Private CAs:** point `OTEL_EXPORTER_OTLP_CERTIFICATE` at a PEM bundle for
  backends with certificates outside the system trust store. There is no
  skip-verify option.

## Span structure

A run produces this span hierarchy (span names match the `name` field in
`run-telemetry.jsonl` — the exported spans and the local file are two views
of the same trace, with identical span ids):

```
run (root; Consumer when dispatched with TRACEPARENT, else Internal)
├── sandbox_create (gen_ai.operation.name=create_agent)
└── agent           (one per iteration; gen_ai.operation.name=invoke_agent)
```

### GenAI semantic conventions

Spans carry [OTEL GenAI semantic convention](https://opentelemetry.io/docs/specs/semconv/gen-ai/) attributes:

| Attribute | Example | On |
|-----------|---------|-----|
| `gen_ai.operation.name` | `invoke_agent` | `run` and `agent` spans (`create_agent` on `sandbox_create`) |
| `gen_ai.agent.name` | `triage` | `run` and `agent` spans |
| `gen_ai.request.model` | `claude-opus-4-6` | `agent` spans (resolved model) |
| `gen_ai.system` | `anthropic` | `agent` spans (the model vendor, from the runtime) |
| `gen_ai.usage.input_tokens` / `output_tokens` / `cache_*_input_tokens` | `109938` | `agent` spans |

These attributes enable LLM-aware backends to recognize fullsend spans as
agent operations and surface them in GenAI-specific dashboards.

### SpanKind

- **Consumer**: The root span when a valid inbound `TRACEPARENT` was adopted
  (the run was dispatched by an instrumented system).
- **Internal**: The root span for local/manual invocations, and all child
  spans.

## Custom attributes

Fullsend-specific attributes:

| Attribute | On | Description |
|-----------|-----|-------------|
| `fullsend.work_item_id` | `run` span | Work item identity (e.g. `owner/repo#123`) — the primary cross-run correlation key |
| `fullsend.cost_usd` | `agent` spans | Iteration cost in USD, rounded to cents |
| `fullsend.tool_calls` | `agent` spans | Tool invocations in the iteration |
| `fullsend.agent` | `run` span | Agent name (renamed from bare `agent` in the OTel SDK migration) |

## Cross-run trace correlation

Multi-agent pipelines (triage → code → review) propagate trace context via
the `TRACEPARENT` environment variable (W3C Trace Context).

When a workflow dispatches a child run:

```yaml
env:
  TRACEPARENT: ${{ steps.parent.outputs.traceparent }}
```

The child run's root span becomes part of the parent trace, creating a
unified view of the entire pipeline.

For separate workflow runs on the same work item (triage → code → review as
independent GHA workflows), `TRACEPARENT` must be propagated manually — for
example, via hidden issue/PR comments. GitHub webhooks do not support custom
trace headers natively.
