package cache

import (
	"sync"
	"time"
)

// SimpleCache provides a basic in-memory cache with TTL support
type SimpleCache struct {
	items map[string]*CacheItem
	mutex sync.RWMutex
	ttl   time.Duration
}

// CacheItem represents a cached item with expiration
type CacheItem struct {
	Value     interface{}
	ExpiresAt time.Time
}

// NewSimpleCache creates a new simple cache with the specified TTL
func NewSimpleCache(ttl time.Duration) *SimpleCache {
	return &SimpleCache{
		items: make(map[string]*CacheItem),
		ttl:   ttl,
	}
}

// Get retrieves an item from the cache
func (c *SimpleCache) Get(key string) (interface{}, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return nil, false
	}

	// Check if item has expired
	if time.Now().After(item.ExpiresAt) {
		delete(c.items, key)
		return nil, false
	}

	return item.Value, true
}

// Set stores an item in the cache
func (c *SimpleCache) Set(key string, value interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.items[key] = &CacheItem{
		Value:     value,
		ExpiresAt: time.Now().Add(c.ttl),
	}
}

// Delete removes an item from the cache
func (c *SimpleCache) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.items, key)
}

// Clear removes all items from the cache
func (c *SimpleCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.items = make(map[string]*CacheItem)
}

// Size returns the number of items in the cache
func (c *SimpleCache) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	// Clean up expired items while we're at it
	now := time.Now()
	for key, item := range c.items {
		if now.After(item.ExpiresAt) {
			delete(c.items, key)
		}
	}

	return len(c.items)
}

// Keys returns all non-expired keys in the cache
func (c *SimpleCache) Keys() []string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	now := time.Now()
	var keys []string

	for key, item := range c.items {
		if now.Before(item.ExpiresAt) {
			keys = append(keys, key)
		}
	}

	return keys
}

// Global cache instance for GitPersona
var globalCache = NewSimpleCache(5 * time.Minute)

// GetGlobalCache returns the global cache instance
func GetGlobalCache() *SimpleCache {
	return globalCache
}

// Cache keys for common operations
const (
	ConfigKey           = "gitpersona:config"
	AccountsKey         = "gitpersona:accounts"
	CurrentAccountKey   = "gitpersona:current_account"
	GitConfigKey        = "gitpersona:git_config"
	SSHAgentStatusKey   = "gitpersona:ssh_agent_status"
	ProjectConfigPrefix = "gitpersona:project:"
)
