package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IchenDEV/larkfs/pkg/config"
)

func TestCommandConstructors(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
	}{
		{"mount", newMountCmd().Use},
		{"serve", newServeCmd().Use},
		{"status", newStatusCmd().Use},
		{"unmount", newUnmountCmd().Use},
		{"doctor", newDoctorCmd().Use},
		{"init", newInitCmd().Use},
		{"version", newVersionCmd().Use},
	}
	for _, tt := range tests {
		if !strings.Contains(tt.cmd, tt.name) {
			t.Fatalf("command use %q should contain %q", tt.cmd, tt.name)
		}
	}
}

func TestVersionCommand(t *testing.T) {
	oldVersion, oldCommit, oldDate := version, commit, date
	version, commit, date = "v1", "abc", "today"
	t.Cleanup(func() { version, commit, date = oldVersion, oldCommit, oldDate })

	cmd := newVersionCmd()
	var out bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	cmd.Run(cmd, nil)
	_ = w.Close()
	os.Stdout = oldStdout
	_, _ = out.ReadFrom(r)
	if !strings.Contains(out.String(), "v1") || !strings.Contains(out.String(), "abc") {
		t.Fatalf("version output = %q", out.String())
	}
}

func TestRunInitAlreadyConfiguredAndLoggedIn(t *testing.T) {
	dir := t.TempDir()
	writeFakeLarkCLI(t, dir, `#!/bin/sh
case "$1 $2" in
  "config show") printf '{"appId":"app"}' ;;
  "auth status") printf '{"tokenStatus":"valid"}' ;;
  *) printf '{}' ;;
esac
`)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	if err := runInit(); err != nil {
		t.Fatalf("runInit() error: %v", err)
	}
}

func TestRunInitConfiguresWhenMissing(t *testing.T) {
	dir := t.TempDir()
	writeFakeLarkCLI(t, dir, `#!/bin/sh
case "$1 $2" in
  "config show") printf '{"appId":""}' ;;
  "config init") exit 0 ;;
  "auth status") printf '{"tokenStatus":"valid"}' ;;
  *) printf '{}' ;;
esac
`)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	if err := runInit(); err != nil {
		t.Fatalf("runInit() error: %v", err)
	}
}

func TestRunDoctorWithFakeCLI(t *testing.T) {
	dir := t.TempDir()
	writeFakeLarkCLI(t, dir, `#!/bin/sh
case "$1 $2" in
  "auth status") printf '{"userName":"Alice","identity":"user"}' ;;
  "doctor ") printf '{"ok":true,"checks":[{"name":"network","status":"pass","message":"ok"}]}' ;;
  *) printf '{"ok":true,"checks":[{"name":"network","status":"pass","message":"ok"}]}' ;;
esac
`)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	if err := runDoctor(); err != nil {
		t.Fatalf("runDoctor() error: %v", err)
	}
}

func TestRunDoctorMissingCLI(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if err := runDoctor(); err == nil {
		t.Fatal("runDoctor() expected missing CLI error")
	}
}

func TestStatusCommandNoMounts(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cmd := newStatusCmd()
	if err := cmd.Execute(); err != nil {
		t.Fatalf("status command error: %v", err)
	}
	cmd = newStatusCmd()
	cmd.SetArgs([]string{"--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("status --json command error: %v", err)
	}
}

func TestUnmountCommandRequiresMountpoint(t *testing.T) {
	cmd := newUnmountCmd()
	if err := cmd.Execute(); err == nil {
		t.Fatal("unmount without args should fail")
	}
}

func TestRunMountFastFailure(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	err := runMount(config.MountConfig{
		Mountpoint:  filepath.Join(home, "mnt"),
		LarkCLIPath: filepath.Join(home, "missing-lark-cli"),
		MetadataTTL: 60,
	})
	if err == nil || !strings.Contains(err.Error(), "failed to create FUSE server") {
		t.Fatalf("runMount() error = %v", err)
	}
}

func TestRunServeFastFailure(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cliPath := filepath.Join(home, "lark-cli")
	if err := os.WriteFile(cliPath, []byte("#!/bin/sh\nprintf '{}'\n"), 0o755); err != nil {
		t.Fatalf("write fake lark-cli: %v", err)
	}
	err := runServe(config.ServeConfig{
		Addr:        "localhost",
		Port:        -1,
		Domains:     "contact",
		LarkCLIPath: cliPath,
	})
	if err == nil {
		t.Fatal("runServe() expected invalid address error")
	}
}

func writeFakeLarkCLI(t *testing.T, dir, body string) {
	t.Helper()
	path := filepath.Join(dir, "lark-cli")
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("write fake lark-cli: %v", err)
	}
}
