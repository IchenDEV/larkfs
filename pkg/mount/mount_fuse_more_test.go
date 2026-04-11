package mount

import (
	"context"
	"strings"
	"syscall"
	"testing"

	"github.com/IchenDEV/larkfs/pkg/cache"
	"github.com/hanwen/go-fuse/v2/fuse"
)

func TestFUSEFileHandleReadWriteFlushSetattr(t *testing.T) {
	runner := &mockRunner{out: []byte(`{"data":[{"name":"Alice"}]}`)}
	ops := newMountTestOps(runner)
	if _, err := ops.ReadDir(context.Background(), "/contact"); err != nil {
		t.Fatalf("ReadDir(contact) error: %v", err)
	}
	if _, err := ops.ReadDir(context.Background(), "/contact/_queries"); err != nil {
		t.Fatalf("ReadDir(contact/_queries) error: %v", err)
	}
	vnode := ops.Tree().Resolve("/contact/_queries/search-user.request.json")
	content, err := cache.NewContentCache(t.TempDir(), 1024*1024)
	if err != nil {
		t.Fatalf("NewContentCache() error: %v", err)
	}
	node := &larkfsNode{ops: ops, vnode: vnode, content: content}
	handle := &larkfsFileHandle{node: node, data: []byte{}, dirty: true, loaded: true}

	if n, errno := handle.Write(context.Background(), []byte(`{"query":"Bob"}`), 0); errno != 0 || n != uint32(len(`{"query":"Bob"}`)) {
		t.Fatalf("Write() = %d, %v", n, errno)
	}
	read, errno := handle.Read(context.Background(), make([]byte, 20), 0)
	if errno != 0 {
		t.Fatalf("Read() errno=%v", errno)
	}
	bytes, _ := read.Bytes(make([]byte, read.Size()))
	if !strings.Contains(string(bytes), "Bob") {
		t.Fatalf("Read() = %q", bytes)
	}
	attrHandle := &larkfsFileHandle{node: node, data: []byte("hello"), loaded: true}
	in := &fuse.SetAttrIn{}
	in.Valid = fuse.FATTR_SIZE
	in.Size = 3
	var out fuse.AttrOut
	if errno := attrHandle.Setattr(context.Background(), in, &out); errno != 0 || out.Size != 3 {
		t.Fatalf("Setattr() errno=%v size=%d", errno, out.Size)
	}
	if errno := handle.Flush(context.Background()); errno != 0 {
		t.Fatalf("Flush() errno=%v", errno)
	}
	if !strings.Contains(strings.Join(runner.lastArgs, " "), "contact +search-user") {
		t.Fatalf("Flush runner args = %v", runner.lastArgs)
	}
	if errno := handle.Release(context.Background()); errno != 0 {
		t.Fatalf("Release() errno=%v", errno)
	}
}

func TestFUSERootAndNodeErrnos(t *testing.T) {
	content, err := cache.NewContentCache(t.TempDir(), 1024)
	if err != nil {
		t.Fatalf("NewContentCache() error: %v", err)
	}
	root := &larkfsRoot{ops: newMountTestOps(&mockRunner{}), content: content}
	if _, errno := root.Mkdir(context.Background(), "x", 0, nil); errno != syscall.EROFS {
		t.Fatalf("root Mkdir errno=%v", errno)
	}
	if _, _, _, errno := root.Create(context.Background(), "x", 0, 0, nil); errno != syscall.EROFS {
		t.Fatalf("root Create errno=%v", errno)
	}
	if errno := root.Unlink(context.Background(), "x"); errno != syscall.EROFS {
		t.Fatalf("root Unlink errno=%v", errno)
	}
	if errno := root.Rmdir(context.Background(), "x"); errno != syscall.EROFS {
		t.Fatalf("root Rmdir errno=%v", errno)
	}
	if errno := root.Rename(context.Background(), "x", root, "y", 0); errno != syscall.EROFS {
		t.Fatalf("root Rename errno=%v", errno)
	}
}

func TestFUSERootAndNodeReadPaths(t *testing.T) {
	runner := &mockRunner{out: []byte(`{"data":[{"name":"Alice"}]}`)}
	ops := newMountTestOps(runner)
	content, err := cache.NewContentCache(t.TempDir(), 1024*1024)
	if err != nil {
		t.Fatalf("NewContentCache() error: %v", err)
	}
	root := &larkfsRoot{ops: ops, content: content}
	var attr fuse.AttrOut
	if errno := root.Getattr(context.Background(), nil, &attr); errno != 0 || attr.Mode&syscall.S_IFDIR == 0 {
		t.Fatalf("root Getattr errno=%v mode=%o", errno, attr.Mode)
	}
	if _, errno := root.Readdir(context.Background()); errno != 0 {
		t.Fatalf("root Readdir errno=%v", errno)
	}

	contact := ops.Tree().Resolve("/contact")
	node := &larkfsNode{ops: ops, vnode: contact, content: content}
	if errno := node.Getattr(context.Background(), nil, &attr); errno != 0 || attr.Mode&syscall.S_IFDIR == 0 {
		t.Fatalf("node Getattr dir errno=%v mode=%o", errno, attr.Mode)
	}
	if _, errno := node.Readdir(context.Background()); errno != 0 {
		t.Fatalf("node Readdir errno=%v", errno)
	}
}

