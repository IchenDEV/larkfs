package doctype

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/IchenDEV/larkfs/pkg/cli"
)

type FileHandler struct {
	exec     *cli.Executor
	cacheDir string
}

func NewFileHandler(exec *cli.Executor, cacheDir string) *FileHandler {
	return &FileHandler{exec: exec, cacheDir: cacheDir}
}

func (h *FileHandler) IsDirectory() bool { return false }
func (h *FileHandler) Extension() string { return "" }

func (h *FileHandler) List(_ context.Context, _ string) ([]Entry, error) { return nil, nil }

func (h *FileHandler) Read(ctx context.Context, token string) ([]byte, error) {
	tmpDir := filepath.Join(h.cacheDir, "downloads")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return nil, err
	}

	dest := filepath.Join(tmpDir, token)
	_, err := h.exec.Run(ctx, "drive", "+download", "--file-token", token, "--output", dest)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(dest)
	os.Remove(dest)
	return data, err
}

func (h *FileHandler) Write(ctx context.Context, token string, data []byte) error {
	tmpFile := filepath.Join(h.cacheDir, "uploads", token)
	if err := os.MkdirAll(filepath.Dir(tmpFile), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(tmpFile, data, 0o644); err != nil {
		return err
	}
	defer os.Remove(tmpFile)

	_, err := h.exec.Run(ctx, "drive", "+upload", "--file-path", tmpFile)
	return err
}

func (h *FileHandler) Create(ctx context.Context, parentToken string, name string, data []byte) (string, error) {
	tmpFile := filepath.Join(h.cacheDir, "uploads", name)
	if err := os.MkdirAll(filepath.Dir(tmpFile), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(tmpFile, data, 0o644); err != nil {
		return "", err
	}
	defer os.Remove(tmpFile)

	out, err := h.exec.Run(ctx,
		"drive", "+upload",
		"--parent-token", parentToken,
		"--file-path", tmpFile)
	if err != nil {
		return "", err
	}

	var result struct {
		Token string `json:"file_token"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return "", err
	}
	return result.Token, nil
}

func (h *FileHandler) Delete(ctx context.Context, token string) error {
	params := cli.JSONParam(map[string]any{"file_token": token, "type": "file"})
	_, err := h.exec.Run(ctx, "drive", "files", "delete", "--params", params)
	return err
}
