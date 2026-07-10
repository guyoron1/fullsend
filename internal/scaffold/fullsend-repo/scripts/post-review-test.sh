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
# only set inside the approve branch, by either the protected-path or
# governance-document downgrade path), but verify the label logic handles it.
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
# Governance document detection — reimplements the loop from post-review.sh
# to verify all-governance vs mixed-file classification.
# ---------------------------------------------------------------------------
GOVERNANCE_DOC_PATTERNS=(
  "MAINTAINERS.md"
  "GOVERNANCE.md"
  "CODE_OF_CONDUCT.md"
  "SECURITY.md"
)

is_all_governance() {
  local pr_files="$1"
  local all_gov=true
  while IFS= read -r file; do
    [ -z "${file}" ] && continue
    local base
    base=$(basename "${file}")
    local is_gov=false
    for pattern in "${GOVERNANCE_DOC_PATTERNS[@]}"; do
      if [ "${base}" = "${pattern}" ]; then
        is_gov=true
        break
      fi
    done
    if [ "${is_gov}" = "false" ]; then
      all_gov=false
      break
    fi
  done <<< "${pr_files}"
  echo "${all_gov}"
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

# --- Governance detection test cases ---

# All governance files → downgrade
run_gov_test "all-governance-single" \
  "GOVERNANCE.md" "true"

run_gov_test "all-governance-multiple" \
  "GOVERNANCE.md
MAINTAINERS.md
CODE_OF_CONDUCT.md" "true"

# Governance files in subdirectories → still governance
run_gov_test "governance-in-subdir" \
  "docs/GOVERNANCE.md
community/CODE_OF_CONDUCT.md" "true"

# Mixed: governance + non-governance → no downgrade
run_gov_test "mixed-files-no-downgrade" \
  "GOVERNANCE.md
src/main.go" "false"

# Non-governance only → no downgrade
run_gov_test "non-governance-only" \
  "src/main.go
README.md" "false"

# Substring mismatch: MY_GOVERNANCE.md is not a governance doc
run_gov_test "substring-no-match" \
  "MY_GOVERNANCE.md" "false"

# All four governance patterns
run_gov_test "all-four-patterns" \
  "MAINTAINERS.md
GOVERNANCE.md
CODE_OF_CONDUCT.md
SECURITY.md" "true"

# --- Summary ---

echo ""
if [ "${FAILURES}" -gt 0 ]; then
  echo "${FAILURES} test(s) failed"
  exit 1
fi
echo "All tests passed"
