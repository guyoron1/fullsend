"""
Tests for shim enrollment workflow integration.

Validates that the reconcile-repos.sh script correctly skips repos with
up-to-date shims and creates update PRs for repos with stale shims.
These tests exercise the script end-to-end with mocked external commands.

STP Reference: outputs/stp/GH-8/GH-8_test_plan.md
Jira: GH-8
"""

import os
import pathlib
import subprocess

import pytest

from conftest import encode_base64, write_mock_script

pytestmark = [
    pytest.mark.tier2,
]


class TestShimEnrollmentWorkflow:
    """Tests for shim enrollment workflow via reconcile-repos.sh.

    Markers:
        - tier2
        - p1

    Preconditions:
        - Bash 4.0+ with GNU coreutils
        - reconcile-repos.sh script available in hack/
        - Mock gh, yq, base64 commands in test PATH
    """

    def test_ts_gh_8_004_skip_repo_with_up_to_date_shim(
        self,
        reconcile_script_path,
        mock_command_dir,
        env_with_mock_path,
        managed_shim_content,
    ):
        """TS-GH-8-004: Verify enrollment skips repos with up-to-date shim.

        Scenario: When a repo has an up-to-date shim, the script should skip
        the repo without creating an update PR. This was the user-facing
        symptom of issue #2247 — unnecessary PRs were created for repos
        that already had the correct shim.

        The mock GitHub API returns base64 content that is identical (after
        decoding) to the managed content.
        """
        # SETUP-01: Temporary directory already created via mock_command_dir fixture

        # SETUP-02: Create mock gh command returning current shim content
        managed_b64 = encode_base64(managed_shim_content)
        gh_log_file = mock_command_dir / "gh_calls.log"

        write_mock_script(
            directory=mock_command_dir,
            name="gh",
            script_body=(
                f'echo "$@" >> "{gh_log_file}"\n'
                f'if echo "$@" | grep -q "contents"; then\n'
                f'  echo \'{{"content": "{managed_b64}"}}\' | '
                f"jq -r '.content'\n"
                f"fi\n"
                f'if echo "$@" | grep -q "pr create"; then\n'
                f'  echo "PR_CREATED" >> "{gh_log_file}"\n'
                f"fi"
            ),
        )

        # SETUP-03: Create mock yq command returning enrolled repo list
        write_mock_script(
            directory=mock_command_dir,
            name="yq",
            script_body='echo "test-org/test-repo"',
        )

        # SETUP-04: Mock base64 command that properly decodes
        write_mock_script(
            directory=mock_command_dir,
            name="base64",
            script_body=(
                'if [ "$1" = "-d" ] || [ "$1" = "--decode" ]; then\n'
                "  /usr/bin/base64 -d\n"
                "else\n"
                "  /usr/bin/base64 $@\n"
                "fi"
            ),
        )

        # TEST-01: Execute reconcile-repos.sh
        result = subprocess.run(
            ["bash", str(reconcile_script_path)],
            capture_output=True,
            text=True,
            env=env_with_mock_path,
            timeout=30,
        )
        output = result.stdout + result.stderr

        # ASSERT-01: Script exits successfully
        assert result.returncode == 0, (
            f"Script should exit successfully, got rc={result.returncode}.\n"
            f"stdout: {result.stdout}\nstderr: {result.stderr}"
        )

        # ASSERT-02: No update PR created for up-to-date repo
        gh_log_content = ""
        if gh_log_file.exists():
            gh_log_content = gh_log_file.read_text()
        assert "PR_CREATED" not in gh_log_content, (
            "No update PR should be created for a repo with current shim"
        )

        # ASSERT-03: Script output indicates repo was skipped
        output_lower = output.lower()
        assert any(
            keyword in output_lower
            for keyword in ["skip", "up-to-date", "up to date", "current", "no drift"]
        ), f"Script output should indicate repo was skipped. Got: {output}"

    def test_ts_gh_8_005_create_update_pr_for_stale_shim(
        self,
        reconcile_script_path,
        mock_command_dir,
        env_with_mock_path,
        managed_shim_content,
        outdated_shim_content,
    ):
        """TS-GH-8-005: Verify enrollment creates update PR for stale shim.

        Scenario: When a repo has a stale shim (referencing an old version),
        the script should detect the drift and create an update PR. This is
        the primary mechanism for keeping enrolled repos on the current shim
        version.

        The mock GitHub API returns base64 content that differs from the
        managed content after decoding.
        """
        # SETUP-01: Temporary directory already created via mock_command_dir fixture

        # SETUP-02: Create mock gh command returning outdated shim content
        outdated_b64 = encode_base64(outdated_shim_content)
        gh_log_file = mock_command_dir / "gh_calls.log"

        write_mock_script(
            directory=mock_command_dir,
            name="gh",
            script_body=(
                f'echo "$@" >> "{gh_log_file}"\n'
                f'if echo "$@" | grep -q "contents"; then\n'
                f'  echo \'{{"content": "{outdated_b64}"}}\' | '
                f"jq -r '.content'\n"
                f"fi\n"
                f'if echo "$@" | grep -q "pr create"; then\n'
                f'  echo "PR_CREATED" >> "{gh_log_file}"\n'
                f'  echo "https://github.com/test-org/test-repo/pull/1"\n'
                f"fi"
            ),
        )

        # SETUP-03: Create mock yq and other required commands
        write_mock_script(
            directory=mock_command_dir,
            name="yq",
            script_body='echo "test-org/test-repo"',
        )

        write_mock_script(
            directory=mock_command_dir,
            name="base64",
            script_body=(
                'if [ "$1" = "-d" ] || [ "$1" = "--decode" ]; then\n'
                "  /usr/bin/base64 -d\n"
                "else\n"
                "  /usr/bin/base64 $@\n"
                "fi"
            ),
        )

        # TEST-01: Execute reconcile-repos.sh
        result = subprocess.run(
            ["bash", str(reconcile_script_path)],
            capture_output=True,
            text=True,
            env=env_with_mock_path,
            timeout=30,
        )
        output = result.stdout + result.stderr

        # ASSERT-01: Script exits successfully
        assert result.returncode == 0, (
            f"Script should exit successfully, got rc={result.returncode}.\n"
            f"stdout: {result.stdout}\nstderr: {result.stderr}"
        )

        # ASSERT-02: Update PR created for stale repo
        gh_log_content = ""
        if gh_log_file.exists():
            gh_log_content = gh_log_file.read_text()
        assert "pr create" in gh_log_content or "PR_CREATED" in gh_log_content, (
            "An update PR should be created for a repo with stale shim.\n"
            f"gh log: {gh_log_content}"
        )

        # ASSERT-03: Script output indicates drift was detected
        output_lower = output.lower()
        assert any(
            keyword in output_lower
            for keyword in ["stale", "drift", "update", "creating", "outdated"]
        ), f"Script output should indicate drift was detected. Got: {output}"
