package scaffold

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/forge"
)

// =============================================================================
// TS-GH-2351-001: ListRepositoryFiles returns all blob paths from recursive tree
// Tier: 1 | Priority: P0 | MVP: true
// =============================================================================

func TestListRepositoryFiles_ReturnsAllBlobPaths(t *testing.T) {
	// Arrange: create FakeClient with FileContents map containing multiple files
	client := forge.NewFakeClient()
	client.FileContents = map[string][]byte{
		"myorg/myrepo/cmd/main.go":           []byte("package main"),
		"myorg/myrepo/internal/foo/bar.go":    []byte("package foo"),
		"myorg/myrepo/README.md":              []byte("# README"),
	}

	// Act: call ListRepositoryFiles with valid owner and repo
	paths, err := client.ListRepositoryFiles(context.Background(), "myorg", "myrepo")

	// Assert: returned paths match expected file paths
	require.NoError(t, err, "ListRepositoryFiles should not error for valid repos")
	sort.Strings(paths)
	assert.Equal(t, []string{
		"README.md",
		"cmd/main.go",
		"internal/foo/bar.go",
	}, paths, "all file paths should be returned with owner/repo prefix stripped")
}

// =============================================================================
// TS-GH-2351-002: ListRepositoryFiles returns error for truncated tree response
// Tier: 1 | Priority: P0 | MVP: true
// =============================================================================

func TestListRepositoryFiles_ErrorOnTruncatedTree(t *testing.T) {
	// Arrange: configure FakeClient to return truncation error
	client := forge.NewFakeClient()
	client.Errors["ListRepositoryFiles"] = fmt.Errorf("repository tree too large (truncated)")

	// Act: call ListRepositoryFiles
	paths, err := client.ListRepositoryFiles(context.Background(), "myorg", "myrepo")

	// Assert: error is returned containing "truncated"
	require.Error(t, err, "should return error for truncated tree")
	assert.Contains(t, err.Error(), "truncated", "error should indicate truncation")
	assert.Nil(t, paths, "no partial path list should be returned")
}

// =============================================================================
// TS-GH-2351-003: ListRepositoryFiles returns empty for nonexistent repository
// Tier: 1 | Priority: P0 | MVP: true
// =============================================================================

func TestListRepositoryFiles_EmptyForNonexistentRepo(t *testing.T) {
	// Arrange: create FakeClient with files for a different repo only
	client := forge.NewFakeClient()
	client.FileContents = map[string][]byte{
		"other/repo/file.go": []byte("content"),
	}

	// Act: call ListRepositoryFiles with nonexistent owner/repo
	paths, err := client.ListRepositoryFiles(context.Background(), "nonexistent", "repo")

	// Assert: returns empty result for nonexistent repo
	require.NoError(t, err, "FakeClient returns no error for non-matching repo")
	assert.Empty(t, paths, "should return empty paths for nonexistent repo")
}

// =============================================================================
// TS-GH-2351-004: All paths reported present when all exist in repo
// Tier: 1 | Priority: P0 | MVP: true
// =============================================================================

func TestComparePathPresence_AllPresent_GH2351(t *testing.T) {
	// Arrange: FakeClient with files matching all expected paths
	client := &forge.FakeClient{
		FileContents: map[string][]byte{
			"myorg/myrepo/cmd/main.go":        []byte("package main"),
			"myorg/myrepo/internal/foo/bar.go": []byte("package foo"),
			"myorg/myrepo/README.md":           []byte("# README"),
		},
	}

	expectedPaths := []string{
		"cmd/main.go",
		"internal/foo/bar.go",
		"README.md",
	}

	// Act: call ComparePathPresence
	missing, err := ComparePathPresence(context.Background(), client, "myorg", "myrepo", expectedPaths)

	// Assert: no error, no missing paths
	require.NoError(t, err, "should not error when all paths exist")
	assert.Empty(t, missing, "no paths should be reported as missing")
}

// =============================================================================
// TS-GH-2351-005: Correct missing paths returned when some are absent
// Tier: 1 | Priority: P0 | MVP: true
// =============================================================================

func TestComparePathPresence_SomeMissing_GH2351(t *testing.T) {
	// Arrange: FakeClient with only some expected paths
	client := &forge.FakeClient{
		FileContents: map[string][]byte{
			"myorg/myrepo/cmd/main.go": []byte("package main"),
			"myorg/myrepo/README.md":   []byte("# README"),
		},
	}

	allPaths := []string{
		"cmd/main.go",
		"README.md",
		"CONTRIBUTING.md",
		"docs/guide.md",
	}

	// Act: call ComparePathPresence
	missing, err := ComparePathPresence(context.Background(), client, "myorg", "myrepo", allPaths)

	// Assert: exactly the absent paths are returned as missing
	require.NoError(t, err, "should not error for valid input")
	assert.ElementsMatch(t, []string{"CONTRIBUTING.md", "docs/guide.md"}, missing,
		"missing slice should contain exactly the absent paths")
}

