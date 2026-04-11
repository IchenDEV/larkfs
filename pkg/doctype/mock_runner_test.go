package doctype

import "context"

type mockRunner struct {
	runFn func(context.Context, ...string) ([]byte, error)
}

func (m *mockRunner) Path() string { return "mock-lark-cli" }

func (m *mockRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	return m.runFn(ctx, args...)
}
