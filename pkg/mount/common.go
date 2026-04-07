package mount

import (
	"context"
	"time"

	"github.com/IchenDEV/larkfs/pkg/adapter"
	"github.com/IchenDEV/larkfs/pkg/cache"
	clipkg "github.com/IchenDEV/larkfs/pkg/cli"
	"github.com/IchenDEV/larkfs/pkg/config"
	"github.com/IchenDEV/larkfs/pkg/doctype"
	lkerr "github.com/IchenDEV/larkfs/pkg/errors"
	"github.com/IchenDEV/larkfs/pkg/naming"
	"github.com/IchenDEV/larkfs/pkg/vfs"
)

type mountState struct {
	ops          *vfs.Operations
	meta         *cache.MetadataCache
	content      *cache.ContentCache
	authRecovery *lkerr.AuthRecovery
}

func buildMount(cfg config.MountConfig) (*mountState, error) {
	exec, err := clipkg.NewExecutor(cfg.LarkCLIPath)
	if err != nil {
		return nil, err
	}

	authRecovery := lkerr.NewAuthRecovery(exec.Path())

	exec.SetMiddleware(func(ctx context.Context, fn func() ([]byte, error)) ([]byte, error) {
		result, err := lkerr.WithRetry(ctx, lkerr.DefaultRetry, fn)
		if err != nil {
			if recovered := authRecovery.HandleError(ctx, err); recovered == nil {
				return fn()
			}
		}
		return result, err
	})

	ttl := time.Duration(cfg.MetadataTTL) * time.Second
	meta := cache.NewMetadataCache(ttl)

	contentCache, err := cache.NewContentCache(cfg.CacheDir, 500*1024*1024)
	if err != nil {
		return nil, err
	}

	registry := doctype.NewRegistry(exec, cfg.CacheDir)
	namer := naming.NewResolver(config.BaseDir())
	tree := vfs.NewTree(cfg.EnabledDomains())

	driveAdapter := adapter.NewDriveAdapter(exec, registry, meta, namer)
	wikiAdapter := adapter.NewWikiAdapter(exec, registry, meta, namer)
	calendarAdapter := adapter.NewCalendarAdapter(exec, meta, namer)
	taskAdapter := adapter.NewTaskAdapter(exec, meta, namer)
	imAdapter := adapter.NewIMAdapter(exec, meta, namer)
	mailAdapter := adapter.NewMailAdapter(exec, meta, namer)
	meetingAdapter := adapter.NewMeetingAdapter(exec, meta, namer, cfg.CacheDir)

	ops := vfs.NewOperations(vfs.OperationsConfig{
		Tree:     tree,
		Drive:    driveAdapter,
		Wiki:     wikiAdapter,
		Calendar: calendarAdapter,
		Task:     taskAdapter,
		IM:       imAdapter,
		Mail:     mailAdapter,
		Meeting:  meetingAdapter,
		ReadOnly: cfg.ReadOnly,
		TTL:      ttl,
	})

	return &mountState{
		ops:          ops,
		meta:         meta,
		content:      contentCache,
		authRecovery: authRecovery,
	}, nil
}
