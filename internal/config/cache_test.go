package config

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestAtomicWrite_Success(t *testing.T) {
	dir := t.TempDir()
	cache := NewCache(
		filepath.Join(dir, "config.yaml"),
		filepath.Join(dir, "config.yaml.meta"),
		"",
	)

	content, _ := yaml.Marshal(DefaultConfig())
	meta := BuildMeta("test-id", "TEST_GROUP", "", content)

	if err := cache.AtomicWrite(content, meta); err != nil {
		t.Fatalf("AtomicWrite failed: %v", err)
	}

	// Main cache file must exist and be parseable.
	data, err := os.ReadFile(cache.File)
	if err != nil {
		t.Fatalf("cache file not readable: %v", err)
	}
	var hcfg HookConfig
	if err := yaml.Unmarshal(data, &hcfg); err != nil {
		t.Errorf("cache file not valid YAML: %v", err)
	}

	// Meta file must exist.
	if _, err := os.Stat(cache.MetaFile); err != nil {
		t.Errorf("meta file not created: %v", err)
	}
}

func TestAtomicWrite_InvalidYAMLRejected(t *testing.T) {
	dir := t.TempDir()
	cache := NewCache(
		filepath.Join(dir, "config.yaml"),
		filepath.Join(dir, "config.yaml.meta"),
		"",
	)

	// Write valid config first so there is an existing cache.
	valid, _ := yaml.Marshal(DefaultConfig())
	meta := BuildMeta("id", "G", "", valid)
	if err := cache.AtomicWrite(valid, meta); err != nil {
		t.Fatalf("initial write failed: %v", err)
	}

	oldContent, _ := os.ReadFile(cache.File)

	// Attempt to overwrite with invalid YAML.
	err := cache.AtomicWrite([]byte(":{invalid"), meta)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}

	// Old cache must be intact.
	newContent, _ := os.ReadFile(cache.File)
	if string(newContent) != string(oldContent) {
		t.Error("old cache was overwritten despite invalid new content")
	}
}

func TestAtomicWrite_InvalidConfigRejected(t *testing.T) {
	dir := t.TempDir()
	cache := NewCache(
		filepath.Join(dir, "config.yaml"),
		filepath.Join(dir, "config.yaml.meta"),
		"",
	)

	bad := DefaultConfig()
	bad.Version = "99.0" // unsupported version
	badContent, _ := yaml.Marshal(bad)

	err := cache.AtomicWrite(badContent, Meta{})
	if err == nil {
		t.Error("expected error for invalid config structure")
	}

	// No file should be written.
	if _, err := os.Stat(cache.File); !os.IsNotExist(err) {
		t.Error("cache file should not exist after rejected write")
	}
}

func TestContentMD5(t *testing.T) {
	a := ContentMD5([]byte("hello"))
	b := ContentMD5([]byte("hello"))
	if a != b {
		t.Error("MD5 of same content should be identical")
	}
	c := ContentMD5([]byte("world"))
	if a == c {
		t.Error("MD5 of different content should differ")
	}
}
