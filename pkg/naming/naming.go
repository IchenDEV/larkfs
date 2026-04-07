package naming

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"
)

type Resolver struct {
	mu      sync.RWMutex
	forward map[string]string // token -> filename
	reverse map[string]string // filename -> token
	mapFile string
}

func NewResolver(baseDir string) *Resolver {
	r := &Resolver{
		forward: make(map[string]string),
		reverse: make(map[string]string),
		mapFile: filepath.Join(baseDir, "namemap.json"),
	}
	r.load()
	return r
}

type NameEntry struct {
	Name  string
	Token string
}

func (r *Resolver) ResolveNames(entries []NameEntry) map[string]string {
	r.mu.Lock()
	defer r.mu.Unlock()

	grouped := make(map[string][]NameEntry)
	for _, e := range entries {
		grouped[e.Name] = append(grouped[e.Name], e)
	}

	result := make(map[string]string, len(entries))
	for name, group := range grouped {
		if len(group) == 1 {
			e := group[0]
			if existing, ok := r.forward[e.Token]; ok {
				result[e.Token] = existing
			} else {
				result[e.Token] = name
				r.forward[e.Token] = name
				r.reverse[name] = e.Token
			}
		} else {
			for _, e := range group {
				if existing, ok := r.forward[e.Token]; ok {
					result[e.Token] = existing
					continue
				}
				fname := addTokenSuffix(name, e.Token)
				result[e.Token] = fname
				r.forward[e.Token] = fname
				r.reverse[fname] = e.Token
			}

			for _, e := range group {
				fname := result[e.Token]
				if fname == name {
					delete(r.reverse, fname)
					fname = addTokenSuffix(name, e.Token)
					result[e.Token] = fname
					r.forward[e.Token] = fname
					r.reverse[fname] = e.Token
				}
			}
		}
	}

	r.save()
	return result
}

func addTokenSuffix(name, token string) string {
	ext := ""
	if dot := strings.LastIndex(name, "."); dot > 0 {
		ext = name[dot:]
		name = name[:dot]
	}
	return name + "~" + shortToken(token, 7) + ext
}

func (r *Resolver) TokenForName(filename string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.reverse[filename]
	return t, ok
}

func (r *Resolver) load() {
	data, err := os.ReadFile(r.mapFile)
	if err != nil {
		return
	}
	var m map[string]string
	if json.Unmarshal(data, &m) == nil {
		r.forward = m
		for token, fname := range m {
			r.reverse[fname] = token
		}
	}
}

func (r *Resolver) save() {
	data, err := json.MarshalIndent(r.forward, "", "  ")
	if err != nil {
		slog.Warn("naming: failed to marshal map", "error", err)
		return
	}
	if err := os.MkdirAll(filepath.Dir(r.mapFile), 0o755); err != nil {
		slog.Warn("naming: failed to create dir", "error", err)
		return
	}
	if err := os.WriteFile(r.mapFile, data, 0o644); err != nil {
		slog.Warn("naming: failed to persist name map", "path", r.mapFile, "error", err)
	}
}

func shortToken(token string, n int) string {
	if len(token) <= n {
		return token
	}
	return token[len(token)-n:]
}

func SanitizeName(name string) string {
	illegal := `/\:*?"<>|`
	for _, c := range illegal {
		name = strings.ReplaceAll(name, string(c), "_")
	}
	name = strings.Trim(name, " .")
	name = truncateUTF8(name, 200)
	if name == "" {
		name = "untitled"
	}
	return name
}

func truncateUTF8(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	for maxBytes > 0 && !utf8.RuneStart(s[maxBytes]) {
		maxBytes--
	}
	return s[:maxBytes]
}
