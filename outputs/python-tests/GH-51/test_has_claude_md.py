"""Tests for hasClaudeMD casing detection.

STP Reference: outputs/stp/GH-51/GH-51_test_plan.md
Jira: GH-51
"""

import pytest
from conftest import has_claude_md, CLAUDE_MD_VARIANTS


class TestHasClaudeMDCasingDetection:
    """[GH-51] hasClaudeMD casing detection tests."""

    def test_ts_gh_51_004_should_detect_uppercase_claude_md(self, repo_dir):
        """[test_id:TS-GH-51-004] Verify detection of CLAUDE.md uppercase."""
        (repo_dir / "CLAUDE.md").write_text("# Claude instructions\n")
        assert has_claude_md(repo_dir) is True

    def test_ts_gh_51_005_should_detect_lowercase_claude_md(self, repo_dir):
        """[test_id:TS-GH-51-005] Verify detection of claude.md lowercase."""
        (repo_dir / "claude.md").write_text("# claude instructions\n")
        assert has_claude_md(repo_dir) is True

    def test_ts_gh_51_006_should_detect_dot_prefixed_claude_md(self, repo_dir):
        """[test_id:TS-GH-51-006] Verify detection of .claude.md dot-prefixed."""
        (repo_dir / ".claude.md").write_text("# hidden claude instructions\n")
        assert has_claude_md(repo_dir) is True

    def test_ts_gh_51_007_should_return_false_when_no_variants_exist(self, repo_dir):
        """[test_id:TS-GH-51-007] Verify false when no CLAUDE.md variants exist."""
        (repo_dir / "README.md").write_text("# Just a readme\n")
        assert has_claude_md(repo_dir) is False

    @pytest.mark.parametrize(
        "variant",
        CLAUDE_MD_VARIANTS,
        ids=["uppercase", "lowercase", "titlecase", "dot-prefixed"],
    )
    def test_ts_gh_51_012_should_detect_all_casing_variants(self, repo_dir, variant):
        """[test_id:TS-GH-51-012] Verify all supported casing variants detected."""
        (repo_dir / variant).write_text("# Claude instructions\n")
        assert has_claude_md(repo_dir) is True, (
            f"hasClaudeMD should detect {variant}"
        )
