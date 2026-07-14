#!/usr/bin/env bash
# pre-retro-test.sh — Test pre-retro.sh GH_TOKEN validation.
#
# Uses a mock gh command to simulate auth success/failure.
# Run from the repo root: bash internal/scaffold/fullsend-repo/scripts/pre-retro-test.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PRE_SCRIPT="${SCRIPT_DIR}/pre-retro.sh"
FAILURES=0

TMPDIR="$(mktemp -d)"
trap 'rm -rf "${TMPDIR}"' EXIT

# --- Helpers ---

# build_mock creates a mock gh binary.
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

# --- Test cases ---

# GH_TOKEN valid → pass through with success message.
run_test_stdout "valid-token-passes" \
  0 \
  "GH_TOKEN validated successfully." \
  0 \
  "GH_TOKEN=fake-valid-token"

# GH_TOKEN invalid → exit 1 with error.
run_test_stdout "invalid-token-fails" \
  1 \
  "GH_TOKEN is invalid" \
  1 \
  "GH_TOKEN=bad-token"

# GH_TOKEN unset → warn, don't fail.
run_test_stdout "no-token-warns" \
  0 \
  "GH_TOKEN is not set" \
  0 \
  "GH_TOKEN="

# GH_TOKEN valid → still shows retro target after validation.
run_test_stdout "valid-token-continues-to-completion" \
  0 \
  "Pre-retro validation complete." \
  0 \
  "GH_TOKEN=fake-valid-token"

# --- Summary ---

echo ""
if [[ ${FAILURES} -gt 0 ]]; then
  echo "${FAILURES} test(s) failed"
  exit 1
fi
echo "All tests passed"
