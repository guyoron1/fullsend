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

// threeRepoYqMock returns a yq mock script that outputs three enrolled repos.
func threeRepoYqMock() string {
	return `#!/bin/bash
# Mock yq - returns 3 enrolled repos
if echo "$@" | grep -q "enabled == true"; then
  echo "repo-alpha"
  echo "repo-beta"
  echo "repo-gamma"
else
  echo ""
fi
`
}

// ghMockUpToDate returns a gh mock script that serves up-to-date (current)
// shim content for any repo queried. All API calls are logged to the file
// at MOCK_API_LOG.
func ghMockUpToDate(contentB64 string) string {
	return `#!/bin/bash
echo "$@" >> "${MOCK_API_LOG}"
case "$*" in
  *"contents/.github/workflows/fullsend.yaml"*)
    echo '{"content":"` + contentB64 + `","sha":"uptodate"}'
    ;;
  *"--jq .private"*|*"--jq '.private'"*)
    echo "false"
    ;;
  *"--jq .default_branch"*)
    echo "main"
    ;;
  *"api repos/"*)
    if echo "$*" | grep -q "private"; then
      echo "false"
    elif echo "$*" | grep -q "default_branch"; then
      echo "main"
    else
      echo '{"private":false,"default_branch":"main"}'
    fi
    ;;
  *"pr list"*)
    echo "[]"
    ;;
  *"pr close"*|*"--method DELETE"*)
    ;;
  *)
    echo "{}"
    ;;
esac
`
}

