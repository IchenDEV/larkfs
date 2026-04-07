package mount

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/IchenDEV/larkfs/pkg/config"
	"github.com/IchenDEV/larkfs/pkg/vfs"
	"golang.org/x/net/webdav"
)

type WebDAVServer struct {
	handler *webdav.Handler
	srv     *http.Server
	state   *mountState
}

func NewWebDAVServer(cfg config.ServeConfig) (*WebDAVServer, error) {
	mountCfg := config.MountConfig{
		LogLevel:    cfg.LogLevel,
		ReadOnly:    cfg.ReadOnly,
		Domains:     cfg.Domains,
		LarkCLIPath: cfg.LarkCLIPath,
		MetadataTTL: 60,
	}
	if err := mountCfg.Resolve(); err != nil {
		return nil, err
	}

	state, err := buildMount(mountCfg)
	if err != nil {
		return nil, err
	}

	fs := &webdavFS{ops: state.ops}
	handler := &webdav.Handler{
		FileSystem: fs,
		LockSystem: webdav.NewMemLS(),
		Prefix:     "/",
	}

	return &WebDAVServer{handler: handler, state: state}, nil
}

func (s *WebDAVServer) Serve(addr string) error {
	s.srv = &http.Server{
		Addr:    addr,
		Handler: s.handler,
	}
	return s.srv.ListenAndServe()
}

func (s *WebDAVServer) Close() {
	if s.srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.srv.Shutdown(ctx)
	}
	s.state.meta.Close()
}

type webdavFS struct {
	ops *vfs.Operations
}

func (f *webdavFS) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	return webdav.ErrNotImplemented
}

func (f *webdavFS) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	name = strings.TrimPrefix(name, "/")
	node := f.ops.Tree().Resolve(name)
	if node == nil {
		return nil, os.ErrNotExist
	}
	return &webdavFile{ops: f.ops, node: node, ctx: ctx}, nil
}

func (f *webdavFS) RemoveAll(ctx context.Context, name string) error {
	return webdav.ErrNotImplemented
}

func (f *webdavFS) Rename(ctx context.Context, oldName, newName string) error {
	return webdav.ErrNotImplemented
}

func (f *webdavFS) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	name = strings.TrimPrefix(name, "/")
	node := f.ops.Tree().Resolve(name)
	if node == nil {
		return nil, os.ErrNotExist
	}
	return &vnodeFileInfo{node: node}, nil
}

type webdavFile struct {
	ops       *vfs.Operations
	node      *vfs.VNode
	ctx       context.Context
	data      []byte
	offset    int64
	dirOffset int
}

func (f *webdavFile) Close() error { return nil }

func (f *webdavFile) Read(p []byte) (int, error) {
	if err := f.ensureData(); err != nil {
		return 0, err
	}
	if f.offset >= int64(len(f.data)) {
		return 0, io.EOF
	}
	n := copy(p, f.data[f.offset:])
	f.offset += int64(n)
	return n, nil
}

func (f *webdavFile) Seek(offset int64, whence int) (int64, error) {
	if err := f.ensureData(); err != nil {
		return 0, err
	}
	var newOffset int64
	switch whence {
	case io.SeekStart:
		newOffset = offset
	case io.SeekCurrent:
		newOffset = f.offset + offset
	case io.SeekEnd:
		newOffset = int64(len(f.data)) + offset
	default:
		return 0, fmt.Errorf("invalid whence: %d", whence)
	}
	if newOffset < 0 {
		return 0, fmt.Errorf("seek: negative position")
	}
	f.offset = newOffset
	return f.offset, nil
}

func (f *webdavFile) Readdir(count int) ([]os.FileInfo, error) {
	children, err := f.ops.ReadDir(f.ctx, f.node.Path())
	if err != nil {
		return nil, err
	}

	if f.dirOffset >= len(children) {
		if count > 0 {
			return nil, io.EOF
		}
		return nil, nil
	}

	remaining := children[f.dirOffset:]
	if count > 0 && count < len(remaining) {
		remaining = remaining[:count]
	}
	f.dirOffset += len(remaining)

	infos := make([]os.FileInfo, len(remaining))
	for i, c := range remaining {
		infos[i] = &vnodeFileInfo{node: c}
	}
	return infos, nil
}

func (f *webdavFile) Stat() (os.FileInfo, error) {
	return &vnodeFileInfo{node: f.node}, nil
}

func (f *webdavFile) Write(p []byte) (int, error) {
	if err := f.ops.Write(f.ctx, f.node.Path(), p); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (f *webdavFile) ensureData() error {
	if f.data != nil {
		return nil
	}
	var err error
	f.data, err = f.ops.Read(f.ctx, f.node.Path())
	return err
}

type vnodeFileInfo struct {
	node *vfs.VNode
}

func (i *vnodeFileInfo) Name() string       { return i.node.Name }
func (i *vnodeFileInfo) Size() int64        { return i.node.Size }
func (i *vnodeFileInfo) ModTime() time.Time { return i.node.ModTime }
func (i *vnodeFileInfo) Sys() interface{}   { return nil }

func (i *vnodeFileInfo) Mode() os.FileMode {
	if i.node.IsDir() {
		return os.ModeDir | 0o755
	}
	return 0o644
}

func (i *vnodeFileInfo) IsDir() bool {
	return i.node.IsDir()
}
