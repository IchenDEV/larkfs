package errors

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/IchenDEV/larkfs/pkg/cli"
)

func TestWithRetrySuccess(t *testing.T) {
	calls := 0
	result, err := WithRetry(context.Background(), DefaultRetry, func() ([]byte, error) {
		calls++
		return []byte("ok"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(result) != "ok" {
		t.Errorf("expected ok, got %s", string(result))
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestWithRetryNonRetryable(t *testing.T) {
	calls := 0
	_, err := WithRetry(context.Background(), DefaultRetry, func() ([]byte, error) {
		calls++
		return nil, cli.ErrNotFound
	})
	if err != cli.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
	if calls != 1 {
		t.Errorf("non-retryable should only call once, got %d", calls)
	}
}

func TestWithRetryRetryable(t *testing.T) {
	cfg := RetryConfig{MaxRetries: 2, InitialBackoff: 10 * time.Millisecond, MaxBackoff: 100 * time.Millisecond}
	calls := 0
	_, err := WithRetry(context.Background(), cfg, func() ([]byte, error) {
		calls++
		return nil, cli.ErrRateLimited
	})
	if err != cli.ErrRateLimited {
		t.Errorf("expected ErrRateLimited, got %v", err)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls (1 + 2 retries), got %d", calls)
	}
}

func TestWithRetryEventualSuccess(t *testing.T) {
	cfg := RetryConfig{MaxRetries: 3, InitialBackoff: 10 * time.Millisecond, MaxBackoff: 100 * time.Millisecond}
	calls := 0
	result, err := WithRetry(context.Background(), cfg, func() ([]byte, error) {
		calls++
		if calls < 3 {
			return nil, cli.ErrRateLimited
		}
		return []byte("ok"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(result) != "ok" {
		t.Errorf("expected ok, got %s", string(result))
	}
}

func TestWithRetryContextCancel(t *testing.T) {
	cfg := RetryConfig{MaxRetries: 10, InitialBackoff: time.Second, MaxBackoff: time.Second}
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err := WithRetry(ctx, cfg, func() ([]byte, error) {
		return nil, cli.ErrRateLimited
	})
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		err  error
		want bool
	}{
		{cli.ErrRateLimited, true},
		{cli.ErrAuthExpired, false},
		{cli.ErrNotFound, false},
		{cli.ErrForbidden, false},
		{fmt.Errorf("random error"), false},
		{&cli.CLIError{ExitCode: 500}, true},
		{&cli.CLIError{ExitCode: 1}, false},
	}

	for _, tt := range tests {
		got := isRetryable(tt.err)
		if got != tt.want {
			t.Errorf("isRetryable(%v) = %v, want %v", tt.err, got, tt.want)
		}
	}
}
