package cache_test

import (
	"testing"
	"time"

	"github.com/IchenDEV/larkfs/pkg/cache"
)

func TestContentCacheSizeEvictionAndInvalidateBlackbox(t *testing.T) {
	c, err := cache.NewContentCache(t.TempDir(), 5)
	if err != nil {
		t.Fatalf("NewContentCache() error: %v", err)
	}
	if c.Size() != 0 {
		t.Fatalf("initial Size() = %d", c.Size())
	}
	if err := c.Set("a", []byte("123")); err != nil {
		t.Fatalf("Set(a) error: %v", err)
	}
	if got := c.Size(); got != 3 {
		t.Fatalf("Size() = %d, want 3", got)
	}
	if data, ok := c.Get("a"); !ok || string(data) != "123" {
		t.Fatalf("Get(a) = %q, %v", data, ok)
	}
	if err := c.Set("b", []byte("4567")); err != nil {
		t.Fatalf("Set(b) error: %v", err)
	}
	if _, ok := c.Get("a"); ok {
		t.Fatal("expected a to be evicted")
	}
	if got := c.Size(); got != 4 {
		t.Fatalf("Size() after eviction = %d, want 4", got)
	}
	c.Invalidate("b")
	if got := c.Size(); got != 0 {
		t.Fatalf("Size() after invalidate = %d, want 0", got)
	}
}

func TestMetadataCacheInvalidatePrefixBlackbox(t *testing.T) {
	c := cache.NewMetadataCache(time.Minute)
	t.Cleanup(c.Close)
	c.Set("drive:a", 1)
	c.Set("drive:b", 2)
	c.Set("wiki:a", 3)
	c.InvalidatePrefix("drive:")
	if _, ok := c.Get("drive:a"); ok {
		t.Fatal("drive:a should be invalidated")
	}
	if _, ok := c.Get("drive:b"); ok {
		t.Fatal("drive:b should be invalidated")
	}
	if got, ok := c.Get("wiki:a"); !ok || got.(int) != 3 {
		t.Fatalf("wiki:a = %v, %v", got, ok)
	}
}
