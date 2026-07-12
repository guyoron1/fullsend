#!/usr/bin/env bash
# pre-retro-test.sh — Test pre-retro.sh GH_TOKEN validation and URL checks.
#
# Uses a mock gh command to simulate auth status responses without hitting GitHub.
# Run from the repo root: bash internal/scaffold/fullsend-repo/scripts/pre-retro-test.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PRE_SCRIPT="${SCRIPT_DIR}/pre-retro.sh"
FAILURES=0

# Create a temp directory for mock state.
TMPDIR="$(mktemp -d)"
trap 'rm -rf "${TMPDIR}"' EXIT

# --- Helpers ---

# build_mock creates a mock gh binary that returns a preconfigured exit code
# for "gh auth status" calls.
# Arguments:
#   $1 — exit code for "gh auth status" (0 = valid, 1 = invalid)
build_mock() {
  local auth_exit="$1"
  local mock_bin="${TMPDIR}/bin"

  rm -rf "${mock_bin}"
  mkdir -p "${mock_bin}"

  cat > "${mock_bin}/gh" <<MOCKEOF
#!/usr/bin/env bash
if [[ "\$1" == "auth" && "\$2" == "status" ]]; then
  exit ${auth_exit}
fi
exit 0
MOCKEOF

  chmod +x "${mock_bin}/gh"
  echo "${mock_bin}"
}

run_test() {
  local test_name="$1"
  local expected_stdout="$2"
  local expect_exit="$3"
  local gh_token="${4:-fake-token}"
  local auth_exit="${5:-0}"

  local mock_bin
  mock_bin="$(build_mock "${auth_exit}")"

  local env_cmd=(
    env
    PATH="${mock_bin}:${PATH}"
    ORIGINATING_URL="https://github.com/test-org/test-repo/pull/42"
  )

  # Set or unset GH_TOKEN based on the value.
  if [[ "${gh_token}" == "__UNSET__" ]]; then
    env_cmd+=("GH_TOKEN=")
  else
    env_cmd+=("GH_TOKEN=${gh_token}")
  fi

  local exit_code=0
  "${env_cmd[@]}" bash "${PRE_SCRIPT}" > "${TMPDIR}/stdout.log" 2>&1 || exit_code=$?

  # Check exit code.
  if [[ ${exit_code} -ne ${expect_exit} ]]; then
    echo "FAIL: ${test_name} — expected exit ${expect_exit}, got ${exit_code}"
    cat "${TMPDIR}/stdout.log"
    FAILURES=$((FAILURES + 1))
    return
  fi

  # Check expected pattern in stdout.
  if [[ -n "${expected_stdout}" ]]; then
    if ! grep -qF "${expected_stdout}" "${TMPDIR}/stdout.log" 2>/dev/null; then
      echo "FAIL: ${test_name} — expected stdout '${expected_stdout}' not found"
      echo "Actual stdout:"
      cat "${TMPDIR}/stdout.log"
      FAILURES=$((FAILURES + 1))
      return
    fi
  fi

  echo "PASS: ${test_name}"
}

# --- Test cases ---

# Valid GH_TOKEN → gh auth status exits 0 → script proceeds.
run_test "valid-token-proceeds" \
  "GH_TOKEN validation passed." \
  0 \
  "fake-valid-token" \
  0

# Valid token → script completes fully.
run_test "valid-token-completes" \
  "Pre-retro validation complete." \
  0 \
  "fake-valid-token" \
  0

# Invalid GH_TOKEN → gh auth status exits 1 → script exits 1 with error.
run_test "invalid-token-exits" \
  "GH_TOKEN is set but invalid" \
  1 \
  "fake-bad-token" \
  1

# Unset GH_TOKEN → warning emitted, script proceeds.
run_test "unset-token-warns" \
  "GH_TOKEN is not set" \
  0 \
  "__UNSET__" \
  0

# Unset GH_TOKEN → script still completes.
run_test "unset-token-completes" \
  "Pre-retro validation complete." \
  0 \
  "__UNSET__" \
  0

# Invalid ORIGINATING_URL → script exits 1 before token check.
run_test_url_validation() {
  local test_name="$1"
  local url="$2"
  local expected_stdout="$3"
  local expect_exit="$4"

  local mock_bin
  mock_bin="$(build_mock 0)"

  local exit_code=0
  env PATH="${mock_bin}:${PATH}" \
    ORIGINATING_URL="${url}" \
    GH_TOKEN="fake-token" \
    bash "${PRE_SCRIPT}" > "${TMPDIR}/stdout.log" 2>&1 || exit_code=$?

  if [[ ${exit_code} -ne ${expect_exit} ]]; then
    echo "FAIL: ${test_name} — expected exit ${expect_exit}, got ${exit_code}"
    cat "${TMPDIR}/stdout.log"
    FAILURES=$((FAILURES + 1))
    return
  fi

  if [[ -n "${expected_stdout}" ]]; then
    if ! grep -qF "${expected_stdout}" "${TMPDIR}/stdout.log" 2>/dev/null; then
      echo "FAIL: ${test_name} — expected stdout '${expected_stdout}' not found"
      echo "Actual stdout:"
      cat "${TMPDIR}/stdout.log"
      FAILURES=$((FAILURES + 1))
      return
    fi
  fi

  echo "PASS: ${test_name}"
}

# Valid PR URL → accepted.
run_test_url_validation "valid-pr-url" \
  "https://github.com/test-org/test-repo/pull/42" \
  "Pre-retro validation complete." \
  0

# Valid issue URL → accepted.
run_test_url_validation "valid-issue-url" \
  "https://github.com/test-org/test-repo/issues/42" \
  "Pre-retro validation complete." \
  0

# Invalid URL → rejected.
run_test_url_validation "invalid-url-rejected" \
  "https://example.com/not-github" \
  "::error::ORIGINATING_URL does not match expected pattern" \
  1

# Retro comment triggers on-demand message.
run_test_retro_comment() {
  local test_name="$1"
  local retro_comment="$2"
  local expected_stdout="$3"

  local mock_bin
  mock_bin="$(build_mock 0)"

  local env_cmd=(
    env
    PATH="${mock_bin}:${PATH}"
    ORIGINATING_URL="https://github.com/test-org/test-repo/pull/42"
    GH_TOKEN="fake-token"
  )

  if [[ -n "${retro_comment}" ]]; then
    env_cmd+=("RETRO_COMMENT=${retro_comment}")
  fi

  local exit_code=0
  "${env_cmd[@]}" bash "${PRE_SCRIPT}" > "${TMPDIR}/stdout.log" 2>&1 || exit_code=$?

  if [[ ${exit_code} -ne 0 ]]; then
    echo "FAIL: ${test_name} — expected exit 0, got ${exit_code}"
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

# With RETRO_COMMENT → on-demand message.
run_test_retro_comment "retro-comment-on-demand" \
  "/retro please" \
  "Retro triggered on-demand with comment."

# Without RETRO_COMMENT → automatic message.
run_test_retro_comment "retro-automatic" \
  "" \
  "Retro triggered automatically (PR close)."

# --- Summary ---

echo ""
if [[ ${FAILURES} -gt 0 ]]; then
  echo "${FAILURES} test(s) failed"
  exit 1
fi
echo "All tests passed"
