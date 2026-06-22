package binary

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GH-73-TC-011: Verify explicit source dir takes precedence
func TestQF_ResolveVendorRoot_ExplicitSourceDir(t *testing.T) {
	root, err := ModuleRoot()
	if err != nil {
		t.Skip("not in fullsend checkout")
	}

	vr, err := ResolveVendorRoot(root, "")
	require.NoError(t, err)
	assert.Equal(t, root, vr.Path, "should use the explicit source directory")
	assert.Nil(t, vr.Cleanup, "explicit source should not have a cleanup function")
}

// GH-73-TC-012: Verify fallback to ModuleRoot
func TestQF_ResolveVendorRoot_FallbackToModuleRoot(t *testing.T) {
	root, err := ModuleRoot()
	if err != nil {
		t.Skip("not in fullsend checkout")
	}

	// With empty sourceDir, should fall back to ModuleRoot
	vr, err := ResolveVendorRoot("", "")
	require.NoError(t, err)
	assert.Equal(t, root, vr.Path, "should fall back to ModuleRoot")
}

// GH-73-TC-014: Verify error for dev build without checkout
func TestQF_ResolveVendorRoot_DevBuildNoCheckout(t *testing.T) {
	// Create a temp dir that is NOT a Go module root and run from there
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() { os.Chdir(origDir) })

	// "dev" is not a released version, so remote fetch should not be attempted
	_, err := ResolveVendorRoot("", "dev")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dev build")
}

// GH-73-TC-013: Verify fallback to GitHub source fetch
// When no explicit source dir and ModuleRoot is unavailable, ResolveVendorRoot
// should attempt to fetch source from GitHub for released versions.
func TestQF_ResolveVendorRoot_FallbackToGitHubFetch(t *testing.T) {
	// We can't easily make ModuleRoot fail from within the checkout,
	// so we test the FetchSourceTree function directly which is the
	// underlying mechanism for the GitHub fetch fallback.
	origURL := SourceArchiveBaseURL
	t.Cleanup(func() { SourceArchiveBaseURL = origURL })

	// Create a source archive with the expected structure
	archiveData := makeTarGzDir(t, map[string]string{
		"fullsend-v1.0.0/go.mod":         "module github.com/fullsend-ai/fullsend",
		"fullsend-v1.0.0/cmd/fullsend/main.go": "package main",
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(archiveData)
	}))
	defer srv.Close()

	SourceArchiveBaseURL = srv.URL

	destDir := t.TempDir()
	err := FetchSourceTree("1.0.0", destDir)
	require.NoError(t, err)

	// Verify files were extracted with root prefix stripped
	_, err = os.Stat(filepath.Join(destDir, "go.mod"))
	assert.NoError(t, err, "go.mod should exist after extraction")
}

// makeTarGzDir creates a tar.gz with multiple file entries.
func makeTarGzDir(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	for name, content := range files {
		data := []byte(content)
		require.NoError(t, tw.WriteHeader(&tar.Header{
			Name:     name,
			Size:     int64(len(data)),
			Mode:     0o644,
			Typeflag: tar.TypeReg,
		}))
		_, err := tw.Write(data)
		require.NoError(t, err)
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())
	return buf.Bytes()
}

// GH-73-TC-011 supplemental: Verify explicit source dir rejects invalid path
func TestQF_ResolveVendorRoot_ExplicitInvalidPath(t *testing.T) {
	_, err := ResolveVendorRoot("/nonexistent/path", "1.0.0")
	assert.Error(t, err, "should reject nonexistent explicit source dir")
}

// GH-73-TC-011 supplemental: Verify explicit source dir rejects non-fullsend module
func TestQF_ResolveVendorRoot_ExplicitWrongModule(t *testing.T) {
	dir := t.TempDir()
	// Create a go.mod but for a different module
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/other"), 0o644))

	_, err := ResolveVendorRoot(dir, "1.0.0")
	assert.Error(t, err, "should reject non-fullsend module checkout")
	assert.Contains(t, err.Error(), "not a fullsend module")
}
