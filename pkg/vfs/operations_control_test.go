package vfs

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/IchenDEV/larkfs/pkg/doctype"
)

type mockRunner struct {
	lastArgs []string
	out      []byte
}

func (m *mockRunner) Path() string { return "mock-lark-cli" }

func (m *mockRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	m.lastArgs = append([]string(nil), args...)
	return append([]byte(nil), m.out...), nil
}

func TestControlQueryWritesResults(t *testing.T) {
	runner := &mockRunner{out: []byte(`{"data":[{"name":"Alice"}]}`)}
	ops := NewOperations(OperationsConfig{
		CLI:  runner,
		Tree: NewTree([]string{"contact"}),
		TTL:  time.Minute,
	})

	if _, err := ops.ReadDir(context.Background(), "/contact"); err != nil {
		t.Fatalf("ReadDir(/contact) error: %v", err)
	}

	reqPath := "/contact/_queries/search-user.request.json"
	if err := ops.Write(context.Background(), reqPath, []byte(`{"query":"Alice"}`)); err != nil {
		t.Fatalf("Write(%s) error: %v", reqPath, err)
	}

	gotArgs := stringsJoin(runner.lastArgs)
	wantArgs := "contact +search-user --query Alice"
	if gotArgs != wantArgs {
		t.Fatalf("runner args = %q, want %q", gotArgs, wantArgs)
	}

	result, err := ops.Read(context.Background(), "/contact/_views/search-user/results.json")
	if err != nil {
		t.Fatalf("Read(view result) error: %v", err)
	}
	if len(result) == 0 {
		t.Fatal("expected non-empty view result")
	}
}

func TestQueryRequestUsesFlagsParamsAndData(t *testing.T) {
	runner := &mockRunner{out: []byte(`{"ok":true}`)}
	ops := NewOperations(OperationsConfig{
		CLI:  runner,
		Tree: NewTree([]string{"docs"}),
		TTL:  time.Minute,
	})

	if _, err := ops.ReadDir(context.Background(), "/docs/_queries"); err != nil {
		t.Fatalf("ReadDir(/docs/_queries) error: %v", err)
	}

	reqPath := "/docs/_queries/fetch.request.json"
	payload := []byte(`{"flags":{"doc-token":"doc_1"},"params":{"revision":"latest"},"data":{"format":"md"}}`)
	if err := ops.Write(context.Background(), reqPath, payload); err != nil {
		t.Fatalf("Write(%s) error: %v", reqPath, err)
	}

	gotArgs := stringsJoin(runner.lastArgs)
	wantParts := []string{
		"docs +fetch",
		"--params {\"revision\":\"latest\"}",
		"--data {\"format\":\"md\"}",
		"--doc-token doc_1",
	}
	for _, want := range wantParts {
		if !strings.Contains(gotArgs, want) {
			t.Fatalf("runner args = %q, want to contain %q", gotArgs, want)
		}
	}
}

func TestSystemExecPrefixesNothing(t *testing.T) {
	runner := &mockRunner{out: []byte(`{"version":"1.0.7"}`)}
	ops := NewOperations(OperationsConfig{
		CLI:  runner,
		Tree: NewTree([]string{"_system"}),
		TTL:  time.Minute,
	})

	if _, err := ops.ReadDir(context.Background(), "/_system"); err != nil {
		t.Fatalf("ReadDir(/_system) error: %v", err)
	}

	reqPath := "/_system/_ops/exec.request.json"
	if err := ops.Write(context.Background(), reqPath, []byte(`{"args":["schema","drive.files.list"]}`)); err != nil {
		t.Fatalf("Write(%s) error: %v", reqPath, err)
	}

	gotArgs := stringsJoin(runner.lastArgs)
	wantArgs := "schema drive.files.list"
	if gotArgs != wantArgs {
		t.Fatalf("runner args = %q, want %q", gotArgs, wantArgs)
	}
}

