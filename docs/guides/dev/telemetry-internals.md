# Telemetry Internals

How fullsend's tracing system works, how to run with local backends during
development, and how to add new spans. For enabling telemetry in production,
see the [Telemetry user guide](../user/telemetry.md). For span structure and
attribute reference, see the
[Distributed Tracing Reference](../infrastructure/distributed-tracing.md).

Decided in [ADR 0050](../../ADRs/0050-distributed-tracing-instrumentation.md).

## How tracing works

Fullsend uses OpenTelemetry to emit structured spans at three opt-in levels:

1. **Level 1 (local baseline)** — every run writes `run-telemetry.jsonl` to
   the output directory. No configuration required, no data leaves the runner.
2. **Level 2 (OTLP export)** — when an OTLP endpoint is configured, metadata
   spans are exported live via the OTel SDK's batch processor.
3. **Level 3 (content capture)** — when explicitly opted in, full
   prompt/completion content is included in exported spans.

Spans are emitted at source during the run — not reconstructed afterwards.
The OTel SDK's batch processor flushes spans as they complete and does a
final flush on shutdown (5-second budget). If the endpoint is unreachable,
the run continues normally.

The local file and OTLP export are two views of the same trace with
identical span IDs. See the
[reference](../infrastructure/distributed-tracing.md#span-structure) for the
span hierarchy.

## Local development

Run an agent locally with traces going to a local backend:

1. Start a local Jaeger instance (OTLP-compatible):

   ```bash
   podman run -d --name jaeger \
     -p 16686:16686 \
     -p 4318:4318 \
     jaegertracing/jaeger
   ```

2. Point the exporter at it and run an agent:

   ```bash
   export OTEL_EXPORTER_OTLP_ENDPOINT="http://localhost:4318"
   fullsend run triage --issue 42
   ```

3. View the traces at <http://localhost:16686>.

### Other lightweight local backends

| Backend | Command | UI |
|---------|---------|-----|
| Jaeger | `podman run -p 16686:16686 -p 4318:4318 jaegertracing/jaeger` | `localhost:16686` |
| Arize Phoenix | `podman run -p 6006:6006 -p 4318:4318 arizephoenix/phoenix` | `localhost:6006` |
| MLflow ≥ 3.6 | `uvx "mlflow>=3.6" server --backend-store-uri sqlite:///mlflow.db` (native OTLP at `/v1/traces`; requires the `x-mlflow-experiment-id` header — see the [MLflow example](../user/telemetry.md#mlflow-example)) | `localhost:5000` |

## Adding new spans

Spans follow the
[OTEL GenAI semantic conventions](https://opentelemetry.io/docs/specs/semconv/gen-ai/).
When adding instrumentation:

1. Use the existing span hierarchy (`run` → `sandbox_create` / `agent`) as the
   parent context. See the
   [span structure reference](../infrastructure/distributed-tracing.md#span-structure).
2. Set `gen_ai.operation.name` on spans that represent agent operations.
3. Include token usage attributes (`gen_ai.usage.input_tokens`, etc.) on
   `agent` spans.
4. Add fullsend-specific attributes under the `fullsend.*` namespace. See the
   [custom attributes reference](../infrastructure/distributed-tracing.md#custom-attributes).
5. Never include prompt/completion content in spans unless
   `OTEL_INSTRUMENTATION_GENAI_CAPTURE_MESSAGE_CONTENT` is set — the
   metadata-by-default contract is a security boundary.

## Deferred work

From [ADR 0050](../../ADRs/0050-distributed-tracing-instrumentation.md):

1. **Sub-agent recursive span expansion** — When an agent dispatches
   sub-agents via `tool:Agent` (e.g., review agent's 6 sub-agents), their
   turns should become nested span subtrees, not flat spans. The transcript
   contract must handle recursive agent invocations.

2. **Pre/post script span instrumentation** — Pre-scripts, post-scripts, and
   validation scripts do significant work but aren't addressed in span
   structure. Define whether the framework instruments their execution
   automatically or provides a contract for scripts to emit spans.
