package mount

import (
	"context"
	"fmt"
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

func (s *WebDAVServer) Close() {
	if s.srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.srv.Shutdown(ctx)
	}
	s.state.meta.Close()
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

type webdavFS struct {
	ops *vfs.Operations
}

func (f *webdavFS) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	name = strings.TrimPrefix(name, "/")
	_, err := f.ops.Mkdir(ctx, "/"+name)
	return err
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
		return &webdavFile{ops: f.ops, node: newNode, ctx: ctx, flags: flag, dirty: true}, nil
	}
	file := &webdavFile{ops: f.ops, node: node, ctx: ctx, flags: flag}
	if flag&os.O_TRUNC != 0 && !node.IsDir() {
		file.data = []byte{}
		file.dirty = true
	}
	return file, nil
}

func (f *webdavFS) RemoveAll(ctx context.Context, name string) error {
	name = strings.TrimPrefix(name, "/")
	return f.ops.Remove(ctx, "/"+name)
}

func (f *webdavFS) Rename(ctx context.Context, oldName, newName string) error {
	oldName = strings.TrimPrefix(oldName, "/")
	newName = strings.TrimPrefix(newName, "/")
	return f.ops.Rename(ctx, "/"+oldName, "/"+newName)
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
