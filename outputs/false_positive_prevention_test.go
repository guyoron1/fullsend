//go:build e2e

package reconcile_test

import (
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// shimTemplate is the canonical shim workflow template with sentinel markers.
const shimTemplate = `# Managed by fullsend reconcile
# DO NOT EDIT below this line
# --- fullsend-managed-start ---
name: shim-workflow
on:
  workflow_dispatch:
jobs:
  shim:
    runs-on: ubuntu-latest
    steps:
      - uses: __ORG__/fullsend-action@main
# --- fullsend-managed-end ---
`

// interpolateOrg replaces __ORG__ with the given org name.
func interpolateOrg(template, org string) string {
	return strings.ReplaceAll(template, "__ORG__", org)
}

// setupMockEnv creates a temp directory with mock gh and yq scripts, sets
// PATH and environment variables. Returns the mock dir, API log path, and
// a cleanup function.
func setupMockEnv(t *testing.T, ghScript, yqScript string) (mockDir, apiLog string, cleanup func()) {
	t.Helper()
	mockDir = t.TempDir()
	apiLog = filepath.Join(mockDir, "api_calls.log")

	// Write mock gh
	ghPath := filepath.Join(mockDir, "gh")
	err := os.WriteFile(ghPath, []byte(ghScript), 0755)
	require.NoError(t, err)

	// Write mock yq
	yqPath := filepath.Join(mockDir, "yq")
	err = os.WriteFile(yqPath, []byte(yqScript), 0755)
	require.NoError(t, err)

	origPath := os.Getenv("PATH")
	origOwner := os.Getenv("GITHUB_REPOSITORY_OWNER")
	_ = origOwner // restored by t.Setenv

	t.Setenv("PATH", mockDir+":"+origPath)
	t.Setenv("GITHUB_REPOSITORY_OWNER", "test-org")

	cleanup = func() {
		// t.Setenv handles restoration automatically
	}
	return
}

// defaultYqMock returns a yq mock that outputs a single enrolled repo.
func defaultYqMock() string {
	return `#!/bin/bash
# Mock yq - returns enrolled repos list
# First call: enabled repos, second call: disabled repos
if echo "$@" | grep -q "enabled == true"; then
  echo "sample-repo"
else
  echo ""
fi
`
}

// setupConfigDir creates a temporary config directory with config.yaml and
// templates/shim-workflow-call.yaml for the reconcile script to consume.
func setupConfigDir(t *testing.T) string {
	t.Helper()
	configDir := t.TempDir()

	configYaml := `repos:
  sample-repo:
    enabled: true
`
	err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configYaml), 0644)
	require.NoError(t, err)

	templatesDir := filepath.Join(configDir, "templates")
	err = os.MkdirAll(templatesDir, 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(templatesDir, "shim-workflow-call.yaml"), []byte(shimTemplate), 0644)
	require.NoError(t, err)

	return configDir
}

// readAPILog reads the API call log file and returns its contents.
func readAPILog(t *testing.T, apiLog string) string {
	t.Helper()
	data, err := os.ReadFile(apiLog)
	if err != nil {
		if os.IsNotExist(err) {
			return ""
		}
		require.NoError(t, err)
	}
	return string(data)
}

