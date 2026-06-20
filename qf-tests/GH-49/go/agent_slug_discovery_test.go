package tests

import (
	"bytes"
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

/*
Agent Slug Discovery — Harness-First Preference Tests

STP Reference: outputs/stp/GH-49/GH-49_test_plan.md
STD Reference: outputs/std/GH-49/GH-49_test_description.yaml
Jira: GH-49
*/

var _ = Describe("[GH-49] Agent Slug Discovery", func() {

	Context("Harness-first agent discovery preference", Ordered, func() {
		var (
			ctx       context.Context
			mockForge *MockForgeClient
			agents    []AgentInfo
			err       error
		)

		// TS-GH-49-001: Verify harness files with valid role+slug are preferred over config.yaml
		Context("when harness files have valid role and slug fields", Ordered, func() {
			BeforeAll(func() {
				ctx = context.Background()
				mockForge = NewMockForgeClient(
					withHarnessFiles(map[string]HarnessWrapperFile{
						"agent-a.yaml": {Role: "agent-role-a", Slug: "agent-slug-a"},
						"agent-b.yaml": {Role: "agent-role-b", Slug: "agent-slug-b"},
					}),
					withConfigAgents([]string{"legacy-agent-1", "legacy-agent-2"}),
				)
			})

			It("[test_id:TS-GH-49-001] should prefer harness-discovered agents over config.yaml", func() {
				printer := NewPrinter(new(bytes.Buffer))
				agents, err = DiscoverAgentSlugs(ctx, mockForge, "config-repo", "main", printer)

				Expect(err).NotTo(HaveOccurred())
				Expect(agents).To(HaveLen(2))
				Expect(agents[0].Role).To(Equal("agent-role-a"))
				Expect(agents[0].Slug).To(Equal("agent-slug-a"))
				Expect(agents[1].Role).To(Equal("agent-role-b"))
				Expect(agents[1].Slug).To(Equal("agent-slug-b"))

				// Verify no legacy agents in results
				for _, a := range agents {
					Expect(a.Slug).NotTo(Equal("legacy-agent-1"))
					Expect(a.Slug).NotTo(Equal("legacy-agent-2"))
				}
			})
		})

		// TS-GH-49-002: Verify config.yaml is not consulted when harness succeeds
		Context("when harness discovery succeeds", Ordered, func() {
			BeforeAll(func() {
				ctx = context.Background()
				mockForge = NewMockForgeClient(
					withHarnessFiles(map[string]HarnessWrapperFile{
						"agent.yaml": {Role: "agent-role", Slug: "agent-slug"},
					}),
					withConfigAgents([]string{"legacy-agent"}),
				)
			})

			It("[test_id:TS-GH-49-002] should not consult config.yaml agents block", func() {
				printer := NewPrinter(new(bytes.Buffer))
				_, err = DiscoverAgentSlugs(ctx, mockForge, "config-repo", "main", printer)

				Expect(err).NotTo(HaveOccurred())
				Expect(mockForge.ConfigYAMLAccessed()).To(BeFalse(),
					"config.yaml should not be accessed when harness discovery succeeds")
			})
		})
	})

	Context("Fallback to legacy config.yaml", Ordered, func() {
		var (
			ctx       context.Context
			mockForge *MockForgeClient
			agents    []AgentInfo
			err       error
		)

		// TS-GH-49-003: Verify fallback when no harness directory exists
		Context("when no harness directory exists", Ordered, func() {
			BeforeAll(func() {
				ctx = context.Background()
				mockForge = NewMockForgeClient(
					withoutHarnessDir(),
					withConfigAgents([]string{"legacy-agent-1", "legacy-agent-2"}),
				)
			})

			It("[test_id:TS-GH-49-003] should fall back to config.yaml agents block", func() {
				printer := NewPrinter(new(bytes.Buffer))
				agents, err = DiscoverAgentSlugs(ctx, mockForge, "config-repo", "main", printer)

				Expect(err).NotTo(HaveOccurred())
				Expect(agents).To(HaveLen(2))
				Expect(agents[0].Slug).To(Equal("legacy-agent-1"))
				Expect(agents[1].Slug).To(Equal("legacy-agent-2"))
			})
		})

		// TS-GH-49-004: Verify fallback when harness files have no role/slug
		Context("when harness files contain no role/slug fields", Ordered, func() {
			BeforeAll(func() {
				ctx = context.Background()
				// Harness files exist but have empty role and slug — treated as no valid agents
				mockForge = NewMockForgeClient(
					withHarnessFiles(map[string]HarnessWrapperFile{
						"placeholder.yaml": {Role: "", Slug: ""},
					}),
					withConfigAgents([]string{"legacy-agent-1", "legacy-agent-2"}),
				)
			})

			It("[test_id:TS-GH-49-004] should fall back to config.yaml agents block", func() {
				printer := NewPrinter(new(bytes.Buffer))
				agents, err = DiscoverAgentSlugs(ctx, mockForge, "config-repo", "main", printer)

				Expect(err).NotTo(HaveOccurred())
				Expect(agents).To(HaveLen(2))
				Expect(agents[0].Slug).To(Equal("legacy-agent-1"))
				Expect(agents[1].Slug).To(Equal("legacy-agent-2"))
			})
		})

		// TS-GH-49-005: Verify nil when neither source provides agents
		Context("when neither harness nor config.yaml provides agents", Ordered, func() {
			BeforeAll(func() {
				ctx = context.Background()
				mockForge = NewMockForgeClient(
					withoutHarnessDir(),
					withEmptyConfig(),
				)
			})

			It("[test_id:TS-GH-49-005] should return nil", func() {
				printer := NewPrinter(new(bytes.Buffer))
				agents, err = DiscoverAgentSlugs(ctx, mockForge, "config-repo", "main", printer)

				Expect(err).NotTo(HaveOccurred())
				Expect(agents).To(BeNil())
			})
		})
	})
})
