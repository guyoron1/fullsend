#!/usr/bin/env bash
# post-fix-test.sh — Test the companion script fallback resolution logic
# from post-fix.sh.
#
# Verifies that process-fix-result.py is found via the BASH_SOURCE path
# first, falling back to ${FULLSEND_DIR}/scripts/ when not co-located
# (e.g., when post-fix.sh runs from the content-addressed cache).
#
# Run from the repo root:
#   bash internal/scaffold/fullsend-repo/scripts/post-fix-test.sh

set -euo pipefail

FAILURES=0

# ---------------------------------------------------------------------------
# Test helper — reimplements the companion resolution logic from post-fix.sh
# so we can test it without running the full post-script.
# ---------------------------------------------------------------------------
resolve_companion() {
  local script_dir="$1"
  local fullsend_dir="$2"

  local process_script="${script_dir}/process-fix-result.py"
  if [ ! -f "${process_script}" ]; then
    script_dir="${fullsend_dir:-.}/scripts"
    process_script="${script_dir}/process-fix-result.py"
  fi

  if [ -f "${process_script}" ]; then
    echo "resolved:${process_script}"
  else
    echo "not-found"
  fi
}

run_test() {
  local test_name="$1"
  local expected_prefix="$2"
  local actual="$3"

  if [[ "${actual}" != ${expected_prefix}* ]]; then
    echo "FAIL: ${test_name}"
    echo "  expected prefix: '${expected_prefix}'"
    echo "  actual:          '${actual}'"
    FAILURES=$((FAILURES + 1))
    return
  fi

  echo "PASS: ${test_name}"
}

# ---------------------------------------------------------------------------
# Setup: create temp directories simulating cache and workspace layouts.
# ---------------------------------------------------------------------------
TMPROOT="$(mktemp -d)"
trap 'rm -rf "${TMPROOT}"' EXIT

CACHE_DIR="${TMPROOT}/cache/sha256/abc123/content"
WORKSPACE_DIR="${TMPROOT}/workspace"

mkdir -p "${CACHE_DIR}"
mkdir -p "${WORKSPACE_DIR}/scripts"

# The companion file exists only in the workspace scripts directory.
touch "${WORKSPACE_DIR}/scripts/process-fix-result.py"

# --- Test cases ---

# 1. Cache-resolved path (no companion) with FULLSEND_DIR fallback
result="$(resolve_companion "${CACHE_DIR}" "${WORKSPACE_DIR}")"
run_test "cache-path-falls-back-to-workspace" "resolved:${WORKSPACE_DIR}/scripts/process-fix-result.py" "${result}"

# 2. Direct path when companion is co-located (workspace-resolved)
result="$(resolve_companion "${WORKSPACE_DIR}/scripts" "${WORKSPACE_DIR}")"
run_test "workspace-path-resolves-directly" "resolved:${WORKSPACE_DIR}/scripts/process-fix-result.py" "${result}"

# 3. Neither path has companion — reports not-found
EMPTY_DIR="${TMPROOT}/empty"
mkdir -p "${EMPTY_DIR}/scripts"
result="$(resolve_companion "${CACHE_DIR}" "${EMPTY_DIR}")"
run_test "neither-path-reports-not-found" "not-found" "${result}"

# 4. FULLSEND_DIR unset (defaults to ".") — uses ./scripts/ as fallback
result="$(resolve_companion "${CACHE_DIR}" "")"
run_test "empty-fullsend-dir-defaults-to-dot" "not-found" "${result}"

# --- Summary ---

echo ""
if [ ${FAILURES} -gt 0 ]; then
  echo "${FAILURES} test(s) failed"
  exit 1
fi
echo "All tests passed"
