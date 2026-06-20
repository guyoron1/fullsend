#!/usr/bin/env bash
# post-fix-test.sh — Test the push retry logic from post-fix.sh.
#
# Extracts and tests the push-with-fallback logic in isolation using shell
# functions. This avoids needing a full git repo or GitHub API access.
#
# Run from the repo root:
#   bash internal/scaffold/fullsend-repo/scripts/post-fix-test.sh

set -euo pipefail

FAILURES=0

# ---------------------------------------------------------------------------
# Test helper — reimplements the push retry logic from post-fix.sh section 6.
# Given a push exit code and output, returns the action the script would take.
# ---------------------------------------------------------------------------
decide_push_action() {
  local push_rc="$1"
  local push_output="$2"

  if [ "${push_rc}" -eq 0 ]; then
    echo "success"
    return 0
  fi

  if echo "${push_output}" | grep -qi "non-fast-forward\|rejected\|fetch first"; then
    echo "retry:force-with-lease"
    return 0
  fi

  echo "fail:unexpected-error"
  return 0
}

run_push_test() {
  local test_name="$1"
  local push_rc="$2"
  local push_output="$3"
  local expected_prefix="$4"

  local actual
  actual="$(decide_push_action "${push_rc}" "${push_output}")"

  if [[ "${actual}" != ${expected_prefix}* ]]; then
    echo "FAIL: ${test_name}"
    echo "  push_rc:         '${push_rc}'"
    echo "  push_output:     '${push_output}'"
    echo "  expected prefix: '${expected_prefix}'"
    echo "  actual:          '${actual}'"
    FAILURES=$((FAILURES + 1))
    return
  fi

  echo "PASS: ${test_name}"
}

# --- Push retry test cases ---

# Successful push (normal fix iteration — no rebase)
run_push_test "push-success-normal-fix" \
  "0" "Everything up-to-date" "success"

# Non-fast-forward error after rebase — should retry with --force-with-lease
run_push_test "push-non-fast-forward-rebase" \
  "1" "error: failed to push some refs: non-fast-forward" "retry:force-with-lease"

# Rejected error after rebase — should retry with --force-with-lease
run_push_test "push-rejected-rebase" \
  "1" "! [rejected] agent/42 -> agent/42 (fetch first)" "retry:force-with-lease"

# Unknown error — should fail without retrying
run_push_test "push-unexpected-error" \
  "1" "fatal: repository not found" "fail:unexpected-error"

# Authentication error — should fail without retrying
run_push_test "push-auth-error" \
  "1" "fatal: Authentication failed" "fail:unexpected-error"

# --- Summary ---

echo ""
if [ ${FAILURES} -gt 0 ]; then
  echo "${FAILURES} test(s) failed"
  exit 1
fi
echo "All tests passed"
