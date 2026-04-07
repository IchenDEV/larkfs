package cli

import (
	"errors"
	"strconv"
	"strings"
)

var (
	ErrAuthExpired = errors.New("authentication token expired")
	ErrRateLimited = errors.New("rate limited")
	ErrNotFound    = errors.New("resource not found")
	ErrForbidden   = errors.New("permission denied")
)

type CLIError struct {
	ExitCode int
	Stderr   string
	Cmd      string
}

func (e *CLIError) Error() string {
	return "lark-cli error: " + e.Stderr + " (cmd: " + e.Cmd + ", exit: " + strconv.Itoa(e.ExitCode) + ")"
}

func classifyError(exitCode int, stderr string, args []string) error {
	cmd := strings.Join(args, " ")
	lower := strings.ToLower(stderr)

	switch {
	case strings.Contains(lower, "token expired"),
		strings.Contains(lower, "invalid_grant"),
		strings.Contains(lower, "unauthorized"):
		return ErrAuthExpired

	case strings.Contains(lower, "rate limit"),
		strings.Contains(lower, "too many requests"):
		return ErrRateLimited

	case containsNotFound(lower):
		return ErrNotFound

	case strings.Contains(lower, "permission denied"),
		strings.Contains(lower, "forbidden"),
		strings.Contains(lower, "403"):
		return ErrForbidden
	}

	return &CLIError{ExitCode: exitCode, Stderr: stderr, Cmd: cmd}
}

func containsNotFound(s string) bool {
	if strings.Contains(s, "command not found") || strings.Contains(s, "binary not found") {
		return false
	}
	return strings.Contains(s, "not found") || strings.Contains(s, "404")
}