func TestNoDriftPRSuppression(t *testing.T) {
	t.Run("[test_id:TS-GH-2247-017] should not create PR for up-to-date enrolled repo", func(t *testing.T) {
		org := "test-org"
		currentContent := interpolateOrg(shimTemplate, org)
		currentB64 := base64.StdEncoding.EncodeToString([]byte(currentContent))

		mockDir, apiLog, cleanup := setupMockEnv(t, ghMockUpToDate(currentB64), defaultYqMock())
		defer cleanup()

		scriptPath := os.Getenv("RECONCILE_REPO_ROOT")
		if scriptPath != "" {
			scriptPath = filepath.Join(scriptPath, "scripts", "reconcile-repos.sh")
		} else {
			scriptPath, _ = filepath.Abs(filepath.Join("hack", "reconcile-repos.sh"))
		}

		configDir := filepath.Join(mockDir, "config")
		require.NoError(t, os.MkdirAll(filepath.Join(configDir, "templates"), 0o755))

		templateContent := interpolateOrg(shimTemplate, "__ORG__")
		require.NoError(t, os.WriteFile(
			filepath.Join(configDir, "templates", "shim-workflow-call.yaml"),
			[]byte(templateContent), 0o644,
		))

		configYAML := `repos:
  sample-repo:
    enabled: true
`
		require.NoError(t, os.WriteFile(
			filepath.Join(configDir, "config.yaml"),
			[]byte(configYAML), 0o644,
		))

		cmd := exec.Command("bash", scriptPath, configDir)
		cmd.Env = append(os.Environ(),
			"PATH="+mockDir+":"+os.Getenv("PATH"),
			"GITHUB_REPOSITORY_OWNER="+org,
			"GITHUB_SHA=test-sha-017",
			"MOCK_API_LOG="+apiLog,
		)
		output, err := cmd.CombinedOutput()
		outputStr := string(output)
		t.Logf("Script output:\n%s", outputStr)

		assert.NoError(t, err, "script should exit cleanly for up-to-date repo")

		logData, readErr := os.ReadFile(apiLog)
		require.NoError(t, readErr)
		logStr := string(logData)

		// No PR creation API call should appear
		assert.NotContains(t, logStr, "pr create",
			"no PR should be created for an up-to-date enrolled repo")

		// Script should indicate repo is already enrolled
		assert.True(t,
			strings.Contains(outputStr, "already enrolled") || strings.Contains(outputStr, "up to date"),
			"output should confirm repo is already enrolled / up to date",
		)
	})

	t.Run("[test_id:TS-GH-2247-018] should increment skip count for up-to-date repos", func(t *testing.T) {
		org := "test-org"
		currentContent := interpolateOrg(shimTemplate, org)
		currentB64 := base64.StdEncoding.EncodeToString([]byte(currentContent))

		// Use the three-repo yq mock
		mockDir, apiLog, cleanup := setupMockEnv(t, ghMockUpToDate(currentB64), threeRepoYqMock())
		defer cleanup()

		scriptPath := os.Getenv("RECONCILE_REPO_ROOT")
		if scriptPath != "" {
			scriptPath = filepath.Join(scriptPath, "scripts", "reconcile-repos.sh")
		} else {
			scriptPath, _ = filepath.Abs(filepath.Join("hack", "reconcile-repos.sh"))
		}

		configDir := filepath.Join(mockDir, "config")
		require.NoError(t, os.MkdirAll(filepath.Join(configDir, "templates"), 0o755))

		templateContent := interpolateOrg(shimTemplate, "__ORG__")
		require.NoError(t, os.WriteFile(
			filepath.Join(configDir, "templates", "shim-workflow-call.yaml"),
			[]byte(templateContent), 0o644,
		))

		// Config with 3 enabled repos matching the yq mock output
		configYAML := `repos:
  repo-alpha:
    enabled: true
  repo-beta:
    enabled: true
  repo-gamma:
    enabled: true
`
		require.NoError(t, os.WriteFile(
			filepath.Join(configDir, "config.yaml"),
			[]byte(configYAML), 0o644,
		))

		cmd := exec.Command("bash", scriptPath, configDir)
		cmd.Env = append(os.Environ(),
			"PATH="+mockDir+":"+os.Getenv("PATH"),
			"GITHUB_REPOSITORY_OWNER="+org,
			"GITHUB_SHA=test-sha-018",
			"MOCK_API_LOG="+apiLog,
		)
		output, err := cmd.CombinedOutput()
		outputStr := string(output)
		t.Logf("Script output:\n%s", outputStr)

		assert.NoError(t, err, "script should succeed when all repos are up to date")

		// The summary line should show Skipped count of 3
		// The script outputs: "Skipped (already reconciled): N"
		assert.Contains(t, outputStr, "Skipped (already reconciled): 3",
			"skip count should be 3 for 3 up-to-date repos")

		// Updated count should be 0
		assert.Contains(t, outputStr, "Updated (stale shim): 0",
			"no repos should be updated")

		// Enrolled count should be 0
		assert.Contains(t, outputStr, "Enrolled: 0",
			"no new enrollments should occur")
	})

	t.Run("[test_id:TS-GH-2247-019] should make no blob API call for identical content", func(t *testing.T) {
		org := "test-org"
		currentContent := interpolateOrg(shimTemplate, org)
		currentB64 := base64.StdEncoding.EncodeToString([]byte(currentContent))

		mockDir, apiLog, cleanup := setupMockEnv(t, ghMockUpToDate(currentB64), defaultYqMock())
		defer cleanup()

		scriptPath := os.Getenv("RECONCILE_REPO_ROOT")
		if scriptPath != "" {
			scriptPath = filepath.Join(scriptPath, "scripts", "reconcile-repos.sh")
		} else {
			scriptPath, _ = filepath.Abs(filepath.Join("hack", "reconcile-repos.sh"))
		}

		configDir := filepath.Join(mockDir, "config")
		require.NoError(t, os.MkdirAll(filepath.Join(configDir, "templates"), 0o755))

		templateContent := interpolateOrg(shimTemplate, "__ORG__")
		require.NoError(t, os.WriteFile(
			filepath.Join(configDir, "templates", "shim-workflow-call.yaml"),
			[]byte(templateContent), 0o644,
		))

		configYAML := `repos:
  sample-repo:
    enabled: true
`
		require.NoError(t, os.WriteFile(
			filepath.Join(configDir, "config.yaml"),
			[]byte(configYAML), 0o644,
		))

		cmd := exec.Command("bash", scriptPath, configDir)
		cmd.Env = append(os.Environ(),
			"PATH="+mockDir+":"+os.Getenv("PATH"),
			"GITHUB_REPOSITORY_OWNER="+org,
			"GITHUB_SHA=test-sha-019",
			"MOCK_API_LOG="+apiLog,
		)
		output, err := cmd.CombinedOutput()
		outputStr := string(output)
		t.Logf("Script output:\n%s", outputStr)

		assert.NoError(t, err, "script should exit cleanly for identical content")

		logData, readErr := os.ReadFile(apiLog)
		require.NoError(t, readErr)
		logStr := string(logData)

		// No blob, tree, or PR creation API calls should be made
		assert.NotContains(t, logStr, "git/blobs",
			"no git/blobs API call should be made for identical content")
		assert.NotContains(t, logStr, "git/trees",
			"no git/trees API call should be made for identical content")

		// Check that no PR-related write calls were made
		// The mock logs all args, so "pr create" would appear if called
		assert.NotContains(t, logStr, "pr create",
			"no PR creation call should be made for identical content")

		// Also verify no POST to pulls endpoint via the API
		for _, line := range strings.Split(logStr, "\n") {
			if strings.Contains(line, "pulls") && strings.Contains(line, "POST") {
				t.Errorf("unexpected POST to pulls endpoint: %s", line)
			}
		}
	})
}
