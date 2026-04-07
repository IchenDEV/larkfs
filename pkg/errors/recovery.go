package errors

import (
	"context"
	"errors"
	"log/slog"
	"os/exec"
	"sync"

	"github.com/IchenDEV/larkfs/pkg/cli"
)

type AuthRecovery struct {
	mu         sync.Mutex
	cliPath    string
	degraded   bool
}

func NewAuthRecovery(cliPath string) *AuthRecovery {
	return &AuthRecovery{cliPath: cliPath}
}

func (r *AuthRecovery) IsDegraded() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.degraded
}

func (r *AuthRecovery) HandleError(ctx context.Context, err error) error {
	if !errors.Is(err, cli.ErrAuthExpired) {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	slog.Info("auth expired, attempting refresh")
	if refreshErr := r.refresh(ctx); refreshErr != nil {
		slog.Error("auth refresh failed, entering degraded mode", "error", refreshErr)
		r.degraded = true
		return cli.ErrAuthExpired
	}

	slog.Info("auth refresh succeeded")
	r.degraded = false
	return nil
}

func (r *AuthRecovery) refresh(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, r.cliPath, "auth", "refresh")
	return cmd.Run()
}
