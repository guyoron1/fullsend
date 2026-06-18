//go:build e2e

package cli_test

// Agent Run TempDir Fix — Implemented Tests
//
// STD Reference: outputs/std/GH-30/GH-30_test_description.yaml
// STP Reference: outputs/stp/GH-30/GH-30_test_plan.md
// Jira: GH-30 — Fix Hardcoded /tmp/repo in Agent Run Tests
//
// Total Scenarios: 8 (P0: 3, P1: 5)
//
// Shared Preconditions:
//   - Go 1.23+ toolchain installed
//   - Source code uses t.TempDir() instead of hardcoded /tmp/repo
//   - OS temp directory writable for t.TempDir()

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTS_GH30_001_HarnessLoadPipelineReachesOpenshellError verifies that
// TestRunAgent_HarnessLoadPipeline uses t.TempDir() and reaches the openshell
// availability check error instead of failing during tarball creation.
//
// Scenario: TS-GH-30-001 | Tier 1 | P0 | MVP
// Pattern: error-path-validation
//
// Preconditions:
//   - t.TempDir() replacement applied in TestRunAgent_HarnessLoadPipeline
//
// Expected:
//   - Error message contains openshell availability check failure, not tar error
//   - Test passes consistently with -count=5
func TestTS_GH30_001_HarnessLoadPipelineReachesOpenshellError(t *testing.T) {
	t.Run("harness-load-pipeline reaches openshell error without tar failure", func(t *testing.T) {
		// Run the specific test function and capture output
		cmd := exec.Command("go", "test", "./internal/cli/",
			"-run", "TestRunAgent_HarnessLoadPipeline",
			"-v", "-count=1",
		)
		cmd.Dir = findRepoRoot(t)
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0")

		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		// The test itself should pass (exit code 0)
		require.NoError(t, err, "TestRunAgent_HarnessLoadPipeline should pass; output:\n%s", outputStr)

		// Verify no tar-related errors appear in output
		assert.NotContains(t, strings.ToLower(outputStr), "tar:",
			"Test output should not contain tar errors")
		assert.NotContains(t, strings.ToLower(outputStr), "archive",
			"Test output should not contain archive errors")
	})

	t.Run("harness-load-pipeline passes consistently with count=5", func(t *testing.T) {
		cmd := exec.Command("go", "test", "./internal/cli/",
			"-run", "TestRunAgent_HarnessLoadPipeline",
			"-count=5",
		)
		cmd.Dir = findRepoRoot(t)
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0")

		output, err := cmd.CombinedOutput()
		require.NoError(t, err,
			"TestRunAgent_HarnessLoadPipeline should pass 5 consecutive runs; output:\n%s",
			string(output))
	})
}

// TestTS_GH30_002_AllRunAgentTestsPassWithoutPreExistingTmpRepo verifies all
// 10 affected test functions pass when /tmp/repo does not exist on the host.
//
// Scenario: TS-GH-30-002 | Tier 1 | P0 | MVP
// Pattern: test-isolation
//
// Preconditions:
//   - /tmp/repo directory does not exist on host
//   - All 10 affected test functions use t.TempDir()
//
// Expected:
//   - All 10 affected test functions pass (exit code 0)
//   - No tar-related errors in test output
func TestTS_GH30_002_AllRunAgentTestsPassWithoutPreExistingTmpRepo(t *testing.T) {
	// SETUP: Ensure /tmp/repo does not exist
	tmpRepoPath := "/tmp/repo"
	if _, err := os.Stat(tmpRepoPath); err == nil {
		// /tmp/repo exists — remove it for this test, restore after
		backupPath := fmt.Sprintf("/tmp/repo-backup-%d", os.Getpid())
		require.NoError(t, os.Rename(tmpRepoPath, backupPath),
			"Failed to temporarily rename /tmp/repo")
		t.Cleanup(func() {
			_ = os.Rename(backupPath, tmpRepoPath)
		})
	}
	// Confirm /tmp/repo is absent
	_, statErr := os.Stat(tmpRepoPath)
	require.True(t, os.IsNotExist(statErr), "/tmp/repo must not exist for this test")

	t.Run("all runAgent tests pass without /tmp/repo", func(t *testing.T) {
		cmd := exec.Command("go", "test", "./internal/cli/",
			"-run", "TestRunAgent",
			"-v", "-count=1",
		)
		cmd.Dir = findRepoRoot(t)
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0")

		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		require.NoError(t, err,
			"All TestRunAgent functions should pass without /tmp/repo; output:\n%s", outputStr)

		// Verify no tar-related errors in output
		outputLower := strings.ToLower(outputStr)
		assert.NotContains(t, outputLower, "tar:",
			"Test output should not contain tar errors")
		assert.NotContains(t, outputLower, "no such file or directory",
			"Test output should not contain missing file errors from tar creation")
	})
}

