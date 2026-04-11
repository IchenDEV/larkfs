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
	exec  clipkg.Runner
	meta  *cache.MetadataCache
	namer *naming.Resolver
}

func NewIMAdapter(exec clipkg.Runner, meta *cache.MetadataCache, namer *naming.Resolver) *IMAdapter {
	return &IMAdapter{exec: exec, meta: meta, namer: namer}
}

type Chat struct {
	ChatID string `json:"chat_id"`
	Name   string `json:"name"`
}

func (a *IMAdapter) ListChats(ctx context.Context) (doctype.ListResult, error) {
	if cached, ok := a.meta.Get("im:chats"); ok {
		return cached.(doctype.ListResult), nil
	}

	out, err := a.exec.Run(ctx, "im", "chats", "list", "--format", "json", "--page-all", "--page-limit", "0")
	if err != nil {
		return doctype.ListResult{}, err
	}

	var result struct {
		Data struct {
			Items      []Chat `json:"items"`
			HasMore    bool   `json:"has_more"`
			NextCursor string `json:"page_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return doctype.ListResult{}, err
	}

	var entries []doctype.Entry
	var nameEntries []naming.NameEntry
	for _, c := range result.Data.Items {
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

	list := doctype.ListResult{
		Entries: entries,
		Page: doctype.PageInfo{
			HasMore:    result.Data.HasMore,
			NextCursor: result.Data.NextCursor,
			WindowSize: len(entries),
			SortKey:    "name",
			Truncated:  result.Data.HasMore,
		},
	}
	a.meta.Set("im:chats", list)
	return list, nil
}

func (a *IMAdapter) ListChatContents(_ context.Context, chatID string) (doctype.ListResult, error) {
	return doctype.ListResult{
		Entries: []doctype.Entry{
			{Name: "latest.md", Token: chatID + "|latest", Type: doctype.TypeFile},
			{Name: "_send.md", Token: chatID + "|send", Type: doctype.TypeFile},
			{Name: "files", Token: chatID + "|files", Type: doctype.TypeFolder, IsDir: true},
		},
		Page: doctype.PageInfo{WindowSize: 3, SortKey: "fixed"},
	}, nil
}

func (a *IMAdapter) ListChatFiles(ctx context.Context, chatID string) (doctype.ListResult, error) {
	cacheKey := "im:files:" + chatID
	if cached, ok := a.meta.Get(cacheKey); ok {
		return cached.(doctype.ListResult), nil
	}

	out, err := a.exec.Run(ctx, "im", "+chat-messages-list", "--chat-id", chatID, "--format", "json")
	if err != nil {
		return doctype.ListResult{}, err
	}

	var result struct {
		Data struct {
			Messages []struct {
				MsgType   string `json:"msg_type"`
				MessageID string `json:"message_id"`
				Content   string `json:"content"`
			} `json:"messages"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return doctype.ListResult{}, err
	}

	var entries []doctype.Entry
	for _, msg := range result.Data.Messages {
		if msg.MsgType != "file" && msg.MsgType != "image" && msg.MsgType != "media" {
			continue
		}
		name := naming.SanitizeName(msg.MessageID)
		switch msg.MsgType {
		case "image":
			name += ".png"
		case "file", "media":
			name += ".bin"
		}
		entries = append(entries, doctype.Entry{
			Name:  name,
			Token: chatID + "|file|" + msg.MessageID,
			Type:  doctype.TypeFile,
		})
	}

	list := doctype.ListResult{
		Entries: entries,
		Page: doctype.PageInfo{
			WindowSize: len(entries),
			SortKey:    "message_id",
		},
	}
	a.meta.Set(cacheKey, list)
	return list, nil
}

func (a *IMAdapter) ReadMessages(ctx context.Context, chatID string) ([]byte, error) {
	out, err := a.exec.Run(ctx, "im", "+chat-messages-list", "--chat-id", chatID, "--format", "json")
	if err != nil {
		return nil, err
	}

	var result struct {
		Data struct {
			Messages []struct {
				Content    string `json:"content"`
				CreateTime string `json:"create_time"`
				MsgType    string `json:"msg_type"`
				Sender     struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"sender"`
			} `json:"messages"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, err
	}

	var md string
	for _, msg := range result.Data.Messages {
		sender := msg.Sender.Name
		if sender == "" {
			sender = msg.Sender.ID
		}
		md += fmt.Sprintf("**%s** (%s):\n%s\n\n", sender, msg.CreateTime, msg.Content)
	}
	return []byte(md), nil
}

func (a *IMAdapter) SendMessage(ctx context.Context, chatID string, content []byte) error {
	_, err := a.exec.Run(ctx, "im", "+messages-send", "--chat-id", chatID, "--text", string(content))
	return err
}
