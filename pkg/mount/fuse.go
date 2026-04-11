package mount

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/IchenDEV/larkfs/pkg/cache"
	"github.com/IchenDEV/larkfs/pkg/config"
	"github.com/IchenDEV/larkfs/pkg/vfs"
	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

type FUSEServer struct {
	server     *fuse.Server
	state      *mountState
	mountpoint string
}

func NewFUSEServer(cfg config.MountConfig) (*FUSEServer, error) {
	state, err := buildMount(cfg)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(cfg.Mountpoint, 0o755); err != nil {
		return nil, err
	}

	root := &larkfsRoot{ops: state.ops, content: state.content}
	ttl := time.Duration(cfg.MetadataTTL) * time.Second
	server, err := fs.Mount(cfg.Mountpoint, root, &fs.Options{
		MountOptions: fuse.MountOptions{
			Name:          "larkfs",
			FsName:        "larkfs",
			DisableXAttrs: true,
			MaxBackground: 12,
		},
		AttrTimeout:  &ttl,
		EntryTimeout: &ttl,
	})
	if err != nil {
		return nil, err
	}

	return &FUSEServer{server: server, state: state, mountpoint: cfg.Mountpoint}, nil
}

func (s *FUSEServer) Wait() {
	s.server.Wait()
}

func (s *FUSEServer) Unmount() {
	if err := s.server.Unmount(); err != nil {
		slog.Warn("standard unmount failed, trying lazy unmount", "error", err)
		if err := lazyUnmount(s.mountpoint); err != nil {
			slog.Error("lazy unmount failed", "error", err)
		}
	}
	s.state.meta.Close()
}

func lazyUnmount(mountpoint string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("umount", mountpoint).Run()
	default:
		if err := exec.Command("fusermount", "-uz", mountpoint).Run(); err != nil {
			return exec.Command("fusermount3", "-uz", mountpoint).Run()
		}
		return nil
	}
}

func errnoFromError(err error) syscall.Errno {
	if err == nil {
		return 0
	}
	switch {
	case errors.Is(err, vfs.ErrReadOnly):
		return syscall.EROFS
	case errors.Is(err, vfs.ErrNotFound):
		return syscall.ENOENT
	case errors.Is(err, vfs.ErrUnsupported):
		return syscall.ENOTSUP
	}

	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "read-only"):
		return syscall.EROFS
	case strings.Contains(msg, "not found"):
		return syscall.ENOENT
	case strings.Contains(msg, "not supported"), strings.Contains(msg, "unsupported"):
		return syscall.ENOTSUP
	default:
		return syscall.EIO
	}
}

type larkfsRoot struct {
	fs.Inode
	ops     *vfs.Operations
	content *cache.ContentCache
}

var _ = (fs.NodeReaddirer)((*larkfsRoot)(nil))
var _ = (fs.NodeLookuper)((*larkfsRoot)(nil))
var _ = (fs.NodeGetattrer)((*larkfsRoot)(nil))
var _ = (fs.NodeMkdirer)((*larkfsRoot)(nil))
var _ = (fs.NodeCreater)((*larkfsRoot)(nil))
var _ = (fs.NodeUnlinker)((*larkfsRoot)(nil))
var _ = (fs.NodeRmdirer)((*larkfsRoot)(nil))
var _ = (fs.NodeRenamer)((*larkfsRoot)(nil))

func (r *larkfsRoot) Getattr(ctx context.Context, fh fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	out.Mode = syscall.S_IFDIR | 0o755
	now := time.Now()
	out.Atime = uint64(now.Unix())
	out.Mtime = uint64(now.Unix())
	return 0
}

func (r *larkfsRoot) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	children, err := r.ops.ReadDir(ctx, "/")
	if err != nil {
		slog.Error("readdir root failed", "error", err)
		return nil, syscall.EIO
	}

	entries := make([]fuse.DirEntry, len(children))
	for i, child := range children {
		mode := uint32(syscall.S_IFREG | 0o644)
		if child.IsDir() {
			mode = syscall.S_IFDIR | 0o755
		}
		entries[i] = fuse.DirEntry{Name: child.Name, Mode: mode}
	}
	return fs.NewListDirStream(entries), 0
}

