#!/usr/bin/env bash
# post-retro-test.sh — Test post-retro.sh with fixture JSON inputs.
#
# Uses a mock gh command to capture calls without hitting GitHub.
# Run from the repo root: bash internal/scaffold/fullsend-repo/scripts/post-retro-test.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
POST_SCRIPT="${SCRIPT_DIR}/post-retro.sh"
FAILURES=0

# Create a temp directory for test fixtures and mock state.
TMPDIR="$(mktemp -d)"
trap 'rm -rf "${TMPDIR}"' EXIT

# Mock gh: record all calls to a log file and serve fixtures for API GETs.
export GH_LOG="${TMPDIR}/gh-calls.log"
export GH_FIXTURES="${TMPDIR}/fixtures"
MOCK_BIN="${TMPDIR}/bin"
mkdir -p "${MOCK_BIN}" "${GH_FIXTURES}"
cat > "${MOCK_BIN}/gh" <<'MOCKEOF'
#!/usr/bin/env bash
# Consume stdin if --input is present to avoid SIGPIPE on the writer.
if [[ "$*" == *"--input"* ]]; then
  cat > /dev/null
fi

echo "gh $*" >> "${GH_LOG}"

# gh api GET (no --input, no -X): serve fixture if available.
if [[ "$1" == "api" ]] && [[ "$*" != *"--input"* ]] && [[ "$*" != *"-X "* ]]; then
  # Normalize API path to fixture filename: repos/org/repo/issues/123 -> repos_org_repo_issues_123
  FIXTURE_PATH=$(echo "$2" | tr '/' '_')
  if [[ -f "${GH_FIXTURES}/${FIXTURE_PATH}" ]]; then
    cat "${GH_FIXTURES}/${FIXTURE_PATH}"
    exit 0
  fi
  # No fixture: simulate API error.
  echo "Not Found" >&2
  exit 1
fi

# gh issue create: return a fake issue URL.
if [[ "$1" == "issue" ]] && [[ "$2" == "create" ]]; then
  # Extract --repo value for the URL.
  REPO=""
  PREV=""
  for arg in "$@"; do
    if [[ "${PREV}" == "--repo" ]]; then
      REPO="${arg}"
    fi
    PREV="${arg}"
  done
  echo "https://github.com/${REPO:-test-org/test-repo}/issues/999"
  exit 0
fi

exit 0
MOCKEOF
chmod +x "${MOCK_BIN}/gh"

export PATH="${MOCK_BIN}:${PATH}"
export ORIGINATING_URL="https://github.com/test-org/test-repo/pull/98"
export GH_TOKEN="fake-token"

# Helper: set up a fixture for gh api GET responses.
# Usage: set_issue_fixture "org/repo" 123 '{"state":"open","body":"..."}'
set_issue_fixture() {
  local repo="$1"
  local number="$2"
  local json="$3"
  local fixture_name="repos_${repo//\//_}_issues_${number}"
  echo "${json}" > "${GH_FIXTURES}/${fixture_name}"
}

run_test() {
  local test_name="$1"
  local json_content="$2"
  local expected_pattern="$3"
  local expect_failure="${4:-false}"

  # Create iteration output structure.
  local run_dir="${TMPDIR}/run-${test_name}"
  mkdir -p "${run_dir}/iteration-1/output"
  echo "${json_content}" > "${run_dir}/iteration-1/output/agent-result.json"

  # Clear gh call log.
  : > "${GH_LOG}"

  # Run the post-script.
  local exit_code=0
  (cd "${run_dir}" && bash "${POST_SCRIPT}") > "${TMPDIR}/stdout.log" 2>&1 || exit_code=$?

  if [[ "${expect_failure}" == "true" ]]; then
    if [[ ${exit_code} -eq 0 ]]; then
      echo "FAIL: ${test_name} — expected failure but got success"
      FAILURES=$((FAILURES + 1))
      return
    fi
    echo "PASS: ${test_name} (expected failure, got exit code ${exit_code})"
    return
  fi

  if [[ ${exit_code} -ne 0 ]]; then
    echo "FAIL: ${test_name} — exit code ${exit_code}"
    cat "${TMPDIR}/stdout.log"
    FAILURES=$((FAILURES + 1))
    return
  fi

  if [[ -n "${expected_pattern}" ]] && ! grep -qF "${expected_pattern}" "${GH_LOG}"; then
    echo "FAIL: ${test_name} — expected gh call pattern '${expected_pattern}' not found"
    echo "Actual calls:"
    cat "${GH_LOG}"
    FAILURES=$((FAILURES + 1))
    return
  fi

  echo "PASS: ${test_name}"
}

