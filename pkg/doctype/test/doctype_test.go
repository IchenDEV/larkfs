package doctype_test

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IchenDEV/larkfs/pkg/doctype"
	"github.com/IchenDEV/larkfs/tests/testutil"
)

func TestDocTypeHandlersBlackbox(t *testing.T) {
	runner := &testutil.Runner{RunFn: func(_ context.Context, args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		switch {
		case strings.HasPrefix(joined, "docs +fetch"):
			return []byte(`{"data":{"markdown":"# Hello"}}`), nil
		case strings.HasPrefix(joined, "docs +update"):
			return []byte(`{"ok":true}`), nil
		case strings.HasPrefix(joined, "docs +create"):
			return []byte(`{"data":{"doc_id":"doc_created"}}`), nil
		case strings.HasPrefix(joined, "sheets +info"):
			return []byte(`{"data":{"sheets":{"sheets":[{"sheet_id":"s1","title":"Sheet One"}]}}}`), nil
		case strings.HasPrefix(joined, "sheets +read"):
			return []byte(`{"data":{"valueRange":{"values":[["A","B"],[1,true]]}}}`), nil
		case strings.HasPrefix(joined, "sheets +write"):
			return []byte(`{"ok":true}`), nil
		case strings.HasPrefix(joined, "sheets +create"):
			return []byte(`{"data":{"spreadsheet_token":"shtcn_created"}}`), nil
		case strings.HasPrefix(joined, "base +table-list"):
			return []byte(`{"data":{"items":[{"table_id":"tbl1","table_name":"Tasks"}]}}`), nil
		case strings.HasPrefix(joined, "base +record-list"):
			return []byte(`{"data":{"data":[{"id":"rec1"},{"id":"rec2"}]}}`), nil
		case strings.HasPrefix(joined, "base +record-upsert"):
			return []byte(`{"ok":true}`), nil
		case strings.HasPrefix(joined, "drive metas batch_query"):
			return []byte(`{"data":{"metas":[{"title":"Deck"}]}}`), nil
		case strings.HasPrefix(joined, "drive files delete"):
			return []byte(`{"code":0}`), nil
		default:
			t.Fatalf("unexpected args: %v", args)
			return nil, nil
		}
	}}

	docx := doctype.NewDocxHandler(runner)
	if docx.IsDirectory() || docx.Extension() != ".md" {
		t.Fatal("docx metadata mismatch")
	}
	if list, err := docx.List(context.Background(), "doc"); err != nil || len(list.Entries) != 0 {
		t.Fatalf("List(docx) = %+v, %v", list, err)
	}
	data, err := docx.Read(context.Background(), "doc_1")
	if err != nil || string(data) != "# Hello" {
		t.Fatalf("Read(docx) = %q, %v", data, err)
	}
	if err := docx.Write(context.Background(), "doc_1", []byte("body")); err != nil {
		t.Fatalf("Write(docx) error: %v", err)
	}
	token, err := docx.Create(context.Background(), "folder", "Title", []byte("body"))
	if err != nil || token != "doc_created" {
		t.Fatalf("Create(docx) = %q, %v", token, err)
	}
	if err := docx.Delete(context.Background(), "doc_1"); err != nil {
		t.Fatalf("Delete(docx) error: %v", err)
	}

	sheet := doctype.NewSheetHandler(runner)
	if !sheet.IsDirectory() || sheet.Extension() != ".sheet" {
		t.Fatalf("sheet metadata mismatch")
	}
	list, err := sheet.List(context.Background(), "shtcn")
	if err != nil || len(list.Entries) != 2 || list.Entries[1].Token != "shtcn|s1" {
		t.Fatalf("List(sheet) = %+v, %v", list, err)
	}
	csvData, err := sheet.Read(context.Background(), "shtcn|s1")
	if err != nil || string(csvData) != "A,B\n1,true\n" {
		t.Fatalf("Read(sheet) = %q, %v", csvData, err)
	}
	sheetMeta, err := sheet.Read(context.Background(), "shtcn")
	if err != nil || !strings.Contains(string(sheetMeta), "Sheet One") {
		t.Fatalf("Read(sheet meta) = %s, %v", sheetMeta, err)
	}
	if err := sheet.Write(context.Background(), "shtcn|s1", []byte("A,B\n1,2\n")); err != nil {
		t.Fatalf("Write(sheet) error: %v", err)
	}
	if err := sheet.Write(context.Background(), "shtcn", []byte("A,B\n")); !stderrors.Is(err, doctype.ErrReadOnly) {
		t.Fatalf("Write(sheet meta) error = %v, want ErrReadOnly", err)
	}
	token, err = sheet.Create(context.Background(), "", "Book", nil)
	if err != nil || token != "shtcn_created" {
		t.Fatalf("Create(sheet) = %q, %v", token, err)
	}
	if err := sheet.Delete(context.Background(), "shtcn_created"); err != nil {
		t.Fatalf("Delete(sheet) error: %v", err)
	}

	base := doctype.NewBitableHandler(runner)
	if !base.IsDirectory() || base.Extension() != ".base" {
		t.Fatal("bitable metadata mismatch")
	}
	list, err = base.List(context.Background(), "bascn")
	if err != nil || len(list.Entries) != 2 || list.Entries[1].Name != "Tasks.jsonl" {
		t.Fatalf("List(base) = %+v, %v", list, err)
	}
	baseData, err := base.Read(context.Background(), "bascn|tbl1")
	if err != nil || string(baseData) != "{\"id\":\"rec1\"}\n{\"id\":\"rec2\"}\n" {
		t.Fatalf("Read(base) = %q, %v", baseData, err)
	}
	baseMeta, err := base.Read(context.Background(), "bascn")
	if err != nil || !strings.Contains(string(baseMeta), "Tasks") {
		t.Fatalf("Read(base meta) = %s, %v", baseMeta, err)
	}
	if err := base.Write(context.Background(), "bascn|tbl1", []byte("{\"id\":\"rec1\"}\n\n{\"id\":\"rec2\"}\n")); err != nil {
		t.Fatalf("Write(base) error: %v", err)
	}
	if _, err := base.Create(context.Background(), "", "Base", nil); err == nil {
		t.Fatal("Create(base) expected unsupported error")
	}
	if err := base.Delete(context.Background(), "bascn|tbl1"); err != nil {
		t.Fatalf("Delete(base) error: %v", err)
	}

	readonly := doctype.NewReadonlyHandler(runner, doctype.TypeSlides)
	if readonly.IsDirectory() || readonly.Extension() != ".slides.json" {
		t.Fatal("readonly metadata mismatch")
	}
	if list, err := readonly.List(context.Background(), "sl_1"); err != nil || len(list.Entries) != 0 {
		t.Fatalf("List(readonly) = %+v, %v", list, err)
	}
	readOnlyData, err := readonly.Read(context.Background(), "sl_1")
	if err != nil {
		t.Fatalf("Read(readonly) error: %v", err)
	}
	var got map[string]string
	if err := json.Unmarshal(readOnlyData, &got); err != nil || got["title"] != "Deck" {
		t.Fatalf("Read(readonly) = %s, %v", readOnlyData, err)
	}
	if err := readonly.Write(context.Background(), "sl_1", nil); !stderrors.Is(err, doctype.ErrReadOnly) {
		t.Fatalf("Write(readonly) error = %v, want ErrReadOnly", err)
	}
	if _, err := readonly.Create(context.Background(), "", "", nil); !stderrors.Is(err, doctype.ErrReadOnly) {
		t.Fatalf("Create(readonly) error = %v, want ErrReadOnly", err)
	}
	if err := readonly.Delete(context.Background(), "sl_1"); !stderrors.Is(err, doctype.ErrReadOnly) {
		t.Fatalf("Delete(readonly) error = %v, want ErrReadOnly", err)
	}
}

