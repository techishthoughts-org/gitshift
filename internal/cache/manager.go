package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/techishthoughts/GitPersona/internal/observability"
)

// CacheManager manages performance caching for GitPersona
type CacheManager struct {
	logger        observability.Logger
	cacheDir      string
	entries       map[string]*CacheEntry
	mutex         sync.RWMutex
	maxEntries    int
	defaultTTL    time.Duration
	cleanupTicker *time.Ticker
	stopChan      chan struct{}
}

// CacheEntry represents a cached item
type CacheEntry struct {
	Key         string      `json:"key"`
	Value       interface{} `json:"value"`
	CreatedAt   time.Time   `json:"created_at"`
	ExpiresAt   time.Time   `json:"expires_at"`
	AccessCount int64       `json:"access_count"`
	LastAccess  time.Time   `json:"last_access"`
	Size        int64       `json:"size"`
	Tags        []string    `json:"tags"`
}

// CacheStats provides cache statistics
type CacheStats struct {
	TotalEntries   int64     `json:"total_entries"`
	HitCount       int64     `json:"hit_count"`
	MissCount      int64     `json:"miss_count"`
	HitRate        float64   `json:"hit_rate"`
	TotalSize      int64     `json:"total_size"`
	LastCleanup    time.Time `json:"last_cleanup"`
	EntriesEvicted int64     `json:"entries_evicted"`
	MemoryUsage    int64     `json:"memory_usage"`
}

// CacheOptions configures cache behavior
type CacheOptions struct {
	TTL  time.Duration
	Tags []string
}

// NewCacheManager creates a new cache manager
func NewCacheManager(logger observability.Logger, cacheDir string) *CacheManager {
	if cacheDir == "" {
		homeDir, _ := os.UserHomeDir()
		cacheDir = filepath.Join(homeDir, ".cache", "gitpersona")
	}

	// Ensure cache directory exists
	os.MkdirAll(cacheDir, 0755)

	cm := &CacheManager{
		logger:     logger,
		cacheDir:   cacheDir,
		entries:    make(map[string]*CacheEntry),
		maxEntries: 1000,
		defaultTTL: 30 * time.Minute,
		stopChan:   make(chan struct{}),
	}

	// Start cleanup routine
	cm.startCleanupRoutine()

	// Load existing cache entries
	cm.loadFromDisk()

	return cm
}

// Get retrieves a value from cache
func (cm *CacheManager) Get(ctx context.Context, key string) (interface{}, bool) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	entry, exists := cm.entries[key]
	if !exists {
		cm.logCacheMiss(ctx, key)
		return nil, false
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		delete(cm.entries, key)
		cm.logCacheExpired(ctx, key)
		return nil, false
	}

	// Update access statistics
	entry.AccessCount++
	entry.LastAccess = time.Now()

	cm.logCacheHit(ctx, key)
	return entry.Value, true
}

// Set stores a value in cache
func (cm *CacheManager) Set(ctx context.Context, key string, value interface{}, options *CacheOptions) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Use default options if none provided
	if options == nil {
		options = &CacheOptions{
			TTL: cm.defaultTTL,
		}
	}

	// Calculate size estimate
	size := cm.estimateSize(value)

	entry := &CacheEntry{
		Key:         key,
		Value:       value,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(options.TTL),
		AccessCount: 0,
		LastAccess:  time.Now(),
		Size:        size,
		Tags:        options.Tags,
	}

	cm.entries[key] = entry

	// Check if we need to evict entries
	if len(cm.entries) > cm.maxEntries {
		cm.evictLRU()
	}

	cm.logger.Debug(ctx, "cache_entry_stored",
		observability.F("key", key),
		observability.F("size", size),
		observability.F("ttl", options.TTL.String()),
	)

	// Persist to disk for important entries
	if cm.shouldPersist(key) {
		go cm.persistToDisk(key, entry)
	}

	return nil
}

// Delete removes an entry from cache
func (cm *CacheManager) Delete(ctx context.Context, key string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	delete(cm.entries, key)

	cm.logger.Debug(ctx, "cache_entry_deleted",
		observability.F("key", key),
	)

	// Remove from disk if exists
	diskPath := filepath.Join(cm.cacheDir, fmt.Sprintf("%s.json", key))
	os.Remove(diskPath)
}

