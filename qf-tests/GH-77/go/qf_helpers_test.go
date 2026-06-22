package scaffold

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Constants shared across GH-77 test files. These mirror the values in
// qf-tests/GH-2247/go/helpers_test.go to maintain consistency.
const (
	sentinel      = "# --- fullsend managed below - do not edit ---"
	freshTemplate = "fresh shim template"
	staleTemplate = "stale shim template"
	testOrg       = "test-org"
	testRepo      = "test-repo"
	testGHToken   = "ghp_fake_token_for_testing"
)

// reconcileEnv holds the isolated filesystem and mock binaries needed to
// run reconcile-repos.sh under test.
type reconcileEnv struct {
	t          *testing.T
	tmpDir     string
	configDir  string
	mockBinDir string
	scriptPath string
	ghCallsLog string
}

// newReconcileEnv creates a fully isolated test environment with config,
// shim template, and mock binaries (yq, gh).
func newReconcileEnv(t *testing.T) *reconcileEnv {
	t.Helper()

	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "config")
	require.NoError(t, os.MkdirAll(filepath.Join(configDir, "templates"), 0o755))

	mockBinDir := filepath.Join(tmpDir, "bin")
	require.NoError(t, os.MkdirAll(mockBinDir, 0o755))

	ghCallsLog := filepath.Join(tmpDir, "gh-calls.log")

	// Write config.yaml with one enabled repo.
	configYAML := fmt.Sprintf("repos:\n  %s:\n    enabled: true\n", testRepo)
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configYAML), 0o644))

	// Write shim template containing the sentinel and fresh managed content.
	shimTemplate := sentinel + "\n" + freshTemplate + "\n"
	require.NoError(t, os.WriteFile(
		filepath.Join(configDir, "templates", "shim-workflow-call.yaml"),
		[]byte(shimTemplate), 0o644))

	// Mock yq — returns the repo name for enabled queries, empty for disabled.
	writeScript(t, filepath.Join(mockBinDir, "yq"), `#!/usr/bin/env bash
args="$*"
if echo "$args" | grep -q 'enabled == true'; then
  echo "`+testRepo+`"
elif echo "$args" | grep -q 'enabled == false'; then
  echo ""
fi
`)

	// Symlink real jq if available.
	realJQ, err := exec.LookPath("jq")
	if err == nil {
		os.Symlink(realJQ, filepath.Join(mockBinDir, "jq"))
	}

	scriptPath := findScriptPath(t)

	env := &reconcileEnv{
		t:          t,
		tmpDir:     tmpDir,
		configDir:  configDir,
		mockBinDir: mockBinDir,
		scriptPath: scriptPath,
		ghCallsLog: ghCallsLog,
	}

	env.writeDefaultGHMock("")
	return env
}

// writeDefaultGHMock writes the mock gh script. remoteContentB64 is the
// base64-encoded content returned for the contents API endpoint.
func (e *reconcileEnv) writeDefaultGHMock(remoteContentB64 string) {
	e.t.Helper()

	contentsHandler := `echo "not-found" >&2; exit 1`
	if remoteContentB64 != "" {
		contentsHandler = fmt.Sprintf(`printf '%%s' '%s'`, remoteContentB64)
	}

	mockGH := fmt.Sprintf(`#!/usr/bin/env bash
echo "$@" >> "%s"

case "$1" in
  api)
    endpoint="$2"
    case "$endpoint" in
      repos/*/contents/*)
        %s
        ;;
      repos/*/git/ref/heads/*)
        echo "mock-default-branch-sha"
        ;;
      repos/*/git/commits/*)
        echo "mock-tree-sha"
        ;;
      repos/*/git/blobs)
        echo "mock-blob-sha"
        ;;
      repos/*/git/trees)
        echo "mock-tree-sha-new"
        ;;
      repos/*/git/commits)
        echo "mock-commit-sha"
        ;;
      repos/*/git/refs)
        exit 0
        ;;
      repos/*/git/refs/heads/*)
        if echo "$@" | grep -q "PATCH"; then
          exit 0
        elif echo "$@" | grep -q "DELETE"; then
          exit 0
        fi
        echo "mock-ref-sha"
        ;;
      repos/*/actions/variables/*)
        printf '{"status":"404","message":"Not Found"}'
        exit 1
        ;;
      *)
        if echo "$@" | grep -q '\.private'; then
          echo "false"
        elif echo "$@" | grep -q '\.default_branch'; then
          echo "main"
        elif echo "$@" | grep -q '\.visibility'; then
          echo "public"
        else
          echo "{}"
        fi
        ;;
    esac
    ;;
  pr)
    case "$2" in
      list)
        echo ""
        ;;
      create)
        echo "https://github.com/%s/%s/pull/99"
        ;;
      close)
        exit 0
        ;;
    esac
    ;;
esac
`, e.ghCallsLog, contentsHandler, testOrg, testRepo)

	writeScript(e.t, filepath.Join(e.mockBinDir, "gh"), mockGH)
}

