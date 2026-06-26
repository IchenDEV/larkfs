package vfs

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/IchenDEV/larkfs/pkg/doctype"
)

var datePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func (o *Operations) fetchEntries(ctx context.Context, node *VNode) (doctype.ListResult, error) {
	domain := node.Domain
	if domain == "" {
		domain = o.domainFromPath(node)
	}

	switch domain {
	case "drive":
		if node.Token == "" {
			return o.drive.ListRoot(ctx)
		}
		if node.DocType != "" && node.DocType != doctype.TypeFolder {
			return o.drive.ListByType(ctx, node.Token, node.DocType)
		}
		return o.drive.ListFolder(ctx, node.Token)
	case "wiki":
		if node.Token == "" {
			return o.wiki.ListSpaces(ctx)
		}
		return o.wiki.ListNodes(ctx, node.Token)
	case "im":
		if node.Token == "" {
			return o.im.ListChats(ctx)
		}
		if strings.HasSuffix(node.Token, "|files") {
			chatID := strings.TrimSuffix(node.Token, "|files")
			return o.im.ListChatFiles(ctx, chatID)
		}
		return o.im.ListChatContents(ctx, node.Token)
	case "calendar":
		return o.calendar.ListEvents(ctx)
	case "tasks":
		return o.task.ListTasks(ctx)
	case "mail":
		if node.Token == "" {
			return o.mail.ListFolders(ctx)
		}
		return o.mail.ListMessages(ctx, node.Name)
	case "meetings":
		if node.Token == "" {
			return o.meeting.ListDateDirs(), nil
		}
		if datePattern.MatchString(node.Token) {
			return o.meeting.ListMeetings(ctx, node.Token)
		}
		return o.meeting.ListMeetingContents(node.Token), nil
	case "apps", "approval", "attendance", "base", "contact", "docs", "event", "markdown", "minutes", "note", "okr", "sheets", "slides", "vc", "whiteboard", "_system":
		return staticDomainEntries(domain, node.Token), nil
	}

	return doctype.ListResult{}, fmt.Errorf("unknown domain: %s", domain)
}

func (o *Operations) readContent(ctx context.Context, node *VNode) ([]byte, error) {
	if node.PendingCreate {
		return []byte{}, nil
	}

	switch node.Domain {
	case "drive":
		return o.drive.Read(ctx, node.Token, node.DocType)
	case "wiki":
		return o.wiki.Read(ctx, node.Token)
	case "im":
		if strings.HasSuffix(node.Token, "|latest") {
			chatID := strings.TrimSuffix(node.Token, "|latest")
			return o.im.ReadMessages(ctx, chatID)
		}
		return []byte{}, nil
	case "calendar":
		if node.Token == "_create" {
			return []byte("# New Event\n\nWrite event details here.\n"), nil
		}
		return o.calendar.ReadEvent(ctx, node.Token)
	case "tasks":
		if node.Token == "_create" {
			return []byte("# New Task\n\nWrite task summary here.\n"), nil
		}
		return o.task.ReadTask(ctx, node.Token)
	case "mail":
		return o.mail.ReadMessage(ctx, node.Token)
	case "meetings":
		return o.readMeetingContent(ctx, node)
	}

	return nil, fmt.Errorf("unsupported read: %s", node.Path())
}

func (o *Operations) writeContent(ctx context.Context, node *VNode, data []byte) error {
	switch node.Domain {
	case "drive":
		if node.PendingCreate {
			parent := node.Parent()
			if parent == nil {
				return fmt.Errorf("%w: cannot create drive resource at root", ErrUnsupported)
			}
			token, err := o.drive.Create(ctx, parent.Token, driveRemoteName(node.Name, node.DocType), node.DocType, data)
			if err != nil {
				return err
			}
			node.Token = token
			node.PendingCreate = false
			node.SetModTime(time.Now())
			return nil
		}
		if node.DocType == doctype.TypeFile && !strings.Contains(node.Token, "|") {
			return fmt.Errorf("%w: updating existing drive file content is not mapped", ErrUnsupported)
		}
		err := o.drive.Write(ctx, node.Token, node.DocType, data)
		if errors.Is(err, doctype.ErrReadOnly) {
			return fmt.Errorf("%w: write blocked for %s", ErrUnsupported, node.DocType)
		}
		return err
	case "wiki":
		return o.wiki.Write(ctx, node.Token, data)
	case "im":
		if strings.HasSuffix(node.Token, "|send") {
			chatID := strings.TrimSuffix(node.Token, "|send")
			return o.im.SendMessage(ctx, chatID, data)
		}
		return fmt.Errorf("%w: im resource is not writable", ErrReadOnly)
	case "calendar":
		if node.Token == "_create" {
			return o.calendar.CreateEvent(ctx, data)
		}
		return fmt.Errorf("%w: calendar event is not writable", ErrReadOnly)
	case "tasks":
		if node.Token == "_create" {
			return o.task.CreateTask(ctx, data)
		}
		return fmt.Errorf("%w: task is not writable", ErrReadOnly)
	}

	return fmt.Errorf("unsupported write: %s", node.Path())
}