// Clear removes all entries from cache
func (cm *CacheManager) Clear(ctx context.Context) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	entryCount := len(cm.entries)
	cm.entries = make(map[string]*CacheEntry)

	cm.logger.Info(ctx, "cache_cleared",
		observability.F("entries_removed", entryCount),
	)

	// Clear disk cache
	os.RemoveAll(cm.cacheDir)
	os.MkdirAll(cm.cacheDir, 0755)
}

// GetStats returns cache statistics
func (cm *CacheManager) GetStats() *CacheStats {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	totalSize := int64(0)
	totalAccess := int64(0)

	for _, entry := range cm.entries {
		totalSize += entry.Size
		totalAccess += entry.AccessCount
	}

	hitRate := float64(0)
	if totalAccess > 0 {
		hitRate = float64(totalAccess) / float64(totalAccess) * 100
	}

	return &CacheStats{
		TotalEntries: int64(len(cm.entries)),
		HitCount:     totalAccess,
		HitRate:      hitRate,
		TotalSize:    totalSize,
		LastCleanup:  time.Now(), // This would track actual cleanup time
	}
}

// Invalidate removes entries by tag
func (cm *CacheManager) Invalidate(ctx context.Context, tag string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	keysToDelete := make([]string, 0)

	for key, entry := range cm.entries {
		for _, entryTag := range entry.Tags {
			if entryTag == tag {
				keysToDelete = append(keysToDelete, key)
				break
			}
		}
	}

	for _, key := range keysToDelete {
		delete(cm.entries, key)
	}

	cm.logger.Info(ctx, "cache_invalidated_by_tag",
		observability.F("tag", tag),
		observability.F("entries_removed", len(keysToDelete)),
	)
}

// Warmup preloads cache with common data
func (cm *CacheManager) Warmup(ctx context.Context) error {
	cm.logger.Info(ctx, "starting_cache_warmup")

	// Common cache keys to warmup
	warmupKeys := []string{
		"git_config_global",
		"ssh_agent_status",
		"github_user_info",
		"account_list",
	}

	for _, key := range warmupKeys {
		// This would call the appropriate service to populate cache
		cm.logger.Debug(ctx, "warming_up_cache_key",
			observability.F("key", key),
		)
	}

	cm.logger.Info(ctx, "cache_warmup_completed")
	return nil
}

// startCleanupRoutine starts the background cleanup routine
func (cm *CacheManager) startCleanupRoutine() {
	cm.cleanupTicker = time.NewTicker(5 * time.Minute)

	go func() {
		for {
			select {
			case <-cm.cleanupTicker.C:
				cm.cleanup()
			case <-cm.stopChan:
				cm.cleanupTicker.Stop()
				return
			}
		}
	}()
}

// cleanup removes expired entries
func (cm *CacheManager) cleanup() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	now := time.Now()
	keysToDelete := make([]string, 0)

	for key, entry := range cm.entries {
		if now.After(entry.ExpiresAt) {
			keysToDelete = append(keysToDelete, key)
		}
	}

	for _, key := range keysToDelete {
		delete(cm.entries, key)
	}

	if len(keysToDelete) > 0 {
		cm.logger.Debug(context.Background(), "cache_cleanup_completed",
			observability.F("entries_removed", len(keysToDelete)),
		)
	}
}

// evictLRU evicts least recently used entries
func (cm *CacheManager) evictLRU() {
	if len(cm.entries) <= cm.maxEntries {
		return
	}

	// Find the entry with the oldest last access time
	var oldestKey string
	var oldestTime time.Time = time.Now()

	for key, entry := range cm.entries {
		if entry.LastAccess.Before(oldestTime) {
			oldestTime = entry.LastAccess
			oldestKey = key
		}
	}

	if oldestKey != "" {
		delete(cm.entries, oldestKey)
	}
}