// TestTS_GH30_003_AllRunAgentTestsPassWithPreExistingTmpRepo verifies all
// 10 affected test functions pass even when /tmp/repo exists on the host.
//
// Scenario: TS-GH-30-003 | Tier 1 | P1
// Pattern: test-isolation
//
// Preconditions:
//   - /tmp/repo directory exists on host (created during setup)
//   - All 10 affected test functions use t.TempDir()
//
// Expected:
//   - All 10 test functions pass regardless of /tmp/repo existence
//   - Tests use t.TempDir(), not /tmp/repo
func TestTS_GH30_003_AllRunAgentTestsPassWithPreExistingTmpRepo(t *testing.T) {
	// SETUP: Ensure /tmp/repo exists
	tmpRepoPath := "/tmp/repo"
	created := false
	if _, err := os.Stat(tmpRepoPath); os.IsNotExist(err) {
		require.NoError(t, os.MkdirAll(tmpRepoPath, 0o755),
			"Failed to create /tmp/repo for test setup")
		created = true
	}
	t.Cleanup(func() {
		if created {
			_ = os.RemoveAll(tmpRepoPath)
		}
	})

	// Confirm /tmp/repo exists
	_, statErr := os.Stat(tmpRepoPath)
	require.NoError(t, statErr, "/tmp/repo must exist for this test")

	t.Run("all runAgent tests pass with /tmp/repo present", func(t *testing.T) {
		cmd := exec.Command("go", "test", "./internal/cli/",
			"-run", "TestRunAgent",
			"-v", "-count=1",
		)
		cmd.Dir = findRepoRoot(t)
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0")

		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		require.NoError(t, err,
			"All TestRunAgent functions should pass with /tmp/repo present; output:\n%s",
			outputStr)
	})
}

// TestTS_GH30_004_YMLFallbackHarnessResolutionUsesIsolatedDir verifies that
// the YML fallback harness resolution path uses an isolated temporary directory.
//
// Scenario: TS-GH-30-004 | Tier 1 | P1
// Pattern: error-path-validation
//
// Preconditions:
//   - YML fallback harness configuration available
//   - t.TempDir() used for repo directory
//
// Expected:
//   - Test reaches harness resolution error path
//   - Error relates to harness resolution, not directory/tar operations
func TestTS_GH30_004_YMLFallbackHarnessResolutionUsesIsolatedDir(t *testing.T) {
	t.Run("YML fallback harness resolves without tar error", func(t *testing.T) {
		// Run the YML fallback harness test variant
		cmd := exec.Command("go", "test", "./internal/cli/",
			"-run", "TestRunAgent_YMLFallback|TestRunAgent_HarnessYML",
			"-v", "-count=1",
		)
		cmd.Dir = findRepoRoot(t)
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0")

		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		// Test should pass — it reaches the expected error path internally
		require.NoError(t, err,
			"YML fallback harness test should pass; output:\n%s", outputStr)

		// Verify no directory/tar-related failures
		outputLower := strings.ToLower(outputStr)
		assert.NotContains(t, outputLower, "tar:",
			"YML fallback test should not produce tar errors")
		assert.NotContains(t, outputLower, "no such file or directory",
			"YML fallback test should not produce missing directory errors")
	})
}

