package doctype

import (
	"context"
	"encoding/json"

	"github.com/IchenDEV/larkfs/pkg/cli"
)

type ReadonlyHandler struct {
	exec    *cli.Executor
	docType DocType
}

func NewReadonlyHandler(exec *cli.Executor, docType DocType) *ReadonlyHandler {
	return &ReadonlyHandler{exec: exec, docType: docType}
}

func (h *ReadonlyHandler) IsDirectory() bool { return false }

func (h *ReadonlyHandler) Extension() string {
	return FileExtension(h.docType)
}

func (h *ReadonlyHandler) List(_ context.Context, _ string) ([]Entry, error) { return nil, nil }

func (h *ReadonlyHandler) Read(ctx context.Context, token string) ([]byte, error) {
	reqDoc := map[string]any{"doc_token": token, "doc_type": string(h.docType)}
	dataParam := cli.JSONParam(map[string]any{"request_docs": []any{reqDoc}})

	out, err := h.exec.Run(ctx,
		"drive", "metas", "batch_query",
		"--data", dataParam,
		"--format", "json")
	if err != nil {
		return nil, err
	}

	var result struct {
		Data struct {
			Metas []json.RawMessage `json:"metas"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, err
	}
	if len(result.Data.Metas) > 0 {
		return result.Data.Metas[0], nil
	}
	return []byte("{}"), nil
}

func (h *ReadonlyHandler) Write(_ context.Context, _ string, _ []byte) error {
	return ErrReadOnly
}

func (h *ReadonlyHandler) Create(_ context.Context, _ string, _ string, _ []byte) (string, error) {
	return "", ErrReadOnly
}

func (h *ReadonlyHandler) Delete(_ context.Context, _ string) error {
	return ErrReadOnly
}
