package scaffold

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// reconcileHarness encapsulates common test infrastructure for reconcile-repos.sh tests.
// Each test creates a harness, customizes the mock gh binary, runs the script,
// and asserts on stdout/stderr/artifacts.
type reconcileHarness struct {
	t         *testing.T
	tmpDir    string
	configDir string
	mockBin   string
	ghLog     string
	scriptPath string
}

// newReconcileHarness creates a temporary directory with config.yaml, shim template,
// and mock base64/yq binaries. The caller must provide a mock gh binary via writeGHMock.
func newReconcileHarness(t *testing.T) *reconcileHarness {
	t.Helper()
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "config")
	mockBin := filepath.Join(tmpDir, "bin")
	ghLog := filepath.Join(tmpDir, "gh-calls.log")

	require.NoError(t, os.MkdirAll(filepath.Join(configDir, "templates"), 0o755))
	require.NoError(t, os.MkdirAll(mockBin, 0o755))

	// Resolve the absolute path to reconcile-repos.sh from the test's working directory.
	scriptPath, err := filepath.Abs("fullsend-repo/scripts/reconcile-repos.sh")
	require.NoError(t, err)
	require.FileExists(t, scriptPath)

	h := &reconcileHarness{
		t:          t,
		tmpDir:     tmpDir,
		configDir:  configDir,
		mockBin:    mockBin,
		ghLog:      ghLog,
		scriptPath: scriptPath,
	}

	h.writeDefaultConfig()
	h.writeDefaultTemplate()
	h.writeMockBase64()
	h.writeMockYQ([]string{"test-repo"}, nil)

	return h
}

// writeDefaultConfig writes a config.yaml with a single enabled repo.
func (h *reconcileHarness) writeDefaultConfig() {
	h.t.Helper()
	config := `version: 1
repos:
  test-repo:
    enabled: true
`
	require.NoError(h.t, os.WriteFile(filepath.Join(h.configDir, "config.yaml"), []byte(config), 0o644))
}

// writeConfig writes a custom config.yaml.
func (h *reconcileHarness) writeConfig(content string) {
	h.t.Helper()
	require.NoError(h.t, os.WriteFile(filepath.Join(h.configDir, "config.yaml"), []byte(content), 0o644))
}

// writeDefaultTemplate writes the shim template with sentinel + "fresh shim template".
func (h *reconcileHarness) writeDefaultTemplate() {
	h.t.Helper()
	template := "# --- fullsend managed below - do not edit ---\nfresh shim template\n"
	require.NoError(h.t, os.WriteFile(
		filepath.Join(h.configDir, "templates", "shim-workflow-call.yaml"),
		[]byte(template), 0o644))
}

// writeMockBase64 creates a mock base64 that delegates to /usr/bin/base64
// but strips newlines when called with -w0.
func (h *reconcileHarness) writeMockBase64() {
	h.t.Helper()
	script := `#!/usr/bin/env bash
if [[ "${1:-}" == "-w0" ]]; then
  shift
  /usr/bin/base64 "$@" | tr -d '\r\n'
else
  /usr/bin/base64 "$@"
fi
`
	path := filepath.Join(h.mockBin, "base64")
	require.NoError(h.t, os.WriteFile(path, []byte(script), 0o755))
}

// writeMockYQ creates a mock yq that returns the given enabled and disabled repos.
func (h *reconcileHarness) writeMockYQ(enabled, disabled []string) {
	h.t.Helper()
	enabledStr := strings.Join(enabled, "\n")
	disabledStr := strings.Join(disabled, "\n")
	script := fmt.Sprintf(`#!/usr/bin/env bash
query="${1:-}"
if [[ "$query" == *"enabled == true"* ]]; then
  printf '%%s\n' %s
elif [[ "$query" == *"enabled == false"* ]]; then
  printf '%%s\n' %s
else
  echo "unexpected yq query: $*" >&2
  exit 1
fi
`, shellescape(enabledStr), shellescape(disabledStr))
	path := filepath.Join(h.mockBin, "yq")
	require.NoError(h.t, os.WriteFile(path, []byte(script), 0o755))
}

// shellescape wraps a string in single quotes for safe shell embedding.
func shellescape(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// writeGHMock writes a mock gh binary. The caseBlock is inserted into a
// case statement that matches on the API endpoint. The prBlock handles
// "gh pr" subcommands. Blob input is automatically captured.
func (h *reconcileHarness) writeGHMock(opts ghMockOpts) {
	h.t.Helper()

	script := fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail
printf 'gh' >> %s
for arg in "$@"; do
  printf ' %%q' "$arg" >> %s
done
printf '\n' >> %s

# Handle pr subcommands.
if [[ "$1" == "pr" ]]; then
  %s
  exit 0
fi

if [[ "$1" != "api" ]]; then
  exit 0
fi

jq_filter=""
has_input=false
method="GET"
shift  # consume "api"
endpoint="$1"; shift
while [[ $# -gt 0 ]]; do
  case "$1" in
    --jq) jq_filter="$2"; shift 2 ;;
    --input) has_input=true; shift 2 ;;
    --method) method="$2"; shift 2 ;;
    --field) shift 2 ;;
    --silent) shift ;;
    *) shift ;;
  esac