run_test_stdout() {
  local test_name="$1"
  local json_content="$2"
  local expected_stdout="$3"

  local run_dir="${TMPDIR}/run-${test_name}"
  mkdir -p "${run_dir}/iteration-1/output"
  echo "${json_content}" > "${run_dir}/iteration-1/output/agent-result.json"
  : > "${GH_LOG}"

  local exit_code=0
  (cd "${run_dir}" && bash "${POST_SCRIPT}") > "${TMPDIR}/stdout.log" 2>&1 || exit_code=$?

  if [[ ${exit_code} -ne 0 ]]; then
    echo "FAIL: ${test_name} — exit code ${exit_code}"
    cat "${TMPDIR}/stdout.log"
    FAILURES=$((FAILURES + 1))
    return
  fi

  if ! grep -qF "${expected_stdout}" "${TMPDIR}/stdout.log"; then
    echo "FAIL: ${test_name} — expected stdout pattern '${expected_stdout}' not found"
    echo "Actual stdout:"
    cat "${TMPDIR}/stdout.log"
    FAILURES=$((FAILURES + 1))
    return
  fi

  echo "PASS: ${test_name}"
}

run_test_no_pattern() {
  local test_name="$1"
  local json_content="$2"
  local forbidden_pattern="$3"

  local run_dir="${TMPDIR}/run-${test_name}"
  mkdir -p "${run_dir}/iteration-1/output"
  echo "${json_content}" > "${run_dir}/iteration-1/output/agent-result.json"
  : > "${GH_LOG}"

  local exit_code=0
  (cd "${run_dir}" && bash "${POST_SCRIPT}") > "${TMPDIR}/stdout.log" 2>&1 || exit_code=$?

  if [[ ${exit_code} -ne 0 ]]; then
    echo "FAIL: ${test_name} — exit code ${exit_code}"
    cat "${TMPDIR}/stdout.log"
    FAILURES=$((FAILURES + 1))
    return
  fi

  if grep -qF "${forbidden_pattern}" "${GH_LOG}"; then
    echo "FAIL: ${test_name} — forbidden pattern '${forbidden_pattern}' was found"
    echo "Actual calls:"
    cat "${GH_LOG}"
    FAILURES=$((FAILURES + 1))
    return
  fi

  echo "PASS: ${test_name}"
}

# --- Existing behavior: proposals still work ---

run_test "proposals-filed" \
  '{"summary":"Summary text.","proposals":[{"target_repo":"org/repo","title":"Fix thing","what_happened":"Timeline.","what_could_go_better":"Improvement.","proposed_change":"Do X.","validation_criteria":"Check Y."}]}' \
  "gh issue create --repo org/repo --title Fix thing"

run_test "summary-posted" \
  '{"summary":"Summary text.","proposals":[]}' \
  "gh api repos/test-org/test-repo/issues/98/comments --input -"

run_test "no-proposals-no-retractions" \
  '{"summary":"Nothing to report.","proposals":[]}' \
  "gh api repos/test-org/test-repo/issues/98/comments --input -"

run_test "missing-summary-fails" \
  '{"proposals":[]}' \
  "" \
  "true"

run_test "invalid-json-fails" \
  "not json" \
  "" \
  "true"

# --- Retraction: happy path ---

# Set up a fixture: open issue filed by retro bot.
set_issue_fixture "org/repo" 295 \
  '{"state":"open","body":"## What happened\n\nTimeline.\n\n---\n_Generated by retro agent from https://github.com/org/repo/pull/1_"}'

run_test "retraction-comments-on-issue" \
  '{"summary":"Corrected analysis.","proposals":[],"retractions":[{"issue_number":295,"target_repo":"org/repo","reason":"Original analysis was based on wrong diff"}]}' \
  "gh api repos/org/repo/issues/295/comments --input -"

run_test "retraction-closes-issue" \
  '{"summary":"Corrected analysis.","proposals":[],"retractions":[{"issue_number":295,"target_repo":"org/repo","reason":"Original analysis was based on wrong diff"}]}' \
  "gh issue close 295 --repo org/repo --reason not planned"

