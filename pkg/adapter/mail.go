package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/IchenDEV/larkfs/pkg/cache"
	clipkg "github.com/IchenDEV/larkfs/pkg/cli"
	"github.com/IchenDEV/larkfs/pkg/doctype"
	"github.com/IchenDEV/larkfs/pkg/naming"
)

type MailAdapter struct {
	exec  *clipkg.Executor
	meta  *cache.MetadataCache
	namer *naming.Resolver
}

func NewMailAdapter(exec *clipkg.Executor, meta *cache.MetadataCache, namer *naming.Resolver) *MailAdapter {
	return &MailAdapter{exec: exec, meta: meta, namer: namer}
}

func (a *MailAdapter) ListFolders(ctx context.Context) ([]doctype.Entry, error) {
	if cached, ok := a.meta.Get("mail:folders"); ok {
		return cached.([]doctype.Entry), nil
	}

	params := clipkg.JSONParam(map[string]any{"user_mailbox_id": "me"})
	out, err := a.exec.Run(ctx,
		"mail", "user_mailbox.folders", "list",
		"--params", params, "--format", "json")
	if err != nil {
		if strings.Contains(err.Error(), "1230002") || strings.Contains(err.Error(), "1230003") {
			return nil, nil
		}
		return nil, err
	}

	var result struct {
		Data struct {
			Items []struct {
				FolderID string `json:"folder_id"`
				Name     string `json:"name"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, err
	}

	entries := make([]doctype.Entry, 0, len(result.Data.Items)+2)
	for _, f := range result.Data.Items {
		entries = append(entries, doctype.Entry{
			Name:  naming.SanitizeName(f.Name),
			Token: f.FolderID,
			Type:  doctype.TypeFolder,
			IsDir: true,
		})
	}
	entries = append(entries,
		doctype.Entry{Name: "_compose.md", Token: "compose", Type: doctype.TypeFile},
		doctype.Entry{Name: "_send.md", Token: "send", Type: doctype.TypeFile},
	)

	a.meta.Set("mail:folders", entries)
	return entries, nil
}

func (a *MailAdapter) ListMessages(ctx context.Context, folder string) ([]doctype.Entry, error) {
	cacheKey := "mail:messages:" + folder
	if cached, ok := a.meta.Get(cacheKey); ok {
		return cached.([]doctype.Entry), nil
	}

	out, err := a.exec.Run(ctx, "mail", "+triage", "--folder", folder, "--format", "json")
	if err != nil {
		return nil, err
	}

	var result struct {
		Data []struct {
			MessageID string `json:"message_id"`
			From      string `json:"from"`
			Subject   string `json:"subject"`
			Date      string `json:"date"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, err
	}

	var entries []doctype.Entry
	var nameEntries []naming.NameEntry
	for _, m := range result.Data {
		date := m.Date
		if idx := strings.IndexByte(date, 'T'); idx >= 0 {
			date = date[:idx]
		}
		name := naming.SanitizeName(date+"_"+m.From+"_"+m.Subject) + ".md"
		entries = append(entries, doctype.Entry{
			Name:  name,
			Token: m.MessageID,
			Type:  doctype.TypeFile,
		})
		nameEntries = append(nameEntries, naming.NameEntry{Name: name, Token: m.MessageID})
	}

	resolved := a.namer.ResolveNames(nameEntries)
	for i := range entries {
		if fname, ok := resolved[entries[i].Token]; ok {
			entries[i].Name = fname
		}
	}

	a.meta.Set(cacheKey, entries)
	return entries, nil
}

func (a *MailAdapter) ReadMessage(ctx context.Context, messageID string) ([]byte, error) {
	out, err := a.exec.Run(ctx, "mail", "+message", "--message-id", messageID, "--format", "json")
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data struct {
			MessageID string   `json:"message_id"`
			ThreadID  string   `json:"thread_id"`
			From      string   `json:"from"`
			To        []string `json:"to"`
			CC        []string `json:"cc"`
			Date      string   `json:"date"`
			Subject   string   `json:"subject"`
			Body      string   `json:"body"`
			Labels    []string `json:"labels"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, err
	}

	md := fmt.Sprintf("---\nmessage_id: %q\nthread_id: %q\nfrom: %q\nto: %v\ncc: %v\ndate: %q\nsubject: %q\nlabels: %v\n---\n\n%s\n",
		resp.Data.MessageID, resp.Data.ThreadID, resp.Data.From, resp.Data.To, resp.Data.CC, resp.Data.Date, resp.Data.Subject, resp.Data.Labels, resp.Data.Body)
	return []byte(md), nil
}

func (a *MailAdapter) Send(ctx context.Context, to, subject, body string) error {
	_, err := a.exec.Run(ctx,
		"mail", "+send", "--to", to, "--subject", subject, "--body", body, "--confirm-send")
	return err
}

func (a *MailAdapter) Reply(ctx context.Context, messageID, body string) error {
	_, err := a.exec.Run(ctx,
		"mail", "+reply", "--message-id", messageID, "--body", body, "--confirm-send")
	return err
}

func (a *MailAdapter) Trash(ctx context.Context, messageID string) error {
	params := clipkg.JSONParam(map[string]any{"user_mailbox_id": "me", "message_id": messageID})
	_, err := a.exec.Run(ctx, "mail", "user_mailbox.messages", "trash", "--params", params)
	if err == nil {
		a.meta.InvalidatePrefix("mail:")
	}
	return err
}
