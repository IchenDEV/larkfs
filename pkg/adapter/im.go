package adapter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/IchenDEV/larkfs/pkg/cache"
	clipkg "github.com/IchenDEV/larkfs/pkg/cli"
	"github.com/IchenDEV/larkfs/pkg/doctype"
	"github.com/IchenDEV/larkfs/pkg/naming"
)

type IMAdapter struct {
	exec  *clipkg.Executor
	meta  *cache.MetadataCache
	namer *naming.Resolver
}

func NewIMAdapter(exec *clipkg.Executor, meta *cache.MetadataCache, namer *naming.Resolver) *IMAdapter {
	return &IMAdapter{exec: exec, meta: meta, namer: namer}
}

type Chat struct {
	ChatID string `json:"chat_id"`
	Name   string `json:"name"`
}

func (a *IMAdapter) ListChats(ctx context.Context) ([]doctype.Entry, error) {
	if cached, ok := a.meta.Get("im:chats"); ok {
		return cached.([]doctype.Entry), nil
	}

	out, err := a.exec.Run(ctx, "im", "chat", "list", "--format", "json", "--page-all")
	if err != nil {
		return nil, err
	}

	var result struct {
		Items []Chat `json:"items"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, err
	}

	var entries []doctype.Entry
	var nameEntries []naming.NameEntry
	for _, c := range result.Items {
		name := naming.SanitizeName(c.Name)
		if name == "untitled" {
			name = c.ChatID
		}
		entries = append(entries, doctype.Entry{
			Name:  name,
			Token: c.ChatID,
			Type:  doctype.TypeFolder,
			IsDir: true,
		})
		nameEntries = append(nameEntries, naming.NameEntry{Name: name, Token: c.ChatID})
	}

	resolved := a.namer.ResolveNames(nameEntries)
	for i := range entries {
		if fname, ok := resolved[entries[i].Token]; ok {
			entries[i].Name = fname
		}
	}

	a.meta.Set("im:chats", entries)
	return entries, nil
}

func (a *IMAdapter) ListChatContents(_ context.Context, chatID string) ([]doctype.Entry, error) {
	return []doctype.Entry{
		{Name: "latest.md", Token: chatID + "|latest", Type: doctype.TypeFile},
		{Name: "_send.md", Token: chatID + "|send", Type: doctype.TypeFile},
		{Name: "files", Token: chatID + "|files", Type: doctype.TypeFolder, IsDir: true},
	}, nil
}

func (a *IMAdapter) ReadMessages(ctx context.Context, chatID string) ([]byte, error) {
	params := clipkg.JSONParam(map[string]any{
		"container_id_type": "chat",
		"container_id":      chatID,
	})
	out, err := a.exec.Run(ctx, "im", "message", "list", "--params", params, "--format", "json")
	if err != nil {
		return nil, err
	}

	var result struct {
		Items []struct {
			MsgType string          `json:"msg_type"`
			Body    json.RawMessage `json:"body"`
			Sender  struct {
				ID string `json:"id"`
			} `json:"sender"`
		} `json:"items"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, err
	}

	var md string
	for _, msg := range result.Items {
		md += fmt.Sprintf("**%s**: %s\n\n", msg.Sender.ID, string(msg.Body))
	}
	return []byte(md), nil
}

func (a *IMAdapter) SendMessage(ctx context.Context, chatID string, content []byte) error {
	_, err := a.exec.Run(ctx, "im", "+messages-send", "--chat-id", chatID, "--text", string(content))
	return err
}
