//go:build e2e

package scaffold_test

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

const (
	sentinelLine = "# --- fullsend managed below - do not edit ---"
	freshTemplate = "fresh shim template"
)

// testEnv holds shared state for a reconcile-repos.sh test scenario.
type testEnv struct {
	t             *testing.T
	tmpDir        string
	configDir     string
	mockBin       string
	ghLog         string
	commitMsgsLog string
	scriptPath    string
}

// newTestEnv creates a temporary environment with config, template, and mock
// base64 binary. The caller is responsible for installing gh and yq mocks via
// the provided helper methods before running the script.
func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "config")
	mockBin := filepath.Join(tmpDir, "bin")
	require.NoError(t, os.MkdirAll(filepath.Join(configDir, "templates"), 0o755))
	require.NoError(t, os.MkdirAll(mockBin, 0o755))

	ghLog := filepath.Join(tmpDir, "gh-calls.log")
	commitMsgsLog := filepath.Join(tmpDir, "commit-msgs.log")

	// Write config.yaml with a single test-repo by default.
	writeFile(t, filepath.Join(configDir, "config.yaml"), `version: 1
repos:
  test-repo:
    enabled: true
`)

	// Write shim template with sentinel and fresh content.
	writeFile(t, filepath.Join(configDir, "templates", "shim-workflow-call.yaml"),
		sentinelLine+"\n"+freshTemplate+"\n")

	// Write mock base64 binary that handles -w0 flag.
	writeFile(t, filepath.Join(mockBin, "base64"), `#!/usr/bin/env bash
if [[ "${1:-}" == "-w0" ]]; then
  shift
  /usr/bin/base64 "$@" | tr -d '\r\n'
else
  /usr/bin/base64 "$@"
fi
`)
	makeExecutable(t, filepath.Join(mockBin, "base64"))

	// Write default yq mock (single repo).
	writeFile(t, filepath.Join(mockBin, "yq"), `#!/usr/bin/env bash
query="${1:-}"
if [[ "$query" == *"enabled == true"* ]]; then
  echo "test-repo"
elif [[ "$query" == *"enabled == false"* ]]; then
  echo ""
else
  echo "unexpected yq query: $*" >&2
  exit 1
fi
`)
	makeExecutable(t, filepath.Join(mockBin, "yq"))

	// Locate reconcile-repos.sh relative to the repo root.
	scriptPath := findReconcileScript(t)

	return &testEnv{
		t:             t,
		tmpDir:        tmpDir,
		configDir:     configDir,
		mockBin:       mockBin,
		ghLog:         ghLog,
		commitMsgsLog: commitMsgsLog,
		scriptPath:    scriptPath,
	}
}

// findReconcileScript locates reconcile-repos.sh by walking up from CWD or
// using known paths.
func findReconcileScript(t *testing.T) string {
	t.Helper()

	candidates := []string{
		"internal/scaffold/fullsend-repo/scripts/reconcile-repos.sh",
	}

	// Try from FULLSEND_TARGET_REPO_DIR if set.
	if repoDir := os.Getenv("FULLSEND_TARGET_REPO_DIR"); repoDir != "" {
		for _, c := range candidates {
			p := filepath.Join(repoDir, c)
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}

	// Try from current working directory and parents.
	dir, _ := os.Getwd()
	for i := 0; i < 5; i++ {
		for _, c := range candidates {
			p := filepath.Join(dir, c)
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
		dir = filepath.Dir(dir)
	}

	t.Fatalf("reconcile-repos.sh not found")
	return ""
}

// installGhMock writes a mock gh binary with the given bash script body.
// The body is the case statement handling for endpoint matching.
// A standard preamble that logs calls and parses flags is prepended.
func (e *testEnv) installGhMock(t *testing.T, caseBody string) {
	t.Helper()

	script := fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail
printf 'gh' >> "%s"
for arg in "$@"; do
  printf ' %%q' "$arg" >> "%s"
done
printf '\n' >> "%s"

# Handle pr subcommands.
if [[ "$1" == "pr" ]]; then
  case "$2" in
    list)
      repo_arg=""
      head_arg=""
      prev=""
      for arg in "$@"; do
        case "$prev" in
          --repo) repo_arg="$arg" ;;
          --head) head_arg="$arg" ;;
        esac
        prev="$arg"
      done
      ;;
    create) echo "https://github.com/test-org/mock/pull/99"; exit 0 ;;
    close) exit 0 ;;
  esac
  exit 0
fi

if [[ "$1" != "api" ]]; then
  exit 0
fi

jq_filter=""
has_input=false
method="GET"
shift
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

