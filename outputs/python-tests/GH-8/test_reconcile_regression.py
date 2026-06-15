"""
Tests for reconcile-repos-test.sh regression suite.

Validates that the existing regression test script passes end-to-end,
covering the trailing-newline encoding fix and stale-shim atomic update
behavior from issue fullsend-ai#2247.

STP Reference: outputs/stp/GH-8/GH-8_test_plan.md
Jira: GH-8
"""

import subprocess

import pytest

pytestmark = [
    pytest.mark.tier2,
]


class TestReconcileRegression:
    """Tests for reconcile-repos-test.sh regression suite execution.

    Markers:
        - tier2
        - p0

    Preconditions:
        - Bash 4.0+ with GNU coreutils
        - reconcile-repos-test.sh script present and executable
        - reconcile-repos.sh script present (sourced by test script)
    """

    def test_ts_gh_8_006_regression_suite_passes_end_to_end(
        self,
        reconcile_test_script_path,
    ):
        """TS-GH-8-006: Verify reconcile-repos-test.sh regression test passes end-to-end.

        Scenario: Execute the existing reconcile-repos-test.sh regression test
        script that was added as part of the fix for issue #2247. This script
        contains multiple test cases including the specific regression test for
        trailing-newline encoding and the existing stale-shim atomic update test.

        The regression test script is the primary CI gate for the shim
        reconciliation logic. Running it end-to-end validates that both the
        fix and all existing functionality work correctly together.
        """
        # SETUP-01: Locate reconcile-repos-test.sh (provided by fixture)
        assert reconcile_test_script_path.exists(), (
            f"Test script must exist at {reconcile_test_script_path}"
        )

        # TEST-01: Execute reconcile-repos-test.sh
        result = subprocess.run(
            ["bash", str(reconcile_test_script_path)],
            capture_output=True,
            text=True,
            timeout=120,
        )
        output = result.stdout + result.stderr

        # TEST-02: Check exit code
        # ASSERT-01: Regression test script exits with code 0
        assert result.returncode == 0, (
            f"Regression test script should exit with code 0, "
            f"got {result.returncode}.\n"
            f"stdout: {result.stdout}\n"
            f"stderr: {result.stderr}"
        )

        # TEST-03: Verify no failures in output
        # ASSERT-02: No test failures in script output
        output_upper = output.upper()
        assert "FAIL" not in output_upper, (
            f"No test failures should appear in output.\nOutput: {output}"
        )

        # ASSERT-03: Script output shows all tests passed
        output_lower = output.lower()
        has_pass_indicator = any(
            keyword in output_lower
            for keyword in ["pass", "success", "ok", "all tests"]
        )
        if not has_pass_indicator:
            # Warn but don't fail — some scripts have no explicit pass message
            import warnings

            warnings.warn(
                f"No explicit pass indicator found in test output. "
                f"Output: {output[:500]}",
                stacklevel=1,
            )
