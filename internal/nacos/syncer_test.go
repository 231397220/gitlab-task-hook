package nacos

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"gitlab-task-hook/internal/bootstrap"
	"gitlab-task-hook/internal/config"
	"gitlab-task-hook/internal/log"
)

// mockClient implements NacosClient for testing.
type mockClient struct {
	content   string
	getErr    error
	listenErr error
	onChange  func(string)
}

func (m *mockClient) GetConfig() (string, error) {
	return m.content, m.getErr
}

func (m *mockClient) ListenConfig(onChange func(string)) error {
	if m.listenErr != nil {
		return m.listenErr
	}
	m.onChange = onChange
	return nil
}

func (m *mockClient) CancelListenConfig() error { return nil }

// trigger simulates a Nacos config-change push.
func (m *mockClient) trigger(data string) {
	if m.onChange != nil {
		m.onChange(data)
	}
}

func newTestSyncer(t *testing.T, client NacosClient) (*Syncer, string) {
	t.Helper()
	dir := t.TempDir()
	cacheFile := filepath.Join(dir, "config.yaml")
	metaFile := filepath.Join(dir, "config.yaml.meta")
	bsCfg := bootstrap.NacosConf{
		DataID:              "test.yaml",
		Group:               "TEST",
		CacheFile:           cacheFile,
		CacheMetaFile:       metaFile,
		WatchEnabled:        true,
		PollIntervalSeconds: 1,
	}
	syncer := NewSyncer(client, bsCfg, log.New(os.Stderr, "error"))
	return syncer, cacheFile
}

func validYAML() string {
	data, _ := yaml.Marshal(config.DefaultConfig())
	return string(data)
}

func TestSyncOnce_Success(t *testing.T) {
	client := &mockClient{content: validYAML()}
	syncer, cacheFile := newTestSyncer(t, client)

	if err := syncer.SyncOnce(); err != nil {
		t.Fatalf("SyncOnce failed: %v", err)
	}
	if _, err := os.Stat(cacheFile); err != nil {
		t.Errorf("cache file not created: %v", err)
	}
}

func TestSyncOnce_NacosError(t *testing.T) {
	client := &mockClient{getErr: errors.New("connection refused")}
	syncer, cacheFile := newTestSyncer(t, client)

	err := syncer.SyncOnce()
	if err == nil {
		t.Error("expected error when nacos is unavailable")
	}
	if _, err := os.Stat(cacheFile); !os.IsNotExist(err) {
		t.Error("cache file should not be created on nacos error")
	}
}

func TestSyncOnce_InvalidConfigNotWritten(t *testing.T) {
	// Write a valid cache first.
	client := &mockClient{content: validYAML()}
	syncer, cacheFile := newTestSyncer(t, client)
	if err := syncer.SyncOnce(); err != nil {
		t.Fatalf("initial sync failed: %v", err)
	}
	original, _ := os.ReadFile(cacheFile)

	// Now Nacos returns an invalid config.
	client.content = "version: \"99.0\"\nenabled: true\nmode:\n  default: enforce\n"
	err := syncer.SyncOnce()
	if err == nil {
		t.Error("expected error for invalid config from nacos")
	}

	// Old cache must be preserved.
	updated, _ := os.ReadFile(cacheFile)
	if string(updated) != string(original) {
		t.Error("old cache was overwritten despite invalid new config from nacos")
	}
}

func TestWatch_ListenFallbackToPoll(t *testing.T) {
	client := &mockClient{
		content:   validYAML(),
		listenErr: errors.New("listen not supported"),
	}
	syncer, cacheFile := newTestSyncer(t, client)
	syncer.bsCfg.PollIntervalSeconds = 1

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- syncer.Watch(ctx) }()

	// Wait for poll to run at least once.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(cacheFile); err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	cancel()
	<-done

	if _, err := os.Stat(cacheFile); err != nil {
		t.Error("cache file should exist after poll fallback")
	}
}
