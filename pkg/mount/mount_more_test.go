package mount

import (
	"context"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IchenDEV/larkfs/pkg/cache"
	"github.com/IchenDEV/larkfs/pkg/config"
	"github.com/IchenDEV/larkfs/pkg/vfs"
)

func TestNewWebDAVServerBuildsMount(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cliPath := filepath.Join(home, "lark-cli")
	if err := os.WriteFile(cliPath, []byte("#!/bin/sh\nprintf '{}'\n"), 0o755); err != nil {
		t.Fatalf("write fake cli: %v", err)
	}
	server, err := NewWebDAVServer(config.ServeConfig{
		LogLevel:    "error",
		Domains:     "contact,docs",
		LarkCLIPath: cliPath,
	})
	if err != nil {
		t.Fatalf("NewWebDAVServer() error: %v", err)
	}
	if server.handler == nil || server.state == nil || server.state.ops == nil {
		t.Fatalf("server not wired: %+v", server)
	}
	server.Close()
}

func TestNewFUSEServerFastFailure(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	_, err := NewFUSEServer(config.MountConfig{
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

func newMountTestOps(runner *mockRunner) *vfs.Operations {
	return vfs.NewOperations(vfs.OperationsConfig{
		CLI:  runner,
		Tree: vfs.NewTree([]string{"contact", "docs"}),
		TTL:  time.Minute,
	})
}

func TestWebDAVFSStatReaddirReadWriteAndSeek(t *testing.T) {
	runner := &mockRunner{out: []byte(`{"data":[{"name":"Alice"}]}`)}
	ops := newMountTestOps(runner)
	fs := &webdavFS{ops: ops}

	info, err := fs.Stat(context.Background(), "/contact")
	if err != nil || !info.IsDir() {
		t.Fatalf("Stat(/contact) = %+v, %v", info, err)
	}
	dir, err := fs.OpenFile(context.Background(), "/contact", os.O_RDONLY, 0)
	if err != nil {
		t.Fatalf("OpenFile(dir) error: %v", err)
	}
	infos, err := dir.Readdir(2)
	if err != nil || len(infos) != 2 {
		t.Fatalf("Readdir(2) = %+v, %v", infos, err)
	}

	file, err := fs.OpenFile(context.Background(), "/contact/_queries/search-user.request.json", os.O_RDWR|os.O_TRUNC, 0o644)
	if err != nil {
		t.Fatalf("OpenFile(control request) error: %v", err)
	}
	if _, err := file.Write([]byte(`{"query":"Alice"}`)); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	if pos, err := file.Seek(0, io.SeekStart); err != nil || pos != 0 {
		t.Fatalf("Seek(start) = %d, %v", pos, err)
	}
	buf := make([]byte, 20)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("Read() error: %v", err)
	}
	if !strings.Contains(string(buf[:n]), "Alice") {
		t.Fatalf("Read() = %q", buf[:n])
	}
	if _, err := file.Seek(-1, io.SeekStart); err == nil {
		t.Fatal("Seek(negative) expected error")
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}
	if len(runner.lastArgs) == 0 {
		t.Fatal("expected Close() to execute query")
	}

	if err := fs.RemoveAll(context.Background(), "/contact/_queries/search-user.request.json"); err == nil {
		t.Fatal("RemoveAll(control node) expected unsupported error")
	}
	if err := fs.Rename(context.Background(), "/contact/_queries", "/contact/_views"); err == nil {
		t.Fatal("Rename(control node) expected unsupported error")
	}
	if err := fs.Mkdir(context.Background(), "/contact/new", 0o755); err == nil {
		t.Fatal("Mkdir(contact) expected unsupported error")
	}
	if _, err := fs.Stat(context.Background(), "/contact/missing"); !os.IsNotExist(err) {
		t.Fatalf("Stat(missing) err=%v, want os.ErrNotExist", err)
	}
	if node := fs.tryLoadParent(context.Background(), "missing"); node != nil {
		t.Fatalf("tryLoadParent(no slash) = %+v, want nil", node)
	}
	if _, err := fs.OpenFile(context.Background(), "/contact/missing.md", os.O_RDONLY, 0); !os.IsNotExist(err) {
		t.Fatalf("OpenFile(missing) err=%v, want os.ErrNotExist", err)
	}
}

func TestWebDAVServerHandleHead(t *testing.T) {
	ops := newMountTestOps(&mockRunner{})
	node := ops.Tree().Resolve("/contact")
	node.AddChild(&vfs.VNode{
		Name:     "doc.md",
		NodeType: vfs.NodeFile,
		Kind:     vfs.NodeKindResource,
		Domain:   "contact",
		Size:     12,
		ModTime:  time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC),
	})
	s := &WebDAVServer{state: &mountState{ops: ops, meta: cache.NewMetadataCache(time.Minute)}}
	t.Cleanup(s.state.meta.Close)

	req := httptest.NewRequest("HEAD", "/contact/doc.md", nil)
	rec := httptest.NewRecorder()
	s.handleHead(rec, req)
	if rec.Code != 200 || rec.Header().Get("Content-Type") != "text/markdown; charset=utf-8" {
		t.Fatalf("HEAD file code=%d headers=%v", rec.Code, rec.Header())
	}
	req = httptest.NewRequest("HEAD", "/contact", nil)
	rec = httptest.NewRecorder()
	s.handleHead(rec, req)
	if rec.Code != 405 {
		t.Fatalf("HEAD dir code=%d, want 405", rec.Code)
	}
}

func TestWebDAVFileSmallBranches(t *testing.T) {
	ops := newMountTestOps(&mockRunner{out: []byte(`{"ok":true}`)})
	if _, err := ops.ReadDir(context.Background(), "/contact/_queries"); err != nil {
		t.Fatalf("ReadDir(_queries) error: %v", err)
	}
	fileNode := ops.Tree().Resolve("/contact/_queries/search-user.request.json")
	file := &webdavFile{ops: ops, node: fileNode, ctx: context.Background()}
	if info, err := file.Stat(); err != nil || info.Name() != "search-user.request.json" {
		t.Fatalf("file Stat() = %+v, %v", info, err)
	}
	if err := file.ensureData(); err != nil || len(file.data) == 0 {
		t.Fatalf("ensureData() data=%q err=%v", file.data, err)
	}
	if pos, err := file.Seek(-1, io.SeekEnd); err != nil || pos < 0 {
		t.Fatalf("Seek(end) = %d, %v", pos, err)
	}
	if _, err := file.Seek(0, 99); err == nil {
		t.Fatal("Seek(invalid whence) expected error")
	}

	dir := &webdavFile{ops: ops, node: ops.Tree().Resolve("/contact"), ctx: context.Background()}
	if n, err := dir.Read(make([]byte, 1)); n != 0 || err != io.EOF {
		t.Fatalf("dir Read() = %d, %v", n, err)
	}
	if pos, err := dir.Seek(10, io.SeekStart); pos != 0 || err != nil {
		t.Fatalf("dir Seek() = %d, %v", pos, err)
	}
	infos, err := dir.Readdir(0)
	if err != nil || len(infos) == 0 {
		t.Fatalf("dir Readdir(0) = %+v, %v", infos, err)
	}
	_, err = dir.Readdir(1000)
	if err != nil && err != io.EOF {
		t.Fatalf("dir Readdir(rest) error: %v", err)
	}
	info := &vnodeFileInfo{node: &vfs.VNode{Name: "doc.md", NodeType: vfs.NodeFile}}
	ct, err := info.ContentType(context.Background())
	if err != nil || ct != "text/markdown; charset=utf-8" {
		t.Fatalf("ContentType(file) = %q, %v", ct, err)
	}
	dirInfo := &vnodeFileInfo{node: &vfs.VNode{Name: "dir", NodeType: vfs.NodeDir}}
	if _, err := dirInfo.ContentType(context.Background()); err == nil {
		t.Fatal("ContentType(dir) expected not implemented")
	}
	if info.Sys() != nil {
		t.Fatal("Sys() should be nil")
	}
}

func TestJoinNodePath(t *testing.T) {
	if got := joinNodePath("/", "drive"); got != "/drive" {
		t.Fatalf("join root = %q", got)
	}
	if got := joinNodePath("/drive/", "doc.md"); got != "/drive/doc.md" {
		t.Fatalf("join child = %q", got)
	}
}