func (r *larkfsRoot) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	children, err := r.ops.ReadDir(ctx, "/")
	if err != nil {
		return nil, syscall.EIO
	}

	for _, child := range children {
		if child.Name == name {
			stable := fs.StableAttr{Mode: syscall.S_IFDIR | 0o755}
			node := &larkfsNode{ops: r.ops, vnode: child, content: r.content}
			return r.NewInode(ctx, node, stable), 0
		}
	}
	return nil, syscall.ENOENT
}

func (r *larkfsRoot) Mkdir(ctx context.Context, name string, mode uint32, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	return nil, syscall.EROFS
}

func (r *larkfsRoot) Create(ctx context.Context, name string, flags uint32, mode uint32, out *fuse.EntryOut) (*fs.Inode, fs.FileHandle, uint32, syscall.Errno) {
	return nil, nil, 0, syscall.EROFS
}

func (r *larkfsRoot) Unlink(ctx context.Context, name string) syscall.Errno {
	return syscall.EROFS
}

func (r *larkfsRoot) Rmdir(ctx context.Context, name string) syscall.Errno {
	return syscall.EROFS
}

func (r *larkfsRoot) Rename(ctx context.Context, name string, newParent fs.InodeEmbedder, newName string, flags uint32) syscall.Errno {
	return syscall.EROFS
}

type larkfsNode struct {
	fs.Inode
	ops     *vfs.Operations
	vnode   *vfs.VNode
	content *cache.ContentCache

	mu       sync.Mutex
	dataOnce sync.Once
	cached   []byte
	dataSize atomic.Int64
}

var _ = (fs.NodeReaddirer)((*larkfsNode)(nil))
var _ = (fs.NodeLookuper)((*larkfsNode)(nil))
var _ = (fs.NodeGetattrer)((*larkfsNode)(nil))
var _ = (fs.NodeOpener)((*larkfsNode)(nil))
var _ = (fs.NodeReader)((*larkfsNode)(nil))
var _ = (fs.NodeWriter)((*larkfsNode)(nil))
var _ = (fs.NodeSetattrer)((*larkfsNode)(nil))
var _ = (fs.NodeMkdirer)((*larkfsNode)(nil))
var _ = (fs.NodeCreater)((*larkfsNode)(nil))
var _ = (fs.NodeUnlinker)((*larkfsNode)(nil))
var _ = (fs.NodeRmdirer)((*larkfsNode)(nil))
var _ = (fs.NodeRenamer)((*larkfsNode)(nil))

func (n *larkfsNode) Getattr(ctx context.Context, fh fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	if n.vnode.IsDir() {
		out.Mode = syscall.S_IFDIR | 0o755
	} else {
		out.Mode = syscall.S_IFREG | 0o644
		if sz := n.dataSize.Load(); sz > 0 {
			out.Size = uint64(sz)
		} else {
			out.Size = 4096
		}
	}
	out.Atime = uint64(n.vnode.ModTime.Unix())
	out.Mtime = uint64(n.vnode.ModTime.Unix())
	return 0
}

func (n *larkfsNode) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	children, err := n.ops.ReadDir(ctx, n.vnode.Path())
	if err != nil {
		slog.Error("readdir failed", "path", n.vnode.Path(), "error", err)
		return nil, errnoFromError(err)
	}

	entries := make([]fuse.DirEntry, len(children))
	for i, child := range children {
		mode := uint32(syscall.S_IFREG | 0o644)
		if child.IsDir() {
			mode = syscall.S_IFDIR | 0o755
		}
		entries[i] = fuse.DirEntry{Name: child.Name, Mode: mode}
	}
	return fs.NewListDirStream(entries), 0
}

func (n *larkfsNode) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	children, err := n.ops.ReadDir(ctx, n.vnode.Path())
	if err != nil {
		return nil, errnoFromError(err)
	}

	for _, child := range children {
		if child.Name == name {
			stable := fs.StableAttr{Mode: syscall.S_IFREG | 0o644}
			if child.IsDir() {
				stable.Mode = syscall.S_IFDIR | 0o755
			}
			node := &larkfsNode{ops: n.ops, vnode: child, content: n.content}
			return n.NewInode(ctx, node, stable), 0
		}
	}
	return nil, syscall.ENOENT
}

func (n *larkfsNode) Mkdir(ctx context.Context, name string, mode uint32, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	child, err := n.ops.Mkdir(ctx, joinNodePath(n.vnode.Path(), name))
	if err != nil {
		return nil, errnoFromError(err)
	}
	node := &larkfsNode{ops: n.ops, vnode: child, content: n.content}
	return n.NewInode(ctx, node, fs.StableAttr{Mode: syscall.S_IFDIR | 0o755}), 0
}

