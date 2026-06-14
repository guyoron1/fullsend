"""
Tests for vibe-to-spec workflow — core spec generation and error handling.

Covers scenarios:
    - TS-GH-4-001: Valid spec generation from prototype code
    - TS-GH-4-002: Exploration artifacts cleanup after spec generation
    - TS-GH-4-003: Error handling for prototype with no testable behavior
    - TS-GH-4-009: Error handling for ambiguous/contradictory prototype input

STP Reference:
    outputs/stp/GH-4/GH-4_test_plan.md

PR Reference:
    https://github.com/fullsend-ai/fullsend/pull/4
"""
from pathlib import Path

import pytest
import yaml

from conftest import run_fullsend_command

pytestmark = [
    pytest.mark.tier2,
    pytest.mark.gh_4,
]


class TestVibeToSpecWorkflow:
    """Tests for the vibe-to-spec workflow core functionality.

    Markers:
        - tier2
        - gh_4
        - serial

    Preconditions:
        - fullsend binary available in PATH
        - AI/LLM inference endpoint accessible
        - Go toolchain installed (Go 1.23+)
    """

    @pytest.mark.serial
    def test_ts_gh_4_001_generate_valid_spec_from_prototype(
        self,
        fullsend_binary: str,
        llm_endpoint: str,
        prototype_with_testable_behavior: Path,
        spec_output_dir_scope_function: Path,
    ) -> None:
        """TS-GH-4-001: Verify vibe-to-spec workflow produces valid spec from prototype code.

        Priority: P0
        MVP: True

        This test validates that the vibe-to-spec workflow correctly generates
        a valid, structured formal specification from developer prototype code.

        Acceptance Criteria:
            - Workflow accepts prototype code directory as input and completes without error
            - Generated specification contains a functional requirements section
            - Generated specification contains acceptance scenarios with pass/fail criteria
            - Generated specification is valid YAML/structured format
        """
        # SETUP: Verify prototype directory contains Go files
        go_files = list(prototype_with_testable_behavior.glob("*.go"))
        assert len(go_files) > 0, "Prototype directory must contain .go files"

        # TEST-01: Execute vibe-to-spec workflow on prototype directory
        result = run_fullsend_command(
            binary=fullsend_binary,
            subcommand="vibe-to-spec",
            args=[
                "--input", str(prototype_with_testable_behavior),
                "--output", str(spec_output_dir_scope_function),
            ],
        )

        # ASSERT-01: Workflow completes without error
        assert result.returncode == 0, (
            f"vibe-to-spec workflow failed with exit code {result.returncode}. "
            f"stderr: {result.stderr}"
        )

        # TEST-02: Validate generated specification structure
        spec_files = list(spec_output_dir_scope_function.glob("*.yaml")) + \
            list(spec_output_dir_scope_function.glob("*.yml"))
        assert len(spec_files) > 0, (
            "No specification files generated in output directory"
        )

        spec_content = spec_files[0].read_text()
        spec_data = yaml.safe_load(spec_content)

        # ASSERT-02: Generated spec contains functional requirements
        assert spec_data is not None, "Spec file is empty or invalid YAML"
        assert "functional_requirements" in spec_data, (
            "Generated spec missing 'functional_requirements' section. "
            f"Available keys: {list(spec_data.keys())}"
        )
        assert len(spec_data["functional_requirements"]) > 0, (
            "functional_requirements section is empty"
        )

        # ASSERT-03: Generated spec contains acceptance scenarios
        assert "acceptance_scenarios" in spec_data, (
            "Generated spec missing 'acceptance_scenarios' section. "
            f"Available keys: {list(spec_data.keys())}"
        )
        scenarios = spec_data["acceptance_scenarios"]
        assert len(scenarios) > 0, "acceptance_scenarios section is empty"

        for scenario in scenarios:
            assert "pass_criteria" in scenario or "criteria" in scenario, (
                f"Acceptance scenario missing pass/fail criteria: {scenario}"
            )

        # ASSERT-04: Generated spec is valid structured format (already parsed above)
        # Re-parse to confirm round-trip validity
        reparsed = yaml.safe_load(yaml.dump(spec_data))
        assert reparsed == spec_data, "Spec data is not round-trip stable"

    @pytest.mark.serial
    def test_ts_gh_4_002_cleanup_exploration_artifacts(
        self,
        fullsend_binary: str,
        llm_endpoint: str,
        prototype_with_testable_behavior: Path,
        spec_output_dir_scope_function: Path,
    ) -> None:
        """TS-GH-4-002: Verify exploration artifacts are cleaned up after spec generation.

        Priority: P1
        MVP: False

        This test validates that after the vibe-to-spec workflow generates a
        formal specification, all exploration/prototype artifacts are properly
        cleaned up and do not persist in the working directory.

        Acceptance Criteria:
            - After spec generation completes, exploration artifact directory no longer exists
            - No prototype source files remain in the working directory
            - Generated spec file is the only output preserved
        """
        exploration_dir = prototype_with_testable_behavior

        # SETUP-01: Verify exploration directory exists with files
        assert exploration_dir.exists(), "Exploration directory should exist before workflow"
        go_files_before = list(exploration_dir.glob("*.go"))
        assert len(go_files_before) > 0, "Exploration directory must have Go files"

        # SETUP-02: Run vibe-to-spec workflow to completion
        result = run_fullsend_command(
            binary=fullsend_binary,
            subcommand="vibe-to-spec",
            args=[
                "--input", str(exploration_dir),
                "--output", str(spec_output_dir_scope_function),
            ],
        )
        assert result.returncode == 0, (
            f"vibe-to-spec workflow failed: {result.stderr}"
        )

        # TEST-01: Check exploration artifact directory no longer exists
        # ASSERT-01: Exploration directory is removed after spec generation
        assert not exploration_dir.exists(), (
            f"Exploration directory {exploration_dir} still exists after "
            f"spec generation. Prototype code may leak into production commits."
        )

        # TEST-02: Verify no prototype files remain
        parent_dir = exploration_dir.parent
        remaining_go_files = list(parent_dir.rglob("*.go"))
        exploration_go_files = [
            f for f in remaining_go_files
            if str(exploration_dir.name) in str(f)
        ]
        assert len(exploration_go_files) == 0, (
            f"Prototype Go files still found: {exploration_go_files}"
        )

        # TEST-03: Verify generated spec file is preserved
        # ASSERT-02: Generated spec is preserved after cleanup
        spec_files = list(spec_output_dir_scope_function.glob("*.yaml")) + \
            list(spec_output_dir_scope_function.glob("*.yml"))
        assert len(spec_files) > 0, (
            "Generated spec file was not preserved after cleanup. "
            "Cleanup is too aggressive and removes wanted output."
        )

    @pytest.mark.serial
    def test_ts_gh_4_003_error_for_no_testable_behavior(
        self,
        fullsend_binary: str,
        prototype_without_testable_behavior: Path,
        spec_output_dir_scope_function: Path,
    ) -> None:
        """TS-GH-4-003: Verify error returned when prototype contains no testable behavior.

        Priority: P1
        MVP: False

        This test validates that the vibe-to-spec workflow returns a clear,
        actionable error message when the input prototype contains no testable
        behavior.

        Acceptance Criteria:
            - Workflow returns a non-zero exit code
            - Error message clearly indicates prototype lacks testable behavior
            - Error message suggests what the developer should do
        """
        # SETUP-01: Verify prototype directory has no testable functions
        content = (prototype_without_testable_behavior / "main.go").read_text()
        assert "func " not in content or "func " not in content.split("//")[0], (
            "Test setup error: prototype should have no exported functions"
        )

        # TEST-01: Execute vibe-to-spec workflow on empty prototype
        result = run_fullsend_command(
            binary=fullsend_binary,
            subcommand="vibe-to-spec",
            args=[
                "--input", str(prototype_without_testable_behavior),
                "--output", str(spec_output_dir_scope_function),
            ],
        )

        # ASSERT-01: Workflow returns non-zero exit code
        assert result.returncode != 0, (
            "Workflow should fail with non-zero exit code when prototype "
            "has no testable behavior, but returned 0"
        )

        # TEST-02: Capture and validate error message
        error_output = result.stderr.lower()

        # ASSERT-02: Error message is actionable
        testable_keywords = [
            "testable",
            "no functions",
            "no exported",
            "empty",
            "no behavior",
            "insufficient",
        ]
        has_relevant_error = any(
            keyword in error_output for keyword in testable_keywords
        )
        assert has_relevant_error, (
            f"Error message should indicate prototype lacks testable behavior. "
            f"Got: {result.stderr}"
        )

    @pytest.mark.serial
    def test_ts_gh_4_009_error_for_ambiguous_prototype(
        self,
        fullsend_binary: str,
        prototype_with_ambiguous_behavior: Path,
        spec_output_dir_scope_function: Path,
    ) -> None:
        """TS-GH-4-009: Verify error for ambiguous or contradictory prototype input.

        Priority: P1
        MVP: False

        This test validates that the vibe-to-spec workflow returns a clear
        error when the input prototype contains ambiguous or contradictory
        behavior that cannot be reliably converted to a formal specification.

        Acceptance Criteria:
            - Workflow returns a non-zero exit code for ambiguous input
            - Error message explains why the prototype is ambiguous
            - Error message suggests how to resolve the ambiguity
        """
        # SETUP-01: Verify prototype contains contradictory code
        content = (prototype_with_ambiguous_behavior / "contradictory.go").read_text()
        assert "uppercase" in content.lower() and "ToLower" in content, (
            "Test setup error: prototype should contain contradictory behavior"
        )

        # TEST-01: Execute vibe-to-spec on ambiguous prototype
        result = run_fullsend_command(
            binary=fullsend_binary,
            subcommand="vibe-to-spec",
            args=[
                "--input", str(prototype_with_ambiguous_behavior),
                "--output", str(spec_output_dir_scope_function),
            ],
        )

        # ASSERT-01: Workflow returns error for ambiguous input
        assert result.returncode != 0, (
            "Workflow should fail with non-zero exit code for ambiguous "
            "prototype input, but returned 0. Incorrect specs may be generated "
            "from ambiguous prototypes."
        )

        # TEST-02: Validate error message explains ambiguity
        error_output = result.stderr.lower()

        # ASSERT-02: Error message is descriptive
        ambiguity_keywords = [
            "ambiguous",
            "contradictory",
            "conflict",
            "inconsistent",
            "unclear",
            "mismatch",
        ]
        has_ambiguity_error = any(
            keyword in error_output for keyword in ambiguity_keywords
        )
        assert has_ambiguity_error, (
            f"Error message should explain the ambiguity in prototype. "
            f"Got: {result.stderr}"
        )
