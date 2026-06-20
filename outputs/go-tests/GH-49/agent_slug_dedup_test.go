package tests

import (
	"bytes"
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

/*
Agent Slug Discovery — Duplicate Role Handling Tests

STP Reference: outputs/stp/GH-49/GH-49_test_plan.md
STD Reference: outputs/std/GH-49/GH-49_test_description.yaml
Jira: GH-49
*/

var _ = Describe("[GH-49] Agent Slug Discovery Deduplication", func() {

	Context("Duplicate role handling", Ordered, func() {
		var (
			ctx           context.Context
			mockForge     *MockForgeClient
			printerOutput *bytes.Buffer
			agents        []AgentInfo
			err           error
		)

		// TS-GH-49-010: Verify duplicate roles keep first occurrence
		Context("when harness files contain duplicate roles", Ordered, func() {
			BeforeAll(func() {
				ctx = context.Background()
				mockForge = NewMockForgeClient(
					withHarnessFiles(map[string]HarnessWrapperFile{
						"dup-a.yaml": {Role: "shared-role", Slug: "slug-first"},
						"dup-b.yaml": {Role: "shared-role", Slug: "slug-second"},
						"unique.yaml": {Role: "unique-role", Slug: "unique-slug"},
					}),
				)
			})

			It("[test_id:TS-GH-49-010] should keep first occurrence sorted by Role then Filename", func() {
				printerOutput = new(bytes.Buffer)
				printer := NewPrinter(printerOutput)
				agents, err = DiscoverAgentSlugs(ctx, mockForge, "config-repo", "main", printer)

				Expect(err).NotTo(HaveOccurred())

				// Count agents with the shared role — should be exactly 1
				sharedRoleCount := 0
				var retainedSlug string
				for _, a := range agents {
					if a.Role == "shared-role" {
						sharedRoleCount++
						retainedSlug = a.Slug
					}
				}
				Expect(sharedRoleCount).To(Equal(1),
					"only one agent per duplicate role should be retained")

				// First occurrence by filename sort: dup-a.yaml < dup-b.yaml
				Expect(retainedSlug).To(Equal("slug-first"),
					"first occurrence by Role+Filename sort order should be retained")

				// Unique role should still be present
				hasUnique := false
				for _, a := range agents {
					if a.Role == "unique-role" {
						hasUnique = true
					}
				}
				Expect(hasUnique).To(BeTrue(), "non-duplicate roles should be preserved")
			})
		})

		// TS-GH-49-011: Verify info message logged for duplicate role
		Context("when duplicate roles are detected", Ordered, func() {
			BeforeAll(func() {
				ctx = context.Background()
				mockForge = NewMockForgeClient(
					withHarnessFiles(map[string]HarnessWrapperFile{
						"first.yaml":  {Role: "dup-role", Slug: "slug-1"},
						"second.yaml": {Role: "dup-role", Slug: "slug-2"},
					}),
				)
			})

			It("[test_id:TS-GH-49-011] should log info message about duplicate", func() {
				printerOutput = new(bytes.Buffer)
				printer := NewPrinter(printerOutput)
				_, err = DiscoverAgentSlugs(ctx, mockForge, "config-repo", "main", printer)

				Expect(err).NotTo(HaveOccurred())

				output := printerOutput.String()
				Expect(output).To(SatisfyAny(
					ContainSubstring("duplicate"),
					ContainSubstring("already"),
				), "info message should be logged when duplicate role is detected")
			})
		})
	})
})