func TestFUSENodeCRUDErrorBranches(t *testing.T) {
	ops := newMountTestOps(&mockRunner{})
	content, err := cache.NewContentCache(t.TempDir(), 1024)
	if err != nil {
		t.Fatalf("NewContentCache() error: %v", err)
	}
	contact := ops.Tree().Resolve("/contact")
	node := &larkfsNode{ops: ops, vnode: contact, content: content}
	if _, errno := node.Lookup(context.Background(), "missing", nil); errno != syscall.ENOENT {
		t.Fatalf("Lookup missing errno=%v", errno)
	}
	if _, errno := node.Mkdir(context.Background(), "folder", 0, nil); errno != syscall.ENOTSUP {
		t.Fatalf("Mkdir unsupported errno=%v", errno)
	}
	if _, _, _, errno := node.Create(context.Background(), "file.md", 0, 0, nil); errno != syscall.ENOTSUP {
		t.Fatalf("Create unsupported errno=%v", errno)
	}
	if errno := node.Unlink(context.Background(), "missing.md"); errno != syscall.ENOENT {
		t.Fatalf("Unlink missing errno=%v", errno)
	}
	if errno := node.Rmdir(context.Background(), "missing"); errno != syscall.ENOENT {
		t.Fatalf("Rmdir missing errno=%v", errno)
	}
	if errno := node.Rename(context.Background(), "missing", node, "other", 0); errno != syscall.ENOENT {
		t.Fatalf("Rename missing errno=%v", errno)
	}
	if errno := node.Rename(context.Background(), "missing", nil, "other", 0); errno != syscall.ENOTSUP {
		t.Fatalf("Rename bad parent errno=%v", errno)
	}
}

func TestFUSENodeOpenReadWriteSetattr(t *testing.T) {
	runner := &mockRunner{out: []byte(`{"data":[{"name":"Alice"}]}`)}
	ops := newMountTestOps(runner)
	if _, err := ops.ReadDir(context.Background(), "/contact/_queries"); err != nil {
		t.Fatalf("ReadDir(_queries) error: %v", err)
	}
	content, err := cache.NewContentCache(t.TempDir(), 1024*1024)
	if err != nil {
		t.Fatalf("NewContentCache() error: %v", err)
	}
	vnode := ops.Tree().Resolve("/contact/_queries/search-user.request.json")
	node := &larkfsNode{ops: ops, vnode: vnode, content: content}

	fh, _, errno := node.Open(context.Background(), syscall.O_TRUNC)
	if errno != 0 {
		t.Fatalf("Open() errno=%v", errno)
	}
	if n, errno := node.Write(context.Background(), fh, []byte(`{"query":"Alice"}`), 0); errno != 0 || n == 0 {
		t.Fatalf("Write() = %d, %v", n, errno)
	}
	if errno := fh.(*larkfsFileHandle).Flush(context.Background()); errno != 0 {
		t.Fatalf("Flush() errno=%v", errno)
	}
	read, errno := node.Read(context.Background(), fh, make([]byte, 50), 0)
	if errno != 0 {
		t.Fatalf("Read() errno=%v", errno)
	}
	bytes, _ := read.Bytes(make([]byte, read.Size()))
	if !strings.Contains(string(bytes), "Alice") {
		t.Fatalf("Read() = %q", bytes)
	}
	in := &fuse.SetAttrIn{}
	in.Valid = fuse.FATTR_SIZE
	in.Size = 2
	var out fuse.AttrOut
	if errno := node.Setattr(context.Background(), fh, in, &out); errno != 0 || out.Size != 2 {
		t.Fatalf("Setattr(handle) errno=%v size=%d", errno, out.Size)
	}
	node.invalidateCache()
	if _, errno := node.ensureData(context.Background()); errno != 0 {
		t.Fatalf("ensureData() errno=%v", errno)
	}
}

func TestFUSENodeDirectReadWriteSetattr(t *testing.T) {
	runner := &mockRunner{out: []byte(`{"data":[{"name":"Alice"}]}`)}
	ops := newMountTestOps(runner)
	if _, err := ops.ReadDir(context.Background(), "/contact/_queries"); err != nil {
		t.Fatalf("ReadDir(_queries) error: %v", err)
	}
	content, err := cache.NewContentCache(t.TempDir(), 1024*1024)
	if err != nil {
		t.Fatalf("NewContentCache() error: %v", err)
	}
	vnode := ops.Tree().Resolve("/contact/_queries/search-user.request.json")
	node := &larkfsNode{ops: ops, vnode: vnode, content: content}

	read, errno := node.Read(context.Background(), nil, make([]byte, 64), 0)
	if errno != 0 {
		t.Fatalf("direct Read() errno=%v", errno)
	}
	bytes, _ := read.Bytes(make([]byte, read.Size()))
	if !strings.Contains(string(bytes), "search-user") {
		t.Fatalf("direct Read() = %q", bytes)
	}
	if n, errno := node.Write(context.Background(), nil, []byte(`{"query":"Alice"}`), 0); errno != 0 || n == 0 {
		t.Fatalf("direct Write() = %d, %v", n, errno)
	}
	in := &fuse.SetAttrIn{}
	in.Valid = fuse.FATTR_SIZE
	in.Size = uint64(len(`{"query":"Alice"}`))
	var out fuse.AttrOut
	if errno := node.Setattr(context.Background(), nil, in, &out); errno != 0 {
		t.Fatalf("direct Setattr() errno=%v", errno)
	}
	if read, errno := node.Read(context.Background(), nil, make([]byte, 4), 999); errno != 0 || read.Size() != 0 {
		t.Fatalf("direct Read past EOF size=%d errno=%v", read.Size(), errno)
	}
}
