"""
Shared fixtures for GH-4: Use AI to Help Formalise Intent After Rapid Local Prototyping.

STP Reference:
    outputs/stp/GH-4/GH-4_test_plan.md
"""
import os
import shutil
import subprocess
import tempfile
from pathlib import Path
from typing import Generator

import pytest


@pytest.fixture(scope="session")
def fullsend_binary() -> str:
    """Resolve the fullsend binary path.

    Returns:
        Absolute path to the fullsend binary.

    Raises:
        pytest.skip: If fullsend binary is not found in PATH.
    """
    binary = shutil.which("fullsend")
    if binary is None:
        pytest.skip("fullsend binary not found in PATH")
    return binary


@pytest.fixture(scope="session")
def llm_endpoint() -> str:
    """Resolve the LLM inference endpoint from environment.

    Returns:
        URL of the LLM inference endpoint.

    Raises:
        pytest.skip: If LLM_ENDPOINT environment variable is not set.
    """
    endpoint = os.environ.get("LLM_ENDPOINT", "")
    if not endpoint:
        pytest.skip("LLM_ENDPOINT environment variable not set")
    return endpoint


@pytest.fixture(scope="function")
def prototype_dir_scope_function() -> Generator[Path, None, None]:
    """Create a temporary directory for prototype code.

    Yields:
        Path to the temporary prototype directory.
    """
    dirpath = Path(tempfile.mkdtemp(prefix="test-prototype-"))
    yield dirpath
    shutil.rmtree(str(dirpath), ignore_errors=True)


@pytest.fixture(scope="function")
def spec_output_dir_scope_function() -> Generator[Path, None, None]:
    """Create a temporary directory for spec output.

    Yields:
        Path to the temporary spec output directory.
    """
    dirpath = Path(tempfile.mkdtemp(prefix="test-spec-output-"))
    yield dirpath
    shutil.rmtree(str(dirpath), ignore_errors=True)


@pytest.fixture(scope="function")
def prototype_with_testable_behavior(
    prototype_dir_scope_function: Path,
) -> Path:
    """Create a prototype directory containing Go code with testable behavior.

    Args:
        prototype_dir_scope_function: Temporary directory for prototype files.

    Returns:
        Path to the prototype directory with testable Go code.
    """
    main_go = prototype_dir_scope_function / "main.go"
    main_go.write_text(
        'package main\n\n'
        '// Add returns the sum of two integers.\n'
        'func Add(a, b int) int { return a + b }\n\n'
        '// Subtract returns the difference of two integers.\n'
        'func Subtract(a, b int) int { return a - b }\n'
    )
    return prototype_dir_scope_function


@pytest.fixture(scope="function")
def prototype_without_testable_behavior(
    prototype_dir_scope_function: Path,
) -> Path:
    """Create a prototype directory containing Go code with no testable behavior.

    Args:
        prototype_dir_scope_function: Temporary directory for prototype files.

    Returns:
        Path to the prototype directory with no exported functions.
    """
    main_go = prototype_dir_scope_function / "main.go"
    main_go.write_text(
        'package main\n\n'
        '// This file has no testable behavior\n'
        '// Only comments and unexported declarations\n'
    )
    return prototype_dir_scope_function


@pytest.fixture(scope="function")
def prototype_with_ambiguous_behavior(
    prototype_dir_scope_function: Path,
) -> Path:
    """Create a prototype directory with contradictory code behavior.

    The comment says uppercase but the implementation does lowercase,
    creating ambiguity that the workflow should detect.

    Args:
        prototype_dir_scope_function: Temporary directory for prototype files.

    Returns:
        Path to the prototype directory with contradictory code.
    """
    contradictory_go = prototype_dir_scope_function / "contradictory.go"
    contradictory_go.write_text(
        'package main\n\n'
        'import "strings"\n\n'
        '// Process returns uppercase for valid input\n'
        'func Process(s string) string { return strings.ToLower(s) }\n'
        '// The comment says uppercase, implementation does lowercase\n'
    )
    return prototype_dir_scope_function


def run_fullsend_command(
    binary: str,
    subcommand: str,
    args: list[str],
    timeout_seconds: int = 300,
) -> subprocess.CompletedProcess:
    """Execute a fullsend CLI command.

    Args:
        binary: Path to the fullsend binary.
        subcommand: The fullsend subcommand to run.
        args: Additional arguments for the subcommand.
        timeout_seconds: Maximum time to wait for command completion.

    Returns:
        CompletedProcess with stdout, stderr, and returncode.
    """
    cmd = [binary, subcommand, *args]
    return subprocess.run(
        cmd,
        capture_output=True,
        text=True,
        timeout=timeout_seconds,
    )
