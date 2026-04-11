package doctype

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/IchenDEV/larkfs/pkg/cli"
)

type SheetHandler struct {
	exec cli.Runner
}

func NewSheetHandler(exec cli.Runner) *SheetHandler {
	return &SheetHandler{exec: exec}
}

func (h *SheetHandler) IsDirectory() bool { return true }
func (h *SheetHandler) Extension() string { return ".sheet" }

func (h *SheetHandler) List(ctx context.Context, token string) (ListResult, error) {
	out, err := h.exec.Run(ctx,
		"sheets", "+info", "--spreadsheet-token", token)
	if err != nil {
		return ListResult{}, err
	}

	var result struct {
		Data struct {
			Sheets struct {
				Sheets []struct {
					SheetID string `json:"sheet_id"`
					Title   string `json:"title"`
				} `json:"sheets"`
			} `json:"sheets"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return ListResult{}, err
	}

	sheets := result.Data.Sheets.Sheets
	entries := make([]Entry, 0, len(sheets)+1)
	entries = append(entries, Entry{Name: "_meta.json", Token: token, Type: TypeFile})
	for _, s := range sheets {
		entries = append(entries, Entry{
			Name:  s.Title + ".csv",
			Token: token + "|" + s.SheetID,
			Type:  TypeFile,
		})
	}
	return ListResult{
		Entries: entries,
		Page: PageInfo{
			WindowSize: len(entries),
			SortKey:    "sheet_order",
		},
	}, nil
}

func (h *SheetHandler) Read(ctx context.Context, token string) ([]byte, error) {
	parts := strings.SplitN(token, "|", 2)
	if len(parts) != 2 {
		return h.readMeta(ctx, parts[0])
	}

	spreadsheetToken, sheetID := parts[0], parts[1]
	out, err := h.exec.Run(ctx,
		"sheets", "+read",
		"--spreadsheet-token", spreadsheetToken,
		"--range", sheetID)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data struct {
			ValueRange struct {
				Values [][]any `json:"values"`
			} `json:"valueRange"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, err
	}
	return valuesToCSV(result.Data.ValueRange.Values)
}

func (h *SheetHandler) Write(ctx context.Context, token string, data []byte) error {
	parts := strings.SplitN(token, "|", 2)
	if len(parts) != 2 {
		return ErrReadOnly
	}

	rows, err := csvToValues(data)
	if err != nil {
		return fmt.Errorf("parse CSV: %w", err)
	}

	valJSON, _ := json.Marshal(rows)
	_, err = h.exec.Run(ctx,
		"sheets", "+write",
		"--spreadsheet-token", parts[0],
		"--range", parts[1]+"!A1",
		"--values", string(valJSON))
	return err
}

func (h *SheetHandler) Create(ctx context.Context, _ string, name string, _ []byte) (string, error) {
	out, err := h.exec.Run(ctx, "sheets", "+create", "--title", name)
	if err != nil {
		return "", err
	}
	var result struct {
		Data struct {
			Token string `json:"spreadsheet_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return "", err
	}
	return result.Data.Token, nil
}

func (h *SheetHandler) Delete(ctx context.Context, token string) error {
	params := cli.JSONParam(map[string]any{"file_token": token, "type": "sheet"})
	_, err := h.exec.Run(ctx, "drive", "files", "delete", "--params", params)
	return err
}

func (h *SheetHandler) readMeta(ctx context.Context, token string) ([]byte, error) {
	return h.exec.Run(ctx, "sheets", "+info", "--spreadsheet-token", token, "--format", "json")
}

func valuesToCSV(values [][]any) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	for _, row := range values {
		record := make([]string, len(row))
		for i, v := range row {
			record[i] = fmt.Sprintf("%v", v)
		}
		if err := w.Write(record); err != nil {
			return nil, err
		}
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}

func csvToValues(data []byte) ([][]string, error) {
	r := csv.NewReader(bytes.NewReader(data))
	return r.ReadAll()
}
