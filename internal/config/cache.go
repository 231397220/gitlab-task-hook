package config

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

// Meta is written as JSON alongside the main YAML cache file.
type Meta struct {
	DataID       string `json:"data_id"`
	Group        string `json:"group"`
	NamespaceID  string `json:"namespace_id"`
	MD5          string `json:"md5"`
	LastSyncTime string `json:"last_sync_time"`
	Source       string `json:"source"`
	SyncStatus   string `json:"sync_status"`
}

// Cache manages atomic writes to the local YAML cache and its meta file.
type Cache struct {
	File     string
	MetaFile string
	LockFile string
}

// NewCache builds a Cache from explicit paths. lockFile defaults to File+".lock".
func NewCache(file, metaFile, lockFile string) *Cache {
	if lockFile == "" {
		lockFile = file + ".lock"
	}
	return &Cache{File: file, MetaFile: metaFile, LockFile: lockFile}
}

// AtomicWrite validates content, then atomically replaces the cache and meta files.
// If content fails YAML parsing or config validation the old files are untouched.
func (c *Cache) AtomicWrite(content []byte, meta Meta) error {
	// Validate before touching any file.
	hcfg := &HookConfig{}
	if err := yaml.Unmarshal(content, hcfg); err != nil {
		return fmt.Errorf("cache write aborted: yaml parse: %w", err)
	}
	if errs := Validate(hcfg); len(errs) > 0 {
		return fmt.Errorf("cache write aborted: invalid config: %v", errs)
	}

	lock, err := c.acquireLock()
	if err != nil {
		return fmt.Errorf("acquire lock %s: %w", c.LockFile, err)
	}
	defer c.releaseLock(lock)

	if err := atomicWriteFile(c.File, content, 0640); err != nil {
		return fmt.Errorf("write cache %s: %w", c.File, err)
	}

	if c.MetaFile != "" {
		metaJSON, err := json.MarshalIndent(meta, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal meta: %w", err)
		}
		if err := atomicWriteFile(c.MetaFile, metaJSON, 0640); err != nil {
			return fmt.Errorf("write meta %s: %w", c.MetaFile, err)
		}
	}

	return nil
}

// ContentMD5 returns the hex-encoded MD5 of content.
func ContentMD5(content []byte) string {
	h := md5.Sum(content)
	return hex.EncodeToString(h[:])
}

// BuildMeta constructs a Meta from bootstrap fields and content.
func BuildMeta(dataID, group, namespaceID string, content []byte) Meta {
	return Meta{
		DataID:       dataID,
		Group:        group,
		NamespaceID:  namespaceID,
		MD5:          ContentMD5(content),
		LastSyncTime: time.Now().Format(time.RFC3339),
		Source:       "nacos",
		SyncStatus:   "success",
	}
}

// acquireLock opens (or creates) the lock file and obtains an exclusive flock.
func (c *Cache) acquireLock() (*os.File, error) {
	f, err := os.OpenFile(c.LockFile, os.O_CREATE|os.O_RDWR, 0640)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}

func (c *Cache) releaseLock(f *os.File) {
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	f.Close()
}

// atomicWriteFile writes data to a temp file, fsyncs it, then renames it to dst.
func atomicWriteFile(dst string, data []byte, perm os.FileMode) error {
	tmp := dst + ".tmp." + strconv.Itoa(os.Getpid())

	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}

	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}

	if err := os.Rename(tmp, dst); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}
