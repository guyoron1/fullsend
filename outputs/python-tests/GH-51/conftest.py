"""Shared fixtures for GH-51 CLAUDE.md pointer injection tests."""

import os
import pytest


@pytest.fixture
def repo_dir(tmp_path):
    """Create a temporary directory simulating a repo root."""
    return tmp_path


@pytest.fixture
def repo_with_agents_md(repo_dir):
    """Create a repo directory with AGENTS.md present."""
    agents_file = repo_dir / "AGENTS.md"
    agents_file.write_text("# Agent rules\n")
    return repo_dir


@pytest.fixture
def repo_with_claude_md(repo_with_agents_md):
    """Create a repo directory with both AGENTS.md and CLAUDE.md present."""
    claude_file = repo_with_agents_md / "CLAUDE.md"
    claude_file.write_text("# My custom Claude instructions\n")
    return repo_with_agents_md


CLAUDE_MD_VARIANTS = ["CLAUDE.md", "claude.md", "Claude.md", ".claude.md"]


def has_claude_md(directory):
    """Check if any CLAUDE.md casing variant exists in the directory."""
    for variant in CLAUDE_MD_VARIANTS:
        if (directory / variant).exists():
            return True
    return False


def do_inject_claude_md_pointer(repo_dir, exec_func=None):
    """Simulate CLAUDE.md pointer injection.

    Writes a CLAUDE.md file with pointer content referencing AGENTS.md,
    then calls exec_func to add it to git exclude.
    """
    claude_path = repo_dir / "CLAUDE.md"
    pointer_content = (
        "# CLAUDE.md — Auto-generated pointer\n"
        "# This file was injected by fullsend to bridge AGENTS.md for Claude Code.\n"
        "# See AGENTS.md for the canonical agent rules.\n\n"
        "Read and follow the instructions in AGENTS.md\n"
    )
    claude_path.write_text(pointer_content)

    if exec_func is not None:
        exec_func(f'echo "CLAUDE.md" >> {repo_dir}/.git/info/exclude')

    return None
