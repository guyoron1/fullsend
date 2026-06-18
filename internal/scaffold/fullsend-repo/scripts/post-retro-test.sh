#!/usr/bin/env bash
# post-retro-test.sh — Test the comment-posting error handling in post-retro.sh.
#
# Uses a mock gh command to simulate API responses without hitting GitHub.
# Run from the repo root:
#   bash internal/scaffold/fullsend-repo/scripts/post-retro-test.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
POST_SCRIPT="${SCRIPT_DIR}/post-retro.sh"
FAILURES=0

# Create a temp directory for test fixtures and mock state.
TMPDIR="$(mktemp -d)"
trap 'rm -rf "${TMPDIR}"' EXIT

# Shared test fixture: minimal valid agent-result.json with one proposal.
VALID_RESULT='{"summary":"Retro complete.","proposals":[{"target_repo":"test-org/test-repo","title":"Improve error handling","what_happened":"Errors were not caught.","what_could_go_better":"Add try/catch.","proposed_change":"Wrap in error handler.","validation_criteria":"Tests pass."}]}'

# Shared test fixture: valid agent-result.json with no proposals.
VALID_RESULT_NO_PROPOSALS='{"summary":"Retro complete, no proposals.","proposals":[]}'

# ---------------------------------------------------------------------------
# Mock gh builder — creates a mock gh binary that behaves according to the
# test scenario. The mock records all calls to a log file and can be
# configured to fail on specific API calls.
#
# Arguments:
#   $1 — COMMENT_API_RC: exit code for the issues/comments API call
#   $2 — COMMENT_API_STDERR: stderr output for the issues/comments API call
# ---------------------------------------------------------------------------
setup_mock() {
  local comment_api_rc="${1:-0}"
  local comment_api_stderr="${2:-}"

  local mock_bin="${TMPDIR}/bin"
  mkdir -p "${mock_bin}"

  GH_LOG="${TMPDIR}/gh-calls.log"
  : > "${GH_LOG}"

  cat > "${mock_bin}/gh" <<MOCKEOF
#!/usr/bin/env bash
echo "gh \$*" >> "${GH_LOG}"

# Issue creation — always succeed and return a fake URL.
if [[ "\$1" == "issue" ]] && [[ "\$2" == "create" ]]; then
  echo "https://github.com/test-org/test-repo/issues/99"
  exit 0
fi

# Comment-posting API call — return configured behavior.
# Consume stdin to avoid SIGPIPE on the upstream jq pipe.
if [[ "\$1" == "api" ]] && [[ "\$2" == *"/comments"* ]]; then
  cat > /dev/null 2>&1 || true
  if [[ -n "${comment_api_stderr}" ]]; then
    echo "${comment_api_stderr}" >&2
  fi
  exit ${comment_api_rc}
fi

# Default: succeed silently.
exit 0
MOCKEOF
  chmod +x "${mock_bin}/gh"

  # Also mock jq to be available (it should be installed, but export PATH).
  export PATH="${mock_bin}:${PATH}"
}

# ---------------------------------------------------------------------------
# run_retro_test — run post-retro.sh with a fixture and check the result.
#
# Arguments:
#   $1 — test name
#   $2 — JSON content for agent-result.json
#   $3 — expected exit code (0 or 1)
#   $4 — expected pattern in stdout (optional, checked with grep -qF)
#   $5 — comment API exit code for mock
#   $6 — comment API stderr for mock
# ---------------------------------------------------------------------------
run_retro_test() {
  local test_name="$1"
  local json_content="$2"
  local expected_exit="$3"
  local expected_stdout="${4:-}"
  local comment_rc="${5:-0}"
  local comment_stderr="${6:-}"

  # Set up mock gh for this test.
  setup_mock "${comment_rc}" "${comment_stderr}"

  # Create iteration output structure.
  local run_dir="${TMPDIR}/run-${test_name}"
  mkdir -p "${run_dir}/iteration-1/output"
  echo "${json_content}" > "${run_dir}/iteration-1/output/agent-result.json"

  # Run the post-script.
  local exit_code=0
  (
    cd "${run_dir}"
    export ORIGINATING_URL="https://github.com/test-org/test-repo/issues/42"
    export GH_TOKEN="fake-token"
    bash "${POST_SCRIPT}"
  ) > "${TMPDIR}/stdout-${test_name}.log" 2>&1 || exit_code=$?

  if [[ ${exit_code} -ne ${expected_exit} ]]; then
    echo "FAIL: ${test_name} — expected exit ${expected_exit}, got ${exit_code}"
    cat "${TMPDIR}/stdout-${test_name}.log"
    FAILURES=$((FAILURES + 1))
    return
  fi

  if [[ -n "${expected_stdout}" ]]; then
    if ! grep -qF "${expected_stdout}" "${TMPDIR}/stdout-${test_name}.log"; then
      echo "FAIL: ${test_name} — expected stdout pattern '${expected_stdout}' not found"
      echo "Actual stdout:"
      cat "${TMPDIR}/stdout-${test_name}.log"
      FAILURES=$((FAILURES + 1))
      return
    fi
  fi

  echo "PASS: ${test_name}"
}

# --- Test cases ---

# 1. Happy path: comment posts successfully, script exits 0.
run_retro_test "comment-posts-successfully" \
  "${VALID_RESULT}" \
  0 \
  "Post-retro complete." \
  0 ""

# 2. HTTP 403 — script should emit a warning and exit 0.
run_retro_test "comment-403-non-fatal" \
  "${VALID_RESULT}" \
  0 \
  "::warning::Failed to post summary comment (HTTP auth error, non-fatal):" \
  1 "HTTP 403 - Resource not accessible by integration"

# 3. HTTP 401 — script should emit a warning and exit 0.
run_retro_test "comment-401-non-fatal" \
  "${VALID_RESULT}" \
  0 \
  "::warning::Failed to post summary comment (HTTP auth error, non-fatal):" \
  1 "HTTP 401 - Bad credentials"

# 4. Non-auth error (e.g., HTTP 500) — script should exit 1.
run_retro_test "comment-500-fatal" \
  "${VALID_RESULT}" \
  1 \
  "ERROR: failed to post summary comment:" \
  1 "HTTP 500 - Internal Server Error"

# 5. No proposals, comment succeeds — script exits 0.
run_retro_test "no-proposals-comment-succeeds" \
  "${VALID_RESULT_NO_PROPOSALS}" \
  0 \
  "Post-retro complete." \
  0 ""

# 6. No proposals, comment 403 — still exits 0.
run_retro_test "no-proposals-comment-403-non-fatal" \
  "${VALID_RESULT_NO_PROPOSALS}" \
  0 \
  "::warning::Failed to post summary comment (HTTP auth error, non-fatal):" \
  1 "HTTP 403 - Resource not accessible by integration"

# 7. GHA command injection in error output is sanitized.
run_retro_test "comment-403-output-sanitized" \
  "${VALID_RESULT}" \
  0 \
  "HTTP 403  :error :injected command" \
  1 "HTTP 403 ::error::injected command"

# --- Summary ---

echo ""
if [[ ${FAILURES} -gt 0 ]]; then
  echo "${FAILURES} test(s) failed"
  exit 1
fi
echo "All tests passed"
