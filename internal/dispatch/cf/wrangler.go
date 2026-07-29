package cf

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
)

// ValidatePreviewAlias validates that an alias is a valid DNS label.
// Valid aliases are lowercase alphanumeric, hyphens allowed in middle, 1-63 chars.
func ValidatePreviewAlias(alias string) error {
	if alias == "" {
		return fmt.Errorf("invalid preview alias: empty string")
	}
	pattern := regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
	if !pattern.MatchString(alias) {
		return fmt.Errorf("invalid preview alias: %q (must be lowercase alphanumeric, hyphens in middle, 1-63 chars)", alias)
	}
	return nil
}

// ValidateWorkerName validates that a worker name is a valid DNS label.
func ValidateWorkerName(name string) error {
	if name == "" {
		return fmt.Errorf("invalid worker name: empty string")
	}
	pattern := regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
	if !pattern.MatchString(name) {
		return fmt.Errorf("invalid worker name: %q (must be lowercase alphanumeric, hyphens in middle, 1-63 chars)", name)
	}
	return nil
}

// PreviewURL returns the preview URL for a Cloudflare Worker preview alias.
func PreviewURL(alias, workerName, subdomain string) string {
	return fmt.Sprintf("https://%s-%s.%s.workers.dev", alias, workerName, subdomain)
}

// ProductionURL returns the production URL for a Cloudflare Worker.
func ProductionURL(workerName, subdomain string) string {
	return fmt.Sprintf("https://%s.%s.workers.dev", workerName, subdomain)
}

// Runner is the interface for running Wrangler commands.
type Runner interface {
	Deploy(ctx context.Context, workerName, sourceDir string) error
	PreviewUpload(ctx context.Context, workerName, sourceDir, alias string) error
	Teardown(ctx context.Context, workerName string) error
	PreviewTeardown(ctx context.Context, workerName, alias string) error
}

// LiveRunner implements Runner using the wrangler CLI.
type LiveRunner struct{}

// Deploy runs wrangler deploy to deploy a worker to production.
func (r *LiveRunner) Deploy(ctx context.Context, workerName, sourceDir string) error {
	cmd := exec.CommandContext(ctx, "wrangler", "deploy", "--name="+workerName)
	cmd.Dir = sourceDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("wrangler deploy failed: %w\nOutput: %s", err, output)
	}
	return nil
}

// PreviewUpload runs wrangler versions upload with a preview alias.
func (r *LiveRunner) PreviewUpload(ctx context.Context, workerName, sourceDir, alias string) error {
	cmd := exec.CommandContext(ctx, "wrangler", "versions", "upload", "--name="+workerName, "--preview-alias="+alias)
	cmd.Dir = sourceDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("wrangler versions upload failed: %w\nOutput: %s", err, output)
	}
	return nil
}

// Teardown runs wrangler delete to remove a worker.
func (r *LiveRunner) Teardown(ctx context.Context, workerName string) error {
	cmd := exec.CommandContext(ctx, "wrangler", "delete", "--name="+workerName, "--force")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("wrangler delete failed: %w\nOutput: %s", err, output)
	}
	return nil
}

// PreviewTeardown is intentionally a no-op.
// Preview aliases are ephemeral and don't need explicit cleanup.
// The durable Worker script must not be deleted.
func (r *LiveRunner) PreviewTeardown(ctx context.Context, workerName, alias string) error {
	// ponytail: no-op by design — preview aliases are ephemeral, cleanup would delete the durable Worker
	return nil
}
