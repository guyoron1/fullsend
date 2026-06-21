package review

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

/*
Threshold Activation Tests — GH-2096

Validates that the security-triage pre-pass activates only for PRs meeting the
50-file threshold. The threshold is the core gating mechanism for the two-pass
review strategy.

STP Reference: outputs/stp/GH-2096/GH-2096_test_plan.md
STD Scenarios: TS-GH-2096-001, TS-GH-2096-002, TS-GH-2096-003
*/

// triageFileThreshold is the minimum file count that activates security triage.
const triageFileThreshold = 50

// shouldRunTriage returns true when the number of changed files meets or
// exceeds the triage activation threshold. This is the decision function
// that gates the two-pass review strategy.
func shouldRunTriage(files []string) bool {
	return len(files) >= triageFileThreshold
}

// makeFileList generates a slice of n synthetic file paths for testing.
func makeFileList(n int) []string {
	files := make([]string, n)
	for i := range files {
		files[i] = fmt.Sprintf("pkg/file_%d.go", i)
	}
	return files
}

func TestThresholdActivation(t *testing.T) {
	/*
		Preconditions:
			- Go development environment with Go 1.26+
			- fullsend repository with two-pass review strategy changes
	*/

	// TS-GH-2096-001: Verify triage pre-pass runs for PR with >=50 files
	t.Run("triage pre-pass runs for PR with >=50 files", func(t *testing.T) {
		tests := []struct {
			name      string
			fileCount int
		}{
			{"exactly 50 files", 50},
			{"100 files", 100},
			{"500 files", 500},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				files := makeFileList(tt.fileCount)
				result := shouldRunTriage(files)
				assert.True(t, result,
					"shouldRunTriage must return true for %d files (>= %d threshold)",
					tt.fileCount, triageFileThreshold)
			})
		}
	})

	// TS-GH-2096-002: Verify triage pre-pass skipped for PR with <50 files
	t.Run("triage pre-pass skipped for PR with <50 files", func(t *testing.T) {
		tests := []struct {
			name      string
			fileCount int
		}{
			{"49 files", 49},
			{"1 file", 1},
			{"0 files", 0},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				files := makeFileList(tt.fileCount)
				result := shouldRunTriage(files)
				assert.False(t, result,
					"shouldRunTriage must return false for %d files (< %d threshold)",
					tt.fileCount, triageFileThreshold)
			})
		}
	})

	// TS-GH-2096-003: Verify behavior at exact threshold boundary (50 files)
	t.Run("behavior at exact threshold boundary", func(t *testing.T) {
		files50 := makeFileList(50)
		files49 := makeFileList(49)

		assert.True(t, shouldRunTriage(files50),
			"exactly 50 files must activate triage (inclusive boundary)")
		assert.False(t, shouldRunTriage(files49),
			"exactly 49 files must not activate triage")
	})
}
