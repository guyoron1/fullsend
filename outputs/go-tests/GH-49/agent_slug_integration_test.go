package tests

import (
	"bytes"
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

/*
Agent Slug Discovery — Install Setup Integration Tests

STP Reference: outputs/stp/GH-49/GH-49_test_plan.md
STD Reference: outputs/std/GH-49/GH-49_test_description.yaml
Jira: GH-49
*/

var _ = Describe("[GH-49] Agent Slug Discovery Integration", func() {

	Context("Install setup integration with harness-discovered agents", Ordered, func() {
		var (
			ctx           context.Context
			mockForge     *MockForgeClient
			printerOutput *bytes.Buffer
		)

		// TS-GH-49-016: Verify install setup uses harness-discovered slugs
		Context("when install setup uses harness-discovered agents", Ordered, func() {
			BeforeAll(func() {
				ctx = context.Background()
				mockForge = NewMockForgeClient(
					withHarnessFiles(map[string]HarnessWrapperFile{
						"app-agent.yaml":     {Role: "app-role", Slug: "app-slug"},
						"infra-agent.yaml":   {Role: "infra-role", Slug: "infra-slug"},
					}),
				)
			})

			It("[test_id:TS-GH-49-016] should initiate app configuration with harness agent slugs", func() {
				printerOutput = new(bytes.Buffer)
				printer := NewPrinter(printerOutput)

				appConfigs, err := InstallSetup(ctx, mockForge, "config-repo", "main", printer)

				Expect(err).NotTo(HaveOccurred())
				Expect(appConfigs).NotTo(BeEmpty(),
					"install setup should return agent configurations")

				// Verify harness-discovered slugs are used
				slugs := make(map[string]bool)
				for _, a := range appConfigs {
					slugs[a.Slug] = true
				}
				Expect(slugs).To(HaveKey("app-slug"),
					"app-slug from harness should be in app configs")
				Expect(slugs).To(HaveKey("infra-slug"),
					"infra-slug from harness should be in app configs")
			})
		})

		// TS-GH-49-017: Verify agent filtering by app-set
		Context("when filtering harness-discovered agents by app-set", Ordered, func() {
			BeforeAll(func() {
				ctx = context.Background()
				mockForge = NewMockForgeClient(
					withHarnessFiles(map[string]HarnessWrapperFile{
						"set-a-agent.yaml": {Role: "agent-in-set-a", Slug: "slug-set-a"},
						"set-b-agent.yaml": {Role: "agent-in-set-b", Slug: "slug-set-b"},
						"set-a-other.yaml": {Role: "other-in-set-a", Slug: "other-slug-set-a"},
					}),
				)
			})

			It("[test_id:TS-GH-49-017] should correctly filter agents by app-set membership", func() {
				printerOutput = new(bytes.Buffer)
				printer := NewPrinter(printerOutput)

				// First discover all agents
				allAgents, err := DiscoverAgentSlugs(ctx, mockForge, "config-repo", "main", printer)
				Expect(err).NotTo(HaveOccurred())
				Expect(allAgents).To(HaveLen(3))

				// Filter by app-set "set-a"
				filteredAgents := FilterAgentsByAppSet(allAgents, "set-a")

				Expect(filteredAgents).To(HaveLen(2),
					"only agents matching app-set 'set-a' should be returned")

				// Verify set-a agents present
				filteredSlugs := make(map[string]bool)
				for _, a := range filteredAgents {
					filteredSlugs[a.Slug] = true
				}
				Expect(filteredSlugs).To(HaveKey("slug-set-a"))
				Expect(filteredSlugs).To(HaveKey("other-slug-set-a"))

				// Verify set-b agent excluded
				Expect(filteredSlugs).NotTo(HaveKey("slug-set-b"),
					"agents from other app-sets should be excluded")
			})
		})
	})
})
