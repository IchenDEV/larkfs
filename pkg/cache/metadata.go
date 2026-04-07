package cache

import (
	"sync"
	"time"
)

type metaEntry struct {
	data      any
	expiresAt time.Time
}

type MetadataCache struct {
	mu       sync.RWMutex
	ttl      time.Duration
	m        map[string]metaEntry
	stop     chan struct{}
	closeOnce sync.Once
}

func NewMetadataCache(ttl time.Duration) *MetadataCache {
	c := &MetadataCache{
		ttl:  ttl,
		m:    make(map[string]metaEntry),
		stop: make(chan struct{}),
	}
	go c.evictLoop()
	return c
}

func (c *MetadataCache) Close() {
	c.closeOnce.Do(func() { close(c.stop) })
}

func (c *MetadataCache) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.m[key]
	if !ok || time.Now().After(e.expiresAt) {
		return nil, false
	}
	return e.data, true
}

func (c *MetadataCache) Set(key string, data any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[key] = metaEntry{data: data, expiresAt: time.Now().Add(c.ttl)}
}

func (c *MetadataCache) Invalidate(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.m, key)
}

func (c *MetadataCache) InvalidatePrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k := range c.m {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			delete(c.m, k)
		}
	}
}

func (c *MetadataCache) evictLoop() {
	ticker := time.NewTicker(c.ttl)
	defer ticker.Stop()
	for {
		select {
		case <-c.stop:
			return
		case <-ticker.C:
			c.mu.Lock()
			now := time.Now()
			for k, v := range c.m {
				if now.After(v.expiresAt) {
					delete(c.m, k)
				}
			}
			c.mu.Unlock()
		}
	}
}
