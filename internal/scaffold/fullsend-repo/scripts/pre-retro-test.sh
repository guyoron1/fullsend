#!/usr/bin/env bash
# pre-retro-test.sh — Test prefetch logic in pre-retro.sh
#
# Verifies that the PR context prefetch works correctly: produces valid
# JSON for PR and issue URLs, handles failures gracefully (non-blocking),
# and respects size limits.
#
# Run from the repo root:
#   bash internal/scaffold/fullsend-repo/scripts/pre-retro-test.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT="${SCRIPT_DIR}/pre-retro.sh"
FAILURES=0

TMPDIR="$(mktemp -d)"
trap 'rm -rf "${TMPDIR}"' EXIT

# --- Helpers ---

# build_mock creates a mock gh binary that returns preconfigured JSON
# responses based on the subcommand and arguments.
#   $1 — JSON for gh pr view
#   $2 — JSON for gh api .../comments
#   $3 — JSON for gh api .../reviews
#   $4 — JSON for gh api .../actions/runs (optional, defaults to empty)
#   $5 — JSON for gh api .../issues/N (optional, for issue metadata)
build_mock() {
  local pr_meta="${1:-}"
  local comments="${2:-[]}"
  local reviews="${3:-[]}"
  local workflow_runs="${4:-}"
  local issue_meta="${5:-}"
  local mock_bin="${TMPDIR}/bin"

  rm -rf "${mock_bin}"
  mkdir -p "${mock_bin}"

  printf '%s' "${pr_meta}" > "${TMPDIR}/pr-meta.json"
  printf '%s' "${comments}" > "${TMPDIR}/comments.json"
  printf '%s' "${reviews}" > "${TMPDIR}/reviews.json"
  printf '%s' "${workflow_runs:-"{\"workflow_runs\":[]}"}" > "${TMPDIR}/runs.json"
  printf '%s' "${issue_meta}" > "${TMPDIR}/issue-meta.json"

  cat > "${mock_bin}/gh" <<MOCKSCRIPT
#!/usr/bin/env bash
set -euo pipefail

# Route based on subcommand and endpoint
if [[ "\$1" == "pr" && "\$2" == "view" ]]; then
  cat "${TMPDIR}/pr-meta.json"
  exit 0
fi

if [[ "\$1" == "api" ]]; then
  endpoint="\$2"
  jq_filter=""
  prev_arg=""
  for arg in "\$@"; do
    if [[ "\${prev_arg}" == "--jq" ]]; then
      jq_filter="\${arg}"
    fi
    prev_arg="\${arg}"
  done

  respond() {
    if [[ -n "\${jq_filter}" ]]; then
      jq "\${jq_filter}" "\$1"
    else
      cat "\$1"
    fi
  }

  if [[ "\${endpoint}" == *"/comments"* ]]; then
    respond "${TMPDIR}/comments.json"
    exit 0
  fi
  if [[ "\${endpoint}" == *"/reviews"* ]]; then
    respond "${TMPDIR}/reviews.json"
    exit 0
  fi
  if [[ "\${endpoint}" == *"/actions/runs"* ]]; then
    respond "${TMPDIR}/runs.json"
    exit 0
  fi
  if [[ "\${endpoint}" == *"/issues/"* ]]; then
    respond "${TMPDIR}/issue-meta.json"
    exit 0
  fi
fi

echo "mock gh: unhandled command: \$*" >&2
exit 1
MOCKSCRIPT

  chmod +x "${mock_bin}/gh"
  echo "${mock_bin}"
}

# build_failing_mock creates a gh binary that always fails.
build_failing_mock() {
  local mock_bin="${TMPDIR}/bin-fail"
  rm -rf "${mock_bin}"
  mkdir -p "${mock_bin}"
  cat > "${mock_bin}/gh" <<'FAILSCRIPT'
#!/usr/bin/env bash
echo "gh: token is invalid" >&2
exit 1
FAILSCRIPT
  chmod +x "${mock_bin}/gh"
  echo "${mock_bin}"
}

