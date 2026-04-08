package doctype

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/IchenDEV/larkfs/pkg/cli"
)

type BitableHandler struct {
	exec *cli.Executor
}

func NewBitableHandler(exec *cli.Executor) *BitableHandler {
	return &BitableHandler{exec: exec}
}

func (h *BitableHandler) IsDirectory() bool { return true }
func (h *BitableHandler) Extension() string { return ".base" }

func (h *BitableHandler) List(ctx context.Context, token string) ([]Entry, error) {
	out, err := h.exec.Run(ctx,
		"base", "+table-list", "--base-token", token, "--format", "json")
	if err != nil {
		return nil, err
	}

	var result struct {
		Data struct {
			Items []struct {
				TableID string `json:"table_id"`
				Name    string `json:"name"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, err
	}

	entries := make([]Entry, 0, len(result.Data.Items)+1)
	entries = append(entries, Entry{Name: "_meta.json", Token: token, Type: TypeFile})
	for _, t := range result.Data.Items {
		entries = append(entries, Entry{
			Name:  t.Name + ".jsonl",
			Token: token + "|" + t.TableID,
			Type:  TypeFile,
		})
	}
	return entries, nil
}

func (h *BitableHandler) Read(ctx context.Context, token string) ([]byte, error) {
	parts := strings.SplitN(token, "|", 2)
	if len(parts) != 2 {
		return h.readMeta(ctx, parts[0])
	}

	baseToken, tableID := parts[0], parts[1]
	out, err := h.exec.Run(ctx,
		"base", "+record-list",
		"--base-token", baseToken,
		"--table-id", tableID,
		"--format", "json")
	if err != nil {
		return nil, err
	}

	var result struct {
		Data struct {
			Items []json.RawMessage `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	for _, item := range result.Data.Items {
		buf.Write(item)
		buf.WriteByte('\n')
	}
	return buf.Bytes(), nil
}

func (h *BitableHandler) Write(ctx context.Context, token string, data []byte) error {
	parts := strings.SplitN(token, "|", 2)
	if len(parts) != 2 {
		return ErrReadOnly
	}

	baseToken, tableID := parts[0], parts[1]
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		_, err := h.exec.Run(ctx,
			"base", "+record-upsert",
			"--base-token", baseToken,
			"--table-id", tableID,
			"--json", string(line))
		if err != nil {
			return fmt.Errorf("upsert record: %w", err)
		}
	}
	return scanner.Err()
}

func (h *BitableHandler) Create(_ context.Context, _ string, _ string, _ []byte) (string, error) {
	return "", fmt.Errorf("bitable creation via VFS not supported")
}

func (h *BitableHandler) Delete(ctx context.Context, token string) error {
	parts := strings.SplitN(token, "|", 2)
	params := cli.JSONParam(map[string]any{"file_token": parts[0], "type": "bitable"})
	_, err := h.exec.Run(ctx, "drive", "files", "delete", "--params", params)
	return err
}

func (h *BitableHandler) readMeta(ctx context.Context, token string) ([]byte, error) {
	return h.exec.Run(ctx, "base", "+table-list", "--base-token", token, "--format", "json")
}