// setRemoteContent configures the mock to return the given decoded string
// as the remote shim content (base64-encoded for the API mock).
func (e *reconcileEnv) setRemoteContent(content string) {
	e.t.Helper()
	b64 := base64.StdEncoding.EncodeToString([]byte(content))
	e.writeDefaultGHMock(b64)
}

// run executes reconcile-repos.sh with the test environment.
func (e *reconcileEnv) run() (string, error) {
	e.t.Helper()

	cmd := exec.Command("bash", e.scriptPath, e.configDir)
	cmd.Env = []string{
		"PATH=" + e.mockBinDir + ":" + os.Getenv("PATH"),
		"HOME=" + e.tmpDir,
		"GITHUB_REPOSITORY_OWNER=" + testOrg,
		"GH_TOKEN=" + testGHToken,
		"GITHUB_SHA=test-sha-abc123",
	}
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// ghCalls returns all logged gh CLI invocations.
func (e *reconcileEnv) ghCalls() []string {
	e.t.Helper()
	data, err := os.ReadFile(e.ghCallsLog)
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil
	}
	return lines
}

// hasBlobCall returns true if any gh call hit the git/blobs endpoint.
func (e *reconcileEnv) hasBlobCall() bool {
	for _, call := range e.ghCalls() {
		if strings.Contains(call, "git/blobs") {
			return true
		}
	}
	return false
}

// blobCreated returns true if a blob creation API call was made.
func (e *reconcileEnv) blobCreated() bool {
	return e.hasBlobCall()
}

// runBashFunc runs a bash function from reconcile-repos.sh in isolation.
func (e *reconcileEnv) runBashFunc(code string) (string, error) {
	e.t.Helper()

	wrapper := fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail
SENTINEL="%s"
extract_managed_content() {
  awk -v sentinel="$SENTINEL" '
    found { print; next }
    $0 == sentinel { found=1; print }
  '
}
extract_user_header() {
  awk -v sentinel="$SENTINEL" '
    $0 == sentinel { exit }
    { print }
  '
}
%s
`, sentinel, code)

	cmd := exec.Command("bash", "-c", wrapper)
	cmd.Env = []string{
		"PATH=" + e.mockBinDir + ":" + os.Getenv("PATH"),
		"HOME=" + e.tmpDir,
	}
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// writeScript creates an executable script file.
func writeScript(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(content), 0o755))
}

// findScriptPath locates reconcile-repos.sh by walking up from the working
// directory to find the repository root.
func findScriptPath(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	require.NoError(t, err)

	for {
		candidate := filepath.Join(dir, "internal", "scaffold", "fullsend-repo", "scripts", "reconcile-repos.sh")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			candidate = filepath.Join(dir, "internal", "scaffold", "fullsend-repo", "scripts", "reconcile-repos.sh")
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	for _, root := range []string{
		os.Getenv("GITHUB_WORKSPACE"),
		"/sandbox/workspace/pr-repo",
	} {
		if root == "" {
			continue
		}
		candidate := filepath.Join(root, "internal", "scaffold", "fullsend-repo", "scripts", "reconcile-repos.sh")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	t.Fatal("reconcile-repos.sh not found — set GITHUB_WORKSPACE or run from repo root")
	return ""
}

// b64Encode base64-encodes a string with no line wrapping.
func b64Encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}
