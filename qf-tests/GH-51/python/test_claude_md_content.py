"""Tests for CLAUDE.md pointer content after injection.

STP Reference: outputs/stp/GH-51/GH-51_test_plan.md
Jira: GH-51
"""

import pytest
from conftest import do_inject_claude_md_pointer


class TestClaudeMDPointerContent:
    """[GH-51] CLAUDE.md pointer content validation."""

    def test_ts_gh_51_002_should_contain_reference_to_agents_md(
        self, repo_with_agents_md
    ):
        """[test_id:TS-GH-51-002] Verify injected CLAUDE.md content references AGENTS.md.

        After successful injection, the CLAUDE.md file content must contain
        a reference to AGENTS.md so that Claude Code discovers agent rules.
        """
        do_inject_claude_md_pointer(repo_with_agents_md)

        claude_path = repo_with_agents_md / "CLAUDE.md"
        assert claude_path.exists(), "CLAUDE.md should exist after injection"

        content = claude_path.read_text()
        assert "AGENTS.md" in content, (
            f"CLAUDE.md content should reference AGENTS.md, got: {content}"
        )
