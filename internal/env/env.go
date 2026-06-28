package env

import (
	"os"
	"strings"

	"gitlab-task-hook/internal/config"
)

// Config holds all runtime configuration derived from environment variables.
type Config struct {
	GLUsername    string
	GLProjectPath string
	GLProtocol    string
	HookMode      string // "enforce" or "warn"; default is enforce

	WhitelistUsers        []string // from WHITELIST_USERS, space-separated
	WhitelistProjectNames []string // from WHITELIST_PROJECT_NAMES, comma-separated
	ExemptMessageRegex    string   // from EXEMPT_MESSAGE_REGEX
	ExemptMergeCommit     bool     // from EXEMPT_MERGE_COMMIT, default true
}

// IsWarnMode returns true only when HOOK_MODE is explicitly set to "warn".
func (c Config) IsWarnMode() bool {
	return strings.ToLower(c.HookMode) == "warn"
}

// Load reads all relevant environment variables and returns a Config.
func Load() Config {
	c := Config{
		GLUsername:        getenv("GL_USERNAME"),
		GLProjectPath:     getenv("GL_PROJECT_PATH"),
		GLProtocol:        getenv("GL_PROTOCOL"),
		HookMode:          getenv("HOOK_MODE"),
		ExemptMessageRegex: getenv("EXEMPT_MESSAGE_REGEX"),
		ExemptMergeCommit: config.DefaultExemptMergeCommit,
	}

	// Parse WHITELIST_USERS (space-separated, lowercase for comparison)
	if raw := getenv("WHITELIST_USERS"); raw != "" {
		for _, u := range strings.Fields(raw) {
			if u != "" {
				c.WhitelistUsers = append(c.WhitelistUsers, strings.ToLower(u))
			}
		}
	}

	// Parse WHITELIST_PROJECT_NAMES (comma-separated, lowercase for comparison)
	if raw := getenv("WHITELIST_PROJECT_NAMES"); raw != "" {
		for _, p := range strings.Split(raw, ",") {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				c.WhitelistProjectNames = append(c.WhitelistProjectNames, strings.ToLower(trimmed))
			}
		}
	}

	// Parse EXEMPT_MERGE_COMMIT (default true; only false when explicitly "false")
	if raw := strings.ToLower(getenv("EXEMPT_MERGE_COMMIT")); raw == "false" {
		c.ExemptMergeCommit = false
	}

	return c
}

func getenv(key string) string {
	return os.Getenv(key)
}
