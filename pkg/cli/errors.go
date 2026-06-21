package cli

import (
	"encoding/json"
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

type errorEnvelope struct {
	Error struct {
		Type    string `json:"type"`
		Subtype string `json:"subtype"`
		Code    int    `json:"code"`
		Message string `json:"message"`
		Hint    string `json:"hint"`
	} `json:"error"`
}

func classifyError(exitCode int, stderr string, args []string) error {
	if err := classifyTypedError(stderr); err != nil {
		return err
	}

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

func classifyTypedError(stderr string) error {
	var envelope errorEnvelope
	if err := json.Unmarshal([]byte(strings.TrimSpace(stderr)), &envelope); err != nil {
		return nil
	}
	if envelope.Error.Type == "" && envelope.Error.Subtype == "" && envelope.Error.Message == "" {
		return nil
	}

	lower := strings.ToLower(strings.Join([]string{
		envelope.Error.Type,
		envelope.Error.Subtype,
		envelope.Error.Message,
		envelope.Error.Hint,
	}, " "))

	switch {
	case envelope.Error.Code == 401,
		strings.Contains(lower, "unauthenticated"),
		strings.Contains(lower, "unauthorized"),
		strings.Contains(lower, "invalid_grant"),
		strings.Contains(lower, "token expired"),
		strings.Contains(lower, "token_expired"):
		return ErrAuthExpired
	case envelope.Error.Code == 429,
		strings.Contains(lower, "rate_limited"),
		strings.Contains(lower, "too_many_requests"),
		strings.Contains(lower, "rate limit"):
		return ErrRateLimited
	case envelope.Error.Code == 404,
		strings.Contains(lower, "not_found"):
		return ErrNotFound
	case envelope.Error.Code == 403,
		strings.Contains(lower, "permission_denied"),
		strings.Contains(lower, "forbidden"):
		return ErrForbidden
	}
	return nil
}

func containsNotFound(s string) bool {
	if strings.Contains(s, "command not found") || strings.Contains(s, "binary not found") {
		return false
	}
	return strings.Contains(s, "not found") || strings.Contains(s, "404")
}
