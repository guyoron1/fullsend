package tests

import (
	"bytes"
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

/*
Agent Slug Discovery — Error Resilience Tests

STP Reference: outputs/stp/GH-49/GH-49_test_plan.md
STD Reference: outputs/std/GH-49/GH-49_test_description.yaml
Jira: GH-49
*/

var _ = Describe("[GH-49] Agent Slug Discovery Resilience", func() {

	Context("Partial read error resilience", Ordered, func() {
		var (
			ctx           context.Context
			mockForge     *MockForgeClient
			printerOutput *bytes.Buffer
			agents        []AgentInfo
			err           error
		)

		// TS-GH-49-012: Verify partial read errors still return valid agents
		Context("when partial read errors occur during harness discovery", Ordered, func() {
			BeforeAll(func() {
				ctx = context.Background()
				mockForge = NewMockForgeClient(
					withHarnessFiles(map[string]HarnessWrapperFile{
						"valid.yaml": {Role: "valid-agent", Slug: "valid-slug"},
						"error.yaml": {Role: "error-agent", Slug: "error-slug"},
					}),
					withFileReadErrors(map[string]error{
						"error.yaml": fmt.Errorf("simulated read failure"),
					}),
				)
			})

			It("[test_id:TS-GH-49-012] should return successfully parsed agents", func() {
				printerOutput = new(bytes.Buffer)
				printer := NewPrinter(printerOutput)
				agents, err = DiscoverAgentSlugs(ctx, mockForge, "config-repo", "main", printer)

				Expect(err).NotTo(HaveOccurred())
				Expect(agents).NotTo(BeEmpty(),
					"valid agents should be returned despite partial errors")

				// Verify the valid agent is present
				hasValid := false
				for _, a := range agents {
					if a.Role == "valid-agent" && a.Slug == "valid-slug" {
						hasValid = true
					}
				}
				Expect(hasValid).To(BeTrue(),
					"successfully parsed agent should be in results")
			})
		})

		// TS-GH-49-013: Verify hard error falls back to config.yaml
		Context("when harness discovery returns a hard error", Ordered, func() {
			BeforeAll(func() {
				ctx = context.Background()
				mockForge = NewMockForgeClient(
					withHarnessError(fmt.Errorf("permission denied: cannot list harness directory")),
					withConfigAgents([]string{"fallback-agent-1", "fallback-agent-2"}),
				)
			})

			It("[test_id:TS-GH-49-013] should fall back to legacy config.yaml", func() {
				printerOutput = new(bytes.Buffer)
				printer := NewPrinter(printerOutput)
				agents, err = DiscoverAgentSlugs(ctx, mockForge, "config-repo", "main", printer)

				Expect(err).NotTo(HaveOccurred())
				Expect(agents).To(HaveLen(2))
				Expect(agents[0].Slug).To(Equal("fallback-agent-1"))
				Expect(agents[1].Slug).To(Equal("fallback-agent-2"))
			})
		})

		// TS-GH-49-014: Verify warning logged for discovery errors
		Context("when harness discovery encounters errors", Ordered, func() {
			BeforeAll(func() {
				ctx = context.Background()
				mockForge = NewMockForgeClient(
					withHarnessError(fmt.Errorf("network timeout")),
					withConfigAgents([]string{"fallback-agent"}),
				)
			})

			It("[test_id:TS-GH-49-014] should log warning about discovery errors", func() {
				printerOutput = new(bytes.Buffer)
				printer := NewPrinter(printerOutput)
				_, err = DiscoverAgentSlugs(ctx, mockForge, "config-repo", "main", printer)

				Expect(err).NotTo(HaveOccurred())
				Expect(printerOutput.String()).To(ContainSubstring("warning"),
					"warning should be logged when harness discovery encounters errors")
			})
		})
	})

	Context("Malformed configuration handling", Ordered, func() {
		var (
			ctx       context.Context
			mockForge *MockForgeClient
			agents    []AgentInfo
			err       error
		)

		// TS-GH-49-015: Verify malformed config.yaml returns nil without panic
		Context("when config.yaml is malformed", Ordered, func() {
			BeforeAll(func() {
				ctx = context.Background()
				mockForge = NewMockForgeClient(
					withoutHarnessDir(),
					withMalformedConfig(),
				)
			})

			It("[test_id:TS-GH-49-015] should return nil without panic", func() {
				printerOutput := new(bytes.Buffer)
				printer := NewPrinter(printerOutput)

				// This must not panic — wrap in a function to catch panics
				Expect(func() {
					agents, err = DiscoverAgentSlugs(ctx, mockForge, "config-repo", "main", printer)
				}).NotTo(Panic(), "function should not panic on malformed config.yaml")

				Expect(agents).To(BeNil(),
					"nil should be returned for agents when config is malformed")
			})
		})
	})
})
