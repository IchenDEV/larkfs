package cli

import (
	"strings"
	"testing"
)

func TestCLIErrorFormat(t *testing.T) {
	tests := []struct {
		exitCode int
		cmd      string
	}{
		{1, "test cmd"},
		{10, "another cmd"},
		{127, "missing cmd"},
		{255, "bad cmd"},
	}

	for _, tt := range tests {
		e := &CLIError{ExitCode: tt.exitCode, Stderr: "some error", Cmd: tt.cmd}
		msg := e.Error()
		if !strings.Contains(msg, tt.cmd) {
			t.Errorf("error message should contain cmd %q, got: %s", tt.cmd, msg)
		}
		if strings.ContainsRune(msg, rune(0)) {
			t.Errorf("error message contains null byte for exit code %d: %q", tt.exitCode, msg)
		}
	}
}

func TestClassifyError(t *testing.T) {
	tests := []struct {
		stderr string
		want   error
	}{
		{"token expired", ErrAuthExpired},
		{"rate limit exceeded", ErrRateLimited},
		{"resource not found", ErrNotFound},
		{"permission denied", ErrForbidden},
	}

	for _, tt := range tests {
		err := classifyError(1, tt.stderr, []string{"test"})
		if err != tt.want {
			t.Errorf("classifyError(%q) = %v, want %v", tt.stderr, err, tt.want)
		}
	}
}
