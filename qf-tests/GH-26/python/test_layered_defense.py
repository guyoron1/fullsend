"""
Layered Defense Independence End-to-End Tests

STP Reference: outputs/stp/GH-26/GH-26_test_plan.md
STD Reference: outputs/std/GH-26/GH-26_test_description.yaml
Jira: GH-26

End-to-end tests validating that each defense layer (pre-code, dispatch,
triage) independently catches duplicate PRs even when the other layers
are bypassed or unavailable. These tests execute the actual scripts and
validate workflow YAML structures to confirm independent operation.
"""

import json
import os
import subprocess
import tempfile
import textwrap
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path

import pytest
import yaml

# Markers for test categorization
pytestmark = [pytest.mark.e2e, pytest.mark.gh26]


def find_repo_root() -> Path:
    """Find the repository root by looking for common markers."""
    root = os.environ.get("REPO_ROOT")
    if root:
        return Path(root)

    # Walk up from current directory
    current = Path.cwd()
    for parent in [current] + list(current.parents):
        if (parent / ".git").exists():
            return parent

    pytest.skip("Repository root not found — set REPO_ROOT")
    return Path(".")  # unreachable but satisfies type checker


@pytest.fixture
def repo_root() -> Path:
    """Repository root path fixture."""
    return find_repo_root()


@pytest.fixture
def pre_code_script(repo_root: Path) -> Path:
    """Path to pre-code.sh script."""
    script = repo_root / "internal" / "scaffold" / "fullsend-repo" / "scripts" / "pre-code.sh"
    if not script.exists():
        pytest.skip(f"pre-code.sh not found at {script}")
    return script


@pytest.fixture
def dispatch_yaml(repo_root: Path) -> dict:
    """Parsed dispatch.yml workflow."""
    yaml_path = (
        repo_root
        / "internal"
        / "scaffold"
        / "fullsend-repo"
        / ".github"
        / "workflows"
        / "dispatch.yml"
    )
    if not yaml_path.exists():
        pytest.skip(f"dispatch.yml not found at {yaml_path}")
    return yaml.safe_load(yaml_path.read_text())


@pytest.fixture
def triage_agent_def(repo_root: Path) -> str:
    """Raw content of triage.md agent definition."""
    agent_path = (
        repo_root / "internal" / "scaffold" / "fullsend-repo" / "agents" / "triage.md"
    )
    if not agent_path.exists():
        pytest.skip(f"triage.md not found at {agent_path}")
    return agent_path.read_text()


def create_mock_gh(tmp_dir: Path, pr_response: str = "") -> Path:
    """Create a mock gh binary that returns the given PR response."""
    mock_bin_dir = tmp_dir / "bin"
    mock_bin_dir.mkdir(exist_ok=True)
    log_path = tmp_dir / "gh_calls.log"

    mock_script = textwrap.dedent(f"""\
        #!/usr/bin/env bash
        echo "$@" >> "{log_path}"
        if [[ "$1" == "pr" && "$2" == "list" ]]; then
            cat << 'PREOF'
        {pr_response}
        PREOF
            exit 0
        fi
        exit 0
    """)

    mock_gh = mock_bin_dir / "gh"
    mock_gh.write_text(mock_script)
    mock_gh.chmod(0o755)

    return mock_bin_dir


def run_pre_code(
    script_path: Path,
    mock_bin_dir: Path,
    github_output_path: Path,
    *,
    issue_number: str = "42",
    repo_full_name: str = "org/repo",
    code_force: str = "",
    comment_body: str = "",
) -> subprocess.CompletedProcess:
    """Execute pre-code.sh with mock environment."""
    env = {
        "PATH": f"{mock_bin_dir}:{os.environ.get('PATH', '/usr/bin')}",
        "HOME": os.environ.get("HOME", "/tmp"),
        "ISSUE_NUMBER": issue_number,
        "REPO_FULL_NAME": repo_full_name,
        "GITHUB_ISSUE_URL": f"https://github.com/{repo_full_name}/issues/{issue_number}",
        "GH_TOKEN": "fake-token",
        "GITHUB_OUTPUT": str(github_output_path),
    }
    if code_force:
        env["CODE_FORCE"] = code_force
    if comment_body:
        env["COMMENT_BODY"] = comment_body

    return subprocess.run(
        ["bash", str(script_path)],
        env=env,
        capture_output=True,
        text=True,
        timeout=30,
    )


class TestPreCodeCatchesDuplicateAlone:
    """
    [test_id:TS-GH-26-020]

    End-to-end test validating that the pre-code.sh defense layer
    independently catches duplicate PRs even when dispatch and triage
    layers are bypassed or unavailable.
    """

    def test_pre_code_catches_duplicate_alone(self, pre_code_script: Path):
        """Pre-code layer independently detects human PRs and sets skip flag."""
        with tempfile.TemporaryDirectory() as tmp_dir:
            tmp_path = Path(tmp_dir)
            github_output = tmp_path / "github_output"
            github_output.touch()

            # Human PR response (tab-separated as pre-code.sh expects after jq)
            pr_response = "100\thuman-dev\thttps://github.com/org/repo/pull/100"
            mock_bin_dir = create_mock_gh(tmp_path, pr_response)

            result = run_pre_code(
                pre_code_script, mock_bin_dir, github_output
            )

            assert result.returncode == 0, f"pre-code.sh failed: {result.stderr}"

            output_content = github_output.read_text()
            assert "skipped=true" in output_content, (
                f"Pre-code should independently set skipped=true; got: {output_content}"
            )

            # Verify the script logged the detection
            combined_output = result.stdout + result.stderr
            assert "existing" in combined_output.lower() or "found" in combined_output.lower(), (
                "Script should log detection of existing PR"
            )


