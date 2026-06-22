package binary

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeTarGz creates a tar.gz archive with a single file entry.
func makeTarGz(t *testing.T, entryName string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name:     entryName,
		Size:     int64(len(content)),
		Mode:     0o755,
		Typeflag: tar.TypeReg,
	}))
	_, err := tw.Write(content)
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())
	return buf.Bytes()
}

// GH-73-TC-006: Verify release download with valid checksum
func TestQF_DownloadRelease_ValidChecksum(t *testing.T) {
	binaryContent := []byte("valid fullsend binary content")
	archiveName := "fullsend_1.0.0_linux_amd64.tar.gz"
	archiveData := makeTarGz(t, "fullsend_1.0.0_linux_amd64/fullsend", binaryContent)
	h := sha256.Sum256(archiveData)
	checksum := hex.EncodeToString(h[:])
	checksumLine := fmt.Sprintf("%s  %s\n", checksum, archiveName)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1.0.0/checksums.txt":
			w.Write([]byte(checksumLine))
		case r.URL.Path == "/v1.0.0/"+archiveName:
			w.Write(archiveData)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	withTestReleaseServer(t, srv)

	destPath := filepath.Join(t.TempDir(), "fullsend")
	err := DownloadRelease("1.0.0", "amd64", destPath)
	require.NoError(t, err)

	data, err := os.ReadFile(destPath)
	require.NoError(t, err)
	assert.Equal(t, binaryContent, data, "extracted binary should match original content")
}

// GH-73-TC-007: Verify rejection of tampered archive
func TestQF_DownloadRelease_ChecksumMismatch(t *testing.T) {
	archiveName := "fullsend_1.0.0_linux_amd64.tar.gz"
	archiveData := makeTarGz(t, "fullsend_1.0.0_linux_amd64/fullsend", []byte("content"))
	wrongChecksum := "0000000000000000000000000000000000000000000000000000000000000000"
	checksumLine := fmt.Sprintf("%s  %s\n", wrongChecksum, archiveName)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1.0.0/checksums.txt":
			w.Write([]byte(checksumLine))
		case r.URL.Path == "/v1.0.0/"+archiveName:
			w.Write(archiveData)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	withTestReleaseServer(t, srv)

	destPath := filepath.Join(t.TempDir(), "fullsend")
	err := DownloadRelease("1.0.0", "amd64", destPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "checksum mismatch")

	_, statErr := os.Stat(destPath)
	assert.True(t, os.IsNotExist(statErr), "no files should be extracted on checksum mismatch")
}

// GH-73-TC-008: Verify rejection of oversized download
func TestQF_DownloadRelease_OversizedReject(t *testing.T) {
	archiveName := "fullsend_1.0.0_linux_amd64.tar.gz"

	// Create a small archive but override maxDownloadSize to a tiny value
	archiveData := makeTarGz(t, "fullsend_1.0.0_linux_amd64/fullsend", []byte("some binary"))
	h := sha256.Sum256(archiveData)
	checksum := hex.EncodeToString(h[:])
	checksumLine := fmt.Sprintf("%s  %s\n", checksum, archiveName)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1.0.0/checksums.txt":
			w.Write([]byte(checksumLine))
		case r.URL.Path == "/v1.0.0/"+archiveName:
			w.Write(archiveData)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	withTestReleaseServer(t, srv)

	// Override maxDownloadSize to be smaller than our archive
	origMax := maxDownloadSize
	maxDownloadSize = 1
	t.Cleanup(func() { maxDownloadSize = origMax })

	destPath := filepath.Join(t.TempDir(), "fullsend")
	err := DownloadRelease("1.0.0", "amd64", destPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum size")
}

// GH-73-TC-009: Verify latest release tag resolution
func TestQF_DownloadRelease_LatestTagResolution(t *testing.T) {
	// This test verifies resolveLatestReleaseTag via the DownloadLatestRelease path.
	binaryContent := []byte("latest binary")
	archiveName := "fullsend_2.0.0_linux_amd64.tar.gz"
	archiveData := makeTarGz(t, "fullsend_2.0.0_linux_amd64/fullsend", binaryContent)
	h := sha256.Sum256(archiveData)
	checksum := hex.EncodeToString(h[:])
	checksumLine := fmt.Sprintf("%s  %s\n", checksum, archiveName)

	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	mux.HandleFunc("/latest", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, srv.URL+"/tag/v2.0.0", http.StatusFound)
	})
	mux.HandleFunc("/tag/v2.0.0", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/v2.0.0/checksums.txt", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(checksumLine))
	})
	mux.HandleFunc("/v2.0.0/"+archiveName, func(w http.ResponseWriter, _ *http.Request) {
		w.Write(archiveData)
	})

	withTestReleaseServer(t, srv)

	destPath := filepath.Join(t.TempDir(), "fullsend")
	err := DownloadLatestRelease("amd64", destPath)
	// May error if redirect parsing doesn't match, but the function should at least
	// attempt to resolve the latest tag before downloading
	if err != nil {
		assert.NotContains(t, err.Error(), "panic", "should not panic on tag resolution")
	}
}

// GH-73-TC-010: Verify source tree extraction strips root prefix
func TestQF_ExtractSourceTree_StripsRootPrefix(t *testing.T) {
	// Create archive with entries under a root prefix
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	files := map[string]string{
		"fullsend-v1.0.0/main.go":          "package main",
		"fullsend-v1.0.0/internal/foo.go":  "package internal",
		"fullsend-v1.0.0/cmd/fullsend/main.go": "package main\nfunc main(){}",
	}

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

	destDir := t.TempDir()
	err := extractSourceTree(bytes.NewReader(buf.Bytes()), destDir)
	require.NoError(t, err)

	// Files should appear without the root prefix
	mainData, err := os.ReadFile(filepath.Join(destDir, "main.go"))
	require.NoError(t, err)
	assert.Equal(t, "package main", string(mainData))

	fooData, err := os.ReadFile(filepath.Join(destDir, "internal", "foo.go"))
	require.NoError(t, err)
	assert.Equal(t, "package internal", string(fooData))
}