done

# Capture blob input.
input_data=""
if [[ "$has_input" == "true" ]]; then
  input_data=$(cat)
  if [[ "$endpoint" == *"/git/blobs" ]]; then
    blob_repo=$(printf '%%s' "$endpoint" | sed 's|repos/[^/]*/||;s|/git/blobs||')
    printf '%%s' "$input_data" > %s/blob-input-${blob_repo}.json
  fi
fi

json=""
rc=0
case "$endpoint" in
  repos/test-org/*/actions/variables/*)
    json='{"status":"404","message":"Not Found"}'
    rc=1
    ;;
  %s
  repos/test-org/*/git/ref/heads/*)
    json='{"object":{"sha":"base-sha"}}'
    ;;
  repos/test-org/*/git/commits/base-sha)
    json='{"tree":{"sha":"base-tree-sha"}}'
    ;;
  repos/test-org/*/git/blobs)
    json='{"sha":"blob-sha"}'
    ;;
  repos/test-org/*/git/trees)
    json='{"sha":"tree-sha"}'
    ;;
  repos/test-org/*/git/commits)
    json='{"sha":"desired-commit-sha"}'
    ;;
  repos/test-org/*/git/refs)
    rc=1
    ;;
  repos/test-org/*/git/refs/heads/*)
    rc=0
    ;;
  repos/test-org/*)
    json='{"default_branch":"main","private":false}'
    ;;
  *)
    rc=0
    ;;
esac

if [[ -n "$json" ]]; then
  if [[ -n "$jq_filter" ]]; then
    printf '%%s' "$json" | jq -r "$jq_filter"
  else
    printf '%%s\n' "$json"
  fi
fi
exit "$rc"
`,
		shellescape(h.ghLog),
		shellescape(h.ghLog),
		shellescape(h.ghLog),
		opts.prBlock,
		shellescape(h.tmpDir),
		opts.apiCases,
	)
	path := filepath.Join(h.mockBin, "gh")
	require.NoError(h.t, os.WriteFile(path, []byte(script), 0o755))
}

// ghMockOpts configures the mock gh binary behavior.
type ghMockOpts struct {
	// prBlock is shell code handling "gh pr" subcommands (runs inside if [[ "$1" == "pr" ]]).
	prBlock string
	// apiCases are additional case clauses for the API endpoint case statement.
	// They must end with ;; and should be placed before the wildcard repos/test-org/* case.
	apiCases string
}

// run executes reconcile-repos.sh and returns stdout+stderr combined output.
func (h *reconcileHarness) run() (string, int) {
	h.t.Helper()
	cmd := exec.Command("bash", h.scriptPath, h.configDir)
	cmd.Env = []string{
		"PATH=" + h.mockBin + string(os.PathListSeparator) + os.Getenv("PATH"),
		"GITHUB_REPOSITORY_OWNER=test-org",
		"GITHUB_SHA=test-sha",
		"GH_TOKEN=fake-token",
		"HOME=" + os.Getenv("HOME"),
	}
	output, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			h.t.Fatalf("failed to run reconcile-repos.sh: %v", err)
		}
	}
	return string(output), exitCode
}

// blobContent reads and base64-decodes the blob input captured for a given repo.
// Returns empty string if no blob was captured.
func (h *reconcileHarness) blobContent(repo string) string {
	h.t.Helper()
	path := filepath.Join(h.tmpDir, fmt.Sprintf("blob-input-%s.json", repo))
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return ""
	}
	require.NoError(h.t, err)

	var blob struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	if err := json.Unmarshal(data, &blob); err != nil {
		h.t.Logf("blob JSON parse error: %v, raw: %s", err, string(data))
		return ""
	}
	if blob.Content == "" {
		return ""
	}
	decoded, err := base64.StdEncoding.DecodeString(blob.Content)
	if err != nil {
		// Try with padding adjustment.
		decoded, err = base64.RawStdEncoding.DecodeString(blob.Content)
		require.NoError(h.t, err, "failed to decode blob content: %s", blob.Content)
	}
	return string(decoded)
}

// blobExists checks whether a blob input file was captured for the given repo.
func (h *reconcileHarness) blobExists(repo string) bool {
	h.t.Helper()
	path := filepath.Join(h.tmpDir, fmt.Sprintf("blob-input-%s.json", repo))
	_, err := os.Stat(path)
	return err == nil
}

// ghCallsLog returns the content of the gh-calls.log file.
func (h *reconcileHarness) ghCallsLog() string {
	h.t.Helper()
	data, err := os.ReadFile(h.ghLog)
	if os.IsNotExist(err) {
		return ""
	}
	require.NoError(h.t, err)
	return string(data)
}

// b64encode base64-encodes a string (no line wrapping).
func b64encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}