// estimateSize estimates the memory size of a value
func (cm *CacheManager) estimateSize(value interface{}) int64 {
	// Simple size estimation - in a real implementation, you'd use reflection
	// or a more sophisticated method
	data, err := json.Marshal(value)
	if err != nil {
		return 0
	}
	return int64(len(data))
}

// shouldPersist determines if an entry should be persisted to disk
func (cm *CacheManager) shouldPersist(key string) bool {
	// Persist important configuration and account data
	persistentKeys := []string{
		"account_",
		"config_",
		"ssh_keys_",
		"git_config_",
	}

	for _, prefix := range persistentKeys {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			return true
		}
	}

	return false
}

// persistToDisk saves an entry to disk
func (cm *CacheManager) persistToDisk(key string, entry *CacheEntry) {
	diskPath := filepath.Join(cm.cacheDir, fmt.Sprintf("%s.json", key))

	data, err := json.Marshal(entry)
	if err != nil {
		cm.logger.Error(context.Background(), "failed_to_marshal_cache_entry",
			observability.F("key", key),
			observability.F("error", err.Error()),
		)
		return
	}

	if err := os.WriteFile(diskPath, data, 0644); err != nil {
		cm.logger.Error(context.Background(), "failed_to_persist_cache_entry",
			observability.F("key", key),
			observability.F("path", diskPath),
			observability.F("error", err.Error()),
		)
	}
}

// loadFromDisk loads cache entries from disk
func (cm *CacheManager) loadFromDisk() {
	files, err := os.ReadDir(cm.cacheDir)
	if err != nil {
		return
	}

	loadedCount := 0
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			filePath := filepath.Join(cm.cacheDir, file.Name())
			if entry := cm.loadEntryFromFile(filePath); entry != nil {
				// Check if not expired
				if time.Now().Before(entry.ExpiresAt) {
					cm.entries[entry.Key] = entry
					loadedCount++
				} else {
					// Remove expired file
					os.Remove(filePath)
				}
			}
		}
	}

	if loadedCount > 0 {
		cm.logger.Info(context.Background(), "cache_loaded_from_disk",
			observability.F("entries_loaded", loadedCount),
		)
	}
}

// loadEntryFromFile loads a cache entry from a file
func (cm *CacheManager) loadEntryFromFile(filePath string) *CacheEntry {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil
	}

	return &entry
}

// logCacheHit logs a cache hit
func (cm *CacheManager) logCacheHit(ctx context.Context, key string) {
	cm.logger.Debug(ctx, "cache_hit",
		observability.F("key", key),
	)
}

// logCacheMiss logs a cache miss
func (cm *CacheManager) logCacheMiss(ctx context.Context, key string) {
	cm.logger.Debug(ctx, "cache_miss",
		observability.F("key", key),
	)
}

// logCacheExpired logs an expired cache entry
func (cm *CacheManager) logCacheExpired(ctx context.Context, key string) {
	cm.logger.Debug(ctx, "cache_expired",
		observability.F("key", key),
	)
}

// Stop stops the cache manager
func (cm *CacheManager) Stop() {
	close(cm.stopChan)
}

// GetMemoryUsage returns estimated memory usage
func (cm *CacheManager) GetMemoryUsage() int64 {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	totalSize := int64(0)
	for _, entry := range cm.entries {
		totalSize += entry.Size
	}

	return totalSize
}

// PrefetchAccounts preloads account data into cache
func (cm *CacheManager) PrefetchAccounts(ctx context.Context, accounts []string) error {
	cm.logger.Info(ctx, "prefetching_accounts",
		observability.F("account_count", len(accounts)),
	)

	for _, account := range accounts {
		// This would load account-specific data into cache
		cacheKey := fmt.Sprintf("account_%s", account)

		// Check if already cached
		if _, exists := cm.Get(ctx, cacheKey); !exists {
			// Load account data (this would call actual services)
			accountData := map[string]interface{}{
				"alias":     account,
				"loaded_at": time.Now(),
			}

			cm.Set(ctx, cacheKey, accountData, &CacheOptions{
				TTL:  60 * time.Minute,
				Tags: []string{"account", account},
			})
		}
	}

	return nil
}