input_data=""
if [[ "$has_input" == "true" ]]; then
  input_data=$(cat)
  if [[ "$endpoint" == *"/git/blobs" ]]; then
    blob_repo=$(printf '%%s' "$endpoint" | cut -d/ -f3)
    printf '%%s' "$input_data" > "%s/blob-input-${blob_repo}.json"
  fi
fi

json=""
rc=0
%s

if [[ -n "$json" ]]; then
  if [[ -n "$jq_filter" ]]; then
    printf '%%s' "$json" | jq -r "$jq_filter"
  else
    printf '%%s\n' "$json"
  fi
fi
exit "$rc"
`, e.ghLog, e.ghLog, e.ghLog, e.tmpDir, caseBody)

	writeFile(t, filepath.Join(e.mockBin, "gh"), script)
	makeExecutable(t, filepath.Join(e.mockBin, "gh"))
}

// installYqMock writes a yq mock that returns the given repo list.
func (e *testEnv) installYqMock(t *testing.T, enabledRepos []string, disabledRepos []string) {
	t.Helper()

	enabled := strings.Join(enabledRepos, "\n")
	disabled := strings.Join(disabledRepos, "\n")

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
`, quoteRepos(enabledRepos, enabled), quoteRepos(disabledRepos, disabled))

	writeFile(t, filepath.Join(e.mockBin, "yq"), script)
	makeExecutable(t, filepath.Join(e.mockBin, "yq"))
}

func quoteRepos(repos []string, _ string) string {
	if len(repos) == 0 {
		return `""`
	}
	quoted := make([]string, len(repos))
	for i, r := range repos {
		quoted[i] = fmt.Sprintf("%q", r)
	}
	return strings.Join(quoted, " ")
}

// setConfig writes a custom config.yaml to the test environment.
func (e *testEnv) setConfig(t *testing.T, content string) {
	t.Helper()
	writeFile(t, filepath.Join(e.configDir, "config.yaml"), content)
}

// runReconcile executes the reconcile-repos.sh script and returns
// combined stdout+stderr output and any error.
func (e *testEnv) runReconcile(t *testing.T) (string, error) {
	t.Helper()

	cmd := exec.Command("bash", e.scriptPath, e.configDir)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("PATH=%s:%s", e.mockBin, os.Getenv("PATH")),
		"GITHUB_REPOSITORY_OWNER=test-org",
		"GITHUB_SHA=test-sha",
		"GH_TOKEN=fake-token",
	)

	out, err := cmd.CombinedOutput()
	return string(out), err
}

// ghLogContents reads the mock gh call log.
func (e *testEnv) ghLogContents(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(e.ghLog)
	if err != nil {
		return ""
	}
	return string(data)
}

// blobContent reads and decodes the captured blob content for a repo.
func (e *testEnv) blobContent(t *testing.T, repo string) (string, bool) {
	t.Helper()
	blobFile := filepath.Join(e.tmpDir, fmt.Sprintf("blob-input-%s.json", repo))
	data, err := os.ReadFile(blobFile)
	if err != nil {
		return "", false
	}

	// Extract the base64 content from the JSON using simple parsing.
	// The blob JSON looks like: {"content":"<b64>","encoding":"base64"}
	s := string(data)
	prefix := `"content":"`
	idx := strings.Index(s, prefix)
	if idx < 0 {
		t.Logf("blob JSON has no content field: %s", s)
		return "", false
	}
	start := idx + len(prefix)
	end := strings.Index(s[start:], `"`)
	if end < 0 {
		t.Logf("blob JSON malformed: %s", s)
		return "", false
	}
	b64 := s[start : start+end]

	decoded, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		// Try with padding normalization.
		decoded, err = base64.RawStdEncoding.DecodeString(strings.TrimRight(b64, "="))
		if err != nil {
			t.Logf("failed to decode blob base64: %v", err)
			return "", false
		}
	}

	return string(decoded), true
}

// blobExists checks whether a blob file was created for the given repo.
func (e *testEnv) blobExists(t *testing.T, repo string) bool {
	t.Helper()
	blobFile := filepath.Join(e.tmpDir, fmt.Sprintf("blob-input-%s.json", repo))
	_, err := os.Stat(blobFile)
	return err == nil
}

// encodeB64 base64-encodes a string (no line wrapping).
func encodeB64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// writeFile writes content to a file, creating parent dirs as needed.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

// makeExecutable sets the executable permission on a file.
func makeExecutable(t *testing.T, path string) {
	t.Helper()
	require.NoError(t, os.Chmod(path, 0o755))
}
