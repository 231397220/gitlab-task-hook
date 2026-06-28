package env

import (
	"os"
	"strings"
)

// Config holds the per-push runtime values injected by Gitaly via environment
// variables. All other configuration (whitelists, regexes, mode) comes from the
// YAML config file loaded by the hook subcommand.
type Config struct {
	GLUsername    string
	GLProjectPath string
	GLProtocol    string
	// HookMode is the optional HOOK_MODE override ("enforce" or "warn").
	// When non-empty it takes precedence over the YAML mode.default setting.
	HookMode string
}

// IsWarnMode returns true only when HOOK_MODE is explicitly set to "warn".
// If HookMode is empty the hook subcommand falls back to the YAML mode.default.
func (c Config) IsWarnMode() bool {
	return strings.ToLower(c.HookMode) == "warn"
}

// Load reads the Gitaly-injected environment variables into a Config.
func Load() Config {
	return Config{
		GLUsername:    os.Getenv("GL_USERNAME"),
		GLProjectPath: os.Getenv("GL_PROJECT_PATH"),
		GLProtocol:    os.Getenv("GL_PROTOCOL"),
		HookMode:      os.Getenv("HOOK_MODE"),
	}
}
