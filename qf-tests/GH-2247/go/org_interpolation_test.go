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

func TestOrgInterpolation(t *testing.T) {
	t.Run("[test_id:TS-GH-2247-020] should match ORG-interpolated expected against deployed content", func(t *testing.T) {
		org := "test-org"
		interpolatedShim := interpolateOrg(shimTemplate, org)
		encodedShim := base64.StdEncoding.EncodeToString([]byte(interpolatedShim))

		// Mock gh: return the interpolated shim for content fetch, report repo as public,
		// return empty PR list, and handle other API calls.
		ghScript := `#!/usr/bin/env bash
LOGFILE="$MOCK_API_LOG"
echo "CALL: $*" >> "$LOGFILE"

# gh api repos/ORG/REPO/contents/PATH -> return matching shim content
if [[ "$*" == *"/contents/"* ]] && [[ "$*" != *"--method"* ]]; then
    echo '{"content":"` + encodedShim + `","sha":"abc123"}'
    exit 0
fi

# gh api repos/ORG/REPO -> repo metadata (public)
if [[ "$*" =~ ^api\ repos/[^/]+/[^/]+$ ]] && [[ "$*" != *"--method"* ]]; then
    echo '{"private":false,"default_branch":"main"}'
    exit 0
fi
if [[ "$*" == *"--jq"* ]] && [[ "$*" == *".private"* ]]; then
    echo "false"
    exit 0
fi
if [[ "$*" == *"--jq"* ]] && [[ "$*" == *".default_branch"* ]]; then
    echo "main"
    exit 0
fi

# gh pr list -> no existing PRs
if [[ "$1" == "pr" ]] && [[ "$2" == "list" ]]; then
    echo ""
    exit 0
fi

# Default: succeed silently
exit 0
`
		mockDir, apiLog, cleanup := setupMockEnv(t, ghScript, defaultYqMock())
		defer cleanup()

		// Create the shim template file in the config directory
		configDir := filepath.Join(mockDir, "configdir")
		templateDir := filepath.Join(configDir, "templates")
		require.NoError(t, os.MkdirAll(templateDir, 0o755))

		// Write config.yaml with one enabled repo
		configYaml := "repos:\n  my-repo:\n    enabled: true\n"
		require.NoError(t, os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configYaml), 0o644))

		// Write the shim template (with __ORG__ placeholder)
		require.NoError(t, os.WriteFile(filepath.Join(templateDir, "shim-workflow-call.yaml"), []byte(shimTemplate), 0o644))

		// Find the reconcile script
		scriptPath, err := filepath.Abs("../internal/scaffold/fullsend-repo/scripts/reconcile-repos.sh")
		if err != nil || !fileExists(scriptPath) {
			// Try alternative paths
			for _, candidate := range []string{
				"internal/scaffold/fullsend-repo/scripts/reconcile-repos.sh",
				"hack/reconcile-repos.sh",
			} {
				abs, _ := filepath.Abs(candidate)
				if fileExists(abs) {
					scriptPath = abs
					break
				}
			}
		}
		require.True(t, fileExists(scriptPath), "reconcile-repos.sh must exist")

		cmd := exec.Command("bash", scriptPath, configDir)
		cmd.Env = append(os.Environ(),
			"GITHUB_REPOSITORY_OWNER="+org,
			"PATH="+mockDir+":"+os.Getenv("PATH"),
			"MOCK_API_LOG="+apiLog,
			"GH_TOKEN=fake-token",
			"GITHUB_SHA=abc123",
		)
		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		// Script should succeed
		assert.NoError(t, err, "reconcile should succeed when shim matches; output: %s", outputStr)

		// No PR should be created -- the content matches
		logBytes, _ := os.ReadFile(apiLog)
		logStr := string(logBytes)
		assert.NotContains(t, logStr, "pr create", "no PR should be created when shim is up to date")

		// Output should indicate repo is already enrolled
		assert.Contains(t, outputStr, "already enrolled", "output should say repo is already enrolled")
	})

	t.Run("[test_id:TS-GH-2247-021] should handle special characters in ORG name", func(t *testing.T) {
		org := "my-org-with-hyphens-123"
		interpolatedShim := interpolateOrg(shimTemplate, org)
		encodedShim := base64.StdEncoding.EncodeToString([]byte(interpolatedShim))

		ghScript := `#!/usr/bin/env bash
LOGFILE="$MOCK_API_LOG"
echo "CALL: $*" >> "$LOGFILE"

if [[ "$*" == *"/contents/"* ]] && [[ "$*" != *"--method"* ]]; then
    echo '{"content":"` + encodedShim + `","sha":"abc123"}'
    exit 0
fi

if [[ "$*" =~ ^api\ repos/[^/]+/[^/]+$ ]] && [[ "$*" != *"--method"* ]]; then
    echo '{"private":false,"default_branch":"main"}'
    exit 0
fi
if [[ "$*" == *"--jq"* ]] && [[ "$*" == *".private"* ]]; then
    echo "false"
    exit 0
fi
if [[ "$*" == *"--jq"* ]] && [[ "$*" == *".default_branch"* ]]; then
    echo "main"
    exit 0
fi

if [[ "$1" == "pr" ]] && [[ "$2" == "list" ]]; then
    echo ""
    exit 0
fi

exit 0
`
		mockDir, apiLog, cleanup := setupMockEnv(t, ghScript, defaultYqMock())
		defer cleanup()

		configDir := filepath.Join(mockDir, "configdir")
		templateDir := filepath.Join(configDir, "templates")
		require.NoError(t, os.MkdirAll(templateDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(configDir, "config.yaml"),
			[]byte("repos:\n  target-repo:\n    enabled: true\n"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(templateDir, "shim-workflow-call.yaml"),
			[]byte(shimTemplate), 0o644))

		scriptPath := findReconcileScript(t)

		cmd := exec.Command("bash", scriptPath, configDir)
		cmd.Env = append(os.Environ(),
			"GITHUB_REPOSITORY_OWNER="+org,
			"PATH="+mockDir+":"+os.Getenv("PATH"),
			"MOCK_API_LOG="+apiLog,
			"GH_TOKEN=fake-token",
			"GITHUB_SHA=abc123",
		)
		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		// Should succeed without sed/regex errors
		assert.NoError(t, err, "reconcile should handle special chars in org; output: %s", outputStr)

		// No errors in output
		assert.NotContains(t, outputStr, "::error::", "no errors expected for org with hyphens and numbers")

		// No PR should be created (content matches)
		logBytes, _ := os.ReadFile(apiLog)
		logStr := string(logBytes)
		assert.NotContains(t, logStr, "pr create", "no PR should be created when content matches")
	})

	t.Run("[test_id:TS-GH-2247-022] should error when ORG environment variable is unset", func(t *testing.T) {
		ghScript := `#!/usr/bin/env bash
echo "CALL: $*" >> "$MOCK_API_LOG"
exit 0
`
		mockDir, apiLog, cleanup := setupMockEnv(t, ghScript, defaultYqMock())
		defer cleanup()

		configDir := filepath.Join(mockDir, "configdir")
		templateDir := filepath.Join(configDir, "templates")
		require.NoError(t, os.MkdirAll(templateDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(configDir, "config.yaml"),
			[]byte("repos:\n  some-repo:\n    enabled: true\n"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(templateDir, "shim-workflow-call.yaml"),
			[]byte(shimTemplate), 0o644))

		scriptPath := findReconcileScript(t)

		cmd := exec.Command("bash", scriptPath, configDir)
		// Build env WITHOUT GITHUB_REPOSITORY_OWNER
		filteredEnv := []string{}
		for _, e := range os.Environ() {
			if !strings.HasPrefix(e, "GITHUB_REPOSITORY_OWNER=") {
				filteredEnv = append(filteredEnv, e)
			}
		}
		cmd.Env = append(filteredEnv,
			"PATH="+mockDir+":"+os.Getenv("PATH"),
			"MOCK_API_LOG="+apiLog,
			"GH_TOKEN=fake-token",
			"GITHUB_SHA=abc123",
		)
		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		// Script should fail with non-zero exit code
		assert.Error(t, err, "reconcile should fail when GITHUB_REPOSITORY_OWNER is unset")

		// Error message should mention the missing variable
		assert.True(t,
			strings.Contains(outputStr, "GITHUB_REPOSITORY_OWNER") ||
				strings.Contains(outputStr, "must be set"),
			"error output should mention GITHUB_REPOSITORY_OWNER; got: %s", outputStr)
	})
}

// fileExists returns true if the path exists and is a regular file.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// findReconcileScript locates the reconcile-repos.sh script from known paths.
func findReconcileScript(t *testing.T) string {
	t.Helper()
	candidates := []string{
		"internal/scaffold/fullsend-repo/scripts/reconcile-repos.sh",
		"hack/reconcile-repos.sh",
		"../internal/scaffold/fullsend-repo/scripts/reconcile-repos.sh",
	}
	for _, c := range candidates {
		abs, err := filepath.Abs(c)
		if err == nil && fileExists(abs) {
			return abs
		}
	}
	t.Fatal("reconcile-repos.sh not found in any known location")
	return ""
}
