package executor

import (
	"sync"
	"time"
)

type Cache struct {
	mu    sync.RWMutex
	items map[string]cacheItem
}

type cacheItem struct {
	data      []byte
	expiresAt time.Time
}

func NewCache() *Cache {
	return &Cache{items: make(map[string]cacheItem)}
}

func (c *Cache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	item, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(item.expiresAt) {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return nil, false
	}
	return item.data, true
}

func (c *Cache) Set(key string, data []byte, ttl time.Duration) {
	if key == "" {
		return
	}
	c.mu.Lock()
	c.items[key] = cacheItem{
		data:      data,
		expiresAt: time.Now().Add(ttl),
	}
	c.mu.Unlock()
}
