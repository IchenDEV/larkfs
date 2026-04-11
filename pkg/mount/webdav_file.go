package mount

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/IchenDEV/larkfs/pkg/vfs"
	"golang.org/x/net/webdav"
)

type webdavFile struct {
	ops       *vfs.Operations
	node      *vfs.VNode
	ctx       context.Context
	data      []byte
	offset    int64
	dirOffset int
	flags     int
	dirty     bool
}

func (f *webdavFile) Close() error {
	if f.node.IsDir() || !f.dirty {
		return nil
	}
	if err := f.ops.Write(f.ctx, f.node.Path(), f.data); err != nil {
		return err
	}
	f.node.Size = int64(len(f.data))
	f.node.ModTime = time.Now()
	f.dirty = false
	return nil
}

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
	if err := f.ensureData(); err != nil {
		return 0, err
	}
	end := f.offset + int64(len(p))
	if end > int64(len(f.data)) {
		buf := make([]byte, end)
		copy(buf, f.data)
		f.data = buf
	}
	copy(f.data[f.offset:end], p)
	f.offset = end
	f.dirty = true
	f.node.Size = int64(len(f.data))
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
	if err == nil {
		f.node.Size = int64(len(f.data))
	}
	return err
}

func (f *webdavFile) DeadProps() (map[xml.Name]webdav.Property, error) {
	props := make(map[xml.Name]webdav.Property)
	if !f.node.CreatedTime.IsZero() {
		name := xml.Name{Space: "DAV:", Local: "creationdate"}
		props[name] = webdav.Property{
			XMLName:  name,
			InnerXML: []byte(f.node.CreatedTime.UTC().Format(time.RFC3339)),
		}
	}
	return props, nil
}

func (f *webdavFile) Patch(patches []webdav.Proppatch) ([]webdav.Propstat, error) {
	return []webdav.Propstat{{
		Status: http.StatusForbidden,
		Props:  []webdav.Property{},
	}}, nil
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