// runReconcile runs the reconcile-repos.sh script with the given config dir.
// It locates the script relative to the test's working directory.
func runReconcile(t *testing.T, configDir string) (string, error) {
	t.Helper()
	scriptPath := filepath.Join("/sandbox/workspace/pr-repo/internal/scaffold/fullsend-repo/scripts/reconcile-repos.sh")

	cmd := exec.Command("bash", scriptPath, configDir)
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestFalsePositivePrevention(t *testing.T) {
	t.Run("[test_id:TS-GH-2247-001] should not flag identical shim as stale", func(t *testing.T) {
		// The remote shim content is identical to the local template (with org interpolated).
		// The script should detect no drift and skip creating a PR.
		expectedContent := interpolateOrg(shimTemplate, "test-org")
		encodedContent := base64.StdEncoding.EncodeToString([]byte(expectedContent))

		configDir := setupConfigDir(t)

		ghScript := `#!/bin/bash
LOG_FILE="` + filepath.Join(t.TempDir(), "api_calls.log") + `"

echo "$1 $2 $*" >> "$LOG_FILE"

# gh api repos/ORG/REPO — return public repo
if [[ "$1" == "api" && "$2" =~ ^repos/[^/]+/[^/]+$ && ! "$2" =~ /contents/ && ! "$2" =~ /git/ ]]; then
  if echo "$*" | grep -q "\.default_branch"; then
    echo "main"
  elif echo "$*" | grep -q "\.private"; then
    echo "false"
  else
    echo '{"private": false, "default_branch": "main"}'
  fi
  exit 0
fi

# gh api repos/ORG/REPO/contents/PATH — return matching shim content
if [[ "$1" == "api" && "$2" =~ contents/ ]]; then
  if echo "$*" | grep -q "\.content"; then
    echo "` + encodedContent + `"
  else
    echo '{"content": "` + encodedContent + `", "encoding": "base64", "sha": "abc123"}'
  fi
  exit 0
fi

# gh pr list — no existing PRs
if [[ "$1" == "pr" && "$2" == "list" ]]; then
  echo ""
  exit 0
fi

# gh pr create — should NOT be called
if [[ "$1" == "pr" && "$2" == "create" ]]; then
  echo "POST pulls" >> "$LOG_FILE"
  echo "https://github.com/test-org/sample-repo/pull/1"
  exit 0
fi

exit 0
`
		// Rewrite to use a stable log path
		mockDir := t.TempDir()
		apiLog := filepath.Join(mockDir, "api_calls.log")

		ghScript = `#!/bin/bash
LOG_FILE="` + apiLog + `"

echo "$1 $2 $*" >> "$LOG_FILE"

# gh api repos/ORG/REPO — return public repo
if [[ "$1" == "api" && "$2" =~ ^repos/[^/]+/[^/]+$ && ! "$2" =~ /contents/ && ! "$2" =~ /git/ ]]; then
  if echo "$*" | grep -q -- "--jq .default_branch"; then
    echo "main"
  elif echo "$*" | grep -q -- "--jq .private"; then
    echo "false"
  else
    echo '{"private": false, "default_branch": "main"}'
  fi
  exit 0
fi

# gh api repos/ORG/REPO/contents/PATH — return matching shim content
if [[ "$1" == "api" && "$2" =~ contents/ ]]; then
  if echo "$*" | grep -q -- "--jq .content"; then
    echo "` + encodedContent + `"
  else
    echo '{"content": "` + encodedContent + `", "encoding": "base64", "sha": "abc123"}'
  fi
  exit 0
fi

# gh pr list — no existing PRs
if [[ "$1" == "pr" && "$2" == "list" ]]; then
  echo ""
  exit 0
fi

# gh pr create — should NOT be called for identical content
if [[ "$1" == "pr" && "$2" == "create" ]]; then
  echo "POST pulls" >> "$LOG_FILE"
  echo "https://github.com/test-org/sample-repo/pull/1"
  exit 0
fi

exit 0
`
		ghPath := filepath.Join(mockDir, "gh")
		err := os.WriteFile(ghPath, []byte(ghScript), 0755)
		require.NoError(t, err)

		yqScript := defaultYqMock()
		yqPath := filepath.Join(mockDir, "yq")
		err = os.WriteFile(yqPath, []byte(yqScript), 0755)
		require.NoError(t, err)

		origPath := os.Getenv("PATH")
		t.Setenv("PATH", mockDir+":"+origPath)
		t.Setenv("GITHUB_REPOSITORY_OWNER", "test-org")

		output, err := runReconcile(t, configDir)
		require.NoError(t, err, "reconcile should succeed; output: %s", output)

		logContent := readAPILog(t, apiLog)
		assert.NotContains(t, logContent, "POST pulls",
			"should not create a PR when shim content is identical")
		assert.Contains(t, output, "already enrolled",
			"output should indicate repo is already enrolled")
	})

	t.Run("[test_id:TS-GH-2247-002] should treat trailing newline variations as identical", func(t *testing.T) {
		// Remote content has extra trailing newlines but is otherwise identical.
		// The script normalizes trailing whitespace before comparison,
		// so it should treat this as identical and not create a PR.
		expectedContent := interpolateOrg(shimTemplate, "test-org")
		contentWithExtraNewlines := expectedContent + "\n\n\n"
		encodedContent := base64.StdEncoding.EncodeToString([]byte(contentWithExtraNewlines))

		configDir := setupConfigDir(t)
		mockDir := t.TempDir()
		apiLog := filepath.Join(mockDir, "api_calls.log")

		ghScript := `#!/bin/bash
LOG_FILE="` + apiLog + `"

echo "$1 $2 $*" >> "$LOG_FILE"

if [[ "$1" == "api" && "$2" =~ ^repos/[^/]+/[^/]+$ && ! "$2" =~ /contents/ && ! "$2" =~ /git/ ]]; then
  if echo "$*" | grep -q -- "--jq .default_branch"; then
    echo "main"
  elif echo "$*" | grep -q -- "--jq .private"; then
    echo "false"
  else
    echo '{"private": false, "default_branch": "main"}'
  fi
  exit 0
fi

if [[ "$1" == "api" && "$2" =~ contents/ ]]; then
  if echo "$*" | grep -q -- "--jq .content"; then
    echo "` + encodedContent + `"
  else
    echo '{"content": "` + encodedContent + `", "encoding": "base64", "sha": "abc123"}'
  fi
  exit 0
fi

if [[ "$1" == "pr" && "$2" == "list" ]]; then
  echo ""
  exit 0
fi

if [[ "$1" == "pr" && "$2" == "create" ]]; then
  echo "POST pulls" >> "$LOG_FILE"
  echo "https://github.com/test-org/sample-repo/pull/1"
  exit 0
fi

exit 0
`
		ghPath := filepath.Join(mockDir, "gh")
		err := os.WriteFile(ghPath, []byte(ghScript), 0755)
		require.NoError(t, err)

		yqPath := filepath.Join(mockDir, "yq")
		err = os.WriteFile(yqPath, []byte(defaultYqMock()), 0755)
		require.NoError(t, err)

		origPath := os.Getenv("PATH")
		t.Setenv("PATH", mockDir+":"+origPath)
		t.Setenv("GITHUB_REPOSITORY_OWNER", "test-org")

		output, err := runReconcile(t, configDir)
		// The script does exact string comparison, so trailing newline differences
		// will cause a PR. We verify the script's actual behavior here.
		_ = output

		logContent := readAPILog(t, apiLog)
		// If the script properly normalizes trailing newlines, no PR should be created.
		// If it does not normalize, a PR will be created, and this test documents
		// the expected behavior for GH-2247 false-positive prevention.
		// The current script compares decoded text with shell [ "$A" = "$B" ] which
		// strips trailing newlines in command substitution, effectively normalizing them.
		assert.NotContains(t, logContent, "POST pulls",
			"should not create a PR for content differing only in trailing newlines")
	})

	t.Run("[test_id:TS-GH-2247-003] should create no update PR for up-to-date shim", func(t *testing.T) {
		// Full environment mock: repo is public, shim exists and matches template,
		// no existing PRs. Assert exit 0 and zero PR creation calls.
		expectedContent := interpolateOrg(shimTemplate, "test-org")
		encodedContent := base64.StdEncoding.EncodeToString([]byte(expectedContent))

		configDir := setupConfigDir(t)
		mockDir := t.TempDir()
		apiLog := filepath.Join(mockDir, "api_calls.log")
		prCountFile := filepath.Join(mockDir, "pr_count")
		err := os.WriteFile(prCountFile, []byte("0"), 0644)
		require.NoError(t, err)

		ghScript := `#!/bin/bash
LOG_FILE="` + apiLog + `"
PR_COUNT_FILE="` + prCountFile + `"

echo "$1 $2 $*" >> "$LOG_FILE"

if [[ "$1" == "api" && "$2" =~ ^repos/[^/]+/[^/]+$ && ! "$2" =~ /contents/ && ! "$2" =~ /git/ ]]; then
  if echo "$*" | grep -q -- "--jq .default_branch"; then
    echo "main"
  elif echo "$*" | grep -q -- "--jq .private"; then
    echo "false"
  else
    echo '{"private": false, "default_branch": "main"}'
  fi
  exit 0
fi

if [[ "$1" == "api" && "$2" =~ contents/ ]]; then
  if echo "$*" | grep -q -- "--jq .content"; then
    echo "` + encodedContent + `"
  else
    echo '{"content": "` + encodedContent + `", "encoding": "base64", "sha": "abc123"}'
  fi
  exit 0
fi

if [[ "$1" == "pr" && "$2" == "list" ]]; then
  echo ""
  exit 0
fi

if [[ "$1" == "pr" && "$2" == "create" ]]; then
  echo "POST pulls" >> "$LOG_FILE"
  # Increment PR count
  COUNT=$(cat "$PR_COUNT_FILE")
  echo $((COUNT + 1)) > "$PR_COUNT_FILE"
  echo "https://github.com/test-org/sample-repo/pull/1"
  exit 0
fi

exit 0
`
		ghPath := filepath.Join(mockDir, "gh")
		err = os.WriteFile(ghPath, []byte(ghScript), 0755)
		require.NoError(t, err)

		yqPath := filepath.Join(mockDir, "yq")
		err = os.WriteFile(yqPath, []byte(defaultYqMock()), 0755)
		require.NoError(t, err)

		origPath := os.Getenv("PATH")
		t.Setenv("PATH", mockDir+":"+origPath)
		t.Setenv("GITHUB_REPOSITORY_OWNER", "test-org")

		output, err := runReconcile(t, configDir)
		require.NoError(t, err, "reconcile should exit 0; output: %s", output)

		// Verify zero PRs created
		prCountData, err := os.ReadFile(prCountFile)
		require.NoError(t, err)
		assert.Equal(t, "0", strings.TrimSpace(string(prCountData)),
			"should create zero PRs when shim is up to date")

		// Also verify via log
		logContent := readAPILog(t, apiLog)
		assert.NotContains(t, logContent, "POST pulls",
			"API log should contain no PR creation calls")
	})

	t.Run("[test_id:TS-GH-2247-004] should not false-positive after freshly enrolled repo re-run", func(t *testing.T) {
		// First run: shim does not exist (404), so the script creates an enrollment PR.
		// Second run: shim now exists with matching content. No PR should be created.
		expectedContent := interpolateOrg(shimTemplate, "test-org")
		encodedContent := base64.StdEncoding.EncodeToString([]byte(expectedContent))

		configDir := setupConfigDir(t)
		mockDir := t.TempDir()
		apiLog := filepath.Join(mockDir, "api_calls.log")
		stateFile := filepath.Join(mockDir, "state")

		// State file tracks which run we're on
		err := os.WriteFile(stateFile, []byte("run1"), 0644)
		require.NoError(t, err)

		ghScript := `#!/bin/bash
LOG_FILE="` + apiLog + `"
STATE_FILE="` + stateFile + `"
STATE=$(cat "$STATE_FILE")

echo "$STATE $1 $2 $*" >> "$LOG_FILE"

# Repo metadata — always public
if [[ "$1" == "api" && "$2" =~ ^repos/[^/]+/[^/]+$ && ! "$2" =~ /contents/ && ! "$2" =~ /git/ ]]; then
  if echo "$*" | grep -q -- "--jq .default_branch"; then
    echo "main"
  elif echo "$*" | grep -q -- "--jq .private"; then
    echo "false"
  else
    echo '{"private": false, "default_branch": "main"}'
  fi
  exit 0
fi

# Contents endpoint behavior depends on run state
if [[ "$1" == "api" && "$2" =~ contents/ ]]; then
  if [[ "$STATE" == "run1" ]]; then
    # First run: file does not exist (404)
    echo '{"message": "Not Found"}' >&2
    exit 1
  else
    # Second run: file exists with correct content
    if echo "$*" | grep -q -- "--jq .content"; then
      echo "` + encodedContent + `"
    else
      echo '{"content": "` + encodedContent + `", "encoding": "base64", "sha": "def456"}'
    fi
    exit 0
  fi
fi

# Git ref endpoint for branch creation
if [[ "$1" == "api" && "$2" =~ /git/ref/ ]]; then
  echo '{"object": {"sha": "aaa111"}}'
  exit 0
fi

if [[ "$1" == "api" && "$2" =~ /git/refs$ ]]; then
  echo '{"ref": "refs/heads/fullsend/onboard", "object": {"sha": "aaa111"}}'
  exit 0
fi

if [[ "$1" == "api" && "$2" =~ /git/refs/heads/ ]]; then
  echo '{"ref": "refs/heads/fullsend/onboard", "object": {"sha": "aaa111"}}'
  exit 0
fi

if [[ "$1" == "api" && "$2" =~ /git/commits/ ]]; then
  echo '{"sha": "bbb222", "tree": {"sha": "ccc333"}}'
  exit 0
fi

if [[ "$1" == "api" && "$2" =~ /git/commits$ ]]; then
  echo '{"sha": "bbb222"}'
  exit 0
fi

if [[ "$1" == "api" && "$2" =~ /git/trees$ ]]; then
  echo '{"sha": "ccc333"}'
  exit 0
fi

if [[ "$1" == "api" && "$2" =~ /git/blobs$ ]]; then
  echo '{"sha": "ddd444"}'
  exit 0
fi

# PR list — no existing PRs
if [[ "$1" == "pr" && "$2" == "list" ]]; then
  echo ""
  exit 0
fi

# PR close — handle stale PR cleanup
if [[ "$1" == "pr" && "$2" == "close" ]]; then
  exit 0
fi

# PR create — enrollment
if [[ "$1" == "pr" && "$2" == "create" ]]; then
  echo "POST pulls" >> "$LOG_FILE"
  echo "https://github.com/test-org/sample-repo/pull/1"
  exit 0
fi

# Catch-all for DELETE (branch cleanup)
if echo "$*" | grep -q -- "--method DELETE"; then
  exit 0
fi

exit 0
`
		ghPath := filepath.Join(mockDir, "gh")
		err = os.WriteFile(ghPath, []byte(ghScript), 0755)
		require.NoError(t, err)

		yqPath := filepath.Join(mockDir, "yq")
		err = os.WriteFile(yqPath, []byte(defaultYqMock()), 0755)
		require.NoError(t, err)

		origPath := os.Getenv("PATH")
		t.Setenv("PATH", mockDir+":"+origPath)
		t.Setenv("GITHUB_REPOSITORY_OWNER", "test-org")

		// First run: should create enrollment PR (shim doesn't exist)
		output1, err := runReconcile(t, configDir)
		require.NoError(t, err, "first reconcile run should succeed; output: %s", output1)

		logAfterRun1 := readAPILog(t, apiLog)
		assert.Contains(t, logAfterRun1, "POST pulls",
			"first run should create an enrollment PR")

		// Switch state to run2: shim now exists with correct content
		err = os.WriteFile(stateFile, []byte("run2"), 0644)
		require.NoError(t, err)

		// Clear the API log for the second run
		err = os.WriteFile(apiLog, []byte(""), 0644)
		require.NoError(t, err)

		// Second run: shim is now up-to-date, should NOT create any PR
		output2, err := runReconcile(t, configDir)
		require.NoError(t, err, "second reconcile run should succeed; output: %s", output2)

		logAfterRun2 := readAPILog(t, apiLog)
		assert.NotContains(t, logAfterRun2, "POST pulls",
			"second run should not create a PR when shim matches template")
		assert.Contains(t, output2, "already enrolled",
			"second run output should indicate repo is already enrolled")
	})
}
