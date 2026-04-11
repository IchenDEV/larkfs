package vfs

import "errors"

var (
	ErrNotFound    = errors.New("not found")
	ErrReadOnly    = errors.New("read-only")
	ErrUnsupported = errors.New("unsupported operation")
)
