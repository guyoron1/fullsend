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
# Severity guardrail test helper — checks whether structured findings or body
# text contain HIGH/CRITICAL severity.
# Arguments:
#   $1 — JSON string representing the result file content
# Returns "true" or "false".
# ---------------------------------------------------------------------------
check_severity_downgrade() {
  local result_json="$1"

  # Check structured findings
  local high_count
  high_count=$(echo "${result_json}" | jq '[(.findings // [])[] | select(.severity | ascii_downcase | test("^(high|critical)$"))] | length' 2>/dev/null || echo "0")
  if [ "${high_count}" -gt 0 ]; then
    echo "true"
    return
  fi

  # Fallback: check body for severity section headers with content.
  # Only used when NO structured findings exist (regardless of severity).
  local total_count
  total_count=$(echo "${result_json}" | jq '[(.findings // [])[]] | length' 2>/dev/null || echo "0")
  if [ "${total_count}" -eq 0 ]; then
    local body
    body=$(echo "${result_json}" | jq -r '.body // ""')
    local section_content
    section_content=$(printf '%s\n' "${body}" | awk '
      BEGIN { IGNORECASE=1 }
      /^###+ +(High|Critical) *$/ { in_section=1; next }
      /^#/ { in_section=0 }
      in_section && /[^[:space:]]/ { print 1; exit }
    ')
    if [ -n "${section_content}" ]; then
      echo "true"
      return
    fi
  fi

  echo "false"
}

run_severity_test() {
  local test_name="$1"
  local result_json="$2"
  local expected="$3"

  local actual
  actual="$(check_severity_downgrade "${result_json}")"

  if [ "${actual}" != "${expected}" ]; then
    echo "FAIL: ${test_name}"
    echo "  expected: '${expected}'"
    echo "  actual:   '${actual}'"
    FAILURES=$((FAILURES + 1))
    return
  fi

  echo "PASS: ${test_name}"
}

# --- Severity guardrail test cases ---

# Structured HIGH finding triggers downgrade
run_severity_test "severity-structured-high" \
  '{"body":"Review","action":"approve","findings":[{"severity":"high","description":"bug"}]}' \
  "true"

# Structured CRITICAL finding triggers downgrade
run_severity_test "severity-structured-critical" \
  '{"body":"Review","action":"approve","findings":[{"severity":"critical","description":"vuln"}]}' \
  "true"

# Only LOW/MEDIUM structured findings — no downgrade
run_severity_test "severity-structured-low-medium" \
  '{"body":"Review","action":"approve","findings":[{"severity":"low","description":"nit"},{"severity":"medium","description":"issue"}]}' \
  "false"

# No structured findings, HIGH in body — triggers downgrade
run_severity_test "severity-body-high-section" \
  '{"body":"## Review\n\n#### High\n\n- Missing error handling\n\n#### Low\n\n- Style","action":"approve"}' \
  "true"

# No structured findings, CRITICAL in body — triggers downgrade
run_severity_test "severity-body-critical-section" \
  '{"body":"## Review\n\n### Critical\n\n- SQL injection","action":"approve"}' \
  "true"

# Empty HIGH section header (no content) — no downgrade
run_severity_test "severity-body-empty-high-section" \
  '{"body":"## Review\n\n#### High\n\n#### Medium\n\n- Real finding","action":"approve"}' \
  "false"

# No findings, no severity headers — no downgrade
run_severity_test "severity-no-findings-no-headers" \
  '{"body":"## Review\n\nLooks good!","action":"approve"}' \
  "false"

# Structured findings present but only low; body has HIGH header — no downgrade
# (structured findings take precedence; body fallback only when no structured)
run_severity_test "severity-structured-overrides-body" \
  '{"body":"## Review\n\n#### High\n\n- Bug\n","action":"approve","findings":[{"severity":"low","description":"nit"}]}' \
  "false"

# Severity-downgraded approve → requires-manual-review (same as protected-path downgrade)
run_test "approve-severity-downgrade" \
  "approve" "true" "requires-manual-review"

# --- Summary ---

echo ""
if [ "${FAILURES}" -gt 0 ]; then
  echo "${FAILURES} test(s) failed"
  exit 1
fi
echo "All tests passed"
