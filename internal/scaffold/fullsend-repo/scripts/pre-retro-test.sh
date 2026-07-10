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

# build_mock creates a mock gh binary that returns a preconfigured exit code
# for "gh auth status" calls.
# Arguments:
#   $1 — exit code for "gh auth status" (0 = valid token, 1 = invalid token).
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
    env -i
    PATH="${mock_bin}:/usr/bin:/bin"
    HOME="${TMPDIR}"
    ORIGINATING_URL="https://github.com/test-org/test-repo/pull/42"
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

# --- Test cases ---

# Valid token → auth status exits 0 → script succeeds.
run_test_stdout "valid-token-succeeds" \
  0 \
  "GH_TOKEN validated successfully." \
  0 \
  "GH_TOKEN=fake-token"

# Invalid token → auth status exits 1 → script fails with error annotation.
run_test_stdout "invalid-token-fails" \
  1 \
  "::error::GH_TOKEN is invalid" \
  1 \
  "GH_TOKEN=bad-token"

# Absent token → script fails with error annotation (no sandbox wasted).
run_test_stdout "absent-token-fails" \
  0 \
  "::error::GH_TOKEN is not set" \
  1

# Valid token with retro comment → script succeeds with on-demand message.
run_test_stdout "retro-comment-on-demand" \
  0 \
  "Retro triggered on-demand with comment." \
  0 \
  "$(printf '%s\n%s' 'GH_TOKEN=fake-token' 'RETRO_COMMENT=look at this')"

# Valid token without retro comment → script succeeds with automatic message.
run_test_stdout "retro-automatic-trigger" \
  0 \
  "Retro triggered automatically (PR close)." \
  0 \
  "GH_TOKEN=fake-token"

# --- Summary ---

echo ""
if [[ ${FAILURES} -gt 0 ]]; then
  echo "${FAILURES} test(s) failed"
  exit 1
fi
echo "All tests passed"
