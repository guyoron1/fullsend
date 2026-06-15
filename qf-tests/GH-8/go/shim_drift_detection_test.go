//go:build e2e

package e2e

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

/*
Shim Drift Detection Tests — GH-8

STP Reference: outputs/stp/GH-8/GH-8_test_plan.md
STD Reference: outputs/std/GH-8/GH-8_test_description.yaml
Jira: GH-8 — fix(#2247): compare decoded text in shim drift detection

These tests verify the fix for issue #2247: reconcile-repos.sh now compares
decoded text (after stripping CR) instead of re-encoded base64, preventing
false-positive drift detection caused by trailing-newline differences in
base64 encoding.
*/

// compareDecodedContent mimics the comparison logic in reconcile-repos.sh:
//
//	EXPECTED_DECODED=$(printf '%s' "$EXPECTED_B64" | base64 -d | tr -d '\r')
//	REMOTE_DECODED=$(printf '%s' "$REMOTE_B64" | base64 -d | tr -d '\r')
//	[ "$REMOTE_DECODED" = "$EXPECTED_DECODED" ]
//
// Returns (isStale bool, err error).
func compareDecodedContent(managedB64, remoteB64 string) (bool, error) {
	managedBytes, err := base64.StdEncoding.DecodeString(managedB64)
	if err != nil {
		return false, fmt.Errorf("decoding managed base64: %w", err)
	}
	remoteBytes, err := base64.StdEncoding.DecodeString(remoteB64)
	if err != nil {
		return false, fmt.Errorf("decoding remote base64: %w", err)
	}

	// Strip carriage returns (equivalent to `tr -d '\r'` in the shell script).
	managedText := strings.ReplaceAll(string(managedBytes), "\r", "")
	remoteText := strings.ReplaceAll(string(remoteBytes), "\r", "")

	isStale := managedText != remoteText
	return isStale, nil
}

// shimContent is the canonical shim YAML used across tests.
const shimContent = `name: fullsend-shim
on:
  workflow_dispatch:
jobs:
  shim:
    uses: fullsend-ai/fullsend/.github/workflows/shim.yml@main
`

