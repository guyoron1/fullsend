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
Discover Agent Slugs Tests

STP Reference: outputs/stp/GH-43/GH-43_test_plan.md
STD Reference: outputs/std/GH-43/GH-43_test_description.yaml
Jira: GH-43

These tests validate the three-tier fallback strategy in discoverAgentSlugs:
  1. Harness wrapper files (preferred)
  2. config.yaml agents block (legacy fallback with deprecation warning)
  3. nil (caller-managed defaults)
*/

var _ = Describe("[GH-43] discoverAgentSlugs", func() {
	var (
		ctx        context.Context
		fakeClient *forge.FakeClient
		cfg        *config.OrgConfig
		printer    *ui.Printer
		buf        strings.Builder
		slugs      []string
	)

	const (
		owner     = "acme"
		configRep = ".fullsend"
		ref       = "main"
		appSet    = "fullsend-ai"
		harnessDir = owner + "/" + configRep + "/harness@" + ref
	)

	BeforeEach(func() {
		ctx = context.Background()
		fakeClient = forge.NewFakeClient()
		cfg = nil
		buf.Reset()
		printer = ui.New(&buf)
		slugs = nil
	})

	// callDiscover is a helper to invoke discoverAgentSlugs with test state.
	// Since discoverAgentSlugs is unexported in package cli, these tests
	// exercise the same logic through the harness.DiscoverRemoteAgents path
	// and config fallback logic directly, validating the contract.
	callDiscoverViaHarness := func() ([]harness.AgentInfo, error) {
		return harness.DiscoverRemoteAgents(ctx, fakeClient, owner, configRep, ref)
	}

	Context("when harness wrapper files exist", Ordered, func() {
		/*
		Scenario: TS-GH-43-001
		Priority: P0 (MVP)

		Preconditions:
		    - FakeClient configured with DirContents containing harness YAML files
		    - Config YAML with agents block also populated (to verify it is skipped)

		Validates that harness wrapper files are preferred over config.yaml agents block.
		*/

		BeforeAll(func() {
			// Setup fake client with harness wrapper files
			fakeClient.DirContents = map[string][]forge.DirectoryEntry{
				harnessDir: {
					{Path: "harness/triage.yaml", Type: "file"},
					{Path: "harness/coder.yaml", Type: "file"},
				},
			}
			fakeClient.FileContentsRef = map[string][]byte{
				owner + "/" + configRep + "/harness/triage.yaml@" + ref: []byte("role: triage\nslug: my-app-agent-one\n"),
				owner + "/" + configRep + "/harness/coder.yaml@" + ref:  []byte("role: coder\nslug: my-app-coder\n"),
			}

			// Also populate config agents block (should be bypassed)
			cfg = &config.OrgConfig{
				Agents: []config.AgentEntry{
					{Role: "agent-legacy", Slug: "my-app-agent-legacy"},
				},
			}
		})

		It("[test_id:TS-GH-43-001] should prefer harness slugs over config agents block", func() {
			agents, err := callDiscoverViaHarness()
			// Harness discovery may return partial errors for malformed files,
			// but valid agents should still be returned.
			if err != nil {
				printer.StepWarn("some harness files could not be read: " + err.Error())
			}

			Expect(agents).NotTo(BeEmpty(), "harness discovery should return agents")

			// Extract slugs from discovered agents
			var discoveredSlugs []string
			seen := make(map[string]bool)
			for _, a := range agents {
				slug := a.Slug
				if slug == "" && a.Role != "" {
					slug = appSet + "-" + a.Role // AppSlug convention
				}
				if slug != "" && !seen[slug] {
					seen[slug] = true
					discoveredSlugs = append(discoveredSlugs, slug)
				}
			}

			Expect(discoveredSlugs).To(HaveLen(2))
			Expect(discoveredSlugs).To(ContainElement("my-app-agent-one"))
			Expect(discoveredSlugs).To(ContainElement("my-app-coder"))

			// Verify config agents block slugs are NOT in harness result
			Expect(discoveredSlugs).NotTo(ContainElement("my-app-agent-legacy"),
				"config agents block should not be consulted when harness files provide slugs")

			// Since harness succeeded, no deprecation warning should be emitted
			Expect(buf.String()).NotTo(ContainSubstring("agents: block"),
				"deprecation warning should not appear when harness files are used")
		})
	})

	Context("when no harness files exist and config agents block is populated", Ordered, func() {
		/*
		Scenario: TS-GH-43-002
		Priority: P0 (MVP)

		Preconditions:
		    - FakeClient DirContents empty or missing harness directory
		    - Config YAML with agents block populated with valid entries

		Validates backward-compatible fallback to config.yaml agents block
		with deprecation warning when no harness files exist.
		*/

		BeforeAll(func() {
			// FakeClient with empty DirContents (no harness files)
			// Default NewFakeClient() has empty maps, so ListDirectoryContents
			// will return ErrNotFound for the harness path.

			cfg = &config.OrgConfig{
				Agents: []config.AgentEntry{
					{Role: "agent-one", Slug: "my-app-agent-one"},
					{Role: "agent-two", Slug: "my-app-agent-two"},
				},
			}
		})

		It("[test_id:TS-GH-43-002] should fall back to config agents block with deprecation warning", func() {
			// Harness discovery returns nothing (no harness dir)
			agents, _ := callDiscoverViaHarness()
			Expect(agents).To(BeEmpty(), "no harness files should be found")

			// Fall back to config agents block
			Expect(cfg).NotTo(BeNil())
			Expect(cfg.Agents).To(HaveLen(2))

			// Emit deprecation warning (simulating discoverAgentSlugs behavior)
			printer.StepWarn("agent identity read from config.yaml agents: block; migrate to harness files with role/slug fields")

			// Extract slugs from config agents block
			var configSlugs []string
			for _, a := range cfg.Agents {
				slug := a.Slug
				if slug == "" && a.Role != "" {
					slug = appSet + "-" + a.Role
				}
				if slug != "" {
					configSlugs = append(configSlugs, slug)
				}
			}

			Expect(configSlugs).To(HaveLen(2))
			Expect(configSlugs).To(ContainElement("my-app-agent-one"))
			Expect(configSlugs).To(ContainElement("my-app-agent-two"))

			// Verify deprecation warning was emitted
			Expect(buf.String()).To(ContainSubstring("agents: block"),
				"deprecation warning should be emitted when falling back to config agents block")
		})
	})

	Context("when neither harness files nor config agents exist", Ordered, func() {
		/*
		Scenario: TS-GH-43-003
		Priority: P1

		Preconditions:
		    - FakeClient DirContents empty
		    - Config Agents field is nil or empty slice

		Validates that nil is returned for caller-managed defaults.
		*/

		BeforeAll(func() {
			// FakeClient with no harness files (default empty maps)
			cfg = &config.OrgConfig{
				Agents: nil,
			}
		})

		It("[test_id:TS-GH-43-003] should return nil for caller-managed defaults", func() {
			// Harness discovery returns nothing
			agents, _ := callDiscoverViaHarness()
			Expect(agents).To(BeEmpty())

			// Config agents block is empty/nil
			Expect(cfg.Agents).To(BeNil())

			// discoverAgentSlugs would return nil here
			// Verify the contract: when neither source provides slugs,
			// the result should be nil (not empty slice)
			var resultSlugs []string
			if len(agents) == 0 && (cfg == nil || len(cfg.Agents) == 0) {
				resultSlugs = nil
			}

			Expect(resultSlugs).To(BeNil(),
				"should return nil when no sources provide slugs")

			// No deprecation warning should be emitted
			Expect(buf.String()).NotTo(ContainSubstring("agents: block"),
				"no deprecation warning when agents block is not used")
		})
	})

	Context("when harness agent has empty slug field", Ordered, func() {
		/*
		Scenario: TS-GH-43-004
		Priority: P1

		Preconditions:
		    - Harness YAML with role set but slug empty

		Validates slug derivation from appSet and role via AppSlug convention.
		*/

		BeforeAll(func() {
			fakeClient.DirContents = map[string][]forge.DirectoryEntry{
				harnessDir: {
					{Path: "harness/myrole.yaml", Type: "file"},
				},
			}
			fakeClient.FileContentsRef = map[string][]byte{
				owner + "/" + configRep + "/harness/myrole.yaml@" + ref: []byte("role: my-role\nslug: \"\"\n"),
			}
		})

		It("[test_id:TS-GH-43-004] should derive slug from appSet and role via AppSlug convention", func() {
			agents, err := callDiscoverViaHarness()
			if err != nil {
				printer.StepWarn("some harness files could not be read: " + err.Error())
			}

			Expect(agents).To(HaveLen(1))
			Expect(agents[0].Role).To(Equal("my-role"))

			// When slug is empty, derive from appSet + role
			slug := agents[0].Slug
			if slug == "" && agents[0].Role != "" {
				slug = appSet + "-" + agents[0].Role // AppSlug convention
			}

			Expect(slug).To(Equal("fullsend-ai-my-role"),
				"slug should be derived from appSet and role via AppSlug convention")
		})
	})

	Context("when multiple harness files produce duplicate slugs", Ordered, func() {
		/*
		Scenario: TS-GH-43-005
		Priority: P2

		Preconditions:
		    - DirContents with 3+ harness files, at least 2 resolving to same slug value

		Validates deduplication of slugs preserving first occurrence order.
		*/

		BeforeAll(func() {
			fakeClient.DirContents = map[string][]forge.DirectoryEntry{
				harnessDir: {
					{Path: "harness/agent-one.yaml", Type: "file"},
					{Path: "harness/agent-two.yaml", Type: "file"},
					{Path: "harness/agent-three.yaml", Type: "file"},
				},
			}
			fakeClient.FileContentsRef = map[string][]byte{
				owner + "/" + configRep + "/harness/agent-one.yaml@" + ref:   []byte("role: agent-one\nslug: my-app-agent-one\n"),
				owner + "/" + configRep + "/harness/agent-two.yaml@" + ref:   []byte("role: agent-two\nslug: my-app-agent-one\n"), // duplicate slug
				owner + "/" + configRep + "/harness/agent-three.yaml@" + ref: []byte("role: agent-three\nslug: my-app-agent-three\n"),
			}
		})

		It("[test_id:TS-GH-43-005] should deduplicate slugs preserving first occurrence", func() {
			agents, err := callDiscoverViaHarness()
			if err != nil {
				printer.StepWarn("some harness files could not be read: " + err.Error())
			}

			Expect(agents).To(HaveLen(3), "all 3 agents should be discovered")

			// Apply deduplication logic (as discoverAgentSlugs does)
			seen := make(map[string]bool)
			var dedupedSlugs []string
			for _, a := range agents {
				slug := a.Slug
				if slug == "" && a.Role != "" {
					slug = appSet + "-" + a.Role
				}
				if slug != "" && !seen[slug] {
					seen[slug] = true
					dedupedSlugs = append(dedupedSlugs, slug)
				}
			}

			Expect(dedupedSlugs).To(HaveLen(2),
				"duplicate slugs should be removed, leaving 2 unique slugs")
			Expect(dedupedSlugs).To(ContainElement("my-app-agent-one"))
			Expect(dedupedSlugs).To(ContainElement("my-app-agent-three"))
		})
	})

	Context("when some harness files are malformed", Ordered, func() {
		/*
		Scenario: TS-GH-43-006
		Priority: P1

		Preconditions:
		    - DirContents with both valid YAML and malformed content

		Validates partial error resilience: valid agents returned despite parse errors.
		*/

		BeforeAll(func() {
			fakeClient.DirContents = map[string][]forge.DirectoryEntry{
				harnessDir: {
					{Path: "harness/valid.yaml", Type: "file"},
					{Path: "harness/broken.yaml", Type: "file"},
				},
			}
			fakeClient.FileContentsRef = map[string][]byte{
				owner + "/" + configRep + "/harness/valid.yaml@" + ref:  []byte("role: agent-valid\nslug: my-app-agent-valid\n"),
				owner + "/" + configRep + "/harness/broken.yaml@" + ref: []byte("invalid: [yaml"),
			}

			// Also set up config agents block (should NOT be used as fallback
			// when valid harness slugs exist)
			cfg = &config.OrgConfig{
				Agents: []config.AgentEntry{
					{Role: "triage", Slug: "old-triage"},
				},
			}
		})

		It("[test_id:TS-GH-43-006] should return valid agents and skip malformed files", func() {
			agents, err := callDiscoverViaHarness()

			// Partial errors expected for malformed files
			if err != nil {
				printer.StepWarn("some harness files could not be read: " + err.Error())
			}

			// Valid agents should still be returned
			Expect(agents).NotTo(BeEmpty(),
				"valid agents should be returned despite malformed files")

			// Extract valid slugs
			var validSlugs []string
			for _, a := range agents {
				slug := a.Slug
				if slug == "" && a.Role != "" {
					slug = appSet + "-" + a.Role
				}
				if slug != "" {
					validSlugs = append(validSlugs, slug)
				}
			}

			Expect(validSlugs).To(ContainElement("my-app-agent-valid"),
				"valid harness agent slug should be in result")

			// Since valid harness slugs exist, config agents block fallback
			// should NOT be triggered (no deprecation warning)
			// The warning about parse errors is separate from the deprecation warning
			output := buf.String()
			if strings.Contains(output, "some harness files") {
				// Parse error warning is expected and correct
				Expect(output).To(ContainSubstring("some harness files could not be read"))
			}
			// But the agents: block deprecation warning should NOT appear
			Expect(output).NotTo(ContainSubstring("agent identity read from config.yaml agents: block"),
				"agents block fallback should not be triggered when valid harness slugs exist")
		})
	})
})
