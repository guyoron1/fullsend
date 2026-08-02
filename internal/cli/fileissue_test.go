package cli

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/forge"
	"github.com/fullsend-ai/fullsend/internal/ui"
)

func TestTitleSimilarity(t *testing.T) {
	tests := []struct {
		name    string
		a, b    string
		similar bool
	}{
		{
			name:    "identical titles",
			a:       "Add empty-diff guard to PR creation",
			b:       "Add empty-diff guard to PR creation",
			similar: true,
		},
		{
			name:    "near-identical with minor rewording",
			a:       "Add empty-diff guard to PR creation",
			b:       "Add empty-diff guard for PR creation",
			similar: true,
		},
		{
			name:    "same concept slightly rephrased",
			a:       "Add empty-diff guard to PR creation",
			b:       "Add empty diff guard to PR creation logic",
			similar: true,
		},
		{
			name:    "completely different titles",
			a:       "Add empty-diff guard to PR creation",
			b:       "Fix authentication timeout in retry loop",
			similar: false,
		},
		{
			name:    "different topics same repo",
			a:       "Add retro dedup guard for concurrent proposals",
			b:       "Improve CI pipeline caching strategy",
			similar: false,
		},
		{
			name:    "empty titles",
			a:       "",
			b:       "",
			similar: true,
		},
		{
			name:    "one empty title",
			a:       "Add feature",
			b:       "",
			similar: false,
		},
		{
			name:    "case insensitive match",
			a:       "Add Empty-Diff Guard",
			b:       "add empty-diff guard",
			similar: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := titlesSimilar(tt.a, tt.b)
			assert.Equal(t, tt.similar, got,
				"titlesSimilar(%q, %q) = %v, want %v (similarity=%.2f)",
				tt.a, tt.b, got, tt.similar, titleSimilarity(tt.a, tt.b))
		})
	}
}