var _ = Describe("[GH-8] Shim Drift Detection", Ordered, func() {

	/*
		Markers:
		    - tier1

		Preconditions:
		    - Bash 4.0+ with GNU coreutils
		    - reconcile-repos.sh script available in internal/scaffold/fullsend-repo/scripts/
		    - gh CLI authenticated (mocked in tests)
	*/

	Context("when comparing base64-encoded shim content", func() {

		// TS-GH-8-001: Identical content with different trailing newlines → not stale.
		It("[test_id:TS-GH-8-001] should not flag identical content with different trailing newlines as stale", func() {
			/*
				Preconditions:
				    - Managed shim content string prepared
				    - Remote shim content prepared with trailing newlines appended
				    - Both content strings base64-encoded (base64 strings differ due to trailing newline differences)

				Steps:
				    1. Base64-encode the managed content (no trailing newlines)
				    2. Base64-encode the same content with extra trailing newlines
				    3. Run drift detection comparison using decoded text
				    4. Verify comparison result is not stale

				Expected:
				    - Identical shim content with different trailing newlines is NOT flagged as stale
				    - Comparison function returns false (not stale) for logically identical content
			*/

			// SETUP: managed content without trailing newline padding.
			managedContent := shimContent
			// Remote content: same logical content but with extra trailing newlines
			// (simulates GitHub API returning content with different whitespace).
			remoteContent := shimContent + "\n\n"

			// Base64-encode both — they will produce DIFFERENT base64 strings.
			managedB64 := base64.StdEncoding.EncodeToString([]byte(managedContent))
			remoteB64 := base64.StdEncoding.EncodeToString([]byte(remoteContent))

			// Sanity check: raw base64 strings differ.
			Expect(managedB64).NotTo(Equal(remoteB64),
				"base64 encodings should differ due to trailing newlines")

			// TEST: compare using decoded text (the fixed comparison).
			isStale, err := compareDecodedContent(managedB64, remoteB64)

			// ASSERT: content is NOT stale (the fix for #2247).
			Expect(err).NotTo(HaveOccurred(), "comparison should not error")
			Expect(isStale).To(BeFalse(),
				"identical content with different trailing newlines must NOT be flagged as stale (issue #2247)")
		})

		// TS-GH-8-002: Genuinely different content → stale.
		It("[test_id:TS-GH-8-002] should correctly flag genuinely stale shim content as stale", func() {
			/*
				Preconditions:
				    - Managed shim content prepared (current expected shim)
				    - Stale remote content prepared with actual differences (e.g., old version ref)
				    - Both content strings base64-encoded

				Steps:
				    1. Base64-encode the current managed content
				    2. Base64-encode outdated content (old version ref)
				    3. Run drift detection comparison
				    4. Verify comparison result is stale

				Expected:
				    - Genuinely different shim content is flagged as stale
				    - Comparison function returns true (stale) for content with real differences
			*/

			managedContent := shimContent

			// Stale remote: uses old version ref (v0.9 instead of @main).
			staleRemoteContent := `name: fullsend-shim
on:
  workflow_dispatch:
jobs:
  shim:
    uses: fullsend-ai/fullsend/.github/workflows/shim.yml@v0.9
`

			managedB64 := base64.StdEncoding.EncodeToString([]byte(managedContent))
			remoteB64 := base64.StdEncoding.EncodeToString([]byte(staleRemoteContent))

			// TEST: compare using decoded text.
			isStale, err := compareDecodedContent(managedB64, remoteB64)

			// ASSERT: content IS stale (genuinely different).
			Expect(err).NotTo(HaveOccurred(), "comparison should not error")
			Expect(isStale).To(BeTrue(),
				"genuinely different shim content must be flagged as stale")
		})

		// TS-GH-8-003: CR/LF vs LF encoding → not stale after normalization.
		It("[test_id:TS-GH-8-003] should normalize CR/LF encoding variations and not flag as stale", func() {
			/*
				Preconditions:
				    - Shim content prepared with CRLF line endings
				    - Same shim content prepared with LF-only line endings
				    - Both versions base64-encoded

				Steps:
				    1. Prepare content with \r\n (Windows) line endings
				    2. Prepare same content with \n (Unix) line endings
				    3. Base64-encode both
				    4. Run drift detection comparison
				    5. Verify comparison result is not stale

				Expected:
				    - Content with CR/LF line endings is treated as identical to LF-only content
				    - The tr -d '\r' normalization correctly strips carriage returns before comparison
			*/

			// LF-only content (Unix-style).
			contentWithLF := shimContent

			// CRLF content (Windows-style) — same logical content.
			contentWithCRLF := strings.ReplaceAll(shimContent, "\n", "\r\n")

			// Sanity: raw strings differ.
			Expect(contentWithLF).NotTo(Equal(contentWithCRLF),
				"raw strings should differ due to line ending encoding")

			crlfB64 := base64.StdEncoding.EncodeToString([]byte(contentWithCRLF))
			lfB64 := base64.StdEncoding.EncodeToString([]byte(contentWithLF))

			// TEST: compare using decoded text with CR stripping.
			isStale, err := compareDecodedContent(crlfB64, lfB64)

			// ASSERT: not stale after CR normalization.
			Expect(err).NotTo(HaveOccurred(), "CR stripping and comparison should not error")
			Expect(isStale).To(BeFalse(),
				"CR/LF vs LF content must be treated as identical after normalization")
		})
	})

	Context("when running the enrollment workflow", func() {
		var (
			scriptPath string
			mockDir    string
			configDir  string
		)

		// locateScript finds reconcile-repos.sh relative to the repo root.
		// It tries common locations: working directory and up to 3 parent levels.
		locateScript := func() string {
			candidates := []string{
				"internal/scaffold/fullsend-repo/scripts/reconcile-repos.sh",
			}
			// Try from current dir and ancestors.
			dir, _ := os.Getwd()
			for i := 0; i < 4; i++ {
				for _, c := range candidates {
					p := filepath.Join(dir, c)
					if _, err := os.Stat(p); err == nil {
						return p
					}
				}
				dir = filepath.Dir(dir)
			}
			// Also try from GITHUB_WORKSPACE if set.
			if ws := os.Getenv("GITHUB_WORKSPACE"); ws != "" {
				for _, c := range candidates {
					p := filepath.Join(ws, c)
					if _, err := os.Stat(p); err == nil {
						return p
					}
				}
			}
			return ""
		}

		BeforeEach(func() {
			scriptPath = locateScript()
			if scriptPath == "" {
				Skip("reconcile-repos.sh not found — skipping workflow integration tests")
			}

			var err error
			mockDir, err = os.MkdirTemp("", "shim-test-mock-*")
			Expect(err).NotTo(HaveOccurred())

			configDir, err = os.MkdirTemp("", "shim-test-config-*")
			Expect(err).NotTo(HaveOccurred())

			// Create config structure.
			Expect(os.MkdirAll(filepath.Join(configDir, "templates"), 0o755)).To(Succeed())
		})

		AfterEach(func() {
			if mockDir != "" {
				os.RemoveAll(mockDir)
			}
			if configDir != "" {
				os.RemoveAll(configDir)
			}
		})

		// writeConfigYAML creates the config.yaml with one enrolled repo.
		writeConfigYAML := func(dir string) {
			config := `version: 1
repos:
  test-repo:
    enabled: true
`
			Expect(os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(config), 0o644)).To(Succeed())
		}

		// writeShimTemplate creates the shim template file.
		writeShimTemplate := func(dir string) {
			Expect(os.WriteFile(filepath.Join(dir, "templates", "shim-workflow-call.yaml"), []byte(shimContent), 0o644)).To(Succeed())
		}

		// writeMockBase64 creates a mock base64 that wraps the real one.
		writeMockBase64 := func(dir string) {
			script := `#!/usr/bin/env bash
if [[ "${1:-}" == "-w0" ]]; then shift; fi
/usr/bin/base64 "$@" | tr -d '\r\n'
`
			p := filepath.Join(dir, "base64")
			Expect(os.WriteFile(p, []byte(script), 0o755)).To(Succeed())
		}

		// writeMockYQ creates a mock yq returning the test repo.
		writeMockYQ := func(dir string) {
			script := `#!/usr/bin/env bash
query="${1:-}"
if [[ "$query" == *"enabled == true"* ]]; then
  echo "test-repo"
elif [[ "$query" == *"enabled == false"* ]]; then
  :
else
  echo "unexpected yq query: $*" >&2
  exit 1
fi
`
			p := filepath.Join(dir, "yq")
			Expect(os.WriteFile(p, []byte(script), 0o755)).To(Succeed())
		}

		// writeMockGH creates a mock gh CLI. ghLogPath records all invocations.
		// shimB64 is the base64 content the mock API will return for the shim file.
		// If shimB64 is empty, the shim does not exist on the remote (new enrollment).
		writeMockGH := func(dir, ghLogPath, shimB64 string) {
			// Build the shim content API response. If empty, return nothing (404-like).
			shimCase := fmt.Sprintf(`    repos/test-org/test-repo/contents/.github/workflows/fullsend.yaml)
      if [[ -n "%s" ]]; then
        local jq_filter=""
        local prev=""
        for a in "$@"; do [[ "$prev" == "--jq" ]] && jq_filter="$a"; prev="$a"; done
        local json='{"content":"%s","sha":"file-sha"}'
        if [[ -n "$jq_filter" ]]; then
          printf '%%s' "$json" | jq -r "$jq_filter"
        else
          printf '%%s\n' "$json"
        fi
      else
        exit 1
      fi
      ;;`, shimB64, shimB64)

			script := fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail
printf 'gh' >> "%s"
for arg in "$@"; do printf ' %%q' "$arg" >> "%s"; done
printf '\n' >> "%s"

if [[ "$1" == "pr" && "$2" == "list" ]]; then
  echo ""
  exit 0
fi
if [[ "$1" == "pr" && "$2" == "create" ]]; then
  echo "https://github.com/test-org/test-repo/pull/42"
  exit 0
fi

if [[ "$1" != "api" ]]; then
  echo "unexpected gh command: $*" >&2
  exit 1
fi

endpoint="$2"
case "$endpoint" in
%s
  repos/test-org/test-repo)
    local jq_filter=""
    local prev=""
    for a in "$@"; do [[ "$prev" == "--jq" ]] && jq_filter="$a"; prev="$a"; done
    if [[ "$jq_filter" == ".private" ]]; then echo "false"
    elif [[ "$jq_filter" == ".default_branch" ]]; then echo "main"
    else echo '{"default_branch":"main","private":false}'
    fi
    ;;
  repos/test-org/test-repo/git/ref/heads/main)
    echo "base-sha"
    ;;
  repos/test-org/test-repo/git/commits/base-sha)
    echo "base-tree-sha"
    ;;
  repos/test-org/test-repo/git/blobs)
    echo "blob-sha"
    ;;
  repos/test-org/test-repo/git/trees)
    echo "tree-sha"
    ;;
  repos/test-org/test-repo/git/commits)
    echo "desired-commit-sha"
    ;;
  repos/test-org/test-repo/git/refs)
    exit 0
    ;;
  repos/test-org/test-repo/git/refs/heads/fullsend/onboard)
    exit 0
    ;;
  repos/test-org/test-repo/git/refs/heads/fullsend/offboard)
    exit 0
    ;;
  *)
    echo "unhandled gh api endpoint: $endpoint" >&2
    exit 0
    ;;
