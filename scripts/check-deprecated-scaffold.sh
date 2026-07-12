#!/usr/bin/env bash
# check-deprecated-scaffold.sh — Fail CI when a PR touches deprecated scaffold paths.
#
# The files under internal/scaffold/fullsend-repo/ have moved to the
# fullsend-ai/agents repo. This check redirects contributors before they
# spend time on a PR that will be closed.
#
# When running outside GitHub Actions PR context, this script is a no-op.
# In CI on a pull_request event, it fetches the PR's changed files via the
# GitHub API and fails if any match the deprecated prefix.
#
# Environment (set automatically by GitHub Actions):
#   GITHUB_ACTIONS      — "true" when running in CI
#   GITHUB_EVENT_NAME   — the event type (e.g., "pull_request")
#   GITHUB_REPOSITORY   — owner/repo
#   GITHUB_EVENT_PATH   — path to the event payload JSON
#
# Optional:
#   GH_TOKEN / GITHUB_TOKEN — for gh api calls (set automatically in CI)

set -euo pipefail

# ponytail: no-op outside CI PR context, skip early
if [[ "${GITHUB_ACTIONS:-}" != "true" ]] || [[ "${GITHUB_EVENT_NAME:-}" != "pull_request" ]]; then
  echo "check-deprecated-scaffold: not a CI pull_request — skipping."
  exit 0
fi

if [[ ! -f "${GITHUB_EVENT_PATH:-}" ]]; then
  echo "check-deprecated-scaffold: GITHUB_EVENT_PATH not set or missing — skipping."
  exit 0
fi

PR_NUMBER="$(jq -r '.pull_request.number // .number // empty' "${GITHUB_EVENT_PATH}" 2>/dev/null || true)"
if [[ -z "${PR_NUMBER}" ]]; then
  echo "check-deprecated-scaffold: could not extract PR number — skipping."
  exit 0
fi

CHANGED=$(gh api \
  "repos/${GITHUB_REPOSITORY}/pulls/${PR_NUMBER}/files" \
  --paginate --jq '.[].filename' 2>/dev/null || true)

DEPRECATED=$(echo "${CHANGED}" | grep '^internal/scaffold/fullsend-repo/' || true)

if [[ -z "${DEPRECATED}" ]]; then
  echo "check-deprecated-scaffold: no deprecated scaffold paths touched — OK."
  exit 0
fi

echo "::error::Files under internal/scaffold/fullsend-repo/ have moved to fullsend-ai/agents."
echo ""
echo "The following changed files are at a deprecated location:"
echo "${DEPRECATED}"
echo ""
echo "Please open your PR in https://github.com/fullsend-ai/agents instead."
exit 1
