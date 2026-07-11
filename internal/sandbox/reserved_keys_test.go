package sandbox

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsReservedEnvKey_ExactMatch(t *testing.T) {
	// Spot-check representatives from each category.
	reserved := []string{
		"PATH", "HOME", "SHELL", "USER", "LOGNAME",
		"LD_PRELOAD", "LD_LIBRARY_PATH",
		"BASH_ENV", "ENV", "PROMPT_COMMAND",
		"http_proxy", "HTTP_PROXY", "https_proxy", "HTTPS_PROXY",
		"no_proxy", "NO_PROXY", "ALL_PROXY", "all_proxy",
		"ftp_proxy", "FTP_PROXY",
		"SSL_CERT_FILE", "SSL_CERT_DIR", "CURL_CA_BUNDLE",
		"NODE_EXTRA_CA_CERTS", "REQUESTS_CA_BUNDLE",
		"GIT_SSL_CAINFO", "GIT_SSL_CAPATH", "GIT_SSL_NO_VERIFY",
		"GIT_CONFIG_GLOBAL", "GIT_CONFIG_SYSTEM",
		"GIT_EXEC_PATH", "GIT_TEMPLATE_DIR",
	}

	for _, key := range reserved {
		assert.True(t, IsReservedEnvKey(key), "expected %q to be reserved", key)
	}
}

func TestIsReservedEnvKey_CaseInsensitive(t *testing.T) {
	// Non-proxy vars should match case-insensitively.
	assert.True(t, IsReservedEnvKey("path"))
	assert.True(t, IsReservedEnvKey("Path"))
	assert.True(t, IsReservedEnvKey("ld_preload"))
	assert.True(t, IsReservedEnvKey("bash_env"))
	assert.True(t, IsReservedEnvKey("Ssl_Cert_File"))
}

func TestIsReservedEnvKey_FullsendPrefix(t *testing.T) {
	assert.True(t, IsReservedEnvKey("FULLSEND_OUTPUT_DIR"))
	assert.True(t, IsReservedEnvKey("FULLSEND_TRACE_ID"))
	assert.True(t, IsReservedEnvKey("fullsend_custom"))
	assert.True(t, IsReservedEnvKey("Fullsend_Anything"))
}

func TestIsReservedEnvKey_AllowedKeys(t *testing.T) {
	// Application-specific keys should not be blocked.
	allowed := []string{
		"API_KEY",
		"MY_SECRET",
		"DATABASE_URL",
		"ANTHROPIC_API_KEY",
		"OPENAI_BASE_URL",
		"GH_TOKEN",
		"GITHUB_TOKEN",
		"REPO_NAME",
		"CUSTOM_VAR",
	}

	for _, key := range allowed {
		assert.False(t, IsReservedEnvKey(key), "expected %q to be allowed", key)
	}
}

func TestValidateCredentialKeys_RejectsReserved(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{"blocks PATH", "PATH"},
		{"blocks LD_PRELOAD", "LD_PRELOAD"},
		{"blocks http_proxy", "http_proxy"},
		{"blocks FULLSEND_ prefix", "FULLSEND_OUTPUT_DIR"},
		{"blocks GIT_CONFIG_GLOBAL", "GIT_CONFIG_GLOBAL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creds := map[string]string{tt.key: "value"}
			err := ValidateCredentialKeys(creds)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.key)
			assert.Contains(t, err.Error(), "reserved")
		})
	}
}

func TestValidateCredentialKeys_AllowsSafe(t *testing.T) {
	creds := map[string]string{
		"API_KEY":       "secret",
		"ANTHROPIC_KEY": "secret",
		"DATABASE_URL":  "postgres://...",
	}
	err := ValidateCredentialKeys(creds)
	assert.NoError(t, err)
}

func TestValidateCredentialKeys_EmptyMap(t *testing.T) {
	err := ValidateCredentialKeys(map[string]string{})
	assert.NoError(t, err)
}

func TestValidateCredentialKeys_NilMap(t *testing.T) {
	err := ValidateCredentialKeys(nil)
	assert.NoError(t, err)
}

func TestValidateEnvKeys_RejectsReserved(t *testing.T) {
	env := map[string]string{"LD_PRELOAD": "/malicious.so"}
	err := ValidateEnvKeys(env, "runner_env")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "runner_env")
	assert.Contains(t, err.Error(), "LD_PRELOAD")
	assert.Contains(t, err.Error(), "reserved")
}

func TestValidateEnvKeys_AllowsSafe(t *testing.T) {
	env := map[string]string{
		"REPO_NAME": "my-repo",
		"GH_TOKEN":  "ghp_xxx",
	}
	err := ValidateEnvKeys(env, "runner_env")
	assert.NoError(t, err)
}

// TestBlocklistParity verifies that ValidateCredentialKeys and
// ValidateEnvKeys use the same underlying blocklist. Every key
// blocked as a credential must also be blocked as a runner_env key,
// and vice versa.
func TestBlocklistParity(t *testing.T) {
	// Collect all keys from the canonical ReservedEnvKeys map plus
	// representative FULLSEND_* prefix keys.
	allReserved := make([]string, 0, len(ReservedEnvKeys)+2)
	for k := range ReservedEnvKeys {
		allReserved = append(allReserved, k)
	}
	allReserved = append(allReserved, "FULLSEND_OUTPUT_DIR", "FULLSEND_TRACE_ID")

	for _, key := range allReserved {
		credErr := ValidateCredentialKeys(map[string]string{key: "val"})
		envErr := ValidateEnvKeys(map[string]string{key: "val"}, "test")

		// Both must reject the same key.
		assert.Error(t, credErr, "credential path should block %q", key)
		assert.Error(t, envErr, "env path should block %q", key)
	}
}
