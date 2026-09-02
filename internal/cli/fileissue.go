package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/spf13/cobra"

	"github.com/fullsend-ai/fullsend/internal/forge"
	"github.com/fullsend-ai/fullsend/internal/ui"
)

// FileIssueResult is the JSON output of the file-issue command.
type FileIssueResult struct {
	Created     bool   `json:"created"`
	URL         string `json:"url"`
	Number      int    `json:"number"`
	DuplicateOf string `json:"duplicate_of,omitempty"`
}

// defaultDedupWindow is the lookback duration for dedup searches.
const defaultDedupWindow = 30 * time.Minute

// defaultSimilarityThreshold is the minimum Jaccard word-overlap
// coefficient for two titles to be considered duplicates.
const defaultSimilarityThreshold = 0.6

func newFileIssueCmd() *cobra.Command {
	var (
		repo        string
		title       string
		body        string
		labels      []string
		token       string
		creator     string
		dedupWindow time.Duration
		dryRun      bool
	)

	cmd := &cobra.Command{
		Use:   "file-issue",
		Short: "Create a GitHub issue with dedup guard",
		Long: `Creates a GitHub issue after checking for recent duplicates.

Before filing, searches for issues created by the same author in the
target repo within a configurable time window. If an existing issue
has a similar title (measured by word-overlap similarity), the new
issue is skipped, a comment is added to the existing issue noting
the additional evidence, and the existing issue URL is returned.

This prevents duplicate issue filing when multiple concurrent agents
(e.g., retro agents) identify the same improvement independently.

Output is JSON with fields: created, url, number, duplicate_of.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			printer := ui.New(os.Stderr)

			if token == "" {
				token = os.Getenv("GITHUB_TOKEN")
			}
			if token == "" {
				return fmt.Errorf("--token or GITHUB_TOKEN required")
			}

			parts := strings.SplitN(repo, "/", 2)
			if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
				return fmt.Errorf("--repo must be in owner/repo format, got %q", repo)
			}
			owner, repoName := parts[0], parts[1]

			if title == "" {
				return fmt.Errorf("--title is required")
			}

			bodyText := body
			if bodyText == "" {
				// Read body from stdin if not provided via flag.
				raw, err := readBody("-")
				if err != nil {
					return fmt.Errorf("reading issue body from stdin: %w", err)
				}
				bodyText = raw
			}

			client := newGitHubLiveClient(token, "")

			result, err := fileIssueWithDedup(
				cmd.Context(), client, owner, repoName,
				title, bodyText, labels,
				creator, dedupWindow, dryRun, printer,
			)
			if err != nil {
				return err
			}

			return json.NewEncoder(os.Stdout).Encode(result)
		},
	}

	cmd.Flags().StringVar(&repo, "repo", "", "repository in owner/repo format (required)")
	cmd.Flags().StringVar(&title, "title", "", "issue title (required)")
	cmd.Flags().StringVar(&body, "body", "", "issue body (reads stdin if omitted)")
	cmd.Flags().StringSliceVar(&labels, "label", nil, "labels to apply (repeatable)")
	cmd.Flags().StringVar(&token, "token", "", "GitHub token (default: $GITHUB_TOKEN)")
	cmd.Flags().StringVar(&creator, "creator", "", "author login for dedup search (skip dedup if empty)")
	cmd.Flags().DurationVar(&dedupWindow, "dedup-window", defaultDedupWindow, "lookback window for dedup search")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "check for duplicates without creating")
	_ = cmd.MarkFlagRequired("repo")
	_ = cmd.MarkFlagRequired("title")

	return cmd
}

// fileIssueWithDedup checks for duplicates and, if none are found,
// creates the issue. It is exported-name-style (lowercase, unexported)
// to keep the function testable via the forge.Client interface.
func fileIssueWithDedup(
	ctx context.Context,
	client forge.Client,
	owner, repo, title, body string,
	labels []string,
	creator string,
	dedupWindow time.Duration,
	dryRun bool,
	printer *ui.Printer,
) (*FileIssueResult, error) {
	printer.Header("File Issue")

	// Phase 1: Dedup search (only when creator is specified).
	if creator != "" {
		printer.StepStart(fmt.Sprintf("Searching for recent issues by %s in %s/%s", creator, owner, repo))

		since := time.Now().UTC().Add(-dedupWindow)
		existing, err := client.SearchIssues(ctx, forge.IssueSearchOptions{
			Owner:   owner,
			Repo:    repo,
			Creator: creator,
			Since:   since,
			State:   "open",
		})
		if err != nil {
			// Non-fatal: if the search fails (e.g., rate limit, unsupported
			// forge), fall through to creation rather than blocking filing.
			printer.StepWarn(fmt.Sprintf("Dedup search failed: %v (proceeding with creation)", err))
		} else {
			printer.StepDone(fmt.Sprintf("Found %d recent issue(s)", len(existing)))

			for _, issue := range existing {
				if titlesSimilar(title, issue.Title) {
					printer.StepInfo(fmt.Sprintf(
						"Duplicate detected: #%d %q (similarity above threshold)",
						issue.Number, issue.Title,
					))

					// Add a comment on the existing issue noting the
					// additional evidence, per issue #848.
					commentBody := fmt.Sprintf(
						"Duplicate proposal detected — another agent independently proposed:\n\n"+
							"> **%s**\n\n"+
							"Skipping duplicate issue creation.",
						title,
					)
					if _, err := client.CreateIssueComment(ctx, owner, repo, issue.Number, commentBody); err != nil {
						// Non-fatal: log and proceed with the dedup result.
						printer.StepWarn(fmt.Sprintf("Failed to comment on #%d: %v", issue.Number, err))
					} else {
						printer.StepDone(fmt.Sprintf("Added evidence comment to #%d", issue.Number))
					}

					return &FileIssueResult{
						Created:     false,
						URL:         issue.URL,
						Number:      issue.Number,
						DuplicateOf: issue.URL,
					}, nil
				}
			}
			printer.StepDone("No duplicates found")
		}
	} else {
		printer.StepInfo("Skipping dedup (no --creator specified)")
	}

	// Phase 2: Create the issue.
	if dryRun {
		printer.StepInfo("Dry run — would create issue")
		return &FileIssueResult{Created: false}, nil
	}

	printer.StepStart(fmt.Sprintf("Creating issue in %s/%s", owner, repo))
	issue, err := client.CreateIssue(ctx, owner, repo, title, body, labels...)
	if err != nil {
		return nil, fmt.Errorf("creating issue: %w", err)
	}
	printer.StepDone(fmt.Sprintf("Created #%d: %s", issue.Number, issue.URL))

	return &FileIssueResult{
		Created: true,
		URL:     issue.URL,
		Number:  issue.Number,
	}, nil
}

// titlesSimilar reports whether two issue titles are similar enough to
// be considered duplicates. It uses the Jaccard coefficient of word
// sets (intersection / union) after normalizing to lowercase and
// stripping punctuation. A coefficient >= defaultSimilarityThreshold
// is considered a match.
func titlesSimilar(a, b string) bool {
	return titleSimilarity(a, b) >= defaultSimilarityThreshold
}

// titleSimilarity computes the Jaccard similarity coefficient between
// the word sets of two strings after normalization.
func titleSimilarity(a, b string) float64 {
	wordsA := normalizeWords(a)
	wordsB := normalizeWords(b)

	if len(wordsA) == 0 && len(wordsB) == 0 {
		return 1.0
	}
	if len(wordsA) == 0 || len(wordsB) == 0 {
		return 0.0
	}

	setA := make(map[string]bool, len(wordsA))
	for _, w := range wordsA {
		setA[w] = true
	}
	setB := make(map[string]bool, len(wordsB))
	for _, w := range wordsB {
		setB[w] = true
	}

	intersection := 0
	for w := range setA {
		if setB[w] {
			intersection++
		}
	}

	union := len(setA) + len(setB) - intersection
	if union == 0 {
		return 1.0
	}

	return float64(intersection) / float64(union)
}

// normalizeWords splits a string into lowercase words, stripping
// punctuation and filtering out common stop words that don't
// contribute to semantic similarity.
func normalizeWords(s string) []string {
	lower := strings.ToLower(s)
	// Split on non-letter, non-digit characters.
	fields := strings.FieldsFunc(lower, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	// Filter out very short words and common stop words.
	result := make([]string, 0, len(fields))
	for _, w := range fields {
		if len(w) <= 1 {
			continue
		}
		if isStopWord(w) {
			continue
		}
		result = append(result, w)
	}
	return result
}

// isStopWord returns true for common English stop words that add noise
// to title similarity comparisons.
func isStopWord(w string) bool {
	switch w {
	case "the", "in", "on", "at", "to", "for", "of", "and", "or",
		"is", "it", "an", "by", "be", "as", "do", "if", "so",
		"no", "up", "add", "with", "from", "that", "this", "when":
		return true
	}
	return false
}
