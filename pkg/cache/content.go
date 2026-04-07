package cache

import (
	"container/list"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sync"
)

type ContentCache struct {
	mu       sync.Mutex
	dir      string
	maxBytes int64
	curBytes int64
	lru      *list.List
	items    map[string]*list.Element
}

type lruEntry struct {
	key  string
	size int64
}

func NewContentCache(dir string, maxBytes int64) (*ContentCache, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &ContentCache{
		dir:      dir,
		maxBytes: maxBytes,
		lru:      list.New(),
		items:    make(map[string]*list.Element),
	}, nil
}

func (c *ContentCache) Get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.items[key]
	if !ok {
		return nil, false
	}
	c.lru.MoveToFront(elem)

	data, err := os.ReadFile(c.filePath(key))
	if err != nil {
		c.removeElement(elem)
		return nil, false
	}
	return data, true
}

func (c *ContentCache) Set(key string, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.removeElement(elem)
	}

	for c.curBytes+int64(len(data)) > c.maxBytes && c.lru.Len() > 0 {
		c.evictOldest()
	}

	path := c.filePath(key)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}

	entry := &lruEntry{key: key, size: int64(len(data))}
	elem := c.lru.PushFront(entry)
	c.items[key] = elem
	c.curBytes += entry.size
	return nil
}

func (c *ContentCache) Invalidate(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if elem, ok := c.items[key]; ok {
		c.removeElement(elem)
	}
}

func (c *ContentCache) Size() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.curBytes
}

func (c *ContentCache) filePath(key string) string {
	h := sha256.Sum256([]byte(key))
	return filepath.Join(c.dir, hex.EncodeToString(h[:])+".cache")
}

func (c *ContentCache) evictOldest() {
	elem := c.lru.Back()
	if elem == nil {
		return
	}
	c.removeElement(elem)
}

func (c *ContentCache) removeElement(elem *list.Element) {
	entry := elem.Value.(*lruEntry)
	c.lru.Remove(elem)
	delete(c.items, entry.key)
	c.curBytes -= entry.size
	os.Remove(c.filePath(entry.key))
}