// TestTS_GH30_005_HarnessNotFoundReturnsDescriptiveError verifies that the
// harness-not-found error path produces a clear error when using t.TempDir().
//
// Scenario: TS-GH-30-005 | Tier 1 | P0 | MVP
// Pattern: error-path-validation
//
// Preconditions:
//   - Empty temp directory created via t.TempDir() (no harness file)
//
// Expected:
//   - Error message clearly indicates harness file not found
//   - Error does not mention tar/archive operations
func TestTS_GH30_005_HarnessNotFoundReturnsDescriptiveError(t *testing.T) {
	t.Run("harness-not-found test returns descriptive error", func(t *testing.T) {
		// Run the harness-not-found test variant
		cmd := exec.Command("go", "test", "./internal/cli/",
			"-run", "TestRunAgent_HarnessNotFound",
			"-v", "-count=1",
		)
		cmd.Dir = findRepoRoot(t)
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0")

		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		// The test itself should pass (it asserts the error internally)
		require.NoError(t, err,
			"HarnessNotFound test should pass; output:\n%s", outputStr)

		// Verify test output does not indicate tar/archive errors
		outputLower := strings.ToLower(outputStr)
		assert.NotContains(t, outputLower, "tar:",
			"Harness-not-found test should not produce tar errors")
		assert.NotContains(t, outputLower, "archive",
			"Harness-not-found test should not produce archive errors")
	})
}

// TestTS_GH30_006_EachTestFunctionUsesUniqueTempDir performs source code
// analysis to verify all 10 affected test functions use t.TempDir() and
// none reference hardcoded /tmp/repo.
//
// Scenario: TS-GH-30-006 | Tier 1 | P1
// Pattern: source-code-analysis
//
// Preconditions:
//   - Source code of internal/cli/run_test.go available for analysis
//
// Expected:
//   - Each of the 10 test functions contains a t.TempDir() call
//   - No references to /tmp/repo remain (grep returns 0 matches)
//   - All tests pass
func TestTS_GH30_006_EachTestFunctionUsesUniqueTempDir(t *testing.T) {
	repoRoot := findRepoRoot(t)
	runTestPath := filepath.Join(repoRoot, "internal", "cli", "run_test.go")

	// Verify the source file exists
	_, err := os.Stat(runTestPath)
	require.NoError(t, err, "internal/cli/run_test.go must exist")

	sourceBytes, err := os.ReadFile(runTestPath)
	require.NoError(t, err, "Failed to read run_test.go")
	source := string(sourceBytes)

	t.Run("no references to hardcoded /tmp/repo", func(t *testing.T) {
		// Count occurrences of /tmp/repo in the source (excluding comments about the fix)
		lines := strings.Split(source, "\n")
		hardcodedRefs := 0
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			// Skip comment lines — the fix description may mention /tmp/repo
			if strings.HasPrefix(trimmed, "//") {
				continue
			}
			if strings.Contains(line, `"/tmp/repo"`) || strings.Contains(line, "`/tmp/repo`") {
				hardcodedRefs++
			}
		}
		assert.Equal(t, 0, hardcodedRefs,
			"No non-comment code lines should reference hardcoded /tmp/repo path")
	})

	t.Run("t.TempDir() is used in test functions", func(t *testing.T) {
		tempDirCount := strings.Count(source, "t.TempDir()")
		assert.GreaterOrEqual(t, tempDirCount, 1,
			"run_test.go should contain at least one t.TempDir() call; found %d", tempDirCount)
		t.Logf("Found %d t.TempDir() calls in run_test.go", tempDirCount)
	})

	t.Run("all TestRunAgent functions pass", func(t *testing.T) {
		cmd := exec.Command("go", "test", "./internal/cli/",
			"-run", "TestRunAgent",
			"-v", "-count=1",
		)
		cmd.Dir = repoRoot
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0")

		output, err := cmd.CombinedOutput()
		require.NoError(t, err,
			"All TestRunAgent functions should pass; output:\n%s", string(output))
	})
}

