package config_test

import (
	"testing"

	"github.com/IchenDEV/larkfs/pkg/config"
)

func TestParseByteSizeBlackbox(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		raw   string
		want  int64
		isErr bool
	}{
		{name: "bytes", raw: "42", want: 42},
		{name: "kb", raw: "2KB", want: 2 * 1024},
		{name: "mb lower", raw: "3mb", want: 3 * 1024 * 1024},
		{name: "spaces", raw: " 5 GB ", want: 5 * 1024 * 1024 * 1024},
		{name: "invalid unit", raw: "12XB", isErr: true},
		{name: "zero", raw: "0", isErr: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := config.ParseByteSize(tc.raw)
			if tc.isErr {
				if err == nil {
					t.Fatalf("ParseByteSize(%q) unexpectedly succeeded: %d", tc.raw, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseByteSize(%q) error: %v", tc.raw, err)
			}
			if got != tc.want {
				t.Fatalf("ParseByteSize(%q) = %d, want %d", tc.raw, got, tc.want)
			}
		})
	}
}

func TestMountConfigContentCacheSizeBytesDefaultsBlackbox(t *testing.T) {
	t.Parallel()

	cfg := config.MountConfig{}
	got, err := cfg.ContentCacheSizeBytes()
	if err != nil {
		t.Fatalf("ContentCacheSizeBytes() error: %v", err)
	}
	want := int64(500 * 1024 * 1024)
	if got != want {
		t.Fatalf("ContentCacheSizeBytes() = %d, want %d", got, want)
	}
}
