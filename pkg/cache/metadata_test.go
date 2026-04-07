package cache

import (
	"testing"
	"time"
)

func TestMetadataCacheGetSet(t *testing.T) {
	c := NewMetadataCache(1 * time.Second)
	defer c.Close()

	c.Set("key1", "value1")
	got, ok := c.Get("key1")
	if !ok || got != "value1" {
		t.Errorf("Get(key1) = %v, %v; want value1, true", got, ok)
	}

	_, ok = c.Get("nonexistent")
	if ok {
		t.Error("Get(nonexistent) should return false")
	}
}

func TestMetadataCacheTTL(t *testing.T) {
	c := NewMetadataCache(50 * time.Millisecond)
	defer c.Close()

	c.Set("key1", "value1")
	time.Sleep(100 * time.Millisecond)

	_, ok := c.Get("key1")
	if ok {
		t.Error("Get(key1) should return false after TTL expiry")
	}
}

func TestMetadataCacheInvalidate(t *testing.T) {
	c := NewMetadataCache(1 * time.Second)
	defer c.Close()

	c.Set("key1", "value1")
	c.Invalidate("key1")

	_, ok := c.Get("key1")
	if ok {
		t.Error("Get(key1) should return false after Invalidate")
	}
}

func TestMetadataCacheInvalidatePrefix(t *testing.T) {
	c := NewMetadataCache(1 * time.Second)
	defer c.Close()

	c.Set("drive:list:abc", "v1")
	c.Set("drive:list:def", "v2")
	c.Set("wiki:spaces", "v3")

	c.InvalidatePrefix("drive:")

	_, ok1 := c.Get("drive:list:abc")
	_, ok2 := c.Get("drive:list:def")
	_, ok3 := c.Get("wiki:spaces")

	if ok1 || ok2 {
		t.Error("drive: keys should be invalidated")
	}
	if !ok3 {
		t.Error("wiki:spaces should still exist")
	}
}
