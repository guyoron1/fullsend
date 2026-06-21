package review

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

/*
File Classification Tests — GH-2096

Validates that the security-triage classifier correctly categorizes files as
security-critical or standard based on path patterns and content heuristics.
Accurate classification is the foundation of the two-pass review strategy.

STP Reference: outputs/stp/GH-2096/GH-2096_test_plan.md
STD Scenarios: TS-GH-2096-004, TS-GH-2096-005, TS-GH-2096-006, TS-GH-2096-007
*/

// FileClassification represents the security relevance of a changed file.
type FileClassification string

const (
	SecurityCritical FileClassification = "security-critical"
	Standard         FileClassification = "standard"
)

// securityPathPatterns are directory path segments that indicate security-critical code.
var securityPathPatterns = []string{
	"/mint/", "/mintcore/", "/auth/", "/oidc/", "/rbac/",
	"/permissions/", "/secrets/", "/crypto/", "/token/", "/tokens/",
	"/trust/", "/policies/",
}

// classifyFile classifies a file by path pattern alone.
func classifyFile(path string) FileClassification {
	for _, pattern := range securityPathPatterns {
		if strings.Contains(path, pattern) {
			return SecurityCritical
		}
	}
	if strings.Contains(path, "CODEOWNERS") {
		return SecurityCritical
	}
	return Standard
}

// classifyFileWithContent classifies a file using both path and diff content.
// Content heuristics catch security-relevant changes that path patterns miss.
// When in doubt, the function errs on the side of SecurityCritical.
func classifyFileWithContent(path, diffContent string) FileClassification {
	// Path-based classification first
	if classifyFile(path) == SecurityCritical {
		return SecurityCritical
	}

	// Content heuristics for workflow files
	if strings.Contains(path, ".github/workflows/") {
		contentHeuristics := []string{
			"permissions:", "secrets:", "pull_request_target",
		}
		for _, keyword := range contentHeuristics {
			if strings.Contains(diffContent, keyword) {
				return SecurityCritical
			}
		}
	}

	// Content heuristics for auth-related keywords in any file
	authKeywords := []string{
		"auth", "token", "credential", "secret", "permission",
		"oauth", "jwt", "certificate", "session",
	}
	lowerDiff := strings.ToLower(diffContent)
	for _, keyword := range authKeywords {
		if strings.Contains(lowerDiff, keyword) {
			return SecurityCritical
		}
	}

	return Standard
}

func TestFileClassification(t *testing.T) {
	/*
		Preconditions:
			- Go development environment with Go 1.26+
			- fullsend repository with two-pass review strategy changes
	*/

	// TS-GH-2096-004: Verify mint/auth/oidc paths classified as security-critical
	t.Run("mint/auth/oidc paths classified as security-critical", func(t *testing.T) {
		securityPaths := []struct {
			path     string
			category string
		}{
			{"internal/mint/handler.go", "mint"},
			{"internal/mintcore/wif.go", "mintcore"},
			{"internal/auth/oauth.go", "auth"},
			{"cmd/oidc/provider.go", "oidc"},
		}

		for _, tc := range securityPaths {
			t.Run(tc.category, func(t *testing.T) {
				result := classifyFile(tc.path)
				assert.Equal(t, SecurityCritical, result,
					"file %q (category: %s) must be classified as security-critical",
					tc.path, tc.category)
			})
		}
	})

	// TS-GH-2096-005: Verify workflow files with permissions blocks classified as security-critical
	t.Run("workflow files with permissions blocks classified as security-critical", func(t *testing.T) {
		workflowPath := ".github/workflows/deploy.yml"

		t.Run("with permissions block", func(t *testing.T) {
			diffContent := `+permissions:
+  contents: write
+  id-token: write`
			result := classifyFileWithContent(workflowPath, diffContent)
			assert.Equal(t, SecurityCritical, result,
				"workflow with permissions block must be security-critical")
		})

		t.Run("without permissions block", func(t *testing.T) {
			diffContent := `+  - name: Run tests
+    run: go test ./...`
			result := classifyFileWithContent(workflowPath, diffContent)
			assert.Equal(t, Standard, result,
				"workflow without security-sensitive content should be standard")
		})
	})

	// TS-GH-2096-006: Verify non-security files classified as standard
	t.Run("non-security files classified as standard", func(t *testing.T) {
		standardPaths := []struct {
			path     string
			category string
		}{
			{"docs/guide.md", "documentation"},
			{"internal/cli/run_test.go", "test file"},
			{"web/components/Button.tsx", "UI component"},
			{"config/settings.yaml", "configuration"},
			{"README.md", "readme"},
		}

		for _, tc := range standardPaths {
			t.Run(tc.category, func(t *testing.T) {
				result := classifyFile(tc.path)
				assert.Equal(t, Standard, result,
					"file %q (category: %s) must be classified as standard",
					tc.path, tc.category)
			})
		}
	})

	// TS-GH-2096-007: Verify ambiguous files default to security-critical
	t.Run("ambiguous files default to security-critical", func(t *testing.T) {
		t.Run("auth keywords in non-security path", func(t *testing.T) {
			path := "internal/api/handler.go"
			diffContent := `+func (h *Handler) ValidateAuthToken(ctx context.Context) error {`
			result := classifyFileWithContent(path, diffContent)
			assert.Equal(t, SecurityCritical, result,
				"files mentioning auth keywords in diff must default to security-critical")
		})

		t.Run("credential reference in utils", func(t *testing.T) {
			path := "pkg/utils/config.go"
			diffContent := `+  credential := os.Getenv("SERVICE_CREDENTIAL")`
			result := classifyFileWithContent(path, diffContent)
			assert.Equal(t, SecurityCritical, result,
				"files referencing credentials must default to security-critical")
		})
	})
}
