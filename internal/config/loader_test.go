package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLoad_FileAbsent(t *testing.T) {
	cfg, err := Load("/nonexistent/path/to/config.yaml")
	if !errors.Is(err, ErrConfigNotFound) {
		t.Errorf("expected ErrConfigNotFound, got %v", err)
	}
	if cfg == nil {
		t.Error("expected non-nil default config when file is absent")
	}
}

func TestLoad_ValidFile(t *testing.T) {
	path := writeConfigFile(t, DefaultConfig())
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.Version != "1.0" {
		t.Errorf("version = %q, want 1.0", cfg.Version)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.yaml")
	if err := os.WriteFile(path, []byte(":{invalid yaml"), 0640); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoad_InvalidConfig(t *testing.T) {
	bad := DefaultConfig()
	bad.Version = "99.0"
	path := writeConfigFile(t, bad)
	_, err := Load(path)
	if err == nil {
		t.Error("expected error for unsupported version")
	}
}

func TestLoad_EnvPathOverride(t *testing.T) {
	path := writeConfigFile(t, DefaultConfig())
	t.Setenv(EnvCachePath, path)
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
}

// writeConfigFile marshals cfg to YAML and writes it to a temp file.
func writeConfigFile(t *testing.T, cfg *HookConfig) string {
	t.Helper()
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.yaml")
	if err := os.WriteFile(path, data, 0640); err != nil {
		t.Fatalf("write config file: %v", err)
	}
	return path
}
