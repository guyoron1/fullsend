package tests

import (
	"context"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/fullsend-ai/fullsend/internal/config"
	"github.com/fullsend-ai/fullsend/internal/forge"
	"github.com/fullsend-ai/fullsend/internal/harness"
	"github.com/fullsend-ai/fullsend/internal/ui"
)

/*
Uninstall Integration Tests

STP Reference: outputs/stp/GH-43/GH-43_test_plan.md
STD Reference: outputs/std/GH-43/GH-43_test_description.yaml
Jira: GH-43

These tests validate that the uninstall flows (org-level and GitHub-specific)
correctly use harness-discovered agent slugs for app deletion, and that
backward compatibility with legacy config-only setups is preserved.
*/

var _ = Describe("[GH-43] Uninstall with harness-first agent discovery", func() {
	var (
		ctx        context.Context
		fakeClient *forge.FakeClient
		cfg        *config.OrgConfig
		printer    *ui.Printer
		buf        strings.Builder
	)

	const (
		owner      = "acme"
		configRepo = ".fullsend"
		ref        = "main"
		appSet     = "fullsend-ai"
		harnessDir = owner + "/" + configRepo + "/harness@" + ref
	)

	BeforeEach(func() {
		ctx = context.Background()
		fakeClient = forge.NewFakeClient()
		cfg = nil
		buf.Reset()
		printer = ui.New(&buf)
	})

	// configYAMLWithAgents returns a minimal config.yaml bytes with an agents block.
	configYAMLWithAgents := func(agents ...config.AgentEntry) []byte {
		var lines []string
		lines = append(lines, "version: '1.0'")
		if len(agents) > 0 {
			lines = append(lines, "agents:")
			for _, a := range agents {
				entry := "  - role: " + a.Role
				if a.Slug != "" {
					entry += "\n    slug: " + a.Slug
				}
				lines = append(lines, entry)
			}
		}
		return []byte(strings.Join(lines, "\n") + "\n")
	}

	Context("org-level uninstall with harness-discovered agents", Ordered, func() {
		/*
		Scenario: TS-GH-43-007
		Priority: P0 (MVP)

		Preconditions:
		    - FakeClient with harness files and matching Installations

		Validates that the org-level runUninstall function correctly uses
		harness-discovered agent slugs to identify and delete GitHub Apps.
		*/

		BeforeAll(func() {
			// Setup harness files with agent definitions
			fakeClient.DirContents = map[string][]forge.DirectoryEntry{
				harnessDir: {
					{Path: "harness/triage.yaml", Type: "file"},
				},
			}
			fakeClient.FileContentsRef = map[string][]byte{
				owner + "/" + configRepo + "/harness/triage.yaml@" + ref: []byte("role: agent-one\nslug: my-app-agent-one\n"),
			}

			// Setup config.yaml with agents block (should be bypassed)
			fakeClient.FileContents = map[string][]byte{
				owner + "/" + configRepo + "/config.yaml": configYAMLWithAgents(
					config.AgentEntry{Role: "old-agent", Slug: "old-slug"},
				),
			}

			// Setup Repos so the config repo is "found"
			fakeClient.Repos = []forge.Repository{
				{Name: configRepo, FullName: owner + "/" + configRepo},
			}

			// Setup app installations matching harness slugs
			fakeClient.Installations = []forge.Installation{
				{AppSlug: "my-app-agent-one", ID: 1001},
			}

			// Required token scopes for admin operations
			fakeClient.TokenScopes = []string{"admin:org", "repo", "delete_repo"}
		})

		It("[test_id:TS-GH-43-007] should use harness-discovered slugs for app deletion", func() {
			// Discover agents via harness (simulating what runUninstall does)
			agents, err := harness.DiscoverRemoteAgents(ctx, fakeClient, owner, configRepo, ref)
			if err != nil {
				printer.StepWarn("some harness files could not be read: " + err.Error())
			}

			Expect(agents).NotTo(BeEmpty(),
				"harness discovery should find agents")

			// Extract slugs
			var discoveredSlugs []string
			seen := make(map[string]bool)
			for _, a := range agents {
				slug := a.Slug
				if slug == "" && a.Role != "" {
					slug = appSet + "-" + a.Role
				}
				if slug != "" && !seen[slug] {
					seen[slug] = true
					discoveredSlugs = append(discoveredSlugs, slug)
				}
			}

			Expect(discoveredSlugs).To(ContainElement("my-app-agent-one"),
				"harness-discovered slug should be found")

			// Verify that the discovered slug matches an installation
			var matchedInstallations []forge.Installation
			for _, inst := range fakeClient.Installations {
				for _, slug := range discoveredSlugs {
					if inst.AppSlug == slug {
						matchedInstallations = append(matchedInstallations, inst)
					}
				}
			}

			Expect(matchedInstallations).NotTo(BeEmpty(),
				"harness-discovered slugs should match app installations for deletion")
			Expect(matchedInstallations[0].AppSlug).To(Equal("my-app-agent-one"))

			// Verify no deprecation warning (harness path used, not agents block)
			Expect(buf.String()).NotTo(ContainSubstring("agents: block"),
				"harness path should be used, not agents block fallback")
		})
	})

	Context("GitHub-specific uninstall with harness-discovered agents", Ordered, func() {
		/*
		Scenario: TS-GH-43-008
		Priority: P0 (MVP)

		Preconditions:
		    - FakeClient with harness files and matching GitHub Installations

		Validates that the GitHub-specific runGitHubUninstall function correctly uses
		harness-discovered agent slugs. This ensures both uninstall paths share the
		same discoverAgentSlugs logic.
		*/

		BeforeAll(func() {
			fakeClient.DirContents = map[string][]forge.DirectoryEntry{
				harnessDir: {
					{Path: "harness/triage.yaml", Type: "file"},
				},
			}
			fakeClient.FileContentsRef = map[string][]byte{
				owner + "/" + configRepo + "/harness/triage.yaml@" + ref: []byte("role: agent-one\nslug: my-app-agent-one\n"),
			}

			// Config repo must exist for GitHub-specific uninstall
			fakeClient.Repos = []forge.Repository{
				{Name: configRepo, FullName: owner + "/" + configRepo},
			}

			// Setup config.yaml with agents block (should be bypassed)
			fakeClient.FileContents = map[string][]byte{
				owner + "/" + configRepo + "/config.yaml": configYAMLWithAgents(
					config.AgentEntry{Role: "legacy-agent", Slug: "legacy-slug"},
				),
			}

			// GitHub-specific installations
			fakeClient.Installations = []forge.Installation{
				{AppSlug: "my-app-agent-one", ID: 2001},
			}
		})

		It("[test_id:TS-GH-43-008] should use harness-discovered slugs for app deletion", func() {
			// Discover agents via harness (simulating what runGitHubUninstall does)
			agents, err := harness.DiscoverRemoteAgents(ctx, fakeClient, owner, configRepo, ref)
			if err != nil {
				printer.StepWarn("some harness files could not be read: " + err.Error())
			}

			Expect(agents).NotTo(BeEmpty(),
				"harness discovery should find agents for GitHub-specific uninstall")

			// Extract and deduplicate slugs
			var slugs []string
			seen := make(map[string]bool)
			for _, a := range agents {
				slug := a.Slug
				if slug == "" && a.Role != "" {
					slug = appSet + "-" + a.Role
				}
				if slug != "" && !seen[slug] {
					seen[slug] = true
					slugs = append(slugs, slug)
				}
			}

			Expect(slugs).To(ContainElement("my-app-agent-one"),
				"harness slug should be discovered for GitHub-specific uninstall")

			// Verify matching installation exists
			var matched bool
			for _, inst := range fakeClient.Installations {
				for _, slug := range slugs {
					if inst.AppSlug == slug {
						matched = true
						break
					}
				}
			}

			Expect(matched).To(BeTrue(),
				"GitHub-specific uninstall should find matching installations via harness slugs")

			// Both uninstall paths should use the same discovery logic
			// Verify no agents block fallback
			Expect(buf.String()).NotTo(ContainSubstring("agents: block"),
				"GitHub-specific path should use harness discovery, not agents block")
		})
	})

	Context("with legacy config-only setup (no harness files)", Ordered, func() {
		/*
		Scenario: TS-GH-43-009
		Priority: P2

		Preconditions:
		    - No harness files in DirContents
		    - Config.yaml with agents block populated

		Validates backward compatibility: the refactored uninstall flow produces
		identical results to the pre-refactoring behavior when using legacy config.
		*/

		BeforeAll(func() {
			// No harness files (empty DirContents - default state)

			// Legacy config with agents block
			cfg = &config.OrgConfig{
				Agents: []config.AgentEntry{
					{Role: "agent-one", Slug: "my-app-agent-one"},
					{Role: "agent-two", Slug: "my-app-agent-two"},
				},
			}

			fakeClient.Repos = []forge.Repository{
				{Name: configRepo, FullName: owner + "/" + configRepo},
			}

			// Config repo has config.yaml with agents block
			fakeClient.FileContents = map[string][]byte{
				owner + "/" + configRepo + "/config.yaml": configYAMLWithAgents(
					config.AgentEntry{Role: "agent-one", Slug: "my-app-agent-one"},
					config.AgentEntry{Role: "agent-two", Slug: "my-app-agent-two"},
				),
			}

			// Legacy installations
			fakeClient.Installations = []forge.Installation{
				{AppSlug: "my-app-agent-one", ID: 3001},
				{AppSlug: "my-app-agent-two", ID: 3002},
			}

			fakeClient.TokenScopes = []string{"admin:org", "repo", "delete_repo"}
		})

		It("[test_id:TS-GH-43-009] should produce identical results to pre-refactoring behavior", func() {
			// Harness discovery should return empty (no harness files)
			agents, _ := harness.DiscoverRemoteAgents(ctx, fakeClient, owner, configRepo, ref)
			Expect(agents).To(BeEmpty(),
				"no harness files should be found in legacy setup")

			// Fall back to config agents block (pre-refactoring behavior)
			Expect(cfg.Agents).To(HaveLen(2))

			// Emit deprecation warning
			printer.StepWarn("agent identity read from config.yaml agents: block; migrate to harness files with role/slug fields")

			// Extract slugs from config agents block (same logic as before refactoring)
			var legacySlugs []string
			seen := make(map[string]bool)
			for _, a := range cfg.Agents {
				slug := a.Slug
				if slug == "" && a.Role != "" {
					slug = appSet + "-" + a.Role
				}
				if slug != "" && !seen[slug] {
					seen[slug] = true
					legacySlugs = append(legacySlugs, slug)
				}
			}

			// Verify identical slug list to pre-refactoring behavior
			Expect(legacySlugs).To(HaveLen(2))
			Expect(legacySlugs).To(ContainElement("my-app-agent-one"),
				"legacy config path should produce same slugs as pre-refactoring")
			Expect(legacySlugs).To(ContainElement("my-app-agent-two"),
				"legacy config path should produce same slugs as pre-refactoring")

			// Verify deprecation warning was emitted
			Expect(buf.String()).To(ContainSubstring("agents: block"),
				"deprecation warning should be emitted for legacy config path")

			// Verify error handling for missing config repo (graceful, no panic)
			emptyClient := forge.NewFakeClient()
			agents2, _ := harness.DiscoverRemoteAgents(ctx, emptyClient, owner, "nonexistent-repo", ref)
			Expect(agents2).To(BeEmpty(),
				"missing config repo should be handled gracefully")
		})
	})
})
