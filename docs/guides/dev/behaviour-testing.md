# Behaviour Testing

Writing behaviour tests with dummy agents, fixture authoring checklist, and result schema validation.

Behaviour tests (BT) use `.feature` files with Gherkin syntax to test the fullsend harness end-to-end. Unlike traditional unit tests, BT scenarios dispatch real agent stages using dummy agents (pre-written transcripts) to validate the full pipeline without calling the inference API.

## Overview

A behaviour test consists of:

- **Feature file** — Gherkin scenarios describing the test case (Given/When/Then format)
- **Dummy agent table** — Pre-written agent transcripts that emit fixtures for each dispatched stage
- **Harness pipeline** — The same runtime that runs in production (sandbox, validation, post-scripts)
- **Assertions** — Validation that the harness produced the expected GitHub mutations or output artifacts

The harness executes BT scenarios identically to production runs, except it reads agent output from the dummy agent table instead of calling the inference API. This makes BT fast, deterministic, and independent of model behavior.

## How a scenario runs

1. The test harness reads the `.feature` file and parses the scenario
2. For each dispatched stage (triage, review, fix, retro, prioritize), the harness looks up the dummy agent table
3. The dummy agent emits a pre-written fixture (JSON conforming to the stage's result schema) to `output/agent-result.json`
4. The harness validates the fixture against the stage's JSON Schema (`internal/scaffold/fullsend-repo/schemas/<stage>-result.schema.json`)
5. The harness runs the post-script (`internal/scaffold/fullsend-repo/scripts/post-<stage>.sh`) which performs GitHub mutations and downstream validation
6. Assertions check the final state (labels, comments, PR reviews, etc.)

## Fixture authoring

Every BT scenario that dispatches an agent stage **must** include a `write_fixture` row in the dummy agent table. The fixture must:

- Be written to `output/agent-result.json` (relative to the iteration directory)
- Conform to the stage's result schema (see table below)
- Satisfy downstream validation in the post-script (JSON Schema alone is not sufficient)

### Checklist for new BT scenarios

When writing a new BT scenario that dispatches agent stages:

- [ ] **Create a `write_fixture` row** for each dispatched stage
- [ ] **Emit `output/agent-result.json`** with valid JSON conforming to the stage's schema
- [ ] **Include all required fields** per the schema table below
- [ ] **Include conditional required fields** based on the `action` value
- [ ] **Test downstream validation** — ensure the post-script accepts the fixture (e.g., `head_sha` must be a full 40-char SHA for review, `duplicate_of` cannot equal the current issue number for triage)
- [ ] **Avoid trailing whitespace** in fixture JSON to pass linting

### Stage result schemas

| Stage | Schema file | Default output | Required fields |
|-------|-------------|----------------|-----------------|
| triage | `triage-result.schema.json` | `agent-result.json` | `action`, `reasoning`, `comment` (+ conditional: `clarity_scores` for insufficient/sufficient, `duplicate_of` for duplicate, `triage_summary` for sufficient, `blocked_by` for blocked) |
| review | `review-result.schema.json` | `agent-result.json` | `action`, `pr_number`, `repo` (+ conditional: `body`+`head_sha` for approve/request-changes/comment/reject, `findings` for request-changes/reject, `reason` for failure) |
| fix | `fix-result.schema.json` | `agent-result.json` | `pr_number`, `trigger_source`, `actions`, `summary`, `tests_passed`, `files_changed` |
| retro | `retro-result.schema.json` | `agent-result.json` | `summary`, `proposals` |
| prioritize | `prioritize-result.schema.json` | `agent-result.json` | `reach`, `impact`, `confidence`, `effort`, `reasoning` |

**Conditional requirements** are enforced by JSON Schema `allOf` rules. For example:
- **Triage:** `action: insufficient` requires `clarity_scores`; `action: duplicate` requires `duplicate_of`; `action: sufficient` requires `clarity_scores` and `triage_summary`; `action: blocked` requires `blocked_by`
- **Review:** `action: approve|request-changes|comment|reject` requires `body` and `head_sha`; `action: request-changes|reject` also requires `findings`; `action: failure` requires `reason`

### Downstream validation beyond JSON Schema

Post-scripts perform additional validation that JSON Schema cannot express:

- **Triage (`post-triage.sh`):**
  - `GITHUB_ISSUE_URL` must match regex `^https://github\.com/[^/]+/[^/]+/issues/[0-9]+$`
  - `duplicate_of` cannot equal the current issue number (self-referential duplicates rejected)
  - `action: sufficient` with non-empty `information_gaps` array is rejected (schema allows it, post-script forbids it)
  - `action: blocked` must have `blocked_by` present (enforced by schema, but post-script also validates format)

- **Review (`post-review.sh`):**
  - `head_sha` is used in the GitHub review submission API, which requires a full 40-character SHA (schema only enforces `minLength: 7`, but the API call will fail with shorter SHAs)
  - Protected-path downgrade logic runs in the post-script — if the PR touches sensitive paths, `action: approve` is silently downgraded to `action: comment` before posting (does not modify the fixture)

- **Validation script (`validate-output-schema.sh`):**
  - Uses env var `FULLSEND_OUTPUT_SCHEMA` for schema path
  - Uses env var `FULLSEND_OUTPUT_FILE` for output filename (defaults to `agent-result.json`)
  - Falls back to `result.json` if `agent-result.json` is not found (common agent mistake)

### Worked example: adding a new triage fixture

Scenario: A bug report arrives and the triage agent marks it as `sufficient` (ready to code).

**Step 1:** Identify required fields from the schema table above:
- Base: `action`, `reasoning`, `comment`
- Conditional (for `action: sufficient`): `clarity_scores`, `triage_summary`

**Step 2:** Create the fixture JSON:

```json
{
  "action": "sufficient",
  "reasoning": "Issue provides clear symptom, reproduction steps, and impact. The root cause hypothesis is plausible and the proposed fix is specific enough for implementation.",
  "comment": "Thank you for the detailed bug report. This issue has been triaged and is ready for implementation.\n\n**Summary:**\n- **Severity:** medium\n- **Category:** bug\n- **Impact:** Users cannot save preferences when localStorage is disabled\n\n**Recommended fix:** Add a fallback to cookies or in-memory state when localStorage.setItem throws QuotaExceededError.\n\n**Next steps:** This issue is now labeled `ready-to-code` and will be prioritized for the next development cycle.",
  "clarity_scores": {
    "symptom": 0.9,
    "cause": 0.7,
    "reproduction": 1.0,
    "impact": 0.8,
    "overall": 0.85
  },
  "triage_summary": {
    "title": "Preferences not saved when localStorage disabled",
    "severity": "medium",
    "category": "bug",
    "problem": "User preferences (theme, language, notifications) are not persisted when localStorage is disabled or quota exceeded.",
    "root_cause_hypothesis": "The app uses localStorage.setItem without error handling. When storage is unavailable, the exception is uncaught and preferences silently fail to save.",
    "reproduction_steps": [
      "Open browser in private/incognito mode with localStorage disabled",
      "Navigate to Settings and change theme to dark mode",
      "Refresh the page",
      "Observe theme reverts to default (light mode)"
    ],
    "environment": "Chrome 118, Firefox 119 (any browser with localStorage disabled)",
    "impact": "All users in private browsing or with storage disabled cannot persist preferences. Approximately 5-10% of sessions based on telemetry.",
    "recommended_fix": "Wrap localStorage.setItem in try-catch. On QuotaExceededError or SecurityError, fall back to sessionStorage or in-memory Map. Display a non-blocking toast notification when fallback is used.",
    "proposed_test_case": "Unit test: mock localStorage.setItem to throw QuotaExceededError, verify preferences still saved to fallback. E2E: disable localStorage in browser flags, verify preferences persist across page reloads using sessionStorage."
  }
}
```

**Step 3:** Add the `write_fixture` row to the dummy agent table:

```gherkin
When the triage agent runs
  | write_fixture | output/agent-result.json | <fixture JSON from Step 2> |
```

**Step 4:** Verify downstream validation passes:
- No trailing whitespace in JSON
- `clarity_scores.overall` is the mean of the other four scores (or close enough)
- `triage_summary.severity` is one of `critical|high|medium|low`
- `triage_summary.category` is one of `bug|performance|security|documentation|feature|other`
- `triage_summary.reproduction_steps` has at least one item

**Step 5:** Run the BT scenario and check assertions:
- Issue has `ready-to-code` label applied
- Issue has `triaged` label applied
- Issue has a comment matching the `comment` field
- No `needs-info` or `duplicate` labels

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| `FAIL: agent-result.json not found` | Dummy agent did not emit fixture, or emitted to wrong path | Add `write_fixture` row with `output/agent-result.json` |
| `FAIL: schema validation error: 'foo' is a required property` | Fixture is missing a required field | Check schema table above, add missing field |
| `post-triage.sh: ERROR: GITHUB_ISSUE_URL does not match expected pattern` | Fixture URL has wrong format | Use full HTML URL: `https://github.com/org/repo/issues/42` |
| `post-review.sh: ERROR: failed to post review` | `head_sha` is too short (< 40 chars) | Use full 40-character commit SHA |
| `ERROR: duplicate_of cannot be the current issue` | Triage fixture marks issue as duplicate of itself | Change `duplicate_of` to a different issue number |
| JSON parse error in post-script | Trailing whitespace, invalid escaping, or syntax error | Run fixture JSON through `jq .` to validate and format |
