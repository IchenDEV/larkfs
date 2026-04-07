package naming

import (
	"strings"
	"testing"
)

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello.md", "hello.md"},
		{"my/file:name", "my_file_name"},
		{" .leading spaces. ", "leading spaces"},
		{"", "untitled"},
		{strings.Repeat("a", 300), strings.Repeat("a", 200)},
	}
	for _, tt := range tests {
		got := SanitizeName(tt.input)
		if got != tt.want {
			t.Errorf("SanitizeName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSanitizeNameUTF8(t *testing.T) {
	input := strings.Repeat("你", 100) // 300 bytes
	got := SanitizeName(input)
	if len(got) > 200 {
		t.Errorf("expected <= 200 bytes, got %d", len(got))
	}
	for i, r := range got {
		if r == 0xFFFD {
			t.Errorf("invalid UTF-8 rune at position %d", i)
		}
	}
}

func TestTruncateUTF8(t *testing.T) {
	s := "abc你好def"
	got := truncateUTF8(s, 5)
	if got != "abc" {
		t.Errorf("truncateUTF8(%q, 5) = %q, want %q", s, got, "abc")
	}

	got = truncateUTF8(s, 6)
	if got != "abc你" {
		t.Errorf("truncateUTF8(%q, 6) = %q, want %q", s, got, "abc你")
	}
}

func TestResolveNamesUnique(t *testing.T) {
	r := &Resolver{
		forward: make(map[string]string),
		reverse: make(map[string]string),
		mapFile: "/dev/null",
	}

	entries := []NameEntry{
		{Name: "doc.md", Token: "tok1"},
		{Name: "readme.md", Token: "tok2"},
	}
	result := r.ResolveNames(entries)

	if result["tok1"] != "doc.md" {
		t.Errorf("expected doc.md, got %s", result["tok1"])
	}
	if result["tok2"] != "readme.md" {
		t.Errorf("expected readme.md, got %s", result["tok2"])
	}
}

func TestResolveNamesConflict(t *testing.T) {
	r := &Resolver{
		forward: make(map[string]string),
		reverse: make(map[string]string),
		mapFile: "/dev/null",
	}

	entries := []NameEntry{
		{Name: "doc.md", Token: "abc1234567890"},
		{Name: "doc.md", Token: "xyz9876543210"},
	}
	result := r.ResolveNames(entries)

	name1 := result["abc1234567890"]
	name2 := result["xyz9876543210"]

	if name1 == name2 {
		t.Errorf("conflicting names should be different, got %q and %q", name1, name2)
	}
	if !strings.Contains(name1, "~") {
		t.Errorf("expected ~ suffix in %q", name1)
	}
	if !strings.Contains(name2, "~") {
		t.Errorf("expected ~ suffix in %q", name2)
	}
}
