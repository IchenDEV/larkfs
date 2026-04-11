package larkfs_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersionCommandBlackbox(t *testing.T) {
	out, err := runLarkFS(t, nil, "version")
	if err != nil {
		t.Fatalf("larkfs version error: %v\n%s", err, out)
	}
	if !strings.Contains(out, "larkfs dev") || !strings.Contains(out, "commit: unknown") {
		t.Fatalf("version output = %q", out)
	}
}

func TestStatusCommandNoMountsBlackbox(t *testing.T) {
	env := map[string]string{"HOME": t.TempDir()}
	out, err := runLarkFS(t, env, "status")
	if err != nil {
		t.Fatalf("larkfs status error: %v\n%s", err, out)
	}
	if !strings.Contains(out, "No active mounts.") {
		t.Fatalf("status output = %q", out)
	}
	out, err = runLarkFS(t, env, "status", "--json")
	if err != nil {
		t.Fatalf("larkfs status --json error: %v\n%s", err, out)
	}
	if !strings.Contains(out, "No active mounts.") {
		t.Fatalf("status --json output = %q", out)
	}
}

func TestUnmountCommandRequiresMountpointBlackbox(t *testing.T) {
	out, err := runLarkFS(t, nil, "unmount")
	if err == nil {
		t.Fatalf("larkfs unmount unexpectedly succeeded:\n%s", out)
	}
	if !strings.Contains(out, "mountpoint required") {
		t.Fatalf("unmount output = %q", out)
	}
}

func TestInitAlreadyConfiguredAndLoggedInBlackbox(t *testing.T) {
	dir := t.TempDir()
	writeFakeLarkCLI(t, dir, `#!/bin/sh
case "$1 $2" in
  "config show") printf '{"appId":"app"}' ;;
  "auth status") printf '{"tokenStatus":"valid"}' ;;
  *) printf '{}' ;;
esac
`)
	out, err := runLarkFS(t, map[string]string{"PATH": dir}, "init")
	if err != nil {
		t.Fatalf("larkfs init error: %v\n%s", err, out)
	}
	if !strings.Contains(out, "lark-cli already configured") || !strings.Contains(out, "User already logged in") {
		t.Fatalf("init output = %q", out)
	}
}

func TestInitConfiguresWhenMissingBlackbox(t *testing.T) {
	dir := t.TempDir()
	writeFakeLarkCLI(t, dir, `#!/bin/sh
case "$1 $2" in
  "config show") printf '{"appId":""}' ;;
  "config init") exit 0 ;;
  "auth status") printf '{"tokenStatus":"valid"}' ;;
  *) printf '{}' ;;
esac
`)
	out, err := runLarkFS(t, map[string]string{"PATH": dir}, "init")
	if err != nil {
		t.Fatalf("larkfs init error: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Configuration complete") {
		t.Fatalf("init output = %q", out)
	}
}

func TestDoctorMissingCLIBlackbox(t *testing.T) {
	out, err := runLarkFS(t, map[string]string{"PATH": t.TempDir()}, "doctor")
	if err == nil {
		t.Fatalf("larkfs doctor unexpectedly succeeded:\n%s", out)
	}
	if !strings.Contains(out, "lark-cli: not found") {
		t.Fatalf("doctor output = %q", out)
	}
}

func TestDoctorWithFakeCLIBlackbox(t *testing.T) {
	dir := t.TempDir()
	writeFakeLarkCLI(t, dir, `#!/bin/sh
case "$1 $2" in
  "auth status") printf '{"userName":"Alice","identity":"user"}' ;;
  "doctor ") printf '{"ok":true,"checks":[{"name":"network","status":"pass","message":"ok"}]}' ;;
  *) printf '{"ok":true,"checks":[{"name":"network","status":"pass","message":"ok"}]}' ;;
esac
`)
	out, _ := runLarkFS(t, map[string]string{"PATH": dir}, "doctor")
	if !strings.Contains(out, "lark-cli auth: logged in as Alice") {
		t.Fatalf("doctor output = %q", out)
	}
}

func TestMountAndServeFastFailuresBlackbox(t *testing.T) {
	home := t.TempDir()
	out, err := runLarkFS(t, map[string]string{"HOME": home}, "mount", filepath.Join(home, "mnt"), "--lark-cli", filepath.Join(home, "missing-lark-cli"))
	if err == nil {
		t.Fatalf("larkfs mount unexpectedly succeeded:\n%s", out)
	}
	if !strings.Contains(out, "failed to create FUSE server") {
		t.Fatalf("mount output = %q", out)
	}

	cliPath := filepath.Join(home, "lark-cli")
	if err := os.WriteFile(cliPath, []byte("#!/bin/sh\nprintf '{}'\n"), 0o755); err != nil {
		t.Fatalf("write fake lark-cli: %v", err)
	}
	out, err = runLarkFS(t, map[string]string{"HOME": home}, "serve", "--addr", "localhost", "--port", "-1", "--domains", "contact", "--lark-cli", cliPath)
	if err == nil {
		t.Fatalf("larkfs serve unexpectedly succeeded:\n%s", out)
	}
}

func runLarkFS(t *testing.T, overrides map[string]string, args ...string) (string, error) {
	t.Helper()
	goPath, err := exec.LookPath("go")
	if err != nil {
		t.Fatalf("find go: %v", err)
	}
	env := map[string]string{}
	for key, value := range overrides {
		env[key] = value
	}
	if _, ok := env["GOMODCACHE"]; !ok {
		env["GOMODCACHE"] = stableCacheDir(t, "larkfs-test-gomodcache", os.Getenv("GOMODCACHE"))
	}
	if _, ok := env["GOCACHE"]; !ok {
		env["GOCACHE"] = stableCacheDir(t, "larkfs-test-gocache", os.Getenv("GOCACHE"))
	}
	cmdArgs := append([]string{"run", ".."}, args...)
	cmd := exec.Command(goPath, cmdArgs...)
	cmd.Dir = "."
	cmd.Env = mergedEnv(env)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err = cmd.Run()
	return out.String(), err
}

func stableCacheDir(t *testing.T, name, existing string) string {
	t.Helper()
	if existing != "" {
		return existing
	}
	dir := filepath.Join(os.TempDir(), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create %s: %v", dir, err)
	}
	return dir
}

func mergedEnv(overrides map[string]string) []string {
	env := os.Environ()
	for key, value := range overrides {
		prefix := key + "="
		replaced := false
		for i, entry := range env {
			if strings.HasPrefix(entry, prefix) {
				env[i] = prefix + value
				replaced = true
				break
			}
		}
		if !replaced {
			env = append(env, prefix+value)
		}
	}
	return env
}

func writeFakeLarkCLI(t *testing.T, dir, body string) {
	t.Helper()
	path := filepath.Join(dir, "lark-cli")
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("write fake lark-cli: %v", err)
	}
}
