"""Shared fixtures for GH-41 e2e tests."""
import json
import logging
import os
import subprocess

import pytest

logger = logging.getLogger(__name__)


@pytest.fixture
def gh_api():
    """Helper to call GitHub API via gh CLI.

    Returns a callable that invokes ``gh api`` with the given HTTP method,
    endpoint, and optional JSON body.  The raw ``subprocess.CompletedProcess``
    is returned so callers can inspect both stdout and returncode.
    """

    def _call(method, endpoint, data=None):
        cmd = ["gh", "api", "-X", method, endpoint]
        if data:
            input_data = json.dumps(data)
            cmd.extend(["--input", "-"])
            result = subprocess.run(
                cmd,
                input=input_data,
                capture_output=True,
                text=True,
                timeout=30,
            )
        else:
            result = subprocess.run(cmd, capture_output=True, text=True, timeout=30)
        logger.debug(
            "gh api %s %s -> rc=%d stdout=%s stderr=%s",
            method,
            endpoint,
            result.returncode,
            result.stdout[:200],
            result.stderr[:200],
        )
        return result

    return _call


@pytest.fixture
def github_repo():
    """Return the test repository in owner/repo format."""
    return os.environ.get("TEST_REPO", "")


@pytest.fixture
def pr_number():
    """Return the test PR number."""
    return int(os.environ.get("TEST_PR_NUMBER", "0"))


@pytest.fixture
def pr_files(gh_api, github_repo, pr_number):
    """Return the list of files changed in the test PR.

    This is used by tests that need to pick a real file path present in
    the PR diff when constructing review comment payloads.
    """
    result = gh_api(
        "GET",
        f"/repos/{github_repo}/pulls/{pr_number}/files",
    )
    assert result.returncode == 0, f"Failed to fetch PR files: {result.stderr}"
    files = json.loads(result.stdout)
    return [f["filename"] for f in files]