// =============================================================================
// TS-GH-2351-006: All paths reported missing for empty repository
// Tier: 1 | Priority: P0 | MVP: true
// =============================================================================

func TestComparePathPresence_AllMissingEmptyRepo_GH2351(t *testing.T) {
	// Arrange: FakeClient with empty FileContents
	client := &forge.FakeClient{
		FileContents: map[string][]byte{},
	}

	expectedPaths := []string{
		"cmd/main.go",
		"internal/foo/bar.go",
		"README.md",
	}

	// Act: call ComparePathPresence
	missing, err := ComparePathPresence(context.Background(), client, "myorg", "myrepo", expectedPaths)

	// Assert: all expected paths reported as missing
	require.NoError(t, err, "empty repos are valid — should not error")
	assert.ElementsMatch(t, expectedPaths, missing, "all paths should be missing for empty repo")
}

// =============================================================================
// TS-GH-2351-007: Empty input returns nil without API calls
// Tier: 1 | Priority: P0 | MVP: true
// =============================================================================

func TestComparePathPresence_EmptyInputReturnsNil_GH2351(t *testing.T) {
	// Arrange: FakeClient with error injection — if ListRepositoryFiles
	// were called, it would error, proving the short-circuit works
	client := forge.NewFakeClient()
	client.Errors["ListRepositoryFiles"] = errors.New("should not be called")

	// Act: call ComparePathPresence with nil expected paths
	missing, err := ComparePathPresence(context.Background(), client, "myorg", "myrepo", nil)

	// Assert: nil result without calling ListRepositoryFiles
	assert.Nil(t, missing, "missing paths should be nil for empty input")
	assert.Nil(t, err, "error should be nil for empty input")

	// Also test with empty slice
	missing2, err2 := ComparePathPresence(context.Background(), client, "myorg", "myrepo", []string{})
	assert.Nil(t, missing2, "missing paths should be nil for empty slice input")
	assert.Nil(t, err2, "error should be nil for empty slice input")
}

// =============================================================================
// TS-GH-2351-008: Error propagation when ListRepositoryFiles fails
// Tier: 1 | Priority: P0 | MVP: true
// =============================================================================

func TestComparePathPresence_ErrorPropagation_GH2351(t *testing.T) {
	// Arrange: FakeClient with injected ListRepositoryFiles error
	injectedErr := errors.New("network timeout")
	client := forge.NewFakeClient()
	client.Errors["ListRepositoryFiles"] = injectedErr

	// Act: call ComparePathPresence with valid expected paths
	missing, err := ComparePathPresence(context.Background(), client, "myorg", "myrepo", []string{
		"cmd/main.go",
		"README.md",
	})

	// Assert: error is propagated from ListRepositoryFiles
	require.Error(t, err, "error from ListRepositoryFiles must be propagated")
	assert.True(t, errors.Is(err, injectedErr),
		"propagated error should wrap the original injected error")
	assert.Contains(t, err.Error(), "listing repository files",
		"error should include ComparePathPresence context")
	assert.Nil(t, missing, "missing paths should be nil when error occurs")
}

// =============================================================================
// TS-GH-2351-009: GetFileContent is never called by ComparePathPresence (guard)
// Tier: 1 | Priority: P0 | MVP: true
// =============================================================================

func TestComparePathPresence_UsesOneAPICall_GH2351(t *testing.T) {
	// Arrange: inject error on GetFileContent to ensure it is never called.
	// If ComparePathPresence regresses to per-path GetFileContent calls,
	// this test will fail with the injected error.
	client := &forge.FakeClient{
		FileContents: map[string][]byte{
			"org/repo/path-a": []byte("a"),
			"org/repo/path-b": []byte("b"),
			"org/repo/path-c": []byte("c"),
		},
		Errors: map[string]error{
			"GetFileContent": errors.New("should not be called — O(N) pattern regression"),
		},
	}

	// Act: call ComparePathPresence with several expected paths
	missing, err := ComparePathPresence(context.Background(), client, "org", "repo", []string{
		"path-a",
		"path-b",
		"path-c",
		"path-d",
	})

	// Assert: succeeds (GetFileContent was never called)
	require.NoError(t, err, "GetFileContent should not be called — batch pattern expected")
	assert.Equal(t, []string{"path-d"}, missing,
		"only truly missing paths should be reported")
}

// =============================================================================
// TS-GH-2351-010: Single ListRepositoryFiles call replaces N GetFileContent calls
// Tier: 1 | Priority: P0 | MVP: true
// =============================================================================

