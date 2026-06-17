//go:build e2e

package forge_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/forge"
	gh "github.com/fullsend-ai/fullsend/internal/forge/github"
)

/*
ListRepositoryFiles Tests

STP Reference: outputs/stp/GH-25/GH-25_test_plan.md
Jira: GH-25
*/

// gitTreeEntry models an entry in a GitHub Git Tree response.
type gitTreeEntry struct {
	Path string `json:"path"`
	Type string `json:"type"` // "blob" or "tree"
	Mode string `json:"mode"`
	SHA  string `json:"sha"`
}

// newGitHubMockServer creates an httptest server that simulates the GitHub
// Git Trees API ref-chain: get repo → get branch ref → get commit → recursive tree.
func newGitHubMockServer(t *testing.T, treeEntries []gitTreeEntry, truncated bool) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		path := r.URL.Path

		switch {
		// Step 1: GET /repos/{owner}/{repo} → default branch
		case strings.HasSuffix(path, "/test-owner/test-repo") && !strings.Contains(path, "/git/"):
			json.NewEncoder(w).Encode(map[string]string{
				"default_branch": "main",
			})

		// Step 2: GET /repos/{owner}/{repo}/git/ref/heads/{branch} → commit SHA
		case strings.Contains(path, "/git/ref/heads/main"):
			json.NewEncoder(w).Encode(map[string]interface{}{
				"object": map[string]string{
					"sha": "abc123commit",
				},
			})

		// Step 3: GET /repos/{owner}/{repo}/git/commits/{sha} → tree SHA
		case strings.Contains(path, "/git/commits/abc123commit"):
			json.NewEncoder(w).Encode(map[string]interface{}{
				"tree": map[string]string{
					"sha": "def456tree",
				},
			})

		// Step 4: GET /repos/{owner}/{repo}/git/trees/{sha}?recursive=1 → file list
		case strings.Contains(path, "/git/trees/def456tree"):
			json.NewEncoder(w).Encode(map[string]interface{}{
				"tree":      treeEntries,
				"truncated": truncated,
			})

		default:
			http.NotFound(w, r)
		}
	}))
}

// newClientWithServer creates a LiveClient pointing at the test server.
func newClientWithServer(serverURL string) *gh.LiveClient {
	return gh.New("test-token").WithBaseURL(serverURL)
}