# build_partial_failing_mock creates a gh binary where `gh pr view` succeeds
# but all `gh api` calls fail — exercises the per-endpoint fallback paths.
build_partial_failing_mock() {
  local pr_meta="${1}"
  local mock_bin="${TMPDIR}/bin-partial"
  rm -rf "${mock_bin}"
  mkdir -p "${mock_bin}"

  printf '%s' "${pr_meta}" > "${TMPDIR}/pr-meta-partial.json"

  cat > "${mock_bin}/gh" <<MOCKSCRIPT
#!/usr/bin/env bash
set -euo pipefail

if [[ "\$1" == "pr" && "\$2" == "view" ]]; then
  cat "${TMPDIR}/pr-meta-partial.json"
  exit 0
fi

echo "gh api: server error" >&2
exit 1
MOCKSCRIPT

  chmod +x "${mock_bin}/gh"
  echo "${mock_bin}"
}

run_test() {
  local test_name="$1"
  shift

  echo "--- TEST: ${test_name}"
  if "$@"; then
    echo "    PASS"
  else
    echo "    FAIL"
    FAILURES=$((FAILURES + 1))
  fi
}

# --- Test Data ---

PR_META='{
  "title": "Fix widget",
  "body": "Fixes the widget",
  "state": "MERGED",
  "author": {"login": "alice"},
  "labels": [{"name": "bug"}],
  "baseRefName": "main",
  "headRefName": "fix-widget",
  "headRefOid": "abc123",
  "additions": 10,
  "deletions": 5,
  "changedFiles": 2,
  "commits": [{"oid": "abc123"}],
  "createdAt": "2026-01-01T00:00:00Z",
  "closedAt": "2026-01-02T00:00:00Z",
  "mergedAt": "2026-01-02T00:00:00Z",
  "mergedBy": {"login": "bob"},
  "number": 42,
  "url": "https://github.com/org/repo/pull/42"
}'

COMMENTS='[{"id":1,"body":"LGTM","user":{"login":"bob"}}]'
REVIEWS='[{"id":2,"state":"APPROVED","user":{"login":"bob"},"body":"Approved"}]'
WORKFLOW_RUNS='{"workflow_runs":[{"id":100,"name":"CI","status":"completed","conclusion":"success","created_at":"2026-01-01T01:00:00Z","html_url":"https://github.com/org/repo/actions/runs/100","event":"pull_request"}]}'

ISSUE_META='{"number":99,"title":"Bug report","body":"Something broke","state":"open","user":{"login":"alice"},"labels":[{"name":"bug"}]}'

# --- Tests ---

test_pr_prefetch() {
  local mock_bin
  mock_bin="$(build_mock "${PR_META}" "${COMMENTS}" "${REVIEWS}" "${WORKFLOW_RUNS}")"
  local workspace="${TMPDIR}/ws-pr"
  mkdir -p "${workspace}"

  ORIGINATING_URL="https://github.com/org/repo/pull/42" \
  REPO_FULL_NAME="org/repo" \
  GH_TOKEN="test-token" \
  GITHUB_WORKSPACE="${workspace}" \
  PATH="${mock_bin}:${PATH}" \
    bash "${SCRIPT}" > /dev/null 2>&1

  # Verify output file exists and is valid JSON
  [[ -f "${workspace}/pr-context.json" ]] || { echo "    pr-context.json not created"; return 1; }
  jq empty "${workspace}/pr-context.json" || { echo "    Invalid JSON"; return 1; }

  # Verify expected fields
  local source_type source_number
  source_type=$(jq -r '.source_type' "${workspace}/pr-context.json")
  source_number=$(jq -r '.source_number' "${workspace}/pr-context.json")
  [[ "${source_type}" == "pull" ]] || { echo "    source_type=${source_type}, expected pull"; return 1; }
  [[ "${source_number}" == "42" ]] || { echo "    source_number=${source_number}, expected 42"; return 1; }

  # Verify PR metadata is present
  local pr_title
  pr_title=$(jq -r '.pr.title' "${workspace}/pr-context.json")
  [[ "${pr_title}" == "Fix widget" ]] || { echo "    pr.title=${pr_title}, expected Fix widget"; return 1; }

  # Verify comments and reviews
  local comment_count review_count
  comment_count=$(jq '.comments | length' "${workspace}/pr-context.json")
  review_count=$(jq '.reviews | length' "${workspace}/pr-context.json")
  [[ "${comment_count}" == "1" ]] || { echo "    comments count=${comment_count}, expected 1"; return 1; }
  [[ "${review_count}" == "1" ]] || { echo "    reviews count=${review_count}, expected 1"; return 1; }
}

