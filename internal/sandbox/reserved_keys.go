package sandbox

import (
	"fmt"
	"strings"
)

// reservedEnvKeys is the canonical set of environment variable names that
// must not be used as provider credential keys or harness runner_env keys.
// Both code paths (credential delivery and sandbox env injection) inject
// variables into the sandbox child process, so a variable that is dangerous
// in one path is equally dangerous in the other.
//
// Categories:
//   - Infrastructure: shell/OS vars the sandbox relies on
//   - Dynamic linker: LD_PRELOAD / LD_LIBRARY_PATH allow arbitrary code injection
//   - Shell injection: BASH_ENV, ENV, PROMPT_COMMAND are sourced/executed by shells
//   - Proxy/network: allow MITM of outbound traffic
//   - TLS trust chain: override CA bundles, enabling TLS interception
//   - Git config/command: override git configuration or execute arbitrary
//     commands via git/SSH operations
//   - Runtime injection: NODE_OPTIONS, JAVA_TOOL_OPTIONS inject code into
//     language runtimes
//
// The FULLSEND_ prefix is checked separately in ValidateCredentialKeys
// (not here) because harness authors legitimately set FULLSEND_* keys in
// runner_env for platform integration.
var reservedEnvKeys = map[string]bool{
	// Infrastructure
	"PATH":    true,
	"HOME":    true,
	"SHELL":   true,
	"USER":    true,
	"LOGNAME": true,

	// Dynamic linker injection
	"LD_PRELOAD":      true,
	"LD_LIBRARY_PATH": true,

	// Shell injection
	"BASH_ENV":       true,
	"ENV":            true,
	"PROMPT_COMMAND": true,

	// Proxy / network interception
	"http_proxy":  true,
	"HTTP_PROXY":  true,
	"https_proxy": true,
	"HTTPS_PROXY": true,
	"no_proxy":    true,
	"NO_PROXY":    true,
	"ALL_PROXY":   true,
	"all_proxy":   true,
	"ftp_proxy":   true,
	"FTP_PROXY":   true,

	// TLS trust chain override
	"SSL_CERT_FILE":       true,
	"SSL_CERT_DIR":        true,
	"CURL_CA_BUNDLE":      true,
	"NODE_EXTRA_CA_CERTS": true,
	"REQUESTS_CA_BUNDLE":  true,
	"GIT_SSL_CAINFO":      true,
	"GIT_SSL_CAPATH":      true,
	"GIT_SSL_NO_VERIFY":   true,

	// Git config injection
	"GIT_CONFIG_GLOBAL": true,
	"GIT_CONFIG_SYSTEM": true,
	"GIT_EXEC_PATH":     true,
	"GIT_TEMPLATE_DIR":  true,

	// Git/SSH command execution
	"GIT_SSH_COMMAND":   true,
	"GIT_ASKPASS":       true,
	"GIT_EXTERNAL_DIFF": true,
	"SSH_ASKPASS":       true,

	// Runtime injection
	"NODE_OPTIONS":      true,
	"JAVA_TOOL_OPTIONS": true,
}

// reservedEnvKeysUpper is a case-folded lookup used by IsReservedEnvKey
// for case-insensitive matching. It is built from all entries in
// reservedEnvKeys during init(). Proxy variables are case-sensitive in
// practice (curl distinguishes http_proxy from HTTP_PROXY), so the
// canonical map lists both forms explicitly; the upper-case map provides
// an additional case-insensitive catch-all for any variant.
var reservedEnvKeysUpper map[string]bool

func init() {
	reservedEnvKeysUpper = make(map[string]bool, len(reservedEnvKeys))
	for k := range reservedEnvKeys {
		reservedEnvKeysUpper[strings.ToUpper(k)] = true
	}
}

// IsReservedEnvKey reports whether key is a security-sensitive environment
// variable that must not be set via provider credentials or harness
// runner_env. The check covers:
//  1. Exact match against reservedEnvKeys (proxy vars are case-sensitive)
//  2. Case-insensitive match for all entries (e.g., "path" matches "PATH")
//
// Note: the FULLSEND_ prefix is NOT checked here. It is checked separately
// in ValidateCredentialKeys because harness authors legitimately configure
// FULLSEND_* keys in runner_env (e.g., FULLSEND_OUTPUT_SCHEMA).
func IsReservedEnvKey(key string) bool {
	// Exact match (handles case-sensitive proxy vars).
	if reservedEnvKeys[key] {
		return true
	}

	// Case-insensitive match for the full set.
	if reservedEnvKeysUpper[strings.ToUpper(key)] {
		return true
	}

	return false
}

// ValidateCredentialKeys checks that none of the credential key names are
// security-sensitive reserved keys or use the FULLSEND_ prefix (reserved
// for platform internals). Returns an error naming the first reserved key
// found, or nil if all keys are safe.
func ValidateCredentialKeys(credentials map[string]string) error {
	for k := range credentials {
		if IsReservedEnvKey(k) {
			return fmt.Errorf("credential key %q is a reserved environment variable and cannot be used as a provider credential", k)
		}
		if strings.HasPrefix(strings.ToUpper(k), "FULLSEND_") {
			return fmt.Errorf("credential key %q uses the reserved FULLSEND_ prefix and cannot be used as a provider credential", k)
		}
	}
	return nil
}

// ValidateEnvKeys checks that none of the given environment variable key
// names are security-sensitive reserved keys. The context string is used
// in error messages (e.g., "runner_env", "sandbox env").
//
// FULLSEND_ prefix keys are allowed here because harness authors
// legitimately set them for platform integration (e.g., FULLSEND_OUTPUT_SCHEMA).
func ValidateEnvKeys(env map[string]string, context string) error {
	for k := range env {
		if IsReservedEnvKey(k) {
			return fmt.Errorf("%s key %q is a reserved environment variable and cannot be set", context, k)
		}
	}
	return nil
}