func (n *larkfsNode) Create(ctx context.Context, name string, flags uint32, mode uint32, out *fuse.EntryOut) (*fs.Inode, fs.FileHandle, uint32, syscall.Errno) {
	child, err := n.ops.Create(ctx, joinNodePath(n.vnode.Path(), name))
	if err != nil {
		return nil, nil, 0, errnoFromError(err)
	}
	node := &larkfsNode{ops: n.ops, vnode: child, content: n.content}
	inode := n.NewInode(ctx, node, fs.StableAttr{Mode: syscall.S_IFREG | 0o644})
	handle := &larkfsFileHandle{node: node, data: []byte{}, flags: flags, dirty: true, loaded: true}
	return inode, handle, 0, 0
}

func (n *larkfsNode) Unlink(ctx context.Context, name string) syscall.Errno {
	if err := n.ops.Remove(ctx, joinNodePath(n.vnode.Path(), name)); err != nil {
		return errnoFromError(err)
	}
	return 0
}

func (n *larkfsNode) Rmdir(ctx context.Context, name string) syscall.Errno {
	if err := n.ops.Remove(ctx, joinNodePath(n.vnode.Path(), name)); err != nil {
		return errnoFromError(err)
	}
	return 0
}

func (n *larkfsNode) Rename(ctx context.Context, name string, newParent fs.InodeEmbedder, newName string, flags uint32) syscall.Errno {
	var targetPath string
	switch p := newParent.(type) {
	case *larkfsNode:
		targetPath = joinNodePath(p.vnode.Path(), newName)
	case *larkfsRoot:
		targetPath = "/" + newName
	default:
		return syscall.ENOTSUP
	}
	if err := n.ops.Rename(ctx, joinNodePath(n.vnode.Path(), name), targetPath); err != nil {
		return errnoFromError(err)
	}
	return 0
}

func (n *larkfsNode) Open(ctx context.Context, flags uint32) (fs.FileHandle, uint32, syscall.Errno) {
	if n.vnode.IsDir() {
		return nil, 0, 0
	}
	data, errno := n.ensureData(ctx)
	if errno != 0 {
		return nil, 0, errno
	}
	handle := &larkfsFileHandle{
		node:   n,
		data:   append([]byte(nil), data...),
		flags:  flags,
		dirty:  flags&syscall.O_TRUNC != 0,
		trunc:  flags&syscall.O_TRUNC != 0,
		loaded: true,
	}
	if handle.trunc {
		handle.data = []byte{}
	}
	return handle, 0, 0
}

func (n *larkfsNode) Read(ctx context.Context, fh fs.FileHandle, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	if handle, ok := fh.(*larkfsFileHandle); ok {
		return handle.Read(ctx, dest, off)
	}
	data, errno := n.ensureData(ctx)
	if errno != 0 {
		return nil, errno
	}

	if off >= int64(len(data)) {
		return fuse.ReadResultData(nil), 0
	}
	end := off + int64(len(dest))
	if end > int64(len(data)) {
		end = int64(len(data))
	}
	return fuse.ReadResultData(data[off:end]), 0
}

func (n *larkfsNode) Write(ctx context.Context, fh fs.FileHandle, data []byte, off int64) (uint32, syscall.Errno) {
	if handle, ok := fh.(*larkfsFileHandle); ok {
		return handle.Write(ctx, data, off)
	}
	if err := n.ops.Write(ctx, n.vnode.Path(), data); err != nil {
		slog.Error("write failed", "path", n.vnode.Path(), "error", err)
		return 0, syscall.EIO
	}
	n.invalidateCache()
	return uint32(len(data)), 0
}

func (n *larkfsNode) Setattr(ctx context.Context, fh fs.FileHandle, in *fuse.SetAttrIn, out *fuse.AttrOut) syscall.Errno {
	if handle, ok := fh.(*larkfsFileHandle); ok {
		return handle.Setattr(ctx, in, out)
	}
	if sz, ok := in.GetSize(); ok {
		data, errno := n.ensureData(ctx)
		if errno != 0 {
			return errno
		}
		if sz < uint64(len(data)) {
			data = data[:sz]
		} else if sz > uint64(len(data)) {
			buf := make([]byte, sz)
			copy(buf, data)
			data = buf
		}
		if err := n.ops.Write(ctx, n.vnode.Path(), data); err != nil {
			return syscall.EIO
		}
		n.invalidateCache()
	}
	return n.Getattr(ctx, fh, out)
}

