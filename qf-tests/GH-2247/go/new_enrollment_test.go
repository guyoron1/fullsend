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

func TestNewEnrollment(t *testing.T) {
	t.Run("[test_id:TS-GH-2247-023] should create shim with sentinel and ORG interpolated for new enrollment", func(t *testing.T) {
		org := "enroll-org"

		// Mock gh: 404 on content fetch (not enrolled), accept blob/tree/commit/ref/PR creation,
		// log blob payload for inspection.
		ghScript := `#!/usr/bin/env bash
LOGFILE="$MOCK_API_LOG"
echo "CALL: $*" >> "$LOGFILE"

# gh api repos/ORG/REPO -> repo metadata (public)
if [[ "$1" == "api" ]] && [[ "$2" =~ ^repos/[^/]+/[^/]+$ ]] && [[ "$*" != *"--method"* ]]; then
    if [[ "$*" == *"--jq"* ]] && [[ "$*" == *".private"* ]]; then
        echo "false"
    elif [[ "$*" == *"--jq"* ]] && [[ "$*" == *".default_branch"* ]]; then
        echo "main"
    else
        echo '{"private":false,"default_branch":"main"}'
    fi
    exit 0
fi

# gh api repos/ORG/REPO/contents/PATH -> 404 (not enrolled)
if [[ "$*" == *"/contents/"* ]] && [[ "$*" != *"--method"* ]]; then
    echo "Not Found" >&2
    exit 1
fi

# gh pr list -> no existing PRs
if [[ "$1" == "pr" ]] && [[ "$2" == "list" ]]; then
    echo ""
    exit 0
fi

# gh api repos/ORG/REPO/git/ref/heads/main -> default branch SHA
if [[ "$*" == *"/git/ref/heads/"* ]] && [[ "$*" != *"--method"* ]]; then
    if [[ "$*" == *"--jq"* ]]; then
        echo "deadbeef1234567890"
    else
        echo '{"object":{"sha":"deadbeef1234567890"}}'
    fi
    exit 0
fi

# gh api repos/ORG/REPO/git/commits/SHA -> tree SHA
if [[ "$*" == *"/git/commits/"* ]] && [[ "$*" != *"--method"* ]]; then
    if [[ "$*" == *"--jq"* ]]; then
        echo "tree1234567890"
    else
        echo '{"tree":{"sha":"tree1234567890"}}'
    fi
    exit 0
fi

# gh api repos/ORG/REPO/git/blobs POST -> create blob, log content
if [[ "$*" == *"/git/blobs"* ]] && [[ "$*" == *"--method"* ]] && [[ "$*" == *"POST"* ]]; then
    # Read stdin and log it as the blob payload
    INPUT=$(cat)
    echo "BLOB_PAYLOAD: $INPUT" >> "$LOGFILE"
    if [[ "$*" == *"--jq"* ]]; then
        echo "blob_sha_001"
    else
        echo '{"sha":"blob_sha_001"}'
    fi
    exit 0
fi

# gh api repos/ORG/REPO/git/trees POST -> create tree
if [[ "$*" == *"/git/trees"* ]] && [[ "$*" == *"--method"* ]] && [[ "$*" == *"POST"* ]]; then
    cat > /dev/null  # consume stdin
    if [[ "$*" == *"--jq"* ]]; then
        echo "tree_sha_001"
    else
        echo '{"sha":"tree_sha_001"}'
    fi
    exit 0
fi

# gh api repos/ORG/REPO/git/commits POST -> create commit
if [[ "$*" == *"/git/commits"* ]] && [[ "$*" == *"--method"* ]] && [[ "$*" == *"POST"* ]]; then
    cat > /dev/null  # consume stdin
    if [[ "$*" == *"--jq"* ]]; then
        echo "commit_sha_001"
    else
        echo '{"sha":"commit_sha_001"}'
    fi
    exit 0
fi

# gh api repos/ORG/REPO/git/refs POST -> create ref
if [[ "$*" == *"/git/refs"* ]] && [[ "$*" == *"--method"* ]]; then
    echo '{"ref":"refs/heads/fullsend/onboard"}'
    exit 0
fi

# gh pr create -> log the PR creation
if [[ "$1" == "pr" ]] && [[ "$2" == "create" ]]; then
    echo "PR_CREATE: $*" >> "$LOGFILE"
    echo "https://github.com/enroll-org/test-repo/pull/1"
    exit 0
fi

# gh pr close / other pr commands
if [[ "$1" == "pr" ]]; then
    exit 0
fi

# Default: succeed
exit 0
`
		mockDir, apiLog, cleanup := setupMockEnv(t, ghScript, defaultYqMock())
		defer cleanup()

		configDir := filepath.Join(mockDir, "configdir")
		templateDir := filepath.Join(configDir, "templates")
		require.NoError(t, os.MkdirAll(templateDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(configDir, "config.yaml"),
			[]byte("repos:\n  test-repo:\n    enabled: true\n"), 0o644))
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

		require.NoError(t, err, "reconcile should succeed for new enrollment; output: %s", outputStr)

		// Read the API log and extract the blob payload
		logBytes, err := os.ReadFile(apiLog)
		require.NoError(t, err)
		logStr := string(logBytes)

		// Find the BLOB_PAYLOAD line and extract the base64 content
		var blobContent string
		for _, line := range strings.Split(logStr, "\n") {
			if strings.HasPrefix(line, "BLOB_PAYLOAD:") {
				payload := strings.TrimPrefix(line, "BLOB_PAYLOAD: ")
				// The payload is JSON: {"content":"<base64>","encoding":"base64"}
				// Extract the base64 content between "content":" and ","
				if idx := strings.Index(payload, `"content":"`); idx >= 0 {
					start := idx + len(`"content":"`)
					end := strings.Index(payload[start:], `"`)
					if end >= 0 {
						b64str := payload[start : start+end]
						decoded, decErr := base64.StdEncoding.DecodeString(b64str)
						if decErr == nil {
							blobContent = string(decoded)
						}
					}
				}
				break
			}
		}

		require.NotEmpty(t, blobContent, "should have captured blob content from mock log")

		// Assert blob contains sentinel markers (the shim template has "fullsend shim workflow" comments)
		assert.Contains(t, blobContent, "fullsend", "blob should contain fullsend sentinel marker")
		assert.Contains(t, blobContent, "name: fullsend", "blob should contain the workflow name sentinel")

		// Assert __ORG__ was fully interpolated (no literal placeholder remains)
		assert.NotContains(t, blobContent, "__ORG__", "blob must not contain literal __ORG__ placeholder")

		// Assert the org was actually interpolated into the content
		assert.Contains(t, blobContent, org, "blob should contain the interpolated org name")
	})

	t.Run("[test_id:TS-GH-2247-024] should create enrollment PR with correct title and body", func(t *testing.T) {
		org := "pr-test-org"

		// Mock gh that captures PR creation args
		ghScript := `#!/usr/bin/env bash
LOGFILE="$MOCK_API_LOG"
echo "CALL: $*" >> "$LOGFILE"

if [[ "$1" == "api" ]] && [[ "$2" =~ ^repos/[^/]+/[^/]+$ ]] && [[ "$*" != *"--method"* ]]; then
    if [[ "$*" == *"--jq"* ]] && [[ "$*" == *".private"* ]]; then
        echo "false"
    elif [[ "$*" == *"--jq"* ]] && [[ "$*" == *".default_branch"* ]]; then
        echo "main"
    else
        echo '{"private":false,"default_branch":"main"}'
    fi
    exit 0
fi

if [[ "$*" == *"/contents/"* ]] && [[ "$*" != *"--method"* ]]; then
    echo "Not Found" >&2
    exit 1
fi

if [[ "$1" == "pr" ]] && [[ "$2" == "list" ]]; then
    echo ""
    exit 0
fi

if [[ "$*" == *"/git/ref/heads/"* ]] && [[ "$*" != *"--method"* ]]; then
    if [[ "$*" == *"--jq"* ]]; then
        echo "deadbeef1234567890"
    else
        echo '{"object":{"sha":"deadbeef1234567890"}}'
    fi
    exit 0
fi

if [[ "$*" == *"/git/commits/"* ]] && [[ "$*" != *"--method"* ]]; then
    if [[ "$*" == *"--jq"* ]]; then
        echo "tree1234567890"
    else
        echo '{"tree":{"sha":"tree1234567890"}}'
    fi
    exit 0
fi

if [[ "$*" == *"/git/blobs"* ]] && [[ "$*" == *"POST"* ]]; then
    cat > /dev/null
    if [[ "$*" == *"--jq"* ]]; then echo "blob_sha_001"; else echo '{"sha":"blob_sha_001"}'; fi
    exit 0
fi

if [[ "$*" == *"/git/trees"* ]] && [[ "$*" == *"POST"* ]]; then
    cat > /dev/null
    if [[ "$*" == *"--jq"* ]]; then echo "tree_sha_001"; else echo '{"sha":"tree_sha_001"}'; fi
    exit 0
fi

if [[ "$*" == *"/git/commits"* ]] && [[ "$*" == *"POST"* ]]; then
    cat > /dev/null
    if [[ "$*" == *"--jq"* ]]; then echo "commit_sha_001"; else echo '{"sha":"commit_sha_001"}'; fi
    exit 0
fi

if [[ "$*" == *"/git/refs"* ]] && [[ "$*" == *"--method"* ]]; then
    echo '{"ref":"refs/heads/fullsend/onboard"}'
    exit 0
fi

if [[ "$1" == "pr" ]] && [[ "$2" == "create" ]]; then
    # Log all args for inspection
    echo "PR_CREATE_ARGS: $*" >> "$LOGFILE"
    # Also log individual flags
    TITLE=""
    BODY=""
    CAPTURE_NEXT=""
    for arg in "$@"; do
        if [[ "$CAPTURE_NEXT" == "title" ]]; then
            TITLE="$arg"
            CAPTURE_NEXT=""
        elif [[ "$CAPTURE_NEXT" == "body" ]]; then
            BODY="$arg"
            CAPTURE_NEXT=""
        elif [[ "$arg" == "--title" ]]; then
            CAPTURE_NEXT="title"
        elif [[ "$arg" == "--body" ]]; then
            CAPTURE_NEXT="body"
        fi
    done
    echo "PR_TITLE: $TITLE" >> "$LOGFILE"
    echo "PR_BODY: $BODY" >> "$LOGFILE"
    echo "https://github.com/pr-test-org/enrolled-repo/pull/1"
    exit 0
fi

if [[ "$1" == "pr" ]]; then
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
			[]byte("repos:\n  enrolled-repo:\n    enabled: true\n"), 0o644))
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

		require.NoError(t, err, "reconcile should succeed for new enrollment; output: %s", outputStr)

		// Read the API log and extract PR creation details
		logBytes, err := os.ReadFile(apiLog)
		require.NoError(t, err)
		logStr := string(logBytes)

		// Verify a PR was created
		assert.Contains(t, logStr, "PR_CREATE_ARGS:", "a PR creation call should have been logged")

		// Extract and validate PR title
		var prTitle, prBody string
		for _, line := range strings.Split(logStr, "\n") {
			if strings.HasPrefix(line, "PR_TITLE: ") {
				prTitle = strings.TrimPrefix(line, "PR_TITLE: ")
			}
			if strings.HasPrefix(line, "PR_BODY: ") {
				prBody = strings.TrimPrefix(line, "PR_BODY: ")
			}
		}

		require.NotEmpty(t, prTitle, "PR title should have been captured")

		// PR title should reference shim/enrollment/fullsend
		titleLower := strings.ToLower(prTitle)
		assert.True(t,
			strings.Contains(titleLower, "shim") ||
				strings.Contains(titleLower, "enroll") ||
				strings.Contains(titleLower, "fullsend"),
			"PR title should reference shim, enrollment, or fullsend; got: %s", prTitle)

		// PR body should explain the purpose
		require.NotEmpty(t, prBody, "PR body should have been captured")
		bodyLower := strings.ToLower(prBody)
		assert.True(t,
			strings.Contains(bodyLower, "shim") ||
				strings.Contains(bodyLower, "workflow") ||
				strings.Contains(bodyLower, "fullsend") ||
				strings.Contains(bodyLower, "agent"),
			"PR body should explain the purpose; got: %s", prBody)
	})

	t.Run("[test_id:TS-GH-2247-025] should skip enrollment for private repos", func(t *testing.T) {
		org := "private-org"

		// Mock gh: return private=true for repo metadata
		ghScript := `#!/usr/bin/env bash
LOGFILE="$MOCK_API_LOG"
echo "CALL: $*" >> "$LOGFILE"

# gh api repos/ORG/REPO -> repo metadata (PRIVATE)
if [[ "$1" == "api" ]] && [[ "$2" =~ ^repos/[^/]+/[^/]+$ ]] && [[ "$*" != *"--method"* ]]; then
    if [[ "$*" == *"--jq"* ]] && [[ "$*" == *".private"* ]]; then
        echo "true"
    elif [[ "$*" == *"--jq"* ]] && [[ "$*" == *".default_branch"* ]]; then
        echo "main"
    else
        echo '{"private":true,"default_branch":"main"}'
    fi
    exit 0
fi

# gh pr list -> no existing PRs
if [[ "$1" == "pr" ]] && [[ "$2" == "list" ]]; then
    echo ""
    exit 0
fi

# gh pr close / other pr commands
if [[ "$1" == "pr" ]]; then
    exit 0
fi

# Content fetch -- should not be reached for private repos
if [[ "$*" == *"/contents/"* ]]; then
    echo "UNEXPECTED_CONTENT_FETCH" >> "$LOGFILE"
    echo "Not Found" >&2
    exit 1
fi

# gh pr create should NOT be called
if [[ "$1" == "pr" ]] && [[ "$2" == "create" ]]; then
    echo "UNEXPECTED_PR_CREATE: $*" >> "$LOGFILE"
    echo "https://github.com/private-org/private-repo/pull/999"
    exit 0
fi

# Default for API calls (refs, etc)
if [[ "$*" == *"/git/refs"* ]]; then
    echo '{"ref":"refs/heads/foo"}'
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
			[]byte("repos:\n  private-repo:\n    enabled: true\n"), 0o644))
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

		// Script should succeed (skipping is not a failure)
		assert.NoError(t, err, "reconcile should succeed even when skipping private repos; output: %s", outputStr)

		// Read the API log
		logBytes, err := os.ReadFile(apiLog)
		require.NoError(t, err)
		logStr := string(logBytes)

		// No PR should have been created
		assert.NotContains(t, logStr, "UNEXPECTED_PR_CREATE", "no PR should be created for a private repo")
		assert.NotContains(t, logStr, "pr create", "no PR creation should be attempted for private repos")

		// Output should indicate the repo was skipped due to being private
		outputLower := strings.ToLower(outputStr)
		assert.True(t,
			strings.Contains(outputLower, "skip") || strings.Contains(outputLower, "private"),
			"output should indicate private repo was skipped; got: %s", outputStr)
	})
}