func TestComparePathPresence_SingleCallForManyPaths_GH2351(t *testing.T) {
	// Arrange: FakeClient with 50+ file entries
	client := forge.NewFakeClient()
	const numFiles = 60
	const numExpected = 70 // 60 present + 10 absent
	for i := 0; i < numFiles; i++ {
		key := fmt.Sprintf("org/repo/path/to/file_%03d.go", i)
		client.FileContents[key] = []byte("content")
	}
	// Also inject GetFileContent error as guard
	client.Errors["GetFileContent"] = errors.New("should not be called")

	// Build expected paths: 60 present + 10 absent
	expected := make([]string, 0, numExpected)
	for i := 0; i < numFiles; i++ {
		expected = append(expected, fmt.Sprintf("path/to/file_%03d.go", i))
	}
	absentPaths := make([]string, 0, 10)
	for i := numFiles; i < numExpected; i++ {
		p := fmt.Sprintf("path/to/file_%03d.go", i)
		expected = append(expected, p)
		absentPaths = append(absentPaths, p)
	}
	sort.Strings(absentPaths) // ComparePathPresence sorts missing

	// Act: call ComparePathPresence with many expected paths
	missing, err := ComparePathPresence(context.Background(), client, "org", "repo", expected)

	// Assert: correct results for large path set with O(1) API calls
	require.NoError(t, err, "batch pattern must handle many paths without error")
	assert.Equal(t, absentPaths, missing,
		"missing should contain exactly the 10 absent paths")
}

// =============================================================================
// TS-GH-2351-011: FakeClient returns paths matching owner/repo/ prefix
// Tier: 1 | Priority: P1
// =============================================================================

func TestFakeClient_ListRepositoryFiles_PrefixFiltering(t *testing.T) {
	// Arrange: FakeClient with files for multiple repos
	client := &forge.FakeClient{
		FileContents: map[string][]byte{
			"org1/repo1/file1.go": []byte("content"),
			"org1/repo1/file2.go": []byte("content"),
			"org2/repo2/other.go": []byte("content"),
		},
	}

	// Act: call ListRepositoryFiles for org1/repo1 only
	paths, err := client.ListRepositoryFiles(context.Background(), "org1", "repo1")

	// Assert: only org1/repo1 paths returned, org2/repo2 excluded
	require.NoError(t, err)
	sort.Strings(paths)
	assert.Equal(t, []string{"file1.go", "file2.go"}, paths,
		"only paths from requested repo should be returned")
	assert.NotContains(t, paths, "other.go",
		"paths from other repos must be excluded")
}

// =============================================================================
// TS-GH-2351-012: FakeClient returns empty slice for no matching files
// Tier: 1 | Priority: P1
// =============================================================================

func TestFakeClient_ListRepositoryFiles_NoMatch(t *testing.T) {
	// Arrange: FakeClient with files for an unrelated repo
	client := &forge.FakeClient{
		FileContents: map[string][]byte{
			"other/repo/file.go": []byte("content"),
		},
	}

	// Act: call ListRepositoryFiles for non-matching repo
	paths, err := client.ListRepositoryFiles(context.Background(), "target", "repo")

	// Assert: empty slice returned, no error
	require.NoError(t, err, "no-match is not an error condition")
	assert.Empty(t, paths, "should return empty slice for non-matching repo")
}

// =============================================================================
// TS-GH-2351-013: FakeClient returns injected error when configured
// Tier: 1 | Priority: P1
// =============================================================================

func TestFakeClient_ListRepositoryFiles_InjectedError(t *testing.T) {
	// Arrange: FakeClient with ListRepositoryFiles error injection
	sentinelErr := errors.New("simulated API failure")
	client := forge.NewFakeClient()
	client.Errors["ListRepositoryFiles"] = sentinelErr

	// Act: call ListRepositoryFiles
	paths, err := client.ListRepositoryFiles(context.Background(), "myorg", "myrepo")

	// Assert: injected error is returned
	require.Error(t, err, "injected error must be returned")
	assert.True(t, errors.Is(err, sentinelErr),
		"returned error should be the injected sentinel error")
	assert.Nil(t, paths, "paths should be nil when error is returned")
}

// =============================================================================
// TS-GH-2351-014: No data races with 20 concurrent goroutines
// Tier: 1 | Priority: P2
// =============================================================================

func TestFakeClient_ListRepositoryFiles_ThreadSafe(t *testing.T) {
	// Arrange: shared FakeClient with FileContents
	client := forge.NewFakeClient()
	client.FileContents = map[string][]byte{
		"org/repo/file1.go":     []byte("content1"),
		"org/repo/file2.go":     []byte("content2"),
		"org/repo/dir/file3.go": []byte("content3"),
	}

	const goroutines = 20
	var wg sync.WaitGroup
	errs := make([]error, goroutines)
	results := make([][]string, goroutines)

	// Act: launch 20 concurrent goroutines calling ListRepositoryFiles
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			paths, err := client.ListRepositoryFiles(context.Background(), "org", "repo")
			errs[idx] = err
			sort.Strings(paths)
			results[idx] = paths
		}(i)
	}
	wg.Wait()

	// Assert: all goroutines got correct results, no race detected (via -race flag)
	expected := []string{"dir/file3.go", "file1.go", "file2.go"}
	for i := 0; i < goroutines; i++ {
		assert.NoError(t, errs[i], "goroutine %d should not error", i)
		assert.Equal(t, expected, results[i],
			"goroutine %d should get correct sorted results", i)
	}
}
