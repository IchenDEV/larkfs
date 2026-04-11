package mount_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/IchenDEV/larkfs/pkg/config"
	"github.com/IchenDEV/larkfs/pkg/mount"
)

func TestWebDAVServerServeAndCloseBlackbox(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cliPath := filepath.Join(home, "lark-cli")
	if err := os.WriteFile(cliPath, []byte("#!/bin/sh\nprintf '{}'\n"), 0o755); err != nil {
		t.Fatalf("write fake cli: %v", err)
	}
	server, err := mount.NewWebDAVServer(config.ServeConfig{
		LogLevel:    "error",
		Domains:     "contact,docs",
		LarkCLIPath: cliPath,
	})
	if err != nil {
		t.Fatalf("NewWebDAVServer() error: %v", err)
	}
	if err := server.Serve("127.0.0.1:-1"); err == nil {
		t.Fatal("Serve() expected invalid address error")
	}
	server.Close()
}

func TestFUSEServerMissingCLIBlackbox(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	_, err := mount.NewFUSEServer(config.MountConfig{
		Mountpoint:  filepath.Join(home, "mnt"),
		CacheDir:    filepath.Join(home, "cache"),
		LarkCLIPath: filepath.Join(home, "missing-lark-cli"),
		MetadataTTL: 60,
		Domains:     "contact",
	})
	if err == nil {
		t.Fatal("NewFUSEServer() expected missing cli error")
	}
}
