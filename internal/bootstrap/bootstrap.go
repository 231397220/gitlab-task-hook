package bootstrap

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const (
	DefaultPath = "/etc/gitlab-task-hook/bootstrap.yaml"
	EnvPath     = "GITLAB_TASK_HOOK_BOOTSTRAP"
)

// Config is the bootstrap configuration for Nacos connectivity.
// It is loaded once at startup and is not stored in Nacos itself.
type Config struct {
	Nacos NacosConf `yaml:"nacos"`
}

// NacosConf holds all Nacos connection and sync parameters.
type NacosConf struct {
	Enabled             bool   `yaml:"enabled"`
	ServerAddr          string `yaml:"server_addr"`
	NamespaceID         string `yaml:"namespace_id"`
	Group               string `yaml:"group"`
	DataID              string `yaml:"data_id"`
	Username            string `yaml:"username"`
	Password            string `yaml:"password"`   // never log
	AccessKey           string `yaml:"access_key"`
	SecretKey           string `yaml:"secret_key"` // never log
	TimeoutSeconds      int    `yaml:"timeout_seconds"`
	WatchEnabled        bool   `yaml:"watch_enabled"`
	PollIntervalSeconds int    `yaml:"poll_interval_seconds"`
	CacheFile           string `yaml:"cache_file"`
	CacheMetaFile       string `yaml:"cache_meta_file"`
	LogFile             string `yaml:"log_file"`
}

// Load reads the bootstrap config from path.
// If path is empty it falls back to the GITLAB_TASK_HOOK_BOOTSTRAP env var,
// then to the default path /etc/gitlab-task-hook/bootstrap.yaml.
func Load(path string) (*Config, error) {
	if path == "" {
		if envPath := os.Getenv(EnvPath); envPath != "" {
			path = envPath
		} else {
			path = DefaultPath
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read bootstrap %s: %w", path, err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse bootstrap %s: %w", path, err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid bootstrap %s: %w", path, err)
	}

	return cfg, nil
}

func (c *Config) validate() error {
	n := &c.Nacos
	if !n.Enabled {
		return nil
	}
	if n.ServerAddr == "" {
		return fmt.Errorf("nacos.server_addr is required")
	}
	if n.Group == "" {
		return fmt.Errorf("nacos.group is required")
	}
	if n.DataID == "" {
		return fmt.Errorf("nacos.data_id is required")
	}
	if n.CacheFile == "" {
		return fmt.Errorf("nacos.cache_file is required")
	}
	if n.TimeoutSeconds <= 0 {
		n.TimeoutSeconds = 5
	}
	if n.PollIntervalSeconds <= 0 {
		n.PollIntervalSeconds = 30
	}
	if n.CacheMetaFile == "" {
		n.CacheMetaFile = n.CacheFile + ".meta"
	}
	return nil
}
