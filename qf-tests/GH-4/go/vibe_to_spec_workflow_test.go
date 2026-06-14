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

var _ = Describe("Vibe-to-spec workflow", Serial, func() {
	var (
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		// Ensure the fullsend CLI is available
		_, err := exec.LookPath("fullsend")
		if err != nil {
			Skip("fullsend CLI not found in PATH — skipping vibe-to-spec tests")
		}
		// Ensure LLM endpoint is configured
		if os.Getenv("LLM_ENDPOINT") == "" {
			Skip("LLM_ENDPOINT not set — skipping vibe-to-spec tests")
		}
	})

	Context("Verify vibe-to-spec workflow produces valid spec from prototype code", Ordered, func() {
		var (
			prototypeDir string
			specOutputDir string
		)

		BeforeAll(func() {
			var err error

			// Create a temporary prototype directory with testable behavior
			prototypeDir, err = os.MkdirTemp("", "vibe-to-spec-prototype-*")
			Expect(err).NotTo(HaveOccurred(), "failed to create prototype directory")

			// Write a prototype Go file with clear, testable functions
			mainGoContent := `package main

import "fmt"

// Add returns the sum of two integers.
func Add(a, b int) int {
	return a + b
}

// Subtract returns the difference of two integers.
func Subtract(a, b int) int {
	return a - b
}

// Greet returns a greeting message for the given name.
func Greet(name string) string {
	return fmt.Sprintf("Hello, %s!", name)
}

func main() {
	fmt.Println(Add(2, 3))
	fmt.Println(Subtract(5, 3))
	fmt.Println(Greet("World"))
}
`
			goModContent := `module prototype

go 1.23
`
			err = os.WriteFile(filepath.Join(prototypeDir, "main.go"), []byte(mainGoContent), 0644)
			Expect(err).NotTo(HaveOccurred(), "failed to write main.go")

			err = os.WriteFile(filepath.Join(prototypeDir, "go.mod"), []byte(goModContent), 0644)
			Expect(err).NotTo(HaveOccurred(), "failed to write go.mod")

			// Create spec output directory
			specOutputDir, err = os.MkdirTemp("", "vibe-to-spec-output-*")
			Expect(err).NotTo(HaveOccurred(), "failed to create spec output directory")
		})

		AfterAll(func() {
			if prototypeDir != "" {
				os.RemoveAll(prototypeDir)
			}
			if specOutputDir != "" {
				os.RemoveAll(specOutputDir)
			}
		})

		It("[test_id:TS-GH-4-001] should generate a valid formal specification from developer prototype code", func() {
			// Execute the vibe-to-spec workflow on the prototype directory
			cmdCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
			defer cancel()

			cmd := exec.CommandContext(cmdCtx, "fullsend", "vibe-to-spec",
				"--input", prototypeDir,
				"--output", specOutputDir,
			)
			cmd.Env = append(os.Environ(),
				fmt.Sprintf("LLM_ENDPOINT=%s", os.Getenv("LLM_ENDPOINT")),
			)

			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(),
				"vibe-to-spec workflow should complete without error.\nOutput: %s", string(output))

			// Verify the spec output directory contains at least one file
			entries, err := os.ReadDir(specOutputDir)
			Expect(err).NotTo(HaveOccurred(), "failed to read spec output directory")
			Expect(entries).NotTo(BeEmpty(), "spec output directory should contain generated files")

			// Find the generated spec file (YAML or JSON)
			var specFilePath string
			for _, entry := range entries {
				if strings.HasSuffix(entry.Name(), ".yaml") || strings.HasSuffix(entry.Name(), ".yml") || strings.HasSuffix(entry.Name(), ".json") {
					specFilePath = filepath.Join(specOutputDir, entry.Name())
					break
				}
			}
			Expect(specFilePath).NotTo(BeEmpty(), "no spec file (YAML/JSON) found in output directory")

			// Read and validate the spec file content
			specBytes, err := os.ReadFile(specFilePath)
			Expect(err).NotTo(HaveOccurred(), "failed to read generated spec file")
			Expect(specBytes).NotTo(BeEmpty(), "generated spec file should not be empty")

			specContent := string(specBytes)

			// Verify the spec contains functional requirements
			Expect(specContent).To(SatisfyAny(
				ContainSubstring("functional_requirements"),
				ContainSubstring("functional-requirements"),
				ContainSubstring("requirements"),
			), "generated spec should contain a functional requirements section")

			// Verify the spec contains acceptance scenarios
			Expect(specContent).To(SatisfyAny(
				ContainSubstring("acceptance_scenarios"),
				ContainSubstring("acceptance-scenarios"),
				ContainSubstring("acceptance_criteria"),
				ContainSubstring("scenarios"),
			), "generated spec should contain acceptance scenarios")

			// Verify the spec is parseable as structured data (YAML)
			// A basic check — the file should have key-value structure
			Expect(specContent).To(MatchRegexp(`\w+:\s`),
				"generated spec should be in a structured key-value format")
		})
	})

	Context("Verify exploration artifacts are cleaned up after spec generation completes", Ordered, func() {
		var (
			explorationDir string
			specOutputDir  string
		)

		BeforeAll(func() {
			var err error

			// Create an exploration/prototype directory
			explorationDir, err = os.MkdirTemp("", "exploration-artifacts-*")
			Expect(err).NotTo(HaveOccurred(), "failed to create exploration directory")

			// Write prototype exploration files
			prototypeContent := `package main

// Explore demonstrates a prototype function with testable behavior.
func Explore(input string) string {
	if input == "" {
		return "default"
	}
	return input
}
`
			goModContent := `module exploration

go 1.23
`
			err = os.WriteFile(filepath.Join(explorationDir, "prototype.go"), []byte(prototypeContent), 0644)
			Expect(err).NotTo(HaveOccurred(), "failed to write prototype.go")

			err = os.WriteFile(filepath.Join(explorationDir, "go.mod"), []byte(goModContent), 0644)
			Expect(err).NotTo(HaveOccurred(), "failed to write go.mod")

			// Create spec output directory
			specOutputDir, err = os.MkdirTemp("", "exploration-spec-output-*")
			Expect(err).NotTo(HaveOccurred(), "failed to create spec output directory")

			// Run the vibe-to-spec workflow to completion
			cmdCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			cmd := exec.CommandContext(cmdCtx, "fullsend", "vibe-to-spec",
				"--input", explorationDir,
				"--output", specOutputDir,
			)
			output, runErr := cmd.CombinedOutput()
			Expect(runErr).NotTo(HaveOccurred(),
				"vibe-to-spec workflow should complete successfully for cleanup test.\nOutput: %s", string(output))
		})

		AfterAll(func() {
			// Clean up any remaining artifacts
			if explorationDir != "" {
				os.RemoveAll(explorationDir)
			}
			if specOutputDir != "" {
				os.RemoveAll(specOutputDir)
			}
		})

		It("[test_id:TS-GH-4-002] should remove all exploration artifacts after spec generation completes", func() {
			// Verify that exploration artifact directory no longer exists
			// The vibe-to-spec workflow should clean up the input prototype directory
			_, err := os.Stat(explorationDir)
			if err == nil {
				// If the directory still exists, check that prototype files are removed
				entries, readErr := os.ReadDir(explorationDir)
				if readErr == nil {
					// Check no .go prototype files remain
					var protoFiles []string
					for _, entry := range entries {
						if strings.HasSuffix(entry.Name(), ".go") {
							protoFiles = append(protoFiles, entry.Name())
						}
					}
					Expect(protoFiles).To(BeEmpty(),
						"no prototype .go files should remain in exploration directory after spec generation, found: %v", protoFiles)
				}
			}
			// If os.Stat returns an error (directory doesn't exist), that's the expected behavior

			// Verify that the generated spec file IS preserved
			entries, err := os.ReadDir(specOutputDir)
			Expect(err).NotTo(HaveOccurred(), "spec output directory should still exist")
			Expect(entries).NotTo(BeEmpty(), "generated spec file should be preserved after cleanup")
		})
	})

	Context("Verify error returned when prototype contains no testable behavior", Ordered, func() {
		var (
			emptyPrototypeDir string
			specOutputDir     string
		)

		BeforeAll(func() {
			var err error

			// Create a prototype directory with no testable behavior
			emptyPrototypeDir, err = os.MkdirTemp("", "empty-prototype-*")
			Expect(err).NotTo(HaveOccurred(), "failed to create empty prototype directory")

			// Write a Go file with NO exported functions (no testable behavior)
			emptyContent := `package main

// This file intentionally has no testable behavior.
// It contains only comments and unexported code.

// internal is not exported and cannot be tested externally.
func internal() {
	// no-op
}
`
			goModContent := `module empty-prototype

go 1.23
`
			err = os.WriteFile(filepath.Join(emptyPrototypeDir, "main.go"), []byte(emptyContent), 0644)
			Expect(err).NotTo(HaveOccurred(), "failed to write main.go")

			err = os.WriteFile(filepath.Join(emptyPrototypeDir, "go.mod"), []byte(goModContent), 0644)
			Expect(err).NotTo(HaveOccurred(), "failed to write go.mod")

			// Create spec output directory
			specOutputDir, err = os.MkdirTemp("", "empty-prototype-output-*")
			Expect(err).NotTo(HaveOccurred(), "failed to create spec output directory")
		})

		AfterAll(func() {
			if emptyPrototypeDir != "" {
				os.RemoveAll(emptyPrototypeDir)
			}
			if specOutputDir != "" {
				os.RemoveAll(specOutputDir)
			}
		})

		It("[test_id:TS-GH-4-003] should return a clear error when prototype has no testable behavior", func() {
			// Execute the vibe-to-spec workflow on the empty prototype
			cmdCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
			defer cancel()

			cmd := exec.CommandContext(cmdCtx, "fullsend", "vibe-to-spec",
				"--input", emptyPrototypeDir,
				"--output", specOutputDir,
			)

			output, err := cmd.CombinedOutput()
			outputStr := string(output)

			// Workflow should return a non-zero exit code
			Expect(err).To(HaveOccurred(),
				"vibe-to-spec should fail with non-zero exit code when prototype has no testable behavior.\nOutput: %s", outputStr)

			// Error message should clearly indicate the problem
			Expect(outputStr).To(SatisfyAny(
				ContainSubstring("no testable"),
				ContainSubstring("no exported"),
				ContainSubstring("no functions"),
				ContainSubstring("insufficient"),
				ContainSubstring("empty"),
			), "error message should indicate prototype lacks testable behavior.\nActual output: %s", outputStr)
		})
	})

	Context("Verify error for ambiguous or contradictory prototype input", Ordered, func() {
		var (
			ambiguousPrototypeDir string
			specOutputDir         string
		)

		BeforeAll(func() {
			var err error

			// Create a prototype directory with contradictory behavior
			ambiguousPrototypeDir, err = os.MkdirTemp("", "ambiguous-prototype-*")
			Expect(err).NotTo(HaveOccurred(), "failed to create ambiguous prototype directory")

			// Write Go code where comments contradict the implementation
			contradictoryContent := `package main

import "strings"

// ToUpperCase converts the input string to uppercase.
// This function MUST return all characters in UPPER CASE.
func ToUpperCase(s string) string {
	// BUG: implementation does lowercase, contradicting the doc and name
	return strings.ToLower(s)
}

// IsPositive returns true if the number is positive (greater than zero).
func IsPositive(n int) bool {
	// BUG: returns true for negative numbers too
	return n != 0
}

// Reverse returns the input string reversed.
func Reverse(s string) string {
	// BUG: returns the same string, not reversed
	return s
}
`
			goModContent := `module ambiguous-prototype

go 1.23
`
			err = os.WriteFile(filepath.Join(ambiguousPrototypeDir, "contradictory.go"), []byte(contradictoryContent), 0644)
			Expect(err).NotTo(HaveOccurred(), "failed to write contradictory.go")

			err = os.WriteFile(filepath.Join(ambiguousPrototypeDir, "go.mod"), []byte(goModContent), 0644)
			Expect(err).NotTo(HaveOccurred(), "failed to write go.mod")

			// Create spec output directory
			specOutputDir, err = os.MkdirTemp("", "ambiguous-output-*")
			Expect(err).NotTo(HaveOccurred(), "failed to create spec output directory")
		})

		AfterAll(func() {
			if ambiguousPrototypeDir != "" {
				os.RemoveAll(ambiguousPrototypeDir)
			}
			if specOutputDir != "" {
				os.RemoveAll(specOutputDir)
			}
		})

		It("[test_id:TS-GH-4-009] should return a clear error when prototype input is ambiguous or contradictory", func() {
			// Execute the vibe-to-spec workflow on the ambiguous prototype
			cmdCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
			defer cancel()

			cmd := exec.CommandContext(cmdCtx, "fullsend", "vibe-to-spec",
				"--input", ambiguousPrototypeDir,
				"--output", specOutputDir,
			)

			output, err := cmd.CombinedOutput()
			outputStr := string(output)

			// Workflow should return a non-zero exit code for ambiguous input
			Expect(err).To(HaveOccurred(),
				"vibe-to-spec should fail when prototype has contradictory behavior.\nOutput: %s", outputStr)

			// Error message should explain the ambiguity
			Expect(outputStr).To(SatisfyAny(
				ContainSubstring("ambiguous"),
				ContainSubstring("contradictory"),
				ContainSubstring("inconsistent"),
				ContainSubstring("conflict"),
				ContainSubstring("mismatch"),
			), "error message should explain the prototype ambiguity.\nActual output: %s", outputStr)
		})
	})
})
