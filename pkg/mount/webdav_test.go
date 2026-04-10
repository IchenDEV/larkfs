package mount

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDepthInfinityBlocked(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if depth := r.Header.Get("Depth"); strings.EqualFold(depth, "infinity") {
			http.Error(w, "Depth: infinity is not supported", http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		depth  string
		expect int
	}{
		{"infinity", http.StatusForbidden},
		{"Infinity", http.StatusForbidden},
		{"0", http.StatusOK},
		{"1", http.StatusOK},
		{"", http.StatusOK},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("PROPFIND", "/drive/", nil)
		if tt.depth != "" {
			req.Header.Set("Depth", tt.depth)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != tt.expect {
			t.Errorf("Depth=%q: got %d, want %d", tt.depth, rec.Code, tt.expect)
		}
	}
}

func TestContentTypeFromName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"doc.md", "text/markdown; charset=utf-8"},
		{"data.csv", "text/csv; charset=utf-8"},
		{"config.json", "application/json; charset=utf-8"},
		{"log.jsonl", "application/x-ndjson; charset=utf-8"},
		{"readme.txt", "text/plain; charset=utf-8"},
		{"video.mp4", "video/mp4"},
		{"image.png", "image/png"},
		{"data.bin", "application/octet-stream"},
		{"unknown.xyz", "application/octet-stream"},
	}
	for _, tt := range tests {
		got := contentTypeFromName(tt.name)
		if got != tt.want {
			t.Errorf("contentTypeFromName(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}