func TestNormalizeWords(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "simple phrase",
			input: "Add empty-diff guard",
			want:  []string{"empty", "diff", "guard"},
		},
		{
			name:  "strips stop words",
			input: "Add the guard to the PR for creation",
			want:  []string{"guard", "pr", "creation"},
		},
		{
			name:  "strips single chars",
			input: "a b c hello world",
			want:  []string{"hello", "world"},
		},
		{
			name:  "preserves numbers",
			input: "fix issue 42 in module",
			want:  []string{"fix", "issue", "42", "module"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeWords(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFileIssueWithDedup_DuplicateFound(t *testing.T) {
	fc := &forge.FakeClient{
		OpenIssues: map[string][]forge.Issue{
			"org/repo": {
				{
					Number: 42,
					Title:  "Add empty-diff guard to PR creation",
					URL:    "https://github.com/org/repo/issues/42",
				},
			},
		},
	}

	printer := ui.New(&discardWriter{})
	result, err := fileIssueWithDedup(
		context.Background(), fc, "org", "repo",
		"Add empty-diff guard for PR creation", // similar title
		"Body text",
		[]string{"feature"},
		"retro-bot[bot]",
		30*time.Minute,
		false,
		printer,
	)
	require.NoError(t, err)
	assert.False(t, result.Created)
	assert.Equal(t, "https://github.com/org/repo/issues/42", result.DuplicateOf)
	assert.Equal(t, 42, result.Number)
	// No issue should have been created.
	assert.Empty(t, fc.CreatedIssues)
	// A comment should have been added to the existing issue.
	comments := fc.IssueComments["org/repo/42"]
	require.Len(t, comments, 1)
	assert.Contains(t, comments[0].Body, "Duplicate proposal detected")
	assert.Contains(t, comments[0].Body, "Add empty-diff guard for PR creation")
}

func TestFileIssueWithDedup_DuplicateCommentFailureNonFatal(t *testing.T) {
	fc := &forge.FakeClient{
		OpenIssues: map[string][]forge.Issue{
			"org/repo": {
				{
					Number: 42,
					Title:  "Add empty-diff guard to PR creation",
					URL:    "https://github.com/org/repo/issues/42",
				},
			},
		},
		Errors: map[string]error{
			"CreateIssueComment": assert.AnError,
		},
	}

	printer := ui.New(&discardWriter{})
	result, err := fileIssueWithDedup(
		context.Background(), fc, "org", "repo",
		"Add empty-diff guard for PR creation", // similar title
		"Body text",
		[]string{"feature"},
		"retro-bot[bot]",
		30*time.Minute,
		false,
		printer,
	)
	// Comment failure should not prevent dedup from working.
	require.NoError(t, err)
	assert.False(t, result.Created)
	assert.Equal(t, "https://github.com/org/repo/issues/42", result.DuplicateOf)
	assert.Empty(t, fc.CreatedIssues)
}

func TestFileIssueWithDedup_NoDuplicate(t *testing.T) {
	fc := &forge.FakeClient{
		OpenIssues: map[string][]forge.Issue{
			"org/repo": {
				{
					Number: 10,
					Title:  "Completely unrelated issue",
					URL:    "https://github.com/org/repo/issues/10",
				},
			},
		},
	}

	printer := ui.New(&discardWriter{})
	result, err := fileIssueWithDedup(
		context.Background(), fc, "org", "repo",
		"Add empty-diff guard for PR creation",
		"Body text",
		[]string{"feature"},
		"retro-bot[bot]",
		30*time.Minute,
		false,
		printer,
	)
	require.NoError(t, err)
	assert.True(t, result.Created)
	assert.Equal(t, "", result.DuplicateOf)
	assert.Len(t, fc.CreatedIssues, 1)
	assert.Equal(t, "Add empty-diff guard for PR creation", fc.CreatedIssues[0].Title)
}

func TestFileIssueWithDedup_NoCreator(t *testing.T) {
	fc := &forge.FakeClient{}

	printer := ui.New(&discardWriter{})
	result, err := fileIssueWithDedup(
		context.Background(), fc, "org", "repo",
		"New issue title",
		"Body text",
		nil,
		"", // no creator — skip dedup
		30*time.Minute,
		false,
		printer,
	)
	require.NoError(t, err)
	assert.True(t, result.Created)
	assert.Len(t, fc.CreatedIssues, 1)
}

func TestFileIssueWithDedup_SearchFailsFallsThrough(t *testing.T) {
	fc := &forge.FakeClient{
		Errors: map[string]error{
			"SearchIssues": assert.AnError,
		},
	}

	printer := ui.New(&discardWriter{})
	result, err := fileIssueWithDedup(
		context.Background(), fc, "org", "repo",
		"New issue title",
		"Body text",
		nil,
		"retro-bot[bot]",
		30*time.Minute,
		false,
		printer,
	)
	require.NoError(t, err)
	// Should still create despite search failure.
	assert.True(t, result.Created)
	assert.Len(t, fc.CreatedIssues, 1)
}

func TestFileIssueWithDedup_DryRun(t *testing.T) {
	fc := &forge.FakeClient{}

	printer := ui.New(&discardWriter{})
	result, err := fileIssueWithDedup(
		context.Background(), fc, "org", "repo",
		"New issue title",
		"Body text",
		nil,
		"retro-bot[bot]",
		30*time.Minute,
		true, // dry run
		printer,
	)
	require.NoError(t, err)
	assert.False(t, result.Created)
	assert.Empty(t, fc.CreatedIssues)
}

func TestFileIssueWithDedup_DistinctProposalsPass(t *testing.T) {
	// Scenario 2 from the issue: distinct proposals targeting the
	// same repo should both be created.
	fc := &forge.FakeClient{
		OpenIssues: map[string][]forge.Issue{
			"org/repo": {
				{
					Number: 1,
					Title:  "Add empty-diff guard to PR creation",
					URL:    "https://github.com/org/repo/issues/1",
				},
			},
		},
	}

	printer := ui.New(&discardWriter{})
	result, err := fileIssueWithDedup(
		context.Background(), fc, "org", "repo",
		"Improve CI pipeline caching strategy", // different topic
		"Body text",
		nil,
		"retro-bot[bot]",
		30*time.Minute,
		false,
		printer,
	)
	require.NoError(t, err)
	assert.True(t, result.Created)
	assert.Equal(t, "", result.DuplicateOf)
}

func TestFileIssueWithDedup_DifferentReposNeverSuppress(t *testing.T) {
	// Edge case: proposals targeting different repos should never
	// suppress each other — the search is scoped to owner/repo.
	fc := &forge.FakeClient{
		OpenIssues: map[string][]forge.Issue{
			"org/repo-a": {
				{
					Number: 1,
					Title:  "Add empty-diff guard to PR creation",
					URL:    "https://github.com/org/repo-a/issues/1",
				},
			},
			// org/repo-b has no issues
		},
	}

	printer := ui.New(&discardWriter{})
	result, err := fileIssueWithDedup(
		context.Background(), fc, "org", "repo-b",
		"Add empty-diff guard to PR creation", // same title, different repo
		"Body text",
		nil,
		"retro-bot[bot]",
		30*time.Minute,
		false,
		printer,
	)
	require.NoError(t, err)
	assert.True(t, result.Created)
	assert.Equal(t, "", result.DuplicateOf)
}
