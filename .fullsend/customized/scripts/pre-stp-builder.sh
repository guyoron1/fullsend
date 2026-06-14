#!/usr/bin/env bash
# pre-stp-builder.sh — Validate inputs before the STP builder agent runs.
#
# Required env vars (always):
#   JIRA_TICKET      — Ticket key (e.g., MYPROJ-123 or GH-42)
#   GH_TOKEN         — GitHub token for PR data
#
# Required env vars (Jira mode only, skipped when ISSUE_SOURCE=github):
#   JIRA_BASE_URL    — Jira instance URL
#   JIRA_API_TOKEN   — Jira API token
#   JIRA_USER_EMAIL  — Jira user email
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/lib/common.sh"

echo "::notice::QualityFlow stp-builder: ${JIRA_TICKET:-unset} (source: ${ISSUE_SOURCE:-jira})"

errors=0
require_env JIRA_TICKET GH_TOKEN || errors=$((errors + $?))

if [[ "${ISSUE_SOURCE:-jira}" != "github" ]]; then
  require_env JIRA_BASE_URL JIRA_API_TOKEN JIRA_USER_EMAIL || errors=$((errors + $?))
fi

require_jira_format || errors=$((errors + $?))
require_config || errors=$((errors + $?))

if [[ "${errors}" -gt 0 ]]; then
  echo "::error::Pre-script failed with ${errors} error(s). Aborting."
  exit 1
fi

echo "Input validation passed: ${JIRA_TICKET}"
