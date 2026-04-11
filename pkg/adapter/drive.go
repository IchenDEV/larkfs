package adapter

import (
	"context"
	"fmt"
	"strings"

	"github.com/IchenDEV/larkfs/pkg/cache"
	clipkg "github.com/IchenDEV/larkfs/pkg/cli"
	"github.com/IchenDEV/larkfs/pkg/doctype"
	"github.com/IchenDEV/larkfs/pkg/naming"
)

const rootFolderToken = ""

type DriveAdapter struct {
	exec     clipkg.Runner
	registry *doctype.Registry
	meta     *cache.MetadataCache
	namer    *naming.Resolver
}

func NewDriveAdapter(exec clipkg.Runner, registry *doctype.Registry, meta *cache.MetadataCache, namer *naming.Resolver) *DriveAdapter {
	return &DriveAdapter{exec: exec, registry: registry, meta: meta, namer: namer}
}

func (a *DriveAdapter) ListRoot(ctx context.Context) (doctype.ListResult, error) {
	return a.ListFolder(ctx, rootFolderToken)
}

func (a *DriveAdapter) ListFolder(ctx context.Context, token string) (doctype.ListResult, error) {
	cacheKey := "drive:list:" + token
	if cached, ok := a.meta.Get(cacheKey); ok {
		return cached.(doctype.ListResult), nil
	}

	handler := a.registry.Handler(doctype.TypeFolder)
	list, err := handler.List(ctx, token)
	if err != nil {
		return doctype.ListResult{}, err
	}

	nameEntries := make([]naming.NameEntry, len(list.Entries))
	for i, e := range list.Entries {
		nameEntries[i] = naming.NameEntry{
			Name:  naming.SanitizeName(e.Name) + doctype.FileExtension(e.Type),
			Token: e.Token,
		}
	}

	resolved := a.namer.ResolveNames(nameEntries)
	for i := range list.Entries {
		if fname, ok := resolved[list.Entries[i].Token]; ok {
			list.Entries[i].Name = fname
		}
	}

	a.meta.Set(cacheKey, list)
	return list, nil
}

func (a *DriveAdapter) ListByType(ctx context.Context, token string, dt doctype.DocType) (doctype.ListResult, error) {
	cacheKey := fmt.Sprintf("drive:list:%s:%s", dt, token)
	if cached, ok := a.meta.Get(cacheKey); ok {
		return cached.(doctype.ListResult), nil
	}

	list, err := a.registry.Handler(dt).List(ctx, token)
	if err != nil {
		return doctype.ListResult{}, err
	}

	a.meta.Set(cacheKey, list)
	return list, nil
}

func (a *DriveAdapter) Read(ctx context.Context, token string, dt doctype.DocType) ([]byte, error) {
	if dt == doctype.TypeFile && strings.Contains(token, "|") {
		if resolved := a.resolveCompositeType(token); resolved != dt {
			return a.registry.Handler(resolved).Read(ctx, token)
		}
	}
	return a.registry.Handler(dt).Read(ctx, token)
}

func (a *DriveAdapter) resolveCompositeType(token string) doctype.DocType {
	prefix := strings.SplitN(token, "|", 2)[0]
	if strings.HasPrefix(prefix, "shtcn") {
		return doctype.TypeSheet
	}
	if strings.HasPrefix(prefix, "bascn") {
		return doctype.TypeBitable
	}
	return doctype.TypeFile
}

func (a *DriveAdapter) Write(ctx context.Context, token string, dt doctype.DocType, data []byte) error {
	if dt == doctype.TypeFile && strings.Contains(token, "|") {
		if resolved := a.resolveCompositeType(token); resolved != dt {
			dt = resolved
		}
	}
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