run_test_stdout "retraction-logged" \
  '{"summary":"Corrected analysis.","proposals":[],"retractions":[{"issue_number":295,"target_repo":"org/repo","reason":"Original analysis was based on wrong diff"}]}' \
  "Retracted: org/repo#295"

# --- Retraction: already closed ---

set_issue_fixture "org/repo" 100 \
  '{"state":"closed","body":"_Generated by retro agent from https://github.com/org/repo/pull/1_"}'

run_test_stdout "retraction-already-closed-skipped" \
  '{"summary":"Summary.","proposals":[],"retractions":[{"issue_number":100,"target_repo":"org/repo","reason":"wrong analysis"}]}' \
  "retraction[0]: org/repo#100 is already closed, skipping"

run_test_no_pattern "retraction-already-closed-no-close-call" \
  '{"summary":"Summary.","proposals":[],"retractions":[{"issue_number":100,"target_repo":"org/repo","reason":"wrong analysis"}]}' \
  "gh issue close 100"

# --- Retraction: not filed by retro bot ---

set_issue_fixture "org/repo" 200 \
  '{"state":"open","body":"This is a human-filed issue with no retro signature."}'

run_test_stdout "retraction-not-retro-bot-skipped" \
  '{"summary":"Summary.","proposals":[],"retractions":[{"issue_number":200,"target_repo":"org/repo","reason":"wrong analysis"}]}' \
  "::warning::retraction[0]: org/repo#200 was not filed by the retro agent, skipping"

run_test_no_pattern "retraction-not-retro-bot-no-close-call" \
  '{"summary":"Summary.","proposals":[],"retractions":[{"issue_number":200,"target_repo":"org/repo","reason":"wrong analysis"}]}' \
  "gh issue close 200"

# --- Retraction: inaccessible issue (no fixture = mock returns error) ---

run_test_stdout "retraction-inaccessible-warns" \
  '{"summary":"Summary.","proposals":[],"retractions":[{"issue_number":999,"target_repo":"other-org/other-repo","reason":"wrong analysis"}]}' \
  "::warning::retraction[0]: could not fetch other-org/other-repo#999"

# --- Retraction: summary includes retraction links ---

run_test "retraction-summary-includes-retractions-section" \
  '{"summary":"Corrected analysis.","proposals":[],"retractions":[{"issue_number":295,"target_repo":"org/repo","reason":"Original analysis was based on wrong diff"}]}' \
  "gh api repos/test-org/test-repo/issues/98/comments --input -"

# --- Retraction with proposals: both processed ---

run_test "retraction-and-proposals-both-processed" \
  '{"summary":"Mixed run.","proposals":[{"target_repo":"org/repo","title":"New improvement","what_happened":"T.","what_could_go_better":"I.","proposed_change":"C.","validation_criteria":"V."}],"retractions":[{"issue_number":295,"target_repo":"org/repo","reason":"wrong analysis"}]}' \
  "gh issue create --repo org/repo --title New improvement"

run_test "retraction-and-proposals-retraction-also-processed" \
  '{"summary":"Mixed run.","proposals":[{"target_repo":"org/repo","title":"New improvement","what_happened":"T.","what_could_go_better":"I.","proposed_change":"C.","validation_criteria":"V."}],"retractions":[{"issue_number":295,"target_repo":"org/repo","reason":"wrong analysis"}]}' \
  "gh issue close 295 --repo org/repo --reason not planned"

# --- No retractions field: existing behavior unchanged ---

run_test "no-retractions-field-works" \
  '{"summary":"Normal run.","proposals":[{"target_repo":"org/repo","title":"Fix thing","what_happened":"T.","what_could_go_better":"I.","proposed_change":"C.","validation_criteria":"V."}]}' \
  "gh issue create --repo org/repo --title Fix thing"

# --- Empty retractions array ---

run_test "empty-retractions-array-works" \
  '{"summary":"No retractions needed.","proposals":[],"retractions":[]}' \
  "gh api repos/test-org/test-repo/issues/98/comments --input -"

# --- Summary ---

echo ""
if [[ ${FAILURES} -gt 0 ]]; then
  echo "${FAILURES} test(s) failed"
  exit 1
fi
echo "All tests passed"
