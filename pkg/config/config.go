package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type MountConfig struct {
	Mountpoint  string
	Daemon      bool
	CacheDir    string
	CacheSize   string
	MetadataTTL int
	LogFile     string
	LogLevel    string
	ReadOnly    bool
	Domains     string
	LarkCLIPath string
}

type ServeConfig struct {
	Port        int
	Addr        string
	Daemon      bool
	LogLevel    string
	ReadOnly    bool
	Domains     string
	LarkCLIPath string
}

func (c *MountConfig) Resolve() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}

	baseDir := filepath.Join(home, ".larkfs")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return fmt.Errorf("create base dir: %w", err)
	}

	if c.CacheDir == "" {
		c.CacheDir = filepath.Join(baseDir, "cache")
	}
	if err := os.MkdirAll(c.CacheDir, 0o755); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}

	if c.LogFile == "" {
		c.LogFile = filepath.Join(baseDir, "larkfs.log")
	}

	mountsDir := filepath.Join(baseDir, "mounts")
	if err := os.MkdirAll(mountsDir, 0o755); err != nil {
		return fmt.Errorf("create mounts dir: %w", err)
	}

	if c.Mountpoint != "" {
		abs, err := filepath.Abs(c.Mountpoint)
		if err != nil {
			return fmt.Errorf("resolve mountpoint: %w", err)
		}
		c.Mountpoint = abs
	}

	return nil
}

func (c *MountConfig) EnabledDomains() []string {
	if c.Domains == "" {
		return []string{"drive", "wiki", "im", "calendar", "tasks", "mail", "meetings"}
	}
	return strings.Split(c.Domains, ",")
}

func BaseDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".larkfs")
}

func MountsDir() string {
	return filepath.Join(BaseDir(), "mounts")
}
