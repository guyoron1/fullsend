//go:build e2e

package e2e

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AI feature file generation", Serial, func() {
	var (
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		// Ensure the fullsend CLI is available
		_, err := exec.LookPath("fullsend")
		if err != nil {
			Skip("fullsend CLI not found in PATH — skipping AI feature file tests")
		}
		// Ensure LLM endpoint is configured
		if os.Getenv("LLM_ENDPOINT") == "" {
			Skip("LLM_ENDPOINT not set — skipping AI feature file tests")
		}
	})

	Context("Verify AI generates functional requirements section from prototype input", Ordered, func() {
		var (
			prototypeDir  string
			featureOutput string
		)

		BeforeAll(func() {
			var err error

			// Create prototype with multiple testable functions
			prototypeDir, err = os.MkdirTemp("", "feature-prototype-*")
			Expect(err).NotTo(HaveOccurred())

			calcContent := `package calculator

// Add returns the sum of two integers.
func Add(a, b int) int {
	return a + b
}

// Subtract returns the difference of two integers.
func Subtract(a, b int) int {
	return a - b
}

// IsEven returns true if the number is even.
func IsEven(n int) bool {
	return n%2 == 0
}
`
			goModContent := `module calculator

go 1.23
`
			err = os.WriteFile(filepath.Join(prototypeDir, "calculator.go"), []byte(calcContent), 0644)
			Expect(err).NotTo(HaveOccurred())
			err = os.WriteFile(filepath.Join(prototypeDir, "go.mod"), []byte(goModContent), 0644)
			Expect(err).NotTo(HaveOccurred())

			// Create feature output directory
			featureOutput, err = os.MkdirTemp("", "feature-output-*")
			Expect(err).NotTo(HaveOccurred())

			// Generate the feature file from prototype
			cmdCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			cmd := exec.CommandContext(cmdCtx, "fullsend", "vibe-to-spec",
				"--input", prototypeDir,
				"--output", featureOutput,
			)
			output, runErr := cmd.CombinedOutput()
			Expect(runErr).NotTo(HaveOccurred(),
				"feature file generation should succeed.\nOutput: %s", string(output))
		})

		AfterAll(func() {
			if prototypeDir != "" {
				os.RemoveAll(prototypeDir)
			}
			if featureOutput != "" {
				os.RemoveAll(featureOutput)
			}
		})

		It("[test_id:TS-GH-4-007] should generate a feature file containing a functional requirements section", func() {
			// Find the generated feature file
			featureFile := findFeatureFile(featureOutput)
			Expect(featureFile).NotTo(BeEmpty(),
				"feature file should exist in output directory %s", featureOutput)

			// Read the feature file content
			content, err := os.ReadFile(featureFile)
			Expect(err).NotTo(HaveOccurred(), "failed to read generated feature file")
			Expect(content).NotTo(BeEmpty(), "generated feature file should not be empty")

			contentStr := string(content)

			// Verify the feature file contains a functional requirements section
			Expect(contentStr).To(SatisfyAny(
				ContainSubstring("functional_requirements"),
				ContainSubstring("functional-requirements"),
				ContainSubstring("Functional Requirements"),
				ContainSubstring("requirements"),
			), "feature file should contain a functional requirements section.\nContent preview: %.500s", contentStr)

			// Verify requirements are structured — look for numbered/discrete items
			// Requirements should appear as a list or structured entries
			hasStructuredRequirements := false
			lines := strings.Split(contentStr, "\n")
			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				// Check for list markers (YAML list, numbered list, etc.)
				if strings.HasPrefix(trimmed, "- ") ||
					strings.HasPrefix(trimmed, "1.") ||
					strings.HasPrefix(trimmed, "id:") ||
					strings.HasPrefix(trimmed, "\"id\"") {
					hasStructuredRequirements = true
					break
				}
			}
			Expect(hasStructuredRequirements).To(BeTrue(),
				"functional requirements should be structured as discrete items.\nContent preview: %.500s", contentStr)
		})
	})

	Context("Verify AI generates acceptance scenarios with pass/fail criteria from prototype", Ordered, func() {
		var (
			prototypeDir   string
			scenarioOutput string
		)

		BeforeAll(func() {
			var err error

			// Create prototype with clear input/output behavior
			prototypeDir, err = os.MkdirTemp("", "scenario-prototype-*")
			Expect(err).NotTo(HaveOccurred())

			handlerContent := `package handler

import (
	"fmt"
	"strings"
)

// Process takes a string input, validates it, and returns the uppercase version.
// Returns an error if the input is empty.
func Process(input string) (string, error) {
	if input == "" {
		return "", fmt.Errorf("empty input: input must not be empty")
	}
	return strings.ToUpper(input), nil
}

// Validate checks if the input meets minimum length requirements.
// Returns true if the input has at least 3 characters.
func Validate(input string) bool {
	return len(input) >= 3
}
`
			goModContent := `module handler

go 1.23
`
			err = os.WriteFile(filepath.Join(prototypeDir, "handler.go"), []byte(handlerContent), 0644)
			Expect(err).NotTo(HaveOccurred())
			err = os.WriteFile(filepath.Join(prototypeDir, "go.mod"), []byte(goModContent), 0644)
			Expect(err).NotTo(HaveOccurred())

			// Create output directory
			scenarioOutput, err = os.MkdirTemp("", "scenario-output-*")
			Expect(err).NotTo(HaveOccurred())

			// Generate the feature file
			cmdCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			cmd := exec.CommandContext(cmdCtx, "fullsend", "vibe-to-spec",
				"--input", prototypeDir,
				"--output", scenarioOutput,
			)
			output, runErr := cmd.CombinedOutput()
			Expect(runErr).NotTo(HaveOccurred(),
				"feature file generation should succeed.\nOutput: %s", string(output))
		})

		AfterAll(func() {
			if prototypeDir != "" {
				os.RemoveAll(prototypeDir)
			}
			if scenarioOutput != "" {
				os.RemoveAll(scenarioOutput)
			}
		})

		It("[test_id:TS-GH-4-008] should generate acceptance scenarios with pass/fail criteria", func() {
			// Find the generated feature file
			featureFile := findFeatureFile(scenarioOutput)
			Expect(featureFile).NotTo(BeEmpty(),
				"feature file should exist in output directory %s", scenarioOutput)

			// Read the feature file content
			content, err := os.ReadFile(featureFile)
			Expect(err).NotTo(HaveOccurred(), "failed to read generated feature file")

			contentStr := string(content)

			// Verify the feature file contains acceptance scenarios
			Expect(contentStr).To(SatisfyAny(
				ContainSubstring("acceptance_scenarios"),
				ContainSubstring("acceptance-scenarios"),
				ContainSubstring("Acceptance Scenarios"),
				ContainSubstring("acceptance_criteria"),
				ContainSubstring("scenarios"),
				ContainSubstring("test_cases"),
			), "feature file should contain acceptance scenarios section.\nContent preview: %.500s", contentStr)

			// Verify scenarios have pass/fail criteria
			hasPassCriteria := strings.Contains(contentStr, "pass") ||
				strings.Contains(contentStr, "Pass") ||
				strings.Contains(contentStr, "PASS") ||
				strings.Contains(contentStr, "success") ||
				strings.Contains(contentStr, "expected") ||
				strings.Contains(contentStr, "should")

			hasFailCriteria := strings.Contains(contentStr, "fail") ||
				strings.Contains(contentStr, "Fail") ||
				strings.Contains(contentStr, "FAIL") ||
				strings.Contains(contentStr, "error") ||
				strings.Contains(contentStr, "invalid") ||
				strings.Contains(contentStr, "should not")

			Expect(hasPassCriteria).To(BeTrue(),
				"acceptance scenarios should include pass criteria.\nContent preview: %.500s", contentStr)

			Expect(hasFailCriteria).To(BeTrue(),
				"acceptance scenarios should include fail criteria.\nContent preview: %.500s", contentStr)
		})
	})
})

// findFeatureFile searches the given directory for a feature/spec file.
func findFeatureFile(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	// Look for feature file first, then any YAML/JSON file
	for _, entry := range entries {
		name := strings.ToLower(entry.Name())
		if strings.Contains(name, "feature") || strings.Contains(name, "spec") {
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