func (n *larkfsNode) ensureData(ctx context.Context) ([]byte, syscall.Errno) {
	n.mu.Lock()
	if n.cached != nil {
		data := n.cached
		n.mu.Unlock()
		return data, 0
	}
	n.mu.Unlock()

	path := n.vnode.Path()
	if data, ok := n.content.Get(path); ok {
		n.mu.Lock()
		n.cached = data
		n.mu.Unlock()
		n.dataSize.Store(int64(len(data)))
		return data, 0
	}

	data, err := n.ops.Read(ctx, path)
	if err != nil {
		slog.Error("read failed", "path", path, "error", err)
		return nil, syscall.EIO
	}

	_ = n.content.Set(path, data)
	n.mu.Lock()
	n.cached = data
	n.mu.Unlock()
	n.dataSize.Store(int64(len(data)))
	return data, 0
}

func (n *larkfsNode) invalidateCache() {
	n.mu.Lock()
	n.cached = nil
	n.dataOnce = sync.Once{}
	n.mu.Unlock()
	n.content.Invalidate(n.vnode.Path())
}

type larkfsFileHandle struct {
	node   *larkfsNode
	data   []byte
	flags  uint32
	dirty  bool
	trunc  bool
	loaded bool
	mu     sync.Mutex
}

var _ = (fs.FileReader)((*larkfsFileHandle)(nil))
var _ = (fs.FileWriter)((*larkfsFileHandle)(nil))
var _ = (fs.FileFlusher)((*larkfsFileHandle)(nil))
var _ = (fs.FileReleaser)((*larkfsFileHandle)(nil))
var _ = (fs.FileSetattrer)((*larkfsFileHandle)(nil))

func (h *larkfsFileHandle) Read(ctx context.Context, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if off >= int64(len(h.data)) {
		return fuse.ReadResultData(nil), 0
	}
	end := off + int64(len(dest))
	if end > int64(len(h.data)) {
		end = int64(len(h.data))
	}
	return fuse.ReadResultData(h.data[off:end]), 0
}

func (h *larkfsFileHandle) Write(ctx context.Context, data []byte, off int64) (uint32, syscall.Errno) {
	h.mu.Lock()
	defer h.mu.Unlock()
	end := off + int64(len(data))
	if end > int64(len(h.data)) {
		buf := make([]byte, end)
		copy(buf, h.data)
		h.data = buf
	}
	copy(h.data[off:end], data)
	h.dirty = true
	h.node.vnode.Size = int64(len(h.data))
	return uint32(len(data)), 0
}

func (h *larkfsFileHandle) Flush(ctx context.Context) syscall.Errno {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.dirty {
		return 0
	}
	if err := h.node.ops.Write(ctx, h.node.vnode.Path(), h.data); err != nil {
		slog.Error("flush failed", "path", h.node.vnode.Path(), "error", err)
		return syscall.EIO
	}
	h.node.vnode.Size = int64(len(h.data))
	h.node.vnode.ModTime = time.Now()
	h.node.invalidateCache()
	h.dirty = false
	return 0
}

func (h *larkfsFileHandle) Release(ctx context.Context) syscall.Errno {
	return h.Flush(ctx)
}

func (h *larkfsFileHandle) Setattr(ctx context.Context, in *fuse.SetAttrIn, out *fuse.AttrOut) syscall.Errno {
	h.mu.Lock()
	defer h.mu.Unlock()
	if sz, ok := in.GetSize(); ok {
		if sz < uint64(len(h.data)) {
			h.data = h.data[:sz]
		} else if sz > uint64(len(h.data)) {
			buf := make([]byte, sz)
			copy(buf, h.data)
			h.data = buf
		}
		h.dirty = true
	}
	out.Size = uint64(len(h.data))
	now := time.Now()
	out.Mtime = uint64(now.Unix())
	out.Atime = uint64(now.Unix())
	return 0
}

func joinNodePath(parent, child string) string {
	if parent == "/" {
		return "/" + child
	}
	return strings.TrimRight(parent, "/") + "/" + child
}
