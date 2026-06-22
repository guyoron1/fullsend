package cli

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/forge"
	"github.com/fullsend-ai/fullsend/internal/sticky"
	"github.com/fullsend-ai/fullsend/internal/ui"

	"bytes"
)

// TS-GH-69-012: Verify post-review command flow posts sanitized content to forge.
//
// This functional test exercises the sanitize-then-post flow that the
// post-review command performs: parse review result -> sanitize -> post
// to forge via sticky.Post + submitFormalReview. Uses a fake forge client
// to capture the content actually delivered to the API.
func TestPostReviewCommand_PostsSanitizedContentToForge(t *testing.T) {
	var buf bytes.Buffer
	printer := ui.New(&buf)

	secret := "ghp_1234567890abcdefABCDEF1234567890abcd"
	parsed := ReviewResult{
		Body:   "Review complete. CI token: " + secret,
		Action: "comment",
	}

	// Step 1: Sanitize (mirrors the command's sanitization call).
	parsed = sanitizeReviewResult(parsed, printer)

	// Step 2: Post to forge via sticky.Post (uses fake client).
	fc := forge.NewFakeClient()
	fc.AuthenticatedUser = "fullsend-bot"

	cfg := sticky.Config{
		Marker: reviewMarker,
	}

	commentURL, err := sticky.Post(context.Background(), fc, "acme", "repo", 1, parsed.Body, cfg, printer)
	require.NoError(t, err)
	assert.NotEmpty(t, commentURL)

	// ASSERT-01: Forge API receives sanitized content — no raw secret.
	comments := fc.IssueComments["acme/repo/1"]
	require.NotEmpty(t, comments, "comment should be posted to forge")

	capturedBody := comments[0].Body
	assert.NotContains(t, capturedBody, "ghp_1234567890",
		"forge API should receive sanitized content without raw secret")
	assert.NotContains(t, capturedBody, secret,
		"full secret must not appear in posted content")

	// ASSERT-02: Non-secret content preserved in forge post.
	assert.Contains(t, capturedBody, "Review complete",
		"non-secret review text should be preserved in posted content")
}
