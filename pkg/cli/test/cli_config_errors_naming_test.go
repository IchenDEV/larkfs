package cli_test

import (
	"bytes"
	"context"
	stderrors "errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"

	"github.com/IchenDEV/larkfs/pkg/cli"
	"github.com/IchenDEV/larkfs/pkg/config"
	larkerrors "github.com/IchenDEV/larkfs/pkg/errors"
	"github.com/IchenDEV/larkfs/pkg/naming"
)

func TestCLIParsersAndExecutor(t *testing.T) {
	type item struct {
		Name string `json:"name"`
	}
	parsed, err := cli.ParseJSON[item]([]byte(`{"name":"one"}`))
	if err != nil || parsed.Name != "one" {
		t.Fatalf("ParseJSON() = %+v, %v", parsed, err)
	}
	items, err := cli.ParseNDJSON[item]([]byte("{\"name\":\"one\"}\n\n{\"name\":\"two\"}\n"))
	if err != nil || !reflect.DeepEqual([]item{{"one"}, {"two"}}, items) {
		t.Fatalf("ParseNDJSON() = %+v, %v", items, err)
	}
	var streamed []item
	err = cli.StreamNDJSON(bytes.NewBufferString("{\"name\":\"one\"}\n{\"name\":\"two\"}\n"), func(v item) error {
		streamed = append(streamed, v)
		return nil
	})
	if err != nil || len(streamed) != 2 {
		t.Fatalf("StreamNDJSON() = %+v, %v", streamed, err)
	}
	wrapped, err := cli.ParseWrappedData[item]([]byte(`{"data":{"name":"wrapped"}}`))
	if err != nil || wrapped.Name != "wrapped" {
		t.Fatalf("ParseWrappedData() = %+v, %v", wrapped, err)
	}
	if _, err := cli.ParseJSON[item]([]byte(`{`)); err == nil {
		t.Fatal("ParseJSON() expected invalid JSON error")
	}

	dir := t.TempDir()
	script := filepath.Join(dir, "lark-cli-test")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nprintf '%s' \"$*\"\n"), 0o755); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}
	exec, err := cli.NewExecutor(script)
	if err != nil {
		t.Fatalf("NewExecutor() error: %v", err)
	}
	out, err := exec.Run(context.Background(), "drive", "files")
	if err != nil || string(out) != "drive files" {
		t.Fatalf("Run() = %q, %v", out, err)
	}
	exec.SetMiddleware(func(ctx context.Context, fn func() ([]byte, error)) ([]byte, error) {
		out, err := fn()
		return append([]byte("mw:"), out...), err
	})
	out, err = exec.RunJSON(context.Background(), "drive")
	if err != nil || !strings.Contains(string(out), "mw:drive --format json") {
		t.Fatalf("RunJSON() = %q, %v", out, err)
	}
	if _, err := cli.NewExecutor(filepath.Join(dir, "missing")); err == nil {
		t.Fatal("NewExecutor() expected missing binary error")
	}

	fail := filepath.Join(dir, "fail-cli")
	if err := os.WriteFile(fail, []byte("#!/bin/sh\necho 'permission denied' >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}
	exec, err = cli.NewExecutor(fail)
	if err != nil {
		t.Fatalf("NewExecutor() error: %v", err)
	}
	if _, err := exec.Run(context.Background(), "drive"); !stderrors.Is(err, cli.ErrForbidden) {
		t.Fatalf("Run() error = %v, want ErrForbidden", err)
	}
	t.Setenv("PATH", dir)
	link := filepath.Join(dir, "lark")
	if err := os.WriteFile(link, []byte("#!/bin/sh\nprintf ok\n"), 0o755); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}
	found, err := cli.FindLarkCLI("")
	if err != nil || found != link {
		t.Fatalf("FindLarkCLI() = %q, %v", found, err)
	}
}

func TestConfigErrorsAndNamingBlackbox(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := config.MountConfig{Mountpoint: "mnt"}
	if err := cfg.Resolve(); err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}
	if cfg.CacheDir != filepath.Join(home, ".larkfs", "cache") {
		t.Fatalf("CacheDir = %q", cfg.CacheDir)
	}
	if cfg.LogFile != filepath.Join(home, ".larkfs", "larkfs.log") {
		t.Fatalf("LogFile = %q", cfg.LogFile)
	}
	if !filepath.IsAbs(cfg.Mountpoint) {
		t.Fatalf("Mountpoint should be absolute, got %q", cfg.Mountpoint)
	}
	if defaults := cfg.EnabledDomains(); len(defaults) == 0 || defaults[len(defaults)-1] != "_system" {
		t.Fatalf("EnabledDomains defaults = %#v", defaults)
	}
	cfg.Domains = "drive,wiki"
	if got := cfg.EnabledDomains(); len(got) != 2 || got[0] != "drive" || got[1] != "wiki" {
		t.Fatalf("EnabledDomains custom = %#v", got)
	}
	if config.BaseDir() != filepath.Join(home, ".larkfs") {
		t.Fatalf("BaseDir() = %q", config.BaseDir())
	}
	if config.MountsDir() != filepath.Join(home, ".larkfs", "mounts") {
		t.Fatalf("MountsDir() = %q", config.MountsDir())
	}

	if got := larkerrors.ToErrno(cli.ErrForbidden); got != syscall.EACCES {
		t.Fatalf("ToErrno() = %v, want EACCES", got)
	}
	failPath := filepath.Join(t.TempDir(), "fail")
	if err := os.WriteFile(failPath, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("write fail script: %v", err)
	}
	recovery := larkerrors.NewAuthRecovery(failPath)
	if err := recovery.HandleError(context.Background(), cli.ErrAuthExpired); err != cli.ErrAuthExpired || !recovery.IsDegraded() {
		t.Fatalf("failed auth recovery error=%v degraded=%v", err, recovery.IsDegraded())
	}

	dir := t.TempDir()
	resolver := naming.NewResolver(dir)
	result := resolver.ResolveNames([]naming.NameEntry{{Name: "doc.md", Token: "token1234567"}})
	if result["token1234567"] != "doc.md" {
		t.Fatalf("ResolveNames() = %#v", result)
	}
	if token, ok := resolver.TokenForName("doc.md"); !ok || token != "token1234567" {
		t.Fatalf("TokenForName() = %q, %v", token, ok)
	}
	if _, err := os.Stat(filepath.Join(dir, "namemap.json")); err != nil {
		t.Fatalf("expected persisted map: %v", err)
	}
	reloaded := naming.NewResolver(dir)
	if token, ok := reloaded.TokenForName("doc.md"); !ok || token != "token1234567" {
		t.Fatalf("reloaded TokenForName() = %q, %v", token, ok)
	}
}
