package adapter

import (
	"context"

	"github.com/IchenDEV/larkfs/pkg/cache"
	clipkg "github.com/IchenDEV/larkfs/pkg/cli"
	"github.com/IchenDEV/larkfs/pkg/doctype"
	"github.com/IchenDEV/larkfs/pkg/naming"
)

const rootFolderToken = ""

type DriveAdapter struct {
	exec     *clipkg.Executor
	registry *doctype.Registry
	meta     *cache.MetadataCache
	namer    *naming.Resolver
}

func NewDriveAdapter(exec *clipkg.Executor, registry *doctype.Registry, meta *cache.MetadataCache, namer *naming.Resolver) *DriveAdapter {
	return &DriveAdapter{exec: exec, registry: registry, meta: meta, namer: namer}
}

func (a *DriveAdapter) ListRoot(ctx context.Context) ([]doctype.Entry, error) {
	return a.ListFolder(ctx, rootFolderToken)
}

func (a *DriveAdapter) ListFolder(ctx context.Context, token string) ([]doctype.Entry, error) {
	cacheKey := "drive:list:" + token
	if cached, ok := a.meta.Get(cacheKey); ok {
		return cached.([]doctype.Entry), nil
	}

	handler := a.registry.Handler(doctype.TypeFolder)
	entries, err := handler.List(ctx, token)
	if err != nil {
		return nil, err
	}

	nameEntries := make([]naming.NameEntry, len(entries))
	for i, e := range entries {
		nameEntries[i] = naming.NameEntry{
			Name:  naming.SanitizeName(e.Name) + doctype.FileExtension(e.Type),
			Token: e.Token,
		}
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

func (a *DriveAdapter) Read(ctx context.Context, token string, dt doctype.DocType) ([]byte, error) {
	return a.registry.Handler(dt).Read(ctx, token)
}

func (a *DriveAdapter) Write(ctx context.Context, token string, dt doctype.DocType, data []byte) error {
	err := a.registry.Handler(dt).Write(ctx, token, data)
	if err == nil {
		a.meta.InvalidatePrefix("drive:")
	}
	return err
}

func (a *DriveAdapter) Create(ctx context.Context, parentToken, name string, dt doctype.DocType, data []byte) (string, error) {
	token, err := a.registry.Handler(dt).Create(ctx, parentToken, name, data)
	if err == nil {
		a.meta.InvalidatePrefix("drive:")
	}
	return token, err
}

func (a *DriveAdapter) Delete(ctx context.Context, token string, dt doctype.DocType) error {
	err := a.registry.Handler(dt).Delete(ctx, token)
	if err == nil {
		a.meta.InvalidatePrefix("drive:")
	}
	return err
}
