package adapter

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/IchenDEV/larkfs/pkg/cache"
)

type mockRunner struct {
	runFn func(context.Context, ...string) ([]byte, error)
}

func (m *mockRunner) Path() string { return "mock-lark-cli" }

func (m *mockRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	return m.runFn(ctx, args...)
}

func TestIMListChatFilesReturnsRunnerError(t *testing.T) {
	want := errors.New("runner failed")
	adapter := NewIMAdapter(&mockRunner{
		runFn: func(_ context.Context, args ...string) ([]byte, error) {
			return nil, want
		},
	}, cache.NewMetadataCache(time.Minute), nil)

	_, err := adapter.ListChatFiles(context.Background(), "chat-1")
	if !errors.Is(err, want) {
		t.Fatalf("ListChatFiles() error = %v, want %v", err, want)
	}
}
