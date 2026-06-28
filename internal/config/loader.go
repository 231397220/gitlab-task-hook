package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	DefaultCachePath = "/var/opt/gitlab/gitaly/custom_hooks/config/gitlab-task-hook.yaml"
	EnvCachePath     = "GITLAB_TASK_HOOK_CONFIG"
)

// ErrConfigNotFound is returned (alongside a default CompiledConfig) when the
// config file does not exist. The caller should output a warning but may continue.
var ErrConfigNotFound = errors.New("config file not found, using built-in defaults")

// Load reads and validates the YAML config from path.
// If path is empty it falls back to GITLAB_TASK_HOOK_CONFIG env var,
// then to DefaultCachePath.
//
//   - file absent    → (defaultConfig, ErrConfigNotFound)
//   - parse/validate fails → (nil, error)
//   - success        → (compiled, nil)
func Load(path string) (*CompiledConfig, error) {
	resolved := resolvePath(path)

	data, err := os.ReadFile(resolved)
	if err != nil {
		if os.IsNotExist(err) {
			compiled, _ := Compile(DefaultConfig())
			return compiled, ErrConfigNotFound
		}
		return nil, fmt.Errorf("read config %s: %w", resolved, err)
	}

	hcfg := &HookConfig{}
	if err := yaml.Unmarshal(data, hcfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", resolved, err)
	}

	if errs := Validate(hcfg); len(errs) > 0 {
		msgs := make([]string, len(errs))
		for i, e := range errs {
			msgs[i] = e.Error()
		}
		return nil, fmt.Errorf("invalid config %s: %s", resolved, strings.Join(msgs, "; "))
	}

	compiled, err := Compile(hcfg)
	if err != nil {
		return nil, fmt.Errorf("compile config: %w", err)
	}

	return compiled, nil
}

func resolvePath(path string) string {
	if path != "" {
		return path
	}
	if envPath := os.Getenv(EnvCachePath); envPath != "" {
		return envPath
	}
	return DefaultCachePath
}
