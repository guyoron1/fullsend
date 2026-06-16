"""
Content Validation Tests — Full Implementation

STP Reference: outputs/stp/GH-15/GH-15_test_plan.md
STD Reference: outputs/std/GH-15/GH-15_test_description.yaml
Jira: GH-15
"""

import os
import re

import pytest


@pytest.fixture
def repo_root():
    """Resolve the repository root directory."""
    root = os.environ.get("REPO_ROOT", "")
    if not root:
        cwd = os.getcwd()
        root = cwd
        while True:
            if os.path.isdir(os.path.join(root, ".git")):
                break
            parent = os.path.dirname(root)
            if parent == root:
                pytest.fail("Could not locate repository root (.git directory)")
            root = parent
    return root


@pytest.fixture
def doc_content(repo_root):
    """Read the performance verification problem document."""
    doc_path = os.path.join(
        repo_root, "docs", "problems", "performance-verification.md"
    )
    assert os.path.isfile(doc_path), (
        f"Problem document not found at {doc_path}"
    )
    with open(doc_path, "r", encoding="utf-8") as f:
        return f.read()


class TestContentValidation:
    """
    [GH-15] Content Validation — Scenario 007

    Verify that the problem document covers all topics declared in the
    PR description: static analysis, benchmark suites, load testing,
    profiling gates, performance budgets, runtime anomaly detection,
    and agent-specific anti-patterns.
    """

    def test_detection_approaches_section_exists(self, doc_content):
        """[test_id:TS-GH-15-007a] Detection approaches section exists."""
        assert re.search(
            r"^## .*[Dd]etection approaches", doc_content, re.MULTILINE
        ), "Document must contain a '## Detection approaches' section"

    @pytest.mark.parametrize(
        "topic",
        [
            "static analysis",
            "benchmark",
            "load",
            "profiling",
            "performance budget",
            "runtime anomaly",
        ],
        ids=[
            "static-analysis",
            "benchmark-suites",
            "load-testing",
            "profiling-gates",
            "performance-budgets",
            "runtime-anomaly-detection",
        ],
    )
    def test_detection_approach_topic_covered(self, doc_content, topic):
        """[test_id:TS-GH-15-007b] Each declared detection approach topic is covered."""
        assert topic.lower() in doc_content.lower(), (
            f"Document must cover detection approach topic: '{topic}'"
        )

    def test_agent_specific_anti_patterns_section(self, doc_content):
        """[test_id:TS-GH-15-007c] Agent-specific anti-patterns section exists with content."""
        # Find the anti-patterns section
        match = re.search(
            r"^## .*[Aa]gent-specific anti-patterns.*$",
            doc_content,
            re.MULTILINE,
        )
        assert match, (
            "Document must contain an 'Agent-specific anti-patterns' section"
        )

        # Extract content between this heading and the next ## heading (or EOF)
        section_start = match.end()
        next_heading = re.search(r"^## ", doc_content[section_start:], re.MULTILINE)
        if next_heading:
            section_content = doc_content[section_start : section_start + next_heading.start()]
        else:
            section_content = doc_content[section_start:]

        # Section must have substantive content (not just whitespace)
        stripped = section_content.strip()
        assert len(stripped) > 50, (
            "Agent-specific anti-patterns section must contain substantive content "
            f"(found {len(stripped)} chars)"
        )

        # Should contain at least one list item or concrete pattern
        has_list_items = bool(re.search(r"^[\s]*[-*]", stripped, re.MULTILINE))
        has_subsections = bool(re.search(r"^###", stripped, re.MULTILINE))
        assert has_list_items or has_subsections, (
            "Anti-patterns section must contain concrete items (bullet list or subsections)"
        )

    def test_open_questions_section(self, doc_content):
        """[test_id:TS-GH-15-007d] Open questions section exists with actionable items."""
        # Find the open questions section
        match = re.search(
            r"^## .*[Oo]pen questions.*$",
            doc_content,
            re.MULTILINE,
        )
        assert match, "Document must contain an 'Open questions' section"

        # Extract content between this heading and the next ## heading (or EOF)
        section_start = match.end()
        next_heading = re.search(r"^## ", doc_content[section_start:], re.MULTILINE)
        if next_heading:
            section_content = doc_content[section_start : section_start + next_heading.start()]
        else:
            section_content = doc_content[section_start:]

        stripped = section_content.strip()
        assert len(stripped) > 20, (
            "Open questions section must contain content "
            f"(found {len(stripped)} chars)"
        )

        # Should contain at least one question (line with ?)
        # or list items indicating questions
        has_questions = "?" in stripped
        has_list_items = bool(re.search(r"^[\s]*[-*]", stripped, re.MULTILINE))
        assert has_questions or has_list_items, (
            "Open questions section must contain at least one actionable question"
        )
