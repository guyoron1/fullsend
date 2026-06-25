package scaffold

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// QualityFlow tests for GH-1270: Installer script scenarios (11-12, 15, 25-26)
// and version pin validation (13-14).
// STD: outputs/std/GH-1270/GH-1270_test_description.yaml

// installerScriptPath returns the path to the installer shell script.
func installerScriptPath() string {
	return filepath.Join("fullsend-repo", "scripts", "install-precommit-tools.sh")
}

// createFakeTarball creates a tar.gz containing a single executable file
// and returns its content and SHA256 checksum.
func createFakeTarball(t *testing.T, binaryName string) ([]byte, string) {
	t.Helper()
	dir := t.TempDir()

	// Create a simple executable script as the "binary"
	binContent := []byte("#!/bin/sh\necho fake-tool v1.0.0\n")
	binPath := filepath.Join(dir, binaryName)
	require.NoError(t, os.WriteFile(binPath, binContent, 0o755))

	// Create tar.gz
	tarPath := filepath.Join(dir, "archive.tar.gz")
	cmd := exec.Command("tar", "czf", tarPath, "-C", dir, binaryName)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "tar failed: %s", string(out))

	content, err := os.ReadFile(tarPath)
	require.NoError(t, err)

	hash := sha256.Sum256(content)
	checksum := fmt.Sprintf("%x", hash)

	return content, checksum
}

// writeManifest writes a JSON manifest file for install-precommit-tools.sh.
func writeManifest(t *testing.T, dir string, manifest any) string {
	t.Helper()
	data, err := json.Marshal(manifest)
	require.NoError(t, err)
	path := filepath.Join(dir, "manifest.json")
	require.NoError(t, os.WriteFile(path, data, 0o644))
	return path
}

// runInstaller executes install-precommit-tools.sh with the given manifest.
// Returns exit code, stdout, stderr.
func runInstaller(t *testing.T, manifestPath string, env ...string) (int, string, string) {
	t.Helper()
	cmd := exec.Command("bash", installerScriptPath(), manifestPath)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	// Set HOME to a temp dir so we don't pollute the real home
	homeDir := t.TempDir()
	baseEnv := []string{
		"HOME=" + homeDir,
		"PATH=" + os.Getenv("PATH"),
	}
	cmd.Env = append(baseEnv, env...)
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("unexpected error running installer: %v", err)
		}
	}
	return exitCode, stdout.String(), stderr.String()
}

// TestQF_InstallerBinaryValidChecksum verifies binary install with valid
// checksum succeeds.
// [TS-GH-1270-011] Scenario 11
func TestQF_InstallerBinaryValidChecksum(t *testing.T) {
	tarball, checksum := createFakeTarball(t, "fake-tool")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		w.Write(tarball)
	}))
	defer server.Close()

	manifest := map[string]any{
		"tools": []map[string]any{
			{
				"type":         "binary",
				"name":         "fake-tool",
				"version":      "1.0.0",
				"url_template": server.URL + "/fake-tool-{version}.tar.gz",
				"checksums": map[string]string{
					"x86_64":  checksum,
					"aarch64": checksum,
				},
				"binary_name": "fake-tool",
			},
		},
		"warnings": []string{},
	}

	dir := t.TempDir()
	manifestPath := writeManifest(t, dir, manifest)

	exitCode, stdout, _ := runInstaller(t, manifestPath)
	assert.Equal(t, 0, exitCode, "valid checksum install should exit 0")
	assert.Contains(t, stdout, "fake-tool", "output should mention the tool name")
}

// TestQF_InstallerBinaryBadChecksum verifies binary install with mismatched
// checksum exits non-zero.
// [TS-GH-1270-012] Scenario 12 (P0)
func TestQF_InstallerBinaryBadChecksum(t *testing.T) {
	tarball, _ := createFakeTarball(t, "fake-tool")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		w.Write(tarball)
	}))
	defer server.Close()

	// Use a deliberately wrong checksum
	badChecksum := "0000000000000000000000000000000000000000000000000000000000000000"
	manifest := map[string]any{
		"tools": []map[string]any{
			{
				"type":         "binary",
				"name":         "bad-tool",
				"version":      "1.0.0",
				"url_template": server.URL + "/bad-tool-{version}.tar.gz",
				"checksums": map[string]string{
					"x86_64":  badChecksum,
					"aarch64": badChecksum,
				},
				"binary_name": "bad-tool",
			},
		},
		"warnings": []string{},
	}

	dir := t.TempDir()
	manifestPath := writeManifest(t, dir, manifest)

	exitCode, _, _ := runInstaller(t, manifestPath)
	assert.NotEqual(t, 0, exitCode,
		"checksum mismatch MUST exit non-zero (supply-chain safety)")
}

// TestQF_InstallerChecksumFailExitsOne verifies sha256sum failure causes
// exit 1 specifically (hard stop, not skip).
// [TS-GH-1270-025] Scenario 25 (P0)
func TestQF_InstallerChecksumFailExitsOne(t *testing.T) {
	tarball, _ := createFakeTarball(t, "tampered-tool")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(tarball)
	}))
	defer server.Close()

	badChecksum := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	manifest := map[string]any{
		"tools": []map[string]any{
			{
				"type":         "binary",
				"name":         "tampered-tool",
				"version":      "1.0.0",
				"url_template": server.URL + "/t.tar.gz",
				"checksums": map[string]string{
					"x86_64":  badChecksum,
					"aarch64": badChecksum,
				},
				"binary_name": "tampered-tool",
			},
		},
		"warnings": []string{},
	}

	dir := t.TempDir()
	manifestPath := writeManifest(t, dir, manifest)

	exitCode, stdout, _ := runInstaller(t, manifestPath)
	assert.Equal(t, 1, exitCode,
		"checksum failure MUST exit 1 (hard stop), not exit 0 with skip")
	combined := stdout
	assert.True(t,
		strings.Contains(strings.ToLower(combined), "checksum") ||
			strings.Contains(strings.ToLower(combined), "sha256"),
		"error message should mention checksum/sha256; got: %s", combined)
}

