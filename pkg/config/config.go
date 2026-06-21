package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const DefaultCacheSize = "500MB"

var defaultDomains = []string{
	"drive",
	"wiki",
	"im",
	"calendar",
	"tasks",
	"mail",
	"meetings",
	"apps",
	"approval",
	"attendance",
	"base",
	"contact",
	"docs",
	"event",
	"markdown",
	"minutes",
	"note",
	"okr",
	"sheets",
	"slides",
	"vc",
	"whiteboard",
	"_system",
}

var DefaultDomainsValue = strings.Join(defaultDomains, ",")

func DefaultDomains() []string {
	return append([]string(nil), defaultDomains...)
}

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
	CacheDir    string
	CacheSize   string
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
		return DefaultDomains()
	}
	return strings.Split(c.Domains, ",")
}

func (c MountConfig) ContentCacheSizeBytes() (int64, error) {
	size := c.CacheSize
	if size == "" {
		size = DefaultCacheSize
	}
	return ParseByteSize(size)
}

func BaseDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".larkfs")
}

func MountsDir() string {
	return filepath.Join(BaseDir(), "mounts")
}

func ParseByteSize(raw string) (int64, error) {
	value := strings.ToUpper(strings.TrimSpace(raw))
	if value == "" {
		return 0, fmt.Errorf("cache size cannot be empty")
	}

	multiplier := int64(1)
	for _, unit := range []struct {
		Suffix string
		Bytes  int64
	}{
		{Suffix: "TB", Bytes: 1024 * 1024 * 1024 * 1024},
		{Suffix: "GB", Bytes: 1024 * 1024 * 1024},
		{Suffix: "MB", Bytes: 1024 * 1024},
		{Suffix: "KB", Bytes: 1024},
		{Suffix: "B", Bytes: 1},
	} {
		if strings.HasSuffix(value, unit.Suffix) {
			multiplier = unit.Bytes
			value = strings.TrimSpace(strings.TrimSuffix(value, unit.Suffix))
			break
		}
	}

	amount, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse cache size %q: %w", raw, err)
	}
	if amount <= 0 {
		return 0, fmt.Errorf("cache size must be positive")
	}
	return amount * multiplier, nil
}
