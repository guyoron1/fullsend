package tests

import (
	"bytes"
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

/*
Agent Slug Discovery — Warning and Deprecation Behavior Tests

STP Reference: outputs/stp/GH-49/GH-49_test_plan.md
STD Reference: outputs/std/GH-49/GH-49_test_description.yaml
Jira: GH-49
*/

var _ = Describe("[GH-49] Agent Slug Discovery Warnings", func() {

	Context("Deprecation warning for legacy path usage", Ordered, func() {
		var (
			ctx           context.Context
			mockForge     *MockForgeClient
			printerOutput *bytes.Buffer
			err           error
		)

		// TS-GH-49-006: Verify deprecation warning when config.yaml is used
		Context("when legacy config.yaml path is used", Ordered, func() {
			BeforeAll(func() {
				ctx = context.Background()
				mockForge = NewMockForgeClient(
					withoutHarnessDir(),
					withConfigAgents([]string{"legacy-agent-1"}),
				)
			})

			It("[test_id:TS-GH-49-006] should log deprecation warning", func() {
				printerOutput = new(bytes.Buffer)
				printer := NewPrinter(printerOutput)
				_, err = DiscoverAgentSlugs(ctx, mockForge, "config-repo", "main", printer)

				Expect(err).NotTo(HaveOccurred())
				Expect(printerOutput.String()).To(ContainSubstring("deprecat"),
					"deprecation warning should be emitted when legacy config.yaml is used")
			})
		})

		// TS-GH-49-007: Verify no deprecation warning when harness succeeds
		Context("when harness discovery succeeds", Ordered, func() {
			BeforeAll(func() {
				ctx = context.Background()
				mockForge = NewMockForgeClient(
					withHarnessFiles(map[string]HarnessWrapperFile{
						"agent.yaml": {Role: "agent-role", Slug: "agent-slug"},
					}),
				)
			})

			It("[test_id:TS-GH-49-007] should not emit deprecation warning", func() {
				printerOutput = new(bytes.Buffer)
				printer := NewPrinter(printerOutput)
				_, err = DiscoverAgentSlugs(ctx, mockForge, "config-repo", "main", printer)

				Expect(err).NotTo(HaveOccurred())
				Expect(printerOutput.String()).NotTo(ContainSubstring("deprecat"),
					"no deprecation warning should appear when harness discovery succeeds")
			})
		})
	})

	Context("Incomplete harness entry handling", Ordered, func() {
		var (
			ctx           context.Context
			mockForge     *MockForgeClient
			printerOutput *bytes.Buffer
			agents        []AgentInfo
			err           error
		)

		// TS-GH-49-008: Verify entry with role but no slug is skipped with warning
		Context("when harness entry has role but no slug", Ordered, func() {
			BeforeAll(func() {
				ctx = context.Background()
				mockForge = NewMockForgeClient(
					withHarnessFiles(map[string]HarnessWrapperFile{
						"incomplete.yaml": {Role: "agent-role-incomplete", Slug: ""},
						"valid.yaml":      {Role: "valid-role", Slug: "valid-slug"},
					}),
				)
			})

			It("[test_id:TS-GH-49-008] should skip entry and log warning", func() {
				printerOutput = new(bytes.Buffer)
				printer := NewPrinter(printerOutput)
				agents, err = DiscoverAgentSlugs(ctx, mockForge, "config-repo", "main", printer)

				Expect(err).NotTo(HaveOccurred())

				// Verify incomplete entry is not in results
				for _, a := range agents {
					Expect(a.Role).NotTo(Equal("agent-role-incomplete"),
						"entry with role but no slug should be excluded from results")
				}

				// Verify warning was logged about missing slug
				Expect(printerOutput.String()).To(ContainSubstring("no slug"),
					"warning should mention missing slug for incomplete entry")
			})
		})

		// TS-GH-49-009: Verify entry with empty role and slug is silently skipped
		Context("when harness entry has empty role and empty slug", Ordered, func() {
			BeforeAll(func() {
				ctx = context.Background()
				mockForge = NewMockForgeClient(
					withHarnessFiles(map[string]HarnessWrapperFile{
						"empty.yaml": {Role: "", Slug: ""},
					}),
					withEmptyConfig(),
				)
			})

			It("[test_id:TS-GH-49-009] should silently skip entry", func() {
				printerOutput = new(bytes.Buffer)
				printer := NewPrinter(printerOutput)
				_, err = DiscoverAgentSlugs(ctx, mockForge, "config-repo", "main", printer)

				Expect(err).NotTo(HaveOccurred())
				// The empty entry produces no agents and no harness discovery succeeds,
				// so it falls back to config.yaml. But config is empty, so we get nil.
				// The key assertion: no warning output for the empty entry itself.
				// Check that no warning about role/slug was emitted for the empty entry.
				output := printerOutput.String()
				Expect(output).NotTo(ContainSubstring("empty.yaml"),
					"no warning should be produced for entry with empty role and empty slug")
			})
		})
	})
})
