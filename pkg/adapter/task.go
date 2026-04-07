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

type TaskAdapter struct {
	exec  *clipkg.Executor
	meta  *cache.MetadataCache
	namer *naming.Resolver
}

func NewTaskAdapter(exec *clipkg.Executor, meta *cache.MetadataCache, namer *naming.Resolver) *TaskAdapter {
	return &TaskAdapter{exec: exec, meta: meta, namer: namer}
}

type Task struct {
	TaskID  string `json:"task_id"`
	Summary string `json:"summary"`
	Due     string `json:"due,omitempty"`
	Status  string `json:"status,omitempty"`
}

func (a *TaskAdapter) ListTasks(ctx context.Context) ([]doctype.Entry, error) {
	if cached, ok := a.meta.Get("tasks:list"); ok {
		return cached.([]doctype.Entry), nil
	}

	out, err := a.exec.Run(ctx, "api", "GET", "/open-apis/task/v2/tasks", "--format", "json")
	if err != nil {
		return nil, err
	}

	var result struct {
		Data struct {
			Items []Task `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, err
	}

	var entries []doctype.Entry
	var nameEntries []naming.NameEntry
	for _, t := range result.Data.Items {
		name := naming.SanitizeName(t.Summary) + ".md"
		entries = append(entries, doctype.Entry{
			Name:  name,
			Token: t.TaskID,
			Type:  doctype.TypeFile,
		})
		nameEntries = append(nameEntries, naming.NameEntry{Name: name, Token: t.TaskID})
	}

	resolved := a.namer.ResolveNames(nameEntries)
	for i := range entries {
		if fname, ok := resolved[entries[i].Token]; ok {
			entries[i].Name = fname
		}
	}

	entries = append(entries, doctype.Entry{Name: "_create.md", Token: "_create", Type: doctype.TypeFile})
	a.meta.Set("tasks:list", entries)
	return entries, nil
}

func (a *TaskAdapter) ReadTask(ctx context.Context, taskID string) ([]byte, error) {
	out, err := a.exec.Run(ctx, "api", "GET", "/open-apis/task/v2/tasks/"+taskID, "--format", "json")
	if err != nil {
		return nil, err
	}

	var result struct {
		Data struct {
			Task Task `json:"task"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, err
	}
	return formatTaskMarkdown(result.Data.Task), nil
}

func (a *TaskAdapter) CreateTask(ctx context.Context, data []byte) error {
	_, err := a.exec.Run(ctx, "api", "POST", "/open-apis/task/v2/tasks", "--data", string(data))
	if err == nil {
		a.meta.InvalidatePrefix("tasks:")
	}
	return err
}

func formatTaskMarkdown(t Task) []byte {
	md := fmt.Sprintf("---\ntask_id: %q\nsummary: %q\nstatus: %q\ndue: %q\n---\n\n# %s\n\n- Status: %s\n- Due: %s\n",
		t.TaskID, t.Summary, t.Status, t.Due,
		t.Summary, t.Status, t.Due)
	return []byte(md)
}
