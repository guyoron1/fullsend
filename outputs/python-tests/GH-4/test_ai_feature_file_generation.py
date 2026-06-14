"""
Tests for AI feature file generation — output structure and acceptance criteria validation.

Covers scenarios:
    - TS-GH-4-007: AI generates functional requirements section from prototype
    - TS-GH-4-008: AI generates acceptance scenarios with pass/fail criteria

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


class TestAiFeatureFileGeneration:
    """Tests for AI-generated feature file structure and content.

    Markers:
        - tier2
        - gh_4
        - serial

    Preconditions:
        - fullsend binary available in PATH
        - AI/LLM inference endpoint accessible
        - Prototype code with well-defined exported functions
    """

    @pytest.mark.serial
    def test_ts_gh_4_007_generate_functional_requirements_section(
        self,
        fullsend_binary: str,
        llm_endpoint: str,
        prototype_with_testable_behavior: Path,
        spec_output_dir_scope_function: Path,
    ) -> None:
        """TS-GH-4-007: Verify AI generates functional requirements section from prototype.

        Priority: P1
        MVP: False

        This test validates that when AI generates a feature file from a
        prototype, the output contains a properly structured functional
        requirements section with machine-evaluable criteria.

        Acceptance Criteria:
            - Generated feature file contains a 'functional_requirements' section
            - Functional requirements are structured as discrete, numbered items
            - Each requirement is machine-evaluable (not ambiguous prose)
        """
        # SETUP-01: Verify prototype has multiple testable functions
        go_files = list(prototype_with_testable_behavior.glob("*.go"))
        assert len(go_files) > 0, "Prototype must contain Go files with exported functions"

        # TEST-01: Generate feature file from prototype
        result = run_fullsend_command(
            binary=fullsend_binary,
            subcommand="vibe-to-spec",
            args=[
                "--input", str(prototype_with_testable_behavior),
                "--output", str(spec_output_dir_scope_function),
            ],
        )
        assert result.returncode == 0, (
            f"Feature file generation failed: {result.stderr}"
        )

        # TEST-02: Parse generated feature file
        spec_files = list(spec_output_dir_scope_function.glob("*.yaml")) + \
            list(spec_output_dir_scope_function.glob("*.yml"))
        assert len(spec_files) > 0, "No feature files generated"

        feature_content = spec_files[0].read_text()
        feature_data = yaml.safe_load(feature_content)
        assert feature_data is not None, "Feature file is empty or invalid YAML"

        # TEST-03: Validate functional requirements section exists
        # ASSERT-01: Feature file contains functional requirements section
        assert "functional_requirements" in feature_data, (
            f"Generated feature file missing 'functional_requirements' section. "
            f"Generated specs cannot drive review agent enforcement. "
            f"Available keys: {list(feature_data.keys())}"
        )

        requirements = feature_data["functional_requirements"]
        assert isinstance(requirements, list), (
            "functional_requirements should be a list of discrete items"
        )
        assert len(requirements) > 0, (
            "functional_requirements section is empty"
        )

        # TEST-04: Validate requirements are machine-evaluable
        # ASSERT-02: Requirements are structured and numbered
        for idx, requirement in enumerate(requirements):
            if isinstance(requirement, dict):
                structured_fields = {"id", "description", "criteria"}
                present_fields = set(requirement.keys()) & structured_fields
                assert len(present_fields) >= 2, (
                    f"Requirement {idx} lacks structured fields. "
                    f"Expected at least 'id' and 'description' or 'criteria'. "
                    f"Got keys: {list(requirement.keys())}. "
                    f"Requirements are unstructured prose, not machine-evaluable."
                )

    @pytest.mark.serial
    def test_ts_gh_4_008_generate_acceptance_scenarios_with_criteria(
        self,
        fullsend_binary: str,
        llm_endpoint: str,
        prototype_with_testable_behavior: Path,
        spec_output_dir_scope_function: Path,
    ) -> None:
        """TS-GH-4-008: Verify AI generates acceptance scenarios with pass/fail criteria.

        Priority: P1
        MVP: False

        This test validates that the AI-generated feature file contains
        acceptance scenarios with explicit pass/fail criteria that can be
        used by review agents to evaluate code compliance.

        Acceptance Criteria:
            - Generated feature file contains an 'acceptance_scenarios' section
            - Each scenario has explicit pass criteria
            - Each scenario has explicit fail criteria
            - Scenarios are testable by review agents
        """
        # SETUP-01: Create prototype with clear input/output behavior
        # Using the shared prototype fixture which has Add/Subtract functions
        assert prototype_with_testable_behavior.exists(), (
            "Prototype directory should exist"
        )

        # TEST-01: Generate feature file from prototype
        result = run_fullsend_command(
            binary=fullsend_binary,
            subcommand="vibe-to-spec",
            args=[
                "--input", str(prototype_with_testable_behavior),
                "--output", str(spec_output_dir_scope_function),
            ],
        )
        assert result.returncode == 0, (
            f"Feature file generation failed: {result.stderr}"
        )

        # TEST-02: Validate acceptance scenarios section exists
        spec_files = list(spec_output_dir_scope_function.glob("*.yaml")) + \
            list(spec_output_dir_scope_function.glob("*.yml"))
        assert len(spec_files) > 0, "No feature files generated"

        feature_data = yaml.safe_load(spec_files[0].read_text())
        assert feature_data is not None, "Feature file is empty"

        # ASSERT-01: Feature file contains acceptance scenarios
        assert "acceptance_scenarios" in feature_data, (
            f"Generated feature file missing 'acceptance_scenarios' section. "
            f"Generated specs lack testable scenarios for review agents. "
            f"Available keys: {list(feature_data.keys())}"
        )

        scenarios = feature_data["acceptance_scenarios"]
        assert isinstance(scenarios, list), (
            "acceptance_scenarios should be a list"
        )
        assert len(scenarios) > 0, (
            "acceptance_scenarios section is empty"
        )

        # TEST-03: Validate each scenario has pass/fail criteria
        # ASSERT-02: Each scenario has pass/fail criteria
        for idx, scenario in enumerate(scenarios):
            if isinstance(scenario, dict):
                has_pass = "pass_criteria" in scenario or "pass" in scenario
                has_fail = "fail_criteria" in scenario or "fail" in scenario
                has_criteria = "criteria" in scenario

                assert has_pass or has_criteria, (
                    f"Acceptance scenario {idx} missing pass criteria. "
                    f"Review agents cannot make binary compliance decisions. "
                    f"Scenario keys: {list(scenario.keys())}"
                )
                assert has_fail or has_criteria, (
                    f"Acceptance scenario {idx} missing fail criteria. "
                    f"Review agents cannot make binary compliance decisions. "
                    f"Scenario keys: {list(scenario.keys())}"
                )
