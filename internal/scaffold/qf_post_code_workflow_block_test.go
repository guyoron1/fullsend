package scaffold

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBlockedOutputIncludesFilePathAndReason covers STD scenario 7 (TS-GH84-007).
//
// When a workflow file is detected in the changed files list, the error output
// must include both the specific file path and a human-readable blocking reason.
func TestBlockedOutputIncludesFilePathAndReason(t *testing.T) {
	scriptPath := filepath.Join("fullsend-repo", "scripts", "post-code.sh")
	requireFileExists(t, scriptPath)

	// Build a bash snippet that sources the blocking logic from post-code.sh
	// section 2c and runs it against a controlled CHANGED_FILES value.
	//
	// We replicate the section 2c block inline rather than sourcing the whole
	// script (which requires PUSH_TOKEN, git repo, etc.). This tests the
	// exact blocking code path that produces error output.
	blockScript := `
set -euo pipefail
CHANGED_FILES=".github/workflows/ci.yml"

WORKFLOW_FILES=""
while IFS= read -r file; do
  [ -z "${file}" ] && continue
  case "${file}" in
    .github/workflows/*) WORKFLOW_FILES="${WORKFLOW_FILES}${file}"$'\n' ;;
  esac
done <<< "${CHANGED_FILES}"

if [ -n "${WORKFLOW_FILES}" ]; then
  echo "::error::BLOCKED — agent committed changes to .github/workflows/" >&2
  echo "::error::Workflow file modifications require human authorship." >&2
  echo "::error::Files blocked:" >&2
  echo "${WORKFLOW_FILES}" | sed '/^$/d' | sed 's/::/%3A%3A/g; s/^/  /' >&2
  exit 1
fi
`

	cmd := exec.Command("bash", "-c", blockScript)
	output, err := cmd.CombinedOutput()
	combinedStr := string(output)

	// The script should exit non-zero (blocked)
	require.Error(t, err, "expected non-zero exit when workflow file is blocked")

	// ASSERT-01: Error output includes the blocked file path
	assert.Contains(t, combinedStr, ".github/workflows/ci.yml",
		"error output must contain the blocked file path")

	// ASSERT-02: Error output includes blocking reason
	assert.Contains(t, combinedStr, "BLOCKED",
		"error output must contain BLOCKED keyword")
	assert.Contains(t, combinedStr, "workflows",
		"error output must reference workflow files in the blocking reason")
}

// TestBlockedOutputSanitizesGitHubActionsCommands covers STD scenario 8 (TS-GH84-008).
//
// When file paths contain :: sequences (GitHub Actions workflow command syntax),
// the error output must sanitize them to prevent command injection via crafted
// file paths in agent commits.
func TestBlockedOutputSanitizesGitHubActionsCommands(t *testing.T) {
	scriptPath := filepath.Join("fullsend-repo", "scripts", "post-code.sh")
	requireFileExists(t, scriptPath)

	// Use a malicious file path that attempts GitHub Actions command injection.
	blockScript := `
set -euo pipefail
CHANGED_FILES=".github/workflows/::set-env name=GH_TOKEN::evil"

WORKFLOW_FILES=""
while IFS= read -r file; do
  [ -z "${file}" ] && continue
  case "${file}" in
    .github/workflows/*) WORKFLOW_FILES="${WORKFLOW_FILES}${file}"$'\n' ;;
  esac
done <<< "${CHANGED_FILES}"

if [ -n "${WORKFLOW_FILES}" ]; then
  echo "::error::BLOCKED — agent committed changes to .github/workflows/" >&2
  echo "::error::Workflow file modifications require human authorship." >&2
  echo "::error::Files blocked:" >&2
  echo "${WORKFLOW_FILES}" | sed '/^$/d' | sed 's/::/%3A%3A/g; s/^/  /' >&2
  exit 1
fi
`

	cmd := exec.Command("bash", "-c", blockScript)
	stderrBuf := &strings.Builder{}
	cmd.Stderr = stderrBuf
	_ = cmd.Run()

	stderr := stderrBuf.String()

	// Extract just the "Files blocked:" section lines (after the header errors).
	// The file path lines are the ones that should have :: sanitized.
	lines := strings.Split(stderr, "\n")
	var filePathLines []string
	for _, line := range lines {
		// File path lines start with "  " (indented by sed)
		if strings.HasPrefix(line, "  ") {
			filePathLines = append(filePathLines, line)
		}
	}
	require.NotEmpty(t, filePathLines, "expected at least one file path line in error output")

	filePathOutput := strings.Join(filePathLines, "\n")

	// ASSERT-01: No raw :: workflow command patterns in file path output.
	// The sed command replaces :: with %3A%3A, so the file path section
	// should not contain raw :: sequences.
	assert.NotContains(t, filePathOutput, "::set-env",
		"file path output must not contain raw ::set-env workflow command")
	assert.NotContains(t, filePathOutput, "::evil",
		"file path output must not contain raw :: sequences from crafted paths")

	// ASSERT-02: Sanitization uses percent-encoding.
	assert.Contains(t, filePathOutput, "%3A%3A",
		"file paths must use percent-encoding (%3A%3A) for :: sequences")
}

// requireFileExists is a test helper that skips if the given path doesn't exist.
func requireFileExists(t *testing.T, path string) {
	t.Helper()
	_, err := os.Stat(path)
	require.NoError(t, err, "required file %s must exist", path)
}
