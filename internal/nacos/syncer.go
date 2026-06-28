package nacos

import (
	"context"
	"fmt"
	"time"

	"gitlab-task-hook/internal/bootstrap"
	"gitlab-task-hook/internal/config"
	"gitlab-task-hook/internal/log"
)

// Syncer pulls config from Nacos and writes it to the local cache.
type Syncer struct {
	client NacosClient
	cache  *config.Cache
	bsCfg  bootstrap.NacosConf
	log    *log.Logger
}

// NewSyncer constructs a Syncer from a live NacosClient.
func NewSyncer(client NacosClient, bsCfg bootstrap.NacosConf, logger *log.Logger) *Syncer {
	cache := config.NewCache(bsCfg.CacheFile, bsCfg.CacheMetaFile, "")
	return &Syncer{client: client, cache: cache, bsCfg: bsCfg, log: logger}
}

// SyncOnce fetches the config once, validates it, and writes it to the local cache.
// Returns an error on any failure; does NOT overwrite the old cache on failure.
func (s *Syncer) SyncOnce() error {
	s.log.Info("nacos config sync started",
		"server", s.bsCfg.ServerAddr,
		"data_id", s.bsCfg.DataID,
		"group", s.bsCfg.Group,
		"namespace", s.bsCfg.NamespaceID,
	)

	raw, err := s.client.GetConfig()
	if err != nil {
		return fmt.Errorf("get config from nacos: %w", err)
	}

	if err := s.applyConfig(raw); err != nil {
		return err
	}

	s.log.Info("nacos config sync succeeded",
		"md5", config.ContentMD5([]byte(raw)),
		"cache_file", s.bsCfg.CacheFile,
	)
	return nil
}

// Watch performs an initial sync, then starts listening for changes.
// It blocks until ctx is cancelled. Falls back to polling when
// watch_enabled=false or ListenConfig fails.
func (s *Syncer) Watch(ctx context.Context) error {
	// Initial sync (best-effort; old cache may already exist).
	if err := s.SyncOnce(); err != nil {
		s.log.Error("initial nacos sync failed", "err", err)
	}

	if s.bsCfg.WatchEnabled {
		err := s.client.ListenConfig(func(data string) {
			s.log.Info("nacos config change detected")
			if err := s.applyConfig(data); err != nil {
				s.log.Error("failed to apply nacos config change", "err", err)
			} else {
				s.log.Info("local cache updated",
					"md5", config.ContentMD5([]byte(data)),
				)
			}
		})
		if err != nil {
			s.log.Warn("nacos listen failed, falling back to polling", "err", err)
		} else {
			s.log.Info("nacos watch started")
			<-ctx.Done()
			_ = s.client.CancelListenConfig()
			s.log.Info("nacos watch stopped")
			return nil
		}
	}

	return s.runPoll(ctx)
}

// runPoll periodically fetches config and updates the cache when MD5 changes.
func (s *Syncer) runPoll(ctx context.Context) error {
	interval := time.Duration(s.bsCfg.PollIntervalSeconds) * time.Second
	if interval <= 0 {
		interval = 30 * time.Second
	}

	s.log.Info("nacos poll started", "interval_seconds", int(interval.Seconds()))

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var lastMD5 string
	for {
		select {
		case <-ctx.Done():
			s.log.Info("nacos poll stopped")
			return nil
		case <-ticker.C:
			raw, err := s.client.GetConfig()
			if err != nil {
				s.log.Error("nacos poll get config failed", "err", err)
				continue
			}
			m := config.ContentMD5([]byte(raw))
			if m == lastMD5 {
				continue
			}
			if err := s.applyConfig(raw); err != nil {
				s.log.Error("nacos poll write cache failed", "err", err)
			} else {
				lastMD5 = m
				s.log.Info("local cache updated via poll", "md5", m)
			}
		}
	}
}

// applyConfig validates raw YAML and writes it atomically to the local cache.
// The old cache is never overwritten on validation failure.
func (s *Syncer) applyConfig(raw string) error {
	content := []byte(raw)
	meta := config.BuildMeta(s.bsCfg.DataID, s.bsCfg.Group, s.bsCfg.NamespaceID, content)
	if err := s.cache.AtomicWrite(content, meta); err != nil {
		return fmt.Errorf("atomic write cache: %w", err)
	}
	return nil
}