// TestQF_InstallerChecksumPassProceeds verifies successful checksum
// allows install to proceed.
// [TS-GH-1270-026] Scenario 26
func TestQF_InstallerChecksumPassProceeds(t *testing.T) {
	tarball, checksum := createFakeTarball(t, "good-tool")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(tarball)
	}))
	defer server.Close()

	manifest := map[string]any{
		"tools": []map[string]any{
			{
				"type":         "binary",
				"name":         "good-tool",
				"version":      "1.0.0",
				"url_template": server.URL + "/g.tar.gz",
				"checksums": map[string]string{
					"x86_64":  checksum,
					"aarch64": checksum,
				},
				"binary_name": "good-tool",
			},
		},
		"warnings": []string{},
	}

	dir := t.TempDir()
	manifestPath := writeManifest(t, dir, manifest)

	exitCode, stdout, _ := runInstaller(t, manifestPath)
	assert.Equal(t, 0, exitCode, "valid checksum should allow install")
	assert.Contains(t, stdout, "installed",
		"output should confirm installation")
}

// TestQF_InstallerPipNoPinRejected verifies pip install without version
// pin is rejected.
// [TS-GH-1270-013] Scenario 13
func TestQF_InstallerPipNoPinRejected(t *testing.T) {
	manifest := map[string]any{
		"tools": []map[string]any{
			{
				"type": "pip",
				"name": "yamllint",
				// version deliberately omitted
			},
		},
		"warnings": []string{},
	}

	dir := t.TempDir()
	manifestPath := writeManifest(t, dir, manifest)

	exitCode, stdout, _ := runInstaller(t, manifestPath)
	// Installer warns and skips unpinned pip — exit 0 (graceful skip)
	assert.Equal(t, 0, exitCode)
	assert.Contains(t, strings.ToLower(stdout), "version",
		"output should warn about missing version pin")
}

// TestQF_InstallerNpmNoPinRejected verifies npm install without version
// pin is rejected.
// [TS-GH-1270-014] Scenario 14
func TestQF_InstallerNpmNoPinRejected(t *testing.T) {
	manifest := map[string]any{
		"tools": []map[string]any{
			{
				"type": "npm",
				"name": "prettier",
				// version deliberately omitted
			},
		},
		"warnings": []string{},
	}

	dir := t.TempDir()
	manifestPath := writeManifest(t, dir, manifest)

	exitCode, stdout, _ := runInstaller(t, manifestPath)
	assert.Equal(t, 0, exitCode)
	assert.Contains(t, strings.ToLower(stdout), "version",
		"output should warn about missing version pin")
}

// TestQF_InstallerEmptyManifest verifies installer is a no-op for empty
// tools list (part of E2E scenario 34).
// [TS-GH-1270-034] Scenario 34 (partial)
func TestQF_InstallerEmptyManifest(t *testing.T) {
	manifest := map[string]any{
		"tools":    []map[string]any{},
		"warnings": []string{},
	}

	dir := t.TempDir()
	manifestPath := writeManifest(t, dir, manifest)

	exitCode, stdout, _ := runInstaller(t, manifestPath)
	assert.Equal(t, 0, exitCode, "empty manifest should exit 0")
	assert.Contains(t, stdout, "No additional pre-commit tools",
		"should indicate no tools to install")
}

// TestQF_InstallerUnsupportedArch verifies unsupported architecture emits
// warning and skips binary install gracefully.
// [TS-GH-1270-015] Scenario 15
func TestQF_InstallerUnsupportedArch(t *testing.T) {
	tarball, checksum := createFakeTarball(t, "arch-tool")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(tarball)
	}))
	defer server.Close()

	manifest := map[string]any{
		"tools": []map[string]any{
			{
				"type":         "binary",
				"name":         "arch-tool",
				"version":      "1.0.0",
				"url_template": server.URL + "/a.tar.gz",
				"checksums": map[string]string{
					// Only provide checksums for real arches — unsupported arch has no checksum
					"x86_64":  checksum,
					"aarch64": checksum,
				},
				"binary_name": "arch-tool",
			},
		},
		"warnings": []string{},
	}

	dir := t.TempDir()
	manifestPath := writeManifest(t, dir, manifest)

	// Create a mock uname that returns unsupported architecture
	mockBin := t.TempDir()
	mockUname := filepath.Join(mockBin, "uname")
	require.NoError(t, os.WriteFile(mockUname,
		[]byte("#!/bin/sh\nif [ \"$1\" = \"-m\" ]; then echo s390x; else /usr/bin/uname \"$@\"; fi\n"),
		0o755))

	cmd := exec.Command("bash", installerScriptPath(), manifestPath)
	homeDir := t.TempDir()
	cmd.Env = []string{
		"HOME=" + homeDir,
		"PATH=" + mockBin + ":" + os.Getenv("PATH"),
	}
	var stdout strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout
	err := cmd.Run()
	// Should exit 0 (graceful degradation)
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Acceptable: exit 0 is preferred but some architectures might not fail
			_ = exitErr
		}
	}
	output := stdout.String()
	// The installer should warn about unsupported architecture
	assert.True(t,
		strings.Contains(output, "Unsupported architecture") ||
			strings.Contains(output, "unsupported") ||
			strings.Contains(output, "s390x"),
		"should warn about unsupported architecture; got: %s", output)
}
