#!/usr/bin/env bash
# check-deprecated-scaffold-test.sh — Tests for check-deprecated-scaffold.sh
#
# Run from repo root: bash scripts/check-deprecated-scaffold-test.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHECK_SCRIPT="${SCRIPT_DIR}/check-deprecated-scaffold.sh"
FAILURES=0

TMPDIR="$(mktemp -d)"
trap 'rm -rf "${TMPDIR}"' EXIT

MOCK_BIN="${TMPDIR}/bin"
mkdir -p "${MOCK_BIN}"

# --- Mock gh that simulates `gh api ... --jq '.[].filename'` ---
GH_RESPONSE_FILE="${TMPDIR}/gh-response.txt"

cat >"${MOCK_BIN}/gh" <<'MOCK'
#!/usr/bin/env bash
cat "${GH_RESPONSE_FILE}"
MOCK
chmod +x "${MOCK_BIN}/gh"
export PATH="${MOCK_BIN}:${PATH}"
export GH_TOKEN="test-token"
export GH_RESPONSE_FILE

# --- Helpers ---
# Write filenames one per line (the output format of gh api --jq '.[].filename')
set_gh_response() {
  printf '%s\n' "$@" > "${GH_RESPONSE_FILE}"
}

make_event_file() {
  local pr_number="$1"
  local event_file="${TMPDIR}/event-${pr_number}.json"
  echo "{\"pull_request\":{\"number\":${pr_number}}}" > "${event_file}"
  echo "${event_file}"
}

run_case() {
  local name="$1"
  local expected_exit="$2"
  local expected_grep="${3:-}"

  local output rc=0
  output="$("${CHECK_SCRIPT}" 2>&1)" || rc=$?

  if [[ "${rc}" -ne "${expected_exit}" ]]; then
    echo "FAIL: ${name} — expected exit ${expected_exit}, got ${rc}"
    echo "  output: ${output}"
    FAILURES=$((FAILURES + 1))
    return
  fi

  if [[ -n "${expected_grep}" ]] && ! echo "${output}" | grep -q "${expected_grep}"; then
    echo "FAIL: ${name} — expected output to contain '${expected_grep}'"
    echo "  output: ${output}"
    FAILURES=$((FAILURES + 1))
    return
  fi

  echo "PASS: ${name}"
}

# --- Test cases ---

# 1. Not in CI — should skip
unset GITHUB_ACTIONS GITHUB_EVENT_NAME GITHUB_REPOSITORY GITHUB_EVENT_PATH
run_case "not in CI — skips" 0 "skipping"

# 2. In CI but not a PR — should skip
export GITHUB_ACTIONS="true"
export GITHUB_EVENT_NAME="push"
run_case "push event — skips" 0 "not a CI pull_request"

# 3. PR with no deprecated files — should pass
export GITHUB_EVENT_NAME="pull_request"
export GITHUB_REPOSITORY="test-org/test-repo"
export GITHUB_EVENT_PATH="$(make_event_file 42)"
set_gh_response "cmd/main.go" "docs/README.md"
run_case "PR with no deprecated files — passes" 0 "no deprecated scaffold paths"

# 4. PR with deprecated files — should fail
set_gh_response "internal/scaffold/fullsend-repo/skills/docs-review/SKILL.md" "cmd/main.go"
run_case "PR with deprecated scaffold file — fails" 1 "fullsend-ai/agents"

# 5. PR with only deprecated files — should fail
set_gh_response "internal/scaffold/fullsend-repo/AGENTS.md"
run_case "PR with only deprecated file — fails" 1 "fullsend-ai/agents"

# 6. Missing event path — should skip
export GITHUB_EVENT_PATH="/nonexistent/path.json"
run_case "missing event path — skips" 0 "skipping"

# --- Summary ---
if [[ "${FAILURES}" -gt 0 ]]; then
  echo "${FAILURES} test(s) failed"
  exit 1
fi

echo "All check-deprecated-scaffold tests passed."