test_issue_prefetch() {
  local mock_bin
  mock_bin="$(build_mock "" "[]" "[]" "" "${ISSUE_META}")"
  local workspace="${TMPDIR}/ws-issue"
  mkdir -p "${workspace}"

  ORIGINATING_URL="https://github.com/org/repo/issues/99" \
  REPO_FULL_NAME="org/repo" \
  GH_TOKEN="test-token" \
  GITHUB_WORKSPACE="${workspace}" \
  PATH="${mock_bin}:${PATH}" \
    bash "${SCRIPT}" > /dev/null 2>&1

  [[ -f "${workspace}/pr-context.json" ]] || { echo "    pr-context.json not created"; return 1; }
  jq empty "${workspace}/pr-context.json" || { echo "    Invalid JSON"; return 1; }

  local source_type
  source_type=$(jq -r '.source_type' "${workspace}/pr-context.json")
  [[ "${source_type}" == "issue" ]] || { echo "    source_type=${source_type}, expected issue"; return 1; }

  local issue_title
  issue_title=$(jq -r '.issue.title' "${workspace}/pr-context.json")
  [[ "${issue_title}" == "Bug report" ]] || { echo "    issue.title=${issue_title}, expected Bug report"; return 1; }
}

test_no_token_skips_prefetch() {
  local workspace="${TMPDIR}/ws-notoken"
  mkdir -p "${workspace}"

  # No GH_TOKEN — prefetch should skip gracefully
  ORIGINATING_URL="https://github.com/org/repo/pull/42" \
  REPO_FULL_NAME="org/repo" \
  GH_TOKEN="" \
  GITHUB_WORKSPACE="${workspace}" \
    bash "${SCRIPT}" > /dev/null 2>&1

  # Script should succeed but no file created
  [[ ! -f "${workspace}/pr-context.json" ]] || { echo "    pr-context.json should not exist without GH_TOKEN"; return 1; }
}

test_api_failure_non_blocking() {
  local mock_bin
  mock_bin="$(build_failing_mock)"
  local workspace="${TMPDIR}/ws-fail"
  mkdir -p "${workspace}"

  # API fails — script should still exit 0
  local exit_code=0
  ORIGINATING_URL="https://github.com/org/repo/pull/42" \
  REPO_FULL_NAME="org/repo" \
  GH_TOKEN="test-token" \
  GITHUB_WORKSPACE="${workspace}" \
  PATH="${mock_bin}:${PATH}" \
    bash "${SCRIPT}" > /dev/null 2>&1 || exit_code=$?

  [[ "${exit_code}" -eq 0 ]] || { echo "    Script exited ${exit_code}, expected 0"; return 1; }
}

test_url_validation_still_works() {
  local exit_code=0
  ORIGINATING_URL="not-a-url" \
    bash "${SCRIPT}" > /dev/null 2>&1 || exit_code=$?

  [[ "${exit_code}" -ne 0 ]] || { echo "    Should have failed on invalid URL"; return 1; }
}

test_missing_originating_url_fails() {
  local exit_code=0
  unset ORIGINATING_URL
  bash "${SCRIPT}" > /dev/null 2>&1 || exit_code=$?

  [[ "${exit_code}" -ne 0 ]] || { echo "    Should have failed without ORIGINATING_URL"; return 1; }
}

test_repo_parsed_from_url() {
  # No REPO_FULL_NAME set — should parse from URL
  local mock_bin
  mock_bin="$(build_mock "${PR_META}" "${COMMENTS}" "${REVIEWS}" "${WORKFLOW_RUNS}")"
  local workspace="${TMPDIR}/ws-parse"
  mkdir -p "${workspace}"

  ORIGINATING_URL="https://github.com/org/repo/pull/42" \
  REPO_FULL_NAME="" \
  GH_TOKEN="test-token" \
  GITHUB_WORKSPACE="${workspace}" \
  PATH="${mock_bin}:${PATH}" \
    bash "${SCRIPT}" > /dev/null 2>&1

  [[ -f "${workspace}/pr-context.json" ]] || { echo "    pr-context.json not created"; return 1; }

  local source_repo
  source_repo=$(jq -r '.source_repo' "${workspace}/pr-context.json")
  [[ "${source_repo}" == "org/repo" ]] || { echo "    source_repo=${source_repo}, expected org/repo"; return 1; }
}

