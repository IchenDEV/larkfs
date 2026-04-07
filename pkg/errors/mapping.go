package errors

import (
	"errors"
	"syscall"

	"github.com/IchenDEV/larkfs/pkg/cli"
)

func ToErrno(err error) syscall.Errno {
	if err == nil {
		return 0
	}

	switch {
	case errors.Is(err, cli.ErrNotFound):
		return syscall.ENOENT
	case errors.Is(err, cli.ErrForbidden):
		return syscall.EACCES
	case errors.Is(err, cli.ErrRateLimited):
		return syscall.EAGAIN
	case errors.Is(err, cli.ErrAuthExpired):
		return syscall.EROFS
	}

	var cliErr *cli.CLIError
	if errors.As(err, &cliErr) {
		if cliErr.ExitCode == -1 {
			return syscall.ETIMEDOUT
		}
		return syscall.EIO
	}

	return syscall.EIO
}
