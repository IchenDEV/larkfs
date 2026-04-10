package mount

import (
	"encoding/xml"
	"testing"
	"time"

	"github.com/IchenDEV/larkfs/pkg/vfs"
	"golang.org/x/net/webdav"
)

func TestVnodeFileInfoDir(t *testing.T) {
	node := &vfs.VNode{
		Name:     "folder",
		NodeType: vfs.NodeDir,
		ModTime:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	info := &vnodeFileInfo{node: node}

	if !info.IsDir() {
		t.Error("expected IsDir() true for NodeDir")
	}
	if info.Mode()&0o755 == 0 {
		t.Error("expected directory mode bits")
	}
	if info.Name() != "folder" {
		t.Errorf("expected name=folder, got %s", info.Name())
	}
}

func TestVnodeFileInfoFile(t *testing.T) {
	node := &vfs.VNode{
		Name:     "test.md",
		NodeType: vfs.NodeFile,
		Size:     42,
		ModTime:  time.Date(2026, 3, 15, 10, 30, 0, 0, time.UTC),
	}
	info := &vnodeFileInfo{node: node}

	if info.IsDir() {
		t.Error("expected IsDir() false for NodeFile")
	}
	if info.Size() != 42 {
		t.Errorf("expected size=42, got %d", info.Size())
	}
	if !info.ModTime().Equal(time.Date(2026, 3, 15, 10, 30, 0, 0, time.UTC)) {
		t.Errorf("unexpected modtime: %v", info.ModTime())
	}
}

func TestDeadPropsWithCreatedTime(t *testing.T) {
	created := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	node := &vfs.VNode{
		Name:        "doc.md",
		NodeType:    vfs.NodeFile,
		CreatedTime: created,
	}
	f := &webdavFile{node: node}

	props, err := f.DeadProps()
	if err != nil {
		t.Fatalf("DeadProps() error: %v", err)
	}

	key := xml.Name{Space: "DAV:", Local: "creationdate"}
	prop, ok := props[key]
	if !ok {
		t.Fatal("expected creationdate property")
	}
	want := created.UTC().Format(time.RFC3339)
	got := string(prop.InnerXML)
	if got != want {
		t.Errorf("creationdate = %q, want %q", got, want)
	}
}

func TestDeadPropsWithoutCreatedTime(t *testing.T) {
	node := &vfs.VNode{
		Name:     "doc.md",
		NodeType: vfs.NodeFile,
	}
	f := &webdavFile{node: node}

	props, err := f.DeadProps()
	if err != nil {
		t.Fatalf("DeadProps() error: %v", err)
	}
	if len(props) != 0 {
		t.Errorf("expected empty props for zero CreatedTime, got %d", len(props))
	}
}

func TestPatchIsForbidden(t *testing.T) {
	node := &vfs.VNode{Name: "doc.md", NodeType: vfs.NodeFile}
	f := &webdavFile{node: node}

	pstats, err := f.Patch([]webdav.Proppatch{})
	if err != nil {
		t.Fatalf("Patch() error: %v", err)
	}
	if len(pstats) != 1 || pstats[0].Status != 403 {
		t.Errorf("expected 403 forbidden propstat, got %+v", pstats)
	}
}
