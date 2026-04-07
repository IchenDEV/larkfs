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

type CalendarAdapter struct {
	exec  *clipkg.Executor
	meta  *cache.MetadataCache
	namer *naming.Resolver
}

func NewCalendarAdapter(exec *clipkg.Executor, meta *cache.MetadataCache, namer *naming.Resolver) *CalendarAdapter {
	return &CalendarAdapter{exec: exec, meta: meta, namer: namer}
}

type CalendarEvent struct {
	EventID  string `json:"event_id"`
	Summary  string `json:"summary"`
	Start    string `json:"start_time"`
	End      string `json:"end_time"`
	Location string `json:"location,omitempty"`
}

func (a *CalendarAdapter) ListEvents(ctx context.Context) ([]doctype.Entry, error) {
	if cached, ok := a.meta.Get("calendar:events"); ok {
		return cached.([]doctype.Entry), nil
	}

	out, err := a.exec.Run(ctx, "calendar", "+agenda", "--format", "json")
	if err != nil {
		return nil, err
	}

	var result struct {
		Items []CalendarEvent `json:"items"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, err
	}

	var entries []doctype.Entry
	var nameEntries []naming.NameEntry
	for _, e := range result.Items {
		name := naming.SanitizeName(e.Summary) + ".md"
		entry := doctype.Entry{
			Name:  name,
			Token: e.EventID,
			Type:  doctype.TypeFile,
		}
		entries = append(entries, entry)
		nameEntries = append(nameEntries, naming.NameEntry{Name: name, Token: e.EventID})
	}

	resolved := a.namer.ResolveNames(nameEntries)
	for i := range entries {
		if fname, ok := resolved[entries[i].Token]; ok {
			entries[i].Name = fname
		}
	}

	entries = append(entries, doctype.Entry{Name: "_create.md", Token: "_create", Type: doctype.TypeFile})
	a.meta.Set("calendar:events", entries)
	return entries, nil
}

func (a *CalendarAdapter) ReadEvent(ctx context.Context, eventID string) ([]byte, error) {
	out, err := a.exec.Run(ctx, "calendar", "+agenda", "--format", "json")
	if err != nil {
		return nil, err
	}

	var result struct {
		Items []CalendarEvent `json:"items"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, err
	}

	for _, e := range result.Items {
		if e.EventID == eventID {
			return formatEventMarkdown(e), nil
		}
	}
	return nil, fmt.Errorf("event not found: %s", eventID)
}

func (a *CalendarAdapter) CreateEvent(ctx context.Context, data []byte) error {
	_, err := a.exec.Run(ctx, "calendar", "+create", "--summary", string(data))
	if err == nil {
		a.meta.InvalidatePrefix("calendar:")
	}
	return err
}

func formatEventMarkdown(e CalendarEvent) []byte {
	md := fmt.Sprintf("---\nevent_id: %q\nsummary: %q\nstart: %q\nend: %q\nlocation: %q\n---\n\n# %s\n\n- Start: %s\n- End: %s\n- Location: %s\n",
		e.EventID, e.Summary, e.Start, e.End, e.Location,
		e.Summary, e.Start, e.End, e.Location)
	return []byte(md)
}
