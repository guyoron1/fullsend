#!/usr/bin/env bash
# pre-retro-test.sh — Test pre-retro.sh with mock gh to verify GH_TOKEN validation.
#
# Uses a mock gh command to capture calls without hitting GitHub.
# Run from the repo root: bash internal/scaffold/fullsend-repo/scripts/pre-retro-test.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PRE_SCRIPT="${SCRIPT_DIR}/pre-retro.sh"
FAILURES=0

# Create a temp directory for mock state.
TMPDIR="$(mktemp -d)"
trap 'rm -rf "${TMPDIR}"' EXIT

# --- Helpers ---

# build_mock creates a mock gh binary that returns preconfigured responses.
# Arguments:
#   $1 — exit code for "gh api" calls (0 = success, 1 = failure).
#   $2 — (optional) stderr/stdout output for failed "gh api" calls.
build_mock() {
  local api_exit="$1"
  local api_error="${2:-}"
  local mock_bin="${TMPDIR}/bin"
  local gh_log="${TMPDIR}/gh-calls.log"

  rm -rf "${mock_bin}"
  mkdir -p "${mock_bin}"
  : > "${gh_log}"

  # Write the api error to a file so the mock can read it.
  printf '%s' "${api_error}" > "${TMPDIR}/api-error.txt"

  cat > "${mock_bin}/gh" <<MOCKEOF
#!/usr/bin/env bash
CALL_LOG="${gh_log}"

echo "gh \$*" >> "\${CALL_LOG}"

# Route by subcommand
if [[ "\$1" == "api" ]]; then
  ERROR_FILE="${TMPDIR}/api-error.txt"
  if [[ ${api_exit} -ne 0 ]] && [[ -s "\${ERROR_FILE}" ]]; then
    cat "\${ERROR_FILE}" >&2
  fi
  exit ${api_exit}
fi
MOCKEOF

  chmod +x "${mock_bin}/gh"

  echo "${mock_bin}"
}

run_test_stdout() {
  local test_name="$1"
  local api_exit="$2"
  local api_error="$3"
  local expected_stdout="$4"
  local expect_exit="$5"
  local extra_env="${6:-}"

  local mock_bin
  mock_bin="$(build_mock "${api_exit}" "${api_error}")"

  local env_cmd=(
    env
    PATH="${mock_bin}:${PATH}"
    ORIGINATING_URL="https://github.com/test-org/test-repo/pull/42"
    GH_TOKEN="fake-token"
  )

  # Add extra env vars if provided.
  if [[ -n "${extra_env}" ]]; then
    while IFS= read -r kv; do
      [[ -n "${kv}" ]] && env_cmd+=("${kv}")
    done <<< "${extra_env}"
  fi

  local exit_code=0
  "${env_cmd[@]}" bash "${PRE_SCRIPT}" > "${TMPDIR}/stdout.log" 2>&1 || exit_code=$?

  if [[ ${exit_code} -ne ${expect_exit} ]]; then
    echo "FAIL: ${test_name} — expected exit ${expect_exit}, got ${exit_code}"
    cat "${TMPDIR}/stdout.log"
    FAILURES=$((FAILURES + 1))
    return
  fi

  if ! grep -qF "${expected_stdout}" "${TMPDIR}/stdout.log" 2>/dev/null; then
    echo "FAIL: ${test_name} — expected stdout '${expected_stdout}' not found"
    echo "Actual stdout:"
    cat "${TMPDIR}/stdout.log"
    FAILURES=$((FAILURES + 1))
    return
  fi

  echo "PASS: ${test_name}"
}

# Check stdout does NOT contain a specific string.
run_test_stdout_absent() {
  local test_name="$1"
  local api_exit="$2"
  local api_error="$3"
  local absent_stdout="$4"
  local expect_exit="$5"
  local extra_env="${6:-}"

  local mock_bin
  mock_bin="$(build_mock "${api_exit}" "${api_error}")"

  local env_cmd=(
    env
    PATH="${mock_bin}:${PATH}"
    ORIGINATING_URL="https://github.com/test-org/test-repo/pull/42"
    GH_TOKEN="fake-token"
  )

  if [[ -n "${extra_env}" ]]; then
    while IFS= read -r kv; do
      [[ -n "${kv}" ]] && env_cmd+=("${kv}")
    done <<< "${extra_env}"
  fi

  local exit_code=0
  "${env_cmd[@]}" bash "${PRE_SCRIPT}" > "${TMPDIR}/stdout.log" 2>&1 || exit_code=$?

  if [[ ${exit_code} -ne ${expect_exit} ]]; then
    echo "FAIL: ${test_name} — expected exit ${expect_exit}, got ${exit_code}"
    cat "${TMPDIR}/stdout.log"
    FAILURES=$((FAILURES + 1))
    return
  fi

  if grep -qF "${absent_stdout}" "${TMPDIR}/stdout.log" 2>/dev/null; then
    echo "FAIL: ${test_name} — stdout should NOT contain '${absent_stdout}'"
    echo "Actual stdout:"
    cat "${TMPDIR}/stdout.log"
    FAILURES=$((FAILURES + 1))
    return
  fi

  echo "PASS: ${test_name}"
}

# --- Test cases ---

# Empty GH_TOKEN → exit 1 with error.
run_test_stdout "empty-token-exits" \
  0 "" \
  "GH_TOKEN is required" \
  1 \
  "GH_TOKEN="

# Unset GH_TOKEN → exit 1 with error (unset via empty assignment to override default).
run_test_stdout "unset-token-exits" \
  0 "" \
  "GH_TOKEN is required" \
  1 \
  "GH_TOKEN="

# Invalid GH_TOKEN (gh api fails) → exit 1 with error.
run_test_stdout "invalid-token-exits" \
  1 "401 Unauthorized" \
  "GH_TOKEN is invalid" \
  1

# Invalid GH_TOKEN → error output is logged (not inside ::error:: annotation).
run_test_stdout "invalid-token-shows-output" \
  1 "401 Unauthorized" \
  "401 Unauthorized" \
  1

# Invalid GH_TOKEN → auth_output is NOT inside a workflow command annotation.
run_test_stdout_absent "invalid-token-no-annotated-output" \
  1 "401 Unauthorized" \
  "::error::gh api /rate_limit output:" \
  1

# Valid GH_TOKEN → success message.
run_test_stdout "valid-token-succeeds" \
  0 "" \
  "GH_TOKEN validated successfully" \
  0

# Valid GH_TOKEN → token is masked.
run_test_stdout "valid-token-masks-token" \
  0 "" \
  "::add-mask::fake-token" \
  0

# Valid GH_TOKEN with RETRO_COMMENT → on-demand trigger message.
run_test_stdout "on-demand-trigger" \
  0 "" \
  "Retro triggered on-demand with comment" \
  0 \
  "RETRO_COMMENT=/retro"

# Valid GH_TOKEN without RETRO_COMMENT → automatic trigger message.
run_test_stdout "automatic-trigger" \
  0 "" \
  "Retro triggered automatically" \
  0

# Validation complete message appears.
run_test_stdout "validation-complete" \
  0 "" \
  "Pre-retro validation complete" \
  0

# --- Summary ---

echo ""
if [[ ${FAILURES} -gt 0 ]]; then
  echo "${FAILURES} test(s) failed"
  exit 1
fi
echo "All tests passed"
