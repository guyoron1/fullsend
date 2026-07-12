#!/usr/bin/env bash
# post-review-test.sh — Test the outcome-label logic in post-review.sh.
#
# Extracts and tests the label-application logic in isolation using shell
# functions. This avoids needing a live GitHub API or fullsend CLI.
#
# Run from the repo root:
#   bash internal/scaffold/fullsend-repo/scripts/post-review-test.sh

set -euo pipefail

FAILURES=0

# ---------------------------------------------------------------------------
# Test helper — reimplements the outcome-label logic from post-review.sh
# so we can test it without network access.
#
# Arguments:
#   $1 — ACTION (the original action from agent-result.json)
#   $2 — DOWNGRADED ("true" or "false")
#
# Prints the label that would be applied, or "none" if no label.
# ---------------------------------------------------------------------------
determine_outcome_label() {
  local action="$1"
  local downgraded="$2"

  if [ "${action}" = "approve" ] && [ "${downgraded}" = "false" ]; then
    echo "ready-for-merge"
  elif [ "${action}" = "approve" ] && [ "${downgraded}" = "true" ]; then
    echo "requires-manual-review"
  elif [ "${action}" = "comment" ]; then
    echo "requires-manual-review"
  elif [ "${action}" = "request_changes" ]; then
    echo "none"
  elif [ "${action}" = "reject" ]; then
    echo "rejected"
  else
    echo "none"
  fi
}

run_test() {
  local test_name="$1"
  local action="$2"
  local downgraded="$3"
  local expected="$4"

  local actual
  actual="$(determine_outcome_label "${action}" "${downgraded}")"

  if [ "${actual}" != "${expected}" ]; then
    echo "FAIL: ${test_name}"
    echo "  action:     '${action}'"
    echo "  downgraded: '${downgraded}'"
    echo "  expected:   '${expected}'"
    echo "  actual:     '${actual}'"
    FAILURES=$((FAILURES + 1))
    return
  fi

  echo "PASS: ${test_name}"
}

# --- Test cases ---

# Approve without protected-path downgrade → ready-for-merge
run_test "approve-no-downgrade" \
  "approve" "false" "ready-for-merge"

# Approve with protected-path downgrade → requires-manual-review
run_test "approve-with-downgrade" \
  "approve" "true" "requires-manual-review"

# Comment (split/conflicting review) → requires-manual-review
run_test "comment-split-review" \
  "comment" "false" "requires-manual-review"

# request_changes → no outcome label
run_test "request-changes-no-label" \
  "request_changes" "false" "none"

# reject → rejected
run_test "reject-label" \
  "reject" "false" "rejected"

# Defensive: comment + downgraded=true can't occur in production (DOWNGRADED is
# only set inside the approve branch), but verify the label logic handles it.
run_test "comment-with-downgrade-flag" \
  "comment" "true" "requires-manual-review"

# Edge cases: ensure unknown/empty actions produce no label
run_test "empty-action-no-label" \
  "" "false" "none"

run_test "failure-action-no-label" \
  "failure" "false" "none"

run_test "unknown-action-no-label" \
  "banana" "false" "none"

# ---------------------------------------------------------------------------
# Governance file-matching logic — reimplements the governance-document check
# from post-review.sh so we can test it without network access.
#
# Arguments:
#   $1 — PR_FILES (newline-separated list of changed file paths)
#
# Prints "true" if ALL files are governance documents, "false" otherwise.
# ---------------------------------------------------------------------------
is_all_governance() {
  local pr_files="$1"

  local GOVERNANCE_DOC_PATTERNS=(
    "MAINTAINERS.md"
    "GOVERNANCE.md"
    "CODE_OF_CONDUCT.md"
    "SECURITY.md"
  )

  local ALL_GOVERNANCE=true
  [ -z "${pr_files}" ] && ALL_GOVERNANCE=false
  while IFS= read -r file; do
    [ -z "${file}" ] && continue
    local basename_val
    basename_val=$(basename "${file}")
    local is_gov=false
    for pattern in "${GOVERNANCE_DOC_PATTERNS[@]}"; do
      if [ "${basename_val}" = "${pattern}" ]; then
        is_gov=true
        break
      fi
    done
    if [ "${is_gov}" = "false" ]; then
      ALL_GOVERNANCE=false
      break
    fi
  done <<< "${pr_files}"

  echo "${ALL_GOVERNANCE}"
}

run_gov_test() {
  local test_name="$1"
  local pr_files="$2"
  local expected="$3"

  local actual
  actual="$(is_all_governance "${pr_files}")"

  if [ "${actual}" != "${expected}" ]; then
    echo "FAIL: ${test_name}"
    echo "  files:    '${pr_files}'"
    echo "  expected: '${expected}'"
    echo "  actual:   '${actual}'"
    FAILURES=$((FAILURES + 1))
    return
  fi

  echo "PASS: ${test_name}"
}

# --- Governance file-matching test cases ---

# Single governance file → all governance
run_gov_test "single-governance-file" \
  "GOVERNANCE.md" "true"

# All four governance files → all governance
run_gov_test "all-four-governance-files" \
  "MAINTAINERS.md
GOVERNANCE.md
CODE_OF_CONDUCT.md
SECURITY.md" "true"

# Governance file in subdirectory → matches via basename
run_gov_test "governance-in-subdirectory" \
  "docs/GOVERNANCE.md" "true"

run_gov_test "security-in-subdirectory" \
  "subdir/SECURITY.md" "true"

# Non-governance file → not all governance
run_gov_test "non-governance-file" \
  "README.md" "false"

# Mixed governance and non-governance → not all governance
run_gov_test "mixed-governance-and-code" \
  "GOVERNANCE.md
src/main.go" "false"

# Substring match should NOT trigger (MY_GOVERNANCE.md ≠ GOVERNANCE.md)
run_gov_test "substring-no-match" \
  "MY_GOVERNANCE.md" "false"

# Case-sensitive: lowercase should not match
run_gov_test "case-sensitive-no-match" \
  "governance.md" "false"

# Multiple governance files in subdirectories → all governance
run_gov_test "multiple-governance-in-subdirs" \
  "docs/GOVERNANCE.md
.github/SECURITY.md" "true"

# Empty input → not all governance (defense-in-depth; upstream guard
# exits on empty PR_FILES before this check runs in production)
run_gov_test "empty-input-not-governance" \
  "" "false"

# --- Summary ---

echo ""
if [ "${FAILURES}" -gt 0 ]; then
  echo "${FAILURES} test(s) failed"
  exit 1
fi
echo "All tests passed"
