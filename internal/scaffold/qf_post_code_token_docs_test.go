package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCodeAgentEnvDocumentsWritePermissions covers STD scenario 9 (TS-GH84-009).
//
// The code-agent.env file comments must accurately describe the coder role's
// actual permissions (write access) instead of incorrectly stating the token
// is read-only.
func TestCodeAgentEnvDocumentsWritePermissions(t *testing.T) {
	envPath := filepath.Join("fullsend-repo", "env", "code-agent.env")

	content, err := os.ReadFile(envPath)
	require.NoError(t, err, "code-agent.env must be readable")

	fileContent := string(content)

	// Extract only comment lines for inspection (lines starting with #).
	var commentLines []string
	for _, line := range strings.Split(fileContent, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			commentLines = append(commentLines, trimmed)
		}
	}
	comments := strings.Join(commentLines, "\n")

	// ASSERT-01: Comments reference the coder role and its token.
	// The GH_TOKEN comment should exist and describe the coder app token.
	assert.Contains(t, comments, "coder",
		"comments should reference the coder role")

	// ASSERT-02: No misleading read-only claims about the sandbox token.
	// We check specifically for "read-only" or "read only" claims that
	// describe the GH_TOKEN or sandbox token as read-only, which would be
	// inaccurate for the coder role.
	lowerComments := strings.ToLower(comments)
	readOnlyPresent := strings.Contains(lowerComments, "read-only") ||
		strings.Contains(lowerComments, "read only")
	assert.False(t, readOnlyPresent,
		"comments must not describe the sandbox GH_TOKEN as read-only; "+
			"the coder role has write permissions")
}

// TestCodeAgentEnvDocumentsWorkflowsWriteOmission covers STD scenario 10 (TS-GH84-010).
//
// The code-agent.env file must document that the coder GitHub App role
// intentionally omits the workflows:write permission as a defense-in-depth
// measure.
func TestCodeAgentEnvDocumentsWorkflowsWriteOmission(t *testing.T) {
	envPath := filepath.Join("fullsend-repo", "env", "code-agent.env")

	content, err := os.ReadFile(envPath)
	require.NoError(t, err, "code-agent.env must be readable")

	fileContent := string(content)

	// Extract comment lines.
	var commentLines []string
	for _, line := range strings.Split(fileContent, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			commentLines = append(commentLines, trimmed)
		}
	}
	comments := strings.Join(commentLines, "\n")
	lowerComments := strings.ToLower(comments)

	// ASSERT-01: Comments document workflows:write omission.
	// The comments should mention "workflows:write" or "workflows" in the
	// context of permission documentation.
	assert.True(t,
		strings.Contains(lowerComments, "workflows:write") ||
			strings.Contains(lowerComments, "workflows"),
		"comments must reference workflows:write omission in permission docs")

	// Additional check: the omission should be intentional/security-related.
	assert.True(t,
		strings.Contains(lowerComments, "omit") ||
			strings.Contains(lowerComments, "reject") ||
			strings.Contains(lowerComments, "without"),
		"comments should explain the security rationale for omitting workflows:write")
}
