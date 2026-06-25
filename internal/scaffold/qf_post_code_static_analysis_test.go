package scaffold

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestPostCodeShellcheckClean covers STD scenario 11 (TS-GH84-011).
//
// The modified post-code.sh script must pass shellcheck static analysis
// without producing new warnings. This validates that the workflow file
// detection code follows shell scripting best practices.
func TestPostCodeShellcheckClean(t *testing.T) {
	scriptPath := filepath.Join("fullsend-repo", "scripts", "post-code.sh")
	requireFileExists(t, scriptPath)

	// Check if shellcheck is available; skip if not installed.
	shellcheckPath, err := exec.LookPath("shellcheck")
	if err != nil {
		t.Skip("shellcheck not available on PATH — install shellcheck to run this test")
	}

	// Run shellcheck on post-code.sh.
	cmd := exec.Command(shellcheckPath, scriptPath)
	output, err := cmd.CombinedOutput()

	// ASSERT-01: shellcheck exits with code 0.
	assert.NoError(t, err,
		"shellcheck must exit cleanly (code 0) for post-code.sh; output:\n%s", string(output))

	if err != nil {
		// Log the full shellcheck output for debugging.
		t.Logf("shellcheck output:\n%s", string(output))
	}
}

// TestPostCodeTestShellcheckClean is a bonus check — verifies the test script
// itself also passes shellcheck.
func TestPostCodeTestShellcheckClean(t *testing.T) {
	scriptPath := filepath.Join("fullsend-repo", "scripts", "post-code-test.sh")
	requireFileExists(t, scriptPath)

	shellcheckPath, err := exec.LookPath("shellcheck")
	if err != nil {
		t.Skip("shellcheck not available on PATH — install shellcheck to run this test")
	}

	cmd := exec.Command(shellcheckPath, scriptPath)
	output, err := cmd.CombinedOutput()

	assert.NoError(t, err,
		"shellcheck must exit cleanly for post-code-test.sh; output:\n%s", string(output))

	if err != nil {
		t.Logf("shellcheck output:\n%s", string(output))
	}
}

// requireFileExists is defined in qf_post_code_workflow_block_test.go (same package).
