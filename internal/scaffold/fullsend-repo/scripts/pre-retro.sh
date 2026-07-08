#!/usr/bin/env bash
# pre-retro.sh — Validate inputs for the retro agent.
#
# Runs on the host via the harness pre_script mechanism. Validates the
# originating URL (PR or issue) and logs the trigger context.
#
# Required env vars:
#   ORIGINATING_URL — HTML URL of the PR or issue that triggered retro
#
# Optional env vars:
#   GH_TOKEN        — GitHub token for API access (validated if set)
#   RETRO_COMMENT   — The /retro comment text (empty for automatic triggers)

set -euo pipefail

: "${ORIGINATING_URL:?ORIGINATING_URL is required}"

# Accept both issue and PR URLs.
if [[ ! "${ORIGINATING_URL}" =~ ^https://github\.com/[a-zA-Z0-9._-]+/[a-zA-Z0-9._-]+/(issues|pull)/[0-9]+$ ]]; then
  echo "ERROR: ORIGINATING_URL does not match expected pattern"
  exit 1
fi

echo "::notice::Retro target: ${ORIGINATING_URL}"

# ---------------------------------------------------------------------------
# Validate GH_TOKEN before starting the sandbox
# ---------------------------------------------------------------------------
if [[ -n "${GH_TOKEN:-}" ]]; then
  if ! gh auth status 2>/dev/null; then
    echo "::error::GH_TOKEN is invalid — retro agent requires GitHub API access"
    exit 1
  fi
  echo "GH_TOKEN validated successfully."
else
  echo "::warning::GH_TOKEN is not set — retro agent will have limited functionality"
fi

if [[ -n "${RETRO_COMMENT:-}" ]]; then
  echo "Retro triggered on-demand with comment."
else
  echo "Retro triggered automatically (PR close)."
fi

echo "Pre-retro validation complete."
