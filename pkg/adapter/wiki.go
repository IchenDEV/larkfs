package adapter

import (
	"context"
	"encoding/json"

	"github.com/IchenDEV/larkfs/pkg/cache"
	clipkg "github.com/IchenDEV/larkfs/pkg/cli"
	"github.com/IchenDEV/larkfs/pkg/doctype"
	"github.com/IchenDEV/larkfs/pkg/naming"
)

type WikiAdapter struct {
	exec     *clipkg.Executor
	registry *doctype.Registry
	meta     *cache.MetadataCache
	namer    *naming.Resolver
}

func NewWikiAdapter(exec *clipkg.Executor, registry *doctype.Registry, meta *cache.MetadataCache, namer *naming.Resolver) *WikiAdapter {
	return &WikiAdapter{exec: exec, registry: registry, meta: meta, namer: namer}
}

type WikiSpace struct {
	SpaceID string `json:"space_id"`
	Name    string `json:"name"`
}

func (a *WikiAdapter) ListSpaces(ctx context.Context) ([]doctype.Entry, error) {
	if cached, ok := a.meta.Get("wiki:spaces"); ok {
		return cached.([]doctype.Entry), nil
	}

	out, err := a.exec.Run(ctx, "wiki", "spaces", "list", "--format", "json", "--page-all")
	if err != nil {
		return nil, err
	}

	var result struct {
		Items []WikiSpace `json:"items"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, err
	}

	entries := make([]doctype.Entry, len(result.Items))
	for i, s := range result.Items {
		entries[i] = doctype.Entry{
			Name:  naming.SanitizeName(s.Name),
			Token: s.SpaceID,
			Type:  doctype.TypeFolder,
			IsDir: true,
		}
	}

	a.meta.Set("wiki:spaces", entries)
	return entries, nil
}

func (a *WikiAdapter) ListNodes(ctx context.Context, spaceID string) ([]doctype.Entry, error) {
	cacheKey := "wiki:nodes:" + spaceID
	if cached, ok := a.meta.Get(cacheKey); ok {
		return cached.([]doctype.Entry), nil
	}

	params := clipkg.JSONParam(map[string]any{"space_id": spaceID})
	out, err := a.exec.Run(ctx,
		"wiki", "nodes", "list",
		"--params", params,
		"--format", "json", "--page-all")
	if err != nil {
		return nil, err
	}

	var result struct {
		Items []struct {
			NodeToken string `json:"node_token"`
			Title     string `json:"title"`
			ObjType   string `json:"obj_type"`
			HasChild  bool   `json:"has_child"`
		} `json:"items"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, err
	}

	entries := make([]doctype.Entry, len(result.Items))
	nameEntries := make([]naming.NameEntry, len(result.Items))
	for i, n := range result.Items {
		dt := doctype.DocType(n.ObjType)
		isDir := n.HasChild || doctype.IsDirectory(dt)
		name := naming.SanitizeName(n.Title) + doctype.FileExtension(dt)

		entries[i] = doctype.Entry{
			Name:  name,
			Token: n.NodeToken,
			Type:  dt,
			IsDir: isDir,
		}
		nameEntries[i] = naming.NameEntry{Name: name, Token: n.NodeToken}
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

type nodeInfo struct {
	ObjType  string `json:"obj_type"`
	ObjToken string `json:"obj_token"`
}

func (a *WikiAdapter) ResolveNode(ctx context.Context, nodeToken string) (doctype.DocType, string, error) {
	cacheKey := "wiki:node:" + nodeToken
	if cached, ok := a.meta.Get(cacheKey); ok {
		info := cached.(nodeInfo)
		return doctype.DocType(info.ObjType), info.ObjToken, nil
	}

	params := clipkg.JSONParam(map[string]any{"token": nodeToken})
	out, err := a.exec.Run(ctx,
		"wiki", "spaces", "get_node",
		"--params", params,
		"--format", "json")
	if err != nil {
		return "", "", err
	}

	var result struct {
		Node struct {
			ObjType  string `json:"obj_type"`
			ObjToken string `json:"obj_token"`
		} `json:"node"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return "", "", err
	}

	info := nodeInfo{ObjType: result.Node.ObjType, ObjToken: result.Node.ObjToken}
	a.meta.Set(cacheKey, info)
	return doctype.DocType(info.ObjType), info.ObjToken, nil
}

func (a *WikiAdapter) Read(ctx context.Context, nodeToken string) ([]byte, error) {
	dt, objToken, err := a.ResolveNode(ctx, nodeToken)
	if err != nil {
		return nil, err
	}
	return a.registry.Handler(dt).Read(ctx, objToken)
}

func (a *WikiAdapter) Write(ctx context.Context, nodeToken string, data []byte) error {
	dt, objToken, err := a.ResolveNode(ctx, nodeToken)
	if err != nil {
		return err
	}
	return a.registry.Handler(dt).Write(ctx, objToken, data)
}
