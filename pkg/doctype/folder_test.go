package doctype

import (
	"context"
	"testing"
)

func TestFolderHandlerCreateParsesWrappedData(t *testing.T) {
	h := NewFolderHandler(&mockRunner{
		runFn: func(_ context.Context, args ...string) ([]byte, error) {
			return []byte(`{"data":{"token":"folder-token-123"}}`), nil
		},
	})

	token, err := h.Create(context.Background(), "parent", "demo", nil)
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if token != "folder-token-123" {
		t.Fatalf("Create() token = %q, want folder-token-123", token)
	}
}
