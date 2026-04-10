package mount

import (
	"context"
	"fmt"
	"io"
	"log/slog"
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
	mux := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if depth := r.Header.Get("Depth"); strings.EqualFold(depth, "infinity") {
			http.Error(w, "Depth: infinity is not supported", http.StatusForbidden)
			return
		}

		if r.Method == http.MethodHead {
			s.handleHead(w, r)
			return
		}

		s.handler.ServeHTTP(w, r)
	})

	s.srv = &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	return s.srv.ListenAndServe()
}

func (s *WebDAVServer) handleHead(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/")
	node := s.state.ops.Tree().Resolve(name)
	if node == nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	if node.IsDir() {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	info := &vnodeFileInfo{node: node}
	w.Header().Set("Content-Type", contentTypeFromName(node.Name))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))
	w.Header().Set("Last-Modified", info.ModTime().UTC().Format(http.TimeFormat))
	w.WriteHeader(http.StatusOK)
}

func contentTypeFromName(name string) string {
	switch {
	case strings.HasSuffix(name, ".md"):
		return "text/markdown; charset=utf-8"
	case strings.HasSuffix(name, ".csv"):
		return "text/csv; charset=utf-8"
	case strings.HasSuffix(name, ".json"):
		return "application/json; charset=utf-8"
	case strings.HasSuffix(name, ".jsonl"):
		return "application/x-ndjson; charset=utf-8"
	case strings.HasSuffix(name, ".txt"):
		return "text/plain; charset=utf-8"
	case strings.HasSuffix(name, ".mp4"):
		return "video/mp4"
	case strings.HasSuffix(name, ".png"):
		return "image/png"
	case strings.HasSuffix(name, ".bin"):
		return "application/octet-stream"
	default:
		return "application/octet-stream"
	}
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
		node = f.tryLoadParent(ctx, name)
	}
	if node == nil {
		if flag&os.O_CREATE == 0 {
			return nil, os.ErrNotExist
		}
		newNode, err := f.ops.Create(ctx, name)
		if err != nil {
			return nil, err
		}
		return &webdavFile{ops: f.ops, node: newNode, ctx: ctx}, nil
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
		node = f.tryLoadParent(ctx, name)
	}
	if node == nil {
		return nil, os.ErrNotExist
	}
	return &vnodeFileInfo{node: node}, nil
}

func (f *webdavFS) tryLoadParent(ctx context.Context, name string) *vfs.VNode {
	idx := strings.LastIndex(name, "/")
	if idx < 0 {
		return nil
	}
	parentPath := name[:idx]
	if _, err := f.ops.ReadDir(ctx, parentPath); err != nil {
		return nil
	}
	return f.ops.Tree().Resolve(name)
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
	if f.node.IsDir() {
		return 0, io.EOF
	}
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
	if f.node.IsDir() {
		return 0, nil
	}
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
		slog.Warn("readdir failed, returning empty listing", "path", f.node.Path(), "error", err)
		children = nil
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
	if f.data == nil && err == nil {
		f.data = []byte{}
	}
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

func (i *vnodeFileInfo) ContentType(_ context.Context) (string, error) {
	if i.node.IsDir() {
		return "", webdav.ErrNotImplemented
	}
	return contentTypeFromName(i.node.Name), nil
}