func staticDomainEntries(domain, token string) doctype.ListResult {
	if token != "" {
		return doctype.ListResult{Page: doctype.PageInfo{SortKey: "fixed"}}
	}

	names := map[string][]string{
		"apps":       {"apps", "html", "local-dev", "database", "releases", "access-scope", "sessions"},
		"approval":   {"instances", "tasks"},
		"attendance": {"user-tasks"},
		"base":       {"bases", "blocks", "tables", "records", "fields", "views", "dashboards", "dashboard-blocks", "forms", "form-questions", "roles", "workflows", "advanced-permissions"},
		"contact":    {"users", "search"},
		"docs":       {"search", "by-token", "media", "resources", "whiteboard"},
		"event":      {"list", "schema", "status", "consume"},
		"markdown":   {"create", "fetch", "diff", "overwrite", "patch"},
		"minutes":    {"minutes", "media", "search", "speakers"},
		"note":       {"detail", "transcript"},
		"okr":        {"cycles", "objectives", "key-results", "progress", "indicators", "ordering", "weights", "images"},
		"sheets":     {"workbooks", "cells", "sheets", "dimensions", "filters", "filter-views", "conditional-formats", "dropdowns", "charts", "images", "pivots", "sparklines"},
		"slides":     {"presentations", "slides", "media", "screenshots"},
		"vc":         {"meetings", "active-meetings", "events", "notes", "recordings"},
		"whiteboard": {"query", "update"},
		"_system":    {"api", "schema", "auth", "config", "profile", "doctor", "event", "skills", "update"},
	}
	entries := make([]doctype.Entry, 0, len(names[domain]))
	for _, name := range names[domain] {
		entries = append(entries, doctype.Entry{
			Name:  name,
			Token: domain + ":" + name,
			Type:  doctype.TypeFolder,
			IsDir: true,
		})
	}
	return doctype.ListResult{
		Entries: entries,
		Page: doctype.PageInfo{
			WindowSize: len(entries),
			SortKey:    "fixed",
		},
	}
}

func (o *Operations) readMeetingContent(ctx context.Context, node *VNode) ([]byte, error) {
	parts := strings.SplitN(node.Token, "|", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid meeting token: %s", node.Token)
	}
	meetingID, part := parts[0], parts[1]

	switch part {
	case "meta":
		return o.meeting.ReadMeta(ctx, meetingID)
	case "summary":
		return o.meeting.ReadSummary(ctx, meetingID)
	case "transcript":
		return o.meeting.ReadTranscript(ctx, meetingID)
	case "todos":
		summary, err := o.meeting.ReadSummary(ctx, meetingID)
		if err != nil {
			return nil, err
		}
		return extractTodos(summary), nil
	case "recording":
		return o.meeting.ReadRecording(ctx, meetingID)
	}

	return nil, fmt.Errorf("unsupported meeting part: %s", part)
}

func extractTodos(markdown []byte) []byte {
	var todos []string
	for _, line := range strings.Split(string(markdown), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- [ ]") || strings.HasPrefix(trimmed, "- [x]") ||
			strings.Contains(strings.ToLower(trimmed), "todo") {
			todos = append(todos, line)
		}
	}
	if len(todos) == 0 {
		return []byte("No todos found.\n")
	}
	return []byte(strings.Join(todos, "\n") + "\n")
}
