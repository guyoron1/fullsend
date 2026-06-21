package github

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// TS-GH-2351-015: LiveClient follows refs → commit SHA → tree SHA pipeline
// Tier: 2 | Priority: P1
// =============================================================================

func TestLiveClient_ListRepositoryFiles_APIPipeline(t *testing.T) {
	// Arrange: mock HTTP server returning ref→commit→tree responses
	var callOrder []string

	mux := http.NewServeMux()

	// Step 1: GET /repos/{owner}/{repo} → default branch
	mux.HandleFunc("/repos/testorg/testrepo", func(w http.ResponseWriter, r *http.Request) {
		callOrder = append(callOrder, "repo")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"default_branch": "main",
		})
	})

	// Step 2: GET /repos/{owner}/{repo}/git/ref/heads/main → commit SHA
	mux.HandleFunc("/repos/testorg/testrepo/git/ref/heads/main", func(w http.ResponseWriter, r *http.Request) {
		callOrder = append(callOrder, "ref")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object": map[string]string{"sha": "abc123commit"},
		})
	})

	// Step 3: GET /repos/{owner}/{repo}/git/commits/{sha} → tree SHA
	mux.HandleFunc("/repos/testorg/testrepo/git/commits/abc123commit", func(w http.ResponseWriter, r *http.Request) {
		callOrder = append(callOrder, "commit")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"tree": map[string]string{"sha": "def456tree"},
		})
	})

	// Step 4: GET /repos/{owner}/{repo}/git/trees/{sha}?recursive=1 → file paths
	mux.HandleFunc("/repos/testorg/testrepo/git/trees/def456tree", func(w http.ResponseWriter, r *http.Request) {
		callOrder = append(callOrder, "tree")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"tree": []map[string]string{
				{"path": "cmd/main.go", "type": "blob"},
				{"path": "internal/foo/bar.go", "type": "blob"},
				{"path": "README.md", "type": "blob"},
			},
			"truncated": false,
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := New("test-token").WithBaseURL(server.URL)

	// Act: call LiveClient.ListRepositoryFiles
	paths, err := client.ListRepositoryFiles(t.Context(), "testorg", "testrepo")

	// Assert: correct paths returned, 4 API calls made in expected order
	require.NoError(t, err)
	sort.Strings(paths)
	assert.Equal(t, []string{"README.md", "cmd/main.go", "internal/foo/bar.go"}, paths)
	assert.Equal(t, []string{"repo", "ref", "commit", "tree"}, callOrder,
		"API calls should follow repo→ref→commit→tree pipeline")
}

// =============================================================================
// TS-GH-2351-016: LiveClient filters tree entries to blobs only
// Tier: 2 | Priority: P1
// =============================================================================

func TestLiveClient_ListRepositoryFiles_BlobsOnly(t *testing.T) {
	// Arrange: mock tree response with both blob and tree entries
	mux := http.NewServeMux()
	setupRepoAndRef(mux)

	mux.HandleFunc("/repos/testorg/testrepo/git/commits/abc123commit", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"tree": map[string]string{"sha": "def456tree"},
		})
	})

	mux.HandleFunc("/repos/testorg/testrepo/git/trees/def456tree", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"tree": []map[string]string{
				{"path": "cmd/main.go", "type": "blob"},
				{"path": "cmd", "type": "tree"},          // directory — should be excluded
				{"path": "internal", "type": "tree"},      // directory — should be excluded
				{"path": "internal/foo/bar.go", "type": "blob"},
				{"path": "internal/foo", "type": "tree"},  // directory — should be excluded
			},
			"truncated": false,
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := New("test-token").WithBaseURL(server.URL)

	// Act: call ListRepositoryFiles
	paths, err := client.ListRepositoryFiles(t.Context(), "testorg", "testrepo")

	// Assert: only blob-type paths returned, tree entries excluded
	require.NoError(t, err)
	sort.Strings(paths)
	assert.Equal(t, []string{"cmd/main.go", "internal/foo/bar.go"}, paths,
		"only blob-type entries should be returned")
	for _, p := range paths {
		assert.NotEqual(t, "cmd", p, "directory entries should be excluded")
		assert.NotEqual(t, "internal", p, "directory entries should be excluded")
		assert.NotEqual(t, "internal/foo", p, "directory entries should be excluded")
	}
}

// =============================================================================
// TS-GH-2351-017: LiveClient returns error when default branch ref lookup fails
// Tier: 2 | Priority: P1
// =============================================================================

func TestLiveClient_ListRepositoryFiles_RefLookupError(t *testing.T) {
	// Arrange: mock returns 404 for repo endpoint
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/testorg/nonexistent", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"Not Found"}`, http.StatusNotFound)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := New("test-token").WithBaseURL(server.URL)

	// Act: call ListRepositoryFiles with nonexistent repo
	paths, err := client.ListRepositoryFiles(t.Context(), "testorg", "nonexistent")

	// Assert: error is returned
	require.Error(t, err, "should return error when repo lookup fails")
	assert.Nil(t, paths, "paths should be nil on error")
}

// =============================================================================
// TS-GH-2351-018: LiveClient retries transient errors on branch ref lookup
// Tier: 2 | Priority: P1
// =============================================================================

func TestLiveClient_ListRepositoryFiles_RetriesTransientErrors(t *testing.T) {
	// Arrange: mock returns 502 on first ref request, 200 on second
	var refRequestCount int64

	mux := http.NewServeMux()
	setupRepoAndRef_WithRetry(mux, &refRequestCount)

	mux.HandleFunc("/repos/testorg/testrepo/git/commits/abc123commit", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"tree": map[string]string{"sha": "def456tree"},
		})
	})

	mux.HandleFunc("/repos/testorg/testrepo/git/trees/def456tree", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"tree": []map[string]string{
				{"path": "file.go", "type": "blob"},
			},
			"truncated": false,
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := New("test-token").WithBaseURL(server.URL)

	// Act: call ListRepositoryFiles
	paths, err := client.ListRepositoryFiles(t.Context(), "testorg", "testrepo")

	// Assert: succeeds after retry
	require.NoError(t, err, "should succeed after transient error clears")
	assert.Equal(t, []string{"file.go"}, paths)
	assert.GreaterOrEqual(t, atomic.LoadInt64(&refRequestCount), int64(2),
		"mock should have received multiple ref requests (retry happened)")
}

// =============================================================================
// Test Helpers
// =============================================================================

// setupRepoAndRef registers mock handlers for the repo and ref endpoints
// with fixed responses (default branch = main, commit SHA = abc123commit).
func setupRepoAndRef(mux *http.ServeMux) {
	mux.HandleFunc("/repos/testorg/testrepo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"default_branch": "main",
		})
	})

	mux.HandleFunc("/repos/testorg/testrepo/git/ref/heads/main", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object": map[string]string{"sha": "abc123commit"},
		})
	})
}

// setupRepoAndRef_WithRetry registers mock handlers where the ref endpoint
// returns 502 on the first request, then succeeds on subsequent requests.
func setupRepoAndRef_WithRetry(mux *http.ServeMux, refCount *int64) {
	mux.HandleFunc("/repos/testorg/testrepo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"default_branch": "main",
		})
	})

	mux.HandleFunc("/repos/testorg/testrepo/git/ref/heads/main", func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt64(refCount, 1)
		if count == 1 {
			// First request: return transient 502
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
			return
		}
		// Subsequent requests: succeed
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object": map[string]string{"sha": "abc123commit"},
		})
	})
}
