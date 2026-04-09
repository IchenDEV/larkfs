package doctype

import (
	"context"
	"encoding/json"

	"github.com/IchenDEV/larkfs/pkg/cli"
)

type DocxHandler struct {
	exec *cli.Executor
}

func NewDocxHandler(exec *cli.Executor) *DocxHandler {
	return &DocxHandler{exec: exec}
}

func (h *DocxHandler) IsDirectory() bool { return false }
func (h *DocxHandler) Extension() string { return ".md" }

func (h *DocxHandler) List(_ context.Context, _ string) ([]Entry, error) {
	return nil, nil
}

func (h *DocxHandler) Read(ctx context.Context, token string) ([]byte, error) {
	out, err := h.exec.Run(ctx, "docs", "+fetch", "--doc", token, "--format", "json")
	if err != nil {
		return nil, err
	}
	var result struct {
		Data struct {
			Markdown string `json:"markdown"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, err
	}
	return []byte(result.Data.Markdown), nil
}

func (h *DocxHandler) Write(ctx context.Context, token string, data []byte) error {
	_, err := h.exec.Run(ctx,
		"docs", "+update",
		"--doc", token,
		"--mode", "overwrite",
		"--markdown", string(data),
	)
	return err
}

func (h *DocxHandler) Create(ctx context.Context, parentToken string, name string, data []byte) (string, error) {
	out, err := h.exec.Run(ctx,
		"docs", "+create",
		"--title", name,
		"--folder-token", parentToken,
		"--markdown", string(data),
	)
	if err != nil {
		return "", err
	}
	var result struct {
		Data struct {
			Token string `json:"doc_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return "", err
	}
	return result.Data.Token, nil
}

func (h *DocxHandler) Delete(ctx context.Context, token string) error {
	params := cli.JSONParam(map[string]any{"file_token": token, "type": "docx"})
	_, err := h.exec.Run(ctx, "drive", "files", "delete", "--params", params)
	return err
}
