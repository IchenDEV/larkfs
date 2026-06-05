package vfs_test

import (
	"context"
	stderrors "errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/IchenDEV/larkfs/pkg/adapter"
	"github.com/IchenDEV/larkfs/pkg/cache"
	"github.com/IchenDEV/larkfs/pkg/doctype"
	"github.com/IchenDEV/larkfs/pkg/naming"
	"github.com/IchenDEV/larkfs/pkg/vfs"
	"github.com/IchenDEV/larkfs/tests/testutil"
)

func newFullTestOps(t *testing.T, runner *testutil.Runner) *vfs.Operations {
	t.Helper()
	meta := cache.NewMetadataCache(time.Minute)
	t.Cleanup(meta.Close)
	registry := doctype.NewRegistry(runner, t.TempDir())
	namer := naming.NewResolver(t.TempDir())
	return vfs.NewOperations(vfs.OperationsConfig{
		CLI:      runner,
		Tree:     vfs.NewTree([]string{"drive", "wiki", "im", "mail", "calendar", "tasks", "meetings"}),
		Drive:    adapter.NewDriveAdapter(runner, registry, meta, namer),
		Wiki:     adapter.NewWikiAdapter(runner, registry, meta, namer),
		IM:       adapter.NewIMAdapter(runner, meta, namer),
		Mail:     adapter.NewMailAdapter(runner, meta, namer),
		Calendar: adapter.NewCalendarAdapter(runner, meta, namer),
		Task:     adapter.NewTaskAdapter(runner, meta, namer),
		Meeting:  adapter.NewMeetingAdapter(runner, meta, namer, t.TempDir()),
		TTL:      time.Minute,
	})
}

