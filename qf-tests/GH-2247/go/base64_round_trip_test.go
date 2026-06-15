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

/*
Shim Drift Detection -- Base64 Encoding/Decoding Round-Trip Tests

STP Reference: outputs/stp/GH-2247/GH-2247_test_plan.md
Jira: GH-2247

These tests verify that the decoded text comparison handles trailing
newline variations, carriage returns, and empty content gracefully.
*/

// buildGHMockForContent returns a gh mock script that serves the given
// base64-encoded content as the remote shim file. It logs all API calls.
// When the remote content matches the expected shim, no PR should be created.
func buildGHMockForContent(remoteB64 string) string {
	return `#!/usr/bin/env bash
set -euo pipefail
LOG_FILE="$MOCK_API_LOG"
printf 'gh' >> "$LOG_FILE"
for arg in "$@"; do printf ' %q' "$arg" >> "$LOG_FILE"; done
printf '\n' >> "$LOG_FILE"

if [[ "$1" == "pr" && "$2" == "list" ]]; then
  echo ""
  exit 0
fi
if [[ "$1" == "pr" && "$2" == "create" ]]; then
  echo "PR_CREATED" >> "$LOG_FILE"
  echo "https://github.com/test-org/test-repo/pull/99"
  exit 0
fi
if [[ "$1" != "api" ]]; then
  exit 0
fi

# Handle --input - (consume stdin to avoid SIGPIPE)
for arg in "$@"; do
  if [[ "$arg" == "--input" ]]; then
    cat > /dev/null 2>&1 || true
    break
  fi
done

endpoint="$2"
case "$endpoint" in
  repos/test-org/test-repo)
    echo "false"
    ;;
  repos/test-org/test-repo/contents/.github/workflows/fullsend.yaml)
    # Return the remote content as base64 (GitHub API returns content field)
    echo '` + remoteB64 + `'
    ;;
  repos/test-org/test-repo/git/ref/heads/main)
    echo "abc123"
    ;;
  repos/test-org/test-repo/git/commits/abc123)
    echo "tree-base-sha"
    ;;
  repos/test-org/test-repo/git/blobs)
    echo "blob-sha"
    ;;
  repos/test-org/test-repo/git/trees)
    echo "tree-sha"
    ;;
  repos/test-org/test-repo/git/commits)
    echo "commit-sha"
    ;;
  repos/test-org/test-repo/git/refs)
    echo '{"ref":"refs/heads/fullsend/onboard"}'
    ;;
  repos/test-org/test-repo/git/refs/heads/fullsend/onboard)
    exit 0
    ;;
  repos/test-org/test-repo/git/refs/heads/fullsend/offboard)
    exit 0
    ;;
  *)
    echo "mock-ok"
    ;;
esac
`
}

// runReconcileWithTemplate executes reconcile-repos.sh with the given mock env and template
// content. Returns combined output, the API log contents, and any error.
func runReconcileWithTemplate(t *testing.T, scriptPath, mockDir, apiLog, templateContent string) (string, string, error) {
	t.Helper()

	configDir := filepath.Join(mockDir, "config")
	require.NoError(t, os.MkdirAll(configDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(configDir, "config.yaml"),
		[]byte("version: 1\nrepos:\n  test-repo:\n    enabled: true\n"),
		0o644,
	))

	templateDir := filepath.Join(configDir, "templates")
	require.NoError(t, os.MkdirAll(templateDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(templateDir, "shim-workflow-call.yaml"),
		[]byte(templateContent),
		0o644,
	))

	cmd := exec.Command("bash", scriptPath, configDir)
	cmd.Env = append(os.Environ(),
		"PATH="+mockDir+":"+os.Getenv("PATH"),
		"GITHUB_REPOSITORY_OWNER=test-org",
		"GITHUB_SHA=test-sha-roundtrip",
		"GH_TOKEN=fake-token",
		"MOCK_API_LOG="+apiLog,
	)
	out, err := cmd.CombinedOutput()

	logBytes, readErr := os.ReadFile(apiLog)
	require.NoError(t, readErr)

	return string(out), string(logBytes), err
}

func TestBase64RoundTrip(t *testing.T) {
	scriptPath, err := filepath.Abs("../../internal/scaffold/fullsend-repo/scripts/reconcile-repos.sh")
	require.NoError(t, err)

	t.Run("[test_id:TS-GH-2247-011] should match decoded content despite trailing newline differences", func(t *testing.T) {
		// The reconcile script compares decoded text. Trailing newline
		// variations should be treated as equivalent (no PR created).
		templateContent := interpolateOrg(shimTemplate, "test-org")

		// Three variants: 0, 1, 2 trailing newlines
		variants := []string{
			strings.TrimRight(templateContent, "\n"),
			strings.TrimRight(templateContent, "\n") + "\n",
			strings.TrimRight(templateContent, "\n") + "\n\n",
		}

		for i, remoteVariant := range variants {
			// Encode the variant as base64 (simulating what GitHub API returns)
			remoteB64 := base64.StdEncoding.EncodeToString([]byte(remoteVariant))

			ghScript := buildGHMockForContent(remoteB64)
			mockDir, apiLog, cleanup := setupMockEnv(t, ghScript, defaultYqMock())

			_, logStr, _ := runReconcileWithTemplate(t, scriptPath, mockDir, apiLog, shimTemplate)

			assert.NotContains(t, logStr, "PR_CREATED",
				"variant %d (trailing newlines): should not create PR for content that differs only in trailing newlines", i)

			cleanup()
		}
	})

	t.Run("[test_id:TS-GH-2247-012] should be resilient to carriage return in remote content", func(t *testing.T) {
		// Mock returns content with CRLF line endings. The script should
		// normalize with tr -d '\r' before comparison.
		templateContent := interpolateOrg(shimTemplate, "test-org")

		// Convert LF to CRLF
		crlfContent := strings.ReplaceAll(templateContent, "\n", "\r\n")
		remoteB64 := base64.StdEncoding.EncodeToString([]byte(crlfContent))

		ghScript := buildGHMockForContent(remoteB64)
		mockDir, apiLog, cleanup := setupMockEnv(t, ghScript, defaultYqMock())
		defer cleanup()

		_, logStr, _ := runReconcileWithTemplate(t, scriptPath, mockDir, apiLog, shimTemplate)

		assert.NotContains(t, logStr, "PR_CREATED",
			"CRLF remote content should not trigger false drift detection")
	})

	t.Run("[test_id:TS-GH-2247-013] should handle empty content gracefully", func(t *testing.T) {
		// Mock returns base64 of empty string. Script should not crash and
		// should detect the empty content as stale (create an update PR).
		emptyB64 := base64.StdEncoding.EncodeToString([]byte(""))

		ghScript := buildGHMockForContent(emptyB64)
		mockDir, apiLog, cleanup := setupMockEnv(t, ghScript, defaultYqMock())
		defer cleanup()

		_, logStr, err := runReconcileWithTemplate(t, scriptPath, mockDir, apiLog, shimTemplate)

		// Script should not crash (exit 0 or exit 1 from failed count, but
		// not a bash crash). A non-zero exit from FAILED counter is acceptable
		// as long as the script ran to completion.
		if err != nil {
			// If it exited non-zero, it should still have produced the summary
			assert.Contains(t, logStr, "git/blobs",
				"script should have attempted blob creation for stale (empty) content")
		}

		// Empty remote content is stale -- an update PR should be created.
		assert.Contains(t, logStr, "PR_CREATED",
			"empty remote content should be detected as stale and trigger an update PR")
	})
}
