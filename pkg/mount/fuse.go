package mount

import (
	"context"
	"log/slog"
	"os"
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
	server *fuse.Server
	state  *mountState
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

	return &FUSEServer{server: server, state: state}, nil
}

func (s *FUSEServer) Wait() {
	s.server.Wait()
}

func (s *FUSEServer) Unmount() {
	if err := s.server.Unmount(); err != nil {
		slog.Error("unmount failed", "error", err)
	}
	s.state.meta.Close()
}

type larkfsRoot struct {
	fs.Inode
	ops     *vfs.Operations
	content *cache.ContentCache
}

var _ = (fs.NodeReaddirer)((*larkfsRoot)(nil))
var _ = (fs.NodeLookuper)((*larkfsRoot)(nil))

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

func (n *larkfsNode) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	children, err := n.ops.ReadDir(ctx, n.vnode.Path())
	if err != nil {
		return nil, syscall.EIO
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

func (n *larkfsNode) Open(ctx context.Context, flags uint32) (fs.FileHandle, uint32, syscall.Errno) {
	n.mu.Lock()
	n.dataOnce = sync.Once{}
	n.cached = nil
	n.mu.Unlock()
	return nil, fuse.FOPEN_KEEP_CACHE, 0
}

func (n *larkfsNode) Read(ctx context.Context, fh fs.FileHandle, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
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
	if err := n.ops.Write(ctx, n.vnode.Path(), data); err != nil {
		slog.Error("write failed", "path", n.vnode.Path(), "error", err)
		return 0, syscall.EIO
	}
	n.mu.Lock()
	n.cached = nil
	n.dataOnce = sync.Once{}
	n.mu.Unlock()
	n.content.Invalidate(n.vnode.Path())
	return uint32(len(data)), 0
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
