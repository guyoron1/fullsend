# Behaviour testing

Guide for writing and maintaining behaviour tests (BT) for the fullsend harness and agent pipeline.

Behaviour tests use `.feature` files with Gherkin syntax. Each scenario drives the harness with a **dummy agent** — a table of scripted operations that replace the real LLM agent — and asserts on the harness output (exit code, files produced, post-script mutations). The dummy agent table supports operations like `write_fixture`, `run_command`, and `set_env`.

## How a scenario runs

1. The test runner reads the `.feature` file and builds a dummy agent table.
2. The harness launches as normal: pre-script, sandbox creation, agent execution, validation loop, post-script.
3. Instead of an LLM, the sandbox executes the dummy agent operations in order.
4. Assertions check that the harness produced the expected output and side effects.

Because the harness validates `output/agent-result.json` against the stage's JSON Schema (via `validate-output-schema.sh`) and the post-script parses it, every scenario that dispatches an agent stage must emit a valid result file — otherwise the harness fails before the test can assert anything useful.

## Fixture authoring

This section covers the most common source of BT failures: missing or invalid fixture files.

### Checklist for new scenarios

When adding a new BT scenario for a dispatched agent stage (triage, review, code, fix, retro, prioritize), follow this checklist:

- [ ] **Include a `write_fixture` row** in the dummy agent table that emits `output/agent-result.json` (or the stage's custom output filename, e.g., `code-result.json` for the code stage).
- [ ] **Point at a fixture file** in `e2e/behaviour/fixtures/<stage>/` that conforms to the stage's result schema (`schemas/<stage>-result.schema.json`).
- [ ] **Verify schema conformance** by running `validate-output-schema.sh` against the fixture before committing (see the example below).
- [ ] **Check downstream validation** — fixture field values must satisfy not just the JSON schema but also the post-script's runtime checks. For example:
  - `head_sha` in review results must be a full-length (40-character) hex SHA, not a short prefix.
  - `action` values must be recognized by the post-script's `case` statement.
  - URLs (e.g., `blocked_by` in triage results) must match the regex pattern in the schema.
- [ ] **Check the harness config** for the stage (`harness/<stage>.yaml`). If `FULLSEND_OUTPUT_FILE` is set to a non-default name (e.g., `code-result.json`), the `write_fixture` row must use that filename, not `agent-result.json`.

### Existing fixtures

The table below lists the result schemas in `schemas/` and the required fields for each stage. When creating a new fixture, copy an existing one for the same stage and adapt the field values.

| Stage | Schema file | Output filename | Required fields |
|-------|-------------|-----------------|----------------|
| triage | `schemas/triage-result.schema.json` | `agent-result.json` | `action`, `reasoning`, `comment` (plus conditional fields per action) |
| review | `schemas/review-result.schema.json` | `agent-result.json` | `action`, `pr_number`, `repo` (plus conditional fields per action) |
| code | (post-script validated) | `code-result.json` | `target_branch` (see harness config) |
| fix | `schemas/fix-result.schema.json` | `fix-result.json` | `pr_number`, `trigger_source`, `actions`, `summary`, `tests_passed`, `files_changed` |
| retro | `schemas/retro-result.schema.json` | `agent-result.json` | `summary`, `proposals` |
| prioritize | `schemas/prioritize-result.schema.json` | `agent-result.json` | `reach`, `impact`, `confidence`, `effort`, `reasoning` |

**Conditional fields:** Some schemas use `allOf`/`if`/`then` rules. For example, the triage schema requires `clarity_scores` when `action` is `"insufficient"` and `triage_summary` when `action` is `"sufficient"`. Check the schema file for the full set of conditional requirements.

### Downstream validation beyond JSON Schema

The JSON schema is necessary but not sufficient. Post-scripts (`scripts/post-<stage>.sh`) perform additional runtime validation:

- **Triage:** `post-triage.sh` extracts `action`, `comment`, and conditionally `duplicate_of`, `blocked_by`, and `label_actions`. It validates that `GITHUB_ISSUE_URL` matches a regex, that a duplicate issue is not self-referential, and that sufficient results contain no `information_gaps`.
- **Review:** `post-review.sh` uses `head_sha` in the GitHub review submission API, which requires a full 40-character commit SHA — short hashes cause API errors.
- **Fix:** `post-fix.sh` uses `pr_number` and `trigger_source` to decide its workflow.

When a fixture passes schema validation but the BT scenario still fails, check the post-script for runtime guards that go beyond what the schema enforces.

### Example: adding a fixture for a new triage scenario

1. **Copy an existing fixture.** Start from a known-good fixture for the same stage:

   ```bash
   cp e2e/behaviour/fixtures/triage/sufficient.json \
      e2e/behaviour/fixtures/triage/my-new-scenario.json
   ```

2. **Edit the fixture** to match your scenario's expected output. For a "blocked" triage result:

   ```json
   {
     "action": "blocked",
     "reasoning": "Upstream dependency not yet resolved.",
     "comment": "This issue is blocked on https://github.com/org/repo/issues/99.",
     "blocked_by": "https://github.com/org/repo/issues/99"
   }
   ```

3. **Validate against the schema:**

   ```bash
   FULLSEND_OUTPUT_SCHEMA=schemas/triage-result.schema.json \
     python3 -c "
   import json, sys
   from jsonschema import validate
   with open(sys.argv[1]) as f: instance = json.load(f)
   with open(sys.argv[2]) as f: schema = json.load(f)
   validate(instance=instance, schema=schema)
   print('OK')
   " e2e/behaviour/fixtures/triage/my-new-scenario.json \
     schemas/triage-result.schema.json
   ```

4. **Add the `write_fixture` row** to your scenario's dummy agent table:

   ```gherkin
   Scenario: Triage blocks on upstream dependency
     Given a GitHub issue with an upstream blocker
     When the triage agent runs with dummy agent:
       | op            | path                        | fixture                                    |
       | write_fixture | output/agent-result.json    | fixtures/triage/my-new-scenario.json       |
     Then the harness exits successfully
     And the issue has label "blocked"
   ```

5. **Run the BT scenario** to confirm it passes end-to-end.