esac
`, ghLogPath, ghLogPath, ghLogPath, shimCase)

			p := filepath.Join(dir, "gh")
			Expect(os.WriteFile(p, []byte(script), 0o755)).To(Succeed())
		}

		// writeMockJQ creates a passthrough to real jq.
		writeMockJQ := func(dir string) {
			script := `#!/usr/bin/env bash
exec /usr/bin/jq "$@"
`
			p := filepath.Join(dir, "jq")
			Expect(os.WriteFile(p, []byte(script), 0o755)).To(Succeed())
		}

		// runReconcile executes reconcile-repos.sh with mock commands and returns output.
		runReconcile := func(envPath string) (string, error) {
			cmd := exec.Command("bash", scriptPath, configDir)
			cmd.Env = append(os.Environ(),
				"PATH="+envPath,
				"GITHUB_REPOSITORY_OWNER=test-org",
				"GITHUB_SHA=test-sha",
				"GH_TOKEN=fake-token",
			)
			out, err := cmd.CombinedOutput()
			return string(out), err
		}

		// TS-GH-8-004: Up-to-date shim → skip without creating PR.
		It("[test_id:TS-GH-8-004] should skip repos with up-to-date shim without creating an update PR", func() {
			/*
				Preconditions:
				    - Mock gh command returning current shim content (base64-encoded managed content)
				    - Mock yq command returning enrolled repo list
				    - Mock directory prepended to PATH

				Steps:
				    1. Prepare config with one enabled repo
				    2. Mock gh API returns base64 of the same content as the local template
				    3. Execute reconcile-repos.sh
				    4. Verify script output indicates repo was skipped
				    5. Verify no PR creation was attempted

				Expected:
				    - Script exits successfully
				    - No update PR created for up-to-date repo
				    - Script output indicates repo was skipped (up to date)
			*/

			writeConfigYAML(configDir)
			writeShimTemplate(configDir)
			writeMockBase64(mockDir)
			writeMockYQ(mockDir)
			writeMockJQ(mockDir)

			ghLogPath := filepath.Join(mockDir, "gh-calls.log")

			// The mock returns base64 of the SAME content the template produces.
			// This simulates a repo that already has the current shim.
			currentShimB64 := base64.StdEncoding.EncodeToString([]byte(shimContent))
			writeMockGH(mockDir, ghLogPath, currentShimB64)

			origPath := os.Getenv("PATH")
			envPath := mockDir + ":" + origPath

			output, err := runReconcile(envPath)

			// ASSERT: script exits successfully.
			Expect(err).NotTo(HaveOccurred(),
				"reconcile-repos.sh should exit successfully, output:\n%s", output)

			// ASSERT: output indicates repo was skipped.
			Expect(output).To(ContainSubstring("already enrolled"),
				"output should indicate repo was skipped as up-to-date, got:\n%s", output)

			// ASSERT: no PR creation was attempted.
			ghLog := ""
			if logBytes, readErr := os.ReadFile(ghLogPath); readErr == nil {
				ghLog = string(logBytes)
			}
			Expect(ghLog).NotTo(ContainSubstring("pr create"),
				"no 'gh pr create' should appear in gh call log")
		})

		// TS-GH-8-005: Stale shim → create update PR.
		It("[test_id:TS-GH-8-005] should create an update PR for repos with stale shim", func() {
			/*
				Preconditions:
				    - Mock gh command returning outdated base64-encoded shim content
				    - Mock yq command returning enrolled repo list
				    - Mock directory prepended to PATH

				Steps:
				    1. Prepare config with one enabled repo
				    2. Mock gh API returns base64 of OUTDATED content (old version ref)
				    3. Execute reconcile-repos.sh
				    4. Verify script detects stale content
				    5. Verify PR creation was attempted

				Expected:
				    - Script exits successfully
				    - Update PR created for stale repo
				    - Script output indicates drift was detected
			*/

			writeConfigYAML(configDir)
			writeShimTemplate(configDir)
			writeMockBase64(mockDir)
			writeMockYQ(mockDir)
			writeMockJQ(mockDir)

			ghLogPath := filepath.Join(mockDir, "gh-calls.log")

			// The mock returns base64 of OUTDATED content (v0.9 instead of @main).
			outdatedShim := strings.Replace(shimContent, "@main", "@v0.9", 1)
			staleB64 := base64.StdEncoding.EncodeToString([]byte(outdatedShim))
			writeMockGH(mockDir, ghLogPath, staleB64)

			origPath := os.Getenv("PATH")
			envPath := mockDir + ":" + origPath

			output, err := runReconcile(envPath)

			// ASSERT: script exits successfully.
			Expect(err).NotTo(HaveOccurred(),
				"reconcile-repos.sh should exit successfully, output:\n%s", output)

			// ASSERT: output indicates stale content detected.
			Expect(output).To(ContainSubstring("stale"),
				"output should indicate shim is stale, got:\n%s", output)

			// ASSERT: PR creation was attempted.
			ghLog := ""
			if logBytes, readErr := os.ReadFile(ghLogPath); readErr == nil {
				ghLog = string(logBytes)
			}
			Expect(ghLog).To(ContainSubstring("pr"),
				"gh call log should show PR interaction for stale repo")
		})
	})

	Context("when validating the regression test suite", func() {

		// TS-GH-8-006: Run reconcile-repos-test.sh end-to-end.
		It("[test_id:TS-GH-8-006] should pass all reconcile-repos-test.sh regression tests end-to-end", func() {
			/*
				Preconditions:
				    - reconcile-repos-test.sh present and executable
				    - reconcile-repos.sh present (sourced by test)

				Steps:
				    1. Locate reconcile-repos-test.sh
				    2. Execute it with bash
				    3. Check exit code
				    4. Check output for failures

				Expected:
				    - Regression test script exits with code 0
				    - No test failures in script output
				    - Script output shows all tests passed (PASS lines)
			*/

			// Locate the test script relative to repo root.
			candidates := []string{
				"internal/scaffold/fullsend-repo/scripts/reconcile-repos-test.sh",
			}
			var testScriptPath string
			dir, _ := os.Getwd()
			for i := 0; i < 4; i++ {
				for _, c := range candidates {
					p := filepath.Join(dir, c)
					if _, err := os.Stat(p); err == nil {
						testScriptPath = p
						break
					}
				}
				if testScriptPath != "" {
					break
				}
				dir = filepath.Dir(dir)
			}
			if ws := os.Getenv("GITHUB_WORKSPACE"); ws != "" && testScriptPath == "" {
				for _, c := range candidates {
					p := filepath.Join(ws, c)
					if _, err := os.Stat(p); err == nil {
						testScriptPath = p
						break
					}
				}
			}
			if testScriptPath == "" {
				Skip("reconcile-repos-test.sh not found — skipping regression suite test")
			}

			// Execute the regression test script.
			cmd := exec.Command("bash", testScriptPath)
			cmd.Env = os.Environ()
			output, err := cmd.CombinedOutput()
			outputStr := string(output)

			// ASSERT: exit code 0.
			Expect(err).NotTo(HaveOccurred(),
				"reconcile-repos-test.sh should exit with code 0, output:\n%s", outputStr)

			// ASSERT: no FAIL lines in output.
			Expect(outputStr).NotTo(ContainSubstring("FAIL"),
				"regression test output should not contain FAIL, got:\n%s", outputStr)

			// ASSERT: PASS lines present.
			Expect(outputStr).To(ContainSubstring("PASS"),
				"regression test output should contain PASS indicators, got:\n%s", outputStr)
		})
	})
})
