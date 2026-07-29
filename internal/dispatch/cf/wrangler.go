// Package cf implements Cloudflare Workers deployment for the token mint.
// It shells out to Wrangler for deploy and teardown, supporting both
// production deploys (wrangler deploy) and preview deploys
// (wrangler versions upload --preview-alias).
package cf

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// previewAliasPattern validates Wrangler preview alias values.
// Aliases are used as DNS labels, so they must be lowercase alphanumeric
// with hyphens, no leading/trailing hyphens, max 63 chars.
var previewAliasPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

// ValidatePreviewAlias checks that alias is a valid Wrangler preview alias.
func ValidatePreviewAlias(alias string) error {
	if alias == "" {
		return fmt.Errorf("preview alias cannot be empty")
	}
	if !previewAliasPattern.MatchString(alias) {
		return fmt.Errorf("invalid preview alias %q: must be lowercase alphanumeric with hyphens, 1-63 chars, no leading/trailing hyphens", alias)
	}
	return nil
}

// workerNamePattern validates Cloudflare Worker script names.
var workerNamePattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

// ValidateWorkerName checks that name is a valid Worker script name.
func ValidateWorkerName(name string) error {
	if name == "" {
		return fmt.Errorf("worker name cannot be empty")
	}
	if !workerNamePattern.MatchString(name) {
		return fmt.Errorf("invalid worker name %q: must be lowercase alphanumeric with hyphens, 1-63 chars", name)
	}
	return nil
}

// Runner abstracts Wrangler CLI operations for testing.
type Runner interface {
	// Deploy runs a production deploy (wrangler deploy).
	Deploy(ctx context.Context, workerName, sourceDir string) error

	// PreviewUpload runs a preview version upload
	// (wrangler versions upload --preview-alias=<alias>).
	PreviewUpload(ctx context.Context, workerName, sourceDir, alias string) error

	// Teardown removes a production Worker (wrangler delete).
	Teardown(ctx context.Context, workerName string) error

	// PreviewTeardown is a no-op: preview aliases are ephemeral and do not
	// require explicit cleanup. Deleting the durable Worker script would be
	// destructive; the alias simply stops routing when a new version is
	// promoted or the alias is reassigned.
	PreviewTeardown(ctx context.Context, workerName, alias string) error
}

// LiveRunner executes real Wrangler CLI commands.
type LiveRunner struct {
	// WranglerBin is the path to the wrangler binary (default: "wrangler").
	WranglerBin string
}

// NewLiveRunner creates a LiveRunner with default settings.
func NewLiveRunner() *LiveRunner {
	return &LiveRunner{WranglerBin: "wrangler"}
}

func (r *LiveRunner) wranglerBin() string {
	if r.WranglerBin != "" {
		return r.WranglerBin
	}
	return "wrangler"
}

// Deploy runs: wrangler deploy --name=<workerName>
func (r *LiveRunner) Deploy(ctx context.Context, workerName, sourceDir string) error {
	args := []string{"deploy", "--name=" + workerName}
	cmd := exec.CommandContext(ctx, r.wranglerBin(), args...)
	cmd.Dir = sourceDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("wrangler deploy failed: %w\noutput: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// PreviewUpload runs: wrangler versions upload --name=<workerName> --preview-alias=<alias>
func (r *LiveRunner) PreviewUpload(ctx context.Context, workerName, sourceDir, alias string) error {
	args := []string{"versions", "upload", "--name=" + workerName, "--preview-alias=" + alias}
	cmd := exec.CommandContext(ctx, r.wranglerBin(), args...)
	cmd.Dir = sourceDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("wrangler versions upload failed: %w\noutput: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Teardown runs: wrangler delete --name=<workerName> --force
func (r *LiveRunner) Teardown(ctx context.Context, workerName string) error {
	args := []string{"delete", "--name=" + workerName, "--force"}
	cmd := exec.CommandContext(ctx, r.wranglerBin(), args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("wrangler delete failed: %w\noutput: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// PreviewTeardown is intentionally a no-op. Preview aliases are ephemeral
// Wrangler constructs that do not need explicit cleanup. Removing them
// would require deleting the entire Worker script, which is destructive
// and violates the acceptance criterion: "preview teardown does not delete
// the durable Worker script by mistake."
func (r *LiveRunner) PreviewTeardown(_ context.Context, _, _ string) error {
	return nil
}

// PreviewURL returns the deterministic preview URL for a given worker name,
// alias, and workers.dev subdomain.
//
// The Wrangler preview-alias URL pattern is:
//
//	https://<alias>-<worker-name>.<subdomain>.workers.dev
//
// Callers (BT, operators) can compute this URL from known inputs without
// scraping Wrangler output.
func PreviewURL(alias, workerName, subdomain string) string {
	return fmt.Sprintf("https://%s-%s.%s.workers.dev", alias, workerName, subdomain)
}

// ProductionURL returns the production URL for a given worker name and
// workers.dev subdomain.
//
//	https://<worker-name>.<subdomain>.workers.dev
func ProductionURL(workerName, subdomain string) string {
	return fmt.Sprintf("https://%s.%s.workers.dev", workerName, subdomain)
}