func TestActionRequestUsesDomainTemplate(t *testing.T) {
	runner := &mockRunner{out: []byte(`{"ok":true}`)}
	ops := NewOperations(OperationsConfig{
		CLI:  runner,
		Tree: NewTree([]string{"mail"}),
		TTL:  time.Minute,
	})

	if _, err := ops.ReadDir(context.Background(), "/mail/_ops"); err != nil {
		t.Fatalf("ReadDir(/mail/_ops) error: %v", err)
	}

	reqPath := "/mail/_ops/reply.request.json"
	if err := ops.Write(context.Background(), reqPath, []byte(`{"flags":{"message-id":"om_1","body":"hi","confirm-send":true}}`)); err != nil {
		t.Fatalf("Write(%s) error: %v", reqPath, err)
	}

	gotArgs := stringsJoin(runner.lastArgs)
	wantArgs := "mail +reply --body hi --confirm-send --message-id om_1"
	if gotArgs != wantArgs {
		t.Fatalf("runner args = %q, want %q", gotArgs, wantArgs)
	}
}

func TestRequestTemplateIncludesBaseArgs(t *testing.T) {
	ops := NewOperations(OperationsConfig{
		CLI:  &mockRunner{},
		Tree: NewTree([]string{"docs"}),
		TTL:  time.Minute,
	})

	if _, err := ops.ReadDir(context.Background(), "/docs"); err != nil {
		t.Fatalf("ReadDir(/docs) error: %v", err)
	}
	template, err := ops.Read(context.Background(), "/docs/_queries/search.request.json")
	if err != nil {
		t.Fatalf("Read(template) error: %v", err)
	}
	if !strings.Contains(string(template), `"base_args"`) {
		t.Fatalf("expected template to include base_args, got %s", template)
	}
}

func TestStaticDomainRootEntries(t *testing.T) {
	ops := NewOperations(OperationsConfig{
		CLI:  &mockRunner{},
		Tree: NewTree([]string{"base", "sheets", "vc"}),
		TTL:  time.Minute,
	})

	for _, root := range []string{"/base", "/sheets", "/vc"} {
		children, err := ops.ReadDir(context.Background(), root)
		if err != nil {
			t.Fatalf("ReadDir(%s) error: %v", root, err)
		}
		if len(children) == 0 {
			t.Fatalf("expected %s to expose static resource entries", root)
		}
	}
}

func TestCreateUnderControlPathIsRejected(t *testing.T) {
	ops := NewOperations(OperationsConfig{
		CLI:  &mockRunner{},
		Tree: NewTree([]string{"drive"}),
		TTL:  time.Minute,
	})

	_, err := ops.Create(context.Background(), "/drive/_ops/accidental.md")
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("Create under control path error = %v, want ErrUnsupported", err)
	}
}

func TestRenameDoesNotFakeRemoteBasenameChange(t *testing.T) {
	tree := NewTree([]string{"drive"})
	driveRoot := tree.DomainNode("drive")
	driveRoot.AddChild(&VNode{
		Name:       "a.md",
		Token:      "doc_1",
		DocType:    doctype.TypeDocx,
		NodeType:   NodeFile,
		Kind:       NodeKindResource,
		Domain:     "drive",
		TargetPath: "/drive/a.md",
		children:   make(map[string]*VNode),
	})
	ops := NewOperations(OperationsConfig{
		CLI:  &mockRunner{},
		Tree: tree,
		TTL:  time.Minute,
	})

	err := ops.Rename(context.Background(), "/drive/a.md", "/drive/b.md")
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("Rename basename change error = %v, want ErrUnsupported", err)
	}
	if tree.Resolve("/drive/a.md") == nil {
		t.Fatal("expected original node to remain after rejected rename")
	}
	if tree.Resolve("/drive/b.md") != nil {
		t.Fatal("unexpected local-only rename after rejected remote rename")
	}
}

func stringsJoin(parts []string) string {
	result := ""
	for i, part := range parts {
		if i > 0 {
			result += " "
		}
		result += part
	}
	return result
}
