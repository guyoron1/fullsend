#!/usr/bin/env bash
# pre-retro-test.sh — Test pre-retro.sh with mock gh to verify token validation.
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

# build_mock creates a mock gh binary with configurable auth status behavior.
# Arguments:
#   $1 — exit code for "gh auth status" (0 = valid token, 1 = invalid token).
build_mock() {
  local auth_exit="$1"
  local mock_bin="${TMPDIR}/bin"
  local gh_log="${TMPDIR}/gh-calls.log"

  rm -rf "${mock_bin}"
  mkdir -p "${mock_bin}"
  : > "${gh_log}"

  cat > "${mock_bin}/gh" <<MOCKEOF
#!/usr/bin/env bash
CALL_LOG="${gh_log}"

echo "gh \$*" >> "\${CALL_LOG}"

# Route by subcommand
if [[ "\$1" == "auth" && "\$2" == "status" ]]; then
  exit ${auth_exit}
fi
MOCKEOF

  chmod +x "${mock_bin}/gh"

  echo "${mock_bin}"
}

run_test_stdout() {
  local test_name="$1"
  local auth_exit="$2"
  local expected_stdout="$3"
  local expect_exit="$4"
  local extra_env="${5:-}"

  local mock_bin
  mock_bin="$(build_mock "${auth_exit}")"

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

  if ! grep -qF "${expected_stdout}" "${TMPDIR}/stdout.log" 2>/dev/null; then
    echo "FAIL: ${test_name} — expected stdout '${expected_stdout}' not found"
    echo "Actual stdout:"
    cat "${TMPDIR}/stdout.log"
    FAILURES=$((FAILURES + 1))
    return
  fi

  echo "PASS: ${test_name}"
}

# Check that gh auth status was called.
run_test_gh_call() {
  local test_name="$1"
  local auth_exit="$2"
  local expected_pattern="$3"
  local expect_exit="$4"
  local extra_env="${5:-}"

  local mock_bin
  mock_bin="$(build_mock "${auth_exit}")"
  local gh_log="${TMPDIR}/gh-calls.log"

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

  if [[ -n "${expected_pattern}" ]]; then
    if ! grep -qF "${expected_pattern}" "${gh_log}" 2>/dev/null; then
      echo "FAIL: ${test_name} — expected gh call pattern '${expected_pattern}' not found"
      echo "Actual calls:"
      cat "${gh_log}" 2>/dev/null || echo "(no calls)"
      FAILURES=$((FAILURES + 1))
      return
    fi
  fi

  echo "PASS: ${test_name}"
}

# Check stdout does NOT contain a specific string.
run_test_stdout_absent() {
  local test_name="$1"
  local auth_exit="$2"
  local absent_stdout="$3"
  local expect_exit="$4"
  local extra_env="${5:-}"

  local mock_bin
  mock_bin="$(build_mock "${auth_exit}")"

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

# Valid token → gh auth status exits 0 → script succeeds with validation message.
run_test_stdout "valid-token-passes" \
  0 \
  "GH_TOKEN validated successfully." \
  0

run_test_gh_call "valid-token-calls-auth-status" \
  0 \
  "gh auth status" \
  0

# Valid token → script completes with pre-retro validation message.
run_test_stdout "valid-token-completes" \
  0 \
  "Pre-retro validation complete." \
  0

# Invalid token → gh auth status exits 1 → script exits 1 with error.
run_test_stdout "invalid-token-fails" \
  1 \
  "GH_TOKEN is invalid" \
  1

run_test_gh_call "invalid-token-calls-auth-status" \
  1 \
  "gh auth status" \
  1

# Invalid token → script does NOT reach completion message.
run_test_stdout_absent "invalid-token-no-completion" \
  1 \
  "Pre-retro validation complete." \
  1

# No token → prints warning, continues (exit 0).
run_test_stdout "no-token-warns" \
  0 \
  "GH_TOKEN is not set" \
  0 \
  "GH_TOKEN="

# No token → script still completes.
run_test_stdout "no-token-completes" \
  0 \
  "Pre-retro validation complete." \
  0 \
  "GH_TOKEN="

# No token → does NOT call gh auth status.
run_test_stdout_absent "no-token-no-auth-call" \
  0 \
  "GH_TOKEN validated successfully." \
  0 \
  "GH_TOKEN="

# Valid token + RETRO_COMMENT → on-demand trigger message shown.
run_test_stdout "valid-token-with-comment" \
  0 \
  "Retro triggered on-demand with comment." \
  0 \
  "RETRO_COMMENT=test retro"

# Valid token + no RETRO_COMMENT → automatic trigger message shown.
run_test_stdout "valid-token-automatic-trigger" \
  0 \
  "Retro triggered automatically (PR close)." \
  0

# Issue URL (not just PR) still works with valid token.
run_test_stdout "issue-url-valid-token" \
  0 \
  "GH_TOKEN validated successfully." \
  0 \
  "ORIGINATING_URL=https://github.com/test-org/test-repo/issues/42"

# --- Summary ---

echo ""
if [[ ${FAILURES} -gt 0 ]]; then
  echo "${FAILURES} test(s) failed"
  exit 1
fi
echo "All tests passed"
