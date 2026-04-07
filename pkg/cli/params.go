package cli

import "encoding/json"

func JSONParam(kv map[string]any) string {
	b, _ := json.Marshal(kv)
	return string(b)
}
