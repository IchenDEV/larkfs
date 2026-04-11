package daemon_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IchenDEV/larkfs/pkg/config"
	"github.com/IchenDEV/larkfs/pkg/daemon"
)

func TestDaemonPIDLifecycleBlackbox(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	mountpoint := filepath.Join(home, "mnt")
	if err := daemon.WritePIDFile(mountpoint, "fuse", []string{"drive"}); err != nil {
		t.Fatalf("WritePIDFile() error: %v", err)
	}
	mounts, err := daemon.ListMounts()
	if err != nil || len(mounts) != 1 {
		t.Fatalf("ListMounts() = %+v, %v", mounts, err)
	}
	info := mounts[0]
	if info.Mountpoint != mountpoint || info.Backend != "fuse" || info.Status() != "healthy" || info.Uptime() == "" {
		t.Fatalf("unexpected PIDInfo: %+v", info)
	}
	path := pidFilePathForTest(mountpoint)
	read, err := daemon.ReadPIDFile(path)
	if err != nil || read.Mountpoint != mountpoint {
		t.Fatalf("ReadPIDFile() = %+v, %v", read, err)
	}
	daemon.RemovePIDFile(mountpoint)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("PID file should be removed, stat err=%v", err)
	}
}

func TestDaemonUnmountStalePIDFileBlackbox(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	oldArgs := os.Args
	os.Args = []string{"larkfs", daemon.DaemonChildFlag}
	t.Cleanup(func() { os.Args = oldArgs })
	if !daemon.IsDaemonChild() {
		t.Fatal("IsDaemonChild() should detect flag")
	}

	mountpoint := filepath.Join(home, "mnt")
	writeStalePIDFile(t, mountpoint)
	if err := daemon.CheckExistingMount(mountpoint); err != nil {
		t.Fatalf("CheckExistingMount() error: %v", err)
	}
	if _, err := os.Stat(pidFilePathForTest(mountpoint)); !os.IsNotExist(err) {
		t.Fatalf("stale PID file should be removed, stat err=%v", err)
	}

	writeStalePIDFile(t, mountpoint)
	if err := daemon.Unmount(mountpoint, false, 1); err != nil {
		t.Fatalf("Unmount(stale) error: %v", err)
	}
	if _, err := os.Stat(pidFilePathForTest(mountpoint)); !os.IsNotExist(err) {
		t.Fatalf("stale PID file should be removed by Unmount, stat err=%v", err)
	}
	if err := daemon.CleanStaleMounts(); err != nil {
		t.Fatalf("CleanStaleMounts() error: %v", err)
	}
	if err := daemon.UnmountAll(false, 1); err != nil {
		t.Fatalf("UnmountAll() error: %v", err)
	}
	if got := (daemon.PIDInfo{StartedAt: time.Now().Add(-90 * time.Minute)}).Uptime(); !strings.Contains(got, "1h") {
		t.Fatalf("Uptime(hour) = %q", got)
	}
	if got := (daemon.PIDInfo{StartedAt: time.Now().Add(-2 * time.Minute)}).Uptime(); !strings.Contains(got, "2m") {
		t.Fatalf("Uptime(minute) = %q", got)
	}
	healthScript := filepath.Join(home, "health")
	if err := os.WriteFile(healthScript, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write health script: %v", err)
	}
	checker := daemon.NewHealthChecker(healthScript, time.Nanosecond)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	checker.Run(ctx)
}

func writeStalePIDFile(t *testing.T, mountpoint string) {
	t.Helper()
	started := time.Now()
	data := []byte(fmt.Sprintf(`{"pid":-1,"mountpoint":%q,"backend":"fuse","started_at":%q}`, mountpoint, started.Format(time.RFC3339Nano)))
	if err := os.MkdirAll(config.MountsDir(), 0o755); err != nil {
		t.Fatalf("MkdirAll() error: %v", err)
	}
	if err := os.WriteFile(pidFilePathForTest(mountpoint), data, 0o644); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}
}

func pidFilePathForTest(mountpoint string) string {
	h := sha256.Sum256([]byte(mountpoint))
	name := hex.EncodeToString(h[:8]) + ".pid"
	return filepath.Join(config.MountsDir(), name)
}
