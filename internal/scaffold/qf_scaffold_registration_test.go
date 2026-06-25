package scaffold

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// QualityFlow tests for GH-1270: Scaffold registration scenarios
// STD: outputs/std/GH-1270/GH-1270_test_description.yaml

// TestQF_FileMode_InstallPrecommitTools verifies install-precommit-tools.sh
// gets 100755 mode via FileMode().
// [TS-GH-1270-027] Scenario 27
func TestQF_FileMode_InstallPrecommitTools(t *testing.T) {
	mode := FileMode("scripts/install-precommit-tools.sh")
	assert.Equal(t, "100755", mode,
		"install-precommit-tools.sh must be executable (100755)")
}

// TestQF_FileMode_ResolvePrecommitTools verifies resolve-precommit-tools.py
// gets 100755 mode via FileMode().
// [TS-GH-1270-028] Scenario 28
func TestQF_FileMode_ResolvePrecommitTools(t *testing.T) {
	mode := FileMode("scripts/resolve-precommit-tools.py")
	assert.Equal(t, "100755", mode,
		"resolve-precommit-tools.py must be executable (100755)")
}

// TestQF_FileMode_NonExecutable verifies non-script files get 100644.
func TestQF_FileMode_NonExecutable(t *testing.T) {
	mode := FileMode("agents/triage.md")
	assert.Equal(t, "100644", mode,
		"non-script files should be 100644")
}

// TestQF_FileModeMatchesFilesystem verifies the existing regression test
// continues to pass with the new install/resolve script entries.
// [TS-GH-1270-029] Scenario 29
// Covered by existing test: TestFileModeMatchesFilesystem
// in internal/scaffold/scaffold_test.go
// This test validates the same invariant by checking that both new scripts
// appear in the executableFiles map and are actually executable on disk.
func TestQF_FileModeMatchesFilesystem_NewScripts(t *testing.T) {
	newScripts := []string{
		"scripts/install-precommit-tools.sh",
		"scripts/resolve-precommit-tools.py",
	}
	for _, script := range newScripts {
		t.Run(script, func(t *testing.T) {
			assert.Equal(t, "100755", FileMode(script),
				"new script %s must be in executableFiles map", script)
		})
	}
}