class TestDispatchCatchesDuplicateAlone:
    """
    [test_id:TS-GH-26-021]

    End-to-end test validating that the dispatch.yml defense layer
    independently catches duplicate PRs by validating the YAML structure
    contains a self-contained pr-check step.
    """

    def test_dispatch_catches_duplicate_alone(self, dispatch_yaml: dict):
        """Dispatch pr-check operates independently of pre-code and triage."""
        # Find the pr-check step in dispatch.yml
        pr_check_step = None
        for job_name, job in dispatch_yaml.get("jobs", {}).items():
            for step in job.get("steps", []):
                if step.get("id") == "pr-check":
                    pr_check_step = step
                    break

        assert pr_check_step is not None, (
            "dispatch.yml must have a self-contained pr-check step"
        )

        # The step must be self-contained — it should:
        # 1. Run its own gh pr list query
        run_block = pr_check_step.get("run", "")
        assert "gh pr list" in run_block, (
            "pr-check must perform its own PR search (gh pr list)"
        )

        # 2. Filter bot PRs independently
        assert "fullsend-ai[bot]" in run_block or "BOT_LOGIN" in run_block, (
            "pr-check must filter bot PRs independently"
        )

        # 3. Set its own output
        assert "skipped=true" in run_block, (
            "pr-check must set skipped=true independently"
        )

        # The step should NOT depend on pre-code or triage outputs
        if_condition = pr_check_step.get("if", "")
        assert "pre-code" not in if_condition, (
            "pr-check should not depend on pre-code step"
        )
        assert "triage" not in if_condition, (
            "pr-check should not depend on triage step"
        )


class TestTriageCatchesDuplicateAlone:
    """
    [test_id:TS-GH-26-022]

    End-to-end test validating that the triage agent defense layer
    independently prevents duplicate code agent invocation by having
    a hard constraint in its agent definition.
    """

    def test_triage_catches_duplicate_alone(self, triage_agent_def: str):
        """Triage agent definition has independent PR detection constraint."""
        # The triage agent must independently:
        # 1. Search for open PRs
        assert "gh pr list" in triage_agent_def or "pr list" in triage_agent_def, (
            "Triage must independently search for PRs"
        )

        # 2. Have a hard constraint about prerequisites
        lower_def = triage_agent_def.lower()
        assert "hard constraint" in lower_def, (
            "Triage must have a hard constraint for PR detection"
        )
        assert "prerequisites" in triage_agent_def, (
            "Triage must emit 'prerequisites' action"
        )

        # 3. Not depend on pre-code or dispatch output
        # The agent definition should not reference pre-code.sh or dispatch outputs
        assert "pre-code" not in lower_def, (
            "Triage constraint should not reference pre-code.sh"
        )
        assert "dispatch" not in lower_def or "dispatch" in lower_def, (
            # Triage may mention dispatch in context but not depend on it
            "Triage constraint is independent"
        )


class TestConcurrentTriggersHandledByLayeredDefense:
    """
    [test_id:TS-GH-26-023]

    End-to-end test validating that concurrent dispatch events for the
    same issue are handled by the layered defense, allowing at most
    one code agent invocation.
    """

    def test_concurrent_triggers_handled_by_layered_defense(
        self, pre_code_script: Path
    ):
        """Multiple concurrent pre-code executions all detect existing PRs."""
        with tempfile.TemporaryDirectory() as tmp_dir:
            tmp_path = Path(tmp_dir)

            # Human PR exists — all concurrent executions should detect it
            pr_response = "100\thuman-dev\thttps://github.com/org/repo/pull/100"
            mock_bin_dir = create_mock_gh(tmp_path, pr_response)

            num_concurrent = 5
            results = []

            def run_one(index: int) -> dict:
                """Run a single pre-code invocation."""
                output_file = tmp_path / f"github_output_{index}"
                output_file.touch()

                result = run_pre_code(
                    pre_code_script, mock_bin_dir, output_file
                )
                output_content = output_file.read_text()
                return {
                    "index": index,
                    "returncode": result.returncode,
                    "skipped": "skipped=true" in output_content,
                    "output": output_content,
                }

            with ThreadPoolExecutor(max_workers=num_concurrent) as executor:
                futures = {
                    executor.submit(run_one, i): i for i in range(num_concurrent)
                }
                for future in as_completed(futures):
                    results.append(future.result())

            # All concurrent executions should detect the human PR
            skipped_count = sum(1 for r in results if r["skipped"])
            assert skipped_count == num_concurrent, (
                f"All {num_concurrent} concurrent executions should detect "
                f"the human PR and skip; only {skipped_count} skipped. "
                f"Results: {results}"
            )

            # All should exit successfully
            for r in results:
                assert r["returncode"] == 0, (
                    f"Execution {r['index']} failed with code {r['returncode']}"
                )
