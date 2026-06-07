package doctype

import (
	"context"
	"encoding/json"
	"html"

	"github.com/IchenDEV/larkfs/pkg/cli"
)

type DocxHandler struct {
	exec cli.Runner
}

func NewDocxHandler(exec cli.Runner) *DocxHandler {
	return &DocxHandler{exec: exec}
}

func (h *DocxHandler) IsDirectory() bool { return false }
func (h *DocxHandler) Extension() string { return ".md" }

func (h *DocxHandler) List(_ context.Context, _ string) (ListResult, error) {
	return ListResult{}, nil
}

func (h *DocxHandler) Read(ctx context.Context, token string) ([]byte, error) {
	out, err := h.exec.Run(ctx,
		"docs", "+fetch",
		"--api-version", "v2",
		"--doc", token,
		"--doc-format", "markdown",
		"--format", "json")
	if err != nil {
		return nil, err
	}
	return docxMarkdown(out)
}

func docxMarkdown(out []byte) ([]byte, error) {
	var result struct {
		Data struct {
			Markdown string `json:"markdown"`
			Content  string `json:"content"`
			Text     string `json:"text"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, err
	}
	switch {
	case result.Data.Markdown != "":
		return []byte(result.Data.Markdown), nil
	case result.Data.Content != "":
		return []byte(result.Data.Content), nil
	case result.Data.Text != "":
		return []byte(result.Data.Text), nil
	default:
		return nil, nil
	}
}

func (h *DocxHandler) Write(ctx context.Context, token string, data []byte) error {
	_, err := h.exec.Run(ctx,
		"docs", "+update",
		"--api-version", "v2",
		"--doc", token,
		"--command", "overwrite",
		"--doc-format", "markdown",
		"--content", string(data),
	)
	return err
}

func (h *DocxHandler) Create(ctx context.Context, parentToken string, name string, data []byte) (string, error) {
	out, err := h.exec.Run(ctx,
		"docs", "+create",
		"--api-version", "v2",
		"--parent-token", parentToken,
		"--doc-format", "markdown",
		"--content", docxCreateContent(name, data),
	)
	if err != nil {
		return "", err
	}
	var result struct {
		Data struct {
			DocumentID string `json:"document_id"`
			DocID      string `json:"doc_id"`
			Token      string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return "", err
	}
	if result.Data.DocumentID != "" {
		return result.Data.DocumentID, nil
	}
	if result.Data.DocID != "" {
		return result.Data.DocID, nil
	}
	return result.Data.Token, nil
}

func docxCreateContent(name string, data []byte) string {
	title := html.EscapeString(name)
	if len(data) == 0 {
		return "<title>" + title + "</title>\n"
	}
	return "<title>" + title + "</title>\n" + string(data)
}

func (h *DocxHandler) Delete(ctx context.Context, token string) error {
	return deleteDriveResource(ctx, h.exec, token, TypeDocx)
}
