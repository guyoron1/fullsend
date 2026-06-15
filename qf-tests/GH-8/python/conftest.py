"""
Shared fixtures for GH-8 shim drift detection tests.

STP Reference: outputs/stp/GH-8/GH-8_test_plan.md
"""

import base64
import os
import pathlib
import stat
import tempfile

import pytest


@pytest.fixture(scope="session")
def repo_root():
    """Root directory of the repository under test.

    Walks up from this file to find the repo root (contains hack/ directory).
    """
    current = pathlib.Path(__file__).resolve().parent
    for parent in [current] + list(current.parents):
        if (parent / "hack").is_dir():
            return parent
    pytest.skip("Repository root with hack/ directory not found")


@pytest.fixture(scope="session")
def reconcile_script_path(repo_root):
    """Path to the reconcile-repos.sh script under test."""
    script = repo_root / "hack" / "reconcile-repos.sh"
    if not script.exists():
        pytest.skip(f"reconcile-repos.sh not found at {script}")
    return script


@pytest.fixture(scope="session")
def reconcile_test_script_path(repo_root):
    """Path to the reconcile-repos-test.sh regression test script."""
    script = repo_root / "hack" / "reconcile-repos-test.sh"
    if not script.exists():
        pytest.skip(f"reconcile-repos-test.sh not found at {script}")
    return script


@pytest.fixture(scope="function")
def mock_command_dir():
    """Temporary directory for mock shell commands.

    Creates a temporary directory that can be prepended to PATH
    to override real commands with mock implementations.

    Yields the path to the temporary directory, then cleans up.
    """
    with tempfile.TemporaryDirectory(prefix="shim-test-") as tmpdir:
        yield pathlib.Path(tmpdir)


@pytest.fixture(scope="function")
def env_with_mock_path(mock_command_dir):
    """Environment dict with mock_command_dir prepended to PATH."""
    env = os.environ.copy()
    env["PATH"] = f"{mock_command_dir}:{env.get('PATH', '')}"
    return env


@pytest.fixture(scope="session")
def managed_shim_content():
    """The canonical managed shim workflow content."""
    return (
        "name: fullsend-shim\n"
        "on:\n"
        "  workflow_dispatch:\n"
        "jobs:\n"
        "  shim:\n"
        "    uses: fullsend-ai/fullsend/.github/workflows/shim.yml@main\n"
    )


@pytest.fixture(scope="session")
def outdated_shim_content():
    """Outdated shim workflow content referencing an old version."""
    return (
        "name: fullsend-shim\n"
        "on:\n"
        "  workflow_dispatch:\n"
        "jobs:\n"
        "  shim:\n"
        "    uses: fullsend-ai/fullsend/.github/workflows/shim.yml@v0.9\n"
    )


def encode_base64(content):
    """Encode a string to base64, returning a string."""
    return base64.b64encode(content.encode("utf-8")).decode("utf-8")


def write_mock_script(directory, name, script_body):
    """Write an executable mock shell script to a directory.

    Args:
        directory: Path to directory where the script will be created.
        name: Name of the script file (e.g., "gh").
        script_body: Shell script content (without shebang).

    Returns:
        Path to the created script.
    """
    script_path = pathlib.Path(directory) / name
    script_path.write_text(f"#!/usr/bin/env bash\n{script_body}\n")
    script_path.chmod(script_path.stat().st_mode | stat.S_IEXEC)
    return script_path
