package doctype

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/IchenDEV/larkfs/pkg/cli"
)

const fullSheetCSVRange = "A1:ZZ100000"

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
		"sheets", "+workbook-info", "--spreadsheet-token", token, "--format", "json")
	if err != nil {
		return ListResult{}, err
	}

	sheets, err := parseWorkbookSheets(out)
	if err != nil {
		return ListResult{}, err
	}

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
		"sheets", "+csv-get",
		"--spreadsheet-token", spreadsheetToken,
		"--sheet-id", sheetID,
		"--range", fullSheetCSVRange,
		"--format", "json")
	if err != nil {
		return nil, err
	}
	return csvFromOutput(out)
}

func (h *SheetHandler) Write(ctx context.Context, token string, data []byte) error {
	parts := strings.SplitN(token, "|", 2)
	if len(parts) != 2 {
		return ErrReadOnly
	}

	_, err := h.exec.Run(ctx,
		"sheets", "+csv-put",
		"--spreadsheet-token", parts[0],
		"--sheet-id", parts[1],
		"--start-cell", "A1",
		"--csv", string(data))
	return err
}

func (h *SheetHandler) Create(ctx context.Context, _ string, name string, _ []byte) (string, error) {
	out, err := h.exec.Run(ctx, "sheets", "+workbook-create", "--title", name)
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
	return deleteDriveResource(ctx, h.exec, token, TypeSheet)
}

func (h *SheetHandler) readMeta(ctx context.Context, token string) ([]byte, error) {
	return h.exec.Run(ctx, "sheets", "+workbook-info", "--spreadsheet-token", token, "--format", "json")
}

type workbookSheet struct {
	SheetID string `json:"sheet_id"`
	Title   string `json:"title"`
}

func parseWorkbookSheets(out []byte) ([]workbookSheet, error) {
	if sheets, ok, err := parseCurrentWorkbookSheets(out); ok || err != nil {
		return sheets, err
	}
	if sheets, ok, err := parseLegacyWorkbookSheets(out); ok || err != nil {
		return sheets, err
	}
	if sheets, ok, err := parseNestedWorkbookSheets(out); ok || err != nil {
		return sheets, err
	}
	return nil, nil
}

func parseCurrentWorkbookSheets(out []byte) ([]workbookSheet, bool, error) {
	var result struct {
		Data struct {
			Sheets []workbookSheet `json:"sheets"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, false, nil
	}
	return result.Data.Sheets, result.Data.Sheets != nil, nil
}

func parseLegacyWorkbookSheets(out []byte) ([]workbookSheet, bool, error) {
	var result struct {
		Data struct {
			Sheets struct {
				Sheets []workbookSheet `json:"sheets"`
			} `json:"sheets"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, false, nil
	}
	sheets := result.Data.Sheets.Sheets
	return sheets, sheets != nil, nil
}

func parseNestedWorkbookSheets(out []byte) ([]workbookSheet, bool, error) {
	var result struct {
		Data struct {
			Workbook struct {
				Sheets []workbookSheet `json:"sheets"`
			} `json:"workbook"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, false, nil
	}
	sheets := result.Data.Workbook.Sheets
	return sheets, sheets != nil, nil
}

func csvFromOutput(out []byte) ([]byte, error) {
	var result struct {
		Data struct {
			CSV     string `json:"csv"`
			Content string `json:"content"`
			Result  string `json:"result"`
			Text    string `json:"text"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return out, nil
	}
	switch {
	case result.Data.CSV != "":
		return []byte(result.Data.CSV), nil
	case result.Data.Content != "":
		return []byte(result.Data.Content), nil
	case result.Data.Result != "":
		return []byte(result.Data.Result), nil
	case result.Data.Text != "":
		return []byte(result.Data.Text), nil
	default:
		return nil, nil
	}
}
