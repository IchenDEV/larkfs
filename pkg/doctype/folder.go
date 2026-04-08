package doctype

import (
	"context"
	"encoding/json"

	"github.com/IchenDEV/larkfs/pkg/cli"
)

type FolderHandler struct {
	exec *cli.Executor
}

func NewFolderHandler(exec *cli.Executor) *FolderHandler {
	return &FolderHandler{exec: exec}
}

func (h *FolderHandler) IsDirectory() bool { return true }
func (h *FolderHandler) Extension() string { return "" }

func (h *FolderHandler) List(ctx context.Context, token string) ([]Entry, error) {
	params := cli.JSONParam(map[string]any{"folder_token": token})
	out, err := h.exec.Run(ctx,
		"drive", "files", "list",
		"--params", params,
		"--format", "json", "--page-all")
	if err != nil {
		return nil, err
	}

	var result struct {
		Data struct {
			Files []struct {
				Token string `json:"token"`
				Name  string `json:"name"`
				Type  string `json:"type"`
			} `json:"files"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, err
	}

	entries := make([]Entry, 0, len(result.Data.Files))
	for _, f := range result.Data.Files {
		dt := DocType(f.Type)
		entries = append(entries, Entry{
			Name:  f.Name,
			Token: f.Token,
			Type:  dt,
			IsDir: IsDirectory(dt),
		})
	}
	return entries, nil
}

func (h *FolderHandler) Read(_ context.Context, _ string) ([]byte, error) {
	return nil, ErrReadOnly
}

func (h *FolderHandler) Write(_ context.Context, _ string, _ []byte) error {
	return ErrReadOnly
}

func (h *FolderHandler) Create(ctx context.Context, parentToken string, name string, _ []byte) (string, error) {
	params := cli.JSONParam(map[string]any{"folder_token": parentToken})
	data := cli.JSONParam(map[string]any{"name": name})
	out, err := h.exec.Run(ctx,
		"drive", "files", "create_folder",
		"--params", params,
		"--data", data)
	if err != nil {
		return "", err
	}
	var result struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return "", err
	}
	return result.Token, nil
}

func (h *FolderHandler) Delete(ctx context.Context, token string) error {
	params := cli.JSONParam(map[string]any{"file_token": token, "type": "folder"})
	_, err := h.exec.Run(ctx, "drive", "files", "delete", "--params", params)
	return err
}
