package errors_test

import (
	"context"
	stderrors "errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/IchenDEV/larkfs/pkg/cli"
	larkerrors "github.com/IchenDEV/larkfs/pkg/errors"
)

func TestRetryAndAuthRecoveryBlackbox(t *testing.T) {
	attempts := 0
	out, err := larkerrors.WithRetry(context.Background(), larkerrors.RetryConfig{
		MaxRetries:     2,
		InitialBackoff: time.Nanosecond,
		MaxBackoff:     time.Nanosecond,
	}, func() ([]byte, error) {
		attempts++
		if attempts == 1 {
			return nil, cli.ErrRateLimited
		}
		return []byte("ok"), nil
	})
	if err != nil || string(out) != "ok" || attempts != 2 {
		t.Fatalf("WithRetry(retry) = %q, %v attempts=%d", out, err, attempts)
	}

	_, err = larkerrors.WithRetry(context.Background(), larkerrors.RetryConfig{
		MaxRetries:     2,
		InitialBackoff: time.Nanosecond,
		MaxBackoff:     time.Nanosecond,
	}, func() ([]byte, error) {
		return nil, cli.ErrForbidden
	})
	if !stderrors.Is(err, cli.ErrForbidden) {
		t.Fatalf("WithRetry(non-retryable) error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = larkerrors.WithRetry(ctx, larkerrors.RetryConfig{
		MaxRetries:     1,
		InitialBackoff: time.Hour,
		MaxBackoff:     time.Hour,
	}, func() ([]byte, error) {
		return nil, &cli.CLIError{ExitCode: 500}
	})
	if !stderrors.Is(err, context.Canceled) {
		t.Fatalf("WithRetry(canceled) error = %v", err)
	}

	failPath := filepath.Join(t.TempDir(), "fail")
	if err := os.WriteFile(failPath, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("write fail script: %v", err)
	}
	recovery := larkerrors.NewAuthRecovery(failPath)
	if err := recovery.HandleError(context.Background(), cli.ErrNotFound); err != cli.ErrNotFound {
		t.Fatalf("non-auth error = %v", err)
	}
	if err := recovery.HandleError(context.Background(), cli.ErrAuthExpired); err != cli.ErrAuthExpired || !recovery.IsDegraded() {
		t.Fatalf("failed auth recovery error=%v degraded=%v", err, recovery.IsDegraded())
	}

	okPath := filepath.Join(t.TempDir(), "ok")
	if err := os.WriteFile(okPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write ok script: %v", err)
	}
	ok := larkerrors.NewAuthRecovery(okPath)
	if err := ok.HandleError(context.Background(), cli.ErrAuthExpired); err != nil || ok.IsDegraded() {
		t.Fatalf("successful auth recovery error=%v degraded=%v", err, ok.IsDegraded())
	}
}