func TestVFSDriveCRUDControlAndRenameBlackbox(t *testing.T) {
	uploadCount := 0
	runner := &testutil.Runner{RunFn: func(_ context.Context, args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		switch {
		case strings.HasPrefix(joined, "drive files list"):
			return []byte(`{"data":{"files":[{"token":"doc_1","name":"Doc","type":"docx"},{"token":"folder_1","name":"Folder","type":"folder"}]}}`), nil
		case strings.HasPrefix(joined, "docs +fetch"):
			return []byte(`{"data":{"markdown":"# Doc"}}`), nil
		case strings.HasPrefix(joined, "docs +update"):
			return []byte(`{"ok":true}`), nil
		case strings.HasPrefix(joined, "docs +create"):
			return []byte(`{"data":{"doc_id":"doc_new"}}`), nil
		case strings.HasPrefix(joined, "drive +upload"):
			uploadCount++
			return []byte(`{"data":{"file_token":"file_uploaded_` + strconv.Itoa(uploadCount) + `"}}`), nil
		case strings.HasPrefix(joined, "drive +create-folder"):
			return []byte(`{"data":{"token":"folder_new"}}`), nil
		case strings.HasPrefix(joined, "drive files patch"):
			return []byte(`{"code":0}`), nil
		case strings.HasPrefix(joined, "drive +delete"):
			return []byte(`{"code":0}`), nil
		case strings.HasPrefix(joined, "drive +move"):
			return []byte(`{"ok":true}`), nil
		case strings.HasPrefix(joined, "docs +search"):
			return []byte(`{"ok":true}`), nil
		default:
			t.Fatalf("unexpected args: %v", args)
			return nil, nil
		}
	}}
	ops := newFullTestOps(t, runner)

	children, err := ops.ReadDir(context.Background(), "/drive")
	if err != nil || len(children) < 2 {
		t.Fatalf("ReadDir(/drive) = %+v, %v", children, err)
	}
	data, err := ops.Read(context.Background(), "/drive/Doc.md")
	if err != nil || string(data) != "# Doc" {
		t.Fatalf("Read() = %q, %v", data, err)
	}
	if err := ops.Write(context.Background(), "/drive/Doc.md", []byte("updated")); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	created, err := ops.Create(context.Background(), "/drive/New.md")
	if err != nil || created.Token != "doc_new" {
		t.Fatalf("Create() = %+v, %v", created, err)
	}
	dir, err := ops.Mkdir(context.Background(), "/drive/NewFolder")
	if err != nil || dir.Token != "folder_new" {
		t.Fatalf("Mkdir() = %+v, %v", dir, err)
	}
	uploaded, err := ops.Create(context.Background(), "/drive/blob.bin")
	if err != nil {
		t.Fatalf("Create(file) error: %v", err)
	}
	if uploaded.Token != "" || !uploaded.PendingCreate || uploaded.DocType != doctype.TypeFile {
		t.Fatalf("Create(file) = %+v", uploaded)
	}
	replaceTemplate, err := ops.Read(context.Background(), "/drive/blob.bin._replace.request.json")
	if err != nil || !strings.Contains(string(replaceTemplate), `"target_path": "/drive/blob.bin"`) {
		t.Fatalf("Read(replace template) = %s, %v", replaceTemplate, err)
	}
	if err := ops.Write(context.Background(), "/drive/blob.bin", []byte("payload")); err != nil {
		t.Fatalf("Write(file) error: %v", err)
	}
	if uploaded.Token != "file_uploaded_1" || uploaded.PendingCreate {
		t.Fatalf("uploaded file state = %+v", uploaded)
	}
	if err := ops.Write(context.Background(), "/drive/blob.bin", []byte("payload-2")); !stderrors.Is(err, vfs.ErrUnsupported) {
		t.Fatalf("Write(existing file) error = %v, want ErrUnsupported", err)
	}
	result, err := ops.ExecuteOp(context.Background(), "/drive/blob.bin._replace.request.json", []byte(`{"data":{"content":"payload-3"}}`))
	if err != nil {
		t.Fatalf("ExecuteOp(replace) error: %v", err)
	}
	if !strings.Contains(string(result), `"old_token": "file_uploaded_1"`) || !strings.Contains(string(result), `"new_token": "file_uploaded_2"`) {
		t.Fatalf("replace result = %s", result)
	}
	if uploaded.Token != "file_uploaded_2" {
		t.Fatalf("replace should update in-memory token, got %+v", uploaded)
	}
	if siblingResult, err := ops.Read(context.Background(), "/drive/blob.bin._replace.result.json"); err != nil || !strings.Contains(string(siblingResult), `"new_token": "file_uploaded_2"`) {
		t.Fatalf("Read(replace result) = %s, %v", siblingResult, err)
	}
	if err := ops.Rename(context.Background(), "/drive/blob.bin", "/drive/blob-v2.bin"); err != nil {
		t.Fatalf("Rename(file basename) error: %v", err)
	}
	if ops.Tree().Resolve("/drive/blob.bin._replace.request.json") != nil {
		t.Fatal("old replace helper path should be removed after rename")
	}
	if ops.Tree().Resolve("/drive/blob-v2.bin._replace.request.json") == nil {
		t.Fatal("renamed file should get a new replace helper path")
	}
	if err := ops.Rename(context.Background(), "/drive/Doc.md", "/drive/Renamed.md"); err != nil {
		t.Fatalf("Rename(basename) error: %v", err)
	}
	if ops.Tree().Resolve("/drive/Renamed.md") == nil {
		t.Fatal("renamed doc should be addressable at new path")
	}
	if err := ops.Rename(context.Background(), "/drive/Renamed.md", "/drive/Folder/Renamed.md"); err != nil {
		t.Fatalf("Rename(move) error: %v", err)
	}
	if got := testutil.JoinArgs(runner.Calls[len(runner.Calls)-1]); !strings.Contains(got, "drive +move") {
		t.Fatalf("move args = %q", got)
	}
	if err := ops.Remove(context.Background(), "/drive/New.md"); err != nil {
		t.Fatalf("Remove() error: %v", err)
	}
	if _, err := ops.ReadDir(context.Background(), "/missing"); !stderrors.Is(err, vfs.ErrNotFound) {
		t.Fatalf("ReadDir(missing) error = %v, want ErrNotFound", err)
	}

	if _, err := ops.Create(context.Background(), "/drive/_ops/accidental.md"); !stderrors.Is(err, vfs.ErrUnsupported) {
		t.Fatalf("Create under control path error = %v, want ErrUnsupported", err)
	}
	if _, err := ops.ReadDir(context.Background(), "/drive/_ops"); err != nil {
		t.Fatalf("ReadDir(_ops) error: %v", err)
	}
}

