package sandbox

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReservedEnvKeys_CoversSecurity verifies that all security-sensitive
// categories are represented in the unified blocklist. This prevents future
// drift where one category is accidentally omitted.
func TestReservedEnvKeys_CoversSecurity(t *testing.T) {
	categories := map[string][]string{
		"infrastructure":  {"PATH", "HOME", "SHELL"},
		"dynamic_linker":  {"LD_PRELOAD", "LD_LIBRARY_PATH"},
		"shell_injection": {"BASH_ENV", "ENV"},
		"proxy": {
			"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy",
			"ALL_PROXY", "all_proxy", "NO_PROXY", "no_proxy",
		},
		"tls_trust_chain": {
			"SSL_CERT_FILE", "SSL_CERT_DIR", "NODE_EXTRA_CA_CERTS",
			"REQUESTS_CA_BUNDLE", "CURL_CA_BUNDLE",
			"GIT_SSL_CAINFO", "GIT_SSL_CAPATH", "GIT_SSL_NO_VERIFY",
		},
		"git_config": {
			"GIT_CONFIG_GLOBAL", "GIT_CONFIG_SYSTEM",
			"GIT_EXEC_PATH", "GIT_TEMPLATE_DIR",
		},
		"runtime_injection": {
			"PYTHONSTARTUP", "PYTHONPATH", "NODE_OPTIONS",
			"PERL5LIB", "PERL5OPT", "RUBYLIB", "RUBYOPT",
		},
	}
	for category, keys := range categories {
		for _, key := range keys {
			assert.True(t, ReservedEnvKeys[key],
				"ReservedEnvKeys must include %q (category: %s)", key, category)
		}
	}
}

func TestIsReservedEnvKey_ExactMatch(t *testing.T) {
	reserved := []string{
		"LD_PRELOAD", "LD_LIBRARY_PATH",
		"HTTP_PROXY", "HTTPS_PROXY",
		"PATH", "HOME", "SHELL",
		"BASH_ENV", "ENV",
		"SSL_CERT_FILE", "GIT_CONFIG_GLOBAL",
		"NODE_OPTIONS", "PYTHONSTARTUP",
	}
	for _, key := range reserved {
		assert.True(t, IsReservedEnvKey(key), "%q should be reserved", key)
	}
}

func TestIsReservedEnvKey_FullsendPrefix(t *testing.T) {
	assert.True(t, IsReservedEnvKey("FULLSEND_OUTPUT_DIR"))
	assert.True(t, IsReservedEnvKey("FULLSEND_TOKEN"))
	assert.True(t, IsReservedEnvKey("FULLSEND_TRACE_ID"))
}

func TestIsReservedEnvKey_SafeKeys(t *testing.T) {
	safe := []string{
		"API_KEY",
		"MY_SECRET",
		"ANTHROPIC_API_KEY",
		"OPENAI_API_KEY",
		"GH_TOKEN",
		"REPO_NAME",
	}
	for _, key := range safe {
		assert.False(t, IsReservedEnvKey(key), "%q should not be reserved", key)
	}
}

func TestValidateCredentialKeys_RejectsReserved(t *testing.T) {
	tests := []struct {
		name        string
		credentials map[string]string
		wantKey     string
	}{
		{
			name:        "LD_PRELOAD",
			credentials: map[string]string{"API_KEY": "secret", "LD_PRELOAD": "/malicious.so"},
			wantKey:     "LD_PRELOAD",
		},
		{
			name:        "HTTP_PROXY",
			credentials: map[string]string{"HTTP_PROXY": "http://evil.proxy"},
			wantKey:     "HTTP_PROXY",
		},
		{
			name:        "FULLSEND_prefix",
			credentials: map[string]string{"FULLSEND_TOKEN": "secret"},
			wantKey:     "FULLSEND_TOKEN",
		},
		{
			name:        "PATH",
			credentials: map[string]string{"PATH": "/override"},
			wantKey:     "PATH",
		},
		{
			name:        "GIT_CONFIG_GLOBAL",
			credentials: map[string]string{"GIT_CONFIG_GLOBAL": "/evil/gitconfig"},
			wantKey:     "GIT_CONFIG_GLOBAL",
		},
		{
			name:        "SSL_CERT_FILE",
			credentials: map[string]string{"SSL_CERT_FILE": "/evil/cert.pem"},
			wantKey:     "SSL_CERT_FILE",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCredentialKeys(tt.credentials)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantKey)
			assert.Contains(t, err.Error(), "reserved")
		})
	}
}

func TestValidateCredentialKeys_AcceptsSafe(t *testing.T) {
	creds := map[string]string{
		"API_KEY":       "secret",
		"ANTHROPIC_KEY": "sk-ant-1234",
		"OPENAI_KEY":    "sk-1234",
	}
	require.NoError(t, ValidateCredentialKeys(creds))
}

func TestValidateCredentialKeys_EmptyMap(t *testing.T) {
	require.NoError(t, ValidateCredentialKeys(map[string]string{}))
}

func TestValidateCredentialKeys_NilMap(t *testing.T) {
	require.NoError(t, ValidateCredentialKeys(nil))
}

func TestValidateEnvKeys_RejectsReserved(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantKey string
	}{
		{
			name:    "LD_PRELOAD",
			env:     map[string]string{"MY_VAR": "safe", "LD_PRELOAD": "/malicious.so"},
			wantKey: "LD_PRELOAD",
		},
		{
			name:    "HTTP_PROXY",
			env:     map[string]string{"HTTP_PROXY": "http://evil.proxy"},
			wantKey: "HTTP_PROXY",
		},
		{
			name:    "NODE_OPTIONS",
			env:     map[string]string{"NODE_OPTIONS": "--require /evil.js"},
			wantKey: "NODE_OPTIONS",
		},
		{
			name:    "FULLSEND_prefix",
			env:     map[string]string{"FULLSEND_OUTPUT_DIR": "/override"},
			wantKey: "FULLSEND_OUTPUT_DIR",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEnvKeys(tt.env)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantKey)
			assert.Contains(t, err.Error(), "reserved")
		})
	}
}

func TestValidateEnvKeys_AcceptsSafe(t *testing.T) {
	env := map[string]string{
		"MY_VAR":    "safe",
		"REPO_NAME": "fullsend",
		"GH_TOKEN":  "${GH_TOKEN}",
	}
	require.NoError(t, ValidateEnvKeys(env))
}

func TestValidateEnvKeys_EmptyMap(t *testing.T) {
	require.NoError(t, ValidateEnvKeys(map[string]string{}))
}

func TestValidateEnvKeys_NilMap(t *testing.T) {
	require.NoError(t, ValidateEnvKeys(nil))
}

// TestBlocklistCoversLowerCaseProxyVariants ensures that both upper and lower
// case proxy variables are blocked. Many HTTP clients honour both forms.
func TestBlocklistCoversLowerCaseProxyVariants(t *testing.T) {
	pairs := [][2]string{
		{"HTTP_PROXY", "http_proxy"},
		{"HTTPS_PROXY", "https_proxy"},
		{"ALL_PROXY", "all_proxy"},
		{"NO_PROXY", "no_proxy"},
		{"FTP_PROXY", "ftp_proxy"},
	}
	for _, pair := range pairs {
		assert.True(t, ReservedEnvKeys[pair[0]],
			"upper-case %q should be in blocklist", pair[0])
		assert.True(t, ReservedEnvKeys[pair[1]],
			"lower-case %q should be in blocklist", pair[1])
	}
}