func TestFileAndFolderHandlersBlackbox(t *testing.T) {
	cacheDir := t.TempDir()
	runner := &testutil.Runner{RunFn: func(_ context.Context, args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		switch {
		case strings.HasPrefix(joined, "drive +download"):
			dest := args[len(args)-1]
			if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				t.Fatalf("MkdirAll() error: %v", err)
			}
			if err := os.WriteFile(dest, []byte("file-body"), 0o644); err != nil {
				t.Fatalf("WriteFile() error: %v", err)
			}
			return []byte(`{"ok":true}`), nil
		case strings.HasPrefix(joined, "drive +upload"):
			return []byte(`{"data":{"file_token":"file_token"}}`), nil
		case strings.HasPrefix(joined, "drive files list"):
			return []byte(`{"data":{"files":[{"token":"doc","name":"Doc","type":"docx","modified_time":1760000000,"created_time":1750000000}],"has_more":true,"next_page_token":"n"}}`), nil
		case strings.HasPrefix(joined, "drive files delete"):
			return []byte(`{"code":0}`), nil
		default:
			t.Fatalf("unexpected args: %v", args)
			return nil, nil
		}
	}}

	file := doctype.NewFileHandler(runner, cacheDir)
	if file.IsDirectory() || file.Extension() != "" {
		t.Fatal("file metadata mismatch")
	}
	if list, err := file.List(context.Background(), "file"); err != nil || len(list.Entries) != 0 {
		t.Fatalf("List(file) = %+v, %v", list, err)
	}
	data, err := file.Read(context.Background(), "file_token")
	if err != nil || string(data) != "file-body" {
		t.Fatalf("Read(file) = %q, %v", data, err)
	}
	if err := file.Write(context.Background(), "file_token", []byte("new")); !stderrors.Is(err, doctype.ErrReadOnly) {
		t.Fatalf("Write(file) error = %v, want ErrReadOnly", err)
	}
	token, err := file.Create(context.Background(), "folder", "Name", []byte("body"))
	if err != nil || token != "file_token" {
		t.Fatalf("Create(file) = %q, %v", token, err)
	}
	if err := file.Delete(context.Background(), "file_token"); err != nil {
		t.Fatalf("Delete(file) error: %v", err)
	}

	folder := doctype.NewFolderHandler(runner)
	if !folder.IsDirectory() || folder.Extension() != "" {
		t.Fatal("folder metadata mismatch")
	}
	list, err := folder.List(context.Background(), "folder")
	if err != nil || len(list.Entries) != 1 || !list.Page.HasMore || list.Page.NextCursor != "n" {
		t.Fatalf("List(folder) = %+v, %v", list, err)
	}
	if _, err := folder.Read(context.Background(), "folder"); !stderrors.Is(err, doctype.ErrReadOnly) {
		t.Fatalf("Read(folder) error = %v, want ErrReadOnly", err)
	}
	if err := folder.Write(context.Background(), "folder", nil); !stderrors.Is(err, doctype.ErrReadOnly) {
		t.Fatalf("Write(folder) error = %v, want ErrReadOnly", err)
	}
	if err := folder.Delete(context.Background(), "folder"); err != nil {
		t.Fatalf("Delete(folder) error: %v", err)
	}

	registry := doctype.NewRegistry(runner, cacheDir)
	if registry.Handler(doctype.TypeDocx) == nil || registry.Handler(doctype.DocType("unknown")) == nil {
		t.Fatal("registry should return handlers for known and fallback types")
	}
	if !doctype.IsReadOnly(doctype.TypeDoc) || !doctype.IsReadOnly(doctype.TypeSlides) || doctype.IsReadOnly(doctype.TypeDocx) {
		t.Fatal("IsReadOnly returned unexpected values")
	}
	if !doctype.IsDirectory(doctype.TypeFolder) || !doctype.IsDirectory(doctype.TypeSheet) || doctype.IsDirectory(doctype.TypeFile) {
		t.Fatal("IsDirectory returned unexpected values")
	}
	if doctype.FileExtension(doctype.TypeDocx) != ".md" || doctype.FileExtension(doctype.TypeBitable) != ".base" || doctype.FileExtension(doctype.TypeFile) != "" {
		t.Fatal("FileExtension returned unexpected values")
	}
}
