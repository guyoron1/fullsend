package sandbox

import (
	"fmt"
	"strings"
)

// ReservedEnvKeys is the unified set of environment variable names that must
// never be used as provider credential keys or sandbox environment variable
// names. A variable that is unsafe as a provider credential key is equally
// unsafe when injected via runner_env, since both paths result in env vars
// in the sandbox child process.
//
// Categories:
//   - Infrastructure: core process environment variables
//   - Dynamic linker injection: library preloading vectors
//   - Shell injection: startup script variables
//   - Proxy / network: HTTP(S) proxy override variables
//   - TLS trust chain: certificate trust store overrides
//   - Git config: git configuration injection vectors
//   - Runtime injection: language-specific startup/path injection
//
// The FULLSEND_ prefix is also reserved (checked by [IsReservedEnvKey]).
var ReservedEnvKeys = map[string]bool{
	// Infrastructure
	"PATH":     true,
	"HOME":     true,
	"SHELL":    true,
	"USER":     true,
	"LOGNAME":  true,
	"HOSTNAME": true,
	"TERM":     true,

	// Dynamic linker injection
	"LD_PRELOAD":            true,
	"LD_LIBRARY_PATH":       true,
	"DYLD_INSERT_LIBRARIES": true,
	"DYLD_LIBRARY_PATH":     true,

	// Shell injection
	"BASH_ENV":       true,
	"ENV":            true,
	"PROMPT_COMMAND": true,

	// Proxy / network
	"HTTP_PROXY":  true,
	"HTTPS_PROXY": true,
	"http_proxy":  true,
	"https_proxy": true,
	"ALL_PROXY":   true,
	"all_proxy":   true,
	"NO_PROXY":    true,
	"no_proxy":    true,
	"FTP_PROXY":   true,
	"ftp_proxy":   true,

	// TLS trust chain
	"SSL_CERT_FILE":       true,
	"SSL_CERT_DIR":        true,
	"NODE_EXTRA_CA_CERTS": true,
	"REQUESTS_CA_BUNDLE":  true,
	"CURL_CA_BUNDLE":      true,
	"GIT_SSL_CAINFO":      true,
	"GIT_SSL_CAPATH":      true,
	"GIT_SSL_NO_VERIFY":   true,
	"PIP_CERT":            true,
	"AWS_CA_BUNDLE":       true,

	// Git config
	"GIT_CONFIG_GLOBAL": true,
	"GIT_CONFIG_SYSTEM": true,
	"GIT_EXEC_PATH":     true,
	"GIT_TEMPLATE_DIR":  true,

	// Runtime injection
	"PYTHONSTARTUP": true,
	"PYTHONPATH":    true,
	"NODE_OPTIONS":  true,
	"PERL5LIB":      true,
	"PERL5OPT":      true,
	"RUBYLIB":       true,
	"RUBYOPT":       true,
}

// IsReservedEnvKey reports whether key is a reserved environment variable
// name. It checks both the exact key against [ReservedEnvKeys] and the
// FULLSEND_ prefix.
func IsReservedEnvKey(key string) bool {
	if ReservedEnvKeys[key] {
		return true
	}
	return strings.HasPrefix(key, "FULLSEND_")
}

// ValidateCredentialKeys checks that none of the credential key names in
// the map are reserved environment variables. Returns an error naming the
// first reserved key found.
func ValidateCredentialKeys(credentials map[string]string) error {
	for key := range credentials {
		if IsReservedEnvKey(key) {
			return fmt.Errorf("credential key %q is a reserved environment variable and cannot be used", key)
		}
	}
	return nil
}

// ValidateEnvKeys checks that none of the environment variable key names
// in the map are reserved. Returns an error naming the first reserved key
// found. Use this to validate runner_env keys before injecting them into
// the sandbox.
func ValidateEnvKeys(env map[string]string) error {
	for key := range env {
		if IsReservedEnvKey(key) {
			return fmt.Errorf("environment key %q is a reserved variable and cannot be set in sandbox env", key)
		}
	}
	return nil
}
