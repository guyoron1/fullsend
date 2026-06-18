package tests

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/fullsend-ai/fullsend/internal/sandbox"
)

/*
EnsureProvider Secret Redaction Tests

STP Reference: outputs/stp/GH-28/GH-28_test_plan.md
STD Reference: outputs/std/GH-28/GH-28_test_description.yaml
Jira: GH-28
*/

var _ = Describe("[GH-28] EnsureProvider secret redaction", func() {
	/*
		Markers:
			- tier1

		Preconditions:
			- Go toolchain 1.21+ available
			- Mock openshell binaries configured via GinkgoT().TempDir() and PATH override
	*/

	Context("when initial create fails with non-AlreadyExists error", Ordered, func() {
		var (
			tmpDir      string
			secretValue string
		)

		BeforeAll(func() {
			tmpDir = GinkgoT().TempDir()
			secretValue = "super-secret-token-98765"
			GinkgoT().Setenv("MY_TOKEN", secretValue)

			// Mock openshell: create fails with error containing the secret value.
			fakeOpenshell := filepath.Join(tmpDir, "openshell")
			script := "#!/bin/sh\n" +
				"echo \"Error: auth failed with token " + secretValue + "\" >&2\n" +
				"exit 1\n"
			Expect(os.WriteFile(fakeOpenshell, []byte(script), 0o755)).To(Succeed())
			GinkgoT().Setenv("PATH", tmpDir)
		})

		It("[test_id:TS-GH-28-004] should redact credential values from error output", func() {
			err := sandbox.EnsureProvider("test-provider", "github",
				map[string]string{"MY_TOKEN": "${MY_TOKEN}"}, nil)
			Expect(err).To(HaveOccurred(), "EnsureProvider should return error on create failure")
			Expect(err.Error()).NotTo(ContainSubstring(secretValue),
				"error message must not contain raw secret value")
			Expect(err.Error()).To(ContainSubstring("***"),
				"error should contain redacted placeholder")
			Expect(err.Error()).To(ContainSubstring("test-provider"),
				"error should still contain provider name for debugging context")
		})
	})

	Context("when recreate fails after successful delete", Ordered, func() {
		var (
			tmpDir      string
			secretValue string
		)

		BeforeAll(func() {
			tmpDir = GinkgoT().TempDir()
			secretValue = "recreate-secret-value-abc123"
			GinkgoT().Setenv("MY_SECRET", secretValue)

			// Mock openshell: first create returns AlreadyExists,
			// delete succeeds, second create fails with secret in output.
			fakeOpenshell := filepath.Join(tmpDir, "openshell")
			markerFile := filepath.Join(tmpDir, "state")
			script := "#!/bin/sh\n" +
				"if [ \"$1\" = \"provider\" ] && [ \"$2\" = \"create\" ]; then\n" +
				"  if [ ! -f \"" + markerFile + "\" ]; then\n" +
				"    touch \"" + markerFile + "\"\n" +
				"    echo 'Error: AlreadyExists' >&2\n" +
				"    exit 1\n" +
				"  else\n" +
				"    echo \"Error: recreate failed with secret " + secretValue + " in output\" >&2\n" +
				"    exit 1\n" +
				"  fi\n" +
				"fi\n" +
				"if [ \"$1\" = \"provider\" ] && [ \"$2\" = \"delete\" ]; then\n" +
				"  exit 0\n" +
				"fi\n" +
				"exit 1\n"
			Expect(os.WriteFile(fakeOpenshell, []byte(script), 0o755)).To(Succeed())
			GinkgoT().Setenv("PATH", tmpDir)
		})

		It("[test_id:TS-GH-28-005] should redact credential values from recreate error", func() {
			err := sandbox.EnsureProvider("test-provider", "github",
				map[string]string{"MY_SECRET": "${MY_SECRET}"}, nil)
			Expect(err).To(HaveOccurred(), "EnsureProvider should return error on recreate failure")
			Expect(err.Error()).NotTo(ContainSubstring(secretValue),
				"recreate error must not contain raw secret value")
			Expect(err.Error()).To(ContainSubstring("***"),
				"error should contain redacted placeholder")
			Expect(err.Error()).To(ContainSubstring("recreate"),
				"error should indicate recreate failure specifically")
		})
	})

	Context("when error contains multiple credential values", Ordered, func() {
		var (
			tmpDir       string
			clientID     string
			clientSecret string
			token        string
		)

		BeforeAll(func() {
			tmpDir = GinkgoT().TempDir()
			clientID = "client-id-unique-val-999"
			clientSecret = "client-secret-unique-val-888"
			token = "api-token-unique-val-777"

			GinkgoT().Setenv("CLIENT_ID", clientID)
			GinkgoT().Setenv("CLIENT_SECRET", clientSecret)
			GinkgoT().Setenv("API_TOKEN", token)

			// Mock openshell: create fails with all credential values in error output.
			fakeOpenshell := filepath.Join(tmpDir, "openshell")
			script := "#!/bin/sh\n" +
				"echo \"Error: failed with client_id=" + clientID +
				" client_secret=" + clientSecret +
				" token=" + token + "\" >&2\n" +
				"exit 1\n"
			Expect(os.WriteFile(fakeOpenshell, []byte(script), 0o755)).To(Succeed())
			GinkgoT().Setenv("PATH", tmpDir)
		})

		It("[test_id:TS-GH-28-006] should redact all credential values", func() {
			err := sandbox.EnsureProvider("test-provider", "github",
				map[string]string{
					"CLIENT_ID":     "${CLIENT_ID}",
					"CLIENT_SECRET": "${CLIENT_SECRET}",
					"API_TOKEN":     "${API_TOKEN}",
				}, nil)
			Expect(err).To(HaveOccurred(), "EnsureProvider should return error")

			// Verify each credential is independently redacted.
			errMsg := err.Error()
			Expect(errMsg).NotTo(ContainSubstring(clientID),
				"client_id must be redacted from error")
			Expect(errMsg).NotTo(ContainSubstring(clientSecret),
				"client_secret must be redacted from error")
			Expect(errMsg).NotTo(ContainSubstring(token),
				"api token must be redacted from error")
			Expect(errMsg).To(ContainSubstring("***"),
				"error should contain redacted placeholders")
		})
	})
})