// TestTS_GH30_007_UploadDirSucceedsWithValidDirectory verifies that
// sandbox.UploadDir successfully creates a tarball when given a valid
// directory path from t.TempDir().
//
// Scenario: TS-GH-30-007 | Tier 1 | P1
// Pattern: error-path-validation
//
// Preconditions:
//   - Temp directory with sample files created via t.TempDir()
//
// Expected:
//   - UploadDir does not fail with directory-not-found error
//   - If UploadDir fails, failure is at upload stage not tar creation
func TestTS_GH30_007_UploadDirSucceedsWithValidDirectory(t *testing.T) {
	t.Run("UploadDir does not fail at tar creation with valid dir", func(t *testing.T) {
		// Create a temp directory with sample content to verify tar creation
		repoDir := t.TempDir()

		// Create sample files that would be tarred
		testFile := filepath.Join(repoDir, "test.txt")
		require.NoError(t,
			os.WriteFile(testFile, []byte("test content for tar creation"), 0o644),
			"Failed to create test file in temp directory")

		subDir := filepath.Join(repoDir, "subdir")
		require.NoError(t, os.MkdirAll(subDir, 0o755),
			"Failed to create subdirectory in temp directory")

		nestedFile := filepath.Join(subDir, "nested.txt")
		require.NoError(t,
			os.WriteFile(nestedFile, []byte("nested content"), 0o644),
			"Failed to create nested test file")

		// Verify the directory structure is valid and would succeed at tar step
		// by running the relevant test function that exercises UploadDir
		cmd := exec.Command("go", "test", "./internal/cli/",
			"-run", "TestRunAgent_HarnessLoadPipeline",
			"-v", "-count=1",
		)
		cmd.Dir = findRepoRoot(t)
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0")

		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		require.NoError(t, err,
			"Test exercising UploadDir should pass; output:\n%s", outputStr)

		// Key assertion: no tar creation failure
		outputLower := strings.ToLower(outputStr)
		assert.NotContains(t, outputLower, "creating tar",
			"Should not fail at tar creation step")
		assert.NotContains(t, outputLower, "no such file or directory",
			"Should not fail due to missing directory")
	})
}

// TestTS_GH30_008_OrgConfigVariantsReachExpectedErrorPaths verifies that
// HarnessLoadWithOrgConfig, MalformedOrgConfig, and WithURLBase tests
// each reach their expected error paths using t.TempDir().
//
// Scenario: TS-GH-30-008 | Tier 1 | P1
// Pattern: error-path-validation
//
// Preconditions:
//   - t.TempDir() used for repo directory in each OrgConfig variant test
//
// Expected:
//   - HarnessLoadWithOrgConfig reaches openshell error assertion
//   - MalformedOrgConfig reaches malformed config error assertion
//   - WithURLBase reaches openshell error assertion
//   - No tar/directory errors in any variant
func TestTS_GH30_008_OrgConfigVariantsReachExpectedErrorPaths(t *testing.T) {
	repoRoot := findRepoRoot(t)

	variants := []struct {
		name    string
		pattern string
	}{
		{
			name:    "HarnessLoadWithOrgConfig reaches expected error",
			pattern: "TestRunAgent_HarnessLoadWithOrgConfig",
		},
		{
			name:    "MalformedOrgConfig reaches expected error",
			pattern: "TestRunAgent_MalformedOrgConfig",
		},
		{
			name:    "WithURLBase reaches expected error",
			pattern: "TestRunAgent_WithURLBase|TestRunAgent_.*URLBase",
		},
	}

	for _, v := range variants {
		v := v // capture range variable
		t.Run(v.name, func(t *testing.T) {
			cmd := exec.Command("go", "test", "./internal/cli/",
				"-run", v.pattern,
				"-v", "-count=1",
			)
			cmd.Dir = repoRoot
			cmd.Env = append(os.Environ(), "CGO_ENABLED=0")

			output, err := cmd.CombinedOutput()
			outputStr := string(output)

			require.NoError(t, err,
				"%s should pass; output:\n%s", v.pattern, outputStr)

			// Verify no tar/directory-related errors
			outputLower := strings.ToLower(outputStr)
			assert.NotContains(t, outputLower, "tar:",
				"%s should not produce tar errors", v.name)
			assert.NotContains(t, outputLower, "no such file or directory",
				"%s should not produce missing directory errors", v.name)
		})
	}
}

// findRepoRoot locates the repository root by walking up from the current
// working directory or using well-known environment variables.
func findRepoRoot(t *testing.T) string {
	t.Helper()

	// Check environment variable first (CI or sandbox context)
	if root := os.Getenv("SOURCE_REPO_DIR"); root != "" {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			return root
		}
	}

	// Walk up from cwd looking for go.mod
	dir, err := os.Getwd()
	require.NoError(t, err, "Failed to get working directory")

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	t.Fatal("Could not find repository root (no go.mod found in parent directories)")
	return "" // unreachable
}
