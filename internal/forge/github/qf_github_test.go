package github

// QualityFlow generated tests for GH-72
// Suite: TS-GH72-009 — Git Trees API truncation error handling
// STD: outputs/std/GH-72/GH-72_test_description.yaml

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TC-GH72-051: ListRepositoryFiles returns error on truncated tree response
func TestQFListRepositoryFiles_Truncated(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")

		switch {
		// Step 1: Get repo info (default branch)
		case r.URL.Path == "/repos/org/large-repo":
			json.NewEncoder(w).Encode(map[string]any{
				"default_branch": "main",
			})

		// Step 2: Get branch ref → commit SHA
		case r.URL.Path == "/repos/org/large-repo/git/ref/heads/main":
			json.NewEncoder(w).Encode(map[string]any{
				"object": map[string]any{
					"sha": "abc123",
				},
			})

		// Step 3: Get commit → tree SHA
		case r.URL.Path == "/repos/org/large-repo/git/commits/abc123":
			json.NewEncoder(w).Encode(map[string]any{
				"tree": map[string]any{
					"sha": "tree456",
				},
			})

		// Step 4: Get recursive tree — return truncated response
		case r.URL.Path == "/repos/org/large-repo/git/trees/tree456":
			json.NewEncoder(w).Encode(map[string]any{
				"tree": []map[string]any{
					{"path": "file1.go", "type": "blob"},
					{"path": "file2.go", "type": "blob"},
				},
				"truncated": true,
			})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	files, err := client.ListRepositoryFiles(context.Background(), "org", "large-repo")

	require.Error(t, err, "should return error on truncated tree response")
	assert.Contains(t, err.Error(), "truncated",
		"error message should be descriptive for operators")
	assert.Nil(t, files, "no partial file list should be returned")
}
