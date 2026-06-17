"""
Shared fixtures for GH-26 layered defense E2E tests.

Provides repository path resolution, mock gh CLI infrastructure,
and common test utilities.
"""

import os
import textwrap
from pathlib import Path

import pytest


def pytest_configure(config):
    """Register custom markers."""
    config.addinivalue_line("markers", "e2e: end-to-end integration tests")
    config.addinivalue_line("markers", "gh26: tests for GH-26 defense-in-depth")


@pytest.fixture(scope="session")
def repo_root() -> Path:
    """Repository root path, resolved once per session."""
    root = os.environ.get("REPO_ROOT")
    if root:
        return Path(root)

    current = Path.cwd()
    for parent in [current] + list(current.parents):
        if (parent / ".git").exists():
            return parent

    pytest.skip("Repository root not found — set REPO_ROOT env var")
    return Path(".")
