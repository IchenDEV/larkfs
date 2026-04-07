package cache

import (
	"os"
	"testing"
)

func TestContentCacheBasic(t *testing.T) {
	dir, err := os.MkdirTemp("", "larkfs-cache-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	c, err := NewContentCache(dir, 1024)
	if err != nil {
		t.Fatal(err)
	}

	data := []byte("hello world")
	if err := c.Set("key1", data); err != nil {
		t.Fatal(err)
	}

	got, ok := c.Get("key1")
	if !ok {
		t.Fatal("expected key1 to be cached")
	}
	if string(got) != "hello world" {
		t.Errorf("got %q, want %q", string(got), "hello world")
	}
}

func TestContentCacheEviction(t *testing.T) {
	dir, err := os.MkdirTemp("", "larkfs-cache-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	c, err := NewContentCache(dir, 20)
	if err != nil {
		t.Fatal(err)
	}

	c.Set("key1", []byte("1234567890"))
	c.Set("key2", []byte("1234567890"))
	c.Set("key3", []byte("1234567890"))

	_, ok := c.Get("key1")
	if ok {
		t.Error("key1 should have been evicted")
	}

	_, ok = c.Get("key3")
	if !ok {
		t.Error("key3 should still be cached")
	}
}

func TestContentCacheInvalidate(t *testing.T) {
	dir, err := os.MkdirTemp("", "larkfs-cache-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	c, err := NewContentCache(dir, 1024)
	if err != nil {
		t.Fatal(err)
	}

	c.Set("key1", []byte("data"))
	c.Invalidate("key1")

	_, ok := c.Get("key1")
	if ok {
		t.Error("key1 should be invalidated")
	}
}
