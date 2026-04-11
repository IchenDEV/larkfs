package mount

import (
	"context"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/IchenDEV/larkfs/pkg/vfs"
)

type mockRunner struct {
	lastArgs []string
	out      []byte
}

func TestFUSEErrnoFromError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want syscall.Errno
	}{
		{name: "read only", err: vfs.ErrReadOnly, want: syscall.EROFS},
		{name: "not found", err: vfs.ErrNotFound, want: syscall.ENOENT},
		{name: "unsupported", err: vfs.ErrUnsupported, want: syscall.ENOTSUP},
	}

	for _, tt := range tests {
		if got := errnoFromError(tt.err); got != tt.want {
			t.Fatalf("%s: errnoFromError() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func (m *mockRunner) Path() string { return "mock-lark-cli" }

func (m *mockRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	m.lastArgs = append([]string(nil), args...)
	return append([]byte(nil), m.out...), nil
}

func TestWebDAVBuffersWritesUntilClose(t *testing.T) {
	runner := &mockRunner{out: []byte(`{"ok":true}`)}
	ops := vfs.NewOperations(vfs.OperationsConfig{
		CLI:  runner,
		Tree: vfs.NewTree([]string{"contact"}),
		TTL:  time.Minute,
	})
	fs := &webdavFS{ops: ops}

	if _, err := ops.ReadDir(context.Background(), "/contact"); err != nil {
		t.Fatalf("ReadDir(/contact) error: %v", err)
	}

	file, err := fs.OpenFile(context.Background(), "/contact/_queries/search-user.request.json", os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		t.Fatalf("OpenFile() error: %v", err)
	}

	if _, err := file.Write([]byte(`{"query":"Alice"}`)); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	if len(runner.lastArgs) != 0 {
		t.Fatalf("expected runner not to execute before Close, got %v", runner.lastArgs)
	}

	if err := file.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}
	if got := len(runner.lastArgs); got == 0 {
		t.Fatal("expected runner to execute on Close")
	}
}
