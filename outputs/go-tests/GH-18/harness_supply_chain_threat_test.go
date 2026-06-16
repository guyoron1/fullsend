package harness_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/fullsend-ai/fullsend/internal/harness"
)

// boolPtr is a test helper that returns a pointer to a bool.
func boolPtr(b bool) *bool { return &b }

var _ = Describe("Supply Chain Threat Model — Harness", Ordered, func() {

	// =========================================================================
	// TS-GH-18-006: Model Provider Diversity Support
	// =========================================================================
	Describe("Model Provider Diversity Support", func() {

		Context("when config has multiple providers", Ordered, func() {
			var tmpDir string

			BeforeAll(func() {
				var err error
				tmpDir, err = os.MkdirTemp("", "provider-test-*")
				Expect(err).NotTo(HaveOccurred())

				// Create two provider YAML files
				provider1 := `name: anthropic
type: anthropic
credentials:
  ANTHROPIC_API_KEY: "test-key-1"
config:
  ANTHROPIC_BASE_URL: "https://api.anthropic.com"
`
				provider2 := `name: openai
type: openai
credentials:
  OPENAI_API_KEY: "test-key-2"
config:
  OPENAI_BASE_URL: "https://api.openai.com"
`
				Expect(os.WriteFile(filepath.Join(tmpDir, "anthropic.yaml"), []byte(provider1), 0644)).To(Succeed())
				Expect(os.WriteFile(filepath.Join(tmpDir, "openai.yaml"), []byte(provider2), 0644)).To(Succeed())
			})

			AfterAll(func() {
				os.RemoveAll(tmpDir)
			})

			It("[test_id:TS-GH-18-006a] should load all provider definitions", func() {
				providers, err := harness.LoadProviderDefs(tmpDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(providers).To(HaveLen(2), "Should load both provider definitions")

				// Verify providers are distinct
				names := map[string]bool{}
				for _, p := range providers {
					names[p.Name] = true
				}
				Expect(names).To(HaveLen(2), "Providers should have distinct names")
			})
		})

		Context("when provider has credentials configured", Ordered, func() {
			var tmpDir string

			BeforeAll(func() {
				var err error
				tmpDir, err = os.MkdirTemp("", "provider-creds-test-*")
				Expect(err).NotTo(HaveOccurred())

				provider := `name: test-provider
type: anthropic
credentials:
  API_KEY: "expected-api-key-value"
config:
  BASE_URL: "https://api.example.com"
`
				Expect(os.WriteFile(filepath.Join(tmpDir, "test.yaml"), []byte(provider), 0644)).To(Succeed())
			})

			AfterAll(func() {
				os.RemoveAll(tmpDir)
			})

			It("[test_id:TS-GH-18-006b] should map credentials correctly", func() {
				providers, err := harness.LoadProviderDefs(tmpDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(providers).To(HaveLen(1))

				provider := providers[0]
				Expect(provider.Name).To(Equal("test-provider"))
				Expect(provider.Type).To(Equal("anthropic"))
				Expect(provider.Credentials).To(HaveKeyWithValue("API_KEY", "expected-api-key-value"),
					"API key should be mapped from config")
				Expect(provider.Config).To(HaveKeyWithValue("BASE_URL", "https://api.example.com"),
					"Endpoint URL should be mapped from config")
			})
		})

		Context("when provider config is invalid", Ordered, func() {
			It("[test_id:TS-GH-18-006c] should return descriptive error", func() {
				tmpDir, err := os.MkdirTemp("", "provider-invalid-test-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(tmpDir)

				// Provider missing required 'name' field
				invalidProvider := `type: anthropic
credentials:
  API_KEY: "some-key"
`
				Expect(os.WriteFile(filepath.Join(tmpDir, "invalid.yaml"), []byte(invalidProvider), 0644)).To(Succeed())

				_, err = harness.LoadProviderDefs(tmpDir)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("name"),
					"Error should identify the missing required field")

				// Provider missing required 'type' field
				tmpDir2, err := os.MkdirTemp("", "provider-invalid2-test-*")
				Expect(err).NotTo(HaveOccurred())
				defer os.RemoveAll(tmpDir2)

				invalidProvider2 := `name: some-provider
credentials:
  API_KEY: "some-key"
`
				Expect(os.WriteFile(filepath.Join(tmpDir2, "invalid.yaml"), []byte(invalidProvider2), 0644)).To(Succeed())

				_, err = harness.LoadProviderDefs(tmpDir2)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("type"),
					"Error should identify the missing type field")
			})
		})
	})

	// =========================================================================
	// TS-GH-18-007: Security Configuration Defaults
	// =========================================================================
	Describe("Security Configuration Defaults", func() {

		Context("when using default security config", Ordered, func() {
			It("[test_id:TS-GH-18-007a] should default to fail-closed mode", func() {
				h := &harness.Harness{Agent: "test-agent.md"}
				Expect(h.FailModeClosed()).To(BeTrue(),
					"Default security config should use fail-closed mode")

				// Also test with explicit nil Security
				h2 := &harness.Harness{Agent: "test-agent.md", Security: nil}
				Expect(h2.FailModeClosed()).To(BeTrue(),
					"Nil security config should default to fail-closed")

				// Test with empty SecurityConfig
				h3 := &harness.Harness{
					Agent:    "test-agent.md",
					Security: &harness.SecurityConfig{},
				}
				Expect(h3.FailModeClosed()).To(BeTrue(),
					"Empty SecurityConfig should default to fail-closed")
			})
		})

		Context("when checking default security state", Ordered, func() {
			It("[test_id:TS-GH-18-007b] should be enabled by default", func() {
				h := &harness.Harness{Agent: "test-agent.md"}
				Expect(h.SecurityEnabled()).To(BeTrue(),
					"Security should be enabled by default")

				// Also test with explicit nil Security
				h2 := &harness.Harness{Agent: "test-agent.md", Security: nil}
				Expect(h2.SecurityEnabled()).To(BeTrue(),
					"Nil security config should default to enabled")

				// Test with empty SecurityConfig (Enabled is nil)
				h3 := &harness.Harness{
					Agent:    "test-agent.md",
					Security: &harness.SecurityConfig{},
				}
				Expect(h3.SecurityEnabled()).To(BeTrue(),
					"Empty SecurityConfig should default to enabled")
			})
		})

		Context("when toggles are explicitly configured", Ordered, func() {
			It("[test_id:TS-GH-18-007c] should respect explicit toggle values", func() {
				// Test BoolDefault with explicit true
				trueVal := true
				Expect(harness.BoolDefault(&trueVal, false)).To(BeTrue(),
					"Explicit true should override false default")

				// Test BoolDefault with explicit false
				falseVal := false
				Expect(harness.BoolDefault(&falseVal, true)).To(BeFalse(),
					"Explicit false should override true default")

				// Test SecurityEnabled with explicit true
				hEnabled := &harness.Harness{
					Agent:    "test-agent.md",
					Security: &harness.SecurityConfig{Enabled: boolPtr(true)},
				}
				Expect(hEnabled.SecurityEnabled()).To(BeTrue())

				// Test SecurityEnabled with explicit false
				hDisabled := &harness.Harness{
					Agent:    "test-agent.md",
					Security: &harness.SecurityConfig{Enabled: boolPtr(false)},
				}
				Expect(hDisabled.SecurityEnabled()).To(BeFalse(),
					"Explicit false should disable security")

				// Test FailModeClosed with explicit "open"
				hOpen := &harness.Harness{
					Agent:    "test-agent.md",
					Security: &harness.SecurityConfig{FailMode: "open"},
				}
				Expect(hOpen.FailModeClosed()).To(BeFalse(),
					"Explicit 'open' mode should not be fail-closed")

				// Test FailModeClosed with explicit "closed"
				hClosed := &harness.Harness{
					Agent:    "test-agent.md",
					Security: &harness.SecurityConfig{FailMode: "closed"},
				}
				Expect(hClosed.FailModeClosed()).To(BeTrue(),
					"Explicit 'closed' mode should be fail-closed")
			})
		})

		Context("when toggle pointers are nil", Ordered, func() {
			It("[test_id:TS-GH-18-007d] should apply safe default values", func() {
				// BoolDefault with nil pointer should return the default
				Expect(harness.BoolDefault(nil, true)).To(BeTrue(),
					"Nil toggle with true default should return true")
				Expect(harness.BoolDefault(nil, false)).To(BeFalse(),
					"Nil toggle with false default should return false")

				// SecurityEnabled with nil Security.Enabled should default to true
				h := &harness.Harness{
					Agent:    "test-agent.md",
					Security: &harness.SecurityConfig{Enabled: nil},
				}
				Expect(h.SecurityEnabled()).To(BeTrue(),
					"Nil Enabled pointer should default to true (secure by default)")

				// FailModeClosed with empty FailMode should default to closed
				h2 := &harness.Harness{
					Agent:    "test-agent.md",
					Security: &harness.SecurityConfig{FailMode: ""},
				}
				Expect(h2.FailModeClosed()).To(BeTrue(),
					"Empty FailMode should default to fail-closed")
			})
		})
	})
})
