package doctype

import (
	"context"
	"errors"
	"testing"
)

func TestFileHandlerCreateParsesWrappedData(t *testing.T) {
	h := NewFileHandler(&mockRunner{
		runFn: func(_ context.Context, args ...string) ([]byte, error) {
			return []byte(`{"data":{"file_token":"file-token-123"}}`), nil
		},
	}, t.TempDir())

	token, err := h.Create(context.Background(), "parent", "demo.txt", []byte("hello"))
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if token != "file-token-123" {
		t.Fatalf("Create() token = %q, want file-token-123", token)
	}
}

func TestFileHandlerWriteIsReadOnly(t *testing.T) {
	h := NewFileHandler(&mockRunner{
		runFn: func(_ context.Context, args ...string) ([]byte, error) {
			return nil, errors.New("should not be called")
		},
	}, t.TempDir())

	err := h.Write(context.Background(), "file-token", []byte("hello"))
	if !errors.Is(err, ErrReadOnly) {
		t.Fatalf("Write() error = %v, want ErrReadOnly", err)
	}
}
