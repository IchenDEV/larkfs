package cli

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const defaultTimeout = 30 * time.Second

type RunMiddleware func(ctx context.Context, fn func() ([]byte, error)) ([]byte, error)

type Runner interface {
	Path() string
	Run(ctx context.Context, args ...string) ([]byte, error)
}

type Executor struct {
	binaryPath string
	timeout    time.Duration
	middleware RunMiddleware
}

func NewExecutor(binaryPath string) (*Executor, error) {
	path, err := FindLarkCLI(binaryPath)
	if err != nil {
		return nil, err
	}
	return &Executor{binaryPath: path, timeout: defaultTimeout}, nil
}

func (e *Executor) Path() string { return e.binaryPath }

func (e *Executor) SetMiddleware(mw RunMiddleware) {
	e.middleware = mw
}

func (e *Executor) Run(ctx context.Context, args ...string) ([]byte, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), e.timeout)
		defer cancel()
	}

	if e.middleware != nil {
		return e.middleware(ctx, func() ([]byte, error) {
			return e.runOnce(ctx, args...)
		})
	}
	return e.runOnce(ctx, args...)
}

func (e *Executor) runOnce(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, e.binaryPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, &CLIError{ExitCode: -1, Stderr: "timeout", Cmd: strings.Join(args, " ")}
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, classifyError(exitErr.ExitCode(), stderr.String(), args)
		}
		return nil, fmt.Errorf("exec lark-cli: %w", err)
	}
	return stdout.Bytes(), nil
}

func (e *Executor) RunJSON(ctx context.Context, args ...string) ([]byte, error) {
	fullArgs := make([]string, len(args), len(args)+2)
	copy(fullArgs, args)
	fullArgs = append(fullArgs, "--format", "json")
	return e.Run(ctx, fullArgs...)
}

func FindLarkCLI(hint string) (string, error) {
	if hint != "" {
		if _, err := exec.LookPath(hint); err == nil {
			return hint, nil
		}
		return "", fmt.Errorf("lark-cli not found at %q", hint)
	}
	for _, name := range []string{"lark-cli", "lark"} {
		if p, err := exec.LookPath(name); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("lark-cli not found in PATH; install via: npm install -g @larksuite/cli")
}
