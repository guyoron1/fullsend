"""
Tests for review agent spec enforcement — compliance checking and scope creep detection.

Covers scenarios:
    - TS-GH-4-004: Review agent blocks non-compliant code
    - TS-GH-4-005: Review agent approves compliant code
    - TS-GH-4-006: Review agent detects scope creep

STP Reference:
    outputs/stp/GH-4/GH-4_test_plan.md

PR Reference:
    https://github.com/fullsend-ai/fullsend/pull/4
"""
import tempfile
from pathlib import Path

import pytest

from conftest import run_fullsend_command

pytestmark = [
    pytest.mark.tier2,
    pytest.mark.gh_4,
]


class TestReviewAgentEnforcement:
    """Tests for the review agent's spec enforcement capabilities.

    Markers:
        - tier2
        - gh_4
        - serial

    Preconditions:
        - fullsend binary available in PATH
        - AI/LLM inference endpoint accessible
        - Review agent configured and operational
    """

    @pytest.fixture(scope="function")
    def spec_checklist_scope_function(
        self,
        fullsend_binary: str,
        llm_endpoint: str,
        prototype_with_testable_behavior: Path,
        spec_output_dir_scope_function: Path,
    ) -> Path:
        """Generate a spec checklist from prototype code for review agent tests.

        Args:
            fullsend_binary: Path to the fullsend CLI binary.
            llm_endpoint: URL of the LLM inference endpoint.
            prototype_with_testable_behavior: Prototype directory with testable code.
            spec_output_dir_scope_function: Output directory for generated spec.

        Returns:
            Path to the generated spec checklist file.
        """
        result = run_fullsend_command(
            binary=fullsend_binary,
            subcommand="vibe-to-spec",
            args=[
                "--input", str(prototype_with_testable_behavior),
                "--output", str(spec_output_dir_scope_function),
            ],
        )
        assert result.returncode == 0, (
            f"Failed to generate spec checklist: {result.stderr}"
        )

        checklist_path = spec_output_dir_scope_function / "checklist.yaml"
        if not checklist_path.exists():
            # Fall back to any YAML file in the output
            yaml_files = list(spec_output_dir_scope_function.glob("*.yaml"))
            assert len(yaml_files) > 0, "No spec checklist generated"
            checklist_path = yaml_files[0]

        return checklist_path

    @staticmethod
    def _write_diff_file(content: str, prefix: str) -> Path:
        """Write a temporary diff file with the given content.

        Args:
            content: The diff content to write.
            prefix: Prefix for the temporary file name.

        Returns:
            Path to the created diff file.
        """
        diff_file = Path(tempfile.mktemp(prefix=prefix, suffix=".diff"))
        diff_file.write_text(content)
        return diff_file

    @pytest.mark.serial
    def test_ts_gh_4_004_block_non_compliant_code(
        self,
        fullsend_binary: str,
        spec_checklist_scope_function: Path,
    ) -> None:
        """TS-GH-4-004: Verify review agent blocks code not matching generated spec.

        Priority: P0
        MVP: True

        This test validates that the review agent correctly identifies and
        blocks a PR when the submitted code does not match the generated
        spec checklist.

        Acceptance Criteria:
            - Review agent returns a 'blocked' or 'changes_requested' status
            - Review agent identifies specific spec checklist items not satisfied
            - Review output includes actionable feedback
        """
        # SETUP-02: Prepare code change that violates spec
        # The spec requires Add(a, b) but the code implements Subtract(a, b)
        non_compliant_diff = self._write_diff_file(
            content=(
                "--- a/main.go\n"
                "+++ b/main.go\n"
                "@@ -1,3 +1,5 @@\n"
                " package main\n"
                "+\n"
                "+// Subtract returns the difference (NOT what spec requires)\n"
                "+func Subtract(a, b int) int { return a - b }\n"
            ),
            prefix="non-compliant-",
        )

        try:
            # TEST-01: Run review agent against non-compliant code
            result = run_fullsend_command(
                binary=fullsend_binary,
                subcommand="review",
                args=[
                    "--spec", str(spec_checklist_scope_function),
                    "--diff", str(non_compliant_diff),
                ],
            )

            # TEST-02: Check review agent verdict
            combined_output = (result.stdout + result.stderr).lower()

            # ASSERT-01: Review agent blocks non-compliant code
            blocked_indicators = ["blocked", "changes_requested", "fail", "rejected"]
            is_blocked = any(
                indicator in combined_output for indicator in blocked_indicators
            )
            assert is_blocked, (
                f"Review agent should block non-compliant code but did not. "
                f"stdout: {result.stdout}, stderr: {result.stderr}"
            )

            # TEST-03: Verify review agent provides specific feedback
            # ASSERT-02: Review agent cites specific spec violations
            feedback_indicators = [
                "spec", "checklist", "requirement", "missing", "violation"
            ]
            has_specific_feedback = any(
                indicator in combined_output for indicator in feedback_indicators
            )
            assert has_specific_feedback, (
                f"Review agent should cite specific spec violations. "
                f"Output: {result.stdout}"
            )

        finally:
            non_compliant_diff.unlink(missing_ok=True)

    @pytest.mark.serial
    def test_ts_gh_4_005_approve_compliant_code(
        self,
        fullsend_binary: str,
        spec_checklist_scope_function: Path,
    ) -> None:
        """TS-GH-4-005: Verify review agent permits code matching generated spec checklist.

        Priority: P0
        MVP: True

        This test validates that the review agent correctly approves a PR
        when the submitted code satisfies all items in the generated spec
        checklist.

        Acceptance Criteria:
            - Review agent returns an 'approved' or 'pass' status
            - All spec checklist items are marked as satisfied
            - No false positive violations reported
        """
        # SETUP-02: Prepare code change that matches spec
        compliant_diff = self._write_diff_file(
            content=(
                "--- a/main.go\n"
                "+++ b/main.go\n"
                "@@ -1,3 +1,8 @@\n"
                " package main\n"
                "+\n"
                "+// Add returns the sum of two integers.\n"
                "+func Add(a, b int) int { return a + b }\n"
                "+\n"
                "+// Subtract returns the difference of two integers.\n"
                "+func Subtract(a, b int) int { return a - b }\n"
            ),
            prefix="compliant-",
        )

        try:
            # TEST-01: Run review agent against compliant code
            result = run_fullsend_command(
                binary=fullsend_binary,
                subcommand="review",
                args=[
                    "--spec", str(spec_checklist_scope_function),
                    "--diff", str(compliant_diff),
                ],
            )

            # TEST-02: Check review agent verdict
            combined_output = (result.stdout + result.stderr).lower()

            # ASSERT-01: Review agent approves compliant code
            approved_indicators = ["approved", "pass", "lgtm", "accepted", "success"]
            is_approved = any(
                indicator in combined_output for indicator in approved_indicators
            )
            assert is_approved, (
                f"Review agent should approve compliant code but did not. "
                f"Compliant code is incorrectly blocked, creating developer friction. "
                f"stdout: {result.stdout}, stderr: {result.stderr}"
            )

            # ASSERT-02: No false positive spec violations
            violation_indicators = ["violation", "blocked", "rejected", "fail"]
            has_violations = any(
                indicator in combined_output for indicator in violation_indicators
            )
            assert not has_violations, (
                f"Review agent reported false positive violations on compliant code. "
                f"Output: {result.stdout}"
            )

        finally:
            compliant_diff.unlink(missing_ok=True)

    @pytest.mark.serial
    def test_ts_gh_4_006_detect_scope_creep(
        self,
        fullsend_binary: str,
        spec_checklist_scope_function: Path,
    ) -> None:
        """TS-GH-4-006: Verify review agent detects and blocks scope creep beyond spec.

        Priority: P0
        MVP: True

        This test validates that the review agent detects when a PR includes
        functionality that goes beyond what the generated spec defines, even
        when the spec requirements themselves are also satisfied.

        Acceptance Criteria:
            - Review agent returns 'blocked' status for code with extra functionality
            - Review agent specifically identifies the out-of-scope additions
            - Review agent distinguishes between missing spec items and scope creep
        """
        # SETUP-02: Prepare code with scope creep (satisfies spec + extra)
        scope_creep_diff = self._write_diff_file(
            content=(
                "--- a/main.go\n"
                "+++ b/main.go\n"
                "@@ -1,3 +1,11 @@\n"
                " package main\n"
                "+\n"
                "+// Add returns the sum of two integers.\n"
                "+func Add(a, b int) int { return a + b }\n"
                "+\n"
                "+// Subtract returns the difference of two integers.\n"
                "+func Subtract(a, b int) int { return a - b }\n"
                "+\n"
                "+// Multiply returns the product — NOT in spec (scope creep)\n"
                "+func Multiply(a, b int) int { return a * b }\n"
            ),
            prefix="scope-creep-",
        )

        try:
            # TEST-01: Run review agent against code with scope creep
            result = run_fullsend_command(
                binary=fullsend_binary,
                subcommand="review",
                args=[
                    "--spec", str(spec_checklist_scope_function),
                    "--diff", str(scope_creep_diff),
                ],
            )

            combined_output = (result.stdout + result.stderr).lower()

            # TEST-02: Check review agent detects scope creep
            scope_creep_indicators = [
                "scope", "creep", "unauthorized", "out-of-scope",
                "out of scope", "extra", "unexpected", "not in spec",
            ]
            detected_scope_creep = any(
                indicator in combined_output
                for indicator in scope_creep_indicators
            )

            # ASSERT-01: Review agent blocks code with scope creep
            blocked_indicators = ["blocked", "changes_requested", "fail", "rejected"]
            is_blocked = any(
                indicator in combined_output for indicator in blocked_indicators
            )
            assert is_blocked, (
                f"Review agent should block code with scope creep. "
                f"Unauthorized functionality can be merged unchecked. "
                f"stdout: {result.stdout}, stderr: {result.stderr}"
            )

            # TEST-03: Verify review agent blocks the PR with scope creep reason
            # ASSERT-02: Review agent identifies specific out-of-scope additions
            assert detected_scope_creep, (
                f"Review agent should identify out-of-scope additions. "
                f"Developers cannot identify which code to remove. "
                f"Output: {result.stdout}"
            )

        finally:
            scope_creep_diff.unlink(missing_ok=True)
