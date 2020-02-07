package ttlcache

import (
	"sync"
	"time"
)

// Cache is a synchronised map of items that auto-expire once stale
type Cache struct {
	mutex sync.RWMutex
	ttl   time.Duration
	items map[string]*Item
}

// Set is a thread-safe way to add new items to the map
func (cache *Cache) Set(key string, data interface{}) {
	cache.SetWithTTL(key, data, cache.ttl)
}

// SetWithTTL allows adding items with a custom TTL
func (cache *Cache) SetWithTTL(key string, data interface{}, ttl time.Duration) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	item := &Item{data: data}
	item.touch(ttl)
	cache.items[key] = item
}

// Delete allows items to be removed from the cache, irrespective of the TTL
func (cache *Cache) Delete(key string) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	delete(cache.items, key)
}

// Get is a thread-safe way to lookup items
// Every lookup, also touches the item, hence extending it's life
func (cache *Cache) Read(key string) (data interface{}, found bool) {
	return cache.Get(key, true)
}

// Get is useful for non-LRU caches. Like for DNS results
func (cache *Cache) Get(key string, shouldTouch bool) (data interface{}, found bool) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	item, exists := cache.items[key]
	if !exists || item.expired() {
		data = ""
		found = false
	} else if shouldTouch {
		item.touch(cache.ttl)
		data = item.data
		found = true
	}

	return
}

// Count returns the number of items in the cache
// (helpful for tracking memory leaks)
func (cache *Cache) Count() int {
	cache.mutex.RLock()
	count := len(cache.items)
	cache.mutex.RUnlock()
	return count
}

func (cache *Cache) cleanup() {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	for key, item := range cache.items {
		if item.expired() {
			delete(cache.items, key)
		}
	}
}

// Run a timer that cleans up expired items every second
func (cache *Cache) startCleanupTimer() {
	duration := cache.ttl
	if duration < time.Second {
		duration = time.Second
	}
	ticker := time.Tick(duration)
	go (func() {
		for {
			select {
			case <-ticker:
				cache.cleanup()
			}
		}
	})()
}

// NewCache is a helper to create instance of the Cache struct
func NewCache(duration time.Duration) *Cache {
	cache := &Cache{
		ttl:   duration,
		items: map[string]*Item{},
	}
	cache.startCleanupTimer()
	return cache
}
