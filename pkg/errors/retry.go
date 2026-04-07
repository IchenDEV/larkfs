package errors

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/IchenDEV/larkfs/pkg/cli"
)

type RetryConfig struct {
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
}

var DefaultRetry = RetryConfig{
	MaxRetries:     3,
	InitialBackoff: 1 * time.Second,
	MaxBackoff:     30 * time.Second,
}

func WithRetry(ctx context.Context, cfg RetryConfig, fn func() ([]byte, error)) ([]byte, error) {
	var lastErr error
	for i := 0; i <= cfg.MaxRetries; i++ {
		result, err := fn()
		if err == nil {
			return result, nil
		}
		lastErr = err

		if !isRetryable(err) {
			return nil, err
		}

		backoff := calcBackoff(cfg, i, err)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
	}
	return nil, lastErr
}

func isRetryable(err error) bool {
	if errors.Is(err, cli.ErrRateLimited) {
		return true
	}
	if errors.Is(err, cli.ErrAuthExpired) {
		return true
	}
	var cliErr *cli.CLIError
	if errors.As(err, &cliErr) {
		return cliErr.ExitCode >= 500
	}
	return false
}

func calcBackoff(cfg RetryConfig, attempt int, _ error) time.Duration {
	d := time.Duration(float64(cfg.InitialBackoff) * math.Pow(2, float64(attempt)))
	if d > cfg.MaxBackoff {
		d = cfg.MaxBackoff
	}
	return d
}
