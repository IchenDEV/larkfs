package daemon

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/IchenDEV/larkfs/pkg/config"
)

const DaemonChildFlag = "--_daemon-child"

func IsDaemonChild() bool {
	for _, arg := range os.Args {
		if arg == DaemonChildFlag {
			return true
		}
	}
	return false
}

func ForkDaemon(identifier string, origArgs []string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable: %w", err)
	}

	args := filterArgs(origArgs[1:], "-d", "--daemon")
	args = append(args, DaemonChildFlag)

	cmd := exec.Command(exe, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("fork daemon: %w", err)
	}

	fmt.Printf("Started daemon (pid %d) for %s\n", cmd.Process.Pid, identifier)
	return nil
}

func filterArgs(args []string, remove ...string) []string {
	set := make(map[string]bool, len(remove))
	for _, r := range remove {
		set[r] = true
	}
	var out []string
	for _, a := range args {
		if !set[a] {
			out = append(out, a)
		}
	}
	return out
}

func CheckExistingMount(mountpoint string) error {
	path := pidFilePath(mountpoint)
	info, err := ReadPIDFile(path)
	if err != nil {
		return nil
	}
	if isProcessAlive(info.PID) {
		return fmt.Errorf("already mounted at %s (pid %d)", mountpoint, info.PID)
	}
	slog.Info("cleaning stale PID file", "mountpoint", mountpoint, "pid", info.PID)
	os.Remove(path)
	return nil
}

func isProcessAlive(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return p.Signal(syscall.Signal(0)) == nil
}

func Unmount(mountpoint string, force bool, timeout int) error {
	path := pidFilePath(mountpoint)
	info, err := ReadPIDFile(path)
	if err != nil {
		return forceUnmountFS(mountpoint)
	}

	if isProcessAlive(info.PID) {
		p, _ := os.FindProcess(info.PID)
		_ = p.Signal(syscall.SIGTERM)

		deadline := time.After(time.Duration(timeout) * time.Second)
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()

	waitLoop:
		for {
			select {
			case <-deadline:
				if force {
					_ = p.Signal(syscall.SIGKILL)
					_ = forceUnmountFS(mountpoint)
					break waitLoop
				}
				return fmt.Errorf("unmount timeout; use --force")
			case <-ticker.C:
				if !isProcessAlive(info.PID) {
					break waitLoop
				}
			}
		}
	}

	RemovePIDFile(mountpoint)
	fmt.Printf("Unmounted %s\n", mountpoint)
	return nil
}

func UnmountAll(force bool, timeout int) error {
	mounts, err := ListMounts()
	if err != nil {
		return err
	}
	for _, m := range mounts {
		if err := Unmount(m.Mountpoint, force, timeout); err != nil {
			slog.Error("failed to unmount", "mountpoint", m.Mountpoint, "error", err)
		}
	}
	return nil
}

func forceUnmountFS(mountpoint string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("umount", "-f", mountpoint).Run()
	default:
		if err := exec.Command("fusermount", "-uz", mountpoint).Run(); err != nil {
			return exec.Command("fusermount3", "-uz", mountpoint).Run()
		}
		return nil
	}
}

func ListMounts() ([]PIDInfo, error) {
	dir := config.MountsDir()
	entries, err := filepath.Glob(filepath.Join(dir, "*.pid"))
	if err != nil {
		return nil, err
	}

	var mounts []PIDInfo
	for _, path := range entries {
		info, err := ReadPIDFile(path)
		if err != nil {
			continue
		}
		mounts = append(mounts, *info)
	}
	return mounts, nil
}

func CleanStaleMounts() error {
	mounts, err := ListMounts()
	if err != nil {
		return err
	}
	for _, m := range mounts {
		if !isProcessAlive(m.PID) {
			slog.Info("cleaning stale mount", "mountpoint", m.Mountpoint, "pid", m.PID)
			_ = forceUnmountFS(m.Mountpoint)
			RemovePIDFile(m.Mountpoint)
		}
	}
	return nil
}