func TestVFSControlQueriesAndStaticViewsBlackbox(t *testing.T) {
	runner := &testutil.Runner{Out: []byte(`{"data":[{"name":"Alice"}]}`)}
	ops := vfs.NewOperations(vfs.OperationsConfig{
		CLI:  runner,
		Tree: vfs.NewTree([]string{"contact", "docs", "meetings", "base", "sheets", "vc", "event", "markdown", "okr", "slides", "whiteboard", "_system"}),
		TTL:  time.Minute,
	})

	if _, err := ops.ReadDir(context.Background(), "/contact"); err != nil {
		t.Fatalf("ReadDir(/contact) error: %v", err)
	}
	if err := ops.Write(context.Background(), "/contact/_queries/search-user.request.json", []byte(`{"query":"Alice"}`)); err != nil {
		t.Fatalf("Write(search-user) error: %v", err)
	}
	if got := testutil.JoinArgs(runner.LastArgs); got != "contact +search-user --query Alice" {
		t.Fatalf("runner args = %q", got)
	}
	result, err := ops.Read(context.Background(), "/contact/_views/search-user/results.json")
	if err != nil || len(result) == 0 {
		t.Fatalf("Read(view result) = %s, %v", result, err)
	}
	query, err := ops.RunQuery(context.Background(), "/docs/_queries/fetch.request.json", []byte(`{"flags":{"doc-token":"doc_1"},"params":{"revision":"latest"},"data":{"format":"md"}}`))
	if err != nil || len(query) == 0 {
		t.Fatalf("RunQuery(fetch) = %s, %v", query, err)
	}
	gotArgs := testutil.JoinArgs(runner.LastArgs)
	for _, want := range []string{"docs +fetch", "--params {\"revision\":\"latest\"}", "--data {\"format\":\"md\"}", "--doc-token doc_1"} {
		if !strings.Contains(gotArgs, want) {
			t.Fatalf("runner args = %q, want to contain %q", gotArgs, want)
		}
	}
	opResult, err := ops.ExecuteOp(context.Background(), "/_system/_ops/exec.request.json", []byte(`{"args":["schema","drive.files.list"]}`))
	if err != nil || len(opResult) == 0 {
		t.Fatalf("ExecuteOp() = %s, %v", opResult, err)
	}
	if got := testutil.JoinArgs(runner.LastArgs); got != "schema drive.files.list" {
		t.Fatalf("system exec args = %q", got)
	}
	view, err := ops.ListView(context.Background(), "/contact/_views/search-user")
	if err != nil || len(view) != 1 {
		t.Fatalf("ListView() = %+v, %v", view, err)
	}
	for _, root := range []string{"/base", "/sheets", "/vc", "/meetings", "/event", "/markdown", "/okr", "/slides", "/whiteboard"} {
		children, err := ops.ReadDir(context.Background(), root)
		if err != nil || len(children) == 0 {
			t.Fatalf("ReadDir(%s) = %+v, %v", root, children, err)
		}
	}

	if _, err := ops.RunQuery(context.Background(), "/markdown/_queries/fetch.request.json", []byte(`{"flags":{"file-token":"md_token"}}`)); err != nil {
		t.Fatalf("RunQuery(markdown fetch) error: %v", err)
	}
	if got := testutil.JoinArgs(runner.LastArgs); got != "markdown +fetch --file-token md_token" {
		t.Fatalf("markdown fetch args = %q", got)
	}
	if _, err := ops.ExecuteOp(context.Background(), "/event/_ops/consume.request.json", []byte(`{"args":["event","consume","im.message.receive_v1","--max-events","1"]}`)); err != nil {
		t.Fatalf("ExecuteOp(event consume) error: %v", err)
	}
}

func TestVFSReadOnlyAndUnsupportedBlackbox(t *testing.T) {
	ops := vfs.NewOperations(vfs.OperationsConfig{
		CLI:      &testutil.Runner{},
		Tree:     vfs.NewTree([]string{"docs"}),
		ReadOnly: true,
		TTL:      time.Minute,
	})
	if err := ops.Write(context.Background(), "/docs/_ops/exec.request.json", []byte(`{}`)); !stderrors.Is(err, vfs.ErrReadOnly) {
		t.Fatalf("Write(read-only) error = %v, want ErrReadOnly", err)
	}
	if _, err := ops.Create(context.Background(), "/docs/file.md"); !stderrors.Is(err, vfs.ErrReadOnly) {
		t.Fatalf("Create(read-only) error = %v, want ErrReadOnly", err)
	}
	if _, err := ops.Mkdir(context.Background(), "/docs/folder"); !stderrors.Is(err, vfs.ErrReadOnly) {
		t.Fatalf("Mkdir(read-only) error = %v, want ErrReadOnly", err)
	}

	writable := vfs.NewOperations(vfs.OperationsConfig{
		CLI:  &testutil.Runner{},
		Tree: vfs.NewTree([]string{"docs"}),
		TTL:  time.Minute,
	})
	if _, err := writable.Create(context.Background(), "/docs/file.md"); !stderrors.Is(err, vfs.ErrUnsupported) {
		t.Fatalf("Create(unsupported) error = %v, want ErrUnsupported", err)
	}
	if err := writable.Remove(context.Background(), "/docs/search"); !stderrors.Is(err, vfs.ErrUnsupported) {
		t.Fatalf("Remove(unsupported) error = %v, want ErrUnsupported", err)
	}
}
