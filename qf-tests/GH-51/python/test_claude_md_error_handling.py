"""Tests for CLAUDE.md injection error handling.

STP Reference: outputs/stp/GH-51/GH-51_test_plan.md
Jira: GH-51
"""

import os
import pytest
from conftest import do_inject_claude_md_pointer


class TestClaudeMDWriteFailure:
    """[GH-51] CLAUDE.md injection write failure handling."""

    def test_ts_gh_51_014_should_raise_on_write_failure(self, repo_dir):
        """[test_id:TS-GH-51-014] [NEGATIVE] Verify error on CLAUDE.md write failure.

        When the target directory is read-only, the write should fail and
        raise an exception. The caller logs a warning but does not abort.
        """
        os.chmod(str(repo_dir), 0o555)
        try:
            with pytest.raises(PermissionError):
                do_inject_claude_md_pointer(repo_dir)
        finally:
            os.chmod(str(repo_dir), 0o755)


class TestClaudeMDGitExcludeFailure:
    """[GH-51] CLAUDE.md git exclude error handling."""

    def test_ts_gh_51_016_should_preserve_file_despite_exclude_failure(
        self, repo_with_agents_md
    ):
        """[test_id:TS-GH-51-016] [NEGATIVE] Verify CLAUDE.md exists after exclude failure.

        When CLAUDE.md is successfully written but the git exclude command
        fails, the file should still exist — exclude is best-effort.
        """
        def failing_exec(cmd):
            raise RuntimeError("git exclude command failed")

        # The write succeeds; exec failure is handled gracefully
        try:
            do_inject_claude_md_pointer(repo_with_agents_md, exec_func=failing_exec)
        except RuntimeError:
            pass  # exec failure is expected

        claude_path = repo_with_agents_md / "CLAUDE.md"
        assert claude_path.exists(), "CLAUDE.md should exist despite exclude failure"

    def test_ts_gh_51_017_should_preserve_content_despite_exclude_failure(
        self, repo_with_agents_md
    ):
        """[test_id:TS-GH-51-017] [NEGATIVE] Verify CLAUDE.md content correct after exclude failure.

        Even when git exclude fails after a successful write, the CLAUDE.md
        content should be valid and reference AGENTS.md.
        """
        def failing_exec(cmd):
            raise RuntimeError("git exclude command failed")

        try:
            do_inject_claude_md_pointer(repo_with_agents_md, exec_func=failing_exec)
        except RuntimeError:
            pass

        claude_path = repo_with_agents_md / "CLAUDE.md"
        assert claude_path.exists()

        content = claude_path.read_text()
        assert "AGENTS.md" in content, (
            f"CLAUDE.md should reference AGENTS.md, got: {content}"
        )
