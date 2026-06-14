//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Review agent spec enforcement", Serial, func() {
	var (
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		// Ensure the fullsend CLI is available
		_, err := exec.LookPath("fullsend")
		if err != nil {
			Skip("fullsend CLI not found in PATH — skipping review agent tests")
		}
		// Ensure LLM endpoint is configured
		if os.Getenv("LLM_ENDPOINT") == "" {
			Skip("LLM_ENDPOINT not set — skipping review agent tests")
		}
	})

	Context("Verify review agent blocks code not matching generated spec", Ordered, func() {
		var (
			prototypeDir     string
			specDir          string
			nonCompliantDiff string
		)

		BeforeAll(func() {
			var err error

			// Create prototype directory
			prototypeDir, err = os.MkdirTemp("", "review-prototype-*")
			Expect(err).NotTo(HaveOccurred())

			// Write prototype: defines an Add function
			protoContent := `package calculator

// Add returns the sum of two integers.
func Add(a, b int) int {
	return a + b
}
`
			goModContent := `module calculator

go 1.23
`
			err = os.WriteFile(filepath.Join(prototypeDir, "calculator.go"), []byte(protoContent), 0644)
			Expect(err).NotTo(HaveOccurred())
			err = os.WriteFile(filepath.Join(prototypeDir, "go.mod"), []byte(goModContent), 0644)
			Expect(err).NotTo(HaveOccurred())

			// Generate spec from prototype
			specDir, err = os.MkdirTemp("", "review-spec-*")
			Expect(err).NotTo(HaveOccurred())

			cmdCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			cmd := exec.CommandContext(cmdCtx, "fullsend", "vibe-to-spec",
				"--input", prototypeDir,
				"--output", specDir,
			)
			output, runErr := cmd.CombinedOutput()
			Expect(runErr).NotTo(HaveOccurred(),
				"spec generation should succeed for review agent test.\nOutput: %s", string(output))

			// Create a non-compliant diff file — implements Subtract instead of Add
			nonCompliantContent := `--- a/calculator.go
+++ b/calculator.go
@@ -1,6 +1,6 @@
 package calculator

-// Add returns the sum of two integers.
-func Add(a, b int) int {
-	return a + b
+// Subtract returns the difference of two integers.
+func Subtract(a, b int) int {
+	return a - b
 }
`
			nonCompliantDiff, err = writeTempFile("non-compliant-*.diff", nonCompliantContent)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterAll(func() {
			if prototypeDir != "" {
				os.RemoveAll(prototypeDir)
			}
			if specDir != "" {
				os.RemoveAll(specDir)
			}
			if nonCompliantDiff != "" {
				os.Remove(nonCompliantDiff)
			}
		})

		It("[test_id:TS-GH-4-004] should block a PR whose code does not match the generated spec checklist", func() {
			// Find the checklist/spec file in the spec output directory
			specFile := findSpecFile(specDir)
			Expect(specFile).NotTo(BeEmpty(), "spec checklist file should exist in %s", specDir)

			// Run the review agent against the non-compliant code
			cmdCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
			defer cancel()

			cmd := exec.CommandContext(cmdCtx, "fullsend", "review",
				"--spec", specFile,
				"--diff", nonCompliantDiff,
			)

			output, err := cmd.CombinedOutput()
			outputStr := string(output)

			// The review agent may return a non-zero exit code for blocked PRs,
			// or it may return zero with a "blocked" status in the output.
			// Check the output for blocking indicators regardless of exit code.
			_ = err

			// Verify the review agent indicates the PR is blocked or requests changes
			Expect(outputStr).To(SatisfyAny(
				ContainSubstring("blocked"),
				ContainSubstring("changes_requested"),
				ContainSubstring("BLOCKED"),
				ContainSubstring("CHANGES_REQUESTED"),
				ContainSubstring("non-compliant"),
				ContainSubstring("does not match"),
				ContainSubstring("violation"),
				ContainSubstring("failed"),
			), "review agent should block non-compliant code.\nOutput: %s", outputStr)

			// Verify the review agent identifies specific spec items that are not satisfied
			Expect(outputStr).To(SatisfyAny(
				ContainSubstring("Add"),
				ContainSubstring("checklist"),
				ContainSubstring("requirement"),
				ContainSubstring("spec"),
			), "review agent should reference specific spec violations.\nOutput: %s", outputStr)
		})
	})

	Context("Verify review agent permits code matching generated spec checklist", Ordered, func() {
		var (
			prototypeDir  string
			specDir       string
			compliantDiff string
		)

		BeforeAll(func() {
			var err error

			// Create prototype directory
			prototypeDir, err = os.MkdirTemp("", "review-compliant-prototype-*")
			Expect(err).NotTo(HaveOccurred())

			// Write prototype: defines an Add function
			protoContent := `package calculator

// Add returns the sum of two integers.
func Add(a, b int) int {
	return a + b
}
`
			goModContent := `module calculator

go 1.23
`
			err = os.WriteFile(filepath.Join(prototypeDir, "calculator.go"), []byte(protoContent), 0644)
			Expect(err).NotTo(HaveOccurred())
			err = os.WriteFile(filepath.Join(prototypeDir, "go.mod"), []byte(goModContent), 0644)
			Expect(err).NotTo(HaveOccurred())

			// Generate spec from prototype
			specDir, err = os.MkdirTemp("", "review-compliant-spec-*")
			Expect(err).NotTo(HaveOccurred())

			cmdCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			cmd := exec.CommandContext(cmdCtx, "fullsend", "vibe-to-spec",
				"--input", prototypeDir,
				"--output", specDir,
			)
			output, runErr := cmd.CombinedOutput()
			Expect(runErr).NotTo(HaveOccurred(),
				"spec generation should succeed.\nOutput: %s", string(output))

			// Create a compliant diff — implements exactly what the spec requires
			compliantContent := `--- a/calculator.go
+++ b/calculator.go
@@ -1,6 +1,8 @@
 package calculator

 // Add returns the sum of two integers.
 func Add(a, b int) int {
-	return a + b
+	// Implementation with input validation
+	result := a + b
+	return result
 }
`
			compliantDiff, err = writeTempFile("compliant-*.diff", compliantContent)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterAll(func() {
			if prototypeDir != "" {
				os.RemoveAll(prototypeDir)
			}
			if specDir != "" {
				os.RemoveAll(specDir)
			}
			if compliantDiff != "" {
				os.Remove(compliantDiff)
			}
		})

		It("[test_id:TS-GH-4-005] should approve a PR whose code matches the generated spec checklist", func() {
			// Find the checklist/spec file
			specFile := findSpecFile(specDir)
			Expect(specFile).NotTo(BeEmpty(), "spec checklist file should exist in %s", specDir)

			// Run the review agent against the compliant code
			cmdCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
			defer cancel()

			cmd := exec.CommandContext(cmdCtx, "fullsend", "review",
				"--spec", specFile,
				"--diff", compliantDiff,
			)

			output, err := cmd.CombinedOutput()
			outputStr := string(output)

			// Review agent should succeed (exit code 0) for compliant code
			Expect(err).NotTo(HaveOccurred(),
				"review agent should not fail for compliant code.\nOutput: %s", outputStr)

			// Verify the review agent approves the PR
			Expect(outputStr).To(SatisfyAny(
				ContainSubstring("approved"),
				ContainSubstring("pass"),
				ContainSubstring("APPROVED"),
				ContainSubstring("PASS"),
				ContainSubstring("compliant"),
				ContainSubstring("satisfied"),
			), "review agent should approve compliant code.\nOutput: %s", outputStr)

			// Verify no false positive violations are reported
			lowerOutput := strings.ToLower(outputStr)
			Expect(lowerOutput).NotTo(ContainSubstring("violation"),
				"review agent should not report violations for compliant code.\nOutput: %s", outputStr)
		})
	})

	Context("Verify review agent detects and blocks scope creep beyond spec boundaries", Ordered, func() {
		var (
			prototypeDir   string
			specDir        string
			scopeCreepDiff string
		)

		BeforeAll(func() {
			var err error

			// Create prototype directory
			prototypeDir, err = os.MkdirTemp("", "review-scope-creep-prototype-*")
			Expect(err).NotTo(HaveOccurred())

			// Write prototype: defines only an Add function
			protoContent := `package calculator

// Add returns the sum of two integers.
func Add(a, b int) int {
	return a + b
}
`
			goModContent := `module calculator

go 1.23
`
			err = os.WriteFile(filepath.Join(prototypeDir, "calculator.go"), []byte(protoContent), 0644)
			Expect(err).NotTo(HaveOccurred())
			err = os.WriteFile(filepath.Join(prototypeDir, "go.mod"), []byte(goModContent), 0644)
			Expect(err).NotTo(HaveOccurred())

			// Generate spec from prototype
			specDir, err = os.MkdirTemp("", "review-scope-creep-spec-*")
			Expect(err).NotTo(HaveOccurred())

			cmdCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			cmd := exec.CommandContext(cmdCtx, "fullsend", "vibe-to-spec",
				"--input", prototypeDir,
				"--output", specDir,
			)
			output, runErr := cmd.CombinedOutput()
			Expect(runErr).NotTo(HaveOccurred(),
				"spec generation should succeed.\nOutput: %s", string(output))

			// Create a diff with scope creep: implements Add (in spec) PLUS Multiply (NOT in spec)
			scopeCreepContent := `--- a/calculator.go
+++ b/calculator.go
@@ -1,6 +1,12 @@
 package calculator

 // Add returns the sum of two integers.
 func Add(a, b int) int {
 	return a + b
 }
+
+// Multiply returns the product of two integers.
+// NOTE: This function is NOT in the spec — it is scope creep.
+func Multiply(a, b int) int {
+	return a * b
+}
`
			scopeCreepDiff, err = writeTempFile("scope-creep-*.diff", scopeCreepContent)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterAll(func() {
			if prototypeDir != "" {
				os.RemoveAll(prototypeDir)
			}
			if specDir != "" {
				os.RemoveAll(specDir)
			}
			if scopeCreepDiff != "" {
				os.Remove(scopeCreepDiff)
			}
		})

		It("[test_id:TS-GH-4-006] should detect and block code that adds functionality beyond the spec", func() {
			// Find the checklist/spec file
			specFile := findSpecFile(specDir)
			Expect(specFile).NotTo(BeEmpty(), "spec checklist file should exist in %s", specDir)

			// Run the review agent against code with scope creep
			cmdCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
			defer cancel()

			cmd := exec.CommandContext(cmdCtx, "fullsend", "review",
				"--spec", specFile,
				"--diff", scopeCreepDiff,
			)

			output, err := cmd.CombinedOutput()
			outputStr := string(output)
			_ = err // Review agent may or may not return non-zero for scope creep

			// Verify the review agent detects scope creep and blocks
			Expect(outputStr).To(SatisfyAny(
				ContainSubstring("scope"),
				ContainSubstring("blocked"),
				ContainSubstring("BLOCKED"),
				ContainSubstring("unauthorized"),
				ContainSubstring("out-of-scope"),
				ContainSubstring("beyond"),
				ContainSubstring("extra"),
				ContainSubstring("not in spec"),
			), "review agent should detect scope creep.\nOutput: %s", outputStr)

			// Verify the review agent identifies the specific out-of-scope additions
			Expect(outputStr).To(SatisfyAny(
				ContainSubstring("Multiply"),
				ContainSubstring("additional"),
				ContainSubstring("unauthorized"),
				ContainSubstring("not specified"),
			), "review agent should identify the out-of-scope code (Multiply function).\nOutput: %s", outputStr)
		})
	})
})

// writeTempFile creates a temporary file with the given content and returns its path.
func writeTempFile(pattern, content string) (string, error) {
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(content); err != nil {
		os.Remove(f.Name())
		return "", fmt.Errorf("writing temp file: %w", err)
	}
	return f.Name(), nil
}

// findSpecFile searches the given directory for a spec/checklist file.
func findSpecFile(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	// Look for checklist file first, then any YAML/JSON file
	for _, entry := range entries {
		name := strings.ToLower(entry.Name())
		if strings.Contains(name, "checklist") {
			return filepath.Join(dir, entry.Name())
		}
	}
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".json") {
			return filepath.Join(dir, name)
		}
	}
	return ""
}
