package errors

import (
	"syscall"
	"testing"

	"github.com/IchenDEV/larkfs/pkg/cli"
)

func TestToErrno(t *testing.T) {
	tests := []struct {
		err  error
		want syscall.Errno
	}{
		{nil, 0},
		{cli.ErrNotFound, syscall.ENOENT},
		{cli.ErrForbidden, syscall.EACCES},
		{cli.ErrRateLimited, syscall.EAGAIN},
		{cli.ErrAuthExpired, syscall.EROFS},
		{&cli.CLIError{ExitCode: -1}, syscall.ETIMEDOUT},
		{&cli.CLIError{ExitCode: 1}, syscall.EIO},
	}

	for _, tt := range tests {
		got := ToErrno(tt.err)
		if got != tt.want {
			t.Errorf("ToErrno(%v) = %v, want %v", tt.err, got, tt.want)
		}
	}
}
