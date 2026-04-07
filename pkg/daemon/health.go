package daemon

import (
	"context"
	"log/slog"
	"os/exec"
	"time"
)

type HealthChecker struct {
	cliPath  string
	interval time.Duration
}

func NewHealthChecker(cliPath string, interval time.Duration) *HealthChecker {
	return &HealthChecker{cliPath: cliPath, interval: interval}
}

func (h *HealthChecker) Run(ctx context.Context) {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.check(ctx)
		}
	}
}

func (h *HealthChecker) check(ctx context.Context) {
	cmd := exec.CommandContext(ctx, h.cliPath, "auth", "status")
	if err := cmd.Run(); err != nil {
		slog.Warn("health check: auth status failed", "error", err)
	}
}
