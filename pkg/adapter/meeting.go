package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/IchenDEV/larkfs/pkg/cache"
	clipkg "github.com/IchenDEV/larkfs/pkg/cli"
	"github.com/IchenDEV/larkfs/pkg/doctype"
	"github.com/IchenDEV/larkfs/pkg/naming"
)

type MeetingAdapter struct {
	exec     *clipkg.Executor
	meta     *cache.MetadataCache
	namer    *naming.Resolver
	cacheDir string
}

func NewMeetingAdapter(exec *clipkg.Executor, meta *cache.MetadataCache, namer *naming.Resolver, cacheDir string) *MeetingAdapter {
	return &MeetingAdapter{exec: exec, meta: meta, namer: namer, cacheDir: cacheDir}
}

type Meeting struct {
	MeetingID string `json:"meeting_id"`
	Topic     string `json:"topic"`
	StartTime string `json:"start_time"`
}

func (a *MeetingAdapter) ListDateDirs() []doctype.Entry {
	today := time.Now()
	entries := make([]doctype.Entry, 0, 30)
	for i := 0; i < 30; i++ {
		d := today.AddDate(0, 0, -i)
		name := d.Format("2006-01-02")
		entries = append(entries, doctype.Entry{
			Name:  name,
			Token: name,
			Type:  doctype.TypeFolder,
			IsDir: true,
		})
	}
	return entries
}

func (a *MeetingAdapter) ListMeetings(ctx context.Context, date string) ([]doctype.Entry, error) {
	cacheKey := "meetings:" + date
	if cached, ok := a.meta.Get(cacheKey); ok {
		return cached.([]doctype.Entry), nil
	}

	out, err := a.exec.Run(ctx, "vc", "+search", "--start", date, "--end", date, "--format", "json")
	if err != nil {
		return nil, err
	}

	var result struct {
		Items []Meeting `json:"items"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, err
	}

	var entries []doctype.Entry
	var nameEntries []naming.NameEntry
	for _, m := range result.Items {
		name := naming.SanitizeName(m.Topic)
		entries = append(entries, doctype.Entry{
			Name:  name,
			Token: m.MeetingID,
			Type:  doctype.TypeFolder,
			IsDir: true,
		})
		nameEntries = append(nameEntries, naming.NameEntry{Name: name, Token: m.MeetingID})
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

func (a *MeetingAdapter) ListMeetingContents(meetingID string) []doctype.Entry {
	return []doctype.Entry{
		{Name: "_meta.json", Token: meetingID + "|meta", Type: doctype.TypeFile},
		{Name: "summary.md", Token: meetingID + "|summary", Type: doctype.TypeFile},
		{Name: "todos.md", Token: meetingID + "|todos", Type: doctype.TypeFile},
		{Name: "transcript.md", Token: meetingID + "|transcript", Type: doctype.TypeFile},
		{Name: "recording.mp4", Token: meetingID + "|recording", Type: doctype.TypeFile},
	}
}

func (a *MeetingAdapter) ReadMeta(ctx context.Context, meetingID string) ([]byte, error) {
	params := clipkg.JSONParam(map[string]any{"meeting_id": meetingID, "with_participants": true})
	return a.exec.Run(ctx, "vc", "meeting", "get", "--params", params, "--format", "json")
}

func (a *MeetingAdapter) ReadSummary(ctx context.Context, meetingID string) ([]byte, error) {
	noteToken, err := a.getNoteToken(ctx, meetingID)
	if err != nil {
		return nil, err
	}

	out, err := a.exec.Run(ctx, "docs", "+fetch", "--doc", noteToken, "--format", "json")
	if err != nil {
		return nil, err
	}

	var result struct {
		Markdown string `json:"markdown"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, err
	}
	return []byte(result.Markdown), nil
}

func (a *MeetingAdapter) ReadTranscript(ctx context.Context, meetingID string) ([]byte, error) {
	verbatimToken, err := a.getVerbatimToken(ctx, meetingID)
	if err != nil {
		return nil, err
	}

	out, err := a.exec.Run(ctx, "docs", "+fetch", "--doc", verbatimToken, "--format", "json")
	if err != nil {
		return nil, err
	}

	var result struct {
		Markdown string `json:"markdown"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, err
	}
	return []byte(result.Markdown), nil
}

func (a *MeetingAdapter) getNoteToken(ctx context.Context, meetingID string) (string, error) {
	out, err := a.exec.Run(ctx, "vc", "+notes", "--meeting-ids", meetingID, "--format", "json")
	if err != nil {
		return "", err
	}

	var result struct {
		Items []struct {
			NoteDocToken string `json:"note_doc_token"`
		} `json:"items"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return "", err
	}
	if len(result.Items) == 0 {
		return "", fmt.Errorf("no notes found for meeting %s", meetingID)
	}
	return result.Items[0].NoteDocToken, nil
}

func (a *MeetingAdapter) getVerbatimToken(ctx context.Context, meetingID string) (string, error) {
	out, err := a.exec.Run(ctx, "vc", "+notes", "--meeting-ids", meetingID, "--format", "json")
	if err != nil {
		return "", err
	}

	var result struct {
		Items []struct {
			VerbatimDocToken string `json:"verbatim_doc_token"`
		} `json:"items"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return "", err
	}
	if len(result.Items) == 0 {
		return "", fmt.Errorf("no transcript found for meeting %s", meetingID)
	}
	return result.Items[0].VerbatimDocToken, nil
}
