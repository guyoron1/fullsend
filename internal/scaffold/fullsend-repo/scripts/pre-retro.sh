#!/usr/bin/env bash
# pre-retro.sh — Validate inputs for the retro agent.
#
# Runs on the host via the harness pre_script mechanism. Validates the
# originating URL (PR or issue) and logs the trigger context.
#
# Required env vars:
#   ORIGINATING_URL — HTML URL of the PR or issue that triggered retro
#   GH_TOKEN        — GitHub token for API access
#
# Optional env vars:
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
# Validate GH_TOKEN before starting the sandbox.
# post-retro.sh hard-requires GH_TOKEN (${GH_TOKEN:?}) for filing issues and
# posting comments.  Fail fast here to avoid wasting sandbox + inference costs.
# Uses /rate_limit instead of `gh auth status` because the latter calls
# GET /user, which returns 403 for GitHub App installation tokens.
# ---------------------------------------------------------------------------
if [[ -z "${GH_TOKEN:-}" ]]; then
  echo "::error::GH_TOKEN is required — retro agent needs GitHub API access for post-retro"
  exit 1
fi

if ! auth_output=$(gh api /rate_limit --silent 2>&1); then
  echo "::error::GH_TOKEN is invalid — retro agent requires GitHub API access"
  echo "::error::gh api /rate_limit output: ${auth_output}"
  exit 1
fi
echo "GH_TOKEN validated successfully."

if [[ -n "${RETRO_COMMENT:-}" ]]; then
  echo "Retro triggered on-demand with comment."
else
  echo "Retro triggered automatically (PR close)."
fi

echo "Pre-retro validation complete."