test_partial_api_failure_writes_output() {
  local mock_bin workspace
  mock_bin="$(build_partial_failing_mock "${PR_META}")"
  workspace="${TMPDIR}/ws-partial"
  mkdir -p "${workspace}"

  ORIGINATING_URL="https://github.com/org/repo/pull/42" \
  REPO_FULL_NAME="org/repo" \
  GH_TOKEN="test-token" \
  GITHUB_WORKSPACE="${workspace}" \
  PATH="${mock_bin}:${PATH}" \
    bash "${SCRIPT}" > /dev/null 2>&1

  # Output file should exist — PR metadata succeeded, API failures fell back to []
  [[ -f "${workspace}/pr-context.json" ]] || { echo "    pr-context.json not created"; return 1; }
  jq empty "${workspace}/pr-context.json" || { echo "    Invalid JSON"; return 1; }

  # PR metadata present
  local pr_title
  pr_title=$(jq -r '.pr.title' "${workspace}/pr-context.json")
  [[ "${pr_title}" == "Fix widget" ]] || { echo "    pr.title=${pr_title}, expected Fix widget"; return 1; }

  # Comments and reviews fell back to empty arrays
  local comment_count review_count
  comment_count=$(jq '.comments | length' "${workspace}/pr-context.json")
  review_count=$(jq '.reviews | length' "${workspace}/pr-context.json")
  [[ "${comment_count}" == "0" ]] || { echo "    comments count=${comment_count}, expected 0"; return 1; }
  [[ "${review_count}" == "0" ]] || { echo "    reviews count=${review_count}, expected 0"; return 1; }
}

test_size_limit_enforcement() {
  local mock_bin workspace
  mock_bin="$(build_mock "${PR_META}" "${COMMENTS}" "${REVIEWS}" "${WORKFLOW_RUNS}")"
  workspace="${TMPDIR}/ws-size"
  mkdir -p "${workspace}"

  # ponytail: mock wc to report >5MB instead of generating a truly huge payload
  # (jq --argjson hits ARG_MAX before we can reach the size check with real data)
  cat > "${mock_bin}/wc" <<'WCMOCK'
#!/usr/bin/env bash
echo "6000000"
WCMOCK
  chmod +x "${mock_bin}/wc"

  ORIGINATING_URL="https://github.com/org/repo/pull/42" \
  REPO_FULL_NAME="org/repo" \
  GH_TOKEN="test-token" \
  GITHUB_WORKSPACE="${workspace}" \
  PATH="${mock_bin}:${PATH}" \
    bash "${SCRIPT}" > /dev/null 2>&1 || true

  # File should have been removed — mock wc reports >5MB
  [[ ! -f "${workspace}/pr-context.json" ]] || { echo "    pr-context.json should be removed (over 5MB limit)"; return 1; }
}

# --- Run ---

run_test "PR prefetch produces valid context JSON" test_pr_prefetch
run_test "Issue prefetch produces valid context JSON" test_issue_prefetch
run_test "No GH_TOKEN skips prefetch gracefully" test_no_token_skips_prefetch
run_test "API failure is non-blocking" test_api_failure_non_blocking
run_test "URL validation still rejects bad URLs" test_url_validation_still_works
run_test "Missing ORIGINATING_URL fails" test_missing_originating_url_fails
run_test "Repo parsed from URL when REPO_FULL_NAME empty" test_repo_parsed_from_url
run_test "Partial API failure still writes output with fallbacks" test_partial_api_failure_writes_output
run_test "Size limit enforcement removes oversized file" test_size_limit_enforcement

echo ""
if [[ "${FAILURES}" -gt 0 ]]; then
  echo "FAILED: ${FAILURES} test(s) failed"
  exit 1
else
  echo "All tests passed."
fi