func TestListRepositoryFiles(t *testing.T) {
	ctx := context.Background()

	// [test_id:TS-GH-25-001] returns all blob paths for repository with files
	t.Run("[test_id:TS-GH-25-001] should return all blob paths for repository with files", func(t *testing.T) {
		entries := []gitTreeEntry{
			{Path: "README.md", Type: "blob", Mode: "100644", SHA: "aaa"},
			{Path: "src", Type: "tree", Mode: "040000", SHA: "bbb"},
			{Path: "src/main.go", Type: "blob", Mode: "100644", SHA: "ccc"},
			{Path: "src/util", Type: "tree", Mode: "040000", SHA: "ddd"},
			{Path: "src/util/helper.go", Type: "blob", Mode: "100644", SHA: "eee"},
			{Path: "go.mod", Type: "blob", Mode: "100644", SHA: "fff"},
		}
		server := newGitHubMockServer(t, entries, false)
		defer server.Close()

		client := newClientWithServer(server.URL)
		paths, err := client.ListRepositoryFiles(ctx, "test-owner", "test-repo")

		require.NoError(t, err)
		// Should include only blobs (4 files), not trees (2 directories)
		assert.Len(t, paths, 4)
		assert.Contains(t, paths, "README.md")
		assert.Contains(t, paths, "src/main.go")
		assert.Contains(t, paths, "src/util/helper.go")
		assert.Contains(t, paths, "go.mod")
		// No tree/directory entries
		assert.NotContains(t, paths, "src")
		assert.NotContains(t, paths, "src/util")
	})

	// [test_id:TS-GH-25-002] follows ref chain with exactly expected API calls
	t.Run("[test_id:TS-GH-25-002] should follow ref chain with exactly 4 API calls", func(t *testing.T) {
		var apiCallCount atomic.Int32

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiCallCount.Add(1)
			w.Header().Set("Content-Type", "application/json")
			path := r.URL.Path

			switch {
			case strings.HasSuffix(path, "/test-owner/test-repo") && !strings.Contains(path, "/git/"):
				json.NewEncoder(w).Encode(map[string]string{
					"default_branch": "main",
				})
			case strings.Contains(path, "/git/ref/heads/main"):
				json.NewEncoder(w).Encode(map[string]interface{}{
					"object": map[string]string{"sha": "commit-sha"},
				})
			case strings.Contains(path, "/git/commits/commit-sha"):
				json.NewEncoder(w).Encode(map[string]interface{}{
					"tree": map[string]string{"sha": "tree-sha"},
				})
			case strings.Contains(path, "/git/trees/tree-sha"):
				json.NewEncoder(w).Encode(map[string]interface{}{
					"tree":      []gitTreeEntry{{Path: "file.txt", Type: "blob"}},
					"truncated": false,
				})
			default:
				http.NotFound(w, r)
			}
		}))
		defer server.Close()

		client := newClientWithServer(server.URL)
		_, err := client.ListRepositoryFiles(ctx, "test-owner", "test-repo")

		require.NoError(t, err)
		// Exactly 4 API calls: get repo, get ref, get commit, get tree
		assert.Equal(t, int32(4), apiCallCount.Load(),
			"expected exactly 4 API calls in the ref chain")
	})

	// [test_id:TS-GH-25-003] returns ErrNotFound for non-existent repository
	t.Run("[test_id:TS-GH-25-003] should return ErrNotFound for non-existent repository", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{
				"message": "Not Found",
			})
		}))
		defer server.Close()

		client := newClientWithServer(server.URL)
		paths, err := client.ListRepositoryFiles(ctx, "ghost-owner", "no-repo")

		require.Error(t, err)
		assert.Nil(t, paths)
	})

	// [test_id:TS-GH-25-004] returns error on truncated tree
	t.Run("[test_id:TS-GH-25-004] should return error on truncated tree", func(t *testing.T) {
		entries := []gitTreeEntry{
			{Path: "file1.go", Type: "blob"},
		}
		server := newGitHubMockServer(t, entries, true /* truncated */)
		defer server.Close()

		client := newClientWithServer(server.URL)
		paths, err := client.ListRepositoryFiles(ctx, "test-owner", "test-repo")

		require.Error(t, err)
		assert.Nil(t, paths)
		assert.Contains(t, err.Error(), "truncated",
			"error should mention truncation")
	})

	// [test_id:TS-GH-25-005] returns empty slice for empty repository
	t.Run("[test_id:TS-GH-25-005] should return empty slice for empty repository", func(t *testing.T) {
		server := newGitHubMockServer(t, []gitTreeEntry{}, false)
		defer server.Close()

		client := newClientWithServer(server.URL)
		paths, err := client.ListRepositoryFiles(ctx, "test-owner", "test-repo")

		require.NoError(t, err)
		assert.NotNil(t, paths, "should return empty slice, not nil")
		assert.Empty(t, paths)
	})

	// [test_id:TS-GH-25-006] retries on transient failures during ref resolution
	t.Run("[test_id:TS-GH-25-006] should retry on transient failures during ref resolution", func(t *testing.T) {
		var refCallCount atomic.Int32

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			path := r.URL.Path

			switch {
			case strings.HasSuffix(path, "/test-owner/test-repo") && !strings.Contains(path, "/git/"):
				json.NewEncoder(w).Encode(map[string]string{
					"default_branch": "main",
				})
			case strings.Contains(path, "/git/ref/heads/main"):
				count := refCallCount.Add(1)
				if count == 1 {
					// First call: transient 502
					w.WriteHeader(http.StatusBadGateway)
					json.NewEncoder(w).Encode(map[string]string{
						"message": "Bad Gateway",
					})
					return
				}
				// Subsequent calls: success
				json.NewEncoder(w).Encode(map[string]interface{}{
					"object": map[string]string{"sha": "commit-sha"},
				})
			case strings.Contains(path, "/git/commits/commit-sha"):
				json.NewEncoder(w).Encode(map[string]interface{}{
					"tree": map[string]string{"sha": "tree-sha"},
				})
			case strings.Contains(path, "/git/trees/tree-sha"):
				json.NewEncoder(w).Encode(map[string]interface{}{
					"tree":      []gitTreeEntry{{Path: "file.txt", Type: "blob"}},
					"truncated": false,
				})
			default:
				http.NotFound(w, r)
			}
		}))
		defer server.Close()

		client := newClientWithServer(server.URL)
		paths, err := client.ListRepositoryFiles(ctx, "test-owner", "test-repo")

		require.NoError(t, err, "should succeed after retry")
		assert.NotEmpty(t, paths)
		assert.True(t, refCallCount.Load() > 1,
			"expected retry: ref endpoint should have been called more than once")
	})
}

func TestFakeListRepositoryFiles(t *testing.T) {
	ctx := context.Background()

	// [test_id:TS-GH-25-007] returns paths from FileContents map
	t.Run("[test_id:TS-GH-25-007] should return paths from FileContents map", func(t *testing.T) {
		fake := forge.NewFakeClient()
		fake.FileContents = map[string][]byte{
			"myorg/myrepo/README.md":     []byte("readme"),
			"myorg/myrepo/src/main.go":   []byte("package main"),
			"myorg/myrepo/docs/guide.md": []byte("guide"),
			"other-org/other/file.txt":   []byte("unrelated"),
		}

		paths, err := fake.ListRepositoryFiles(ctx, "myorg", "myrepo")

		require.NoError(t, err)
		assert.Len(t, paths, 3, "should return only paths for myorg/myrepo")
		assert.ElementsMatch(t, []string{"README.md", "src/main.go", "docs/guide.md"}, paths)
	})

	// [test_id:TS-GH-25-008] returns injected error
	t.Run("[test_id:TS-GH-25-008] should return injected error", func(t *testing.T) {
		testErr := fmt.Errorf("simulated API failure")
		fake := forge.NewFakeClient()
		fake.Errors = map[string]error{
			"ListRepositoryFiles": testErr,
		}
		fake.FileContents = map[string][]byte{
			"org/repo/file.go": []byte("content"),
		}

		paths, err := fake.ListRepositoryFiles(ctx, "org", "repo")

		require.Error(t, err)
		assert.ErrorIs(t, err, testErr)
		assert.Nil(t, paths)
	})
}
