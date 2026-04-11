package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/IchenDEV/larkfs/pkg/cache"
	"github.com/IchenDEV/larkfs/pkg/doctype"
	"github.com/IchenDEV/larkfs/pkg/naming"
)

type Runner struct {
	Out      []byte
	LastArgs []string
	Calls    [][]string
	RunFn    func(context.Context, ...string) ([]byte, error)
}

func (m *Runner) Path() string { return "mock-lark-cli" }

func (m *Runner) Run(ctx context.Context, args ...string) ([]byte, error) {
	m.LastArgs = append([]string(nil), args...)
	m.Calls = append(m.Calls, append([]string(nil), args...))
	if m.RunFn != nil {
		return m.RunFn(ctx, args...)
	}
	return append([]byte(nil), m.Out...), nil
}

func NewDeps(t *testing.T, runner *Runner) (*cache.MetadataCache, *naming.Resolver, *doctype.Registry) {
	t.Helper()
	meta := cache.NewMetadataCache(time.Minute)
	t.Cleanup(meta.Close)
	namer := naming.NewResolver(t.TempDir())
	registry := doctype.NewRegistry(runner, t.TempDir())
	return meta, namer, registry
}

func JoinArgs(parts []string) string {
	result := ""
	for i, part := range parts {
		if i > 0 {
			result += " "
		}
		result += part
	}
	return result
}
